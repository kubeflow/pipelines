// Copyright 2025 The Kubeflow Authors
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

package argocompiler

import (
	"testing"
	"time"

	wfapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/v2/config"
	"github.com/spf13/viper"
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

func TestAddImporterTemplate_PropagatesIterationIndex(t *testing.T) {
	c := &workflowCompiler{
		templates: make(map[string]*wfapi.Template),
		wf: &wfapi.Workflow{
			Spec: wfapi.WorkflowSpec{
				Templates: []wfapi.Template{},
			},
		},
		spec: &pipelinespec.PipelineSpec{
			PipelineInfo: &pipelinespec.PipelineInfo{Name: "test-pipeline"},
		},
	}

	name := c.addImporterTemplate(false, nil)
	require.Equal(t, "system-importer", name)

	tmpl, exists := c.templates[name]
	require.True(t, exists)
	require.NotNil(t, tmpl)
	require.NotNil(t, tmpl.Container)

	assert.Contains(t, tmpl.Container.Args, "--iteration_index")
	assert.Contains(t, tmpl.Container.Args, inputValue(paramIterationIndex))
	assertAdjacentArgPair(t, tmpl.Container.Args, "--ml_pipeline_server_address", config.GetMLPipelineServerConfig().Address)
	assertAdjacentArgPair(t, tmpl.Container.Args, "--ml_pipeline_server_port", config.GetMLPipelineServerConfig().Port)

	var foundIterationInput bool
	for _, param := range tmpl.Inputs.Parameters {
		if param.Name == paramIterationIndex {
			foundIterationInput = true
			require.NotNil(t, param.Default)
			assert.EqualValues(t, "-1", *param.Default)
		}
	}
	assert.True(t, foundIterationInput, "importer template should declare the iteration index input")

	foundLauncherConfigMount := false
	for _, volumeMount := range tmpl.Container.VolumeMounts {
		if volumeMount.Name == launcherConfigVolumeName && volumeMount.MountPath == config.LauncherConfigMountPath {
			foundLauncherConfigMount = true
			break
		}
	}
	assert.True(t, foundLauncherConfigMount, "importer should optionally mount kfp-launcher config")
}

func TestAddImporterTemplate_PropagatesCustomEndpoint(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	viper.Set(common.MLPipelineServiceName, "custom-ml-pipeline")
	viper.Set(common.PodNamespace, "pipelines-ns")
	viper.Set(common.ClusterDomain, "cluster.example")

	expectedAddress := "custom-ml-pipeline.pipelines-ns.svc.cluster.example"
	expectedPort := "8887"
	require.Equal(t, expectedAddress, config.GetMLPipelineServerConfig().Address)
	require.Equal(t, expectedPort, config.GetMLPipelineServerConfig().Port)

	c := &workflowCompiler{
		templates: make(map[string]*wfapi.Template),
		wf: &wfapi.Workflow{
			Spec: wfapi.WorkflowSpec{
				Templates: []wfapi.Template{},
			},
		},
		spec: &pipelinespec.PipelineSpec{
			PipelineInfo: &pipelinespec.PipelineInfo{Name: "test-pipeline"},
		},
	}

	name := c.addImporterTemplate(false, nil)
	require.Equal(t, "system-importer", name)
	tmpl, exists := c.templates[name]
	require.True(t, exists)
	require.NotNil(t, tmpl.Container)

	assertAdjacentArgPair(t, tmpl.Container.Args, "--ml_pipeline_server_address", expectedAddress)
	assertAdjacentArgPair(t, tmpl.Container.Args, "--ml_pipeline_server_port", expectedPort)
}

func assertAdjacentArgPair(t *testing.T, args []string, flag, value string) {
	t.Helper()
	for index, arg := range args {
		if arg != flag {
			continue
		}
		require.Less(t, index+1, len(args), "flag %s missing value", flag)
		assert.Equal(t, value, args[index+1], "flag %s should be followed by %s", flag, value)
		return
	}
	t.Fatalf("flag %s not found in args %v", flag, args)
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

func TestAddImporterTemplate_RetryPolicyInputsHaveDefaults(t *testing.T) {
	c := newImporterTestCompiler()
	// A partial retry policy: getTaskRetryParametersWithValues omits the
	// duration/max-duration arguments when they're nil, so the template's
	// inputs must declare defaults for all four or Argo submission fails
	// with missing required arguments.
	retryPolicy := &pipelinespec.PipelineTaskSpec_RetryPolicy{MaxRetryCount: 3}

	name := c.addImporterTemplate(false, retryPolicy)

	tmpl, exists := c.templates[name]
	require.True(t, exists)
	retryParamNames := []string{
		paramRetryMaxCount,
		paramRetryBackOffDuration,
		paramRetryBackOffFactor,
		paramRetryBackOffMaxDuration,
	}
	found := map[string]bool{}
	for _, param := range tmpl.Inputs.Parameters {
		for _, retryParamName := range retryParamNames {
			if param.Name == retryParamName {
				require.NotNil(t, param.Default, "retry input %s should have a default", param.Name)
				assert.Equal(t, "0", string(*param.Default))
				found[param.Name] = true
			}
		}
	}
	for _, retryParamName := range retryParamNames {
		assert.True(t, found[retryParamName], "expected retry input %s to be declared", retryParamName)
	}
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
	taskWithRetryA := &pipelinespec.PipelineTaskSpec{
		ComponentRef: &pipelinespec.ComponentRef{Name: "comp-retry-a"},
		RetryPolicy: &pipelinespec.PipelineTaskSpec_RetryPolicy{
			MaxRetryCount:   3,
			BackoffDuration: durationpb.New(10 * time.Second),
			BackoffFactor:   2.0,
		},
	}
	taskWithRetryB := &pipelinespec.PipelineTaskSpec{
		ComponentRef: &pipelinespec.ComponentRef{Name: "comp-retry-b"},
		RetryPolicy: &pipelinespec.PipelineTaskSpec_RetryPolicy{
			MaxRetryCount:      5,
			BackoffFactor:      1.5,
			BackoffMaxDuration: durationpb.New(120 * time.Second),
		},
	}
	taskWithoutRetry := &pipelinespec.PipelineTaskSpec{
		ComponentRef: &pipelinespec.ComponentRef{Name: "comp-without-retry"},
	}
	c.executors = map[string]*pipelinespec.PipelineDeploymentConfig_ExecutorSpec{}
	require.NoError(t, c.Importer("comp-retry-a", &pipelinespec.ComponentSpec{}, &pipelinespec.PipelineDeploymentConfig_ImporterSpec{}))
	require.NoError(t, c.Importer("comp-retry-b", &pipelinespec.ComponentSpec{}, &pipelinespec.PipelineDeploymentConfig_ImporterSpec{}))
	require.NoError(t, c.Importer("comp-without-retry", &pipelinespec.ComponentSpec{}, &pipelinespec.PipelineDeploymentConfig_ImporterSpec{}))

	dagTaskA, err := c.importerTask("retry-a", taskWithRetryA, "retry-a", "parent-dag-id", false)
	require.NoError(t, err)
	dagTaskB, err := c.importerTask("retry-b", taskWithRetryB, "retry-b", "parent-dag-id", false)
	require.NoError(t, err)
	dagTaskWithoutRetry, err := c.importerTask("without-retry", taskWithoutRetry, "without-retry", "parent-dag-id", false)
	require.NoError(t, err)

	assert.Equal(t, "retry-system-importer", dagTaskA.Template)
	assert.Equal(t, "retry-system-importer", dagTaskB.Template)
	assert.Equal(t, "system-importer", dagTaskWithoutRetry.Template)
	assert.NotNil(t, c.templates["retry-system-importer"].RetryStrategy)
	assert.Nil(t, c.templates["system-importer"].RetryStrategy)

	// Both retry-enabled tasks share the "retry-system-importer" template,
	// so their distinct retry values must be threaded through per-task DAG
	// arguments rather than leaking into each other or into the template.
	argsA := retryArgValues(dagTaskA)
	assert.Equal(t, "3", argsA[paramRetryMaxCount])
	assert.Equal(t, "10", argsA[paramRetryBackOffDuration])
	assert.Equal(t, "2", argsA[paramRetryBackOffFactor])
	_, hasMaxDuration := argsA[paramRetryBackOffMaxDuration]
	assert.False(t, hasMaxDuration, "task A did not set a backoff max duration")

	argsB := retryArgValues(dagTaskB)
	assert.Equal(t, "5", argsB[paramRetryMaxCount])
	assert.Equal(t, "1.5", argsB[paramRetryBackOffFactor])
	assert.Equal(t, "120", argsB[paramRetryBackOffMaxDuration])
	_, hasDuration := argsB[paramRetryBackOffDuration]
	assert.False(t, hasDuration, "task B did not set a backoff duration")
}

func retryArgValues(dagTask *wfapi.DAGTask) map[string]string {
	values := map[string]string{}
	for _, param := range dagTask.Arguments.Parameters {
		if param.Value != nil {
			values[param.Name] = string(*param.Value)
		}
	}
	return values
}
