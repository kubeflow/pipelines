// Copyright 2026 The Kubeflow Authors
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

package component

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient/kfpapi"
	"github.com/kubeflow/pipelines/backend/src/v2/client_manager"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
	"k8s.io/client-go/kubernetes/fake"
)

func TestParentNeedsOutputRepublish(t *testing.T) {
	paramDefs := &pipelinespec.ComponentOutputsSpec{
		Parameters: map[string]*pipelinespec.ComponentOutputsSpec_ParameterSpec{
			"pipeline-output": {ParameterType: pipelinespec.ParameterType_STRING},
		},
	}
	artifactDefs := &pipelinespec.ComponentOutputsSpec{
		Artifacts: map[string]*pipelinespec.ComponentOutputsSpec_ArtifactSpec{
			"model": {},
		},
	}
	mixedDefs := &pipelinespec.ComponentOutputsSpec{
		Parameters: map[string]*pipelinespec.ComponentOutputsSpec_ParameterSpec{
			"pipeline-output": {ParameterType: pipelinespec.ParameterType_STRING},
		},
		Artifacts: map[string]*pipelinespec.ComponentOutputsSpec_ArtifactSpec{
			"model": {},
		},
	}

	assert.False(t, ParentNeedsOutputRepublish(nil, paramDefs))
	assert.False(t, ParentNeedsOutputRepublish(&apiv2beta1.PipelineTask{}, nil))
	assert.False(t, ParentNeedsOutputRepublish(&apiv2beta1.PipelineTask{}, &pipelinespec.ComponentOutputsSpec{}))
	assert.True(t, ParentNeedsOutputRepublish(&apiv2beta1.PipelineTask{}, paramDefs))
	assert.True(t, ParentNeedsOutputRepublish(&apiv2beta1.PipelineTask{
		Outputs: &apiv2beta1.PipelineTask_InputOutputs{},
	}, paramDefs))
	assert.False(t, ParentNeedsOutputRepublish(&apiv2beta1.PipelineTask{
		Outputs: &apiv2beta1.PipelineTask_InputOutputs{
			Parameters: []*apiv2beta1.PipelineTask_InputOutputs_IOParameter{{
				ParameterKey: "pipeline-output",
				Value:        structpb.NewStringValue("v"),
			}},
		},
	}, paramDefs))
	// Artifacts present but declared parameter still missing.
	assert.True(t, ParentNeedsOutputRepublish(&apiv2beta1.PipelineTask{
		Outputs: &apiv2beta1.PipelineTask_InputOutputs{
			Artifacts: []*apiv2beta1.PipelineTask_InputOutputs_IOArtifact{{
				ArtifactKey: "model",
				Artifacts:   []*apiv2beta1.Artifact{{ArtifactId: "art-1"}},
			}},
		},
	}, mixedDefs))
	assert.False(t, ParentNeedsOutputRepublish(&apiv2beta1.PipelineTask{
		Outputs: &apiv2beta1.PipelineTask_InputOutputs{
			Artifacts: []*apiv2beta1.PipelineTask_InputOutputs_IOArtifact{{
				ArtifactKey: "model",
				Artifacts:   []*apiv2beta1.Artifact{{ArtifactId: "art-1"}},
			}},
		},
	}, artifactDefs))
}

func TestOmitArtifactTasksAlreadyPresentOnTasks(t *testing.T) {
	mockAPI := kfpapi.NewMockAPI()
	run := &apiv2beta1.Run{RunId: "run-omit"}
	mockAPI.AddRun(run)

	parentID := "parent"
	ancestorID := "ancestor"
	artifactID := "art-1"

	_, err := mockAPI.CreateTask(context.Background(), &apiv2beta1.CreateTaskRequest{
		RunId: run.GetRunId(),
		Task: &apiv2beta1.PipelineTask{
			TaskId: parentID,
			RunId:  run.GetRunId(),
			Name:   "nested",
			State:  apiv2beta1.PipelineTask_RUNNING,
			Type:   apiv2beta1.PipelineTask_DAG,
		},
	})
	require.NoError(t, err)
	_, err = mockAPI.CreateTask(context.Background(), &apiv2beta1.CreateTaskRequest{
		RunId: run.GetRunId(),
		Task: &apiv2beta1.PipelineTask{
			TaskId: ancestorID,
			RunId:  run.GetRunId(),
			Name:   "root",
			State:  apiv2beta1.PipelineTask_RUNNING,
			Type:   apiv2beta1.PipelineTask_DAG,
		},
	})
	require.NoError(t, err)
	_, err = mockAPI.CreateArtifact(context.Background(), &apiv2beta1.CreateArtifactRequest{
		Artifact: &apiv2beta1.Artifact{ArtifactId: artifactID, Name: "model", Uri: util.StringPointer("gs://bucket/model")},
		TaskId:   parentID,
		RunId:    run.GetRunId(),
	})
	require.NoError(t, err)
	_, err = mockAPI.CreateArtifactTasks(context.Background(), &apiv2beta1.CreateArtifactTasksBulkRequest{
		ArtifactTasks: []*apiv2beta1.ArtifactTask{
			{ArtifactId: artifactID, TaskId: parentID, RunId: run.GetRunId(), Key: "existing-model", Type: apiv2beta1.IOType_OUTPUT},
			{ArtifactId: artifactID, TaskId: ancestorID, RunId: run.GetRunId(), Key: "existing-model", Type: apiv2beta1.IOType_OUTPUT},
		},
	})
	require.NoError(t, err)

	batchUpdater := NewBatchUpdater()
	batchUpdater.QueueArtifactTask(&apiv2beta1.ArtifactTask{
		ArtifactId: artifactID,
		TaskId:     parentID,
		Key:        "missing-model",
		Type:       apiv2beta1.IOType_OUTPUT,
	})
	batchUpdater.QueueArtifactTask(&apiv2beta1.ArtifactTask{
		ArtifactId: artifactID,
		TaskId:     parentID,
		Key:        "existing-model",
		Type:       apiv2beta1.IOType_OUTPUT,
	})
	batchUpdater.QueueArtifactTask(&apiv2beta1.ArtifactTask{
		ArtifactId: artifactID,
		TaskId:     ancestorID,
		Key:        "existing-model",
		Type:       apiv2beta1.IOType_OUTPUT,
	})

	require.NoError(t, batchUpdater.OmitArtifactTasksAlreadyPresentOnTasks(
		context.Background(), mockAPI, run.GetRunId(),
	))
	require.Len(t, batchUpdater.artifactTasks, 1)
	assert.Equal(t, parentID, batchUpdater.artifactTasks[0].GetTaskId())
	assert.Equal(t, "missing-model", batchUpdater.artifactTasks[0].GetKey())
}

func TestOmitArtifactTasksAlreadyPresentOnTasks_KeepsDistinctSameKeyArtifacts(t *testing.T) {
	mockAPI := kfpapi.NewMockAPI()
	run := &apiv2beta1.Run{RunId: "run-omit-multi"}
	mockAPI.AddRun(run)

	parentID := "parent"
	existingArtifactID := "art-existing"
	missingArtifactID := "art-missing"

	_, err := mockAPI.CreateTask(context.Background(), &apiv2beta1.CreateTaskRequest{
		RunId: run.GetRunId(),
		Task: &apiv2beta1.PipelineTask{
			TaskId: parentID,
			RunId:  run.GetRunId(),
			Name:   "producer",
			State:  apiv2beta1.PipelineTask_RUNNING,
			Type:   apiv2beta1.PipelineTask_RUNTIME,
		},
	})
	require.NoError(t, err)
	_, err = mockAPI.CreateArtifact(context.Background(), &apiv2beta1.CreateArtifactRequest{
		Artifact: &apiv2beta1.Artifact{ArtifactId: existingArtifactID, Name: "model-a", Uri: util.StringPointer("gs://bucket/a")},
		TaskId:   parentID,
		RunId:    run.GetRunId(),
	})
	require.NoError(t, err)
	_, err = mockAPI.CreateArtifactTasks(context.Background(), &apiv2beta1.CreateArtifactTasksBulkRequest{
		ArtifactTasks: []*apiv2beta1.ArtifactTask{
			{ArtifactId: existingArtifactID, TaskId: parentID, RunId: run.GetRunId(), Key: "models", Type: apiv2beta1.IOType_OUTPUT},
		},
	})
	require.NoError(t, err)

	batchUpdater := NewBatchUpdater()
	batchUpdater.QueueArtifactTask(&apiv2beta1.ArtifactTask{
		ArtifactId: existingArtifactID,
		TaskId:     parentID,
		Key:        "models",
		Type:       apiv2beta1.IOType_OUTPUT,
	})
	batchUpdater.QueueArtifactTask(&apiv2beta1.ArtifactTask{
		ArtifactId: missingArtifactID,
		TaskId:     parentID,
		Key:        "models",
		Type:       apiv2beta1.IOType_OUTPUT,
	})
	iter0 := int64(0)
	iter1 := int64(1)
	batchUpdater.QueueArtifactTask(&apiv2beta1.ArtifactTask{
		ArtifactId: "art-iter-0",
		TaskId:     parentID,
		Key:        "loop-out",
		Type:       apiv2beta1.IOType_ITERATOR_OUTPUT,
		Producer:   &apiv2beta1.IOProducer{TaskName: "producer", Iteration: &iter0},
	})
	batchUpdater.QueueArtifactTask(&apiv2beta1.ArtifactTask{
		ArtifactId: "art-iter-1",
		TaskId:     parentID,
		Key:        "loop-out",
		Type:       apiv2beta1.IOType_ITERATOR_OUTPUT,
		Producer:   &apiv2beta1.IOProducer{TaskName: "producer", Iteration: &iter1},
	})

	require.NoError(t, batchUpdater.OmitArtifactTasksAlreadyPresentOnTasks(
		context.Background(), mockAPI, run.GetRunId(),
	))
	require.Len(t, batchUpdater.artifactTasks, 3)
	keys := make([]string, 0, len(batchUpdater.artifactTasks))
	for _, artifactTask := range batchUpdater.artifactTasks {
		keys = append(keys, artifactTask.GetArtifactId())
	}
	assert.ElementsMatch(t, []string{missingArtifactID, "art-iter-0", "art-iter-1"}, keys)
}

func TestOmitArtifactTasksAlreadyPresentOnTasks_IteratorPartialRepair(t *testing.T) {
	mockAPI := kfpapi.NewMockAPI()
	run := &apiv2beta1.Run{RunId: "run-omit-iter"}
	mockAPI.AddRun(run)

	parentID := "parent"
	sharedArtifactID := "art-shared"
	iter0 := int64(0)
	iter1 := int64(1)

	_, err := mockAPI.CreateTask(context.Background(), &apiv2beta1.CreateTaskRequest{
		RunId: run.GetRunId(),
		Task: &apiv2beta1.PipelineTask{
			TaskId: parentID,
			RunId:  run.GetRunId(),
			Name:   "producer",
			State:  apiv2beta1.PipelineTask_RUNNING,
			Type:   apiv2beta1.PipelineTask_RUNTIME,
		},
	})
	require.NoError(t, err)
	_, err = mockAPI.CreateArtifact(context.Background(), &apiv2beta1.CreateArtifactRequest{
		Artifact:       &apiv2beta1.Artifact{ArtifactId: sharedArtifactID, Name: "loop-0", Uri: util.StringPointer("gs://bucket/0")},
		TaskId:         parentID,
		RunId:          run.GetRunId(),
		ProducerKey:    "loop-out",
		IterationIndex: &iter0,
	})
	require.NoError(t, err)

	batchUpdater := NewBatchUpdater()
	// Exact duplicate of the seeded (artifact, task, type, iteration, key) — must be omitted.
	batchUpdater.QueueArtifactTask(&apiv2beta1.ArtifactTask{
		ArtifactId: sharedArtifactID,
		TaskId:     parentID,
		Key:        "loop-out",
		Type:       apiv2beta1.IOType_ITERATOR_OUTPUT,
		Producer:   &apiv2beta1.IOProducer{TaskName: "producer", Iteration: &iter0},
	})
	// Same artifact/task/type/key, different iteration — must be kept.
	batchUpdater.QueueArtifactTask(&apiv2beta1.ArtifactTask{
		ArtifactId: sharedArtifactID,
		TaskId:     parentID,
		Key:        "loop-out",
		Type:       apiv2beta1.IOType_ITERATOR_OUTPUT,
		Producer:   &apiv2beta1.IOProducer{TaskName: "producer", Iteration: &iter1},
	})
	// Same artifact/task/iteration/key, different I/O type — must be kept.
	batchUpdater.QueueArtifactTask(&apiv2beta1.ArtifactTask{
		ArtifactId: sharedArtifactID,
		TaskId:     parentID,
		Key:        "loop-out",
		Type:       apiv2beta1.IOType_OUTPUT,
		Producer:   &apiv2beta1.IOProducer{TaskName: "producer", Iteration: &iter0},
	})
	// Same artifact/task/type/iteration, different key — must be kept.
	batchUpdater.QueueArtifactTask(&apiv2beta1.ArtifactTask{
		ArtifactId: sharedArtifactID,
		TaskId:     parentID,
		Key:        "other-out",
		Type:       apiv2beta1.IOType_ITERATOR_OUTPUT,
		Producer:   &apiv2beta1.IOProducer{TaskName: "producer", Iteration: &iter0},
	})

	require.NoError(t, batchUpdater.OmitArtifactTasksAlreadyPresentOnTasks(
		context.Background(), mockAPI, run.GetRunId(),
	))
	require.Len(t, batchUpdater.artifactTasks, 3)
	keptKeys := make([]string, 0, len(batchUpdater.artifactTasks))
	for _, artifactTask := range batchUpdater.artifactTasks {
		require.Equal(t, sharedArtifactID, artifactTask.GetArtifactId())
		require.Equal(t, parentID, artifactTask.GetTaskId())
		keptKeys = append(keptKeys, fmt.Sprintf("%v|%d|%s",
			artifactTask.GetType(),
			artifactTaskIterationIdentity(artifactTask),
			artifactTask.GetKey(),
		))
	}
	assert.ElementsMatch(t, []string{
		fmt.Sprintf("%v|%d|%s", apiv2beta1.IOType_ITERATOR_OUTPUT, iter1, "loop-out"),
		fmt.Sprintf("%v|%d|%s", apiv2beta1.IOType_OUTPUT, iter0, "loop-out"),
		fmt.Sprintf("%v|%d|%s", apiv2beta1.IOType_ITERATOR_OUTPUT, iter0, "other-out"),
	}, keptKeys)
}

func TestRepublishPreservedChildOutputsToDAG_RestoresParentParamsFromSucceededSibling(t *testing.T) {
	pipelineSpec := &pipelinespec.PipelineSpec{
		Root: &pipelinespec.ComponentSpec{
			Implementation: &pipelinespec.ComponentSpec_Dag{
				Dag: &pipelinespec.DagSpec{
					Tasks: map[string]*pipelinespec.PipelineTaskSpec{
						"success-child": {
							TaskInfo:     &pipelinespec.PipelineTaskInfo{Name: "success-child"},
							ComponentRef: &pipelinespec.ComponentRef{Name: "success-comp"},
						},
						"failed-child": {
							TaskInfo:     &pipelinespec.PipelineTaskInfo{Name: "failed-child"},
							ComponentRef: &pipelinespec.ComponentRef{Name: "failed-comp"},
						},
					},
					Outputs: &pipelinespec.DagOutputsSpec{
						Parameters: map[string]*pipelinespec.DagOutputsSpec_DagOutputParameterSpec{
							"pipeline-output": {
								Kind: &pipelinespec.DagOutputsSpec_DagOutputParameterSpec_ValueFromParameter{
									ValueFromParameter: &pipelinespec.DagOutputsSpec_ParameterSelectorSpec{
										ProducerSubtask:    "success-child",
										OutputParameterKey: "result",
									},
								},
							},
						},
					},
				},
			},
			OutputDefinitions: &pipelinespec.ComponentOutputsSpec{
				Parameters: map[string]*pipelinespec.ComponentOutputsSpec_ParameterSpec{
					"pipeline-output": {ParameterType: pipelinespec.ParameterType_STRING},
				},
			},
		},
		Components: map[string]*pipelinespec.ComponentSpec{
			"success-comp": {
				Implementation: &pipelinespec.ComponentSpec_ExecutorLabel{ExecutorLabel: "success"},
				OutputDefinitions: &pipelinespec.ComponentOutputsSpec{
					Parameters: map[string]*pipelinespec.ComponentOutputsSpec_ParameterSpec{
						"result": {ParameterType: pipelinespec.ParameterType_STRING},
					},
				},
			},
			"failed-comp": {
				Implementation: &pipelinespec.ComponentSpec_ExecutorLabel{ExecutorLabel: "failed"},
			},
		},
	}
	pipelineSpecStruct, err := pipelineSpecToStruct(t, pipelineSpec)
	require.NoError(t, err)

	run := &apiv2beta1.Run{RunId: "run-republish"}
	parentTaskID := "parent-dag"
	successChildID := "success-child-task"
	failedChildID := "failed-child-task"
	parentTaskIDPtr := util.StringPointer(parentTaskID)

	parentTask := &apiv2beta1.PipelineTask{
		TaskId:    parentTaskID,
		RunId:     run.GetRunId(),
		Name:      "root",
		State:     apiv2beta1.PipelineTask_RUNNING,
		Type:      apiv2beta1.PipelineTask_DAG,
		ScopePath: "root",
		// Simulate retry reset: parent outputs cleared.
		Outputs: &apiv2beta1.PipelineTask_InputOutputs{},
	}
	successChild := &apiv2beta1.PipelineTask{
		TaskId:       successChildID,
		RunId:        run.GetRunId(),
		Name:         "success-child",
		State:        apiv2beta1.PipelineTask_SUCCEEDED,
		Type:         apiv2beta1.PipelineTask_RUNTIME,
		ScopePath:    "root.success-child",
		ParentTaskId: parentTaskIDPtr,
		Outputs: &apiv2beta1.PipelineTask_InputOutputs{
			Parameters: []*apiv2beta1.PipelineTask_InputOutputs_IOParameter{{
				ParameterKey: "result",
				Value:        structpb.NewStringValue("kept"),
				Type:         apiv2beta1.IOType_OUTPUT,
				Producer:     &apiv2beta1.IOProducer{TaskName: "success-child"},
			}},
		},
	}
	failedChild := &apiv2beta1.PipelineTask{
		TaskId:       failedChildID,
		RunId:        run.GetRunId(),
		Name:         "failed-child",
		State:        apiv2beta1.PipelineTask_FAILED,
		Type:         apiv2beta1.PipelineTask_RUNTIME,
		ScopePath:    "root.failed-child",
		ParentTaskId: parentTaskIDPtr,
		Outputs:      &apiv2beta1.PipelineTask_InputOutputs{},
	}

	mockAPI := kfpapi.NewMockAPI()
	mockAPI.AddRun(run)
	_, err = mockAPI.CreateTask(context.Background(), &apiv2beta1.CreateTaskRequest{RunId: run.GetRunId(), Task: parentTask})
	require.NoError(t, err)
	_, err = mockAPI.CreateTask(context.Background(), &apiv2beta1.CreateTaskRequest{RunId: run.GetRunId(), Task: successChild})
	require.NoError(t, err)
	_, err = mockAPI.CreateTask(context.Background(), &apiv2beta1.CreateTaskRequest{RunId: run.GetRunId(), Task: failedChild})
	require.NoError(t, err)

	parentScope, err := util.NewScopePathFromStruct(pipelineSpecStruct)
	require.NoError(t, err)
	clientManager := client_manager.NewFakeClientManager(fake.NewClientset(), mockAPI)

	err = RepublishPreservedChildOutputsToDAG(context.Background(), DAGOutputRepublishOptions{
		Run:          run,
		ParentTask:   parentTask,
		ParentScope:  parentScope,
		PipelineSpec: pipelineSpecStruct,
	}, clientManager)
	require.NoError(t, err)

	updatedParent, err := mockAPI.GetTask(context.Background(), &apiv2beta1.GetTaskRequest{
		TaskId: parentTaskID,
		RunId:  run.GetRunId(),
	})
	require.NoError(t, err)
	require.Len(t, updatedParent.GetOutputs().GetParameters(), 1)
	outputParam := updatedParent.GetOutputs().GetParameters()[0]
	assert.Equal(t, "pipeline-output", outputParam.GetParameterKey())
	assert.Equal(t, "kept", outputParam.GetValue().GetStringValue())
	require.NotNil(t, outputParam.GetProducer())
	assert.Equal(t, "success-child", outputParam.GetProducer().GetTaskName())
}

func TestRepublishPreservedChildOutputsToDAG_PagesThroughChildren(t *testing.T) {
	prevPageSize := republishChildTasksPageSize
	republishChildTasksPageSize = 1
	t.Cleanup(func() { republishChildTasksPageSize = prevPageSize })

	pipelineSpec := &pipelinespec.PipelineSpec{
		Root: &pipelinespec.ComponentSpec{
			Implementation: &pipelinespec.ComponentSpec_Dag{
				Dag: &pipelinespec.DagSpec{
					Tasks: map[string]*pipelinespec.PipelineTaskSpec{
						"child-a": {
							TaskInfo:     &pipelinespec.PipelineTaskInfo{Name: "child-a"},
							ComponentRef: &pipelinespec.ComponentRef{Name: "child-comp"},
						},
						"child-b": {
							TaskInfo:     &pipelinespec.PipelineTaskInfo{Name: "child-b"},
							ComponentRef: &pipelinespec.ComponentRef{Name: "child-comp"},
						},
					},
					Outputs: &pipelinespec.DagOutputsSpec{
						Parameters: map[string]*pipelinespec.DagOutputsSpec_DagOutputParameterSpec{
							"from-b": {
								Kind: &pipelinespec.DagOutputsSpec_DagOutputParameterSpec_ValueFromParameter{
									ValueFromParameter: &pipelinespec.DagOutputsSpec_ParameterSelectorSpec{
										ProducerSubtask:    "child-b",
										OutputParameterKey: "result",
									},
								},
							},
						},
					},
				},
			},
			OutputDefinitions: &pipelinespec.ComponentOutputsSpec{
				Parameters: map[string]*pipelinespec.ComponentOutputsSpec_ParameterSpec{
					"from-b": {ParameterType: pipelinespec.ParameterType_STRING},
				},
			},
		},
		Components: map[string]*pipelinespec.ComponentSpec{
			"child-comp": {
				Implementation: &pipelinespec.ComponentSpec_ExecutorLabel{ExecutorLabel: "child"},
				OutputDefinitions: &pipelinespec.ComponentOutputsSpec{
					Parameters: map[string]*pipelinespec.ComponentOutputsSpec_ParameterSpec{
						"result": {ParameterType: pipelinespec.ParameterType_STRING},
					},
				},
			},
		},
	}
	pipelineSpecStruct, err := pipelineSpecToStruct(t, pipelineSpec)
	require.NoError(t, err)

	run := &apiv2beta1.Run{RunId: "run-paginate"}
	parentID := "parent"
	parentIDPtr := util.StringPointer(parentID)
	parentTask := &apiv2beta1.PipelineTask{
		TaskId:    parentID,
		RunId:     run.GetRunId(),
		Name:      "root",
		State:     apiv2beta1.PipelineTask_RUNNING,
		Type:      apiv2beta1.PipelineTask_DAG,
		ScopePath: "root",
		Outputs:   &apiv2beta1.PipelineTask_InputOutputs{},
	}
	childA := &apiv2beta1.PipelineTask{
		TaskId:       "task-a",
		RunId:        run.GetRunId(),
		Name:         "child-a",
		State:        apiv2beta1.PipelineTask_SUCCEEDED,
		Type:         apiv2beta1.PipelineTask_RUNTIME,
		ScopePath:    "root.child-a",
		ParentTaskId: parentIDPtr,
		Outputs: &apiv2beta1.PipelineTask_InputOutputs{
			Parameters: []*apiv2beta1.PipelineTask_InputOutputs_IOParameter{{
				ParameterKey: "result",
				Value:        structpb.NewStringValue("a"),
				Type:         apiv2beta1.IOType_OUTPUT,
				Producer:     &apiv2beta1.IOProducer{TaskName: "child-a"},
			}},
		},
	}
	childB := &apiv2beta1.PipelineTask{
		TaskId:       "task-b",
		RunId:        run.GetRunId(),
		Name:         "child-b",
		State:        apiv2beta1.PipelineTask_SUCCEEDED,
		Type:         apiv2beta1.PipelineTask_RUNTIME,
		ScopePath:    "root.child-b",
		ParentTaskId: parentIDPtr,
		Outputs: &apiv2beta1.PipelineTask_InputOutputs{
			Parameters: []*apiv2beta1.PipelineTask_InputOutputs_IOParameter{{
				ParameterKey: "result",
				Value:        structpb.NewStringValue("b"),
				Type:         apiv2beta1.IOType_OUTPUT,
				Producer:     &apiv2beta1.IOProducer{TaskName: "child-b"},
			}},
		},
	}

	mockAPI := kfpapi.NewMockAPI()
	mockAPI.AddRun(run)
	_, err = mockAPI.CreateTask(context.Background(), &apiv2beta1.CreateTaskRequest{RunId: run.GetRunId(), Task: parentTask})
	require.NoError(t, err)
	_, err = mockAPI.CreateTask(context.Background(), &apiv2beta1.CreateTaskRequest{RunId: run.GetRunId(), Task: childA})
	require.NoError(t, err)
	_, err = mockAPI.CreateTask(context.Background(), &apiv2beta1.CreateTaskRequest{RunId: run.GetRunId(), Task: childB})
	require.NoError(t, err)

	parentScope, err := util.NewScopePathFromStruct(pipelineSpecStruct)
	require.NoError(t, err)
	clientManager := client_manager.NewFakeClientManager(fake.NewClientset(), mockAPI)

	err = RepublishPreservedChildOutputsToDAG(context.Background(), DAGOutputRepublishOptions{
		Run:          run,
		ParentTask:   parentTask,
		ParentScope:  parentScope,
		PipelineSpec: pipelineSpecStruct,
	}, clientManager)
	require.NoError(t, err)

	updatedParent, err := mockAPI.GetTask(context.Background(), &apiv2beta1.GetTaskRequest{
		TaskId: parentID,
		RunId:  run.GetRunId(),
	})
	require.NoError(t, err)
	require.Len(t, updatedParent.GetOutputs().GetParameters(), 1)
	assert.Equal(t, "from-b", updatedParent.GetOutputs().GetParameters()[0].GetParameterKey())
	assert.Equal(t, "b", updatedParent.GetOutputs().GetParameters()[0].GetValue().GetStringValue())
}

func TestRepublishPreservedChildOutputsToDAG_RepairsParamsWhenArtifactsAlreadyPresent(t *testing.T) {
	pipelineSpec := &pipelinespec.PipelineSpec{
		Root: &pipelinespec.ComponentSpec{
			Implementation: &pipelinespec.ComponentSpec_Dag{
				Dag: &pipelinespec.DagSpec{
					Tasks: map[string]*pipelinespec.PipelineTaskSpec{
						"worker": {
							TaskInfo:     &pipelinespec.PipelineTaskInfo{Name: "worker"},
							ComponentRef: &pipelinespec.ComponentRef{Name: "worker-comp"},
						},
					},
					Outputs: &pipelinespec.DagOutputsSpec{
						Parameters: map[string]*pipelinespec.DagOutputsSpec_DagOutputParameterSpec{
							"pipeline-output": {
								Kind: &pipelinespec.DagOutputsSpec_DagOutputParameterSpec_ValueFromParameter{
									ValueFromParameter: &pipelinespec.DagOutputsSpec_ParameterSelectorSpec{
										ProducerSubtask:    "worker",
										OutputParameterKey: "result",
									},
								},
							},
						},
						Artifacts: map[string]*pipelinespec.DagOutputsSpec_DagOutputArtifactSpec{
							"model": {
								ArtifactSelectors: []*pipelinespec.DagOutputsSpec_ArtifactSelectorSpec{{
									ProducerSubtask:   "worker",
									OutputArtifactKey: "model",
								}},
							},
						},
					},
				},
			},
			OutputDefinitions: &pipelinespec.ComponentOutputsSpec{
				Parameters: map[string]*pipelinespec.ComponentOutputsSpec_ParameterSpec{
					"pipeline-output": {ParameterType: pipelinespec.ParameterType_STRING},
				},
				Artifacts: map[string]*pipelinespec.ComponentOutputsSpec_ArtifactSpec{
					"model": {},
				},
			},
		},
		Components: map[string]*pipelinespec.ComponentSpec{
			"worker-comp": {
				Implementation: &pipelinespec.ComponentSpec_ExecutorLabel{ExecutorLabel: "worker"},
				OutputDefinitions: &pipelinespec.ComponentOutputsSpec{
					Parameters: map[string]*pipelinespec.ComponentOutputsSpec_ParameterSpec{
						"result": {ParameterType: pipelinespec.ParameterType_STRING},
					},
					Artifacts: map[string]*pipelinespec.ComponentOutputsSpec_ArtifactSpec{
						"model": {},
					},
				},
			},
		},
	}
	pipelineSpecStruct, err := pipelineSpecToStruct(t, pipelineSpec)
	require.NoError(t, err)

	run := &apiv2beta1.Run{RunId: "run-partial"}
	parentID := "parent"
	parentIDPtr := util.StringPointer(parentID)
	artifactID := "art-model"

	parentTask := &apiv2beta1.PipelineTask{
		TaskId:    parentID,
		RunId:     run.GetRunId(),
		Name:      "root",
		State:     apiv2beta1.PipelineTask_RUNNING,
		Type:      apiv2beta1.PipelineTask_DAG,
		ScopePath: "root",
		// Partial prior republish: artifact link durable, parameter update missing.
		Outputs: &apiv2beta1.PipelineTask_InputOutputs{},
	}
	childTask := &apiv2beta1.PipelineTask{
		TaskId:       "worker-task",
		RunId:        run.GetRunId(),
		Name:         "worker",
		State:        apiv2beta1.PipelineTask_SUCCEEDED,
		Type:         apiv2beta1.PipelineTask_RUNTIME,
		ScopePath:    "root.worker",
		ParentTaskId: parentIDPtr,
		Outputs: &apiv2beta1.PipelineTask_InputOutputs{
			Parameters: []*apiv2beta1.PipelineTask_InputOutputs_IOParameter{{
				ParameterKey: "result",
				Value:        structpb.NewStringValue("repaired"),
				Type:         apiv2beta1.IOType_OUTPUT,
				Producer:     &apiv2beta1.IOProducer{TaskName: "worker"},
			}},
			Artifacts: []*apiv2beta1.PipelineTask_InputOutputs_IOArtifact{{
				ArtifactKey: "model",
				Artifacts:   []*apiv2beta1.Artifact{{ArtifactId: artifactID, Name: "model"}},
				Type:        apiv2beta1.IOType_OUTPUT,
				Producer:    &apiv2beta1.IOProducer{TaskName: "worker"},
			}},
		},
	}

	baseMock := kfpapi.NewMockAPI()
	mockAPI := &uniqueLinkEnforcingMockAPI{MockAPI: baseMock, seen: map[string]struct{}{}}
	mockAPI.AddRun(run)
	_, err = mockAPI.CreateTask(context.Background(), &apiv2beta1.CreateTaskRequest{RunId: run.GetRunId(), Task: parentTask})
	require.NoError(t, err)
	_, err = mockAPI.CreateTask(context.Background(), &apiv2beta1.CreateTaskRequest{RunId: run.GetRunId(), Task: childTask})
	require.NoError(t, err)
	_, err = mockAPI.CreateArtifact(context.Background(), &apiv2beta1.CreateArtifactRequest{
		Artifact: &apiv2beta1.Artifact{ArtifactId: artifactID, Name: "model", Uri: util.StringPointer("gs://bucket/model")},
		TaskId:   childTask.GetTaskId(),
		RunId:    run.GetRunId(),
	})
	require.NoError(t, err)
	_, err = mockAPI.CreateArtifactTasks(context.Background(), &apiv2beta1.CreateArtifactTasksBulkRequest{
		ArtifactTasks: []*apiv2beta1.ArtifactTask{{
			ArtifactId: artifactID,
			TaskId:     parentID,
			RunId:      run.GetRunId(),
			Key:        "model",
			Type:       apiv2beta1.IOType_OUTPUT,
			Producer:   &apiv2beta1.IOProducer{TaskName: "worker"},
		}},
	})
	require.NoError(t, err)

	parentScope, err := util.NewScopePathFromStruct(pipelineSpecStruct)
	require.NoError(t, err)
	clientManager := client_manager.NewFakeClientManager(fake.NewClientset(), mockAPI)

	err = RepublishPreservedChildOutputsToDAG(context.Background(), DAGOutputRepublishOptions{
		Run:          run,
		ParentTask:   parentTask,
		ParentScope:  parentScope,
		PipelineSpec: pipelineSpecStruct,
	}, clientManager)
	require.NoError(t, err)

	updatedParent, err := mockAPI.GetTask(context.Background(), &apiv2beta1.GetTaskRequest{
		TaskId: parentID,
		RunId:  run.GetRunId(),
	})
	require.NoError(t, err)
	require.Len(t, updatedParent.GetOutputs().GetParameters(), 1)
	assert.Equal(t, "repaired", updatedParent.GetOutputs().GetParameters()[0].GetValue().GetStringValue())
	require.Len(t, updatedParent.GetOutputs().GetArtifacts(), 1)
	assert.Equal(t, "model", updatedParent.GetOutputs().GetArtifacts()[0].GetArtifactKey())
}

func TestRepublishPreservedChildOutputsToDAG_OmitsAncestorArtifactLinks(t *testing.T) {
	pipelineSpec := &pipelinespec.PipelineSpec{
		Root: &pipelinespec.ComponentSpec{
			Implementation: &pipelinespec.ComponentSpec_Dag{
				Dag: &pipelinespec.DagSpec{
					Tasks: map[string]*pipelinespec.PipelineTaskSpec{
						"nested": {
							TaskInfo:     &pipelinespec.PipelineTaskInfo{Name: "nested"},
							ComponentRef: &pipelinespec.ComponentRef{Name: "nested-comp"},
						},
					},
					Outputs: &pipelinespec.DagOutputsSpec{
						Parameters: map[string]*pipelinespec.DagOutputsSpec_DagOutputParameterSpec{
							"root-out": {
								Kind: &pipelinespec.DagOutputsSpec_DagOutputParameterSpec_ValueFromParameter{
									ValueFromParameter: &pipelinespec.DagOutputsSpec_ParameterSelectorSpec{
										ProducerSubtask:    "nested",
										OutputParameterKey: "nested-out",
									},
								},
							},
						},
						Artifacts: map[string]*pipelinespec.DagOutputsSpec_DagOutputArtifactSpec{
							"root-model": {
								ArtifactSelectors: []*pipelinespec.DagOutputsSpec_ArtifactSelectorSpec{{
									ProducerSubtask:   "nested",
									OutputArtifactKey: "nested-model",
								}},
							},
						},
					},
				},
			},
			OutputDefinitions: &pipelinespec.ComponentOutputsSpec{
				Parameters: map[string]*pipelinespec.ComponentOutputsSpec_ParameterSpec{
					"root-out": {ParameterType: pipelinespec.ParameterType_STRING},
				},
				Artifacts: map[string]*pipelinespec.ComponentOutputsSpec_ArtifactSpec{
					"root-model": {},
				},
			},
		},
		Components: map[string]*pipelinespec.ComponentSpec{
			"nested-comp": {
				Implementation: &pipelinespec.ComponentSpec_Dag{
					Dag: &pipelinespec.DagSpec{
						Tasks: map[string]*pipelinespec.PipelineTaskSpec{
							"worker": {
								TaskInfo:     &pipelinespec.PipelineTaskInfo{Name: "worker"},
								ComponentRef: &pipelinespec.ComponentRef{Name: "worker-comp"},
							},
						},
						Outputs: &pipelinespec.DagOutputsSpec{
							Parameters: map[string]*pipelinespec.DagOutputsSpec_DagOutputParameterSpec{
								"nested-out": {
									Kind: &pipelinespec.DagOutputsSpec_DagOutputParameterSpec_ValueFromParameter{
										ValueFromParameter: &pipelinespec.DagOutputsSpec_ParameterSelectorSpec{
											ProducerSubtask:    "worker",
											OutputParameterKey: "result",
										},
									},
								},
							},
							Artifacts: map[string]*pipelinespec.DagOutputsSpec_DagOutputArtifactSpec{
								"nested-model": {
									ArtifactSelectors: []*pipelinespec.DagOutputsSpec_ArtifactSelectorSpec{{
										ProducerSubtask:   "worker",
										OutputArtifactKey: "model",
									}},
								},
							},
						},
					},
				},
				OutputDefinitions: &pipelinespec.ComponentOutputsSpec{
					Parameters: map[string]*pipelinespec.ComponentOutputsSpec_ParameterSpec{
						"nested-out": {ParameterType: pipelinespec.ParameterType_STRING},
					},
					Artifacts: map[string]*pipelinespec.ComponentOutputsSpec_ArtifactSpec{
						"nested-model": {},
					},
				},
			},
			"worker-comp": {
				Implementation: &pipelinespec.ComponentSpec_ExecutorLabel{ExecutorLabel: "worker"},
				OutputDefinitions: &pipelinespec.ComponentOutputsSpec{
					Parameters: map[string]*pipelinespec.ComponentOutputsSpec_ParameterSpec{
						"result": {ParameterType: pipelinespec.ParameterType_STRING},
					},
					Artifacts: map[string]*pipelinespec.ComponentOutputsSpec_ArtifactSpec{
						"model": {},
					},
				},
			},
		},
	}
	pipelineSpecStruct, err := pipelineSpecToStruct(t, pipelineSpec)
	require.NoError(t, err)

	run := &apiv2beta1.Run{RunId: "run-nested-partial"}
	rootID := "root-task"
	nestedID := "nested-task"
	workerID := "worker-task"
	artifactID := "art-nested"
	rootIDPtr := util.StringPointer(rootID)
	nestedIDPtr := util.StringPointer(nestedID)

	rootTask := &apiv2beta1.PipelineTask{
		TaskId:    rootID,
		RunId:     run.GetRunId(),
		Name:      "root",
		State:     apiv2beta1.PipelineTask_RUNNING,
		Type:      apiv2beta1.PipelineTask_DAG,
		ScopePath: "root",
		Outputs:   &apiv2beta1.PipelineTask_InputOutputs{},
	}
	nestedTask := &apiv2beta1.PipelineTask{
		TaskId:       nestedID,
		RunId:        run.GetRunId(),
		Name:         "nested",
		State:        apiv2beta1.PipelineTask_RUNNING,
		Type:         apiv2beta1.PipelineTask_DAG,
		ScopePath:    "root.nested",
		ParentTaskId: rootIDPtr,
		Outputs:      &apiv2beta1.PipelineTask_InputOutputs{},
	}
	workerTask := &apiv2beta1.PipelineTask{
		TaskId:       workerID,
		RunId:        run.GetRunId(),
		Name:         "worker",
		State:        apiv2beta1.PipelineTask_SUCCEEDED,
		Type:         apiv2beta1.PipelineTask_RUNTIME,
		ScopePath:    "root.nested.worker",
		ParentTaskId: nestedIDPtr,
		Outputs: &apiv2beta1.PipelineTask_InputOutputs{
			Parameters: []*apiv2beta1.PipelineTask_InputOutputs_IOParameter{{
				ParameterKey: "result",
				Value:        structpb.NewStringValue("nested-repaired"),
				Type:         apiv2beta1.IOType_OUTPUT,
				Producer:     &apiv2beta1.IOProducer{TaskName: "worker"},
			}},
			Artifacts: []*apiv2beta1.PipelineTask_InputOutputs_IOArtifact{{
				ArtifactKey: "model",
				Artifacts:   []*apiv2beta1.Artifact{{ArtifactId: artifactID, Name: "model"}},
				Type:        apiv2beta1.IOType_OUTPUT,
				Producer:    &apiv2beta1.IOProducer{TaskName: "worker"},
			}},
		},
	}

	baseMock := kfpapi.NewMockAPI()
	mockAPI := &uniqueLinkEnforcingMockAPI{MockAPI: baseMock, seen: map[string]struct{}{}}
	mockAPI.AddRun(run)
	_, err = mockAPI.CreateTask(context.Background(), &apiv2beta1.CreateTaskRequest{RunId: run.GetRunId(), Task: rootTask})
	require.NoError(t, err)
	_, err = mockAPI.CreateTask(context.Background(), &apiv2beta1.CreateTaskRequest{RunId: run.GetRunId(), Task: nestedTask})
	require.NoError(t, err)
	_, err = mockAPI.CreateTask(context.Background(), &apiv2beta1.CreateTaskRequest{RunId: run.GetRunId(), Task: workerTask})
	require.NoError(t, err)
	_, err = mockAPI.CreateArtifact(context.Background(), &apiv2beta1.CreateArtifactRequest{
		Artifact: &apiv2beta1.Artifact{ArtifactId: artifactID, Name: "model", Uri: util.StringPointer("gs://bucket/model")},
		TaskId:   workerID,
		RunId:    run.GetRunId(),
	})
	require.NoError(t, err)
	// Partial flush already wrote artifact links for nested and root.
	_, err = mockAPI.CreateArtifactTasks(context.Background(), &apiv2beta1.CreateArtifactTasksBulkRequest{
		ArtifactTasks: []*apiv2beta1.ArtifactTask{
			{
				ArtifactId: artifactID,
				TaskId:     nestedID,
				RunId:      run.GetRunId(),
				Key:        "nested-model",
				Type:       apiv2beta1.IOType_OUTPUT,
				Producer:   &apiv2beta1.IOProducer{TaskName: "worker"},
			},
			{
				ArtifactId: artifactID,
				TaskId:     rootID,
				RunId:      run.GetRunId(),
				Key:        "root-model",
				Type:       apiv2beta1.IOType_OUTPUT,
				Producer:   &apiv2beta1.IOProducer{TaskName: "nested"},
			},
		},
	})
	require.NoError(t, err)

	nestedScope, err := util.ScopePathFromDotNotation(pipelineSpecStruct, "root.nested")
	require.NoError(t, err)
	clientManager := client_manager.NewFakeClientManager(fake.NewClientset(), mockAPI)

	err = RepublishPreservedChildOutputsToDAG(context.Background(), DAGOutputRepublishOptions{
		Run:          run,
		ParentTask:   nestedTask,
		ParentScope:  nestedScope,
		PipelineSpec: pipelineSpecStruct,
	}, clientManager)
	require.NoError(t, err)

	updatedNested, err := mockAPI.GetTask(context.Background(), &apiv2beta1.GetTaskRequest{
		TaskId: nestedID,
		RunId:  run.GetRunId(),
	})
	require.NoError(t, err)
	require.Len(t, updatedNested.GetOutputs().GetParameters(), 1)
	assert.Equal(t, "nested-repaired", updatedNested.GetOutputs().GetParameters()[0].GetValue().GetStringValue())

	updatedRoot, err := mockAPI.GetTask(context.Background(), &apiv2beta1.GetTaskRequest{
		TaskId: rootID,
		RunId:  run.GetRunId(),
	})
	require.NoError(t, err)
	require.Len(t, updatedRoot.GetOutputs().GetParameters(), 1)
	assert.Equal(t, "nested-repaired", updatedRoot.GetOutputs().GetParameters()[0].GetValue().GetStringValue())
	require.Len(t, updatedRoot.GetOutputs().GetArtifacts(), 1)
	assert.Equal(t, "root-model", updatedRoot.GetOutputs().GetArtifacts()[0].GetArtifactKey())
}

// Regression: a prior partial UpdateTasksBulk can leave the immediate nested DAG
// complete while the root ancestor is still missing declared outputs. Republish
// must continue in that case so preserved children can repair the root.
func TestRepublishPreservedChildOutputsToDAG_RepairsIncompleteAncestorWhenNestedComplete(t *testing.T) {
	pipelineSpec := &pipelinespec.PipelineSpec{
		Root: &pipelinespec.ComponentSpec{
			Implementation: &pipelinespec.ComponentSpec_Dag{
				Dag: &pipelinespec.DagSpec{
					Tasks: map[string]*pipelinespec.PipelineTaskSpec{
						"nested": {
							TaskInfo:     &pipelinespec.PipelineTaskInfo{Name: "nested"},
							ComponentRef: &pipelinespec.ComponentRef{Name: "nested-comp"},
						},
					},
					Outputs: &pipelinespec.DagOutputsSpec{
						Parameters: map[string]*pipelinespec.DagOutputsSpec_DagOutputParameterSpec{
							"root-out": {
								Kind: &pipelinespec.DagOutputsSpec_DagOutputParameterSpec_ValueFromParameter{
									ValueFromParameter: &pipelinespec.DagOutputsSpec_ParameterSelectorSpec{
										ProducerSubtask:    "nested",
										OutputParameterKey: "nested-out",
									},
								},
							},
						},
						Artifacts: map[string]*pipelinespec.DagOutputsSpec_DagOutputArtifactSpec{
							"root-model": {
								ArtifactSelectors: []*pipelinespec.DagOutputsSpec_ArtifactSelectorSpec{{
									ProducerSubtask:   "nested",
									OutputArtifactKey: "nested-model",
								}},
							},
						},
					},
				},
			},
			OutputDefinitions: &pipelinespec.ComponentOutputsSpec{
				Parameters: map[string]*pipelinespec.ComponentOutputsSpec_ParameterSpec{
					"root-out": {ParameterType: pipelinespec.ParameterType_STRING},
				},
				Artifacts: map[string]*pipelinespec.ComponentOutputsSpec_ArtifactSpec{
					"root-model": {},
				},
			},
		},
		Components: map[string]*pipelinespec.ComponentSpec{
			"nested-comp": {
				Implementation: &pipelinespec.ComponentSpec_Dag{
					Dag: &pipelinespec.DagSpec{
						Tasks: map[string]*pipelinespec.PipelineTaskSpec{
							"worker": {
								TaskInfo:     &pipelinespec.PipelineTaskInfo{Name: "worker"},
								ComponentRef: &pipelinespec.ComponentRef{Name: "worker-comp"},
							},
						},
						Outputs: &pipelinespec.DagOutputsSpec{
							Parameters: map[string]*pipelinespec.DagOutputsSpec_DagOutputParameterSpec{
								"nested-out": {
									Kind: &pipelinespec.DagOutputsSpec_DagOutputParameterSpec_ValueFromParameter{
										ValueFromParameter: &pipelinespec.DagOutputsSpec_ParameterSelectorSpec{
											ProducerSubtask:    "worker",
											OutputParameterKey: "result",
										},
									},
								},
							},
							Artifacts: map[string]*pipelinespec.DagOutputsSpec_DagOutputArtifactSpec{
								"nested-model": {
									ArtifactSelectors: []*pipelinespec.DagOutputsSpec_ArtifactSelectorSpec{{
										ProducerSubtask:   "worker",
										OutputArtifactKey: "model",
									}},
								},
							},
						},
					},
				},
				OutputDefinitions: &pipelinespec.ComponentOutputsSpec{
					Parameters: map[string]*pipelinespec.ComponentOutputsSpec_ParameterSpec{
						"nested-out": {ParameterType: pipelinespec.ParameterType_STRING},
					},
					Artifacts: map[string]*pipelinespec.ComponentOutputsSpec_ArtifactSpec{
						"nested-model": {},
					},
				},
			},
			"worker-comp": {
				Implementation: &pipelinespec.ComponentSpec_ExecutorLabel{ExecutorLabel: "worker"},
				OutputDefinitions: &pipelinespec.ComponentOutputsSpec{
					Parameters: map[string]*pipelinespec.ComponentOutputsSpec_ParameterSpec{
						"result": {ParameterType: pipelinespec.ParameterType_STRING},
					},
					Artifacts: map[string]*pipelinespec.ComponentOutputsSpec_ArtifactSpec{
						"model": {},
					},
				},
			},
		},
	}
	pipelineSpecStruct, err := pipelineSpecToStruct(t, pipelineSpec)
	require.NoError(t, err)

	run := &apiv2beta1.Run{RunId: "run-nested-complete-root-incomplete"}
	rootID := "root-task"
	nestedID := "nested-task"
	workerID := "worker-task"
	artifactID := "art-ancestor-repair"
	rootIDPtr := util.StringPointer(rootID)
	nestedIDPtr := util.StringPointer(nestedID)

	rootTask := &apiv2beta1.PipelineTask{
		TaskId:    rootID,
		RunId:     run.GetRunId(),
		Name:      "root",
		State:     apiv2beta1.PipelineTask_RUNNING,
		Type:      apiv2beta1.PipelineTask_DAG,
		ScopePath: "root",
		Outputs:   &apiv2beta1.PipelineTask_InputOutputs{},
	}
	// Immediate parent already has all declared outputs from a prior partial flush.
	nestedTask := &apiv2beta1.PipelineTask{
		TaskId:       nestedID,
		RunId:        run.GetRunId(),
		Name:         "nested",
		State:        apiv2beta1.PipelineTask_RUNNING,
		Type:         apiv2beta1.PipelineTask_DAG,
		ScopePath:    "root.nested",
		ParentTaskId: rootIDPtr,
		Outputs: &apiv2beta1.PipelineTask_InputOutputs{
			Parameters: []*apiv2beta1.PipelineTask_InputOutputs_IOParameter{{
				ParameterKey: "nested-out",
				Value:        structpb.NewStringValue("nested-complete"),
				Type:         apiv2beta1.IOType_OUTPUT,
				Producer:     &apiv2beta1.IOProducer{TaskName: "worker"},
			}},
			Artifacts: []*apiv2beta1.PipelineTask_InputOutputs_IOArtifact{{
				ArtifactKey: "nested-model",
				Artifacts:   []*apiv2beta1.Artifact{{ArtifactId: artifactID, Name: "model"}},
				Type:        apiv2beta1.IOType_OUTPUT,
				Producer:    &apiv2beta1.IOProducer{TaskName: "worker"},
			}},
		},
	}
	workerTask := &apiv2beta1.PipelineTask{
		TaskId:       workerID,
		RunId:        run.GetRunId(),
		Name:         "worker",
		State:        apiv2beta1.PipelineTask_SUCCEEDED,
		Type:         apiv2beta1.PipelineTask_RUNTIME,
		ScopePath:    "root.nested.worker",
		ParentTaskId: nestedIDPtr,
		Outputs: &apiv2beta1.PipelineTask_InputOutputs{
			Parameters: []*apiv2beta1.PipelineTask_InputOutputs_IOParameter{{
				ParameterKey: "result",
				Value:        structpb.NewStringValue("nested-complete"),
				Type:         apiv2beta1.IOType_OUTPUT,
				Producer:     &apiv2beta1.IOProducer{TaskName: "worker"},
			}},
			Artifacts: []*apiv2beta1.PipelineTask_InputOutputs_IOArtifact{{
				ArtifactKey: "model",
				Artifacts:   []*apiv2beta1.Artifact{{ArtifactId: artifactID, Name: "model"}},
				Type:        apiv2beta1.IOType_OUTPUT,
				Producer:    &apiv2beta1.IOProducer{TaskName: "worker"},
			}},
		},
	}

	baseMock := kfpapi.NewMockAPI()
	mockAPI := &uniqueLinkEnforcingMockAPI{MockAPI: baseMock, seen: map[string]struct{}{}}
	mockAPI.AddRun(run)
	_, err = mockAPI.CreateTask(context.Background(), &apiv2beta1.CreateTaskRequest{RunId: run.GetRunId(), Task: rootTask})
	require.NoError(t, err)
	_, err = mockAPI.CreateTask(context.Background(), &apiv2beta1.CreateTaskRequest{RunId: run.GetRunId(), Task: nestedTask})
	require.NoError(t, err)
	_, err = mockAPI.CreateTask(context.Background(), &apiv2beta1.CreateTaskRequest{RunId: run.GetRunId(), Task: workerTask})
	require.NoError(t, err)
	_, err = mockAPI.CreateArtifact(context.Background(), &apiv2beta1.CreateArtifactRequest{
		Artifact:    &apiv2beta1.Artifact{ArtifactId: artifactID, Name: "model", Uri: util.StringPointer("gs://bucket/model")},
		TaskId:      workerID,
		RunId:       run.GetRunId(),
		ProducerKey: "model",
	})
	require.NoError(t, err)
	_, err = mockAPI.CreateArtifactTasks(context.Background(), &apiv2beta1.CreateArtifactTasksBulkRequest{
		ArtifactTasks: []*apiv2beta1.ArtifactTask{{
			ArtifactId: artifactID,
			TaskId:     nestedID,
			RunId:      run.GetRunId(),
			Key:        "nested-model",
			Type:       apiv2beta1.IOType_OUTPUT,
			Producer:   &apiv2beta1.IOProducer{TaskName: "worker"},
		}},
	})
	require.NoError(t, err)

	nestedScope, err := util.ScopePathFromDotNotation(pipelineSpecStruct, "root.nested")
	require.NoError(t, err)
	clientManager := client_manager.NewFakeClientManager(fake.NewClientset(), mockAPI)

	err = RepublishPreservedChildOutputsToDAG(context.Background(), DAGOutputRepublishOptions{
		Run:          run,
		ParentTask:   nestedTask,
		ParentScope:  nestedScope,
		PipelineSpec: pipelineSpecStruct,
	}, clientManager)
	require.NoError(t, err)

	updatedRoot, err := mockAPI.GetTask(context.Background(), &apiv2beta1.GetTaskRequest{
		TaskId: rootID,
		RunId:  run.GetRunId(),
	})
	require.NoError(t, err)
	require.Len(t, updatedRoot.GetOutputs().GetParameters(), 1)
	assert.Equal(t, "nested-complete", updatedRoot.GetOutputs().GetParameters()[0].GetValue().GetStringValue())
	require.Len(t, updatedRoot.GetOutputs().GetArtifacts(), 1)
	assert.Equal(t, "root-model", updatedRoot.GetOutputs().GetArtifacts()[0].GetArtifactKey())
}

// uniqueLinkEnforcingMockAPI rejects duplicate artifact-task rows that would
// violate UniqueLink in production storage.
type uniqueLinkEnforcingMockAPI struct {
	*kfpapi.MockAPI
	seen map[string]struct{}
}

func artifactTaskUniqueLinkKey(artifactTask *apiv2beta1.ArtifactTask) string {
	iteration := int64(-1)
	if artifactTask.GetProducer() != nil && artifactTask.GetProducer().Iteration != nil {
		iteration = artifactTask.GetProducer().GetIteration()
	}
	return artifactTask.GetArtifactId() + "|" +
		artifactTask.GetTaskId() + "|" +
		artifactTask.GetType().String() + "|" +
		strconv.FormatInt(iteration, 10) + "|" +
		artifactTask.GetKey()
}

func (m *uniqueLinkEnforcingMockAPI) CreateArtifactTasks(
	ctx context.Context,
	req *apiv2beta1.CreateArtifactTasksBulkRequest,
) (*apiv2beta1.CreateArtifactTasksBulkResponse, error) {
	for _, artifactTask := range req.GetArtifactTasks() {
		key := artifactTaskUniqueLinkKey(artifactTask)
		if _, exists := m.seen[key]; exists {
			return nil, fmt.Errorf("UniqueLink violation for artifact-task %s", key)
		}
		m.seen[key] = struct{}{}
	}
	return m.MockAPI.CreateArtifactTasks(ctx, req)
}

func TestRepublishPreservedChildOutputsToDAG_NoOpWhenParentAlreadyHasOutputs(t *testing.T) {
	pipelineSpec := &pipelinespec.PipelineSpec{
		Root: &pipelinespec.ComponentSpec{
			Implementation: &pipelinespec.ComponentSpec_Dag{
				Dag: &pipelinespec.DagSpec{
					Tasks: map[string]*pipelinespec.PipelineTaskSpec{
						"worker": {
							TaskInfo:     &pipelinespec.PipelineTaskInfo{Name: "worker"},
							ComponentRef: &pipelinespec.ComponentRef{Name: "worker-comp"},
						},
					},
					Outputs: &pipelinespec.DagOutputsSpec{
						Parameters: map[string]*pipelinespec.DagOutputsSpec_DagOutputParameterSpec{
							"pipeline-output": {
								Kind: &pipelinespec.DagOutputsSpec_DagOutputParameterSpec_ValueFromParameter{
									ValueFromParameter: &pipelinespec.DagOutputsSpec_ParameterSelectorSpec{
										ProducerSubtask:    "worker",
										OutputParameterKey: "result",
									},
								},
							},
						},
					},
				},
			},
			OutputDefinitions: &pipelinespec.ComponentOutputsSpec{
				Parameters: map[string]*pipelinespec.ComponentOutputsSpec_ParameterSpec{
					"pipeline-output": {ParameterType: pipelinespec.ParameterType_STRING},
				},
			},
		},
		Components: map[string]*pipelinespec.ComponentSpec{
			"worker-comp": {
				Implementation: &pipelinespec.ComponentSpec_ExecutorLabel{ExecutorLabel: "worker"},
				OutputDefinitions: &pipelinespec.ComponentOutputsSpec{
					Parameters: map[string]*pipelinespec.ComponentOutputsSpec_ParameterSpec{
						"result": {ParameterType: pipelinespec.ParameterType_STRING},
					},
				},
			},
		},
	}
	pipelineSpecStruct, err := pipelineSpecToStruct(t, pipelineSpec)
	require.NoError(t, err)

	run := &apiv2beta1.Run{RunId: "run-noop"}
	parentID := "parent"
	parentIDPtr := util.StringPointer(parentID)
	parentTask := &apiv2beta1.PipelineTask{
		TaskId:    parentID,
		RunId:     run.GetRunId(),
		Name:      "root",
		State:     apiv2beta1.PipelineTask_RUNNING,
		Type:      apiv2beta1.PipelineTask_DAG,
		ScopePath: "root",
		Outputs: &apiv2beta1.PipelineTask_InputOutputs{
			Parameters: []*apiv2beta1.PipelineTask_InputOutputs_IOParameter{{
				ParameterKey: "pipeline-output",
				Value:        structpb.NewStringValue("already-there"),
				Type:         apiv2beta1.IOType_OUTPUT,
				Producer:     &apiv2beta1.IOProducer{TaskName: "worker"},
			}},
		},
	}
	childTask := &apiv2beta1.PipelineTask{
		TaskId:       "worker-task",
		RunId:        run.GetRunId(),
		Name:         "worker",
		State:        apiv2beta1.PipelineTask_SUCCEEDED,
		Type:         apiv2beta1.PipelineTask_RUNTIME,
		ScopePath:    "root.worker",
		ParentTaskId: parentIDPtr,
		Outputs: &apiv2beta1.PipelineTask_InputOutputs{
			Parameters: []*apiv2beta1.PipelineTask_InputOutputs_IOParameter{{
				ParameterKey: "result",
				Value:        structpb.NewStringValue("new-value"),
				Type:         apiv2beta1.IOType_OUTPUT,
				Producer:     &apiv2beta1.IOProducer{TaskName: "worker"},
			}},
		},
	}

	mockAPI := kfpapi.NewMockAPI()
	mockAPI.AddRun(run)
	_, err = mockAPI.CreateTask(context.Background(), &apiv2beta1.CreateTaskRequest{RunId: run.GetRunId(), Task: parentTask})
	require.NoError(t, err)
	_, err = mockAPI.CreateTask(context.Background(), &apiv2beta1.CreateTaskRequest{RunId: run.GetRunId(), Task: childTask})
	require.NoError(t, err)

	parentScope, err := util.NewScopePathFromStruct(pipelineSpecStruct)
	require.NoError(t, err)
	clientManager := client_manager.NewFakeClientManager(fake.NewClientset(), mockAPI)

	err = RepublishPreservedChildOutputsToDAG(context.Background(), DAGOutputRepublishOptions{
		Run:          run,
		ParentTask:   parentTask,
		ParentScope:  parentScope,
		PipelineSpec: pipelineSpecStruct,
	}, clientManager)
	require.NoError(t, err)

	updatedParent, err := mockAPI.GetTask(context.Background(), &apiv2beta1.GetTaskRequest{
		TaskId: parentID,
		RunId:  run.GetRunId(),
	})
	require.NoError(t, err)
	require.Len(t, updatedParent.GetOutputs().GetParameters(), 1)
	assert.Equal(t, "already-there", updatedParent.GetOutputs().GetParameters()[0].GetValue().GetStringValue())
}

func TestRepublishPreservedChildOutputsToDAG_NoOpWhenNoChildren(t *testing.T) {
	pipelineSpec := &pipelinespec.PipelineSpec{
		Root: &pipelinespec.ComponentSpec{
			Implementation: &pipelinespec.ComponentSpec_Dag{
				Dag: &pipelinespec.DagSpec{
					Tasks: map[string]*pipelinespec.PipelineTaskSpec{},
				},
			},
			OutputDefinitions: &pipelinespec.ComponentOutputsSpec{
				Parameters: map[string]*pipelinespec.ComponentOutputsSpec_ParameterSpec{
					"pipeline-output": {ParameterType: pipelinespec.ParameterType_STRING},
				},
			},
		},
	}
	pipelineSpecStruct, err := pipelineSpecToStruct(t, pipelineSpec)
	require.NoError(t, err)

	run := &apiv2beta1.Run{RunId: "run-empty"}
	parentTask := &apiv2beta1.PipelineTask{
		TaskId:    "parent",
		RunId:     run.GetRunId(),
		Name:      "root",
		State:     apiv2beta1.PipelineTask_RUNNING,
		Type:      apiv2beta1.PipelineTask_DAG,
		ScopePath: "root",
	}

	mockAPI := kfpapi.NewMockAPI()
	mockAPI.AddRun(run)
	_, err = mockAPI.CreateTask(context.Background(), &apiv2beta1.CreateTaskRequest{RunId: run.GetRunId(), Task: parentTask})
	require.NoError(t, err)

	parentScope, err := util.NewScopePathFromStruct(pipelineSpecStruct)
	require.NoError(t, err)
	clientManager := client_manager.NewFakeClientManager(fake.NewClientset(), mockAPI)

	err = RepublishPreservedChildOutputsToDAG(context.Background(), DAGOutputRepublishOptions{
		Run:          run,
		ParentTask:   parentTask,
		ParentScope:  parentScope,
		PipelineSpec: pipelineSpecStruct,
	}, clientManager)
	require.NoError(t, err)
}
