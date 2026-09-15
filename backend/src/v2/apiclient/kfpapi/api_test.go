package kfpapi

import (
	"context"
	"fmt"
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
	updatedTask, err := r.MockAPI.UpdateTask(context.Background(), req)
	if err != nil {
		return nil, err
	}
	r.updatedTasks = append(r.updatedTasks, updatedTask)
	return updatedTask, nil
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
	api.AddRun(run)
	for _, task := range run.GetTasks() {
		_, err := api.CreateTask(context.Background(), &gc.CreateTaskRequest{Task: task, RunId: run.GetRunId()})
		require.NoError(t, err)
	}
	require.NoError(t, updateStatuses(context.Background(), run, api, pipelineSpecStruct, failedChild))
	require.Len(t, api.updatedTasks, 1)
	require.Equal(t, gc.PipelineTask_FAILED, api.updatedTasks[0].GetState())
	require.Equal(t, rootTask.GetTaskId(), api.updatedTasks[0].GetTaskId())
}

func TestUpdateStatuses_PropagatesFailedDespiteRunningSibling(t *testing.T) {
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

	for _, order := range []string{"failed-first", "running-first"} {
		t.Run(order, func(t *testing.T) {
			rootTask := &gc.PipelineTask{
				TaskId:    "root-task",
				RunId:     "run-mixed",
				Name:      "ROOT",
				Type:      gc.PipelineTask_ROOT,
				State:     gc.PipelineTask_RUNNING,
				ScopePath: "root",
			}
			failedChild := &gc.PipelineTask{
				TaskId:       "child-a-task",
				RunId:        "run-mixed",
				Name:         "child-a",
				ParentTaskId: &rootTask.TaskId,
				State:        gc.PipelineTask_FAILED,
			}
			runningChild := &gc.PipelineTask{
				TaskId:       "child-b-task",
				RunId:        "run-mixed",
				Name:         "child-b",
				ParentTaskId: &rootTask.TaskId,
				State:        gc.PipelineTask_RUNNING,
			}
			tasks := []*gc.PipelineTask{rootTask, failedChild, runningChild}
			if order == "running-first" {
				tasks = []*gc.PipelineTask{rootTask, runningChild, failedChild}
			}
			run := &gc.Run{RunId: "run-mixed", Tasks: tasks}

			api := &recordingAPI{MockAPI: NewMockAPI()}
			api.AddRun(run)
			for _, task := range run.GetTasks() {
				_, err := api.CreateTask(context.Background(), &gc.CreateTaskRequest{Task: task, RunId: run.GetRunId()})
				require.NoError(t, err)
			}
			require.NoError(t, updateStatuses(context.Background(), run, api, pipelineSpecStruct, failedChild))
			require.Len(t, api.updatedTasks, 1)
			require.Equal(t, rootTask.GetTaskId(), api.updatedTasks[0].GetTaskId())
			require.Equal(t, gc.PipelineTask_FAILED, api.updatedTasks[0].GetState())
		})
	}
}

func TestUpdateStatuses_PropagatesThroughNestedParentsOnce(t *testing.T) {
	pipelineSpec := &pipelinespec.PipelineSpec{
		Root: &pipelinespec.ComponentSpec{
			Implementation: &pipelinespec.ComponentSpec_Dag{
				Dag: &pipelinespec.DagSpec{
					Tasks: map[string]*pipelinespec.PipelineTaskSpec{
						"mid": {
							TaskInfo:     &pipelinespec.PipelineTaskInfo{Name: "mid"},
							ComponentRef: &pipelinespec.ComponentRef{Name: "mid-comp"},
						},
					},
				},
			},
		},
		Components: map[string]*pipelinespec.ComponentSpec{
			"mid-comp": {
				Implementation: &pipelinespec.ComponentSpec_Dag{
					Dag: &pipelinespec.DagSpec{
						Tasks: map[string]*pipelinespec.PipelineTaskSpec{
							"leaf": {TaskInfo: &pipelinespec.PipelineTaskInfo{Name: "leaf"}},
						},
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
		RunId:     "run-nested",
		Name:      "ROOT",
		Type:      gc.PipelineTask_ROOT,
		State:     gc.PipelineTask_RUNNING,
		ScopePath: "root",
	}
	midTask := &gc.PipelineTask{
		TaskId:       "mid-task",
		RunId:        "run-nested",
		Name:         "mid",
		Type:         gc.PipelineTask_DAG,
		State:        gc.PipelineTask_RUNNING,
		ParentTaskId: &rootTask.TaskId,
		ScopePath:    "root.mid",
	}
	failedLeaf := &gc.PipelineTask{
		TaskId:       "leaf-task",
		RunId:        "run-nested",
		Name:         "leaf",
		ParentTaskId: &midTask.TaskId,
		State:        gc.PipelineTask_FAILED,
	}
	run := &gc.Run{
		RunId: "run-nested",
		Tasks: []*gc.PipelineTask{rootTask, midTask, failedLeaf},
	}

	api := &recordingAPI{MockAPI: NewMockAPI()}
	api.AddRun(run)
	for _, task := range run.GetTasks() {
		_, err := api.CreateTask(context.Background(), &gc.CreateTaskRequest{Task: task, RunId: run.GetRunId()})
		require.NoError(t, err)
	}
	require.NoError(t, updateStatuses(context.Background(), run, api, pipelineSpecStruct, failedLeaf))
	require.Len(t, api.updatedTasks, 2)
	require.Equal(t, midTask.GetTaskId(), api.updatedTasks[0].GetTaskId())
	require.Equal(t, gc.PipelineTask_FAILED, api.updatedTasks[0].GetState())
	require.Equal(t, rootTask.GetTaskId(), api.updatedTasks[1].GetTaskId())
	require.Equal(t, gc.PipelineTask_FAILED, api.updatedTasks[1].GetState())
}

func TestUpdateStatuses_AllSucceededUpdatesParentOnce(t *testing.T) {
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
		RunId:     "run-success",
		Name:      "ROOT",
		Type:      gc.PipelineTask_ROOT,
		State:     gc.PipelineTask_RUNNING,
		ScopePath: "root",
	}
	childA := &gc.PipelineTask{
		TaskId:       "child-a-task",
		RunId:        "run-success",
		Name:         "child-a",
		ParentTaskId: &rootTask.TaskId,
		State:        gc.PipelineTask_SUCCEEDED,
	}
	childB := &gc.PipelineTask{
		TaskId:       "child-b-task",
		RunId:        "run-success",
		Name:         "child-b",
		ParentTaskId: &rootTask.TaskId,
		State:        gc.PipelineTask_SUCCEEDED,
	}
	run := &gc.Run{RunId: "run-success", Tasks: []*gc.PipelineTask{rootTask, childA, childB}}

	api := &recordingAPI{MockAPI: NewMockAPI()}
	api.AddRun(run)
	for _, task := range run.GetTasks() {
		_, err := api.CreateTask(context.Background(), &gc.CreateTaskRequest{Task: task, RunId: run.GetRunId()})
		require.NoError(t, err)
	}
	require.NoError(t, updateStatuses(context.Background(), run, api, pipelineSpecStruct, childA))
	require.Len(t, api.updatedTasks, 1)
	require.Equal(t, rootTask.GetTaskId(), api.updatedTasks[0].GetTaskId())
	require.Equal(t, gc.PipelineTask_SUCCEEDED, api.updatedTasks[0].GetState())
}

func TestUpdateStatuses_CachedAndSkippedAggregation(t *testing.T) {
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

	runCase := func(t *testing.T, runID string, childAState, childBState, wantParentState gc.PipelineTask_TaskState, triggerChild string) {
		rootTask := &gc.PipelineTask{
			TaskId:    "root-task",
			RunId:     runID,
			Name:      "ROOT",
			Type:      gc.PipelineTask_ROOT,
			State:     gc.PipelineTask_RUNNING,
			ScopePath: "root",
		}
		childA := &gc.PipelineTask{
			TaskId:       "child-a-task",
			RunId:        runID,
			Name:         "child-a",
			ParentTaskId: &rootTask.TaskId,
			State:        childAState,
		}
		childB := &gc.PipelineTask{
			TaskId:       "child-b-task",
			RunId:        runID,
			Name:         "child-b",
			ParentTaskId: &rootTask.TaskId,
			State:        childBState,
		}
		run := &gc.Run{RunId: runID, Tasks: []*gc.PipelineTask{rootTask, childA, childB}}

		api := &recordingAPI{MockAPI: NewMockAPI()}
		api.AddRun(run)
		for _, task := range run.GetTasks() {
			_, err := api.CreateTask(context.Background(), &gc.CreateTaskRequest{Task: task, RunId: run.GetRunId()})
			require.NoError(t, err)
		}

		triggerTask := childA
		if triggerChild == "child-b" {
			triggerTask = childB
		}
		require.NoError(t, updateStatuses(context.Background(), run, api, pipelineSpecStruct, triggerTask))
		require.Len(t, api.updatedTasks, 1)
		require.Equal(t, rootTask.GetTaskId(), api.updatedTasks[0].GetTaskId())
		require.Equal(t, wantParentState, api.updatedTasks[0].GetState())
	}

	t.Run("all CACHED children -> parent CACHED", func(t *testing.T) {
		runCase(t, "run-cached", gc.PipelineTask_CACHED, gc.PipelineTask_CACHED, gc.PipelineTask_CACHED, "child-a")
	})

	t.Run("all SKIPPED children -> parent SKIPPED", func(t *testing.T) {
		runCase(t, "run-skipped", gc.PipelineTask_SKIPPED, gc.PipelineTask_SKIPPED, gc.PipelineTask_SKIPPED, "child-b")
	})

	t.Run("SUCCEEDED plus CACHED children -> parent SUCCEEDED", func(t *testing.T) {
		runCase(t, "run-succeeded-cached", gc.PipelineTask_SUCCEEDED, gc.PipelineTask_CACHED, gc.PipelineTask_SUCCEEDED, "child-a")
	})

	t.Run("SUCCEEDED plus SKIPPED children -> parent SUCCEEDED", func(t *testing.T) {
		runCase(t, "run-succeeded-skipped", gc.PipelineTask_SUCCEEDED, gc.PipelineTask_SKIPPED, gc.PipelineTask_SUCCEEDED, "child-b")
	})

	t.Run("CACHED plus SKIPPED children -> parent SUCCEEDED", func(t *testing.T) {
		runCase(t, "run-cached-skipped", gc.PipelineTask_CACHED, gc.PipelineTask_SKIPPED, gc.PipelineTask_SUCCEEDED, "child-a")
	})
}

func TestUpdateStatuses_LoopCardinality(t *testing.T) {
	// Two loop-body tasks × two iterations requires four children before the
	// loop can complete. A regression that multiplies only by iteration_count
	// (ignoring body task count) would incorrectly succeed with two children.
	pipelineSpec := &pipelinespec.PipelineSpec{
		Root: &pipelinespec.ComponentSpec{
			Implementation: &pipelinespec.ComponentSpec_Dag{
				Dag: &pipelinespec.DagSpec{
					Tasks: map[string]*pipelinespec.PipelineTaskSpec{
						"loop": {
							TaskInfo:     &pipelinespec.PipelineTaskInfo{Name: "loop"},
							ComponentRef: &pipelinespec.ComponentRef{Name: "loop-comp"},
						},
					},
				},
			},
		},
		Components: map[string]*pipelinespec.ComponentSpec{
			"loop-comp": {
				Implementation: &pipelinespec.ComponentSpec_Dag{
					Dag: &pipelinespec.DagSpec{
						Tasks: map[string]*pipelinespec.PipelineTaskSpec{
							"body-a": {TaskInfo: &pipelinespec.PipelineTaskInfo{Name: "body-a"}},
							"body-b": {TaskInfo: &pipelinespec.PipelineTaskInfo{Name: "body-b"}},
						},
					},
				},
			},
		},
	}
	pipelineSpecJSON, err := protojson.Marshal(pipelineSpec)
	require.NoError(t, err)
	pipelineSpecStruct := &structpb.Struct{}
	require.NoError(t, protojson.Unmarshal(pipelineSpecJSON, pipelineSpecStruct))

	iterationCount := int64(2)

	t.Run("4 SUCCEEDED children (2 tasks x 2 iters) -> parent SUCCEEDED", func(t *testing.T) {
		rootTask := &gc.PipelineTask{
			TaskId:    "root-task",
			RunId:     "run-loop",
			Name:      "ROOT",
			Type:      gc.PipelineTask_ROOT,
			State:     gc.PipelineTask_RUNNING,
			ScopePath: "root",
		}
		loopTask := &gc.PipelineTask{
			TaskId:       "loop-task",
			RunId:        "run-loop",
			Name:         "loop",
			Type:         gc.PipelineTask_LOOP,
			State:        gc.PipelineTask_RUNNING,
			ParentTaskId: &rootTask.TaskId,
			ScopePath:    "root.loop",
			TypeAttributes: &gc.PipelineTask_TypeAttributes{
				IterationCount: &iterationCount,
			},
		}
		bodyChildren := []*gc.PipelineTask{
			{TaskId: "body-a-0", RunId: "run-loop", Name: "body-a", ParentTaskId: &loopTask.TaskId, State: gc.PipelineTask_SUCCEEDED},
			{TaskId: "body-b-0", RunId: "run-loop", Name: "body-b", ParentTaskId: &loopTask.TaskId, State: gc.PipelineTask_SUCCEEDED},
			{TaskId: "body-a-1", RunId: "run-loop", Name: "body-a", ParentTaskId: &loopTask.TaskId, State: gc.PipelineTask_SUCCEEDED},
			{TaskId: "body-b-1", RunId: "run-loop", Name: "body-b", ParentTaskId: &loopTask.TaskId, State: gc.PipelineTask_SUCCEEDED},
		}
		runTasks := []*gc.PipelineTask{rootTask, loopTask}
		runTasks = append(runTasks, bodyChildren...)
		run := &gc.Run{RunId: "run-loop", Tasks: runTasks}

		api := &recordingAPI{MockAPI: NewMockAPI()}
		api.AddRun(run)
		for _, task := range run.GetTasks() {
			_, err := api.CreateTask(context.Background(), &gc.CreateTaskRequest{Task: task, RunId: run.GetRunId()})
			require.NoError(t, err)
		}
		require.NoError(t, updateStatuses(context.Background(), run, api, pipelineSpecStruct, bodyChildren[3]))
		require.Len(t, api.updatedTasks, 2)
		require.Equal(t, loopTask.GetTaskId(), api.updatedTasks[0].GetTaskId())
		require.Equal(t, gc.PipelineTask_SUCCEEDED, api.updatedTasks[0].GetState())
		require.Equal(t, rootTask.GetTaskId(), api.updatedTasks[1].GetTaskId())
		require.Equal(t, gc.PipelineTask_SUCCEEDED, api.updatedTasks[1].GetState())
	})

	t.Run("only 2 children when 4 expected -> no update (still waiting)", func(t *testing.T) {
		rootTask := &gc.PipelineTask{
			TaskId:    "root-task",
			RunId:     "run-loop-partial",
			Name:      "ROOT",
			Type:      gc.PipelineTask_ROOT,
			State:     gc.PipelineTask_RUNNING,
			ScopePath: "root",
		}
		loopTask := &gc.PipelineTask{
			TaskId:       "loop-task",
			RunId:        "run-loop-partial",
			Name:         "loop",
			Type:         gc.PipelineTask_LOOP,
			State:        gc.PipelineTask_RUNNING,
			ParentTaskId: &rootTask.TaskId,
			ScopePath:    "root.loop",
			TypeAttributes: &gc.PipelineTask_TypeAttributes{
				IterationCount: &iterationCount,
			},
		}
		// Two children equals iteration_count alone; body task count must also
		// be applied (expected = 2 iterations × 2 body tasks = 4).
		bodyChild0 := &gc.PipelineTask{
			TaskId:       "body-a-0",
			RunId:        "run-loop-partial",
			Name:         "body-a",
			ParentTaskId: &loopTask.TaskId,
			State:        gc.PipelineTask_SUCCEEDED,
		}
		bodyChild1 := &gc.PipelineTask{
			TaskId:       "body-a-1",
			RunId:        "run-loop-partial",
			Name:         "body-a",
			ParentTaskId: &loopTask.TaskId,
			State:        gc.PipelineTask_SUCCEEDED,
		}
		run := &gc.Run{RunId: "run-loop-partial", Tasks: []*gc.PipelineTask{rootTask, loopTask, bodyChild0, bodyChild1}}

		api := &recordingAPI{MockAPI: NewMockAPI()}
		api.AddRun(run)
		for _, task := range run.GetTasks() {
			_, err := api.CreateTask(context.Background(), &gc.CreateTaskRequest{Task: task, RunId: run.GetRunId()})
			require.NoError(t, err)
		}
		require.NoError(t, updateStatuses(context.Background(), run, api, pipelineSpecStruct, bodyChild1))
		require.Empty(t, api.updatedTasks, "no update expected when children < iteration_count * body_tasks")
	})
}

func TestUpdateStatuses_ThreeLevelPropagation(t *testing.T) {
	pipelineSpec := &pipelinespec.PipelineSpec{
		Root: &pipelinespec.ComponentSpec{
			Implementation: &pipelinespec.ComponentSpec_Dag{
				Dag: &pipelinespec.DagSpec{
					Tasks: map[string]*pipelinespec.PipelineTaskSpec{
						"outer": {
							TaskInfo:     &pipelinespec.PipelineTaskInfo{Name: "outer"},
							ComponentRef: &pipelinespec.ComponentRef{Name: "outer-comp"},
						},
					},
				},
			},
		},
		Components: map[string]*pipelinespec.ComponentSpec{
			"outer-comp": {
				Implementation: &pipelinespec.ComponentSpec_Dag{
					Dag: &pipelinespec.DagSpec{
						Tasks: map[string]*pipelinespec.PipelineTaskSpec{
							"mid": {
								TaskInfo:     &pipelinespec.PipelineTaskInfo{Name: "mid"},
								ComponentRef: &pipelinespec.ComponentRef{Name: "mid-comp"},
							},
						},
					},
				},
			},
			"mid-comp": {
				Implementation: &pipelinespec.ComponentSpec_Dag{
					Dag: &pipelinespec.DagSpec{
						Tasks: map[string]*pipelinespec.PipelineTaskSpec{
							"leaf": {TaskInfo: &pipelinespec.PipelineTaskInfo{Name: "leaf"}},
						},
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
		RunId:     "run-3level",
		Name:      "ROOT",
		Type:      gc.PipelineTask_ROOT,
		State:     gc.PipelineTask_RUNNING,
		ScopePath: "root",
	}
	outerTask := &gc.PipelineTask{
		TaskId:       "outer-task",
		RunId:        "run-3level",
		Name:         "outer",
		Type:         gc.PipelineTask_DAG,
		State:        gc.PipelineTask_RUNNING,
		ParentTaskId: &rootTask.TaskId,
		ScopePath:    "root.outer",
	}
	midTask := &gc.PipelineTask{
		TaskId:       "mid-task",
		RunId:        "run-3level",
		Name:         "mid",
		Type:         gc.PipelineTask_DAG,
		State:        gc.PipelineTask_RUNNING,
		ParentTaskId: &outerTask.TaskId,
		ScopePath:    "root.outer.mid",
	}
	leafTask := &gc.PipelineTask{
		TaskId:       "leaf-task",
		RunId:        "run-3level",
		Name:         "leaf",
		ParentTaskId: &midTask.TaskId,
		State:        gc.PipelineTask_FAILED,
	}
	run := &gc.Run{RunId: "run-3level", Tasks: []*gc.PipelineTask{rootTask, outerTask, midTask, leafTask}}

	api := &recordingAPI{MockAPI: NewMockAPI()}
	api.AddRun(run)
	for _, task := range run.GetTasks() {
		_, err := api.CreateTask(context.Background(), &gc.CreateTaskRequest{Task: task, RunId: run.GetRunId()})
		require.NoError(t, err)
	}
	require.NoError(t, updateStatuses(context.Background(), run, api, pipelineSpecStruct, leafTask))
	require.Len(t, api.updatedTasks, 3, "expect 3 updates: mid, outer, root")
	require.Equal(t, midTask.GetTaskId(), api.updatedTasks[0].GetTaskId())
	require.Equal(t, gc.PipelineTask_FAILED, api.updatedTasks[0].GetState())
	require.Equal(t, outerTask.GetTaskId(), api.updatedTasks[1].GetTaskId())
	require.Equal(t, gc.PipelineTask_FAILED, api.updatedTasks[1].GetState())
	require.Equal(t, rootTask.GetTaskId(), api.updatedTasks[2].GetTaskId())
	require.Equal(t, gc.PipelineTask_FAILED, api.updatedTasks[2].GetState())
}

func TestUpdateStatuses_MissingParentErrors(t *testing.T) {
	pipelineSpec := &pipelinespec.PipelineSpec{
		Root: &pipelinespec.ComponentSpec{
			Implementation: &pipelinespec.ComponentSpec_Dag{
				Dag: &pipelinespec.DagSpec{
					Tasks: map[string]*pipelinespec.PipelineTaskSpec{
						"child": {TaskInfo: &pipelinespec.PipelineTaskInfo{Name: "child"}},
					},
				},
			},
		},
	}
	pipelineSpecJSON, err := protojson.Marshal(pipelineSpec)
	require.NoError(t, err)
	pipelineSpecStruct := &structpb.Struct{}
	require.NoError(t, protojson.Unmarshal(pipelineSpecJSON, pipelineSpecStruct))

	missingParentID := "nonexistent-parent"
	child := &gc.PipelineTask{
		TaskId:       "child-task",
		RunId:        "run-missing-parent",
		Name:         "child",
		ParentTaskId: &missingParentID,
		State:        gc.PipelineTask_FAILED,
	}
	run := &gc.Run{RunId: "run-missing-parent", Tasks: []*gc.PipelineTask{child}}

	api := &recordingAPI{MockAPI: NewMockAPI()}
	api.AddRun(run)
	_, err = api.CreateTask(context.Background(), &gc.CreateTaskRequest{Task: child, RunId: run.GetRunId()})
	require.NoError(t, err)

	err = updateStatuses(context.Background(), run, api, pipelineSpecStruct, child)
	require.Error(t, err)
	require.Contains(t, err.Error(), "parent task")
	require.Contains(t, err.Error(), "not found")
}

// failingGetRunRecordingAPI wraps recordingAPI but fails GetRun after a
// configurable number of successful parent updates.
type failingGetRunRecordingAPI struct {
	*MockAPI
	updatedTasks      []*gc.PipelineTask
	updateCount       int
	failAfterNUpdates int
	getRunCount       int
}

func (f *failingGetRunRecordingAPI) UpdateTask(_ context.Context, req *gc.UpdateTaskRequest) (*gc.PipelineTask, error) {
	updatedTask, err := f.MockAPI.UpdateTask(context.Background(), req)
	if err != nil {
		return nil, err
	}
	f.updatedTasks = append(f.updatedTasks, updatedTask)
	f.updateCount++
	return updatedTask, nil
}

func (f *failingGetRunRecordingAPI) GetRun(_ context.Context, req *gc.GetRunRequest) (*gc.Run, error) {
	f.getRunCount++
	if f.updateCount >= f.failAfterNUpdates {
		return nil, fmt.Errorf("simulated GetRun failure after %d updates", f.updateCount)
	}
	return f.MockAPI.GetRun(context.Background(), req)
}

// failingUpdateTaskAPI fails the first UpdateTask call and records GetRun
// invocations so tests can assert traversal stops without refresh.
type failingUpdateTaskAPI struct {
	*MockAPI
	updateAttempts int
	getRunCount    int
}

func (f *failingUpdateTaskAPI) UpdateTask(_ context.Context, _ *gc.UpdateTaskRequest) (*gc.PipelineTask, error) {
	f.updateAttempts++
	return nil, fmt.Errorf("simulated UpdateTask failure")
}

func (f *failingUpdateTaskAPI) GetRun(_ context.Context, req *gc.GetRunRequest) (*gc.Run, error) {
	f.getRunCount++
	return f.MockAPI.GetRun(context.Background(), req)
}

func TestUpdateStatuses_RefreshOrUpdateFailure(t *testing.T) {
	pipelineSpec := &pipelinespec.PipelineSpec{
		Root: &pipelinespec.ComponentSpec{
			Implementation: &pipelinespec.ComponentSpec_Dag{
				Dag: &pipelinespec.DagSpec{
					Tasks: map[string]*pipelinespec.PipelineTaskSpec{
						"mid": {
							TaskInfo:     &pipelinespec.PipelineTaskInfo{Name: "mid"},
							ComponentRef: &pipelinespec.ComponentRef{Name: "mid-comp"},
						},
					},
				},
			},
		},
		Components: map[string]*pipelinespec.ComponentSpec{
			"mid-comp": {
				Implementation: &pipelinespec.ComponentSpec_Dag{
					Dag: &pipelinespec.DagSpec{
						Tasks: map[string]*pipelinespec.PipelineTaskSpec{
							"leaf": {TaskInfo: &pipelinespec.PipelineTaskInfo{Name: "leaf"}},
						},
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
		RunId:     "run-fail-refresh",
		Name:      "ROOT",
		Type:      gc.PipelineTask_ROOT,
		State:     gc.PipelineTask_RUNNING,
		ScopePath: "root",
	}
	midTask := &gc.PipelineTask{
		TaskId:       "mid-task",
		RunId:        "run-fail-refresh",
		Name:         "mid",
		Type:         gc.PipelineTask_DAG,
		State:        gc.PipelineTask_RUNNING,
		ParentTaskId: &rootTask.TaskId,
		ScopePath:    "root.mid",
	}
	leafTask := &gc.PipelineTask{
		TaskId:       "leaf-task",
		RunId:        "run-fail-refresh",
		Name:         "leaf",
		ParentTaskId: &midTask.TaskId,
		State:        gc.PipelineTask_FAILED,
	}
	run := &gc.Run{RunId: "run-fail-refresh", Tasks: []*gc.PipelineTask{rootTask, midTask, leafTask}}

	mockAPI := NewMockAPI()
	mockAPI.AddRun(run)
	for _, task := range run.GetTasks() {
		_, err := mockAPI.CreateTask(context.Background(), &gc.CreateTaskRequest{Task: task, RunId: run.GetRunId()})
		require.NoError(t, err)
	}

	api := &failingGetRunRecordingAPI{
		MockAPI:           mockAPI,
		failAfterNUpdates: 1,
	}

	err = updateStatuses(context.Background(), run, api, pipelineSpecStruct, leafTask)
	require.Error(t, err)
	require.Contains(t, err.Error(), "simulated GetRun failure")
	require.Len(t, api.updatedTasks, 1, "first parent should have been updated before failure")
	require.Equal(t, midTask.GetTaskId(), api.updatedTasks[0].GetTaskId())
}

func TestUpdateStatuses_UpdateTaskFailureStopsWithoutRefresh(t *testing.T) {
	pipelineSpec := &pipelinespec.PipelineSpec{
		Root: &pipelinespec.ComponentSpec{
			Implementation: &pipelinespec.ComponentSpec_Dag{
				Dag: &pipelinespec.DagSpec{
					Tasks: map[string]*pipelinespec.PipelineTaskSpec{
						"mid": {
							TaskInfo:     &pipelinespec.PipelineTaskInfo{Name: "mid"},
							ComponentRef: &pipelinespec.ComponentRef{Name: "mid-comp"},
						},
					},
				},
			},
		},
		Components: map[string]*pipelinespec.ComponentSpec{
			"mid-comp": {
				Implementation: &pipelinespec.ComponentSpec_Dag{
					Dag: &pipelinespec.DagSpec{
						Tasks: map[string]*pipelinespec.PipelineTaskSpec{
							"leaf": {TaskInfo: &pipelinespec.PipelineTaskInfo{Name: "leaf"}},
						},
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
		RunId:     "run-fail-update",
		Name:      "ROOT",
		Type:      gc.PipelineTask_ROOT,
		State:     gc.PipelineTask_RUNNING,
		ScopePath: "root",
	}
	midTask := &gc.PipelineTask{
		TaskId:       "mid-task",
		RunId:        "run-fail-update",
		Name:         "mid",
		Type:         gc.PipelineTask_DAG,
		State:        gc.PipelineTask_RUNNING,
		ParentTaskId: &rootTask.TaskId,
		ScopePath:    "root.mid",
	}
	leafTask := &gc.PipelineTask{
		TaskId:       "leaf-task",
		RunId:        "run-fail-update",
		Name:         "leaf",
		ParentTaskId: &midTask.TaskId,
		State:        gc.PipelineTask_FAILED,
	}
	run := &gc.Run{RunId: "run-fail-update", Tasks: []*gc.PipelineTask{rootTask, midTask, leafTask}}

	mockAPI := NewMockAPI()
	mockAPI.AddRun(run)
	for _, task := range run.GetTasks() {
		_, err := mockAPI.CreateTask(context.Background(), &gc.CreateTaskRequest{Task: task, RunId: run.GetRunId()})
		require.NoError(t, err)
	}

	api := &failingUpdateTaskAPI{MockAPI: mockAPI}
	err = updateStatuses(context.Background(), run, api, pipelineSpecStruct, leafTask)
	require.Error(t, err)
	require.Contains(t, err.Error(), "simulated UpdateTask failure")
	require.Equal(t, 1, api.updateAttempts, "should stop after the first UpdateTask failure")
	require.Zero(t, api.getRunCount, "must not refresh after UpdateTask failure")
}

func TestUpdateStatuses_ZeroIterationSkippedLoopPropagatesToRoot(t *testing.T) {
	// Production DAG driver marks a childless zero-iteration loop SKIPPED and
	// then propagates that terminal task. Model that path here: the loop is
	// already SKIPPED with no children, and the root must become SKIPPED.
	pipelineSpec := &pipelinespec.PipelineSpec{
		Root: &pipelinespec.ComponentSpec{
			Implementation: &pipelinespec.ComponentSpec_Dag{
				Dag: &pipelinespec.DagSpec{
					Tasks: map[string]*pipelinespec.PipelineTaskSpec{
						"loop": {
							TaskInfo:     &pipelinespec.PipelineTaskInfo{Name: "loop"},
							ComponentRef: &pipelinespec.ComponentRef{Name: "loop-comp"},
						},
					},
				},
			},
		},
		Components: map[string]*pipelinespec.ComponentSpec{
			"loop-comp": {
				Implementation: &pipelinespec.ComponentSpec_Dag{
					Dag: &pipelinespec.DagSpec{
						Tasks: map[string]*pipelinespec.PipelineTaskSpec{
							"body-task": {TaskInfo: &pipelinespec.PipelineTaskInfo{Name: "body-task"}},
						},
					},
				},
			},
		},
	}
	pipelineSpecJSON, err := protojson.Marshal(pipelineSpec)
	require.NoError(t, err)
	pipelineSpecStruct := &structpb.Struct{}
	require.NoError(t, protojson.Unmarshal(pipelineSpecJSON, pipelineSpecStruct))

	zeroIterations := int64(0)
	rootTask := &gc.PipelineTask{
		TaskId:    "root-task",
		RunId:     "run-zero-iter",
		Name:      "ROOT",
		Type:      gc.PipelineTask_ROOT,
		State:     gc.PipelineTask_RUNNING,
		ScopePath: "root",
	}
	loopTask := &gc.PipelineTask{
		TaskId:       "loop-task",
		RunId:        "run-zero-iter",
		Name:         "loop",
		Type:         gc.PipelineTask_LOOP,
		State:        gc.PipelineTask_SKIPPED,
		ParentTaskId: &rootTask.TaskId,
		ScopePath:    "root.loop",
		TypeAttributes: &gc.PipelineTask_TypeAttributes{
			IterationCount: &zeroIterations,
		},
	}
	run := &gc.Run{RunId: "run-zero-iter", Tasks: []*gc.PipelineTask{rootTask, loopTask}}

	api := &recordingAPI{MockAPI: NewMockAPI()}
	api.AddRun(run)
	for _, task := range run.GetTasks() {
		_, err := api.CreateTask(context.Background(), &gc.CreateTaskRequest{Task: task, RunId: run.GetRunId()})
		require.NoError(t, err)
	}
	require.NoError(t, updateStatuses(context.Background(), run, api, pipelineSpecStruct, loopTask))
	require.Len(t, api.updatedTasks, 1, "expect exactly one root update; loop is already terminal")
	require.Equal(t, rootTask.GetTaskId(), api.updatedTasks[0].GetTaskId())
	require.Equal(t, gc.PipelineTask_SKIPPED, api.updatedTasks[0].GetState())
}
