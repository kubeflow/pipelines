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
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/experiment_model"
	upload_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_client/pipeline_upload_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_model"
	recurring_run_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/recurring_run_client/recurring_run_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/recurring_run_model"
	run_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_client/run_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_model"
	workflowutils "github.com/kubeflow/pipelines/backend/test/compiler/utils"
	"github.com/kubeflow/pipelines/backend/test/config"
	. "github.com/kubeflow/pipelines/backend/test/constants"
	e2e_utils "github.com/kubeflow/pipelines/backend/test/end2end/utils"
	"github.com/kubeflow/pipelines/backend/test/logger"
	"github.com/kubeflow/pipelines/backend/test/testutil"
	apitests "github.com/kubeflow/pipelines/backend/test/v2/api"

	"github.com/go-openapi/strfmt"
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
	. "github.com/onsi/gomega"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Upload and Verify Pipeline Run >", Label(FullRegression), func() {
	var testContext *apitests.TestContext

	// ################## SET AND TEARDOWN ##################

	BeforeEach(func() {
		logger.Log("################### Setup before each Pipeline Upload test #####################")
		logger.Log("################### Global Setup before each test #####################")
		testContext = &apitests.TestContext{
			TestStartTimeUTC: time.Now(),
		}
		logger.Log("Test Context: %p", testContext)
		randomName = strconv.FormatInt(time.Now().UnixNano(), 10)
		testContext.Pipeline.UploadParams = upload_params.NewUploadPipelineParams()
		testContext.Pipeline.PipelineGeneratedName = "e2e-test-" + randomName
		testContext.Pipeline.CreatedPipelines = make([]*pipeline_upload_model.V2beta1Pipeline, 0)
		testContext.PipelineRun.CreatedRunIds = make([]string, 0)
		testContext.Pipeline.ExpectedPipeline = new(pipeline_upload_model.V2beta1Pipeline)
		testContext.Pipeline.ExpectedPipeline.CreatedAt = strfmt.DateTime(testContext.TestStartTimeUTC)
		var secrets []*v1.Secret
		secret1 := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-secret-1",
				Namespace: testutil.GetNamespace()},
			Data: map[string][]byte{
				"username": []byte("user1"),
			},
			Type: v1.SecretTypeOpaque,
		}
		secret2 := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-secret-2",
				Namespace: testutil.GetNamespace()},
			Data: map[string][]byte{
				"password": []byte("psw1"),
			},
			Type: v1.SecretTypeOpaque,
		}
		secret3 := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-secret-3",
				Namespace: testutil.GetNamespace()},
			Data: map[string][]byte{
				"password": []byte("psw2"),
			},
			Type: v1.SecretTypeOpaque,
		}
		secrets = append(secrets, secret1, secret2, secret3)
		for _, secret := range secrets {
			testutil.CreateSecret(k8Client, testutil.GetNamespace(), secret)
		}
	})

	AfterEach(func() {
		logger.Log("################### Global Cleanup after each test #####################")
	})

	ReportAfterEach(func(specReport types.SpecReport) {
		if testContext == nil {
			return
		}
		logger.Log("Deleting %d recurring run(s)", len(testContext.RecurringRun.CreatedRecurringRunIds))
		for _, recurringRunID := range testContext.RecurringRun.CreatedRecurringRunIds {
			testutil.DeleteRecurringRun(recurringRunClient, recurringRunID)
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
		if len(testContext.Experiment.CreatedExperimentIds) > 0 {
			for _, experimentID := range testContext.Experiment.CreatedExperimentIds {
				experimentID := experimentID
				testutil.DeleteExperiment(experimentClient, experimentID)
			}
		}
		logger.Log("Deleting %d pipeline(s)", len(testContext.Pipeline.CreatedPipelines))
		for _, pipeline := range testContext.Pipeline.CreatedPipelines {
			pipelineID := pipeline.PipelineID
			testutil.DeletePipeline(pipelineClient, pipelineID)
		}
	})

	// ################## TESTS ##################

	Context("Upload a pipeline file, run it and verify that pipeline run succeeds >", FlakeAttempts(2), Label(E2eEssential), func() {
		var pipelineDir = "valid/essential"
		pipelineFiles := testutil.GetListOfFilesInADir(filepath.Join(testutil.GetPipelineFilesDir(), pipelineDir))
		for _, pipelineFile := range pipelineFiles {
			It(fmt.Sprintf("Upload %s pipeline", pipelineFile), func() {
				validatePipelineRunSuccess(pipelineFile, pipelineDir, testContext)
			})
		}
	})

	// Few of the following pipelines randomly fail in Multi User Mode during CI run - which is why a FlakeAttempt is added, but we need to investigate, create ticket and fix it in the future
	Context("Upload a pipeline file, run it and verify that pipeline run succeeds >", FlakeAttempts(2), Label("Sample", E2eCritical), func() {
		var pipelineDir = "valid/critical"
		pipelineFiles := testutil.GetListOfFilesInADir(filepath.Join(testutil.GetPipelineFilesDir(), pipelineDir))
		for _, pipelineFile := range pipelineFiles {
			It(fmt.Sprintf("Upload %s pipeline", pipelineFile), FlakeAttempts(2), func() {
				validatePipelineRunSuccess(pipelineFile, pipelineDir, testContext)
			})
		}
	})

	Context("Upload a pipeline file, run it and verify that pipeline run succeeds Smoke >", FlakeAttempts(2), Label(Smoke), func() {
		var pipelineDir = "valid"
		pipelineFiles := []string{
			"essential/iris_pipeline_compiled.yaml",
			"essential/component_with_pip_index_urls.yaml",
			"critical/flip_coin.yaml",
			"critical/pipeline_with_artifact_upload_download.yaml",
			"critical/parallel_for_after_dependency.yaml",
		}
		for _, pipelineFile := range pipelineFiles {
			It(fmt.Sprintf("Upload %s pipeline", pipelineFile), FlakeAttempts(2), func() {
				validatePipelineRunSuccess(pipelineFile, pipelineDir, testContext)
			})
		}
	})

	Context("Upload a pipeline file, run it and verify that pipeline run succeeds >", FlakeAttempts(2), Label(Sanity), func() {
		var pipelineDir = "valid"
		pipelineFiles := []string{
			"essential/component_with_pip_index_urls.yaml",
			"essential/lightweight_python_functions_pipeline.yaml",
			"essential/pipeline_in_pipeline.yaml",
			"critical/pipeline_with_secret_as_env.yaml",
			"critical/pipeline_with_input_status_state.yaml",
			"critical/notebook_component_simple.yaml",
		}
		for _, pipelineFile := range pipelineFiles {
			It(fmt.Sprintf("Upload %s pipeline", pipelineFile), FlakeAttempts(2), func() {
				validatePipelineRunSuccess(pipelineFile, pipelineDir, testContext)
			})
		}
	})

	Context("Upload a pipeline file, run it and verify that pipeline run succeeds Smoke >", FlakeAttempts(1), Label(Integration), func() {
		var pipelineDir = "valid/integration"
		pipelineFiles := testutil.GetListOfFilesInADir(filepath.Join(testutil.GetPipelineFilesDir(), pipelineDir))
		for _, pipelineFile := range pipelineFiles {
			It(fmt.Sprintf("Upload %s pipeline", pipelineFile), FlakeAttempts(2), func() {
				validatePipelineRunSuccess(pipelineFile, pipelineDir, testContext)
			})
		}
	})

	Context("Create a pipeline run with HTTP proxy >", Label(E2eProxy), func() {
		var pipelineDir = "valid"
		pipelineFile := "env-var.yaml"
		It(fmt.Sprintf("Create a pipeline run with http proxy, using specs: %s", pipelineFile), func() {
			pipelineFilePath := filepath.Join(testutil.GetPipelineFilesDir(), pipelineDir, pipelineFile)
			uploadedPipeline, uploadErr := testutil.UploadPipeline(pipelineUploadClient, pipelineFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)
			Expect(uploadErr).To(BeNil(), "Failed to upload pipeline %s", pipelineFile)
			testContext.Pipeline.CreatedPipelines = append(testContext.Pipeline.CreatedPipelines, uploadedPipeline)
			createdPipelineVersion := testutil.GetLatestPipelineVersion(pipelineClient, &uploadedPipeline.PipelineID)
			createdExperiment := testutil.CreateExperimentWithParams(experimentClient, &experiment_model.V2beta1Experiment{
				DisplayName: "ProxyTest-" + randomName,
				Namespace:   testutil.GetNamespace(),
			})
			testContext.Experiment.CreatedExperimentIds = append(testContext.Experiment.CreatedExperimentIds, createdExperiment.ExperimentID)
			pipelineRuntimeInputs := map[string]interface{}{
				"env_var": "http_proxy",
			}
			createdRunID := e2e_utils.CreatePipelineRunAndWaitForItToFinish(runClient, testContext, uploadedPipeline.PipelineID, uploadedPipeline.DisplayName, &createdPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, pipelineRuntimeInputs, maxPipelineWaitTime)
			if *config.RunProxyTests {
				logger.Log("Deserializing expected compiled workflow file '%s' for the pipeline", pipelineFile)
				compiledWorkflow := workflowutils.UnmarshallWorkflowYAML(filepath.Join(testutil.GetCompiledWorkflowsFilesDir(), pipelineFile))
				e2e_utils.ValidateComponentStatuses(runClient, k8Client, testContext, createdRunID, compiledWorkflow)
			} else {
				runState := testutil.GetPipelineRun(runClient, &createdRunID).State
				expectedRunState := run_model.V2beta1RuntimeStateFAILED
				Expect(runState).To(Equal(&expectedRunState), fmt.Sprintf("Expected run with id=%s to fail with proxy=false", createdRunID))
			}
		})
	})

	Context("Upload a pipeline file, run it and verify that pipeline run fails >", Label(E2eFailed), func() {
		var pipelineDir = "valid/failing"
		pipelineFiles := testutil.GetListOfFilesInADir(filepath.Join(testutil.GetPipelineFilesDir(), pipelineDir))
		for _, pipelineFile := range pipelineFiles {
			It(fmt.Sprintf("Upload %s pipeline", pipelineFile), func() {
				testutil.CheckIfSkipping(pipelineFile)
				pipelineFilePath := filepath.Join(testutil.GetPipelineFilesDir(), pipelineDir, pipelineFile)
				logger.Log("Uploading pipeline file %s", pipelineFile)
				uploadedPipeline, uploadErr := testutil.UploadPipeline(pipelineUploadClient, pipelineFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)
				Expect(uploadErr).To(BeNil(), "Failed to upload pipeline %s", pipelineFile)
				testContext.Pipeline.CreatedPipelines = append(testContext.Pipeline.CreatedPipelines, uploadedPipeline)
				logger.Log("Upload of pipeline file '%s' successful", pipelineFile)
				uploadedPipelineVersion := testutil.GetLatestPipelineVersion(pipelineClient, &uploadedPipeline.PipelineID)
				pipelineRuntimeInputs := testutil.GetPipelineRunTimeInputs(pipelineFilePath)
				createdRunID := e2e_utils.CreatePipelineRunAndWaitForItToFinish(runClient, testContext, uploadedPipeline.PipelineID, uploadedPipeline.DisplayName, &uploadedPipelineVersion.PipelineVersionID, experimentID, pipelineRuntimeInputs, maxPipelineWaitTime)
				logger.Log("Fetching updated pipeline run details for run with id=%s", createdRunID)
				updatedRun := testutil.GetPipelineRun(runClient, &createdRunID)
				Expect(updatedRun.State).NotTo(BeNil(), "Updated pipeline run state is Nil")
				Expect(*updatedRun.State).To(Equal(run_model.V2beta1RuntimeStateFAILED), "Pipeline run was expected to fail, but is "+*updatedRun.State)

			})
		}
	})

	Context("Pipeline run parallelism tests >", Label(E2eEssential), func() {
		var pipelineFile = "essential/pipeline_with_max_active_runs.yaml"
		var pipelineDir = "valid"

		It("Test 1: > MaxParallelism runs of a single pipeline version - only MaxParallelism runs should be active", func() {
			pipelineFilePath := filepath.Join(testutil.GetPipelineFilesDir(), pipelineDir, pipelineFile)
			limit, err := e2e_utils.MaxActiveRuns(pipelineFilePath)
			Expect(err).NotTo(HaveOccurred(), "Pipeline should have max_active_runs configured")
			Expect(limit).To(BeNumerically(">", 0), "max_active_runs should be greater than 0")

			uploadedPipeline, uploadErr := testutil.UploadPipeline(pipelineUploadClient, pipelineFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)
			Expect(uploadErr).To(BeNil(), "Failed to upload pipeline")
			testContext.Pipeline.CreatedPipelines = append(testContext.Pipeline.CreatedPipelines, uploadedPipeline)
			uploadedPipelineVersion := testutil.GetLatestPipelineVersion(pipelineClient, &uploadedPipeline.PipelineID)

			// Launch (limit + 2) runs to exercise the semaphore
			targetRuns := int(limit) + 2
			runInfos := make([]e2e_utils.RunInfo, 0, targetRuns)
			for i := 0; i < targetRuns; i++ {
				created := e2e_utils.CreatePipelineRun(runClient, testContext, &uploadedPipeline.PipelineID, &uploadedPipelineVersion.PipelineVersionID, experimentID, nil)
				runInfos = append(runInfos, e2e_utils.RunInfo{
					RunID:             created.RunID,
					PipelineID:        uploadedPipeline.PipelineID,
					PipelineVersionID: uploadedPipelineVersion.PipelineVersionID,
				})
			}

			versionLimitMap := map[string]int32{
				uploadedPipelineVersion.PipelineVersionID: limit,
			}
			e2e_utils.ValidateParallelismAcrossRuns(runClient, runInfos, versionLimitMap, maxPipelineWaitTime)
		})

		It("Test 2: > MaxParallelism runs of a single pipeline but with different versions - all runs should be active", func() {
			pipelineFilePath := filepath.Join(testutil.GetPipelineFilesDir(), pipelineDir, pipelineFile)
			limit, ok := e2e_utils.MaxActiveRuns(pipelineFilePath)
			Expect(ok).To(BeTrue(), "Pipeline should have max_active_runs configured")

			uploadedPipeline, uploadErr := testutil.UploadPipeline(pipelineUploadClient, pipelineFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)
			Expect(uploadErr).To(BeNil(), "Failed to upload pipeline")
			testContext.Pipeline.CreatedPipelines = append(testContext.Pipeline.CreatedPipelines, uploadedPipeline)
			version1 := testutil.GetLatestPipelineVersion(pipelineClient, &uploadedPipeline.PipelineID)

			// Upload a second version of the same pipeline
			uploadParams := upload_params.NewUploadPipelineVersionParams()
			uploadParams.Pipelineid = &uploadedPipeline.PipelineID
			version2, uploadErr2 := pipelineUploadClient.UploadPipelineVersion(pipelineFilePath, uploadParams)
			Expect(uploadErr2).To(BeNil(), "Failed to upload second pipeline version")

			// Launch all runs for version1 first, then all runs for version2 (sequential batches)
			targetRuns := int(limit) + 2
			runInfos := make([]e2e_utils.RunInfo, 0, targetRuns*2)
			// Launch all version1 runs first
			for i := 0; i < targetRuns; i++ {
				created1 := e2e_utils.CreatePipelineRun(runClient, testContext, &uploadedPipeline.PipelineID, &version1.PipelineVersionID, experimentID, nil)
				runInfos = append(runInfos, e2e_utils.RunInfo{
					RunID:             created1.RunID,
					PipelineID:        uploadedPipeline.PipelineID,
					PipelineVersionID: version1.PipelineVersionID,
				})
			}
			// Then launch all version2 runs
			for i := 0; i < targetRuns; i++ {
				created2 := e2e_utils.CreatePipelineRun(runClient, testContext, &uploadedPipeline.PipelineID, &version2.PipelineVersionID, experimentID, nil)
				runInfos = append(runInfos, e2e_utils.RunInfo{
					RunID:             created2.RunID,
					PipelineID:        uploadedPipeline.PipelineID,
					PipelineVersionID: version2.PipelineVersionID,
				})
			}

			// Both versions should have their own limits, so all runs from different versions should be active
			versionLimitMap := map[string]int32{
				version1.PipelineVersionID: limit,
				version2.PipelineVersionID: limit,
			}
			e2e_utils.ValidateParallelismAcrossRuns(runClient, runInfos, versionLimitMap, maxPipelineWaitTime)
		})

		It("Test 3: > MaxParallelism runs, mix of single pipeline with different versions + same version - only MaxParallelism runs off the same version should be allowed but all runs from different versions should be allowed", func() {
			pipelineFilePath := filepath.Join(testutil.GetPipelineFilesDir(), pipelineDir, pipelineFile)
			limit, ok := e2e_utils.MaxActiveRuns(pipelineFilePath)
			Expect(ok).To(BeTrue(), "Pipeline should have max_active_runs configured")

			uploadedPipeline, uploadErr := testutil.UploadPipeline(pipelineUploadClient, pipelineFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)
			Expect(uploadErr).To(BeNil(), "Failed to upload pipeline")
			testContext.Pipeline.CreatedPipelines = append(testContext.Pipeline.CreatedPipelines, uploadedPipeline)
			version1 := testutil.GetLatestPipelineVersion(pipelineClient, &uploadedPipeline.PipelineID)

			// Upload a second version of the same pipeline
			uploadParams := upload_params.NewUploadPipelineVersionParams()
			uploadParams.Pipelineid = &uploadedPipeline.PipelineID
			version2, uploadErr2 := pipelineUploadClient.UploadPipelineVersion(pipelineFilePath, uploadParams)
			Expect(uploadErr2).To(BeNil(), "Failed to upload second pipeline version")

			// Test a MIX scenario: Launch runs to test dynamic mixing
			runInfos := make([]e2e_utils.RunInfo, 0)

			// Step 1: Launch runs from version1 (same version) to exceed the limit
			version1Runs := int(limit) + 2
			for i := 0; i < version1Runs; i++ {
				created1 := e2e_utils.CreatePipelineRun(runClient, testContext, &uploadedPipeline.PipelineID, &version1.PipelineVersionID, experimentID, nil)
				runInfos = append(runInfos, e2e_utils.RunInfo{
					RunID:             created1.RunID,
					PipelineID:        uploadedPipeline.PipelineID,
					PipelineVersionID: version1.PipelineVersionID,
				})
			}

			// Step 2: Launch runs from version2 (different version) - should be allowed independently
			version2Runs := int(limit) + 2
			for i := 0; i < version2Runs; i++ {
				created2 := e2e_utils.CreatePipelineRun(runClient, testContext, &uploadedPipeline.PipelineID, &version2.PipelineVersionID, experimentID, nil)
				runInfos = append(runInfos, e2e_utils.RunInfo{
					RunID:             created2.RunID,
					PipelineID:        uploadedPipeline.PipelineID,
					PipelineVersionID: version2.PipelineVersionID,
				})
			}

			// Step 3: Launch more runs from version1 (same version again) - should still be limited
			additionalVersion1Runs := int(limit) + 1
			for i := 0; i < additionalVersion1Runs; i++ {
				created1 := e2e_utils.CreatePipelineRun(runClient, testContext, &uploadedPipeline.PipelineID, &version1.PipelineVersionID, experimentID, nil)
				runInfos = append(runInfos, e2e_utils.RunInfo{
					RunID:             created1.RunID,
					PipelineID:        uploadedPipeline.PipelineID,
					PipelineVersionID: version1.PipelineVersionID,
				})
			}

			// Verify: The mix of "different versions + same version" doesn't cause interference
			versionLimitMap := map[string]int32{
				version1.PipelineVersionID: limit,
				version2.PipelineVersionID: limit,
			}
			e2e_utils.ValidateParallelismAcrossRuns(runClient, runInfos, versionLimitMap, maxPipelineWaitTime)
		})

		It("Test 4: > MaxParallelism runs different pipelines - all runs should be allowed", func() {
			pipelineFilePath := filepath.Join(testutil.GetPipelineFilesDir(), pipelineDir, pipelineFile)
			limit, ok := e2e_utils.MaxActiveRuns(pipelineFilePath)
			Expect(ok).To(BeTrue(), "Pipeline should have max_active_runs configured")

			// Upload first pipeline
			uploadedPipeline1, uploadErr1 := testutil.UploadPipeline(pipelineUploadClient, pipelineFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)
			Expect(uploadErr1).To(BeNil(), "Failed to upload first pipeline")
			testContext.Pipeline.CreatedPipelines = append(testContext.Pipeline.CreatedPipelines, uploadedPipeline1)
			version1 := testutil.GetLatestPipelineVersion(pipelineClient, &uploadedPipeline1.PipelineID)

			// Upload second pipeline (different pipeline, not just a version)
			pipelineName2 := testContext.Pipeline.PipelineGeneratedName + "-pipeline2"
			uploadedPipeline2, uploadErr2 := testutil.UploadPipeline(pipelineUploadClient, pipelineFilePath, &pipelineName2, nil)
			Expect(uploadErr2).To(BeNil(), "Failed to upload second pipeline")
			testContext.Pipeline.CreatedPipelines = append(testContext.Pipeline.CreatedPipelines, uploadedPipeline2)
			version2 := testutil.GetLatestPipelineVersion(pipelineClient, &uploadedPipeline2.PipelineID)

			// Launch (limit + 2) runs for each pipeline
			targetRuns := int(limit) + 2
			runInfos := make([]e2e_utils.RunInfo, 0, targetRuns*2)
			for i := 0; i < targetRuns; i++ {
				created1 := e2e_utils.CreatePipelineRun(runClient, testContext, &uploadedPipeline1.PipelineID, &version1.PipelineVersionID, experimentID, nil)
				runInfos = append(runInfos, e2e_utils.RunInfo{
					RunID:             created1.RunID,
					PipelineID:        uploadedPipeline1.PipelineID,
					PipelineVersionID: version1.PipelineVersionID,
				})
				created2 := e2e_utils.CreatePipelineRun(runClient, testContext, &uploadedPipeline2.PipelineID, &version2.PipelineVersionID, experimentID, nil)
				runInfos = append(runInfos, e2e_utils.RunInfo{
					RunID:             created2.RunID,
					PipelineID:        uploadedPipeline2.PipelineID,
					PipelineVersionID: version2.PipelineVersionID,
				})
			}

			// Each pipeline version should have its own limit, so runs from different pipelines should not interfere
			versionLimitMap := map[string]int32{
				version1.PipelineVersionID: limit,
				version2.PipelineVersionID: limit,
			}
			e2e_utils.ValidateParallelismAcrossRuns(runClient, runInfos, versionLimitMap, maxPipelineWaitTime)
		})
	})

	Context("Recurring run parallelism tests >", Label(E2eEssential), func() {
		const (
			pipelineDir              = "valid"
			pipelineFile             = "essential/pipeline_with_max_active_runs.yaml"
			recurringIntervalSeconds = int64(30)
		)

		removeRecurringRunID := func(id string) {
			for idx, storedID := range testContext.RecurringRun.CreatedRecurringRunIds {
				if storedID == id {
					testContext.RecurringRun.CreatedRecurringRunIds = append(
						testContext.RecurringRun.CreatedRecurringRunIds[:idx],
						testContext.RecurringRun.CreatedRecurringRunIds[idx+1:]...,
					)
					break
				}
			}
		}

		collectRunInfos := func(recurringRunID string, fallbackPipelineID string, fallbackPipelineVersionID string, targetRuns int) []e2e_utils.RunInfo {
			runInfos := make([]e2e_utils.RunInfo, 0, targetRuns)
			seen := make(map[string]struct{})

			Eventually(func() bool {
				runs, _, _, err := runClient.List(&run_params.RunServiceListRunsParams{
					ExperimentID: experimentID,
				})
				Expect(err).To(BeNil(), "Failed to list runs for recurring run %s", recurringRunID)

				for _, run := range runs {
					if run.RecurringRunID == "" || run.RecurringRunID != recurringRunID {
						continue
					}
					if _, exists := seen[run.RunID]; exists {
						continue
					}

					pipelineID := fallbackPipelineID
					if run.PipelineVersionReference != nil && run.PipelineVersionReference.PipelineID != "" {
						pipelineID = run.PipelineVersionReference.PipelineID
					}

					pipelineVersionID := fallbackPipelineVersionID
					if run.PipelineVersionReference != nil && run.PipelineVersionReference.PipelineVersionID != "" {
						pipelineVersionID = run.PipelineVersionReference.PipelineVersionID
					}

					seen[run.RunID] = struct{}{}
					runInfos = append(runInfos, e2e_utils.RunInfo{
						RunID:             run.RunID,
						PipelineID:        pipelineID,
						PipelineVersionID: pipelineVersionID,
					})
					testContext.PipelineRun.CreatedRunIds = append(testContext.PipelineRun.CreatedRunIds, run.RunID)
				}

				return len(runInfos) >= targetRuns
			}, 6*time.Minute, 5*time.Second).Should(BeTrue(), "Expected recurring run %s to create at least %d runs", recurringRunID, targetRuns)

			if len(runInfos) > targetRuns {
				runInfos = runInfos[:targetRuns]
			}

			return runInfos
		}

		It("Recurring run referencing pipeline ID only respects max_active_runs", func() {
			pipelineFilePath := filepath.Join(testutil.GetPipelineFilesDir(), pipelineDir, pipelineFile)
			limit, err := e2e_utils.MaxActiveRuns(pipelineFilePath)
			Expect(err).NotTo(HaveOccurred(), "Pipeline should have max_active_runs configured")
			Expect(limit).To(BeNumerically(">", 0), "max_active_runs should be greater than 0")

			uploadedPipeline, uploadErr := testutil.UploadPipeline(pipelineUploadClient, pipelineFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)
			Expect(uploadErr).To(BeNil(), "Failed to upload pipeline")
			testContext.Pipeline.CreatedPipelines = append(testContext.Pipeline.CreatedPipelines, uploadedPipeline)
			latestVersion := testutil.GetLatestPipelineVersion(pipelineClient, &uploadedPipeline.PipelineID)
			Expect(latestVersion.PipelineVersionID).NotTo(BeEmpty(), "Expected latest pipeline version to have an ID")

			effectiveLimit := int64(limit)
			if effectiveLimit < 1 {
				effectiveLimit = 1
			} else if effectiveLimit > 10 {
				effectiveLimit = 10
			}

			createParams := &recurring_run_params.RecurringRunServiceCreateRecurringRunParams{
				RecurringRun: &recurring_run_model.V2beta1RecurringRun{
					DisplayName:  fmt.Sprintf("recurring-id-%s", randomName),
					ExperimentID: *experimentID,
					PipelineVersionReference: &recurring_run_model.V2beta1PipelineVersionReference{
						PipelineID: uploadedPipeline.PipelineID,
					},
					MaxConcurrency: effectiveLimit,
					Mode:           recurring_run_model.RecurringRunModeENABLE.Pointer(),
					Trigger: &recurring_run_model.V2beta1Trigger{
						PeriodicSchedule: &recurring_run_model.V2beta1PeriodicSchedule{
							IntervalSecond: 5,
						},
					},
				},
			}
			recurringRun, err := recurringRunClient.Create(createParams)
			Expect(err).To(BeNil(), "Failed to create recurring run referencing pipeline ID")
			Expect(recurringRun).NotTo(BeNil(), "Recurring run response should not be nil")
			testContext.RecurringRun.CreatedRecurringRunIds = append(testContext.RecurringRun.CreatedRecurringRunIds, recurringRun.RecurringRunID)

			targetRuns := int(limit)
			runInfos := collectRunInfos(recurringRun.RecurringRunID, uploadedPipeline.PipelineID, latestVersion.PipelineVersionID, targetRuns)

			versionLimitMap := make(map[string]int32)
			for _, info := range runInfos {
				versionLimitMap[info.PipelineVersionID] = limit
			}

			e2e_utils.ValidateParallelismAcrossRuns(runClient, runInfos, versionLimitMap, maxPipelineWaitTime)

			testutil.DeleteRecurringRun(recurringRunClient, recurringRun.RecurringRunID)
			removeRecurringRunID(recurringRun.RecurringRunID)
		})

		It("Recurring run with embedded workflow spec respects max_active_runs", func() {
			pipelineFilePath := filepath.Join(testutil.GetPipelineFilesDir(), pipelineDir, pipelineFile)
			limit, ok := e2e_utils.MaxActiveRuns(pipelineFilePath)
			Expect(ok).To(BeTrue(), "Pipeline should have max_active_runs configured")
			Expect(limit).To(BeNumerically(">", 0), "max_active_runs should be greater than 0")

			specTemplate := testutil.ParseFileToSpecs(pipelineFilePath, false, nil)
			Expect(specTemplate).NotTo(BeNil(), "Failed to parse pipeline spec template")
			pipelineSpec := &structpb.Struct{}
			specBytes, err := protojson.Marshal(specTemplate.PipelineSpec())
			Expect(err).NotTo(HaveOccurred(), "Failed to marshal pipeline spec")
			Expect(protojson.Unmarshal(specBytes, pipelineSpec)).To(Succeed(), "Failed to unmarshal pipeline spec")

			effectiveLimit := int64(limit)
			if effectiveLimit < 1 {
				effectiveLimit = 1
			} else if effectiveLimit > 10 {
				effectiveLimit = 10
			}

			createParams := &recurring_run_params.RecurringRunServiceCreateRecurringRunParams{
				RecurringRun: &recurring_run_model.V2beta1RecurringRun{
					DisplayName:    fmt.Sprintf("recurring-spec-%s", randomName),
					ExperimentID:   *experimentID,
					PipelineSpec:   pipelineSpec,
					MaxConcurrency: effectiveLimit,
					Mode:           recurring_run_model.RecurringRunModeENABLE.Pointer(),
					Trigger: &recurring_run_model.V2beta1Trigger{
						PeriodicSchedule: &recurring_run_model.V2beta1PeriodicSchedule{
							IntervalSecond: 5,
						},
					},
				},
			}
			recurringRun, err := recurringRunClient.Create(createParams)
			Expect(err).To(BeNil(), "Failed to create recurring run with embedded workflow spec")
			Expect(recurringRun).NotTo(BeNil(), "Recurring run response should not be nil")
			testContext.RecurringRun.CreatedRecurringRunIds = append(testContext.RecurringRun.CreatedRecurringRunIds, recurringRun.RecurringRunID)

			targetRuns := int(limit)
			runInfos := collectRunInfos(recurringRun.RecurringRunID, "", "", targetRuns)

			versionLimitMap := make(map[string]int32)
			for _, info := range runInfos {
				versionLimitMap[info.PipelineVersionID] = limit
			}

			e2e_utils.ValidateParallelismAcrossRuns(runClient, runInfos, versionLimitMap, maxPipelineWaitTime)

			testutil.DeleteRecurringRun(recurringRunClient, recurringRun.RecurringRunID)
			removeRecurringRunID(recurringRun.RecurringRunID)
		})
	})
})

func validatePipelineRunSuccess(pipelineFile string, pipelineDir string, testContext *apitests.TestContext) {
	testutil.CheckIfSkipping(pipelineFile)
	pipelineFilePath := filepath.Join(testutil.GetPipelineFilesDir(), pipelineDir, pipelineFile)
	logger.Log("Uploading pipeline file %s", pipelineFile)
	uploadedPipeline, uploadErr := testutil.UploadPipeline(pipelineUploadClient, pipelineFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)
	Expect(uploadErr).To(BeNil(), "Failed to upload pipeline %s", pipelineFile)
	testContext.Pipeline.CreatedPipelines = append(testContext.Pipeline.CreatedPipelines, uploadedPipeline)
	logger.Log("Upload of pipeline file '%s' successful", pipelineFile)
	uploadedPipelineVersion := testutil.GetLatestPipelineVersion(pipelineClient, &uploadedPipeline.PipelineID)
	pipelineRuntimeInputs := testutil.GetPipelineRunTimeInputs(pipelineFilePath)
	createdRunID := e2e_utils.CreatePipelineRunAndWaitForItToFinish(runClient, testContext, uploadedPipeline.PipelineID, uploadedPipeline.DisplayName, &uploadedPipelineVersion.PipelineVersionID, experimentID, pipelineRuntimeInputs, maxPipelineWaitTime)
	logger.Log("Deserializing expected compiled workflow file '%s' for the pipeline", pipelineFile)
	if strings.Contains(pipelineFile, "/") {
		pipelineFile = strings.Split(pipelineFile, "/")[1]
	}
	compiledWorkflow := workflowutils.UnmarshallWorkflowYAML(filepath.Join(testutil.GetCompiledWorkflowsFilesDir(), pipelineFile))
	e2e_utils.ValidateComponentStatuses(runClient, k8Client, testContext, createdRunID, compiledWorkflow)
	if limit, ok := e2e_utils.MaxActiveRuns(pipelineFilePath); ok {
		e2e_utils.ValidateWorkflowParallelismAcrossRuns(runClient, testContext, uploadedPipeline.PipelineID, uploadedPipelineVersion.PipelineVersionID, experimentID, limit, maxPipelineWaitTime)
	}

}
