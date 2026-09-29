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
	"testing"

	wfapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestCompileTaskRetryExcludesDrivers(t *testing.T) {
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

			for _, name := range []string{"system-container-driver", "system-dag-driver"} {
				driver := templateByName(t, wf, name)
				assert.Nil(t, driver.RetryStrategy, "driver %s must leave retry policy to the deployment", name)
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
				assert.Equal(t, pair.driver, driver.Template)
				assert.Equal(t, pair.task+"-driver.Succeeded", task.Depends)
				for _, parameter := range retryParameters {
					assert.NotContains(t, parameterNames(driver.Arguments.Parameters), parameter)
				}
			}

			// A nested pipeline's retry policy reaches its child executor, while
			// both the nested DAG driver and the child's container driver stay separate.
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

func compileDriverRetryContract(t *testing.T, retry *pipelinespec.PipelineTaskSpec_RetryPolicy) *wfapi.Workflow {
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
	encodedSpec, err := protojson.Marshal(spec)
	require.NoError(t, err)
	job := &pipelinespec.PipelineJob{PipelineSpec: &structpb.Struct{}}
	require.NoError(t, protojson.Unmarshal(encodedSpec, job.PipelineSpec))
	wf, err := Compile(job, nil, nil)
	require.NoError(t, err)
	return wf
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
