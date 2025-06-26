// Copyright 2025 The Kubeflow Authors
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

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/expression"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	"google.golang.org/protobuf/types/known/structpb"
)

func validateDAG(opts Options) (err error) {
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

func DAG(ctx context.Context, opts Options, mlmd *metadata.Client) (execution *Execution, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("driver.DAG(%s) failed: %w", opts.info(), err)
		}
	}()
	b, _ := json.Marshal(opts)
	glog.V(4).Info("DAG opts: ", string(b))
	err = validateDAG(opts)
	if err != nil {
		return nil, err
	}
	var iterationIndex *int
	if opts.IterationIndex >= 0 {
		index := opts.IterationIndex
		iterationIndex = &index
	}
	// TODO(Bobgy): there's no need to pass any parameters, because pipeline
	// and pipeline run context have been created by root DAG driver.
	pipeline, err := mlmd.GetPipeline(ctx, opts.PipelineName, opts.RunID, "", "", "", "")
	if err != nil {
		return nil, err
	}
	dag, err := mlmd.GetDAG(ctx, opts.DAGExecutionID)
	if err != nil {
		return nil, err
	}
	glog.Infof("parent DAG: %+v", dag.Execution)
	expr, err := expression.New()
	if err != nil {
		return nil, err
	}
	inputs, err := resolveInputs(ctx, dag, iterationIndex, pipeline, opts, mlmd, expr)
	if err != nil {
		return nil, err
	}
	executorInput := &pipelinespec.ExecutorInput{
		Inputs: inputs,
	}
	glog.Infof("executorInput value: %+v", executorInput)
	execution = &Execution{ExecutorInput: executorInput}
	condition := opts.Task.GetTriggerPolicy().GetCondition()
	if condition != "" {
		willTrigger, err := expr.Condition(executorInput, condition)
		if err != nil {
			return execution, err
		}
		execution.Condition = &willTrigger
	}
	ecfg, err := metadata.GenerateExecutionConfig(executorInput)
	if err != nil {
		return execution, err
	}

	// Set task name to display name if not specified. This is the case of
	// specialty tasks such as OneOfs and ParallelFors where there are not
	// explicit dag tasks defined in the pipeline, but rather generated at
	// compile time and assigned a display name.
	taskName := opts.Task.GetTaskInfo().GetTaskName()
	if taskName == "" {
		taskName = opts.Task.GetTaskInfo().GetName()
	}
	ecfg.TaskName = taskName
	ecfg.DisplayName = opts.Task.GetTaskInfo().GetName()
	ecfg.ExecutionType = metadata.DagExecutionTypeName
	ecfg.ParentDagID = dag.Execution.GetID()
	ecfg.IterationIndex = iterationIndex
	ecfg.NotTriggered = !execution.WillTrigger()

	// Handle writing output parameters to MLMD.
	ecfg.OutputParameters = opts.Component.GetDag().GetOutputs().GetParameters()
	glog.V(4).Info("outputParameters: ", ecfg.OutputParameters)

	// Handle writing output artifacts to MLMD.
	ecfg.OutputArtifacts = opts.Component.GetDag().GetOutputs().GetArtifacts()
	glog.V(4).Info("outputArtifacts: ", ecfg.OutputArtifacts)

	totalDagTasks := len(opts.Component.GetDag().GetTasks())
	ecfg.TotalDagTasks = &totalDagTasks
	glog.V(4).Info("totalDagTasks: ", *ecfg.TotalDagTasks)

	if opts.Task.GetArtifactIterator() != nil {
		return execution, fmt.Errorf("ArtifactIterator is not implemented")
	}
	isIterator := opts.Task.GetParameterIterator() != nil && opts.IterationIndex < 0
	// Fan out iterations
	if execution.WillTrigger() && isIterator {
		iterator := opts.Task.GetParameterIterator()
		report := func(err error) error {
			return fmt.Errorf("iterating on item input %q failed: %w", iterator.GetItemInput(), err)
		}
		// Check the items type of parameterIterator:
		// It can be "inputParameter" or "Raw"
		var value *structpb.Value
		switch iterator.GetItems().GetKind().(type) {
		case *pipelinespec.ParameterIteratorSpec_ItemsSpec_InputParameter:
			var ok bool
			value, ok = executorInput.GetInputs().GetParameterValues()[iterator.GetItems().GetInputParameter()]
			if !ok {
				return execution, report(fmt.Errorf("cannot find input parameter"))
			}
		case *pipelinespec.ParameterIteratorSpec_ItemsSpec_Raw:
			value_raw := iterator.GetItems().GetRaw()
			var unmarshalled_raw interface{}
			err = json.Unmarshal([]byte(value_raw), &unmarshalled_raw)
			if err != nil {
				return execution, fmt.Errorf("error unmarshall raw string: %q", err)
			}
			value, err = structpb.NewValue(unmarshalled_raw)
			if err != nil {
				return execution, fmt.Errorf("error converting unmarshalled raw string into protobuf Value type: %q", err)
			}
			// Add the raw input to the executor input
			execution.ExecutorInput.Inputs.ParameterValues[iterator.GetItemInput()] = value
		default:
			return execution, fmt.Errorf("cannot find parameter iterator")
		}
		items, err := getItems(value)
		if err != nil {
			return execution, report(err)
		}
		count := len(items)
		ecfg.IterationCount = &count
		execution.IterationCount = &count
	}

	glog.V(4).Info("pipeline: ", pipeline)
	b, _ = json.Marshal(*ecfg)
	glog.V(4).Info("ecfg: ", string(b))
	glog.V(4).Infof("dag: %v", dag)

	// TODO(Bobgy): change execution state to pending, because this is driver, execution hasn't started.
	createdExecution, err := mlmd.CreateExecution(ctx, pipeline, ecfg)
	if err != nil {
		return execution, err
	}
	glog.Infof("Created execution: %s", createdExecution)
	execution.ID = createdExecution.GetID()
	return execution, nil
}
