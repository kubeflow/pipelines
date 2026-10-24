package poc

import (
	"context"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/driver"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubernetesfake "k8s.io/client-go/kubernetes/fake"
)

func TestWorkspaceCleanupRecoversAnUnrecordedPVCWithoutCreatingOne(t *testing.T) {
	for _, owner := range []string{"run", "another-run"} {
		t.Run(owner, func(t *testing.T) {
			pvc := &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{
				Name: driver.GetWorkspacePVCName("run"), Namespace: "ns", UID: "pvc-uid",
				Labels: map[string]string{attemptRunLabel: owner},
			}}
			client := kubernetesfake.NewSimpleClientset(pvc)
			coordinator := NewCoordinator(nil, nil, nil, nil, nil, &fakeKubernetesCoreWithClientset{clientSet: client}, nil, false)
			run := &model.Run{
				UUID: "run", K8SName: "run", Namespace: "ns",
				RunDetails: model.RunDetails{WorkspaceProvisioner: util.StringPointer(WorkspaceTypePVC)},
			}
			err := coordinator.cleanupWorkspaceForRun(context.Background(), run, NewRuntimeManifest(ExecutorKubernetes))
			if owner != run.UUID {
				require.Error(t, err, "a replacement owner's PVC must not be deleted")
			} else {
				require.NoError(t, err)
			}
			_, err = client.CoreV1().PersistentVolumeClaims("ns").Get(context.Background(), pvc.Name, metav1.GetOptions{})
			if owner == run.UUID {
				require.True(t, apierrors.IsNotFound(err), "a takeover must clean the PVC even if its reference was never checkpointed")
				require.NoError(t, coordinator.cleanupWorkspaceForRun(context.Background(), run, NewRuntimeManifest(ExecutorKubernetes)))
			} else {
				require.NoError(t, err)
			}
			for _, action := range client.Actions() {
				require.NotEqual(t, "create", action.GetVerb(), "cleanup must never provision a previously absent workspace")
			}
		})
	}
}
