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
	"encoding/json"
	"fmt"
	"github.com/go-openapi/strfmt"
	upload_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_client/pipeline_upload_service"
	model "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_model"
	. "github.com/kubeflow/pipelines/backend/test/v2/api/constants"
	matcher "github.com/kubeflow/pipelines/backend/test/v2/api/matcher"
	utils "github.com/kubeflow/pipelines/backend/test/v2/api/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"strconv"
	"time"
)

// ####################################################################################################################################################################
// ################################################################### CLASS VARIABLES ################################################################################
// ####################################################################################################################################################################

const (
	helloWorldPipelineFileName = "hello-world.yaml"
	pipelineWithArgsFileName   = "arguments-parameters.yaml"
	zipPipelineFileName        = "arguments.pipeline.zip"
	tarPipelineFileName        = "tar-pipeline.tar.gz"
)

var pipelineGeneratedName string
var expectedPipeline *model.V2beta1Pipeline
var createdPipelines []*model.V2beta1Pipeline
var testStartTime strfmt.DateTime
var uploadParams *upload_params.UploadPipelineParams

// ####################################################################################################################################################################
// ################################################################### SET AND TEARDOWN ################################################################################
// ####################################################################################################################################################################

var _ = BeforeEach(func() {
	GinkgoWriter.Printf("################### Setup before each test #####################")
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
	GinkgoWriter.Printf("################### Cleanup after each test #####################")
	GinkgoWriter.Printf("Deleting %d pipeline(s)", len(createdPipelines))
	for _, pipeline := range createdPipelines {
		utils.DeletePipeline(pipelineClient, pipeline.PipelineID)
	}
})

// ####################################################################################################################################################################
// ################################################################### TESTS ################################################################################
// ####################################################################################################################################################################

// ################################################################################################################
// ########################################################## POSITIVE TESTS ######################################
// ################################################################################################################

var _ = Describe("Verify Pipeline Upload >", Label("Positive", "PipelineUpload", S1.String()), func() {
	var pipelineDir = "positive"
	pipelineFiles := []string{helloWorldPipelineFileName, pipelineWithArgsFileName, zipPipelineFileName, tarPipelineFileName}

	/* Positive Scenarios of uploading a pipeline file */
	Context("Upload a pipeline and verify pipeline metadata after upload >", func() {
		for _, pipelineFile := range pipelineFiles {
			It(fmt.Sprintf("Upload %s pipeline", pipelineFile), Label(SMOKE.String()), func() {
				uploadPipelineAndVerify(pipelineDir, pipelineFile, &pipelineGeneratedName)
			})
		}

		It(fmt.Sprintf("Upload %s pipeline with custom name and description", helloWorldPipelineFileName), func() {
			description := "Some pipeline description"
			uploadParams.SetDescription(&description)
			expectedPipeline.Description = description
			uploadPipelineAndVerify(pipelineDir, helloWorldPipelineFileName, &pipelineGeneratedName)
		})
	})
})

var _ = Describe("Verify Pipeline Upload Version >", Label("Positive", "PipelineUpload", S1.String()), func() {
	var pipelineDir = "positive"

	/* Positive Scenarios of uploading a pipeline file */
	Context("Upload a pipeline and upload the same pipeline to change version >", func() {
		const pipelineFile = helloWorldPipelineFileName
		It(fmt.Sprintf("Upload %s pipeline file and upload a new verison with the same file", pipelineFile), Label(SMOKE.String()), func() {
			uploadPipelineAndChangePipelineVersion(pipelineDir, pipelineFile, pipelineFile)
		})
		It(fmt.Sprintf("Upload %s pipeline file and upload a new verison with the different file %s", pipelineFile, pipelineWithArgsFileName), func() {
			uploadPipelineAndChangePipelineVersion(pipelineDir, pipelineFile, pipelineWithArgsFileName)
		})

	})
})

// ################################################################################################################
// ########################################################## NEGATIVE TESTS ######################################
// ################################################################################################################

var _ = Describe("Verify Pipeline Upload Failure >", Label("Negative", "PipelineUpload"), func() {
	errorScenario := []struct {
		pipelineFileName     string
		expectedErrorMessage string
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
			var pipelineDir = "positive"
			createdPipeline := uploadPipelineAndVerify(pipelineDir, helloWorldPipelineFileName, &pipelineGeneratedName)
			uploadPipelineAndVerifyFailure(pipelineDir, helloWorldPipelineFileName, &(createdPipeline.DisplayName), "Failed to upload pipeline")
		})

		for _, scenario := range errorScenario {
			It(fmt.Sprintf("Upload a %s pipeline and verify the failure", scenario.pipelineFileName), func() {
				var pipelineDir = "negative"
				uploadPipelineAndVerifyFailure(pipelineDir, scenario.pipelineFileName, &pipelineGeneratedName, scenario.expectedErrorMessage)
			})
		}
	})
})

var _ = Describe("Verify Pipeline Upload Version Failure >", Label("Negative", "PipelineUpload"), func() {
	var pipelineDir = "positive"

	/* Positive Scenarios of uploading a pipeline file */
	Context("Upload a pipeline and try changing the version with a different metric >", func() {
		const pipelineFileName = helloWorldPipelineFileName
		It(fmt.Sprintf("Change %s pipeline's name to be same as original version", pipelineFileName), func() {
			createdPipeline := uploadPipelineAndVerify(pipelineDir, pipelineFileName, &pipelineGeneratedName)

			parameters := upload_params.NewUploadPipelineVersionParams()
			parameters.Pipelineid = &(createdPipeline.PipelineID)
			parameters.SetName(&createdPipeline.DisplayName)
			uploadPipelineVersionAndVerifyFailure(pipelineDir, pipelineFileName, parameters, "Failed to upload pipeline version")
		})
		It(fmt.Sprintf("Change %s pipeline's id with fake pipeline id", pipelineFileName), func() {
			uploadPipelineAndVerify(pipelineDir, pipelineFileName, &pipelineGeneratedName)

			parameters := upload_params.NewUploadPipelineVersionParams()
			fakePipelineId := "12345"
			parameters.Pipelineid = &fakePipelineId
			uploadPipelineVersionAndVerifyFailure(pipelineDir, pipelineFileName, parameters, "Failed to upload pipeline version")
		})
	})
})

// ####################################################################################################################################################################
// ################################################################### UTILITY METHODS ################################################################################
// ####################################################################################################################################################################

/*
A common method that creates a pipeline and then creates a new pipeline version
@param pipelineFileNameForCreation - pipeline file name for initial pipeline upload
@param pipelineFileNameWhenChangingVersion - the pipeline file name that you wish to upload when creating a new version
*/
func uploadPipelineAndChangePipelineVersion(pipelineDir string, pipelineFileNameForCreation string, pipelineFileNameWhenChangingVersion string) {
	createdPipeline := uploadPipelineAndVerify(pipelineDir, pipelineFileNameForCreation, &pipelineGeneratedName)

	// Construct a payload to create new pipeline version
	parameters := upload_params.NewUploadPipelineVersionParams()
	expectedPipelineVersion := new(model.V2beta1PipelineVersion)
	descriptionNew := "Some changed pipeline description"
	pipelineNameNew := createdPipeline.DisplayName + "-1"
	parameters.Pipelineid = &(createdPipeline.PipelineID)
	parameters.SetDescription(&descriptionNew)
	parameters.SetName(&pipelineNameNew)

	// Construct expected Pipeline Spec from the uploaded file
	pipelineVersionFilePath := fmt.Sprintf("../resources/pipelines/%s/%s", pipelineDir, pipelineFileNameWhenChangingVersion)
	jsonSpecFromFile := utils.JsonFromYAML(pipelineVersionFilePath)

	// Construct expected pipeline version object for comparison
	expectedPipelineVersion.Description = descriptionNew
	json.Unmarshal(jsonSpecFromFile, &expectedPipelineVersion.PipelineSpec)
	expectedPipelineVersion.DisplayName = pipelineNameNew
	uploadPipelineVersionAndVerify(pipelineDir, pipelineFileNameWhenChangingVersion, parameters, expectedPipelineVersion)
}

func uploadPipeline(pipelineDir string, pipelineFileName string, pipelineName *string) (*model.V2beta1Pipeline, error) {
	pipelineFile := fmt.Sprintf("../resources/pipelines/%s/%s", pipelineDir, pipelineFileName)
	uploadParams.SetName(pipelineName)
	expectedPipeline.DisplayName = *pipelineName
	GinkgoWriter.Printf("Uploading pipeline with name=%s, from file %s", *pipelineName, pipelineFile)
	return pipelineUploadClient.UploadFile(pipelineFile, uploadParams)
}

func uploadPipelineAndVerify(pipelineDir string, pipelineFileName string, pipelineName *string) *model.V2beta1Pipeline {
	createdPipeline, err := uploadPipeline(pipelineDir, pipelineFileName, pipelineName)
	GinkgoWriter.Printf("Verifying that NO error was returned in the response to confirm that the pipeline was successfully uploaded")
	Expect(err).NotTo(HaveOccurred())
	createdPipelines = append(createdPipelines, createdPipeline)

	createdPipelineFromDB := utils.GetPipeline(pipelineClient, createdPipeline.PipelineID)
	Expect(createdPipelineFromDB).To(Equal(*createdPipeline))
	matcher.MatchPipelines(&createdPipelineFromDB, expectedPipeline)
	return createdPipeline
}

func uploadPipelineAndVerifyFailure(pipelineDir string, pipelineFileName string, pipelineName *string, errorMessage string) {
	_, err := uploadPipeline(pipelineDir, pipelineFileName, pipelineName)
	GinkgoWriter.Printf("Verifying error in the response")
	Expect(err).To(HaveOccurred())
	Expect(err.Error()).To(ContainSubstring(errorMessage))
}

func uploadPipelineVersion(pipelineDir string, pipelineFileName string, parameters *upload_params.UploadPipelineVersionParams) (*model.V2beta1PipelineVersion, error) {
	pipelineFile := fmt.Sprintf("../resources/pipelines/%s/%s", pipelineDir, pipelineFileName)
	GinkgoWriter.Printf("Uploading pipeline version for pipeline with id=%s, from file %s", *parameters.Pipelineid, pipelineFile)
	return pipelineUploadClient.UploadPipelineVersion(pipelineFile, parameters)
}

func uploadPipelineVersionAndVerify(pipelineDir string, pipelineFileName string, parameters *upload_params.UploadPipelineVersionParams, expectedPipelineVersion *model.V2beta1PipelineVersion) *model.V2beta1PipelineVersion {
	createdPipelineVersion, err := uploadPipelineVersion(pipelineDir, pipelineFileName, parameters)
	GinkgoWriter.Printf("Verifying that NO error was returned in the response to confirm that the pipeline was successfully uploaded")
	Expect(err).NotTo(HaveOccurred())
	matcher.MatchPipelineVersions(createdPipelineVersion, expectedPipelineVersion)
	return createdPipelineVersion
}

func uploadPipelineVersionAndVerifyFailure(pipelineDir string, pipelineFileName string, parameters *upload_params.UploadPipelineVersionParams, errorMessage string) {
	_, err := uploadPipelineVersion(pipelineDir, pipelineFileName, parameters)
	GinkgoWriter.Printf("Verifying error in the response")
	Expect(err).To(HaveOccurred())
	Expect(err.Error()).To(ContainSubstring(errorMessage))
}
