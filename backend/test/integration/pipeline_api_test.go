package integration

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/kubeflow/pipelines/backend/test"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/golang/glog"
	params "github.com/kubeflow/pipelines/backend/api/go_http_client/pipeline_client/pipeline_service"
	model "github.com/kubeflow/pipelines/backend/api/go_http_client/pipeline_model"

	"github.com/kubeflow/pipelines/backend/api/go_http_client/pipeline_model"
	uploadParams "github.com/kubeflow/pipelines/backend/api/go_http_client/pipeline_upload_client/pipeline_upload_service"
	"github.com/kubeflow/pipelines/backend/src/common/client/api_server"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// This test suit tests various methods to import pipeline to pipeline system, including
// - upload yaml file
// - upload tarball file
// - providing YAML file url
// - Providing tarball file url
type PipelineApiTest struct {
	suite.Suite
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

	/* ---------- Upload pipelines YAML ---------- */
	argumentYAMLPipeline, err := s.pipelineUploadClient.UploadFile("../resources/arguments-parameters.yaml", uploadParams.NewUploadPipelineParams())
	assert.Nil(t, err)
	assert.Equal(t, "arguments-parameters.yaml", argumentYAMLPipeline.Name)

	/* ---------- Upload the same pipeline again. Should fail due to name uniqueness ---------- */
	_, err = s.pipelineUploadClient.UploadFile("../resources/arguments-parameters.yaml", uploadParams.NewUploadPipelineParams())
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed to upload pipeline.")

	/* ---------- Import pipeline YAML by URL ---------- */
	time.Sleep(1 * time.Second)
	sequentialPipeline, err := s.pipelineClient.Create(&params.CreatePipelineParams{
		Body: &pipeline_model.APIPipeline{Name: "sequential", URL: &pipeline_model.APIURL{
			PipelineURL: "https://storage.googleapis.com/ml-pipeline-dataset/sequential.yaml"}}})
	assert.Nil(t, err)
	assert.Equal(t, "sequential", sequentialPipeline.Name)

	/* ---------- Upload pipelines zip ---------- */
	time.Sleep(1 * time.Second)
	argumentUploadPipeline, err := s.pipelineUploadClient.UploadFile(
		"../resources/arguments.pipeline.zip", &uploadParams.UploadPipelineParams{Name: util.StringPointer("zip-arguments-parameters")})
	assert.Nil(t, err)
	assert.Equal(t, "zip-arguments-parameters", argumentUploadPipeline.Name)

	/* ---------- Import pipeline tarball by URL ---------- */
	time.Sleep(1 * time.Second)
	argumentUrlPipeline, err := s.pipelineClient.Create(&params.CreatePipelineParams{
		Body: &pipeline_model.APIPipeline{URL: &pipeline_model.APIURL{
			PipelineURL: "https://storage.googleapis.com/ml-pipeline-dataset/arguments.pipeline.zip"}}})
	assert.Nil(t, err)
	assert.Equal(t, "arguments.pipeline.zip", argumentUrlPipeline.Name)

	/* ---------- Verify list pipeline works ---------- */
	pipelines, totalSize, _, err := s.pipelineClient.List(params.NewListPipelinesParams())
	assert.Nil(t, err)
	assert.Equal(t, 4, len(pipelines))
	assert.Equal(t, 4, totalSize)
	for _, p := range pipelines {
		// Sampling one of the pipelines and verify the result is expected.
		if p.Name == "arguments-parameters.yaml" {
			verifyPipeline(t, p)
		}
	}

	/* ---------- Verify list pipeline sorted by names ---------- */
	listFirstPagePipelines, totalSize, nextPageToken, err := s.pipelineClient.List(
		&params.ListPipelinesParams{PageSize: util.Int32Pointer(2), SortBy: util.StringPointer("name")})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(listFirstPagePipelines))
	assert.Equal(t, 4, totalSize)
	assert.Equal(t, "arguments-parameters.yaml", listFirstPagePipelines[0].Name)
	assert.Equal(t, "arguments.pipeline.zip", listFirstPagePipelines[1].Name)
	assert.NotEmpty(t, nextPageToken)

	listSecondPagePipelines, totalSize, nextPageToken, err := s.pipelineClient.List(
		&params.ListPipelinesParams{PageToken: util.StringPointer(nextPageToken), PageSize: util.Int32Pointer(2), SortBy: util.StringPointer("name")})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(listSecondPagePipelines))
	assert.Equal(t, 4, totalSize)
	assert.Equal(t, "sequential", listSecondPagePipelines[0].Name)
	assert.Equal(t, "zip-arguments-parameters", listSecondPagePipelines[1].Name)
	assert.Empty(t, nextPageToken)

	/* ---------- Verify list pipeline sorted by creation time ---------- */
	listFirstPagePipelines, totalSize, nextPageToken, err = s.pipelineClient.List(
		&params.ListPipelinesParams{PageSize: util.Int32Pointer(2), SortBy: util.StringPointer("created_at")})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(listFirstPagePipelines))
	assert.Equal(t, 4, totalSize)
	assert.Equal(t, "arguments-parameters.yaml", listFirstPagePipelines[0].Name)
	assert.Equal(t, "sequential", listFirstPagePipelines[1].Name)
	assert.NotEmpty(t, nextPageToken)

	listSecondPagePipelines, totalSize, nextPageToken, err = s.pipelineClient.List(
		&params.ListPipelinesParams{PageToken: util.StringPointer(nextPageToken), PageSize: util.Int32Pointer(2), SortBy: util.StringPointer("created_at")})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(listSecondPagePipelines))
	assert.Equal(t, 4, totalSize)
	assert.Equal(t, "zip-arguments-parameters", listSecondPagePipelines[0].Name)
	assert.Equal(t, "arguments.pipeline.zip", listSecondPagePipelines[1].Name)
	assert.Empty(t, nextPageToken)

	/* ---------- List pipelines sort by unsupported description field. Should fail. ---------- */
	_, _, _, err = s.pipelineClient.List(&params.ListPipelinesParams{
		PageSize: util.Int32Pointer(2), SortBy: util.StringPointer("unknownfield")})
	assert.NotNil(t, err)

	/* ---------- List pipelines sorted by names descend order ---------- */
	listFirstPagePipelines, totalSize, nextPageToken, err = s.pipelineClient.List(
		&params.ListPipelinesParams{PageSize: util.Int32Pointer(2), SortBy: util.StringPointer("name desc")})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(listFirstPagePipelines))
	assert.Equal(t, 4, totalSize)
	assert.Equal(t, "zip-arguments-parameters", listFirstPagePipelines[0].Name)
	assert.Equal(t, "sequential", listFirstPagePipelines[1].Name)
	assert.NotEmpty(t, nextPageToken)

	listSecondPagePipelines, totalSize, nextPageToken, err = s.pipelineClient.List(&params.ListPipelinesParams{
		PageToken: util.StringPointer(nextPageToken), PageSize: util.Int32Pointer(2), SortBy: util.StringPointer("name desc")})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(listSecondPagePipelines))
	assert.Equal(t, 4, totalSize)
	assert.Equal(t, "arguments.pipeline.zip", listSecondPagePipelines[0].Name)
	assert.Equal(t, "arguments-parameters.yaml", listSecondPagePipelines[1].Name)
	assert.Empty(t, nextPageToken)

	/* ---------- Verify get pipeline works ---------- */
	pipeline, err := s.pipelineClient.Get(&params.GetPipelineParams{ID: argumentYAMLPipeline.ID})
	assert.Nil(t, err)
	verifyPipeline(t, pipeline)

	/* ---------- Verify get template works ---------- */
	template, err := s.pipelineClient.GetTemplate(&params.GetTemplateParams{ID: argumentYAMLPipeline.ID})
	assert.Nil(t, err)
	expected, err := ioutil.ReadFile("../resources/arguments-parameters.yaml")
	assert.Nil(t, err)
	var expectedWorkflow v1alpha1.Workflow
	err = yaml.Unmarshal(expected, &expectedWorkflow)
	assert.Equal(t, expectedWorkflow, *template)
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
