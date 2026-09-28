package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"time"

	runParams "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_client/run_service"
	runModel "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_model"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

const (
	cacheDiagnosticBudget        = time.Minute
	cacheDiagnosticCallBudget    = 5 * time.Second
	cacheDiagnosticMaxBytes      = 1 << 20
	cacheDiagnosticMaxTotalBytes = 8 << 20
	cacheDiagnosticMaxErrorBytes = 64 << 10
	cacheDiagnosticMaxRuns       = 6
	cacheDiagnosticMaxPods       = 12
)

// TearDownTest runs before the next SetupTest or TearDownSuite deletes executions.
// Diagnostics are best effort: they must never replace the original assertion failure.
func (s *CacheTestSuite) TearDownTest() {
	if !*runIntegrationTests || !s.T().Failed() {
		return
	}
	namespace := s.namespace
	if s.resourceNamespace != "" {
		namespace = s.resourceNamespace
	}
	if namespace == "" {
		s.T().Log("Cannot collect cache diagnostics: test namespace unavailable")
		return
	}
	dir, err := os.MkdirTemp("", "tmp-cache-diagnostics-")
	if err != nil {
		s.T().Logf("Cannot create cache diagnostics: %v", err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), cacheDiagnosticBudget)
	defer cancel()
	d := cacheDiagnostics{ctx: ctx, dir: dir, namespace: namespace, command: cacheDiagnosticCommand}
	d.write("test.txt", []byte(s.T().Name()))
	ids := append([]string(nil), s.diagnosticRunIDs...)
	full := string(runModel.V2beta1GetRunRequestViewModeFULL)
	if s.runClient != nil && s.diagnosticRecurringRunID != "" {
		filter := cacheDiagnosticRecurringFilter(s.diagnosticRecurringRunID)
		params := runParams.NewRunServiceListRunsParamsWithTimeout(d.timeout()).WithNamespace(&namespace).WithFilter(&filter).WithPageSize(int32Pointer(cacheDiagnosticMaxRuns)).WithView(&full)
		runs, _, _, err := s.runClient.List(params)
		d.recordJSON("recurring-runs.json", runs, err)
		for _, run := range runs {
			ids = append(ids, run.RunID)
		}
	}
	d.collect(ids, func(id string, timeout time.Duration) (*runModel.V2beta1Run, error) {
		if s.runClient == nil {
			return nil, fmt.Errorf("run client unavailable")
		}
		return s.runClient.Get(runParams.NewRunServiceGetRunParamsWithTimeout(timeout).WithRunID(id).WithView(&full))
	})
	s.T().Logf("Cache failure diagnostics: %s", dir)
}

func cacheDiagnosticRecurringFilter(id string) string {
	data, _ := json.Marshal(map[string]interface{}{"predicates": []map[string]string{{"key": "recurring_run_id", "operation": "EQUALS", "string_value": id}}})
	return string(data)
}
func int32Pointer(value int32) *int32 { return &value }

type cacheDiagnostics struct {
	ctx                      context.Context
	dir, namespace           string
	command                  func(context.Context, ...string) ([]byte, []byte, error)
	writtenBytes, errorBytes int
}

func (d *cacheDiagnostics) timeout() time.Duration {
	deadline, _ := d.ctx.Deadline()
	return min(cacheDiagnosticCallBudget, time.Until(deadline))
}

func (d *cacheDiagnostics) exhausted() bool {
	return d.ctx.Err() != nil || d.writtenBytes >= cacheDiagnosticMaxTotalBytes-cacheDiagnosticMaxErrorBytes
}

func (d *cacheDiagnostics) write(name string, data []byte) {
	limit := max(0, min(cacheDiagnosticMaxBytes, cacheDiagnosticMaxTotalBytes-cacheDiagnosticMaxErrorBytes-d.writtenBytes))
	truncated := len(data) > limit
	if truncated {
		data = data[:limit]
	}
	if err := os.WriteFile(filepath.Join(d.dir, name), data, 0600); err != nil {
		d.problem("write %s: %v", name, err)
	} else {
		d.writtenBytes += len(data)
	}
	if truncated {
		d.problem("%s truncated at %d bytes (per-file or total output limit)", name, limit)
	}
}

func (d *cacheDiagnostics) problem(format string, args ...interface{}) {
	limit := min(cacheDiagnosticMaxErrorBytes-d.errorBytes, cacheDiagnosticMaxTotalBytes-d.writtenBytes)
	if limit <= 0 {
		return
	}
	data := []byte(fmt.Sprintf(format+"\n", args...))
	if len(data) > limit {
		const marker = "\nDiagnostic errors truncated.\n"
		if limit >= len(marker) {
			data = append(data[:limit-len(marker)], marker...)
		} else {
			data = data[:limit]
		}
	}
	file, err := os.OpenFile(filepath.Join(d.dir, "errors.txt"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err == nil {
		defer file.Close()
		n, _ := file.Write(data)
		d.errorBytes += n
		d.writtenBytes += n
	}
}

func (d *cacheDiagnostics) recordJSON(name string, value interface{}, err error) {
	if err != nil {
		d.problem("%s: %v", name, err)
		return
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		d.problem("%s: %v", name, err)
		return
	}
	d.write(name, data)
}

func (d *cacheDiagnostics) capture(name string, args ...string) []byte {
	if d.exhausted() {
		d.problem("%s: diagnostic time or output budget exhausted", name)
		return nil
	}
	ctx, cancel := context.WithTimeout(d.ctx, cacheDiagnosticCallBudget)
	defer cancel()
	args = append([]string{"--namespace", d.namespace, "--request-timeout=5s"}, args...)
	data, stderr, err := d.command(ctx, args...)
	d.write(name, data)
	if len(stderr) > 0 {
		d.problem("%s stderr: %s", name, stderr)
	}
	if err != nil {
		d.problem("%s: %v", name, err)
	}
	return data
}

func (d *cacheDiagnostics) collect(ids []string, getRun func(string, time.Duration) (*runModel.V2beta1Run, error)) {
	if d.namespace == "" {
		d.problem("test namespace unavailable")
		return
	}
	var pods []corev1.Pod
	captureState := func(prefix, selector string) {
		// Keep status before a large full spec can consume the output budget.
		d.capture(prefix+"-workflow-status.txt", "get", "workflows.argoproj.io", "-l", selector, "-o",
			`jsonpath={range .items[*]}{.metadata.name}{"\n"}{.status}{"\n"}{end}`)
		// Pod discovery is independent of workflow availability and JSON decoding.
		data := d.capture(prefix+"-pods.json", "get", "pods", "-l", selector, "-o", "json")
		var list corev1.PodList
		if err := json.Unmarshal(data, &list); err != nil {
			d.problem("decode pods: %v", err)
		} else {
			pods = append(pods, list.Items...)
		}
		d.capture(prefix+"-workflows.json", "get", "workflows.argoproj.io", "-l", selector, "-o", "json")
	}
	seen := map[string]bool{}
	for _, id := range ids {
		if id == "" || len(validation.IsValidLabelValue(id)) != 0 || seen[id] {
			continue
		}
		if len(seen) == cacheDiagnosticMaxRuns || d.exhausted() {
			d.problem("run, time or output limit reached")
			break
		}
		seen[id] = true
		prefix := fmt.Sprintf("run-%d", len(seen))
		run, err := getRun(id, d.timeout())
		d.recordJSON(prefix+".json", run, err)
		captureState(prefix, "pipeline/runid="+id)
	}
	if len(seen) == 0 && !d.exhausted() {
		// This serial suite deletes all runs before each test. If create/list failed,
		// retain its labeled pods even when no API run ID was available to track.
		captureState("namespace", "pipeline/runid")
	}
	sort.SliceStable(pods, func(i, j int) bool {
		finished := func(p corev1.Pod) bool {
			return p.Status.Phase == corev1.PodSucceeded || p.Status.Phase == corev1.PodFailed
		}
		return !finished(pods[i]) && finished(pods[j])
	})
	if len(pods) > cacheDiagnosticMaxPods {
		d.problem("pod limit reached: retaining %d of %d pods, unfinished first", cacheDiagnosticMaxPods, len(pods))
		pods = pods[:cacheDiagnosticMaxPods]
	}
	// Save state and events before spending the remaining budget on container logs.
	for i, pod := range pods {
		if d.exhausted() {
			break
		}
		if pod.UID != "" {
			d.capture(fmt.Sprintf("pod-%d-events.json", i), "get", "events", "--field-selector", "involvedObject.uid="+string(pod.UID), "-o", "json")
		}
	}
	for i, pod := range pods {
		containers := append(append([]corev1.Container(nil), pod.Spec.InitContainers...), pod.Spec.Containers...)
		statuses := append(append([]corev1.ContainerStatus(nil), pod.Status.InitContainerStatuses...), pod.Status.ContainerStatuses...)
		for j, container := range containers {
			if d.exhausted() {
				return
			}
			if j == 6 {
				d.problem("container limit reached for %s", pod.Name)
				break
			}
			previousLogs := []bool{false}
			for _, status := range statuses {
				if status.Name == container.Name && status.RestartCount > 0 {
					previousLogs = append(previousLogs, true)
					break
				}
			}
			for _, previous := range previousLogs {
				args := []string{"logs", pod.Name, "-c", container.Name, "--timestamps", "--tail=200", "--limit-bytes=65536", fmt.Sprintf("--previous=%t", previous)}
				d.capture(fmt.Sprintf("pod-%d-%s-previous-%t.log", i, container.Name, previous), args...)
			}
		}
	}
}

// Discard excess bytes while allowing the subprocess to drain and exit normally.
type cacheDiagnosticBuffer struct {
	buffer    bytes.Buffer
	limit     int
	truncated bool
}

func (b *cacheDiagnosticBuffer) Write(data []byte) (int, error) {
	n := len(data)
	remaining := b.limit - b.buffer.Len()
	if len(data) > remaining {
		data = data[:remaining]
		b.truncated = true
	}
	_, _ = b.buffer.Write(data)
	return n, nil
}

func (b *cacheDiagnosticBuffer) Bytes() []byte  { return b.buffer.Bytes() }
func (b *cacheDiagnosticBuffer) String() string { return b.buffer.String() }

func cacheDiagnosticCommand(ctx context.Context, args ...string) ([]byte, []byte, error) {
	command := exec.CommandContext(ctx, "kubectl", args...)
	command.WaitDelay = time.Second
	stdout := &cacheDiagnosticBuffer{limit: cacheDiagnosticMaxBytes}
	stderr := &cacheDiagnosticBuffer{limit: 16 << 10}
	command.Stdout, command.Stderr = stdout, stderr
	err := command.Run()
	if stdout.truncated || stderr.truncated {
		err = fmt.Errorf("kubectl output truncated (command error: %v)", err)
	}
	return stdout.Bytes(), stderr.Bytes(), err
}
