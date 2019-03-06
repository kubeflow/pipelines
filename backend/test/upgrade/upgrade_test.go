package upgrade

import (
	"testing"

	"github.com/golang/glog"
	experimentParams "github.com/kubeflow/pipelines/backend/api/go_http_client/experiment_client/experiment_service"
	"github.com/kubeflow/pipelines/backend/api/go_http_client/experiment_model"
	jobParams "github.com/kubeflow/pipelines/backend/api/go_http_client/job_client/job_service"
	"github.com/kubeflow/pipelines/backend/api/go_http_client/job_model"
	pipelineParams "github.com/kubeflow/pipelines/backend/api/go_http_client/pipeline_client/pipeline_service"
	"github.com/kubeflow/pipelines/backend/api/go_http_client/pipeline_model"
	uploadParams "github.com/kubeflow/pipelines/backend/api/go_http_client/pipeline_upload_client/pipeline_upload_service"
	runParams "github.com/kubeflow/pipelines/backend/api/go_http_client/run_client/run_service"
	"github.com/kubeflow/pipelines/backend/api/go_http_client/run_model"
	"github.com/kubeflow/pipelines/backend/src/common/client/api_server"
	"github.com/kubeflow/pipelines/backend/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type UpgradeTest struct {
	suite.Suite
	namespace            string
	pipelineClient       *api_server.PipelineClient
	pipelineUploadClient *api_server.PipelineUploadClient
	experimentClient     *api_server.ExperimentClient
	jobClient            *api_server.JobClient
	runClient            *api_server.RunClient
}

// Check the namespace have ML pipeline installed and ready
func (s *UpgradeTest) SetupTest() {
	if !*runIntegrationTests {
		s.T().SkipNow()
		return
	}

	err := test.WaitForReady(*namespace, *initializeTimeout)
	if err != nil {
		glog.Exitf("Failed to initialize test. Error: %s", err.Error())
	}
	s.namespace = *namespace
	clientConfig := test.GetClientConfig(*namespace)
	s.experimentClient, err = api_server.NewExperimentClient(clientConfig, false)
	if err != nil {
		glog.Exitf("Failed to get pipeline upload client. Error: %s", err.Error())
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
	s.jobClient, err = api_server.NewJobClient(clientConfig, false)
	if err != nil {
		glog.Exitf("Failed to get job client. Error: %s", err.Error())
	}
	if !*cleanup {
		// The first time to run the upgrade test, before cluster is upgraded.
		// Initialize the cluster with some user data.
		s.hydrateDatabase()
	}
}

func (s *UpgradeTest) TearDownTest() {
	if *cleanup {
		s.CleanupCluster()
	}
}

func (s *UpgradeTest) CleanupCluster() {
	t := s.T()
	test.DeleteAllExperiments(s.experimentClient, t)
	test.DeleteAllPipelines(s.pipelineClient, t)
	test.DeleteAllRuns(s.runClient, t)
	test.DeleteAllJobs(s.jobClient, t)
}

// hydrateDatabase Function to initialize the database with some user data.
func (s *UpgradeTest) hydrateDatabase() {
	t := s.T()
	s.CleanupCluster()

	/* ---------- Create a pipeline ---------- */
	helloWorldPipeline, err := s.pipelineUploadClient.UploadFile("../resources/arguments-parameters.yaml", uploadParams.NewUploadPipelineParams())
	assert.Nil(t, err)

	/* ---------- Create an experiment ---------- */
	experiment := &experiment_model.APIExperiment{Name: "hello-world-experiment", Description: "my first experiment"}
	helloWorldExperiment, err := s.experimentClient.Create(&experimentParams.CreateExperimentParams{
		Body: experiment,
	})
	assert.Nil(t, err)

	/* ---------- Create a job ---------- */
	createJobRequest := &jobParams.CreateJobParams{Body: &job_model.APIJob{
		Name:        "hello world",
		Description: "this is hello world",
		PipelineSpec: &job_model.APIPipelineSpec{
			PipelineID: helloWorldPipeline.ID,
			Parameters: []*job_model.APIParameter{
				{Name: "param1", Value: "goodbye"},
				{Name: "param2", Value: "world"},
			},
		},
		ResourceReferences: []*job_model.APIResourceReference{
			{Key: &job_model.APIResourceKey{Type: job_model.APIResourceTypeEXPERIMENT, ID: helloWorldExperiment.ID},
				Relationship: job_model.APIRelationshipOWNER},
		},
		MaxConcurrency: 10,
		// Not enable it so it doesn't trigger any run.
		Enabled: false,
	}}
	_, err = s.jobClient.Create(createJobRequest)
	assert.Nil(t, err)

	/* ---------- Create a run ---------- */
	createRunRequest := &runParams.CreateRunParams{Body: &run_model.APIRun{
		Name:        "hello world",
		Description: "this is hello world",
		PipelineSpec: &run_model.APIPipelineSpec{
			PipelineID: helloWorldPipeline.ID,
			Parameters: []*run_model.APIParameter{
				{Name: "param1", Value: "hola"},
				{Name: "param2", Value: "world"},
			},
		},
		ResourceReferences: []*run_model.APIResourceReference{
			{Key: &run_model.APIResourceKey{Type: run_model.APIResourceTypeEXPERIMENT, ID: helloWorldExperiment.ID},
				Relationship: run_model.APIRelationshipOWNER},
		},
	}}
	_, _, err = s.runClient.Create(createRunRequest)
	assert.Nil(t, err)
}

func (s *UpgradeTest) TestUpgrade() {
	t := s.T()

	/* ---------- Verify pipelines ---------- */
	pipelines, totalSize, _, err := s.pipelineClient.List(pipelineParams.NewListPipelinesParams())
	assert.Nil(t, err)
	assert.Equal(t, 1, totalSize)
	assert.NotEmpty(t, pipelines[0].ID)
	assert.NotEmpty(t, pipelines[0].CreatedAt)
	expectedPipeline := &pipeline_model.APIPipeline{
		ID:        pipelines[0].ID,
		CreatedAt: pipelines[0].CreatedAt,
		Name:      "arguments-parameters.yaml",
		Parameters: []*pipeline_model.APIParameter{
			{Name: "param1", Value: "hello"},
			{Name: "param2"},
		},
	}
	assert.Equal(t, expectedPipeline, pipelines[0])

	/* ---------- Verify experiments ---------- */
	experiments, totalSize, _, err := s.experimentClient.List(&experimentParams.ListExperimentParams{})
	assert.Nil(t, err)
	assert.Equal(t, 1, totalSize)
	assert.NotEmpty(t, experiments[0].ID)
	assert.NotEmpty(t, experiments[0].CreatedAt)
	expectedExperiment := &experiment_model.APIExperiment{
		ID: experiments[0].ID, Name: "hello-world-experiment",
		Description: "my first experiment", CreatedAt: experiments[0].CreatedAt}
	assert.Equal(t, expectedExperiment, experiments[0])

	/* ---------- Verify jobs ---------- */
	jobs, totalSize, _, err := s.jobClient.List(&jobParams.ListJobsParams{})
	assert.Nil(t, err)
	assert.Equal(t, 1, totalSize)
	assert.NotEmpty(t, jobs[0].ID)
	assert.NotEmpty(t, jobs[0].PipelineSpec.PipelineID)
	assert.NotEmpty(t, jobs[0].PipelineSpec.WorkflowManifest)
	assert.NotEmpty(t, jobs[0].CreatedAt)
	assert.NotEmpty(t, jobs[0].UpdatedAt)
	assert.NotEmpty(t, jobs[0].Status)
	assert.NotNil(t, jobs[0].Trigger)

	expectedJob := &job_model.APIJob{
		ID:          jobs[0].ID,
		Name:        "hello world",
		Description: "this is hello world",
		PipelineSpec: &job_model.APIPipelineSpec{
			PipelineID:       jobs[0].PipelineSpec.PipelineID,
			WorkflowManifest: jobs[0].PipelineSpec.WorkflowManifest,
			Parameters: []*job_model.APIParameter{
				{Name: "param1", Value: "goodbye"},
				{Name: "param2", Value: "world"},
			},
		},
		ResourceReferences: []*job_model.APIResourceReference{
			{Key: &job_model.APIResourceKey{Type: job_model.APIResourceTypeEXPERIMENT, ID: experiments[0].ID},
				Relationship: job_model.APIRelationshipOWNER,
			},
		},
		MaxConcurrency: 10,
		Enabled:        false,
		CreatedAt:      jobs[0].CreatedAt,
		UpdatedAt:      jobs[0].UpdatedAt,
		Status:         jobs[0].Status,
		Trigger:        jobs[0].Trigger,
	}
	assert.Equal(t, expectedJob, jobs[0])

	/* ---------- Verify runs ---------- */
	runs, totalSize, _, err := s.runClient.List(&runParams.ListRunsParams{})
	assert.Nil(t, err)
	assert.Equal(t, 1, totalSize)
	assert.NotEmpty(t, runs[0].ID)
	assert.NotEmpty(t, runs[0].PipelineSpec.WorkflowManifest)
	assert.NotEmpty(t, runs[0].CreatedAt)
	assert.NotEmpty(t, runs[0].ScheduledAt)

	// Check runtime workflow manifest is not empty
	expectedRun := &run_model.APIRun{
		ID:          runs[0].ID,
		Name:        "hello world",
		Description: "this is hello world",
		Status:      runs[0].Status,
		PipelineSpec: &run_model.APIPipelineSpec{
			WorkflowManifest: runs[0].PipelineSpec.WorkflowManifest,
			PipelineID:       pipelines[0].ID,
			Parameters: []*run_model.APIParameter{
				{Name: "param1", Value: "hola"},
				{Name: "param2", Value: "world"},
			},
		},
		ResourceReferences: []*run_model.APIResourceReference{
			{Key: &run_model.APIResourceKey{Type: run_model.APIResourceTypeEXPERIMENT, ID: experiments[0].ID},
				Relationship: run_model.APIRelationshipOWNER,
			},
		},
		CreatedAt:   runs[0].CreatedAt,
		ScheduledAt: runs[0].ScheduledAt,
	}
	assert.Equal(t, expectedRun, runs[0])
}

func TestUpgrade(t *testing.T) {
	suite.Run(t, new(UpgradeTest))
}
