// Copyright 2021-2023 The Kubeflow Authors
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
	"fmt"
	"path/filepath"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/compiler"
	workflowutils "github.com/kubeflow/pipelines/backend/test/compiler/utils"
	. "github.com/kubeflow/pipelines/backend/test/constants"
	"github.com/kubeflow/pipelines/backend/test/testutil"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type TestVisitor struct {
	visited []string
}

func (visitor *TestVisitor) Container(name string, component *pipelinespec.ComponentSpec, executor *pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec) error {
	visitor.visited = append(visitor.visited, fmt.Sprintf("container(name=%q)", name))
	return nil
}
func (visitor *TestVisitor) Importer(name string, component *pipelinespec.ComponentSpec, importer *pipelinespec.PipelineDeploymentConfig_ImporterSpec) error {
	visitor.visited = append(visitor.visited, fmt.Sprintf("importer(name=%q)", name))
	return nil
}
func (visitor *TestVisitor) Resolver(name string, component *pipelinespec.ComponentSpec, resolver *pipelinespec.PipelineDeploymentConfig_ResolverSpec) error {
	visitor.visited = append(visitor.visited, fmt.Sprintf("resolver(name=%q)", name))
	return nil
}
func (visitor *TestVisitor) DAG(name string, component *pipelinespec.ComponentSpec, dag *pipelinespec.DagSpec) error {
	visitor.visited = append(visitor.visited, fmt.Sprintf("DAG(name=%q)", name))
	return nil
}
func (visitor *TestVisitor) AddKubernetesSpec(name string, kubernetesSpec *structpb.Struct) error {
	visitor.visited = append(visitor.visited, fmt.Sprintf("DAG(name=%q)", name))
	return nil
}

var _ = Describe("Verify iteration over the pipeline components >", Label(POSITIVE, WorkflowCompiler, WorkflowCompilerVisits), func() {
	Context("Validate that compiler starts with the correct root", func() {

		testParams := []struct {
			pipelineSpecPath string
			expectedVisited  []string
		}{
			{
				pipelineSpecPath: "hello-world.yaml",
				expectedVisited: []string{
					"container(name=\"comp-echo\")",
					"DAG(name=\"root\")",
				},
			},
			{
				pipelineSpecPath: "pipeline_with_volume_no_cache.yaml",
				expectedVisited: []string{"DAG(name=\"comp-consumer\")",
					"container(name=\"comp-consumer\")",
					"container(name=\"comp-createpvc\")",
					"container(name=\"comp-deletepvc\")",
					"DAG(name=\"comp-producer\")",
					"container(name=\"comp-producer\")",
					"DAG(name=\"root\")",
				},
			},
			{
				pipelineSpecPath: "create_pod_metadata_complex.yaml",
				expectedVisited: []string{
					"container(name=\"comp-validate-no-pod-metadata\")",
					"DAG(name=\"comp-validate-pod-metadata\")",
					"container(name=\"comp-validate-pod-metadata\")",
					"DAG(name=\"comp-validate-pod-metadata-2\")",
					"container(name=\"comp-validate-pod-metadata-2\")",
					"DAG(name=\"root\")",
				},
			},
			{
				pipelineSpecPath: "critical/nested_pipeline_opt_input_child_level_compiled.yaml",
				expectedVisited: []string{
					"container(name=\"comp-component-a-bool\")",
					"container(name=\"comp-component-a-int\")",
					"container(name=\"comp-component-a-str\")",
					"container(name=\"comp-component-b-bool\")",
					"container(name=\"comp-component-b-int\")",
					"container(name=\"comp-component-b-str\")",
					"DAG(name=\"comp-nested-pipeline\")",
					"DAG(name=\"root\")",
				},
			},
			{
				pipelineSpecPath: "critical/parallel_for_after_dependency.yaml",
				expectedVisited: []string{
					"container(name=\"comp-print-op\")",
					"DAG(name=\"comp-for-loop-2\")",
					"container(name=\"comp-print-op-2\")",
					"container(name=\"comp-print-op-3\")",
					"DAG(name=\"root\")",
				},
			},
		}
		for _, testParam := range testParams {
			pipelineSpecFilePath := filepath.Join(testutil.GetValidPipelineFilesDir(), testParam.pipelineSpecPath)
			It(fmt.Sprintf("Load the the pipeline IR yaml %s, and verify the visited component", testParam.pipelineSpecPath), func() {
				pipelineJob, platformSpec := workflowutils.LoadPipelineSpecsFromIR(pipelineSpecFilePath, false, nil)

				actualVisitor := &TestVisitor{visited: make([]string, 0)}
				err := compiler.Accept(pipelineJob, platformSpec, actualVisitor)
				Expect(err).To(BeNil(), "Failed to iterate over the pipeline specs")
				Expect(actualVisitor.visited).To(Equal(testParam.expectedVisited))
			})
		}
	})
})
