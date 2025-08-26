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
	"strconv"
	"time"

	"github.com/go-openapi/strfmt"
	upload_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_client/pipeline_upload_service"
	model "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_model"
	. "github.com/kubeflow/pipelines/backend/test/v2/api/constants"
	"github.com/kubeflow/pipelines/backend/test/v2/api/logger"
	matcher "github.com/kubeflow/pipelines/backend/test/v2/api/matcher"
	utils "github.com/kubeflow/pipelines/backend/test/v2/api/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// ####################################################################################################################################################################
// ################################################################### CLASS VARIABLES ################################################################################
// ####################################################################################################################################################################

const (
	helloWorldPipelineFileName = "hello-world.yaml"
	pipelineWithArgsFileName   = "arguments-parameters.yaml"
)

// ####################################################################################################################################################################
// ################################################################### SET AND TEARDOWN ################################################################################
// ####################################################################################################################################################################

var _ = BeforeEach(func() {
	logger.Log("################### Setup before each test #####################")
	testStartTime, _ := strfmt.ParseDateTime(time.Now().Format(time.DateTime))
	testContext.CreatedPipelines = []*model.V2beta1Pipeline{}
	testContext.ExpectedPipeline = new(model.V2beta1Pipeline)
	testContext.ExpectedPipeline.CreatedAt = testStartTime
	testContext.PipelineGeneratedName = "apitest-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	testContext.UploadParams = upload_params.NewUploadPipelineParams()
	testContext.ExpectedPipeline.Namespace = *namespace
})

// ####################################################################################################################################################################
// ################################################################### TESTS ################################################################################
// ####################################################################################################################################################################

// ################################################################################################################
// ########################################################## POSITIVE TESTS ######################################
// ################################################################################################################

var _ = Describe("Verify Pipeline Upload >", Label("Positive", "PipelineUpload", "ApiServerTests", FullRegression), func() {

	/* Critical Positive Scenarios of uploading a pipeline file */
	Context("Upload a valid critical pipeline file and verify pipeline metadata after upload >", Label(Smoke), func() {
		var pipelineDir = "valid/critical"
		criticalPipelineFiles := utils.GetListOfFileInADir(filepath.Join(pipelineFilesRootDir, pipelineDir))
		for _, pipelineFile := range criticalPipelineFiles {
			pipelineFile := pipelineFile
			It(fmt.Sprintf("Upload %s pipeline", pipelineFile), func() {
				uploadPipelineAndVerify(pipelineDir, pipelineFile, &testContext.PipelineGeneratedName, nil)
			})
		}

		It(fmt.Sprintf("Upload %s pipeline file with custom name and description", helloWorldPipelineFileName), func() {
			description := "Some pipeline description"
			testContext.UploadParams.SetDescription(&description)
			testContext.ExpectedPipeline.Description = description
			uploadPipelineAndVerify(pipelineDir, helloWorldPipelineFileName, &testContext.PipelineGeneratedName, nil)
		})

		It(fmt.Sprintf("Upload %s pipeline file with custom name, display name and description", helloWorldPipelineFileName), func() {
			description := "Some pipeline description"
			displayName := fmt.Sprintf("Pipeline Display Name - %s ", testContext.PipelineGeneratedName)
			testContext.UploadParams.SetDescription(&description)
			testContext.ExpectedPipeline.Description = description
			uploadPipelineAndVerify(pipelineDir, helloWorldPipelineFileName, &testContext.PipelineGeneratedName, &displayName)
		})
	})

	/* Positive Scenarios of uploading a pipeline file */
	Context("Upload a valid pipeline and verify pipeline metadata after upload >", func() {
		var pipelineDir = "valid"
		validPipelineFiles := utils.GetListOfFileInADir(filepath.Join(pipelineFilesRootDir, pipelineDir))
		for _, pipelineFile := range validPipelineFiles {
			pipelineFile := pipelineFile
			It(fmt.Sprintf("Upload %s pipeline", pipelineFile), func() {
				uploadPipelineAndVerify(pipelineDir, pipelineFile, &testContext.PipelineGeneratedName, nil)
			})
		}
	})
})

var _ = Describe("Verify Pipeline Upload Version >", Label("Positive", "PipelineUpload", "ApiServerTests", FullRegression), func() {
	var pipelineDir = "valid/critical"

	/* Positive Scenarios of uploading a pipeline file */
	Context("Upload a pipeline and upload the same pipeline to change version >", func() {
		const pipelineFile = helloWorldPipelineFileName
		It(fmt.Sprintf("Upload %s pipeline file and upload a new version with the same file", pipelineFile), Label(Smoke), func() {
			uploadPipelineAndChangePipelineVersion(pipelineDir, pipelineFile, pipelineFile, &testContext.PipelineGeneratedName, nil)
		})
		It(fmt.Sprintf("Upload %s pipeline file and upload a new verison with the different file %s", pipelineFile, pipelineWithArgsFileName), func() {
			uploadPipelineAndChangePipelineVersion(pipelineDir, pipelineFile, pipelineWithArgsFileName, &testContext.PipelineGeneratedName, nil)
		})

	})
})

// ################################################################################################################
// ########################################################## NEGATIVE TESTS ######################################
// ################################################################################################################

var _ = Describe("Verify Pipeline Upload Failure >", Label("Negative", "PipelineUpload", "ApiServerTests", FullRegression), func() {
	var pipelineDir = "invalid"
	invalidPipelineFiles := utils.GetListOfFileInADir(filepath.Join(pipelineFilesRootDir, pipelineDir))

	/* Negative scenarios of uploading a pipeline  */
	Context("Upload a failing pipeline and verify the error in the response >", func() {
		It("Upload a pipeline twice and verify that it should fail the second time", func() {
			var pipelineDir = "valid/critical"
			createdPipeline := uploadPipelineAndVerify(pipelineDir, helloWorldPipelineFileName, &testContext.PipelineGeneratedName, nil)
			uploadPipelineAndVerifyFailure(pipelineDir, helloWorldPipelineFileName, &(createdPipeline.Name), nil, "Failed to upload pipeline")
		})

		for _, fileName := range invalidPipelineFiles {
			fileName := fileName
			It(fmt.Sprintf("Upload a %s pipeline and verify the failure", fileName), func() {
				var pipelineDir = "invalid"
				uploadPipelineAndVerifyFailure(pipelineDir, fileName, &testContext.PipelineGeneratedName, nil, "Failed to upload pipeline")
			})
		}
	})
})

var _ = Describe("Verify Pipeline Upload Version Failure >", Label("Negative", "PipelineUpload", "ApiServerTests", FullRegression), func() {
	var pipelineDir = "valid/critical"
	const pipelineFileName = helloWorldPipelineFileName
	/* Negative Scenarios of uploading a pipeline file */
	Context("Upload a pipeline and try changing the version with a different metric >", func() {
		It(fmt.Sprintf("Change %s pipeline's name to be same as original version", pipelineFileName), func() {
			createdPipeline := uploadPipelineAndVerify(pipelineDir, pipelineFileName, &testContext.PipelineGeneratedName, nil)

			parameters := upload_params.NewUploadPipelineVersionParams()
			parameters.Pipelineid = &(createdPipeline.PipelineID)
			parameters.SetName(&createdPipeline.DisplayName)
			uploadPipelineVersionAndVerifyFailure(pipelineDir, pipelineFileName, parameters, "Failed to upload pipeline version")
		})
		It(fmt.Sprintf("Change %s pipeline's id with fake pipeline id", pipelineFileName), func() {
			uploadPipelineAndVerify(pipelineDir, pipelineFileName, &testContext.PipelineGeneratedName, nil)

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
func uploadPipelineAndChangePipelineVersion(pipelineDir string, pipelineFileNameForCreation string, pipelineFileNameWhenChangingVersion string, pipelineName *string, pipelineDisplayName *string) {
	createdPipeline := uploadPipelineAndVerify(pipelineDir, pipelineFileNameForCreation, pipelineName, pipelineDisplayName)

	// Construct a payload to create new pipeline version
	parameters := upload_params.NewUploadPipelineVersionParams()
	expectedPipelineVersion := new(model.V2beta1PipelineVersion)
	descriptionNew := "Some changed pipeline description"
	pipelineNameNew := createdPipeline.DisplayName + "-1"
	parameters.Pipelineid = &(createdPipeline.PipelineID)
	parameters.SetDescription(&descriptionNew)
	parameters.SetName(&pipelineNameNew)

	// Construct expected Pipeline Spec from the uploaded file
	pipelineVersionFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, pipelineFileNameWhenChangingVersion)
	inputFileContent := utils.ParseFileToSpecs(pipelineVersionFilePath, true, nil)

	// Construct expected pipeline version object for comparison
	expectedPipelineVersion.Description = descriptionNew
	expectedPipelineVersion.PipelineSpec = inputFileContent
	expectedPipelineVersion.DisplayName = pipelineNameNew
	uploadPipelineVersionAndVerify(pipelineDir, pipelineFileNameWhenChangingVersion, parameters, expectedPipelineVersion)
}

func uploadPipeline(pipelineDir string, pipelineFileName string, pipelineName *string, pipelineDisplayName *string) (*model.V2beta1Pipeline, error) {
	pipelineFile := filepath.Join(pipelineFilesRootDir, pipelineDir, pipelineFileName)
	testContext.UploadParams.SetName(pipelineName)
	if pipelineDisplayName != nil {
		testContext.ExpectedPipeline.DisplayName = *pipelineDisplayName
		testContext.UploadParams.SetDisplayName(pipelineDisplayName)
	} else {
		testContext.ExpectedPipeline.DisplayName = *pipelineName
	}
	logger.Log("Uploading pipeline with name=%s, from file %s", *pipelineName, pipelineFile)
	return pipelineUploadClient.UploadFile(pipelineFile, testContext.UploadParams)
}

func uploadPipelineAndVerify(pipelineDir string, pipelineFileName string, pipelineName *string, pipelineDisplayName *string) *model.V2beta1Pipeline {
	createdPipeline, err := uploadPipeline(pipelineDir, pipelineFileName, pipelineName, pipelineDisplayName)
	logger.Log("Verifying that NO error was returned in the response to confirm that the pipeline was successfully uploaded")
	Expect(err).NotTo(HaveOccurred())
	testContext.CreatedPipelines = append(testContext.CreatedPipelines, createdPipeline)

	createdPipelineFromDB := utils.GetPipeline(pipelineClient, createdPipeline.PipelineID)
	Expect(createdPipelineFromDB).To(Equal(*createdPipeline))
	matcher.MatchPipelines(&createdPipelineFromDB, testContext.ExpectedPipeline, *isKubeflowMode)

	// Validate the created pipeline spec (by API server) matches the input file
	pipelineVersionFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, pipelineFileName)
	expectedPipelineSpec := utils.ParseFileToSpecs(pipelineVersionFilePath, true, nil)
	logger.Log("Verifying that the generated pipeline spec matches the input yaml file")
	versions := utils.GetSortedPipelineVersionsByCreatedAt(pipelineClient, createdPipeline.PipelineID, nil)
	Expect(versions).Should(HaveLen(1), "Expected to find only one pipeline version after pipeline upload")
	actualPipelineSpec := versions[0].PipelineSpec.(map[string]interface{})
	matcher.MatchPipelineSpecs(actualPipelineSpec, expectedPipelineSpec)
	return createdPipeline
}

func uploadPipelineAndVerifyFailure(pipelineDir string, pipelineFileName string, pipelineName *string, pipelineDisplayName *string, errorMessage string) {
	_, err := uploadPipeline(pipelineDir, pipelineFileName, pipelineName, pipelineDisplayName)
	logger.Log("Verifying error in the response")
	Expect(err).To(HaveOccurred())
	Expect(err.Error()).To(ContainSubstring(errorMessage))
}

func uploadPipelineVersion(pipelineDir string, pipelineFileName string, parameters *upload_params.UploadPipelineVersionParams) (*model.V2beta1PipelineVersion, error) {
	pipelineFile := filepath.Join(pipelineFilesRootDir, pipelineDir, pipelineFileName)
	logger.Log("Uploading pipeline version for pipeline with id=%s, from file %s", *parameters.Pipelineid, pipelineFile)
	return pipelineUploadClient.UploadPipelineVersion(pipelineFile, parameters)
}

func uploadPipelineVersionAndVerify(pipelineDir string, pipelineFileName string, parameters *upload_params.UploadPipelineVersionParams, expectedPipelineVersion *model.V2beta1PipelineVersion) *model.V2beta1PipelineVersion {
	createdPipelineVersion, err := uploadPipelineVersion(pipelineDir, pipelineFileName, parameters)
	logger.Log("Verifying that NO error was returned in the response to confirm that the pipeline was successfully uploaded")
	Expect(err).NotTo(HaveOccurred())
	matcher.MatchPipelineVersions(createdPipelineVersion, expectedPipelineVersion)
	return createdPipelineVersion
}

func uploadPipelineVersionAndVerifyFailure(pipelineDir string, pipelineFileName string, parameters *upload_params.UploadPipelineVersionParams, errorMessage string) {
	_, err := uploadPipelineVersion(pipelineDir, pipelineFileName, parameters)
	logger.Log("Verifying error in the response")
	Expect(err).To(HaveOccurred())
	Expect(err.Error()).To(ContainSubstring(errorMessage))
}
