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
	"strconv"
	"time"
)

const (
	helloWorldPipelineFileName = "hello-world.yaml"
	pipelineWithArgsFileName   = "arguments-parameters.yaml"
	zipPipelineFileName        = "arguments.pipeline.zip"
	tarPipelineFileName        = "tar-pipeline.tar.gz"
)

var pipelineDir string
var pipelineGeneratedName string
var expectedPipeline *model.V2beta1Pipeline
var createdPipelines []*model.V2beta1Pipeline
var testStartTime strfmt.DateTime
var uploadParams *upload_params.UploadPipelineParams

var _ = BeforeEach(func() {
	log.Printf("################### Setup before each test #####################")
	testStartTime, _ = strfmt.ParseDateTime(time.Now().Format(time.DateTime))
	createdPipelines = []*model.V2beta1Pipeline{}
	expectedPipeline = new(model.V2beta1Pipeline)
	expectedPipeline.CreatedAt = testStartTime
	pipelineGeneratedName = "apitest-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	uploadParams = upload_params.NewUploadPipelineParams()
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

var _ = Describe("Verify Pipeline Upload >", Label("Positive"), func() {
	pipelineDir = "positive"
	pipelineFiles := []string{helloWorldPipelineFileName, pipelineWithArgsFileName, zipPipelineFileName, tarPipelineFileName}

	/* Positive Scenarios of uploading a pipeline file */
	Context("Upload a pipeline and verify pipeline metadata after upload >", func() {
		for _, pipelineFile := range pipelineFiles {
			It(fmt.Sprintf("Upload %s pipeline", pipelineFile), Label("Smoke"), func() {
				createPipelineAndVerify(pipelineFile, &pipelineGeneratedName)
			})
		}

		It(fmt.Sprintf("Upload %s pipeline with custom name and description", helloWorldPipelineFileName), func() {
			description := "Some pipeline description"
			namespace := *namespace
			uploadParams.SetDescription(&description)
			uploadParams.SetNamespace(&namespace)
			expectedPipeline.Description = description
			createPipelineAndVerify(helloWorldPipelineFileName, &pipelineGeneratedName)
		})
	})
})

var _ = Describe("Verify Pipeline Upload Failure >", Label("Negative"), func() {
	errorScenario := []struct {
		pipeline_name          string
		expected_error_message string
	}{
		{"empty_file.yaml", "Failed to upload pipeline"},
		{"empty_zip.zip", "Failed to upload pipeline"},
		{"invalid_yaml.yaml", "Failed to upload pipeline"},
		{"no_name.yaml", "Failed to upload pipeline"},
		{"wrong_format.png", "Failed to upload pipeline"},
		{"invalid-tar.tar.gz", "Failed to upload pipeline"},
		{"invalid-zip.zip", "Failed to upload pipeline"},
	}

	/* Negative scenarios of uploading a pipeline  */
	Context("Upload a failing pipeline and verify the error in the response >", func() {
		It("Upload a pipeline twice and verify that it should fail the second time", func() {
			pipelineDir = "positive"
			createdPipeline := createPipelineAndVerify(helloWorldPipelineFileName, &pipelineGeneratedName)
			uploadPipelineAndVerifyFailure(helloWorldPipelineFileName, &(createdPipeline.DisplayName), "Failed to upload pipeline")
		})

		for _, scenario := range errorScenario {
			It("Upload a pipeline and verify the failure", func() {
				pipelineDir = "negative"
				uploadPipelineAndVerifyFailure(scenario.pipeline_name, &pipelineGeneratedName, scenario.expected_error_message)
			})
		}
	})
})

func createPipeline(pipelineFileName string, pipelineName *string) (*model.V2beta1Pipeline, error) {
	pipelineFile := fmt.Sprintf("../resources/pipelines/%s/%s", pipelineDir, pipelineFileName)
	uploadParams.SetName(pipelineName)
	expectedPipeline.DisplayName = *pipelineName
	log.Printf("Creating pipeline with name=%s, from file %s", *pipelineName, pipelineFile)
	return pipelineUploadClient.UploadFile(pipelineFile, uploadParams)
}

func createPipelineAndVerify(pipelineFileName string, pipelineName *string) *model.V2beta1Pipeline {
	createdPipeline, err := createPipeline(pipelineFileName, pipelineName)
	Expect(err).NotTo(HaveOccurred())
	createdPipelines = append(createdPipelines, createdPipeline)
	matcher.MatchPipelines(createdPipeline, expectedPipeline)
	return createdPipeline
}

func uploadPipelineAndVerifyFailure(pipelineFileName string, pipelineName *string, errorMessage string) {
	_, err := createPipeline(pipelineFileName, pipelineName)
	log.Printf("Verifying error in the response")
	Expect(err).To(HaveOccurred())
	Expect(err.Error()).To(ContainSubstring(errorMessage))
}
