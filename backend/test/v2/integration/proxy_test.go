// Copyright 2025 The Kubeflow Authors
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
	experimentparams "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/experiment_client/experiment_service"
	uploadparams "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_client/pipeline_upload_service"
	runparams "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_client/run_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_model"
	apiserver "github.com/kubeflow/pipelines/backend/src/common/client/api_server/v2"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/test/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ProxyTestSuite struct {
	suite.Suite
	namespace            string
	resourceNamespace    string
	experimentClient     *apiserver.ExperimentClient
	pipelineClient       *apiserver.PipelineClient
	pipelineUploadClient *apiserver.PipelineUploadClient
	runClient            *apiserver.RunClient
}

func (s *ProxyTestSuite) SetupTest() {
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

	var newExperimentClient func() (*apiserver.ExperimentClient, error)
	var newPipelineUploadClient func() (*apiserver.PipelineUploadClient, error)
	var newPipelineClient func() (*apiserver.PipelineClient, error)
	var newRunClient func() (*apiserver.RunClient, error)

	if *isKubeflowMode {
		s.resourceNamespace = *resourceNamespace

		newExperimentClient = func() (*apiserver.ExperimentClient, error) {
			return apiserver.NewKubeflowInClusterExperimentClient(s.namespace, *isDebugMode)
		}
		newPipelineUploadClient = func() (*apiserver.PipelineUploadClient, error) {
			return apiserver.NewKubeflowInClusterPipelineUploadClient(s.namespace, *isDebugMode)
		}
		newPipelineClient = func() (*apiserver.PipelineClient, error) {
			return apiserver.NewKubeflowInClusterPipelineClient(s.namespace, *isDebugMode)
		}
		newRunClient = func() (*apiserver.RunClient, error) {
			return apiserver.NewKubeflowInClusterRunClient(s.namespace, *isDebugMode)
		}
	} else {
		clientConfig := test.GetClientConfig(*namespace)

		newExperimentClient = func() (*apiserver.ExperimentClient, error) {
			return apiserver.NewExperimentClient(clientConfig, *isDebugMode)
		}
		newPipelineUploadClient = func() (*apiserver.PipelineUploadClient, error) {
			return apiserver.NewPipelineUploadClient(clientConfig, *isDebugMode)
		}
		newPipelineClient = func() (*apiserver.PipelineClient, error) {
			return apiserver.NewPipelineClient(clientConfig, *isDebugMode)
		}
		newRunClient = func() (*apiserver.RunClient, error) {
			return apiserver.NewRunClient(clientConfig, *isDebugMode)
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

func (s *ProxyTestSuite) TestEnvVar() {
	t := s.T()

	/* ---------- Upload pipelines YAML ---------- */
	envVarPipeline, err := s.pipelineUploadClient.UploadFile("../resources/env-var.yaml", uploadparams.NewUploadPipelineParams())
	require.Nil(t, err)

	/* ---------- Upload a pipeline version YAML under envVarPipeline ---------- */
	time.Sleep(1 * time.Second)
	envVarPipelineVersion, err := s.pipelineUploadClient.UploadPipelineVersion(
		"../resources/env-var.yaml", &uploadparams.UploadPipelineVersionParams{
			Name:       util.StringPointer("env-var-version"),
			Pipelineid: util.StringPointer(envVarPipeline.PipelineID),
		})
	require.Nil(t, err)

	/* ---------- Create a new env var experiment ---------- */
	experiment := test.MakeExperiment("env var experiment", "", s.resourceNamespace)
	envVarExperiment, err := s.experimentClient.Create(&experimentparams.ExperimentServiceCreateExperimentParams{Experiment: experiment})
	require.Nil(t, err)

	/* ---------- Create a new env var run by specifying pipeline version ID ---------- */
	createRunRequest := &runparams.RunServiceCreateRunParams{Run: &run_model.V2beta1Run{
		DisplayName:  "env var",
		Description:  "this is env var",
		ExperimentID: envVarExperiment.ExperimentID,
		PipelineVersionReference: &run_model.V2beta1PipelineVersionReference{
			PipelineID:        envVarPipelineVersion.PipelineID,
			PipelineVersionID: envVarPipelineVersion.PipelineVersionID,
		},
		RuntimeConfig: &run_model.V2beta1RuntimeConfig{
			Parameters: map[string]interface{}{
				"env_var": "http_proxy",
			},
		},
	}}
	envVarRunDetail, err := s.runClient.Create(createRunRequest)
	require.Nil(t, err)

	expectedState := run_model.V2beta1RuntimeStateFAILED
	if *useProxy {
		expectedState = run_model.V2beta1RuntimeStateSUCCEEDED
	}

	assert.Eventually(t, func() bool {
		envVarRunDetail, err = s.runClient.Get(&runparams.RunServiceGetRunParams{RunID: envVarRunDetail.RunID})
		t.Logf("Pipeline state: %v", envVarRunDetail.State)
		return err == nil && *envVarRunDetail.State == expectedState
	}, 2*time.Minute, 10*time.Second)
}

func TestProxy(t *testing.T) {
	suite.Run(t, new(ProxyTestSuite))
}

func (s *ProxyTestSuite) TearDownSuite() {
	if *runIntegrationTests {
		if !*isDevMode {
			s.cleanUp()
		}
	}
}

func (s *ProxyTestSuite) cleanUp() {
	/* ---------- Clean up ---------- */
	test.DeleteAllRuns(s.runClient, s.resourceNamespace, s.T())
	test.DeleteAllPipelines(s.pipelineClient, s.T())
	test.DeleteAllExperiments(s.experimentClient, s.resourceNamespace, s.T())
}
