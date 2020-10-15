package integration

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/golang/glog"
	params "github.com/kubeflow/pipelines/backend/api/go_http_client/pipeline_client/pipeline_service"
	"github.com/kubeflow/pipelines/backend/api/go_http_client/pipeline_model"
	uploadParams "github.com/kubeflow/pipelines/backend/api/go_http_client/pipeline_upload_client/pipeline_upload_service"
	"github.com/kubeflow/pipelines/backend/src/common/client/api_server"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/test"
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
	time.Sleep(1 * time.Second)
	pipelineVersionParams := uploadParams.NewUploadPipelineVersionParams()
	pipelineVersionParams.SetPipelineid(&pipelineId)
	argumentYAMLPipelineVersion, err := s.pipelineUploadClient.UploadPipelineVersion("../resources/arguments-parameters.yaml", pipelineVersionParams)
	assert.Nil(t, err)
	assert.Equal(t, "arguments-parameters.yaml", argumentYAMLPipelineVersion.Name)

	/* ---------- Update pipeline default version ---------- */
	time.Sleep(1 * time.Second)
	sortBy := "created_at"
	versions, _, _, err := s.pipelineClient.ListPipelineVersions(&params.ListPipelineVersionsParams{ResourceKeyID: &pipelineId, SortBy: &sortBy})
	assert.Nil(t, err)

	err = s.pipelineClient.UpdateDefaultVersion(&params.UpdatePipelineDefaultVersionParams{PipelineID: pipelineId,
		VersionID: versions[0].ID})
	assert.Nil(t, err)

	time.Sleep(1 * time.Second)
	pipelineSelected, err := s.pipelineClient.Get(&params.GetPipelineParams{ID: pipelineId})
	assert.Nil(t, err)
	assert.Equal(t, pipelineSelected.DefaultVersion.ID, versions[0].ID)

	/* ---------- Upload the same pipeline version again. Should fail due to name uniqueness ---------- */
	time.Sleep(1 * time.Second)
	_, err = s.pipelineUploadClient.UploadPipelineVersion("../resources/arguments-parameters.yaml", uploadParams.NewUploadPipelineVersionParams())
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed to upload pipeline version.")

	/* ---------- Import pipeline version YAML by URL ---------- */
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

	/* ---------- Upload pipeline version zip ---------- */
	time.Sleep(1 * time.Second)
	argumentUploadPipelineVersion, err := s.pipelineUploadClient.UploadPipelineVersion(
		"../resources/arguments.pipeline.zip", &uploadParams.UploadPipelineVersionParams{
			Name:       util.StringPointer("zip-arguments-parameters"),
			Pipelineid: util.StringPointer(pipelineId),
		})
	assert.Nil(t, err)
	assert.Equal(t, "zip-arguments-parameters", argumentUploadPipelineVersion.Name)

	/* ---------- Import pipeline tarball by URL ---------- */
	time.Sleep(1 * time.Second)
	argumentUrlPipelineVersion, err := s.pipelineClient.CreatePipelineVersion(&params.CreatePipelineVersionParams{
		Body: &pipeline_model.APIPipelineVersion{
			Name: "arguments",
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
	assert.Equal(t, "arguments", argumentUrlPipelineVersion.Name)

	/* ---------- Verify list pipeline version works ---------- */
	pipelineVersions, totalSize, _, err := s.pipelineClient.ListPipelineVersions(&params.ListPipelineVersionsParams{
		ResourceKeyID:   util.StringPointer(pipelineId),
		ResourceKeyType: util.StringPointer("PIPELINE"),
	})
	assert.Nil(t, err)
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
				})
		}
	}

	/* ---------- Verify list pipeline sorted by names ---------- */
	listFirstPagePipelineVersions, totalSize, nextPageToken, err := s.pipelineClient.ListPipelineVersions(
		&params.ListPipelineVersionsParams{
			PageSize:        util.Int32Pointer(3),
			SortBy:          util.StringPointer("name"),
			ResourceKeyID:   util.StringPointer(pipelineId),
			ResourceKeyType: util.StringPointer("PIPELINE"),
		})
	assert.Nil(t, err)
	assert.Equal(t, 3, len(listFirstPagePipelineVersions))
	assert.Equal(t, 5, totalSize)
	assert.Equal(t, "arguments", listFirstPagePipelineVersions[0].Name)
	assert.Equal(t, "arguments-parameters.yaml", listFirstPagePipelineVersions[1].Name)
	assert.Equal(t, "sequential", listFirstPagePipelineVersions[2].Name)
	assert.NotEmpty(t, nextPageToken)

	listSecondPagePipelineVersions, totalSize, nextPageToken, err := s.pipelineClient.ListPipelineVersions(
		&params.ListPipelineVersionsParams{
			PageToken:       util.StringPointer(nextPageToken),
			PageSize:        util.Int32Pointer(3),
			SortBy:          util.StringPointer("name"),
			ResourceKeyID:   util.StringPointer(pipelineId),
			ResourceKeyType: util.StringPointer("PIPELINE"),
		})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(listSecondPagePipelineVersions))
	assert.Equal(t, 5, totalSize)
	assert.Equal(t, "test_pipeline", listSecondPagePipelineVersions[0].Name)
	assert.Equal(t, "zip-arguments-parameters", listSecondPagePipelineVersions[1].Name)
	assert.Empty(t, nextPageToken)

	/* ---------- Verify list pipeline version sorted by creation time ---------- */
	listFirstPagePipelineVersions, totalSize, nextPageToken, err = s.pipelineClient.ListPipelineVersions(
		&params.ListPipelineVersionsParams{
			PageSize:        util.Int32Pointer(3),
			SortBy:          util.StringPointer("created_at"),
			ResourceKeyID:   util.StringPointer(pipelineId),
			ResourceKeyType: util.StringPointer("PIPELINE"),
		})
	assert.Nil(t, err)
	assert.Equal(t, 3, len(listFirstPagePipelineVersions))
	assert.Equal(t, 5, totalSize)
	assert.Equal(t, "test_pipeline", listFirstPagePipelineVersions[0].Name)
	assert.Equal(t, "arguments-parameters.yaml", listFirstPagePipelineVersions[1].Name)
	assert.Equal(t, "sequential", listFirstPagePipelineVersions[2].Name)
	assert.NotEmpty(t, nextPageToken)

	listSecondPagePipelineVersions, totalSize, nextPageToken, err = s.pipelineClient.ListPipelineVersions(
		&params.ListPipelineVersionsParams{
			PageToken:       util.StringPointer(nextPageToken),
			PageSize:        util.Int32Pointer(3),
			SortBy:          util.StringPointer("created_at"),
			ResourceKeyID:   util.StringPointer(pipelineId),
			ResourceKeyType: util.StringPointer("PIPELINE"),
		})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(listSecondPagePipelineVersions))
	assert.Equal(t, 5, totalSize)
	assert.Equal(t, "zip-arguments-parameters", listSecondPagePipelineVersions[0].Name)
	assert.Equal(t, "arguments", listSecondPagePipelineVersions[1].Name)
	assert.Empty(t, nextPageToken)

	/* ---------- List pipeline versions sort by unsupported description field. Should fail. ---------- */
	_, _, _, err = s.pipelineClient.ListPipelineVersions(&params.ListPipelineVersionsParams{
		PageSize:        util.Int32Pointer(2),
		SortBy:          util.StringPointer("unknownfield"),
		ResourceKeyID:   util.StringPointer(pipelineId),
		ResourceKeyType: util.StringPointer("PIPELINE"),
	})
	assert.NotNil(t, err)

	/* ---------- List pipeline versions sorted by names descend order ---------- */
	listFirstPagePipelineVersions, totalSize, nextPageToken, err = s.pipelineClient.ListPipelineVersions(
		&params.ListPipelineVersionsParams{
			PageSize:        util.Int32Pointer(3),
			SortBy:          util.StringPointer("name desc"),
			ResourceKeyID:   util.StringPointer(pipelineId),
			ResourceKeyType: util.StringPointer("PIPELINE"),
		})
	assert.Nil(t, err)
	assert.Equal(t, 3, len(listFirstPagePipelineVersions))
	assert.Equal(t, 5, totalSize)
	assert.Equal(t, "zip-arguments-parameters", listFirstPagePipelineVersions[0].Name)
	assert.Equal(t, "test_pipeline", listFirstPagePipelineVersions[1].Name)
	assert.Equal(t, "sequential", listFirstPagePipelineVersions[2].Name)
	assert.NotEmpty(t, nextPageToken)

	listSecondPagePipelineVersions, totalSize, nextPageToken, err = s.pipelineClient.ListPipelineVersions(
		&params.ListPipelineVersionsParams{
			PageToken:       util.StringPointer(nextPageToken),
			PageSize:        util.Int32Pointer(3),
			SortBy:          util.StringPointer("name desc"),
			ResourceKeyID:   util.StringPointer(pipelineId),
			ResourceKeyType: util.StringPointer("PIPELINE"),
		})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(listSecondPagePipelineVersions))
	assert.Equal(t, 5, totalSize)
	assert.Equal(t, "arguments-parameters.yaml", listSecondPagePipelineVersions[0].Name)
	assert.Equal(t, "arguments", listSecondPagePipelineVersions[1].Name)
	assert.Empty(t, nextPageToken)

	/* ---------- Verify get pipeline version works ---------- */
	pipelineVersion, err := s.pipelineClient.GetPipelineVersion(&params.GetPipelineVersionParams{VersionID: argumentUrlPipelineVersion.ID})
	assert.Nil(t, err)
	assert.Equal(t, pipelineVersion.Name, "arguments")
	assert.NotNil(t, pipelineVersion.CreatedAt)
	assert.Equal(t, pipelineVersion.Parameters,
		[]*pipeline_model.APIParameter{
			{Name: "param1", Value: "hello"},
			{Name: "param2"},
		})

	/* ---------- Verify get template works ---------- */
	template, err := s.pipelineClient.GetPipelineVersionTemplate(&params.GetPipelineVersionTemplateParams{VersionID: argumentYAMLPipelineVersion.ID})
	assert.Nil(t, err)
	expected, err := ioutil.ReadFile("../resources/arguments-parameters.yaml")
	assert.Nil(t, err)
	var expectedWorkflow v1alpha1.Workflow
	err = yaml.Unmarshal(expected, &expectedWorkflow)
	assert.Equal(t, expectedWorkflow, *template)
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
	// Delete pipelines will delete pipelines and their versions.
	test.DeleteAllPipelines(s.pipelineClient, s.T())
}
