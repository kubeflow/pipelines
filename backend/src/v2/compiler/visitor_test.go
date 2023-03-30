// Copyright 2021-2023 The Kubeflow Authors
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
package compiler_test

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/golang/protobuf/jsonpb"
	"github.com/google/go-cmp/cmp"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/compiler"
	"google.golang.org/protobuf/types/known/structpb"
)

type testVisitor struct {
	visited []string
}

func (v *testVisitor) Container(name string, component *pipelinespec.ComponentSpec, executor *pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec) error {
	v.visited = append(v.visited, fmt.Sprintf("container(name=%q)", name))
	return nil
}
func (v *testVisitor) Importer(name string, component *pipelinespec.ComponentSpec, importer *pipelinespec.PipelineDeploymentConfig_ImporterSpec) error {
	v.visited = append(v.visited, fmt.Sprintf("importer(name=%q)", name))
	return nil
}
func (v *testVisitor) Resolver(name string, component *pipelinespec.ComponentSpec, resolver *pipelinespec.PipelineDeploymentConfig_ResolverSpec) error {
	v.visited = append(v.visited, fmt.Sprintf("resolver(name=%q)", name))
	return nil
}
func (v *testVisitor) DAG(name string, component *pipelinespec.ComponentSpec, dag *pipelinespec.DagSpec) error {
	v.visited = append(v.visited, fmt.Sprintf("DAG(name=%q)", name))
	return nil
}
func (v *testVisitor) AddKubernetesSpec(name string, kubernetesSpec *structpb.Struct) error {
	v.visited = append(v.visited, fmt.Sprintf("DAG(name=%q)", name))
	return nil
}

func Test_AcceptTestVisitor(t *testing.T) {
	tests := []struct {
		specPath string
		expected []string
	}{
		{
			specPath: "testdata/hello_world.json",
			expected: []string{`container(name="comp-hello-world")`, `DAG(name="root")`},
		},
		{
			specPath: "testdata/producer_consumer_param.json",
			expected: []string{`container(name="comp-consumer")`, `container(name="comp-producer")`, `DAG(name="root")`},
		},
		{
			// Component comp-hello-world used twice, but it should only be visited once.
			specPath: "testdata/component_used_twice.json",
			expected: []string{`container(name="comp-hello-world")`, `DAG(name="root")`},
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%q", tt.specPath), func(t *testing.T) {
			job := load(t, tt.specPath)
			v := &testVisitor{visited: make([]string, 0)}
			err := compiler.Accept(job, nil, v)
			if err != nil {
				t.Fatal(err)
			}
			if !cmp.Equal(v.visited, tt.expected) {
				t.Errorf("   got: %v\nexpect: %v", v.visited, tt.expected)
			}
		})
	}
}

func load(t *testing.T, path string) *pipelinespec.PipelineJob {
	t.Helper()
	content, err := ioutil.ReadFile(path)
	if err != nil {
		t.Error(err)
	}
	json := string(content)
	job := &pipelinespec.PipelineJob{}
	if err := jsonpb.UnmarshalString(json, job); err != nil {
		t.Errorf("Failed to parse pipeline job, error: %s, job: %v", err, json)
	}
	return job
}
