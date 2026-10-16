// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package driver

import (
	"context"
	"fmt"
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	api "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	clientmanager "github.com/kubeflow/pipelines/backend/src/v2/client_manager"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

type driverRetryArtifactFaultAPI struct {
	*driverRetryFaultAPI
	artifactTaskCalls int
	committedTaskID   string
}

func (f *driverRetryArtifactFaultAPI) CreateArtifactTasks(ctx context.Context, req *api.CreateArtifactTasksBulkRequest) (*api.CreateArtifactTasksBulkResponse, error) {
	f.artifactTaskCalls++
	response, err := f.MockAPI.CreateArtifactTasks(ctx, proto.Clone(req).(*api.CreateArtifactTasksBulkRequest))
	if err != nil {
		return nil, err
	}
	if f.artifactTaskCalls == 1 {
		f.committedTaskID = req.GetArtifactTasks()[0].GetTaskId()
		return nil, fmt.Errorf("lost artifact association response")
	}
	return proto.Clone(response).(*api.CreateArtifactTasksBulkResponse), nil
}

func TestDriverRetryContainerRecoversCommittedCachedArtifact(t *testing.T) {
	ctx := context.Background()
	tc, opts, faults := retryContainerContext(t)
	opts.CacheDisabled = false
	opts.Task = proto.Clone(opts.Task).(*pipelinespec.PipelineTaskSpec)
	opts.Task.CachingOptions = &pipelinespec.PipelineTaskSpec_CachingOptions{EnableCache: true}
	cacheSource, err := tc.MockAPI.CreateTask(ctx, &api.CreateTaskRequest{Task: &api.PipelineTask{
		TaskId: "cache-source", RunId: "previous-run", Name: "source-task", ScopePath: "root.source-task",
		Type: api.PipelineTask_RUNTIME, State: api.PipelineTask_SUCCEEDED,
	}})
	require.NoError(t, err)
	artifact, err := tc.MockAPI.CreateArtifact(ctx, &api.CreateArtifactRequest{
		TaskId: cacheSource.GetTaskId(), RunId: cacheSource.GetRunId(), ProducerKey: "output_dataset",
		Artifact: &api.Artifact{
			ArtifactId: "cached-dataset", Name: "cached dataset", Type: api.Artifact_Dataset,
			Uri: proto.String("s3://pipeline-artifacts/cached-dataset"),
		},
	})
	require.NoError(t, err)
	faults.cacheTask, err = tc.MockAPI.GetTask(ctx, &api.GetTaskRequest{
		TaskId: cacheSource.GetTaskId(), RunId: cacheSource.GetRunId(),
	})
	require.NoError(t, err)
	artifactFaults := &driverRetryArtifactFaultAPI{driverRetryFaultAPI: faults}
	tc.ClientManager = clientmanager.NewFakeClientManager(tc.ClientManager.K8sClient(), artifactFaults)

	_, err = Container(ctx, opts, tc.ClientManager)
	require.ErrorContains(t, err, "lost artifact association response")
	require.Equal(t, 1, artifactFaults.artifactTaskCalls)
	require.NotEmpty(t, artifactFaults.committedTaskID)
	task, err := tc.MockAPI.GetTask(ctx, &api.GetTaskRequest{
		TaskId: artifactFaults.committedTaskID, RunId: opts.Run.GetRunId(),
	})
	require.NoError(t, err)
	assert.Equal(t, api.PipelineTask_RUNNING, task.GetState())
	assert.Nil(t, task.EndTime)
	assert.Empty(t, task.GetStatusMetadata().GetCustomProperties()[driverCheckpointKey].GetStringValue())
	require.NotEmpty(t, task.GetStatusMetadata().GetCustomProperties()[driverCachedOutputsKey].GetStringValue())

	linksForTask := func() []*api.ArtifactTask {
		links, err := tc.MockAPI.ListArtifactTasks(ctx, &api.ListArtifactTasksRequest{})
		require.NoError(t, err)
		var result []*api.ArtifactTask
		for _, link := range links.GetArtifactTasks() {
			if link.GetTaskId() == artifactFaults.committedTaskID {
				result = append(result, link)
			}
		}
		return result
	}
	committedLinks := linksForTask()
	require.Len(t, committedLinks, 1, "the failed call already committed the cached artifact association")
	committedLink := proto.Clone(committedLinks[0]).(*api.ArtifactTask)
	cacheCalls := faults.cacheCalls
	require.Equal(t, 1, cacheCalls)
	faults.cacheErr = fmt.Errorf("cache unavailable after original hit")

	// The first replay resumes incomplete driver work; the second restores the
	// completed handoff. Both must keep the original frozen artifact decision.
	for attempt := 1; attempt <= 2; attempt++ {
		opts.DriverRetryAttempt = attempt
		opts.PodUID = fmt.Sprintf("artifact-retry-%d", attempt)
		execution, err := Container(ctx, opts, tc.ClientManager)
		require.NoError(t, err)
		require.NotNil(t, execution.Cached)
		assert.True(t, *execution.Cached)
		assert.Equal(t, artifactFaults.committedTaskID, execution.TaskID)
		assert.Equal(t, cacheCalls, faults.cacheCalls, "replay must use the original cached artifacts")
		assert.Equal(t, 1, artifactFaults.artifactTaskCalls, "replay must recognize the committed association before writing")
		links := linksForTask()
		require.Len(t, links, 1)
		assert.True(t, proto.Equal(committedLink, links[0]), "the original association must survive without replacement or duplication")
		task, err = tc.MockAPI.GetTask(ctx, &api.GetTaskRequest{TaskId: execution.TaskID, RunId: opts.Run.GetRunId()})
		require.NoError(t, err)
		assert.Equal(t, api.PipelineTask_CACHED, task.GetState())
		assert.Empty(t, task.GetStatusMetadata().GetMessage())
		require.Len(t, task.GetOutputs().GetArtifacts(), 1)
		output := task.GetOutputs().GetArtifacts()[0]
		assert.Equal(t, "output_dataset", output.GetArtifactKey())
		assert.Equal(t, api.IOType_OUTPUT, output.GetType())
		assert.Equal(t, opts.TaskName, output.GetProducer().GetTaskName())
		require.Len(t, output.GetArtifacts(), 1)
		assert.True(t, proto.Equal(artifact, output.GetArtifacts()[0]))
	}
}
