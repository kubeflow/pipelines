// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package readinessobservation records bounded source observations without
// participating in authentication, authorization, or request execution.
package readinessobservation

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const SourceRevision = "2511cdbd74cd531c6633f2e094d235082a32917d"
const maxRecords = 512
const maxBytes = 1024 * 1024
const queueCapacity = 128

var operations = []string{"upload_pipeline", "upload_pipeline_version", "create_run", "create_recurring_run", "retry_run", "read_run_log", "read_artifact", "authorization"}
var unsupported = []string{"frontend_operations", "artifact_destinations", "caller_groups", "authenticated_identity_on_bypasses", "target_policy_evaluation", "request_authorization_correlation", "workload_execution", "operations_before_or_after_interval"}
var active atomic.Pointer[recorder]

type event struct {
	ObservedAt       string `json:"observed_at"`
	Operation        string `json:"operation"`
	APIVersion       string `json:"api_version,omitempty"`
	Namespace        string `json:"namespace,omitempty"`
	NamespaceOmitted *bool  `json:"namespace_omitted,omitempty"`
	Caller           string `json:"caller,omitempty"`
	Resource         string `json:"resource,omitempty"`
	Verb             string `json:"verb,omitempty"`
	Result           string `json:"result,omitempty"`
}

type health struct {
	WriteFailures uint64 `json:"write_failures"`
	DroppedQueue  uint64 `json:"dropped_queue"`
	DroppedLimit  uint64 `json:"dropped_limit"`
	InvalidFields uint64 `json:"invalid_fields"`
}

type limits struct {
	DurationSeconds int `json:"duration_seconds"`
	MaxRecords      int `json:"max_records"`
	MaxBytes        int `json:"max_bytes"`
}

type report struct {
	SchemaVersion        string            `json:"schema_version"`
	SourceVersion        string            `json:"source_version"`
	SourceRevision       string            `json:"source_revision"`
	StartedAt            string            `json:"started_at"`
	PlannedEndAt         string            `json:"planned_end_at"`
	CheckpointAt         string            `json:"checkpoint_at"`
	EndedAt              string            `json:"ended_at,omitempty"`
	Status               string            `json:"status"`
	Health               health            `json:"health"`
	Limits               limits            `json:"limits"`
	SupportedOperations  []string          `json:"supported_operations"`
	UnobservedOperations []string          `json:"unobserved_operations"`
	UnsupportedChecks    []string          `json:"unsupported_checks"`
	Operations           map[string]uint64 `json:"operations"`
	Records              []event           `json:"records"`
}

type recorder struct {
	path       string
	end        time.Time
	queue      chan event
	stop       chan struct{}
	done       chan struct{}
	stopOnce   sync.Once
	acceptance sync.RWMutex
	closed     atomic.Bool
	dropped    atomic.Uint64
	state      report
	write      func(string, []byte) error
}

// Start enables observation only when both administrator-owned settings are
// valid. Invalid observation configuration never changes server enforcement.
func Start(ctx context.Context) {
	path := os.Getenv("KFP_READINESS_OBSERVATION_FILE")
	if path == "" {
		return
	}
	seconds, err := strconv.Atoi(os.Getenv("KFP_READINESS_OBSERVATION_SECONDS"))
	if err != nil || seconds < 1 || seconds > 7*24*60*60 {
		log.Print("Readiness observation disabled: invalid duration")
		return
	}
	r, err := newRecorder(path, time.Duration(seconds)*time.Second)
	if err != nil {
		log.Print("Readiness observation disabled: output must be a new file in a private absolute directory")
		return
	}
	if !active.CompareAndSwap(nil, r) {
		log.Print("Readiness observation disabled: observer already started")
		return
	}
	go r.run(ctx)
}

// Shutdown requests a final checkpoint without indefinitely delaying shutdown.
// Abrupt exits can leave an active checkpoint; consumers must not accept it as
// a completed observation interval.
func Shutdown() {
	if r := active.Swap(nil); r != nil {
		r.closed.Store(true)
		r.stopOnce.Do(func() { close(r.stop) })
		select {
		case <-r.done:
		case <-time.After(2 * time.Second):
			log.Print("Readiness observation shutdown incomplete")
		}
	}
}

func newRecorder(path string, duration time.Duration) (*recorder, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path || duration <= 0 || duration > 7*24*time.Hour {
		return nil, errors.New("invalid observation configuration")
	}
	parent, err := os.Lstat(filepath.Dir(path))
	if err != nil || !parent.IsDir() || parent.Mode().Perm()&0077 != 0 {
		return nil, errors.New("output directory must be private and not a symlink")
	}
	// Exclusive creation refuses existing files and symlinks. The private parent
	// directory keeps snapshot replacement under the operator's control.
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, err
	}
	if err := file.Close(); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	r := &recorder{path: path, end: now.Add(duration), queue: make(chan event, queueCapacity), stop: make(chan struct{}), done: make(chan struct{}), write: atomicSnapshot}
	r.state = report{
		SchemaVersion: "kfp-source-observation/v1", SourceVersion: "2.17.2", SourceRevision: SourceRevision,
		StartedAt: now.Format(time.RFC3339Nano), PlannedEndAt: r.end.Format(time.RFC3339Nano), Status: "active",
		Limits: limits{int(duration / time.Second), maxRecords, maxBytes}, SupportedOperations: operations, UnsupportedChecks: unsupported,
		Operations: make(map[string]uint64), Records: []event{},
	}
	for _, operation := range operations {
		r.state.Operations[operation] = 0
	}
	return r, nil
}

// Request observes an existing API boundary, not a successful workload or an
// authenticated caller. Namespace is validated before it enters a report.
func Request(operation, apiVersion, namespace string, omitted bool) {
	if r := active.Load(); r != nil && !r.closed.Load() && time.Now().Before(r.end) {
		observedOmission := omitted
		r.enqueue(event{Operation: operation, APIVersion: apiVersion, Namespace: namespace, NamespaceOmitted: &observedOmission})
	}
}

// Upload records only namespace omission/value; it never reads a body, other
// query parameters, credentials, or the raw URL.
func Upload(operation, apiVersion string, request *http.Request) {
	r := active.Load()
	if r == nil || r.closed.Load() || !time.Now().Before(r.end) {
		return
	}
	if len(r.queue) == cap(r.queue) {
		r.dropped.Add(1)
		return
	}
	if request == nil || request.URL == nil {
		Request(operation, apiVersion, "", true)
		return
	}
	namespace, omitted, valid := uploadNamespace(request.URL.RawQuery)
	if !valid {
		r.enqueue(event{Operation: operation, APIVersion: apiVersion, Namespace: "!invalid"})
		return
	}
	r.enqueue(event{Operation: operation, APIVersion: apiVersion, Namespace: namespace, NamespaceOmitted: &omitted})
}

func uploadNamespace(query string) (string, bool, bool) {
	if len(query) > 8192 {
		return "", false, false
	}
	for range 64 {
		if query == "" {
			return "", true, true
		}
		var field string
		field, query, _ = strings.Cut(query, "&")
		// Match net/url's handling of malformed individual query fields without
		// decoding or retaining unrelated parameter values.
		if strings.Contains(field, ";") {
			continue
		}
		key, value, _ := strings.Cut(field, "=")
		decoded, err := url.QueryUnescape(key)
		if err != nil || decoded != "namespace" {
			continue
		}
		namespace, err := url.QueryUnescape(value)
		if err == nil {
			return namespace, false, true
		}
	}
	return "", false, false
}

// Authorization uses only the identity and result already computed by the
// source server. It never performs authentication or an additional SAR.
func Authorization(caller, namespace, resource, verb, result string) {
	if r := active.Load(); r != nil {
		r.enqueue(event{Operation: "authorization", Caller: caller, Namespace: namespace, Resource: resource, Verb: verb, Result: result})
	}
}

func (r *recorder) enqueue(value event) {
	now := time.Now().UTC()
	if r.closed.Load() || !now.Before(r.end) {
		return
	}
	if !r.acceptance.TryRLock() {
		r.dropped.Add(1)
		return
	}
	defer r.acceptance.RUnlock()
	if r.closed.Load() {
		return
	}
	// Copy only bounded strings into the queue; otherwise retaining a short
	// substring could retain an arbitrarily large request allocation.
	value.ObservedAt = now.Format(time.RFC3339Nano)
	value.Operation = bounded(value.Operation, 40)
	value.APIVersion = bounded(value.APIVersion, 20)
	value.Namespace = bounded(value.Namespace, 63)
	value.Caller = bounded(value.Caller, 256)
	value.Resource = bounded(value.Resource, 40)
	value.Verb = bounded(value.Verb, 30)
	value.Result = bounded(value.Result, 40)
	select {
	case r.queue <- value:
	default:
		r.dropped.Add(1)
	}
}

func bounded(value string, size int) string {
	if len(value) > size {
		return "!invalid"
	}
	return strings.Clone(value)
}

func oneOf(value string, values ...string) bool {
	for _, allowed := range values {
		if value == allowed {
			return true
		}
	}
	return false
}

func identifier(value string, principal bool) bool {
	for _, char := range value {
		if char >= 'a' && char <= 'z' || char >= '0' && char <= '9' || char == '-' || char == '.' {
			continue
		}
		if principal && (char >= 'A' && char <= 'Z' || char == '@' || char == ':' || char == '_' || char == '/') {
			continue
		}
		return false
	}
	return true
}

func (r *recorder) consume(value event) {
	if !oneOf(value.Operation, operations...) {
		r.state.Health.InvalidFields++
		return
	}
	r.state.Operations[value.Operation]++
	if !identifier(value.Namespace, false) {
		value.Namespace = ""
		r.state.Health.InvalidFields++
	}
	if !identifier(value.Caller, true) {
		value.Caller = ""
		r.state.Health.InvalidFields++
	}
	if !oneOf(value.APIVersion, "", "v1beta1", "v2beta1", "shared") {
		value.APIVersion = ""
		r.state.Health.InvalidFields++
	}
	if value.Operation == "authorization" {
		if !oneOf(value.Resource, "pipelines", "runs", "jobs", "experiments", "viewers", "visualizations", "serviceaccounts") {
			value.Resource = "other"
		}
		if !oneOf(value.Verb, "create", "get", "list", "delete", "update", "patch", "archive", "unarchive", "retry", "terminate", "enable", "disable", "reportMetrics", "readArtifact", "readLog", "use") {
			value.Verb = "other"
		}
		if !oneOf(value.Result, "allowed", "denied", "authentication_failed", "authorizer_error", "allowed_with_evaluation_error", "denied_with_evaluation_error", "single_user_bypass", "shared_read_bypass", "incomplete") {
			value.Result = "incomplete"
			r.state.Health.InvalidFields++
		}
	}
	if len(r.state.Records) == maxRecords {
		r.state.Health.DroppedLimit++
		return
	}
	r.state.Records = append(r.state.Records, value)
}

func (r *recorder) checkpoint(status string) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	r.state.CheckpointAt = now
	r.state.Status = status
	if status != "active" {
		r.state.EndedAt = now
	}
	r.state.Health.DroppedQueue = r.dropped.Load()
	r.state.UnobservedOperations = []string{}
	for _, operation := range operations {
		if r.state.Operations[operation] == 0 {
			r.state.UnobservedOperations = append(r.state.UnobservedOperations, operation)
		}
	}
	data, err := json.Marshal(r.state)
	if err != nil || len(data) > maxBytes {
		r.state.Health.WriteFailures++
		return
	}
	if r.write(r.path, data) != nil {
		r.state.Health.WriteFailures++
	}
}

func (r *recorder) run(ctx context.Context) {
	defer close(r.done)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	timer := time.NewTimer(time.Until(r.end))
	defer timer.Stop()
	r.checkpoint("active")
	status := "interrupted"
loop:
	for {
		select {
		case value := <-r.queue:
			r.consume(value)
		case <-ticker.C:
			r.checkpoint("active")
		case <-timer.C:
			status = "completed"
			break loop
		case <-ctx.Done():
			break loop
		case <-r.stop:
			break loop
		}
	}
	r.acceptance.Lock()
	r.closed.Store(true)
	r.acceptance.Unlock()
	// Drain only the fixed queue capacity; request threads never wait for disk.
	for range queueCapacity {
		select {
		case value := <-r.queue:
			r.consume(value)
		default:
			r.checkpoint(status)
			return
		}
	}
	r.checkpoint(status)
}

func atomicSnapshot(path string, data []byte) error {
	file, err := os.CreateTemp(filepath.Dir(path), ".readiness-observation-*")
	if err != nil {
		return err
	}
	temporary := file.Name()
	defer os.Remove(temporary)
	if _, err = file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	return os.Rename(temporary, path)
}
