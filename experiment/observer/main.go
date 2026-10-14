package main

import (
	"bufio"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type config struct {
	ExpectedSeats      int       `json:"expected_seats"`
	BackgroundBulkList bool      `json:"background_bulk_list"`
	PressureOnly       bool      `json:"pressure_only"`
	Kubeconfig         string    `json:"kubeconfig"`
	Context            string    `json:"context"`
	ClusterUID         string    `json:"cluster_uid"`
	Namespace          string    `json:"namespace"`
	ConfigMap          string    `json:"configmap"`
	TokenFile          string    `json:"token_file"`
	KFPFlowUID         string    `json:"kfp_flow_uid"`
	PriorityUID        string    `json:"priority_uid"`
	CompetitorFlowUID  string    `json:"competitor_flow_uid"`
	Seconds            int       `json:"seconds"`
	RPS                float64   `json:"rps"`
	MaxInflight        int       `json:"max_inflight"`
	StartAt            time.Time `json:"start_at"`
	TimelineFile       string    `json:"timeline_file"`
	windows            []window
	baseline           []metric
}

func (c config) validate() error {
	if c.ExpectedSeats != 0 && c.ExpectedSeats != 4 {
		return errors.New("only declared four-seat scenario allowed")
	}
	if c.BackgroundBulkList && c.Namespace != "apf-background" {
		return errors.New("bulk LIST restricted to isolated background namespace")
	}
	if !strings.HasPrefix(c.Context, "kind-kfp-apf-representative-") || c.ClusterUID == "" || c.Kubeconfig == "" || c.Namespace == "" || c.ConfigMap == "" || c.TokenFile == "" || c.KFPFlowUID == "" || c.PriorityUID == "" || c.CompetitorFlowUID == "" {
		return errors.New("missing explicit isolated context/identity/configuration")
	}
	if c.Seconds < 2 || c.Seconds > 2700 || c.RPS <= 0 || c.RPS > 1000 || c.MaxInflight < 1 || c.MaxInflight > 64 {
		return errors.New("invalid duration/rate/concurrency bounds")
	}
	if c.StartAt.IsZero() || c.TimelineFile == "" {
		return errors.New("explicit shared start_at and timeline_file required")
	}
	if !c.PressureOnly && float64(c.Seconds)*c.RPS > 90000 {
		return errors.New("offer count exceeds90000")
	}
	return nil
}

type sink struct {
	maxBytes  int64
	maxEvents int
	mu        sync.Mutex
	file      *os.File
	err       error
	cancel    context.CancelFunc
	bytes     int64
	events    int
	writer    io.Writer
	syncer    func() error
}

func (s *sink) emit(v any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return
	}
	b, err := json.Marshal(v)
	s.err = err
	if s.err == nil {
		maxEvents, maxBytes := s.maxEvents, s.maxBytes
		if maxEvents == 0 {
			maxEvents = 100000
		}
		if maxBytes == 0 {
			maxBytes = 128 << 20
		}
		if s.events >= maxEvents || s.bytes+int64(len(b))+1 > maxBytes {
			s.err = errors.New("event output bound exceeded")
		} else {
			var n int
			w := s.writer
			if w == nil {
				w = s.file
			}
			n, s.err = w.Write(append(b, '\n'))
			if s.err == nil && n != len(b)+1 {
				s.err = io.ErrShortWrite
			}
			s.events++
			s.bytes += int64(n)
		}
	}
	if s.err != nil {
		s.cancel()
	}
}

func (s *sink) sync() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err == nil {
		f := s.syncer
		if f == nil {
			f = s.file.Sync
		}
		s.err = f()
		if s.err != nil {
			s.cancel()
		}
	}
	return s.err
}
func (s *sink) captureError() error { s.mu.Lock(); defer s.mu.Unlock(); return s.err }
func (s *sink) periodicSync(ctx context.Context, done chan<- struct{}) {
	defer close(done)
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if s.sync() != nil {
				return
			}
		}
	}
}

type window struct {
	Name  string    `json:"name"`
	Kind  string    `json:"kind"`
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

func readTimeline(c config) ([]window, string, error) {
	f, e := os.Open(c.TimelineFile)
	if e != nil {
		return nil, "", errors.New("timeline unavailable")
	}
	defer f.Close()
	b, e := io.ReadAll(io.LimitReader(f, (1<<20)+1))
	if e != nil || len(b) > 1<<20 {
		return nil, "", errors.New("timeline exceeds bound")
	}
	var w []window
	if json.Unmarshal(b, &w) != nil || len(w) < 4 || len(w) > 100 {
		return nil, "", errors.New("invalid timeline")
	}
	at := c.StartAt
	names := map[string]bool{}
	for i, v := range w {
		if v.Name == "" || names[v.Name] || !v.Start.Equal(at) || !v.End.After(v.Start) {
			return nil, "", errors.New("timeline gap, duplicate, or invalid boundary")
		}
		names[v.Name] = true
		if i < len(w)-3 && v.Kind != "load" {
			return nil, "", errors.New("nonload phase before reserved windows")
		}
		at = v.End
	}
	tail := w[len(w)-3:]
	if tail[0].Name != "after_load" || tail[0].Kind != "probe" || tail[1].Name != "drain" || tail[1].Kind != "drain" || tail[2].Name != "after_drain" || tail[2].Kind != "probe" {
		return nil, "", errors.New("missing fixed probe/drain windows")
	}
	if at.Sub(c.StartAt) != time.Duration(c.Seconds)*time.Second {
		return nil, "", errors.New("observer duration does not cover exact full timeline")
	}
	return w, fmt.Sprintf("%x", sha256.Sum256(b)), nil
}
func phaseAt(w []window, at time.Time) string {
	for _, v := range w {
		if !at.Before(v.Start) && at.Before(v.End) {
			return v.Name
		}
	}
	return "outside"
}

type requestKey struct{}
type requestInfo struct {
	ID        int
	Operation string
}
type attempt struct {
	Kind             string    `json:"kind"`
	At               time.Time `json:"at"`
	ID               int       `json:"id"`
	Operation        string    `json:"operation"`
	Status           int       `json:"status"`
	HeaderMS         float64   `json:"header_ms"`
	FlowUID          string    `json:"flow_uid"`
	PriorityUID      string    `json:"priority_uid"`
	TransportError   bool      `json:"transport_error"`
	ClassificationOK bool      `json:"classification_ok"`
}
type observedTransport struct {
	base           http.RoundTripper
	out            *sink
	flow, priority string
	mu             sync.Mutex
	missing        int
	attempts       int
	retries429     int
}

func (t *observedTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	start := time.Now()
	v, e := t.base.RoundTrip(r)
	info, _ := r.Context().Value(requestKey{}).(requestInfo)
	a := attempt{Kind: "http_attempt", At: start.UTC(), ID: info.ID, Operation: info.Operation, HeaderMS: float64(time.Since(start).Microseconds()) / 1000, TransportError: e != nil}
	if v != nil {
		a.Status = v.StatusCode
		a.FlowUID = v.Header.Get("X-Kubernetes-PF-FlowSchema-UID")
		a.PriorityUID = v.Header.Get("X-Kubernetes-PF-PriorityLevel-UID")
		a.ClassificationOK = a.FlowUID == t.flow && a.PriorityUID == t.priority
	}
	t.mu.Lock()
	t.attempts++
	if !a.ClassificationOK {
		t.missing++
	}
	if a.Status == 429 {
		t.retries429++
	}
	t.mu.Unlock()
	t.out.emit(a)
	return v, e
}

type metric struct {
	Name   string            `json:"name"`
	Labels map[string]string `json:"labels"`
	Value  float64           `json:"value"`
}

var metricLine = regexp.MustCompile(`^([a-zA-Z_:][a-zA-Z0-9_:]*)(?:\{(.*)\})?\s+([-+0-9.eE]+)(?:\s+[0-9]+)?$`)
var metricLabel = regexp.MustCompile(`([a-zA-Z_][a-zA-Z0-9_]*)="([^"\\]*)"`)

func wantedMetric(n string) bool {
	for _, prefix := range []string{"apiserver_flowcontrol_", "apiserver_request_duration_seconds", "apiserver_request_total", "apiserver_current_inflight_requests", "etcd_request_duration_seconds", "etcd_disk_wal_fsync_duration_seconds", "etcd_disk_backend_commit_duration_seconds", "process_cpu_seconds_total", "process_resident_memory_bytes"} {
		if strings.HasPrefix(n, prefix) {
			return true
		}
	}
	return false
}
func parseMetrics(b []byte) ([]metric, map[string]bool) {
	out := []metric{}
	found := map[string]bool{}
	for _, line := range strings.Split(string(b), "\n") {
		m := metricLine.FindStringSubmatch(line)
		if m == nil || !wantedMetric(m[1]) {
			continue
		}
		v, e := strconv.ParseFloat(m[3], 64)
		if e != nil {
			continue
		}
		labels := map[string]string{}
		for _, l := range metricLabel.FindAllStringSubmatch(m[2], -1) {
			switch l[1] {
			case "priority_level", "flow_schema", "reason", "execute", "le", "code", "verb", "resource", "subresource", "group", "version", "scope", "request_kind", "type", "operation", "component", "dry_run", "contentType", "cluster", "endpoint":
				labels[l[1]] = l[2]
			}
		}
		out = append(out, metric{m[1], labels, v})
		found[m[1]] = true
	}
	return out, found
}

// Metrics are filtered while reading: unrelated families never enter retained
// evidence. Network, line and retained-byte limits fail closed independently.
const metricsNetworkLimit int64 = 32 << 20
const metricsRetainedLimit = 48 << 10

type metricsFetch struct {
	Status        int    `json:"status"`
	Bytes         int64  `json:"bytes"`
	RetainedBytes int    `json:"retained_bytes"`
	Reason        string `json:"reason"`
	DeadlineMS    int    `json:"deadline_ms"`
}

func fetchMetrics(ctx context.Context, client *http.Client, host string, priority ...string) ([]metric, map[string]bool, metricsFetch) {
	d := metricsFetch{DeadlineMS: 900}
	req, err := http.NewRequestWithContext(ctx, "GET", strings.TrimRight(host, "/")+"/metrics", nil)
	if err != nil {
		d.Reason = "request"
		return nil, nil, d
	}
	resp, err := client.Do(req)
	if err != nil {
		d.Reason = "transport"
		if ctx.Err() != nil {
			d.Reason = "deadline_or_cancel"
		}
		return nil, nil, d
	}
	defer resp.Body.Close()
	d.Status = resp.StatusCode
	if resp.StatusCode != http.StatusOK {
		d.Reason = "http_status"
		return nil, nil, d
	}
	reader := &io.LimitedReader{R: resp.Body, N: metricsNetworkLimit + 1}
	scan := bufio.NewScanner(reader)
	scan.Buffer(make([]byte, 4096), 64<<10)
	totals := map[string]metric{}
	selectedPriority := ""
	if len(priority) > 0 {
		selectedPriority = priority[0]
	}
	for scan.Scan() {
		line := scan.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name := line
		if end := strings.IndexAny(name, "{ \t"); end >= 0 {
			name = name[:end]
		}
		if !wantedMetric(name) {
			continue
		}
		if metricLine.FindStringSubmatch(line) == nil {
			d.Reason = "malformed_selected_metric"
			break
		}
		parsed, _ := parseMetrics([]byte(line))
		if len(parsed) != 1 {
			d.Reason = "malformed_selected_metric"
			break
		}
		m, keep := projectMetric(parsed[0], selectedPriority)
		if !keep {
			continue
		}
		key := metricID(m)
		old := totals[key]
		m.Value += old.Value
		totals[key] = m
		if len(totals) > 256 {
			d.Reason = "retained_series_bound"
			break
		}
	}
	d.Bytes = metricsNetworkLimit + 1 - reader.N

	if d.Bytes > metricsNetworkLimit {
		d.Reason = "network_bound"
	}
	if scan.Err() != nil && d.Reason == "" {
		d.Reason = "read_or_line_bound"
		if ctx.Err() != nil {
			d.Reason = "deadline_or_cancel"
		}
	}
	if d.Reason != "" {
		return nil, nil, d
	}
	out := make([]metric, 0, len(totals))
	found := map[string]bool{}
	for _, m := range totals {
		out = append(out, m)
		found[m.Name] = true
	}
	sort.Slice(out, func(i, j int) bool { return metricID(out[i]) < metricID(out[j]) })
	encoded, _ := json.Marshal(out)
	d.RetainedBytes = len(encoded)
	if d.RetainedBytes > metricsRetainedLimit/2 {
		d.Reason = "retained_bound"
		return nil, nil, d
	}
	d.Reason = "ok"
	return out, found, d
}

// APF measurements retain exact labels for the selected shared priority.
// API request counters aggregate by verb/status, latency histograms by bucket,
// and process/etcd health by family/bucket. These projections are stable across
// snapshots; they intentionally cannot attribute global latency to a resource.
func projectMetric(m metric, priority string) (metric, bool) {
	if strings.HasPrefix(m.Name, "apiserver_flowcontrol_") {
		if priority != "" && m.Labels["priority_level"] != priority {
			return m, false
		}
		keep := false
		for _, name := range []string{"apiserver_flowcontrol_current_inqueue_requests", "apiserver_flowcontrol_current_executing_seats", "apiserver_flowcontrol_current_limit_seats", "apiserver_flowcontrol_nominal_limit_seats", "apiserver_flowcontrol_rejected_requests_total", "apiserver_flowcontrol_dispatched_requests_total", "apiserver_flowcontrol_request_wait_duration_seconds_bucket", "apiserver_flowcontrol_request_wait_duration_seconds_sum", "apiserver_flowcontrol_request_wait_duration_seconds_count"} {
			if m.Name == name {
				keep = true
				break
			}
		}
		return m, keep
	}
	labels := map[string]string{}
	if value, ok := m.Labels["le"]; ok {
		labels["le"] = value
	}
	if m.Name == "apiserver_request_total" {
		for _, key := range []string{"verb", "code"} {
			if value, ok := m.Labels[key]; ok {
				labels[key] = value
			}
		}
	}
	m.Labels = labels
	return m, true
}
func metricID(m metric) string {
	keys := []string{}
	for k := range m.Labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	s := m.Name
	for _, k := range keys {
		s += "|" + k + "=" + m.Labels[k]
	}
	return s
}
func deltas(previous, current []metric) ([]metric, bool) {
	old := map[string]float64{}
	for _, m := range previous {
		old[metricID(m)] += m.Value
	}
	totals := map[string]metric{}
	for _, m := range current {
		key := metricID(m)
		previous := totals[key]
		m.Value += previous.Value
		totals[key] = m
	}
	out := []metric{}
	reset := false
	for key, m := range totals {
		if !strings.HasSuffix(m.Name, "_total") && !strings.HasSuffix(m.Name, "_sum") && !strings.HasSuffix(m.Name, "_count") && !strings.HasSuffix(m.Name, "_bucket") {
			continue
		}
		v, ok := old[key]
		if !ok {
			continue
		}
		m.Value -= v
		if m.Value < 0 {
			reset = true
		}
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return metricID(out[i]) < metricID(out[j]) })
	return out, reset
}

type snapshot struct {
	MetricsFetch    metricsFetch `json:"metrics_fetch"`
	Kind            string       `json:"kind"`
	At              time.Time    `json:"at"`
	Index           int          `json:"index"`
	LagMS           float64      `json:"lag_ms"`
	Ready           bool         `json:"ready"`
	Live            bool         `json:"live"`
	Metrics         []metric     `json:"metrics"`
	Delta           []metric     `json:"delta"`
	Missing         []string     `json:"missing"`
	UnknownCounters []string     `json:"unknown_counters"`
	Phase           string       `json:"phase"`
	CounterReset    bool         `json:"counter_reset"`
	PositiveSeats   bool         `json:"positive_seats"`
}

func fetch(ctx context.Context, client *http.Client, base, path string, limit int64) ([]byte, int, error) {
	r, e := http.NewRequestWithContext(ctx, "GET", strings.TrimRight(base, "/")+path, nil)
	if e != nil {
		return nil, 0, e
	}
	v, e := client.Do(r)
	if e != nil {
		return nil, 0, e
	}
	defer v.Body.Close()
	b, e := io.ReadAll(io.LimitReader(v.Body, limit+1))
	if int64(len(b)) > limit {
		return nil, v.StatusCode, errors.New("response bound exceeded")
	}
	return b, v.StatusCode, e
}
func fixedSeats(metrics []metric, priority string, expected int) bool {
	for _, name := range []string{"apiserver_flowcontrol_current_limit_seats", "apiserver_flowcontrol_nominal_limit_seats"} {
		count := 0
		for _, m := range metrics {
			if m.Name == name && m.Labels["priority_level"] == priority {
				count++
				if m.Value != float64(expected) {
					return false
				}
			}
		}
		if count != 1 {
			return false
		}
	}
	return true
}

func collectSnapshot(ctx context.Context, c config, client *http.Client, host, priority string, i int, due time.Time, previous []metric) snapshot {
	s := snapshot{Kind: "snapshot", At: time.Now().UTC(), Index: i, LagMS: float64(time.Since(due).Microseconds()) / 1000, Missing: []string{}}
	// Health probes have their own bounded budgets and start with the scrape.
	// A large metrics response must not consume the readiness probe's deadline.
	type healthResult struct {
		path string
		ok   bool
	}
	health := make(chan healthResult, 2)
	for _, path := range []string{"/readyz", "/livez"} {
		go func(path string) {
			probe, cancel := context.WithTimeout(ctx, 900*time.Millisecond)
			defer cancel()
			_, status, err := fetch(probe, client, host, path, 4096)
			health <- healthResult{path: path, ok: err == nil && status == 200}
		}(path)
	}
	call, stop := context.WithTimeout(ctx, 900*time.Millisecond)
	defer stop()
	metrics, found, diagnostic := fetchMetrics(call, client, host, priority)
	s.MetricsFetch = diagnostic
	if diagnostic.Reason != "ok" {
		s.Missing = append(s.Missing, "metrics_fetch")
	} else {
		s.Metrics = metrics
		for _, required := range []string{"apiserver_flowcontrol_current_inqueue_requests", "apiserver_flowcontrol_current_executing_seats", "apiserver_flowcontrol_current_limit_seats", "apiserver_request_total", "process_cpu_seconds_total", "process_resident_memory_bytes"} {
			if !found[required] {
				s.Missing = append(s.Missing, required)
			}
		}
		for _, name := range []string{"apiserver_flowcontrol_rejected_requests_total", "apiserver_flowcontrol_request_wait_duration_seconds_count"} {
			if !found[name] {
				s.UnknownCounters = append(s.UnknownCounters, name)
			}
		}
		for _, m := range s.Metrics {
			if m.Name == "apiserver_flowcontrol_current_limit_seats" && m.Labels["priority_level"] == priority && m.Value > 0 {
				s.PositiveSeats = true
			}
		}
		if !s.PositiveSeats {
			s.Missing = append(s.Missing, "matched_priority_positive_seats")
		}
		if c.ExpectedSeats != 0 && !fixedSeats(s.Metrics, priority, c.ExpectedSeats) {
			s.Missing = append(s.Missing, "expected_nominal_and_current_seats")
		}
		s.Delta, s.CounterReset = deltas(previous, s.Metrics)
		encoded, _ := json.Marshal([][]metric{s.Metrics, s.Delta})
		if len(encoded) > metricsRetainedLimit {
			s.Missing = append(s.Missing, "retained_snapshot_bound")
			s.Metrics = nil
			s.Delta = nil
		}

	}
	for i := 0; i < 2; i++ {
		result := <-health
		switch result.path {
		case "/readyz":
			s.Ready = result.ok
		case "/livez":
			s.Live = result.ok
		}
	}
	if !s.Ready {
		s.Missing = append(s.Missing, "readyz")
	}
	if !s.Live {
		s.Missing = append(s.Missing, "livez")
	}
	if s.LagMS > 250 {
		s.Missing = append(s.Missing, "sampling_lag")
	}
	s.Phase = phaseAt(c.windows, due)
	return s
}
func observe(ctx context.Context, c config, client *http.Client, host, priority string, out *sink) []snapshot {
	all := []snapshot{}
	previous := c.baseline
	for i := 0; i <= c.Seconds; i++ {
		due := c.StartAt.Add(time.Duration(i) * time.Second)
		if !waitUntil(ctx, due) {
			break
		}
		v := collectSnapshot(ctx, c, client, host, priority, i, due, previous)
		out.emit(v)
		if out.captureError() != nil {
			break
		}
		previous = v.Metrics
		// Full measurements stream to JSONL. Retain compact status only.
		v.Metrics = nil
		v.Delta = nil
		all = append(all, v)
	}
	return all
}
func waitUntil(ctx context.Context, t time.Time) bool {
	d := time.Until(t)
	if d <= 0 {
		return ctx.Err() == nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

type logical struct {
	Kind      string    `json:"kind"`
	ID        int       `json:"id"`
	Operation string    `json:"operation"`
	At        time.Time `json:"at"`
	OfferedAt time.Time `json:"offered_at"`
	LagMS     float64   `json:"lag_ms"`
	ElapsedMS float64   `json:"elapsed_ms"`
	Phase     string    `json:"phase"`
	Outcome   string    `json:"outcome"`
}

func pressurePhase(name string) bool {
	return name == "sustained" || name == "burst1" || name == "burst2" || name == "burst3"
}
func offerAt(c config, id int) (time.Time, bool) {
	if !c.PressureOnly {
		due := c.StartAt.Add(time.Duration(float64(id) * float64(time.Second) / c.RPS))
		return due, id < 90000 && due.Before(c.StartAt.Add(time.Duration(c.Seconds)*time.Second))
	}
	if id >= 210000 {
		return time.Time{}, false
	}
	for _, w := range c.windows {
		if !pressurePhase(w.Name) {
			continue
		}
		n := int(math.Ceil(w.End.Sub(w.Start).Seconds() * c.RPS))
		if id < n {
			return w.Start.Add(time.Duration(float64(id) * float64(time.Second) / c.RPS)), true
		}
		id -= n
	}
	return time.Time{}, false
}

type cappedCompressedWriter struct {
	writer io.Writer
	bytes  int64
}

func (w *cappedCompressedWriter) Write(b []byte) (int, error) {
	if w.bytes+int64(len(b)) > 128<<20 {
		return 0, errors.New("compressed output bound exceeded")
	}
	n, e := w.writer.Write(b)
	w.bytes += int64(n)
	return n, e
}
func compete(ctx context.Context, c config, client kubernetes.Interface, out *sink) []logical {
	results := []logical{}
	var mu sync.Mutex
	sem := make(chan struct{}, c.MaxInflight)
	updateSlot := make(chan struct{}, 1)
	var wg sync.WaitGroup
	interval := time.Duration(float64(time.Second) / c.RPS)
	for id := 0; ; id++ {
		due, ok := offerAt(c, id)
		if !ok || !waitUntil(ctx, due) {
			break
		}
		op := []string{"get", "list", "update"}[id%3]
		r := logical{Kind: "logical", ID: id, Operation: op, At: time.Now().UTC(), OfferedAt: due, LagMS: float64(time.Since(due).Microseconds()) / 1000}
		r.Phase = phaseAt(c.windows, due)
		drop := ""
		if time.Since(due) > interval {
			drop = "scheduler_drop"
		} else {
			select {
			case sem <- struct{}{}:
			default:
				drop = "inflight_drop"
			}
		}
		if drop == "" && op == "update" {
			select {
			case updateSlot <- struct{}{}:
			default:
				drop = "update_busy_drop"
				<-sem
			}
		}
		if drop != "" {
			r.Outcome = drop
			out.emit(r)
			if out.captureError() != nil {
				break
			}
			mu.Lock()
			results = append(results, r)
			mu.Unlock()
			continue
		}
		wg.Add(1)
		go func(r logical) {
			defer wg.Done()
			defer func() { <-sem }()
			if r.Operation == "update" {
				defer func() { <-updateSlot }()
			}
			call, cancel := context.WithTimeout(context.WithValue(ctx, requestKey{}, requestInfo{r.ID, r.Operation}), 10*time.Second)
			defer cancel()
			start := time.Now()
			var e error
			cms := client.CoreV1().ConfigMaps(c.Namespace)
			switch r.Operation {
			case "get":
				_, e = cms.Get(call, c.ConfigMap, metav1.GetOptions{})
			case "list":
				options := metav1.ListOptions{FieldSelector: "metadata.name=" + c.ConfigMap, Limit: 1}
				if c.BackgroundBulkList {
					options = metav1.ListOptions{Limit: 2048}
				}
				var listed *corev1.ConfigMapList
				listed, e = cms.List(call, options)
				if e == nil && c.BackgroundBulkList && (listed.Continue != "" || len(listed.Items) > 2048) {
					e = errors.New("background namespace LIST exceeded fixed bound")
				}
			case "update":
				var cm *corev1.ConfigMap
				cm, e = cms.Get(call, c.ConfigMap, metav1.GetOptions{})
				if e == nil {
					if cm.Annotations == nil {
						cm.Annotations = map[string]string{}
					}
					cm.Annotations["apf-experiment-sequence"] = strconv.Itoa(r.ID)
					_, e = cms.Update(call, cm, metav1.UpdateOptions{})
				}
			}
			r.ElapsedMS = float64(time.Since(start).Microseconds()) / 1000
			r.Outcome = "success"
			if e != nil {
				switch {
				case errors.Is(e, context.DeadlineExceeded):
					r.Outcome = "deadline"
				case apierrors.IsConflict(e):
					r.Outcome = "conflict_409"
				case errors.Is(e, context.Canceled):
					r.Outcome = "canceled"
				default:
					r.Outcome = "api_or_transport_error"
				}
			}
			out.emit(r)
			if out.captureError() != nil {
				return
			}
			mu.Lock()
			results = append(results, r)
			mu.Unlock()
		}(r)
	}
	wg.Wait()
	return results
}
func run(c config, output string) error {
	if e := c.validate(); e != nil {
		return e
	}
	if time.Until(c.StartAt) < 2*time.Second || time.Until(c.StartAt) > time.Minute {
		return errors.New("start_at must allow2s..60s lead time")
	}
	windows, timelineHash, e := readTimeline(c)
	if e != nil {
		return e
	}
	if c.PressureOnly {
		seconds := 0.0
		for _, w := range windows {
			if pressurePhase(w.Name) {
				if w.Kind != "load" {
					return errors.New("pressure only permitted in load windows")
				}
				seconds += w.End.Sub(w.Start).Seconds()
			}
		}
		if seconds <= 0 || seconds > 210 || math.Ceil(seconds*c.RPS) > 210000 {
			return errors.New("pressure offer budget exceeded")
		}
	}
	c.windows = windows
	raw, e := clientcmd.LoadFromFile(c.Kubeconfig)
	if e != nil {
		return errors.New("cannot load dedicated kubeconfig")
	}
	if raw.CurrentContext != c.Context {
		return errors.New("context mismatch")
	}
	base, e := clientcmd.BuildConfigFromFlags("", c.Kubeconfig)
	if e != nil {
		return errors.New("invalid kubeconfig")
	}
	base.QPS = -1
	base.Timeout = 10 * time.Second
	admin, e := kubernetes.NewForConfig(base)
	if e != nil {
		return errors.New("admin client init")
	}
	ctx, cancel := context.WithDeadline(context.Background(), c.StartAt.Add(time.Duration(c.Seconds+12)*time.Second))
	defer cancel()
	system, e := admin.CoreV1().Namespaces().Get(ctx, "kube-system", metav1.GetOptions{})
	if e != nil || string(system.UID) != c.ClusterUID {
		return errors.New("cluster UID mismatch")
	}
	fs, e := admin.FlowcontrolV1().FlowSchemas().List(ctx, metav1.ListOptions{})
	if e != nil {
		return errors.New("policy snapshot unavailable")
	}
	pls, e := admin.FlowcontrolV1().PriorityLevelConfigurations().List(ctx, metav1.ListOptions{})
	if e != nil {
		return errors.New("priority snapshot unavailable")
	}
	priority := ""
	for _, p := range pls.Items {
		if string(p.UID) == c.PriorityUID && p.Spec.Type == "Limited" {
			priority = p.Name
		}
	}
	if priority == "" {
		return errors.New("expected priority missing or exempt")
	}
	matched := 0
	for _, f := range fs.Items {
		if (string(f.UID) == c.KFPFlowUID || string(f.UID) == c.CompetitorFlowUID) && f.Spec.PriorityLevelConfiguration.Name == priority {
			matched++
			if c.KFPFlowUID == c.CompetitorFlowUID {
				matched++
			}
		}
	}
	if matched != 2 {
		return errors.New("flow schemas do not map to same expected limited priority")
	}
	st, e := os.Stat(c.TokenFile)
	if e != nil || st.Mode().Perm()&0077 != 0 {
		return errors.New("token file absent or not private")
	}
	token, e := os.ReadFile(c.TokenFile)
	if e != nil {
		return errors.New("token file read")
	}
	if e = os.Mkdir(output, 0700); e != nil {
		return errors.New("output must be new")
	}
	eventName := "events.jsonl"
	if c.PressureOnly {
		eventName = "events.jsonl.gz"
	}
	f, e := os.OpenFile(filepath.Join(output, eventName), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return e
	}
	defer f.Close()
	out := &sink{file: f, cancel: cancel}
	closeCapture := func() error { return nil }
	if c.PressureOnly {
		compressed := gzip.NewWriter(&cappedCompressedWriter{writer: f})
		out.writer = compressed
		out.maxEvents = 1000000
		out.maxBytes = 512 << 20
		out.syncer = func() error {
			if e := compressed.Flush(); e != nil {
				return e
			}
			return f.Sync()
		}
		closeCapture = func() error {
			if e := compressed.Close(); e != nil {
				return e
			}
			return f.Sync()
		}
		defer closeCapture()
	}
	write := func(name string, v any) error {
		b, e := json.MarshalIndent(v, "", "  ")
		if e != nil {
			return e
		}
		file, err := os.OpenFile(filepath.Join(output, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			cancel()
			return err
		}
		_, err = file.Write(b)
		if err == nil {
			err = file.Sync()
		}
		closeErr := file.Close()
		if err == nil {
			err = closeErr
		}
		if err != nil {
			cancel()
		}
		return err
	}
	if e = write("policies.json", map[string]any{"flowschemas": fs, "priorities": pls, "expected_priority_uid": c.PriorityUID, "cluster_uid": c.ClusterUID}); e != nil {
		return e
	}
	if e = write("timeline.json", windows); e != nil {
		return e
	}
	probe := rest.CopyConfig(base)
	probe.CertData = nil
	probe.KeyData = nil
	probe.CertFile = ""
	probe.KeyFile = ""
	probe.Username = ""
	probe.Password = ""
	probe.BearerToken = strings.TrimSpace(string(token))
	probe.BearerTokenFile = ""
	probe.ExecProvider = nil
	probe.AuthProvider = nil
	probe.RateLimiter = nil
	probe.QPS = -1
	tr := &observedTransport{out: out, flow: c.CompetitorFlowUID, priority: c.PriorityUID}
	probe.WrapTransport = func(rt http.RoundTripper) http.RoundTripper { tr.base = rt; return tr }
	competitor, e := kubernetes.NewForConfig(probe)
	if e != nil {
		return errors.New("competitor client init")
	}
	hc, e := rest.HTTPClientFor(base)
	if e != nil {
		return errors.New("observer client init")
	}
	baselineAt := c.StartAt.Add(-time.Second)
	if !time.Now().Before(baselineAt) {
		return errors.New("missed baseline window; no competitor traffic sent")
	}
	if !waitUntil(ctx, baselineAt) {
		return errors.New("baseline canceled")
	}
	baseline := collectSnapshot(ctx, c, hc, base.Host, priority, -1, baselineAt, nil)
	out.emit(baseline)
	if out.sync() != nil {
		return errors.New("baseline capture failed")
	}
	if !time.Now().Before(c.StartAt) || len(baseline.Missing) > 0 {
		return errors.New("baseline missing or start deadline missed; no competitor traffic sent")
	}
	c.baseline = baseline.Metrics
	if e = write("baseline-ready.json", map[string]any{"captured_at": time.Now().UTC(), "start_at": c.StartAt, "timeline_sha256": timelineHash}); e != nil {
		return e
	}
	syncCtx, stopSync := context.WithCancel(ctx)
	syncDone := make(chan struct{})
	go out.periodicSync(syncCtx, syncDone)
	samples := make(chan []snapshot, 1)
	go func() { samples <- observe(ctx, c, hc, base.Host, priority, out) }()
	logicalResults := compete(ctx, c, competitor, out)
	snapshots := <-samples
	stopSync()
	<-syncDone
	finalSyncErr := out.sync()
	if closeErr := closeCapture(); finalSyncErr == nil {
		finalSyncErr = closeErr
	}
	missing := []string{}
	if len(snapshots) != c.Seconds+1 {
		missing = append(missing, "snapshot_count")
	}
	unknownCounterSamples := 0
	for _, s := range snapshots {
		if len(s.UnknownCounters) > 0 {
			unknownCounterSamples++
		}
	}
	for _, s := range snapshots {
		if len(s.Missing) > 0 || s.CounterReset {
			missing = append(missing, "snapshot_incomplete_or_counter_reset")
			break
		}
	}
	if tr.missing > 0 || tr.attempts == 0 {
		missing = append(missing, "competitor_classification")
	}
	if finalSyncErr != nil {
		missing = append(missing, "event_write")
	}
	summary := map[string]any{"observation_complete": len(missing) == 0, "representative_acceptance_complete": false, "missing": missing, "logical": logicalResults, "snapshot_count": len(snapshots), "timeline_sha256": timelineHash, "baseline_captured_before_start": true, "unknown_counter_samples": unknownCounterSamples, "counter_baselines_complete": len(baseline.UnknownCounters) == 0 && unknownCounterSamples == 0, "raw_http_attempts": tr.attempts, "raw_429": tr.retries429, "classification_failures": tr.missing, "priority_uid": c.PriorityUID, "priority_name": priority, "kfp_live_classification_verified": false, "integration_missing": []string{"live_kfp_header_evidence", "etcd_direct_metrics_if_not_exposed", "apiserver_etcd_container_restart_oom_evidence", "paired_threshold_evaluation"}, "note": "KFP flow identity is externally supplied; this observer verifies policy mapping and competitor headers only"}
	if c.PressureOnly {
		outcomes := map[string]map[string]int{}
		for _, r := range logicalResults {
			if outcomes[r.Phase] == nil {
				outcomes[r.Phase] = map[string]int{}
			}
			outcomes[r.Phase][r.Outcome]++
		}
		delete(summary, "logical")
		summary["pressure_only"] = true
		summary["logical_outcomes_by_phase"] = outcomes
		summary["raw_events_file"] = "events.jsonl.gz"
		summary["offered_total"] = len(logicalResults)
	}
	if e = write("summary.json", summary); e != nil {
		return e
	}
	if len(missing) > 0 {
		return errors.New("incomplete observation; inspect sanitized summary")
	}
	return nil
}
func main() {
	p := flag.String("config", "", "Configuration file")
	out := flag.String("out", "", "New output directory")
	flag.Parse()
	b, e := os.ReadFile(*p)
	if e != nil {
		fmt.Fprintln(os.Stderr, "config read failed")
		os.Exit(1)
	}
	var c config
	if json.Unmarshal(b, &c) != nil {
		fmt.Fprintln(os.Stderr, "invalid config")
		os.Exit(1)
	}
	if e = run(c, *out); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
