package poc

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
)

func TestDockerReadLogsStreamsFullRetainedLogs(t *testing.T) {
	for _, follow := range []bool{false, true} {
		t.Run(map[bool]string{false: "snapshot", true: "follow"}[follow], func(t *testing.T) {
			engine := &fakeDockerEngine{}
			executor := dockerExecutor{command: engine.run}
			reference, err := executor.Submit(context.Background(), dockerTestRequest(t))
			require.NoError(t, err)
			content := strings.Repeat("complete stdout and stderr\n", 10001)
			executor.logCommand = func(_ context.Context, destination io.Writer, args ...string) error {
				expected := []string{"logs", "container-uid"}
				if follow {
					expected = []string{"logs", "--follow", "container-uid"}
				}
				assert.Equal(t, expected, args)
				assert.NotContains(t, args, "--tail")
				_, err := io.WriteString(destination, content)
				return err
			}
			var destination bytes.Buffer
			require.NoError(t, executor.ReadLogs(context.Background(), reference, follow, &destination))
			assert.Equal(t, content, destination.String())
			assert.Zero(t, engine.stopCount)
			assert.Zero(t, engine.removeCount)
		})
	}
}

func TestDockerReadLogsRejectsReplacementAndHonorsContext(t *testing.T) {
	engine := &fakeDockerEngine{}
	executor := dockerExecutor{command: engine.run}
	reference, err := executor.Submit(context.Background(), dockerTestRequest(t))
	require.NoError(t, err)
	calls := 0
	executor.logCommand = func(context.Context, io.Writer, ...string) error { calls++; return nil }
	engine.container.ID = "replacement"
	require.ErrorContains(t, executor.ReadLogs(context.Background(), reference, false, io.Discard), "ID")
	assert.Zero(t, calls)
	engine.container.ID = "container-uid"
	engine.container.Config.Labels[attemptIDLabel] = "replacement-attempt"
	require.ErrorContains(t, executor.ReadLogs(context.Background(), reference, false, io.Discard), "identity mismatch")
	assert.Zero(t, calls)
	engine.container.Config.Labels[attemptIDLabel] = reference.Identity.AttemptID
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	require.ErrorIs(t, executor.ReadLogs(canceled, reference, true, io.Discard), context.Canceled)
	assert.Zero(t, calls)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	started := make(chan struct{})
	executor.logCommand = func(ctx context.Context, destination io.Writer, args ...string) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	}
	done := make(chan error, 1)
	go func() { done <- executor.ReadLogs(ctx, reference, true, io.Discard) }()
	<-started
	cancel()
	require.ErrorIs(t, <-done, context.Canceled)
	assert.Zero(t, engine.stopCount)
	assert.Zero(t, engine.removeCount)
}

type loggingPodClient struct {
	*recordingPodClient
	logClient  typedcorev1.PodInterface
	options    *corev1.PodLogOptions
	beforeLogs func()
}

func (c *loggingPodClient) GetLogs(name string, options *corev1.PodLogOptions) *rest.Request {
	c.options = options.DeepCopy()
	if c.beforeLogs != nil {
		c.beforeLogs()
	}
	return c.logClient.GetLogs(name, options)
}

func kubernetesLogFixture(t *testing.T, handler http.HandlerFunc) (kubernetesExecutor, ExecutionReference, *loggingPodClient) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := kubernetes.NewForConfig(&rest.Config{Host: server.URL})
	require.NoError(t, err)
	podClient := &loggingPodClient{
		recordingPodClient: &recordingPodClient{pods: map[string]*corev1.Pod{}},
		logClient:          client.CoreV1().Pods("execution"),
	}
	executor := kubernetesExecutor{coreClient: &recordingCoreClient{podClient: podClient}}
	identity := TaskAttemptIdentity{RunID: "run", TaskID: "task", AttemptID: "attempt", Namespace: "execution"}
	reference, err := executor.Submit(context.Background(), TaskExecutionRequest{
		Identity: identity, Namespace: "execution",
		Pod: &corev1.Pod{Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "main"}}}},
	})
	require.NoError(t, err)
	return executor, reference, podClient
}

func TestKubernetesReadLogsStreamsFullNamespaceScopedLogs(t *testing.T) {
	for _, follow := range []bool{false, true} {
		t.Run(map[bool]string{false: "snapshot", true: "follow"}[follow], func(t *testing.T) {
			content := strings.Repeat("full launcher execution logs\n", 10001)
			identity := TaskAttemptIdentity{RunID: "run", TaskID: "task", AttemptID: "attempt", Namespace: "execution"}
			executor, reference, client := kubernetesLogFixture(t, func(response http.ResponseWriter, request *http.Request) {
				assert.Equal(t, "/api/v1/namespaces/execution/pods/"+attemptWorkloadName(identity)+"/log", request.URL.Path)
				assert.Equal(t, follow, request.URL.Query().Get("follow") == "true")
				assert.Equal(t, "main", request.URL.Query().Get("container"))
				_, _ = io.WriteString(response, content)
			})
			var destination bytes.Buffer
			require.NoError(t, executor.ReadLogs(context.Background(), reference, follow, &destination))
			assert.Equal(t, content, destination.String())
			assert.Equal(t, follow, client.options.Follow)
			assert.Nil(t, client.options.TailLines)
			assert.Nil(t, client.options.LimitBytes)
			assert.Zero(t, client.deleteCount)
		})
	}
}

func TestKubernetesReadLogsRejectsReplacementAndNamespaceMismatch(t *testing.T) {
	executor, reference, client := kubernetesLogFixture(t, func(response http.ResponseWriter, request *http.Request) {
		_, _ = io.WriteString(response, "replacement logs")
	})
	pod := client.pods[reference.Data["podName"].(string)]
	pod.UID = "replacement"
	require.ErrorContains(t, executor.ReadLogs(context.Background(), reference, false, io.Discard), "UID")
	assert.Nil(t, client.options, "logs must not open before validating the referenced UID")
	pod.UID = "pod-uid-123"
	reference.Data["namespace"] = "other"
	require.ErrorContains(t, executor.ReadLogs(context.Background(), reference, false, io.Discard), "attempt namespace")
	assert.Nil(t, client.options)
	reference.Data["namespace"] = reference.Identity.Namespace
	client.beforeLogs = func() { pod.UID = "replacement-after-opening" }
	var destination bytes.Buffer
	require.ErrorContains(t, executor.ReadLogs(context.Background(), reference, false, &destination), "UID")
	assert.Empty(t, destination.String(), "a replacement race must not publish a different pod's logs")
	assert.Zero(t, client.deleteCount)
}

func TestKubernetesReadLogsHonorsContext(t *testing.T) {
	started := make(chan struct{})
	executor, reference, client := kubernetesLogFixture(t, func(response http.ResponseWriter, request *http.Request) {
		response.WriteHeader(http.StatusOK)
		response.(http.Flusher).Flush()
		close(started)
		<-request.Context().Done()
	})
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	require.ErrorIs(t, executor.ReadLogs(canceled, reference, true, io.Discard), context.Canceled)
	assert.Nil(t, client.options)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- executor.ReadLogs(ctx, reference, true, io.Discard) }()
	<-started
	cancel()
	require.ErrorIs(t, <-done, context.Canceled)
	assert.Zero(t, client.deleteCount)
}

type failingLogWriter struct{}

func (failingLogWriter) Write([]byte) (int, error) { return 0, errors.New("log destination failed") }

func TestReadLogsReportsDestinationErrors(t *testing.T) {
	engine := &fakeDockerEngine{}
	docker := dockerExecutor{command: engine.run, logCommand: func(_ context.Context, destination io.Writer, _ ...string) error {
		_, err := io.WriteString(destination, "logs")
		return err
	}}
	dockerReference, err := docker.Submit(context.Background(), dockerTestRequest(t))
	require.NoError(t, err)
	require.ErrorContains(t, docker.ReadLogs(context.Background(), dockerReference, false, failingLogWriter{}), "log destination failed")
	kubernetes, podReference, _ := kubernetesLogFixture(t, func(response http.ResponseWriter, request *http.Request) { _, _ = io.WriteString(response, "logs") })
	require.ErrorContains(t, kubernetes.ReadLogs(context.Background(), podReference, false, failingLogWriter{}), "log destination failed")
}
