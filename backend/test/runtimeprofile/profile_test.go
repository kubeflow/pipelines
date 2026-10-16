// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package runtimeprofile

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSelectionAndDisabledNoop(t *testing.T) {
	t.Setenv("KFP_E2E_PROFILE_DIR", "")
	p := Start("flip_coin.yaml")
	if p != nil {
		t.Fatal("enabled without opt-in")
	}
	p.Phase("disabled")()
	p.Capture("none", nil)
	p.Metrics("before")
	p.Finish()
	t.Setenv("KFP_E2E_PROFILE_DIR", t.TempDir())
	if Start("unselected.yaml") != nil {
		t.Fatal("profiled unselected fixture")
	}
	for _, fixture := range []string{"flip_coin.yaml", "collected_parameters.yaml", "if_else_with_oneof_artifacts.yaml"} {
		if Start(fixture) == nil {
			t.Fatal("missing selected fixture", fixture)
		}
	}
}

func TestCaptureBeforeCleanupPreservesStatusAndLimitsLogs(t *testing.T) {
	p := &Profile{Directory: t.TempDir()}
	calls := []string{}
	p.command = func(ctx context.Context, args ...string) ([]byte, error) {
		if _, ok := ctx.Deadline(); !ok {
			t.Fatal("capture must have deadline")
		}
		call := strings.Join(args, " ")
		calls = append(calls, call)
		switch {
		case strings.Contains(call, "get workflows"):
			return []byte(`{"items":[{"metadata":{"name":"wf1","creationTimestamp":"2026-01-01T00:00:00Z"},"spec":{"secret":"do-not-copy"},"status":{"phase":"Succeeded","nodes":{"n":{"startedAt":"2026-01-01T00:00:01Z"}}}}]}`), nil
		case strings.Contains(call, "get pods"):
			return []byte(`{"items":[{"metadata":{"name":"pod1","uid":"pod-uid"},"spec":{"env":"do-not-copy"},"status":{"containerStatuses":[{"state":{"terminated":{"startedAt":"2026-01-01T00:00:02Z"}}}]}}]}`), nil
		case strings.Contains(call, "get events"):
			if !strings.Contains(call, "-n user-namespace") || !strings.Contains(call, "involvedObject.uid=pod-uid") {
				t.Fatal(call)
			}
			return []byte(`{"items":[{"reason":"Scheduled","type":"Normal","message":"do-not-copy","firstTimestamp":"2026-01-01T00:00:01Z","lastTimestamp":"2026-01-01T00:00:02Z","count":1,"series":{"count":2,"lastObservedTime":"2026-01-01T00:00:03Z"},"metadata":{"annotations":{"secret":"do-not-copy"}}}]}`), nil
		case strings.Contains(call, "logs"):
			if !strings.Contains(call, "--timestamps=true") || !strings.Contains(call, "--limit-bytes=") {
				t.Fatal("unbounded or untimestamped logs")
			}
			return []byte("2026-01-01T00:00:02Z hello"), nil
		default:
			t.Fatalf("unexpected command %s", call)
		}
		return nil, nil
	}
	p.Phase("upload")()
	p.Capture("user-namespace", []string{"run-id"})
	p.Finish()
	if len(calls) != 4 {
		t.Fatal(calls)
	}
	if !strings.Contains(calls[0], "pipeline/runid=run-id") || !strings.Contains(calls[1], "workflows.argoproj.io/workflow=wf1") {
		t.Fatal(calls)
	}
	for _, name := range []string{"run-0-workflows.json", "run-0-0-pods.json"} {
		data, err := os.ReadFile(filepath.Join(p.Directory, name))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "do-not-copy") {
			t.Fatal("copied object spec")
		}
		if !strings.Contains(string(data), "startedAt") {
			t.Fatal("lost timing status")
		}
	}
	events, _ := os.ReadFile(filepath.Join(p.Directory, "run-0-0-pod-0-events.json"))
	if strings.Contains(string(events), "do-not-copy") || !strings.Contains(string(events), "Scheduled") || !strings.Contains(string(events), "lastObservedTime") {
		t.Fatal(string(events))
	}
	data, _ := os.ReadFile(filepath.Join(p.Directory, "timings.json"))
	var report Profile
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	if len(report.Phases) != 2 || report.Phases[1].Name != "profile_capture" {
		t.Fatal(report.Phases)
	}
}

func TestCaptureFailureIsDiagnosticAndStillWritesTimings(t *testing.T) {
	p := &Profile{Directory: t.TempDir(), command: func(context.Context, ...string) ([]byte, error) { return nil, errors.New("fixture unavailable") }}
	p.Capture("namespace", []string{"run"})
	p.Finish()
	if len(p.Errors) == 0 {
		t.Fatal("missing diagnostic")
	}
	if _, err := os.Stat(filepath.Join(p.Directory, "timings.json")); err != nil {
		t.Fatal(err)
	}
}

func TestEvidenceLimitsKeepFinalReport(t *testing.T) {
	p := &Profile{Directory: t.TempDir(), bytesWritten: 32 << 20}
	p.save("large.log", []byte("too large"))
	p.Finish()
	if len(p.Errors) != 1 {
		t.Fatal(p.Errors)
	}
	if _, err := os.Stat(filepath.Join(p.Directory, "large.log")); !os.IsNotExist(err) {
		t.Fatal("saved beyond limit")
	}
	if _, err := os.Stat(filepath.Join(p.Directory, "timings.json")); err != nil {
		t.Fatal(err)
	}
}

func TestMetricsAreOptInAndRecordedSeparately(t *testing.T) {
	p := &Profile{Directory: t.TempDir()}
	t.Setenv("KFP_E2E_PROFILE_METRICS_URL", "")
	p.Metrics("before")
	if len(p.Phases) != 0 {
		t.Fatal("metrics enabled by default")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("grpc_server_handling_seconds_count 1\n"))
	}))
	defer server.Close()
	t.Setenv("KFP_E2E_PROFILE_METRICS_URL", server.URL)
	p.Metrics("before")
	if len(p.Phases) != 1 || p.Phases[0].Name != "profile_metrics_before" {
		t.Fatal(p.Phases)
	}
	if _, err := os.Stat(filepath.Join(p.Directory, "metrics-before.txt")); err != nil {
		t.Fatal(err)
	}
}

func TestEmptyMatchesAreMarkedIncomplete(t *testing.T) {
	for _, empty := range []string{"workflows", "pods"} {
		t.Run(empty, func(t *testing.T) {
			p := &Profile{Directory: t.TempDir(), command: func(_ context.Context, args ...string) ([]byte, error) {
				if args[1] == empty {
					return []byte(`{"items":[]}`), nil
				}
				return []byte(`{"items":[{"metadata":{"name":"wf1"},"status":{}}]}`), nil
			}}
			p.Capture("namespace", []string{"run"})
			if !strings.Contains(strings.Join(p.Errors, ";"), "no matching "+empty) {
				t.Fatal(p.Errors)
			}
		})
	}
}
