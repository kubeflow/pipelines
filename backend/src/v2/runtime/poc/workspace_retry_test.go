package poc

import (
	"context"
	"testing"
	"time"

	"github.com/kubeflow/pipelines/backend/src/v2/driver"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func workspaceRetryFixture(t *testing.T) (pvcWorkspaceProvisioner, *fake.Clientset, *WorkspaceHandle, WorkspaceRequest) {
	t.Helper()
	request := WorkspaceRequest{RunID: "run", RunName: "run", Namespace: "ns"}
	pvc := &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{
		Name: driver.GetWorkspacePVCName(request.RunName), Namespace: request.Namespace,
		UID: "original", Labels: map[string]string{attemptRunLabel: request.RunID},
	}}
	handle, err := workspacePVCReference(pvc, request)
	require.NoError(t, err)
	client := fake.NewSimpleClientset(pvc)
	return pvcWorkspaceProvisioner{coreClient: &fakeKubernetesCoreWithClientset{clientSet: client}}, client, handle, request
}

func TestWorkspaceRetryCannotAdoptOrAttachATerminatingPVC(t *testing.T) {
	provisioner, client, handle, request := workspaceRetryFixture(t)
	pvc, err := client.CoreV1().PersistentVolumeClaims(request.Namespace).Get(context.Background(), handle.Metadata["pvcName"], metav1.GetOptions{})
	require.NoError(t, err)
	now := metav1.Now()
	pvc.DeletionTimestamp, pvc.Finalizers = &now, []string{"fixture/protect"}
	_, err = client.CoreV1().PersistentVolumeClaims(request.Namespace).Update(context.Background(), pvc, metav1.UpdateOptions{})
	require.NoError(t, err)
	_, err = provisioner.Provision(context.Background(), request)
	require.ErrorContains(t, err, "terminating")
	_, err = provisioner.ResolveAttachment(context.Background(), handle, ExecutionTarget{Executor: ExecutorKubernetes, Namespace: request.Namespace})
	require.ErrorContains(t, err, "terminating")
}

func TestWorkspaceCleanupRequiresConfirmedPVCDeletion(t *testing.T) {
	provisioner, client, handle, request := workspaceRetryFixture(t)
	client.PrependReactor("delete", "persistentvolumeclaims", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, nil
	})
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	require.ErrorIs(t, provisioner.Cleanup(ctx, handle), context.DeadlineExceeded)
	require.NoError(t, client.Tracker().Delete(schema.GroupVersionResource{Version: "v1", Resource: "persistentvolumeclaims"}, request.Namespace, handle.Metadata["pvcName"]))
	require.NoError(t, provisioner.Cleanup(context.Background(), handle))
}
