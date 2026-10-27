package poc

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type stoppingPodClient struct{ recordingPodClient }

func (c *stoppingPodClient) Update(_ context.Context, pod *corev1.Pod, _ metav1.UpdateOptions) (*corev1.Pod, error) {
	if pod.Spec.ActiveDeadlineSeconds == nil || *pod.Spec.ActiveDeadlineSeconds != 1 {
		return nil, fmt.Errorf("cancellation must shorten the active deadline")
	}
	stopped := pod.DeepCopy()
	stopped.Status.Phase = corev1.PodFailed
	stopped.Status.Reason = "DeadlineExceeded"
	c.pods[pod.Name] = stopped
	return stopped.DeepCopy(), nil
}

type scheduledDuringCancellation struct {
	stoppingPodClient
	reads int
}

func (c *scheduledDuringCancellation) startPod(name string) {
	pod := c.pods[name]
	pod.Spec.NodeName = "worker"
	pod.Status.StartTime = &metav1.Time{Time: time.Now()}
	pod.Status.Phase, pod.ResourceVersion = corev1.PodRunning, "started"
}

func (c *scheduledDuringCancellation) Get(ctx context.Context, name string, options metav1.GetOptions) (*corev1.Pod, error) {
	c.reads++
	if c.reads == 2 {
		c.startPod(name)
	}
	return c.recordingPodClient.Get(ctx, name, options)
}

func (c *scheduledDuringCancellation) Delete(ctx context.Context, name string, options metav1.DeleteOptions) error {
	if c.reads == 1 {
		c.startPod(name)
		return apierrors.NewConflict(schema.GroupResource{Resource: "pods"}, name, fmt.Errorf("Pod was scheduled concurrently"))
	}
	return c.recordingPodClient.Delete(ctx, name, options)
}

func TestKubernetesCancellationDoesNotMislabelConcurrentlyStartedWork(t *testing.T) {
	client := &scheduledDuringCancellation{stoppingPodClient: stoppingPodClient{
		recordingPodClient: recordingPodClient{pods: map[string]*corev1.Pod{}},
	}}
	executor := kubernetesExecutor{coreClient: &recordingCoreClient{podClient: client}}
	reference, err := executor.Submit(context.Background(), TaskExecutionRequest{
		Identity:  TaskAttemptIdentity{RunID: "run", TaskID: "task", AttemptID: "attempt"},
		Namespace: "ns", Pod: &corev1.Pod{},
	})
	require.NoError(t, err)
	pending := client.pods[attemptWorkloadName(reference.Identity)]
	pending.Spec.NodeName = ""
	pending.Status = corev1.PodStatus{Phase: corev1.PodPending}
	pending.ResourceVersion = "pending"
	canceled, err := executor.Cancel(context.Background(), reference)
	if err != nil {
		canceled, err = executor.Cancel(context.Background(), reference)
	}
	require.NoError(t, err)
	require.Equal(t, CancellationConfirmed, canceled.State)
	require.False(t, canceled.NeverStarted, "started workload logs must be retained rather than replaced with a no-output message")
	require.Zero(t, client.deleteCount)
}

func TestKubernetesUnstartedCancellationSurvivesARecordedReferenceRetry(t *testing.T) {
	client := &recordingPodClient{pods: map[string]*corev1.Pod{}}
	executor := kubernetesExecutor{coreClient: &recordingCoreClient{podClient: client}}
	reference, err := executor.Submit(context.Background(), TaskExecutionRequest{
		Identity:  TaskAttemptIdentity{RunID: "run", TaskID: "task", AttemptID: "attempt"},
		Namespace: "ns", Pod: &corev1.Pod{},
	})
	require.NoError(t, err)
	pod := client.pods[attemptWorkloadName(reference.Identity)]
	pod.Spec.NodeName = ""
	pod.Status = corev1.PodStatus{Phase: corev1.PodPending}
	first, err := executor.Cancel(context.Background(), reference)
	require.NoError(t, err)
	require.True(t, first.NeverStarted)
	data, err := toJSONData(reference)
	require.NoError(t, err)
	var recorded ExecutionReference
	require.NoError(t, fromJSONData(data, &recorded))
	retry, err := executor.Cancel(context.Background(), recorded)
	require.NoError(t, err)
	require.True(t, retry.NeverStarted, "temporary placeholder-log failure must not make cancellation look for logs on a deleted Pod")
	require.Equal(t, CancellationConfirmed, retry.State)
}

type interruptedCancellationConfirmation struct{ recordingPodClient }

func (c *interruptedCancellationConfirmation) Delete(ctx context.Context, name string, options metav1.DeleteOptions) error {
	if err := c.recordingPodClient.Delete(ctx, name, options); err != nil {
		return err
	}
	c.getErr = context.Canceled
	return nil
}

func TestKubernetesCancellationReturnsProofWhenConfirmationIsInterrupted(t *testing.T) {
	client := &interruptedCancellationConfirmation{recordingPodClient: recordingPodClient{pods: map[string]*corev1.Pod{}}}
	executor := kubernetesExecutor{coreClient: &recordingCoreClient{podClient: client}}
	reference, err := executor.Submit(context.Background(), TaskExecutionRequest{
		Identity:  TaskAttemptIdentity{RunID: "run", TaskID: "task", AttemptID: "attempt"},
		Namespace: "ns", Pod: &corev1.Pod{},
	})
	require.NoError(t, err)
	pod := client.pods[attemptWorkloadName(reference.Identity)]
	pod.Spec.NodeName = ""
	pod.Status = corev1.PodStatus{Phase: corev1.PodPending}
	result, err := executor.Cancel(context.Background(), reference)
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, CancellationUnknown, result.State)
	require.NotNil(t, result.Reference)
	require.Equal(t, true, result.Reference.Data["canceledBeforeStart"])
	client.getErr = nil
	result, err = executor.Cancel(context.Background(), *result.Reference)
	require.NoError(t, err)
	require.True(t, result.NeverStarted)
	require.Equal(t, CancellationConfirmed, result.State)
}

func TestKubernetesCancellationRetainsLogSourceForReplicaRecovery(t *testing.T) {
	for _, running := range []bool{false, true} {
		t.Run(fmt.Sprintf("running=%t", running), func(t *testing.T) {
			client := &stoppingPodClient{recordingPodClient{pods: map[string]*corev1.Pod{}}}
			executor := kubernetesExecutor{coreClient: &recordingCoreClient{podClient: client}, namespace: "ns"}
			reference, err := executor.Submit(context.Background(), TaskExecutionRequest{
				Identity:  TaskAttemptIdentity{RunID: "run", TaskID: "task", AttemptID: "attempt"},
				Namespace: "ns", Pod: &corev1.Pod{},
			})
			require.NoError(t, err)
			if running {
				pod := client.pods[attemptWorkloadName(reference.Identity)]
				pod.Spec.NodeName = "worker-a"
				pod.Status.Phase = corev1.PodRunning
				pod.Status.StartTime = &metav1.Time{Time: time.Now()}
			}
			canceled, err := executor.Cancel(context.Background(), reference)
			require.NoError(t, err)
			require.Equal(t, CancellationConfirmed, canceled.State)
			pod, err := client.Get(context.Background(), attemptWorkloadName(reference.Identity), metav1.GetOptions{})
			require.NoError(t, err, "another replica must still be able to publish logs after cancellation")
			require.Contains(t, []corev1.PodPhase{corev1.PodFailed, corev1.PodSucceeded}, pod.Status.Phase)
			require.Zero(t, client.deleteCount)
			canceled, err = executor.Cancel(context.Background(), reference)
			require.NoError(t, err, "confirmed cancellation is idempotent")
			require.Equal(t, CancellationConfirmed, canceled.State)
		})
	}
}
