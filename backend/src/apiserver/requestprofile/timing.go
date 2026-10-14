// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package requestprofile provides opt-in, payload-free runtime RPC timings.
package requestprofile

import (
	"context"
	"encoding/json"
	"os"
	"regexp"
	"sync"
	"time"

	"github.com/golang/glog"
	"google.golang.org/grpc/metadata"
)

const MetadataKey = "x-kfp-runtime-profile-id"
const Environment = "KFP_RUNTIME_PROFILE_TIMINGS"

type Phase string

const (
	Authorization       Phase = "authorization"
	AuthRunLookup       Phase = "auth_run_lookup"
	TokenReview         Phase = "token_review"
	SubjectAccessReview Phase = "subject_access_review"
	KubernetesThrottle  Phase = "kubernetes_rate_limit_wait"
	KubernetesHTTP      Phase = "kubernetes_http_roundtrip"
	TaskRead            Phase = "task_read"
	TaskCreate          Phase = "task_create_resource"
	TaskUpdate          Phase = "task_update_resource"
	TaskChildren        Phase = "task_children"
	ParentValidation    Phase = "parent_validation"
	RunRead             Phase = "run_read"
)

type key struct{}
type record struct {
	CorrelationID string            `json:"correlation_id"`
	Method        string            `json:"method"`
	TotalMS       float64           `json:"total_ms"`
	PhasesMS      map[Phase]float64 `json:"phases_ms"`
}
type timing struct {
	mu      sync.Mutex
	record  record
	started time.Time
}

var correlationPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func Enabled() bool { return os.Getenv(Environment) == "true" }

// Start emits exactly one bounded record for an explicitly profiled RPC. The
// random correlation ID is not an identity or authorization input. No request,
// run/task ID, credentials, error payload, or SQL is recorded.
func Start(ctx context.Context, method string) (context.Context, func()) {
	return start(ctx, method, Enabled(), func(data []byte) { glog.Infof("KFP_RUNTIME_SERVER_TIMING %s", data) })
}

func start(ctx context.Context, method string, enabled bool, emit func([]byte)) (context.Context, func()) {
	noop := func() {}
	if !enabled || ctx == nil {
		return ctx, noop
	}
	switch method {
	case "GetRun", "GetTask", "CreateTask", "UpdateTask", "UpdateTasksBulk", "ListTasks":
	default:
		return ctx, noop
	}
	values := metadata.ValueFromIncomingContext(ctx, MetadataKey)
	if len(values) != 1 || !correlationPattern.MatchString(values[0]) {
		return ctx, noop
	}
	t := &timing{started: time.Now(), record: record{CorrelationID: values[0], Method: method, PhasesMS: make(map[Phase]float64)}}
	var once sync.Once
	return context.WithValue(ctx, key{}, t), func() {
		once.Do(func() {
			t.mu.Lock()
			t.record.TotalMS = float64(time.Since(t.started)) / float64(time.Millisecond)
			data, _ := json.Marshal(t.record)
			t.mu.Unlock()
			emit(data)
		})
	}
}

// StartPhase records inclusive elapsed time, summed across repeated calls.
// Nested phases overlap: do not add authorization to its lookup/review phases,
// or add Kubernetes throttle/HTTP time again to the surrounding review call.
func StartPhase(ctx context.Context, phase Phase) func() {
	if ctx == nil {
		return func() {}
	}
	t, _ := ctx.Value(key{}).(*timing)
	if t == nil {
		return func() {}
	}
	switch phase {
	case Authorization, AuthRunLookup, TokenReview, SubjectAccessReview, KubernetesThrottle, KubernetesHTTP, TaskRead, TaskCreate, TaskUpdate, TaskChildren, ParentValidation, RunRead:
	default:
		return func() {}
	}
	started := time.Now()
	return func() {
		t.mu.Lock()
		t.record.PhasesMS[phase] += float64(time.Since(started)) / float64(time.Millisecond)
		t.mu.Unlock()
	}
}
