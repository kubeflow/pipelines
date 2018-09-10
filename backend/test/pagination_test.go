// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang/glog"
	api "github.com/googleprivate/ml/backend/api/go_client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
)

type PaginationTestSuit struct {
	suite.Suite
	namespace      string
	conn           *grpc.ClientConn
	pipelineClient api.PipelineServiceClient
	jobClient      api.JobServiceClient
	runClient      api.RunServiceClient
}

// Check the namespace have ML job installed and ready
func (s *PaginationTestSuit) SetupTest() {
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

func (s *PaginationTestSuit) TearDownTest() {
	s.conn.Close()
}

func (s *PaginationTestSuit) TestPagination_E2E() {
	t := s.T()
	clientSet, err := getKubernetesClient()
	if err != nil {
		t.Fatalf("Can't initialize a Kubernete client. Error: %s", err.Error())
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	/* ---------- Upload three pipelines ---------- */
	pipelineBody, writer := uploadPipelineFileOrFail("resources/hello-world.yaml")
	_, err = clientSet.RESTClient().Post().
		AbsPath(fmt.Sprintf(mlPipelineAPIServerBase, s.namespace, "pipelines/upload")).
		SetHeader("Content-Type", writer.FormDataContentType()).
		Body(pipelineBody).Do().Raw()
	assert.Nil(t, err)

	pipelineBody, writer = uploadPipelineFileOrFail("resources/arguments-parameters.yaml")
	_, err = clientSet.RESTClient().Post().
		AbsPath(fmt.Sprintf(mlPipelineAPIServerBase, s.namespace, "pipelines/upload")).
		SetHeader("Content-Type", writer.FormDataContentType()).
		Body(pipelineBody).Do().Raw()
	assert.Nil(t, err)

	pipelineBody, writer = uploadPipelineFileOrFail("resources/loops.yaml")
	_, err = clientSet.RESTClient().Post().
		AbsPath(fmt.Sprintf(mlPipelineAPIServerBase, s.namespace, "pipelines/upload")).
		SetHeader("Content-Type", writer.FormDataContentType()).
		Body(pipelineBody).Do().Raw()
	assert.Nil(t, err)

	/* ---------- List pipeline sorted by names ---------- */
	listFirstPagePipelineResponse, err := s.pipelineClient.ListPipelines(ctx, &api.ListPipelinesRequest{PageSize: 2, SortBy: "name"})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(listFirstPagePipelineResponse.Pipelines))
	assert.Equal(t, "arguments-parameters.yaml", listFirstPagePipelineResponse.Pipelines[0].Name)
	assert.Equal(t, "hello-world.yaml", listFirstPagePipelineResponse.Pipelines[1].Name)
	assert.NotEmpty(t, listFirstPagePipelineResponse.NextPageToken)

	listSecondPagePipelineResponse, err := s.pipelineClient.ListPipelines(ctx, &api.ListPipelinesRequest{PageToken: listFirstPagePipelineResponse.NextPageToken, PageSize: 2, SortBy: "name"})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(listSecondPagePipelineResponse.Pipelines))
	assert.Equal(t, "loops.yaml", listSecondPagePipelineResponse.Pipelines[0].Name)
	assert.Empty(t, listSecondPagePipelineResponse.NextPageToken)

	argumentsParametersPipelineId := listFirstPagePipelineResponse.Pipelines[0].Id
	helloWorldPipelineId := listFirstPagePipelineResponse.Pipelines[1].Id
	loopsPipelineId := listSecondPagePipelineResponse.Pipelines[0].Id

	/* ---------- List pipelines sort by unsupported description field. Should fail. ---------- */
	_, err = s.pipelineClient.ListPipelines(ctx, &api.ListPipelinesRequest{PageSize: 2, SortBy: "description"})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "InvalidArgument")

	/* ---------- List pipelines sorted by names descend order ---------- */
	listFirstPagePipelineResponse, err = s.pipelineClient.ListPipelines(ctx, &api.ListPipelinesRequest{PageSize: 2, SortBy: "name desc"})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(listFirstPagePipelineResponse.Pipelines))
	assert.Equal(t, "loops.yaml", listFirstPagePipelineResponse.Pipelines[0].Name)
	assert.Equal(t, "hello-world.yaml", listFirstPagePipelineResponse.Pipelines[1].Name)
	assert.NotEmpty(t, listFirstPagePipelineResponse.NextPageToken)

	listSecondPagePipelineResponse, err = s.pipelineClient.ListPipelines(ctx, &api.ListPipelinesRequest{PageToken: listFirstPagePipelineResponse.NextPageToken, PageSize: 2, SortBy: "name desc"})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(listSecondPagePipelineResponse.Pipelines))
	assert.Equal(t, "arguments-parameters.yaml", listSecondPagePipelineResponse.Pipelines[0].Name)
	assert.Empty(t, listSecondPagePipelineResponse.NextPageToken)

	/* ---------- Instantiate 3 job using the pipelines ---------- */
	loopsJob := &api.Job{
		Name:       "loops",
		PipelineId: loopsPipelineId,
		Enabled:    true,
	}
	loopsJob, err = s.jobClient.CreateJob(ctx, &api.CreateJobRequest{Job: loopsJob})
	assert.Nil(t, err)

	argumentsParametersJob := &api.Job{
		Name:       "arguments-parameters",
		PipelineId: argumentsParametersPipelineId,
		Enabled:    true,
		Parameters: []*api.Parameter{
			{Name: "param2", Value: "world"},
		},
	}
	argumentsParametersJob, err = s.jobClient.CreateJob(ctx, &api.CreateJobRequest{Job: argumentsParametersJob})
	assert.Nil(t, err)

	helloWorldJob := &api.Job{
		Name:       "hello-world",
		Enabled:    true,
		PipelineId: helloWorldPipelineId,
	}
	helloWorldJob, err = s.jobClient.CreateJob(ctx, &api.CreateJobRequest{Job: helloWorldJob})
	assert.Nil(t, err)

	/* ---------- List jobs sorted by names ---------- */
	listFirstPageJobResponse, err := s.jobClient.ListJobs(ctx, &api.ListJobsRequest{PageSize: 2, SortBy: "created_at"})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(listFirstPageJobResponse.Jobs))
	assert.Equal(t, "loops", listFirstPageJobResponse.Jobs[0].Name)
	assert.Equal(t, "arguments-parameters", listFirstPageJobResponse.Jobs[1].Name)
	assert.NotEmpty(t, listFirstPagePipelineResponse.NextPageToken)

	listSecondPageJobResponse, err := s.jobClient.ListJobs(ctx, &api.ListJobsRequest{PageToken: listFirstPageJobResponse.NextPageToken, PageSize: 2, SortBy: "created_at"})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(listSecondPageJobResponse.Jobs))
	assert.Equal(t, "hello-world", listSecondPageJobResponse.Jobs[0].Name)
	assert.Empty(t, listSecondPagePipelineResponse.NextPageToken)

	/* ---------- List jobs sort by unsupported description field. Should fail. ---------- */
	_, err = s.jobClient.ListJobs(ctx, &api.ListJobsRequest{PageSize: 2, SortBy: "description"})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "InvalidArgument")

	/* ---------- List jobs sorted by names ---------- */
	listFirstPageJobResponse, err = s.jobClient.ListJobs(ctx, &api.ListJobsRequest{PageSize: 2, SortBy: "created_at desc"})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(listFirstPageJobResponse.Jobs))
	assert.Equal(t, "hello-world", listFirstPageJobResponse.Jobs[0].Name)
	assert.Equal(t, "arguments-parameters", listFirstPageJobResponse.Jobs[1].Name)
	assert.NotEmpty(t, listFirstPagePipelineResponse.NextPageToken)

	listSecondPageJobResponse, err = s.jobClient.ListJobs(ctx, &api.ListJobsRequest{PageToken: listFirstPageJobResponse.NextPageToken, PageSize: 2, SortBy: "created_at desc"})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(listSecondPageJobResponse.Jobs))
	assert.Equal(t, "loops", listSecondPageJobResponse.Jobs[0].Name)
	assert.Empty(t, listSecondPagePipelineResponse.NextPageToken)

	// TODO(https://github.com/googleprivate/ml/issues/473): Add tests for list runs.

	/* ---------- Clean up ---------- */
	_, err = s.jobClient.DeleteJob(ctx, &api.DeleteJobRequest{Id: argumentsParametersJob.Id})
	assert.Nil(t, err)
	_, err = s.jobClient.DeleteJob(ctx, &api.DeleteJobRequest{Id: helloWorldJob.Id})
	assert.Nil(t, err)
	_, err = s.jobClient.DeleteJob(ctx, &api.DeleteJobRequest{Id: loopsJob.Id})
	assert.Nil(t, err)

	_, err = s.pipelineClient.DeletePipeline(ctx, &api.DeletePipelineRequest{Id: argumentsParametersPipelineId})
	assert.Nil(t, err)
	_, err = s.pipelineClient.DeletePipeline(ctx, &api.DeletePipelineRequest{Id: helloWorldPipelineId})
	assert.Nil(t, err)
	_, err = s.pipelineClient.DeletePipeline(ctx, &api.DeletePipelineRequest{Id: loopsPipelineId})
	assert.Nil(t, err)

}

func TestPagination(t *testing.T) {
	suite.Run(t, new(PaginationTestSuit))
}
