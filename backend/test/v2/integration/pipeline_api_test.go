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
	params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_client/pipeline_service"
	model "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_model"
	upload_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_client/pipeline_upload_service"
	api_server "github.com/kubeflow/pipelines/backend/src/common/client/api_server/v2"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	test "github.com/kubeflow/pipelines/backend/test/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// This test suit tests various methods to import pipeline to pipeline system, including
// - upload v2 pipeline spec JSON file
// - upload yaml file
// - upload tarball file
// - providing YAML file url
// - providing tarball file url
type PipelineApiTest struct {
	suite.Suite
	namespace            string
	resourceNamespace    string
	pipelineClient       *api_server.PipelineClient
	pipelineUploadClient *api_server.PipelineUploadClient
}

// Check the namespace have ML job installed and ready
func (s *PipelineApiTest) SetupTest() {
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

	var newPipelineUploadClient func() (*api_server.PipelineUploadClient, error)
	var newPipelineClient func() (*api_server.PipelineClient, error)

	if *isKubeflowMode {
		s.resourceNamespace = *resourceNamespace

		newPipelineUploadClient = func() (*api_server.PipelineUploadClient, error) {
			return api_server.NewKubeflowInClusterPipelineUploadClient(s.namespace, *isDebugMode)
		}
		newPipelineClient = func() (*api_server.PipelineClient, error) {
			return api_server.NewKubeflowInClusterPipelineClient(s.namespace, *isDebugMode)
		}
	} else {
		clientConfig := test.GetClientConfig(*namespace)

		newPipelineUploadClient = func() (*api_server.PipelineUploadClient, error) {
			return api_server.NewPipelineUploadClient(clientConfig, *isDebugMode)
		}
		newPipelineClient = func() (*api_server.PipelineClient, error) {
			return api_server.NewPipelineClient(clientConfig, *isDebugMode)
		}
	}

	var err error
	s.pipelineUploadClient, err = newPipelineUploadClient()
	if err != nil {
		glog.Exitf("Failed to get pipeline upload client. Error: %s", err.Error())
	}
	s.pipelineClient, err = newPipelineClient()
	if err != nil {
		glog.Exitf("Failed to get pipeline client. Error: %s", err.Error())
	}

	s.cleanUp()
}

func (s *PipelineApiTest) TestPipelineAPI() {
	t := s.T()

	test.DeleteAllPipelines(s.pipelineClient, t)

	/* ------ Upload v2 pipeline spec YAML --------*/
	helloPipeline, err := s.pipelineUploadClient.UploadFile("../resources/hello-world.yaml", upload_params.NewUploadPipelineParams())
	require.Nil(t, err)
	assert.Equal(t, "hello-world.yaml", helloPipeline.DisplayName)

	/* ---------- Upload pipelines YAML ---------- */
	time.Sleep(1 * time.Second)
	argumentYAMLPipeline, err := s.pipelineUploadClient.UploadFile("../resources/arguments-parameters.yaml", upload_params.NewUploadPipelineParams())
	require.Nil(t, err)
	assert.Equal(t, "arguments-parameters.yaml", argumentYAMLPipeline.DisplayName)

	/* ---------- Upload the same pipeline again. Should fail due to name uniqueness ---------- */
	_, err = s.pipelineUploadClient.UploadFile("../resources/arguments-parameters.yaml", upload_params.NewUploadPipelineParams())
	require.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed to upload pipeline")

	/* ---------- Import pipeline YAML by URL ---------- */
	time.Sleep(1 * time.Second)
	sequentialPipeline, err := s.pipelineClient.CreatePipelineAndVersion(&params.PipelineServiceCreatePipelineAndVersionParams{
		Body: &model.V2beta1CreatePipelineAndVersionRequest{
			Pipeline: &model.V2beta1Pipeline{
				DisplayName: "sequential",
				Description: "sequential pipeline",
			},
			PipelineVersion: &model.V2beta1PipelineVersion{
				PackageURL: &model.V2beta1URL{
					PipelineURL: "https://storage.googleapis.com/ml-pipeline-dataset/v2/sequential.yaml",
				},
			},
		},
	})
	require.Nil(t, err)
	assert.Equal(t, "sequential", sequentialPipeline.DisplayName)
	assert.Equal(t, "sequential pipeline", sequentialPipeline.Description)
	sequentialPipelineVersions, totalSize, _, err := s.pipelineClient.ListPipelineVersions(&params.PipelineServiceListPipelineVersionsParams{PipelineID: sequentialPipeline.PipelineID})
	require.Nil(t, err)
	assert.Equal(t, 1, totalSize)
	assert.Equal(t, 1, len(sequentialPipelineVersions))
	assert.Equal(t, "sequential", sequentialPipelineVersions[0].DisplayName)
	assert.Equal(t, "sequential pipeline", sequentialPipelineVersions[0].Description)
	assert.Equal(t, sequentialPipeline.PipelineID, sequentialPipelineVersions[0].PipelineID)
	assert.Equal(t, "https://storage.googleapis.com/ml-pipeline-dataset/v2/sequential.yaml", sequentialPipelineVersions[0].PackageURL.PipelineURL)

	/* ---------- Upload pipelines zip ---------- */
	time.Sleep(1 * time.Second)
	argumentUploadPipeline, err := s.pipelineUploadClient.UploadFile(
		"../resources/arguments.pipeline.zip", &upload_params.UploadPipelineParams{Name: util.StringPointer("zip-arguments-parameters")})
	require.Nil(t, err)
	assert.Equal(t, "zip-arguments-parameters", argumentUploadPipeline.DisplayName)

	/* ---------- Import pipeline tarball by URL ---------- */
	time.Sleep(1 * time.Second)
	argumentUrlPipeline, err := s.pipelineClient.Create(&params.PipelineServiceCreatePipelineParams{
		Body: &model.V2beta1Pipeline{DisplayName: "arguments.pipeline.zip"},
	})
	require.Nil(t, err)
	argumentUrlPipelineVersion, err := s.pipelineClient.CreatePipelineVersion(
		&params.PipelineServiceCreatePipelineVersionParams{
			PipelineID: argumentUrlPipeline.PipelineID,
			Body: &model.V2beta1PipelineVersion{
				DisplayName: "argumentUrl-v1",
				Description: "1st version of argument url pipeline",
				PipelineID:  sequentialPipeline.PipelineID,
				PackageURL: &model.V2beta1URL{
					PipelineURL: "https://storage.googleapis.com/ml-pipeline-dataset/v2/arguments.pipeline.zip",
				},
			},
		})
	require.Nil(t, err)
	assert.Equal(t, "argumentUrl-v1", argumentUrlPipelineVersion.DisplayName)
	assert.Equal(t, "1st version of argument url pipeline", argumentUrlPipelineVersion.Description)
	assert.Equal(t, argumentUrlPipeline.PipelineID, argumentUrlPipelineVersion.PipelineID)
	assert.Equal(t, "https://storage.googleapis.com/ml-pipeline-dataset/v2/arguments.pipeline.zip", argumentUrlPipelineVersion.PackageURL.PipelineURL)

	/* ---------- Verify list pipeline works ---------- */
	pipelines, totalSize, _, err := s.pipelineClient.List(&params.PipelineServiceListPipelinesParams{})
	require.Nil(t, err)
	assert.Equal(t, 5, len(pipelines))
	assert.Equal(t, 5, totalSize)
	for _, p := range pipelines {
		// Sampling one of the pipelines and verify the result is expected.
		if p.DisplayName == "arguments-parameters.yaml" {
			assert.NotNil(t, *p)
			assert.NotNil(t, p.CreatedAt)
		}
	}

	/* ---------- Verify list pipeline sorted by names ---------- */
	listFirstPagePipelines, totalSize, nextPageToken, err := s.pipelineClient.List(
		&params.PipelineServiceListPipelinesParams{PageSize: util.Int32Pointer(2), SortBy: util.StringPointer("name")})
	require.Nil(t, err)
	assert.Equal(t, 2, len(listFirstPagePipelines))
	assert.Equal(t, 5, totalSize)
	assert.Equal(t, "arguments-parameters.yaml", listFirstPagePipelines[0].DisplayName)
	assert.Equal(t, "arguments.pipeline.zip", listFirstPagePipelines[1].DisplayName)
	assert.NotEmpty(t, nextPageToken)

	listSecondPagePipelines, totalSize, nextPageToken, err := s.pipelineClient.List(
		&params.PipelineServiceListPipelinesParams{PageToken: util.StringPointer(nextPageToken), PageSize: util.Int32Pointer(3), SortBy: util.StringPointer("name")})
	require.Nil(t, err)
	assert.Equal(t, 3, len(listSecondPagePipelines))
	assert.Equal(t, 5, totalSize)
	assert.Equal(t, "hello-world.yaml", listSecondPagePipelines[0].DisplayName)
	assert.Equal(t, "sequential", listSecondPagePipelines[1].DisplayName)
	assert.Equal(t, "zip-arguments-parameters", listSecondPagePipelines[2].DisplayName)
	assert.Empty(t, nextPageToken)

	/* ---------- Verify list pipeline sorted by creation time ---------- */
	listFirstPagePipelines, totalSize, nextPageToken, err = s.pipelineClient.List(
		&params.PipelineServiceListPipelinesParams{PageSize: util.Int32Pointer(3), SortBy: util.StringPointer("created_at")})
	require.Nil(t, err)
	assert.Equal(t, 3, len(listFirstPagePipelines))
	assert.Equal(t, 5, totalSize)
	assert.Equal(t, "hello-world.yaml", listFirstPagePipelines[0].DisplayName)
	assert.Equal(t, "arguments-parameters.yaml", listFirstPagePipelines[1].DisplayName)
	assert.Equal(t, "sequential", listFirstPagePipelines[2].DisplayName)
	assert.NotEmpty(t, nextPageToken)

	listSecondPagePipelines, totalSize, nextPageToken, err = s.pipelineClient.List(
		&params.PipelineServiceListPipelinesParams{PageToken: util.StringPointer(nextPageToken), PageSize: util.Int32Pointer(3), SortBy: util.StringPointer("created_at")})
	require.Nil(t, err)
	assert.Equal(t, 2, len(listSecondPagePipelines))
	assert.Equal(t, 5, totalSize)
	assert.Equal(t, "zip-arguments-parameters", listSecondPagePipelines[0].DisplayName)
	assert.Equal(t, "arguments.pipeline.zip", listSecondPagePipelines[1].DisplayName)
	assert.Empty(t, nextPageToken)

	/* ---------- List pipelines sort by unsupported description field. Should fail. ---------- */
	_, _, _, err = s.pipelineClient.List(&params.PipelineServiceListPipelinesParams{
		PageSize: util.Int32Pointer(2), SortBy: util.StringPointer("unknownfield"),
	})
	assert.NotNil(t, err)

	/* ---------- List pipelines sorted by names descend order ---------- */
	listFirstPagePipelines, totalSize, nextPageToken, err = s.pipelineClient.List(
		&params.PipelineServiceListPipelinesParams{PageSize: util.Int32Pointer(3), SortBy: util.StringPointer("name desc")})
	require.Nil(t, err)
	assert.Equal(t, 3, len(listFirstPagePipelines))
	assert.Equal(t, 5, totalSize)
	assert.Equal(t, "zip-arguments-parameters", listFirstPagePipelines[0].DisplayName)
	assert.Equal(t, "sequential", listFirstPagePipelines[1].DisplayName)
	assert.Equal(t, "hello-world.yaml", listFirstPagePipelines[2].DisplayName)
	assert.NotEmpty(t, nextPageToken)

	listSecondPagePipelines, totalSize, nextPageToken, err = s.pipelineClient.List(&params.PipelineServiceListPipelinesParams{
		PageToken: util.StringPointer(nextPageToken), PageSize: util.Int32Pointer(3), SortBy: util.StringPointer("name desc"),
	})
	require.Nil(t, err)
	assert.Equal(t, 2, len(listSecondPagePipelines))
	assert.Equal(t, 5, totalSize)
	assert.Equal(t, "arguments.pipeline.zip", listSecondPagePipelines[0].DisplayName)
	assert.Equal(t, "arguments-parameters.yaml", listSecondPagePipelines[1].DisplayName)
	assert.Empty(t, nextPageToken)

	/* ---------- Verify get pipeline works ---------- */
	pipeline, err := s.pipelineClient.Get(&params.PipelineServiceGetPipelineParams{PipelineID: argumentYAMLPipeline.PipelineID})
	require.Nil(t, err)

	assert.NotNil(t, *pipeline)
	assert.NotNil(t, pipeline.CreatedAt)
	assert.Equal(t, "arguments-parameters.yaml", pipeline.DisplayName)
}

func TestPipelineAPI(t *testing.T) {
	suite.Run(t, new(PipelineApiTest))
}

func (s *PipelineApiTest) TearDownSuite() {
	if *runIntegrationTests {
		if !*isDevMode {
			s.cleanUp()
		}
	}
}

func (s *PipelineApiTest) cleanUp() {
	test.DeleteAllPipelines(s.pipelineClient, s.T())
}
