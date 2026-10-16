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

package driver

import (
	"context"
	"fmt"
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	api "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	clientmanager "github.com/kubeflow/pipelines/backend/src/v2/client_manager"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestDriverRetryCacheHitPersistsResolvedInputs(t *testing.T) {
	for _, loseCachedResponse := range []bool{false, true} {
		t.Run(fmt.Sprintf("lost_cached_response=%t", loseCachedResponse), func(t *testing.T) {
			tc := NewTestContextWithRootExecuted(t, &pipelinespec.PipelineJob_RuntimeConfig{
				ParameterValues: map[string]*structpb.Value{
					"name_in":      structpb.NewStringValue("requested-name"),
					"number_in":    structpb.NewNumberValue(9),
					"threshold_in": structpb.NewNumberValue(0.25),
					"active_in":    structpb.NewBoolValue(false),
				},
			}, "test_data/componentInput.yaml")
			require.NoError(t, tc.Push("process-inputs"))
			opts := tc.setupContainerOptions(tc.RootTask, tc.GetLast().GetTaskSpec(), nil)
			opts.DriverRetryEnabled = true
			opts.DriverRetryMaxCount = 2
			opts.Task = proto.Clone(opts.Task).(*pipelinespec.PipelineTaskSpec)
			opts.Task.CachingOptions = &pipelinespec.PipelineTaskSpec_CachingOptions{EnableCache: true}
			faults := &driverRetryFaultAPI{
				MockAPI:    tc.MockAPI,
				loseCached: loseCachedResponse,
				cacheTask:  &api.PipelineTask{Outputs: &api.PipelineTask_InputOutputs{}},
			}
			tc.ClientManager = clientmanager.NewFakeClientManager(tc.ClientManager.K8sClient(), faults)

			execution, err := Container(context.Background(), opts, tc.ClientManager)
			if loseCachedResponse {
				require.ErrorContains(t, err, "lost cached task response")
				faults.cacheErr = fmt.Errorf("cache unavailable during recovery")
				opts.DriverRetryAttempt++
				opts.PodUID = "recovered-cache-driver"
				execution, err = Container(context.Background(), opts, tc.ClientManager)
			}
			require.NoError(t, err)
			require.True(t, *execution.Cached)
			require.Equal(t, 1, faults.cacheCalls)
			verifyInputs := func(task *api.PipelineTask) {
				t.Helper()
				expected := map[string]struct {
					value  interface{}
					ioType api.IOType
				}{
					"name":             {"requested-name", api.IOType_COMPONENT_INPUT},
					"number":           {float64(9), api.IOType_COMPONENT_INPUT},
					"threshold":        {0.25, api.IOType_COMPONENT_INPUT},
					"active":           {false, api.IOType_COMPONENT_INPUT},
					"a_runtime_string": {"foo", api.IOType_RUNTIME_VALUE_INPUT},
					"a_runtime_number": {float64(10), api.IOType_RUNTIME_VALUE_INPUT},
					"a_runtime_bool":   {true, api.IOType_RUNTIME_VALUE_INPUT},
				}
				require.Len(t, task.GetInputs().GetParameters(), len(expected))
				for _, parameter := range task.GetInputs().GetParameters() {
					want, ok := expected[parameter.GetParameterKey()]
					require.True(t, ok, "unexpected input %q", parameter.GetParameterKey())
					assert.Equal(t, want.value, parameter.GetValue().AsInterface())
					assert.Equal(t, want.ioType, parameter.GetType())
					assert.Equal(t, tc.RootTask.GetName(), parameter.GetProducer().GetTaskName())
					assert.True(t, proto.Equal(parameter.GetValue(), execution.ExecutorInput.GetInputs().GetParameterValues()[parameter.GetParameterKey()]))
					delete(expected, parameter.GetParameterKey())
				}
				assert.Empty(t, expected)
				assert.Equal(t, api.PipelineTask_CACHED, task.GetState())
				assert.NotEmpty(t, task.GetCacheFingerprint())
				assert.Equal(t, "0", task.GetStatusMetadata().GetCustomProperties()[util.DriverRetryGenerationKey].GetStringValue())
				assert.NotEmpty(t, task.GetStatusMetadata().GetCustomProperties()[driverCachedOutputsKey].GetStringValue())
				assert.NotEmpty(t, task.GetStatusMetadata().GetCustomProperties()[driverCheckpointKey].GetStringValue())
			}
			task, err := tc.MockAPI.GetTask(context.Background(), &api.GetTaskRequest{RunId: opts.Run.GetRunId(), TaskId: execution.TaskID})
			require.NoError(t, err)
			verifyInputs(task)
			assert.Len(t, task.GetPods(), opts.DriverRetryAttempt+1)

			// A successful handoff replay must retain the same resolved provenance.
			opts.DriverRetryAttempt++
			opts.PodUID = "checkpoint-cache-driver"
			replay, err := Container(context.Background(), opts, tc.ClientManager)
			require.NoError(t, err)
			assert.Equal(t, execution.TaskID, replay.TaskID)
			task, err = tc.MockAPI.GetTask(context.Background(), &api.GetTaskRequest{RunId: opts.Run.GetRunId(), TaskId: replay.TaskID})
			require.NoError(t, err)
			verifyInputs(task)
			assert.Len(t, task.GetPods(), opts.DriverRetryAttempt+1)
		})
	}
}
