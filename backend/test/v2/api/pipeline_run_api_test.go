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
	"strings"

	experimentparams "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/experiment_client/experiment_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/experiment_model"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_model"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_model"
	runparams "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_client/run_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_model"
	"github.com/kubeflow/pipelines/backend/test/config"
	"github.com/kubeflow/pipelines/backend/test/constants"
	"github.com/kubeflow/pipelines/backend/test/logger"
	"github.com/kubeflow/pipelines/backend/test/testutil"
	"github.com/kubeflow/pipelines/backend/test/v2/api/matcher"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// ################## CLASS VARIABLES ##################

var experimentName string
var runName string
var runDescription string

// ################## SETUP AND TEARDOWN ##################

var _ = BeforeEach(func() {
	logger.Log("Setting up Pipeline Run Tests")
	runName = "API Test Run - " + randomName
	runDescription = "API Test Run"
	experimentName = "API Test Experiment - " + randomName
})

// ################## TESTS ##################
// ################## POSITIVE TESTS ##################

var _ = Describe("Verify Pipeline Run >", Label(constants.POSITIVE, constants.PipelineRun, constants.APIServerTests, constants.FullRegression), func() {

	type TestParams struct {
		pipelineCacheEnabled bool
	}

	testParams := []TestParams{
		{pipelineCacheEnabled: true},
		{pipelineCacheEnabled: false},
	}
	pipelineDirectory := "valid"
	pipelineFilePaths := testutil.GetListOfAllFilesInDir(filepath.Join(pipelineFilesRootDir, pipelineDirectory))

	Context("Create a valid pipeline and verify the created run >", func() {
		for _, param := range testParams {
			for _, pipelineFilePath := range pipelineFilePaths {
				It(fmt.Sprintf("Create a '%s' pipeline with cacheEnabled=%t and verify run", pipelineFilePath, param.pipelineCacheEnabled), func() {
					createdExperiment := createExperiment(experimentName)
					pipelineFilePath := pipelineFilePath
					pipelineFileName := filepath.Base(pipelineFilePath)
					testutil.CheckIfSkipping(pipelineFileName)
					configuredPipelineSpecFile := configureCacheSettingAndGetPipelineFile(pipelineFilePath, param.pipelineCacheEnabled)
					createdPipeline := uploadAPipeline(configuredPipelineSpecFile, &testContext.Pipeline.PipelineGeneratedName)
					createdPipelineVersion := testutil.GetLatestPipelineVersion(pipelineClient, &createdPipeline.PipelineID)
					pipelineRuntimeInputs := testutil.GetPipelineRunTimeInputs(configuredPipelineSpecFile)
					createdPipelineRun := createPipelineRun(&createdPipeline.PipelineID, &createdPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, pipelineRuntimeInputs)
					createdExpectedRunAndVerify(createdPipelineRun, &createdPipeline.PipelineID, &createdPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, pipelineRuntimeInputs)
				})
			}
		}
		pipelineFile := pipelineFilePaths[0]
		It(fmt.Sprintf("Create a '%s' pipeline, create an experiement and verify run with associated experiment", pipelineFile), Label(constants.SMOKE), func() {
			createdExperiment := createExperiment(experimentName)
			createdPipeline := uploadAPipeline(pipelineFile, &testContext.Pipeline.PipelineGeneratedName)
			createdPipelineVersion := testutil.GetLatestPipelineVersion(pipelineClient, &createdPipeline.PipelineID)
			pipelineRuntimeInputs := testutil.GetPipelineRunTimeInputs(pipelineFile)
			createdPipelineRun := createPipelineRun(&createdPipeline.PipelineID, &createdPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, pipelineRuntimeInputs)
			createdExpectedRunAndVerify(createdPipelineRun, &createdPipeline.PipelineID, &createdPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, pipelineRuntimeInputs)
		})
	})

	Context("Associate a single experiment with multiple pipeline runs >", func() {
		pipelineFile := pipelineFilePaths[0]
		It("Create an experiment and associate it multiple pipeline runs of the same pipeline", func() {
			createdExperiment := createExperiment(experimentName)
			createdPipeline := uploadAPipeline(pipelineFile, &testContext.Pipeline.PipelineGeneratedName)
			createdPipelineVersion := testutil.GetLatestPipelineVersion(pipelineClient, &createdPipeline.PipelineID)
			pipelineRuntimeInputs := testutil.GetPipelineRunTimeInputs(pipelineFile)
			createdPipelineRun1 := createPipelineRun(&createdPipeline.PipelineID, &createdPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, pipelineRuntimeInputs)
			createdExpectedRunAndVerify(createdPipelineRun1, &createdPipeline.PipelineID, &createdPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, pipelineRuntimeInputs)

			createdPipelineRun2 := createPipelineRun(&createdPipeline.PipelineID, &createdPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, pipelineRuntimeInputs)
			createdExpectedRunAndVerify(createdPipelineRun2, &createdPipeline.PipelineID, &createdPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, pipelineRuntimeInputs)
		})
		It("Create an experiment and associate it pipeline runs of different pipelines", func() {

			createdExperiment := createExperiment(experimentName)
			createdPipeline1 := uploadAPipeline(pipelineFile, &testContext.Pipeline.PipelineGeneratedName)
			createdPipeline1Version := testutil.GetLatestPipelineVersion(pipelineClient, &createdPipeline1.PipelineID)
			pipeline2Name := testContext.Pipeline.PipelineGeneratedName + "2"
			createdPipeline2 := uploadAPipeline(pipelineFile, &pipeline2Name)
			createdPipeline2Version := testutil.GetLatestPipelineVersion(pipelineClient, &createdPipeline2.PipelineID)
			pipelineRuntimeInputs := testutil.GetPipelineRunTimeInputs(pipelineFile)
			createdPipelineRun1 := createPipelineRun(&createdPipeline1.PipelineID, &createdPipeline1Version.PipelineVersionID, &createdExperiment.ExperimentID, pipelineRuntimeInputs)
			createdExpectedRunAndVerify(createdPipelineRun1, &createdPipeline1.PipelineID, &createdPipeline1Version.PipelineVersionID, &createdExperiment.ExperimentID, pipelineRuntimeInputs)

			createdPipelineRun2 := createPipelineRun(&createdPipeline2.PipelineID, &createdPipeline2Version.PipelineVersionID, &createdExperiment.ExperimentID, pipelineRuntimeInputs)
			createdExpectedRunAndVerify(createdPipelineRun2, &createdPipeline2.PipelineID, &createdPipeline2Version.PipelineVersionID, &createdExperiment.ExperimentID, pipelineRuntimeInputs)
		})
	})

	Context("Create a pipeline run with HTTP proxy >", func() {
		pipelineFile := filepath.Join(pipelineFilesRootDir, pipelineDirectory, "env-var.yaml")
		It(fmt.Sprintf("Create a pipeline run with http proxy, using specs: %s", pipelineFile), func() {
			createdExperiment := createExperiment(experimentName)
			createdPipeline := uploadAPipeline(pipelineFile, &testContext.Pipeline.PipelineGeneratedName)
			createdPipelineVersion := testutil.GetLatestPipelineVersion(pipelineClient, &createdPipeline.PipelineID)
			pipelineRuntimeInputs := map[string]interface{}{
				"env_var": "http_proxy",
			}
			createdPipelineRun := createPipelineRun(&createdPipeline.PipelineID, &createdPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, pipelineRuntimeInputs)
			createdExpectedRunAndVerify(createdPipelineRun, &createdPipeline.PipelineID, &createdPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, pipelineRuntimeInputs)
		})
	})

	Context("Archive pipeline run(s) >", func() {
		pipelineFile := filepath.Join(pipelineFilesRootDir, pipelineDirectory, "hello-world.yaml")
		It("Create a pipeline run, archive it and verify that the run state does not change on archiving", func() {
			createdExperiment := createExperiment(experimentName)
			createdPipeline := uploadAPipeline(pipelineFile, &testContext.Pipeline.PipelineGeneratedName)
			createdPipelineVersion := testutil.GetLatestPipelineVersion(pipelineClient, &createdPipeline.PipelineID)
			pipelineRuntimeInputs := testutil.GetPipelineRunTimeInputs(pipelineFile)
			createdPipelineRun := createPipelineRun(&createdPipeline.PipelineID, &createdPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, pipelineRuntimeInputs)
			archivePipelineRun(&createdPipelineRun.RunID)
			pipelineRunAfterArchive := testutil.GetPipelineRun(runClient, &createdPipelineRun.RunID)
			Expect(createdPipelineRun.State).To(Equal(pipelineRunAfterArchive.State))
			Expect(*pipelineRunAfterArchive.StorageState).To(Equal(run_model.V2beta1RunStorageStateARCHIVED))

		})

		It("Create a pipeline run, wait for the run to move to RUNNING, archive it and verify that the run state is still RUNNING on archiving", func() {
			createdExperiment := createExperiment(experimentName)
			createdPipeline := uploadAPipeline(pipelineFile, &testContext.Pipeline.PipelineGeneratedName)
			createdPipelineVersion := testutil.GetLatestPipelineVersion(pipelineClient, &createdPipeline.PipelineID)
			pipelineRuntimeInputs := testutil.GetPipelineRunTimeInputs(pipelineFile)
			createdPipelineRun := createPipelineRun(&createdPipeline.PipelineID, &createdPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, pipelineRuntimeInputs)
			testutil.WaitForRunToBeInState(runClient, &createdPipelineRun.RunID, []run_model.V2beta1RuntimeState{run_model.V2beta1RuntimeStateRUNNING}, nil)
			archivePipelineRun(&createdPipelineRun.RunID)
			pipelineRunAfterArchive := testutil.GetPipelineRun(runClient, &createdPipelineRun.RunID)
			Expect(*pipelineRunAfterArchive.State).To(Equal(run_model.V2beta1RuntimeStateRUNNING))
			Expect(*pipelineRunAfterArchive.StorageState).To(Equal(run_model.V2beta1RunStorageStateARCHIVED))

		})
	})

	Context("Unarchive pipeline run(s) >", func() {
		pipelineFile := filepath.Join(pipelineFilesRootDir, pipelineDirectory, "hello-world.yaml")
		It("Create a pipeline run, archive it and unarchive it and verify the storage state", func() {
			createdExperiment := createExperiment(experimentName)
			createdPipeline := uploadAPipeline(pipelineFile, &testContext.Pipeline.PipelineGeneratedName)
			createdPipelineVersion := testutil.GetLatestPipelineVersion(pipelineClient, &createdPipeline.PipelineID)
			pipelineRuntimeInputs := testutil.GetPipelineRunTimeInputs(pipelineFile)
			createdPipelineRun := createPipelineRun(&createdPipeline.PipelineID, &createdPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, pipelineRuntimeInputs)
			archivePipelineRun(&createdPipelineRun.RunID)
			unArchivePipelineRun(&createdPipelineRun.RunID)
			pipelineRunAfterUnArchive := testutil.GetPipelineRun(runClient, &createdPipelineRun.RunID)
			Expect(*pipelineRunAfterUnArchive.StorageState).To(Equal(run_model.V2beta1RunStorageStateAVAILABLE))
		})
	})

	PContext("Terminate a pipeline run >", func() {
		It("Terminate a run in RUNNING state", func() {
		})
		It("Terminate a run in PENDING state", func() {
		})
		It("Terminate a run in SUCCESSFUL or ERRORED state", func() {
		})
	})

	PContext("Get All pipeline run >", func() {
		It("Create a Pipeline Run and validate that it gets returned in the List Runs API call", func() {
		})
		It("Create 2 pipeline Runs, and list it", func() {
		})
		It("List pipeline Runs when no runs exist", func() {
		})
		It("List pipeline Runs and sort by display name in ascending order", func() {
		})
		It("List pipeline Runs and sort by display name in descending order", func() {
		})
		It("List pipeline Runs and sort by id in ascending order", func() {
		})
		It("List pipeline Runs and sort by id in descending order", func() {
		})
		It("List pipeline Runs and sort by pipeline version id in ascending order", func() {
		})
		It("List pipeline Runs and sort by pipeline version id in descending order", func() {
		})
		It("List pipeline Runs and sort by creation date in ascending order", func() {
		})
		It("List pipeline Runs and sort by creation date in descending order", func() {
		})
		It("List pipeline Runs and sort by updated date in ascending order", func() {
		})
		It("List pipeline Runs and sort by updated date in descending order", func() {
		})
		It("List pipeline Runs by specifying page size", func() {
		})
		It("List pipeline Runs by specifying page size and iterate over at least 2 pages", func() {
		})
		It("List pipeline Runs filtering by run id", func() {
		})
		It("List pipeline Runs filtering by display name containing", func() {
		})
		It("List pipeline Runs filtering by `name` EQUALS", func() {
		})
		It("List pipeline Runs filtering by `name` NOT Equals", func() {
		})
		It("List pipeline Runs filtering by creation date", func() {
		})
		It("List pipeline Runs by experiment id", func() {
		})
		It("List pipeline Runs by namespace", func() {
		})
	})
})

// ################## NEGATIVE TESTS ##################

var _ = Describe("Verify Pipeline Run Negative Tests >", Label(constants.NEGATIVE, constants.PipelineRun, constants.APIServerTests, constants.FullRegression), func() {

	var pipelineFile string
	var createdPipeline *pipeline_upload_model.V2beta1Pipeline
	var createdPipelineVersion *pipeline_model.V2beta1PipelineVersion
	var pipelineRuntimeInputs map[string]interface{}

	BeforeEach(func() {
		pipelineFile = filepath.Join(testutil.GetValidPipelineFilesDir(), helloWorldPipelineFileName)
		createdPipeline = uploadAPipeline(pipelineFile, &testContext.Pipeline.PipelineGeneratedName)
		createdPipelineVersion = testutil.GetLatestPipelineVersion(pipelineClient, &createdPipeline.PipelineID)
		pipelineRuntimeInputs = testutil.GetPipelineRunTimeInputs(pipelineFile)
	})

	if *config.MultiUserMode || *config.KubeflowMode {
		Context("Verify pipeline run creation failure in Multi User Mode", func() {

			PIt("Create a run in an experiment that is not in the namespace that you have access to", func() {
			})
			It("Create a run without an experiment in a Multi User Mode deployment", func() {
				createRunRequest := &runparams.RunServiceCreateRunParams{Run: createPipelineRunPayload(&createdPipeline.PipelineID, &createdPipelineVersion.PipelineVersionID, nil, pipelineRuntimeInputs)}
				_, createRunError := runClient.Create(createRunRequest)
				Expect(createRunError).To(HaveOccurred(), "Expected the pipeline run creation to have failed")
				Expect(createRunError.Error()).To(ContainSubstring("Experiment id can not be empty in multi-user mode"), "Expected the pipeline run creation failure to have a specific error message in the response")

			})
		})
	}
	PContext("Unarchive a pipeline run >", func() {
		It("Unarchive a deleted run", func() {
		})
		It("Unarchive a non existent run", func() {
		})
		It("Unarchive an available run", func() {
		})
		if *config.KubeflowMode || *config.MultiUserMode {
			It("In a namespace you don't have access to", func() {
			})
		}
	})
	PContext("Archive a pipeline run >", func() {
		It("Archive a deleted run", func() {
		})
		It("Archive a non existent run", func() {
		})
		It("Archive an already archived run", func() {
		})
		if *config.KubeflowMode {
			It("In a namespace you don't have access to", func() {
			})
		}
	})
	PContext("Terminate a pipeline run >", func() {
		It("Terminate a deleted run", func() {
		})
		It("Terminate a non existent run", func() {
		})
		It("Terminate an already terminated run", func() {
		})
		if *config.KubeflowMode {
			It("In a namespace you don't have access to", func() {
			})
		}
	})
	PContext("Delete a pipeline run >", func() {
		It("Delete a deleted run", func() {
		})
		It("Delete a non existent run", func() {
		})
	})
	PContext("Associate a pipeline run with invalid experiment >", func() {
		It("Associate a run with an archived experiment", func() {
		})
		It("Associate a run with non existent experiment", func() {
		})
		It("Associate a run with deleted experiment", func() {
		})
	})
})

// ################## UTILITY METHODS ##################

func configureCacheSettingAndGetPipelineFile(pipelineFilePath string, cacheDisabled bool) string {
	pipelineSpecsFromFile := testutil.ParseFileToSpecs(pipelineFilePath, cacheDisabled, nil)
	newPipelineFile := testutil.CreateTempFile(pipelineSpecsFromFile.Bytes())
	return newPipelineFile.Name()
}

func uploadAPipeline(pipelineFile string, pipelineName *string) *pipeline_upload_model.V2beta1Pipeline {
	logger.Log("Create a pipeline")
	testContext.Pipeline.UploadParams.SetName(pipelineName)
	logger.Log("Uploading pipeline with name=%s, from file %s", *pipelineName, pipelineFile)
	createdPipeline, uploadErr := testutil.UploadPipeline(pipelineUploadClient, pipelineFile, pipelineName, nil)
	Expect(uploadErr).NotTo(HaveOccurred(), "Failed to upload pipeline")
	testContext.Pipeline.CreatedPipelines = append(testContext.Pipeline.CreatedPipelines, createdPipeline)
	return createdPipeline
}

func createExperiment(experimentName string) *experiment_model.V2beta1Experiment {
	createdExperiment := testutil.CreateExperimentWithParams(experimentClient, &experiment_model.V2beta1Experiment{
		DisplayName: experimentName,
		Namespace:   testutil.GetNamespace(),
	})
	testContext.Experiment.CreatedExperimentIds = append(testContext.Experiment.CreatedExperimentIds, createdExperiment.ExperimentID)
	return createdExperiment
}

func createdExpectedRunAndVerify(createdPipelineRun *run_model.V2beta1Run, pipelineID *string, pipelineVersionID *string, experimentID *string, pipelineInputMap map[string]interface{}) {
	expectedPipelineRun := createExpectedPipelineRun(pipelineID, pipelineVersionID, experimentID, pipelineInputMap, false)
	matcher.MatchPipelineRuns(createdPipelineRun, expectedPipelineRun)
	createdPipelineRunFromDB, createRunError := runClient.Get(&runparams.RunServiceGetRunParams{
		RunID: createdPipelineRun.RunID,
	})
	Expect(createRunError).NotTo(HaveOccurred(), "Failed to get run with Id="+createdPipelineRun.RunID)

	// Making the fields that can be different but we don't care about equal to stabilize tests
	matcher.MatchPipelineRuns(createdPipelineRun, createdPipelineRunFromDB)
}

func createPipelineRun(pipelineID *string, pipelineVersionID *string, experimentID *string, inputParams map[string]interface{}) *run_model.V2beta1Run {
	logger.Log("Create a pipeline run for pipeline with id=%s and versionId=%s", *pipelineID, *pipelineVersionID)
	createRunRequest := &runparams.RunServiceCreateRunParams{Run: createPipelineRunPayload(pipelineID, pipelineVersionID, experimentID, inputParams)}
	createdRun, createRunError := runClient.Create(createRunRequest)
	Expect(createRunError).NotTo(HaveOccurred(), "Failed to create run for pipeline with id="+*pipelineID)
	testContext.PipelineRun.CreatedRunIds = append(testContext.PipelineRun.CreatedRunIds, createdRun.RunID)
	logger.Log("Created Pipeline Run successfully with runId=%s", createdRun.RunID)
	return createdRun
}

func archivePipelineRun(pipelineRunID *string) {
	logger.Log("Archiving a pipeline run with id=%s", *pipelineRunID)
	archiveRunParams := &runparams.RunServiceArchiveRunParams{
		RunID: *pipelineRunID,
	}
	archiveRunError := runClient.Archive(archiveRunParams)
	Expect(archiveRunError).NotTo(HaveOccurred(), "Failed to archive run with id="+*pipelineRunID)
	logger.Log("Successfully archived run with runId=%s", *pipelineRunID)
}

func unArchivePipelineRun(pipelineRunID *string) {
	logger.Log("Unarchiving a pipeline run with id=%s", *pipelineRunID)
	unArchiveRunParams := &runparams.RunServiceUnarchiveRunParams{
		RunID: *pipelineRunID,
	}
	unarchiveRunError := runClient.Unarchive(unArchiveRunParams)
	Expect(unarchiveRunError).NotTo(HaveOccurred(), "Failed to un-archive run with id="+*pipelineRunID)
	logger.Log("Successfully unarchived run with runId=%s", *pipelineRunID)
}

func createPipelineRunPayload(pipelineID *string, pipelineVersionID *string, experimentID *string, inputParams map[string]interface{}) *run_model.V2beta1Run {
	logger.Log("Create a pipeline run body")
	return &run_model.V2beta1Run{
		DisplayName:    runName,
		Description:    runDescription,
		ExperimentID:   testutil.ParsePointersToString(experimentID),
		ServiceAccount: testutil.GetDefaultPipelineRunnerServiceAccount(),
		PipelineVersionReference: &run_model.V2beta1PipelineVersionReference{
			PipelineID:        testutil.ParsePointersToString(pipelineID),
			PipelineVersionID: testutil.ParsePointersToString(pipelineVersionID),
		},
		RuntimeConfig: &run_model.V2beta1RuntimeConfig{
			Parameters: inputParams,
		},
	}
}

func createExpectedPipelineRun(pipelineID *string, pipelineVersionID *string, experimentID *string, inputParams map[string]interface{}, archived bool) *run_model.V2beta1Run {
	expectedRun := createPipelineRunPayload(pipelineID, pipelineVersionID, experimentID, inputParams)
	storageState := run_model.V2beta1RunStorageStateAVAILABLE
	if archived {
		storageState = run_model.V2beta1RunStorageStateARCHIVED
	}
	expectedRun.StorageState = &storageState
	if experimentID == nil {
		logger.Log("Fetch default experiment's experimentId")
		pageSize := int32(1000)
		experminents, expError := experimentClient.ListAll(&experimentparams.ExperimentServiceListExperimentsParams{
			Namespace: config.Namespace,
			PageSize:  &pageSize,
		}, 1000)
		Expect(expError).NotTo(HaveOccurred(), "Failed to list experiments")
		for _, experiment := range experminents {
			if strings.ToLower(experiment.DisplayName) == "default" {
				expectedRun.ExperimentID = experiment.ExperimentID
			}
		}
	}
	return expectedRun
}
