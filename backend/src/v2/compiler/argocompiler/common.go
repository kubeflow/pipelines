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
	k8score "k8s.io/api/core/v1"
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
