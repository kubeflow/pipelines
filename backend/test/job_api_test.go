package test

import (
	"context"
	"encoding/json"
	"time"

	"fmt"

	"testing"

	"io/ioutil"

	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/util/yaml"
)

type JobApiTestSuite struct {
	suite.Suite
	namespace        string
	conn             *grpc.ClientConn
	experimentClient api.ExperimentServiceClient
	jobClient        api.JobServiceClient
	pipelineClient   api.PipelineServiceClient
	runClient        api.RunServiceClient
}

// Check the namespace have ML pipeline installed and ready
func (s *JobApiTestSuite) SetupTest() {
	err := waitForReady(*namespace, *initializeTimeout)
	if err != nil {
		glog.Exitf("Failed to initialize test. Error: %s", err.Error())
	}
	s.namespace = *namespace
	s.conn, err = getRpcConnection(s.namespace)
	if err != nil {
		glog.Exitf("Failed to get RPC connection. Error: %s", err.Error())
	}
	s.experimentClient = api.NewExperimentServiceClient(s.conn)
	s.jobClient = api.NewJobServiceClient(s.conn)
	s.pipelineClient = api.NewPipelineServiceClient(s.conn)
	s.runClient = api.NewRunServiceClient(s.conn)
}

func (s *JobApiTestSuite) TearDownTest() {
	s.conn.Close()
}

func (s *JobApiTestSuite) TestJobApis() {
	t := s.T()
	clientSet, err := getKubernetesClient()
	if err != nil {
		t.Fatalf("Can't initialize a Kubernete client. Error: %s", err.Error())
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	/* ---------- Upload a pipeline ---------- */
	pipelineBody, writer := uploadPipelineFileOrFail("resources/hello-world.yaml")
	response, err := clientSet.RESTClient().Post().
		AbsPath(fmt.Sprintf(mlPipelineAPIServerBase, s.namespace, "pipelines/upload")).
		SetHeader("Content-Type", writer.FormDataContentType()).
		Body(pipelineBody).Do().Raw()
	assert.Nil(t, err)
	var helloWorldPipeline api.Pipeline
	json.Unmarshal(response, &helloWorldPipeline)

	/* ---------- Create a new hello world experiment ---------- */
	createExperimentRequest := &api.CreateExperimentRequest{Experiment: &api.Experiment{Name: "hello world experiment"}}
	helloWorldExperiment, err := s.experimentClient.CreateExperiment(ctx, createExperimentRequest)
	assert.Nil(t, err)

	/* ---------- Create a new hello world job by specifying pipeline ID ---------- */
	requestStartTime := time.Now().Unix()
	createJobRequest := &api.CreateJobRequest{Job: &api.Job{
		Name:        "hello world",
		Description: "this is hello world",
		PipelineSpec: &api.PipelineSpec{
			PipelineId: helloWorldPipeline.Id,
		},
		ResourceReferences: []*api.ResourceReference{
			{Key: &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: helloWorldExperiment.Id},
				Relationship: api.Relationship_OWNER},
		},
		MaxConcurrency: 10,
		Enabled:        true,
	}}
	helloWorldJob, err := s.jobClient.CreateJob(ctx, createJobRequest)
	assert.Nil(t, err)
	s.checkHelloWorldJob(t, helloWorldJob, helloWorldExperiment.Id, helloWorldPipeline.Id, requestStartTime)

	/* ---------- Get hello world job ---------- */
	helloWorldJob, err = s.jobClient.GetJob(ctx, &api.GetJobRequest{Id: helloWorldJob.Id})
	assert.Nil(t, err)
	s.checkHelloWorldJob(t, helloWorldJob, helloWorldExperiment.Id, helloWorldPipeline.Id, requestStartTime)

	/* ---------- Create a new argument parameter experiment ---------- */
	createExperimentRequest = &api.CreateExperimentRequest{Experiment: &api.Experiment{Name: "argument parameter experiment"}}
	argParamsExperiment, err := s.experimentClient.CreateExperiment(ctx, createExperimentRequest)
	assert.Nil(t, err)

	/* ---------- Create a new argument parameter job by uploading workflow manifest ---------- */
	requestStartTime = time.Now().Unix()
	argParamsBytes, err := ioutil.ReadFile("resources/arguments-parameters.yaml")
	assert.Nil(t, err)
	argParamsBytes, err = yaml.ToJSON(argParamsBytes)
	assert.Nil(t, err)
	createJobRequest = &api.CreateJobRequest{Job: &api.Job{
		Name:        "argument parameter",
		Description: "this is argument parameter",
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: string(argParamsBytes),
			Parameters: []*api.Parameter{
				{Name: "param1", Value: "goodbye"},
				{Name: "param2", Value: "world"},
			},
		},
		ResourceReferences: []*api.ResourceReference{
			{Key: &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: argParamsExperiment.Id},
				Relationship: api.Relationship_OWNER},
		},
		MaxConcurrency: 10,
		Enabled:        true,
	}}
	argParamsJob, err := s.jobClient.CreateJob(ctx, createJobRequest)
	assert.Nil(t, err)
	s.checkArgParamsJob(t, argParamsJob, argParamsExperiment.Id, requestStartTime)

	/* ---------- List all the jobs. Both jobs should be returned ---------- */
	listJobsResponse, err := s.jobClient.ListJobs(ctx, &api.ListJobsRequest{})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(listJobsResponse.Jobs))

	/* ---------- List the jobs, paginated, default sort ---------- */
	listJobsResponse, err = s.jobClient.ListJobs(ctx, &api.ListJobsRequest{PageSize: 1})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(listJobsResponse.Jobs))
	assert.Equal(t, "hello world", listJobsResponse.Jobs[0].Name)
	listJobsResponse, err = s.jobClient.ListJobs(ctx, &api.ListJobsRequest{PageSize: 1, PageToken: listJobsResponse.NextPageToken})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(listJobsResponse.Jobs))
	assert.Equal(t, "argument parameter", listJobsResponse.Jobs[0].Name)

	/* ---------- List the jobs, paginated, sort by name ---------- */
	listJobsResponse, err = s.jobClient.ListJobs(ctx, &api.ListJobsRequest{PageSize: 1, SortBy: "name"})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(listJobsResponse.Jobs))
	assert.Equal(t, "argument parameter", listJobsResponse.Jobs[0].Name)
	listJobsResponse, err = s.jobClient.ListJobs(ctx, &api.ListJobsRequest{PageSize: 1, SortBy: "name", PageToken: listJobsResponse.NextPageToken})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(listJobsResponse.Jobs))
	assert.Equal(t, "hello world", listJobsResponse.Jobs[0].Name)

	/* ---------- List the jobs, sort by unsupported field ---------- */
	_, err = s.jobClient.ListJobs(ctx, &api.ListJobsRequest{PageSize: 2, SortBy: "description"})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "InvalidArgument")

	/* ---------- List jobs for hello world experiment. One job should be returned ---------- */
	listJobsResponse, err = s.jobClient.ListJobs(ctx, &api.ListJobsRequest{
		ResourceReferenceKey: &api.ResourceKey{
			Type: api.ResourceType_EXPERIMENT, Id: helloWorldExperiment.Id}})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(listJobsResponse.Jobs))
	assert.Equal(t, "hello world", listJobsResponse.Jobs[0].Name)

	// The scheduledWorkflow CRD would create the run and it synced to the DB by persistent agent.
	// This could take a few seconds to finish.
	// TODO: Retry list run every 5 seconds instead of sleeping for 40 seconds.
	time.Sleep(40 * time.Second)

	/* ---------- Check run for hello world job ---------- */
	listRunsResponse, err := s.runClient.ListRuns(ctx, &api.ListRunsRequest{
		ResourceReferenceKey: &api.ResourceKey{
			Type: api.ResourceType_EXPERIMENT, Id: helloWorldExperiment.Id}})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(listRunsResponse.Runs))
	helloWorldRun := listRunsResponse.Runs[0]
	s.checkHelloWorldRun(t, helloWorldRun, helloWorldExperiment.Id, helloWorldJob.Id, requestStartTime)

	/* ---------- Check run for argument parameter job ---------- */
	listRunsResponse, err = s.runClient.ListRuns(ctx, &api.ListRunsRequest{
		ResourceReferenceKey: &api.ResourceKey{
			Type: api.ResourceType_EXPERIMENT, Id: argParamsExperiment.Id}})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(listRunsResponse.Runs))
	argParamsRun := listRunsResponse.Runs[0]
	s.checkArgParamsRun(t, argParamsRun, argParamsExperiment.Id, argParamsJob.Id, requestStartTime)

	/* ---------- Clean up ---------- */
	_, err = s.pipelineClient.DeletePipeline(ctx, &api.DeletePipelineRequest{Id: helloWorldPipeline.Id})
	assert.Nil(t, err)
	_, err = s.jobClient.DeleteJob(ctx, &api.DeleteJobRequest{Id: helloWorldJob.Id})
	assert.Nil(t, err)
	_, err = s.jobClient.DeleteJob(ctx, &api.DeleteJobRequest{Id: argParamsJob.Id})
	assert.Nil(t, err)
	_, err = s.runClient.DeleteRun(ctx, &api.DeleteRunRequest{Id: helloWorldRun.Id})
	assert.Nil(t, err)
	_, err = s.runClient.DeleteRun(ctx, &api.DeleteRunRequest{Id: argParamsRun.Id})
	assert.Nil(t, err)
	_, err = s.experimentClient.DeleteExperiment(ctx, &api.DeleteExperimentRequest{Id: helloWorldExperiment.Id})
	assert.Nil(t, err)
	_, err = s.experimentClient.DeleteExperiment(ctx, &api.DeleteExperimentRequest{Id: argParamsExperiment.Id})
	assert.Nil(t, err)
}

func (s *JobApiTestSuite) checkHelloWorldJob(t *testing.T, job *api.Job, experimentId string, pipelineId string, requestStartTime int64) {
	// Check workflow manifest is not empty
	assert.Contains(t, job.PipelineSpec.WorkflowManifest, "whalesay")
	assert.True(t, job.CreatedAt.Seconds >= requestStartTime)
	expectedJob := &api.Job{
		Id:          job.Id,
		Name:        "hello world",
		Description: "this is hello world",
		PipelineSpec: &api.PipelineSpec{
			PipelineId:       pipelineId,
			WorkflowManifest: job.PipelineSpec.WorkflowManifest,
		},
		ResourceReferences: []*api.ResourceReference{
			{Key: &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: experimentId},
				Relationship: api.Relationship_OWNER},
		},
		MaxConcurrency: 10,
		Enabled:        true,
		CreatedAt:      &timestamp.Timestamp{Seconds: job.CreatedAt.Seconds},
		UpdatedAt:      &timestamp.Timestamp{Seconds: job.UpdatedAt.Seconds},
		Status:         job.Status,
		Trigger:        &api.Trigger{},
	}

	assert.Equal(t, expectedJob, job)
}

func (s *JobApiTestSuite) checkArgParamsJob(t *testing.T, job *api.Job, experimentId string, requestStartTime int64) {
	argParamsBytes, err := ioutil.ReadFile("resources/arguments-parameters.yaml")
	assert.Nil(t, err)
	argParamsBytes, err = yaml.ToJSON(argParamsBytes)
	assert.Nil(t, err)
	// Check runtime workflow manifest is not empty
	assert.Contains(t, job.PipelineSpec.WorkflowManifest, "arguments-parameters-")
	assert.True(t, job.CreatedAt.Seconds >= requestStartTime)
	expectedJob := &api.Job{
		Id:          job.Id,
		Name:        "argument parameter",
		Description: "this is argument parameter",
		PipelineSpec: &api.PipelineSpec{
			WorkflowManifest: job.PipelineSpec.WorkflowManifest,
			Parameters: []*api.Parameter{
				{Name: "param1", Value: "goodbye"},
				{Name: "param2", Value: "world"},
			},
		},
		ResourceReferences: []*api.ResourceReference{
			{Key: &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: experimentId},
				Relationship: api.Relationship_OWNER},
		},
		MaxConcurrency: 10,
		Enabled:        true,
		CreatedAt:      &timestamp.Timestamp{Seconds: job.CreatedAt.Seconds},
		UpdatedAt:      &timestamp.Timestamp{Seconds: job.UpdatedAt.Seconds},
		Status:         job.Status,
		Trigger:        &api.Trigger{},
	}

	assert.Equal(t, expectedJob, job)
}

func (s *JobApiTestSuite) checkHelloWorldRun(t *testing.T, run *api.Run, experimentId string, jobId string, requestStartTime int64) {
	// Check workflow manifest is not empty
	assert.Contains(t, run.PipelineSpec.WorkflowManifest, "whalesay")
	assert.Contains(t, run.Name, "helloworld")
	// Check runtime workflow manifest is not empty
	assert.True(t, run.CreatedAt.Seconds >= requestStartTime)
	resourceReferences := []*api.ResourceReference{
		{Key: &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: experimentId},
			Relationship: api.Relationship_OWNER,
		},
		{Key: &api.ResourceKey{Type: api.ResourceType_JOB, Id: jobId},
			Relationship: api.Relationship_CREATOR,
		},
	}
	assert.Equal(t, resourceReferences, run.ResourceReferences)
}

func (s *JobApiTestSuite) checkArgParamsRun(t *testing.T, run *api.Run, experimentId string, jobId string, requestStartTime int64) {
	assert.Contains(t, run.Name, "argumentparameter")
	// Check runtime workflow manifest is not empty
	assert.True(t, run.CreatedAt.Seconds >= requestStartTime)
	resourceReferences := []*api.ResourceReference{
		{Key: &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: experimentId},
			Relationship: api.Relationship_OWNER,
		},
		{Key: &api.ResourceKey{Type: api.ResourceType_JOB, Id: jobId},
			Relationship: api.Relationship_CREATOR,
		},
	}
	assert.Equal(t, resourceReferences, run.ResourceReferences)
}

func TestJobApi(t *testing.T) {
	suite.Run(t, new(JobApiTestSuite))
}
