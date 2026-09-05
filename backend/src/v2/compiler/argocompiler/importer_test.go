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

package argocompiler

import (
	"testing"
	"time"

	wfapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"
)

func newImporterTestCompiler() *workflowCompiler {
	return &workflowCompiler{
		spec:      &pipelinespec.PipelineSpec{PipelineInfo: &pipelinespec.PipelineInfo{Name: "test-pipeline"}},
		templates: make(map[string]*wfapi.Template),
		wf: &wfapi.Workflow{
			Spec: wfapi.WorkflowSpec{Templates: []wfapi.Template{}},
		},
	}
}

func TestAddImporterTemplate_NoRetryPolicy(t *testing.T) {
	c := newImporterTestCompiler()

	name := c.addImporterTemplate(false, nil)

	assert.Equal(t, "system-importer", name)
	tmpl, exists := c.templates[name]
	require.True(t, exists)
	assert.Nil(t, tmpl.RetryStrategy)
}

func TestAddImporterTemplate_WithRetryPolicy(t *testing.T) {
	c := newImporterTestCompiler()
	retryPolicy := &pipelinespec.PipelineTaskSpec_RetryPolicy{
		MaxRetryCount:      3,
		BackoffDuration:    durationpb.New(10 * time.Second),
		BackoffFactor:      2.0,
		BackoffMaxDuration: durationpb.New(60 * time.Second),
	}

	name := c.addImporterTemplate(false, retryPolicy)

	assert.Equal(t, "retry-system-importer", name)
	tmpl, exists := c.templates[name]
	require.True(t, exists)
	require.NotNil(t, tmpl.RetryStrategy)
	assert.Equal(t, "{{inputs.parameters.retry-max-count}}", tmpl.RetryStrategy.Limit.StrVal)
	require.NotNil(t, tmpl.RetryStrategy.Backoff)
	assert.Equal(t, "{{inputs.parameters.retry-backoff-duration}}", tmpl.RetryStrategy.Backoff.Duration)
	assert.Equal(t, "{{inputs.parameters.retry-backoff-factor}}", tmpl.RetryStrategy.Backoff.Factor.StrVal)
	assert.Equal(t, "{{inputs.parameters.retry-backoff-max-duration}}", tmpl.RetryStrategy.Backoff.MaxDuration)

	// The plain (non-retry) template must stay untouched by a differently
	// configured importer task, so tasks without a retry policy don't
	// inherit another task's retryStrategy.
	plainName := c.addImporterTemplate(false, nil)
	assert.Equal(t, "system-importer", plainName)
	plainTmpl := c.templates[plainName]
	assert.Nil(t, plainTmpl.RetryStrategy)
}

func TestAddImporterTemplate_WorkspaceWithRetryPolicy(t *testing.T) {
	c := newImporterTestCompiler()
	retryPolicy := &pipelinespec.PipelineTaskSpec_RetryPolicy{MaxRetryCount: 1}

	name := c.addImporterTemplate(true, retryPolicy)

	assert.Equal(t, "retry-system-importer-workspace", name)
	tmpl, exists := c.templates[name]
	require.True(t, exists)
	require.NotNil(t, tmpl.RetryStrategy)
}

func TestImporterTask_MultipleTasksWithDistinctPolicies(t *testing.T) {
	c := newImporterTestCompiler()
	taskWithRetry := &pipelinespec.PipelineTaskSpec{
		ComponentRef: &pipelinespec.ComponentRef{Name: "comp-with-retry"},
		RetryPolicy:  &pipelinespec.PipelineTaskSpec_RetryPolicy{MaxRetryCount: 3},
	}
	taskWithoutRetry := &pipelinespec.PipelineTaskSpec{
		ComponentRef: &pipelinespec.ComponentRef{Name: "comp-without-retry"},
	}
	c.executors = map[string]*pipelinespec.PipelineDeploymentConfig_ExecutorSpec{}
	require.NoError(t, c.Importer("comp-with-retry", &pipelinespec.ComponentSpec{}, &pipelinespec.PipelineDeploymentConfig_ImporterSpec{}))
	require.NoError(t, c.Importer("comp-without-retry", &pipelinespec.ComponentSpec{}, &pipelinespec.PipelineDeploymentConfig_ImporterSpec{}))

	dagTaskWithRetry, err := c.importerTask("with-retry", taskWithRetry, "{}", "parent-dag-id", false)
	require.NoError(t, err)
	dagTaskWithoutRetry, err := c.importerTask("without-retry", taskWithoutRetry, "{}", "parent-dag-id", false)
	require.NoError(t, err)

	assert.Equal(t, "retry-system-importer", dagTaskWithRetry.Template)
	assert.Equal(t, "system-importer", dagTaskWithoutRetry.Template)
	assert.NotNil(t, c.templates["retry-system-importer"].RetryStrategy)
	assert.Nil(t, c.templates["system-importer"].RetryStrategy)
}
