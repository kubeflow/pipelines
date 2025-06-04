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
	"os"
	"testing"
	"time"

	"github.com/golang/glog"
	params "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/pipeline_client/pipeline_service"
	"github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/pipeline_model"
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
// - upload yaml file
// - upload tarball file
// - providing YAML file url
// - Providing tarball file url
type PipelineVersionApiTest struct {
	suite.Suite
	namespace            string
	pipelineClient       *api_server.PipelineClient
	pipelineUploadClient *api_server.PipelineUploadClient
}

// Check the namespace have ML job installed and ready
func (s *PipelineVersionApiTest) SetupTest() {
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

	var newPipelineUploadClient func() (*api_server.PipelineUploadClient, error)
	var newPipelineClient func() (*api_server.PipelineClient, error)

	if *isKubeflowMode {
		s.namespace = *namespace

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

func (s *PipelineVersionApiTest) TestArgoSpec() {
	t := s.T()

	test.DeleteAllPipelines(s.pipelineClient, t)

	/* ---------- Upload a pipeline YAML ---------- */
	pipelineParams := uploadParams.NewUploadPipelineParams()
	pipelineName := "test_pipeline"
	pipelineParams.SetName(&pipelineName)
	pipeline, err := s.pipelineUploadClient.UploadFile("../resources/arguments-parameters.yaml", pipelineParams)
	require.Nil(t, err)
	assert.Equal(t, "test_pipeline", pipeline.Name)

	/* ---------- Get pipeline id ---------- */
	pipelines, totalSize, _, err := s.pipelineClient.List(&params.PipelineServiceListPipelinesV1Params{})
	require.Nil(t, err)
	assert.Equal(t, 1, len(pipelines))
	assert.Equal(t, 1, totalSize)
	pipelineId := pipelines[0].ID

	/* ---------- Upload a pipeline version YAML under test_pipeline ---------- */
	time.Sleep(1 * time.Second)
	pipelineVersionParams := uploadParams.NewUploadPipelineVersionParams()
	pipelineVersionParams.SetPipelineid(&pipelineId)
	argumentYAMLPipelineVersion, err := s.pipelineUploadClient.UploadPipelineVersion("../resources/arguments-parameters.yaml", pipelineVersionParams)
	require.Nil(t, err)
	assert.Equal(t, "arguments-parameters.yaml", argumentYAMLPipelineVersion.Name)

	/* ---------- Update pipeline default version ---------- */
	time.Sleep(1 * time.Second)
	sortBy := "created_at"
	versions, _, _, err := s.pipelineClient.ListPipelineVersions(&params.PipelineServiceListPipelineVersionsV1Params{ResourceKeyID: &pipelineId, SortBy: &sortBy})
	require.Nil(t, err)

	time.Sleep(1 * time.Second)
	pipelineSelected, err := s.pipelineClient.Get(&params.PipelineServiceGetPipelineV1Params{ID: pipelineId})
	require.Nil(t, err)
	assert.Equal(t, pipelineSelected.DefaultVersion.ID, versions[1].ID)

	/* ---------- Upload the same pipeline version again. Should fail due to name uniqueness ---------- */
	time.Sleep(1 * time.Second)
	_, err = s.pipelineUploadClient.UploadPipelineVersion("../resources/arguments-parameters.yaml", uploadParams.NewUploadPipelineVersionParams())
	require.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed to upload pipeline version")

	/* ---------- Import pipeline version YAML by URL ---------- */
	time.Sleep(1 * time.Second)
	sequentialPipelineVersion, err := s.pipelineClient.CreatePipelineVersion(&params.PipelineServiceCreatePipelineVersionV1Params{
		Body: &pipeline_model.APIPipelineVersion{
			Name: "sequential",
			PackageURL: &pipeline_model.APIURL{
				PipelineURL: "https://raw.githubusercontent.com/kubeflow/pipelines/refs/heads/master/backend/test/v2/resources/sequential.yaml",
			},
			ResourceReferences: []*pipeline_model.APIResourceReference{
				{
					Key:          &pipeline_model.APIResourceKey{Type: pipeline_model.APIResourceTypePIPELINE, ID: pipelineId},
					Relationship: pipeline_model.APIRelationshipOWNER,
				},
			},
		},
	})
	require.Nil(t, err)
	assert.Equal(t, "sequential", sequentialPipelineVersion.Name)

	/* ---------- Upload pipeline version zip ---------- */
	time.Sleep(1 * time.Second)
	argumentUploadPipelineVersion, err := s.pipelineUploadClient.UploadPipelineVersion(
		"../resources/arguments.pipeline.zip", &uploadParams.UploadPipelineVersionParams{
			Name:       util.StringPointer("zip-arguments-parameters"),
			Pipelineid: util.StringPointer(pipelineId),
		})
	require.Nil(t, err)
	assert.Equal(t, "zip-arguments-parameters", argumentUploadPipelineVersion.Name)

	/* ---------- Import pipeline tarball by URL ---------- */
	time.Sleep(1 * time.Second)
	pipelineURL := "https://github.com/kubeflow/pipelines/raw/refs/heads/master/backend/test/resources/arguments.pipeline.zip"

	if pullNumber := os.Getenv("PULL_NUMBER"); pullNumber != "" {
		pipelineURL = fmt.Sprintf("https://raw.githubusercontent.com/kubeflow/pipelines/pull/%s/head/backend/test/resources/arguments.pipeline.zip", pullNumber)
	}

	argumentUrlPipelineVersion, err := s.pipelineClient.CreatePipelineVersion(&params.PipelineServiceCreatePipelineVersionV1Params{
		Body: &pipeline_model.APIPipelineVersion{
			Name: "arguments",
			PackageURL: &pipeline_model.APIURL{
				PipelineURL: pipelineURL,
			},
			ResourceReferences: []*pipeline_model.APIResourceReference{
				{
					Key:          &pipeline_model.APIResourceKey{Type: pipeline_model.APIResourceTypePIPELINE, ID: pipelineId},
					Relationship: pipeline_model.APIRelationshipOWNER,
				},
			},
		},
	})
	require.Nil(t, err)
	assert.Equal(t, "arguments", argumentUrlPipelineVersion.Name)

	/* ---------- Verify list pipeline version works ---------- */
	pipelineVersions, totalSize, _, err := s.pipelineClient.ListPipelineVersions(&params.PipelineServiceListPipelineVersionsV1Params{
		ResourceKeyID:   util.StringPointer(pipelineId),
		ResourceKeyType: util.StringPointer("PIPELINE"),
	})
	require.Nil(t, err)
	assert.Equal(t, 5, len(pipelineVersions))
	assert.Equal(t, 5, totalSize)
	for _, p := range pipelineVersions {
		assert.NotNil(t, *p)
		assert.NotNil(t, p.CreatedAt)
		assert.Contains(t, []string{"test_pipeline" /*default version created with pipeline*/, "sequential", "arguments", "arguments-parameters.yaml", "zip-arguments-parameters"}, p.Name)

		if p.Name == "arguments" {
			assert.Equal(t, p.Parameters,
				[]*pipeline_model.APIParameter{
					{Name: "param1", Value: "hello"}, // Default value in the pipeline template
					{Name: "param2"},                 // No default value in the pipeline
				}, "Found wrong parameters in the pipeline version: %s", p.Name)
		}
	}

	/* ---------- Verify list pipeline sorted by names ---------- */
	listFirstPagePipelineVersions, totalSize, nextPageToken, err := s.pipelineClient.ListPipelineVersions(
		&params.PipelineServiceListPipelineVersionsV1Params{
			PageSize:        util.Int32Pointer(3),
			SortBy:          util.StringPointer("name"),
			ResourceKeyID:   util.StringPointer(pipelineId),
			ResourceKeyType: util.StringPointer("PIPELINE"),
		})
	require.Nil(t, err)
	assert.Equal(t, 3, len(listFirstPagePipelineVersions))
	assert.Equal(t, 5, totalSize)
	assert.Equal(t, "arguments", listFirstPagePipelineVersions[0].Name)
	assert.Equal(t, "arguments-parameters.yaml", listFirstPagePipelineVersions[1].Name)
	assert.Equal(t, "sequential", listFirstPagePipelineVersions[2].Name)
	assert.NotEmpty(t, nextPageToken)

	listSecondPagePipelineVersions, totalSize, nextPageToken, err := s.pipelineClient.ListPipelineVersions(
		&params.PipelineServiceListPipelineVersionsV1Params{
			PageToken:       util.StringPointer(nextPageToken),
			PageSize:        util.Int32Pointer(3),
			SortBy:          util.StringPointer("name"),
			ResourceKeyID:   util.StringPointer(pipelineId),
			ResourceKeyType: util.StringPointer("PIPELINE"),
		})
	require.Nil(t, err)
	assert.Equal(t, 2, len(listSecondPagePipelineVersions))
	assert.Equal(t, 5, totalSize)
	assert.Equal(t, "test_pipeline", listSecondPagePipelineVersions[0].Name)
	assert.Equal(t, "zip-arguments-parameters", listSecondPagePipelineVersions[1].Name)
	assert.Empty(t, nextPageToken)

	/* ---------- Verify list pipeline version sorted by creation time ---------- */
	listFirstPagePipelineVersions, totalSize, nextPageToken, err = s.pipelineClient.ListPipelineVersions(
		&params.PipelineServiceListPipelineVersionsV1Params{
			PageSize:        util.Int32Pointer(3),
			SortBy:          util.StringPointer("created_at"),
			ResourceKeyID:   util.StringPointer(pipelineId),
			ResourceKeyType: util.StringPointer("PIPELINE"),
		})
	require.Nil(t, err)
	assert.Equal(t, 3, len(listFirstPagePipelineVersions))
	assert.Equal(t, 5, totalSize)
	assert.Equal(t, "test_pipeline", listFirstPagePipelineVersions[0].Name)
	assert.Equal(t, "arguments-parameters.yaml", listFirstPagePipelineVersions[1].Name)
	assert.Equal(t, "sequential", listFirstPagePipelineVersions[2].Name)
	assert.NotEmpty(t, nextPageToken)

	listSecondPagePipelineVersions, totalSize, nextPageToken, err = s.pipelineClient.ListPipelineVersions(
		&params.PipelineServiceListPipelineVersionsV1Params{
			PageToken:       util.StringPointer(nextPageToken),
			PageSize:        util.Int32Pointer(3),
			SortBy:          util.StringPointer("created_at"),
			ResourceKeyID:   util.StringPointer(pipelineId),
			ResourceKeyType: util.StringPointer("PIPELINE"),
		})
	require.Nil(t, err)
	assert.Equal(t, 2, len(listSecondPagePipelineVersions))
	assert.Equal(t, 5, totalSize)
	assert.Equal(t, "zip-arguments-parameters", listSecondPagePipelineVersions[0].Name)
	assert.Equal(t, "arguments", listSecondPagePipelineVersions[1].Name)
	assert.Empty(t, nextPageToken)

	/* ---------- List pipeline versions sort by unsupported description field. Should fail. ---------- */
	_, _, _, err = s.pipelineClient.ListPipelineVersions(&params.PipelineServiceListPipelineVersionsV1Params{
		PageSize:        util.Int32Pointer(2),
		SortBy:          util.StringPointer("unknownfield"),
		ResourceKeyID:   util.StringPointer(pipelineId),
		ResourceKeyType: util.StringPointer("PIPELINE"),
	})
	assert.NotNil(t, err)

	/* ---------- List pipeline versions sorted by names descend order ---------- */
	listFirstPagePipelineVersions, totalSize, nextPageToken, err = s.pipelineClient.ListPipelineVersions(
		&params.PipelineServiceListPipelineVersionsV1Params{
			PageSize:        util.Int32Pointer(3),
			SortBy:          util.StringPointer("name desc"),
			ResourceKeyID:   util.StringPointer(pipelineId),
			ResourceKeyType: util.StringPointer("PIPELINE"),
		})
	require.Nil(t, err)
	assert.Equal(t, 3, len(listFirstPagePipelineVersions))
	assert.Equal(t, 5, totalSize)
	assert.Equal(t, "zip-arguments-parameters", listFirstPagePipelineVersions[0].Name)
	assert.Equal(t, "test_pipeline", listFirstPagePipelineVersions[1].Name)
	assert.Equal(t, "sequential", listFirstPagePipelineVersions[2].Name)
	assert.NotEmpty(t, nextPageToken)

	listSecondPagePipelineVersions, totalSize, nextPageToken, err = s.pipelineClient.ListPipelineVersions(
		&params.PipelineServiceListPipelineVersionsV1Params{
			PageToken:       util.StringPointer(nextPageToken),
			PageSize:        util.Int32Pointer(3),
			SortBy:          util.StringPointer("name desc"),
			ResourceKeyID:   util.StringPointer(pipelineId),
			ResourceKeyType: util.StringPointer("PIPELINE"),
		})
	require.Nil(t, err)
	assert.Equal(t, 2, len(listSecondPagePipelineVersions))
	assert.Equal(t, 5, totalSize)
	assert.Equal(t, "arguments-parameters.yaml", listSecondPagePipelineVersions[0].Name)
	assert.Equal(t, "arguments", listSecondPagePipelineVersions[1].Name)
	assert.Empty(t, nextPageToken)

	/* ---------- Verify get pipeline version works ---------- */
	pipelineVersion, err := s.pipelineClient.GetPipelineVersion(&params.PipelineServiceGetPipelineVersionV1Params{VersionID: argumentUrlPipelineVersion.ID})
	require.Nil(t, err)
	assert.Equal(t, pipelineVersion.Name, "arguments")
	assert.NotNil(t, pipelineVersion.CreatedAt)
	assert.Equal(t, pipelineVersion.Parameters,
		[]*pipeline_model.APIParameter{
			{Name: "param1", Value: "hello"},
			{Name: "param2"},
		}, "Found wrong parameters in the pipeline version: %s", pipelineVersion.Name)

	/* ---------- Verify get template works ---------- */
	template, err := s.pipelineClient.GetPipelineVersionTemplate(&params.PipelineServiceGetPipelineVersionTemplateParams{VersionID: argumentYAMLPipelineVersion.ID})
	require.Nil(t, err)
	bytes, err := os.ReadFile("../resources/arguments-parameters.yaml")
	require.Nil(t, err)
	expected, err := pipelinetemplate.New(bytes, true, nil)
	require.Nil(t, err)
	assert.Equal(t, expected, template)
}

func (s *PipelineVersionApiTest) TestV2Spec() {
	t := s.T()

	test.DeleteAllPipelines(s.pipelineClient, t)

	/* ---------- Upload a pipeline YAML ---------- */
	pipelineParams := uploadParams.NewUploadPipelineParams()
	pipelineName := "test_v2_pipeline"
	pipelineParams.SetName(&pipelineName)
	pipeline, err := s.pipelineUploadClient.UploadFile("../resources/arguments-parameters.yaml", pipelineParams)
	require.Nil(t, err)
	assert.Equal(t, "test_v2_pipeline", pipeline.Name)

	/* ---------- Upload a pipeline version with v2 pipeline spec YAML ---------- */
	time.Sleep(1 * time.Second)
	v2Version, err := s.pipelineUploadClient.UploadPipelineVersion(
		"../resources/v2-hello-world.yaml", &uploadParams.UploadPipelineVersionParams{
			Name:       util.StringPointer("v2-hello-world"),
			Pipelineid: util.StringPointer(pipeline.ID),
		})
	require.Nil(t, err)
	assert.Equal(t, "v2-hello-world", v2Version.Name)

	/* ---------- Verify get template works ---------- */
	template, err := s.pipelineClient.GetPipelineVersionTemplate(&params.PipelineServiceGetPipelineVersionTemplateParams{VersionID: v2Version.ID})
	require.Nil(t, err)
	bytes, err := os.ReadFile("../resources/v2-hello-world.yaml")
	require.Nil(t, err)
	expected, err := pipelinetemplate.New(bytes, true, nil)
	require.Nil(t, err)
	assert.Equal(t, expected, template, "Discrepancy found in template's pipeline name. Created pipeline's name - %s.", pipeline.Name)
}

func TestPipelineVersionAPI(t *testing.T) {
	suite.Run(t, new(PipelineVersionApiTest))
}

func (s *PipelineVersionApiTest) TearDownSuite() {
	if *runIntegrationTests {
		if !*isDevMode {
			s.cleanUp()
		}
	}
}

func (s *PipelineVersionApiTest) cleanUp() {
	// Delete pipelines and their pipeline versions
	test.DeleteAllPipelines(s.pipelineClient, s.T())
}
