package config

import (
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestWorkspacePVCSpecPreservesDefaults(t *testing.T) {
	defaults := &corev1.PersistentVolumeClaimSpec{
		AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
		StorageClassName: util.StringPointer("default-class"),
		Resources: corev1.VolumeResourceRequirements{Requests: corev1.ResourceList{
			corev1.ResourceStorage: resource.MustParse("1Gi"),
		}},
	}
	patch, err := structpb.NewStruct(map[string]any{"storageClassName": "fast-class"})
	require.NoError(t, err)
	spec, err := WorkspacePVCSpec(&pipelinespec.WorkspaceConfig{
		Size: "4Gi", Kubernetes: &pipelinespec.KubernetesWorkspaceConfig{PvcSpecPatch: patch},
	}, defaults)
	require.NoError(t, err)
	require.Equal(t, defaults.AccessModes, spec.AccessModes)
	require.Equal(t, "fast-class", *spec.StorageClassName)
	require.Equal(t, resource.MustParse("4Gi"), spec.Resources.Requests[corev1.ResourceStorage])
	require.Equal(t, "default-class", *defaults.StorageClassName)
	require.Equal(t, resource.MustParse("1Gi"), defaults.Resources.Requests[corev1.ResourceStorage])
}

func TestWorkspacePVCSpecRejectsUnknownRequirements(t *testing.T) {
	patch, err := structpb.NewStruct(map[string]any{"volumeName": "other-run"})
	require.NoError(t, err)
	_, err = WorkspacePVCSpec(&pipelinespec.WorkspaceConfig{
		Size: "4Gi", Kubernetes: &pipelinespec.KubernetesWorkspaceConfig{PvcSpecPatch: patch},
	}, nil)
	require.ErrorContains(t, err, "unknown fields")
}
