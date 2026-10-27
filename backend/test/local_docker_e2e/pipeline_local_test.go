// Copyright 2026 The Kubeflow Authors
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

package localdocker

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	runparams "github.com/kubeflow/pipelines/backend/api/v2/go_http_client/run_client/run_service"
	"github.com/kubeflow/pipelines/backend/api/v2/go_http_client/run_model"
	workflowutils "github.com/kubeflow/pipelines/backend/test/compiler/utils"
	"github.com/kubeflow/pipelines/backend/test/config"
	. "github.com/kubeflow/pipelines/backend/test/constants"
	e2eutils "github.com/kubeflow/pipelines/backend/test/end2end/utils"
	"github.com/kubeflow/pipelines/backend/test/logger"
	"github.com/kubeflow/pipelines/backend/test/testutil"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const localPipelineWaitTimeSeconds = 720
const localPipelineFilterEnvVar = "KFP_LOCAL_PIPELINE_FILTER"

var _ = Describe("Local Docker Pipeline Runs >", Label(FullRegression), func() {
	Context("Upload a pipeline file, run it and verify that pipeline run succeeds >", FlakeAttempts(2), Label(E2eEssential), func() {
		runPipelinesInDirectory("essential", runExpectingSuccess)
	})

	Context("Upload a nested or a parallel pipeline file, run it and verify that pipeline run succeeds >", FlakeAttempts(2), Label(E2eParallelNested), func() {
		runPipelinesInDirectory("parallel_and_nested", runExpectingSuccess)
	})

	Context("Upload a pipeline file, run it and verify that pipeline run succeeds >", FlakeAttempts(2), Label(E2eCritical), func() {
		runPipelinesInDirectory("critical", runExpectingSuccess)
	})

	Context("Upload a pipeline file, run it and verify that pipeline run succeeds >", FlakeAttempts(1), Label(Integration), func() {
		runPipelinesInDirectory("integration", runExpectingSuccess)
	})

	Context("Upload a pipeline file, run it and verify that pipeline run fails >", Label(E2eFailed), func() {
		runPipelinesInDirectory("failing", runExpectingFailure)
	})

	It("Explicit run retry creates a new attempt without changing task identity", Label(E2eFailed), func() {
		pipeline := filepath.Join(testutil.GetPipelineFilesDir(), "valid", "failing", "fail_v2.yaml")
		runID := runPipeline(pipeline, "fail_v2.yaml")
		before := testutil.GetPipelineRun(runClient, &runID)
		var previous *run_model.V2PipelineTask
		for _, task := range before.Tasks {
			if task.Name == "fail" {
				previous = task
			}
		}
		Expect(previous).NotTo(BeNil(), "failed component task must be exposed by the API")
		Expect(previous.StatusMetadata).NotTo(BeNil())
		attempt, ok := previous.StatusMetadata.CustomProperties["attemptID"].(string)
		Expect(ok).To(BeTrue())
		Expect(attempt).NotTo(BeEmpty())
		Expect(runClient.Retry(&runparams.RunServiceRetryRunParams{RunID: runID})).To(Succeed())
		Eventually(func() error {
			updated := testutil.GetPipelineRun(runClient, &runID)
			if updated.State == nil || *updated.State != run_model.V2RuntimeStateFAILED {
				return fmt.Errorf("retried run has not reached its expected failed outcome: %v", updated.State)
			}
			for _, task := range updated.Tasks {
				if task.Name != "fail" {
					continue
				}
				if task.TaskID != previous.TaskID {
					return fmt.Errorf("retry replaced stable task identity %s with %s", previous.TaskID, task.TaskID)
				}
				if task.StatusMetadata != nil {
					current, _ := task.StatusMetadata.CustomProperties["attemptID"].(string)
					if current != "" && current != attempt {
						return nil
					}
				}
			}
			return fmt.Errorf("retry has not persisted a new attempt for task %s", previous.TaskID)
		}).WithTimeout(time.Duration(localPipelineWaitTimeSeconds) * time.Second).WithPolling(time.Second).Should(Succeed())
	})

	Context("Create a pipeline run with HTTP proxy >", Label(E2eProxy), func() {
		pipelineDir := "valid"
		pipelineFile := "env-var.yaml"
		It(fmt.Sprintf("Create a pipeline run with http proxy, using specs: %s", pipelineFile), func() {
			pipelineFilePath := filepath.Join(testutil.GetPipelineFilesDir(), pipelineDir, pipelineFile)
			runID := runPipelineWithInputs(pipelineFilePath, pipelineFile, map[string]interface{}{
				"env_var": "http_proxy",
			})
			runState := testutil.GetPipelineRun(runClient, &runID).State
			Expect(runState).NotTo(BeNil(), "Expected run state to be reported")
			if *config.RunProxyTests {
				Expect(*runState).To(Equal(run_model.V2RuntimeStateSUCCEEDED), fmt.Sprintf("Expected run with id=%s to succeed with proxy enabled", runID))
			} else {
				Expect(*runState).To(Equal(run_model.V2RuntimeStateFAILED), fmt.Sprintf("Expected run with id=%s to fail without proxy support", runID))
			}
		})
	})
})

func runPipelinesInDirectory(relativeDir string, assertion func(string, string)) {
	pipelineDir := filepath.Join("valid", relativeDir)
	pipelineFiles := testutil.GetListOfFilesInADir(filepath.Join(testutil.GetPipelineFilesDir(), pipelineDir))
	pipelineFilter := os.Getenv(localPipelineFilterEnvVar)
	for _, pipelineFile := range pipelineFiles {
		if pipelineFilter != "" && !strings.Contains(pipelineFile, pipelineFilter) {
			continue
		}
		pipelineFile := pipelineFile
		It(fmt.Sprintf("Upload %s pipeline", pipelineFile), func() {
			assertion(pipelineDir, pipelineFile)
		})
	}
}

func runExpectingSuccess(pipelineDir string, pipelineFile string) {
	pipelineFilePath := filepath.Join(testutil.GetPipelineFilesDir(), pipelineDir, pipelineFile)
	runID := runPipeline(pipelineFilePath, pipelineFile)
	validatePipelineRunSucceeded(runID, pipelineFile)
}

func runExpectingFailure(pipelineDir string, pipelineFile string) {
	pipelineFilePath := filepath.Join(testutil.GetPipelineFilesDir(), pipelineDir, pipelineFile)
	runID := runPipeline(pipelineFilePath, pipelineFile)
	updatedRun := testutil.GetPipelineRun(runClient, &runID)
	Expect(updatedRun.State).NotTo(BeNil(), "Updated pipeline run state is Nil")
	Expect(*updatedRun.State).To(Equal(run_model.V2RuntimeStateFAILED), "Pipeline run was expected to fail, but is "+*updatedRun.State)
}

func runPipeline(pipelineFilePath string, pipelineFile string) string {
	return runPipelineWithInputs(pipelineFilePath, pipelineFile, testutil.GetPipelineRunTimeInputs(pipelineFilePath))
}

func runPipelineWithInputs(pipelineFilePath string, pipelineFile string, pipelineRuntimeInputs map[string]interface{}) string {
	testutil.CheckIfSkipping(pipelineFile)
	logger.Log("Uploading pipeline file %s", pipelineFilePath)
	uploadedPipeline, uploadErr := testutil.UploadPipeline(pipelineUploadClient, pipelineFilePath, stringPointer("local-docker-"+randomName), nil)
	Expect(uploadErr).To(BeNil(), "Failed to upload pipeline %s", pipelineFile)
	testContext.Pipeline.CreatedPipelines = append(testContext.Pipeline.CreatedPipelines, uploadedPipeline)

	uploadedPipelineVersion := testutil.GetLatestPipelineVersion(pipelineClient, &uploadedPipeline.PipelineID)
	runPayload := e2eutils.CreatePipelineRunPayload(
		"Local Docker E2E Run-"+randomName,
		"Run for local Docker coordinator validation",
		&uploadedPipeline.PipelineID,
		&uploadedPipelineVersion.PipelineVersionID,
		experimentID,
		pipelineRuntimeInputs,
	)
	Expect(runPayload.RuntimeConfig).NotTo(BeNil())
	runPayload.RuntimeConfig.PipelineRoot = localPipelineRoot()

	createRunRequest := &runparams.RunServiceCreateRunParams{
		ExperimentID: experimentID,
		Run:          runPayload,
	}
	createdRun, createRunError := runClient.Create(createRunRequest)
	Expect(createRunError).NotTo(HaveOccurred(), "Failed to create run for pipeline with id="+uploadedPipeline.PipelineID)
	testContext.PipelineRun.CreatedRunIds = append(testContext.PipelineRun.CreatedRunIds, createdRun.RunID)
	timeout := time.Duration(localPipelineWaitTimeSeconds) * time.Second
	testutil.WaitForRunToBeInState(
		runClient,
		&createdRun.RunID,
		[]run_model.V2RuntimeState{
			run_model.V2RuntimeStateSUCCEEDED,
			run_model.V2RuntimeStateFAILED,
			run_model.V2RuntimeStateCANCELED,
		},
		&timeout,
	)
	return createdRun.RunID
}

func validatePipelineRunSucceeded(runID string, pipelineFile string) {
	updatedRun := testutil.GetPipelineRun(runClient, &runID)
	Expect(updatedRun.State).NotTo(BeNil(), "Updated pipeline run state is Nil")
	Expect(*updatedRun.State).To(Equal(run_model.V2RuntimeStateSUCCEEDED), "Pipeline run was expected to succeed. Details: %s", runFailureDetails(updatedRun))
	Expect(updatedRun.Tasks).NotTo(BeEmpty(), "Expected task details for run %s", runID)

	compiledWorkflowFile := pipelineFile
	if strings.Contains(compiledWorkflowFile, "/") {
		compiledWorkflowFile = strings.Split(compiledWorkflowFile, "/")[1]
	}
	compiledWorkflow := workflowutils.UnmarshallWorkflowYAML(filepath.Join(testutil.GetCompiledWorkflowsFilesDir(), compiledWorkflowFile))
	expectedTaskDetails := e2eutils.GetTasksFromWorkflow(compiledWorkflow)
	Expect(len(updatedRun.Tasks)).To(BeNumerically(">=", len(expectedTaskDetails)), "Number of created DAG tasks should be >= number of expected tasks")
	for _, task := range updatedRun.Tasks {
		Expect(task.State).NotTo(BeNil(), "Expected task state to be reported for task %s", task.Name)
		Expect(*task.State).To(BeElementOf(
			run_model.PipelineTaskTaskStateSUCCEEDED,
			run_model.PipelineTaskTaskStateSKIPPED,
			run_model.PipelineTaskTaskStateCACHED,
		), "Unexpected task state for task %s", task.Name)
	}
	if filepath.Base(pipelineFile) == "parameters_simple.yaml" {
		Expect(testutil.ValidateCollectedMathOutputs(updatedRun)).To(Succeed(), "Collected numeric outputs must retain every iteration")
	}
}

func runFailureDetails(run *run_model.V2Run) string {
	if run == nil {
		return "run is nil"
	}
	var details []string
	if run.Error != nil && run.Error.Message != "" {
		details = append(details, "runError="+run.Error.Message)
	}
	for _, task := range run.Tasks {
		if task == nil || task.State == nil {
			continue
		}
		details = append(details, fmt.Sprintf("%s=%s", task.Name, *task.State))
	}
	return strings.Join(details, ", ")
}

func stringPointer(value string) *string {
	return &value
}
