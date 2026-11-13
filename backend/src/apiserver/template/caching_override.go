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

package template

import (
	"errors"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"google.golang.org/protobuf/proto"
)

// withCachingOverride returns a copy of spec in which every task of every DAG
// reachable from the root has caching set to enableCaching. It matches the
// SDK's _override_caching_options, so a run behaves the same whether the
// client rewrites an inline spec or the server rewrites a referenced version.
// Cache keys are left unchanged.
func withCachingOverride(spec *pipelinespec.PipelineSpec, enableCaching bool) *pipelinespec.PipelineSpec {
	out := proto.Clone(spec).(*pipelinespec.PipelineSpec)
	// Tasks inside dsl.If, ParallelFor, ExitHandler and nested pipelines live
	// in the DAGs of their parent components, not in the root DAG.
	dags := []*pipelinespec.DagSpec{out.GetRoot().GetDag()}
	visited := map[string]bool{}
	for len(dags) > 0 {
		dag := dags[len(dags)-1]
		dags = dags[:len(dags)-1]
		for _, task := range dag.GetTasks() {
			if task.CachingOptions == nil {
				task.CachingOptions = &pipelinespec.PipelineTaskSpec_CachingOptions{}
			}
			task.CachingOptions.EnableCache = enableCaching
			name := task.GetComponentRef().GetName()
			if visited[name] {
				continue
			}
			if component, ok := out.GetComponents()[name]; ok && component.GetDag() != nil {
				visited[name] = true
				dags = append(dags, component.GetDag())
			}
		}
	}
	return out
}

// ApplyCachingOverride returns manifest with a run's enable_caching override
// applied to its pipeline spec. A nil override returns manifest unchanged.
//
// Drivers read task specs from the manifest stored with the run, so the
// override must be applied to that manifest, not only to the compiled
// workflow. The result is encoded the same way as stored pipeline versions,
// and any platform spec is kept.
func ApplyCachingOverride(manifest []byte, enableCaching *bool) ([]byte, error) {
	if enableCaching == nil {
		return manifest, nil
	}
	tmpl, err := NewV2SpecTemplate(manifest, TemplateOptions{})
	if err != nil {
		return nil, util.Wrap(err, "Failed to apply enable_caching to the pipeline spec")
	}
	tmpl.spec = withCachingOverride(tmpl.spec, *enableCaching)
	bytes := tmpl.Bytes()
	if bytes == nil {
		return nil, util.NewInternalServerError(errors.New("cannot encode the pipeline spec"), "Failed to apply enable_caching to the pipeline spec")
	}
	return bytes, nil
}
