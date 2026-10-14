package main

import (
	"context"
	"encoding/json"
	scheduleutil "github.com/kubeflow/pipelines/backend/src/crd/controller/scheduledworkflow/util"
	swfapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestLifecycleGeneratedWire(t *testing.T) {
	creates, disables := 0, 0
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Error("missing owner auth")
		}
		switch {
		case r.Method == "POST" && r.URL.Path == "/apis/v2beta1/recurringruns":
			creates++
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			if body["namespace"] != "ns" || body["mode"] != "ENABLE" || body["max_concurrency"] != "1" {
				t.Errorf("wrong body %v", body)
			}
			w.Write([]byte(`{"recurring_run_id":"schedule"}`))
		case r.Method == "GET" && r.URL.Path == "/apis/v2beta1/runs":
			w.Write([]byte(`{"runs":[{"run_id":"child","recurring_run_id":"schedule","namespace":"ns","experiment_id":"exp","state":"SUCCEEDED"}]}`))
		case strings.Contains(r.URL.Path, "disable"):
			disables++
			w.Write([]byte(`{}`))
		case r.Method == "GET":
			w.Write([]byte(`{"recurring_run_id":"schedule","status":"DISABLED"}`))
		default:
			t.Errorf("unexpected route %s %s", r.Method, r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer s.Close()
	result := stage(context.Background(), s.URL, "test", Tenant{Name: "owner", Namespace: "ns", ExperimentID: "exp", ServiceAccount: "runner", Fixtures: []Fixture{{PipelineID: "p", VersionID: "v"}}}, "secret", time.Second, time.Millisecond, func(Result) error { return nil })
	if !result.Complete || creates != 1 || disables != 1 {
		t.Fatalf("%+v create%d disable%d", result, creates, disables)
	}
}
func TestCreateDenialNoRetry(t *testing.T) {
	n := 0
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(403)
		w.Write([]byte(`{"code":7,"message":"denied"}`))
	}))
	defer s.Close()
	r := stage(context.Background(), s.URL, "x", Tenant{Name: "owner", Namespace: "ns", ExperimentID: "exp", ServiceAccount: "runner", Fixtures: []Fixture{{PipelineID: "p", VersionID: "v"}}}, "token", time.Second, time.Millisecond, func(Result) error { return nil })
	if r.Complete || n != 1 || r.Failure != "create_failed_or_ambiguous_no_retry" {
		t.Fatalf("%+v calls%d", r, n)
	}
}

func TestActualControllerSchedulesExactlyOneWithinBudget(t *testing.T) {
	now := time.Unix(2000000000, 0).UTC()
	for _, seconds := range []int{30, 300, 600} {
		budget := time.Duration(seconds) * time.Second
		oldStart := metav1.NewTime(now)
		end := metav1.NewTime(now.Add(budget))
		schedule := scheduleutil.NewPeriodicSchedule(&swfapi.PeriodicSchedule{StartTime: &oldStart, EndTime: &end, IntervalSecond: int64(seconds + 60)})
		if got := schedule.GetNextScheduledEpochNoCatchup(nil, now.Unix(), now.Unix()); got != math.MaxInt64 {
			t.Fatalf("oldfixture unexpectedly scheduled:%d", got)
		}
		fixed := metav1.NewTime(firstOccurrenceAnchor(now, budget))
		schedule.StartTime = &fixed
		first := schedule.GetNextScheduledEpochNoCatchup(nil, now.Unix(), now.Unix())
		if first != now.Add(5*time.Second).Unix() {
			t.Fatalf("first=%d want=%d", first, now.Add(5*time.Second).Unix())
		}
		if next := schedule.GetNextScheduledEpochNoCatchup(&first, now.Unix(), first); next != math.MaxInt64 {
			t.Fatalf("unbounded second occurrence:%d", next)
		}
	}
}
