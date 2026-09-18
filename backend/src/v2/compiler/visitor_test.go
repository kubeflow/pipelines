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
	"encoding/json"
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

type dummyVisitor struct {
	visitedContainers []string
	visitedDAGs       []string
}

func (v *dummyVisitor) Container(name string, component *pipelinespec.ComponentSpec, container *pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec) error {
	v.visitedContainers = append(v.visitedContainers, name)
	return nil
}

func (v *dummyVisitor) Importer(name string, component *pipelinespec.ComponentSpec, importer *pipelinespec.PipelineDeploymentConfig_ImporterSpec) error {
	return nil
}

func (v *dummyVisitor) Resolver(name string, component *pipelinespec.ComponentSpec, resolver *pipelinespec.PipelineDeploymentConfig_ResolverSpec) error {
	return nil
}

func (v *dummyVisitor) DAG(name string, component *pipelinespec.ComponentSpec, dag *pipelinespec.DagSpec) error {
	v.visitedDAGs = append(v.visitedDAGs, name)
	return nil
}

func (v *dummyVisitor) AddKubernetesSpec(name string, kubernetesSpec *structpb.Struct) error {
	return nil
}

func TestAccept_NilJob(t *testing.T) {
	v := &dummyVisitor{}
	err := Accept(nil, nil, v)
	assert.NoError(t, err)
}

func TestAccept_ReservedRootComponentName(t *testing.T) {
	spec := &pipelinespec.PipelineSpec{
		Components: map[string]*pipelinespec.ComponentSpec{
			"root": {
				Implementation: &pipelinespec.ComponentSpec_ExecutorLabel{
					ExecutorLabel: "exec-1",
				},
			},
		},
	}
	specStruct, err := structpb.NewStruct(protoStructToMap(t, spec))
	assert.NoError(t, err)

	job := &pipelinespec.PipelineJob{
		PipelineSpec: specStruct,
	}

	v := &dummyVisitor{}
	err = Accept(job, nil, v)
	assert.ErrorContains(t, err, `reserved component name "root" cannot be used as a user component name`)
}

func TestAccept_CircularComponentReference(t *testing.T) {
	spec := &pipelinespec.PipelineSpec{
		Root: &pipelinespec.ComponentSpec{
			Implementation: &pipelinespec.ComponentSpec_Dag{
				Dag: &pipelinespec.DagSpec{
					Tasks: map[string]*pipelinespec.PipelineTaskSpec{
						"task-a": {
							TaskInfo: &pipelinespec.PipelineTaskInfo{Name: "task-a"},
							ComponentRef: &pipelinespec.ComponentRef{
								Name: "comp-a",
							},
						},
					},
				},
			},
		},
		Components: map[string]*pipelinespec.ComponentSpec{
			"comp-a": {
				Implementation: &pipelinespec.ComponentSpec_Dag{
					Dag: &pipelinespec.DagSpec{
						Tasks: map[string]*pipelinespec.PipelineTaskSpec{
							"task-b": {
								TaskInfo: &pipelinespec.PipelineTaskInfo{Name: "task-b"},
								ComponentRef: &pipelinespec.ComponentRef{
									Name: "comp-b",
								},
							},
						},
					},
				},
			},
			"comp-b": {
				Implementation: &pipelinespec.ComponentSpec_Dag{
					Dag: &pipelinespec.DagSpec{
						Tasks: map[string]*pipelinespec.PipelineTaskSpec{
							"task-cycle": {
								TaskInfo: &pipelinespec.PipelineTaskInfo{Name: "task-cycle"},
								ComponentRef: &pipelinespec.ComponentRef{
									Name: "comp-a", // Creates cycle comp-a -> comp-b -> comp-a
								},
							},
						},
					},
				},
			},
		},
	}

	specStruct, err := structpb.NewStruct(protoStructToMap(t, spec))
	assert.NoError(t, err)

	job := &pipelinespec.PipelineJob{
		PipelineSpec: specStruct,
	}

	v := &dummyVisitor{}
	err = Accept(job, nil, v)
	assert.ErrorContains(t, err, "circular reference detected in component graph: comp-a")
}

func TestAccept_EnrichedErrorContextForMissingExecutor(t *testing.T) {
	spec := &pipelinespec.PipelineSpec{
		Root: &pipelinespec.ComponentSpec{
			Implementation: &pipelinespec.ComponentSpec_Dag{
				Dag: &pipelinespec.DagSpec{
					Tasks: map[string]*pipelinespec.PipelineTaskSpec{
						"missing-exec-task": {
							TaskInfo: &pipelinespec.PipelineTaskInfo{Name: "missing-exec-task"},
							ComponentRef: &pipelinespec.ComponentRef{
								Name: "comp-missing",
							},
						},
					},
				},
			},
		},
		Components: map[string]*pipelinespec.ComponentSpec{
			"comp-missing": {
				Implementation: &pipelinespec.ComponentSpec_ExecutorLabel{
					ExecutorLabel: "non-existent-executor",
				},
			},
		},
	}

	specStruct, err := structpb.NewStruct(protoStructToMap(t, spec))
	assert.NoError(t, err)

	job := &pipelinespec.PipelineJob{
		PipelineSpec: specStruct,
	}

	v := &dummyVisitor{}
	err = Accept(job, nil, v)
	assert.ErrorContains(t, err, `error processing task "missing-exec-task" (component name="comp-missing")`)
}

func protoStructToMap(t *testing.T, msg proto.Message) map[string]interface{} {
	jsonBytes, err := protojson.Marshal(msg)
	assert.NoError(t, err)
	var result map[string]interface{}
	err = json.Unmarshal(jsonBytes, &result)
	assert.NoError(t, err)
	return result
}
