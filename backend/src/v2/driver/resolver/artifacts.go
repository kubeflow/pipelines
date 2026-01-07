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

// Package resolver provides parameter and artifact resolution for KFP v2 driver.
package resolver

import (
	"fmt"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/driver/common"
)

func resolveArtifacts(opts common.Options) ([]ArtifactMetadata, error) {
	var artifacts []ArtifactMetadata

	for key, artifactSpec := range opts.Task.GetInputs().GetArtifacts() {
		producerTask, v, ioType, err := resolveInputArtifact(opts, key, artifactSpec, opts.ParentTask.Inputs.GetArtifacts())
		if err != nil {
			return nil, err
		}

		producer := v.GetProducer()
		if producer == nil {
			return nil, fmt.Errorf("producer cannot be nil")
		}

		am := ArtifactMetadata{
			Key:               key,
			InputArtifactSpec: artifactSpec,
			ArtifactIO: &apiv2beta1.PipelineTaskDetail_InputOutputs_IOArtifact{
				Artifacts:   v.Artifacts,
				Type:        ioType,
				ArtifactKey: key,
				Producer:    v.GetProducer(),
			},
		}
		if opts.IterationIndex >= 0 {
			am.ArtifactIO.Producer.Iteration = util.Int64Pointer(int64(opts.IterationIndex))
		}

		// If the artifact is produced by an importer, then we need to register this into the run time artifact's
		// metadata which is later passed as executor input to the executor.
		artifactProducedByImporter := producerTask != nil && producerTask.Type == apiv2beta1.PipelineTaskDetail_IMPORTER
		if artifactProducedByImporter && producerTask.TypeAttributes.GetDownloadToWorkspace() {
			am.DownloadedToWorkSpace = producerTask.TypeAttributes.GetDownloadToWorkspace()
		}
		artifacts = append(artifacts, am)
	}

	return artifacts, nil
}

// resolveInputArtifact resolves the input artifact based on its specification and returns details about its type and producer.
// The function handles artifacts specified as component inputs or task output artifacts and supports error handling.
//
// Returns:
//   - *PipelineTaskDetail: The producer task details, or nil if artifact is a component input
//   - *PipelineTaskDetail_InputOutputs_IOArtifact: The resolved input/output artifact information
//   - IOType: The type of I/O (component input, task output, etc)
//   - error: Error if resolution fails
func resolveInputArtifact(
	opts common.Options,
	name string,
	artifactSpec *pipelinespec.TaskInputsSpec_InputArtifactSpec,
	inputArtifacts []*apiv2beta1.PipelineTaskDetail_InputOutputs_IOArtifact,
) (*apiv2beta1.PipelineTaskDetail, *apiv2beta1.PipelineTaskDetail_InputOutputs_IOArtifact, apiv2beta1.IOType, error) {
	artifactError := func(err error) error {
		return fmt.Errorf("failed to resolve input artifact %s with spec %s: %w", name, artifactSpec, err)
	}
	switch t := artifactSpec.Kind.(type) {
	case *pipelinespec.TaskInputsSpec_InputArtifactSpec_ComponentInputArtifact:
		artifactIO, err := resolveArtifactComponentInputParameter(opts, artifactSpec, inputArtifacts)
		if err != nil {
			return nil, nil, apiv2beta1.IOType_COMPONENT_INPUT, artifactError(err)
		}
		return nil, artifactIO, apiv2beta1.IOType_COMPONENT_INPUT, nil
	case *pipelinespec.TaskInputsSpec_InputArtifactSpec_TaskOutputArtifact:
		producerTask, artifact, err := resolveTaskOutputArtifact(opts, artifactSpec)
		if err != nil {
			return nil, nil, apiv2beta1.IOType_TASK_OUTPUT_INPUT, err
		}
		ioType := apiv2beta1.IOType_TASK_OUTPUT_INPUT
		if artifact.GetType() == apiv2beta1.IOType_COLLECTED_INPUTS {
			ioType = apiv2beta1.IOType_COLLECTED_INPUTS
		}
		return producerTask, artifact, ioType, nil
	default:
		return nil, nil, apiv2beta1.IOType_UNSPECIFIED, artifactError(fmt.Errorf("artifact spec of type %T not implemented yet", t))
	}
}

func resolveArtifactComponentInputParameter(
	opts common.Options,
	artifactSpec *pipelinespec.TaskInputsSpec_InputArtifactSpec,
	inputArtifactsIO []*apiv2beta1.PipelineTaskDetail_InputOutputs_IOArtifact,
) (*apiv2beta1.PipelineTaskDetail_InputOutputs_IOArtifact, error) {
	key := artifactSpec.GetComponentInputArtifact()
	if key == "" {
		return nil, fmt.Errorf("empty component input")
	}

	for _, artifactIO := range inputArtifactsIO {
		ioKey := artifactIO.GetArtifactKey()
		if key == ioKey {
			if !common.IsLoopArgument(key) {
				return artifactIO, nil
			}
			if artifactIO.Producer != nil && artifactIO.Producer.Iteration != nil && *artifactIO.Producer.Iteration == int64(opts.IterationIndex) {
				return artifactIO, nil
			}
			return artifactIO, nil
		}
	}
	return nil, fmt.Errorf("failed to find input artifact %s", key)

}

func resolveTaskOutputArtifact(
	opts common.Options,
	spec *pipelinespec.TaskInputsSpec_InputArtifactSpec,
) (
	*apiv2beta1.PipelineTaskDetail,
	*apiv2beta1.PipelineTaskDetail_InputOutputs_IOArtifact,
	error,
) {
	tasks, err := getSubTasks(opts.ParentTask, opts.Run.Tasks, nil)
	if err != nil {
		return nil, nil, err
	}
	if tasks == nil {
		return nil, nil, fmt.Errorf("failed to get sub tasks for task %s", opts.ParentTask.Name)
	}
	producerTaskAmbiguousName := spec.GetTaskOutputArtifact().GetProducerTask()
	if producerTaskAmbiguousName == "" {
		return nil, nil, fmt.Errorf("producerTask task cannot be empty")
	}
	producerTaskUniqueName := getTaskNameWithTaskID(producerTaskAmbiguousName, opts.ParentTask.GetTaskId())
	if opts.IterationIndex >= 0 {
		producerTaskUniqueName = getTaskNameWithIterationIndex(producerTaskUniqueName, int64(opts.IterationIndex))
	}
	// producerTask is the specific task guaranteed to have the output artifact
	// producerTaskUniqueName may look something like "task_name_a_dag_id_1_idx_0"
	producerTask := tasks[producerTaskUniqueName]
	if producerTask == nil {
		return nil, nil, fmt.Errorf("producerTask task %s not found", producerTaskUniqueName)
	}
	outputKey := spec.GetTaskOutputArtifact().GetOutputArtifactKey()
	outputs := producerTask.GetOutputs().GetArtifacts()
	outputIO, err := findArtifactByProducerKeyInList(outputKey, producerTask.GetName(), outputs)
	if err != nil {
		return nil, nil, err
	}
	if outputIO == nil {
		return nil, nil, fmt.Errorf("output artifact %s not found", outputKey)
	}
	return producerTask, outputIO, nil
}

// resolveArtifactIterator handles Artifact Iterator Input resolution
func resolveArtifactIterator(
	opts common.Options,
	artifacts []ArtifactMetadata,
) ([]ArtifactMetadata, *int, error) {
	artifactIterator := opts.Task.GetArtifactIterator()
	// This should be the key input into the for loop task
	iteratorInputDefinitionKey := artifactIterator.GetItemInput()
	// Used to look up the Artifact from the resolved list
	// The key here should map to a ArtifactMetadata.Key that
	// was resolved in the prior loop.
	sourceInputArtifactKey := artifactIterator.GetItems().GetInputArtifact()
	artifactIO, err := findArtifactByIOKey(sourceInputArtifactKey, artifacts)
	if err != nil {
		return nil, nil, err
	}

	var artifactMetadataList []ArtifactMetadata
	for i, artifact := range artifactIO.Artifacts {
		am := ArtifactMetadata{
			Key: iteratorInputDefinitionKey,
			ArtifactIO: &apiv2beta1.PipelineTaskDetail_InputOutputs_IOArtifact{
				// Iteration over artifact lists is not supported yet.
				Artifacts:   []*apiv2beta1.Artifact{artifact},
				Type:        apiv2beta1.IOType_ITERATOR_INPUT,
				ArtifactKey: iteratorInputDefinitionKey,
				Producer: &apiv2beta1.IOProducer{
					TaskName:  opts.ParentTask.Name,
					Iteration: util.Int64Pointer(int64(i)),
				},
			},
			ArtifactIterator: artifactIterator,
		}
		artifactMetadataList = append(artifactMetadataList, am)
	}

	count := len(artifactIO.Artifacts)
	return artifactMetadataList, &count, nil
}

// generateUniqueTaskName generates a unique task name for a given task.
func generateUniqueTaskName(task, parentTask *apiv2beta1.PipelineTaskDetail) (string, error) {
	if task == nil || task.Name == "" || parentTask == nil {
		return "", fmt.Errorf("parenttask and task can't be nil and task name cannot be empty")
	}
	taskName := getTaskNameWithTaskID(task.Name, parentTask.TaskId)
	if common.IsRuntimeIterationTask(task) {
		if task.TypeAttributes == nil || task.TypeAttributes.IterationIndex == nil {
			return "", fmt.Errorf("iteration index cannot be nil for loop iteration")
		}
		taskName = getTaskNameWithIterationIndex(taskName, *task.TypeAttributes.IterationIndex)
	} else if common.IsRuntimeIterationTask(parentTask) {
		if parentTask.TypeAttributes == nil || parentTask.TypeAttributes.IterationIndex == nil {
			return "", fmt.Errorf("iteration index cannot be nil for loop iteration")
		}
		taskName = getTaskNameWithIterationIndex(taskName, *parentTask.TypeAttributes.IterationIndex)
	}
	return taskName, nil
}

func getChildTasks(
	tasks []*apiv2beta1.PipelineTaskDetail,
	parentTask *apiv2beta1.PipelineTaskDetail,
) (map[string]*apiv2beta1.PipelineTaskDetail, error) {
	if parentTask == nil {
		return nil, fmt.Errorf("parent task cannot be nil")
	}
	var taskMap = make(map[string]*apiv2beta1.PipelineTaskDetail)
	for _, task := range tasks {
		if task.GetParentTaskId() == parentTask.GetTaskId() {
			taskName, err := generateUniqueTaskName(task, parentTask)
			if err != nil {
				return nil, err
			}
			if taskName == "" {
				return nil, fmt.Errorf("task name cannot be empty")
			}
			taskMap[taskName] = task
		}
	}
	return taskMap, nil
}

// getSubTasks creates a map of all subtasks under currentTask.
// The keys of the map are formed by concatenating the task name with the task id.
// If the task is a runtime iteration task, then the key is formed by concatenating
// the task name with the iteration index.
// So you may end up with a map of the form:
//
//	{
//	  "task_name_a_dag_id_1_idx_0": {
//	    ...
//	  },
//	  "task_name_a_dag_id_1_idx_1": {
//	    ...
//	  },
//	  "task_name_b_dag_id_2": {
//	    ...
//	  },
//	},
func getSubTasks(
	currentTask *apiv2beta1.PipelineTaskDetail,
	allRuntasks []*apiv2beta1.PipelineTaskDetail,
	flattenedTasks map[string]*apiv2beta1.PipelineTaskDetail,
) (map[string]*apiv2beta1.PipelineTaskDetail, error) {
	if flattenedTasks == nil {
		flattenedTasks = make(map[string]*apiv2beta1.PipelineTaskDetail)
	}
	taskChildren, err := getChildTasks(allRuntasks, currentTask)
	if err != nil {
		return nil, fmt.Errorf("failed to get child tasks for task %s: %w", currentTask.Name, err)
	}
	for taskName, task := range taskChildren {
		flattenedTasks[taskName] = task
	}
	for _, task := range taskChildren {
		if task.Type != apiv2beta1.PipelineTaskDetail_RUNTIME {
			flattenedTasks, err = getSubTasks(task, allRuntasks, flattenedTasks)
			if err != nil {
				return nil, err
			}
		}
	}
	return flattenedTasks, nil
}

func getTaskNameWithIterationIndex(taskName string, iterationIndex int64) string {
	return fmt.Sprintf("%s_idx_%d", taskName, iterationIndex)
}

func getTaskNameWithTaskID(taskName, taskID string) string {
	return fmt.Sprintf("%s_%s", taskName, taskID)
}

func findArtifactByProducerKeyInList(
	producerKey, producerTaskName string,
	artifactsIO []*apiv2beta1.PipelineTaskDetail_InputOutputs_IOArtifact,
) (*apiv2beta1.PipelineTaskDetail_InputOutputs_IOArtifact, error) {
	var artifactIOList []*apiv2beta1.PipelineTaskDetail_InputOutputs_IOArtifact
	for _, artifactIO := range artifactsIO {
		if artifactIO.GetArtifactKey() == producerKey {
			artifactIOList = append(artifactIOList, artifactIO)
		}
	}
	if len(artifactIOList) == 0 {
		return nil, fmt.Errorf("artifact with producer key %s not found", producerKey)
	}

	// This occurs in the parallelFor case, where multiple iterations resulted in the same
	// producer key.
	isCollection := len(artifactIOList) > 1
	if isCollection {
		var artifacts []*apiv2beta1.Artifact
		for _, artifactIO := range artifactIOList {
			//  Check correctness by validating the type of all parameters
			if artifactIO.Type != apiv2beta1.IOType_ITERATOR_OUTPUT {
				return nil, fmt.Errorf("encountered a non iterator output that has the same producer key (%s)", producerKey)
			}
			// Support for an iterator over list of artifacts is not supported yet.
			artifacts = append(artifacts, artifactIO.Artifacts[0])
		}
		ioType := apiv2beta1.IOType_COLLECTED_INPUTS
		newArtifactIO := &apiv2beta1.PipelineTaskDetail_InputOutputs_IOArtifact{
			Artifacts:   artifacts,
			Type:        ioType,
			ArtifactKey: producerKey,
			// This is unused by the caller
			Producer: &apiv2beta1.IOProducer{
				TaskName: producerTaskName,
			},
		}
		return newArtifactIO, nil
	}
	return artifactIOList[0], nil
}
