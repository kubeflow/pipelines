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

	experimentParams "github.com/kubeflow/pipelines/backend/api/go_http_client/experiment_client/experiment_service"
	"github.com/kubeflow/pipelines/backend/api/go_http_client/experiment_model"
	pipelineParams "github.com/kubeflow/pipelines/backend/api/go_http_client/pipeline_client/pipeline_service"
	"github.com/kubeflow/pipelines/backend/api/go_http_client/pipeline_model"
	uploadParams "github.com/kubeflow/pipelines/backend/api/go_http_client/pipeline_upload_client/pipeline_upload_service"
	runParams "github.com/kubeflow/pipelines/backend/api/go_http_client/run_client/run_service"
	"github.com/kubeflow/pipelines/backend/api/go_http_client/run_model"
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
	runClient            *api_server.RunClient
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
	s.runClient, err = api_server.NewRunClient(clientConfig, false)
	if err != nil {
		glog.Exitf("Failed to get run client. Error: %s", err.Error())
	}

	/* ---------- Clean up ---------- */
	t := s.T()
	test.DeleteAllExperiments(s.experimentClient, t)
	test.DeleteAllPipelines(s.pipelineClient, t)
	test.DeleteAllRuns(s.runClient, t)
}

func (s *UpgradeTests) TearDownSuite() {
	if *runIntegrationTests {
		t := s.T()

		// Clean up after the suite to unblock other tests. (Not needed for upgrade
		// tests because it needs changes in prepare tests to persist and verified
		// later.)
		test.DeleteAllExperiments(s.experimentClient, t)
		test.DeleteAllPipelines(s.pipelineClient, t)
		test.DeleteAllRuns(s.runClient, t)
	}
}

func (s *UpgradeTests) TestPrepareExperiments() {
	t := s.T()

	/* ---------- Create a new experiment ---------- */
	experiment := &experiment_model.APIExperiment{Name: "training", Description: "my first experiment"}
	_, err := s.experimentClient.Create(&experimentParams.CreateExperimentParams{
		Body: experiment,
	})
	assert.Nil(t, err)

	/* ---------- Create a few more new experiment ---------- */
	// This ensures they can be sorted by create time in expected order.
	time.Sleep(1 * time.Second)
	experiment = &experiment_model.APIExperiment{Name: "prediction", Description: "my second experiment"}
	_, err = s.experimentClient.Create(&experimentParams.CreateExperimentParams{
		Body: experiment,
	})
	assert.Nil(t, err)

	time.Sleep(1 * time.Second)
	experiment = &experiment_model.APIExperiment{Name: "moonshot", Description: "my third experiment"}
	_, err = s.experimentClient.Create(&experimentParams.CreateExperimentParams{
		Body: experiment,
	})
	assert.Nil(t, err)
}

func (s *UpgradeTests) TestPreparePipelines() {
	t := s.T()

	test.DeleteAllPipelines(s.pipelineClient, t)

	/* ---------- Upload pipelines YAML ---------- */
	argumentYAMLPipeline, err := s.pipelineUploadClient.UploadFile("../resources/arguments-parameters.yaml", uploadParams.NewUploadPipelineParams())
	assert.Nil(t, err)
	assert.Equal(t, "arguments-parameters.yaml", argumentYAMLPipeline.Name)

	/* ---------- Import pipeline YAML by URL ---------- */
	time.Sleep(1 * time.Second)
	sequentialPipeline, err := s.pipelineClient.Create(&pipelineParams.CreatePipelineParams{
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
	argumentUrlPipeline, err := s.pipelineClient.Create(&pipelineParams.CreatePipelineParams{
		Body: &pipeline_model.APIPipeline{URL: &pipeline_model.APIURL{
			PipelineURL: "https://storage.googleapis.com/ml-pipeline-dataset/arguments.pipeline.zip"}}})
	assert.Nil(t, err)
	assert.Equal(t, "arguments.pipeline.zip", argumentUrlPipeline.Name)
}

func (s *UpgradeTests) TestPrepareRuns() {
	t := s.T()

	/* ---------- Upload pipelines YAML ---------- */
	helloWorldPipeline, err := s.pipelineUploadClient.UploadFile("../resources/hello-world.yaml", uploadParams.NewUploadPipelineParams())
	assert.Nil(t, err)

	/* ---------- Create a new hello world experiment ---------- */
	experiment := &experiment_model.APIExperiment{Name: "hello world experiment"}
	helloWorldExperiment, err := s.experimentClient.Create(&experimentParams.CreateExperimentParams{Body: experiment})
	assert.Nil(t, err)

	/* ---------- Create a new hello world run by specifying pipeline ID ---------- */
	createRunRequest := &runParams.CreateRunParams{Body: &run_model.APIRun{
		Name:        "hello world",
		Description: "this is hello world",
		PipelineSpec: &run_model.APIPipelineSpec{
			PipelineID: helloWorldPipeline.ID,
		},
		ResourceReferences: []*run_model.APIResourceReference{
			{Key: &run_model.APIResourceKey{Type: run_model.APIResourceTypeEXPERIMENT, ID: helloWorldExperiment.ID},
				Name: helloWorldExperiment.Name, Relationship: run_model.APIRelationshipOWNER},
		},
	}}
	_, _, err = s.runClient.Create(createRunRequest)
	assert.Nil(t, err)
}

func (s *UpgradeTests) TestVerifyExperiments() {
	t := s.T()

	/* ---------- Verify list experiments sorted by creation time ---------- */
	experiments, _, _, err := s.experimentClient.List(
		&experimentParams.ListExperimentParams{SortBy: util.StringPointer("created_at")})
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

func (s *UpgradeTests) TestVerifyPipelines() {
	t := s.T()

	/* ---------- Verify list pipeline sorted by creation time ---------- */
	pipelines, _, _, err := s.pipelineClient.List(
		&pipelineParams.ListPipelinesParams{SortBy: util.StringPointer("created_at")})
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
	template, err := s.pipelineClient.GetTemplate(&pipelineParams.GetTemplateParams{ID: pipelines[0].ID})
	assert.Nil(t, err)
	expected, err := ioutil.ReadFile("../resources/arguments-parameters.yaml")
	assert.Nil(t, err)
	var expectedWorkflow v1alpha1.Workflow
	err = yaml.Unmarshal(expected, &expectedWorkflow)
	assert.Equal(t, expectedWorkflow, *template)
}

func (s *UpgradeTests) TestVerifyRuns() {
	t := s.T()

	/* ---------- List the runs, sorted by creation time ---------- */
	runs, _, _, err := s.runClient.List(
		&runParams.ListRunsParams{SortBy: util.StringPointer("created_at")})
	assert.Nil(t, err)
	assert.Equal(t, "hello world", runs[0].Name)

	/* ---------- Get hello world run ---------- */
	helloWorldRunDetail, _, err := s.runClient.Get(&runParams.GetRunParams{RunID: runs[0].ID})
	assert.Nil(t, err)
	checkHelloWorldRunDetail(t, helloWorldRunDetail)
}

func TestUpgrade(t *testing.T) {
	suite.Run(t, new(UpgradeTests))
}

func checkHelloWorldRunDetail(t *testing.T, runDetail *run_model.APIRunDetail) {
	// Check workflow manifest is not empty
	assert.Contains(t, runDetail.Run.PipelineSpec.WorkflowManifest, "whalesay")
	// Check runtime workflow manifest is not empty
	assert.Contains(t, runDetail.PipelineRuntime.WorkflowManifest, "whalesay")

	expectedRun := &run_model.APIRun{
		ID:          runDetail.Run.ID,
		Name:        "hello world",
		Description: "this is hello world",
		Status:      runDetail.Run.Status,
		PipelineSpec: &run_model.APIPipelineSpec{
			PipelineID:       runDetail.Run.PipelineSpec.PipelineID,
			PipelineName:     "hello-world.yaml",
			WorkflowManifest: runDetail.Run.PipelineSpec.WorkflowManifest,
		},
		ResourceReferences: []*run_model.APIResourceReference{
			{Key: &run_model.APIResourceKey{Type: run_model.APIResourceTypeEXPERIMENT, ID: runDetail.Run.ResourceReferences[0].Key.ID},
				Name: "hello world experiment", Relationship: run_model.APIRelationshipOWNER,
			},
		},
		CreatedAt:   runDetail.Run.CreatedAt,
		ScheduledAt: runDetail.Run.ScheduledAt,
		FinishedAt:  runDetail.Run.FinishedAt,
	}
	assert.Equal(t, expectedRun, runDetail.Run)
}
