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
	"testing"
	"time"

	"github.com/golang/glog"
	params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/experiment_client/experiment_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/experiment_model"
	upload_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_client/pipeline_upload_service"
	recurring_run_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/recurring_run_client/recurring_run_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/recurring_run_model"
	run_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_client/run_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_model"
	api_server "github.com/kubeflow/pipelines/backend/src/common/client/api_server/v2"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	test "github.com/kubeflow/pipelines/backend/test/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ExperimentApiTest struct {
	suite.Suite
	namespace            string
	resourceNamespace    string
	experimentClient     *api_server.ExperimentClient
	pipelineClient       *api_server.PipelineClient
	pipelineUploadClient *api_server.PipelineUploadClient
	runClient            *api_server.RunClient
	recurringRunClient   *api_server.RecurringRunClient
}

// Check the namespace have ML job installed and ready
func (s *ExperimentApiTest) SetupTest() {
	if !*runIntegrationTests {
		s.T().SkipNow()
		return
	}

	if !*isDevMode {
		err := test.WaitForReady(*initializeTimeout)
		if err != nil {
			glog.Exitf("Failed to initialize test. Error: %v", err)
		}
	}

	s.namespace = *namespace

	var newExperimentClient func() (*api_server.ExperimentClient, error)
	var newPipelineUploadClient func() (*api_server.PipelineUploadClient, error)
	var newPipelineClient func() (*api_server.PipelineClient, error)
	var newRunClient func() (*api_server.RunClient, error)
	var newRecurringRunClient func() (*api_server.RecurringRunClient, error)

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
		newRecurringRunClient = func() (*api_server.RecurringRunClient, error) {
			return api_server.NewKubeflowInClusterRecurringRunClient(s.namespace, *isDebugMode)
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
		newRecurringRunClient = func() (*api_server.RecurringRunClient, error) {
			return api_server.NewRecurringRunClient(clientConfig, *isDebugMode)
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
	s.recurringRunClient, err = newRecurringRunClient()
	if err != nil {
		glog.Exitf("Failed to get job client. Error: %s", err.Error())
	}

	s.cleanUp()
}

func (s *ExperimentApiTest) TestExperimentAPI() {
	t := s.T()

	/* ---------- Verify only default experiment exists ---------- */
	experiments, totalSize, _, err := test.ListAllExperiment(s.experimentClient, s.resourceNamespace)
	assert.Nil(t, err)
	assert.Equal(t, 1, totalSize)
	assert.True(t, len(experiments) == 1)

	/* ---------- Create a new experiment ---------- */
	experiment := test.MakeExperiment("training", "my first experiment", s.resourceNamespace)
	expectedTrainingExperiment := test.MakeExperiment("training", "my first experiment", s.resourceNamespace)

	trainingExperiment, err := s.experimentClient.Create(&params.ExperimentServiceCreateExperimentParams{
		Body: experiment,
	})
	assert.Nil(t, err)

	expectedTrainingExperiment.ExperimentID = trainingExperiment.ExperimentID
	expectedTrainingExperiment.CreatedAt = trainingExperiment.CreatedAt
	expectedTrainingExperiment.StorageState = "STORAGESTATE_AVAILABLE"
	expectedTrainingExperiment.Namespace = trainingExperiment.Namespace
	assert.Equal(t, expectedTrainingExperiment, trainingExperiment)

	/* ---------- Create an experiment with same name. Should fail due to name uniqueness ---------- */
	_, err = s.experimentClient.Create(&params.ExperimentServiceCreateExperimentParams{Body: experiment})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Please specify a new name")

	/* ---------- Create a few more new experiment ---------- */
	// 1 second interval. This ensures they can be sorted by create time in expected order.
	time.Sleep(1 * time.Second)
	experiment = test.MakeExperiment("prediction", "my second experiment", s.resourceNamespace)
	_, err = s.experimentClient.Create(&params.ExperimentServiceCreateExperimentParams{
		Body: experiment,
	})
	time.Sleep(1 * time.Second)
	experiment = test.MakeExperiment("moonshot", "my second experiment", s.resourceNamespace)
	_, err = s.experimentClient.Create(&params.ExperimentServiceCreateExperimentParams{
		Body: experiment,
	})
	assert.Nil(t, err)

	/* ---------- Verify list experiments works ---------- */
	experiments, totalSize, nextPageToken, err := test.ListAllExperiment(s.experimentClient, s.resourceNamespace)
	assert.Nil(t, err)
	assert.Equal(t, 4, totalSize)
	assert.Equal(t, 4, len(experiments))
	for _, e := range experiments {
		// Sampling one of the experiments and verify the result is expected.
		if e.DisplayName == "training" {
			assert.Equal(t, expectedTrainingExperiment, trainingExperiment)
		}
	}

	/* ---------- Verify list experiments sorted by names ---------- */
	experiments, totalSize, nextPageToken, err = test.ListExperiment(
		s.experimentClient,
		&params.ExperimentServiceListExperimentsParams{
			PageSize: util.Int32Pointer(2),
			SortBy:   util.StringPointer("name"),
		},
		s.resourceNamespace)
	assert.Nil(t, err)
	assert.Equal(t, 4, totalSize)
	assert.Equal(t, 2, len(experiments))
	assert.Equal(t, "Default", experiments[0].DisplayName)
	assert.Equal(t, "moonshot", experiments[1].DisplayName)
	assert.NotEmpty(t, nextPageToken)

	experiments, totalSize, nextPageToken, err = test.ListExperiment(
		s.experimentClient,
		&params.ExperimentServiceListExperimentsParams{
			PageToken: util.StringPointer(nextPageToken),
			PageSize:  util.Int32Pointer(2),
			SortBy:    util.StringPointer("name"),
		},
		s.resourceNamespace)
	assert.Nil(t, err)
	assert.Equal(t, 4, totalSize)
	assert.Equal(t, 2, len(experiments))
	assert.Equal(t, "prediction", experiments[0].DisplayName)
	assert.Equal(t, "training", experiments[1].DisplayName)
	assert.Empty(t, nextPageToken)

	/* ---------- Verify list experiments sorted by creation time ---------- */
	experiments, totalSize, nextPageToken, err = test.ListExperiment(
		s.experimentClient,
		&params.ExperimentServiceListExperimentsParams{
			PageSize: util.Int32Pointer(2),
			SortBy:   util.StringPointer("created_at"),
		},
		s.resourceNamespace)
	assert.Nil(t, err)
	assert.Equal(t, 4, totalSize)
	assert.Equal(t, 2, len(experiments))
	assert.Equal(t, "Default", experiments[0].DisplayName)
	assert.Equal(t, "training", experiments[1].DisplayName)
	assert.NotEmpty(t, nextPageToken)

	experiments, totalSize, nextPageToken, err = test.ListExperiment(
		s.experimentClient,
		&params.ExperimentServiceListExperimentsParams{
			PageToken: util.StringPointer(nextPageToken),
			PageSize:  util.Int32Pointer(2),
			SortBy:    util.StringPointer("created_at"),
		},
		s.resourceNamespace)
	assert.Nil(t, err)
	assert.Equal(t, 4, totalSize)
	assert.Equal(t, 2, len(experiments))
	assert.Equal(t, "prediction", experiments[0].DisplayName)
	assert.Equal(t, "moonshot", experiments[1].DisplayName)
	assert.Empty(t, nextPageToken)

	/* ---------- List experiments sort by unsupported field. Should fail. ---------- */
	_, _, _, err = test.ListExperiment(
		s.experimentClient,
		&params.ExperimentServiceListExperimentsParams{
			PageSize: util.Int32Pointer(2),
			SortBy:   util.StringPointer("unknownfield"),
		},
		s.resourceNamespace)
	assert.NotNil(t, err)

	/* ---------- List experiments sorted by names descend order ---------- */
	experiments, totalSize, nextPageToken, err = test.ListExperiment(
		s.experimentClient,
		&params.ExperimentServiceListExperimentsParams{
			PageSize: util.Int32Pointer(2),
			SortBy:   util.StringPointer("name desc"),
		},
		s.resourceNamespace)
	assert.Nil(t, err)
	assert.Equal(t, 4, totalSize)
	assert.Equal(t, 2, len(experiments))
	assert.Equal(t, "training", experiments[0].DisplayName)
	assert.Equal(t, "prediction", experiments[1].DisplayName)
	assert.NotEmpty(t, nextPageToken)

	experiments, totalSize, nextPageToken, err = test.ListExperiment(
		s.experimentClient,
		&params.ExperimentServiceListExperimentsParams{
			PageToken: util.StringPointer(nextPageToken),
			PageSize:  util.Int32Pointer(2),
			SortBy:    util.StringPointer("name desc"),
		},
		s.resourceNamespace)
	assert.Nil(t, err)
	assert.Equal(t, 4, totalSize)
	assert.Equal(t, 2, len(experiments))
	assert.Equal(t, "moonshot", experiments[0].DisplayName)
	assert.Equal(t, "Default", experiments[1].DisplayName)
	assert.Empty(t, nextPageToken)

	/* ---------- Verify get experiment works ---------- */
	experiment, err = s.experimentClient.Get(&params.ExperimentServiceGetExperimentParams{ExperimentID: trainingExperiment.ExperimentID})
	assert.Nil(t, err)
	assert.Equal(t, expectedTrainingExperiment, experiment)

	/* ---------- Create a pipeline version and two runs and two jobs -------------- */
	pipeline, err := s.pipelineUploadClient.UploadFile("../resources/hello-world.yaml", upload_params.NewUploadPipelineParams())
	assert.Nil(t, err)
	time.Sleep(1 * time.Second)
	pipelineVersion, err := s.pipelineUploadClient.UploadPipelineVersion(
		"../resources/hello-world.yaml", &upload_params.UploadPipelineVersionParams{
			Name:       util.StringPointer("hello-world-version"),
			Pipelineid: util.StringPointer(pipeline.PipelineID),
		})
	assert.Nil(t, err)
	createRunRequest := &run_params.RunServiceCreateRunParams{Body: &run_model.V2beta1Run{
		DisplayName:  "hello world",
		Description:  "this is hello world",
		ExperimentID: experiment.ExperimentID,
		PipelineVersionReference: &run_model.V2beta1PipelineVersionReference{
			PipelineID:        pipelineVersion.PipelineID,
			PipelineVersionID: pipelineVersion.PipelineVersionID,
		},
	}}
	run1, err := s.runClient.Create(createRunRequest)
	assert.Nil(t, err)
	run2, err := s.runClient.Create(createRunRequest)
	assert.Nil(t, err)
	/* ---------- Create a new hello world job by specifying pipeline ID ---------- */
	createRecurringRunRequest := &recurring_run_params.RecurringRunServiceCreateRecurringRunParams{Body: &recurring_run_model.V2beta1RecurringRun{
		DisplayName:  "hello world",
		Description:  "this is hello world",
		ExperimentID: experiment.ExperimentID,
		PipelineVersionReference: &recurring_run_model.V2beta1PipelineVersionReference{
			PipelineID:        pipelineVersion.PipelineID,
			PipelineVersionID: pipelineVersion.PipelineVersionID,
		},
		MaxConcurrency: 10,
		Status:         recurring_run_model.V2beta1RecurringRunStatusENABLED,
	}}
	recurringRun1, err := s.recurringRunClient.Create(createRecurringRunRequest)
	assert.Nil(t, err)
	recurringRun2, err := s.recurringRunClient.Create(createRecurringRunRequest)
	assert.Nil(t, err)

	/* ---------- Archive an experiment -----------------*/
	err = s.experimentClient.Archive(&params.ExperimentServiceArchiveExperimentParams{ExperimentID: trainingExperiment.ExperimentID})

	/* ---------- Verify experiment and its runs ------- */
	experiment, err = s.experimentClient.Get(&params.ExperimentServiceGetExperimentParams{ExperimentID: trainingExperiment.ExperimentID})
	assert.Nil(t, err)
	assert.Equal(t, experiment_model.V2beta1ExperimentStorageStateARCHIVED, experiment.StorageState)
	retrievedRun1, err := s.runClient.Get(&run_params.RunServiceGetRunParams{RunID: run1.RunID})
	assert.Nil(t, err)
	assert.Equal(t, run_model.V2beta1RunStorageStateARCHIVED, retrievedRun1.StorageState)
	retrievedRun2, err := s.runClient.Get(&run_params.RunServiceGetRunParams{RunID: run2.RunID})
	assert.Nil(t, err)
	assert.Equal(t, run_model.V2beta1RunStorageStateARCHIVED, retrievedRun2.StorageState)
	retrievedRecurringRun1, err := s.recurringRunClient.Get(&recurring_run_params.RecurringRunServiceGetRecurringRunParams{RecurringRunID: recurringRun1.RecurringRunID})
	assert.Nil(t, err)
	assert.Equal(t, recurring_run_model.V2beta1RecurringRunStatusDISABLED, retrievedRecurringRun1.Status)
	retrievedRecurringRun2, err := s.recurringRunClient.Get(&recurring_run_params.RecurringRunServiceGetRecurringRunParams{RecurringRunID: recurringRun2.RecurringRunID})
	assert.Nil(t, err)
	assert.Equal(t, recurring_run_model.V2beta1RecurringRunStatusDISABLED, retrievedRecurringRun2.Status)

	/* ---------- Unarchive an experiment -----------------*/
	err = s.experimentClient.Unarchive(&params.ExperimentServiceUnarchiveExperimentParams{ExperimentID: trainingExperiment.ExperimentID})

	/* ---------- Verify experiment and its runs and jobs --------- */
	experiment, err = s.experimentClient.Get(&params.ExperimentServiceGetExperimentParams{ExperimentID: trainingExperiment.ExperimentID})
	assert.Nil(t, err)
	assert.Equal(t, experiment_model.V2beta1ExperimentStorageStateAVAILABLE, experiment.StorageState)
	retrievedRun1, err = s.runClient.Get(&run_params.RunServiceGetRunParams{RunID: run1.RunID})
	assert.Nil(t, err)
	assert.Equal(t, run_model.V2beta1RunStorageStateARCHIVED, retrievedRun1.StorageState)
	retrievedRun2, err = s.runClient.Get(&run_params.RunServiceGetRunParams{RunID: run2.RunID})
	assert.Nil(t, err)
	assert.Equal(t, run_model.V2beta1RunStorageStateARCHIVED, retrievedRun2.StorageState)
	retrievedRecurringRun1, err = s.recurringRunClient.Get(&recurring_run_params.RecurringRunServiceGetRecurringRunParams{RecurringRunID: recurringRun1.RecurringRunID})
	assert.Nil(t, err)
	assert.Equal(t, recurring_run_model.V2beta1RecurringRunStatusDISABLED, retrievedRecurringRun1.Status)
	retrievedRecurringRun2, err = s.recurringRunClient.Get(&recurring_run_params.RecurringRunServiceGetRecurringRunParams{RecurringRunID: recurringRun2.RecurringRunID})
	assert.Nil(t, err)
	assert.Equal(t, recurring_run_model.V2beta1RecurringRunStatusDISABLED, retrievedRecurringRun2.Status)
}

func V2TestExperimentAPI(t *testing.T) {
	suite.Run(t, new(ExperimentApiTest))
}

func (s *ExperimentApiTest) TearDownSuite() {
	if *runIntegrationTests {
		if !*isDevMode {
			s.cleanUp()
		}
	}
}

func (s *ExperimentApiTest) cleanUp() {
	test.DeleteAllRuns(s.runClient, s.resourceNamespace, s.T())
	test.DeleteAllRecurringRuns(s.recurringRunClient, s.resourceNamespace, s.T())
	test.DeleteAllPipelines(s.pipelineClient, s.T())
	test.DeleteAllExperiments(s.experimentClient, s.resourceNamespace, s.T())
}
