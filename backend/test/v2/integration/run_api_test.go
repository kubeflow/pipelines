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

package integration

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"github.com/golang/glog"
	experiment_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/experiment_client/experiment_service"
	upload_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_client/pipeline_upload_service"
	run_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_client/run_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_model"
	api_server "github.com/kubeflow/pipelines/backend/src/common/client/api_server/v2"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	test "github.com/kubeflow/pipelines/backend/test/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/types/known/structpb"
	"sigs.k8s.io/yaml"
)

type RunApiTestSuite struct {
	suite.Suite
	namespace            string
	resourceNamespace    string
	experimentClient     *api_server.ExperimentClient
	pipelineClient       *api_server.PipelineClient
	pipelineUploadClient *api_server.PipelineUploadClient
	runClient            *api_server.RunClient
}

// Check the namespace have ML pipeline installed and ready
func (s *RunApiTestSuite) SetupTest() {
	if !*runIntegrationTests {
		s.T().SkipNow()
		return
	}

	if !*isDevMode {
		err := test.WaitForReady(*initializeTimeout)
		if err != nil {
			glog.Exitf("Failed to initialize test. Error: %s", err.Error())
		}
	}
	s.namespace = *namespace

	var newExperimentClient func() (*api_server.ExperimentClient, error)
	var newPipelineUploadClient func() (*api_server.PipelineUploadClient, error)
	var newPipelineClient func() (*api_server.PipelineClient, error)
	var newRunClient func() (*api_server.RunClient, error)

	if *isKubeflowMode {
		s.resourceNamespace = *resourceNamespace

		newExperimentClient = func() (*api_server.ExperimentClient, error) {
			return api_server.NewKubeflowInClusterExperimentClient(s.namespace, *isDebugMode)
		}
		newPipelineUploadClient = func() (*api_server.PipelineUploadClient, error) {
			return api_server.NewKubeflowInClusterPipelineUploadClient(s.namespace, *isDebugMode)
		}
		newPipelineClient = func() (*api_server.PipelineClient, error) {
			return api_server.NewKubeflowInClusterPipelineClient(s.namespace, *isDebugMode)
		}
		newRunClient = func() (*api_server.RunClient, error) {
			return api_server.NewKubeflowInClusterRunClient(s.namespace, *isDebugMode)
		}
	} else {
		clientConfig := test.GetClientConfig(*namespace)

		newExperimentClient = func() (*api_server.ExperimentClient, error) {
			return api_server.NewExperimentClient(clientConfig, *isDebugMode)
		}
		newPipelineUploadClient = func() (*api_server.PipelineUploadClient, error) {
			return api_server.NewPipelineUploadClient(clientConfig, *isDebugMode)
		}
		newPipelineClient = func() (*api_server.PipelineClient, error) {
			return api_server.NewPipelineClient(clientConfig, *isDebugMode)
		}
		newRunClient = func() (*api_server.RunClient, error) {
			return api_server.NewRunClient(clientConfig, *isDebugMode)
		}
	}

	var err error
	s.experimentClient, err = newExperimentClient()
	if err != nil {
		glog.Exitf("Failed to get experiment client. Error: %v", err)
	}
	s.pipelineUploadClient, err = newPipelineUploadClient()
	if err != nil {
		glog.Exitf("Failed to get pipeline upload client. Error: %s", err.Error())
	}
	s.pipelineClient, err = newPipelineClient()
	if err != nil {
		glog.Exitf("Failed to get pipeline client. Error: %s", err.Error())
	}
	s.runClient, err = newRunClient()
	if err != nil {
		glog.Exitf("Failed to get run client. Error: %s", err.Error())
	}

	s.cleanUp()
}

func (s *RunApiTestSuite) TestRunApis() {
	t := s.T()

	/* ---------- Upload pipelines YAML ---------- */
	helloWorldPipeline, err := s.pipelineUploadClient.UploadFile("../resources/hello-world.yaml", upload_params.NewUploadPipelineParams())
	assert.Nil(t, err)

	/* ---------- Upload a pipeline version YAML under helloWorldPipeline ---------- */
	time.Sleep(1 * time.Second)
	helloWorldPipelineVersion, err := s.pipelineUploadClient.UploadPipelineVersion(
		"../resources/hello-world.yaml", &upload_params.UploadPipelineVersionParams{
			Name:       util.StringPointer("hello-world-version"),
			Pipelineid: util.StringPointer(helloWorldPipeline.PipelineID),
		})
	assert.Nil(t, err)

	/* ---------- Create a new hello world experiment ---------- */
	experiment := test.MakeExperiment("hello world experiment", "", s.resourceNamespace)
	helloWorldExperiment, err := s.experimentClient.Create(&experiment_params.ExperimentServiceCreateExperimentParams{Body: experiment})
	assert.Nil(t, err)

	/* ---------- Create a new hello world run by specifying pipeline version ID ---------- */
	createRunRequest := &run_params.RunServiceCreateRunParams{Body: &run_model.V2beta1Run{
		DisplayName:  "hello world",
		Description:  "this is hello world",
		ExperimentID: helloWorldExperiment.ExperimentID,
		PipelineVersionReference: &run_model.V2beta1PipelineVersionReference{
			PipelineID:        helloWorldPipelineVersion.PipelineID,
			PipelineVersionID: helloWorldPipelineVersion.PipelineVersionID,
		},
	}}
	helloWorldRunDetail, err := s.runClient.Create(createRunRequest)
	assert.Nil(t, err)
	s.checkHelloWorldRunDetail(t, helloWorldRunDetail, helloWorldExperiment.ExperimentID, helloWorldPipelineVersion.PipelineID, helloWorldPipelineVersion.PipelineVersionID)

	/* ---------- Get hello world run ---------- */
	helloWorldRunDetail, err = s.runClient.Get(&run_params.RunServiceGetRunParams{RunID: helloWorldRunDetail.RunID})
	assert.Nil(t, err)
	s.checkHelloWorldRunDetail(t, helloWorldRunDetail, helloWorldExperiment.ExperimentID, helloWorldPipelineVersion.PipelineID, helloWorldPipelineVersion.PipelineVersionID)

	/* ---------- Create a new argument parameter experiment ---------- */
	createExperimentRequest := &experiment_params.ExperimentServiceCreateExperimentParams{
		Body: test.MakeExperiment("argument parameter experiment", "", s.resourceNamespace),
	}
	argParamsExperiment, err := s.experimentClient.Create(createExperimentRequest)
	assert.Nil(t, err)

	/* ---------- Create a new argument parameter run by uploading workflow manifest ---------- */
	argParamsBytes, err := ioutil.ReadFile("../resources/arguments-parameters.yaml")
	assert.Nil(t, err)
	pipeline_spec := &structpb.Struct{}
	err = yaml.Unmarshal(argParamsBytes, pipeline_spec)
	assert.Nil(t, err)

	createRunRequest = &run_params.RunServiceCreateRunParams{Body: &run_model.V2beta1Run{
		DisplayName:  "argument parameter",
		Description:  "this is argument parameter",
		PipelineSpec: pipeline_spec,
		RuntimeConfig: &run_model.V2beta1RuntimeConfig{
			Parameters: map[string]interface{}{
				"param1": "goodbye",
				"param2": "world",
			},
		},
		ExperimentID: argParamsExperiment.ExperimentID,
	}}
	argParamsRunDetail, err := s.runClient.Create(createRunRequest)
	assert.Nil(t, err)
	s.checkArgParamsRunDetail(t, argParamsRunDetail, argParamsExperiment.ExperimentID)

	/* ---------- List all the runs. Both runs should be returned ---------- */
	runs, totalSize, _, err := test.ListAllRuns(s.runClient, s.resourceNamespace)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(runs))
	assert.Equal(t, 2, totalSize)

	/* ---------- List the runs, paginated, sorted by creation time ---------- */
	runs, totalSize, nextPageToken, err := test.ListRuns(
		s.runClient,
		&run_params.RunServiceListRunsParams{
			PageSize: util.Int32Pointer(1),
			SortBy:   util.StringPointer("created_at"),
		},
		s.resourceNamespace)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(runs))
	assert.Equal(t, 2, totalSize)
	/* TODO(issues/1762): fix the following flaky assertion. */
	/* assert.Equal(t, "hello world", runs[0].Name) */
	runs, totalSize, _, err = test.ListRuns(
		s.runClient,
		&run_params.RunServiceListRunsParams{
			PageSize:  util.Int32Pointer(1),
			PageToken: util.StringPointer(nextPageToken),
		},
		s.resourceNamespace)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(runs))
	assert.Equal(t, 2, totalSize)
	/* TODO(issues/1762): fix the following flaky assertion. */
	/* assert.Equal(t, "argument parameter", runs[0].Name) */

	/* ---------- List the runs, paginated, sort by name ---------- */
	runs, totalSize, nextPageToken, err = test.ListRuns(
		s.runClient,
		&run_params.RunServiceListRunsParams{
			PageSize: util.Int32Pointer(1),
			SortBy:   util.StringPointer("name"),
		},
		s.resourceNamespace)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(runs))
	assert.Equal(t, 2, totalSize)
	assert.Equal(t, "argument parameter", runs[0].DisplayName)
	runs, totalSize, _, err = test.ListRuns(
		s.runClient,
		&run_params.RunServiceListRunsParams{
			PageSize:  util.Int32Pointer(1),
			SortBy:    util.StringPointer("name"),
			PageToken: util.StringPointer(nextPageToken),
		},
		s.resourceNamespace)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(runs))
	assert.Equal(t, 2, totalSize)
	assert.Equal(t, "hello world", runs[0].DisplayName)

	/* ---------- List the runs, sort by unsupported field ---------- */
	_, _, _, err = test.ListRuns(
		s.runClient,
		&run_params.RunServiceListRunsParams{PageSize: util.Int32Pointer(2), SortBy: util.StringPointer("unknownfield")},
		s.resourceNamespace)
	assert.NotNil(t, err)

	/* ---------- List runs for hello world experiment. One run should be returned ---------- */
	runs, totalSize, _, err = s.runClient.List(&run_params.RunServiceListRunsParams{
		ExperimentID: util.StringPointer(helloWorldExperiment.ExperimentID),
	})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(runs))
	assert.Equal(t, 1, totalSize)
	assert.Equal(t, "hello world", runs[0].DisplayName)

	/* ---------- List the runs, filtered by created_at, only return the previous two runs ---------- */
	time.Sleep(5 * time.Second) // Sleep for 5 seconds to make sure the previous runs are created at a different timestamp
	filterTime := time.Now().Unix()
	time.Sleep(5 * time.Second)
	// Create a new run
	createRunRequest.Body.DisplayName = "argument parameter 2"
	_, err = s.runClient.Create(createRunRequest)
	assert.Nil(t, err)
	// Check total number of runs is 3
	runs, totalSize, _, err = test.ListAllRuns(s.runClient, s.resourceNamespace)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(runs))
	assert.Equal(t, 3, totalSize)
	// Check number of filtered runs created before filterTime to be 2
	runs, totalSize, _, err = test.ListRuns(
		s.runClient,
		&run_params.RunServiceListRunsParams{
			Filter: util.StringPointer(`{"predicates": [{"key": "created_at", "operation": "LESS_THAN", "string_value": "` + fmt.Sprint(filterTime) + `"}]}`),
		},
		s.resourceNamespace)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(runs))
	assert.Equal(t, 2, totalSize)

	/* ---------- Archive a run ------------*/
	err = s.runClient.Archive(&run_params.RunServiceArchiveRunParams{
		RunID: helloWorldRunDetail.RunID,
	})
	assert.Nil(t, err)

	/* ---------- List runs for hello world experiment. The same run should still be returned, but should be archived ---------- */
	runs, totalSize, _, err = s.runClient.List(&run_params.RunServiceListRunsParams{
		ExperimentID: util.StringPointer(helloWorldExperiment.ExperimentID),
	})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(runs))
	assert.Equal(t, 1, totalSize)
	assert.Equal(t, "hello world", runs[0].DisplayName)
	assert.Equal(t, run_model.V2beta1RunStorageStateARCHIVED, runs[0].StorageState)

	/* ---------- Upload long-running pipeline YAML ---------- */
	longRunningPipeline, err := s.pipelineUploadClient.UploadFile("../resources/long-running.yaml", upload_params.NewUploadPipelineParamsWithTimeout(350))
	assert.Nil(t, err)

	/* ---------- Upload a long-running pipeline version YAML under longRunningPipeline ---------- */
	time.Sleep(1 * time.Second)
	longRunningPipelineVersion, err := s.pipelineUploadClient.UploadPipelineVersion("../resources/long-running.yaml", &upload_params.UploadPipelineVersionParams{
		Name:       util.StringPointer("long-running-version"),
		Pipelineid: util.StringPointer(longRunningPipeline.PipelineID),
	})
	assert.Nil(t, err)

	/* ---------- Create a new long-running run by specifying pipeline ID ---------- */
	createLongRunningRunRequest := &run_params.RunServiceCreateRunParams{Body: &run_model.V2beta1Run{
		DisplayName:  "long running",
		Description:  "this pipeline will run long enough for us to manually terminate it before it finishes",
		ExperimentID: helloWorldExperiment.ExperimentID,
		PipelineVersionReference: &run_model.V2beta1PipelineVersionReference{
			PipelineID:        longRunningPipelineVersion.PipelineID,
			PipelineVersionID: longRunningPipelineVersion.PipelineVersionID,
		},
	}}
	longRunningRun, err := s.runClient.Create(createLongRunningRunRequest)
	assert.Nil(t, err)

	/* ---------- Terminate the long-running run ------------*/
	err = s.runClient.Terminate(&run_params.RunServiceTerminateRunParams{
		RunID: longRunningRun.RunID,
	})
	assert.Nil(t, err)

	/* ---------- Get long-running run ---------- */
	longRunningRun, err = s.runClient.Get(&run_params.RunServiceGetRunParams{RunID: longRunningRun.RunID})
	assert.Nil(t, err)
	s.checkTerminatedRunDetail(t, longRunningRun, helloWorldExperiment.ExperimentID, longRunningPipelineVersion.PipelineID, longRunningPipelineVersion.PipelineVersionID)
}

func (s *RunApiTestSuite) checkTerminatedRunDetail(t *testing.T, run *run_model.V2beta1Run, experimentId string, pipelineId string, pipelineVersionId string) {

	expectedRun := &run_model.V2beta1Run{
		RunID:          run.RunID,
		DisplayName:    "long running",
		Description:    "this pipeline will run long enough for us to manually terminate it before it finishes",
		State:          run_model.V2beta1RuntimeStateCANCELING,
		StateHistory:   run.StateHistory,
		StorageState:   run.StorageState,
		ServiceAccount: test.GetDefaultPipelineRunnerServiceAccount(*isKubeflowMode),
		PipelineSpec:   run.PipelineSpec,
		ExperimentID:   experimentId,
		PipelineVersionReference: &run_model.V2beta1PipelineVersionReference{
			PipelineID:        pipelineId,
			PipelineVersionID: pipelineVersionId,
		},
		CreatedAt:   run.CreatedAt,
		ScheduledAt: run.ScheduledAt,
		FinishedAt:  run.FinishedAt,
	}

	assert.Equal(t, expectedRun, run)
}

func (s *RunApiTestSuite) checkHelloWorldRunDetail(t *testing.T, run *run_model.V2beta1Run, experimentId string, pipelineId string, pipelineVersionId string) {

	expectedRun := &run_model.V2beta1Run{
		RunID:          run.RunID,
		DisplayName:    "hello world",
		Description:    "this is hello world",
		State:          run.State,
		StateHistory:   run.StateHistory,
		StorageState:   run.StorageState,
		ServiceAccount: test.GetDefaultPipelineRunnerServiceAccount(*isKubeflowMode),
		PipelineSpec:   run.PipelineSpec,
		ExperimentID:   experimentId,
		PipelineVersionReference: &run_model.V2beta1PipelineVersionReference{
			PipelineID:        pipelineId,
			PipelineVersionID: pipelineVersionId,
		},
		CreatedAt:   run.CreatedAt,
		ScheduledAt: run.ScheduledAt,
		FinishedAt:  run.FinishedAt,
	}

	assert.Equal(t, expectedRun, run)
}

func (s *RunApiTestSuite) checkArgParamsRunDetail(t *testing.T, run *run_model.V2beta1Run, experimentId string) {

	// Compare the pipeline spec first.
	argParamsBytes, err := ioutil.ReadFile("../resources/arguments-parameters.yaml")
	assert.Nil(t, err)
	// pipeline_spec := &structpb.Struct{}
	// err = yaml.Unmarshal(argParamsBytes, pipeline_spec)
	// assert.Nil(t, err)
	expected_bytes, err := yaml.YAMLToJSON(argParamsBytes)
	assert.Nil(t, err)
	actual_bytes, err := json.Marshal(run.PipelineSpec)
	assert.Nil(t, err)
	assert.Equal(t, string(expected_bytes), string(actual_bytes))

	expectedRun := &run_model.V2beta1Run{
		RunID:          run.RunID,
		DisplayName:    "argument parameter",
		Description:    "this is argument parameter",
		State:          run.State,
		StateHistory:   run.StateHistory,
		StorageState:   run.StorageState,
		ServiceAccount: test.GetDefaultPipelineRunnerServiceAccount(*isKubeflowMode),
		PipelineSpec:   run.PipelineSpec,
		RuntimeConfig: &run_model.V2beta1RuntimeConfig{
			Parameters: map[string]interface{}{
				"param1": "goodbye",
				"param2": "world",
			},
		},
		ExperimentID: experimentId,
		CreatedAt:    run.CreatedAt,
		ScheduledAt:  run.ScheduledAt,
		FinishedAt:   run.FinishedAt,
	}

	assert.Equal(t, expectedRun, run)
}

func TestRunApi(t *testing.T) {
	suite.Run(t, new(RunApiTestSuite))
}

func (s *RunApiTestSuite) TearDownSuite() {
	if *runIntegrationTests {
		if !*isDevMode {
			s.cleanUp()
		}
	}
}

func (s *RunApiTestSuite) cleanUp() {
	/* ---------- Clean up ---------- */
	test.DeleteAllRuns(s.runClient, s.resourceNamespace, s.T())
	test.DeleteAllPipelines(s.pipelineClient, s.T())
	test.DeleteAllExperiments(s.experimentClient, s.resourceNamespace, s.T())
}
