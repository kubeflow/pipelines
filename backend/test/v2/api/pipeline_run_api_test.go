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
	"time"

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

	// getFirstYamlFile returns the first .yaml file from the list of pipeline files
	getFirstYamlFile := func(filePaths []string) string {
		for _, path := range filePaths {
			if strings.HasSuffix(path, ".yaml") {
				return path
			}
		}
		panic("No .yaml file found in pipeline files")
	}

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
		pipelineFile := getFirstYamlFile(pipelineFilePaths)
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
		pipelineFile := getFirstYamlFile(pipelineFilePaths)
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
		pipelineFile := filepath.Join(pipelineFilesRootDir, pipelineDirectory, "hello_world.yaml")
		It("Create a pipeline run, wait for the run state to be reported, archive it and verify that archiving keeps the runtime state reported", func() {
			createdExperiment := createExperiment(experimentName)
			createdPipeline := uploadAPipeline(pipelineFile, &testContext.Pipeline.PipelineGeneratedName)
			createdPipelineVersion := testutil.GetLatestPipelineVersion(pipelineClient, &createdPipeline.PipelineID)
			pipelineRuntimeInputs := testutil.GetPipelineRunTimeInputs(pipelineFile)
			createdPipelineRun := createPipelineRun(&createdPipeline.PipelineID, &createdPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, pipelineRuntimeInputs)
			invalidRuntimeStates := []run_model.V2beta1RuntimeState{
				run_model.V2beta1RuntimeStateCANCELED,
				run_model.V2beta1RuntimeStateCANCELING,
				run_model.V2beta1RuntimeStateRUNTIMESTATEUNSPECIFIED,
			}
			var pipelineRunBeforeArchive *run_model.V2beta1Run
			Eventually(func() *run_model.V2beta1Run {
				pipelineRunBeforeArchive = testutil.GetPipelineRun(runClient, &createdPipelineRun.RunID)
				if pipelineRunBeforeArchive.State == nil {
					return nil
				}
				return pipelineRunBeforeArchive
			}, "30s", "1s").ShouldNot(BeNil(), "Expected pipeline run state to be reported before archiving")
			Expect(*pipelineRunBeforeArchive.State).To(Not(BeElementOf(invalidRuntimeStates)))
			archivePipelineRun(&createdPipelineRun.RunID)
			var pipelineRunAfterArchive *run_model.V2beta1Run
			Eventually(func() *run_model.V2beta1Run {
				pipelineRunAfterArchive = testutil.GetPipelineRun(runClient, &createdPipelineRun.RunID)
				if pipelineRunAfterArchive.State == nil || pipelineRunAfterArchive.StorageState == nil || *pipelineRunAfterArchive.StorageState != run_model.V2beta1RunStorageStateARCHIVED {
					return nil
				}
				return pipelineRunAfterArchive
			}, "30s", "1s").ShouldNot(BeNil(), "Expected archived pipeline run to keep a reported runtime state")
			Expect(*pipelineRunAfterArchive.State).To(Not(BeElementOf(invalidRuntimeStates)))
			Expect(*pipelineRunAfterArchive.StorageState).To(Equal(run_model.V2beta1RunStorageStateARCHIVED))

		})

		It("Create a pipeline run, wait for the run to move to RUNNING or PENDING, archive it and verify that the run state is still RUNNING or PENDING on archiving", func() {
			createdExperiment := createExperiment(experimentName)
			createdPipeline := uploadAPipeline(pipelineFile, &testContext.Pipeline.PipelineGeneratedName)
			createdPipelineVersion := testutil.GetLatestPipelineVersion(pipelineClient, &createdPipeline.PipelineID)
			pipelineRuntimeInputs := testutil.GetPipelineRunTimeInputs(pipelineFile)
			createdPipelineRun := createPipelineRun(&createdPipeline.PipelineID, &createdPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, pipelineRuntimeInputs)
			testutil.WaitForRunToBeInState(runClient, &createdPipelineRun.RunID, []run_model.V2beta1RuntimeState{run_model.V2beta1RuntimeStateRUNNING, run_model.V2beta1RuntimeStatePENDING}, nil)
			archivePipelineRun(&createdPipelineRun.RunID)
			pipelineRunAfterArchive := testutil.GetPipelineRun(runClient, &createdPipelineRun.RunID)
			Expect(*pipelineRunAfterArchive.State).To(BeElementOf([]run_model.V2beta1RuntimeState{run_model.V2beta1RuntimeStateRUNNING, run_model.V2beta1RuntimeStatePENDING}))
			Expect(*pipelineRunAfterArchive.StorageState).To(Equal(run_model.V2beta1RunStorageStateARCHIVED))

		})
	})

	Context("Unarchive pipeline run(s) >", func() {
		pipelineFile := filepath.Join(pipelineFilesRootDir, pipelineDirectory, "hello_world.yaml")
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

	Context("Get All pipeline run >", func() {
		var listPipelineFile string
		var listCreatedPipeline *pipeline_upload_model.V2beta1Pipeline
		var listCreatedPipelineVersion *pipeline_model.V2beta1PipelineVersion
		var listPipelineRuntimeInputs map[string]interface{}

		BeforeEach(func() {
			listPipelineFile = filepath.Join(testutil.GetValidPipelineFilesDir(), helloWorldPipelineFileName)
			listCreatedPipeline = uploadAPipeline(listPipelineFile, &testContext.Pipeline.PipelineGeneratedName)
			listCreatedPipelineVersion = testutil.GetLatestPipelineVersion(pipelineClient, &listCreatedPipeline.PipelineID)
			listPipelineRuntimeInputs = testutil.GetPipelineRunTimeInputs(listPipelineFile)
		})

		It("Create a Pipeline Run and validate that it gets returned in the List Runs API call", func() {
			createdExperiment := createExperiment(experimentName)
			createdRun := createPipelineRun(&listCreatedPipeline.PipelineID, &listCreatedPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, listPipelineRuntimeInputs)

			runs, err := runClient.ListAll(&runparams.RunServiceListRunsParams{
				ExperimentID: &createdExperiment.ExperimentID,
			}, 1000)
			Expect(err).NotTo(HaveOccurred(), "Failed to list runs")
			var found bool
			for _, r := range runs {
				if r.RunID == createdRun.RunID {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue(), "Created run not found in list")
		})

		It("Create 2 pipeline Runs, and list it", func() {
			createdExperiment := createExperiment(experimentName)
			createdRun1 := createPipelineRun(&listCreatedPipeline.PipelineID, &listCreatedPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, listPipelineRuntimeInputs)
			createdRun2 := createPipelineRun(&listCreatedPipeline.PipelineID, &listCreatedPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, listPipelineRuntimeInputs)

			runs, err := runClient.ListAll(&runparams.RunServiceListRunsParams{
				ExperimentID: &createdExperiment.ExperimentID,
			}, 1000)
			Expect(err).NotTo(HaveOccurred(), "Failed to list runs")
			Expect(len(runs)).To(BeNumerically(">=", 2))
			runIDs := make(map[string]bool)
			for _, r := range runs {
				runIDs[r.RunID] = true
			}
			Expect(runIDs[createdRun1.RunID]).To(BeTrue(), "Run 1 not found in list")
			Expect(runIDs[createdRun2.RunID]).To(BeTrue(), "Run 2 not found in list")
		})

		It("List pipeline Runs when no runs exist", func() {
			isolatedExperiment := createExperiment(experimentName + "-isolated")
			runs, err := runClient.ListAll(&runparams.RunServiceListRunsParams{
				ExperimentID: &isolatedExperiment.ExperimentID,
			}, 1000)
			Expect(err).NotTo(HaveOccurred(), "Failed to list runs")
			Expect(runs).To(BeEmpty(), "Expected no runs for isolated experiment")
		})

		It("List pipeline Runs and sort by display name in ascending order", func() {
			createdExperiment := createExperiment(experimentName)
			createPipelineRun(&listCreatedPipeline.PipelineID, &listCreatedPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, listPipelineRuntimeInputs)
			createPipelineRun(&listCreatedPipeline.PipelineID, &listCreatedPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, listPipelineRuntimeInputs)

			sortBy := "display_name asc"
			runs, err := runClient.ListAll(&runparams.RunServiceListRunsParams{
				ExperimentID: &createdExperiment.ExperimentID,
				SortBy:       &sortBy,
			}, 1000)
			Expect(err).NotTo(HaveOccurred(), "Failed to list runs sorted by display name asc")
			for i := 1; i < len(runs); i++ {
				Expect(strings.ToLower(runs[i-1].DisplayName) <= strings.ToLower(runs[i].DisplayName)).To(BeTrue(), "Runs not sorted by display_name asc")
			}
		})

		It("List pipeline Runs and sort by display name in descending order", func() {
			createdExperiment := createExperiment(experimentName)
			createPipelineRun(&listCreatedPipeline.PipelineID, &listCreatedPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, listPipelineRuntimeInputs)
			createPipelineRun(&listCreatedPipeline.PipelineID, &listCreatedPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, listPipelineRuntimeInputs)

			sortBy := "display_name desc"
			runs, err := runClient.ListAll(&runparams.RunServiceListRunsParams{
				ExperimentID: &createdExperiment.ExperimentID,
				SortBy:       &sortBy,
			}, 1000)
			Expect(err).NotTo(HaveOccurred(), "Failed to list runs sorted by display name desc")
			for i := 1; i < len(runs); i++ {
				Expect(strings.ToLower(runs[i-1].DisplayName) >= strings.ToLower(runs[i].DisplayName)).To(BeTrue(), "Runs not sorted by display_name desc")
			}
		})

		It("List pipeline Runs and sort by id in ascending order", func() {
			createdExperiment := createExperiment(experimentName)
			createPipelineRun(&listCreatedPipeline.PipelineID, &listCreatedPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, listPipelineRuntimeInputs)
			createPipelineRun(&listCreatedPipeline.PipelineID, &listCreatedPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, listPipelineRuntimeInputs)

			sortBy := "id asc"
			runs, err := runClient.ListAll(&runparams.RunServiceListRunsParams{
				ExperimentID: &createdExperiment.ExperimentID,
				SortBy:       &sortBy,
			}, 1000)
			Expect(err).NotTo(HaveOccurred(), "Failed to list runs sorted by id asc")
			for i := 1; i < len(runs); i++ {
				Expect(runs[i-1].RunID <= runs[i].RunID).To(BeTrue(), "Runs not sorted by id asc")
			}
		})

		It("List pipeline Runs and sort by id in descending order", func() {
			createdExperiment := createExperiment(experimentName)
			createPipelineRun(&listCreatedPipeline.PipelineID, &listCreatedPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, listPipelineRuntimeInputs)
			createPipelineRun(&listCreatedPipeline.PipelineID, &listCreatedPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, listPipelineRuntimeInputs)

			sortBy := "id desc"
			runs, err := runClient.ListAll(&runparams.RunServiceListRunsParams{
				ExperimentID: &createdExperiment.ExperimentID,
				SortBy:       &sortBy,
			}, 1000)
			Expect(err).NotTo(HaveOccurred(), "Failed to list runs sorted by id desc")
			for i := 1; i < len(runs); i++ {
				Expect(runs[i-1].RunID >= runs[i].RunID).To(BeTrue(), "Runs not sorted by id desc")
			}
		})

		It("List pipeline Runs and sort by scheduled_at in ascending order", func() {
			createdExperiment := createExperiment(experimentName)
			createPipelineRun(&listCreatedPipeline.PipelineID, &listCreatedPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, listPipelineRuntimeInputs)
			createPipelineRun(&listCreatedPipeline.PipelineID, &listCreatedPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, listPipelineRuntimeInputs)

			sortBy := "scheduled_at asc"
			runs, err := runClient.ListAll(&runparams.RunServiceListRunsParams{
				ExperimentID: &createdExperiment.ExperimentID,
				SortBy:       &sortBy,
			}, 1000)
			Expect(err).NotTo(HaveOccurred(), "Failed to list runs sorted by scheduled_at asc")
			Expect(len(runs)).To(BeNumerically(">=", 1))
		})

		It("List pipeline Runs and sort by scheduled_at in descending order", func() {
			createdExperiment := createExperiment(experimentName)
			createPipelineRun(&listCreatedPipeline.PipelineID, &listCreatedPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, listPipelineRuntimeInputs)
			createPipelineRun(&listCreatedPipeline.PipelineID, &listCreatedPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, listPipelineRuntimeInputs)

			sortBy := "scheduled_at desc"
			runs, err := runClient.ListAll(&runparams.RunServiceListRunsParams{
				ExperimentID: &createdExperiment.ExperimentID,
				SortBy:       &sortBy,
			}, 1000)
			Expect(err).NotTo(HaveOccurred(), "Failed to list runs sorted by scheduled_at desc")
			Expect(len(runs)).To(BeNumerically(">=", 1))
		})

		It("List pipeline Runs and sort by creation date in ascending order", func() {
			createdExperiment := createExperiment(experimentName)
			createPipelineRun(&listCreatedPipeline.PipelineID, &listCreatedPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, listPipelineRuntimeInputs)
			createPipelineRun(&listCreatedPipeline.PipelineID, &listCreatedPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, listPipelineRuntimeInputs)

			sortBy := "created_at asc"
			runs, err := runClient.ListAll(&runparams.RunServiceListRunsParams{
				ExperimentID: &createdExperiment.ExperimentID,
				SortBy:       &sortBy,
			}, 1000)
			Expect(err).NotTo(HaveOccurred(), "Failed to list runs sorted by created_at asc")
			for i := 1; i < len(runs); i++ {
				t1 := time.Time(runs[i-1].CreatedAt)
				t2 := time.Time(runs[i].CreatedAt)
				Expect(t1.Before(t2) || t1.Equal(t2)).To(BeTrue(), "Runs not sorted by created_at asc")
			}
		})

		It("List pipeline Runs and sort by creation date in descending order", func() {
			createdExperiment := createExperiment(experimentName)
			createPipelineRun(&listCreatedPipeline.PipelineID, &listCreatedPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, listPipelineRuntimeInputs)
			createPipelineRun(&listCreatedPipeline.PipelineID, &listCreatedPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, listPipelineRuntimeInputs)

			sortBy := "created_at desc"
			runs, err := runClient.ListAll(&runparams.RunServiceListRunsParams{
				ExperimentID: &createdExperiment.ExperimentID,
				SortBy:       &sortBy,
			}, 1000)
			Expect(err).NotTo(HaveOccurred(), "Failed to list runs sorted by created_at desc")
			for i := 1; i < len(runs); i++ {
				t1 := time.Time(runs[i-1].CreatedAt)
				t2 := time.Time(runs[i].CreatedAt)
				Expect(t1.After(t2) || t1.Equal(t2)).To(BeTrue(), "Runs not sorted by created_at desc")
			}
		})

		It("List pipeline Runs and sort by finished_at in ascending order", func() {
			createdExperiment := createExperiment(experimentName)
			createPipelineRun(&listCreatedPipeline.PipelineID, &listCreatedPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, listPipelineRuntimeInputs)
			createPipelineRun(&listCreatedPipeline.PipelineID, &listCreatedPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, listPipelineRuntimeInputs)

			sortBy := "finished_at asc"
			runs, err := runClient.ListAll(&runparams.RunServiceListRunsParams{
				ExperimentID: &createdExperiment.ExperimentID,
				SortBy:       &sortBy,
			}, 1000)
			Expect(err).NotTo(HaveOccurred(), "Failed to list runs sorted by finished_at asc")
			Expect(len(runs)).To(BeNumerically(">=", 1))
		})

		It("List pipeline Runs and sort by finished_at in descending order", func() {
			createdExperiment := createExperiment(experimentName)
			createPipelineRun(&listCreatedPipeline.PipelineID, &listCreatedPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, listPipelineRuntimeInputs)
			createPipelineRun(&listCreatedPipeline.PipelineID, &listCreatedPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, listPipelineRuntimeInputs)

			sortBy := "finished_at desc"
			runs, err := runClient.ListAll(&runparams.RunServiceListRunsParams{
				ExperimentID: &createdExperiment.ExperimentID,
				SortBy:       &sortBy,
			}, 1000)
			Expect(err).NotTo(HaveOccurred(), "Failed to list runs sorted by finished_at desc")
			Expect(len(runs)).To(BeNumerically(">=", 1))
		})

		It("List pipeline Runs by specifying page size", func() {
			createdExperiment := createExperiment(experimentName)
			createPipelineRun(&listCreatedPipeline.PipelineID, &listCreatedPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, listPipelineRuntimeInputs)
			createPipelineRun(&listCreatedPipeline.PipelineID, &listCreatedPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, listPipelineRuntimeInputs)

			pageSize := int32(1)
			runs, totalSize, nextPageToken, err := runClient.List(&runparams.RunServiceListRunsParams{
				ExperimentID: &createdExperiment.ExperimentID,
				PageSize:     &pageSize,
			})
			Expect(err).NotTo(HaveOccurred(), "Failed to list runs with page size")
			Expect(len(runs)).To(Equal(1))
			Expect(totalSize).To(BeNumerically(">=", 2))
			Expect(nextPageToken).NotTo(BeEmpty(), "Expected nextPageToken to be non-empty")
		})

		It("List pipeline Runs by specifying page size and iterate over at least 2 pages", func() {
			createdExperiment := createExperiment(experimentName)
			createPipelineRun(&listCreatedPipeline.PipelineID, &listCreatedPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, listPipelineRuntimeInputs)
			createPipelineRun(&listCreatedPipeline.PipelineID, &listCreatedPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, listPipelineRuntimeInputs)
			createPipelineRun(&listCreatedPipeline.PipelineID, &listCreatedPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, listPipelineRuntimeInputs)

			pageSize := int32(1)
			pageToken := ""
			pagesVisited := 0
			collectedRuns := make([]*run_model.V2beta1Run, 0)
			for {
				params := &runparams.RunServiceListRunsParams{
					ExperimentID: &createdExperiment.ExperimentID,
					PageSize:     &pageSize,
				}
				if pageToken != "" {
					params.PageToken = &pageToken
				}
				runs, _, nextPageToken, err := runClient.List(params)
				Expect(err).NotTo(HaveOccurred(), "Failed to list runs on page iteration")
				collectedRuns = append(collectedRuns, runs...)
				pagesVisited++
				if nextPageToken == "" {
					break
				}
				pageToken = nextPageToken
			}
			Expect(pagesVisited).To(BeNumerically(">=", 2), "Expected at least 2 pages")
			Expect(len(collectedRuns)).To(BeNumerically(">=", 3), "Expected at least 3 runs collected")
		})

		It("List pipeline Runs filtering by run id", func() {
			createdExperiment := createExperiment(experimentName)
			createdRun := createPipelineRun(&listCreatedPipeline.PipelineID, &listCreatedPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, listPipelineRuntimeInputs)

			filterStr := fmt.Sprintf(`{"predicates":[{"key":"run_id","operation":"EQUALS","string_value":"%s"}]}`, createdRun.RunID)
			runs, totalSize, _, err := runClient.List(&runparams.RunServiceListRunsParams{
				ExperimentID: &createdExperiment.ExperimentID,
				Filter:       &filterStr,
			})
			Expect(err).NotTo(HaveOccurred(), "Failed to list runs filtered by run_id")
			Expect(totalSize).To(Equal(1))
			Expect(len(runs)).To(Equal(1))
			Expect(runs[0].RunID).To(Equal(createdRun.RunID))
		})

		It("List pipeline Runs filtering by display name containing", func() {
			createdExperiment := createExperiment(experimentName)
			createdRun := createPipelineRun(&listCreatedPipeline.PipelineID, &listCreatedPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, listPipelineRuntimeInputs)

			filterStr := fmt.Sprintf(`{"predicates":[{"key":"display_name","operation":"IS_SUBSTRING","string_value":"%s"}]}`, createdRun.DisplayName)
			runs, _, _, err := runClient.List(&runparams.RunServiceListRunsParams{
				ExperimentID: &createdExperiment.ExperimentID,
				Filter:       &filterStr,
			})
			Expect(err).NotTo(HaveOccurred(), "Failed to list runs filtered by display_name IS_SUBSTRING")
			var found bool
			for _, r := range runs {
				if r.RunID == createdRun.RunID {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue(), "Run not found after IS_SUBSTRING filter on display_name")
		})

		It("List pipeline Runs filtering by `name` EQUALS", func() {
			createdExperiment := createExperiment(experimentName)
			createdRun := createPipelineRun(&listCreatedPipeline.PipelineID, &listCreatedPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, listPipelineRuntimeInputs)

			filterStr := fmt.Sprintf(`{"predicates":[{"key":"name","operation":"EQUALS","string_value":"%s"}]}`, createdRun.DisplayName)
			runs, _, _, err := runClient.List(&runparams.RunServiceListRunsParams{
				ExperimentID: &createdExperiment.ExperimentID,
				Filter:       &filterStr,
			})
			Expect(err).NotTo(HaveOccurred(), "Failed to list runs filtered by name EQUALS")
			var found bool
			for _, r := range runs {
				if r.RunID == createdRun.RunID {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue(), "Run not found after EQUALS filter on name")
		})

		It("List pipeline Runs filtering by `name` NOT Equals", func() {
			createdExperiment := createExperiment(experimentName)
			createdRun1 := createPipelineRun(&listCreatedPipeline.PipelineID, &listCreatedPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, listPipelineRuntimeInputs)
			createPipelineRun(&listCreatedPipeline.PipelineID, &listCreatedPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, listPipelineRuntimeInputs)

			filterStr := fmt.Sprintf(`{"predicates":[{"key":"name","operation":"NOT_EQUALS","string_value":"%s"}]}`, createdRun1.DisplayName)
			runs, _, _, err := runClient.List(&runparams.RunServiceListRunsParams{
				ExperimentID: &createdExperiment.ExperimentID,
				Filter:       &filterStr,
			})
			Expect(err).NotTo(HaveOccurred(), "Failed to list runs filtered by name NOT_EQUALS")
			for _, r := range runs {
				Expect(r.RunID).NotTo(Equal(createdRun1.RunID), "Excluded run should not appear in results")
			}
		})

		It("List pipeline Runs filtering by creation date", func() {
			createdExperiment := createExperiment(experimentName)
			createdRun := createPipelineRun(&listCreatedPipeline.PipelineID, &listCreatedPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, listPipelineRuntimeInputs)

			// Use the run's created_at timestamp as the filter boundary (must be Unix seconds as long_value)
			createdAtUnix := time.Time(createdRun.CreatedAt).Unix()
			filterStr := fmt.Sprintf(`{"predicates":[{"key":"created_at","operation":"GREATER_THAN_EQUALS","long_value":%d}]}`, createdAtUnix)
			runs, _, _, err := runClient.List(&runparams.RunServiceListRunsParams{
				ExperimentID: &createdExperiment.ExperimentID,
				Filter:       &filterStr,
			})
			Expect(err).NotTo(HaveOccurred(), "Failed to list runs filtered by created_at")
			Expect(len(runs)).To(BeNumerically(">=", 1), "Expected at least 1 run matching created_at filter")
		})

		It("List pipeline Runs by experiment id", func() {
			createdExperiment1 := createExperiment(experimentName + "-exp1")
			createdExperiment2 := createExperiment(experimentName + "-exp2")
			createdRun1 := createPipelineRun(&listCreatedPipeline.PipelineID, &listCreatedPipelineVersion.PipelineVersionID, &createdExperiment1.ExperimentID, listPipelineRuntimeInputs)
			createPipelineRun(&listCreatedPipeline.PipelineID, &listCreatedPipelineVersion.PipelineVersionID, &createdExperiment2.ExperimentID, listPipelineRuntimeInputs)

			runs, _, _, err := runClient.List(&runparams.RunServiceListRunsParams{
				ExperimentID: &createdExperiment1.ExperimentID,
			})
			Expect(err).NotTo(HaveOccurred(), "Failed to list runs by experiment_id")
			Expect(len(runs)).To(Equal(1), "Expected exactly 1 run for experiment 1")
			Expect(runs[0].RunID).To(Equal(createdRun1.RunID), "Expected run 1 to be returned")
		})

		It("List pipeline Runs by namespace", func() {
			if !*config.KubeflowMode && !*config.MultiUserMode {
				// In non-kubeflow mode, just verify the API call succeeds without error
				ns := *config.Namespace
				runs, _, _, err := runClient.List(&runparams.RunServiceListRunsParams{
					Namespace: &ns,
				})
				Expect(err).NotTo(HaveOccurred(), "Failed to list runs by namespace")
				_ = runs
				return
			}
			ns := testutil.GetNamespace()
			createdExperiment := createExperiment(experimentName)
			createPipelineRun(&listCreatedPipeline.PipelineID, &listCreatedPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, listPipelineRuntimeInputs)

			runs, _, _, err := runClient.List(&runparams.RunServiceListRunsParams{
				Namespace: &ns,
			})
			Expect(err).NotTo(HaveOccurred(), "Failed to list runs by namespace in Kubeflow mode")
			Expect(len(runs)).To(BeNumerically(">=", 1))
		})

		It("List pipeline Runs by namespace with filter AND pagination (FilterByResourceReference placeholder numbering)", func() {
			// This test specifically targets the PostgreSQL placeholder-collision bug in
			// FilterByResourceReference. Filtering by Namespace goes through that code path
			// (unlike ExperimentID which bypasses it), and combining it with a display_name
			// filter + pagination produces multiple args that must be numbered sequentially
			// ($1/$2/$3 for subquery, $4 for filter, $5/$6 for page cursor).
			// On PostgreSQL, a collision causes wrong rows or a driver error.
			ns := testutil.GetNamespace()
			uniquePrefix := "pgx-ns-test-" + randomName
			origRunName := runName
			runName = uniquePrefix + "-run"
			createdExperiment := createExperiment(experimentName)
			createPipelineRun(&listCreatedPipeline.PipelineID, &listCreatedPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, listPipelineRuntimeInputs)
			createPipelineRun(&listCreatedPipeline.PipelineID, &listCreatedPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, listPipelineRuntimeInputs)
			createPipelineRun(&listCreatedPipeline.PipelineID, &listCreatedPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, listPipelineRuntimeInputs)
			runName = origRunName

			filterStr := fmt.Sprintf(`{"predicates":[{"key":"display_name","operation":"IS_SUBSTRING","string_value":"%s"}]}`, uniquePrefix)
			pageSize := int32(1)

			// Page 1: Namespace filter + display_name filter + pagination
			page1Runs, _, page1Token, err := runClient.List(&runparams.RunServiceListRunsParams{
				Namespace: &ns,
				Filter:    &filterStr,
				PageSize:  &pageSize,
			})
			Expect(err).NotTo(HaveOccurred(), "Failed to list runs page 1 by namespace with filter+pagination")
			Expect(len(page1Runs)).To(Equal(1), "Page 1 should return exactly 1 run")
			Expect(page1Token).NotTo(BeEmpty(), "Expected non-empty nextPageToken after page 1")

			// Page 2: same params + page token — args must not collide on PostgreSQL
			page2Runs, _, _, err := runClient.List(&runparams.RunServiceListRunsParams{
				Namespace: &ns,
				Filter:    &filterStr,
				PageSize:  &pageSize,
				PageToken: &page1Token,
			})
			Expect(err).NotTo(HaveOccurred(), "Failed to list runs page 2 by namespace with filter+pagination")
			Expect(len(page2Runs)).To(Equal(1), "Page 2 should return exactly 1 run")

			allRunIDs := make(map[string]bool)
			for _, r := range page1Runs {
				allRunIDs[r.RunID] = true
			}
			for _, r := range page2Runs {
				allRunIDs[r.RunID] = true
			}
			Expect(len(allRunIDs)).To(BeNumerically(">=", 2), "Expected at least 2 distinct runs across 2 pages")
		})

		It("List pipeline Runs with filter AND pagination across 2 pages", func() {
			// This test specifically targets pgx multi-level subquery placeholder numbering correctness.
			// When filter + pagination are combined, buildSelectRunsQuery produces SQL with multiple
			// args (filter value + page cursor value). All ?s are unified into $1,$2,... in one pass
			// via PlaceholderFormat(sq.Dollar). If args order is wrong, PostgreSQL returns incorrect
			// rows or an error, which would not be caught by SQLite-based unit tests.
			createdExperiment := createExperiment(experimentName)
			// Use unique prefix in display name to avoid cross-test interference
			uniquePrefix := "pgx-test-" + randomName
			// Override runName to embed uniquePrefix — createPipelineRun uses the package-level runName
			origRunName := runName
			runName = uniquePrefix + "-run"
			createPipelineRun(&listCreatedPipeline.PipelineID, &listCreatedPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, listPipelineRuntimeInputs)
			createPipelineRun(&listCreatedPipeline.PipelineID, &listCreatedPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, listPipelineRuntimeInputs)
			createPipelineRun(&listCreatedPipeline.PipelineID, &listCreatedPipelineVersion.PipelineVersionID, &createdExperiment.ExperimentID, listPipelineRuntimeInputs)
			runName = origRunName

			filterStr := fmt.Sprintf(`{"predicates":[{"key":"display_name","operation":"IS_SUBSTRING","string_value":"%s"}]}`, uniquePrefix)
			pageSize := int32(1)

			// Page 1
			page1Runs, _, page1Token, err := runClient.List(&runparams.RunServiceListRunsParams{
				ExperimentID: &createdExperiment.ExperimentID,
				Filter:       &filterStr,
				PageSize:     &pageSize,
			})
			Expect(err).NotTo(HaveOccurred(), "Failed to list runs page 1 with filter+pagination")
			Expect(len(page1Runs)).To(Equal(1), "Page 1 should return exactly 1 run")
			Expect(page1Token).NotTo(BeEmpty(), "Expected non-empty nextPageToken after page 1")

			// Page 2 — this is the critical call: filter arg + page cursor arg must map to correct $1/$2
			page2Runs, _, _, err := runClient.List(&runparams.RunServiceListRunsParams{
				ExperimentID: &createdExperiment.ExperimentID,
				Filter:       &filterStr,
				PageSize:     &pageSize,
				PageToken:    &page1Token,
			})
			Expect(err).NotTo(HaveOccurred(), "Failed to list runs page 2 with filter+pagination")
			Expect(len(page2Runs)).To(Equal(1), "Page 2 should return exactly 1 run")

			// Combined result
			allRunIDs := make(map[string]bool)
			for _, r := range page1Runs {
				allRunIDs[r.RunID] = true
			}
			for _, r := range page2Runs {
				allRunIDs[r.RunID] = true
			}
			Expect(len(allRunIDs)).To(BeNumerically(">=", 2), "Expected at least 2 distinct runs across 2 pages")
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
