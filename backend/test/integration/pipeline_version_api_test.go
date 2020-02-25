package integration

import (
	"testing"
	"time"

	"github.com/kubeflow/pipelines/backend/test"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/api/go_http_client/pipeline_model"

	params "github.com/kubeflow/pipelines/backend/api/go_http_client/pipeline_client/pipeline_service"
	uploadParams "github.com/kubeflow/pipelines/backend/api/go_http_client/pipeline_upload_client/pipeline_upload_service"
	"github.com/kubeflow/pipelines/backend/src/common/client/api_server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// This test suit tests various methods to import pipeline to pipeline system, including
// - upload yaml file
// - upload tarball file
// - providing YAML file url
// - Providing tarball file url
type PipelineVersionApiTest struct {
	suite.Suite
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

func (s *PipelineVersionApiTest) TestPipelineVersionAPI() {
	t := s.T()

	test.DeleteAllPipelines(s.pipelineClient, t)

	/* ---------- Upload a pipeline YAML ---------- */
	pipelineParams := uploadParams.NewUploadPipelineParams()
	pipelineName := "test_pipeline"
	pipelineParams.SetName(&pipelineName)
	pipeline, err := s.pipelineUploadClient.UploadFile("../resources/arguments-parameters.yaml", pipelineParams)
	assert.Nil(t, err)
	assert.Equal(t, "test_pipeline", pipeline.Name)

	/* ---------- Get pipeline id ---------- */
	pipelines, totalSize, _, err := s.pipelineClient.List(params.NewListPipelinesParams())
	assert.Nil(t, err)
	assert.Equal(t, 1, len(pipelines))
	assert.Equal(t, 1, totalSize)
	pipelineId := pipelines[0].ID

	/* ---------- Upload a pipeline version YAML under test_pipeline ---------- */
	// pipelineVersionParams := uploadParams.NewUploadPipelineVersionParams()
	// pipelineVersionParams.SetPipelineid(&pipelineId)
	// argumentYAMLPipelineVersion, err := s.pipelineUploadClient.UploadPipelineVersion("../resources/arguments-parameters.yaml", pipelineVersionParams)
	// assert.Nil(t, err)
	// assert.Equal(t, "arguments-parameters.yaml", argumentYAMLPipelineVersion.Name)
	// fmt.Printf("JING version %+v\n", argumentYAMLPipelineVersion)

	// /* ---------- Upload the same pipeline again. Should fail due to name uniqueness ---------- */
	// _, err = s.pipelineUploadClient.UploadFile("../resources/arguments-parameters.yaml", uploadParams.NewUploadPipelineParams())
	// assert.NotNil(t, err)
	// assert.Contains(t, err.Error(), "Failed to upload pipeline.")

	/* ---------- Import pipeline YAML by URL ---------- */
	time.Sleep(1 * time.Second)
	sequentialPipelineVersion, err := s.pipelineClient.CreatePipelineVersion(&params.CreatePipelineVersionParams{
		Body: &pipeline_model.APIPipelineVersion{
			Name: "sequential",
			PackageURL: &pipeline_model.APIURL{
				PipelineURL: "https://storage.googleapis.com/ml-pipeline-dataset/sequential.yaml",
			},
			ResourceReferences: []*pipeline_model.APIResourceReference{
				{
					Key:          &pipeline_model.APIResourceKey{Type: pipeline_model.APIResourceTypePIPELINE, ID: pipelineId},
					Relationship: pipeline_model.APIRelationshipOWNER,
				},
			},
		}})
	assert.Nil(t, err)
	assert.Equal(t, "sequential", sequentialPipelineVersion.Name)

	// /* ---------- Upload pipelines zip ---------- */
	// time.Sleep(1 * time.Second)
	// argumentUploadPipeline, err := s.pipelineUploadClient.UploadFile(
	// 	"../resources/arguments.pipeline.zip", &uploadParams.UploadPipelineParams{Name: util.StringPointer("zip-arguments-parameters")})
	// assert.Nil(t, err)
	// assert.Equal(t, "zip-arguments-parameters", argumentUploadPipeline.Name)

	/* ---------- Import pipeline tarball by URL ---------- */
	time.Sleep(1 * time.Second)
	argumentUrlPipelineVersion, err := s.pipelineClient.CreatePipelineVersion(&params.CreatePipelineVersionParams{
		Body: &pipeline_model.APIPipelineVersion{
			PackageURL: &pipeline_model.APIURL{
				PipelineURL: "https://storage.googleapis.com/ml-pipeline-dataset/arguments.pipeline.zip",
			},
			ResourceReferences: []*pipeline_model.APIResourceReference{
				{
					Key:          &pipeline_model.APIResourceKey{Type: pipeline_model.APIResourceTypePIPELINE, ID: pipelineId},
					Relationship: pipeline_model.APIRelationshipOWNER,
				},
			},
		}})
	assert.Nil(t, err)
	assert.Equal(t, "arguments.pipeline.zip", argumentUrlPipelineVersion.Name)

	// /* ---------- Verify list pipeline works ---------- */
	// pipelines, totalSize, _, err := s.pipelineClient.List(params.NewListPipelinesParams())
	// assert.Nil(t, err)
	// assert.Equal(t, 4, len(pipelines))
	// assert.Equal(t, 4, totalSize)
	// for _, p := range pipelines {
	// 	// Sampling one of the pipelines and verify the result is expected.
	// 	if p.Name == "arguments-parameters.yaml" {
	// 		verifyPipeline(t, p)
	// 	}
	// }

	// /* ---------- Verify list pipeline sorted by names ---------- */
	// listFirstPagePipelines, totalSize, nextPageToken, err := s.pipelineClient.List(
	// 	&params.ListPipelinesParams{PageSize: util.Int32Pointer(2), SortBy: util.StringPointer("name")})
	// assert.Nil(t, err)
	// assert.Equal(t, 2, len(listFirstPagePipelines))
	// assert.Equal(t, 4, totalSize)
	// assert.Equal(t, "arguments-parameters.yaml", listFirstPagePipelines[0].Name)
	// assert.Equal(t, "arguments.pipeline.zip", listFirstPagePipelines[1].Name)
	// assert.NotEmpty(t, nextPageToken)

	// listSecondPagePipelines, totalSize, nextPageToken, err := s.pipelineClient.List(
	// 	&params.ListPipelinesParams{PageToken: util.StringPointer(nextPageToken), PageSize: util.Int32Pointer(2), SortBy: util.StringPointer("name")})
	// assert.Nil(t, err)
	// assert.Equal(t, 2, len(listSecondPagePipelines))
	// assert.Equal(t, 4, totalSize)
	// assert.Equal(t, "sequential", listSecondPagePipelines[0].Name)
	// assert.Equal(t, "zip-arguments-parameters", listSecondPagePipelines[1].Name)
	// assert.Empty(t, nextPageToken)

	// /* ---------- Verify list pipeline sorted by creation time ---------- */
	// listFirstPagePipelines, totalSize, nextPageToken, err = s.pipelineClient.List(
	// 	&params.ListPipelinesParams{PageSize: util.Int32Pointer(2), SortBy: util.StringPointer("created_at")})
	// assert.Nil(t, err)
	// assert.Equal(t, 2, len(listFirstPagePipelines))
	// assert.Equal(t, 4, totalSize)
	// assert.Equal(t, "arguments-parameters.yaml", listFirstPagePipelines[0].Name)
	// assert.Equal(t, "sequential", listFirstPagePipelines[1].Name)
	// assert.NotEmpty(t, nextPageToken)

	// listSecondPagePipelines, totalSize, nextPageToken, err = s.pipelineClient.List(
	// 	&params.ListPipelinesParams{PageToken: util.StringPointer(nextPageToken), PageSize: util.Int32Pointer(2), SortBy: util.StringPointer("created_at")})
	// assert.Nil(t, err)
	// assert.Equal(t, 2, len(listSecondPagePipelines))
	// assert.Equal(t, 4, totalSize)
	// assert.Equal(t, "zip-arguments-parameters", listSecondPagePipelines[0].Name)
	// assert.Equal(t, "arguments.pipeline.zip", listSecondPagePipelines[1].Name)
	// assert.Empty(t, nextPageToken)

	// /* ---------- List pipelines sort by unsupported description field. Should fail. ---------- */
	// _, _, _, err = s.pipelineClient.List(&params.ListPipelinesParams{
	// 	PageSize: util.Int32Pointer(2), SortBy: util.StringPointer("unknownfield")})
	// assert.NotNil(t, err)

	// /* ---------- List pipelines sorted by names descend order ---------- */
	// listFirstPagePipelines, totalSize, nextPageToken, err = s.pipelineClient.List(
	// 	&params.ListPipelinesParams{PageSize: util.Int32Pointer(2), SortBy: util.StringPointer("name desc")})
	// assert.Nil(t, err)
	// assert.Equal(t, 2, len(listFirstPagePipelines))
	// assert.Equal(t, 4, totalSize)
	// assert.Equal(t, "zip-arguments-parameters", listFirstPagePipelines[0].Name)
	// assert.Equal(t, "sequential", listFirstPagePipelines[1].Name)
	// assert.NotEmpty(t, nextPageToken)

	// listSecondPagePipelines, totalSize, nextPageToken, err = s.pipelineClient.List(&params.ListPipelinesParams{
	// 	PageToken: util.StringPointer(nextPageToken), PageSize: util.Int32Pointer(2), SortBy: util.StringPointer("name desc")})
	// assert.Nil(t, err)
	// assert.Equal(t, 2, len(listSecondPagePipelines))
	// assert.Equal(t, 4, totalSize)
	// assert.Equal(t, "arguments.pipeline.zip", listSecondPagePipelines[0].Name)
	// assert.Equal(t, "arguments-parameters.yaml", listSecondPagePipelines[1].Name)
	// assert.Empty(t, nextPageToken)

	// /* ---------- Verify get pipeline works ---------- */
	// pipeline, err := s.pipelineClient.Get(&params.GetPipelineParams{ID: argumentYAMLPipeline.ID})
	// assert.Nil(t, err)
	// verifyPipeline(t, pipeline)

	// /* ---------- Verify get template works ---------- */
	// template, err := s.pipelineClient.GetTemplate(&params.GetTemplateParams{ID: argumentYAMLPipeline.ID})
	// assert.Nil(t, err)
	// expected, err := ioutil.ReadFile("../resources/arguments-parameters.yaml")
	// assert.Nil(t, err)
	// var expectedWorkflow v1alpha1.Workflow
	// err = yaml.Unmarshal(expected, &expectedWorkflow)
	// assert.Equal(t, expectedWorkflow, *template)
}

func verifyPipelineVersion(t *testing.T, pipeline *pipeline_model.APIPipelineVersion) {
	assert.NotNil(t, *pipeline)
	assert.NotNil(t, pipeline.CreatedAt)
	expected := pipeline_model.APIPipeline{
		ID:        pipeline.ID,
		CreatedAt: pipeline.CreatedAt,
		Name:      "arguments-parameters.yaml",
		Parameters: []*pipeline_model.APIParameter{
			{Name: "param1", Value: "hello"}, // Default value in the pipeline template
			{Name: "param2"},                 // No default value in the pipeline
		},
		// TODO(jingzhang36): after version API launch, remove the following field.
		// This is because after the version API launch, we won't have defautl
		// version produced automatically when creating pipeline.
		DefaultVersion: &pipeline_model.APIPipelineVersion{
			CreatedAt: pipeline.CreatedAt,
			ID:        pipeline.ID,
			Name:      "arguments-parameters.yaml",
			Parameters: []*pipeline_model.APIParameter{
				{Name: "param1", Value: "hello"},
				{Name: "param2"}},
			ResourceReferences: []*pipeline_model.APIResourceReference{{
				Key:          &pipeline_model.APIResourceKey{ID: pipeline.ID, Type: pipeline_model.APIResourceTypePIPELINE},
				Relationship: pipeline_model.APIRelationshipOWNER}}},
	}
	assert.Equal(t, expected, *pipeline)
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
	test.DeleteAllPipelines(s.pipelineClient, s.T())
}
