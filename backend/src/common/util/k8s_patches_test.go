package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
	k8score "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPatchPVCSpec(t *testing.T) {
	base := &k8score.PersistentVolumeClaimSpec{
		AccessModes: []k8score.PersistentVolumeAccessMode{"ReadWriteOnce"},
		Resources: k8score.VolumeResourceRequirements{
			Requests: k8score.ResourceList{
				k8score.ResourceStorage: resource.MustParse("10Gi"),
			},
		},
		StorageClassName: strPtr("standard"),
	}

	t.Run("patch access modes and storage class", func(t *testing.T) {
		patch := &k8score.PersistentVolumeClaimSpec{
			AccessModes:      []k8score.PersistentVolumeAccessMode{"ReadWriteMany"},
			StorageClassName: strPtr("fast-storage"),
		}
		result := PatchPVCSpec(base.DeepCopy(), patch)
		assert.Equal(t, []k8score.PersistentVolumeAccessMode{"ReadWriteMany"}, result.AccessModes)
		assert.Equal(t, "fast-storage", *result.StorageClassName)
		assert.Equal(t, resource.MustParse("10Gi"), result.Resources.Requests[k8score.ResourceStorage])
	})

	t.Run("patch resources", func(t *testing.T) {
		patch := &k8score.PersistentVolumeClaimSpec{
			Resources: k8score.VolumeResourceRequirements{
				Requests: k8score.ResourceList{
					k8score.ResourceStorage: resource.MustParse("20Gi"),
				},
			},
		}
		result := PatchPVCSpec(base.DeepCopy(), patch)
		assert.Equal(t, resource.MustParse("20Gi"), result.Resources.Requests[k8score.ResourceStorage])
	})

	t.Run("patch is nil", func(t *testing.T) {
		result := PatchPVCSpec(base.DeepCopy(), nil)
		assert.Equal(t, base, result)
	})
}

func TestPatchPVCSpec_BaseNil(t *testing.T) {
	patch := &k8score.PersistentVolumeClaimSpec{}
	result := PatchPVCSpec(nil, patch)
	assert.Nil(t, result)
}

func TestPatchPVCSpec_VolumeName(t *testing.T) {
	base := &k8score.PersistentVolumeClaimSpec{}
	patch := &k8score.PersistentVolumeClaimSpec{
		VolumeName: "my-volume",
	}
	result := PatchPVCSpec(base, patch)
	assert.Equal(t, "my-volume", result.VolumeName)
}

func TestPatchPVCSpec_VolumeMode(t *testing.T) {
	blockMode := k8score.PersistentVolumeBlock
	base := &k8score.PersistentVolumeClaimSpec{}
	patch := &k8score.PersistentVolumeClaimSpec{
		VolumeMode: &blockMode,
	}
	result := PatchPVCSpec(base, patch)
	assert.Equal(t, &blockMode, result.VolumeMode)
}

func TestPatchPVCSpec_DataSource(t *testing.T) {
	base := &k8score.PersistentVolumeClaimSpec{}
	dataSource := &k8score.TypedLocalObjectReference{
		Kind: "VolumeSnapshot",
		Name: "my-snapshot",
	}
	patch := &k8score.PersistentVolumeClaimSpec{
		DataSource: dataSource,
	}
	result := PatchPVCSpec(base, patch)
	assert.Equal(t, "VolumeSnapshot", result.DataSource.Kind)
	assert.Equal(t, "my-snapshot", result.DataSource.Name)
}

func TestPatchPVCSpec_DataSourceRef(t *testing.T) {
	base := &k8score.PersistentVolumeClaimSpec{}
	apiGroup := "snapshot.storage.k8s.io"
	dataSourceRef := &k8score.TypedObjectReference{
		APIGroup: &apiGroup,
		Kind:     "VolumeSnapshot",
		Name:     "my-snapshot",
	}
	patch := &k8score.PersistentVolumeClaimSpec{
		DataSourceRef: dataSourceRef,
	}
	result := PatchPVCSpec(base, patch)
	assert.NotNil(t, result.DataSourceRef)
	assert.Equal(t, "VolumeSnapshot", result.DataSourceRef.Kind)
}

func TestPatchPVCSpec_Selector(t *testing.T) {
	base := &k8score.PersistentVolumeClaimSpec{}
	patch := &k8score.PersistentVolumeClaimSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{"tier": "fast"},
		},
	}
	result := PatchPVCSpec(base, patch)
	assert.NotNil(t, result.Selector)
	assert.Equal(t, "fast", result.Selector.MatchLabels["tier"])
}

func TestPatchPVCSpec_ResourcesNilBase(t *testing.T) {
	base := &k8score.PersistentVolumeClaimSpec{}
	patch := &k8score.PersistentVolumeClaimSpec{
		Resources: k8score.VolumeResourceRequirements{
			Requests: k8score.ResourceList{
				k8score.ResourceStorage: resource.MustParse("5Gi"),
			},
		},
	}
	result := PatchPVCSpec(base, patch)
	assert.Equal(t, resource.MustParse("5Gi"), result.Resources.Requests[k8score.ResourceStorage])
}

func strPtr(s string) *string {
	return &s
}
