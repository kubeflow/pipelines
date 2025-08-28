// Copyright 2018-2023 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import (
	"fmt"
	"github.com/onsi/gomega"
	"maps"
	"path/filepath"
	"sort"
	"time"

	v1 "k8s.io/api/core/v1"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_model"
	workflow_utils "github.com/kubeflow/pipelines/backend/test/compiler/utils"

	model "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_model"
	. "github.com/kubeflow/pipelines/backend/test/v2/api/constants"
	"github.com/kubeflow/pipelines/backend/test/v2/api/logger"
	utils "github.com/kubeflow/pipelines/backend/test/v2/api/utils"
	. "github.com/onsi/ginkgo/v2"
)

// ####################################################################################################################################################################
// ################################################################### SET AND TEARDOWN ################################################################################
// ####################################################################################################################################################################

var _ = BeforeEach(func() {
	logger.Log("################### Setup before each Pipeline Upload test #####################")
	expectedPipeline = new(model.V2beta1Pipeline)
	expectedPipeline.CreatedAt = testStartTime
})

// ####################################################################################################################################################################
// ################################################################### TESTS ################################################################################
// ####################################################################################################################################################################

// ################################################################################################################
// ########################################################## POSITIVE TESTS ######################################
// ################################################################################################################

var _ = Describe("Upload and Verify Pipeline Run >", Label("Positive", "E2E", FullRegression), func() {

	Context("Upload a pipeline file, run it and verify that pipeline run succeeds >", func() {
		var pipelineDir = "valid"
		var compiledWorkflowsDir = "compiled-workflows"
		criticalPipelineFiles := utils.GetListOfFileInADir(filepath.Join(pipelineFilesRootDir, pipelineDir))
		for _, pipelineFile := range criticalPipelineFiles {
			It(fmt.Sprintf("Upload %s pipeline", pipelineFile), func() {
				pipelineFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, pipelineFile)
				logger.Log("Uploading pipeline file %s", pipelineFile)
				uploadedPipeline := uploadPipelineAndVerify(pipelineDir, pipelineFile, &pipelineGeneratedName)
				logger.Log("Upload of pipeline file '%s' successful", pipelineFile)
				uploadedPipelineVersion := utils.GetLatestPipelineVersion(pipelineClient, &uploadedPipeline.PipelineID)
				pipelineRuntimeInputs := utils.GetPipelineRunTimeInputs(pipelineFilePath)
				logger.Log("Create run for pipeline with id: '%s' and name: '%s'", uploadedPipeline.PipelineID, uploadedPipeline.DisplayName)
				uploadedPipelineRun := createPipelineRun(&uploadedPipeline.PipelineID, &uploadedPipelineVersion.PipelineVersionID, nil, pipelineRuntimeInputs)
				logger.Log("Created Pipeline Run with id: %s for pipeline with id: %s", uploadedPipelineRun.RunID, uploadedPipeline.PipelineID)
				utils.WaitForRunToBeInState(runClient, &uploadedPipelineRun.RunID, []run_model.V2beta1RuntimeState{run_model.V2beta1RuntimeStateRUNNING}, nil)
				logger.Log("Created Pipeline Run with id: %s for pipeline with id: %s is now RUNNING", uploadedPipelineRun.RunID, uploadedPipeline.PipelineID)
				timeToWait := time.Duration(300)
				utils.WaitForRunToBeInState(runClient, &uploadedPipelineRun.RunID, []run_model.V2beta1RuntimeState{run_model.V2beta1RuntimeStateSUCCEEDED, run_model.V2beta1RuntimeStateSKIPPED, run_model.V2beta1RuntimeStateFAILED, run_model.V2beta1RuntimeStateCANCELED}, &timeToWait)
				logger.Log("Deserializing expected compiled workflow file '%s' for the pipeline", pipelineFile)
				compiledWorkflow := workflow_utils.UnmarshallWorkflowYAML(filepath.Join(utils.GetProjectDataDir(), compiledWorkflowsDir, pipelineFile))
				validateComponentStatuses(uploadedPipelineRun.RunID, compiledWorkflow)
			})
		}
	})
})

// ####################################################################################################################################################################
// ################################################################### UTILITY METHODS ################################################################################
// ####################################################################################################################################################################

func validateComponentStatuses(runID string, compiledWorkflow *v1alpha1.Workflow) {
	logger.Log("Fetching updated pipeline run details for run with id=%s", runID)
	updatedRun := utils.GetPipelineRun(runClient, &runID)
	actualTaskDetails := updatedRun.RunDetails.TaskDetails
	lengthOfActualTasks := len(actualTaskDetails)
	for _, actualTask := range actualTaskDetails {
		if actualTask.DisplayName == "" {
			// Not sure what this task does, but it's an empty task that shows up in run details
			lengthOfActualTasks = lengthOfActualTasks - 1
		}
	}
	logger.Log("Updated pipeline run details")
	expectedTaskDetails := getTasksFromWorkflow(compiledWorkflow)
	lengthOfExpectedTasks := len(expectedTaskDetails)
	for _, expectedTask := range expectedTaskDetails {
		if expectedTask.Task.Template == "system-container-executor" {
			// For every executor, we create a pod that actually executes the commands
			lengthOfExpectedTasks = lengthOfExpectedTasks + 1
		}
	}
	if updatedRun.State != run_model.V2beta1RuntimeStateSUCCEEDED {
		logger.Log("Looks like the run %s FAILED, so capture pod logs for the failed task", runID)
		capturePodLogsForUnsuccessfulTasks(actualTaskDetails)
		Fail("Failing test because the pipeline run is not a SUCCESS")
	} else {
		logger.Log("Pipeline run succeeded, checking if the number of tasks are what is expected")
		gomega.Expect(lengthOfActualTasks).To(gomega.Equal(lengthOfExpectedTasks), "Number of expected DAG tasks does not match the length of actual task")
	}

}

func capturePodLogsForUnsuccessfulTasks(taskDetails []*run_model.V2beta1PipelineTaskDetail) {
	failedTasks := make(map[string]string)
	sort.Slice(taskDetails, func(i, j int) bool {
		return time.Time(taskDetails[i].EndTime).After(time.Time(taskDetails[j].EndTime)) // Sort Tasks by End Time in descending order
	})
	for _, task := range taskDetails {
		switch task.State {
		case run_model.V2beta1RuntimeStateSUCCEEDED:
			{
				logger.Log("SUCCEEDED - Task %s for run %s has finished successfully", task.DisplayName, task.RunID)
			}
		case run_model.V2beta1RuntimeStateRUNNING:
			{
				logger.Log("RUNNING - Task %s for Run %s is running", task.DisplayName, task.RunID)

			}
		case run_model.V2beta1RuntimeStateSKIPPED:
			{
				logger.Log("SKIPPED - Task %s for Run %s skipped", task.DisplayName, task.RunID)
			}
		case run_model.V2beta1RuntimeStateCANCELED:
			{
				logger.Log("CANCELED - Task %s for Run %s canceled", task.DisplayName, task.RunID)
			}
		case run_model.V2beta1RuntimeStateFAILED:
			{
				logger.Log("%s - Task %s for Run %s did not complete successfully", task.State, task.DisplayName, task.RunID)
				for _, childTask := range task.ChildTasks {
					podName := childTask.PodName
					if podName != "" {
						logger.Log("Capturing pod logs for task %s, with pod name %s", task.DisplayName, podName)
						podLog := utils.ReadPodLogs(k8Client, *namespace, podName, nil, &testStartTimeUTC, podLogLimit)
						logger.Log("Pod logs captured for pod %s", task.DisplayName, podName)
						logger.Log("Attaching pod logs to the report")
						AddReportEntry(fmt.Sprintf("Failing '%s' Component Log", task.DisplayName), podLog)
						logger.Log("Attached pod logs to the report")
					}
				}
				failedTasks[task.DisplayName] = string(task.State)
			}
		default:
			{
				logger.Log("UNKNOWN state - Task %s for Run %s has an UNKNOWN state", task.DisplayName, task.RunID)
			}
		}
	}
	if len(failedTasks) > 0 {
		logger.Log("Found failed tasks: %s", maps.Keys(failedTasks))
	}
}

type TaskDetails struct {
	TaskName  string
	Task      v1alpha1.DAGTask
	Container v1.Container
	DependsOn string
}

func getTasksFromWorkflow(workflow *v1alpha1.Workflow) []TaskDetails {
	var containers = make(map[string]*v1.Container)
	var tasks []TaskDetails
	for _, template := range workflow.Spec.Templates {
		if template.Container != nil {
			containers[template.Name] = template.Container
		}
	}
	for _, template := range workflow.Spec.Templates {
		if template.DAG != nil {
			for _, task := range template.DAG.Tasks {
				container, containerExists := containers[task.Template]
				taskToAppend := TaskDetails{
					TaskName:  task.Name,
					Task:      task,
					DependsOn: task.Depends,
				}
				if containerExists {
					taskToAppend.Container = *container
				}
				tasks = append(tasks, taskToAppend)
			}
		}
	}
	return tasks
}
