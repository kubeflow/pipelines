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
	"time"

	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/experiment_model"
	upload_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_client/pipeline_upload_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_model"
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
	. "github.com/onsi/gomega"
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
		testContext.Pipeline.PipelineGeneratedName = "e2e_test-" + randomName
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

		// Delete pipelines created during the test
		logger.Log("################### Global Cleanup after each test #####################")

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

	Context("Upload a pipeline file, run it and verify that pipeline run succeeds >", Label(E2eEssential), func() {
		var pipelineDir = "valid/essential"
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
				logger.Log("Deserializing expected compiled workflow file '%s' for the pipeline", pipelineFile)
				compiledWorkflow := workflowutils.UnmarshallWorkflowYAML(filepath.Join(testutil.GetCompiledWorkflowsFilesDir(), pipelineFile))
				e2e_utils.ValidateComponentStatuses(runClient, k8Client, testContext, createdRunID, compiledWorkflow)
			})
		}
	})

	// Few of the following pipelines randomly fail in Multi User Mode during CI run - which is why a FlakeAttempt is added, but we need to investigate, create ticket and fix it in the future
	Context("Upload a pipeline file, run it and verify that pipeline run succeeds >", FlakeAttempts(2), Label("Sample", E2eCritical), func() {
		var pipelineDir = "valid/critical"
		pipelineFiles := testutil.GetListOfFilesInADir(filepath.Join(testutil.GetPipelineFilesDir(), pipelineDir))
		for _, pipelineFile := range pipelineFiles {
			It(fmt.Sprintf("Upload %s pipeline", pipelineFile), FlakeAttempts(2), func() {
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
				compiledWorkflow := workflowutils.UnmarshallWorkflowYAML(filepath.Join(testutil.GetCompiledWorkflowsFilesDir(), pipelineFile))
				e2e_utils.ValidateComponentStatuses(runClient, k8Client, testContext, createdRunID, compiledWorkflow)

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
})
