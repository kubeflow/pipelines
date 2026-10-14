// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// https://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This candidate drives actual KFP endpoints; it does not provision a cluster.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-openapi/runtime"
	transport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	client "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_client"
	params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_client/run_service"
	model "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_model"
)

type Fixture struct {
	Name          string                      `json:"name"`
	PipelineID    string                      `json:"pipeline_id"`
	VersionID     string                      `json:"version_id"`
	RuntimeConfig *model.V2beta1RuntimeConfig `json:"runtime_config,omitempty"`
}
type TenantConfig struct {
	Name           string    `json:"name"`
	Namespace      string    `json:"namespace"`
	ExperimentID   string    `json:"experiment_id"`
	ServiceAccount string    `json:"service_account"`
	TokenFile      string    `json:"token_file"`
	Fixtures       []Fixture `json:"fixtures"`
}
type Phase struct {
	Name     string  `json:"name"`
	Seconds  int     `json:"seconds"`
	QuietRPS float64 `json:"quiet_rps"`
	NoisyRPS float64 `json:"noisy_rps"`
}
type Config struct {
	StartAt              time.Time      `json:"start_at"`
	ProbeSeconds         int            `json:"probe_seconds"`
	Endpoint             string         `json:"endpoint"`
	Label                string         `json:"label"`
	RequestSeconds       int            `json:"request_seconds"`
	DrainSeconds         int            `json:"drain_seconds"`
	MaxInflightPerTenant int            `json:"max_inflight_per_tenant"`
	Tenants              []TenantConfig `json:"tenants"`
	Phases               []Phase        `json:"phases"`
}
type Event struct {
	Time          time.Time `json:"time"`
	Phase         string    `json:"phase"`
	Tenant        string    `json:"tenant"`
	Operation     string    `json:"operation"`
	Outcome       string    `json:"outcome"`
	HTTPCode      int       `json:"http_code,omitempty"`
	RPCCode       int32     `json:"rpc_code,omitempty"`
	TargetTenant  string    `json:"target_tenant,omitempty"`
	ElapsedMS     float64   `json:"elapsed_ms,omitempty"`
	ScheduleLagMS float64   `json:"schedule_lag_ms,omitempty"`
	RunID         string    `json:"run_id,omitempty"`
	Fixture       string    `json:"fixture,omitempty"`
	State         string    `json:"state,omitempty"`
	ScheduledAt   string    `json:"scheduled_at,omitempty"`
}

const maxRecordedEvents = 100000
const maxRecordedBytes = 32 << 20

type Recorder struct {
	mu     sync.Mutex
	f      *os.File
	writer io.Writer // Test injection; production writes directly to f.
	events []Event
	err    error
	cancel context.CancelFunc
	bytes  int
}

func (r *Recorder) failLocked(err error) {
	if r.err == nil {
		r.err = err
		if r.cancel != nil {
			r.cancel()
		}
	}
}
func (r *Recorder) add(e Event) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return
	}
	if len(r.events) >= maxRecordedEvents {
		r.failLocked(errors.New("capture event limit exceeded"))
		return
	}
	e.Time = time.Now().UTC()
	b, err := json.Marshal(e)
	if err != nil {
		r.failLocked(err)
		return
	}
	b = append(b, '\n')
	if r.bytes+len(b) > maxRecordedBytes {
		r.failLocked(errors.New("capture byte limit exceeded"))
		return
	}
	w := r.writer
	if w == nil {
		w = r.f
	}
	n, err := w.Write(b)
	r.bytes += n
	if err == nil && n != len(b) {
		err = io.ErrShortWrite
	}
	if err != nil {
		r.failLocked(errors.New("capture write failed"))
		return
	}
	r.events = append(r.events, e)
}
func (r *Recorder) sync() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err == nil {
		if err := r.f.Sync(); err != nil {
			r.failLocked(errors.New("capture sync failed"))
		}
	}
	return r.err
}
func (r *Recorder) captureError() error { r.mu.Lock(); defer r.mu.Unlock(); return r.err }

type Window struct {
	Name  string    `json:"name"`
	Kind  string    `json:"kind"`
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

func timeline(c Config) []Window {
	at := c.StartAt
	windows := []Window{}
	for _, p := range c.Phases {
		end := at.Add(time.Duration(p.Seconds) * time.Second)
		windows = append(windows, Window{p.Name, "load", at, end})
		at = end
	}
	for _, stage := range []struct {
		name, kind string
		seconds    int
	}{{"after_load", "probe", c.ProbeSeconds}, {"drain", "drain", c.DrainSeconds}, {"after_drain", "probe", c.ProbeSeconds}} {
		end := at.Add(time.Duration(stage.seconds) * time.Second)
		windows = append(windows, Window{stage.name, stage.kind, at, end})
		at = end
	}
	return windows
}
func waitUntil(ctx context.Context, at time.Time) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	d := time.Until(at)
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

type Run struct {
	ID      string `json:"id"`
	Fixture string `json:"fixture"`
	State   string `json:"state"`
}
type Tenant struct {
	cfg       TenantConfig
	api       *client.Run
	auth      runtime.ClientAuthInfoWriter
	mu        sync.Mutex
	runs      []Run
	ambiguous bool
	lastID    string
}

func (t *Tenant) latest() string { t.mu.Lock(); defer t.mu.Unlock(); return t.lastID }

func terminal(s string) bool {
	return s == "SUCCEEDED" || s == "FAILED" || s == "CANCELED" || s == "CANCELLED" || s == "SKIPPED"
}
func outcome(err error) (string, int) {
	if err == nil {
		return "ok", 200
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "deadline", 0
	}
	if errors.Is(err, context.Canceled) {
		return "cancelled", 0
	}
	var c interface{ Code() int }
	if errors.As(err, &c) {
		return "http_error", c.Code()
	}
	return "transport_error", 0
}
func validate(c Config) error {
	if c.StartAt.IsZero() || !c.StartAt.After(time.Now()) || time.Until(c.StartAt) > time.Minute {
		return errors.New("start_at must be a future RFC3339 timestamp within one minute")
	}
	if c.ProbeSeconds < 1 || c.ProbeSeconds > 120 {
		return errors.New("probe_seconds must be 1..120")
	}
	u, e := url.Parse(c.Endpoint)
	if e != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Scheme != "http" && u.Scheme != "https") {
		return errors.New("invalid endpoint")
	}
	if c.Label == "" || c.RequestSeconds < 1 || c.RequestSeconds > 30 || c.DrainSeconds < 1 || c.DrainSeconds > 600 || c.MaxInflightPerTenant < 1 || c.MaxInflightPerTenant > 16 || len(c.Tenants) < 2 || len(c.Tenants) > 4 || len(c.Phases) == 0 {
		return errors.New("invalid bounds")
	}
	names := map[string]bool{}
	namespaces := map[string]bool{}
	total := 0
	for _, t := range c.Tenants {
		if names[t.Name] || namespaces[t.Namespace] || t.Name == "" || t.Namespace == "" || t.TokenFile == "" || t.ExperimentID == "" || len(t.Fixtures) == 0 {
			return errors.New("tenant needs unique name/namespace, token file, experiment and fixtures")
		}
		names[t.Name] = true
		namespaces[t.Namespace] = true
		for _, f := range t.Fixtures {
			if f.Name == "" || f.PipelineID == "" || f.VersionID == "" {
				return errors.New("fixture must pin pipeline and version")
			}
		}
	}
	names = map[string]bool{}
	offered := 0.0
	for _, p := range c.Phases {
		if names[p.Name] || p.Name == "" || p.Name == "after_load" || p.Name == "drain" || p.Name == "after_drain" || p.Seconds < 1 || p.Seconds > 300 || math.IsNaN(p.QuietRPS) || math.IsNaN(p.NoisyRPS) || p.QuietRPS < 0.1 || p.QuietRPS > 1000 || p.NoisyRPS < 0.1 || p.NoisyRPS > 1000 {
			return errors.New("invalid phase/rate")
		}
		names[p.Name] = true
		total += p.Seconds
		offered += float64(p.Seconds) * (p.NoisyRPS + float64(len(c.Tenants)-1)*p.QuietRPS)
	}
	if total > 1800 || offered > 90000 {
		return errors.New("phase duration exceeds 30 minutes")
	}
	return nil
}
func makeTenant(c Config, tc TenantConfig) (*Tenant, error) {
	token, e := os.ReadFile(tc.TokenFile)
	if e != nil {
		return nil, errors.New("cannot read tenant token file")
	}
	tok := strings.TrimSpace(string(token))
	if tok == "" {
		return nil, errors.New("empty tenant token")
	}
	u, _ := url.Parse(c.Endpoint)
	base := u.Path
	if base == "" {
		base = "/"
	}
	rt := transport.NewWithClient(u.Host, base, []string{u.Scheme}, &http.Client{Timeout: time.Duration(c.RequestSeconds) * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }})
	return &Tenant{cfg: tc, api: client.New(rt, strfmt.Default), auth: transport.BearerToken(tok)}, nil
}
func requestContext(parent context.Context, c Config) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, time.Duration(c.RequestSeconds)*time.Second)
}
func (t *Tenant) read(parent context.Context, c Config, r *Recorder, phase string, n int, lag float64) {
	ctx, cancel := requestContext(parent, c)
	defer cancel()
	start := time.Now()
	op := "list"
	var err error
	id := t.latest()
	if n%2 == 0 && id != "" {
		op = "get"
		_, err = t.api.RunService.RunServiceGetRunContext(ctx, &params.RunServiceGetRunParams{RunID: id}, t.auth)
	} else {
		size := int32(10)
		_, err = t.api.RunService.RunServiceListRunsContext(ctx, &params.RunServiceListRunsParams{Namespace: &t.cfg.Namespace, ExperimentID: &t.cfg.ExperimentID, PageSize: &size}, t.auth)
	}
	result, code := outcome(err)
	r.add(Event{Phase: phase, Tenant: t.cfg.Name, Operation: op, Outcome: result, HTTPCode: code, ElapsedMS: float64(time.Since(start).Microseconds()) / 1000, ScheduleLagMS: lag})
}

// The absolute schedule records missed offers rather than backpressuring the rate.
func offers(ctx context.Context, c Config, t *Tenant, r *Recorder, p Phase, rate float64, start time.Time) {
	stop := start.Add(time.Duration(p.Seconds) * time.Second)
	interval := time.Duration(float64(time.Second) / rate)
	sem := make(chan struct{}, c.MaxInflightPerTenant)
	var wg sync.WaitGroup
	defer wg.Wait()
	for n := 0; ; n++ {
		due := start.Add(time.Duration(n) * interval)
		if !due.Before(stop) {
			return
		}
		wait := time.Until(due)
		if wait > 0 {
			timer := time.NewTimer(wait)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
		}
		if ctx.Err() != nil {
			return
		}
		lag := float64(time.Since(due).Microseconds()) / 1000
		if time.Now().After(stop) || lag > float64(interval.Microseconds())/1000 {
			r.add(Event{Phase: p.Name, Tenant: t.cfg.Name, Operation: "offer", Outcome: "schedule_drop", ScheduleLagMS: lag})
			continue
		}
		select {
		case sem <- struct{}{}:
			wg.Add(1)
			go func(i int, l float64) { defer wg.Done(); defer func() { <-sem }(); t.read(ctx, c, r, p.Name, i, l) }(n, lag)
		default:
			r.add(Event{Phase: p.Name, Tenant: t.cfg.Name, Operation: "offer", Outcome: "inflight_drop", ScheduleLagMS: lag})
		}
	}
}

// One active pipeline per tenant, cycling immutable fixture versions.
func (t *Tenant) pipelines(ctx context.Context, c Config, r *Recorder, phase string, submit bool) {
	if len(t.runs) > 0 && !terminal(t.runs[len(t.runs)-1].State) {
		rr := &t.runs[len(t.runs)-1]
		call, cancel := requestContext(ctx, c)
		start := time.Now()
		resp, err := t.api.RunService.RunServiceGetRunContext(call, &params.RunServiceGetRunParams{RunID: rr.ID}, t.auth)
		cancel()
		res, code := outcome(err)
		if err == nil && resp.Payload != nil && resp.Payload.State != nil {
			rr.State = string(*resp.Payload.State)
		}
		r.add(Event{Phase: phase, Tenant: t.cfg.Name, Operation: "poll_pipeline", Outcome: res, HTTPCode: code, RunID: rr.ID, Fixture: rr.Fixture, State: rr.State, ElapsedMS: float64(time.Since(start).Microseconds()) / 1000})
		if !terminal(rr.State) {
			return
		}
	}
	if !submit || t.ambiguous || ctx.Err() != nil {
		return
	}
	f := t.cfg.Fixtures[len(t.runs)%len(t.cfg.Fixtures)]
	call, cancel := requestContext(ctx, c)
	defer cancel()
	start := time.Now()
	resp, err := t.api.RunService.RunServiceCreateRunContext(call, &params.RunServiceCreateRunParams{Run: &model.V2beta1Run{DisplayName: fmt.Sprintf("apf-%s-%s-%d", c.Label, t.cfg.Name, len(t.runs)), Namespace: t.cfg.Namespace, ExperimentID: t.cfg.ExperimentID, ServiceAccount: t.cfg.ServiceAccount, PipelineVersionReference: &model.V2beta1PipelineVersionReference{PipelineID: f.PipelineID, PipelineVersionID: f.VersionID}, RuntimeConfig: f.RuntimeConfig}}, t.auth)
	res, code := outcome(err)
	id := ""
	if err == nil && resp.Payload != nil {
		id = resp.Payload.RunID
	}
	if id != "" {
		t.runs = append(t.runs, Run{ID: id, Fixture: f.Name, State: "PENDING"})
		t.mu.Lock()
		t.lastID = id
		t.mu.Unlock()
	} else {
		t.ambiguous = true
		res = "create_unresolved"
	} // Never retry a possibly accepted CreateRun.
	r.add(Event{Phase: phase, Tenant: t.cfg.Name, Operation: "create", Outcome: res, HTTPCode: code, RunID: id, Fixture: f.Name, ElapsedMS: float64(time.Since(start).Microseconds()) / 1000})
}

// Probes are read-only. The owner must prove the exact resource exists before
// any denial can count as ownership evidence. Raw status/error messages stay private.
type ProbeCoverage struct {
	Window   string `json:"window"`
	Expected int    `json:"expected"`
	Passed   int    `json:"passed"`
	Complete bool   `json:"complete"`
}

func permissionProbes(ctx context.Context, c Config, r *Recorder, window string, tenants []*Tenant) ProbeCoverage {
	coverage := ProbeCoverage{Window: window, Expected: len(tenants) * 3}
	for i, owner := range tenants {
		id := owner.latest()
		other := tenants[(i+1)%len(tenants)]
		ownerOK := false
		for _, kind := range []string{"owner_get", "cross_tenant_get", "invalid_token_get"} {
			actor := owner
			auth := owner.auth
			expectedHTTP := 200
			expectedRPC := int32(0)
			if kind == "cross_tenant_get" {
				actor = other
				auth = other.auth
				expectedHTTP = 403
				expectedRPC = 7
			}
			if kind == "invalid_token_get" {
				auth = transport.BearerToken("kfp-apf-intentionally-invalid-token")
				expectedHTTP = 401
				expectedRPC = 16
			}
			event := Event{Phase: window, Tenant: actor.cfg.Name, TargetTenant: owner.cfg.Name, Operation: kind, RunID: id}
			if id == "" {
				event.Outcome = "probe_missing_run"
				r.add(event)
				continue
			}
			if kind != "owner_get" && !ownerOK {
				event.Outcome = "probe_owner_unverified"
				r.add(event)
				continue
			}
			call, cancel := requestContext(ctx, c)
			start := time.Now()
			response, err := actor.api.RunService.RunServiceGetRunContext(call, &params.RunServiceGetRunParams{RunID: id}, auth)
			cancel()
			event.ElapsedMS = float64(time.Since(start).Microseconds()) / 1000
			event.Outcome, event.HTTPCode = outcome(err)
			passed := false
			if kind == "owner_get" {
				passed = err == nil && response != nil && response.Payload != nil && response.Payload.RunID == id && response.Payload.Namespace == owner.cfg.Namespace && response.Payload.ExperimentID == owner.cfg.ExperimentID
				ownerOK = passed
			} else {
				var denied *params.RunServiceGetRunDefault
				if errors.As(err, &denied) && denied.Payload != nil {
					event.RPCCode = denied.Payload.Code
					passed = denied.Code() == expectedHTTP && denied.Payload.Code == expectedRPC
				}
			}
			if passed {
				event.Outcome = "probe_pass"
				coverage.Passed++
			} else {
				event.Outcome = "probe_failed"
			}
			r.add(event)
		}
	}
	coverage.Complete = coverage.Passed == coverage.Expected && coverage.Expected >= 6
	return coverage
}

type Stats struct {
	Count                      int         `json:"count"`
	Errors                     int         `json:"errors"`
	Drops                      int         `json:"drops"`
	P50MS                      float64     `json:"p50_ms"`
	P95MS                      float64     `json:"p95_ms"`
	LateCompletions            int         `json:"late_completions"`
	SuccessfulReadsBy10Seconds map[int]int `json:"successful_reads_by_10s"`
}

func summarize(events []Event, c Config) map[string]*Stats {
	out := map[string]*Stats{}
	samples := map[string][]float64{}
	windows := map[string]Window{}
	for _, w := range timeline(c) {
		if w.Kind != "load" {
			continue
		}
		windows[w.Name] = w
		for _, t := range c.Tenants {
			st := &Stats{SuccessfulReadsBy10Seconds: map[int]int{}}
			for i := 0; i < int(math.Ceil(w.End.Sub(w.Start).Seconds()/10)); i++ {
				st.SuccessfulReadsBy10Seconds[i] = 0
			}
			out[w.Name+"/"+t.Name] = st
		}
	}
	for _, e := range events {
		w, ok := windows[e.Phase]
		if !ok {
			continue
		}
		key := e.Phase + "/" + e.Tenant
		st := out[key]
		if st == nil {
			continue
		}
		if e.Operation == "offer" {
			st.Drops++
			continue
		}
		if e.Operation != "get" && e.Operation != "list" {
			continue
		}
		st.Count++
		samples[key] = append(samples[key], e.ElapsedMS)
		if e.Outcome != "ok" {
			st.Errors++
		}
		if e.Time.Before(w.Start) || !e.Time.Before(w.End) {
			st.LateCompletions++
			continue
		}
		if e.Outcome == "ok" {
			bucket := int(e.Time.Sub(w.Start).Seconds() / 10)
			st.SuccessfulReadsBy10Seconds[bucket]++
		}
	}
	for k, x := range samples {
		sort.Float64s(x)
		out[k].P50MS = x[(len(x)-1)/2]
		out[k].P95MS = x[int(math.Ceil(.95*float64(len(x))))-1]
	}
	return out
}

func writeJSON(path string, v any) error {
	b, e := json.MarshalIndent(v, "", "  ")
	if e != nil {
		return e
	}
	return os.WriteFile(path, append(b, '\n'), 0600)
}
func execute(ctx context.Context, c Config, out string, cleanup bool) error {
	if err := validate(c); err != nil {
		return err
	}
	if err := os.Mkdir(out, 0700); err != nil {
		return err
	} // Refuse overwriting prior evidence.
	f, err := os.OpenFile(filepath.Join(out, "events.jsonl"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	ctx, stopTraffic := context.WithCancel(ctx)
	defer stopTraffic()
	r := &Recorder{f: f, cancel: stopTraffic}
	syncDone := make(chan struct{})
	go func() {
		defer close(syncDone)
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if r.sync() != nil {
					return
				}
			}
		}
	}()
	defer func() { stopTraffic(); <-syncDone }()
	tenants := []*Tenant{}
	for _, tc := range c.Tenants {
		t, e := makeTenant(c, tc)
		if e != nil {
			return e
		}
		tenants = append(tenants, t)
	}
	schedule := timeline(c)
	if err := writeJSON(filepath.Join(out, "timeline.json"), schedule); err != nil {
		stopTraffic()
		return err
	}
	if !time.Now().Before(c.StartAt) {
		return errors.New("missed start_at while preparing workload; no traffic sent")
	}
	if err := waitUntil(ctx, c.StartAt); err != nil {
		return err
	}
	probeResults := []ProbeCoverage{}
	for index, w := range schedule {
		if err := waitUntil(ctx, w.Start); err != nil {
			break
		}
		if !time.Now().Before(w.End) {
			stopTraffic()
			break
		} // Do not shift an overrun into later windows.
		stageCtx, cancel := context.WithDeadline(ctx, w.End)
		for _, t := range tenants {
			r.add(Event{Phase: w.Name, Tenant: t.cfg.Name, Operation: "phase_start", Outcome: "ok", ScheduledAt: w.Start.Format(time.RFC3339Nano)})
		}
		switch w.Kind {
		case "load":
			p := c.Phases[index]
			var wg sync.WaitGroup
			for i, t := range tenants {
				rate := p.QuietRPS
				if i == 0 {
					rate = p.NoisyRPS
				}
				wg.Add(2)
				go func(t *Tenant, rate float64) { defer wg.Done(); offers(stageCtx, c, t, r, p, rate, w.Start) }(t, rate)
				go func(t *Tenant) {
					defer wg.Done()
					ticker := time.NewTicker(time.Second)
					defer ticker.Stop()
					for {
						if stageCtx.Err() != nil {
							return
						}
						t.pipelines(stageCtx, c, r, p.Name, true)
						select {
						case <-stageCtx.Done():
							return
						case <-ticker.C:
						}
					}
				}(t)
			}
			wg.Wait()
		case "probe":
			probeResults = append(probeResults, permissionProbes(stageCtx, c, r, w.Name, tenants))
		case "drain":
			for stageCtx.Err() == nil {
				all := true
				for _, t := range tenants {
					t.pipelines(stageCtx, c, r, w.Name, false)
					if len(t.runs) > 0 && !terminal(t.runs[len(t.runs)-1].State) {
						all = false
					}
				}
				if all {
					break
				}
				select {
				case <-stageCtx.Done():
				case <-time.After(time.Second):
				}
			}
		}
		// Probe/drain stages retain their declared slots even when work finishes early.
		_ = waitUntil(ctx, w.End)
		cancel()
		for _, t := range tenants {
			r.add(Event{Phase: w.Name, Tenant: t.cfg.Name, Operation: "phase_end", Outcome: "ok", ScheduledAt: w.End.Format(time.RFC3339Nano)})
		}
		if err := r.sync(); err != nil {
			return err
		}
	}
	probesValid := len(probeResults) == 2
	for _, p := range probeResults {
		if !p.Complete {
			probesValid = false
		}
	}
	runs := map[string][]Run{}
	valid := ctx.Err() == nil
	for _, t := range tenants {
		runs[t.cfg.Name] = t.runs
		if t.ambiguous || len(t.runs) == 0 {
			valid = false
		}
		for _, rr := range t.runs {
			if rr.State != "SUCCEEDED" {
				valid = false
			}
		}
	}
	// Capture results BEFORE cleanup; users can retain resources for independent logs/metrics.
	if err := r.sync(); err != nil {
		return err
	}
	if err := r.captureError(); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(out, "results.json"), map[string]any{"label": c.Label, "phases": c.Phases, "timeline": schedule, "read_stats": summarize(r.events, c), "runs": runs, "all_recorded_runs_succeeded": valid, "permission_probes": probeResults, "permission_probe_coverage_complete": probesValid, "scope": "workload slice with scheduled post-load/post-drain read permission probes only; during-load denials/APF saturation/fairness/output validation not established"}); err != nil {
		return err
	}
	if cleanup {
		clean, cc := context.WithTimeout(ctx, 2*time.Minute)
		defer cc()
		for _, t := range tenants {
			for _, rr := range t.runs {
				if clean.Err() != nil {
					valid = false
					break
				}
				call, stop := requestContext(clean, c)
				var err error
				op := "delete"
				if terminal(rr.State) {
					_, err = t.api.RunService.RunServiceDeleteRunContext(call, &params.RunServiceDeleteRunParams{RunID: rr.ID}, t.auth)
				} else {
					op = "terminate"
					_, err = t.api.RunService.RunServiceTerminateRunContext(call, &params.RunServiceTerminateRunParams{RunID: rr.ID}, t.auth)
				}
				stop()
				res, code := outcome(err)
				r.add(Event{Phase: "cleanup", Tenant: t.cfg.Name, Operation: op, Outcome: res, HTTPCode: code, RunID: rr.ID})
				if err != nil {
					valid = false
				}
			}
		}
		if err := r.sync(); err != nil {
			return err
		}
	}
	if err := r.captureError(); err != nil {
		return err
	}
	if !valid || !probesValid {
		return errors.New("unresolved create, incomplete/failed workflow, permission probe failure/coverage, cancellation or cleanup error; inspect saved evidence")
	}
	return nil
}
func main() {
	path := flag.String("config", "", "JSON configuration; token values must stay in separate files")
	out := flag.String("out", "", "new evidence directory")
	cleanup := flag.Bool("cleanup", false, "delete terminal runs/terminate active runs AFTER result capture; default retains them")
	flag.Parse()
	b, err := os.ReadFile(*path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot read config")
		os.Exit(1)
	}
	var c Config
	decoder := json.NewDecoder(strings.NewReader(string(b)))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&c); err == nil {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		ctx, cancel := context.WithTimeout(ctx, 45*time.Minute)
		defer cancel()
		err = execute(ctx, c, *out, *cleanup)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
