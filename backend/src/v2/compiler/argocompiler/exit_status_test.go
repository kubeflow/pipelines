// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package argocompiler

import (
	"testing"

	wfapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompileExitDriverReceivesCompletedProducerStatus(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()
	viper.Reset()
	t.Cleanup(viper.Reset)

	for _, component := range []string{"comp-container", "comp-exit-dag"} {
		for _, policy := range []struct {
			name  string
			retry *pipelinespec.PipelineTaskSpec_RetryPolicy
			argo  wfapi.RetryPolicy
		}{
			{name: "unset"},
			{name: "default", retry: &pipelinespec.PipelineTaskSpec_RetryPolicy{MaxRetryCount: 2}, argo: wfapi.RetryPolicyAlways},
			{name: "on error", retry: &pipelinespec.PipelineTaskSpec_RetryPolicy{MaxRetryCount: 2, Policy: pipelinespec.PipelineTaskSpec_RetryPolicy_POLICY_ON_ERROR}, argo: wfapi.RetryPolicyOnError},
		} {
			t.Run(component+"/"+policy.name, func(t *testing.T) {
				wf := compileDriverRetryContract(t, nil, func(spec *pipelinespec.PipelineSpec) {
					spec.Components["comp-exit-dag"] = &pipelinespec.ComponentSpec{Implementation: &pipelinespec.ComponentSpec_Dag{Dag: &pipelinespec.DagSpec{Tasks: map[string]*pipelinespec.PipelineTaskSpec{
						"cleanup": {ComponentRef: &pipelinespec.ComponentRef{Name: "comp-container"}},
					}}}}
					spec.Root.GetDag().Tasks["notify"] = exitStatusTask(component, "nested", "ordinary")
					spec.Root.GetDag().Tasks["notify"].RetryPolicy = policy.retry
				})
				root := templateByName(t, wf, "root")
				for _, producer := range []string{"nested", "ordinary"} {
					task := retryContractTaskByName(t, root, producer)
					hook, ok := task.Hooks[wfapi.ExitLifecycleEvent]
					require.True(t, ok)
					assert.Equal(t, producer, exitStatusParameter(t, hook.Arguments.Parameters, paramExitTaskName))
					assert.Equal(t, "{{tasks."+producer+".status}}", exitStatusParameter(t, hook.Arguments.Parameters, paramExitTaskStatus))
					assert.Equal(t, "-1", exitStatusParameter(t, hook.Arguments.Parameters, paramIterationIndex))
					exitTemplate := templateByName(t, wf, hook.Template)
					driver := retryContractTaskByName(t, exitTemplate, "notify-driver")
					assert.Empty(t, driver.Depends)
					assert.Equal(t, inputParameter(paramParentDagTaskID), exitStatusParameter(t, driver.Arguments.Parameters, paramParentDagTaskID))
					for _, param := range []string{paramExitTaskName, paramExitTaskStatus, paramIterationIndex} {
						assert.Equal(t, inputParameter(param), exitStatusParameter(t, driver.Arguments.Parameters, param))
					}
					driverTemplate := templateByName(t, wf, driver.Template)
					assertAdjacentArgPair(t, driverTemplate.Container.Args, "--exit_task_name", inputParameter(paramExitTaskName))
					assertAdjacentArgPair(t, driverTemplate.Container.Args, "--exit_task_status", inputParameter(paramExitTaskStatus))
					assertRegisteredDriverArgs(t, driverTemplate.Container.Args)
					if policy.retry == nil {
						assert.Nil(t, driverTemplate.RetryStrategy)
						assert.NotContains(t, driverTemplate.Metadata.Annotations, util.AnnotationKeyTaskDriverRetry)
					} else {
						require.NotNil(t, driverTemplate.RetryStrategy)
						assert.Equal(t, policy.argo, driverTemplate.RetryStrategy.RetryPolicy)
						assert.Equal(t, "true", driverTemplate.Metadata.Annotations[util.AnnotationKeyTaskDriverRetry])
						assert.Equal(t, "2", exitStatusParameter(t, driver.Arguments.Parameters, paramRetryMaxCount))
						assertDriverFinalizer(t, wf, driver)
					}
				}
				for _, base := range []string{"system-container-driver", "system-dag-driver"} {
					template := templateByName(t, wf, base)
					assert.NotContains(t, template.Container.Args, "--exit_task_status")
					assert.Nil(t, template.Inputs.GetParameterByName(paramExitTaskStatus))
				}
			})
		}
	}
}

func TestCompileExitDriverPreservesNestedLoopIteration(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()
	viper.Reset()
	t.Cleanup(viper.Reset)

	wf := compileDriverRetryContract(t, &pipelinespec.PipelineTaskSpec_RetryPolicy{MaxRetryCount: 2}, func(spec *pipelinespec.PipelineSpec) {
		spec.Root.GetDag().Tasks["nested"].Iterator = &pipelinespec.PipelineTaskSpec_ParameterIterator{
			ParameterIterator: &pipelinespec.ParameterIteratorSpec{
				ItemInput: "item",
				Items:     &pipelinespec.ParameterIteratorSpec_ItemsSpec{Kind: &pipelinespec.ParameterIteratorSpec_ItemsSpec_Raw{Raw: "[1, 2]"}},
			},
		}
		spec.Components["comp-nested"].GetDag().Tasks["notify"] = exitStatusTask("comp-container", "child")
		spec.Components["comp-nested"].GetDag().Tasks["inner"] = &pipelinespec.PipelineTaskSpec{ComponentRef: &pipelinespec.ComponentRef{Name: "comp-inner"}}
		spec.Components["comp-inner"] = &pipelinespec.ComponentSpec{Implementation: &pipelinespec.ComponentSpec_Dag{Dag: &pipelinespec.DagSpec{Tasks: map[string]*pipelinespec.PipelineTaskSpec{
			"child":  {ComponentRef: &pipelinespec.ComponentRef{Name: "comp-container"}},
			"notify": exitStatusTask("comp-container", "child"),
		}}}}
	})
	for _, component := range []string{"comp-nested", "comp-inner"} {
		body := templateByName(t, wf, component)
		producer := retryContractTaskByName(t, body, "child")
		hook, ok := producer.Hooks[wfapi.ExitLifecycleEvent]
		require.True(t, ok)
		assert.Equal(t, inputParameter(paramIterationIndex), exitStatusParameter(t, hook.Arguments.Parameters, paramIterationIndex))
		assert.Equal(t, inputParameter(paramParentDagTaskID), exitStatusParameter(t, hook.Arguments.Parameters, paramParentDagTaskID))
		exitTemplate := templateByName(t, wf, hook.Template)
		driver := retryContractTaskByName(t, exitTemplate, "notify-driver")
		assert.Equal(t, inputParameter(paramIterationIndex), exitStatusParameter(t, driver.Arguments.Parameters, paramIterationIndex))
		assertDriverFinalizer(t, wf, driver)
	}
}

func exitStatusTask(component string, dependencies ...string) *pipelinespec.PipelineTaskSpec {
	return &pipelinespec.PipelineTaskSpec{
		ComponentRef:   &pipelinespec.ComponentRef{Name: component},
		DependentTasks: dependencies,
		TriggerPolicy:  &pipelinespec.PipelineTaskSpec_TriggerPolicy{Strategy: pipelinespec.PipelineTaskSpec_TriggerPolicy_ALL_UPSTREAM_TASKS_COMPLETED},
	}
}

func exitStatusParameter(t *testing.T, parameters []wfapi.Parameter, name string) string {
	t.Helper()
	for _, parameter := range parameters {
		if parameter.Name == name {
			require.NotNil(t, parameter.Value)
			return parameter.Value.String()
		}
	}
	t.Fatalf("parameter %s missing", name)
	return ""
}
