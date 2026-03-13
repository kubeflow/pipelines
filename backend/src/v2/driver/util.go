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
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/placeholder"
	"google.golang.org/protobuf/types/known/structpb"
)

// ifPresentCondition is the named schema for the IfPresent conditional clause
// embedded in container arguments by the KFP compiler.
type ifPresentCondition struct {
	IfPresent struct {
		InputName string      `json:"InputName"`
		Then      interface{} `json:"Then"`
		Else      interface{} `json:"Else"`
	} `json:"IfPresent"`
}

// inputPipelineChannelPattern define a regex pattern to match the content within single quotes
// example input channel looks like "{{$.inputs.parameters['pipelinechannel--val']}}"
const inputPipelineChannelPattern = `\$.inputs.parameters\['(.+?)'\]`

func isInputParameterChannel(inputChannel string) bool {
	re := regexp.MustCompile(inputPipelineChannelPattern)
	match := re.FindStringSubmatch(inputChannel)
	return len(match) == 2
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
	}
	return "", fmt.Errorf("failed to extract input parameter from channel: %s", inputChannel)
}

// inputParamConstant convert and return value as a RuntimeValue
func inputParamConstant(value string) *pipelinespec.TaskInputsSpec_InputParameterSpec {
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

// inputParamComponent convert and return value as a ComponentInputParameter
func inputParamComponent(value string) *pipelinespec.TaskInputsSpec_InputParameterSpec {
	return &pipelinespec.TaskInputsSpec_InputParameterSpec{
		Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_ComponentInputParameter{
			ComponentInputParameter: value,
		},
	}
}

// inputParamTaskOutput convert and return producerTask & outputParamKey
// as a TaskOutputParameter.
func inputParamTaskOutput(producerTask, outputParamKey string) *pipelinespec.TaskInputsSpec_InputParameterSpec {
	return &pipelinespec.TaskInputsSpec_InputParameterSpec{
		Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskOutputParameter{
			TaskOutputParameter: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskOutputParameterSpec{
				ProducerTask:       producerTask,
				OutputParameterKey: outputParamKey,
			},
		},
	}
}

// Get iteration items from a structpb.Value.
// Return value may be
// * a list of JSON serializable structs
// * a list of structpb.Value
func getItems(value *structpb.Value) (items []*structpb.Value, err error) {
	switch v := value.GetKind().(type) {
	case *structpb.Value_ListValue:
		return v.ListValue.GetValues(), nil
	case *structpb.Value_StringValue:
		listValue := structpb.Value{}
		if err = listValue.UnmarshalJSON([]byte(v.StringValue)); err != nil {
			return nil, err
		}
		return listValue.GetListValue().GetValues(), nil
	default:
		return nil, fmt.Errorf("value of type %T cannot be iterated", v)
	}
}

func isConditionClause(arg string) bool {
	return strings.HasPrefix(strings.TrimSpace(arg), `{"IfPresent":`)
}

func resolveCondition(arg string, executorInput *pipelinespec.ExecutorInput) ([]string, error) {
	var ifPresent ifPresentCondition
	if err := json.Unmarshal([]byte(arg), &ifPresent); err != nil {
		return nil, fmt.Errorf("failed to parse IfPresent JSON: %w", err)
	}

	val, isPresent := executorInput.GetInputs().GetParameterValues()[ifPresent.IfPresent.InputName]
	// Treat null values as absent for IfPresent semantics.
	// The driver can set optional pipeline inputs to structpb.NewNullValue(),
	// which should be treated as "not present".
	if isPresent {
		if _, isNull := val.GetKind().(*structpb.Value_NullValue); isNull {
			isPresent = false
		}
	}
	var values interface{}
	if isPresent {
		values = ifPresent.IfPresent.Then
	} else {
		values = ifPresent.IfPresent.Else
	}

	if values == nil {
		return []string{}, nil
	}

	parameterValues := executorInput.GetInputs().GetParameterValues()
	var resolved []string
	switch v := values.(type) {
	case string:
		resolvedArg, err := placeholder.ResolveInputParameterPlaceholders(v, parameterValues)
		if err != nil {
			return nil, err
		}
		resolved = []string{resolvedArg}
	case []interface{}:
		for _, item := range v {
			str, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("non-string item in IfPresent Then/Else array: %T", item)
			}
			resolvedArg, err := placeholder.ResolveInputParameterPlaceholders(str, parameterValues)
			if err != nil {
				return nil, err
			}
			resolved = append(resolved, resolvedArg)
		}
	default:
		return nil, fmt.Errorf("unexpected type in IfPresent Then/Else: %T", v)
	}
	return resolved, nil
}

func resolveContainerArgs(args []string, executorInput *pipelinespec.ExecutorInput) ([]string, error) {
	parameterValues := executorInput.GetInputs().GetParameterValues()
	var resolvedArgs []string
	for _, arg := range args {
		// Skip args containing output placeholders - these need to be resolved by Argo at runtime
		// Example: {{$.outputs.parameters['sum'].output_file}}
		if strings.Contains(arg, "$.outputs") {
			resolvedArgs = append(resolvedArgs, arg)
			continue
		}

		if isConditionClause(arg) {
			resolved, err := resolveCondition(arg, executorInput)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve condition: %w", err)
			}
			resolvedArgs = append(resolvedArgs, resolved...)
		} else {
			resolvedArg, err := placeholder.ResolveInputParameterPlaceholders(arg, parameterValues)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve input parameters: %w", err)
			}
			resolvedArgs = append(resolvedArgs, resolvedArg)
		}
	}
	return resolvedArgs, nil
}
