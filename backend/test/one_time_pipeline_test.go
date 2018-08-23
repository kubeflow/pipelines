// Copyright 2018 Google LLC
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

package test

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/cenkalti/backoff"
	"github.com/ghodss/yaml"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/googleprivate/ml/backend/api"
	"github.com/googleprivate/ml/backend/src/common/util"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"k8s.io/client-go/kubernetes"
)

var namespace = flag.String("namespace", "default", "The namespace ml pipeline deployed to")
var initializeTimeout = flag.Duration("initializeTimeout", 2*time.Minute, "Duration to wait for test initialization")
var testTimeout = flag.Duration("testTimeout", 2*time.Minute, "Duration to wait for the test to finish")

type OneTimeJobTestSuite struct {
	suite.Suite
	namespace      string
	conn           *grpc.ClientConn
	pipelineClient api.PipelineServiceClient
	jobClient      api.JobServiceClient
	runClient      api.RunServiceClient
}

// Check the namespace have ML pipeline installed and ready
func (s *OneTimeJobTestSuite) SetupTest() {
	err := waitForReady(*namespace, *initializeTimeout)
	if err != nil {
		glog.Exit("Failed to initialize test. Error: %s", err.Error())
	}
	s.namespace = *namespace
	s.conn, err = getRpcConnection(s.namespace)
	if err != nil {
		glog.Exit("Failed to get RPC connection. Error: %s", err.Error())
	}
	s.pipelineClient = api.NewPipelineServiceClient(s.conn)
	s.jobClient = api.NewJobServiceClient(s.conn)
	s.runClient = api.NewRunServiceClient(s.conn)
}

func (s *OneTimeJobTestSuite) TearDownTest() {
	s.conn.Close()
}

// This test case tests a basic use case:
// - Upload a pipeline
// - Create a job with parameter
// - Verify an one-time run is automatically scheduled.
func (s *OneTimeJobTestSuite) TestOneTimeJob_E2E() {
	t := s.T()
	clientSet, err := getKubernetesClient()
	if err != nil {
		t.Fatalf("Can't initialize a Kubernete client. Error: %s", err.Error())
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	/* ---------- Verify no pipeline exist ---------- */
	listPipelineResponse, err := s.pipelineClient.ListPipelines(ctx, &api.ListPipelinesRequest{})
	checkNoPipelineExists(t, listPipelineResponse, err)

	/* ---------- Verify no job exist ---------- */
	listJobResponse, err := s.jobClient.ListJobs(ctx, &api.ListJobsRequest{})
	checkNoJobExists(t, listJobResponse, err)

	/* ---------- Upload a pipeline ---------- */
	requestStartTime := time.Now().Unix()
	pipelineBody, writer := uploadPipelineFileOrFail("resources/arguments-parameters.yaml")
	uploadPipelineResponse, err := clientSet.RESTClient().Post().
		AbsPath(fmt.Sprintf(mlPipelineAPIServerBase, s.namespace, "pipelines/upload")).
		SetHeader("Content-Type", writer.FormDataContentType()).
		Body(pipelineBody).Do().Raw()
	checkUploadPipelineResponse(t, uploadPipelineResponse, err, requestStartTime)

	/* ---------- Verify list pipeline works ---------- */
	listPipelineResponse, err = s.pipelineClient.ListPipelines(ctx, &api.ListPipelinesRequest{})
	checkListPipelinesResponse(t, listPipelineResponse, err, requestStartTime)

	pipelineId := listPipelineResponse.Pipelines[0].Id

	/* ---------- Verify get pipeline works ---------- */
	getPipelineResponse, err := s.pipelineClient.GetPipeline(ctx, &api.GetPipelineRequest{Id: pipelineId})
	checkGetPipelineResponse(t, getPipelineResponse, err, requestStartTime)

	/* ---------- Verify get template works ---------- */
	getTmpResponse, err := s.pipelineClient.GetTemplate(ctx, &api.GetTemplateRequest{Id: pipelineId})
	checkGetTemplateResponse(t, getTmpResponse, err)

	/* ---------- Instantiate job using the pipeline ---------- */
	requestStartTime = time.Now().Unix()
	job := &api.Job{
		Name:        "hello-world",
		PipelineId:  pipelineId,
		Description: "this is my first job",
		Enabled:     true,
		Parameters: []*api.Parameter{
			{Name: "param1", Value: "goodbye"},
			{Name: "param2", Value: "world"},
		},
	}
	newJob, err := s.jobClient.CreateJob(ctx, &api.CreateJobRequest{Job: job})
	checkInstantiateJobResponse(t, newJob, err, requestStartTime, pipelineId)
	jobId := newJob.Id

	/* ---------- Verify list jobs works ---------- */
	listPipResponse, err := s.jobClient.ListJobs(ctx, &api.ListJobsRequest{})
	checkListJobsResponse(t, listPipResponse, err, requestStartTime, pipelineId)

	/* ---------- Verify get job works ---------- */
	getPipResponse, err := s.jobClient.GetJob(ctx, &api.GetJobRequest{Id: jobId})
	checkGetJobResponse(t, getPipResponse, err, requestStartTime, pipelineId)

	// The scheduledWorkflow CRD would create the run and it synced to the DB by persistent agent.
	// This could take a few seconds to finish.
	// TODO: Retry list run every 5 seconds instead of sleeping for 40 seconds.
	time.Sleep(40 * time.Second)

	/* ---------- Verify list run works ---------- */
	listJobRunsResponse, err := s.jobClient.ListJobRuns(ctx, &api.ListJobRunsRequest{JobId: jobId})
	checkListRunsResponse(t, listJobRunsResponse, err, jobId)
	runId := listJobRunsResponse.Runs[0].Id

	/* ---------- Verify run complete successfully ---------- */
	err = s.checkRunSucceed(clientSet, s.namespace, jobId, runId)
	if err != nil {
		assert.Fail(t, "The run doesn't complete. Error: "+err.Error())
	}

	/* ---------- Verify get run works ---------- */
	getRunResponse, err := s.runClient.GetRun(ctx, &api.GetRunRequest{RunId: runId})
	checkGetRunResponse(t, getRunResponse, err, jobId)

	/* ---------- Verify delete job works ---------- */
	_, err = s.jobClient.DeleteJob(ctx, &api.DeleteJobRequest{Id: jobId})
	assert.Nil(t, err)

	/* ---------- Verify no job exist ---------- */
	listPipResponse, err = s.jobClient.ListJobs(ctx, &api.ListJobsRequest{})
	checkNoJobExists(t, listPipResponse, err)

	/* ---------- Verify can't retrieve the run ---------- */
	_, err = s.runClient.GetRun(ctx, &api.GetRunRequest{RunId: runId})
	assert.NotNil(t, err)

	/* ---------- Verify delete pipeline works ---------- */
	_, err = s.pipelineClient.DeletePipeline(ctx, &api.DeletePipelineRequest{Id: pipelineId})
	assert.Nil(t, err)

	/* ---------- Verify no pipeline exist ---------- */
	listPipelineResponse, err = s.pipelineClient.ListPipelines(ctx, &api.ListPipelinesRequest{})
	checkNoPipelineExists(t, listPipelineResponse, err)
}

func (s *OneTimeJobTestSuite) checkRunSucceed(clientSet *kubernetes.Clientset, namespace string, jobId string, runId string) error {
	var waitForRunSucceed = func() error {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		getRunResponse, err := s.runClient.GetRun(ctx, &api.GetRunRequest{RunId: runId})
		if err != nil {
			return errors.New("Can't get run.")
		}
		status := getRunStatus(getRunResponse)
		if status == nil || (*status) == "" {
			return errors.New("Run hasn't started yet.")
		}
		if *status == v1alpha1.NodeSucceeded {
			return nil
		}
		if *status == v1alpha1.NodeRunning {
			return errors.New("Run is still running. Waiting for run to finish.")
		}
		return backoff.Permanent(errors.Errorf("Run failed to run with status " + string(*status)))
	}
	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = *testTimeout
	return backoff.Retry(waitForRunSucceed, b)
}

func getRunStatus(response *api.RunDetail) *v1alpha1.NodePhase {
	if response.Workflow != "" {
		var workflow v1alpha1.Workflow
		yaml.Unmarshal([]byte(response.Workflow), &workflow)
		return &workflow.Status.Phase
	}
	return nil
}

func checkNoPipelineExists(t *testing.T, response *api.ListPipelinesResponse, err error) {
	assert.Nil(t, err)
	assert.Empty(t, response.Pipelines)
}

func checkNoJobExists(t *testing.T, response *api.ListJobsResponse, err error) {
	assert.Nil(t, err)
	assert.Empty(t, response.Jobs)
}

func checkUploadPipelineResponse(t *testing.T, response []byte, err error, requestStartTime int64) {
	assert.Nil(t, err)
	var pipeline api.Pipeline
	util.UnmarshalJsonOrFail(string(response), &pipeline)
	verifyPipeline(t, &pipeline, requestStartTime)
}

func checkListPipelinesResponse(t *testing.T, response *api.ListPipelinesResponse, err error, requestStartTime int64) {
	assert.Nil(t, err)
	assert.Equal(t, 1, len(response.Pipelines))
	verifyPipeline(t, response.Pipelines[0], requestStartTime)
}

func checkGetPipelineResponse(t *testing.T, pipeline *api.Pipeline, err error, requestStartTime int64) {
	assert.Nil(t, err)
	verifyPipeline(t, pipeline, requestStartTime)
}

func checkGetTemplateResponse(t *testing.T, response *api.GetTemplateResponse, err error) {
	assert.Nil(t, err)
	expected, err := ioutil.ReadFile("resources/arguments-parameters.yaml")
	assert.Nil(t, err)
	assert.Equal(t, string(expected), response.Template)
}

func checkInstantiateJobResponse(t *testing.T, response *api.Job, err error, requestStartTime int64, expectPipelineId string) {
	assert.Nil(t, err)
	verifyJob(t, response, requestStartTime, expectPipelineId)
}

func checkListJobsResponse(t *testing.T, response *api.ListJobsResponse, err error, requestStartTime int64, expectPipelineId string) {
	assert.Nil(t, err)
	assert.Equal(t, 1, len(response.Jobs))
	verifyJob(t, response.Jobs[0], requestStartTime, expectPipelineId)
}

func checkGetJobResponse(t *testing.T, response *api.Job, err error, requestStartTime int64, expectPipelineId string) {
	assert.Nil(t, err)
	verifyJob(t, response, requestStartTime, expectPipelineId)
}

func checkListRunsResponse(t *testing.T, response *api.ListJobRunsResponse, err error, jobId string) {
	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, 1, len(response.Runs))
	assert.Equal(t, jobId, response.Runs[0].JobId)
}

func checkGetRunResponse(t *testing.T, response *api.RunDetail, err error, jobId string) {
	assert.Nil(t, err)
	assert.Equal(t, jobId, response.Run.JobId)

	// The Argo workflow might not be created. Only verify if it's created.
	if response.Workflow != "" {
		// Do some very basic verification to make sure argo workflow is returned
		assert.Contains(t, response.Workflow, response.Run.Name)
	}
}

func verifyPipeline(t *testing.T, pipeline *api.Pipeline, requestStartTime int64) {
	// Only verify the time fields have valid value and in the right range.
	assert.NotNil(t, *pipeline)
	assert.NotNil(t, pipeline.CreatedAt)
	// TODO: Investigate this. This is flaky for some reason.
	//assert.True(t, pipeline.CreatedAt.GetSeconds() >= requestStartTime)
	expected := api.Pipeline{
		Id:        pipeline.Id,
		CreatedAt: &timestamp.Timestamp{Seconds: pipeline.CreatedAt.Seconds},
		Name:      "arguments-parameters.yaml",
		Parameters: []*api.Parameter{
			{Name: "param1", Value: "hello"}, // Default value in the pipeline template
			{Name: "param2"},                 // No default value in the pipeline
		},
	}
	assert.Equal(t, expected, *pipeline)
}

func verifyJob(t *testing.T, job *api.Job, requestStartTime int64, expectedPipelineId string) {
	assert.NotNil(t, *job)
	assert.NotNil(t, job.CreatedAt)
	// Only verify the time fields have valid value and in the right range.
	assert.True(t, job.CreatedAt.Seconds >= requestStartTime)

	// Scrub the job status and updateAt, since it's changing as test progress.
	job.Status = ""
	job.UpdatedAt = nil

	expected := api.Job{
		Id:          job.Id,
		CreatedAt:   &timestamp.Timestamp{Seconds: job.CreatedAt.Seconds},
		Name:        "hello-world",
		Description: "this is my first job",
		PipelineId:  expectedPipelineId,
		Enabled:     true,
		Parameters: []*api.Parameter{
			{Name: "param1", Value: "goodbye"},
			{Name: "param2", Value: "world"},
		},
		Trigger: &api.Trigger{},
	}
	assert.Equal(t, expected, *job)
}

func TestOneTimeJob(t *testing.T) {
	suite.Run(t, new(OneTimeJobTestSuite))
}
