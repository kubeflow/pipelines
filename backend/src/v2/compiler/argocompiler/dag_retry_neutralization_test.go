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
	k8sintstr "k8s.io/apimachinery/pkg/util/intstr"
)

// The bundled workflow-controller ConfigMap configures this as
// templateDefaults.retryStrategy.
func bundledDeploymentDefault() *wfapi.RetryStrategy {
	return &wfapi.RetryStrategy{
		Limit:       &k8sintstr.IntOrString{Type: k8sintstr.Int, IntVal: 2},
		RetryPolicy: wfapi.RetryPolicyOnError,
	}
}

// effectiveRetryStrategy mirrors what the controller does with a compiled
// template: mergedTemplateDefaultsInto strategic-merges templateDefaults into
// the template, so a field the template sets wins and the default fills in the
// fields it left unset.
func effectiveRetryStrategy(template wfapi.Template) *wfapi.RetryStrategy {
	effective := bundledDeploymentDefault()
	if template.RetryStrategy == nil {
		return effective
	}
	if template.RetryStrategy.Limit != nil {
		effective.Limit = template.RetryStrategy.Limit
	}
	if template.RetryStrategy.RetryPolicy != "" {
		effective.RetryPolicy = template.RetryStrategy.RetryPolicy
	}
	if template.RetryStrategy.Backoff != nil {
		effective.Backoff = template.RetryStrategy.Backoff
	}
	if template.RetryStrategy.Expression != "" {
		effective.Expression = template.RetryStrategy.Expression
	}
	return effective
}

// compilePipeline compiles one of the SDK-compiled test pipelines the same way
// the API server does.
func compilePipeline(t *testing.T, pipelineFile string) *wfapi.Workflow {
	t.Helper()
	proxy.InitializeConfigWithEmptyForTests()
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

// reachedTemplates returns the templates the run can actually execute, by
// following DAG task references from the entrypoint down. No test here asserts
// a generated template name.
func reachedTemplates(t *testing.T, workflow *wfapi.Workflow) []wfapi.Template {
	t.Helper()
	byName := make(map[string]wfapi.Template, len(workflow.Spec.Templates))
	for _, template := range workflow.Spec.Templates {
		byName[template.Name] = template
	}
	require.NotEmpty(t, workflow.Spec.Entrypoint, "compiled workflow has no entrypoint")

	var reached []wfapi.Template
	seen := map[string]bool{}
	queue := []string{workflow.Spec.Entrypoint}
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		if seen[name] {
			continue
		}
		seen[name] = true
		template, found := byName[name]
		require.True(t, found, "template %q referenced but not defined", name)
		reached = append(reached, template)
		if template.DAG == nil {
			continue
		}
		for _, task := range template.DAG.Tasks {
			queue = append(queue, task.Template)
		}
	}
	return reached
}

// TestCompileNeutralizesDAGRetriesWithoutSetRetry covers the wiring this change
// fixes: a pipeline that never calls set_retry() used to keep strategy-less DAG
// templates, so the deployment default reached them and each retried DAG
// re-ran its subtree. retry_layering_test.go covers neutralizeDAGRetries
// itself; this covers Compile calling it for such a pipeline.
func TestCompileNeutralizesDAGRetriesWithoutSetRetry(t *testing.T) {
	workflow := compilePipeline(t, "hello_world.yaml")

	dags := 0
	var pod *wfapi.Template
	for _, template := range reachedTemplates(t, workflow) {
		if template.DAG == nil {
			if pod == nil {
				reached := template
				pod = &reached
			}
			continue
		}
		dags++
		require.NotNil(t, template.RetryStrategy,
			"DAG template %q must carry an explicit strategy, otherwise templateDefaults supplies one", template.Name)
		assert.Equal(t, "0", effectiveRetryStrategy(template).Limit.String(),
			"DAG template %q must not retry: a retry there re-runs its whole subgraph", template.Name)
	}
	assert.NotZero(t, dags, "expected the compiled workflow to contain DAG templates")

	require.NotNil(t, pod, "expected the pipeline to reach a pod template")
	assert.Nil(t, pod.RetryStrategy, "template %q was not given a strategy", pod.Name)
	assert.Equal(t, bundledDeploymentDefault(), effectiveRetryStrategy(*pod),
		"template %q should keep inheriting the deployment default", pod.Name)
}

// TestCompileNeutralizesDAGRetriesWithSetRetry is the wiring counterpart for
// pipelines that do configure retries: their DAGs must opt out as well, so the
// task's own settings decide how often the work runs. Which strategy survives
// on the task's template is covered by retry_layering_test.go.
func TestCompileNeutralizesDAGRetriesWithSetRetry(t *testing.T) {
	for _, pipelineFile := range []string{
		"pipeline_with_retry_policy.yaml",
		"critical/pipeline_with_retry.yaml",
	} {
		t.Run(pipelineFile, func(t *testing.T) {
			workflow := compilePipeline(t, pipelineFile)

			dags, tasksWithOwnStrategy := 0, 0
			for _, template := range reachedTemplates(t, workflow) {
				if template.DAG != nil {
					dags++
					require.NotNil(t, template.RetryStrategy,
						"DAG template %q must not inherit the deployment default", template.Name)
					continue
				}
				if template.RetryStrategy != nil {
					tasksWithOwnStrategy++
				}
			}
			assert.NotZero(t, dags, "expected the compiled workflow to contain DAG templates")
			assert.NotZero(t, tasksWithOwnStrategy, "expected the task's own retry strategy to survive")
		})
	}
}
