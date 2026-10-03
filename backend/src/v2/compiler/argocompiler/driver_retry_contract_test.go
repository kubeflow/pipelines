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
	"strconv"
	"strings"
	"testing"

	wfapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestCompileTaskRetryIncludesDrivers(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()
	viper.Reset()
	t.Cleanup(viper.Reset)

	for _, tc := range []struct {
		name       string
		configured bool
		count      int32
		policy     pipelinespec.PipelineTaskSpec_RetryPolicy_Policy
		argoPolicy wfapi.RetryPolicy
	}{
		{name: "unset"},
		{name: "default policy", configured: true, count: 3},
		{name: "always", configured: true, count: 3, policy: pipelinespec.PipelineTaskSpec_RetryPolicy_POLICY_ALWAYS, argoPolicy: wfapi.RetryPolicyAlways},
		{name: "on failure", configured: true, count: 3, policy: pipelinespec.PipelineTaskSpec_RetryPolicy_POLICY_ON_FAILURE, argoPolicy: wfapi.RetryPolicyOnFailure},
		{name: "on error", configured: true, count: 3, policy: pipelinespec.PipelineTaskSpec_RetryPolicy_POLICY_ON_ERROR, argoPolicy: wfapi.RetryPolicyOnError},
		{name: "on transient error", configured: true, count: 3, policy: pipelinespec.PipelineTaskSpec_RetryPolicy_POLICY_ON_TRANSIENT_ERROR, argoPolicy: wfapi.RetryPolicyOnTransientError},
		{name: "zero retries", configured: true, policy: pipelinespec.PipelineTaskSpec_RetryPolicy_POLICY_ON_FAILURE, argoPolicy: wfapi.RetryPolicyOnFailure},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var retry *pipelinespec.PipelineTaskSpec_RetryPolicy
			if tc.configured {
				retry = &pipelinespec.PipelineTaskSpec_RetryPolicy{
					MaxRetryCount:      tc.count,
					BackoffDuration:    &durationpb.Duration{Seconds: 5},
					BackoffFactor:      3,
					BackoffMaxDuration: &durationpb.Duration{Seconds: 60},
					Policy:             tc.policy,
				}
			}
			wf := compileDriverRetryContract(t, retry)
			retryParameters := []string{paramRetryMaxCount, paramRetryBackOffDuration, paramRetryBackOffFactor, paramRetryBackOffMaxDuration}
			driverPolicy := tc.argoPolicy
			if driverPolicy == "" {
				driverPolicy = wfapi.RetryPolicyAlways
			}
			if tc.configured {
				assert.Equal(t, "0", wf.Annotations[util.AnnotationKeyRetryGeneration])
			}

			for _, name := range []string{"system-container-driver", "system-dag-driver"} {
				driver := templateByName(t, wf, name)
				assert.Nil(t, driver.RetryStrategy, "driver %s must leave retry policy to the deployment", name)
				assert.NotContains(t, driver.Metadata.Annotations, util.AnnotationKeyTaskDriverRetry)
				for _, parameter := range retryParameters {
					assert.NotContains(t, parameterNames(driver.Inputs.Parameters), parameter)
				}
			}

			for _, pair := range []struct {
				template string
				task     string
				driver   string
			}{
				{tmplEntrypoint, "root", "system-dag-driver"},
				{"root", "ordinary", "system-container-driver"},
				{"root", "nested", "system-dag-driver"},
				{"comp-nested", "child", "system-container-driver"},
			} {
				tmpl := templateByName(t, wf, pair.template)
				driver := retryContractTaskByName(t, tmpl, pair.task+"-driver")
				task := retryContractTaskByName(t, tmpl, pair.task)
				assert.Equal(t, pair.task+"-driver.Succeeded", task.Depends)
				if !tc.configured || pair.template == tmplEntrypoint {
					assert.Equal(t, pair.driver, driver.Template)
					for _, parameter := range retryParameters {
						assert.NotContains(t, parameterNames(driver.Arguments.Parameters), parameter)
					}
					continue
				}
				assert.Equal(t, "retry-"+pair.driver+"-"+strings.ToLower(string(driverPolicy)), driver.Template)
				for name, value := range map[string]string{
					paramRetryMaxCount: strconv.Itoa(int(tc.count)), paramRetryBackOffDuration: "5",
					paramRetryBackOffFactor: "3", paramRetryBackOffMaxDuration: "60",
				} {
					parameter := driver.Arguments.GetParameterByName(name)
					require.NotNil(t, parameter)
					require.NotNil(t, parameter.Value)
					assert.Equal(t, value, parameter.Value.String())
				}
				driverTemplate := templateByName(t, wf, driver.Template)
				require.NotNil(t, driverTemplate.RetryStrategy)
				require.NotNil(t, driverTemplate.RetryStrategy.Limit)
				assert.Equal(t, driverPolicy, driverTemplate.RetryStrategy.RetryPolicy)
				assert.Equal(t, inputParameter(paramRetryMaxCount), driverTemplate.RetryStrategy.Limit.String())
				require.NotNil(t, driverTemplate.RetryStrategy.Backoff)
				assert.Equal(t, inputParameter(paramRetryBackOffDuration), driverTemplate.RetryStrategy.Backoff.Duration)
				assert.Equal(t, inputParameter(paramRetryBackOffFactor), driverTemplate.RetryStrategy.Backoff.Factor.String())
				assert.Equal(t, inputParameter(paramRetryBackOffMaxDuration), driverTemplate.RetryStrategy.Backoff.MaxDuration)
				assert.Equal(t, "true", driverTemplate.Metadata.Annotations[util.AnnotationKeyTaskDriverRetry])
				assert.Contains(t, driverTemplate.Container.Args, "--driver_retry_enabled=true")
				assertAdjacentArgPair(t, driverTemplate.Container.Args, "--driver_retry_attempt", "{{retries}}")
				assertAdjacentArgPair(t, driverTemplate.Container.Args, "--driver_retry_max_count", inputParameter(paramRetryMaxCount))
				assertAdjacentArgPair(t, driverTemplate.Container.Args, "--driver_retry_generation", "{{workflow.annotations."+util.AnnotationKeyRetryGeneration+"}}")
				assertRegisteredDriverArgs(t, driverTemplate.Container.Args)
			}

			// Driver retry settings do not consume or change the executor budget.
			for _, location := range []struct{ template, task string }{{"root", "ordinary"}, {"comp-nested", "child"}} {
				task := retryContractTaskByName(t, templateByName(t, wf, location.template), location.task)
				wrapper := templateByName(t, wf, task.Template)
				require.NotNil(t, wrapper.DAG)
				require.Len(t, wrapper.DAG.Tasks, 1)
				executor := templateByName(t, wf, wrapper.DAG.Tasks[0].Template)
				if !tc.configured {
					assert.Nil(t, executor.RetryStrategy)
					for _, parameter := range retryParameters {
						assert.NotContains(t, parameterNames(task.Arguments.Parameters), parameter)
					}
					continue
				}

				for name, value := range map[string]string{
					paramRetryMaxCount:           strconv.Itoa(int(tc.count)),
					paramRetryBackOffDuration:    "5",
					paramRetryBackOffFactor:      "3",
					paramRetryBackOffMaxDuration: "60",
				} {
					parameter := task.Arguments.GetParameterByName(name)
					require.NotNil(t, parameter)
					require.NotNil(t, parameter.Value)
					assert.Equal(t, value, parameter.Value.String())
				}
				require.NotNil(t, executor.RetryStrategy)
				require.NotNil(t, executor.RetryStrategy.Limit)
				assert.Equal(t, "{{inputs.parameters.retry-max-count}}", executor.RetryStrategy.Limit.String())
				assert.Equal(t, tc.argoPolicy, executor.RetryStrategy.RetryPolicy)
				require.NotNil(t, executor.RetryStrategy.Backoff)
				assert.Equal(t, "{{inputs.parameters.retry-backoff-duration}}", executor.RetryStrategy.Backoff.Duration)
				require.NotNil(t, executor.RetryStrategy.Backoff.Factor)
				assert.Equal(t, "{{inputs.parameters.retry-backoff-factor}}", executor.RetryStrategy.Backoff.Factor.String())
				assert.Equal(t, "{{inputs.parameters.retry-backoff-max-duration}}", executor.RetryStrategy.Backoff.MaxDuration)
				require.NotNil(t, wrapper.RetryStrategy)
				require.NotNil(t, wrapper.RetryStrategy.Limit)
				assert.Equal(t, "0", wrapper.RetryStrategy.Limit.String())
			}
		})
	}
}

func TestCompileDriverRetryPreservesTaskOverrides(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()
	viper.Reset()
	t.Cleanup(viper.Reset)

	wf := compileDriverRetryContract(t, &pipelinespec.PipelineTaskSpec_RetryPolicy{MaxRetryCount: 3}, func(spec *pipelinespec.PipelineSpec) {
		spec.Components["comp-nested"].GetDag().Tasks["child"].RetryPolicy = &pipelinespec.PipelineTaskSpec_RetryPolicy{
			Policy: pipelinespec.PipelineTaskSpec_RetryPolicy_POLICY_ON_FAILURE,
		}
		spec.Root.GetDag().Tasks["unconfigured"] = &pipelinespec.PipelineTaskSpec{
			ComponentRef: &pipelinespec.ComponentRef{Name: "comp-container"},
		}
		spec.Root.GetDag().Tasks["other"] = &pipelinespec.PipelineTaskSpec{
			ComponentRef: &pipelinespec.ComponentRef{Name: "comp-container"},
			RetryPolicy:  &pipelinespec.PipelineTaskSpec_RetryPolicy{MaxRetryCount: 7},
		}
	})
	root := templateByName(t, wf, "root")
	ordinary := retryContractTaskByName(t, root, "ordinary-driver")
	other := retryContractTaskByName(t, root, "other-driver")
	assert.Equal(t, ordinary.Template, other.Template)
	assert.Equal(t, "3", ordinary.Arguments.GetParameterByName(paramRetryMaxCount).Value.String())
	assert.Equal(t, "7", other.Arguments.GetParameterByName(paramRetryMaxCount).Value.String())
	assert.Equal(t, "2", ordinary.Arguments.GetParameterByName(paramRetryBackOffFactor).Value.String())
	assert.Equal(t, "system-container-driver", retryContractTaskByName(t, root, "unconfigured-driver").Template)
	child := retryContractTaskByName(t, templateByName(t, wf, "comp-nested"), "child-driver")
	assert.Equal(t, "retry-system-container-driver-onfailure", child.Template)
	assert.Equal(t, "0", child.Arguments.GetParameterByName(paramRetryMaxCount).Value.String())
	ordinaryTemplate := templateByName(t, wf, ordinary.Template)
	for name, expected := range map[string]string{paramRetryBackOffDuration: "0", paramRetryBackOffMaxDuration: "3600"} {
		parameter := ordinaryTemplate.Inputs.GetParameterByName(name)
		require.NotNil(t, parameter)
		require.NotNil(t, parameter.Default)
		assert.Equal(t, expected, parameter.Default.String())
	}
}

func TestCompileDriverRetryInsideLoop(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()
	viper.Reset()
	t.Cleanup(viper.Reset)

	wf := compileDriverRetryContract(t, &pipelinespec.PipelineTaskSpec_RetryPolicy{MaxRetryCount: 2}, func(spec *pipelinespec.PipelineSpec) {
		spec.Root.GetDag().Tasks["nested"].Iterator = &pipelinespec.PipelineTaskSpec_ParameterIterator{
			ParameterIterator: &pipelinespec.ParameterIteratorSpec{
				ItemInput: "item",
				Items: &pipelinespec.ParameterIteratorSpec_ItemsSpec{
					Kind: &pipelinespec.ParameterIteratorSpec_ItemsSpec_Raw{Raw: "[1, 2]"},
				},
			},
		}
		spec.Components["comp-nested"].GetDag().Tasks["inner"] = &pipelinespec.PipelineTaskSpec{
			ComponentRef: &pipelinespec.ComponentRef{Name: "comp-inner"},
		}
		spec.Components["comp-inner"] = &pipelinespec.ComponentSpec{
			Implementation: &pipelinespec.ComponentSpec_Dag{Dag: &pipelinespec.DagSpec{Tasks: map[string]*pipelinespec.PipelineTaskSpec{
				"leaf": {ComponentRef: &pipelinespec.ComponentRef{Name: "comp-container"}},
			}}},
		}
	})
	loopTask := retryContractTaskByName(t, templateByName(t, wf, "root"), "nested")
	loop := templateByName(t, wf, loopTask.Template)
	loopDriver := retryContractTaskByName(t, loop, "iteration-driver")
	assert.Equal(t, "retry-system-dag-driver-always", loopDriver.Template)
	assert.Equal(t, "2", loopDriver.Arguments.GetParameterByName(paramRetryMaxCount).Value.String())
	for _, location := range []struct{ template, task string }{
		{"comp-nested", "child-driver"}, {"comp-nested", "inner-driver"}, {"comp-inner", "leaf-driver"},
	} {
		driver := retryContractTaskByName(t, templateByName(t, wf, location.template), location.task)
		assert.Equal(t, "2", driver.Arguments.GetParameterByName(paramRetryMaxCount).Value.String())
		parameter := driver.Arguments.GetParameterByName(paramIterationIndex)
		require.NotNil(t, parameter)
		require.NotNil(t, parameter.Value)
		assert.Equal(t, inputParameter(paramIterationIndex), parameter.Value.String())
	}
}

func TestCompileDriverRetryExcludesPVCOperations(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()
	viper.Reset()
	t.Cleanup(viper.Reset)

	for image := range dummyImages {
		t.Run(image, func(t *testing.T) {
			wf := compileDriverRetryContract(t, &pipelinespec.PipelineTaskSpec_RetryPolicy{MaxRetryCount: 3}, func(spec *pipelinespec.PipelineSpec) {
				container := spec.DeploymentSpec.Fields["executors"].GetStructValue().Fields["exec-container"].GetStructValue().Fields["container"].GetStructValue()
				container.Fields["image"] = structpb.NewStringValue(image)
			})
			driver := retryContractTaskByName(t, templateByName(t, wf, "root"), "ordinary")
			assert.Equal(t, "system-container-driver", driver.Template)
			assert.Nil(t, driver.Arguments.GetParameterByName(paramRetryMaxCount))
		})
	}
}

func TestCompileRejectsInvalidDriverRetryPolicy(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()
	viper.Reset()
	t.Cleanup(viper.Reset)
	for _, tc := range []struct {
		name   string
		policy *pipelinespec.PipelineTaskSpec_RetryPolicy
		err    string
	}{
		{"negative count", &pipelinespec.PipelineTaskSpec_RetryPolicy{MaxRetryCount: -1}, "retry max count must be non-negative"},
		{"unknown policy", &pipelinespec.PipelineTaskSpec_RetryPolicy{Policy: 99}, "unsupported retry policy 99"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			job := driverRetryContractJob(t, tc.policy)
			_, err := Compile(job, nil, nil)
			require.ErrorContains(t, err, tc.err)
		})
	}
}

func compileDriverRetryContract(t *testing.T, retry *pipelinespec.PipelineTaskSpec_RetryPolicy, configure ...func(*pipelinespec.PipelineSpec)) *wfapi.Workflow {
	t.Helper()
	job := driverRetryContractJob(t, retry, configure...)
	wf, err := Compile(job, nil, nil)
	require.NoError(t, err)
	require.NoError(t, util.NewWorkflow(wf.DeepCopy()).Validate(true, false))
	return wf
}

func driverRetryContractJob(t *testing.T, retry *pipelinespec.PipelineTaskSpec_RetryPolicy, configure ...func(*pipelinespec.PipelineSpec)) *pipelinespec.PipelineJob {
	t.Helper()
	const pipeline = `{
		"pipelineInfo": {"name": "driver-retry-contract"},
		"root": {"dag": {"tasks": {
			"ordinary": {"componentRef": {"name": "comp-container"}},
			"nested": {"componentRef": {"name": "comp-nested"}}
		}}},
		"components": {
			"comp-container": {"executorLabel": "exec-container"},
			"comp-nested": {"dag": {"tasks": {
				"child": {"componentRef": {"name": "comp-container"}}
			}}}
		},
		"deploymentSpec": {"executors": {
			"exec-container": {"container": {"image": "test-image"}}
		}}
	}`
	spec := &pipelinespec.PipelineSpec{}
	require.NoError(t, protojson.Unmarshal([]byte(pipeline), spec))
	spec.Root.GetDag().Tasks["ordinary"].RetryPolicy = retry
	spec.Root.GetDag().Tasks["nested"].RetryPolicy = retry
	for _, customize := range configure {
		customize(spec)
	}
	encodedSpec, err := protojson.Marshal(spec)
	require.NoError(t, err)
	job := &pipelinespec.PipelineJob{PipelineSpec: &structpb.Struct{}}
	require.NoError(t, protojson.Unmarshal(encodedSpec, job.PipelineSpec))
	return job
}

func retryContractTaskByName(t *testing.T, template wfapi.Template, name string) wfapi.DAGTask {
	t.Helper()
	require.NotNil(t, template.DAG)
	for _, task := range template.DAG.Tasks {
		if task.Name == name {
			return task
		}
	}
	t.Fatalf("task %q not found in template %q", name, template.Name)
	return wfapi.DAGTask{}
}
