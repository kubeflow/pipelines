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
	"io/ioutil"
	"testing"
	"time"

	"github.com/golang/glog"
	params "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/pipeline_client/pipeline_service"
	model "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/pipeline_model"
	uploadParams "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/pipeline_upload_client/pipeline_upload_service"
	pipelinetemplate "github.com/kubeflow/pipelines/backend/src/apiserver/template"
	api_server "github.com/kubeflow/pipelines/backend/src/common/client/api_server/v1"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/test"
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
	v2HelloPipeline, err := s.pipelineUploadClient.UploadFile("../resources/v2-hello-world.yaml", uploadParams.NewUploadPipelineParams())
	require.Nil(t, err)
	assert.Equal(t, "v2-hello-world.yaml", v2HelloPipeline.Name)

	/* ---------- Upload pipelines YAML ---------- */
	time.Sleep(1 * time.Second)
	argumentYAMLPipeline, err := s.pipelineUploadClient.UploadFile("../resources/arguments-parameters.yaml", uploadParams.NewUploadPipelineParams())
	require.Nil(t, err)
	assert.Equal(t, "arguments-parameters.yaml", argumentYAMLPipeline.Name)

	/* ---------- Upload the same pipeline again. Should fail due to name uniqueness ---------- */
	_, err = s.pipelineUploadClient.UploadFile("../resources/arguments-parameters.yaml", uploadParams.NewUploadPipelineParams())
	require.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed to upload pipeline")

	/* ---------- Import pipeline YAML by URL ---------- */
	time.Sleep(1 * time.Second)
	sequentialPipeline, err := s.pipelineClient.Create(&params.PipelineServiceCreatePipelineV1Params{
		Body: &model.APIPipeline{Name: "sequential", URL: &model.APIURL{
			PipelineURL: "https://storage.googleapis.com/ml-pipeline-dataset/sequential.yaml",
		}},
	})
	require.Nil(t, err)
	assert.Equal(t, "sequential", sequentialPipeline.Name)

	/* ---------- Upload pipelines zip ---------- */
	time.Sleep(1 * time.Second)
	argumentUploadPipeline, err := s.pipelineUploadClient.UploadFile(
		"../resources/arguments.pipeline.zip", &uploadParams.UploadPipelineParams{Name: util.StringPointer("zip-arguments-parameters")})
	require.Nil(t, err)
	assert.Equal(t, "zip-arguments-parameters", argumentUploadPipeline.Name)

	/* ---------- Import pipeline tarball by URL ---------- */
	time.Sleep(1 * time.Second)
	argumentUrlPipeline, err := s.pipelineClient.Create(&params.PipelineServiceCreatePipelineV1Params{
		Body: &model.APIPipeline{URL: &model.APIURL{
			PipelineURL: "https://storage.googleapis.com/ml-pipeline-dataset/arguments.pipeline.zip",
		}},
	})
	require.Nil(t, err)
	assert.Equal(t, "arguments.pipeline.zip", argumentUrlPipeline.Name)

	/* ---------- Verify list pipeline works ---------- */
	pipelines, totalSize, _, err := s.pipelineClient.List(&params.PipelineServiceListPipelinesV1Params{})
	require.Nil(t, err)
	assert.Equal(t, 5, len(pipelines))
	assert.Equal(t, 5, totalSize)
	for _, p := range pipelines {
		// Sampling one of the pipelines and verify the result is expected.
		if p.Name == "arguments-parameters.yaml" {
			verifyPipeline(t, p)
		}
	}

	/* ---------- Verify list pipeline sorted by names ---------- */
	listFirstPagePipelines, totalSize, nextPageToken, err := s.pipelineClient.List(
		&params.PipelineServiceListPipelinesV1Params{PageSize: util.Int32Pointer(2), SortBy: util.StringPointer("name")})
	require.Nil(t, err)
	assert.Equal(t, 2, len(listFirstPagePipelines))
	assert.Equal(t, 5, totalSize)
	assert.Equal(t, "arguments-parameters.yaml", listFirstPagePipelines[0].Name)
	assert.Equal(t, "arguments.pipeline.zip", listFirstPagePipelines[1].Name)
	assert.NotEmpty(t, nextPageToken)

	listSecondPagePipelines, totalSize, nextPageToken, err := s.pipelineClient.List(
		&params.PipelineServiceListPipelinesV1Params{PageToken: util.StringPointer(nextPageToken), PageSize: util.Int32Pointer(3), SortBy: util.StringPointer("name")})
	require.Nil(t, err)
	assert.Equal(t, 3, len(listSecondPagePipelines))
	assert.Equal(t, 5, totalSize)
	assert.Equal(t, "sequential", listSecondPagePipelines[0].Name)
	assert.Equal(t, "v2-hello-world.yaml", listSecondPagePipelines[1].Name)
	assert.Equal(t, "zip-arguments-parameters", listSecondPagePipelines[2].Name)
	assert.Empty(t, nextPageToken)

	/* ---------- Verify list pipeline sorted by creation time ---------- */
	listFirstPagePipelines, totalSize, nextPageToken, err = s.pipelineClient.List(
		&params.PipelineServiceListPipelinesV1Params{PageSize: util.Int32Pointer(3), SortBy: util.StringPointer("created_at")})
	require.Nil(t, err)
	assert.Equal(t, 3, len(listFirstPagePipelines))
	assert.Equal(t, 5, totalSize)
	assert.Equal(t, "v2-hello-world.yaml", listFirstPagePipelines[0].Name)
	assert.Equal(t, "arguments-parameters.yaml", listFirstPagePipelines[1].Name)
	assert.Equal(t, "sequential", listFirstPagePipelines[2].Name)
	assert.NotEmpty(t, nextPageToken)

	listSecondPagePipelines, totalSize, nextPageToken, err = s.pipelineClient.List(
		&params.PipelineServiceListPipelinesV1Params{PageToken: util.StringPointer(nextPageToken), PageSize: util.Int32Pointer(3), SortBy: util.StringPointer("created_at")})
	require.Nil(t, err)
	assert.Equal(t, 2, len(listSecondPagePipelines))
	assert.Equal(t, 5, totalSize)
	assert.Equal(t, "zip-arguments-parameters", listSecondPagePipelines[0].Name)
	assert.Equal(t, "arguments.pipeline.zip", listSecondPagePipelines[1].Name)
	assert.Empty(t, nextPageToken)

	/* ---------- List pipelines sort by unsupported description field. Should fail. ---------- */
	_, _, _, err = s.pipelineClient.List(&params.PipelineServiceListPipelinesV1Params{
		PageSize: util.Int32Pointer(2), SortBy: util.StringPointer("unknownfield"),
	})
	assert.NotNil(t, err)

	/* ---------- List pipelines sorted by names descend order ---------- */
	listFirstPagePipelines, totalSize, nextPageToken, err = s.pipelineClient.List(
		&params.PipelineServiceListPipelinesV1Params{PageSize: util.Int32Pointer(3), SortBy: util.StringPointer("name desc")})
	require.Nil(t, err)
	assert.Equal(t, 3, len(listFirstPagePipelines))
	assert.Equal(t, 5, totalSize)
	assert.Equal(t, "zip-arguments-parameters", listFirstPagePipelines[0].Name)
	assert.Equal(t, "v2-hello-world.yaml", listFirstPagePipelines[1].Name)
	assert.Equal(t, "sequential", listFirstPagePipelines[2].Name)
	assert.NotEmpty(t, nextPageToken)

	listSecondPagePipelines, totalSize, nextPageToken, err = s.pipelineClient.List(&params.PipelineServiceListPipelinesV1Params{
		PageToken: util.StringPointer(nextPageToken), PageSize: util.Int32Pointer(3), SortBy: util.StringPointer("name desc"),
	})
	require.Nil(t, err)
	assert.Equal(t, 2, len(listSecondPagePipelines))
	assert.Equal(t, 5, totalSize)
	assert.Equal(t, "arguments.pipeline.zip", listSecondPagePipelines[0].Name)
	assert.Equal(t, "arguments-parameters.yaml", listSecondPagePipelines[1].Name)
	assert.Empty(t, nextPageToken)

	/* ---------- Verify get pipeline works ---------- */
	pipeline, err := s.pipelineClient.Get(&params.PipelineServiceGetPipelineV1Params{ID: argumentYAMLPipeline.ID})
	require.Nil(t, err)
	verifyPipeline(t, pipeline)

	/* ---------- Verify get template works ---------- */
	template, err := s.pipelineClient.GetTemplate(&params.PipelineServiceGetTemplateParams{ID: argumentYAMLPipeline.ID})
	require.Nil(t, err)
	bytes, err := ioutil.ReadFile("../resources/arguments-parameters.yaml")
	require.Nil(t, err)
	expected, _ := pipelinetemplate.New(bytes)
	assert.Equal(t, expected, template)

	template, err = s.pipelineClient.GetTemplate(&params.PipelineServiceGetTemplateParams{ID: v2HelloPipeline.ID})
	require.Nil(t, err)
	bytes, err = ioutil.ReadFile("../resources/v2-hello-world.yaml")
	require.Nil(t, err)
	expected, _ = pipelinetemplate.New(bytes)
	assert.Equal(t, expected, template)
}

func verifyPipeline(t *testing.T, pipeline *model.APIPipeline) {
	assert.NotNil(t, *pipeline)
	assert.NotNil(t, pipeline.CreatedAt)
	expected := model.APIPipeline{
		ID:        pipeline.ID,
		CreatedAt: pipeline.CreatedAt,
		Name:      "arguments-parameters.yaml",
		Parameters: []*model.APIParameter{
			{Name: "param1", Value: "hello"}, // Default value in the pipeline template
			{Name: "param2"},                 // No default value in the pipeline
		},
		DefaultVersion: &model.APIPipelineVersion{
			CreatedAt: pipeline.DefaultVersion.CreatedAt,
			ID:        pipeline.DefaultVersion.ID,
			Name:      "arguments-parameters.yaml",
			Parameters: []*model.APIParameter{
				{Name: "param1", Value: "hello"},
				{Name: "param2"},
			},
			ResourceReferences: []*model.APIResourceReference{{
				Key:          &model.APIResourceKey{ID: pipeline.ID, Type: model.APIResourceTypePIPELINE},
				Relationship: model.APIRelationshipOWNER,
			}},
		},
	}
	assert.True(t, test.VerifyPipelineResourceReferences(pipeline.ResourceReferences, expected.ResourceReferences))
	expected.ResourceReferences = pipeline.ResourceReferences
	expected.URL = pipeline.URL
	assert.True(t, test.VerifyPipelineResourceReferences(pipeline.DefaultVersion.ResourceReferences, expected.DefaultVersion.ResourceReferences))
	expected.DefaultVersion.ResourceReferences = pipeline.DefaultVersion.ResourceReferences
	expected.DefaultVersion.PackageURL = pipeline.DefaultVersion.PackageURL
	assert.Equal(t, expected, *pipeline)
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
