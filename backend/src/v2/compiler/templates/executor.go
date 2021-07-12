// Copyright 2021 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package templates

import (
	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	workflowapi "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/backend/src/v2/common"
	apiv1 "k8s.io/api/core/v1"
)

// TODO(Bobgy): make image configurable
const (
	entrypointImage     = "gcr.io/gongyuan-pipeline-test/kfp-entrypoint"
	entrypointImageRef  = "@sha256:793316de698fe53ab318d43b00213d947f5ac2abbefc780cece46bf8a44381c4"
	entrypointImageFull = entrypointImage + entrypointImageRef
)

const (
	TemplateNameExecutor = "kfp-executor"
	templateNameDummy    = "kfp-dummy"

	// Executor inputs
	ExecutorParamTaskSpec     = DriverParamTaskSpec
	ExecutorParamExecutorSpec = DriverParamExecutorSpec
	ExecutorParamContextName  = paramPrefixKfpInternal + "context-name"
	ExecutorParamOutputsSpec  = PublisherParamOutputsSpec

	executorInternalParamPodName       = paramPrefixKfpInternal + "pod-name"
	executorInternalParamPodSpecPatch  = paramPrefixKfpInternal + "pod-spec-patch"
	executorInternalArtifactParameters = paramPrefixKfpInternal + "parameters"
)

func Executor(executorDriverTemplateName string, executorPublisherTemplateName string) []*workflowapi.Template {
	// TODO(Bobgy): Move dummy template generation out as a separate file.
	var dummy workflowapi.Template
	dummy.Name = templateNameDummy
	// The actual container definition will be injected by pod spec patch
	entrypointVolumeName := "kfp-entrypoint"
	dummy.Volumes = []apiv1.Volume{{
		Name: entrypointVolumeName,
		VolumeSource: apiv1.VolumeSource{
			EmptyDir: &apiv1.EmptyDirVolumeSource{},
		},
	}}
	dummy.Container = &apiv1.Container{
		Image: "dummy",
		VolumeMounts: []apiv1.VolumeMount{{
			Name:      entrypointVolumeName,
			MountPath: common.ExecutorEntrypointVolumePath,
			ReadOnly:  true,
		}},
	}
	// The entrypoint init container copies the entrypoint binary into a volume shared with executor container.
	dummy.InitContainers = []workflowapi.UserContainer{{
		Container: apiv1.Container{
			Name:    "entrypoint-init",
			Image:   entrypointImageFull,
			Command: []string{"cp"},
			Args:    []string{"/bin/kfp-entrypoint", common.ExecutorEntrypointPath},
			VolumeMounts: []apiv1.VolumeMount{{
				Name:      entrypointVolumeName,
				MountPath: common.ExecutorEntrypointVolumePath,
			}},
		},
	}}
	dummy.PodSpecPatch = "{{inputs.parameters." + executorInternalParamPodSpecPatch + "}}"
	dummy.Inputs.Parameters = []workflowapi.Parameter{
		{Name: executorInternalParamPodSpecPatch},
	}
	dummy.Outputs.Parameters = []workflowapi.Parameter{
		{Name: executorInternalParamPodName, Value: v1alpha1.AnyStringPtr(argoVariablePodName)},
	}
	dummy.Outputs.Artifacts = []workflowapi.Artifact{
		{Name: executorInternalArtifactParameters, Path: common.ExecutorOutputPathParameters, Optional: true},
	}

	driverTaskName := "driver"
	executorTaskName := "executor"
	driverType := "EXECUTOR"

	var executorDag workflowapi.Template
	executorDag.Name = TemplateNameExecutor
	executorDag.Inputs.Parameters = []workflowapi.Parameter{
		{Name: ExecutorParamContextName},
		{Name: ExecutorParamTaskSpec},
		{Name: ExecutorParamExecutorSpec},
		{Name: ExecutorParamOutputsSpec},
	}
	taskSpecInJson := "{{inputs.parameters." + ExecutorParamTaskSpec + "}}"
	executorSpecInJson := "{{inputs.parameters." + ExecutorParamExecutorSpec + "}}"
	podSpecPatchValue := "{{tasks." + driverTaskName + ".outputs.parameters." + executorInternalParamPodSpecPatch + "}}"
	parentContextNameValue := "{{inputs.parameters." + ExecutorParamContextName + "}}"
	executorDag.DAG = &workflowapi.DAGTemplate{}
	executorDag.DAG.Tasks = []workflowapi.DAGTask{
		{
			Name:     driverTaskName,
			Template: executorDriverTemplateName,
			Arguments: workflowapi.Arguments{
				Parameters: []workflowapi.Parameter{
					{Name: DriverParamTaskSpec, Value: v1alpha1.AnyStringPtr(taskSpecInJson)},
					{Name: DriverParamDriverType, Value: v1alpha1.AnyStringPtr(driverType)},
					{Name: DriverParamParentContextName, Value: v1alpha1.AnyStringPtr(parentContextNameValue)},
					{Name: DriverParamExecutorSpec, Value: v1alpha1.AnyStringPtr(executorSpecInJson)},
				},
			},
		},
		{
			Name:         executorTaskName,
			Template:     dummy.Name,
			Dependencies: []string{driverTaskName},
			Arguments: workflowapi.Arguments{
				Parameters: []workflowapi.Parameter{
					{Name: executorInternalParamPodSpecPatch, Value: v1alpha1.AnyStringPtr(podSpecPatchValue)},
				},
			},
		},
	}
	return []*workflowapi.Template{
		&executorDag,
		&dummy,
	}
}
