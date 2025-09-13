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

package end2end

import (
	"fmt"
	"github.com/go-openapi/strfmt"
	upload_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_client/pipeline_upload_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_model"
	runparams "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_client/run_service"
	"github.com/kubeflow/pipelines/backend/test/config"
	. "github.com/kubeflow/pipelines/backend/test/constants"
	"github.com/kubeflow/pipelines/backend/test/logger"
	apitests "github.com/kubeflow/pipelines/backend/test/v2/api"
	"maps"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	v1 "k8s.io/api/core/v1"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_model"
	workflowutils "github.com/kubeflow/pipelines/backend/test/compiler/utils"
	"github.com/kubeflow/pipelines/backend/test/test_utils"

	model "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Upload and Verify Pipeline Run >", Label(FullRegression), func() {
	var testContext *apitests.TestContext

	// ####################################################################################################################################################################
	// ################################################################### SET AND TEARDOWN ################################################################################
	// ####################################################################################################################################################################

	BeforeEach(func() {
		logger.Log("################### Setup before each Pipeline Upload test #####################")
		logger.Log("################### Global Setup before each test #####################")
		testContext = &apitests.TestContext{
			TestStartTimeUTC: time.Now(),
		}
		logger.Log("Test Context: %p", &testContext)
		randomName = strconv.FormatInt(time.Now().UnixNano(), 10)
		testContext.Pipeline.UploadParams = upload_params.NewUploadPipelineParams()
		testContext.Pipeline.PipelineGeneratedName = "e2e_test-" + randomName
		testContext.Pipeline.CreatedPipelines = make([]*pipeline_upload_model.V2beta1Pipeline, 0)
		testContext.PipelineRun.CreatedRunIds = make([]string, 0)
		testContext.Experiment.CreatedExperimentIds = make([]string, 0)
		testContext.Pipeline.ExpectedPipeline = new(model.V2beta1Pipeline)
		testContext.Pipeline.ExpectedPipeline.CreatedAt = strfmt.DateTime(testContext.TestStartTimeUTC)
	})

	AfterEach(func() {

		// Delete pipelines created during the test
		logger.Log("################### Global Cleanup after each test #####################")

		logger.Log("Deleting %d run(s)", len(testContext.PipelineRun.CreatedRunIds))
		for _, runID := range testContext.PipelineRun.CreatedRunIds {
			runID := runID
			test_utils.TerminatePipelineRun(runClient, runID)
			test_utils.DeletePipelineRun(runClient, runID)
		}
		logger.Log("Deleting %d experiment(s)", len(testContext.Experiment.CreatedExperimentIds))
		if len(testContext.Experiment.CreatedExperimentIds) > 0 {
			for _, experimentID := range testContext.Experiment.CreatedExperimentIds {
				experimentID := experimentID
				test_utils.DeleteExperiment(experimentClient, experimentID)
			}
		}
		logger.Log("Deleting %d pipeline(s)", len(testContext.Pipeline.CreatedPipelines))
		for _, pipeline := range testContext.Pipeline.CreatedPipelines {
			pipelineID := pipeline.PipelineID
			test_utils.DeletePipeline(pipelineClient, pipelineID)
		}
	})

	// ####################################################################################################################################################################
	// ################################################################### TESTS ################################################################################
	// ####################################################################################################################################################################

	Context("Upload a pipeline file, run it and verify that pipeline run succeeds >", Label("E2ENonCritical"), func() {
		var pipelineDir = "valid"
		pipelineFiles := test_utils.GetListOfFilesInADir(filepath.Join(test_utils.GetPipelineFilesDir(), pipelineDir))
		for _, pipelineFile := range pipelineFiles {
			It(fmt.Sprintf("Upload %s pipeline", pipelineFile), func() {
				test_utils.SkipTest(pipelineFile)
				pipelineFilePath := filepath.Join(test_utils.GetPipelineFilesDir(), pipelineDir, pipelineFile)
				logger.Log("Uploading pipeline file %s", pipelineFile)
				uploadedPipeline, uploadErr := uploadPipeline(testContext, pipelineDir, pipelineFile, &testContext.Pipeline.PipelineGeneratedName, nil)
				Expect(uploadErr).To(BeNil(), "Failed to upload pipeline %s", pipelineFile)
				logger.Log("Upload of pipeline file '%s' successful", pipelineFile)
				uploadedPipelineVersion := test_utils.GetLatestPipelineVersion(pipelineClient, &uploadedPipeline.PipelineID)
				pipelineRuntimeInputs := test_utils.GetPipelineRunTimeInputs(pipelineFilePath)
				createdRunId := createPipelineRunAndWaitForItToFinish(testContext, uploadedPipeline.PipelineID, uploadedPipeline.DisplayName, &uploadedPipelineVersion.PipelineVersionID, nil, pipelineRuntimeInputs)
				logger.Log("Deserializing expected compiled workflow file '%s' for the pipeline", pipelineFile)
				compiledWorkflow := workflowutils.UnmarshallWorkflowYAML(filepath.Join(test_utils.GetCompiledWorkflowsFilesDir(), pipelineFile))
				validateComponentStatuses(testContext, createdRunId, compiledWorkflow)

			})
		}
	})
	Context("Upload a pipeline file, run it and verify that pipeline run succeeds >", Label("Sample", "E2ECritical"), func() {
		var pipelineDir = "valid/critical"
		pipelineFiles := test_utils.GetListOfFilesInADir(filepath.Join(test_utils.GetPipelineFilesDir(), pipelineDir))
		for _, pipelineFile := range pipelineFiles {
			It(fmt.Sprintf("Upload %s pipeline", pipelineFile), func() {
				test_utils.SkipTest(pipelineFile)
				pipelineFilePath := filepath.Join(test_utils.GetPipelineFilesDir(), pipelineDir, pipelineFile)
				logger.Log("Uploading pipeline file %s", pipelineFile)
				uploadedPipeline, uploadErr := uploadPipeline(testContext, pipelineDir, pipelineFile, &testContext.Pipeline.PipelineGeneratedName, nil)
				Expect(uploadErr).To(BeNil(), "Failed to upload pipeline %s", pipelineFile)
				logger.Log("Upload of pipeline file '%s' successful", pipelineFile)
				uploadedPipelineVersion := test_utils.GetLatestPipelineVersion(pipelineClient, &uploadedPipeline.PipelineID)
				pipelineRuntimeInputs := test_utils.GetPipelineRunTimeInputs(pipelineFilePath)
				createdRunId := createPipelineRunAndWaitForItToFinish(testContext, uploadedPipeline.PipelineID, uploadedPipeline.DisplayName, &uploadedPipelineVersion.PipelineVersionID, nil, pipelineRuntimeInputs)
				logger.Log("Deserializing expected compiled workflow file '%s' for the pipeline", pipelineFile)
				compiledWorkflow := workflowutils.UnmarshallWorkflowYAML(filepath.Join(test_utils.GetCompiledWorkflowsFilesDir(), pipelineFile))
				validateComponentStatuses(testContext, createdRunId, compiledWorkflow)

			})
		}
	})
	Context("Create a pipeline run with HTTP proxy >", Label("E2EProxy"), func() {
		var pipelineDir = "valid"
		pipelineFile := "env-var.yaml"
		It(fmt.Sprintf("Create a pipeline run with http proxy, using specs: %s", pipelineFile), func() {
			uploadedPipeline, uploadErr := uploadPipeline(testContext, pipelineDir, pipelineFile, &testContext.Pipeline.PipelineGeneratedName, nil)
			Expect(uploadErr).To(BeNil(), "Failed to upload pipeline %s", pipelineFile)
			createdPipelineVersion := test_utils.GetLatestPipelineVersion(pipelineClient, &uploadedPipeline.PipelineID)
			createdExperiment := test_utils.CreateExperiment(experimentClient, "ProxyTest-"+randomName)
			pipelineRuntimeInputs := map[string]interface{}{
				"env_var": "http_proxy",
			}
			createdRunId := createPipelineRunAndWaitForItToFinish(testContext, uploadedPipeline.PipelineID, uploadedPipeline.DisplayName, &createdPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, pipelineRuntimeInputs)
			if *config.UseProxy {
				logger.Log("Deserializing expected compiled workflow file '%s' for the pipeline", pipelineFile)
				compiledWorkflow := workflowutils.UnmarshallWorkflowYAML(filepath.Join(test_utils.GetCompiledWorkflowsFilesDir(), pipelineFile))
				validateComponentStatuses(testContext, createdRunId, compiledWorkflow)
			} else {
				runState := test_utils.GetPipelineRun(runClient, &createdRunId).State
				expectedRunState := run_model.V2beta1RuntimeStateFAILED
				Expect(runState).To(Equal(&expectedRunState), fmt.Sprintf("Expected run with id=%s to fail with proxy=false", createdRunId))
			}
		})
	})
})

// ####################################################################################################################################################################
// ################################################################### UTILITY METHODS ################################################################################
// ####################################################################################################################################################################

func uploadPipeline(testContext *apitests.TestContext, pipelineDir string, pipelineFileName string, pipelineName *string, pipelineDisplayName *string) (*model.V2beta1Pipeline, error) {
	pipelineFile := filepath.Join(test_utils.GetPipelineFilesDir(), pipelineDir, pipelineFileName)
	testContext.Pipeline.UploadParams.SetName(pipelineName)
	if pipelineDisplayName != nil {
		testContext.Pipeline.ExpectedPipeline.DisplayName = *pipelineDisplayName
		testContext.Pipeline.UploadParams.SetDisplayName(pipelineDisplayName)
	} else {
		testContext.Pipeline.ExpectedPipeline.DisplayName = *pipelineName
	}
	logger.Log("Uploading pipeline with name=%s, from file %s", *pipelineName, pipelineFile)
	return pipelineUploadClient.UploadFile(pipelineFile, testContext.Pipeline.UploadParams)
}

func createPipelineRun(testContext *apitests.TestContext, pipelineID *string, pipelineVersionID *string, experimentID *string, inputParams map[string]interface{}) *run_model.V2beta1Run {
	runName := fmt.Sprintf("E2e Test Run-%v", testContext.TestStartTimeUTC)
	runDescription := fmt.Sprintf("Run for %s", runName)
	logger.Log("Create a pipeline run for pipeline with id=%s and versionId=%s", *pipelineID, *pipelineVersionID)
	createRunRequest := &runparams.RunServiceCreateRunParams{Run: createPipelineRunPayload(runName, runDescription, pipelineID, pipelineVersionID, experimentID, inputParams)}
	createdRun, createRunError := runClient.Create(createRunRequest)
	Expect(createRunError).NotTo(HaveOccurred(), "Failed to create run for pipeline with id="+*pipelineID)
	testContext.PipelineRun.CreatedRunIds = append(testContext.PipelineRun.CreatedRunIds, createdRun.RunID)
	logger.Log("Created Pipeline Run successfully with runId=%s", createdRun.RunID)
	return createdRun
}

func createPipelineRunPayload(runName string, runDescription string, pipelineID *string, pipelineVersionID *string, experimentID *string, inputParams map[string]interface{}) *run_model.V2beta1Run {
	logger.Log("Create a pipeline run body")
	return &run_model.V2beta1Run{
		DisplayName:    runName,
		Description:    runDescription,
		ExperimentID:   test_utils.ParsePointersToString(experimentID),
		ServiceAccount: test_utils.GetDefaultPipelineRunnerServiceAccount(*config.IsKubeflowMode),
		PipelineVersionReference: &run_model.V2beta1PipelineVersionReference{
			PipelineID:        test_utils.ParsePointersToString(pipelineID),
			PipelineVersionID: test_utils.ParsePointersToString(pipelineVersionID),
		},
		RuntimeConfig: &run_model.V2beta1RuntimeConfig{
			Parameters: inputParams,
		},
	}
}

func createPipelineRunAndWaitForItToFinish(testContext *apitests.TestContext, pipelineID string, pipelineDisplayName string, pipelineVersionID *string, experimentID *string, runTimeParams map[string]interface{}) string {
	logger.Log("Create run for pipeline with id: '%s' and name: '%s'", pipelineID, pipelineDisplayName)
	uploadedPipelineRun := createPipelineRun(testContext, &pipelineID, pipelineVersionID, experimentID, runTimeParams)
	logger.Log("Created Pipeline Run with id: %s for pipeline with id: %s", uploadedPipelineRun.RunID, pipelineID)
	test_utils.WaitForRunToBeInState(runClient, &uploadedPipelineRun.RunID, []run_model.V2beta1RuntimeState{run_model.V2beta1RuntimeStateRUNNING}, nil)
	logger.Log("Created Pipeline Run with id: %s for pipeline with id: %s is now RUNNING", uploadedPipelineRun.RunID, pipelineID)
	timeToWait := time.Duration(300)
	test_utils.WaitForRunToBeInState(runClient, &uploadedPipelineRun.RunID, []run_model.V2beta1RuntimeState{run_model.V2beta1RuntimeStateSUCCEEDED, run_model.V2beta1RuntimeStateSKIPPED, run_model.V2beta1RuntimeStateFAILED, run_model.V2beta1RuntimeStateCANCELED}, &timeToWait)
	return uploadedPipelineRun.RunID
}

func validateComponentStatuses(testContext *apitests.TestContext, runID string, compiledWorkflow *v1alpha1.Workflow) {
	logger.Log("Fetching updated pipeline run details for run with id=%s", runID)
	updatedRun := test_utils.GetPipelineRun(runClient, &runID)
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
	if *updatedRun.State != run_model.V2beta1RuntimeStateSUCCEEDED {
		logger.Log("Looks like the run %s FAILED, so capture pod logs for the failed task", runID)
		capturePodLogsForUnsuccessfulTasks(testContext, actualTaskDetails)
		Fail("Failing test because the pipeline run is not a SUCCESS")
	} else {
		logger.Log("Pipeline run succeeded, checking if the number of tasks are what is expected")
		Expect(lengthOfActualTasks).To(Equal(lengthOfExpectedTasks), "Number of expected DAG tasks does not match the length of actual task")
	}

}

func capturePodLogsForUnsuccessfulTasks(testContext *apitests.TestContext, taskDetails []*run_model.V2beta1PipelineTaskDetail) {
	failedTasks := make(map[string]string)
	sort.Slice(taskDetails, func(i, j int) bool {
		return time.Time(taskDetails[i].EndTime).After(time.Time(taskDetails[j].EndTime)) // Sort Tasks by End Time in descending order
	})
	for _, task := range taskDetails {
		if task.State != nil {
			switch *task.State {
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
					logger.Log("%s - Task %s for Run %s did not complete successfully", *task.State, task.DisplayName, task.RunID)
					for _, childTask := range task.ChildTasks {
						podName := childTask.PodName
						if podName != "" {
							logger.Log("Capturing pod logs for task %s, with pod name %s", task.DisplayName, podName)
							podLog := test_utils.ReadPodLogs(k8Client, *config.Namespace, podName, nil, &testContext.TestStartTimeUTC, config.PodLogLimit)
							logger.Log("Pod logs captured for task %s in pod %s", task.DisplayName, podName)
							logger.Log("Attaching pod logs to the report")
							AddReportEntry(fmt.Sprintf("Failing '%s' Component Log", task.DisplayName), podLog)
							logger.Log("Attached pod logs to the report")
						}
					}
					failedTasks[task.DisplayName] = string(*task.State)
				}
			default:
				{
					logger.Log("UNKNOWN state - Task %s for Run %s has an UNKNOWN state", task.DisplayName, task.RunID)
				}
			}
		}
	}
	if len(failedTasks) > 0 {
		logger.Log("Found failed tasks: %v", maps.Keys(failedTasks))
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
