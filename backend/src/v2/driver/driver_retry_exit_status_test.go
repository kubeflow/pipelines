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
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	api "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
)

func TestDriverRetryExitHandlerReceivesFailureBeforeWorkflowEnds(t *testing.T) {
	for _, test := range []struct {
		name, phase string
		attempt     int
	}{
		{"count exhausted", "Failed", 2},
		{"policy stopped retries", "Failed", 0},
		{"controller error", "Error", 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			tc := NewTestContextWithRootExecuted(t, &pipelinespec.PipelineJob_RuntimeConfig{}, "test_data/pipeline_with_input_status_state.yaml")
			_, parent := tc.RunDagDriver("exit-handler-1", tc.RootTask)
			require.NoError(t, tc.Push("some-task"))
			opts := tc.setupContainerOptions(parent, withBrokenParameterIterator(tc.GetLast().GetTaskSpec()), nil)
			opts.DriverRetryEnabled = true
			opts.DriverRetryMaxCount = 2
			opts.DriverRetryAttempt = test.attempt
			_, err := Container(context.Background(), opts, tc.ClientManager)
			require.ErrorContains(t, err, "error unmarshall raw string")
			_, ok := tc.Pop()
			require.True(t, ok)
			tc.ExitDag()
			tc.RefreshRun()
			currentParent, err := tc.MockAPI.GetTask(context.Background(), &api.GetTaskRequest{RunId: tc.Run.RunId, TaskId: parent.TaskId})
			require.NoError(t, err)
			if test.attempt == opts.DriverRetryMaxCount {
				require.Equal(t, api.PipelineTask_FAILED, currentParent.State)
			} else {
				require.Equal(t, api.PipelineTask_RUNNING, currentParent.State)
			}
			require.Equal(t, api.RuntimeState_RUNNING, tc.Run.State, "exit hooks run before the workflow terminal report")

			require.NoError(t, tc.Push("echo-state"))
			exitOpts := tc.setupContainerOptions(tc.RootTask, tc.GetLast().GetTaskSpec(), nil)
			exitOpts.CacheDisabled = true
			exitOpts.ExitTaskName = "exit-handler-1"
			exitOpts.ExitTaskStatus = test.phase
			// The cleanup task needs the final status even without its own retry policy.
			require.False(t, exitOpts.DriverRetryEnabled)
			execution, err := Container(context.Background(), exitOpts, tc.ClientManager)
			require.NoError(t, err)
			status := execution.ExecutorInput.GetInputs().GetParameterValues()["status"].GetStructValue().GetFields()
			assert.Equal(t, "FAILED", status["state"].GetStringValue())
			assert.Equal(t, "exit-handler-1", status["pipelineTaskName"].GetStringValue())
			assert.EqualValues(t, codes.Unknown, status["error"].GetStructValue().GetFields()["code"].GetNumberValue())
			assert.Contains(t, status["error"].GetStructValue().GetFields()["message"].GetStringValue(), "error unmarshall raw string")
			task, err := tc.MockAPI.GetTask(context.Background(), &api.GetTaskRequest{RunId: tc.Run.RunId, TaskId: execution.TaskID})
			require.NoError(t, err)
			require.Len(t, task.GetInputs().GetParameters(), 1)
			assert.Equal(t, "FAILED", task.GetInputs().GetParameters()[0].GetValue().GetStructValue().GetFields()["state"].GetStringValue())
		})
	}
}
