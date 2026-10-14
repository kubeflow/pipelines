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
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func TestMetricDeltasAndMissingSeries(t *testing.T) {
	before, _ := parseMetrics([]byte("apiserver_flowcontrol_rejected_requests_total{priority_level=\"shared\",reason=\"queue-full\"} 4\nprocess_cpu_seconds_total 10\n"))
	after, found := parseMetrics([]byte("apiserver_flowcontrol_rejected_requests_total{priority_level=\"shared\",reason=\"queue-full\"} 9\nprocess_cpu_seconds_total 12\nsecret_metric{token=\"do-not-record\"} 1\n"))
	delta, reset := deltas(before, after)
	if reset || len(delta) != 2 || !found["process_cpu_seconds_total"] {
		t.Fatalf("bad delta evidence: %+v", delta)
	}
	if delta[0].Value != 5 || delta[1].Value != 2 {
		t.Fatal(delta)
	}
	_, reset = deltas(after, before)
	if !reset {
		t.Fatal("counter reset not flagged")
	}
	if len(after) != 2 {
		t.Fatal("unapproved metrics retained")
	}
	if _, ok := found["apiserver_flowcontrol_current_limit_seats"]; ok {
		t.Fatal("missing metric fabricated")
	}
}
func testSink(t *testing.T, cancel context.CancelFunc) *sink {
	t.Helper()
	f, e := os.OpenFile(filepath.Join(t.TempDir(), "events.jsonl"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { f.Close() })
	return &sink{file: f, cancel: cancel}
}
func TestRealWireCompetitorIdentityAndFixedOffers(t *testing.T) {
	var requests atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Header.Get("Authorization") != "Bearer private-credential" {
			t.Error("credential not supplied to wire client")
		}
		time.Sleep(75 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Kubernetes-PF-FlowSchema-UID", "flow")
		w.Header().Set("X-Kubernetes-PF-PriorityLevel-UID", "priority")
		if r.Method == "GET" && r.URL.Path == "/api/v1/namespaces/probe/configmaps" {
			json.NewEncoder(w).Encode(map[string]any{"apiVersion": "v1", "kind": "ConfigMapList", "items": []any{}})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"apiVersion": "v1", "kind": "ConfigMap", "metadata": map[string]any{"name": "target", "namespace": "probe", "resourceVersion": "1"}})
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out := testSink(t, cancel)
	tr := &observedTransport{out: out, flow: "flow", priority: "priority"}
	cfg := &rest.Config{Host: server.URL, BearerToken: "private-credential", QPS: -1, WrapTransport: func(rt http.RoundTripper) http.RoundTripper { tr.base = rt; return tr }}
	client, e := kubernetes.NewForConfig(cfg)
	if e != nil {
		t.Fatal(e)
	}
	c := config{Namespace: "probe", ConfigMap: "target", Seconds: 2, RPS: 30, MaxInflight: 1, StartAt: time.Now().Add(20 * time.Millisecond)}
	results := compete(ctx, c, client, out)
	if len(results) != 60 {
		t.Fatalf("offered=%d want60", len(results))
	}
	drops, success := 0, 0
	for _, r := range results {
		if r.Outcome == "success" {
			success++
		}
		if r.Outcome == "inflight_drop" || r.Outcome == "scheduler_drop" {
			drops++
		}
	}
	if drops == 0 || success == 0 || tr.attempts == 0 || tr.missing != 0 {
		t.Fatalf("drops%d success%d attempts%d missing%d", drops, success, tr.attempts, tr.missing)
	}
	if out.err != nil {
		t.Fatal(out.err)
	}
}
func TestWrongAPFClassificationFailsEvidence(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	out := testSink(t, cancel)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Kubernetes-PF-FlowSchema-UID", "other")
		w.Header().Set("X-Kubernetes-PF-PriorityLevel-UID", "exempt")
		w.WriteHeader(200)
	}))
	defer server.Close()
	tr := &observedTransport{base: http.DefaultTransport, out: out, flow: "expected", priority: "shared"}
	r, _ := http.NewRequestWithContext(ctx, "GET", server.URL, nil)
	v, e := tr.RoundTrip(r)
	if e != nil {
		t.Fatal(e)
	}
	v.Body.Close()
	if tr.missing != 1 {
		t.Fatal("misclassification not recorded")
	}
}
func TestBoundsAndCancellation(t *testing.T) {
	c := config{}
	if c.validate() == nil {
		t.Fatal("empty config accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if waitUntil(ctx, time.Now().Add(time.Hour)) {
		t.Fatal("cancellation ignored")
	}
	c = config{TimelineFile: "timeline.json", Kubeconfig: "k", Context: "kind-kfp-apf-representative-r1", ClusterUID: "uid", Namespace: "ns", ConfigMap: "cm", TokenFile: "file", KFPFlowUID: "kfp", PriorityUID: "pl", CompetitorFlowUID: "probe", Seconds: 2, RPS: 1, MaxInflight: 1, StartAt: time.Now()}
	if e := c.validate(); e != nil {
		t.Fatal(e)
	}
	c.MaxInflight = 65
	if c.validate() == nil {
		t.Fatal("unbounded workers accepted")
	}
}
func TestMetricIdentityAggregation(t *testing.T) {
	before := []metric{{Name: "apiserver_request_total", Labels: map[string]string{"verb": "GET"}, Value: 3}, {Name: "apiserver_request_total", Labels: map[string]string{"verb": "GET"}, Value: 4}}
	after := []metric{{Name: "apiserver_request_total", Labels: map[string]string{"verb": "GET"}, Value: 5}, {Name: "apiserver_request_total", Labels: map[string]string{"verb": "GET"}, Value: 8}}
	d, r := deltas(before, after)
	if r || len(d) != 1 || d[0].Value != 6 {
		t.Fatal(d, r)
	}
}

func TestSnapshotsKeepMissingDataExplicit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" {
			w.Write([]byte("process_cpu_seconds_total 1\n"))
			return
		}
		w.WriteHeader(200)
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out := testSink(t, cancel)
	samples := observe(ctx, config{Seconds: 2, StartAt: time.Now()}, server.Client(), server.URL, "shared", out)
	if len(samples) != 3 {
		t.Fatalf("samples%d", len(samples))
	}
	for _, s := range samples {
		if len(s.Missing) == 0 || s.PositiveSeats || !s.Ready || !s.Live {
			t.Fatalf("missing-data incorrectly represented: %+v", s)
		}
	}
}
func TestOutputBoundCancels(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	out := testSink(t, cancel)
	out.bytes = 128 << 20
	out.emit(map[string]string{"kind": "bounded"})
	if out.err == nil || ctx.Err() == nil {
		t.Fatal("output bound did not stop generation")
	}
}

func TestTimelineCoversProbeAndDrain(t *testing.T) {
	start := time.Now().Add(10 * time.Second)
	windows := []window{{"normal", "load", start, start.Add(2 * time.Second)}, {"after_load", "probe", start.Add(2 * time.Second), start.Add(3 * time.Second)}, {"drain", "drain", start.Add(3 * time.Second), start.Add(5 * time.Second)}, {"after_drain", "probe", start.Add(5 * time.Second), start.Add(6 * time.Second)}}
	p := filepath.Join(t.TempDir(), "timeline.json")
	save := func() {
		b, _ := json.Marshal(windows)
		if e := os.WriteFile(p, b, 0600); e != nil {
			t.Fatal(e)
		}
	}
	save()
	c := config{TimelineFile: p, StartAt: start, Seconds: 6}
	w, h, e := readTimeline(c)
	if e != nil || len(w) != 4 || len(h) != 64 {
		t.Fatal(w, h, e)
	}
	if phaseAt(w, start.Add(4*time.Second)) != "drain" {
		t.Fatal("phase attribution")
	}
	c.Seconds = 2
	if _, _, e = readTimeline(c); e == nil {
		t.Fatal("load-only observer duration accepted")
	}
	c.Seconds = 6
	windows[2].Start = windows[2].Start.Add(time.Second)
	save()
	if _, _, e = readTimeline(c); e == nil {
		t.Fatal("timeline gap accepted")
	}
}

type shortWriter struct{}

func (shortWriter) Write(p []byte) (int, error) { return len(p) - 1, nil }
func TestCaptureWriteAndSyncFailuresStopTraffic(t *testing.T) {
	for _, kind := range []string{"short", "fsync", "event_bound"} {
		t.Run(kind, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			out := testSink(t, cancel)
			switch kind {
			case "short":
				out.writer = shortWriter{}
				out.emit(map[string]string{"kind": "probe"})
			case "fsync":
				out.syncer = func() error { return errors.New("injected fsync") }
				out.sync()
			case "event_bound":
				out.events = 100000
				out.emit(map[string]string{"kind": "probe"})
			}
			if out.captureError() == nil || ctx.Err() == nil {
				t.Fatal("capture failure not canceled")
			}
			b, n := out.bytes, out.events
			out.emit(map[string]string{"kind": "later"})
			if out.bytes != b || out.events != n {
				t.Fatal("capture continued afterfailure")
			}
		})
	}
}
func TestPeriodicSyncCancels(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	out := testSink(t, cancel)
	out.syncer = func() error { return io.ErrClosedPipe }
	done := make(chan struct{})
	go out.periodicSync(ctx, done)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("periodic sync did not stop")
	}
	if ctx.Err() == nil {
		t.Fatal("fsync failure did not cancel")
	}
}
func TestUpdatesAreSingleWriterAndOffersRemainFixed(t *testing.T) {
	var mu sync.Mutex
	version := 1
	conflicts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "GET" && r.URL.Path == "/api/v1/namespaces/probe/configmaps" {
			w.Write([]byte(`{"apiVersion":"v1","kind":"ConfigMapList","items":[]}`))
			return
		}
		if r.Method == "PUT" {
			time.Sleep(100 * time.Millisecond)
			var cm map[string]any
			if err := json.NewDecoder(r.Body).Decode(&cm); err != nil {
				t.Error("invalid JSON request", err)
				http.Error(w, "invalid", 400)
				return
			}
			mu.Lock()
			if cm["metadata"].(map[string]any)["resourceVersion"] != fmt.Sprint(version) {
				conflicts++
			}
			version++
			v := version
			mu.Unlock()
			json.NewEncoder(w).Encode(map[string]any{"apiVersion": "v1", "kind": "ConfigMap", "metadata": map[string]any{"name": "target", "resourceVersion": fmt.Sprint(v)}})
			return
		}
		mu.Lock()
		v := version
		mu.Unlock()
		json.NewEncoder(w).Encode(map[string]any{"apiVersion": "v1", "kind": "ConfigMap", "metadata": map[string]any{"name": "target", "resourceVersion": fmt.Sprint(v)}})
	}))
	defer server.Close()
	client, e := kubernetes.NewForConfig(&rest.Config{Host: server.URL, QPS: -1, ContentConfig: rest.ContentConfig{ContentType: "application/json"}})
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out := testSink(t, cancel)
	r := compete(ctx, config{StartAt: time.Now().Add(10 * time.Millisecond), Seconds: 2, RPS: 90, MaxInflight: 16, Namespace: "probe", ConfigMap: "target"}, client, out)
	drops := 0
	for _, v := range r {
		if v.Outcome == "update_busy_drop" {
			drops++
		}
	}
	if len(r) != 180 || drops == 0 || conflicts != 0 {
		t.Fatalf("offers%d drops%d conflicts%d", len(r), drops, conflicts)
	}
}
func TestRetainedSnapshotsAreCompact(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" {
			w.Write([]byte("process_cpu_seconds_total 1\n"))
		}
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	out := testSink(t, cancel)
	r := observe(ctx, config{Seconds: 2, StartAt: time.Now()}, server.Client(), server.URL, "p", out)
	for _, v := range r {
		if v.Metrics != nil || v.Delta != nil {
			t.Fatal("full snapshot retained")
		}
	}
}

func TestAbsentIdleCountersRemainUnknownNotZero(t *testing.T) {
	metrics := `apiserver_flowcontrol_current_inqueue_requests{priority_level="shared"} 0
apiserver_flowcontrol_current_executing_seats{priority_level="shared"} 0
apiserver_flowcontrol_current_limit_seats{priority_level="shared"} 5
apiserver_request_total{code="200"} 1
process_cpu_seconds_total 1
process_resident_memory_bytes 1
`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" {
			w.Write([]byte(metrics))
		}
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	v := collectSnapshot(ctx, config{}, server.Client(), server.URL, "shared", -1, time.Now(), nil)
	if len(v.Missing) != 0 || len(v.UnknownCounters) != 2 || !v.PositiveSeats {
		t.Fatalf("idle counter handling: %+v", v)
	}
	for _, d := range v.Delta {
		if d.Name == "apiserver_flowcontrol_rejected_requests_total" {
			t.Fatal("missing rejection baseline fabricated")
		}
	}
}
func TestPastStartRejectedBeforeOutput(t *testing.T) {
	c := config{TimelineFile: "not-read", Kubeconfig: "not-read", Context: "kind-kfp-apf-representative-r1", ClusterUID: "u", Namespace: "n", ConfigMap: "c", TokenFile: "t", KFPFlowUID: "f", PriorityUID: "p", CompetitorFlowUID: "g", StartAt: time.Now().Add(-time.Second), Seconds: 4, RPS: 1, MaxInflight: 1}
	out := filepath.Join(t.TempDir(), "absent")
	if run(c, out) == nil {
		t.Fatal("past start accepted")
	}
	if _, e := os.Stat(out); !os.IsNotExist(e) {
		t.Fatal("output created for expiredstart")
	}
}

func TestContextConfigJSONContract(t *testing.T) {
	var c config
	if err := json.Unmarshal([]byte(`{"context":"kind-kfp-apf-representative-test"}`), &c); err != nil {
		t.Fatal(err)
	}
	if c.Context != "kind-kfp-apf-representative-test" {
		t.Fatalf("context not decoded: %q", c.Context)
	}
	encoded, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatal(err)
	}
	if fields["context"] != c.Context {
		t.Fatalf("wrong context JSON key: %s", encoded)
	}
	if _, exists := fields["Context"]; exists {
		t.Fatal("fallback exported-field key leaked")
	}
}

func TestStreamingMetricsLargerThanLegacyBound(t *testing.T) {
	body := strings.Repeat("unrelated_metric{resource=\"irrelevant\"} 1\n", 150000) + "process_cpu_seconds_total 12\napiserver_flowcontrol_current_limit_seats{priority_level=\"shared\"} 4\n"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, body) }))
	defer server.Close()
	got, found, d := fetchMetrics(context.Background(), server.Client(), server.URL)
	if d.Reason != "ok" || d.Bytes <= 4<<20 || len(got) != 2 || !found["apiserver_flowcontrol_current_limit_seats"] || d.RetainedBytes > 512 {
		t.Fatalf("unexpected %+v metrics=%d", d, len(got))
	}
}
func TestStreamingMetricsFailsClosed(t *testing.T) {
	cases := []struct {
		name, body, reason string
		status             int
	}{
		{"malformed", "process_cpu_seconds_total not-a-number\n", "malformed_selected_metric", 200},
		{"line-bound", strings.Repeat("x", 65537), "read_or_line_bound", 200},
		{"network-bound", strings.Repeat("irrelevant 1\n", 3000000), "network_bound", 200},

		{"denied", "secret body", "http_status", 403},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(tc.status); io.WriteString(w, tc.body) }))
			defer server.Close()
			got, _, d := fetchMetrics(context.Background(), server.Client(), server.URL)
			if d.Reason != tc.reason || len(got) != 0 || d.Status != tc.status {
				t.Fatalf("unexpected %+v", d)
			}
		})
	}
}
func TestStreamingMetricsTransportTruncation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "999")
		io.WriteString(w, "process_cpu_seconds_total 1\n")
	}))
	defer server.Close()
	got, _, d := fetchMetrics(context.Background(), server.Client(), server.URL)
	if d.Reason != "read_or_line_bound" || len(got) != 0 {
		t.Fatalf("truncated capture accepted %+v", d)
	}
}

func TestProjectionStableHistogramsDeltasAndBounds(t *testing.T) {
	build := func(n int) string {
		var b strings.Builder
		for i := 0; i < 1000; i++ {
			fmt.Fprintf(&b, "apiserver_request_duration_seconds_bucket{resource=\"r%d\",verb=\"GET\",le=\"0.1\"} %d\n", i, n)
			fmt.Fprintf(&b, "apiserver_request_total{resource=\"r%d\",verb=\"GET\",code=\"200\"} %d\n", i, n)
		}
		b.WriteString("apiserver_flowcontrol_current_limit_seats{priority_level=\"other\"} 100\napiserver_flowcontrol_current_limit_seats{priority_level=\"shared\"} 3\n")
		return b.String()
	}
	var body = build(1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, body) }))
	defer server.Close()
	before, _, d := fetchMetrics(context.Background(), server.Client(), server.URL, "shared")
	if d.Reason != "ok" || len(before) != 3 {
		t.Fatalf("projection failed %+v count%d", d, len(before))
	}
	body = build(2)
	after, _, d := fetchMetrics(context.Background(), server.Client(), server.URL, "shared")
	changes, reset := deltas(before, after)
	if d.Reason != "ok" || reset || len(changes) != 2 || changes[0].Value != 1000 || changes[1].Value != 1000 {
		t.Fatalf("delta mismatch %+v reset%v", changes, reset)
	}
	_, reset = deltas(after, before)
	if !reset {
		t.Fatal("aggregate reset lost")
	}
	if _, ok := changes[0].Labels["le"]; !ok {
		t.Fatal("histogram boundary lost")
	}
	encoded, _ := json.Marshal([][]metric{after, changes})
	if len(encoded) > metricsRetainedLimit {
		t.Fatal("snapshot retention bound exceeded")
	}
	// 1232 baseline/timeline snapshots cap metric payload at58MiB, leaving
	// >69MiB of the128MiB sink for bounded competitor and snapshot metadata.
	if int64(1232*metricsRetainedLimit) > 59<<20 {
		t.Fatal("timeline metric budget exceeded")
	}
}
func TestMetricsDeadlineIsSanitized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	_, _, d := fetchMetrics(ctx, server.Client(), server.URL)
	if d.Reason != "deadline_or_cancel" {
		t.Fatalf("unsafe/incorrect reason %+v", d)
	}
}
