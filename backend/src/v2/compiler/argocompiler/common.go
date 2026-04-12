// Copyright 2021 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package argocompiler

import (
	wfapi "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/v2/component"
	k8score "k8s.io/api/core/v1"
)

const (
	MLPipelineTLSEnabledEnvVar  = "ML_PIPELINE_TLS_ENABLED"
	DefaultMLPipelineTLSEnabled = false
	// volumeNameCustomCA is the name of the volume that holds the custom CA
	// bundle. It must match the name used by ConfigureCustomCABundle.
	volumeNameCustomCA = "custom-ca"
)

// env vars in metadata-grpc-configmap is defined in component package
var metadataConfigIsOptional bool = true
var metadataEnvFrom = k8score.EnvFromSource{
	ConfigMapRef: &k8score.ConfigMapEnvSource{
		LocalObjectReference: k8score.LocalObjectReference{
			Name: "metadata-grpc-configmap",
		},
		Optional: &metadataConfigIsOptional,
	},
}

var commonEnvs = []k8score.EnvVar{{
	Name: "KFP_POD_NAME",
	ValueFrom: &k8score.EnvVarSource{
		FieldRef: &k8score.ObjectFieldSelector{
			FieldPath: "metadata.name",
		},
	},
}, {
	Name: "KFP_POD_UID",
	ValueFrom: &k8score.EnvVarSource{
		FieldRef: &k8score.ObjectFieldSelector{
			FieldPath: "metadata.uid",
		},
	},
}}

// retryIndexEnv injects the Argo retry attempt index into the executor
// container. Argo substitutes {{retries}} with the 0-based attempt index at
// pod creation time, so each retry attempt writes executor-logs to a distinct,
// sequentially-numbered path (executor-logs-0, executor-logs-1, …).
// This variable is only valid inside a template that has a retryStrategy, so
// it must NOT be added to commonEnvs (which is applied to all templates).
var retryIndexEnv = k8score.EnvVar{
	Name:  component.EnvRetryIndex,
	Value: "{{retries}}",
}

// ConfigureCustomCABundle adds CABundle environment variables and volume mounts if CABUNDLE_SECRET_NAME is set.
func ConfigureCustomCABundle(tmpl *wfapi.Template) {
	caBundleSecretName := common.GetCaBundleSecretName()
	caBundleConfigMapName := common.GetCaBundleConfigMapName()
	// If CABUNDLE_KEY_NAME is not set, use "ca.crt".
	caBundleKeyName := common.GetCABundleKey()
	if caBundleKeyName == "" {
		caBundleKeyName = "ca.crt"
	}
	volumeSource := k8score.VolumeSource{}

	// CABUNDLE_SECRET_NAME is prioritized above CABUNDLE_CONFIGMAP_NAME.
	if caBundleSecretName != "" { // nolint:gocritic // ifElseChain is preferred here for clarity over a switch
		volumeSource.Secret = &k8score.SecretVolumeSource{
			SecretName: caBundleSecretName,
			Items: []k8score.KeyToPath{
				{
					Key:  caBundleKeyName,
					Path: "ca.crt",
				},
			},
		}
	} else if caBundleConfigMapName != "" {
		volumeSource.ConfigMap = &k8score.ConfigMapVolumeSource{
			LocalObjectReference: k8score.LocalObjectReference{Name: caBundleConfigMapName},
			Items: []k8score.KeyToPath{
				{
					Key:  caBundleKeyName,
					Path: "ca.crt",
				},
			},
		}
	} else {
		glog.Error("Neither CABUNDLE_SECRET_NAME nor CABUNDLE_CONFIGMAP_NAME is set. Failed to configure custom CA bundle.")
		return
	}
	volume := k8score.Volume{
		Name:         volumeNameCustomCA,
		VolumeSource: volumeSource,
	}
	tmpl.Volumes = append(tmpl.Volumes, volume)

	volumeMount := k8score.VolumeMount{Name: volumeNameCustomCA, MountPath: common.CABundleDir}
	tmpl.Container.VolumeMounts = append(tmpl.Container.VolumeMounts, volumeMount)

}

// addExitTask adds an exit lifecycle hook to a task if exitTemplate is not empty.
func addExitTask(task *wfapi.DAGTask, exitTemplate string, parentDagID string) {
	if exitTemplate == "" {
		return
	}

	task.Hooks = wfapi.LifecycleHooks{
		wfapi.ExitLifecycleEvent: wfapi.LifecycleHook{
			Template: exitTemplate,
			Arguments: wfapi.Arguments{Parameters: []wfapi.Parameter{
				{Name: paramParentDagID, Value: wfapi.AnyStringPtr(parentDagID)},
			}},
		},
	}
}
