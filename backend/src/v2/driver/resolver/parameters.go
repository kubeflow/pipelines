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

package resolver

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/component"
	"github.com/kubeflow/pipelines/backend/src/v2/driver/common"
	"google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

func resolveParameters(opts common.Options) ([]ParameterMetadata, error) {
	var parameters []ParameterMetadata
	for key, paramSpec := range opts.Task.GetInputs().GetParameters() {
		if compParam := opts.Component.GetInputDefinitions().GetParameters()[key]; compParam != nil {
			// Skip resolving dsl.TaskConfig because that information is only available after initPodSpecPatch and
			// extendPodSpecPatch are called.
			if compParam.GetParameterType() == pipelinespec.ParameterType_TASK_CONFIG {
				continue
			}
		}

		v, ioType, err := ResolveInputParameter(opts, paramSpec, opts.ParentTask.Inputs.GetParameters())
		if err != nil {
			if !errors.Is(err, ErrResolvedParameterNull) {
				return nil, err
			}
			componentParam, ok := opts.Component.GetInputDefinitions().GetParameters()[key]
			if ok && componentParam != nil && componentParam.IsOptional {
				// If the resolved parameter was null and the component input parameter is optional,
				// check if there's a default value we should use
				if componentParam.GetDefaultValue() != nil {
					// Add parameter with default value
					pm := ParameterMetadata{
						Key:                key,
						InputParameterSpec: paramSpec,
						ParameterIO: &apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
							Value:        componentParam.GetDefaultValue(),
							Type:         apiv2beta1.IOType_COMPONENT_DEFAULT_INPUT,
							ParameterKey: key,
							Producer: &apiv2beta1.IOProducer{
								TaskName: opts.ParentTask.GetName(),
							},
						},
					}
					if opts.IterationIndex >= 0 {
						pm.ParameterIO.Producer.Iteration = util.Int64Pointer(int64(opts.IterationIndex))
					}
					parameters = append(parameters, pm)
					continue
				}
				// No default value, skip it
				continue
			}
			return nil, err
		}

		producer := v.GetProducer()
		if producer == nil {
			return nil, fmt.Errorf("producer cannot be nil")
		}

		pm := ParameterMetadata{
			Key:                key,
			InputParameterSpec: paramSpec,
			ParameterIO: &apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
				Value:        v.GetValue(),
				Type:         ioType,
				ParameterKey: key,
				Producer:     producer,
			},
		}
		if opts.IterationIndex >= 0 {
			pm.ParameterIO.Producer.Iteration = util.Int64Pointer(int64(opts.IterationIndex))
		}
		parameters = append(parameters, pm)
	}

	// Check for parameters in the Component's InputDefinitions that aren't in the task's inputs
	// and add them with their default values if they have one
	if opts.Component.GetInputDefinitions() != nil {
		// Build a map of existing parameter keys
		existingParams := make(map[string]bool)
		for key := range opts.Task.GetInputs().GetParameters() {
			existingParams[key] = true
		}

		// Find default parameters that aren't already in the task
		for name, paramSpec := range opts.Component.GetInputDefinitions().GetParameters() {
			// Skip if parameter is already in the task's inputs or doesn't have a default value
			if existingParams[name] || paramSpec.GetDefaultValue() == nil {
				continue
			}

			// Skip TASK_CONFIG parameters
			if paramSpec.GetParameterType() == pipelinespec.ParameterType_TASK_CONFIG {
				continue
			}

			// Only add if it's optional
			if !paramSpec.IsOptional {
				continue
			}

			// Add parameter with default value
			pm := ParameterMetadata{
				Key: name,
				ParameterIO: &apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
					Value:        paramSpec.GetDefaultValue(),
					Type:         apiv2beta1.IOType_COMPONENT_DEFAULT_INPUT,
					ParameterKey: name,
					Producer: &apiv2beta1.IOProducer{
						TaskName: opts.ParentTask.GetName(),
					},
				},
			}
			if opts.IterationIndex >= 0 {
				pm.ParameterIO.Producer.Iteration = util.Int64Pointer(int64(opts.IterationIndex))
			}
			parameters = append(parameters, pm)
		}
	}

	return parameters, nil
}

func ResolveInputParameter(
	opts common.Options,
	paramSpec *pipelinespec.TaskInputsSpec_InputParameterSpec,
	inputParams []*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter,
) (*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter, apiv2beta1.IOType, error) {
	switch t := paramSpec.Kind.(type) {
	case *pipelinespec.TaskInputsSpec_InputParameterSpec_ComponentInputParameter:
		glog.V(4).Infof("resolving component input parameter %s", paramSpec.GetComponentInputParameter())
		resolvedInput, err := resolveParameterComponentInputParameter(opts, paramSpec, inputParams)
		if err != nil {
			return nil, apiv2beta1.IOType_COMPONENT_INPUT, err
		}
		return resolvedInput, apiv2beta1.IOType_COMPONENT_INPUT, nil
	case *pipelinespec.TaskInputsSpec_InputParameterSpec_TaskOutputParameter:
		parameter, err := resolveTaskOutputParameter(opts, paramSpec)
		if err != nil {
			return nil, apiv2beta1.IOType_TASK_OUTPUT_INPUT, err
		}
		ioType := apiv2beta1.IOType_TASK_OUTPUT_INPUT
		if parameter.GetType() == apiv2beta1.IOType_COLLECTED_INPUTS {
			ioType = apiv2beta1.IOType_COLLECTED_INPUTS
		}
		return parameter, ioType, nil
	case *pipelinespec.TaskInputsSpec_InputParameterSpec_RuntimeValue:
		glog.V(4).Infof("resolving runtime value %s", paramSpec.GetRuntimeValue().String())
		runtimeValue := paramSpec.GetRuntimeValue()
		switch t := runtimeValue.Value.(type) {
		case *pipelinespec.ValueOrRuntimeParameter_Constant:
			val := runtimeValue.GetConstant()
			valStr := val.GetStringValue()
			var v *structpb.Value
			if strings.Contains(valStr, "{{$.workspace_path}}") {
				v = structpb.NewStringValue(strings.ReplaceAll(valStr, "{{$.workspace_path}}", component.WorkspaceMountPath))
				ioParameter := &apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
					ParameterKey: "",
					Value:        v,
					Producer: &apiv2beta1.IOProducer{
						TaskName: opts.ParentTask.GetName(),
					},
				}
				return ioParameter, apiv2beta1.IOType_RUNTIME_VALUE_INPUT, nil
			}
			switch valStr {
			case "{{$.pipeline_job_name}}":
				v = structpb.NewStringValue(opts.RunDisplayName)
			case "{{$.pipeline_job_resource_name}}":
				v = structpb.NewStringValue(opts.RunName)
			case "{{$.pipeline_job_uuid}}":
				v = structpb.NewStringValue(opts.Run.GetRunId())
			case "{{$.pipeline_task_name}}":
				v = structpb.NewStringValue(opts.TaskName)
			case "{{$.pipeline_task_uuid}}":
				if opts.ParentTask == nil {
					return nil, apiv2beta1.IOType_UNSPECIFIED, fmt.Errorf("parent task should not be nil")
				}
				v = structpb.NewStringValue(opts.ParentTask.GetTaskId())
			default:
				v = val
			}
			// When a constant runtime value is a pipeline channel, then we expect the source value to be found
			// via the pipeline channel's key within this task's inputs.
			isMatch, isPipelineChannel, paramName := common.ParsePipelineParam(v.GetStringValue())
			if isMatch && isPipelineChannel {
				channelParamSpec, ok := opts.Task.Inputs.GetParameters()[paramName]
				if !ok {
					return nil, apiv2beta1.IOType_RUNTIME_VALUE_INPUT, fmt.Errorf("pipeline channel %s not found in task %s", v.GetStringValue(), opts.TaskName)
				}
				return ResolveInputParameter(opts, channelParamSpec, inputParams)
			}

			ioParameter := &apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
				ParameterKey: "",
				Value:        v,
				Producer: &apiv2beta1.IOProducer{
					TaskName: opts.ParentTask.GetName(),
				},
			}
			return ioParameter, apiv2beta1.IOType_RUNTIME_VALUE_INPUT, nil
		default:
			return nil, apiv2beta1.IOType_UNSPECIFIED, paramError(paramSpec, fmt.Errorf("param runtime value spec of type %T not implemented", t))
		}
	case *pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus_:
		value, err := resolveTaskFinalStatus(opts, paramSpec)
		if err != nil {
			return nil, apiv2beta1.IOType_TASK_FINAL_STATUS_OUTPUT, err
		}
		return &apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
			ParameterKey: "status",
			Value:        value,
			Producer: &apiv2beta1.IOProducer{
				TaskName: opts.ParentTask.GetName(),
			},
		}, apiv2beta1.IOType_TASK_FINAL_STATUS_OUTPUT, nil
	default:
		return nil, apiv2beta1.IOType_UNSPECIFIED, paramError(paramSpec, fmt.Errorf("parameter spec of type %T not implemented yet", t))
	}
}

func resolveTaskOutputParameter(
	opts common.Options,
	spec *pipelinespec.TaskInputsSpec_InputParameterSpec,
) (*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter, error) {
	tasks, err := getSubTasks(opts.ParentTask, opts.Run.Tasks, nil)
	if err != nil {
		return nil, err
	}
	if tasks == nil {
		return nil, fmt.Errorf("failed to get sub tasks for task %s", opts.ParentTask.Name)
	}
	producerTaskAmbiguousName := spec.GetTaskOutputParameter().GetProducerTask()
	if producerTaskAmbiguousName == "" {
		return nil, fmt.Errorf("producerTask task cannot be empty")
	}
	producerTaskUniqueName := getTaskNameWithTaskID(producerTaskAmbiguousName, opts.ParentTask.GetTaskId())
	if opts.IterationIndex >= 0 {
		producerTaskUniqueName = getTaskNameWithIterationIndex(producerTaskUniqueName, int64(opts.IterationIndex))
	}
	// producerTask is the specific task guaranteed to have the output parameter
	// producerTaskUniqueName may look something like "task_name_a_dag_id_1_idx_0"
	producerTask := tasks[producerTaskUniqueName]
	if producerTask == nil {
		return nil, fmt.Errorf("producerTask task %s not found", producerTaskUniqueName)
	}
	outputKey := spec.GetTaskOutputParameter().GetOutputParameterKey()
	outputs := producerTask.GetOutputs().GetParameters()
	outputIO, err := findParameterByProducerKeyInList(outputKey, producerTask.GetName(), outputs)
	if err != nil {
		return nil, err
	}
	if outputIO == nil {
		return nil, fmt.Errorf("output parameter %s not found", outputKey)
	}
	return outputIO, nil
}

func resolveParameterComponentInputParameter(
	opts common.Options,
	paramSpec *pipelinespec.TaskInputsSpec_InputParameterSpec,
	inputParams []*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter,
) (*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter, error) {
	paramName := paramSpec.GetComponentInputParameter()
	if paramName == "" {
		return nil, paramError(paramSpec, fmt.Errorf("empty component input"))
	}
	for _, param := range inputParams {
		generateName := param.ParameterKey
		if paramName == generateName {
			if !common.IsLoopArgument(paramName) {
				// This can occur when a runtime config has a "None" optional value.
				// In this case we return "nil" and have the callee handle the
				// ErrResolvedParameterNull case.
				val := param.GetValue()
				if val == nil {
					return nil, ErrResolvedParameterNull
				}
				if _, isNull := val.GetKind().(*structpb.Value_NullValue); isNull {
					return nil, ErrResolvedParameterNull
				}
				return param, nil
			}
			// If the input is a loop argument, we need to check if the iteration index matches the current iteration.
			if param.Producer != nil && param.Producer.Iteration != nil && *param.Producer.Iteration == int64(opts.IterationIndex) {
				return param, nil
			}
		}
	}
	return nil, ErrResolvedParameterNull
}

// resolveParameterIterator handles parameter Iterator Input resolution
func resolveParameterIterator(
	opts common.Options,
	parameters []ParameterMetadata,
) ([]ParameterMetadata, *int, error) {
	var value *structpb.Value
	var iteratorInputDefinitionKey string
	var iterator = opts.Task.GetParameterIterator()
	switch iterator.GetItems().GetKind().(type) {
	case *pipelinespec.ParameterIteratorSpec_ItemsSpec_InputParameter:
		// This should be the key input into the for loop task
		iteratorInputDefinitionKey = iterator.GetItemInput()
		// Used to look up the Parameter from the resolved list
		// The key here should map to a ParameterMetadata.Key that
		// was resolved in the prior loop.
		sourceInputParameterKey := iterator.GetItems().GetInputParameter()

		// Determine if the parameter is a parameter or an artifact

		var err error
		parameterIO, err := findParameterByIOKey(sourceInputParameterKey, parameters)
		if err != nil {
			return nil, nil, err
		}
		value = parameterIO.GetValue()
	case *pipelinespec.ParameterIteratorSpec_ItemsSpec_Raw:
		valueRaw := iterator.GetItems().GetRaw()
		var unmarshalledRaw interface{}
		err := json.Unmarshal([]byte(valueRaw), &unmarshalledRaw)
		if err != nil {
			return nil, nil, fmt.Errorf("error unmarshall raw string: %q", err)
		}
		value, err = structpb.NewValue(unmarshalledRaw)
		if err != nil {
			return nil, nil, fmt.Errorf("error converting unmarshalled raw string into protobuf Value type: %q", err)
		}
		iteratorInputDefinitionKey = iterator.GetItemInput()

	default:
		return nil, nil, fmt.Errorf("cannot find parameter iterator")
	}

	items, err := getItems(value)
	if err != nil {
		return nil, nil, err
	}

	var parameterMetadataList []ParameterMetadata
	for i, item := range items {
		pm := ParameterMetadata{
			Key: iteratorInputDefinitionKey,
			ParameterIO: &apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
				Value:        item,
				Type:         apiv2beta1.IOType_ITERATOR_INPUT,
				ParameterKey: iteratorInputDefinitionKey,
				Producer: &apiv2beta1.IOProducer{
					TaskName:  opts.TaskName,
					Iteration: util.Int64Pointer(int64(i)),
				},
			},
			ParameterIterator: iterator,
		}
		parameterMetadataList = append(parameterMetadataList, pm)
	}
	count := len(items)
	return parameterMetadataList, &count, nil
}

func resolveTaskFinalStatus(opts common.Options,
	spec *pipelinespec.TaskInputsSpec_InputParameterSpec,
) (*structpb.Value, error) {
	tasks, err := getSubTasks(opts.ParentTask, opts.Run.Tasks, nil)
	if err != nil {
		return nil, err
	}
	if tasks == nil {
		return nil, fmt.Errorf("failed to get sub tasks for task %s", opts.ParentTask.Name)
	}
	producerTaskAmbiguousName := spec.GetTaskFinalStatus().GetProducerTask()
	if producerTaskAmbiguousName == "" {
		return nil, fmt.Errorf("producerTask task cannot be empty")
	}
	producerTaskUniqueName := getTaskNameWithTaskID(producerTaskAmbiguousName, opts.ParentTask.GetTaskId())
	producer, ok := tasks[producerTaskUniqueName]

	if len(opts.Task.DependentTasks) == 0 {
		return nil, fmt.Errorf("task %v has no dependent tasks", opts.Task.TaskInfo.GetName())
	}
	if !ok {
		return nil, fmt.Errorf("producer task, %v, not in tasks", producer.GetName())
	}

	finalStatus := pipelinespec.PipelineTaskFinalStatus{
		State:                   producer.GetState().String(),
		PipelineTaskName:        producer.GetName(),
		PipelineJobResourceName: opts.RunName,
		Error: &status.Status{
			Message: producer.GetStatusMetadata().GetMessage(),
			Code:    int32(producer.GetState().Number()),
		},
	}
	finalStatusJSON, err := protojson.Marshal(&finalStatus)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal PipelineTaskFinalStatus: %w", err)
	}

	var finalStatusMap map[string]interface{}
	if err := json.Unmarshal(finalStatusJSON, &finalStatusMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON of PipelineTaskFinalStatus: %w", err)
	}

	finalStatusStruct, err := structpb.NewStruct(finalStatusMap)
	if err != nil {
		return nil, fmt.Errorf("failed to create structpb.Struct: %w", err)
	}

	return structpb.NewStructValue(finalStatusStruct), nil
}

// getItems iteration items from a structpb.Value.
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

// ToListValue will convert []*structpb.Value to a *structpb.Value_ListValue
func ToListValue(items []*structpb.Value) *structpb.Value {
	listValue := structpb.Value{}
	listValue.Kind = &structpb.Value_ListValue{
		ListValue: &structpb.ListValue{
			Values: items,
		},
	}
	return &listValue
}

// ResolveParameterOrPipelineChannel resolves a parameter or pipeline channel value using the executor input.
func ResolveParameterOrPipelineChannel(parameterValueOrPipelineChannel string, executorInput *pipelinespec.ExecutorInput) (string, error) {
	isMatch, isPipelineChannel, paramName := common.ParsePipelineParam(parameterValueOrPipelineChannel)
	if isMatch && isPipelineChannel {
		value, ok := executorInput.GetInputs().GetParameterValues()[paramName]
		if !ok {
			return "", fmt.Errorf("pipeline channel %s not found in executorinput", parameterValueOrPipelineChannel)
		}
		return value.GetStringValue(), nil
	}

	return parameterValueOrPipelineChannel, nil
}

func ResolveK8sJSONParameter[k8sResource any](
	opts common.Options,
	parameter *pipelinespec.TaskInputsSpec_InputParameterSpec,
	params []*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter,
	res *k8sResource,
) error {

	resolvedParam, _, err := ResolveInputParameter(opts, parameter, params)
	if err != nil {
		return fmt.Errorf("failed to resolve k8s parameter: %w", err)
	}
	if resolvedParam == nil || resolvedParam.GetValue() == nil || resolvedParam.GetValue().GetStructValue() == nil {
		return fmt.Errorf("resolved k8s parameter is nil")
	}
	paramJSON, err := resolvedParam.GetValue().GetStructValue().MarshalJSON()
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

// ResolveInputParameterStr is like ResolveInputParameter but returns an error if the resolved value is not a non-empty
// string.
func ResolveInputParameterStr(
	opts common.Options,
	parameter *pipelinespec.TaskInputsSpec_InputParameterSpec,
	params []*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter) (*structpb.Value, error) {

	val, _, err := ResolveInputParameter(opts, parameter, params)
	if err != nil || val == nil || val.GetValue() == nil {
		return nil, fmt.Errorf("failed to resolve input parameter. Error: %w", err)
	}
	if typedVal, ok := val.GetValue().GetKind().(*structpb.Value_StringValue); ok && typedVal != nil {
		if typedVal.StringValue == "" {
			return nil, fmt.Errorf("resolving input parameter with spec %s. Expected a non-empty string", parameter.String())
		}
	} else {
		return nil, fmt.Errorf(
			"resolving input parameter with spec %s. Expected a string but got: %T",
			parameter.String(),
			val.GetValue().GetKind(),
		)
	}

	return val.GetValue(), nil
}

func findParameterByProducerKeyInList(
	producerKey, producerTaskName string,
	parametersIO []*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter,
) (*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter, error) {
	var parameterIOList []*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter
	for _, parameterIO := range parametersIO {
		if parameterIO.GetParameterKey() == producerKey {
			parameterIOList = append(parameterIOList, parameterIO)
		}
	}
	if len(parameterIOList) == 0 {
		return nil, fmt.Errorf("parameter with producer key %s not found", producerKey)
	}

	// This occurs in the parallelFor case, where multiple iterations resulted in the same
	// producer key.
	isCollection := len(parameterIOList) > 1
	if isCollection {
		var parameterValues []*structpb.Value
		for _, parameterIO := range parameterIOList {
			//  Check correctness by validating the type of all parameters
			if parameterIO.Type != apiv2beta1.IOType_ITERATOR_OUTPUT {
				return nil, fmt.Errorf("encountered a non iterator output that has the same producer key (%s)", producerKey)
			}
			// Support for an iterator over list of parameters is not supported yet.
			parameterValues = append(parameterValues, parameterIO.GetValue())
		}
		ioType := apiv2beta1.IOType_COLLECTED_INPUTS
		newParameterIO := &apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
			Value:        ToListValue(parameterValues),
			Type:         ioType,
			ParameterKey: producerKey,
			Producer: &apiv2beta1.IOProducer{
				TaskName: producerTaskName,
			},
		}
		return newParameterIO, nil
	}
	return parameterIOList[0], nil
}
