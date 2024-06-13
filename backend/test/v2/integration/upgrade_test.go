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
	"strings"
	"testing"
	"time"

	"github.com/golang/glog"
	experiment_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/experiment_client/experiment_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/experiment_model"
	params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_client/pipeline_service"
	pipeline_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_client/pipeline_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_model"
	upload_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_client/pipeline_upload_service"
	recurring_run_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/recurring_run_client/recurring_run_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/recurring_run_model"
	runParams "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_client/run_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_model"
	api_server "github.com/kubeflow/pipelines/backend/src/common/client/api_server/v2"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	test "github.com/kubeflow/pipelines/backend/test/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"sigs.k8s.io/yaml"
)

// Methods are organized into two types: "prepare" and "verify".
// "prepare" tests setup resources before upgrade
// "verify" tests verifies resources are expected after upgrade
type UpgradeTests struct {
	suite.Suite
	namespace            string
	resourceNamespace    string
	experimentClient     *api_server.ExperimentClient
	pipelineClient       *api_server.PipelineClient
	pipelineUploadClient *api_server.PipelineUploadClient
	runClient            *api_server.RunClient
	recurringRunClient   *api_server.RecurringRunClient
}

func TestUpgrade(t *testing.T) {
	suite.Run(t, new(UpgradeTests))
}

func (s *UpgradeTests) TestPrepare() {
	t := s.T()

	test.DeleteAllRecurringRuns(s.recurringRunClient, s.resourceNamespace, t)
	test.DeleteAllRuns(s.runClient, s.resourceNamespace, t)
	test.DeleteAllPipelines(s.pipelineClient, t)
	test.DeleteAllExperiments(s.experimentClient, s.resourceNamespace, t)

	s.PrepareExperiments()
	s.PreparePipelines()
	s.PrepareRuns()
	s.PrepareRecurringRuns()
}

func (s *UpgradeTests) TestVerify() {
	s.VerifyExperiments()
	s.VerifyPipelines()
	s.VerifyRuns()
	s.VerifyRecurringRuns()
	s.VerifyCreatingRunsAndRecurringRuns()
}

// Check the namespace have ML job installed and ready
func (s *UpgradeTests) SetupSuite() {
	// Integration tests also run these tests to first ensure they work, so that
	// when integration tests pass and upgrade tests fail, we know for sure
	// upgrade process went wrong somehow.
	if !(*runIntegrationTests || *runUpgradeTests) {
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
}

func (s *UpgradeTests) TearDownSuite() {
	if *runIntegrationTests {
		if !*isDevMode {
			t := s.T()
			// Clean up after the suite to unblock other tests. (Not needed for upgrade
			// tests because it needs changes in prepare tests to persist and verified
			// later.)
			test.DeleteAllRecurringRuns(s.recurringRunClient, s.resourceNamespace, t)
			test.DeleteAllRuns(s.runClient, s.resourceNamespace, t)
			test.DeleteAllPipelines(s.pipelineClient, t)
			test.DeleteAllExperiments(s.experimentClient, s.resourceNamespace, t)
		}
	}
}

func (s *UpgradeTests) PrepareExperiments() {
	t := s.T()

	/* ---------- Create a new experiment ---------- */
	experiment := test.MakeExperiment("training", "my first experiment", s.resourceNamespace)
	_, err := s.experimentClient.Create(&experiment_params.ExperimentServiceCreateExperimentParams{
		Body: experiment,
	})
	require.Nil(t, err)

	/* ---------- Create a few more new experiment ---------- */
	// This ensures they can be sorted by create time in expected order.
	time.Sleep(1 * time.Second)
	experiment = test.MakeExperiment("prediction", "my second experiment", s.resourceNamespace)
	_, err = s.experimentClient.Create(&experiment_params.ExperimentServiceCreateExperimentParams{
		Body: experiment,
	})
	require.Nil(t, err)

	time.Sleep(1 * time.Second)
	experiment = test.MakeExperiment("moonshot", "my third experiment", s.resourceNamespace)
	_, err = s.experimentClient.Create(&experiment_params.ExperimentServiceCreateExperimentParams{
		Body: experiment,
	})
	require.Nil(t, err)
}

func (s *UpgradeTests) VerifyExperiments() {
	t := s.T()

	/* ---------- Verify list experiments sorted by creation time ---------- */
	// This should have the hello-world experiment in addition to the old experiments.
	experiments, _, _, err := test.ListExperiment(
		s.experimentClient,
		&experiment_params.ExperimentServiceListExperimentsParams{SortBy: util.StringPointer("created_at")},
		"",
	)
	require.Nil(t, err)

	allExperiments := make([]string, len(experiments))
	for i, exp := range experiments {
		allExperiments[i] = fmt.Sprintf("%v: %v/%v", i, exp.Namespace, exp.DisplayName)
	}
	fmt.Printf("All experiments: %v", allExperiments)
	assert.Equal(t, 5, len(experiments))

	// Default experiment is no longer deletable
	assert.Equal(t, "Default", experiments[0].DisplayName)
	assert.Contains(t, experiments[0].Description, "All runs created without specifying an experiment will be grouped here")
	assert.NotEmpty(t, experiments[0].ExperimentID)
	assert.NotEmpty(t, experiments[0].CreatedAt)

	assert.Equal(t, "training", experiments[1].DisplayName)
	assert.Equal(t, "my first experiment", experiments[1].Description)
	assert.NotEmpty(t, experiments[1].ExperimentID)
	assert.NotEmpty(t, experiments[1].CreatedAt)

	assert.Equal(t, "prediction", experiments[2].DisplayName)
	assert.Equal(t, "my second experiment", experiments[2].Description)
	assert.NotEmpty(t, experiments[2].ExperimentID)
	assert.NotEmpty(t, experiments[2].CreatedAt)

	assert.Equal(t, "moonshot", experiments[3].DisplayName)
	assert.Equal(t, "my third experiment", experiments[3].Description)
	assert.NotEmpty(t, experiments[3].ExperimentID)
	assert.NotEmpty(t, experiments[3].CreatedAt)

	assert.Equal(t, "hello world experiment", experiments[4].DisplayName)
	assert.Equal(t, "", experiments[4].Description)
	assert.NotEmpty(t, experiments[4].ExperimentID)
	assert.NotEmpty(t, experiments[4].CreatedAt)

}

func (s *UpgradeTests) PreparePipelines() {
	t := s.T()

	test.DeleteAllPipelines(s.pipelineClient, t)

	/* ---------- Upload pipelines YAML ---------- */
	argumentYAMLPipeline, err := s.pipelineUploadClient.UploadFile("../resources/arguments-parameters.yaml", upload_params.NewUploadPipelineParams())
	require.Nil(t, err)
	assert.Equal(t, "arguments-parameters.yaml", argumentYAMLPipeline.DisplayName)

	/* ---------- Import pipeline YAML by URL ---------- */
	time.Sleep(1 * time.Second)
	sequentialPipeline, err := s.pipelineClient.Create(&pipeline_params.PipelineServiceCreatePipelineParams{
		Body: &pipeline_model.V2beta1Pipeline{DisplayName: "sequential"},
	})
	require.Nil(t, err)
	assert.Equal(t, "sequential", sequentialPipeline.DisplayName)
	sequentialPipelineVersion, err := s.pipelineClient.CreatePipelineVersion(&params.PipelineServiceCreatePipelineVersionParams{
		PipelineID: sequentialPipeline.PipelineID,
		Body: &pipeline_model.V2beta1PipelineVersion{
			DisplayName: "sequential",
			PackageURL: &pipeline_model.V2beta1URL{
				PipelineURL: "https://storage.googleapis.com/ml-pipeline-dataset/v2/sequential.yaml",
			},
			PipelineID: sequentialPipeline.PipelineID,
		},
	})
	require.Nil(t, err)
	assert.Equal(t, "sequential", sequentialPipelineVersion.DisplayName)

	/* ---------- Upload pipelines zip ---------- */
	time.Sleep(1 * time.Second)
	argumentUploadPipeline, err := s.pipelineUploadClient.UploadFile(
		"../resources/arguments.pipeline.zip", &upload_params.UploadPipelineParams{Name: util.StringPointer("zip-arguments-parameters")})
	require.Nil(t, err)
	assert.Equal(t, "zip-arguments-parameters", argumentUploadPipeline.DisplayName)

	/* ---------- Import pipeline tarball by URL ---------- */
	time.Sleep(1 * time.Second)
	argumentUrlPipeline, err := s.pipelineClient.Create(&pipeline_params.PipelineServiceCreatePipelineParams{
		Body: &pipeline_model.V2beta1Pipeline{DisplayName: "arguments.pipeline.zip"},
	})
	require.Nil(t, err)
	assert.Equal(t, "arguments.pipeline.zip", argumentUrlPipeline.DisplayName)
	argumentUrlPipelineVersion, err := s.pipelineClient.CreatePipelineVersion(&params.PipelineServiceCreatePipelineVersionParams{
		PipelineID: argumentUrlPipeline.PipelineID,
		Body: &pipeline_model.V2beta1PipelineVersion{
			DisplayName: "arguments",
			PackageURL: &pipeline_model.V2beta1URL{
				PipelineURL: "https://storage.googleapis.com/ml-pipeline-dataset/v2/arguments.pipeline.zip",
			},
			PipelineID: argumentUrlPipeline.PipelineID,
		},
	})
	require.Nil(t, err)
	assert.Equal(t, "arguments", argumentUrlPipelineVersion.DisplayName)

	time.Sleep(1 * time.Second)
}

func (s *UpgradeTests) VerifyPipelines() {
	t := s.T()

	/* ---------- Verify list pipeline sorted by creation time ---------- */
	pipelines, _, _, err := s.pipelineClient.List(
		&pipeline_params.PipelineServiceListPipelinesParams{SortBy: util.StringPointer("created_at")})
	require.Nil(t, err)
	// During upgrade, default pipelines may be installed, so we only verify the
	// 4 oldest pipelines here.
	assert.True(t, len(pipelines) >= 4)
	assert.Equal(t, "arguments-parameters.yaml", pipelines[0].DisplayName)
	assert.Equal(t, "sequential", pipelines[1].DisplayName)
	assert.Equal(t, "zip-arguments-parameters", pipelines[2].DisplayName)
	assert.Equal(t, "arguments.pipeline.zip", pipelines[3].DisplayName)

	/* ---------- Verify pipeline spec ---------- */
	pipelineVersions, totalSize, _, err := s.pipelineClient.ListPipelineVersions(
		&params.PipelineServiceListPipelineVersionsParams{
			PipelineID: pipelines[0].PipelineID,
		})
	require.Nil(t, err)
	assert.Equal(t, totalSize, 1)
	pipelineVersion, err := s.pipelineClient.GetPipelineVersion(&params.PipelineServiceGetPipelineVersionParams{PipelineID: pipelines[0].PipelineID, PipelineVersionID: pipelineVersions[0].PipelineVersionID})
	require.Nil(t, err)
	bytes, err := ioutil.ReadFile("../resources/arguments-parameters.yaml")
	expected_bytes, err := yaml.YAMLToJSON(bytes)
	require.Nil(t, err)
	actual_bytes, err := json.Marshal(pipelineVersion.PipelineSpec)
	require.Nil(t, err)
	// Override pipeline name, then compare
	assert.Equal(t, string(expected_bytes), strings.Replace(string(actual_bytes), "pipeline/arguments-parameters.yaml", "whalesay", 1))
}

func (s *UpgradeTests) PrepareRuns() {
	t := s.T()

	helloWorldPipeline := s.getHelloWorldPipeline(true)
	helloWorldExperiment := s.getHelloWorldExperiment(true)
	if helloWorldExperiment == nil {
		helloWorldExperiment = s.createHelloWorldExperiment()
	}

	hello2 := s.getHelloWorldExperiment(true)
	require.Equal(t, hello2, helloWorldExperiment)

	/* ---------- Create a new hello world run by specifying pipeline ID ---------- */
	createRunRequest := &runParams.RunServiceCreateRunParams{Body: &run_model.V2beta1Run{
		DisplayName:  "hello world",
		Description:  "this is hello world",
		ExperimentID: helloWorldExperiment.ExperimentID,
		PipelineVersionReference: &run_model.V2beta1PipelineVersionReference{
			PipelineID:        helloWorldPipeline.PipelineID,
			PipelineVersionID: helloWorldPipeline.PipelineVersionID,
		},
	}}
	_, err := s.runClient.Create(createRunRequest)
	require.Nil(t, err)
}

func (s *UpgradeTests) VerifyRuns() {
	t := s.T()

	/* ---------- List the runs, sorted by creation time ---------- */
	runs, _, _, err := test.ListRuns(
		s.runClient,
		&runParams.RunServiceListRunsParams{SortBy: util.StringPointer("created_at")},
		s.resourceNamespace)
	require.Nil(t, err)
	assert.True(t, len(runs) >= 1)
	assert.Equal(t, "hello world", runs[0].DisplayName)

	/* ---------- Get hello world run ---------- */
	helloWorldRunDetail, err := s.runClient.Get(&runParams.RunServiceGetRunParams{RunID: runs[0].RunID})
	require.Nil(t, err)
	assert.Equal(t, "hello world", helloWorldRunDetail.DisplayName)
	assert.Equal(t, "this is hello world", helloWorldRunDetail.Description)
}

func (s *UpgradeTests) PrepareRecurringRuns() {
	t := s.T()

	pipeline := s.getHelloWorldPipeline(true)
	experiment := s.getHelloWorldExperiment(true)

	/* ---------- Create a new hello world job by specifying pipeline ID ---------- */
	createRecurringRunRequest := &recurring_run_params.RecurringRunServiceCreateRecurringRunParams{Body: &recurring_run_model.V2beta1RecurringRun{
		DisplayName: "hello world",
		Description: "this is hello world",
		PipelineVersionReference: &recurring_run_model.V2beta1PipelineVersionReference{
			PipelineID:        pipeline.PipelineID,
			PipelineVersionID: pipeline.PipelineVersionID,
		},
		ExperimentID:   experiment.ExperimentID,
		MaxConcurrency: 10,
		Mode:           recurring_run_model.RecurringRunModeENABLE,
		NoCatchup:      true,
	}}
	_, err := s.recurringRunClient.Create(createRecurringRunRequest)
	require.Nil(t, err)
}

func (s *UpgradeTests) VerifyRecurringRuns() {
	t := s.T()

	pipeline := s.getHelloWorldPipeline(false)
	experiment := s.getHelloWorldExperiment(false)

	/* ---------- Get hello world recurring run ---------- */
	recurringRuns, _, _, err := test.ListAllRecurringRuns(s.recurringRunClient, s.resourceNamespace)
	require.Nil(t, err)
	require.Len(t, recurringRuns, 1)
	recurringRun := recurringRuns[0]

	expectedRecurringRun := &recurring_run_model.V2beta1RecurringRun{
		RecurringRunID: recurringRun.RecurringRunID,
		DisplayName:    "hello world",
		Description:    "this is hello world",
		ExperimentID:   experiment.ExperimentID,
		PipelineVersionReference: &recurring_run_model.V2beta1PipelineVersionReference{
			PipelineID:        pipeline.PipelineID,
			PipelineVersionID: pipeline.PipelineVersionID,
		},
		ServiceAccount: test.GetDefaultPipelineRunnerServiceAccount(*isKubeflowMode),
		MaxConcurrency: 10,
		NoCatchup:      true,
		Mode:           recurring_run_model.RecurringRunModeENABLE,
		Namespace:      recurringRun.Namespace,
		CreatedAt:      recurringRun.CreatedAt,
		UpdatedAt:      recurringRun.UpdatedAt,
		Trigger:        recurringRun.Trigger,
		Status:         recurringRun.Status,
	}

	assert.Equal(t, expectedRecurringRun, recurringRun)
}

func (s *UpgradeTests) VerifyCreatingRunsAndRecurringRuns() {
	t := s.T()

	/* ---------- Get the oldest pipeline and the newest experiment ---------- */
	pipelines, _, _, err := s.pipelineClient.List(
		&pipeline_params.PipelineServiceListPipelinesParams{SortBy: util.StringPointer("created_at")})
	require.Nil(t, err)
	assert.Equal(t, "arguments-parameters.yaml", pipelines[0].DisplayName)
	pipelineVersions, totalSize, _, err := s.pipelineClient.ListPipelineVersions(&params.PipelineServiceListPipelineVersionsParams{
		PipelineID: pipelines[0].PipelineID,
	})
	require.Nil(t, err)
	assert.Equal(t, 1, totalSize)

	experiments, _, _, err := test.ListExperiment(
		s.experimentClient,
		&experiment_params.ExperimentServiceListExperimentsParams{SortBy: util.StringPointer("created_at")},
		"",
	)
	require.Nil(t, err)
	assert.Equal(t, "Default", experiments[0].DisplayName)
	assert.Equal(t, "training", experiments[1].DisplayName)
	assert.Equal(t, "hello world experiment", experiments[4].DisplayName)

	/* ---------- Create a new run based on the oldest pipeline and its default pipeline version ---------- */
	createRunRequest := &runParams.RunServiceCreateRunParams{Body: &run_model.V2beta1Run{
		DisplayName: "argument parameter from pipeline",
		Description: "a run from an old pipeline",
		// This run should belong to the newest experiment (created after the upgrade)
		ExperimentID: experiments[4].ExperimentID,
		RuntimeConfig: &run_model.V2beta1RuntimeConfig{
			Parameters: map[string]interface{}{
				"param1": "goodbye",
				"param2": "world",
			},
		},
		PipelineVersionReference: &run_model.V2beta1PipelineVersionReference{
			PipelineID:        pipelineVersions[0].PipelineID,
			PipelineVersionID: pipelineVersions[0].PipelineVersionID,
		},
	}}
	runFromPipeline, err := s.runClient.Create(createRunRequest)
	assert.Nil(t, err)

	assert.Equal(t, experiments[4].ExperimentID, runFromPipeline.ExperimentID)

	/* ---------- Create a new recurring run based on the second oldest pipeline version and belonging to the second oldest experiment ---------- */
	pipelineVersions, totalSize, _, err = s.pipelineClient.ListPipelineVersions(&params.PipelineServiceListPipelineVersionsParams{
		PipelineID: pipelines[1].PipelineID,
	})
	require.Nil(t, err)
	assert.Equal(t, 1, totalSize)

	createRecurringRunRequest := &recurring_run_params.RecurringRunServiceCreateRecurringRunParams{Body: &recurring_run_model.V2beta1RecurringRun{
		DisplayName:  "sequential job from pipeline version",
		Description:  "a recurring run from an old pipeline version",
		ExperimentID: experiments[1].ExperimentID,
		PipelineVersionReference: &recurring_run_model.V2beta1PipelineVersionReference{
			PipelineID:        pipelineVersions[0].PipelineID,
			PipelineVersionID: pipelineVersions[0].PipelineVersionID,
		},
		RuntimeConfig: &recurring_run_model.V2beta1RuntimeConfig{
			Parameters: map[string]interface{}{
				"url": "gs://ml-pipeline-playground/shakespeare1.txt",
			},
		},
		MaxConcurrency: 10,
		Mode:           recurring_run_model.RecurringRunModeENABLE,
	}}
	createdRecurringRun, err := s.recurringRunClient.Create(createRecurringRunRequest)
	assert.Nil(t, err)
	assert.Equal(t, experiments[1].ExperimentID, createdRecurringRun.ExperimentID)
}

func (s *UpgradeTests) createHelloWorldExperiment() *experiment_model.V2beta1Experiment {
	t := s.T()

	experiment := test.MakeExperiment("hello world experiment", "", s.resourceNamespace)
	helloWorldExperiment, err := s.experimentClient.Create(&experiment_params.ExperimentServiceCreateExperimentParams{Body: experiment})
	require.Nil(t, err)

	return helloWorldExperiment
}

func (s *UpgradeTests) getHelloWorldExperiment(createIfNotExist bool) *experiment_model.V2beta1Experiment {
	t := s.T()

	experiments, _, _, err := test.ListExperiment(
		s.experimentClient,
		&experiment_params.ExperimentServiceListExperimentsParams{
			PageSize: util.Int32Pointer(1000),
		},
		s.resourceNamespace)
	require.Nil(t, err)
	var helloWorldExperiment *experiment_model.V2beta1Experiment
	for _, experiment := range experiments {
		if experiment.DisplayName == "hello world experiment" {
			helloWorldExperiment = experiment
		}
	}

	if helloWorldExperiment == nil && createIfNotExist {
		return s.createHelloWorldExperiment()
	}

	return helloWorldExperiment
}

func (s *UpgradeTests) getHelloWorldPipeline(createIfNotExist bool) *pipeline_model.V2beta1PipelineVersion {
	t := s.T()

	pipelines, err := s.pipelineClient.ListAll(&pipeline_params.PipelineServiceListPipelinesParams{}, 1000)
	require.Nil(t, err)
	var helloWorldPipeline *pipeline_model.V2beta1Pipeline
	for _, pipeline := range pipelines {
		if pipeline.DisplayName == "hello-world.yaml" {
			helloWorldPipeline = pipeline
		}
	}

	if helloWorldPipeline == nil && createIfNotExist {
		return s.createHelloWorldPipeline()
	}
	pipelineVersions, totalSize, _, err := s.pipelineClient.ListPipelineVersions(&params.PipelineServiceListPipelineVersionsParams{
		PipelineID: helloWorldPipeline.PipelineID,
	})
	require.Nil(t, err)
	require.Equal(t, 1, totalSize)

	return pipelineVersions[0]
}

func (s *UpgradeTests) createHelloWorldPipeline() *pipeline_model.V2beta1PipelineVersion {
	t := s.T()

	/* ---------- Upload pipelines YAML ---------- */
	uploadedPipeline, err := s.pipelineUploadClient.UploadFile("../resources/hello-world.yaml", upload_params.NewUploadPipelineParams())
	require.Nil(t, err)

	pipelineVersions, totalSize, _, err := s.pipelineClient.ListPipelineVersions(&params.PipelineServiceListPipelineVersionsParams{
		PipelineID: uploadedPipeline.PipelineID,
	})
	require.Nil(t, err)
	require.Equal(t, 1, totalSize)

	return pipelineVersions[0]
}
