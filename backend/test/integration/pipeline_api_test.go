package integration

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/kubeflow/pipelines/backend/test"

	"github.com/golang/glog"
	params "github.com/kubeflow/pipelines/backend/api/go_http_client/pipeline_client/pipeline_service"
	model "github.com/kubeflow/pipelines/backend/api/go_http_client/pipeline_model"

	uploadParams "github.com/kubeflow/pipelines/backend/api/go_http_client/pipeline_upload_client/pipeline_upload_service"
	"github.com/kubeflow/pipelines/backend/src/common/client/api_server"
	"github.com/kubeflow/pipelines/backend/src/common/util"
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
		err := test.WaitForReady(*namespace, *initializeTimeout)
		if err != nil {
			glog.Exitf("Failed to initialize test. Error: %s", err.Error())
		}
	}
	s.namespace = *namespace
	clientConfig := test.GetClientConfig(*namespace)
	var err error
	s.pipelineUploadClient, err = api_server.NewPipelineUploadClient(clientConfig, false)
	if err != nil {
		glog.Exitf("Failed to get pipeline upload client. Error: %s", err.Error())
	}
	s.pipelineClient, err = api_server.NewPipelineClient(clientConfig, false)
	if err != nil {
		glog.Exitf("Failed to get pipeline client. Error: %s", err.Error())
	}

	s.cleanUp()
}

func (s *PipelineApiTest) TestPipelineAPI() {
	t := s.T()

	test.DeleteAllPipelines(s.pipelineClient, t)

	/* ------ Upload v2 pipeline spec JSON --------*/
	v2HelloPipeline, err := s.pipelineUploadClient.UploadFile("../resources/v2-hello-world.json", uploadParams.NewUploadPipelineParams())
	require.Nil(t, err)
	assert.Equal(t, "v2-hello-world.json", v2HelloPipeline.Name)

	/* ---------- Upload pipelines YAML ---------- */
	time.Sleep(1 * time.Second)
	argumentYAMLPipeline, err := s.pipelineUploadClient.UploadFile("../resources/arguments-parameters.yaml", uploadParams.NewUploadPipelineParams())
	require.Nil(t, err)
	assert.Equal(t, "arguments-parameters.yaml", argumentYAMLPipeline.Name)

	/* ---------- Upload the same pipeline again. Should fail due to name uniqueness ---------- */
	_, err = s.pipelineUploadClient.UploadFile("../resources/arguments-parameters.yaml", uploadParams.NewUploadPipelineParams())
	require.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed to upload pipeline.")

	/* ---------- Import pipeline YAML by URL ---------- */
	time.Sleep(1 * time.Second)
	sequentialPipeline, err := s.pipelineClient.Create(&params.CreatePipelineParams{
		Body: &model.APIPipeline{Name: "sequential", URL: &model.APIURL{
			PipelineURL: "https://storage.googleapis.com/ml-pipeline-dataset/sequential.yaml"}}})
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
	argumentUrlPipeline, err := s.pipelineClient.Create(&params.CreatePipelineParams{
		Body: &model.APIPipeline{URL: &model.APIURL{
			PipelineURL: "https://storage.googleapis.com/ml-pipeline-dataset/arguments.pipeline.zip"}}})
	require.Nil(t, err)
	assert.Equal(t, "arguments.pipeline.zip", argumentUrlPipeline.Name)

	/* ---------- Verify list pipeline works ---------- */
	pipelines, totalSize, _, err := s.pipelineClient.List(&params.ListPipelinesParams{})
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
		&params.ListPipelinesParams{PageSize: util.Int32Pointer(2), SortBy: util.StringPointer("name")})
	require.Nil(t, err)
	assert.Equal(t, 2, len(listFirstPagePipelines))
	assert.Equal(t, 5, totalSize)
	assert.Equal(t, "arguments-parameters.yaml", listFirstPagePipelines[0].Name)
	assert.Equal(t, "arguments.pipeline.zip", listFirstPagePipelines[1].Name)
	assert.NotEmpty(t, nextPageToken)

	listSecondPagePipelines, totalSize, nextPageToken, err := s.pipelineClient.List(
		&params.ListPipelinesParams{PageToken: util.StringPointer(nextPageToken), PageSize: util.Int32Pointer(3), SortBy: util.StringPointer("name")})
	require.Nil(t, err)
	assert.Equal(t, 3, len(listSecondPagePipelines))
	assert.Equal(t, 5, totalSize)
	assert.Equal(t, "sequential", listSecondPagePipelines[0].Name)
	assert.Equal(t, "v2-hello-world.json", listSecondPagePipelines[1].Name)
	assert.Equal(t, "zip-arguments-parameters", listSecondPagePipelines[2].Name)
	assert.Empty(t, nextPageToken)

	/* ---------- Verify list pipeline sorted by creation time ---------- */
	listFirstPagePipelines, totalSize, nextPageToken, err = s.pipelineClient.List(
		&params.ListPipelinesParams{PageSize: util.Int32Pointer(3), SortBy: util.StringPointer("created_at")})
	require.Nil(t, err)
	assert.Equal(t, 3, len(listFirstPagePipelines))
	assert.Equal(t, 5, totalSize)
	assert.Equal(t, "v2-hello-world.json", listFirstPagePipelines[0].Name)
	assert.Equal(t, "arguments-parameters.yaml", listFirstPagePipelines[1].Name)
	assert.Equal(t, "sequential", listFirstPagePipelines[2].Name)
	assert.NotEmpty(t, nextPageToken)

	listSecondPagePipelines, totalSize, nextPageToken, err = s.pipelineClient.List(
		&params.ListPipelinesParams{PageToken: util.StringPointer(nextPageToken), PageSize: util.Int32Pointer(3), SortBy: util.StringPointer("created_at")})
	require.Nil(t, err)
	assert.Equal(t, 2, len(listSecondPagePipelines))
	assert.Equal(t, 5, totalSize)
	assert.Equal(t, "zip-arguments-parameters", listSecondPagePipelines[0].Name)
	assert.Equal(t, "arguments.pipeline.zip", listSecondPagePipelines[1].Name)
	assert.Empty(t, nextPageToken)

	/* ---------- List pipelines sort by unsupported description field. Should fail. ---------- */
	_, _, _, err = s.pipelineClient.List(&params.ListPipelinesParams{
		PageSize: util.Int32Pointer(2), SortBy: util.StringPointer("unknownfield")})
	assert.NotNil(t, err)

	/* ---------- List pipelines sorted by names descend order ---------- */
	listFirstPagePipelines, totalSize, nextPageToken, err = s.pipelineClient.List(
		&params.ListPipelinesParams{PageSize: util.Int32Pointer(3), SortBy: util.StringPointer("name desc")})
	require.Nil(t, err)
	assert.Equal(t, 3, len(listFirstPagePipelines))
	assert.Equal(t, 5, totalSize)
	assert.Equal(t, "zip-arguments-parameters", listFirstPagePipelines[0].Name)
	assert.Equal(t, "v2-hello-world.json", listFirstPagePipelines[1].Name)
	assert.Equal(t, "sequential", listFirstPagePipelines[2].Name)
	assert.NotEmpty(t, nextPageToken)

	listSecondPagePipelines, totalSize, nextPageToken, err = s.pipelineClient.List(&params.ListPipelinesParams{
		PageToken: util.StringPointer(nextPageToken), PageSize: util.Int32Pointer(3), SortBy: util.StringPointer("name desc")})
	require.Nil(t, err)
	assert.Equal(t, 2, len(listSecondPagePipelines))
	assert.Equal(t, 5, totalSize)
	assert.Equal(t, "arguments.pipeline.zip", listSecondPagePipelines[0].Name)
	assert.Equal(t, "arguments-parameters.yaml", listSecondPagePipelines[1].Name)
	assert.Empty(t, nextPageToken)

	/* ---------- Verify get pipeline works ---------- */
	pipeline, err := s.pipelineClient.Get(&params.GetPipelineParams{ID: argumentYAMLPipeline.ID})
	require.Nil(t, err)
	verifyPipeline(t, pipeline)

	/* ---------- Verify get template works ---------- */
	template, err := s.pipelineClient.GetTemplate(&params.GetTemplateParams{ID: argumentYAMLPipeline.ID})
	require.Nil(t, err)
	bytes, err := ioutil.ReadFile("../resources/arguments-parameters.yaml")
	require.Nil(t, err)
	expected, err := util.NewTemplate(bytes)
	assert.Equal(t, expected, template)

	template, err = s.pipelineClient.GetTemplate(&params.GetTemplateParams{ID: v2HelloPipeline.ID})
	require.Nil(t, err)
	bytes, err = ioutil.ReadFile("../resources/v2-hello-world.json")
	require.Nil(t, err)
	expected, err = util.NewTemplate(bytes)
	expected.OverrideV2PipelineName("v2-hello-world.json", "")
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
			CreatedAt: pipeline.CreatedAt,
			ID:        pipeline.ID,
			Name:      "arguments-parameters.yaml",
			Parameters: []*model.APIParameter{
				{Name: "param1", Value: "hello"},
				{Name: "param2"}},
			ResourceReferences: []*model.APIResourceReference{{
				Key:          &model.APIResourceKey{ID: pipeline.ID, Type: model.APIResourceTypePIPELINE},
				Relationship: model.APIRelationshipOWNER}}},
	}
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
