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
	"github.com/go-openapi/strfmt"
	upload_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_client/pipeline_upload_service"
	model "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_model"
	matcher "github.com/kubeflow/pipelines/backend/test/v2/api/matcher"
	utils "github.com/kubeflow/pipelines/backend/test/v2/api/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"log"
	"time"
)

const (
	hello_world_pipeline_name = "hello-world.yaml"
	pipeline_with_args_name   = "arguments-parameters.yaml"
)

var pipelineDir = "positive"
var expectedPipeline *model.V2beta1Pipeline
var createdPipelines []*model.V2beta1Pipeline
var testStartTime strfmt.DateTime

var _ = BeforeEach(func() {
	log.Printf("################### Setup before each test #####################")
	testStartTime, _ = strfmt.ParseDateTime(time.Now().Format(time.DateTime))
	createdPipelines = []*model.V2beta1Pipeline{}
	expectedPipeline = new(model.V2beta1Pipeline)
	expectedPipeline.CreatedAt = testStartTime
	//expectedPipeline.Namespace = *namespace
})

var _ = AfterEach(func() {
	// Delete pipelines created during the test
	log.Printf("################### Cleanup after each test #####################")
	log.Printf("Deleting %d pipeline(s)", len(createdPipelines))
	for _, pipeline := range createdPipelines {
		utils.DeletePipeline(pipelineClient, pipeline.PipelineID)
	}
})

var _ = Describe("Verify Pipeline Upload >", func() {

	/* Positive Scenarios of uploading a "yaml" pipeline file */
	Context("Upload a pipeline and verify pipeline metadata after upload >", func() {
		It(fmt.Sprintf("Upload %s pipeline", hello_world_pipeline_name), func() {
			createPipelineAndVerify(hello_world_pipeline_name)
		})

		It(fmt.Sprintf("Upload %s pipeline", pipeline_with_args_name), func() {
			createPipelineAndVerify(pipeline_with_args_name)
		})
	})
})

var _ = Describe("Verify Pipeline Upload Failure >", func() {
	errorScenario := []struct {
		pipeline_name          string
		expected_error_message string
	}{
		{"empty_file.yaml", "Failed to upload pipeline"},
		{"empty_zip.zip", "Failed to upload pipeline"},
		{"invalid_yaml.yaml", "Failed to upload pipeline"},
		{"no_name.yaml", "Failed to upload pipeline"},
		{"wrong_format.png", "Failed to upload pipeline"},
	}

	/* Negative scenarios of uploading a pipeline  */
	Context("Upload a failing pipeline and verify the error in the response >", func() {
		It("Upload a pipeline twice and verify that it should fail the second time", func() {
			createPipelineAndVerify(pipeline_with_args_name)
			uploadPipelineAndVerifyFailure(pipeline_with_args_name, "Failed to upload pipeline")
		})

		for _, scenario := range errorScenario {
			It("Upload a pipeline and verify the failure", func() {
				pipelineDir = "negative"
				uploadPipelineAndVerifyFailure(scenario.pipeline_name, scenario.expected_error_message)
			})
		}
	})
})

func createPipeline(pipelineName string) (*model.V2beta1Pipeline, error) {
	pipelineFile := "../resources/pipelines/" + pipelineDir + "/" + pipelineName
	log.Printf("Creating pipeline with name=%s, from file %s", pipelineName, pipelineFile)
	return pipelineUploadClient.UploadFile(pipelineFile, upload_params.NewUploadPipelineParams())
}

func createPipelineAndVerify(pipelineName string) {
	createdPipeline, err := createPipeline(pipelineName)
	Expect(err).NotTo(HaveOccurred())
	expectedPipeline.DisplayName = pipelineName
	createdPipelines = append(createdPipelines, createdPipeline)
	matcher.MatchPipelines(createdPipeline, expectedPipeline)
}

func uploadPipelineAndVerifyFailure(pipelineName string, errorMessage string) {
	_, err := createPipeline(pipelineName)
	log.Printf("Verifying error in the response")
	Expect(err).To(HaveOccurred())
	Expect(err.Error()).To(ContainSubstring(errorMessage))
}
