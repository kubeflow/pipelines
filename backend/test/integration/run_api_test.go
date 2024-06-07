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
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"github.com/golang/glog"
	api "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	experimentparams "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/experiment_client/experiment_service"
	uploadParams "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/pipeline_upload_client/pipeline_upload_service"
	runparams "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/run_client/run_service"
	"github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/run_model"
	api_server "github.com/kubeflow/pipelines/backend/src/common/client/api_server/v1"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"k8s.io/apimachinery/pkg/util/yaml"
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

type RunResourceReferenceSorter []*run_model.APIResourceReference

func (r RunResourceReferenceSorter) Len() int           { return len(r) }
func (r RunResourceReferenceSorter) Less(i, j int) bool { return r[i].Name < r[j].Name }
func (r RunResourceReferenceSorter) Swap(i, j int)      { r[i], r[j] = r[j], r[i] }

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
	helloWorldPipeline, err := s.pipelineUploadClient.UploadFile("../resources/hello-world.yaml", uploadParams.NewUploadPipelineParams())
	assert.Nil(t, err)

	/* ---------- Upload a pipeline version YAML under helloWorldPipeline ---------- */
	time.Sleep(1 * time.Second)
	helloWorldPipelineVersion, err := s.pipelineUploadClient.UploadPipelineVersion(
		"../resources/hello-world.yaml", &uploadParams.UploadPipelineVersionParams{
			Name:       util.StringPointer("hello-world-version"),
			Pipelineid: util.StringPointer(helloWorldPipeline.ID),
		})
	assert.Nil(t, err)

	/* ---------- Create a new hello world experiment ---------- */
	experiment := test.GetExperiment("hello world experiment", "", s.resourceNamespace)
	helloWorldExperiment, err := s.experimentClient.Create(&experimentparams.ExperimentServiceCreateExperimentV1Params{Body: experiment})
	assert.Nil(t, err)

	/* ---------- Create a new hello world run by specifying pipeline version ID ---------- */
	createRunRequest := &runparams.RunServiceCreateRunV1Params{Body: &run_model.APIRun{
		Name:        "hello world",
		Description: "this is hello world",
		ResourceReferences: []*run_model.APIResourceReference{
			{
				Key:  &run_model.APIResourceKey{Type: run_model.APIResourceTypeEXPERIMENT, ID: helloWorldExperiment.ID},
				Name: helloWorldExperiment.Name, Relationship: run_model.APIRelationshipOWNER,
			},
			{
				Key:          &run_model.APIResourceKey{Type: run_model.APIResourceTypePIPELINEVERSION, ID: helloWorldPipelineVersion.ID},
				Relationship: run_model.APIRelationshipCREATOR,
			},
		},
	}}
	helloWorldRunDetail, _, err := s.runClient.Create(createRunRequest)
	assert.Nil(t, err)
	s.checkHelloWorldRunDetail(t, helloWorldRunDetail, helloWorldExperiment.ID, helloWorldExperiment.Name, helloWorldPipelineVersion.ID, helloWorldPipelineVersion.Name)

	/* ---------- Get hello world run ---------- */
	helloWorldRunDetail, _, err = s.runClient.Get(&runparams.RunServiceGetRunV1Params{RunID: helloWorldRunDetail.Run.ID})
	assert.Nil(t, err)
	s.checkHelloWorldRunDetail(t, helloWorldRunDetail, helloWorldExperiment.ID, helloWorldExperiment.Name, helloWorldPipelineVersion.ID, helloWorldPipelineVersion.Name)

	/* ---------- Create a new argument parameter experiment ---------- */
	createExperimentRequest := &experimentparams.ExperimentServiceCreateExperimentV1Params{
		Body: test.GetExperiment("argument parameter experiment", "", s.resourceNamespace),
	}
	argParamsExperiment, err := s.experimentClient.Create(createExperimentRequest)
	assert.Nil(t, err)

	/* ---------- Create a new argument parameter run by uploading workflow manifest ---------- */
	argParamsBytes, err := ioutil.ReadFile("../resources/arguments-parameters.yaml")
	assert.Nil(t, err)
	argParamsBytes, err = yaml.ToJSON(argParamsBytes)
	assert.Nil(t, err)
	createRunRequest = &runparams.RunServiceCreateRunV1Params{Body: &run_model.APIRun{
		Name:        "argument parameter",
		Description: "this is argument parameter",
		PipelineSpec: &run_model.APIPipelineSpec{
			WorkflowManifest: string(argParamsBytes),
			Parameters: []*run_model.APIParameter{
				{Name: "param1", Value: "goodbye"},
				{Name: "param2", Value: "world"},
			},
		},
		ResourceReferences: []*run_model.APIResourceReference{
			{
				Key:          &run_model.APIResourceKey{Type: run_model.APIResourceTypeEXPERIMENT, ID: argParamsExperiment.ID},
				Relationship: run_model.APIRelationshipOWNER,
			},
		},
	}}
	argParamsRunDetail, _, err := s.runClient.Create(createRunRequest)
	assert.Nil(t, err)
	s.checkArgParamsRunDetail(t, argParamsRunDetail, argParamsExperiment.ID, argParamsExperiment.Name)

	/* ---------- List all the runs. Both runs should be returned ---------- */
	runs, totalSize, _, err := test.ListAllRuns(s.runClient, s.resourceNamespace)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(runs))
	assert.Equal(t, 2, totalSize)

	/* ---------- List the runs, paginated, sorted by creation time ---------- */
	runs, totalSize, nextPageToken, err := test.ListRuns(
		s.runClient,
		&runparams.RunServiceListRunsV1Params{
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
		&runparams.RunServiceListRunsV1Params{
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
		&runparams.RunServiceListRunsV1Params{
			PageSize: util.Int32Pointer(1),
			SortBy:   util.StringPointer("name"),
		},
		s.resourceNamespace)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(runs))
	assert.Equal(t, 2, totalSize)
	assert.Equal(t, "argument parameter", runs[0].Name)
	runs, totalSize, _, err = test.ListRuns(
		s.runClient,
		&runparams.RunServiceListRunsV1Params{
			PageSize:  util.Int32Pointer(1),
			SortBy:    util.StringPointer("name"),
			PageToken: util.StringPointer(nextPageToken),
		},
		s.resourceNamespace)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(runs))
	assert.Equal(t, 2, totalSize)
	assert.Equal(t, "hello world", runs[0].Name)

	/* ---------- List the runs, sort by unsupported field ---------- */
	_, _, _, err = test.ListRuns(
		s.runClient,
		&runparams.RunServiceListRunsV1Params{PageSize: util.Int32Pointer(2), SortBy: util.StringPointer("unknownfield")},
		s.resourceNamespace)
	assert.NotNil(t, err)

	/* ---------- List runs for hello world experiment. One run should be returned ---------- */
	runs, totalSize, _, err = s.runClient.List(&runparams.RunServiceListRunsV1Params{
		ResourceReferenceKeyType: util.StringPointer(string(run_model.APIResourceTypeEXPERIMENT)),
		ResourceReferenceKeyID:   util.StringPointer(helloWorldExperiment.ID),
	})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(runs))
	assert.Equal(t, 1, totalSize)
	assert.Equal(t, "hello world", runs[0].Name)

	/* ---------- List the runs, filtered by created_at, only return the previous two runs ---------- */
	time.Sleep(5 * time.Second) // Sleep for 5 seconds to make sure the previous runs are created at a different timestamp
	filterTime := time.Now().Unix()
	time.Sleep(5 * time.Second)
	// Create a new run
	createRunRequest.Body.Name = "argument parameter 2"
	_, _, err = s.runClient.Create(createRunRequest)
	assert.Nil(t, err)
	// Check total number of runs is 3
	runs, totalSize, _, err = test.ListAllRuns(s.runClient, s.resourceNamespace)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(runs))
	assert.Equal(t, 3, totalSize)
	// Check number of filtered runs created before filterTime to be 2
	runs, totalSize, _, err = test.ListRuns(
		s.runClient,
		&runparams.RunServiceListRunsV1Params{
			Filter: util.StringPointer(`{"predicates": [{"key": "created_at", "op": 6, "string_value": "` + fmt.Sprint(filterTime) + `"}]}`),
		},
		s.resourceNamespace)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(runs))
	assert.Equal(t, 2, totalSize)

	/* ---------- Archive a run ------------*/
	err = s.runClient.Archive(&runparams.RunServiceArchiveRunV1Params{
		ID: helloWorldRunDetail.Run.ID,
	})
	assert.Nil(t, err)

	/* ---------- List runs for hello world experiment. The same run should still be returned, but should be archived ---------- */
	runs, totalSize, _, err = s.runClient.List(&runparams.RunServiceListRunsV1Params{
		ResourceReferenceKeyType: util.StringPointer(string(run_model.APIResourceTypeEXPERIMENT)),
		ResourceReferenceKeyID:   util.StringPointer(helloWorldExperiment.ID),
	})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(runs))
	assert.Equal(t, 1, totalSize)
	assert.Equal(t, "hello world", runs[0].Name)
	assert.Equal(t, string(runs[0].StorageState), api.Run_STORAGESTATE_ARCHIVED.String())

	/* ---------- Upload long-running pipeline YAML ---------- */
	longRunningPipeline, err := s.pipelineUploadClient.UploadFile("../resources/long-running.yaml", uploadParams.NewUploadPipelineParamsWithTimeout(350))
	assert.Nil(t, err)

	/* ---------- Upload a long-running pipeline version YAML under longRunningPipeline ---------- */
	time.Sleep(1 * time.Second)
	longRunningPipelineVersion, err := s.pipelineUploadClient.UploadPipelineVersion("../resources/long-running.yaml", &uploadParams.UploadPipelineVersionParams{
		Name:       util.StringPointer("long-running-version"),
		Pipelineid: util.StringPointer(longRunningPipeline.ID),
	})
	assert.Nil(t, err)

	/* ---------- Create a new long-running run by specifying pipeline ID ---------- */
	createLongRunningRunRequest := &runparams.RunServiceCreateRunV1Params{Body: &run_model.APIRun{
		Name:        "long running",
		Description: "this pipeline will run long enough for us to manually terminate it before it finishes",
		ResourceReferences: []*run_model.APIResourceReference{
			{
				Key:          &run_model.APIResourceKey{Type: run_model.APIResourceTypeEXPERIMENT, ID: helloWorldExperiment.ID},
				Relationship: run_model.APIRelationshipOWNER,
			},
			{
				Key:          &run_model.APIResourceKey{Type: run_model.APIResourceTypePIPELINEVERSION, ID: longRunningPipelineVersion.ID},
				Relationship: run_model.APIRelationshipCREATOR,
			},
		},
	}}
	longRunningRunDetail, _, err := s.runClient.Create(createLongRunningRunRequest)
	assert.Nil(t, err)

	/* ---------- Terminate the long-running run ------------*/
	err = s.runClient.Terminate(&runparams.RunServiceTerminateRunV1Params{
		RunID: longRunningRunDetail.Run.ID,
	})
	assert.Nil(t, err)

	/* ---------- Get long-running run ---------- */
	longRunningRunDetail, _, err = s.runClient.Get(&runparams.RunServiceGetRunV1Params{RunID: longRunningRunDetail.Run.ID})
	assert.Nil(t, err)
	s.checkTerminatedRunDetail(t, longRunningRunDetail, helloWorldExperiment.ID, helloWorldExperiment.Name, longRunningPipelineVersion.ID, longRunningPipelineVersion.Name)
}

func (s *RunApiTestSuite) checkTerminatedRunDetail(t *testing.T, runDetail *run_model.APIRunDetail, experimentId string, experimentName string, pipelineVersionId string, pipelineVersionName string) {
	// Check workflow manifest is not empty
	assert.Contains(t, runDetail.Run.PipelineSpec.WorkflowManifest, "wait-awhile")
	// Check runtime workflow manifest is not empty
	assert.Contains(t, runDetail.PipelineRuntime.WorkflowManifest, "wait-awhile")

	expectedRun := &run_model.APIRun{
		ID:             runDetail.Run.ID,
		Name:           "long running",
		Description:    "this pipeline will run long enough for us to manually terminate it before it finishes",
		Status:         "Terminating",
		ServiceAccount: test.GetDefaultPipelineRunnerServiceAccount(*isKubeflowMode),
		PipelineSpec: &run_model.APIPipelineSpec{
			PipelineID:       runDetail.Run.PipelineSpec.PipelineID,
			PipelineName:     runDetail.Run.PipelineSpec.PipelineName,
			WorkflowManifest: runDetail.Run.PipelineSpec.WorkflowManifest,
		},
		ResourceReferences: []*run_model.APIResourceReference{
			{
				Key:  &run_model.APIResourceKey{Type: run_model.APIResourceTypeEXPERIMENT, ID: experimentId},
				Name: experimentName, Relationship: run_model.APIRelationshipOWNER,
			},
			{
				Key:  &run_model.APIResourceKey{Type: run_model.APIResourceTypePIPELINEVERSION, ID: pipelineVersionId},
				Name: pipelineVersionName, Relationship: run_model.APIRelationshipCREATOR,
			},
		},
		CreatedAt:   runDetail.Run.CreatedAt,
		ScheduledAt: runDetail.Run.ScheduledAt,
		FinishedAt:  runDetail.Run.FinishedAt,
	}

	assert.True(t, test.VerifyRunResourceReferences(runDetail.Run.ResourceReferences, expectedRun.ResourceReferences))
	expectedRun.ResourceReferences = runDetail.Run.ResourceReferences
	assert.Equal(t, expectedRun, runDetail.Run)
}

func (s *RunApiTestSuite) checkHelloWorldRunDetail(t *testing.T, runDetail *run_model.APIRunDetail, experimentId string, experimentName string, pipelineVersionId string, pipelineVersionName string) {
	// Check workflow manifest is not empty
	assert.Contains(t, runDetail.Run.PipelineSpec.WorkflowManifest, "whalesay")
	// Check runtime workflow manifest is not empty
	assert.Contains(t, runDetail.PipelineRuntime.WorkflowManifest, "whalesay")

	expectedRun := &run_model.APIRun{
		ID:             runDetail.Run.ID,
		Name:           "hello world",
		Description:    "this is hello world",
		Status:         runDetail.Run.Status,
		ServiceAccount: test.GetDefaultPipelineRunnerServiceAccount(*isKubeflowMode),
		PipelineSpec: &run_model.APIPipelineSpec{
			PipelineID:       runDetail.Run.PipelineSpec.PipelineID,
			PipelineName:     runDetail.Run.PipelineSpec.PipelineName,
			WorkflowManifest: runDetail.Run.PipelineSpec.WorkflowManifest,
		},
		ResourceReferences: []*run_model.APIResourceReference{
			{
				Key:  &run_model.APIResourceKey{Type: run_model.APIResourceTypeEXPERIMENT, ID: experimentId},
				Name: experimentName, Relationship: run_model.APIRelationshipOWNER,
			},
			{
				Key:  &run_model.APIResourceKey{Type: run_model.APIResourceTypePIPELINEVERSION, ID: pipelineVersionId},
				Name: pipelineVersionName, Relationship: run_model.APIRelationshipCREATOR,
			},
		},
		CreatedAt:   runDetail.Run.CreatedAt,
		ScheduledAt: runDetail.Run.ScheduledAt,
		FinishedAt:  runDetail.Run.FinishedAt,
	}

	assert.True(t, test.VerifyRunResourceReferences(runDetail.Run.ResourceReferences, expectedRun.ResourceReferences))
	expectedRun.ResourceReferences = runDetail.Run.ResourceReferences
	assert.Equal(t, expectedRun, runDetail.Run)
}

func (s *RunApiTestSuite) checkArgParamsRunDetail(t *testing.T, runDetail *run_model.APIRunDetail, experimentId string, experimentName string) {
	argParamsBytes, err := ioutil.ReadFile("../resources/arguments-parameters.yaml")
	assert.Nil(t, err)
	argParamsBytes, err = yaml.ToJSON(argParamsBytes)
	assert.Nil(t, err)
	// Check runtime workflow manifest is not empty
	assert.Contains(t, runDetail.PipelineRuntime.WorkflowManifest, "arguments-parameters-")
	expectedRun := &run_model.APIRun{
		ID:             runDetail.Run.ID,
		Name:           "argument parameter",
		Description:    "this is argument parameter",
		Status:         runDetail.Run.Status,
		ServiceAccount: test.GetDefaultPipelineRunnerServiceAccount(*isKubeflowMode),
		PipelineSpec: &run_model.APIPipelineSpec{
			PipelineID:       runDetail.Run.PipelineSpec.PipelineID,
			PipelineName:     runDetail.Run.PipelineSpec.PipelineName,
			WorkflowManifest: string(argParamsBytes),
			Parameters: []*run_model.APIParameter{
				{Name: "param1", Value: "goodbye"},
				{Name: "param2", Value: "world"},
			},
		},
		ResourceReferences: []*run_model.APIResourceReference{
			{
				Key:  &run_model.APIResourceKey{Type: run_model.APIResourceTypeEXPERIMENT, ID: experimentId},
				Name: experimentName, Relationship: run_model.APIRelationshipOWNER,
			},
		},
		CreatedAt:   runDetail.Run.CreatedAt,
		ScheduledAt: runDetail.Run.ScheduledAt,
		FinishedAt:  runDetail.Run.FinishedAt,
	}

	assert.True(t, test.VerifyRunResourceReferences(runDetail.Run.ResourceReferences, expectedRun.ResourceReferences))
	expectedRun.ResourceReferences = runDetail.Run.ResourceReferences
	assert.Equal(t, expectedRun, runDetail.Run)
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
