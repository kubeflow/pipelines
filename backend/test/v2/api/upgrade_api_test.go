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
	"path/filepath"

	experimentparams "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/experiment_client/experiment_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/experiment_model"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_model"
	uploadparams "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_client/pipeline_upload_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_model"
	recurringrunparams "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/recurring_run_client/recurring_run_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/recurring_run_model"
	runparams "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_client/run_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/test/config"
	"github.com/kubeflow/pipelines/backend/test/constants"
	"github.com/kubeflow/pipelines/backend/test/logger"
	"github.com/kubeflow/pipelines/backend/test/testutil"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	longRunningPipelineFileName = "take_nap_compiled.yaml"
)

// ################## UPGRADE TEST PREPARATION ##################

var _ = Describe("Upgrade Test Preparation >", Label(constants.UpgradePreparation, constants.FullRegression), func() {
	Context("Prepare test data >", func() {

		It("Upload pipelines and create experiments", func() {
			prepareExperiments()
			preparePipelines()
		})
		It("Create pipeline run", func() {
			preparePipelineRun(helloWorldPipelineFileName, "pipeline-1", "Experiment-1", "Run 1")
		})
		It("Create pipeline run and wait it to go to RUNNING state", func() {
			run := preparePipelineRun(longRunningPipelineFileName, "pipeline-3", "Experiment-3", "Run 3")
			testutil.WaitForRunToBeInState(runClient, &run.RunID, []run_model.V2beta1RuntimeState{run_model.V2beta1RuntimeStateRUNNING}, nil)
		})
		It("Create scheduled pipeline run", func() {
			prepareScheduledPipelineRun(helloWorldPipelineFileName, "pipeline-4", "Experiment-4", "Scheduled Run 1")
		})

	})
})

// ################## UPGRADE TEST VERIFICATION ##################

var _ = Describe("Upgrade Test Verification >", Label(constants.UpgradeVerification, constants.FullRegression), func() {
	Context("Verify resources after upgrade >", func() {
		It("Verify that Pipelines & experiments should persist correctly", func() {
			verifyExperiments()
			verifyPipelines()
		})
		It("Verify pipeline run", func() {
			verifyPipelineRun(helloWorldPipelineFileName, "pipeline-1", "Experiment-1", "Run 1")
		})
		It("Verify you can create a pipeline run after upgrade", func() {
			preparePipelineRun(helloWorldPipelineFileName, "pipeline-2", "Experiment-2", "Run 2")
			verifyPipelineRun(longRunningPipelineFileName, "pipeline-2", "Experiment-2", "Run 2")
		})
		It("Verify pipeline run that was in RUNNING state, still exists after the upgrade", func() {
			verifyPipelineRun(longRunningPipelineFileName, "pipeline-3", "Experiment-3", "Run 3")
		})
		It("Verify scheduled pipeline run", func() {
			verifyScheduledPipelineRun(helloWorldPipelineFileName, "pipeline-4", "Experiment-4", "Scheduled Run 1")
		})
		It("Verify you can create scheduled pipeline run after upgrade", func() {
			prepareScheduledPipelineRun(helloWorldPipelineFileName, "pipeline-5", "Experiment-5", "Scheduled Run 2")
			verifyScheduledPipelineRun(helloWorldPipelineFileName, "pipeline-5", "Experiment-5", "Scheduled Run 2")
		})
	})
})

// ################## HELPER FUNCTIONS ##################

// ################## EXPERIMENTS ##################

func getResourceNamespace() string {
	if *config.KubeflowMode || *config.MultiUserMode {
		return *config.UserNamespace
	}
	return *config.Namespace
}

func getExpectedExperiments() []*experiment_model.V2beta1Experiment {
	var experiments []*experiment_model.V2beta1Experiment
	experiment1 := &experiment_model.V2beta1Experiment{
		DisplayName: "training",
		Description: "my first experiment",
		Namespace:   getResourceNamespace(),
	}
	experiment2 := &experiment_model.V2beta1Experiment{
		DisplayName: "prediction",
		Description: "my second experiment",
		Namespace:   getResourceNamespace(),
	}
	experiment3 := &experiment_model.V2beta1Experiment{
		DisplayName: "moonshot",
		Description: "my third experiment",
		Namespace:   getResourceNamespace(),
	}
	experiments = append(experiments, experiment1, experiment2, experiment3)
	return experiments
}

func prepareExperiments() {
	experiments := getExpectedExperiments()
	for _, experiment := range experiments {
		testutil.CreateExperimentWithParams(experimentClient, experiment)
	}
}

func verifyExperiments() {
	namespace := getResourceNamespace()
	allExperiments := testutil.ListExperiments(
		experimentClient,
		&experimentparams.ExperimentServiceListExperimentsParams{
			Namespace: &namespace,
			SortBy:    util.StringPointer("created_at"),
			PageSize:  util.Int32Pointer(1000),
		},
	)
	expectedExperiments := getExpectedExperiments()
	Expect(len(allExperiments)).To(BeNumerically(">", len(expectedExperiments)))
	existingExperimentsMap := make(map[string]experiment_model.V2beta1Experiment)
	for _, exp := range allExperiments {
		existingExperimentsMap[exp.DisplayName] = *exp
	}
	for _, exp := range expectedExperiments {
		existingExperiment := existingExperimentsMap[exp.DisplayName]
		Expect(existingExperiment).ToNot(BeNil())
		Expect(existingExperiment.Description).To(Equal(exp.Description), fmt.Sprintf("Experiment %s description is not same", exp.DisplayName))
		// Experiment API response does not currently return populated Namespace field
		// Expect(existingExperiment.Namespace).To(Equal(exp.Namespace), fmt.Sprintf("Experiment %s namespace is not same"))
	}
}

// ################## PIPELINES ##################

func getExpectedPipelines() []*uploadparams.UploadPipelineParams {
	var pipelines []*uploadparams.UploadPipelineParams
	pipelineParams1 := uploadparams.NewUploadPipelineParams()
	pipelineName1 := "training"
	pipelineDescription1 := "My first pipeline"
	pipelineParams1.SetName(&pipelineName1)
	pipelineParams1.SetDisplayName(&pipelineName1)
	pipelineParams1.SetDescription(&pipelineDescription1)

	pipelineParams2 := uploadparams.NewUploadPipelineParams()
	pipelineName2 := "prediction"
	pipelineDescription2 := "My Second pipeline"
	pipelineParams2.SetName(&pipelineName2)
	pipelineParams2.SetDisplayName(&pipelineName2)
	pipelineParams2.SetDescription(&pipelineDescription2)

	pipelineParams3 := uploadparams.NewUploadPipelineParams()
	pipelineName3 := "moonshot"
	pipelineDescription3 := "My third pipeline"
	pipelineParams3.SetName(&pipelineName3)
	pipelineParams3.SetDisplayName(&pipelineName3)
	pipelineParams3.SetDescription(&pipelineDescription3)

	pipelines = append(pipelines, pipelineParams1, pipelineParams2, pipelineParams3)
	return pipelines
}

// ################## PIPELINE RUNS ##################

func getPipelineAndExperimentForRun(pipelineToUpload string, pipelineName string, experimentName string) (string, string, string) {
	// Check if pipeline already exists or not, if not, then upload a new one
	namespace := getResourceNamespace()
	pipelineFilePath := filepath.Join(testutil.GetValidPipelineFilesDir(), pipelineToUpload)
	var uploadedPipeline *pipeline_upload_model.V2beta1Pipeline
	var err error
	pipelineDisplayName := "Pipeline to Run"
	pipelineDescription := "My Pipeline to Upload"
	pipelineUploadParams := uploadparams.NewUploadPipelineParams()
	pipelineUploadParams.SetName(&pipelineName)
	pipelineUploadParams.SetDescription(&pipelineDescription)
	pipelineUploadParams.SetDisplayName(&pipelineDisplayName)
	existingPipelines := testutil.ListPipelines(pipelineClient, &namespace)
	for _, pipeline := range existingPipelines {
		if pipeline.Name == pipelineName {
			logger.Log("Pipeline with name=%s, already exists", *pipelineUploadParams.Name)
			uploadedPipeline = &pipeline_upload_model.V2beta1Pipeline{
				Name:        pipeline.Name,
				DisplayName: pipeline.DisplayName,
				PipelineID:  pipeline.PipelineID,
				Description: pipeline.Description,
				CreatedAt:   pipeline.CreatedAt,
			}
		}
	}

	if uploadedPipeline == nil {
		logger.Log("Uploading pipeline %s from file %s", pipelineName, pipelineFilePath)
		uploadedPipeline, err = pipelineUploadClient.UploadFile(pipelineFilePath, pipelineUploadParams)
		Expect(err).To(BeNil(), "Failed to upload pipeline: %s", pipelineFilePath)
		logger.Log("Uploaded pipeline from file %s", pipelineFilePath)
	}

	// Get pipeline versions associated with the above pipeline
	logger.Log("Fetch pipeline versions for pipeline with id= %s", uploadedPipeline.PipelineID)
	uploadedPipelineVersions, _, _, pipelineVersionError := testutil.ListPipelineVersions(pipelineClient, uploadedPipeline.PipelineID)
	Expect(pipelineVersionError).To(BeNil(), fmt.Sprintf("Failed to list uploaded pipeline versions for pipeline with id=%s", uploadedPipeline.PipelineID))
	logger.Log("Fetched %d pipeline versions for pipeline with id= %s", len(uploadedPipelineVersions), uploadedPipeline.PipelineID)

	// Get existing experiments and see if expected exists or not, if not, then create a new one
	var createdExperiment *experiment_model.V2beta1Experiment
	experimentParams := &experiment_model.V2beta1Experiment{
		DisplayName: experimentName,
		Description: "my first experiment",
	}
	logger.Log("Fetching all experiments")
	allExperiments := testutil.ListExperiments(
		experimentClient,
		&experimentparams.ExperimentServiceListExperimentsParams{
			SortBy:   util.StringPointer("created_at"),
			PageSize: util.Int32Pointer(1000),
		},
	)
	for _, experiment := range allExperiments {
		if experiment.DisplayName == experimentParams.DisplayName {
			logger.Log("Found experiment with display name %s", experiment.DisplayName)
			createdExperiment = experiment
		}
	}
	if createdExperiment == nil {
		logger.Log("No existing experiment found with name '%s', so Creating new experiment", experimentParams.DisplayName)
		createdExperiment = testutil.CreateExperimentWithParams(experimentClient, experimentParams)
	}
	return uploadedPipeline.PipelineID, uploadedPipelineVersions[0].PipelineVersionID, createdExperiment.ExperimentID
}

func getExpectedPipelineRun(pipelineToUpload string, pipelineName string, experimentName string, pipelineRunName string) *run_model.V2beta1Run {
	pipelineID, pipelineVersionID, experimentID := getPipelineAndExperimentForRun(pipelineToUpload, pipelineName, experimentName)

	return &run_model.V2beta1Run{
		DisplayName:    pipelineRunName,
		Description:    "This is my first pipeline run",
		ExperimentID:   experimentID,
		ServiceAccount: testutil.GetDefaultPipelineRunnerServiceAccount(),
		PipelineVersionReference: &run_model.V2beta1PipelineVersionReference{
			PipelineID:        pipelineID,
			PipelineVersionID: pipelineVersionID,
		},
	}
}

func getExpectedRecurringPipelineRun(pipelineToUpload string, pipelineName string, experimentName string, pipelineRunName string) *recurring_run_model.V2beta1RecurringRun {
	pipelineID, pipelineVersionID, experimentID := getPipelineAndExperimentForRun(pipelineToUpload, pipelineName, experimentName)

	return &recurring_run_model.V2beta1RecurringRun{
		DisplayName:    pipelineRunName,
		Description:    "This is my first recurring pipeline run",
		ExperimentID:   experimentID,
		ServiceAccount: testutil.GetDefaultPipelineRunnerServiceAccount(),
		PipelineVersionReference: &recurring_run_model.V2beta1PipelineVersionReference{
			PipelineID:        pipelineID,
			PipelineVersionID: pipelineVersionID,
		},
		Trigger: &recurring_run_model.V2beta1Trigger{
			CronSchedule: &recurring_run_model.V2beta1CronSchedule{
				Cron: "*/10 * * * *",
			},
		},
		Mode:           recurring_run_model.RecurringRunModeENABLE.Pointer(),
		MaxConcurrency: 1,
	}
}

func preparePipelines() {
	pipelineFilePath := filepath.Join(testutil.GetValidPipelineFilesDir(), helloWorldPipelineFileName)
	for _, pipelineParams := range getExpectedPipelines() {
		logger.Log("Uploading pipeline with name=%s, from file %s", *pipelineParams.Name, pipelineFilePath)
		_, err := pipelineUploadClient.UploadFile(pipelineFilePath, pipelineParams)
		Expect(err).To(BeNil(), "Failed to upload pipeline with name: %s", pipelineParams.Name)
	}
}

func verifyPipelines() {
	namespace := getResourceNamespace()
	existingPipelines := testutil.ListPipelines(pipelineClient, &namespace)
	expectedPipelines := getExpectedPipelines()
	Expect(len(existingPipelines)).To(BeNumerically(">=", len(expectedPipelines)))
	existingPipelinesMap := make(map[string]*pipeline_model.V2beta1Pipeline)
	for _, pipeline := range existingPipelines {
		existingPipelinesMap[pipeline.Name] = pipeline
	}
	for _, pipelineParams := range expectedPipelines {
		existingPipeline := existingPipelinesMap[*pipelineParams.Name]
		Expect(existingPipeline).ToNot(BeNil())
		Expect(existingPipeline.Description).To(Equal(*pipelineParams.Description), fmt.Sprintf("Pipeline %s description is not same", *pipelineParams.Name))
		Expect(existingPipeline.DisplayName).To(Equal(*pipelineParams.DisplayName), fmt.Sprintf("Pipeline %s display name is not same", *pipelineParams.Name))
	}
}

func preparePipelineRun(pipelineToUpload string, pipelineName string, experimentName string, runName string) *run_model.V2beta1Run {
	expectedPipelineRun := getExpectedPipelineRun(pipelineToUpload, pipelineName, experimentName, runName)
	pipelineRun, pipelineRunError := runClient.Create(&runparams.RunServiceCreateRunParams{Run: expectedPipelineRun})
	Expect(pipelineRunError).To(BeNil(), "Failed to create pipeline run")
	return pipelineRun
}

func verifyPipelineRun(uploadedPipeline string, pipelineName string, experimentName string, runName string) {
	runListParams := &runparams.RunServiceListRunsParams{
		SortBy:   util.StringPointer("created_at"),
		PageSize: util.Int32Pointer(1000),
	}

	expectedRun := getExpectedPipelineRun(uploadedPipeline, pipelineName, experimentName, runName)
	allRuns, _, _, err := runClient.List(runListParams)
	Expect(err).To(BeNil(), "Failed to list runs")
	runPassed := false
	for _, run := range allRuns {
		if run.DisplayName == expectedRun.DisplayName && run.PipelineVersionID == expectedRun.PipelineVersionID {
			Expect(run.ExperimentID).To(Equal(expectedRun.ExperimentID), fmt.Sprintf("Experiment id for runid=%s is not same", expectedRun.DisplayName))
			Expect(run.Description).To(Equal(expectedRun.Description), "Run description is not same")
			runPassed = true
		}
	}
	Expect(runPassed).To(BeTrue(), "Failed to find the pipeline run")
}

func prepareScheduledPipelineRun(pipelineToUpload string, pipelineName string, experimentName string, runName string) *recurring_run_model.V2beta1RecurringRun {
	expectedPipelineRun := getExpectedRecurringPipelineRun(pipelineToUpload, pipelineName, experimentName, runName)
	pipelineRun, pipelineRunError := recurringRunClient.Create(&recurringrunparams.RecurringRunServiceCreateRecurringRunParams{RecurringRun: expectedPipelineRun})
	Expect(pipelineRunError).To(BeNil(), "Failed to create pipeline run")
	return pipelineRun
}

func verifyScheduledPipelineRun(uploadedPipeline string, pipelineName string, experimentName string, runName string) {
	runListParams := &recurringrunparams.RecurringRunServiceListRecurringRunsParams{
		SortBy:   util.StringPointer("created_at"),
		PageSize: util.Int32Pointer(1000),
	}

	expectedRun := getExpectedRecurringPipelineRun(uploadedPipeline, pipelineName, experimentName, runName)
	allRuns, _, _, err := recurringRunClient.List(runListParams)
	Expect(err).To(BeNil(), "Failed to list recurring runs")
	runPassed := false
	for _, run := range allRuns {
		if run.DisplayName == expectedRun.DisplayName && run.PipelineVersionID == expectedRun.PipelineVersionID {
			Expect(run.ExperimentID).To(Equal(expectedRun.ExperimentID), fmt.Sprintf("Experiment id for runid=%s is not same", expectedRun.DisplayName))
			Expect(run.Description).To(Equal(expectedRun.Description), "Run description is not same")
			Expect(run.Mode.Pointer()).To(Equal(expectedRun.Mode), "Run mode is not same")
			Expect(run.Trigger.CronSchedule.Cron).To(Equal(expectedRun.Trigger.CronSchedule.Cron), "Cron schedule is not same")
			runPassed = true
		}
	}
	Expect(runPassed).To(BeTrue(), "Failed to find the pipeline run")
}
