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
