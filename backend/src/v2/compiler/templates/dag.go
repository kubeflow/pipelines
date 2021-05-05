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
	"fmt"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	workflowapi "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/golang/protobuf/jsonpb"
	pb "github.com/kubeflow/pipelines/api/v2alpha1/go"
	"github.com/kubeflow/pipelines/backend/src/v2/compiler/util"
	"github.com/pkg/errors"
)

const (
	// Dag Inputs
	DagParamContextName = paramPrefixKfpInternal + "context-name"
)

// Const can not be refered via string pointers, so we use var here.
var (
	argoVariablePodName = "{{pod.name}}"
)

type DagArgs struct {
	Tasks                *[]*pb.PipelineTaskSpec
	DeploymentConfig     *pb.PipelineDeploymentConfig
	ExecutorTemplateName string
}

type taskData struct {
	task *pb.PipelineTaskSpec
	// we may need more stuff put here
}

func Dag(args *DagArgs) (*workflowapi.Template, error) {
	// convenient local variables
	tasks := args.Tasks
	deploymentConfig := args.DeploymentConfig
	executors := deploymentConfig.GetExecutors()

	var dag workflowapi.Template
	dag.Name = getUniqueDagName()
	dag.DAG = &workflowapi.DAGTemplate{}
	dag.Inputs.Parameters = []workflowapi.Parameter{
		{Name: DagParamContextName},
	}
	taskMap := make(map[string]*taskData)
	for index, task := range *tasks {
		name := task.GetTaskInfo().GetName()
		if name == "" {
			return nil, errors.Errorf("Task name is empty for task with index %v and spec: %s", index, task.String())
		}
		sanitizedName := util.SanitizeK8sName(name)
		if taskMap[sanitizedName] != nil {
			return nil, errors.Errorf("Two tasks '%s' and '%s' in the DAG has the same sanitized name: %s", taskMap[sanitizedName].task.GetTaskInfo().GetName(), name, sanitizedName)
		}
		taskMap[sanitizedName] = &taskData{
			task: task,
		}
	}

	// generate tasks
	for _, task := range *tasks {
		// TODO(Bobgy): Move executor template generation out as a separate file.
		executorLabel := task.GetExecutorLabel()
		executorSpec := executors[executorLabel]
		if executorSpec == nil {
			return nil, errors.Errorf("Executor with label '%v' cannot be found in deployment config", executorLabel)
		}
		var executor workflowapi.Template
		executor.Name = util.SanitizeK8sName(executorLabel)

		argoTaskName := util.SanitizeK8sName(task.GetTaskInfo().GetName())
		marshaler := &jsonpb.Marshaler{}
		taskSpecInJson, err := marshaler.MarshalToString(task)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to marshal task spec to JSON: %s", task.String())
		}
		executorSpecInJson, err := marshaler.MarshalToString(executorSpec)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to marshal executor spec to JSON: %s", executorSpec.String())
		}
		// TODO(Bobgy): Task outputs spec is deprecated. Get outputs spec from component output spec once data is ready.
		outputsSpec := task.GetOutputs()
		if outputsSpec == nil {
			// For tasks without outputs spec, marshal an emtpy outputs spec.
			outputsSpec = &pb.TaskOutputsSpec{}
		}
		outputsSpecInJson, err := marshaler.MarshalToString(outputsSpec)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to marshal outputs spec to JSON: %s", task.GetOutputs().String())
		}
		parentContextNameValue := "{{inputs.parameters." + DagParamContextName + "}}"

		dependencies, err := getTaskDependencies(task.GetInputs())
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to get task dependencies for task: %s", task.String())
		}
		// Convert dependency names to sanitized ones and check validity.
		for index, dependency := range *dependencies {
			sanitizedDependencyName := util.SanitizeK8sName(dependency)
			upstreamTask := taskMap[sanitizedDependencyName]
			if upstreamTask == nil {
				return nil, errors.Wrapf(err, "Failed to find dependency '%s' for task: %s", dependency, task.String())
			}
			upstreamTaskName := upstreamTask.task.GetTaskInfo().GetName()
			if upstreamTaskName != dependency {
				return nil, errors.Wrapf(err, "Found slightly different dependency task name '%s', expecting '%s' for task: %s", upstreamTaskName, dependency, task.String())
			}
			(*dependencies)[index] = sanitizedDependencyName
		}

		dag.DAG.Tasks = append(
			dag.DAG.Tasks,
			workflowapi.DAGTask{
				Name:         argoTaskName,
				Template:     args.ExecutorTemplateName,
				Dependencies: *dependencies,
				Arguments: workflowapi.Arguments{
					Parameters: []workflowapi.Parameter{
						{Name: ExecutorParamTaskSpec, Value: v1alpha1.AnyStringPtr(taskSpecInJson)},
						{Name: ExecutorParamContextName, Value: v1alpha1.AnyStringPtr(parentContextNameValue)},
						{Name: ExecutorParamExecutorSpec, Value: v1alpha1.AnyStringPtr(executorSpecInJson)},
						{Name: ExecutorParamOutputsSpec, Value: v1alpha1.AnyStringPtr(outputsSpecInJson)},
					},
				},
			})
	}
	return &dag, nil
}

func getTaskDependencies(inputsSpec *pb.TaskInputsSpec) (*[]string, error) {
	dependencies := make(map[string]bool)
	for _, parameter := range inputsSpec.GetParameters() {
		if parameter.GetTaskOutputParameter() != nil {
			producerTask := parameter.GetTaskOutputParameter().GetProducerTask()
			if producerTask == "" {
				return nil, errors.Errorf("Invalid task input parameter spec, producer task is empty: %v", parameter.String())
			}
			dependencies[producerTask] = true
		}
	}
	dependencyList := make([]string, 0, len(dependencies))
	for dependency := range dependencies {
		dependencyList = append(dependencyList, dependency)
	}
	return &dependencyList, nil
}

// TODO(Bobgy): figure out a better way to generate unique names
var globalDagCount = 0

func getUniqueDagName() string {
	globalDagCount = globalDagCount + 1
	return fmt.Sprintf("dag-%x", globalDagCount)
}
