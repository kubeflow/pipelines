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
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang/glog"
	params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_client/pipeline_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_model"
	upload_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_client/pipeline_upload_service"
	api_server "github.com/kubeflow/pipelines/backend/src/common/client/api_server/v2"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	test "github.com/kubeflow/pipelines/backend/test/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"sigs.k8s.io/yaml"
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

func (s *PipelineVersionApiTest) TestPipelineSpec() {
	t := s.T()

	test.DeleteAllPipelines(s.pipelineClient, t)

	/* ---------- Upload a pipeline YAML ---------- */
	pipelineParams := upload_params.NewUploadPipelineParams()
	pipelineName := "test-pipeline"
	pipelineParams.SetName(&pipelineName)
	pipeline, err := s.pipelineUploadClient.UploadFile("../resources/arguments-parameters.yaml", pipelineParams)
	require.Nil(t, err)
	assert.Equal(t, "test-pipeline", pipeline.DisplayName)
	// Verify that the pipeline name defaults to the display name for backwards compatibility.
	assert.Equal(t, "test-pipeline", pipeline.Name)

	/* ---------- Get pipeline id ---------- */
	pipelines, totalSize, _, err := s.pipelineClient.List(&params.PipelineServiceListPipelinesParams{})
	require.Nil(t, err)
	assert.Equal(t, 1, len(pipelines))
	assert.Equal(t, 1, totalSize)
	pipelineId := pipelines[0].PipelineID

	/* ---------- Upload a pipeline version YAML under test-pipeline ---------- */
	time.Sleep(1 * time.Second)
	pipelineVersionParams := upload_params.NewUploadPipelineVersionParams()
	pipelineVersionParams.SetPipelineid(&pipelineId)
	argumentYAMLPipelineVersion, err := s.pipelineUploadClient.UploadPipelineVersion("../resources/arguments-parameters.yaml", pipelineVersionParams)
	require.Nil(t, err)
	assert.Equal(t, "arguments-parameters.yaml", argumentYAMLPipelineVersion.DisplayName)

	/* ---------- Upload the same pipeline version again. Should fail due to name uniqueness ---------- */
	time.Sleep(1 * time.Second)
	_, err = s.pipelineUploadClient.UploadPipelineVersion("../resources/arguments-parameters.yaml", pipelineVersionParams)
	require.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed to upload pipeline version")

	/* ---------- Import pipeline version YAML by URL ---------- */
	time.Sleep(1 * time.Second)
	sequentialPipelineVersion, err := s.pipelineClient.CreatePipelineVersion(&params.PipelineServiceCreatePipelineVersionParams{
		PipelineID: pipelineId,
		PipelineVersion: &pipeline_model.V2beta1PipelineVersion{
			Name:        "sequential-v2",
			DisplayName: "sequential",
			PackageURL: &pipeline_model.V2beta1URL{
				PipelineURL: "https://raw.githubusercontent.com/kubeflow/pipelines/refs/heads/master/backend/test/v2/resources/sequential-v2.yaml",
			},
			PipelineID: pipelineId,
		},
	})
	require.Nil(t, err)
	assert.Equal(t, "sequential-v2", sequentialPipelineVersion.Name)
	assert.Equal(t, "sequential", sequentialPipelineVersion.DisplayName)

	/* ---------- Upload pipeline version zip ---------- */
	time.Sleep(1 * time.Second)
	argumentUploadPipelineVersion, err := s.pipelineUploadClient.UploadPipelineVersion(
		"../resources/arguments.pipeline.zip", &upload_params.UploadPipelineVersionParams{
			Name:       util.StringPointer("zip-arguments-parameters"),
			Pipelineid: util.StringPointer(pipelineId),
		})
	require.Nil(t, err)
	assert.Equal(t, "zip-arguments-parameters", argumentUploadPipelineVersion.DisplayName)

	/* ---------- Import pipeline tarball by URL ---------- */
	pipelineURL := "https://github.com/kubeflow/pipelines/raw/refs/heads/master/backend/test/v2/resources/arguments.pipeline.zip"

	if pullNumber := os.Getenv("PULL_NUMBER"); pullNumber != "" {
		pipelineURL = fmt.Sprintf("https://raw.githubusercontent.com/kubeflow/pipelines/pull/%s/head/backend/test/v2/resources/arguments.pipeline.zip", pullNumber)
	}

	time.Sleep(1 * time.Second)
	argumentUrlPipelineVersion, err := s.pipelineClient.CreatePipelineVersion(&params.PipelineServiceCreatePipelineVersionParams{
		PipelineID: pipelineId,
		PipelineVersion: &pipeline_model.V2beta1PipelineVersion{
			DisplayName: "arguments",
			PackageURL: &pipeline_model.V2beta1URL{
				PipelineURL: pipelineURL,
			},
			PipelineID: pipelineId,
		},
	})
	require.Nil(t, err)
	assert.Equal(t, "arguments", argumentUrlPipelineVersion.DisplayName)

	/* ---------- Verify list pipeline version works ---------- */
	pipelineVersions, totalSize, _, err := s.pipelineClient.ListPipelineVersions(&params.PipelineServiceListPipelineVersionsParams{
		PipelineID: pipelineId,
	})
	require.Nil(t, err)
	assert.Equal(t, 5, len(pipelineVersions))
	assert.Equal(t, 5, totalSize)
	for _, p := range pipelineVersions {
		assert.NotNil(t, *p)
		assert.NotNil(t, p.CreatedAt)
		assert.Contains(t, []string{"test-pipeline" /*default version created with pipeline*/, "sequential-v2", "arguments", "arguments-parameters.yaml", "zip-arguments-parameters"}, p.Name)
		assert.Contains(t, []string{"test-pipeline" /*default version created with pipeline*/, "sequential", "arguments", "arguments-parameters.yaml", "zip-arguments-parameters"}, p.DisplayName)
	}

	/* ---------- Verify list pipeline sorted by names ---------- */
	listFirstPagePipelineVersions, totalSize, nextPageToken, err := s.pipelineClient.ListPipelineVersions(
		&params.PipelineServiceListPipelineVersionsParams{
			PageSize:   util.Int32Pointer(3),
			SortBy:     util.StringPointer("name"),
			PipelineID: pipelineId,
		})
	require.Nil(t, err)
	assert.Equal(t, 3, len(listFirstPagePipelineVersions))
	assert.Equal(t, 5, totalSize)
	assert.Equal(t, "arguments", listFirstPagePipelineVersions[0].DisplayName)
	assert.Equal(t, "arguments-parameters.yaml", listFirstPagePipelineVersions[1].DisplayName)
	assert.Equal(t, "sequential", listFirstPagePipelineVersions[2].DisplayName)
	assert.NotEmpty(t, nextPageToken)

	listSecondPagePipelineVersions, totalSize, nextPageToken, err := s.pipelineClient.ListPipelineVersions(
		&params.PipelineServiceListPipelineVersionsParams{
			PageToken:  util.StringPointer(nextPageToken),
			PageSize:   util.Int32Pointer(3),
			SortBy:     util.StringPointer("name"),
			PipelineID: pipelineId,
		})
	require.Nil(t, err)
	assert.Equal(t, 2, len(listSecondPagePipelineVersions))
	assert.Equal(t, 5, totalSize)
	assert.Equal(t, "test-pipeline", listSecondPagePipelineVersions[0].DisplayName)
	assert.Equal(t, "zip-arguments-parameters", listSecondPagePipelineVersions[1].DisplayName)
	assert.Empty(t, nextPageToken)

	/* ---------- Verify list pipeline version sorted by creation time ---------- */
	listFirstPagePipelineVersions, totalSize, nextPageToken, err = s.pipelineClient.ListPipelineVersions(
		&params.PipelineServiceListPipelineVersionsParams{
			PageSize:   util.Int32Pointer(3),
			SortBy:     util.StringPointer("created_at"),
			PipelineID: pipelineId,
		})
	require.Nil(t, err)
	assert.Equal(t, 3, len(listFirstPagePipelineVersions))
	assert.Equal(t, 5, totalSize)
	assert.Equal(t, "test-pipeline", listFirstPagePipelineVersions[0].DisplayName)
	assert.Equal(t, "arguments-parameters.yaml", listFirstPagePipelineVersions[1].DisplayName)
	assert.Equal(t, "sequential", listFirstPagePipelineVersions[2].DisplayName)
	assert.NotEmpty(t, nextPageToken)

	listSecondPagePipelineVersions, totalSize, nextPageToken, err = s.pipelineClient.ListPipelineVersions(
		&params.PipelineServiceListPipelineVersionsParams{
			PageToken:  util.StringPointer(nextPageToken),
			PageSize:   util.Int32Pointer(3),
			SortBy:     util.StringPointer("created_at"),
			PipelineID: pipelineId,
		})
	require.Nil(t, err)
	assert.Equal(t, 2, len(listSecondPagePipelineVersions))
	assert.Equal(t, 5, totalSize)
	assert.Equal(t, "zip-arguments-parameters", listSecondPagePipelineVersions[0].DisplayName)
	assert.Equal(t, "arguments", listSecondPagePipelineVersions[1].DisplayName)
	assert.Empty(t, nextPageToken)

	/* ---------- List pipeline versions sort by unsupported description field. Should fail. ---------- */
	_, _, _, err = s.pipelineClient.ListPipelineVersions(&params.PipelineServiceListPipelineVersionsParams{
		PageSize:   util.Int32Pointer(2),
		SortBy:     util.StringPointer("unknownfield"),
		PipelineID: pipelineId,
	})
	assert.NotNil(t, err)

	/* ---------- List pipeline versions sorted by names descend order ---------- */
	listFirstPagePipelineVersions, totalSize, nextPageToken, err = s.pipelineClient.ListPipelineVersions(
		&params.PipelineServiceListPipelineVersionsParams{
			PageSize:   util.Int32Pointer(3),
			SortBy:     util.StringPointer("name desc"),
			PipelineID: pipelineId,
		})
	require.Nil(t, err)
	assert.Equal(t, 3, len(listFirstPagePipelineVersions))
	assert.Equal(t, 5, totalSize)
	assert.Equal(t, "zip-arguments-parameters", listFirstPagePipelineVersions[0].DisplayName)
	assert.Equal(t, "test-pipeline", listFirstPagePipelineVersions[1].DisplayName)
	assert.Equal(t, "sequential", listFirstPagePipelineVersions[2].DisplayName)
	assert.NotEmpty(t, nextPageToken)

	listSecondPagePipelineVersions, totalSize, nextPageToken, err = s.pipelineClient.ListPipelineVersions(
		&params.PipelineServiceListPipelineVersionsParams{
			PageToken:  util.StringPointer(nextPageToken),
			PageSize:   util.Int32Pointer(3),
			SortBy:     util.StringPointer("name desc"),
			PipelineID: pipelineId,
		})
	require.Nil(t, err)
	assert.Equal(t, 2, len(listSecondPagePipelineVersions))
	assert.Equal(t, 5, totalSize)
	assert.Equal(t, "arguments-parameters.yaml", listSecondPagePipelineVersions[0].DisplayName)
	assert.Equal(t, "arguments", listSecondPagePipelineVersions[1].DisplayName)
	assert.Empty(t, nextPageToken)

	/* ---------- Verify get pipeline version works ---------- */
	pipelineVersion, err := s.pipelineClient.GetPipelineVersion(&params.PipelineServiceGetPipelineVersionParams{PipelineID: argumentUrlPipelineVersion.PipelineID, PipelineVersionID: argumentUrlPipelineVersion.PipelineVersionID})
	require.Nil(t, err)
	assert.Equal(t, pipelineVersion.DisplayName, "arguments")
	assert.NotNil(t, pipelineVersion.CreatedAt)

	/* ---------- Verify pipeline spec ---------- */
	bytes, err := os.ReadFile("../resources/arguments-parameters.yaml")
	require.Nil(t, err)
	expected_bytes, err := yaml.YAMLToJSON(bytes)
	require.Nil(t, err)
	actual_bytes, err := json.Marshal(pipelineVersion.PipelineSpec)
	require.Nil(t, err)
	// Override pipeline name, then compare
	assert.Equal(t, string(expected_bytes), strings.Replace(string(actual_bytes), "pipeline/test_pipeline", "echo", 1))
}

func (s *PipelineVersionApiTest) TestV2Spec() {
	t := s.T()

	test.DeleteAllPipelines(s.pipelineClient, t)

	/* ---------- Upload a pipeline YAML ---------- */
	pipelineParams := upload_params.NewUploadPipelineParams()
	pipelineName := "test-v2-pipeline"
	pipelineParams.SetName(&pipelineName)
	pipeline, err := s.pipelineUploadClient.UploadFile("../resources/arguments-parameters.yaml", pipelineParams)
	require.Nil(t, err)
	assert.Equal(t, "test-v2-pipeline", pipeline.DisplayName)

	/* ---------- Upload a pipeline version with v2 pipeline spec YAML ---------- */
	time.Sleep(1 * time.Second)
	v2Version, err := s.pipelineUploadClient.UploadPipelineVersion(
		"../resources/hello-world.yaml", &upload_params.UploadPipelineVersionParams{
			Name:       util.StringPointer("hello-world"),
			Pipelineid: util.StringPointer(pipeline.PipelineID),
		})
	require.Nil(t, err)
	assert.Equal(t, "hello-world", v2Version.DisplayName)

	/* ---------- Verify pipeline spec ---------- */
	bytes, err := os.ReadFile("../resources/hello-world.yaml")
	require.Nil(t, err)
	expected_bytes, err := yaml.YAMLToJSON(bytes)
	require.Nil(t, err)
	actual_bytes, err := json.Marshal(v2Version.PipelineSpec)
	require.Nil(t, err)
	// Override pipeline name, then compare
	assert.Equal(t, string(expected_bytes), strings.Replace(string(actual_bytes), "pipeline/test-v2-pipeline", "echo", 1))
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
