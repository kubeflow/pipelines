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
	d.command = func(ctx context.Context, args ...string) ([]byte, error) {
		require.Equal(t, []string{"--namespace", "test-namespace", "--request-timeout=5s"}, args[:3])
		deadline, ok := ctx.Deadline()
		require.True(t, ok)
		require.LessOrEqual(t, time.Until(deadline), cacheDiagnosticCallBudget)
		call := strings.Join(args[3:], " ")
		calls = append(calls, call)
		switch {
		case strings.HasPrefix(call, "get workflows"):
			require.Contains(t, call, "pipeline/runid=run-1")
			return []byte(`{"items":[{"metadata":{"name":"test-workflow"}}]}`), nil
		case strings.HasPrefix(call, "get pods"):
			require.Contains(t, call, "workflows.argoproj.io/workflow=test-workflow")
			return []byte(`{"items":[{"metadata":{"name":"executor","uid":"pod-uid"},"spec":{"initContainers":[{"name":"launcher"}],"containers":[{"name":"main"},{"name":"wait"}]},"status":{"phase":"Pending","containerStatuses":[{"name":"main","state":{"waiting":{"reason":"ImagePullBackOff"}}}]}}]}`), nil
		case strings.HasPrefix(call, "get events"):
			require.Contains(t, call, "involvedObject.uid=pod-uid")
			return nil, errors.New("events forbidden")
		case strings.HasPrefix(call, "logs"):
			require.Contains(t, call, "--limit-bytes=65536")
			require.Contains(t, call, "--tail=200")
			if strings.Contains(call, "-c launcher ") {
				return []byte("partial launcher output"), errors.New("log stream failed")
			}
			return []byte("container output"), nil
		}
		t.Fatalf("unexpected command: %s", call)
		return nil, nil
	}
	d.collect([]string{"", "run-1", "run-1", "invalid,value"}, func(id string, timeout time.Duration) (*runModel.V2beta1Run, error) {
		require.Equal(t, "run-1", id)
		require.Positive(t, timeout)
		require.LessOrEqual(t, timeout, cacheDiagnosticCallBudget)
		return nil, errors.New("API unavailable")
	})
	require.Len(t, calls, 9)
	require.True(t, strings.HasPrefix(calls[2], "get events"), "state and events must precede logs")
	data, err := os.ReadFile(filepath.Join(dir, "run-1-workflow-0-pods.json"))
	require.NoError(t, err)
	require.True(t, json.Valid(data))
	require.Contains(t, string(data), "ImagePullBackOff")
	for _, container := range []string{"launcher", "main", "wait"} {
		for _, previous := range []bool{false, true} {
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
	d.command = func(ctx context.Context, args ...string) ([]byte, error) {
		calls++
		<-ctx.Done()
		return nil, ctx.Err()
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
		suite.Run(t, &cacheDiagnosticLifecycleSuite{})
		return
	}
	dir := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestCacheDiagnosticsLifecycle$", "-test.v")
	command.Env = append(os.Environ(), "KFP_CACHE_DIAGNOSTIC_CHILD=1", "TMPDIR="+dir)
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
