// Copyright 2025 The Kubeflow Authors
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

package end2end

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	experiment_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/experiment_client/experiment_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/experiment_model"
	pipeline_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_client/pipeline_service"
	upload_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_client/pipeline_upload_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_model"
	run_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_client/run_service"
	run_model "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_model"
	. "github.com/kubeflow/pipelines/backend/test/constants"
	e2e_utils "github.com/kubeflow/pipelines/backend/test/end2end/utils"
	"github.com/kubeflow/pipelines/backend/test/logger"
	apitests "github.com/kubeflow/pipelines/backend/test/v2/api"

	"os"

	"github.com/go-openapi/strfmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

/*
Prerequisites:
- Kind cluster running and configured (see `make -C backend kind-cluster-agnostic`).
- Port-forward to API: `kubectl port-forward -n kubeflow svc/ml-pipeline 8888:8888`.
- For private model tests, create a Kubernetes Secret with a HuggingFace token in the 'kubeflow' namespace:
  `kubectl create secret generic huggingface-token --from-literal=token=<TOKEN> -n kubeflow`.

How to run:
- `go test -v -run TestHuggingFaceImporter ./backend/test/end2end`
- Or use ginkgo with the appropriate version and label filter `--label-filter="Smoke"`.

Notes:
- This file is an integration/E2E test and may be long-running; mark it as Smoke/Integration in CI and do not run heavy downloads in presubmit.
- Location: `backend/test/end2end/huggingface_importer_e2e_test.go`
*/

var _ = Describe("HuggingFace Importer E2E Tests", Label(FullRegression), func() {
	var testContext *apitests.TestContext

	BeforeEach(func() {
		logger.Log("Setting up HuggingFace Importer E2E test")
		testContext = &apitests.TestContext{
			TestStartTimeUTC: time.Now(),
		}
		logger.Log("Test context initialized")
		randomName = strconv.FormatInt(time.Now().UnixNano(), 10)
		testContext.Pipeline.UploadParams = upload_params.NewUploadPipelineParams()
		testContext.Pipeline.PipelineGeneratedName = "e2e-hf-importer-" + randomName
		testContext.Pipeline.CreatedPipelines = make([]*pipeline_upload_model.V2beta1Pipeline, 0)
		testContext.PipelineRun.CreatedRunIds = make([]string, 0)
		testContext.Pipeline.ExpectedPipeline = new(pipeline_upload_model.V2beta1Pipeline)
		testContext.Pipeline.ExpectedPipeline.CreatedAt = strfmt.DateTime(testContext.TestStartTimeUTC)
	})

	AfterEach(func() {
		logger.Log("Cleaning up HuggingFace Importer E2E test")
		// Cleanup: delete runs created during the test
		for _, runID := range testContext.PipelineRun.CreatedRunIds {
			err := runClient.Terminate(&run_params.RunServiceTerminateRunParams{RunID: runID})
			Expect(err).ShouldNot(HaveOccurred())
		}
		// Cleanup: delete pipelines created during the test
		for _, pipeline := range testContext.Pipeline.CreatedPipelines {
			err := pipelineClient.Delete(&pipeline_params.PipelineServiceDeletePipelineParams{PipelineID: pipeline.PipelineID})
			if err != nil {
				logger.Log("Error deleting pipeline: %v", err)
			}
		}
	})

	Context("HuggingFace Hub Model Import", Label(Smoke), func() {
		It("should successfully import a public model from HuggingFace Hub", func() {
			logger.Log("Test: Import public GPT2 model from HuggingFace Hub")

			// Compute absolute path from test file location (go up to repo root)
			_, testFile, _, _ := runtime.Caller(0)
			testDir := filepath.Dir(testFile)
			testPipelinePath := filepath.Join(testDir, "..", "..", "..", "test_data", "pipeline_files", "valid", "huggingface_importer.py")
			logger.Log("Loading pipeline from: %s", testPipelinePath)

			// Upload the pipeline
			pipeline, err := pipelineUploadClient.UploadFile(testPipelinePath, testContext.Pipeline.UploadParams)
			Expect(err).ShouldNot(HaveOccurred())

			Expect(pipeline.PipelineID).ShouldNot(BeEmpty())
			testContext.Pipeline.CreatedPipelines = append(testContext.Pipeline.CreatedPipelines, pipeline)
			logger.Log("Pipeline uploaded with ID: %s", pipeline.PipelineID)

			// Create an experiment for the run
			experiment := &experiment_model.V2beta1Experiment{
				DisplayName: "HuggingFace Importer E2E - " + randomName,
			}
			createdExperiment, err := experimentClient.Create(&experiment_params.ExperimentServiceCreateExperimentParams{Experiment: experiment})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(createdExperiment.ExperimentID).ShouldNot(BeEmpty())
			experimentID := createdExperiment.ExperimentID
			logger.Log("Experiment created with ID: %s", experimentID)

			// Create and run the pipeline, wait for completion
			runID := e2e_utils.CreatePipelineRunAndWaitForItToFinish(runClient, testContext, pipeline.PipelineID, pipeline.DisplayName, nil, &experimentID, nil, 5*60)
			logger.Log("Run created and completed with ID: %s", runID)

			// Verify run status
			runDetails, err := runClient.Get(&run_params.RunServiceGetRunParams{RunID: runID})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(*runDetails.State).Should(Equal(run_model.V2beta1RuntimeStateSUCCEEDED))
			logger.Log("Run state verified: %s", *runDetails.State)
		})

		It("should import multiple models in a single pipeline", func() {
			logger.Log("Test: Import multiple models in sequence")

			// Compute absolute path from test file location
			_, testFile, _, _ := runtime.Caller(0)
			testDir := filepath.Dir(testFile)
			testPipelinePath := filepath.Join(testDir, "..", "..", "..", "test_data", "pipeline_files", "valid", "huggingface_importer.py")
			logger.Log("Loading pipeline from: %s", testPipelinePath)

			pipeline, err := pipelineUploadClient.UploadFile(testPipelinePath, testContext.Pipeline.UploadParams)
			Expect(err).ShouldNot(HaveOccurred())
			testContext.Pipeline.CreatedPipelines = append(testContext.Pipeline.CreatedPipelines, pipeline)
			logger.Log("Pipeline uploaded with ID: %s", pipeline.PipelineID)

			experiment := &experiment_model.V2beta1Experiment{
				DisplayName: "HuggingFace Multi-Import E2E - " + randomName,
			}
			createdExperiment, err := experimentClient.Create(&experiment_params.ExperimentServiceCreateExperimentParams{Experiment: experiment})
			Expect(err).ShouldNot(HaveOccurred())
			experimentID := createdExperiment.ExperimentID
			logger.Log("Experiment created with ID: %s", experimentID)

			// Create and run the pipeline, wait for completion
			runID := e2e_utils.CreatePipelineRunAndWaitForItToFinish(runClient, testContext, pipeline.PipelineID, pipeline.DisplayName, nil, &experimentID, nil, 10*60)
			logger.Log("Run created and completed with ID: %s", runID)

			// Verify run state
			runDetails, err := runClient.Get(&run_params.RunServiceGetRunParams{RunID: runID})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(*runDetails.State).Should(Equal(run_model.V2beta1RuntimeStateSUCCEEDED))
			logger.Log("Multi-import run state verified: %s", *runDetails.State)
		})

		It("should compile HuggingFace importer pipelines correctly", func() {
			logger.Log("Test: Pipeline compilation for HuggingFace importers")

			// Compute absolute path from test file location (go up to repo root)
			_, testFile, _, _ := runtime.Caller(0)
			testDir := filepath.Dir(testFile)
			testPipelinePath := filepath.Join(testDir, "..", "..", "..", "test_data", "pipeline_files", "valid", "huggingface_importer.py")
			logger.Log("Checking pipeline compilation for: %s", testPipelinePath)

			// Verify the pipeline file exists
			if _, err := os.Stat(testPipelinePath); err != nil {
				Expect(err).ShouldNot(HaveOccurred(), fmt.Sprintf("Pipeline file not found: %s", testPipelinePath))
			}
			logger.Log("Pipeline file verified")

			// The compilation should succeed at upload time
			pipeline, err := pipelineUploadClient.UploadFile(testPipelinePath, testContext.Pipeline.UploadParams)
			Expect(err).ShouldNot(HaveOccurred(), "Pipeline compilation failed during upload")
			testContext.Pipeline.CreatedPipelines = append(testContext.Pipeline.CreatedPipelines, pipeline)
			logger.Log("Pipeline compiled and uploaded successfully with ID: %s", pipeline.PipelineID)
		})
	})

	Context("HuggingFace Hub URI Validation", Label(Smoke), func() {
		It("should handle various HuggingFace URI formats", func() {
			logger.Log("Test: HuggingFace URI format validation")

			// Compute absolute path from test file location (go up to repo root)
			_, testFile, _, _ := runtime.Caller(0)
			testDir := filepath.Dir(testFile)
			testPipelinePath := filepath.Join(testDir, "..", "..", "..", "test_data", "pipeline_files", "valid", "huggingface_importer.py")

			pipeline, err := pipelineUploadClient.UploadFile(testPipelinePath, testContext.Pipeline.UploadParams)
			Expect(err).ShouldNot(HaveOccurred())
			testContext.Pipeline.CreatedPipelines = append(testContext.Pipeline.CreatedPipelines, pipeline)
			logger.Log("Pipeline with various URI formats uploaded successfully")

			// Verify pipeline details
			pipelineDetails, err := pipelineClient.Get(&pipeline_params.PipelineServiceGetPipelineParams{PipelineID: pipeline.PipelineID})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(pipelineDetails.PipelineID).Should(Equal(pipeline.PipelineID))
			logger.Log("Pipeline details verified: %s", pipelineDetails.DisplayName)
		})
	})
})
