package integration

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/kubeflow/pipelines/backend/test"

	"github.com/golang/glog"
	experimentparams "github.com/kubeflow/pipelines/backend/api/go_http_client/experiment_client/experiment_service"
	"github.com/kubeflow/pipelines/backend/api/go_http_client/experiment_model"
	jobparams "github.com/kubeflow/pipelines/backend/api/go_http_client/job_client/job_service"
	"github.com/kubeflow/pipelines/backend/api/go_http_client/job_model"
	uploadParams "github.com/kubeflow/pipelines/backend/api/go_http_client/pipeline_upload_client/pipeline_upload_service"
	runParams "github.com/kubeflow/pipelines/backend/api/go_http_client/run_client/run_service"
	"github.com/kubeflow/pipelines/backend/api/go_http_client/run_model"
	"github.com/kubeflow/pipelines/backend/src/common/client/api_server"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/util/yaml"
)

type JobApiTestSuite struct {
	suite.Suite
	namespace            string
	conn                 *grpc.ClientConn
	experimentClient     *api_server.ExperimentClient
	pipelineClient       *api_server.PipelineClient
	pipelineUploadClient *api_server.PipelineUploadClient
	runClient            *api_server.RunClient
	jobClient            *api_server.JobClient
}

// Check the namespace have ML pipeline installed and ready
func (s *JobApiTestSuite) SetupTest() {
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
}

func (s *JobApiTestSuite) TestJobApis() {
	t := s.T()

	/* ---------- Upload pipelines YAML ---------- */
	helloWorldPipeline, err := s.pipelineUploadClient.UploadFile("../resources/hello-world.yaml", uploadParams.NewUploadPipelineParams())
	assert.Nil(t, err)

	/* ---------- Create a new hello world experiment ---------- */
	experiment := &experiment_model.APIExperiment{Name: "hello world experiment"}
	helloWorldExperiment, err := s.experimentClient.Create(&experimentparams.CreateExperimentParams{Body: experiment})
	assert.Nil(t, err)

	/* ---------- Create a new hello world job by specifying pipeline ID ---------- */
	createJobRequest := &jobparams.CreateJobParams{Body: &job_model.APIJob{
		Name:        "hello world",
		Description: "this is hello world",
		PipelineSpec: &job_model.APIPipelineSpec{
			PipelineID: helloWorldPipeline.ID,
		},
		ResourceReferences: []*job_model.APIResourceReference{
			{Key: &job_model.APIResourceKey{Type: job_model.APIResourceTypeEXPERIMENT, ID: helloWorldExperiment.ID},
				Relationship: job_model.APIRelationshipOWNER},
		},
		MaxConcurrency: 10,
		Enabled:        true,
	}}
	helloWorldJob, err := s.jobClient.Create(createJobRequest)
	assert.Nil(t, err)
	s.checkHelloWorldJob(t, helloWorldJob, helloWorldExperiment.ID, helloWorldPipeline.ID)

	/* ---------- Get hello world job ---------- */
	helloWorldJob, err = s.jobClient.Get(&jobparams.GetJobParams{ID: helloWorldJob.ID})
	assert.Nil(t, err)
	s.checkHelloWorldJob(t, helloWorldJob, helloWorldExperiment.ID, helloWorldPipeline.ID)

	/* ---------- Create a new argument parameter experiment ---------- */
	experiment = &experiment_model.APIExperiment{Name: "argument parameter experiment"}
	argParamsExperiment, err := s.experimentClient.Create(&experimentparams.CreateExperimentParams{Body: experiment})
	assert.Nil(t, err)

	/* ---------- Create a new argument parameter job by uploading workflow manifest ---------- */
	argParamsBytes, err := ioutil.ReadFile("../resources/arguments-parameters.yaml")
	assert.Nil(t, err)
	argParamsBytes, err = yaml.ToJSON(argParamsBytes)
	assert.Nil(t, err)
	createJobRequest = &jobparams.CreateJobParams{Body: &job_model.APIJob{
		Name:        "argument parameter",
		Description: "this is argument parameter",
		PipelineSpec: &job_model.APIPipelineSpec{
			WorkflowManifest: string(argParamsBytes),
			Parameters: []*job_model.APIParameter{
				{Name: "param1", Value: "goodbye"},
				{Name: "param2", Value: "world"},
			},
		},
		ResourceReferences: []*job_model.APIResourceReference{
			{Key: &job_model.APIResourceKey{Type: job_model.APIResourceTypeEXPERIMENT, ID: argParamsExperiment.ID},
				Relationship: job_model.APIRelationshipOWNER},
		},
		MaxConcurrency: 10,
		Enabled:        true,
	}}
	argParamsJob, err := s.jobClient.Create(createJobRequest)
	assert.Nil(t, err)
	s.checkArgParamsJob(t, argParamsJob, argParamsExperiment.ID)

	/* ---------- List all the jobs. Both jobs should be returned ---------- */
	jobs, totalSize, _, err := s.jobClient.List(&jobparams.ListJobsParams{})
	assert.Nil(t, err)
	assert.Equal(t, 2, totalSize)
	assert.Equal(t, 2, len(jobs))

	/* ---------- List the jobs, paginated, default sort ---------- */
	jobs, totalSize, nextPageToken, err := s.jobClient.List(&jobparams.ListJobsParams{PageSize: util.Int32Pointer(1)})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(jobs))
	assert.Equal(t, 2, totalSize)
	assert.Equal(t, "hello world", jobs[0].Name)
	jobs, totalSize, _, err = s.jobClient.List(&jobparams.ListJobsParams{
		PageSize: util.Int32Pointer(1), PageToken: util.StringPointer(nextPageToken)})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(jobs))
	assert.Equal(t, 2, totalSize)
	assert.Equal(t, "argument parameter", jobs[0].Name)

	/* ---------- List the jobs, paginated, sort by name ---------- */
	jobs, totalSize, nextPageToken, err = s.jobClient.List(&jobparams.ListJobsParams{
		PageSize: util.Int32Pointer(1), SortBy: util.StringPointer("name")})
	assert.Nil(t, err)
	assert.Equal(t, 2, totalSize)
	assert.Equal(t, 1, len(jobs))
	assert.Equal(t, "argument parameter", jobs[0].Name)
	jobs, totalSize, _, err = s.jobClient.List(&jobparams.ListJobsParams{
		PageSize: util.Int32Pointer(1), SortBy: util.StringPointer("name"), PageToken: util.StringPointer(nextPageToken)})
	assert.Nil(t, err)
	assert.Equal(t, 2, totalSize)
	assert.Equal(t, 1, len(jobs))
	assert.Equal(t, "hello world", jobs[0].Name)

	/* ---------- List the jobs, sort by unsupported field ---------- */
	jobs, _, _, err = s.jobClient.List(&jobparams.ListJobsParams{
		PageSize: util.Int32Pointer(2), SortBy: util.StringPointer("unknown")})
	assert.NotNil(t, err)

	/* ---------- List jobs for hello world experiment. One job should be returned ---------- */
	jobs, totalSize, _, err = s.jobClient.List(&jobparams.ListJobsParams{
		ResourceReferenceKeyType: util.StringPointer(string(run_model.APIResourceTypeEXPERIMENT)),
		ResourceReferenceKeyID:   util.StringPointer(helloWorldExperiment.ID)})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(jobs))
	assert.Equal(t, 1, totalSize)
	assert.Equal(t, "hello world", jobs[0].Name)

	// The scheduledWorkflow CRD would create the run and it synced to the DB by persistent agent.
	// This could take a few seconds to finish.
	// TODO: Retry list run every 5 seconds instead of sleeping for 40 seconds.
	time.Sleep(40 * time.Second)

	/* ---------- Check run for hello world job ---------- */
	runs, totalSize, _, err := s.runClient.List(&runParams.ListRunsParams{
		ResourceReferenceKeyType: util.StringPointer(string(run_model.APIResourceTypeEXPERIMENT)),
		ResourceReferenceKeyID:   util.StringPointer(helloWorldExperiment.ID)})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(runs))
	assert.Equal(t, 1, totalSize)
	helloWorldRun := runs[0]
	s.checkHelloWorldRun(t, helloWorldRun, helloWorldExperiment.ID, helloWorldJob.ID)

	/* ---------- Check run for argument parameter job ---------- */
	runs, totalSize, _, err = s.runClient.List(&runParams.ListRunsParams{
		ResourceReferenceKeyType: util.StringPointer(string(run_model.APIResourceTypeEXPERIMENT)),
		ResourceReferenceKeyID:   util.StringPointer(argParamsExperiment.ID)})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(runs))
	assert.Equal(t, 1, totalSize)
	argParamsRun := runs[0]
	s.checkArgParamsRun(t, argParamsRun, argParamsExperiment.ID, argParamsJob.ID)

	/* ---------- Clean up ---------- */
	test.DeleteAllExperiments(s.experimentClient, t)
	test.DeleteAllPipelines(s.pipelineClient, t)
	test.DeleteAllJobs(s.jobClient, t)
	test.DeleteAllRuns(s.runClient, t)
}

func (s *JobApiTestSuite) checkHelloWorldJob(t *testing.T, job *job_model.APIJob, experimentID string, pipelineID string) {
	// Check workflow manifest is not empty
	assert.Contains(t, job.PipelineSpec.WorkflowManifest, "whalesay")
	expectedJob := &job_model.APIJob{
		ID:          job.ID,
		Name:        "hello world",
		Description: "this is hello world",
		PipelineSpec: &job_model.APIPipelineSpec{
			PipelineID:       pipelineID,
			WorkflowManifest: job.PipelineSpec.WorkflowManifest,
		},
		ResourceReferences: []*job_model.APIResourceReference{
			{Key: &job_model.APIResourceKey{Type: job_model.APIResourceTypeEXPERIMENT, ID: experimentID},
				Relationship: job_model.APIRelationshipOWNER,
			},
		},
		MaxConcurrency: 10,
		Enabled:        true,
		CreatedAt:      job.CreatedAt,
		UpdatedAt:      job.UpdatedAt,
		Status:         job.Status,
		Trigger:        &job_model.APITrigger{},
	}

	assert.Equal(t, expectedJob, job)
}

func (s *JobApiTestSuite) checkArgParamsJob(t *testing.T, job *job_model.APIJob, experimentID string) {
	argParamsBytes, err := ioutil.ReadFile("../resources/arguments-parameters.yaml")
	assert.Nil(t, err)
	argParamsBytes, err = yaml.ToJSON(argParamsBytes)
	assert.Nil(t, err)
	// Check runtime workflow manifest is not empty
	assert.Contains(t, job.PipelineSpec.WorkflowManifest, "arguments-parameters-")
	expectedJob := &job_model.APIJob{
		ID:          job.ID,
		Name:        "argument parameter",
		Description: "this is argument parameter",
		PipelineSpec: &job_model.APIPipelineSpec{
			WorkflowManifest: job.PipelineSpec.WorkflowManifest,
			Parameters: []*job_model.APIParameter{
				{Name: "param1", Value: "goodbye"},
				{Name: "param2", Value: "world"},
			},
		},
		ResourceReferences: []*job_model.APIResourceReference{
			{Key: &job_model.APIResourceKey{Type: job_model.APIResourceTypeEXPERIMENT, ID: experimentID},
				Relationship: job_model.APIRelationshipOWNER,
			},
		},
		MaxConcurrency: 10,
		Enabled:        true,
		CreatedAt:      job.CreatedAt,
		UpdatedAt:      job.UpdatedAt,
		Status:         job.Status,
		Trigger:        &job_model.APITrigger{},
	}

	assert.Equal(t, expectedJob, job)
}

func (s *JobApiTestSuite) checkHelloWorldRun(t *testing.T, run *run_model.APIRun, experimentID string, jobID string) {
	// Check workflow manifest is not empty
	assert.Contains(t, run.PipelineSpec.WorkflowManifest, "whalesay")
	assert.Contains(t, run.Name, "helloworld")
	// Check runtime workflow manifest is not empty
	resourceReferences := []*run_model.APIResourceReference{
		{Key: &run_model.APIResourceKey{Type: run_model.APIResourceTypeEXPERIMENT, ID: experimentID},
			Relationship: run_model.APIRelationshipOWNER,
		},
		{Key: &run_model.APIResourceKey{Type: run_model.APIResourceTypeJOB, ID: jobID},
			Relationship: run_model.APIRelationshipCREATOR,
		},
	}
	assert.Equal(t, resourceReferences, run.ResourceReferences)
}

func (s *JobApiTestSuite) checkArgParamsRun(t *testing.T, run *run_model.APIRun, experimentID string, jobID string) {
	assert.Contains(t, run.Name, "argumentparameter")
	// Check runtime workflow manifest is not empty
	resourceReferences := []*run_model.APIResourceReference{
		{Key: &run_model.APIResourceKey{Type: run_model.APIResourceTypeEXPERIMENT, ID: experimentID},
			Relationship: run_model.APIRelationshipOWNER,
		},
		{Key: &run_model.APIResourceKey{Type: run_model.APIResourceTypeJOB, ID: jobID},
			Relationship: run_model.APIRelationshipCREATOR,
		},
	}
	assert.Equal(t, resourceReferences, run.ResourceReferences)
}

func TestJobApi(t *testing.T) {
	suite.Run(t, new(JobApiTestSuite))
}
