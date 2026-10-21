package integration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	api "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	runModel "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_model"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/encoding/protojson"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestCacheDiagnosticsRecurringFilter(t *testing.T) {
	var filter api.Filter
	require.NoError(t, protojson.Unmarshal([]byte(cacheDiagnosticRecurringFilter("schedule-1")), &filter))
	require.Len(t, filter.Predicates, 1)
	require.Equal(t, "recurring_run_id", filter.Predicates[0].Key)
	require.Equal(t, api.Predicate_EQUALS, filter.Predicates[0].Operation)
	require.Equal(t, "schedule-1", filter.Predicates[0].GetStringValue())
}

func TestCacheDiagnosticsContinuesAfterErrors(t *testing.T) {
	dir := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	var calls []string
	d := cacheDiagnostics{ctx: ctx, dir: dir, namespace: "test-namespace"}
	d.command = func(ctx context.Context, args ...string) ([]byte, []byte, error) {
		require.Equal(t, []string{"--namespace", "test-namespace", "--request-timeout=5s"}, args[:3])
		deadline, ok := ctx.Deadline()
		require.True(t, ok)
		require.LessOrEqual(t, time.Until(deadline), cacheDiagnosticCallBudget)
		call := strings.Join(args[3:], " ")
		calls = append(calls, call)
		switch {
		case strings.HasPrefix(call, "get workflows"):
			require.Contains(t, call, "pipeline/runid=run-1")
			return []byte(`{"items":[{"metadata":{"name":"test-workflow"}}]}`), nil, nil
		case strings.HasPrefix(call, "get pods"):
			require.Contains(t, call, "pipeline/runid=run-1")
			return []byte(`{"items":[{"metadata":{"name":"executor","uid":"pod-uid"},"spec":{"initContainers":[{"name":"launcher"}],"containers":[{"name":"main"},{"name":"wait"}]},"status":{"phase":"Pending","initContainerStatuses":[{"name":"launcher","restartCount":1}],"containerStatuses":[{"name":"main","restartCount":1,"state":{"waiting":{"reason":"ImagePullBackOff"}}}]}}]}`), nil, nil
		case strings.HasPrefix(call, "get events"):
			require.Contains(t, call, "involvedObject.uid=pod-uid")
			return nil, nil, errors.New("events forbidden")
		case strings.HasPrefix(call, "logs"):
			require.Contains(t, call, "--limit-bytes=65536")
			require.Contains(t, call, "--tail=200")
			if strings.Contains(call, "-c launcher ") {
				return []byte("partial launcher output"), nil, errors.New("log stream failed")
			}
			return []byte("container output"), nil, nil
		}
		t.Fatalf("unexpected command: %s", call)
		return nil, nil, nil
	}
	d.collect([]string{"", "run-1", "run-1", "invalid,value"}, func(id string, timeout time.Duration) (*runModel.V2beta1Run, error) {
		require.Equal(t, "run-1", id)
		require.Positive(t, timeout)
		require.LessOrEqual(t, timeout, cacheDiagnosticCallBudget)
		return nil, errors.New("API unavailable")
	})
	require.Len(t, calls, 9)
	require.True(t, strings.HasPrefix(calls[3], "get events"), "state and events must precede logs")
	data, err := os.ReadFile(filepath.Join(dir, "run-1-pods.json"))
	require.NoError(t, err)
	require.True(t, json.Valid(data))
	require.Contains(t, string(data), "ImagePullBackOff")
	for _, container := range []string{"launcher", "main", "wait"} {
		for _, previous := range []bool{false, true} {
			if container == "wait" && previous {
				require.NoFileExists(t, filepath.Join(dir, "pod-0-wait-previous-true.log"))
				continue
			}
			data, err := os.ReadFile(filepath.Join(dir, fmt.Sprintf("pod-0-%s-previous-%t.log", container, previous)))
			require.NoError(t, err)
			require.NotEmpty(t, data)
		}
	}
	data, err = os.ReadFile(filepath.Join(dir, "errors.txt"))
	require.NoError(t, err)
	for _, message := range []string{"API unavailable", "events forbidden", "log stream failed"} {
		require.Contains(t, string(data), message)
	}
}

func TestCacheDiagnosticsDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	d := cacheDiagnostics{ctx: ctx, dir: t.TempDir(), namespace: "test"}
	calls := 0
	d.command = func(ctx context.Context, args ...string) ([]byte, []byte, error) {
		calls++
		<-ctx.Done()
		return nil, nil, ctx.Err()
	}
	started := time.Now()
	d.collect([]string{"run-1", "run-2"}, func(string, time.Duration) (*runModel.V2beta1Run, error) { return nil, errors.New("unavailable") })
	require.Less(t, time.Since(started), time.Second)
	require.Equal(t, 1, calls)
	data, err := os.ReadFile(filepath.Join(d.dir, "errors.txt"))
	require.NoError(t, err)
	require.Contains(t, string(data), "deadline exceeded")
}

func TestCacheDiagnosticBuffer(t *testing.T) {
	b := &cacheDiagnosticBuffer{limit: 4}
	n, err := b.Write([]byte("123456"))
	require.NoError(t, err)
	require.Equal(t, 6, n)
	n, err = b.Write([]byte("789"))
	require.NoError(t, err)
	require.Equal(t, 3, n)
	require.Equal(t, "1234", b.String())
	require.True(t, b.truncated)
	b = &cacheDiagnosticBuffer{limit: 4}
	_, err = io.Copy(b, io.LimitReader(strings.NewReader("12345678"), 8))
	require.NoError(t, err)
	require.Equal(t, "1234", b.String())
}

// Run an intentionally failing suite in a subprocess to exercise testify's real
// failure/teardown ordering without marking this regression test as failed.
func TestCacheDiagnosticsLifecycle(t *testing.T) {
	if os.Getenv("KFP_CACHE_DIAGNOSTIC_CHILD") == "1" {
		*runIntegrationTests = true
		suite.Run(t, &cacheDiagnosticLifecycleSuite{})
		return
	}
	dir := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestCacheDiagnosticsLifecycle$", "-test.v")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "kubectl"), []byte("#!/bin/sh\nprintf '{\"items\":[]}'\n"), 0700))
	command.Env = append(os.Environ(), "KFP_CACHE_DIAGNOSTIC_CHILD=1", "TMPDIR="+dir, "PATH="+dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	output, err := command.CombinedOutput()
	require.Error(t, err, "child suite must retain its original failure")
	require.Contains(t, string(output), "original cache assertion")
	data, err := os.ReadFile(filepath.Join(dir, "cleanup-order.txt"))
	require.NoError(t, err, string(output))
	require.Equal(t, "captured before next setup and final cleanup", string(data))
}

type cacheDiagnosticLifecycleSuite struct {
	suite.Suite
	cache  CacheTestSuite
	setups int
}

func (s *cacheDiagnosticLifecycleSuite) SetupTest() {
	s.setups++
	s.cache.SetT(s.T())
	s.cache.namespace = "test"
	files, err := filepath.Glob(filepath.Join(os.TempDir(), "tmp-cache-diagnostics-*", "test.txt"))
	require.NoError(s.T(), err)
	require.Len(s.T(), files, s.setups-1, "previous failed test must be captured before setup cleanup")
}
func (s *cacheDiagnosticLifecycleSuite) TestFirst() {
	require.FailNow(s.T(), "original cache assertion")
}
func (s *cacheDiagnosticLifecycleSuite) TestSecond() {
	require.FailNow(s.T(), "original cache assertion")
}
func (s *cacheDiagnosticLifecycleSuite) TearDownTest() { s.cache.TearDownTest() }
func (s *cacheDiagnosticLifecycleSuite) TearDownSuite() {
	files, err := filepath.Glob(filepath.Join(os.TempDir(), "tmp-cache-diagnostics-*", "test.txt"))
	require.NoError(s.T(), err)
	require.Len(s.T(), files, 2, "last failed test must be captured before final cleanup")
	require.NoError(s.T(), os.WriteFile(filepath.Join(os.TempDir(), "cleanup-order.txt"), []byte("captured before next setup and final cleanup"), 0600))
}

func TestCacheDiagnosticsMissingWorkflowStillCollectsPods(t *testing.T) {
	for _, ids := range [][]string{{"run-1"}, {"", "invalid,value"}} {
		t.Run(fmt.Sprint(ids), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			d := cacheDiagnostics{ctx: ctx, dir: t.TempDir(), namespace: "test"}
			var calls []string
			d.command = func(_ context.Context, args ...string) ([]byte, []byte, error) {
				call := strings.Join(args[3:], " ")
				calls = append(calls, call)
				switch {
				case strings.HasPrefix(call, "get workflows"):
					return nil, []byte("workflow resource is unavailable"), errors.New("workflow discovery failed")
				case strings.HasPrefix(call, "get pods"):
					selector := "pipeline/runid"
					if ids[0] == "run-1" {
						selector += "=run-1"
					}
					require.Equal(t, "get pods -l "+selector+" -o json", call)
					return []byte(`{"items":[{"metadata":{"name":"executor","uid":"executor-uid"},"spec":{"containers":[{"name":"main"}]},"status":{"phase":"Pending"}}]}`), []byte("successful kubectl warning"), nil
				case strings.HasPrefix(call, "get events"):
					return []byte(`{"items":[]}`), nil, nil
				case strings.HasPrefix(call, "logs executor"):
					return []byte("executor evidence"), nil, nil
				default:
					t.Fatalf("unexpected command: %s", call)
					return nil, nil, nil
				}
			}
			d.collect(ids, func(string, time.Duration) (*runModel.V2beta1Run, error) {
				return nil, errors.New("run API unavailable")
			})
			data, err := os.ReadFile(filepath.Join(d.dir, "pod-0-main-previous-false.log"))
			require.NoError(t, err)
			require.Equal(t, "executor evidence", string(data))
			name := "namespace-pods.json"
			if ids[0] == "run-1" {
				name = "run-1-pods.json"
			}
			data, err = os.ReadFile(filepath.Join(d.dir, name))
			require.NoError(t, err)
			require.True(t, json.Valid(data), "successful stderr must not corrupt pod discovery JSON")
			data, err = os.ReadFile(filepath.Join(d.dir, "errors.txt"))
			require.NoError(t, err)
			require.Contains(t, string(data), "workflow resource is unavailable")
			require.Contains(t, string(data), "successful kubectl warning")
			require.Len(t, calls, 5)
		})
	}
}

func TestCacheDiagnosticsPrioritizesUnfinishedPodsBeforeLimit(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	pods := corev1.PodList{}
	for i := range cacheDiagnosticMaxPods + 2 {
		pods.Items = append(pods.Items, corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("completed-%d", i), UID: types.UID(fmt.Sprintf("uid-%d", i))},
			Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "main"}}},
			Status:     corev1.PodStatus{Phase: corev1.PodSucceeded},
		})
	}
	for _, name := range []string{"stalled-first", "stalled-second"} {
		pods.Items = append(pods.Items, corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: name, UID: types.UID(name)},
			Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "main"}}},
			Status:     corev1.PodStatus{Phase: corev1.PodPending},
		})
	}
	data, err := json.Marshal(pods)
	require.NoError(t, err)
	var events, logs []string
	d := cacheDiagnostics{ctx: ctx, dir: t.TempDir(), namespace: "test"}
	d.command = func(_ context.Context, args ...string) ([]byte, []byte, error) {
		call := strings.Join(args[3:], " ")
		switch {
		case strings.HasPrefix(call, "get pods"):
			return data, nil, nil
		case strings.HasPrefix(call, "get events"):
			require.Empty(t, logs, "all events should be saved before any logs")
			events = append(events, call)
		case strings.HasPrefix(call, "logs"):
			logs = append(logs, args[4])
		}
		return []byte(`{"items":[]}`), nil, nil
	}
	d.collect([]string{"run-1"}, func(string, time.Duration) (*runModel.V2beta1Run, error) { return nil, nil })
	require.Len(t, events, cacheDiagnosticMaxPods)
	require.Len(t, logs, cacheDiagnosticMaxPods)
	require.Equal(t, []string{"stalled-first", "stalled-second", "completed-0"}, logs[:3])
	require.NotContains(t, logs, "completed-10")
}

func TestCacheDiagnosticsTotalOutputBudget(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	pods := corev1.PodList{}
	for i := range cacheDiagnosticMaxPods {
		pods.Items = append(pods.Items, corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("executor-%d", i), UID: types.UID(fmt.Sprintf("uid-%d", i))},
			Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "main"}}},
		})
	}
	data, err := json.Marshal(pods)
	require.NoError(t, err)
	calls := 0
	d := cacheDiagnostics{ctx: ctx, dir: t.TempDir(), namespace: "test"}
	d.command = func(_ context.Context, args ...string) ([]byte, []byte, error) {
		calls++
		if args[4] == "pods" {
			return data, nil, nil
		}
		return []byte(strings.Repeat("x", cacheDiagnosticMaxBytes+1)), []byte(strings.Repeat("warning", 1<<14)), nil
	}
	d.collect([]string{"run-1"}, func(string, time.Duration) (*runModel.V2beta1Run, error) { return nil, nil })
	require.Less(t, calls, cacheDiagnosticMaxPods, "stop commands when aggregate budget is consumed")
	before := calls
	d.capture("after-budget.txt", "get", "pods")
	require.Equal(t, before, calls, "exhausted collector must not launch additional commands")
	entries, err := os.ReadDir(d.dir)
	require.NoError(t, err)
	var total int64
	for _, entry := range entries {
		info, err := entry.Info()
		require.NoError(t, err)
		total += info.Size()
		if entry.Name() == "errors.txt" {
			require.LessOrEqual(t, info.Size(), int64(64<<10))
		} else {
			require.LessOrEqual(t, info.Size(), int64(cacheDiagnosticMaxBytes))
		}
	}
	require.Positive(t, total)
	require.LessOrEqual(t, total, int64(cacheDiagnosticMaxTotalBytes))
}

func TestCacheDiagnosticCommandStreamsAndBounds(t *testing.T) {
	for _, tc := range []struct {
		name, script string
		timeout      time.Duration
		wantErr      bool
	}{
		{"separate streams", `printf '{"items":[]}'; printf 'warning' >&2`, 5 * time.Second, false},
		{"stdout cap", "head -c 2097152 /dev/zero", 5 * time.Second, true},
		{"stderr cap", "head -c 2097152 /dev/zero >&2", 5 * time.Second, true},
		{"deadline", "exec sleep 30", 100 * time.Millisecond, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "kubectl"), []byte("#!/bin/sh\n"+tc.script+"\n"), 0700))
			ctx, cancel := context.WithTimeout(context.Background(), tc.timeout)
			defer cancel()
			started := time.Now()
			stdout, stderr, err := cacheDiagnosticCommand(ctx)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.JSONEq(t, `{"items":[]}`, string(stdout))
				require.Equal(t, "warning", string(stderr))
			}
			if strings.HasSuffix(tc.name, "cap") {
				require.ErrorContains(t, err, "truncated")
			}
			if tc.name == "deadline" {
				require.ErrorIs(t, ctx.Err(), context.DeadlineExceeded)
			}
			require.LessOrEqual(t, len(stdout), cacheDiagnosticMaxBytes)
			require.LessOrEqual(t, len(stderr), 16<<10)
			require.Less(t, time.Since(started), 5*time.Second)
		})
	}
}

func TestCacheDiagnosticsLifecycleGuards(t *testing.T) {
	if scenario := os.Getenv("KFP_CACHE_DIAGNOSTIC_GUARD_CHILD"); scenario != "" {
		*runIntegrationTests = scenario != "disabled"
		suite.Run(t, &cacheDiagnosticGuardSuite{scenario: scenario})
		return
	}
	for _, scenario := range []string{"passed", "disabled", "resource-namespace", "missing-namespace"} {
		t.Run(scenario, func(t *testing.T) {
			dir := t.TempDir()
			script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"$TMPDIR/kubectl-calls.txt\"\nprintf '{\"items\":[]}'\n"
			require.NoError(t, os.WriteFile(filepath.Join(dir, "kubectl"), []byte(script), 0700))
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestCacheDiagnosticsLifecycleGuards$", "-test.v")
			command.Env = append(os.Environ(), "KFP_CACHE_DIAGNOSTIC_GUARD_CHILD="+scenario, "TMPDIR="+dir, "PATH="+dir+string(os.PathListSeparator)+os.Getenv("PATH"))
			output, err := command.CombinedOutput()
			if scenario == "passed" {
				require.NoError(t, err, string(output))
			} else {
				require.Error(t, err, string(output))
				require.Contains(t, string(output), "original cache guard assertion")
			}
			files, err := filepath.Glob(filepath.Join(dir, "tmp-cache-diagnostics-*"))
			require.NoError(t, err)
			if scenario == "resource-namespace" {
				require.Len(t, files, 1)
				calls, err := os.ReadFile(filepath.Join(dir, "kubectl-calls.txt"))
				require.NoError(t, err)
				require.Contains(t, string(calls), "--namespace user-namespace ")
				require.NotContains(t, string(calls), "control-plane")
			} else {
				require.Empty(t, files)
				require.NoFileExists(t, filepath.Join(dir, "kubectl-calls.txt"))
			}
		})
	}
}

type cacheDiagnosticGuardSuite struct {
	suite.Suite
	scenario string
}

func (s *cacheDiagnosticGuardSuite) TestAssertion() {
	if s.scenario != "passed" {
		require.FailNow(s.T(), "original cache guard assertion")
	}
}

func (s *cacheDiagnosticGuardSuite) TearDownTest() {
	cache := CacheTestSuite{namespace: "control-plane", resourceNamespace: "user-namespace"}
	if s.scenario == "missing-namespace" {
		cache.namespace, cache.resourceNamespace = "", ""
	}
	cache.SetT(s.T())
	cache.TearDownTest()
}
