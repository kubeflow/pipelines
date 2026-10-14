// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at https://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func testConfig(t *testing.T, endpoint string) Config {
	t.Helper()
	dir := t.TempDir()
	c := Config{StartAt: time.Now().Add(300 * time.Millisecond), ProbeSeconds: 1, Endpoint: endpoint, Label: "test", RequestSeconds: 1, DrainSeconds: 2, MaxInflightPerTenant: 2, Phases: []Phase{{Name: "normal", Seconds: 1, QuietRPS: 2, NoisyRPS: 2}}}
	for _, name := range []string{"alice", "bob"} {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte("secret-"+name), 0600); err != nil {
			t.Fatal(err)
		}
		c.Tenants = append(c.Tenants, TenantConfig{Name: name, Namespace: name, ExperimentID: "exp-" + name, TokenFile: p, Fixtures: []Fixture{{Name: "fixture", PipelineID: "pipeline", VersionID: "immutable-version"}}})
	}
	return c
}
func TestActualGeneratedClientsTenantAuthAndCaptureBeforeCleanup(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "evidence")
	var creates atomic.Int32
	var cleanupBeforeCapture atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Header.Get("Authorization") == "Bearer kfp-apf-intentionally-invalid-token" {
			w.WriteHeader(401)
			_, _ = w.Write([]byte(`{"code":16,"message":"invalid token"}`))
			return
		}
		who := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer secret-")
		if who != "alice" && who != "bob" {
			t.Errorf("unexpected auth identity")
			w.WriteHeader(403)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case "POST":
			var b map[string]any
			if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
				t.Error(err)
			}
			if b["namespace"] != who {
				t.Errorf("wrong tenant namespace %v", b["namespace"])
			}
			creates.Add(1)
			_, _ = w.Write([]byte(`{"run_id":"run-` + who + `","state":"PENDING"}`))
		case "DELETE":
			if _, err := os.Stat(filepath.Join(dir, "results.json")); err != nil {
				cleanupBeforeCapture.Store(true)
			}
			_, _ = w.Write([]byte(`{}`))
		default:
			if strings.HasSuffix(r.URL.Path, "/run-"+who) {
				_, _ = w.Write([]byte(`{"run_id":"run-` + who + `","namespace":"` + who + `","experiment_id":"exp-` + who + `","state":"SUCCEEDED"}`))
			} else if strings.Contains(r.URL.Path, "/run-") {
				w.WriteHeader(403)
				_, _ = w.Write([]byte(`{"code":7,"message":"forbidden"}`))
			} else {
				if r.URL.Query().Get("namespace") != who {
					t.Error("list namespace missing")
				}
				_, _ = w.Write([]byte(`{"runs":[],"total_size":0}`))
			}
		}
	}))
	defer server.Close()
	if err := execute(context.Background(), testConfig(t, server.URL), dir, true); err != nil {
		t.Fatal(err)
	}
	if creates.Load() != 2 {
		t.Fatalf("expected exactly one active run per tenant, got %d", creates.Load())
	}
	if cleanupBeforeCapture.Load() {
		t.Fatal("cleanup preceded capture")
	}
	for _, file := range []string{"events.jsonl", "results.json"} {
		b, err := os.ReadFile(filepath.Join(dir, file))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(b), "secret-") {
			t.Fatal("credential leaked")
		}
	}
	if err := execute(context.Background(), testConfig(t, server.URL), dir, false); err == nil {
		t.Fatal("reused evidence directory")
	}
}
func TestAmbiguousCreateNeverRetried(t *testing.T) {
	var creates atomic.Int32
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		creates.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(500)
		_, _ = w.Write([]byte(`{"message":"body must not be logged: secret-token","code":13}`))
	}))
	defer s.Close()
	c := testConfig(t, s.URL)
	tenant, err := makeTenant(c, c.Tenants[0])
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.CreateTemp(t.TempDir(), "events")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rec := &Recorder{f: f}
	tenant.pipelines(context.Background(), c, rec, "pressure", true)
	tenant.pipelines(context.Background(), c, rec, "pressure", true)
	if creates.Load() != 1 || !tenant.ambiguous {
		t.Fatal("ambiguous create retried")
	}
	if rec.events[0].Outcome != "create_unresolved" {
		t.Fatal("unresolved create not recorded")
	}
}
func TestReadDeadlineAndFixedRateDrops(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	defer s.Close()
	c := testConfig(t, s.URL)
	c.MaxInflightPerTenant = 1
	tenant, err := makeTenant(c, c.Tenants[0])
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.CreateTemp(t.TempDir(), "events")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rec := &Recorder{f: f}
	start := time.Now()
	offers(context.Background(), c, tenant, rec, Phase{Name: "pressure", Seconds: 1}, 20, time.Now())
	if time.Since(start) > 2*time.Second {
		t.Fatal("deadline failed")
	}
	drops := 0
	deadlines := 0
	for _, e := range rec.events {
		if e.Outcome == "inflight_drop" {
			drops++
		}
		if e.Outcome == "deadline" {
			deadlines++
		}
	}
	if drops == 0 || deadlines != 1 {
		t.Fatalf("drops=%d deadlines=%d", drops, deadlines)
	}
}
func TestConfigRefusesUnboundedOrCrossTenantConfiguration(t *testing.T) {
	c := testConfig(t, "http://localhost:8080")
	if err := validate(c); err != nil {
		t.Fatal(err)
	}
	c.Tenants[1].Namespace = c.Tenants[0].Namespace
	if err := validate(c); err == nil {
		t.Fatal("shared namespace accepted")
	}
	c = testConfig(t, "http://localhost:8080")
	c.Phases[0].Seconds = 301
	if err := validate(c); err == nil {
		t.Fatal("unbounded phase accepted")
	}
}

func TestPermissionProbesRequireOwnerAndExactStructuredDenials(t *testing.T) {
	for _, tc := range []struct {
		name        string
		ownerOK     bool
		crossHTTP   int
		crossRPC    int
		invalidHTTP int
		invalidRPC  int
		want        bool
	}{
		{"strict-success", true, 403, 7, 401, 16, true},
		{"HTTP403-with-wrong-RPC", true, 403, 16, 401, 16, false},
		{"HTTP401-is-not-cross-tenant-denial", true, 401, 16, 401, 16, false},
		{"notfound-does-not-prove-isolation", true, 404, 5, 401, 16, false},
		{"invalid-token-must-not-authorize", true, 403, 7, 200, 0, false},
		{"invalid-token-wrong-RPC", true, 403, 7, 401, 7, false},
		{"owner-mismatch-invalidates-denial", false, 403, 7, 401, 16, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				who := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer secret-")
				owner := strings.TrimPrefix(filepath.Base(r.URL.Path), "run-")
				if r.Header.Get("Authorization") == "Bearer kfp-apf-intentionally-invalid-token" {
					w.WriteHeader(tc.invalidHTTP)
					_ = json.NewEncoder(w).Encode(map[string]any{"code": tc.invalidRPC, "message": "secret-invalid-body"})
					return
				}
				if who != owner {
					w.WriteHeader(tc.crossHTTP)
					_ = json.NewEncoder(w).Encode(map[string]any{"code": tc.crossRPC, "message": "secret-forbidden-body"})
					return
				}
				ns := owner
				if !tc.ownerOK {
					ns = "wrong-namespace"
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"run_id": "run-" + owner, "namespace": ns, "experiment_id": "exp-" + owner})
			}))
			defer s.Close()
			c := testConfig(t, s.URL)
			tenants := []*Tenant{}
			for _, cfg := range c.Tenants {
				tenant, err := makeTenant(c, cfg)
				if err != nil {
					t.Fatal(err)
				}
				tenant.lastID = "run-" + cfg.Name
				tenants = append(tenants, tenant)
			}
			f, err := os.CreateTemp(t.TempDir(), "events")
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			rec := &Recorder{f: f}
			coverage := permissionProbes(context.Background(), c, rec, "window", tenants)
			if coverage.Complete != tc.want || coverage.Expected != 6 {
				t.Fatalf("coverage=%+v", coverage)
			}
			if len(rec.events) != 6 {
				t.Fatalf("missing event records: %d", len(rec.events))
			}
			if err := f.Sync(); err != nil {
				t.Fatal(err)
			}
			b, _ := os.ReadFile(f.Name())
			if strings.Contains(string(b), "secret-") {
				t.Fatal("credential/error-body leak")
			}
		})
	}
}
func TestPermissionProbeMissingRunIsExplicitFailure(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("missing run must not issue request") }))
	defer s.Close()
	c := testConfig(t, s.URL)
	tenants := []*Tenant{}
	for _, cfg := range c.Tenants {
		tenant, err := makeTenant(c, cfg)
		if err != nil {
			t.Fatal(err)
		}
		tenants = append(tenants, tenant)
	}
	f, err := os.CreateTemp(t.TempDir(), "events")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rec := &Recorder{f: f}
	coverage := permissionProbes(context.Background(), c, rec, "window", tenants)
	if coverage.Complete || coverage.Passed != 0 || len(rec.events) != 6 {
		t.Fatalf("missing coverage silently accepted: %+v", coverage)
	}
	for _, e := range rec.events {
		if e.Outcome != "probe_missing_run" {
			t.Fatalf("unexpected outcome %s", e.Outcome)
		}
	}
}

func TestSuccessfulWorkflowsCannotHidePermissionFailure(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		who := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer secret-")
		if r.Header.Get("Authorization") == "Bearer kfp-apf-intentionally-invalid-token" {
			w.WriteHeader(401)
			_, _ = w.Write([]byte(`{"code":16}`))
			return
		}
		if r.Method == "POST" {
			_ = json.NewEncoder(w).Encode(map[string]any{"run_id": "run-" + who, "state": "PENDING"})
			return
		}
		if strings.HasSuffix(r.URL.Path, "/run-"+who) {
			_ = json.NewEncoder(w).Encode(map[string]any{"run_id": "run-" + who, "namespace": who, "experiment_id": "exp-" + who, "state": "SUCCEEDED"})
			return
		}
		if strings.Contains(r.URL.Path, "/run-") {
			w.WriteHeader(403)
			_, _ = w.Write([]byte(`{"code":16}`))
			return
		} // Wrong denial contract.
		_, _ = w.Write([]byte(`{"runs":[]}`))
	}))
	defer s.Close()
	dir := filepath.Join(t.TempDir(), "results")
	err := execute(context.Background(), testConfig(t, s.URL), dir, false)
	if err == nil {
		t.Fatal("permission failure did not fail execution")
	}
	b, err := os.ReadFile(filepath.Join(dir, "results.json"))
	if err != nil {
		t.Fatal(err)
	}
	var results struct {
		Workflows   bool `json:"all_recorded_runs_succeeded"`
		Permissions bool `json:"permission_probe_coverage_complete"`
	}
	if err := json.Unmarshal(b, &results); err != nil {
		t.Fatal(err)
	}
	if !results.Workflows || results.Permissions {
		t.Fatalf("success/permission conflated: %+v", results)
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("disk full") }
func TestCaptureFailureCancelsOffersPromptly(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"runs":[]}`))
	}))
	defer server.Close()
	c := testConfig(t, server.URL)
	tenant, err := makeTenant(c, c.Tenants[0])
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	rec := &Recorder{writer: failingWriter{}, cancel: cancel}
	start := time.Now()
	offers(ctx, c, tenant, rec, Phase{Name: "pressure", Seconds: 5}, 100, time.Now())
	if time.Since(start) > time.Second {
		t.Fatal("traffic did not stop promptly on capture failure")
	}
	if ctx.Err() == nil || rec.captureError() == nil {
		t.Fatal("capture failure failed to cancel")
	}
	n := requests.Load()
	if n < 1 || n > int32(c.MaxInflightPerTenant) {
		t.Fatalf("unexpected requests after failure: %d", n)
	}
	for range 100 {
		rec.add(Event{})
	}
	if len(rec.events) != 0 || rec.bytes != 0 {
		t.Fatal("failed capture retained events or bytes")
	}
}
func TestCaptureBudgetsCancelWithoutGrowth(t *testing.T) {
	for _, kind := range []string{"bytes", "events"} {
		t.Run(kind, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			r := &Recorder{writer: io.Discard, cancel: cancel}
			if kind == "bytes" {
				r.bytes = maxRecordedBytes
			} else {
				r.events = make([]Event, maxRecordedEvents)
			}
			n := len(r.events)
			b := r.bytes
			r.add(Event{})
			if ctx.Err() == nil || r.captureError() == nil {
				t.Fatal("budget overflow did not cancel")
			}
			for range 100 {
				r.add(Event{})
			}
			if len(r.events) != n || r.bytes != b {
				t.Fatal("continued growth after capture budget")
			}
		})
	}
}
func TestSummaryIncludesCompletelySilentBuckets(t *testing.T) {
	start := time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)
	c := Config{StartAt: start, Phases: []Phase{{Name: "pressure", Seconds: 25}}, Tenants: []TenantConfig{{Name: "noisy"}, {Name: "quiet"}}}
	events := []Event{{Time: start.Add(2 * time.Second), Phase: "pressure", Tenant: "noisy", Operation: "get", Outcome: "ok", ElapsedMS: 1}, {Time: start.Add(26 * time.Second), Phase: "pressure", Tenant: "quiet", Operation: "get", Outcome: "ok", ElapsedMS: 1}}
	got := summarize(events, c)
	for _, name := range []string{"noisy", "quiet"} {
		s := got["pressure/"+name]
		if len(s.SuccessfulReadsBy10Seconds) != 3 {
			t.Fatalf("silent bucket missing: %+v", s)
		}
		if s.SuccessfulReadsBy10Seconds[1] != 0 || s.SuccessfulReadsBy10Seconds[2] != 0 {
			t.Fatal("silent window hidden")
		}
	}
	if got["pressure/noisy"].SuccessfulReadsBy10Seconds[0] != 1 || got["pressure/quiet"].SuccessfulReadsBy10Seconds[0] != 0 || got["pressure/quiet"].LateCompletions != 1 {
		t.Fatal("completion outside declared phase credited as progress")
	}
}
func TestTimelineHasExplicitProbeSlotsAndRejectsMissedStart(t *testing.T) {
	c := testConfig(t, "http://127.0.0.1:1")
	c.Phases = []Phase{{Name: "normal", Seconds: 10, QuietRPS: 1, NoisyRPS: 1}, {Name: "pressure", Seconds: 20, QuietRPS: 1, NoisyRPS: 10}}
	c.ProbeSeconds = 30
	c.DrainSeconds = 60
	s := timeline(c)
	if len(s) != 5 || !s[1].Start.Equal(c.StartAt.Add(10*time.Second)) || !s[2].Start.Equal(c.StartAt.Add(30*time.Second)) || !s[3].Start.Equal(c.StartAt.Add(60*time.Second)) || !s[4].End.Equal(c.StartAt.Add(150*time.Second)) {
		t.Fatalf("wrong absolute timeline: %+v", s)
	}
	c.StartAt = time.Now().Add(-time.Second)
	dir := filepath.Join(t.TempDir(), "absent")
	if err := execute(context.Background(), c, dir, false); err == nil {
		t.Fatal("missed start accepted")
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatal("missed start created output or contacted server")
	}
}
func TestOffersRespectSharedFutureStart(t *testing.T) {
	var first atomic.Int64
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		first.CompareAndSwap(0, time.Now().UnixNano())
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"runs":[]}`))
	}))
	defer s.Close()
	c := testConfig(t, s.URL)
	tenant, err := makeTenant(c, c.Tenants[0])
	if err != nil {
		t.Fatal(err)
	}
	rec := &Recorder{writer: io.Discard}
	start := time.Now().Add(100 * time.Millisecond)
	offers(context.Background(), c, tenant, rec, Phase{Name: "normal", Seconds: 1}, 1, start)
	if first.Load() < start.UnixNano() {
		t.Fatal("offered traffic before shared clock")
	}
}

func TestSyncFailureCancelsTraffic(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "events")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r := &Recorder{f: f, cancel: cancel}
	r.add(Event{Operation: "phase_start"})
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if err := r.sync(); err == nil {
		t.Fatal("sync failure ignored")
	}
	if ctx.Err() == nil {
		t.Fatal("sync failure did not cancel traffic")
	}
	before := len(r.events)
	r.add(Event{})
	if len(r.events) != before {
		t.Fatal("capture grew after sync failure")
	}
}

func TestTwoActiveFixtureCycleAndDrainCannotHideOlderRun(t *testing.T) {
	var creates int
	var firstDone bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "POST" {
			creates++
			json.NewEncoder(w).Encode(map[string]any{"run_id": fmt.Sprint(creates), "state": "PENDING"})
			return
		}
		state := "RUNNING"
		if strings.HasSuffix(r.URL.Path, "/1") && firstDone {
			state = "SUCCEEDED"
		}
		json.NewEncoder(w).Encode(map[string]any{"run_id": strings.TrimPrefix(r.URL.Path, "/apis/v2beta1/runs/"), "state": state})
	}))
	defer server.Close()
	c := testConfig(t, server.URL)
	c.MaxActivePerTenant = 2
	c.Tenants[0].Fixtures = []Fixture{{Name: "flip", PipelineID: "p", VersionID: "v1"}, {Name: "collected", PipelineID: "p", VersionID: "v2"}, {Name: "oneof", PipelineID: "p", VersionID: "v3"}}
	tenant, err := makeTenant(c, c.Tenants[0])
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.CreateTemp(t.TempDir(), "events")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rec := &Recorder{f: f, cancel: func() {}}
	tenant.pipelines(context.Background(), c, rec, "normal", true)
	tenant.pipelines(context.Background(), c, rec, "normal", true)
	tenant.pipelines(context.Background(), c, rec, "normal", true)
	if creates != 2 || activeRuns(tenant) != 2 {
		t.Fatal("active cap exceeded")
	}
	firstDone = true
	tenant.pipelines(context.Background(), c, rec, "pressure", true)
	if creates != 3 || tenant.runs[2].Fixture != "oneof" || activeRuns(tenant) != 2 {
		t.Fatal("third fixture did not cycle into freed slot")
	}
	tenant.runs[2].State = "SUCCEEDED"
	if activeRuns(tenant) != 1 {
		t.Fatal("older running fixture hidden by newest terminal fixture")
	}
	tenant.pipelines(context.Background(), c, rec, "drain", false)
	if creates != 3 {
		t.Fatal("drain submitted new work")
	}
}
func TestActivePipelineBounds(t *testing.T) {
	c := testConfig(t, "http://127.0.0.1")
	c.MaxActivePerTenant = 3
	if validate(c) == nil {
		t.Fatal("unsafe active cap accepted")
	}
	c.MaxActivePerTenant = 2
	if validate(c) != nil {
		t.Fatal("bounded variant rejected")
	}
	if activeLimit(Config{}) != 1 {
		t.Fatal("default changed")
	}
}
