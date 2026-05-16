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
	"time"

	pipeline_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_client/pipeline_service"
	uploadparams "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_client/pipeline_upload_service"
	model "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_model"
	"github.com/kubeflow/pipelines/backend/test/config"
	"github.com/kubeflow/pipelines/backend/test/constants"
	"github.com/kubeflow/pipelines/backend/test/logger"
	"github.com/kubeflow/pipelines/backend/test/testutil"
	"github.com/kubeflow/pipelines/backend/test/v2/api/matcher"

	"github.com/go-openapi/strfmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// ################## CLASS VARIABLES ##################

const (
	helloWorldPipelineFileName = "hello_world.yaml"
	pipelineWithArgsFileName   = "arguments_parameters.yaml"
)

// ################## SET AND TEARDOWN ##################

var _ = BeforeEach(func() {
	logger.Log("################### Setup before each test #####################")
	testStartTime, _ := strfmt.ParseDateTime(time.Now().Format(time.DateTime))
	testContext.Pipeline.CreatedPipelines = []*model.V2beta1Pipeline{}
	testContext.Pipeline.ExpectedPipeline = new(model.V2beta1Pipeline)
	testContext.Pipeline.ExpectedPipeline.CreatedAt = testStartTime
	testContext.Pipeline.PipelineGeneratedName = "apitest-" + randomName
	if *config.KubeflowMode || *config.MultiUserMode {
		testContext.Pipeline.ExpectedPipeline.Namespace = *config.UserNamespace
	} else {
		testContext.Pipeline.ExpectedPipeline.Namespace = *config.Namespace
	}
})

// ################## TESTS ##################

// ################## POSITIVE TESTS ##################

var _ = Describe("Verify Pipeline Upload >", Label(constants.POSITIVE, constants.PipelineUpload, constants.APIServerTests, constants.FullRegression), func() {

	/* Positive Scenarios of uploading a pipeline file */
	Context("Upload a valid pipeline and verify pipeline metadata after upload >", func() {
		var pipelineDir = "valid"
		validPipelineFilePaths := testutil.GetListOfAllFilesInDir(filepath.Join(pipelineFilesRootDir, pipelineDir))
		for _, pipelineFilePath := range validPipelineFilePaths {
			It(fmt.Sprintf("Upload %s pipeline", pipelineFilePath), func() {
				uploadPipelineAndVerify(pipelineFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)
			})
		}

		It(fmt.Sprintf("Upload %s pipeline file with custom name and description", helloWorldPipelineFileName), func() {
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			description := "Some pipeline description"
			testContext.Pipeline.UploadParams.SetDescription(&description)
			testContext.Pipeline.ExpectedPipeline.Description = description
			uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)
		})

		It(fmt.Sprintf("Upload %s pipeline file with custom name, display name and description", helloWorldPipelineFileName), func() {
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			description := "Some pipeline description"
			displayName := fmt.Sprintf("Pipeline Display Name - %s ", testContext.Pipeline.PipelineGeneratedName)
			testContext.Pipeline.UploadParams.SetDescription(&description)
			testContext.Pipeline.ExpectedPipeline.Description = description
			uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, &displayName)
		})
	})
})

var _ = Describe("Verify Pipeline Upload with Tags >", Label(constants.POSITIVE, constants.PipelineUpload, constants.APIServerTests, constants.FullRegression), func() {

	Context("Upload a valid pipeline with tags and verify tags are stored >", func() {
		var pipelineDir = "valid"

		It(fmt.Sprintf("Upload %s pipeline with tags and verify tags are returned", helloWorldPipelineFileName), func() {
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			tags := map[string]string{"team": "ml-ops", "env": "prod"}
			tagsJSON, err := testutil.TagsMapToJSONStringPtr(tags)
			Expect(err).NotTo(HaveOccurred())
			testContext.Pipeline.UploadParams.Tags = tagsJSON
			testContext.Pipeline.ExpectedPipeline.Tags = tags
			uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)
		})

		It(fmt.Sprintf("Upload %s pipeline with empty tags map", helloWorldPipelineFileName), func() {
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			emptyTags := map[string]string{}
			tagsJSON, err := testutil.TagsMapToJSONStringPtr(emptyTags)
			Expect(err).NotTo(HaveOccurred())
			testContext.Pipeline.UploadParams.Tags = tagsJSON
			uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)
		})

		It(fmt.Sprintf("Upload %s pipeline without tags and verify no tags are returned", helloWorldPipelineFileName), func() {
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)
		})

		It(fmt.Sprintf("Upload %s pipeline with tags and verify tags via GetPipeline", helloWorldPipelineFileName), func() {
			pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
			tags := map[string]string{"project": "kfp", "owner": "test"}
			tagsJSON, err := testutil.TagsMapToJSONStringPtr(tags)
			Expect(err).NotTo(HaveOccurred())
			testContext.Pipeline.UploadParams.Tags = tagsJSON
			testContext.Pipeline.ExpectedPipeline.Tags = tags
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			// Verify tags are returned via GetPipeline endpoint
			retrievedPipeline := testutil.GetPipeline(pipelineClient, createdPipeline.PipelineID)
			Expect(retrievedPipeline.Tags).To(Equal(tags), "Tags should be returned via GetPipeline")
		})
	})
})

var _ = Describe("Verify Pipeline Upload Version >", Label(constants.POSITIVE, "PipelineUpload", constants.APIServerTests, constants.FullRegression), func() {
	var pipelineDir = "valid"
	helloWorldPipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
	argParamPipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, pipelineWithArgsFileName)
	/* Positive Scenarios of uploading a pipeline file */
	Context("Upload a pipeline and upload the same pipeline to change version >", func() {
		It(fmt.Sprintf("Upload %s pipeline file and upload a new version with the same file", helloWorldPipelineFileName), Label(constants.SMOKE), func() {
			uploadPipelineAndChangePipelineVersion(helloWorldPipelineSpecFilePath, helloWorldPipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)
		})
		It(fmt.Sprintf("Upload %s pipeline file and upload a new version with the different file %s", helloWorldPipelineFileName, pipelineWithArgsFileName), func() {
			uploadPipelineAndChangePipelineVersion(helloWorldPipelineSpecFilePath, argParamPipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)
		})

	})
})

var _ = Describe("Verify Pipeline Upload Version with Tags >", Label(constants.POSITIVE, "PipelineUpload", constants.APIServerTests, constants.FullRegression), func() {
	var pipelineDir = "valid"
	helloWorldPipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)

	Context("Upload a pipeline version with tags and verify tags are stored >", func() {
		It("Upload a pipeline version with tags and verify tags are returned in the response", func() {
			createdPipeline := uploadPipelineAndVerify(helloWorldPipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			parameters := uploadparams.NewUploadPipelineVersionParams()
			expectedPipelineVersion := new(model.V2beta1PipelineVersion)
			versionName := createdPipeline.DisplayName + "-v2-tagged"
			parameters.Pipelineid = &(createdPipeline.PipelineID)
			parameters.SetName(&versionName)

			tags := map[string]string{"team": "ml-ops", "env": "prod"}
			tagsJSON, err := testutil.TagsMapToJSONStringPtr(tags)
			Expect(err).NotTo(HaveOccurred())
			parameters.Tags = tagsJSON

			inputFileContent := testutil.ParseFileToSpecs(helloWorldPipelineSpecFilePath, true, nil)
			expectedPipelineVersion.DisplayName = versionName
			expectedPipelineVersion.PipelineSpec = inputFileContent

			createdVersion := uploadPipelineVersionAndVerify(helloWorldPipelineSpecFilePath, parameters, expectedPipelineVersion)
			Expect(createdVersion.Tags).To(Equal(tags), "Tags should be returned in the upload response")
		})

		It("Upload a pipeline version with tags and verify tags via GetPipelineVersion endpoint", func() {
			createdPipeline := uploadPipelineAndVerify(helloWorldPipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			parameters := uploadparams.NewUploadPipelineVersionParams()
			expectedPipelineVersion := new(model.V2beta1PipelineVersion)
			versionName := createdPipeline.DisplayName + "-v2-tagged-get"
			parameters.Pipelineid = &(createdPipeline.PipelineID)
			parameters.SetName(&versionName)

			tags := map[string]string{"project": "kfp", "owner": "test"}
			tagsJSON, err := testutil.TagsMapToJSONStringPtr(tags)
			Expect(err).NotTo(HaveOccurred())
			parameters.Tags = tagsJSON

			inputFileContent := testutil.ParseFileToSpecs(helloWorldPipelineSpecFilePath, true, nil)
			expectedPipelineVersion.DisplayName = versionName
			expectedPipelineVersion.PipelineSpec = inputFileContent

			createdVersion := uploadPipelineVersionAndVerify(helloWorldPipelineSpecFilePath, parameters, expectedPipelineVersion)

			// Verify tags via GetPipelineVersion endpoint
			retrievedVersion, err := pipelineClient.GetPipelineVersion(&pipeline_params.PipelineServiceGetPipelineVersionParams{
				PipelineID:        createdPipeline.PipelineID,
				PipelineVersionID: createdVersion.PipelineVersionID,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(retrievedVersion.Tags).To(Equal(tags), "Tags should be returned via GetPipelineVersion endpoint")
		})

		It("Upload a pipeline version with tags and verify tags via ListPipelineVersions", func() {
			createdPipeline := uploadPipelineAndVerify(helloWorldPipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			parameters := uploadparams.NewUploadPipelineVersionParams()
			expectedPipelineVersion := new(model.V2beta1PipelineVersion)
			versionName := createdPipeline.DisplayName + "-v2-tagged-list"
			parameters.Pipelineid = &(createdPipeline.PipelineID)
			parameters.SetName(&versionName)

			tags := map[string]string{"stage": "beta", "release": "v1"}
			tagsJSON, err := testutil.TagsMapToJSONStringPtr(tags)
			Expect(err).NotTo(HaveOccurred())
			parameters.Tags = tagsJSON

			inputFileContent := testutil.ParseFileToSpecs(helloWorldPipelineSpecFilePath, true, nil)
			expectedPipelineVersion.DisplayName = versionName
			expectedPipelineVersion.PipelineSpec = inputFileContent

			uploadPipelineVersionAndVerify(helloWorldPipelineSpecFilePath, parameters, expectedPipelineVersion)

			// Verify tags via ListPipelineVersions
			versions := testutil.GetSortedPipelineVersionsByCreatedAt(pipelineClient, createdPipeline.PipelineID, nil)
			Expect(len(versions)).To(BeNumerically(">=", 2))

			var foundTaggedVersion bool
			for _, v := range versions {
				if v.DisplayName == versionName {
					Expect(v.Tags).To(Equal(tags), "Tagged version should include tags in list response")
					foundTaggedVersion = true
					break
				}
			}
			Expect(foundTaggedVersion).To(BeTrue(), "Should find the tagged version in the list")
		})

		It("Upload a pipeline version without tags and verify no tags are returned", func() {
			createdPipeline := uploadPipelineAndVerify(helloWorldPipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			parameters := uploadparams.NewUploadPipelineVersionParams()
			expectedPipelineVersion := new(model.V2beta1PipelineVersion)
			versionName := createdPipeline.DisplayName + "-v2-no-tags"
			parameters.Pipelineid = &(createdPipeline.PipelineID)
			parameters.SetName(&versionName)

			inputFileContent := testutil.ParseFileToSpecs(helloWorldPipelineSpecFilePath, true, nil)
			expectedPipelineVersion.DisplayName = versionName
			expectedPipelineVersion.PipelineSpec = inputFileContent

			createdVersion := uploadPipelineVersionAndVerify(helloWorldPipelineSpecFilePath, parameters, expectedPipelineVersion)
			Expect(createdVersion.Tags).To(BeEmpty(), "Version uploaded without tags should have empty tags")
		})

		It("Upload a pipeline version with empty tags map", func() {
			createdPipeline := uploadPipelineAndVerify(helloWorldPipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			parameters := uploadparams.NewUploadPipelineVersionParams()
			expectedPipelineVersion := new(model.V2beta1PipelineVersion)
			versionName := createdPipeline.DisplayName + "-v2-empty-tags"
			parameters.Pipelineid = &(createdPipeline.PipelineID)
			parameters.SetName(&versionName)

			emptyTags := map[string]string{}
			tagsJSON, err := testutil.TagsMapToJSONStringPtr(emptyTags)
			Expect(err).NotTo(HaveOccurred())
			parameters.Tags = tagsJSON

			inputFileContent := testutil.ParseFileToSpecs(helloWorldPipelineSpecFilePath, true, nil)
			expectedPipelineVersion.DisplayName = versionName
			expectedPipelineVersion.PipelineSpec = inputFileContent

			createdVersion := uploadPipelineVersionAndVerify(helloWorldPipelineSpecFilePath, parameters, expectedPipelineVersion)
			Expect(createdVersion.Tags).To(BeEmpty(), "Version uploaded with empty tags should have empty tags")
		})
	})
})

// ################## NEGATIVE TESTS ##################

var _ = Describe("Verify Pipeline Upload Version with Tags Failure >", Label("Negative", "PipelineUpload", constants.APIServerTests, constants.FullRegression), func() {
	var pipelineDir = "valid"
	pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)

	Context("Upload a pipeline version with invalid tags >", func() {
		It("Upload a pipeline version with tag key exceeding 63 characters", func() {
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			parameters := uploadparams.NewUploadPipelineVersionParams()
			versionName := createdPipeline.DisplayName + "-v2-long-key"
			parameters.Pipelineid = &(createdPipeline.PipelineID)
			parameters.SetName(&versionName)

			longKey := "this-key-is-way-too-long-for-tags-and-exceeds-the-sixty-three-char-limit"
			tags := map[string]string{longKey: "value"}
			tagsJSON, err := testutil.TagsMapToJSONStringPtr(tags)
			Expect(err).NotTo(HaveOccurred())
			parameters.Tags = tagsJSON

			uploadPipelineVersionAndVerifyFailure(pipelineSpecFilePath, parameters, "Failed to upload pipeline version")
		})

		It("Upload a pipeline version with tag value exceeding 63 characters", func() {
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			parameters := uploadparams.NewUploadPipelineVersionParams()
			versionName := createdPipeline.DisplayName + "-v2-long-value"
			parameters.Pipelineid = &(createdPipeline.PipelineID)
			parameters.SetName(&versionName)

			longValue := "this-value-is-way-too-long-for-tags-and-exceeds-the-sixty-three-char-limit"
			tags := map[string]string{"key": longValue}
			tagsJSON, err := testutil.TagsMapToJSONStringPtr(tags)
			Expect(err).NotTo(HaveOccurred())
			parameters.Tags = tagsJSON

			uploadPipelineVersionAndVerifyFailure(pipelineSpecFilePath, parameters, "Failed to upload pipeline version")
		})

		It("Upload a pipeline version with invalid tags JSON format", func() {
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			parameters := uploadparams.NewUploadPipelineVersionParams()
			versionName := createdPipeline.DisplayName + "-v2-bad-json"
			parameters.Pipelineid = &(createdPipeline.PipelineID)
			parameters.SetName(&versionName)

			invalidJSON := "not-valid-json"
			parameters.SetTags(&invalidJSON)

			uploadPipelineVersionAndVerifyFailure(pipelineSpecFilePath, parameters, "Failed to upload pipeline version")
		})
	})
})

var _ = Describe("Verify Pipeline Upload Failure >", Label("Negative", "PipelineUpload", constants.APIServerTests, constants.FullRegression), func() {
	var pipelineDir = "invalid"
	invalidPipelineFiles := testutil.GetListOfFilesInADir(filepath.Join(pipelineFilesRootDir, pipelineDir))

	/* Negative scenarios of uploading a pipeline  */
	Context("Upload an invalid pipeline spec and verify the error in the response >", func() {
		for _, fileName := range invalidPipelineFiles {
			filePath := filepath.Join(pipelineFilesRootDir, pipelineDir, fileName)
			It(fmt.Sprintf("Upload a %s pipeline and verify the failure", fileName), func() {
				uploadPipelineAndVerifyFailure(filePath, &testContext.Pipeline.PipelineGeneratedName, nil, "Failed to upload pipeline")
			})
		}

		It("Upload a pipeline twice and verify that it should fail the second time", func() {
			var validPipelineDir = "valid"
			filePath := filepath.Join(pipelineFilesRootDir, validPipelineDir, helloWorldPipelineFileName)
			createdPipeline := uploadPipelineAndVerify(filePath, &testContext.Pipeline.PipelineGeneratedName, nil)
			uploadPipelineAndVerifyFailure(filePath, &(createdPipeline.Name), nil, "Failed to upload pipeline")
		})
	})
})

var _ = Describe("Verify Pipeline Upload Version Failure >", Label("Negative", "PipelineUpload", constants.APIServerTests, constants.FullRegression), func() {
	var pipelineDir = "valid"
	pipelineSpecFilePath := filepath.Join(pipelineFilesRootDir, pipelineDir, helloWorldPipelineFileName)
	/* Negative Scenarios of uploading a pipeline file */
	Context("Upload a pipeline and try changing the version with a different metric >", func() {
		It(fmt.Sprintf("Change %s pipeline's name to be same as original version", helloWorldPipelineFileName), func() {
			createdPipeline := uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			parameters := uploadparams.NewUploadPipelineVersionParams()
			parameters.Pipelineid = &(createdPipeline.PipelineID)
			parameters.SetName(&createdPipeline.DisplayName)
			uploadPipelineVersionAndVerifyFailure(pipelineSpecFilePath, parameters, "Failed to upload pipeline version")
		})
		It(fmt.Sprintf("Change %s pipeline's id with fake pipeline id", helloWorldPipelineFileName), func() {
			uploadPipelineAndVerify(pipelineSpecFilePath, &testContext.Pipeline.PipelineGeneratedName, nil)

			parameters := uploadparams.NewUploadPipelineVersionParams()
			fakePipelineID := "12345"
			parameters.Pipelineid = &fakePipelineID
			uploadPipelineVersionAndVerifyFailure(pipelineSpecFilePath, parameters, "Failed to upload pipeline version")
		})
	})

	// TODO: To to be implemented
	if *config.KubeflowMode {
		PContext("Upload a pipeline in MultiUser Mode >", func() {
			It("Upload a pipeline in a namespace you don't have access to", func() {
			})
		})
	}
})

// ################## UTILITY METHODS ##################

/*
A common method that creates a pipeline and then creates a new pipeline version
@param pipelineFilePathForCreation - pipeline file path for initial pipeline upload
@param pipelineFilePathWhenChangingVersion - the pipeline file path that you wish to upload when creating a new version
*/
func uploadPipelineAndChangePipelineVersion(pipelineFilePathForCreation string, pipelineFilePathWhenChangingVersion string, pipelineName *string, pipelineDisplayName *string) {
	createdPipeline := uploadPipelineAndVerify(pipelineFilePathForCreation, pipelineName, pipelineDisplayName)

	// Construct a payload to create new pipeline version
	parameters := uploadparams.NewUploadPipelineVersionParams()
	expectedPipelineVersion := new(model.V2beta1PipelineVersion)
	descriptionNew := "Some changed pipeline description"
	pipelineNameNew := createdPipeline.DisplayName + "-1"
	parameters.Pipelineid = &(createdPipeline.PipelineID)
	parameters.SetDescription(&descriptionNew)
	parameters.SetName(&pipelineNameNew)

	// Construct expected Pipeline Spec from the uploaded file
	inputFileContent := testutil.ParseFileToSpecs(pipelineFilePathWhenChangingVersion, true, nil)

	// Construct expected pipeline version object for comparison
	expectedPipelineVersion.Description = descriptionNew
	expectedPipelineVersion.PipelineSpec = inputFileContent
	expectedPipelineVersion.DisplayName = pipelineNameNew
	uploadPipelineVersionAndVerify(pipelineFilePathWhenChangingVersion, parameters, expectedPipelineVersion)
}

func uploadPipeline(pipelineFilePath string, pipelineName *string, pipelineDisplayName *string) (*model.V2beta1Pipeline, error) {
	testContext.Pipeline.UploadParams.SetName(pipelineName)
	if pipelineDisplayName != nil {
		testContext.Pipeline.ExpectedPipeline.DisplayName = *pipelineDisplayName
		testContext.Pipeline.UploadParams.SetDisplayName(pipelineDisplayName)
	} else {
		testContext.Pipeline.ExpectedPipeline.DisplayName = *pipelineName
	}
	// Set namespace so the pipeline is created in the correct namespace (required for multi-user mode)
	namespace := testutil.GetNamespace()
	if namespace != "" {
		testContext.Pipeline.UploadParams.SetNamespace(&namespace)
	}
	logger.Log("Uploading pipeline with name=%s, from file %s", *pipelineName, pipelineFilePath)
	return pipelineUploadClient.UploadFile(pipelineFilePath, testContext.Pipeline.UploadParams)
}

func uploadPipelineAndVerify(pipelineFilePath string, pipelineName *string, pipelineDisplayName *string) *model.V2beta1Pipeline {
	createdPipeline, err := uploadPipeline(pipelineFilePath, pipelineName, pipelineDisplayName)
	logger.Log("Verifying that NO error was returned in the response to confirm that the pipeline was successfully uploaded")
	Expect(err).NotTo(HaveOccurred())
	testContext.Pipeline.CreatedPipelines = append(testContext.Pipeline.CreatedPipelines, createdPipeline)

	createdPipelineFromDB := testutil.GetPipeline(pipelineClient, createdPipeline.PipelineID)
	Expect(createdPipelineFromDB).To(Equal(*createdPipeline))
	matcher.MatchPipelines(&createdPipelineFromDB, testContext.Pipeline.ExpectedPipeline)

	// Validate the created pipeline spec (by API server) matches the input file
	expectedPipelineSpec := testutil.ParseFileToSpecs(pipelineFilePath, true, nil)
	logger.Log("Verifying that the generated pipeline spec matches the input yaml file")
	versions := testutil.GetSortedPipelineVersionsByCreatedAt(pipelineClient, createdPipeline.PipelineID, nil)
	Expect(versions).Should(HaveLen(1), "Expected to find only one pipeline version after pipeline upload")
	actualPipelineSpec := versions[0].PipelineSpec.(map[string]interface{})
	matcher.MatchPipelineSpecs(actualPipelineSpec, expectedPipelineSpec)
	return createdPipeline
}

func uploadPipelineAndVerifyFailure(pipelineFilePath string, pipelineName *string, pipelineDisplayName *string, errorMessage string) {
	_, err := uploadPipeline(pipelineFilePath, pipelineName, pipelineDisplayName)
	logger.Log("Verifying error in the response")
	Expect(err).To(HaveOccurred())
	Expect(err.Error()).To(ContainSubstring(errorMessage))
}

func uploadPipelineVersion(pipelineFilePath string, parameters *uploadparams.UploadPipelineVersionParams) (*model.V2beta1PipelineVersion, error) {
	logger.Log("Uploading pipeline version for pipeline with id=%s, from file %s", *parameters.Pipelineid, pipelineFilePath)
	return pipelineUploadClient.UploadPipelineVersion(pipelineFilePath, parameters)
}

func uploadPipelineVersionAndVerify(pipelineFilePath string, parameters *uploadparams.UploadPipelineVersionParams, expectedPipelineVersion *model.V2beta1PipelineVersion) *model.V2beta1PipelineVersion {
	createdPipelineVersion, err := uploadPipelineVersion(pipelineFilePath, parameters)
	logger.Log("Verifying that NO error was returned in the response to confirm that the pipeline was successfully uploaded")
	Expect(err).NotTo(HaveOccurred())
	matcher.MatchPipelineVersions(createdPipelineVersion, expectedPipelineVersion)
	return createdPipelineVersion
}

func uploadPipelineVersionAndVerifyFailure(pipelineFilePath string, parameters *uploadparams.UploadPipelineVersionParams, errorMessage string) {
	_, err := uploadPipelineVersion(pipelineFilePath, parameters)
	logger.Log("Verifying error in the response")
	Expect(err).To(HaveOccurred())
	Expect(err.Error()).To(ContainSubstring(errorMessage))
}
