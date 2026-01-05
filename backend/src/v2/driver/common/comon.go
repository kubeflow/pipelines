// Copyright 2025 The Kubeflow Authors
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

// Package common provides common utilities for the KFP v2 driver.
package common

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/kubernetes_platform/go/kubernetesplatform"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	WorkspaceMetadataField = "_kfp_workspace"
)

// Options contain driver options
type Options struct {
	// required, pipeline context name
	PipelineName string
	// required, KFP run ID
	Run *apiv2beta1.Run
	// required, Component spec
	Component *pipelinespec.ComponentSpec
	// required
	ParentTask *apiv2beta1.PipelineTaskDetail
	// required
	ScopePath util.ScopePath

	// optional, iteration index. -1 means not an iteration.
	IterationIndex int

	// optional, required only by root DAG driver
	RuntimeConfig *pipelinespec.PipelineJob_RuntimeConfig
	Namespace     string

	// optional, required by non-root drivers
	Task *pipelinespec.PipelineTaskSpec

	// optional, required only by container driver
	Container *pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec

	// optional, allows to specify kubernetes-specific executor config
	KubernetesExecutorConfig *kubernetesplatform.KubernetesExecutorConfig

	// optional, required only if the {{$.pipeline_job_resource_name}} placeholder is used or the run uses a workspace
	RunName string
	// optional, required only if the {{$.pipeline_job_name}} placeholder is used
	RunDisplayName          string
	PipelineLogLevel        string
	PublishLogs             string
	CacheDisabled           bool
	DriverType              string
	TaskName                string
	PodName                 string
	PodUID                  string
	MLPipelineTLSEnabled    bool
	MLPipelineServerAddress string
	MLPipelineServerPort    string
	CaCertPath              string
}

// Info provides information used for debugging
func (o Options) Info() string {
	msg := fmt.Sprintf("pipelineName=%v, runID=%v", o.PipelineName, o.Run.GetRunId())
	if o.Task.GetTaskInfo().GetName() != "" {
		msg += fmt.Sprintf(", taskDisplayName=%q", o.Task.GetTaskInfo().GetName())
	}
	if o.TaskName != "" {
		msg += fmt.Sprintf(", taskName=%q", o.TaskName)
	}
	if o.Task.GetComponentRef().GetName() != "" {
		msg += fmt.Sprintf(", component=%q", o.Task.GetComponentRef().GetName())
	}
	if o.ParentTask != nil {
		msg += fmt.Sprintf(", dagExecutionID=%v", o.ParentTask.GetParentTaskId())
	}
	if o.IterationIndex >= 0 {
		msg += fmt.Sprintf(", iterationIndex=%v", o.IterationIndex)
	}
	if o.RuntimeConfig != nil {
		msg += ", runtimeConfig" // this only means runtimeConfig is not empty
	}
	if o.Component.GetImplementation() != nil {
		msg += ", componentSpec" // this only means componentSpec is not empty
	}
	if o.KubernetesExecutorConfig != nil {
		msg += ", KubernetesExecutorConfig" // this only means KubernetesExecutorConfig is not empty
	}
	return msg
}

const pipelineChannelPrefix = "pipelinechannel--"

func IsLoopArgument(name string) bool {
	// Remove prefix
	nameWithoutPrefix := strings.TrimPrefix(name, pipelineChannelPrefix)
	return strings.HasSuffix(nameWithoutPrefix, "loop-item") || strings.HasPrefix(nameWithoutPrefix, "loop-item")
}

func IsRuntimeIterationTask(task *apiv2beta1.PipelineTaskDetail) bool {
	return task.Type == apiv2beta1.PipelineTaskDetail_RUNTIME && task.TypeAttributes != nil && task.TypeAttributes.IterationIndex != nil
}

// inputPipelineChannelPattern define a regex pattern to match the content within single quotes
// example input channel looks like "{{$.inputs.parameters['pipelinechannel--val']}}"
const inputPipelineChannelPattern = `\$.inputs.parameters\['(.+?)'\]`

func IsInputParameterChannel(inputChannel string) bool {
	re := regexp.MustCompile(inputPipelineChannelPattern)
	match := re.FindStringSubmatch(inputChannel)
	if len(match) == 2 {
		return true
	} else {
		// if len(match) > 2, then this is still incorrect because
		// inputChannel should contain only one parameter channel input
		return false
	}
}

// extractInputParameterFromChannel takes an inputChannel that adheres to
// inputPipelineChannelPattern and extracts the channel parameter name.
// For example given an input channel of the form "{{$.inputs.parameters['pipelinechannel--val']}}"
// the channel parameter name "pipelinechannel--val" is returned.
func extractInputParameterFromChannel(inputChannel string) (string, error) {
	re := regexp.MustCompile(inputPipelineChannelPattern)
	match := re.FindStringSubmatch(inputChannel)
	if len(match) > 1 {
		extractedValue := match[1]
		return extractedValue, nil
	} else {
		return "", fmt.Errorf("failed to extract input parameter from channel: %s", inputChannel)
	}
}

// InputParamConstant convert and return value as a RuntimeValue
func InputParamConstant(value string) *pipelinespec.TaskInputsSpec_InputParameterSpec {
	return &pipelinespec.TaskInputsSpec_InputParameterSpec{
		Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_RuntimeValue{
			RuntimeValue: &pipelinespec.ValueOrRuntimeParameter{
				Value: &pipelinespec.ValueOrRuntimeParameter_Constant{
					Constant: structpb.NewStringValue(value),
				},
			},
		},
	}
}

// InputParamComponent convert and return value as a ComponentInputParameter
func InputParamComponent(value string) *pipelinespec.TaskInputsSpec_InputParameterSpec {
	return &pipelinespec.TaskInputsSpec_InputParameterSpec{
		Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_ComponentInputParameter{
			ComponentInputParameter: value,
		},
	}
}

// InputParamTaskOutput convert and return producerTask & outputParamKey
// as a TaskOutputParameter.
func InputParamTaskOutput(producerTask, outputParamKey string) *pipelinespec.TaskInputsSpec_InputParameterSpec {
	return &pipelinespec.TaskInputsSpec_InputParameterSpec{
		Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskOutputParameter{
			TaskOutputParameter: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskOutputParameterSpec{
				ProducerTask:       producerTask,
				OutputParameterKey: outputParamKey,
			},
		},
	}
}

var paramPattern = regexp.MustCompile(`{{\$.inputs.parameters\['([^']+)'\]}}`)

// ParsePipelineParam takes a string and returns (isMatch, isPipelineChannel, paramName)
func ParsePipelineParam(s string) (bool, bool, string) {
	paramName, ok := extractParameterName(s)
	if !ok {
		return false, false, ""
	}
	return true, isPipelineChannel(paramName), paramName
}

// extractParameterName extracts the inner value inside the brackets ['...']
// e.g. returns "pipelinechannel--cpu_limit" from "{{$.inputs.parameters['pipelinechannel--cpu_limit']}}"
func extractParameterName(s string) (string, bool) {
	matches := paramPattern.FindStringSubmatch(s)
	if len(matches) > 1 {
		return matches[1], true
	}
	return "", false
}

// isPipelineChannel checks if a parameter name follows the "pipelinechannel--" prefix convention
func isPipelineChannel(name string) bool {
	return strings.HasPrefix(name, "pipelinechannel--")
}
