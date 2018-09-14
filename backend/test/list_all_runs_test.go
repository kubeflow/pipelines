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
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/golang/glog"
	api "github.com/googleprivate/ml/backend/api/go_client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
)

type ListAllRunsTestSuit struct {
	suite.Suite
	namespace      string
	conn           *grpc.ClientConn
	pipelineClient api.PipelineServiceClient
	jobClient      api.JobServiceClient
	runClient      api.RunServiceClient
}

// Check the namespace have ML job installed and ready
func (s *ListAllRunsTestSuit) SetupTest() {
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

func (s *ListAllRunsTestSuit) TearDownTest() {
	s.conn.Close()
}

func (s *ListAllRunsTestSuit) TestListAllRuns() {
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
		Param("name", base64.StdEncoding.EncodeToString([]byte("hello-world"))).
		SetHeader("Content-Type", writer.FormDataContentType()).
		Body(pipelineBody).Do().Raw()
	assert.Nil(t, err)

	pipelineBody, writer = uploadPipelineFileOrFail("resources/arguments-parameters.yaml")
	_, err = clientSet.RESTClient().Post().
		AbsPath(fmt.Sprintf(mlPipelineAPIServerBase, s.namespace, "pipelines/upload")).
		Param("name", base64.StdEncoding.EncodeToString([]byte("arguments-parameters"))).
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
	assert.Equal(t, "arguments-parameters", listFirstPagePipelineResponse.Pipelines[0].Name)
	assert.Equal(t, "hello-world", listFirstPagePipelineResponse.Pipelines[1].Name)
	assert.NotEmpty(t, listFirstPagePipelineResponse.NextPageToken)

	listSecondPagePipelineResponse, err := s.pipelineClient.ListPipelines(ctx, &api.ListPipelinesRequest{PageToken: listFirstPagePipelineResponse.NextPageToken, PageSize: 2, SortBy: "name"})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(listSecondPagePipelineResponse.Pipelines))
	assert.Equal(t, "loops.yaml", listSecondPagePipelineResponse.Pipelines[0].Name)
	assert.Empty(t, listSecondPagePipelineResponse.NextPageToken)

	argumentsParametersPipelineId := listFirstPagePipelineResponse.Pipelines[0].Id
	helloWorldPipelineId := listFirstPagePipelineResponse.Pipelines[1].Id
	loopsPipelineId := listSecondPagePipelineResponse.Pipelines[0].Id

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
		PipelineId: helloWorldPipelineId,
		Enabled:    true,
	}
	helloWorldJob, err = s.jobClient.CreateJob(ctx, &api.CreateJobRequest{Job: helloWorldJob})
	assert.Nil(t, err)

	// The scheduledWorkflow CRD would create the run and it synced to the DB by persistent agent.
	// This could take a few seconds to finish.
	// TODO: Retry list run every 5 seconds instead of sleeping for 40 seconds.
	time.Sleep(40 * time.Second)

	/* ---------- Verify three runs listed ---------- */
	listRunsResponse, err := s.runClient.ListRuns(ctx, &api.ListRunsRequest{})
	assert.Equal(t, 3, len(listRunsResponse.Runs))

	pipelineIds := []string{argumentsParametersJob.Id, helloWorldJob.Id, loopsJob.Id}
	// Assert the each job has one run associated with it
	assert.ElementsMatch(t, pipelineIds, []string{
		listRunsResponse.Runs[0].JobId,
		listRunsResponse.Runs[1].JobId,
		listRunsResponse.Runs[2].JobId})

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

func TestListAllRuns(t *testing.T) {
	suite.Run(t, new(ListAllRunsTestSuit))
}
