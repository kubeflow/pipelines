package kfpapi

import (
	"context"
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	gc "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

type recordingAPI struct {
	*MockAPI
	updatedTasks []*gc.PipelineTask
}

func (r *recordingAPI) UpdateTask(_ context.Context, req *gc.UpdateTaskRequest) (*gc.PipelineTask, error) {
	r.updatedTasks = append(r.updatedTasks, req.GetTask())
	return req.GetTask(), nil
}

func TestUpdateStatuses_FailsParentBeforeAllChildrenExist(t *testing.T) {
	pipelineSpec := &pipelinespec.PipelineSpec{
		Root: &pipelinespec.ComponentSpec{
			Implementation: &pipelinespec.ComponentSpec_Dag{
				Dag: &pipelinespec.DagSpec{
					Tasks: map[string]*pipelinespec.PipelineTaskSpec{
						"child-a": {TaskInfo: &pipelinespec.PipelineTaskInfo{Name: "child-a"}},
						"child-b": {TaskInfo: &pipelinespec.PipelineTaskInfo{Name: "child-b"}},
					},
				},
			},
		},
	}
	pipelineSpecJSON, err := protojson.Marshal(pipelineSpec)
	require.NoError(t, err)
	pipelineSpecStruct := &structpb.Struct{}
	require.NoError(t, protojson.Unmarshal(pipelineSpecJSON, pipelineSpecStruct))

	rootTask := &gc.PipelineTask{
		TaskId:    "root-task",
		RunId:     "run-1",
		Name:      "ROOT",
		Type:      gc.PipelineTask_ROOT,
		State:     gc.PipelineTask_RUNNING,
		ScopePath: "root",
	}
	failedChild := &gc.PipelineTask{
		TaskId:       "child-a-task",
		RunId:        "run-1",
		Name:         "child-a",
		ParentTaskId: &rootTask.TaskId,
		State:        gc.PipelineTask_FAILED,
	}
	run := &gc.Run{
		RunId: "run-1",
		Tasks: []*gc.PipelineTask{rootTask, failedChild},
	}

	api := &recordingAPI{MockAPI: NewMockAPI()}
	require.NoError(t, updateStatuses(context.Background(), run, api, pipelineSpecStruct, failedChild))
	require.NotEmpty(t, api.updatedTasks)
	lastUpdate := api.updatedTasks[len(api.updatedTasks)-1]
	require.Equal(t, gc.PipelineTask_FAILED, lastUpdate.GetState())
	require.Equal(t, rootTask.GetTaskId(), lastUpdate.GetTaskId())
}
