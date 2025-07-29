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
	"os"
	"path/filepath"
	"strings"

	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/experiment_model"
	"sigs.k8s.io/yaml"

	experiment_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/experiment_client/experiment_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_model"
	run_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_client/run_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_model"
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

var err error
var experimentName string
var runName string
var runDescription string

// ####################################################################################################################################################################
// ################################################################### SETUP AND TEARDOWN ################################################################################
// ####################################################################################################################################################################

var _ = BeforeEach(func() {
	logger.Log("Setting up Pipeline Run Tests")
	runName = "API Test Run - " + randomName
	runDescription = "API Test Run"
	experimentName = "API Test Experiment - " + randomName
})

// ####################################################################################################################################################################
// ################################################################### TESTS ################################################################################
// ####################################################################################################################################################################

// ################################################################################################################
// ########################################################## POSITIVE TESTS ######################################
// ################################################################################################################

var _ = Describe("Verify Pipeline Run >", Label("Positive", "PipelineRun", FullRegression), func() {

	type TestParams struct {
		pipelineCacheEnabled bool
	}

	testParams := []TestParams{
		{pipelineCacheEnabled: true},
		{pipelineCacheEnabled: false},
	}
	pipelineDirectory := "valid"
	pipelineFiles := utils.GetListOfFileInADir(filepath.Join(pipelineFilesRootDir, pipelineDirectory))

	Context("Create a valid pipeline and verify the created run >", func() {
		for _, param := range testParams {
			for _, pipelineFile := range pipelineFiles {
				It(fmt.Sprintf("Create a '%s' pipeline with cacheEnabled=%t and verify run", pipelineFile, param.pipelineCacheEnabled), func() {
					configuredPipelineSpecFile := configureCacheSettingAndGetPipelineFile(pipelineDirectory, pipelineFile, param.pipelineCacheEnabled)
					createdPipeline := uploadAPipeline(configuredPipelineSpecFile, &pipelineGeneratedName)
					createdPipelineVersion := utils.GetLatestPipelineVersion(pipelineClient, &createdPipeline.PipelineID)
					pipelineRuntimeInputs := utils.GetPipelineRunTimeInputs(configuredPipelineSpecFile)
					createdPipelineRun := createPipelineRun(&createdPipeline.PipelineID, &createdPipelineVersion.PipelineVersionID, nil, pipelineRuntimeInputs)
					createdExpectedRunAndVerify(createdPipelineRun, &createdPipeline.PipelineID, &createdPipelineVersion.PipelineVersionID, nil, pipelineRuntimeInputs)
				})
			}
		}

		It(fmt.Sprintf("Create a '%s' pipeline, create an experiement and verify run with associated experiment", pipelineFiles[0]), Label(Smoke), func() {
			pipelineFile := filepath.Join(pipelineFilesRootDir, pipelineDirectory, pipelineFiles[0])
			createdExperiment := createExperiment(experimentName)
			createdPipeline := uploadAPipeline(pipelineFile, &pipelineGeneratedName)
			createdPipelineVersion := utils.GetLatestPipelineVersion(pipelineClient, &createdPipeline.PipelineID)
			pipelineRuntimeInputs := utils.GetPipelineRunTimeInputs(pipelineFile)
			createdPipelineRun := createPipelineRun(&createdPipeline.PipelineID, &createdPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, pipelineRuntimeInputs)
			createdExpectedRunAndVerify(createdPipelineRun, &createdPipeline.PipelineID, &createdPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, pipelineRuntimeInputs)
		})
	})

	Context("Associate a single experiment with multiple pipeline runs >", func() {
		pipelineFile := filepath.Join(pipelineFilesRootDir, pipelineDirectory, pipelineFiles[0])
		It("Create an experiment and associate it multiple pipeline runs of the same pipeline", func() {
			createdExperiment := createExperiment(experimentName)
			createdPipeline := uploadAPipeline(pipelineFile, &pipelineGeneratedName)
			createdPipelineVersion := utils.GetLatestPipelineVersion(pipelineClient, &createdPipeline.PipelineID)
			pipelineRuntimeInputs := utils.GetPipelineRunTimeInputs(pipelineFile)
			createdPipelineRun1 := createPipelineRun(&createdPipeline.PipelineID, &createdPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, pipelineRuntimeInputs)
			createdExpectedRunAndVerify(createdPipelineRun1, &createdPipeline.PipelineID, &createdPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, pipelineRuntimeInputs)

			createdPipelineRun2 := createPipelineRun(&createdPipeline.PipelineID, &createdPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, pipelineRuntimeInputs)
			createdExpectedRunAndVerify(createdPipelineRun2, &createdPipeline.PipelineID, &createdPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, pipelineRuntimeInputs)
		})
		It("Create an experiment and associate it pipeline runs of different pipelines", func() {

			createdExperiment := createExperiment(experimentName)
			createdPipeline1 := uploadAPipeline(pipelineFile, &pipelineGeneratedName)
			createdPipeline1Version := utils.GetLatestPipelineVersion(pipelineClient, &createdPipeline1.PipelineID)
			pipeline2Name := pipelineGeneratedName + "2"
			createdPipeline2 := uploadAPipeline(pipelineFile, &pipeline2Name)
			createdPipeline2Version := utils.GetLatestPipelineVersion(pipelineClient, &createdPipeline2.PipelineID)
			pipelineRuntimeInputs := utils.GetPipelineRunTimeInputs(pipelineFile)
			createdPipelineRun1 := createPipelineRun(&createdPipeline1.PipelineID, &createdPipeline1Version.PipelineVersionID, &createdExperiment.ExperimentID, pipelineRuntimeInputs)
			createdExpectedRunAndVerify(createdPipelineRun1, &createdPipeline1.PipelineID, &createdPipeline1Version.PipelineVersionID, &createdExperiment.ExperimentID, pipelineRuntimeInputs)

			createdPipelineRun2 := createPipelineRun(&createdPipeline2.PipelineID, &createdPipeline2Version.PipelineVersionID, &createdExperiment.ExperimentID, pipelineRuntimeInputs)
			createdExpectedRunAndVerify(createdPipelineRun2, &createdPipeline2.PipelineID, &createdPipeline2Version.PipelineVersionID, &createdExperiment.ExperimentID, pipelineRuntimeInputs)
		})
	})

	Context("Create a pipeline run with HTTP proxy >", func() {
		pipelineFile := filepath.Join(pipelineFilesRootDir, pipelineDirectory, "env-var.yaml")
		It(fmt.Sprintf("Create a pipeline run with http proxy, using specs: %s", pipelineFile), func() {
			Skip("To be implemented, but also need to know what this is suppose to do")
		})
	})

	Context("Archive pipeline run(s) >", func() {
		pipelineFile := filepath.Join(pipelineFilesRootDir, pipelineDirectory, pipelineFiles[0])
		It("Create a pipeline run, archive it and verify that the run state does not change on archiving", func() {
			createdExperiment := createExperiment(experimentName)
			createdPipeline := uploadAPipeline(pipelineFile, &pipelineGeneratedName)
			createdPipelineVersion := utils.GetLatestPipelineVersion(pipelineClient, &createdPipeline.PipelineID)
			pipelineRuntimeInputs := utils.GetPipelineRunTimeInputs(pipelineFile)
			createdPipelineRun := createPipelineRun(&createdPipeline.PipelineID, &createdPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, pipelineRuntimeInputs)
			archivePipelineRun(&createdPipelineRun.RunID)
			pipelineRunAfterArchive := utils.GetPipelineRun(runClient, &createdPipelineRun.RunID)
			Expect(createdPipelineRun.State).To(Equal(pipelineRunAfterArchive.State))
			Expect(pipelineRunAfterArchive.StorageState).To(Equal(run_model.V2beta1RunStorageStateARCHIVED))

		})

		It("Create a pipeline run, wait for the run to move to RUNNING, archive it and verify that the run state is still RUNNING on archiving", func() {
			createdExperiment := createExperiment(experimentName)
			createdPipeline := uploadAPipeline(pipelineFile, &pipelineGeneratedName)
			createdPipelineVersion := utils.GetLatestPipelineVersion(pipelineClient, &createdPipeline.PipelineID)
			pipelineRuntimeInputs := utils.GetPipelineRunTimeInputs(pipelineFile)
			createdPipelineRun := createPipelineRun(&createdPipeline.PipelineID, &createdPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, pipelineRuntimeInputs)
			utils.WaitForRunToBeInState(runClient, &createdPipelineRun.RunID, []run_model.V2beta1RuntimeState{run_model.V2beta1RuntimeStateRUNNING}, nil)
			archivePipelineRun(&createdPipelineRun.RunID)
			pipelineRunAfterArchive := utils.GetPipelineRun(runClient, &createdPipelineRun.RunID)
			Expect(pipelineRunAfterArchive.State).To(Equal(run_model.V2beta1RuntimeStateRUNNING))
			Expect(pipelineRunAfterArchive.StorageState).To(Equal(run_model.V2beta1RunStorageStateARCHIVED))

		})
	})

	Context("Unarchive pipeline run(s) >", func() {
		pipelineFile := filepath.Join(pipelineFilesRootDir, pipelineDirectory, pipelineFiles[0])
		It("Create a pipeline run, archive it and unarchive it and verify the storage state", func() {
			createdExperiment := createExperiment(experimentName)
			createdPipeline := uploadAPipeline(pipelineFile, &pipelineGeneratedName)
			createdPipelineVersion := utils.GetLatestPipelineVersion(pipelineClient, &createdPipeline.PipelineID)
			pipelineRuntimeInputs := utils.GetPipelineRunTimeInputs(pipelineFile)
			createdPipelineRun := createPipelineRun(&createdPipeline.PipelineID, &createdPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, pipelineRuntimeInputs)
			archivePipelineRun(&createdPipelineRun.RunID)
			unArchivePipelineRun(&createdPipelineRun.RunID)
			pipelineRunAfterUnArchive := utils.GetPipelineRun(runClient, &createdPipelineRun.RunID)
			Expect(pipelineRunAfterUnArchive.StorageState).To(Equal(run_model.V2beta1RunStorageStateAVAILABLE))
		})
	})

	Context("Terminate a pipeline run >", func() {
		It("Terminate a run in RUNNING state", func() {
		})
		It("Terminate a run in PENDING state", func() {
		})
		It("Terminate a run in SUCCESSFUL or ERRORED state", func() {
		})
	})

	Context("Get All pipeline run >", func() {
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
		It("List pipeline Runs by specifying page size and iterate over atleast 2 pages", func() {
		})
		It("List pipeline Runs filtering by run id", func() {
		})
		It("List pipeline Runs filtering by display name containing", func() {
		})
		It("List pipeline Runs filtering by creation date", func() {
		})
		It("List pipeline Runs by experiment id", func() {
		})
		It("List pipeline Runs by namespace", func() {
		})
	})
})

// ################################################################################################################
// ########################################################## NEGATIVE TESTS ######################################
// ################################################################################################################
var _ = Describe("Verify Pipeline Run Negative Tests >", Label("Negative", "PipelineRun", FullRegression), func() {
	Context("Unarchive a pipeline run >", func() {
		It("Unarchive a deleted run", func() {
		})
		It("Unarchive a non existent run", func() {
		})
		It("Unarchive an available run", func() {
		})
		if *isKubeflowMode {
			It("In a namespace you don;t have access to", func() {
			})
		}
	})
	Context("Archive a pipeline run >", func() {
		It("Archive a deleted run", func() {
		})
		It("Archive a non existent run", func() {
		})
		It("Archive an already archived run", func() {
		})
		if *isKubeflowMode {
			It("In a namespace you don;t have access to", func() {
			})
		}
	})
	Context("Terminate a pipeline run >", func() {
		It("Terminate a deleted run", func() {
		})
		It("Terminate a non existent run", func() {
		})
		It("Terminate an already terminated run", func() {
		})
		if *isKubeflowMode {
			It("In a namespace you don;t have access to", func() {
			})
		}
	})
	Context("Delete a pipeline run >", func() {
		It("Delete a deleted run", func() {
		})
		It("Delete a non existent run", func() {
		})
	})
	Context("Associate a pipeline run with invalid experiment >", func() {
		It("Associate a run with an archived experiment", func() {
		})
		It("Associate a run with non existent experiment", func() {
		})
		It("Associate a run with deleted experiment", func() {
		})
	})
})

// ####################################################################################################################################################################
// ################################################################### UTILITY METHODS ################################################################################
// ####################################################################################################################################################################

func configureCacheSettingAndGetPipelineFile(pipelineDirectory string, pipelineFileName string, cacheEnabled bool) string {
	pipelineSpecsFromFile := utils.PipelineSpecFromFile(pipelineFilesRootDir, pipelineDirectory, pipelineFileName)
	var marshalledPipelineSpecs []byte
	var marshalledPlatformSpecs []byte
	var marshallErr error
	var newPipelineFile *os.File
	if _, platformSpecExists := pipelineSpecsFromFile["platform_spec"]; platformSpecExists {
		pipelineSpecs := pipelineSpecsFromFile["pipeline_spec"].(map[string]interface{})
		pipelineSpecs = *configurePipelineCacheSettings(&pipelineSpecs, cacheEnabled)
		pipelineSpecsFromFile["pipeline_spec"] = pipelineSpecs
		marshalledPipelineSpecs, marshallErr = yaml.Marshal(pipelineSpecsFromFile["pipeline_spec"])
		Expect(marshallErr).NotTo(HaveOccurred())
		marshalledPlatformSpecs, marshallErr = yaml.Marshal(pipelineSpecsFromFile["platform_spec"])
		Expect(marshallErr).NotTo(HaveOccurred())
		newPipelineFile = utils.CreateTempFile([][]byte{marshalledPipelineSpecs, []byte("\n---\n"), marshalledPlatformSpecs})
	} else {
		pipelineSpecsFromFile = *configurePipelineCacheSettings(&pipelineSpecsFromFile, cacheEnabled)
		marshalledPipelineSpecs, marshallErr = yaml.Marshal(pipelineSpecsFromFile)
		Expect(marshallErr).NotTo(HaveOccurred(), "Failed to marshall pipeline spec")
		newPipelineFile = utils.CreateTempFile([][]byte{marshalledPipelineSpecs})
	}
	return newPipelineFile.Name()
}

func uploadAPipeline(pipelineFile string, pipelineName *string) *pipeline_upload_model.V2beta1Pipeline {
	logger.Log("Create a pipeline")
	pipelineUploadParams.SetName(pipelineName)
	logger.Log("Uploading pipeline with name=%s, from file %s", *pipelineName, pipelineFile)
	createdPipeline, uploadErr := pipelineUploadClient.UploadFile(pipelineFile, pipelineUploadParams)
	Expect(uploadErr).NotTo(HaveOccurred(), "Failed to upload pipeline")
	createdPipelines = append(createdPipelines, createdPipeline)
	return createdPipeline
}

func createExperiment(experimentName string) *experiment_model.V2beta1Experiment {
	logger.Log("Create an experiment with name %s", experimentName)
	createExperimentParams := experiment_params.NewExperimentServiceCreateExperimentParams()
	createExperimentParams.Body = &experiment_model.V2beta1Experiment{
		DisplayName: experimentName,
		Namespace:   *namespace,
	}
	createdExperiment, experimentErr := experimentClient.Create(createExperimentParams)
	Expect(experimentErr).NotTo(HaveOccurred(), "Failed to create experiment")
	createdExperimentIds = append(createdExperimentIds, createdExperiment.ExperimentID)
	return createdExperiment
}

func createdExpectedRunAndVerify(createdPipelineRun *run_model.V2beta1Run, pipelineID *string, pipelineVersionID *string, experimentID *string, pipelineInputMap map[string]interface{}) {
	expectedPipelineRun := createExpectedPipelineRun(pipelineID, pipelineVersionID, experimentID, pipelineInputMap, false)
	matcher.MatchPipelineRuns(createdPipelineRun, expectedPipelineRun)
	createdPipelineRunFromDB, createRunError := runClient.Get(&run_params.RunServiceGetRunParams{
		RunID: createdPipelineRun.RunID,
	})
	Expect(createRunError).NotTo(HaveOccurred(), "Failed to get run with Id="+createdPipelineRun.RunID)

	// Making the fields that can be different but we don't care about equal to stabilize tests
	matcher.MatchPipelineRuns(createdPipelineRun, createdPipelineRunFromDB)
}

func createPipelineRun(pipelineID *string, pipelineVersionID *string, experimentID *string, inputParams map[string]interface{}) *run_model.V2beta1Run {
	logger.Log("Create a pipeline run for pipeline with id=%s and versionId=%s", pipelineID, pipelineVersionID)
	createRunRequest := &run_params.RunServiceCreateRunParams{Body: createPipelineRunPayload(pipelineID, pipelineVersionID, experimentID, inputParams)}
	createdRun, createRunError := runClient.Create(createRunRequest)
	Expect(createRunError).NotTo(HaveOccurred(), "Failed to create run for pipeline with id="+*pipelineID)
	createdRunIds = append(createdRunIds, createdRun.RunID)
	logger.Log("Created Pipeline Run successfully with runId=%s", createdRun.RunID)
	return createdRun
}

func archivePipelineRun(pipelineRunID *string) {
	logger.Log("Archiving a pipeline run with id=%s and versionId=%s", pipelineRunID)
	archiveRunParams := &run_params.RunServiceArchiveRunParams{
		RunID: *pipelineRunID,
	}
	archiveRunError := runClient.Archive(archiveRunParams)
	Expect(archiveRunError).NotTo(HaveOccurred(), "Failed to archive run with id="+*pipelineRunID)
	logger.Log("Successfully archived run with runId=%s", pipelineRunID)
}

func unArchivePipelineRun(pipelineRunID *string) {
	logger.Log("Unarchiving a pipeline run with id=%s and versionId=%s", pipelineRunID)
	unArchiveRunParams := &run_params.RunServiceUnarchiveRunParams{
		RunID: *pipelineRunID,
	}
	unarchiveRunError := runClient.Unarchive(unArchiveRunParams)
	Expect(unarchiveRunError).NotTo(HaveOccurred(), "Failed to un-archive run with id="+*pipelineRunID)
	logger.Log("Successfully unarchived run with runId=%s", pipelineRunID)
}

func createPipelineRunPayload(pipelineID *string, pipelineVersionID *string, experimentID *string, inputParams map[string]interface{}) *run_model.V2beta1Run {
	logger.Log("Create a pipeline run body")
	return &run_model.V2beta1Run{
		DisplayName:    runName,
		Description:    runDescription,
		ExperimentID:   utils.ParsePointersToString(experimentID),
		ServiceAccount: utils.GetDefaultPipelineRunnerServiceAccount(*isKubeflowMode),
		PipelineVersionReference: &run_model.V2beta1PipelineVersionReference{
			PipelineID:        utils.ParsePointersToString(pipelineID),
			PipelineVersionID: utils.ParsePointersToString(pipelineVersionID),
		},
		RuntimeConfig: &run_model.V2beta1RuntimeConfig{
			Parameters: inputParams,
		},
	}
}

func createExpectedPipelineRun(pipelineID *string, pipelineVersionID *string, experimentID *string, inputParams map[string]interface{}, archived bool) *run_model.V2beta1Run {
	expectedRun := createPipelineRunPayload(pipelineID, pipelineVersionID, experimentID, inputParams)
	if !archived {
		expectedRun.StorageState = run_model.V2beta1RunStorageStateAVAILABLE
	} else {
		expectedRun.StorageState = run_model.V2beta1RunStorageStateARCHIVED
	}
	if experimentID == nil {
		logger.Log("Fetch default experiment's experimentId")
		pageSize := int32(1000)
		experminents, expError := experimentClient.ListAll(&experiment_params.ExperimentServiceListExperimentsParams{
			Namespace: namespace,
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

func configurePipelineCacheSettings(pipelineSpec *map[string]interface{}, cacheEnabled bool) *map[string]interface{} {
	if pipelineTasks, pipelineTasksExists := (*pipelineSpec)["root"].(map[string]interface{})["dag"].(map[string]interface{})["tasks"]; pipelineTasksExists {
		for _, pipelineTask := range pipelineTasks.(map[string]interface{}) {
			task := pipelineTask.(map[string]interface{})
			cachingOptionMap := make(map[string]interface{})
			if cacheEnabled {
				cachingOptionMap["enableCache"] = true
			}
			task["cachingOptions"] = cachingOptionMap
		}
		return pipelineSpec
	} else {
		return nil
	}
}

// DO NOT DELETE - When we have the logic to create pending tasks without AWC, we will use the following code
// func waitForTasksAndGetRunDetails(runID string, numberOfExpectedTasks int) *run_model.V2beta1Run {
//	createdPipelineRunFromDB, _ := runClient.Get(&run_params.RunServiceGetRunParams{
//		RunID: runID,
//	})
//	timeout := time.After(30 * time.Second)
//	for createdPipelineRunFromDB.RunDetails == nil {
//		time.Sleep(1 * time.Second)
//		createdPipelineRunFromDB, _ = runClient.Get(&run_params.RunServiceGetRunParams{
//			RunID: runID,
//		})
//		select {
//		case <-timeout:
//			Fail("Timeout waiting for run details to be available for runId=" + runID)
//		default:
//			if createdPipelineRunFromDB.RunDetails != nil {
//				if len(createdPipelineRunFromDB.RunDetails.TaskDetails) < numberOfExpectedTasks {
//					logger.Log("Not all tasks for the run %s have been generated, tasks=%d/%d", runID, len(createdPipelineRunFromDB.RunDetails.TaskDetails), numberOfExpectedTasks)
//					createdPipelineRunFromDB.RunDetails = nil
//					continue
//				} else {
//					logger.Log("All %d/%d tasks for the run %s have been generated", len(createdPipelineRunFromDB.RunDetails.TaskDetails), numberOfExpectedTasks, runID)
//				}
//			}
//		}
//	}
//	return createdPipelineRunFromDB
//}

// DO NOT DELETE - When we have the logic to create pending tasks without AWC, we will use the following code
// func validatePipelineRunDetails(inputPipelineSpec interface{}, runID string) {
//	expectedPipelineRunDetails := utils.ToRunDetailsFromPipelineSpec(inputPipelineSpec, runID)
//	createdPipelineRunFromDB := waitForTasksAndGetRunDetails(runID, len(expectedPipelineRunDetails.TaskDetails))
//	matcher.MatchPipelineRunDetails(createdPipelineRunFromDB.RunDetails, expectedPipelineRunDetails)
//}
