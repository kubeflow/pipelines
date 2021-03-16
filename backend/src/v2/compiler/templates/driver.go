// Copyright 2021 Google LLC
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
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	workflowapi "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	k8sv1 "k8s.io/api/core/v1"
)

// TODO(Bobgy): make image configurable
// gcr.io/gongyuan-pipeline-test/kfp-driver:latest
const (
	driverImage     = "gcr.io/gongyuan-pipeline-test/kfp-driver"
	driverImageRef  = "@sha256:d3fa780ffc59a22253eb4b4460e89f722811dfdd2ded277d3606f1fce323af87"
	driverImageFull = driverImage + driverImageRef
)

const (
	paramPrefixKfpInternal = "kfp-"
	outputPathPodSpecPatch = "/kfp/outputs/pod-spec-patch.json"
)

const (
	// Inputs
	DriverParamParentContextName = paramPrefixKfpInternal + "parent-context-name"
	DriverParamExecutionName     = paramPrefixKfpInternal + "execution-name"
	DriverParamDriverType        = paramPrefixKfpInternal + "driver-type"
	DriverParamTaskSpec          = paramPrefixKfpInternal + "task-spec"
	DriverParamExecutorSpec      = paramPrefixKfpInternal + "executor-spec"
	// Outputs
	DriverParamExecutionId  = paramPrefixKfpInternal + "execution-id"
	DriverParamContextName  = paramPrefixKfpInternal + "context-name"
	DriverParamPodSpecPatch = paramPrefixKfpInternal + "pod-spec-patch"
)

// Do not modify this, this should be constant too.
// This needs to be used in string pointers, so it is "var" instead of "const".
var (
	mlmdExecutionName = "kfp-executor-{{pod.name}}"
)

// TODO(Bobgy): parameters is no longer needed.
// TODO(Bobgy): reuse existing templates if they are the same.
func Driver(isDag bool) *workflowapi.Template {
	driver := &workflowapi.Template{}
	// driver.Name is not set, it should be set after calling this method.
	driver.Container = &k8sv1.Container{}
	driver.Container.Image = driverImageFull
	driver.Container.Command = []string{"/bin/kfp-driver"}
	driver.Inputs.Parameters = []workflowapi.Parameter{
		{Name: DriverParamParentContextName},
		{Name: DriverParamExecutionName, Value: v1alpha1.AnyStringPtr(mlmdExecutionName)},
		{Name: DriverParamDriverType},
		{Name: DriverParamTaskSpec},
	}
	driver.Outputs.Parameters = []workflowapi.Parameter{
		{Name: DriverParamExecutionId, ValueFrom: &workflowapi.ValueFrom{
			Path: "/kfp/outputs/internal/execution-id",
		}},
	}
	driver.Container.Args = []string{
		"--logtostderr",
		// TODO(Bobgy): make this configurable
		"--mlmd_url=metadata-grpc-service.kubeflow.svc.cluster.local:8080",
		"--parent_context_name={{inputs.parameters." + DriverParamParentContextName + "}}",
		"--execution_name={{inputs.parameters." + DriverParamExecutionName + "}}",
		"--driver_type={{inputs.parameters." + DriverParamDriverType + "}}",
		"--task_spec={{inputs.parameters." + DriverParamTaskSpec + "}}",
		"--output_path_execution_id={{outputs.parameters." + DriverParamExecutionId + ".path}}",
	}
	if isDag {
		driver.Container.Args = append(
			driver.Container.Args,
			"--output_path_context_name={{outputs.parameters."+DriverParamContextName+".path}}",
		)
		driver.Outputs.Parameters = append(
			driver.Outputs.Parameters,
			workflowapi.Parameter{
				Name: DriverParamContextName,
				ValueFrom: &workflowapi.ValueFrom{
					Path: "/kfp/outputs/internal/context-name",
				},
			},
		)
	} else {
		// input executor spec
		driver.Container.Args = append(
			driver.Container.Args,
			"--executor_spec={{inputs.parameters."+DriverParamExecutorSpec+"}}",
		)
		driver.Inputs.Parameters = append(
			driver.Inputs.Parameters,
			workflowapi.Parameter{Name: DriverParamExecutorSpec},
		)
		// output pod spec patch
		driver.Container.Args = append(
			driver.Container.Args,
			"--output_path_pod_spec_patch="+outputPathPodSpecPatch,
		)
		driver.Outputs.Parameters = append(
			driver.Outputs.Parameters,
			workflowapi.Parameter{
				Name: DriverParamPodSpecPatch,
				ValueFrom: &workflowapi.ValueFrom{
					Path: outputPathPodSpecPatch,
				},
			},
		)
	}
	return driver
}
