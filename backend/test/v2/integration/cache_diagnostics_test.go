package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	runParams "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_client/run_service"
	runModel "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_model"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

const (
	cacheDiagnosticBudget     = time.Minute
	cacheDiagnosticCallBudget = 5 * time.Second
	cacheDiagnosticMaxBytes   = 1 << 20
	cacheDiagnosticMaxRuns    = 6
	cacheDiagnosticMaxPods    = 12
)

// TearDownTest runs before the next SetupTest or TearDownSuite deletes executions.
// Diagnostics are best effort: they must never replace the original assertion failure.
func (s *CacheTestSuite) TearDownTest() {
	if !s.T().Failed() {
		return
	}
	dir, err := os.MkdirTemp("", "tmp-cache-diagnostics-")
	if err != nil {
		s.T().Logf("Cannot create cache diagnostics: %v", err)
		return
	}
	namespace := s.namespace
	if s.resourceNamespace != "" {
		namespace = s.resourceNamespace
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
	ctx            context.Context
	dir, namespace string
	command        func(context.Context, ...string) ([]byte, error)
}

func (d *cacheDiagnostics) timeout() time.Duration {
	deadline, _ := d.ctx.Deadline()
	return min(cacheDiagnosticCallBudget, time.Until(deadline))
}

func (d *cacheDiagnostics) write(name string, data []byte) {
	if len(data) > cacheDiagnosticMaxBytes {
		data = data[:cacheDiagnosticMaxBytes]
		d.problem("%s truncated at %d bytes", name, cacheDiagnosticMaxBytes)
	}
	if err := os.WriteFile(filepath.Join(d.dir, name), data, 0600); err != nil {
		d.problem("write %s: %v", name, err)
	}
}

func (d *cacheDiagnostics) problem(format string, args ...interface{}) {
	file, err := os.OpenFile(filepath.Join(d.dir, "errors.txt"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err == nil {
		defer file.Close()
		_, _ = fmt.Fprintf(file, format+"\n", args...)
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
	if d.ctx.Err() != nil {
		d.problem("%s: diagnostic time budget exhausted", name)
		return nil
	}
	ctx, cancel := context.WithTimeout(d.ctx, cacheDiagnosticCallBudget)
	defer cancel()
	args = append([]string{"--namespace", d.namespace, "--request-timeout=5s"}, args...)
	data, err := d.command(ctx, args...)
	d.write(name, data)
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
	seen := map[string]bool{}
	for _, id := range ids {
		if id == "" || len(validation.IsValidLabelValue(id)) != 0 || seen[id] {
			continue
		}
		if len(seen) == cacheDiagnosticMaxRuns || d.ctx.Err() != nil {
			d.problem("run or time limit reached")
			break
		}
		seen[id] = true
		prefix := fmt.Sprintf("run-%d", len(seen))
		run, err := getRun(id, d.timeout())
		d.recordJSON(prefix+".json", run, err)
		data := d.capture(prefix+"-workflows.json", "get", "workflows.argoproj.io", "-l", "pipeline/runid="+id, "-o", "json")
		var workflows metav1.PartialObjectMetadataList
		if err := json.Unmarshal(data, &workflows); err != nil {
			d.problem("decode workflows for %s: %v", id, err)
			continue
		}
		for i, workflow := range workflows.Items {
			if i >= cacheDiagnosticMaxRuns || len(pods) >= cacheDiagnosticMaxPods {
				d.problem("workflow or pod limit reached")
				break
			}
			if len(validation.IsDNS1123Subdomain(workflow.Name)) != 0 || workflow.Name == "" {
				continue
			}
			name := fmt.Sprintf("%s-workflow-%d", prefix, i)
			var list corev1.PodList
			data = d.capture(name+"-pods.json", "get", "pods", "-l", "workflows.argoproj.io/workflow="+workflow.Name, "-o", "json")
			if err := json.Unmarshal(data, &list); err != nil {
				d.problem("decode pods: %v", err)
				continue
			}
			for _, pod := range list.Items {
				if len(pods) == cacheDiagnosticMaxPods {
					d.problem("pod limit reached")
					break
				}
				pods = append(pods, pod)
			}
		}
	}
	// Save state and events before spending the remaining budget on container logs.
	for i, pod := range pods {
		if pod.UID != "" {
			d.capture(fmt.Sprintf("pod-%d-events.json", i), "get", "events", "--field-selector", "involvedObject.uid="+string(pod.UID), "-o", "json")
		}
	}
	for i, pod := range pods {
		containers := append(append([]corev1.Container(nil), pod.Spec.InitContainers...), pod.Spec.Containers...)
		for j, container := range containers {
			if j == 6 {
				d.problem("container limit reached for %s", pod.Name)
				break
			}
			for _, previous := range []bool{false, true} {
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

func cacheDiagnosticCommand(ctx context.Context, args ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, "kubectl", args...)
	command.WaitDelay = time.Second
	stdout := &cacheDiagnosticBuffer{limit: cacheDiagnosticMaxBytes}
	stderr := &cacheDiagnosticBuffer{limit: 16 << 10}
	command.Stdout, command.Stderr = stdout, stderr
	err := command.Run()
	if err != nil {
		err = fmt.Errorf("kubectl: %w: %s", err, stderr.String())
	}
	if stdout.truncated {
		err = fmt.Errorf("kubectl output truncated at %d bytes (command error: %v)", cacheDiagnosticMaxBytes, err)
	}
	return stdout.Bytes(), err
}
