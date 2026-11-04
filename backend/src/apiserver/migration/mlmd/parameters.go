// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mlmd

import (
	"encoding/json"
	"reflect"
	"strconv"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"google.golang.org/protobuf/types/known/structpb"
)

type taskBinding struct {
	task      *pipelinespec.PipelineTaskSpec
	component *pipelinespec.ComponentSpec
}

func parameterValue(rows model.JSONSlice, key string) (any, bool) {
	for _, raw := range rows {
		row := raw.(map[string]any)
		if row["parameterKey"] == key {
			return row["value"], true
		}
	}
	return nil, false
}

func matchesValue(value *structpb.Value, captured any) bool {
	return value != nil && reflect.DeepEqual(value.AsInterface(), captured)
}

// projectParameterOrigins annotates captured values only when the saved
// pipeline binding and the observed value agree.
func projectParameterOrigins(tasks []model.Task, runs map[string]*cacheRun) {
	type scope struct{ run, parent, name string }
	byID := make(map[string]*model.Task, len(tasks))
	siblings := map[scope][]*model.Task{}
	for i := range tasks {
		task := &tasks[i]
		byID[task.UUID] = task
		if task.ParentTaskUUID != nil {
			key := scope{task.RunUUID, *task.ParentTaskUUID, task.Name}
			siblings[key] = append(siblings[key], task)
		}
	}
	bindings := map[string]taskBinding{}
	var bindingFor func(*model.Task, *pipelinespec.PipelineSpec) taskBinding
	bindingFor = func(task *model.Task, spec *pipelinespec.PipelineSpec) taskBinding {
		if binding, found := bindings[task.UUID]; found {
			return binding
		}
		// A malformed cyclic graph cannot establish binding provenance.
		bindings[task.UUID] = taskBinding{}
		if task.ParentTaskUUID == nil {
			if task.Type == model.TaskType(apiv2.PipelineTask_ROOT) {
				bindings[task.UUID] = taskBinding{component: spec.GetRoot()}
			}
			return bindings[task.UUID]
		}
		parent := byID[*task.ParentTaskUUID]
		if parent == nil {
			return taskBinding{}
		}
		owner := bindingFor(parent, spec)
		_, iteration := task.TypeAttrs["iterationIndex"]
		if iteration && task.Name == parent.Name && owner.task.GetParameterIterator() != nil {
			bindings[task.UUID] = owner
			return owner
		}
		child := owner.component.GetDag().GetTasks()[task.Name]
		binding := taskBinding{task: child, component: spec.GetComponents()[child.GetComponentRef().GetName()]}
		bindings[task.UUID] = binding
		return binding
	}
	runtimeValues := map[string]map[string]any{}
	for i := range tasks {
		task := &tasks[i]
		run := runs[task.RunUUID]
		if run == nil || len(task.InputParameters) == 0 {
			continue
		}
		run.prepare()
		binding := bindingFor(task, run.parsed)
		for _, raw := range task.InputParameters {
			row := raw.(map[string]any)
			key, value := row["parameterKey"].(string), row["value"]
			if task.Type == model.TaskType(apiv2.PipelineTask_ROOT) {
				values, loaded := runtimeValues[task.RunUUID]
				if !loaded {
					_ = json.Unmarshal([]byte(run.runtime), &values)
					runtimeValues[task.RunUUID] = values
				}
				if candidate, ok := values[key]; ok && reflect.DeepEqual(candidate, value) {
					row["type"] = "RUNTIME_VALUE_INPUT"
				}
				continue
			}
			inputs := binding.task.GetInputs().GetParameters()
			input := inputs[key]
			if matchesValue(input.GetRuntimeValue().GetConstant(), value) {
				row["type"] = "RUNTIME_VALUE_INPUT"
				continue
			}
			if iterator := binding.task.GetParameterIterator(); iterator != nil && key == iterator.GetItemInput() {
				index, _ := task.TypeAttrs["iterationIndex"].(string)
				position, err := strconv.Atoi(index)
				var items []any
				if err == nil && json.Unmarshal([]byte(iterator.GetItems().GetRaw()), &items) == nil &&
					position >= 0 && position < len(items) && reflect.DeepEqual(items[position], value) {
					row["type"] = "ITERATOR_INPUT"
					row["producer"] = map[string]any{"taskName": task.Name, "iteration": index}
				}
				continue
			}
			if task.ParentTaskUUID != nil {
				parent := byID[*task.ParentTaskUUID]
				if parent != nil && input.GetComponentInputParameter() != "" {
					if candidate, ok := parameterValue(parent.InputParameters, input.GetComponentInputParameter()); ok && reflect.DeepEqual(candidate, value) {
						row["type"] = "COMPONENT_INPUT"
						continue
					}
				}
				if output := input.GetTaskOutputParameter(); output != nil {
					producers := siblings[scope{task.RunUUID, *task.ParentTaskUUID, output.GetProducerTask()}]
					if len(producers) == 1 && producers[0].Type == model.TaskType(apiv2.PipelineTask_RUNTIME) {
						producer := producers[0]
						_, iteration := producer.TypeAttrs["iterationIndex"]
						if candidate, ok := parameterValue(producer.OutputParameters, output.GetOutputParameterKey()); !iteration && ok && reflect.DeepEqual(candidate, value) {
							row["type"] = "TASK_OUTPUT_INPUT"
							row["producer"] = map[string]any{"taskName": producer.Name}
							continue
						}
					}
				}
			}
			if input == nil && matchesValue(binding.component.GetInputDefinitions().GetParameters()[key].GetDefaultValue(), value) {
				row["type"] = "COMPONENT_DEFAULT_INPUT"
			}
		}
	}
}
