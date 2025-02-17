// Copyright 2021-2024 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package driver

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	"github.com/kubeflow/pipelines/kubernetes_platform/go/kubernetesplatform"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"regexp"
)

// inputPipelineChannelPattern define a regex pattern to match the content within single quotes
// example input channel looks like "{{$.inputs.parameters['pipelinechannel--val']}}"
const inputPipelineChannelPattern = `\$.inputs.parameters\['(.+?)'\]`

func isInputParameterChannel(inputChannel string) bool {
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

// resolvePodSpecInputRuntimeParameter resolves runtime value that is intended to be
// utilized within the Pod Spec. parameterValue takes the form of:
// "{{$.inputs.parameters['pipelinechannel--someParameterName']}}"
//
// parameterValue is a runtime parameter value that has been resolved and included within
// the executor input. Since the pod spec patch cannot dynamically update the underlying
// container template's inputs in an Argo Workflow, this is a workaround for resolving
// such parameters.
//
// If parameter value is not a parameter channel, then a constant value is assumed and
// returned as is.
func resolvePodSpecInputRuntimeParameter(parameterValue string, executorInput *pipelinespec.ExecutorInput) (string, error) {
	if isInputParameterChannel(parameterValue) {
		inputImage, err := extractInputParameterFromChannel(parameterValue)
		if err != nil {
			return "", err
		}
		if val, ok := executorInput.Inputs.ParameterValues[inputImage]; ok {
			return val.GetStringValue(), nil
		} else {
			return "", fmt.Errorf("executorInput did not contain container Image input parameter")
		}
	}
	return parameterValue, nil
}

// resolveK8sParameter resolves a k8s JSON and unmarshal it
// to the provided k8s resource.
//
// Parameters:
//   - k8sParamSpec: An input parameter spec that resolve to a valid structpb.Value
//   - inputParams: InputParams that contain resolution context for k8sParamSpec
func resolveK8sParameter(
	ctx context.Context,
	opts Options,
	dag *metadata.DAG,
	pipeline *metadata.Pipeline,
	mlmd *metadata.Client,
	k8sParamSpec *kubernetesplatform.InputParameterSpec,
	inputParams map[string]*structpb.Value,
) (*structpb.Value, error) {
	pipelineParamSpec := &pipelinespec.TaskInputsSpec_InputParameterSpec{}
	err := k8sToPipelineInputParamSpec(k8sParamSpec, pipelineParamSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to convert input parameter spec to pipeline spec: %v", err)
	}
	resolvedParameter, err := resolveInputParameter(ctx, dag, pipeline,
		opts, mlmd, pipelineParamSpec, inputParams)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve input parameter name: %w", err)
	}
	return resolvedParameter, nil
}

// resolveK8sJsonParameter resolves a k8s JSON and unmarshal it
// to the provided k8s resource.
//
// Parameters:
//   - k8sParamSpec: An input parameter spec that resolve to a valid JSON
//   - inputParams: InputParams that contain resolution context for k8sParamSpec
//   - res: The k8s resource to unmarshal the json to
func resolveK8sJsonParameter[k8sResource any](
	ctx context.Context,
	opts Options,
	dag *metadata.DAG,
	pipeline *metadata.Pipeline,
	mlmd *metadata.Client,
	k8sParamSpec *kubernetesplatform.InputParameterSpec,
	inputParams map[string]*structpb.Value,
	res *k8sResource,
) error {
	resolvedParam, err := resolveK8sParameter(ctx, opts, dag, pipeline, mlmd,
		k8sParamSpec, inputParams)
	if err != nil {
		return fmt.Errorf("failed to resolve k8s parameter: %w", err)
	}
	paramJSON, err := resolvedParam.GetStructValue().MarshalJSON()
	if err != nil {
		return err
	}
	err = json.Unmarshal(paramJSON, &res)
	if err != nil {
		return fmt.Errorf("failed to unmarshal k8s Resource json "+
			"ensure that k8s Resource json correctly adheres to its respective k8s spec: %w", err)
	}
	return nil
}

// k8sToPipelineInputParamSpec converts a given kubernetes platform
// input parameter spec to it's the pipeline spec equivalent.
// both src and dst should follow the same structure.
//
// Parameters:
//   - src the InputParameterSpec to be converted
//   - dst the destination target for the PipelineSpec InputParameterSpec
func k8sToPipelineInputParamSpec(src *kubernetesplatform.InputParameterSpec, dst *pipelinespec.TaskInputsSpec_InputParameterSpec) error {
	data, err := protojson.Marshal(proto.MessageV2(src))
	if err != nil {
		return err
	}
	return protojson.Unmarshal(data, proto.MessageV2(dst))
}

// inputParamConstant convert and return value as a RuntimeValue
func inputParamConstant(value string) *kubernetesplatform.InputParameterSpec {
	return &kubernetesplatform.InputParameterSpec{
		Kind: &kubernetesplatform.InputParameterSpec_RuntimeValue{
			RuntimeValue: &kubernetesplatform.ValueOrRuntimeParameter{
				Value: &kubernetesplatform.ValueOrRuntimeParameter_Constant{
					Constant: structpb.NewStringValue(value),
				},
			},
		},
	}
}

// inputParamComponent convert and return value as a ComponentInputParameter
func inputParamComponent(value string) *kubernetesplatform.InputParameterSpec {
	return &kubernetesplatform.InputParameterSpec{
		Kind: &kubernetesplatform.InputParameterSpec_ComponentInputParameter{
			ComponentInputParameter: value,
		},
	}
}

// inputParamTaskOutput convert and return producerTask & outputParamKey
// as a TaskOutputParameter.
func inputParamTaskOutput(producerTask, outputParamKey string) *kubernetesplatform.InputParameterSpec {
	return &kubernetesplatform.InputParameterSpec{
		Kind: &kubernetesplatform.InputParameterSpec_TaskOutputParameter{
			TaskOutputParameter: &kubernetesplatform.InputParameterSpec_TaskOutputParameterSpec{
				ProducerTask:       producerTask,
				OutputParameterKey: outputParamKey,
			},
		},
	}
}
