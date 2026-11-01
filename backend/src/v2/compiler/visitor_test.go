// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package compiler

import (
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

type recordingVisitor struct {
	visited []string
}

func (v *recordingVisitor) Container(name string, _ *pipelinespec.ComponentSpec, _ *pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec) error {
	v.visited = append(v.visited, name)
	return nil
}

func (v *recordingVisitor) Importer(name string, _ *pipelinespec.ComponentSpec, _ *pipelinespec.PipelineDeploymentConfig_ImporterSpec) error {
	v.visited = append(v.visited, name)
	return nil
}

func (v *recordingVisitor) Resolver(name string, _ *pipelinespec.ComponentSpec, _ *pipelinespec.PipelineDeploymentConfig_ResolverSpec) error {
	v.visited = append(v.visited, name)
	return nil
}

func (v *recordingVisitor) DAG(name string, _ *pipelinespec.ComponentSpec, _ *pipelinespec.DagSpec) error {
	v.visited = append(v.visited, name)
	return nil
}

func (v *recordingVisitor) AddKubernetesSpec(string, *structpb.Struct) error {
	return nil
}

// dagComponent returns a DAG component with one task per referenced component.
func dagComponent(componentRefs ...string) *pipelinespec.ComponentSpec {
	tasks := make(map[string]*pipelinespec.PipelineTaskSpec, len(componentRefs))
	for _, ref := range componentRefs {
		tasks["task-"+ref] = &pipelinespec.PipelineTaskSpec{
			TaskInfo:     &pipelinespec.PipelineTaskInfo{Name: "task-" + ref},
			ComponentRef: &pipelinespec.ComponentRef{Name: ref},
		}
	}
	return &pipelinespec.ComponentSpec{
		Implementation: &pipelinespec.ComponentSpec_Dag{Dag: &pipelinespec.DagSpec{Tasks: tasks}},
	}
}

func containerComponent(executorLabel string) *pipelinespec.ComponentSpec {
	return &pipelinespec.ComponentSpec{
		Implementation: &pipelinespec.ComponentSpec_ExecutorLabel{ExecutorLabel: executorLabel},
	}
}

func pipelineJob(t *testing.T, root *pipelinespec.ComponentSpec, components map[string]*pipelinespec.ComponentSpec) *pipelinespec.PipelineJob {
	t.Helper()
	deploymentConfig, err := structpb.NewStruct(map[string]interface{}{
		"executors": map[string]interface{}{
			"exec-leaf": map[string]interface{}{
				"container": map[string]interface{}{"image": "python:3.11"},
			},
		},
	})
	require.NoError(t, err)
	spec := &pipelinespec.PipelineSpec{
		PipelineInfo:   &pipelinespec.PipelineInfo{Name: "test-pipeline"},
		Root:           root,
		Components:     components,
		DeploymentSpec: deploymentConfig,
	}
	specJSON, err := protojson.Marshal(spec)
	require.NoError(t, err)
	specStruct := &structpb.Struct{}
	require.NoError(t, protojson.Unmarshal(specJSON, specStruct))
	return &pipelinespec.PipelineJob{PipelineSpec: specStruct}
}

func TestAccept_RejectsCircularComponentReferences(t *testing.T) {
	tests := []struct {
		name          string
		components    map[string]*pipelinespec.ComponentSpec
		expectedCycle string
	}{
		{
			name: "two components referencing each other",
			components: map[string]*pipelinespec.ComponentSpec{
				"comp-a": dagComponent("comp-b"),
				"comp-b": dagComponent("comp-a"),
			},
			expectedCycle: "root -> comp-a -> comp-b -> comp-a",
		},
		{
			name: "component referencing itself",
			components: map[string]*pipelinespec.ComponentSpec{
				"comp-a": dagComponent("comp-a"),
			},
			expectedCycle: "root -> comp-a -> comp-a",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			visitor := &recordingVisitor{}
			err := Accept(pipelineJob(t, dagComponent("comp-a"), test.components), nil, visitor)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "circular component reference")
			assert.Contains(t, err.Error(), test.expectedCycle)
		})
	}
}

func TestAccept_VisitsSharedComponentOnce(t *testing.T) {
	// comp-a and comp-b both use comp-shared, which is reuse, not a cycle.
	components := map[string]*pipelinespec.ComponentSpec{
		"comp-a":      dagComponent("comp-shared"),
		"comp-b":      dagComponent("comp-shared"),
		"comp-shared": dagComponent("comp-leaf"),
		"comp-leaf":   containerComponent("exec-leaf"),
	}
	visitor := &recordingVisitor{}

	err := Accept(pipelineJob(t, dagComponent("comp-a", "comp-b"), components), nil, visitor)

	require.NoError(t, err)
	assert.Equal(t, []string{"comp-leaf", "comp-shared", "comp-a", "comp-b", "root"}, visitor.visited)
}
