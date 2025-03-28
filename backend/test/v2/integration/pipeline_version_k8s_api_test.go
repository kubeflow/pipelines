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
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang/glog"
	params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_client/pipeline_service"
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
type PipelineVersionK8sApiTest struct {
	suite.Suite
	namespace            string
	pipelineClient       *api_server.PipelineClient
	pipelineUploadClient *api_server.PipelineUploadClient
}

// Check the namespace have ML job installed and ready
func (s *PipelineVersionK8sApiTest) SetupTest() {
	if !*runK8sStoreIntegrationTests {
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

func (s *PipelineVersionK8sApiTest) TestPipelineSpec() {
	t := s.T()

	test.DeleteAllPipelines(s.pipelineClient, t)
	time.Sleep(5 * time.Second)

	/* ---------- Upload a pipeline YAML ---------- */
	pipelineParams := upload_params.NewUploadPipelineParams()
	pipelineName := "test-pipeline"
	pipelineParams.SetName(&pipelineName)
	pipeline, err := s.pipelineUploadClient.UploadFile("../resources/arguments-parameters.yaml", pipelineParams)
	require.Nil(t, err)
	assert.Equal(t, "test-pipeline", pipeline.DisplayName)

	/* ---------- Get pipeline id ---------- */
	pipelines, totalSize, _, err := s.pipelineClient.List(&params.PipelineServiceListPipelinesParams{})
	require.Nil(t, err)
	assert.Equal(t, 1, len(pipelines))
	assert.Equal(t, 1, totalSize)
	pipelineId := pipelines[0].PipelineID

	/* ---------- Upload a pipeline version YAML under test-pipeline ---------- */
	time.Sleep(5 * time.Second)
	pipelineVersionParams := upload_params.NewUploadPipelineVersionParams()
	pipelineVersionParams.SetPipelineid(&pipelineId)
	argumentYAMLPipelineVersion, err := s.pipelineUploadClient.UploadPipelineVersion("../resources/arguments-parameters.yaml", pipelineVersionParams)
	require.Nil(t, err)
	assert.Equal(t, "arguments-parameters.yaml", argumentYAMLPipelineVersion.DisplayName)

	/* ---------- Upload the same pipeline version again. Should fail due to name uniqueness ---------- */
	time.Sleep(5 * time.Second)
	_, err = s.pipelineUploadClient.UploadPipelineVersion("../resources/arguments-parameters.yaml", upload_params.NewUploadPipelineVersionParams())
	require.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed to upload pipeline version")

	/* ---------- Upload pipeline version zip ---------- */
	time.Sleep(5 * time.Second)
	argumentUploadPipelineVersion, err := s.pipelineUploadClient.UploadPipelineVersion(
		"../resources/arguments.pipeline.zip", &upload_params.UploadPipelineVersionParams{
			Name:       util.StringPointer("zip-arguments-parameters"),
			Pipelineid: util.StringPointer(pipelineId),
		})
	require.Nil(t, err)
	assert.Equal(t, "zip-arguments-parameters", argumentUploadPipelineVersion.DisplayName)

	/* ---------- Verify list pipeline version works ---------- */
	pipelineVersions, totalSize, _, err := s.pipelineClient.ListPipelineVersions(&params.PipelineServiceListPipelineVersionsParams{
		PipelineID: pipelineId,
	})
	require.Nil(t, err)
	assert.Equal(t, 3, len(pipelineVersions))
	assert.Equal(t, 3, totalSize)
	for _, p := range pipelineVersions {
		assert.NotNil(t, *p)
		assert.NotNil(t, p.CreatedAt)
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
	assert.Equal(t, 3, totalSize)
	assert.Equal(t, "arguments-parameters.yaml", listFirstPagePipelineVersions[0].DisplayName)
	assert.Equal(t, "test-pipeline", listFirstPagePipelineVersions[1].DisplayName)
	assert.Equal(t, "zip-arguments-parameters", listFirstPagePipelineVersions[2].DisplayName)
	assert.Empty(t, nextPageToken)

	listSecondPagePipelineVersions, totalSize, nextPageToken, err := s.pipelineClient.ListPipelineVersions(
		&params.PipelineServiceListPipelineVersionsParams{
			PageToken:  util.StringPointer(nextPageToken),
			PageSize:   util.Int32Pointer(3),
			SortBy:     util.StringPointer("name"),
			PipelineID: pipelineId,
		})
	require.Nil(t, err)
	assert.Equal(t, 3, len(listSecondPagePipelineVersions))
	assert.Equal(t, 3, totalSize)
	assert.Equal(t, "arguments-parameters.yaml", listSecondPagePipelineVersions[0].DisplayName)
	assert.Equal(t, "test-pipeline", listSecondPagePipelineVersions[1].DisplayName)
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
	assert.Equal(t, 3, totalSize)
	assert.Equal(t, "test-pipeline", listFirstPagePipelineVersions[0].DisplayName)
	assert.Equal(t, "arguments-parameters.yaml", listFirstPagePipelineVersions[1].DisplayName)
	assert.Equal(t, "zip-arguments-parameters", listFirstPagePipelineVersions[2].DisplayName)
	assert.Empty(t, nextPageToken)

	listSecondPagePipelineVersions, totalSize, nextPageToken, err = s.pipelineClient.ListPipelineVersions(
		&params.PipelineServiceListPipelineVersionsParams{
			PageToken:  util.StringPointer(nextPageToken),
			PageSize:   util.Int32Pointer(3),
			SortBy:     util.StringPointer("created_at"),
			PipelineID: pipelineId,
		})
	require.Nil(t, err)
	assert.Equal(t, 3, len(listSecondPagePipelineVersions))
	assert.Equal(t, 3, totalSize)
	assert.Equal(t, "test-pipeline", listSecondPagePipelineVersions[0].DisplayName)
	assert.Equal(t, "arguments-parameters.yaml", listSecondPagePipelineVersions[1].DisplayName)
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
	assert.Equal(t, 3, totalSize)
	assert.Equal(t, "zip-arguments-parameters", listFirstPagePipelineVersions[0].DisplayName)
	assert.Equal(t, "test-pipeline", listFirstPagePipelineVersions[1].DisplayName)
	assert.Equal(t, "arguments-parameters.yaml", listFirstPagePipelineVersions[2].DisplayName)
	assert.Empty(t, nextPageToken)

	listSecondPagePipelineVersions, totalSize, nextPageToken, err = s.pipelineClient.ListPipelineVersions(
		&params.PipelineServiceListPipelineVersionsParams{
			PageToken:  util.StringPointer(nextPageToken),
			PageSize:   util.Int32Pointer(3),
			SortBy:     util.StringPointer("name desc"),
			PipelineID: pipelineId,
		})
	require.Nil(t, err)
	assert.Equal(t, 3, len(listSecondPagePipelineVersions))
	assert.Equal(t, 3, totalSize)
	assert.Equal(t, "zip-arguments-parameters", listSecondPagePipelineVersions[0].DisplayName)
	assert.Equal(t, "test-pipeline", listSecondPagePipelineVersions[1].DisplayName)
	assert.Empty(t, nextPageToken)

	/* ---------- Verify get pipeline version works ---------- */
	pipelineVersion, err := s.pipelineClient.GetPipelineVersion(&params.PipelineServiceGetPipelineVersionParams{PipelineID: argumentUploadPipelineVersion.PipelineID, PipelineVersionID: argumentUploadPipelineVersion.PipelineVersionID})
	require.Nil(t, err)
	assert.Equal(t, pipelineVersion.DisplayName, "zip-arguments-parameters")
	assert.NotNil(t, pipelineVersion.CreatedAt)

	/* ---------- Verify pipeline spec ---------- */
	bytes, err := os.ReadFile("../resources/arguments-parameters.yaml")
	require.Nil(t, err)
	expected_bytes, err := yaml.YAMLToJSON(bytes)
	require.Nil(t, err)
	actual_bytes, err := json.Marshal(pipelineVersion.PipelineSpec)
	require.Nil(t, err)
	// Override pipeline name, then compare
	assert.Equal(t, string(expected_bytes), strings.Replace(string(actual_bytes), "pipeline/test-pipeline", "whalesay", 1))
}

func (s *PipelineVersionK8sApiTest) TestV2Spec() {
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
	time.Sleep(10 * time.Second)
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
	assert.Equal(t, string(expected_bytes), strings.Replace(string(actual_bytes), "pipeline/test_v2_pipeline", "whalesay", 1))
}

func TestPipelineVersionK8sApi(t *testing.T) {
	suite.Run(t, new(PipelineVersionK8sApiTest))
}

func (s *PipelineVersionK8sApiTest) TearDownSuite() {
	if *runK8sStoreIntegrationTests {
		if !*isDevMode {
			s.cleanUp()
		}
	}
}

func (s *PipelineVersionK8sApiTest) cleanUp() {
	// Do nothing
}
