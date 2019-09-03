package integration

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/golang/glog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/kubeflow/pipelines/backend/api/go_http_client/experiment_client/experiment_service"
	"github.com/kubeflow/pipelines/backend/api/go_http_client/experiment_model"
	"github.com/kubeflow/pipelines/backend/api/go_http_client/pipeline_client/pipeline_service"
	"github.com/kubeflow/pipelines/backend/api/go_http_client/pipeline_model"
	"github.com/kubeflow/pipelines/backend/api/go_http_client/pipeline_upload_client/pipeline_upload_service"
	"github.com/kubeflow/pipelines/backend/src/common/client/api_server"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/test"
)

// Methods are organized into two types: "prepare" and "verify".
// "prepare" tests setup resources before upgrade
// "verify" tests verifies resources are expected after upgrade
type UpgradeTests struct {
	suite.Suite
	namespace            string
	experimentClient     *api_server.ExperimentClient
	pipelineClient       *api_server.PipelineClient
	pipelineUploadClient *api_server.PipelineUploadClient
}

// Check the namespace have ML job installed and ready
func (s *UpgradeTests) SetupSuite() {
	// Integration tests also run these tests to first ensure they work, so that
	// when integration tests pass and upgrade tests fail, we know for sure
	// upgrade process went wrong somehow.
	if !(*runIntegrationTests || *runUpgradeTests) {
		s.T().SkipNow()
		return
	}

	err := test.WaitForReady(*namespace, *initializeTimeout)
	if err != nil {
		glog.Exitf("Failed to initialize test. Error: %v", err)
	}
	s.namespace = *namespace
	clientConfig := test.GetClientConfig(*namespace)
	s.experimentClient, err = api_server.NewExperimentClient(clientConfig, false)
	if err != nil {
		glog.Exitf("Failed to get experiment client. Error: %v", err)
	}
	s.pipelineUploadClient, err = api_server.NewPipelineUploadClient(clientConfig, false)
	if err != nil {
		glog.Exitf("Failed to get pipeline upload client. Error: %s", err.Error())
	}
	s.pipelineClient, err = api_server.NewPipelineClient(clientConfig, false)
	if err != nil {
		glog.Exitf("Failed to get pipeline client. Error: %s", err.Error())
	}
}

func (s *UpgradeTests) TearDownSuite() {
	if *runIntegrationTests {
		t := s.T()

		// Clean up after the suite to unblock other tests. (Not needed for upgrade
		// tests because it needs changes in prepare tests to persist and verified
		// later.)
		test.DeleteAllExperiments(s.experimentClient, t)
		test.DeleteAllPipelines(s.pipelineClient, t)
	}
}

func (s *UpgradeTests) TestPrepareExperiments() {
	t := s.T()

	/* ---------- Clean up ---------- */
	test.DeleteAllExperiments(s.experimentClient, t)

	/* ---------- Create a new experiment ---------- */
	experiment := &experiment_model.APIExperiment{Name: "training", Description: "my first experiment"}
	_, err := s.experimentClient.Create(&experiment_service.CreateExperimentParams{
		Body: experiment,
	})
	assert.Nil(t, err)

	/* ---------- Create a few more new experiment ---------- */
	// This ensures they can be sorted by create time in expected order.
	time.Sleep(1 * time.Second)
	experiment = &experiment_model.APIExperiment{Name: "prediction", Description: "my second experiment"}
	_, err = s.experimentClient.Create(&experiment_service.CreateExperimentParams{
		Body: experiment,
	})
	assert.Nil(t, err)

	time.Sleep(1 * time.Second)
	experiment = &experiment_model.APIExperiment{Name: "moonshot", Description: "my third experiment"}
	_, err = s.experimentClient.Create(&experiment_service.CreateExperimentParams{
		Body: experiment,
	})
	assert.Nil(t, err)
}

func (s *UpgradeTests) TestPreparePipelines() {
	t := s.T()

	test.DeleteAllPipelines(s.pipelineClient, t)

	/* ---------- Upload pipelines YAML ---------- */
	argumentYAMLPipeline, err := s.pipelineUploadClient.UploadFile("../resources/arguments-parameters.yaml", pipeline_upload_service.NewUploadPipelineParams())
	assert.Nil(t, err)
	assert.Equal(t, "arguments-parameters.yaml", argumentYAMLPipeline.Name)

	/* ---------- Import pipeline YAML by URL ---------- */
	time.Sleep(1 * time.Second)
	sequentialPipeline, err := s.pipelineClient.Create(&pipeline_service.CreatePipelineParams{
		Body: &pipeline_model.APIPipeline{Name: "sequential", URL: &pipeline_model.APIURL{
			PipelineURL: "https://storage.googleapis.com/ml-pipeline-dataset/sequential.yaml"}}})
	assert.Nil(t, err)
	assert.Equal(t, "sequential", sequentialPipeline.Name)

	/* ---------- Upload pipelines zip ---------- */
	time.Sleep(1 * time.Second)
	argumentUploadPipeline, err := s.pipelineUploadClient.UploadFile(
		"../resources/arguments.pipeline.zip", &pipeline_upload_service.UploadPipelineParams{Name: util.StringPointer("zip-arguments-parameters")})
	assert.Nil(t, err)
	assert.Equal(t, "zip-arguments-parameters", argumentUploadPipeline.Name)

	/* ---------- Import pipeline tarball by URL ---------- */
	time.Sleep(1 * time.Second)
	argumentUrlPipeline, err := s.pipelineClient.Create(&pipeline_service.CreatePipelineParams{
		Body: &pipeline_model.APIPipeline{URL: &pipeline_model.APIURL{
			PipelineURL: "https://storage.googleapis.com/ml-pipeline-dataset/arguments.pipeline.zip"}}})
	assert.Nil(t, err)
	assert.Equal(t, "arguments.pipeline.zip", argumentUrlPipeline.Name)
}

func (s *UpgradeTests) TestVerifyPipelines() {
	t := s.T()

	/* ---------- Verify list pipeline sorted by creation time ---------- */
	pipelines, _, _, err := s.pipelineClient.List(
		&pipeline_service.ListPipelinesParams{SortBy: util.StringPointer("created_at")})
	assert.Nil(t, err)
	// During upgrade, default pipelines may be installed, so we only verify the
	// 4 oldest pipelines here.
	assert.True(t, len(pipelines) >= 4)
	assert.Equal(t, "arguments-parameters.yaml", pipelines[0].Name)
	assert.Equal(t, "sequential", pipelines[1].Name)
	assert.Equal(t, "zip-arguments-parameters", pipelines[2].Name)
	assert.Equal(t, "arguments.pipeline.zip", pipelines[3].Name)

	verifyPipeline(t, pipelines[0])

	/* ---------- Verify get template works ---------- */
	template, err := s.pipelineClient.GetTemplate(&pipeline_service.GetTemplateParams{ID: pipelines[0].ID})
	assert.Nil(t, err)
	expected, err := ioutil.ReadFile("../resources/arguments-parameters.yaml")
	assert.Nil(t, err)
	var expectedWorkflow v1alpha1.Workflow
	err = yaml.Unmarshal(expected, &expectedWorkflow)
	assert.Equal(t, expectedWorkflow, *template)
}

func (s *UpgradeTests) TestVerifyExperiments() {
	t := s.T()

	/* ---------- Verify list experiments sorted by creation time ---------- */
	experiments, _, _, err := s.experimentClient.List(
		&experiment_service.ListExperimentParams{SortBy: util.StringPointer("created_at")})
	assert.Nil(t, err)
	// after upgrade, default experiment may be inserted, but the oldest 3
	// experiments should be the ones created in this test
	assert.True(t, len(experiments) >= 3)

	assert.Equal(t, "training", experiments[0].Name)
	assert.Equal(t, "my first experiment", experiments[0].Description)
	assert.NotEmpty(t, experiments[0].ID)
	assert.NotEmpty(t, experiments[0].CreatedAt)

	assert.Equal(t, "prediction", experiments[1].Name)
	assert.Equal(t, "my second experiment", experiments[1].Description)
	assert.NotEmpty(t, experiments[1].ID)
	assert.NotEmpty(t, experiments[1].CreatedAt)

	assert.Equal(t, "moonshot", experiments[2].Name)
	assert.Equal(t, "my third experiment", experiments[2].Description)
	assert.NotEmpty(t, experiments[2].ID)
	assert.NotEmpty(t, experiments[2].CreatedAt)
}

func TestUpgrade(t *testing.T) {
	suite.Run(t, new(UpgradeTests))
}
