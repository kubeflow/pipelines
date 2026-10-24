package poc

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
			require.NoError(t, executor.Cancel(context.Background(), reference))
			pod, err := client.Get(context.Background(), attemptWorkloadName(reference.Identity), metav1.GetOptions{})
			require.NoError(t, err, "another replica must still be able to publish logs after cancellation")
			require.Contains(t, []corev1.PodPhase{corev1.PodFailed, corev1.PodSucceeded}, pod.Status.Phase)
			require.Zero(t, client.deleteCount)
			require.NoError(t, executor.Cancel(context.Background(), reference), "confirmed cancellation is idempotent")
		})
	}
}
