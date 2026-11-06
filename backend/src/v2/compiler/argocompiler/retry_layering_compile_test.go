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

package argocompiler_test

import (
	"path/filepath"
	"testing"

	wfapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/compiler/argocompiler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

// compilePipeline compiles a pipeline IR fixture the way the api-server does.
func compilePipeline(t *testing.T, pipelineFile string) *wfapi.Workflow {
	t.Helper()
	pipelineSpec, platformSpec, err := util.LoadPipelineAndPlatformSpec(filepath.Join(
		"../../../../../test_data/sdk_compiled_pipelines/valid", pipelineFile))
	require.NoError(t, err)
	pipelineJSON, err := protojson.Marshal(pipelineSpec)
	require.NoError(t, err)
	pipelineStruct := &structpb.Struct{}
	require.NoError(t, protojson.Unmarshal(pipelineJSON, pipelineStruct))
	workflow, err := argocompiler.Compile(
		&pipelinespec.PipelineJob{PipelineSpec: pipelineStruct},
		platformSpec.GetPlatforms()["kubernetes"],
		&argocompiler.Options{},
	)
	require.NoError(t, err)
	return workflow
}

// TestCompileNeutralizesDAGRetriesRegardlessOfSetRetry locks in that every DAG
// template a pipeline compiles to carries an explicit zero retry limit.
//
// Argo's templateDefaults gives every template that does not declare its own
// retryStrategy the deployment-wide one. On a pod that re-runs a single
// container, which is what the deployment default is for. On a DAG it re-runs
// the entire subgraph, and because DAGs nest, those re-runs multiply. KFP never
// sets a meaningful retryStrategy on a DAG, and group-level retry is not a
// supported KFP operation, so any retry on a compiled DAG came from inheritance
// rather than from intent.
//
// The retry-less fixtures are the point of this test: whether a task uses
// set_retry says nothing about whether the deployment configures a cascading
// retry policy, so DAG treatment must not depend on it.
func TestCompileNeutralizesDAGRetriesRegardlessOfSetRetry(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()
	tests := []struct {
		name string
		// pipelineFile is relative to test_data/sdk_compiled_pipelines/valid.
		pipelineFile string
		// wantDAGs are DAG templates the fixture must produce, so a fixture
		// that stops covering a DAG shape fails loudly instead of silently.
		wantDAGs []string
	}{
		{
			name:         "no set_retry",
			pipelineFile: "hello_world.yaml",
			wantDAGs:     []string{"entrypoint", "root", "system-container-executor"},
		},
		{
			name:         "no set_retry, iterator DAGs",
			pipelineFile: "parallel_and_nested/pipeline_with_loops.yaml",
			wantDAGs: []string{
				"entrypoint", "root",
				"comp-for-loop-1", "comp-for-loop-1-for-loop-1-iterator",
			},
		},
		{
			name:         "no set_retry, exit handler DAG",
			pipelineFile: "failing/pipeline_with_exit_handler.yaml",
			wantDAGs:     []string{"entrypoint", "root", "comp-exit-handler-1"},
		},
		{
			name:         "set_retry",
			pipelineFile: "critical/pipeline_with_retry.yaml",
			wantDAGs:     []string{"entrypoint", "root", "retry-system-container-executor"},
		},
		{
			name:         "set_retry with an explicit policy",
			pipelineFile: "pipeline_with_retry_policy.yaml",
			wantDAGs:     []string{"entrypoint", "root", "retry-system-container-executor-onerror"},
		},
		{
			name:         "set_retry on one task only",
			pipelineFile: "driver_plugin/pipeline_with_flaky_retry.yaml",
			wantDAGs: []string{
				"entrypoint", "root",
				"system-container-executor",       // the task without retry
				"retry-system-container-executor", // the task with retry
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			workflow := compilePipeline(t, test.pipelineFile)

			templates := make(map[string]wfapi.Template, len(workflow.Spec.Templates))
			sawDAG := false
			for _, template := range workflow.Spec.Templates {
				templates[template.Name] = template
				if template.DAG == nil {
					continue
				}
				sawDAG = true
				require.NotNil(t, template.RetryStrategy,
					"DAG template %q has no retryStrategy, so templateDefaults supplies one", template.Name)
				require.NotNil(t, template.RetryStrategy.Limit, "DAG template %q", template.Name)
				assert.Equal(t, "0", template.RetryStrategy.Limit.String(),
					"DAG template %q must not retry its subgraph", template.Name)
			}
			require.True(t, sawDAG, "fixture produced no DAG templates")

			for _, name := range test.wantDAGs {
				template, found := templates[name]
				require.True(t, found, "fixture no longer produces template %q", name)
				require.NotNil(t, template.DAG, "template %q is expected to be a DAG", name)
			}
		})
	}
}

// TestCompileLeavesNonDAGTemplatesInheritingTheDeploymentDefault is the other
// half of the invariant: neutralizing DAGs must not reach pods or plugins.
// Retrying one of those re-runs a single node, which is the behavior the
// deployment default exists to provide, and the driver in particular does the
// object-store and metadata I/O that such a default is usually configured for.
func TestCompileLeavesNonDAGTemplatesInheritingTheDeploymentDefault(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()

	t.Run("without set_retry nothing outside a DAG is stamped", func(t *testing.T) {
		workflow := compilePipeline(t, "hello_world.yaml")
		for _, template := range workflow.Spec.Templates {
			if template.DAG != nil {
				continue
			}
			assert.Nil(t, template.RetryStrategy,
				"non-DAG template %q must keep inheriting the deployment default", template.Name)
		}
	})

	t.Run("with set_retry only the task's own pod is stamped", func(t *testing.T) {
		workflow := compilePipeline(t, "critical/pipeline_with_retry.yaml")
		templates := make(map[string]wfapi.Template, len(workflow.Spec.Templates))
		for _, template := range workflow.Spec.Templates {
			templates[template.Name] = template
		}

		// Drivers are plugin templates, and set_retry means "retry my
		// component", not "retry the driver KFP runs on its behalf".
		for _, name := range []string{"system-container-driver", "system-dag-driver"} {
			template, found := templates[name]
			require.True(t, found, "missing driver template %q", name)
			require.NotNil(t, template.Plugin, "template %q is expected to be a plugin", name)
			assert.Nil(t, template.RetryStrategy,
				"driver template %q must keep inheriting the deployment default", name)
		}

		// The task's pod keeps the user's retry settings, not a zero stamp.
		// The limit is a parameter reference rather than a literal, because
		// num_retries is resolved at runtime, so compare the rendered form.
		impl, found := templates["retry-system-container-impl"]
		require.True(t, found, "missing impl template")
		require.NotNil(t, impl.Container, "impl template is expected to be a pod")
		require.NotNil(t, impl.RetryStrategy, "impl template must carry the task's retry settings")
		require.NotNil(t, impl.RetryStrategy.Limit)
		assert.NotEqual(t, "0", impl.RetryStrategy.Limit.String(),
			"the task's own retry limit must survive DAG neutralization")
	})
}
