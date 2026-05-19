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
	"regexp"
	"strings"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiV2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient/kfpapi"
	"github.com/kubeflow/pipelines/backend/src/v2/driver/common"
	"github.com/kubeflow/pipelines/backend/src/v2/driver/resolver"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

// fullInputParameterRe matches the complete {{$.inputs.parameters['name']}} placeholder
// including the surrounding braces, for template substitution in container arguments.
var fullInputParameterRe = regexp.MustCompile(`\{\{\$\.inputs\.parameters\['(.+?)']}}`)

func isInputParameterChannel(inputChannel string) bool {
	re := regexp.MustCompile(inputPipelineChannelPattern)
	match := re.FindStringSubmatch(inputChannel)
	return len(match) == 2
}

// isAlreadyExistsErr checks whether the error is a gRPC AlreadyExists error, or whether
// the error message contains "AlreadyExists" or "Duplicate entry" as a fallback.
func isAlreadyExistsErr(err error) bool {
	if s, ok := status.FromError(err); ok && s.Code() == codes.AlreadyExists {
		return true
	}

	// Some callers still wrap duplicate-entry errors, so also check the
	// message string for known markers as a fallback.
	return strings.Contains(err.Error(), "AlreadyExists") || strings.Contains(err.Error(), "Duplicate entry")
}

// extractInputParameterFromChannel takes an inputChannel that adheres to
// inputPipelineChannelPattern and extracts the channel parameter name.
// For example given an input channel of the form "{{$.inputs.parameters['pipelinechannel--val']}}"
// the channel parameter name "pipelinechannel--val" is returned.
func extractInputParameterFromChannel(inputChannel string) (string, error) {
	re := regexp.MustCompile(inputPipelineChannelPattern)
	match := re.FindStringSubmatch(inputChannel)
	if len(match) > 1 {
		return match[1], nil
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

// pbValueToString converts a structpb.Value to its string representation.
// This handles all parameter types including STRING, NUMBER_INTEGER, NUMBER_DOUBLE, and BOOLEAN,
// unlike GetStringValue() which returns an empty string for non-string types.
// LIST and STRUCT values are serialized as JSON for parity with the launcher's
// placeholder substitution behavior.
func pbValueToString(v *structpb.Value) string {
	switch v.GetKind().(type) {
	case *structpb.Value_StringValue:
		return v.GetStringValue()
	case *structpb.Value_NumberValue:
		n := v.GetNumberValue()
		if n == float64(int64(n)) {
			return fmt.Sprintf("%d", int64(n))
		}
		return fmt.Sprintf("%g", n)
	case *structpb.Value_BoolValue:
		if v.GetBoolValue() {
			return "true"
		}
		return "false"
	case *structpb.Value_NullValue:
		return ""
	default:
		b, err := json.Marshal(v.AsInterface())
		if err != nil {
			return ""
		}
		return string(b)
	}
}

// validateRootDAG contains validation for root DAG driver options.
func validateRootDAG(opts common.Options) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("invalid root DAG driver args: %w", err)
		}
	}()
	if opts.PipelineName == "" {
		return fmt.Errorf("pipeline name is required")
	}
	if opts.Run.GetRunId() == "" {
		return fmt.Errorf("KFP run ID is required")
	}
	if opts.Component == nil {
		return fmt.Errorf("component spec is required")
	}
	if opts.RuntimeConfig == nil {
		return fmt.Errorf("runtime config is required")
	}
	if opts.Namespace == "" {
		return fmt.Errorf("namespace is required")
	}
	if opts.Task.GetTaskInfo().GetName() != "" {
		return fmt.Errorf("task spec is unnecessary")
	}
	if opts.ParentTask != nil && opts.ParentTask.GetTaskId() == "" {
		return fmt.Errorf("parent task is required")
	}
	if opts.Container != nil {
		return fmt.Errorf("container spec is unnecessary")
	}
	if opts.IterationIndex >= 0 {
		return fmt.Errorf("iteration index is unnecessary")
	}
	return nil
}

// validateDAG validates non-root DAG options.
func validateDAG(opts common.Options) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("invalid DAG driver args: %w", err)
		}
	}()
	if opts.Container != nil {
		return fmt.Errorf("container spec is unnecessary")
	}
	return validateNonRoot(opts)
}

func validateNonRoot(opts common.Options) error {
	if opts.PipelineName == "" {
		return fmt.Errorf("pipeline name is required")
	}
	if opts.Run.GetRunId() == "" {
		return fmt.Errorf("KFP run ID is required")
	}
	if opts.Component == nil {
		return fmt.Errorf("component spec is required")
	}
	if opts.Task.GetTaskInfo().GetName() == "" {
		return fmt.Errorf("task spec is required")
	}
	if opts.RuntimeConfig != nil {
		return fmt.Errorf("runtime config is unnecessary")
	}
	if opts.ParentTask != nil && opts.ParentTask.GetTaskId() == "" {
		return fmt.Errorf("parent task is required")
	}
	if opts.ParentTask.GetScopePath() == "" {
		return fmt.Errorf("parent task scope path is required for DAG")
	}
	return nil
}

// handleInputTaskParametersCreation creates a new PipelineTaskDetail_InputOutputs_IOParameter
// for each parameter in the executor input.
func handleInputTaskParametersCreation(
	parameterMetadata []resolver.ParameterMetadata,
	task *apiV2beta1.PipelineTaskDetail,
) (*apiV2beta1.PipelineTaskDetail, error) {
	if task == nil {
		return nil, fmt.Errorf("task is nil")
	}
	if task.Inputs == nil {
		task.Inputs = &apiV2beta1.PipelineTaskDetail_InputOutputs{
			Parameters: []*apiV2beta1.PipelineTaskDetail_InputOutputs_IOParameter{},
		}
	} else if task.Inputs.Parameters == nil {
		task.Inputs.Parameters = []*apiV2beta1.PipelineTaskDetail_InputOutputs_IOParameter{}
	}

	for _, pm := range parameterMetadata {
		task.Inputs.Parameters = append(task.Inputs.Parameters, pm.ParameterIO)
	}
	return task, nil
}

// handleInputTaskArtifactsCreation creates a new ArtifactTask for each input artifact.
// The artifactsTasks are created as input artifacts. This allows KFP backend to
// list input artifacts for this task. Parameters do not require this additional overhead
// because parameters are stored in the task itself.
func handleInputTaskArtifactsCreation(
	ctx context.Context,
	opts common.Options,
	artifactMetadata []resolver.ArtifactMetadata,
	task *apiV2beta1.PipelineTaskDetail,
	kfpAPI kfpapi.API,
) error {
	var artifactTasks []*apiV2beta1.ArtifactTask
	for _, am := range artifactMetadata {
		for _, artifact := range am.ArtifactIO.Artifacts {
			if artifact.ArtifactId == "" {
				return fmt.Errorf("artifact id is required")
			}
			artifactTasks = append(artifactTasks, &apiV2beta1.ArtifactTask{
				ArtifactId: artifact.ArtifactId,
				RunId:      opts.Run.GetRunId(),
				TaskId:     task.TaskId,
				Type:       am.ArtifactIO.Type,
				Producer:   am.ArtifactIO.Producer,
				Key:        am.ArtifactIO.ArtifactKey,
			})
		}
	}
	if len(artifactTasks) > 0 {
		request := apiV2beta1.CreateArtifactTasksBulkRequest{ArtifactTasks: artifactTasks}
		_, err := kfpAPI.CreateArtifactTasks(ctx, &request)
		if err != nil {
			return err
		}
	}
	return nil
}
