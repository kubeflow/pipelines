package poc

import (
	"context"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubernetesfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestRunDeletionCleansEveryAttemptPodWithUIDPreconditions(t *testing.T) {
	coordinator, item, preparation := attemptFixture(t)
	labels := map[string]string{
		util.LabelKeyWorkflowRunId:       item.run.UUID,
		"pipelines.kubeflow.org/task_id": preparation.Task.UUID,
	}
	client := kubernetesfake.NewSimpleClientset(
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "first-attempt", Namespace: "ns", UID: "first-uid", Labels: labels}},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "second-attempt", Namespace: "ns", UID: "second-uid", Labels: labels}},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "unrelated", Namespace: "ns", UID: "other-uid"}},
	)
	coordinator.kubernetesCoreClient = &fakeKubernetesCoreWithClientset{clientSet: client}
	require.NoError(t, coordinator.deleteManagedRunPods(context.Background(), item.run.UUID, "ns"))
	pods, err := client.CoreV1().Pods("ns").List(context.Background(), metav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, pods.Items, 1)
	require.Equal(t, "unrelated", pods.Items[0].Name)
	for _, action := range client.Actions() {
		if deletion, ok := action.(k8stesting.DeleteAction); ok {
			require.NotNil(t, deletion.GetDeleteOptions().Preconditions)
			require.NotNil(t, deletion.GetDeleteOptions().Preconditions.UID)
		}
	}
}

func TestRunDeletionRejectsMismatchedTaskNamespace(t *testing.T) {
	coordinator, item, preparation := attemptFixture(t)
	client := kubernetesfake.NewSimpleClientset(&corev1.Pod{ObjectMeta: metav1.ObjectMeta{
		Name: "mismatched", Namespace: "other", UID: "pod-uid",
		Labels: map[string]string{
			util.LabelKeyWorkflowRunId:       item.run.UUID,
			"pipelines.kubeflow.org/task_id": preparation.Task.UUID,
		},
	}})
	coordinator.kubernetesCoreClient = &fakeKubernetesCoreWithClientset{clientSet: client}
	require.ErrorContains(t, coordinator.deleteManagedRunPods(context.Background(), item.run.UUID, "other"), "ownership")
	_, err := client.CoreV1().Pods("other").Get(context.Background(), "mismatched", metav1.GetOptions{})
	require.NoError(t, err)
}
