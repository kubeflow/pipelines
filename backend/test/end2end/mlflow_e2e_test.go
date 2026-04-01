// Copyright 2026 The Kubeflow Authors
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
	"path/filepath"
	"strconv"
	"time"

	upload_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_client/pipeline_upload_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_model"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_model"
	. "github.com/kubeflow/pipelines/backend/test/constants"
	e2e_utils "github.com/kubeflow/pipelines/backend/test/end2end/utils"
	"github.com/kubeflow/pipelines/backend/test/logger"
	"github.com/kubeflow/pipelines/backend/test/testutil"
	apitests "github.com/kubeflow/pipelines/backend/test/v2/api"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
	. "github.com/onsi/gomega"
)

var _ = Describe("MLflow Integration >", Label(MLflow, FullRegression), func() {
	var testContext *apitests.TestContext
	var mlflowEndpoint string

	const singleTaskPipelineFile = "add_numbers.yaml"
	const singleTaskPipelineDir = "valid/critical"

	const multiTaskPipelineFile = "producer_consumer_param_pipeline.yaml"
	const multiTaskPipelineDir = "valid/critical"

	// ################## SETUP AND TEARDOWN ##################

	BeforeEach(func() {
		e2e_utils.SkipIfMLflowDisabled()
		mlflowEndpoint = e2e_utils.GetMLflowEndpoint()

		logger.Log("################### Setup before each MLflow test #####################")
		testContext = &apitests.TestContext{
			TestStartTimeUTC: time.Now(),
		}
		randomName = strconv.FormatInt(time.Now().UnixNano(), 10)
		testContext.Pipeline.UploadParams = upload_params.NewUploadPipelineParams()
		testContext.Pipeline.PipelineGeneratedName = "mlflow-e2e-" + randomName
		testContext.Pipeline.CreatedPipelines = make([]*pipeline_upload_model.V2beta1Pipeline, 0)
		testContext.PipelineRun.CreatedRunIds = make([]string, 0)
	})

	ReportAfterEach(func(specReport types.SpecReport) {
		if testContext == nil {
			return
		}
		if specReport.Failed() && len(testContext.PipelineRun.CreatedRunIds) > 0 {
			report, _ := testutil.BuildArchivedWorkflowLogsReport(k8Client, testContext.PipelineRun.CreatedRunIds)
			AddReportEntry(testutil.ArchivedWorkflowLogsReportTitle, report)
		}

		logger.Log("Deleting %d run(s)", len(testContext.PipelineRun.CreatedRunIds))
		for _, runID := range testContext.PipelineRun.CreatedRunIds {
			runID := runID
			testutil.TerminatePipelineRun(runClient, runID)
			testutil.ArchivePipelineRun(runClient, runID)
			testutil.DeletePipelineRun(runClient, runID)
		}
		logger.Log("Deleting %d experiment(s)", len(testContext.Experiment.CreatedExperimentIds))
		for _, expID := range testContext.Experiment.CreatedExperimentIds {
			expID := expID
			testutil.DeleteExperiment(experimentClient, expID)
		}
		logger.Log("Deleting %d pipeline(s)", len(testContext.Pipeline.CreatedPipelines))
		for _, pipeline := range testContext.Pipeline.CreatedPipelines {
			testutil.DeletePipeline(pipelineClient, pipeline.PipelineID)
		}
	})

	// ################## HELPER ##################

	// uploadPipeline uploads a pipeline from the given dir/file and returns
	// the pipeline ID and version ID.
	uploadPipeline := func(dir, file string) (string, string) {
		pipelineFilePath := filepath.Join(testutil.GetPipelineFilesDir(), dir, file)
		uploadedPipeline, err := testutil.UploadPipeline(pipelineUploadClient, pipelineFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)
		Expect(err).To(BeNil(), "Failed to upload pipeline %s", file)
		testContext.Pipeline.CreatedPipelines = append(testContext.Pipeline.CreatedPipelines, uploadedPipeline)
		version := testutil.GetLatestPipelineVersion(pipelineClient, &uploadedPipeline.PipelineID)
		return uploadedPipeline.PipelineID, version.PipelineVersionID
	}

	uploadSingleTaskPipeline := func() (string, string) {
		return uploadPipeline(singleTaskPipelineDir, singleTaskPipelineFile)
	}

	uploadMultiTaskPipeline := func() (string, string) {
		return uploadPipeline(multiTaskPipelineDir, multiTaskPipelineFile)
	}

	// ################## TESTS ##################

	Context("Single task pipeline with MLflow enabled >", func() {
		It("Should populate plugins_output.mlflow with experiment_id, root_run_id, and state=PLUGIN_SUCCEEDED", func() {
			pipelineID, versionID := uploadSingleTaskPipeline()
			experimentName := fmt.Sprintf("mlflow-test-%s", randomName)
			pluginsInput := e2e_utils.BuildMLflowPluginsInput(experimentName)
			pipelineRuntimeInputs := testutil.GetPipelineRunTimeInputs(
				filepath.Join(testutil.GetPipelineFilesDir(), singleTaskPipelineDir, singleTaskPipelineFile),
			)

			createdRun := e2e_utils.CreatePipelineRunWithPluginsInput(
				runClient, testContext, &pipelineID, &versionID, experimentID, pipelineRuntimeInputs, pluginsInput,
			)

			timeout := time.Duration(maxPipelineWaitTime)
			testutil.WaitForRunToBeInState(runClient, &createdRun.RunID, []run_model.V2beta1RuntimeState{
				run_model.V2beta1RuntimeStateSUCCEEDED,
				run_model.V2beta1RuntimeStateFAILED,
			}, &timeout)

			updatedRun := testutil.GetPipelineRun(runClient, &createdRun.RunID)
			Expect(updatedRun.State).NotTo(BeNil())
			Expect(*updatedRun.State).To(Equal(run_model.V2beta1RuntimeStateSUCCEEDED),
				"Pipeline run should succeed")

			// Verify KFP plugins_output
			err := e2e_utils.VerifyPluginsOutput(updatedRun, run_model.V2beta1PluginStatePLUGINSUCCEEDED)
			Expect(err).NotTo(HaveOccurred())

			// Verify MLflow side
			rootRunID, err := e2e_utils.GetPluginsOutputEntryValue(updatedRun, "root_run_id")
			Expect(err).NotTo(HaveOccurred())
			Expect(rootRunID).NotTo(BeEmpty(), "root_run_id should not be empty")

			mlflowExperimentID, err := e2e_utils.GetPluginsOutputEntryValue(updatedRun, "experiment_id")
			Expect(err).NotTo(HaveOccurred())
			Expect(mlflowExperimentID).NotTo(BeEmpty(), "experiment_id should not be empty")

			// Verify the MLflow experiment was created with the correct name
			mlflowExp, err := e2e_utils.QueryMLflowExperimentByName(mlflowEndpoint, experimentName)
			Expect(err).NotTo(HaveOccurred())
			Expect(mlflowExp.ID).To(Equal(mlflowExperimentID))

			// Verify parent run status
			err = e2e_utils.VerifyMLflowRunStatus(mlflowEndpoint, rootRunID, "FINISHED")
			Expect(err).NotTo(HaveOccurred())

			// Verify KFP tags on parent run
			err = e2e_utils.VerifyMLflowRunTags(mlflowEndpoint, rootRunID, map[string]string{
				"kfp.pipeline_run_id": updatedRun.RunID,
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should have nested MLflow run(s) with FINISHED status", func() {
			pipelineID, versionID := uploadSingleTaskPipeline()
			experimentName := fmt.Sprintf("mlflow-nested-%s", randomName)
			pluginsInput := e2e_utils.BuildMLflowPluginsInput(experimentName)
			pipelineRuntimeInputs := testutil.GetPipelineRunTimeInputs(
				filepath.Join(testutil.GetPipelineFilesDir(), singleTaskPipelineDir, singleTaskPipelineFile),
			)

			createdRun := e2e_utils.CreatePipelineRunWithPluginsInput(
				runClient, testContext, &pipelineID, &versionID, experimentID, pipelineRuntimeInputs, pluginsInput,
			)

			timeout := time.Duration(maxPipelineWaitTime)
			testutil.WaitForRunToBeInState(runClient, &createdRun.RunID, []run_model.V2beta1RuntimeState{
				run_model.V2beta1RuntimeStateSUCCEEDED,
			}, &timeout)

			updatedRun := testutil.GetPipelineRun(runClient, &createdRun.RunID)
			rootRunID, err := e2e_utils.GetPluginsOutputEntryValue(updatedRun, "root_run_id")
			Expect(err).NotTo(HaveOccurred())
			mlflowExperimentID, err := e2e_utils.GetPluginsOutputEntryValue(updatedRun, "experiment_id")
			Expect(err).NotTo(HaveOccurred())

			// Single-task pipeline should have exactly 1 nested run directly under the root
			nestedRuns, err := e2e_utils.QueryNestedRuns(mlflowEndpoint, rootRunID, mlflowExperimentID)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(nestedRuns)).To(Equal(1),
				"Should have exactly 1 nested MLflow run for a single-task pipeline")

			// Verify the nested run is FINISHED
			nestedRun := nestedRuns[0]
			Expect(nestedRun.Info.Status).To(Equal("FINISHED"),
				"Nested MLflow run should be FINISHED")

			// Verify kfp.task_name tag is present on the nested run
			var hasTaskNameTag bool
			for _, tag := range nestedRun.Data.Tags {
				if tag.Key == "kfp.task_name" {
					hasTaskNameTag = true
					Expect(tag.Value).NotTo(BeEmpty(),
						"kfp.task_name tag should have a non-empty value")
					break
				}
			}
			Expect(hasTaskNameTag).To(BeTrue(),
				"Nested MLflow run should have a kfp.task_name tag")
		})
	})

	Context("Multi-task pipeline with MLflow >", func() {
		// producer_consumer_param_pipeline.yaml has 2 executor tasks: producer → consumer
		const expectedTaskCount = 2

		It("Should create one parent run and one nested run per task, all FINISHED", func() {
			pipelineID, versionID := uploadMultiTaskPipeline()
			experimentName := fmt.Sprintf("mlflow-multi-%s", randomName)
			pluginsInput := e2e_utils.BuildMLflowPluginsInput(experimentName)
			pipelineRuntimeInputs := testutil.GetPipelineRunTimeInputs(
				filepath.Join(testutil.GetPipelineFilesDir(), multiTaskPipelineDir, multiTaskPipelineFile),
			)

			createdRun := e2e_utils.CreatePipelineRunWithPluginsInput(
				runClient, testContext, &pipelineID, &versionID, experimentID, pipelineRuntimeInputs, pluginsInput,
			)

			timeout := time.Duration(maxPipelineWaitTime)
			testutil.WaitForRunToBeInState(runClient, &createdRun.RunID, []run_model.V2beta1RuntimeState{
				run_model.V2beta1RuntimeStateSUCCEEDED,
				run_model.V2beta1RuntimeStateFAILED,
			}, &timeout)

			updatedRun := testutil.GetPipelineRun(runClient, &createdRun.RunID)
			Expect(updatedRun.State).NotTo(BeNil())
			Expect(*updatedRun.State).To(Equal(run_model.V2beta1RuntimeStateSUCCEEDED),
				"Multi-task pipeline run should succeed")

			// Verify KFP plugins_output
			err := e2e_utils.VerifyPluginsOutput(updatedRun, run_model.V2beta1PluginStatePLUGINSUCCEEDED)
			Expect(err).NotTo(HaveOccurred())

			// Verify parent MLflow run
			rootRunID, err := e2e_utils.GetPluginsOutputEntryValue(updatedRun, "root_run_id")
			Expect(err).NotTo(HaveOccurred())
			err = e2e_utils.VerifyMLflowRunStatus(mlflowEndpoint, rootRunID, "FINISHED")
			Expect(err).NotTo(HaveOccurred())

			// Verify there is one nested run per task
			mlflowExperimentID, err := e2e_utils.GetPluginsOutputEntryValue(updatedRun, "experiment_id")
			Expect(err).NotTo(HaveOccurred())
			nestedCount, err := e2e_utils.CountNestedRuns(mlflowEndpoint, rootRunID, mlflowExperimentID)
			Expect(err).NotTo(HaveOccurred())
			Expect(nestedCount).To(Equal(expectedTaskCount),
				fmt.Sprintf("Should have exactly %d nested MLflow runs (one per task)", expectedTaskCount))

			// Verify all nested runs are FINISHED
			allRuns, err := e2e_utils.QueryMLflowRuns(mlflowEndpoint, mlflowExperimentID)
			Expect(err).NotTo(HaveOccurred())
			for _, run := range allRuns {
				Expect(run.Info.Status).To(Equal("FINISHED"),
					fmt.Sprintf("MLflow run %s should be FINISHED", run.Info.RunID))
			}
		})
	})

	Context("Backward compatibility — no plugins_input >", func() {
		It("Should succeed and populate plugins_output with defaults when no plugins_input is provided", func() {
			pipelineID, versionID := uploadSingleTaskPipeline()
			pipelineRuntimeInputs := testutil.GetPipelineRunTimeInputs(
				filepath.Join(testutil.GetPipelineFilesDir(), singleTaskPipelineDir, singleTaskPipelineFile),
			)

			// Create run without plugins_input
			createdRun := e2e_utils.CreatePipelineRun(
				runClient, testContext, &pipelineID, &versionID, experimentID, pipelineRuntimeInputs,
			)

			timeout := time.Duration(maxPipelineWaitTime)
			testutil.WaitForRunToBeInState(runClient, &createdRun.RunID, []run_model.V2beta1RuntimeState{
				run_model.V2beta1RuntimeStateSUCCEEDED,
				run_model.V2beta1RuntimeStateFAILED,
			}, &timeout)

			updatedRun := testutil.GetPipelineRun(runClient, &createdRun.RunID)
			Expect(updatedRun.State).NotTo(BeNil())
			Expect(*updatedRun.State).To(Equal(run_model.V2beta1RuntimeStateSUCCEEDED),
				"Pipeline run should succeed even without plugins_input")

			err := e2e_utils.VerifyPluginsOutput(updatedRun, run_model.V2beta1PluginStatePLUGINSUCCEEDED)
			Expect(err).NotTo(HaveOccurred())

			// Verify the default MLflow experiment and parent run exist
			rootRunID, err := e2e_utils.GetPluginsOutputEntryValue(updatedRun, "root_run_id")
			Expect(err).NotTo(HaveOccurred())
			Expect(rootRunID).NotTo(BeEmpty(), "root_run_id should be present even without plugins_input")

			err = e2e_utils.VerifyMLflowRunStatus(mlflowEndpoint, rootRunID, "FINISHED")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("MLflow opt-out via disabled flag >", func() {
		It("Should succeed with no MLflow output when plugins_input.mlflow.disabled=true", func() {
			pipelineID, versionID := uploadSingleTaskPipeline()
			pluginsInput := e2e_utils.BuildMLflowPluginsInputDisabled()
			pipelineRuntimeInputs := testutil.GetPipelineRunTimeInputs(
				filepath.Join(testutil.GetPipelineFilesDir(), singleTaskPipelineDir, singleTaskPipelineFile),
			)

			createdRun := e2e_utils.CreatePipelineRunWithPluginsInput(
				runClient, testContext, &pipelineID, &versionID, experimentID, pipelineRuntimeInputs, pluginsInput,
			)

			timeout := time.Duration(maxPipelineWaitTime)
			testutil.WaitForRunToBeInState(runClient, &createdRun.RunID, []run_model.V2beta1RuntimeState{
				run_model.V2beta1RuntimeStateSUCCEEDED,
				run_model.V2beta1RuntimeStateFAILED,
			}, &timeout)

			updatedRun := testutil.GetPipelineRun(runClient, &createdRun.RunID)
			Expect(updatedRun.State).NotTo(BeNil())
			Expect(*updatedRun.State).To(Equal(run_model.V2beta1RuntimeStateSUCCEEDED),
				"Pipeline run should succeed when MLflow is disabled via plugins_input")

			// No MLflow output should be present
			err := e2e_utils.VerifyNoPluginsOutput(updatedRun)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("Parallel-for pipeline with MLflow >", func() {
		// parallel_for_after_dependency.yaml has a parallel-for loop with 3 iterations
		// and 2 dependent tasks.
		const parallelForPipelineFile = "parallel_for_after_dependency.yaml"
		const parallelForPipelineDir = "valid/critical"

		It("Should create parent run with correct 2-level nesting hierarchy", func() {
			pipelineID, versionID := uploadPipeline(parallelForPipelineDir, parallelForPipelineFile)
			experimentName := fmt.Sprintf("mlflow-loop-%s", randomName)
			pluginsInput := e2e_utils.BuildMLflowPluginsInput(experimentName)
			pipelineRuntimeInputs := testutil.GetPipelineRunTimeInputs(
				filepath.Join(testutil.GetPipelineFilesDir(), parallelForPipelineDir, parallelForPipelineFile),
			)

			createdRun := e2e_utils.CreatePipelineRunWithPluginsInput(
				runClient, testContext, &pipelineID, &versionID, experimentID, pipelineRuntimeInputs, pluginsInput,
			)

			timeout := time.Duration(maxPipelineWaitTime)
			testutil.WaitForRunToBeInState(runClient, &createdRun.RunID, []run_model.V2beta1RuntimeState{
				run_model.V2beta1RuntimeStateSUCCEEDED,
				run_model.V2beta1RuntimeStateFAILED,
			}, &timeout)

			updatedRun := testutil.GetPipelineRun(runClient, &createdRun.RunID)
			Expect(updatedRun.State).NotTo(BeNil())
			Expect(*updatedRun.State).To(Equal(run_model.V2beta1RuntimeStateSUCCEEDED),
				"Parallel-for pipeline run should succeed")

			// Verify KFP plugins_output
			err := e2e_utils.VerifyPluginsOutput(updatedRun, run_model.V2beta1PluginStatePLUGINSUCCEEDED)
			Expect(err).NotTo(HaveOccurred())

			// Verify parent MLflow run is FINISHED
			rootRunID, err := e2e_utils.GetPluginsOutputEntryValue(updatedRun, "root_run_id")
			Expect(err).NotTo(HaveOccurred())
			err = e2e_utils.VerifyMLflowRunStatus(mlflowEndpoint, rootRunID, "FINISHED")
			Expect(err).NotTo(HaveOccurred())

			// Direct children of root: 1 loop run + 2 dependent tasks = 3
			mlflowExperimentID, err := e2e_utils.GetPluginsOutputEntryValue(updatedRun, "experiment_id")
			Expect(err).NotTo(HaveOccurred())
			const expectedDirectChildren = 3
			directChildCount, err := e2e_utils.CountNestedRuns(mlflowEndpoint, rootRunID, mlflowExperimentID)
			Expect(err).NotTo(HaveOccurred())
			Expect(directChildCount).To(Equal(expectedDirectChildren),
				"Should have exactly 3 direct children of root (1 loop run + 2 dependent tasks)")

			// Find the loop nested run (has iterations as children).
			// It's the one direct child that itself has nested runs.
			allRuns, err := e2e_utils.QueryMLflowRuns(mlflowEndpoint, mlflowExperimentID)
			Expect(err).NotTo(HaveOccurred())

			var loopRunID string
			for _, run := range allRuns {
				if run.Info.RunID == rootRunID {
					continue // skip parent
				}
				childCount, err := e2e_utils.CountNestedRuns(mlflowEndpoint, run.Info.RunID, mlflowExperimentID)
				Expect(err).NotTo(HaveOccurred())
				if childCount > 0 {
					loopRunID = run.Info.RunID
					// The loop run should have exactly 3 iteration children
					Expect(childCount).To(Equal(3),
						"Loop nested run should have exactly 3 iteration children")
					break
				}
			}
			Expect(loopRunID).NotTo(BeEmpty(),
				"Should find a loop nested run with iteration children")

			// Verify all MLflow runs in the experiment are FINISHED
			for _, run := range allRuns {
				Expect(run.Info.Status).To(Equal("FINISHED"),
					fmt.Sprintf("MLflow run %s should be FINISHED", run.Info.RunID))
			}

			// Total runs in experiment: 1 parent + 3 direct + 3 iterations = 7
			Expect(len(allRuns)).To(Equal(7),
				"Should have exactly 7 MLflow runs in the experiment")
		})
	})

	Context("Failed pipeline with MLflow >", func() {
		const failPipelineFile = "fail_v2.yaml"
		const failPipelineDir = "valid/failing"

		It("Should mark parent and nested MLflow runs as FAILED", func() {
			pipelineID, versionID := uploadPipeline(failPipelineDir, failPipelineFile)
			experimentName := fmt.Sprintf("mlflow-fail-%s", randomName)
			pluginsInput := e2e_utils.BuildMLflowPluginsInput(experimentName)
			pipelineRuntimeInputs := testutil.GetPipelineRunTimeInputs(
				filepath.Join(testutil.GetPipelineFilesDir(), failPipelineDir, failPipelineFile),
			)

			createdRun := e2e_utils.CreatePipelineRunWithPluginsInput(
				runClient, testContext, &pipelineID, &versionID, experimentID, pipelineRuntimeInputs, pluginsInput,
			)

			timeout := time.Duration(maxPipelineWaitTime)
			testutil.WaitForRunToBeInState(runClient, &createdRun.RunID, []run_model.V2beta1RuntimeState{
				run_model.V2beta1RuntimeStateFAILED,
			}, &timeout)

			updatedRun := testutil.GetPipelineRun(runClient, &createdRun.RunID)
			Expect(updatedRun.State).NotTo(BeNil())
			Expect(*updatedRun.State).To(Equal(run_model.V2beta1RuntimeStateFAILED),
				"Pipeline run should be FAILED")

			// Verify MLflow plugins_output still records the run
			err := e2e_utils.VerifyPluginsOutput(updatedRun, run_model.V2beta1PluginStatePLUGINSUCCEEDED)
			Expect(err).NotTo(HaveOccurred())

			rootRunID, err := e2e_utils.GetPluginsOutputEntryValue(updatedRun, "root_run_id")
			Expect(err).NotTo(HaveOccurred())
			Expect(rootRunID).NotTo(BeEmpty(), "root_run_id should not be empty")

			// The parent MLflow run should be FAILED because the KFP run failed
			err = e2e_utils.VerifyMLflowRunStatus(mlflowEndpoint, rootRunID, "FAILED")
			Expect(err).NotTo(HaveOccurred())

			// Verify any nested runs are also in a terminal state (FAILED)
			mlflowExperimentID, err := e2e_utils.GetPluginsOutputEntryValue(updatedRun, "experiment_id")
			Expect(err).NotTo(HaveOccurred())
			allRuns, err := e2e_utils.QueryMLflowRuns(mlflowEndpoint, mlflowExperimentID)
			Expect(err).NotTo(HaveOccurred())
			for _, run := range allRuns {
				if run.Info.RunID != rootRunID {
					// Nested runs for failed tasks should be FAILED
					Expect(run.Info.Status).To(Equal("FAILED"),
						fmt.Sprintf("Nested MLflow run %s should be FAILED", run.Info.RunID))
				}
			}
		})
	})

	Context("Failed pipeline + RetryRun with MLflow >", func() {
		const failPipelineFile = "fail_v2.yaml"
		const failPipelineDir = "valid/failing"

		It("Should reopen MLflow runs on retry and then reflect the retried status", func() {
			pipelineID, versionID := uploadPipeline(failPipelineDir, failPipelineFile)
			experimentName := fmt.Sprintf("mlflow-retry-%s", randomName)
			pluginsInput := e2e_utils.BuildMLflowPluginsInput(experimentName)
			pipelineRuntimeInputs := testutil.GetPipelineRunTimeInputs(
				filepath.Join(testutil.GetPipelineFilesDir(), failPipelineDir, failPipelineFile),
			)

			// Create the run and wait for it to FAIL
			createdRun := e2e_utils.CreatePipelineRunWithPluginsInput(
				runClient, testContext, &pipelineID, &versionID, experimentID, pipelineRuntimeInputs, pluginsInput,
			)

			timeout := time.Duration(maxPipelineWaitTime)
			testutil.WaitForRunToBeInState(runClient, &createdRun.RunID, []run_model.V2beta1RuntimeState{
				run_model.V2beta1RuntimeStateFAILED,
			}, &timeout)

			updatedRun := testutil.GetPipelineRun(runClient, &createdRun.RunID)
			Expect(updatedRun.State).NotTo(BeNil())
			Expect(*updatedRun.State).To(Equal(run_model.V2beta1RuntimeStateFAILED),
				"Pipeline run should initially be FAILED")

			rootRunID, err := e2e_utils.GetPluginsOutputEntryValue(updatedRun, "root_run_id")
			Expect(err).NotTo(HaveOccurred())
			Expect(rootRunID).NotTo(BeEmpty(), "root_run_id should not be empty")

			// Retry the run
			e2e_utils.RetryPipelineRun(runClient, createdRun.RunID)

			// Wait for the retried run to reach terminal state
			testutil.WaitForRunToBeInState(runClient, &createdRun.RunID, []run_model.V2beta1RuntimeState{
				run_model.V2beta1RuntimeStateFAILED,
			}, &timeout)

			retriedRun := testutil.GetPipelineRun(runClient, &createdRun.RunID)
			Expect(retriedRun.State).NotTo(BeNil())
			Expect(*retriedRun.State).To(Equal(run_model.V2beta1RuntimeStateFAILED),
				"Retried pipeline run should still be FAILED (fail_v2 always fails)")

			// Verify the MLflow parent run reflects the retry
			err = e2e_utils.VerifyMLflowRunStatus(mlflowEndpoint, rootRunID, "FAILED")
			Expect(err).NotTo(HaveOccurred())

			// Verify plugins_output is still populated after retry
			err = e2e_utils.VerifyPluginsOutput(retriedRun, run_model.V2beta1PluginStatePLUGINSUCCEEDED)
			Expect(err).NotTo(HaveOccurred())

			// Verify retry reused the existing parent run
			retriedRootRunID, err := e2e_utils.GetPluginsOutputEntryValue(retriedRun, "root_run_id")
			Expect(err).NotTo(HaveOccurred())
			Expect(retriedRootRunID).To(Equal(rootRunID),
				"OnRunRetry should reopen the existing parent MLflow run, not create a new one")
		})
	})

	Context("Custom experiment name >", func() {
		It("Should create MLflow experiment with the user-specified name", func() {
			pipelineID, versionID := uploadSingleTaskPipeline()
			customExpName := fmt.Sprintf("custom-exp-%s", randomName)
			pluginsInput := e2e_utils.BuildMLflowPluginsInput(customExpName)
			pipelineRuntimeInputs := testutil.GetPipelineRunTimeInputs(
				filepath.Join(testutil.GetPipelineFilesDir(), singleTaskPipelineDir, singleTaskPipelineFile),
			)

			createdRun := e2e_utils.CreatePipelineRunWithPluginsInput(
				runClient, testContext, &pipelineID, &versionID, experimentID, pipelineRuntimeInputs, pluginsInput,
			)

			timeout := time.Duration(maxPipelineWaitTime)
			testutil.WaitForRunToBeInState(runClient, &createdRun.RunID, []run_model.V2beta1RuntimeState{
				run_model.V2beta1RuntimeStateSUCCEEDED,
			}, &timeout)

			updatedRun := testutil.GetPipelineRun(runClient, &createdRun.RunID)
			mlflowExperimentID, err := e2e_utils.GetPluginsOutputEntryValue(updatedRun, "experiment_id")
			Expect(err).NotTo(HaveOccurred())

			// Verify the MLflow experiment name matches
			mlflowExp, err := e2e_utils.QueryMLflowExperimentByName(mlflowEndpoint, customExpName)
			Expect(err).NotTo(HaveOccurred())
			Expect(mlflowExp.ID).To(Equal(mlflowExperimentID),
				"MLflow experiment ID should match the one in plugins_output")
			Expect(mlflowExp.Name).To(Equal(customExpName),
				"MLflow experiment name should match the user-specified name")
		})
	})
})

// Additional scenarios from proposals/12862-mlflow-integration/MLflow-KFP-Integration-TestPlan*.md.
// Uses PDescribe the same way as pipeline_api_test.go.

var _ = PDescribe("MLflow Integration > CreateRun and validation >", Label(MLflow, FullRegression), func() {
	Context("Experiment and plugins_input >", func() {
		It("Should use default admin experiment name when no plugins_input and assert run_url in plugins_output", func() {
		})
		It("Should prefer experiment_id over experiment_name when both are set", func() {
		})
		It("Should create two separate MLflow experiments and parent runs for two runs with different experiment_name (same pipeline)", func() {
		})
		It("Should recover when concurrent creates race and MLflow reports experiment already exists", func() {
		})
	})
	Context("API and workflow semantics >", func() {
		It("Should reject CreateRun when client supplies plugins_output", func() {
		})
		It("Should expose MLflow runtime JSON env on driver and launcher templates only after compile", func() {
		})
		It("Should update MLflow parent to Failed when MLflow parent exists then workflow create or DB write fails", func() {
		})
	})
	Context("Tags and deep links >", func() {
		It("Should set MLflow tags for pipeline_id and pipeline_version_id when present", func() {
		})
		It("Should produce usable KFP and MLflow deep links in tags and plugins_output", func() {
		})
		It("Should honor experiment description omit, empty string, and custom values per defaults", func() {
		})
	})
})

var _ = PDescribe("MLflow Integration > Terminal state and sync >", Label(MLflow, FullRegression), func() {
	Context("Pipeline outcomes >", func() {
		It("Should map MLflow parent and nested runs when pipeline is Canceled", func() {
		})
	})
	Context("Reliability >", func() {
		It("Should update straggler nested runs with pagination for large fan-out", func() {
		})
		It("Should persist latest plugins_output on the KFP run after terminal handling", func() {
		})
		It("Should record KFP terminal state when MLflow sync returns errors", func() {
		})
		It("Should handle terminal sync with corrupt or missing stored MLflow ids in plugin output", func() {
		})
		It("Should retry MLflow HTTP only where appropriate", func() {
		})
	})
})

var _ = PDescribe("MLflow Integration > RetryRun >", Label(MLflow, FullRegression), func() {
	Context("Edge cases >", func() {
		It("Should have no MLflow side effects when retrying a failed run with no MLflow plugin output", func() {
		})
	})
})

var _ = PDescribe("MLflow Integration > CloneRun >", Label(MLflow, FullRegression), func() {
	Context("Clone behavior >", func() {
		It("Should inherit plugins_input.mlflow on clone and create a new MLflow parent for the cloned run", func() {
		})
		It("Should use distinct MLflow run ids for clone vs original when both complete", func() {
		})
	})
})

var _ = PDescribe("MLflow Integration > Recurring runs >", Label(MLflow, FullRegression), func() {
	Context("Scheduled jobs >", func() {
		It("Should store plugins_input on recurring job and surface it on the scheduled workflow", func() {
		})
		It("Should propagate plugins_input to runs created by each schedule trigger", func() {
		})
		It("Should accept recurring job referencing pipeline version with plugins without inline workflow spec", func() {
		})
		It("Should create one MLflow parent per schedule trigger", func() {
		})
	})
})

var _ = PDescribe("MLflow Integration > Negative and edge cases >", Label(MLflow, FullRegression), func() {
	Context("Configuration and validation >", func() {
		It("Should allow or clearly fail CreateRun when MLflow endpoint is unreachable at create", func() {
		})
		It("Should reject malformed plugins_input.mlflow JSON or unknown fields with validation error", func() {
		})
	})
})

var _ = PDescribe("MLflow Integration > Compiled workflow injection >", Label(MLflow, FullRegression), func() {
	Context("Templates and env >", func() {
		It("Should inject MLflow env only on driver and launcher templates", func() {
		})
		It("Should replace duplicate env keys consistently when template predefines the same name", func() {
		})
		It("Should omit MLflow env when plugins.mlflow is not configured", func() {
		})
	})
})

var _ = PDescribe("MLflow Integration > Cluster and deployment >", Label(MLflow, FullRegression), func() {
	Context("Lifecycle and platform >", func() {
		It("Should leave existing behavior unchanged with no plugins.mlflow (upgrade path)", func() {
		})
		It("Should respect per-namespace kfp-launcher overrides for MLflow in multi-tenant setups", func() {
		})
		It("Should allow deleting KFP run while MLflow runs may remain", func() {
		})
		It("Should succeed against MLflow served over HTTPS", func() {
		})
	})
})
