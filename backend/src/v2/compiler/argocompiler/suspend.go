// Copyright 2024 The Kubeflow Authors
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
	"encoding/json"
	"fmt"

	wfapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

// humanInputParamMeta holds per-parameter metadata embedded by the SDK in the
// args[0] field of the sentinel container spec.
type humanInputParamMeta struct {
	Description string   `json:"description"`
	Default     string   `json:"default"`
	Enum        []string `json:"enum"`
}

// humanInputTask builds the single Argo DAGTask that references the suspend
// template for a human-input component.  It also records the task in
// c.humanInputComponents so that downstream tasks can have their inputs
// rewritten to use Argo parameter substitution instead of an MLMD lookup.
//
// Parameters:
//   - taskName: the logical task name within the parent DAG (e.g. "approval-gate")
//   - componentName: the component key used to build the suspend template name
//   - container: the sentinel container spec whose args[0] carries JSON metadata
func (c *workflowCompiler) humanInputTask(
	taskName string,
	componentName string,
	container *pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec,
) (*wfapi.DAGTask, error) {
	paramsMeta, err := decodeHumanInputArgs(container)
	if err != nil {
		return nil, fmt.Errorf("task %q: decoding human-input metadata: %w", taskName, err)
	}

	tmplName, err := c.addHumanInputTemplate(componentName, paramsMeta)
	if err != nil {
		return nil, fmt.Errorf("task %q: adding suspend template: %w", taskName, err)
	}

	// Record the task (by task name, matching producer_task in downstream specs).
	outputParams := make([]string, 0, len(paramsMeta))
	for paramName := range paramsMeta {
		outputParams = append(outputParams, paramName)
	}
	c.humanInputComponents[taskName] = outputParams

	return &wfapi.DAGTask{
		Name:     taskName,
		Template: tmplName,
	}, nil
}

// addHumanInputTemplate registers a suspend template whose output parameters
// are marked as human-supplied.  If a template with the same name already
// exists it is reused (idempotent).
//
// The suspend template carries:
//   - inputs.parameters: one entry per output parameter with display metadata
//     (description, enum, default) for the UI.
//   - outputs.parameters: one entry per output parameter with
//     valueFrom.supplied set, signaling to Argo that a human must provide
//     the value before the workflow can resume.
func (c *workflowCompiler) addHumanInputTemplate(componentName string, paramsMeta map[string]humanInputParamMeta) (string, error) {
	tmplName := "system-human-input-" + componentName
	if _, ok := c.templates[tmplName]; ok {
		return tmplName, nil
	}

	var inputParams []wfapi.Parameter
	var outputParams []wfapi.Parameter

	for paramName, meta := range paramsMeta {
		var enumValues []wfapi.AnyString
		for _, v := range meta.Enum {
			enumValues = append(enumValues, *wfapi.AnyStringPtr(v))
		}

		desc := meta.Description
		inputParam := wfapi.Parameter{
			Name:        paramName,
			Description: wfapi.AnyStringPtr(desc),
		}
		if meta.Default != "" {
			inputParam.Default = wfapi.AnyStringPtr(meta.Default)
		}
		if len(enumValues) > 0 {
			inputParam.Enum = enumValues
		}
		inputParams = append(inputParams, inputParam)

		outputParams = append(outputParams, wfapi.Parameter{
			Name: paramName,
			ValueFrom: &wfapi.ValueFrom{
				Supplied: &wfapi.SuppliedValueFrom{},
			},
		})
	}

	tmpl := &wfapi.Template{
		Name:    tmplName,
		Suspend: &wfapi.SuspendTemplate{},
		Inputs: wfapi.Inputs{
			Parameters: inputParams,
		},
		Outputs: wfapi.Outputs{
			Parameters: outputParams,
		},
	}
	c.wf.Spec.Templates = append(c.wf.Spec.Templates, *tmpl)
	c.templates[tmplName] = tmpl
	return tmplName, nil
}

// decodeHumanInputArgs parses the JSON parameter-metadata blob that the Python
// SDK stores in args[0] of the sentinel container spec.
//
// The expected format is:
//
//	{"<param>": {"description": "...", "default": "...", "enum": [...]}, ...}
func decodeHumanInputArgs(container *pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec) (map[string]humanInputParamMeta, error) {
	if len(container.GetArgs()) == 0 {
		return nil, fmt.Errorf("human-input sentinel container has no args; expected JSON metadata in args[0]")
	}
	raw := container.GetArgs()[0]
	var meta map[string]humanInputParamMeta
	if err := json.Unmarshal([]byte(raw), &meta); err != nil {
		return nil, fmt.Errorf("parsing human-input metadata JSON: %w", err)
	}
	return meta, nil
}

// rewriteHumanInputTaskSpec returns a modified copy of taskSpec where every
// input parameter sourced from a human-input task's output is replaced by a
// runtime_value containing an Argo template expression.  This lets the
// downstream KFP driver receive the human-supplied value as a concrete constant
// (after Argo resolves the template expression) rather than attempting an MLMD
// lookup.
//
// For example, an input like:
//
//	{"task_output_parameter": {"producerTask":"approval-gate","outputParameterKey":"decision"}}
//
// becomes:
//
//	{"runtime_value": {"constant": "{{tasks.approval-gate.outputs.parameters.decision}}"}}
//
// Argo substitutes the expression before the driver container starts, so the
// driver receives the resolved string directly as a runtime_value constant.
func (c *workflowCompiler) rewriteHumanInputTaskSpec(taskSpec *pipelinespec.PipelineTaskSpec) (*pipelinespec.PipelineTaskSpec, error) {
	if taskSpec.GetInputs() == nil {
		return taskSpec, nil
	}

	// Fast path: skip clone if no inputs reference a human-input task.
	needsRewrite := false
	for _, param := range taskSpec.GetInputs().GetParameters() {
		if v, ok := param.GetKind().(*pipelinespec.TaskInputsSpec_InputParameterSpec_TaskOutputParameter); ok {
			if _, isHI := c.humanInputComponents[v.TaskOutputParameter.GetProducerTask()]; isHI {
				needsRewrite = true
				break
			}
		}
	}
	if !needsRewrite {
		return taskSpec, nil
	}

	// Clone via JSON round-trip to avoid mutating the shared proto.
	cloned, err := cloneTaskSpec(taskSpec)
	if err != nil {
		return nil, err
	}

	for paramName, param := range cloned.GetInputs().GetParameters() {
		v, ok := param.GetKind().(*pipelinespec.TaskInputsSpec_InputParameterSpec_TaskOutputParameter)
		if !ok {
			continue
		}
		producerTask := v.TaskOutputParameter.GetProducerTask()
		outputKey := v.TaskOutputParameter.GetOutputParameterKey()
		if _, isHI := c.humanInputComponents[producerTask]; !isHI {
			continue
		}
		// Replace with an Argo template expression that Argo resolves before
		// the driver container starts, so the driver sees the concrete value.
		argoRef := taskOutputParameter(producerTask, outputKey)
		param.Kind = &pipelinespec.TaskInputsSpec_InputParameterSpec_RuntimeValue{
			RuntimeValue: &pipelinespec.ValueOrRuntimeParameter{
				Value: &pipelinespec.ValueOrRuntimeParameter_Constant{
					Constant: structpb.NewStringValue(argoRef),
				},
			},
		}
		cloned.GetInputs().GetParameters()[paramName] = param
	}
	return cloned, nil
}

// cloneTaskSpec makes a deep copy of a PipelineTaskSpec via JSON round-trip.
func cloneTaskSpec(src *pipelinespec.PipelineTaskSpec) (*pipelinespec.PipelineTaskSpec, error) {
	raw, err := stablyMarshalJSON(src)
	if err != nil {
		return nil, fmt.Errorf("cloning task spec (marshal): %w", err)
	}
	dst := &pipelinespec.PipelineTaskSpec{}
	if err := protojson.Unmarshal([]byte(raw), dst); err != nil {
		return nil, fmt.Errorf("cloning task spec (unmarshal): %w", err)
	}
	return dst, nil
}

// registerHumanInputTasks scans all tasks in dagSpec and pre-populates
// c.humanInputComponents so that task-spec rewriting for downstream tasks works
// regardless of alphabetical compilation order.
//
// This is a pre-pass run at the start of the DAG() visitor, before the main
// task compilation loop.
func (c *workflowCompiler) registerHumanInputTasks(dagSpec *pipelinespec.DagSpec) {
	for taskName, kfpTask := range dagSpec.GetTasks() {
		compName := kfpTask.GetComponentRef().GetName()
		compSpec, found := c.spec.Components[compName]
		if !found {
			continue
		}
		execLabel, ok := compSpec.GetImplementation().(*pipelinespec.ComponentSpec_ExecutorLabel)
		if !ok {
			continue
		}
		executor, found := c.executors[execLabel.ExecutorLabel]
		if !found {
			continue
		}
		container, ok := executor.GetSpec().(*pipelinespec.PipelineDeploymentConfig_ExecutorSpec_Container)
		if !ok {
			continue
		}
		if container.Container.GetImage() != humanInputSentinelImage {
			continue
		}
		paramsMeta, err := decodeHumanInputArgs(container.Container)
		if err != nil {
			// Will fail properly during the compilation pass; skip here.
			continue
		}
		outputParams := make([]string, 0, len(paramsMeta))
		for paramName := range paramsMeta {
			outputParams = append(outputParams, paramName)
		}
		c.humanInputComponents[taskName] = outputParams
	}
}
