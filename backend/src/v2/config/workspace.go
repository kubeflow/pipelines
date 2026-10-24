package config

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// WorkspacePVCSpec resolves pipeline workspace requirements without a workflow engine.
func WorkspacePVCSpec(workspace *pipelinespec.WorkspaceConfig, defaults *corev1.PersistentVolumeClaimSpec) (*corev1.PersistentVolumeClaimSpec, error) {
	if workspace.GetSize() == "" {
		return nil, fmt.Errorf("workspace size is required but not specified")
	}
	spec := &corev1.PersistentVolumeClaimSpec{}
	if defaults != nil {
		spec = defaults.DeepCopy()
	}
	if patch := workspace.GetKubernetes().GetPvcSpecPatch(); patch != nil {
		var unknown []string
		for name := range patch.GetFields() {
			if name != "accessModes" && name != "storageClassName" {
				unknown = append(unknown, name)
			}
		}
		if len(unknown) > 0 {
			sort.Strings(unknown)
			return nil, fmt.Errorf("unknown fields in PVC spec patch: %v", unknown)
		}
		data, err := patch.MarshalJSON()
		if err != nil {
			return nil, err
		}
		var override corev1.PersistentVolumeClaimSpec
		if err := json.Unmarshal(data, &override); err != nil {
			return nil, err
		}
		spec = util.PatchPVCSpec(spec, &override)
	}
	if len(spec.AccessModes) == 0 {
		return nil, fmt.Errorf("workspace PVC spec must specify accessModes")
	}
	if spec.StorageClassName != nil && *spec.StorageClassName == "" {
		return nil, fmt.Errorf("workspace PVC spec storageClassName must be omitted or set to a non-empty value")
	}
	quantity, err := resource.ParseQuantity(workspace.GetSize())
	if err != nil {
		return nil, fmt.Errorf("invalid size value for workspace PVC: %v", err)
	}
	if quantity.Sign() < 0 {
		return nil, fmt.Errorf("negative size value for workspace PVC: %v", workspace.GetSize())
	}
	if spec.Resources.Requests == nil {
		spec.Resources.Requests = corev1.ResourceList{}
	}
	spec.Resources.Requests[corev1.ResourceStorage] = quantity
	return spec, nil
}
