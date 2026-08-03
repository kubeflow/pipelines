// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kfpapi

import (
	"context"
	"testing"

	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/stretchr/testify/require"
)

func TestMockAPI_CreateTask_DifferentParentsCreateDistinctTasks(t *testing.T) {
	api := NewMockAPI()
	parentA := "parent-a"
	parentB := "parent-b"

	first, err := api.CreateTask(context.Background(), &apiv2beta1.CreateTaskRequest{
		RunId: "run-1",
		Task: &apiv2beta1.PipelineTask{
			RunId:        "run-1",
			Name:         "child",
			ScopePath:    "root.child",
			Type:         apiv2beta1.PipelineTask_RUNTIME,
			ParentTaskId: &parentA,
		},
	})
	require.NoError(t, err)

	second, err := api.CreateTask(context.Background(), &apiv2beta1.CreateTaskRequest{
		RunId: "run-1",
		Task: &apiv2beta1.PipelineTask{
			RunId:        "run-1",
			Name:         "child",
			ScopePath:    "root.child",
			Type:         apiv2beta1.PipelineTask_RUNTIME,
			ParentTaskId: &parentB,
		},
	})
	require.NoError(t, err)

	require.NotEqual(t, first.GetTaskId(), second.GetTaskId())
	require.Equal(t, parentA, first.GetParentTaskId())
	require.Equal(t, parentB, second.GetParentTaskId())
}

func TestSameLogicalTaskIdentity_NormalizesEmptyParent(t *testing.T) {
	emptyParent := ""
	existing := &apiv2beta1.PipelineTask{
		RunId:     "run-1",
		Name:      "task",
		ScopePath: "root.task",
		Type:      apiv2beta1.PipelineTask_RUNTIME,
	}
	candidateWithEmpty := &apiv2beta1.PipelineTask{
		RunId:        "run-1",
		Name:         "task",
		ScopePath:    "root.task",
		Type:         apiv2beta1.PipelineTask_RUNTIME,
		ParentTaskId: &emptyParent,
	}
	require.True(t, sameLogicalTaskIdentity(existing, candidateWithEmpty, "run-1"))

	parent := "parent-1"
	candidateWithParent := &apiv2beta1.PipelineTask{
		RunId:        "run-1",
		Name:         "task",
		ScopePath:    "root.task",
		Type:         apiv2beta1.PipelineTask_RUNTIME,
		ParentTaskId: &parent,
	}
	require.False(t, sameLogicalTaskIdentity(existing, candidateWithParent, "run-1"))
}

func TestMockAPI_CreateArtifact_IteratorCreateHydrateParity(t *testing.T) {
	api := NewMockAPI()
	run := &apiv2beta1.Run{RunId: "run-iter"}
	api.AddRun(run)

	task, err := api.CreateTask(context.Background(), &apiv2beta1.CreateTaskRequest{
		RunId: run.GetRunId(),
		Task: &apiv2beta1.PipelineTask{
			TaskId:    "loop-body",
			RunId:     run.GetRunId(),
			Name:      "loop-body",
			Type:      apiv2beta1.PipelineTask_RUNTIME,
			ScopePath: "root.loop-body",
		},
	})
	require.NoError(t, err)

	iter0 := int64(0)
	iter1 := int64(1)
	_, err = api.CreateArtifact(context.Background(), &apiv2beta1.CreateArtifactRequest{
		Artifact:       &apiv2beta1.Artifact{ArtifactId: "art-0a", Name: "out-0a", Uri: strPtr("gs://b/0a")},
		TaskId:         task.GetTaskId(),
		RunId:          run.GetRunId(),
		ProducerKey:    "models",
		IterationIndex: &iter0,
	})
	require.NoError(t, err)
	_, err = api.CreateArtifact(context.Background(), &apiv2beta1.CreateArtifactRequest{
		Artifact:       &apiv2beta1.Artifact{ArtifactId: "art-0b", Name: "out-0b", Uri: strPtr("gs://b/0b")},
		TaskId:         task.GetTaskId(),
		RunId:          run.GetRunId(),
		ProducerKey:    "models",
		IterationIndex: &iter0,
	})
	require.NoError(t, err)
	_, err = api.CreateArtifactsBulk(context.Background(), &apiv2beta1.CreateArtifactsBulkRequest{
		Artifacts: []*apiv2beta1.CreateArtifactRequest{
			{
				Artifact:       &apiv2beta1.Artifact{ArtifactId: "art-1a", Name: "out-1a", Uri: strPtr("gs://b/1a")},
				TaskId:         task.GetTaskId(),
				RunId:          run.GetRunId(),
				ProducerKey:    "models",
				IterationIndex: &iter1,
			},
		},
	})
	require.NoError(t, err)

	// Reuse path should also preserve ITERATOR_OUTPUT.
	_, err = api.CreateArtifact(context.Background(), &apiv2beta1.CreateArtifactRequest{
		Artifact:       &apiv2beta1.Artifact{Name: "out-0a", Uri: strPtr("gs://b/0a")},
		TaskId:         task.GetTaskId(),
		RunId:          run.GetRunId(),
		ProducerKey:    "models",
		IterationIndex: &iter0,
		ReuseIfExists:  true,
	})
	require.NoError(t, err)

	hydrated, err := api.GetTask(context.Background(), &apiv2beta1.GetTaskRequest{TaskId: task.GetTaskId(), RunId: run.GetRunId()})
	require.NoError(t, err)
	require.NotNil(t, hydrated.GetOutputs())
	require.Len(t, hydrated.GetOutputs().GetArtifacts(), 2)

	byIteration := map[int64]*apiv2beta1.PipelineTask_InputOutputs_IOArtifact{}
	for _, artifactIO := range hydrated.GetOutputs().GetArtifacts() {
		require.Equal(t, apiv2beta1.IOType_ITERATOR_OUTPUT, artifactIO.GetType())
		require.Equal(t, "models", artifactIO.GetArtifactKey())
		require.NotNil(t, artifactIO.GetProducer())
		require.NotNil(t, artifactIO.GetProducer().Iteration)
		byIteration[artifactIO.GetProducer().GetIteration()] = artifactIO
	}
	require.Contains(t, byIteration, int64(0))
	require.Contains(t, byIteration, int64(1))
	require.Len(t, byIteration[0].GetArtifacts(), 2)
	require.Len(t, byIteration[1].GetArtifacts(), 1)
}

func strPtr(value string) *string {
	return &value
}
