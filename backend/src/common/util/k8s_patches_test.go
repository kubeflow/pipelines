package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
	k8score "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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

func strPtr(s string) *string {
	return &s
}
