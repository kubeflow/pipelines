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

package resolver

import (
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	api "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/driver/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

func exitStatusOptions() (common.Options, *pipelinespec.TaskInputsSpec_InputParameterSpec) {
	return common.Options{
		ParentTask: &api.PipelineTask{TaskId: "parent", Name: "parent"},
		Run:        &api.Run{RunId: "run", State: api.RuntimeState_RUNNING},
		RunName:    "workflow", IterationIndex: -1,
		ExitTaskName: "producer", ExitTaskStatus: "Failed",
		Task: &pipelinespec.PipelineTaskSpec{
			TaskInfo: &pipelinespec.PipelineTaskInfo{Name: "cleanup"}, DependentTasks: []string{"producer"},
			TriggerPolicy: &pipelinespec.PipelineTaskSpec_TriggerPolicy{Strategy: pipelinespec.PipelineTaskSpec_TriggerPolicy_ALL_UPSTREAM_TASKS_COMPLETED},
		},
	}, &pipelinespec.TaskInputsSpec_InputParameterSpec{Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus_{TaskFinalStatus: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus{ProducerTask: "producer"}}}
}

func TestExitTaskFinalStatusUsesCompletedArgoPhase(t *testing.T) {
	for _, tc := range []struct {
		name, phase string
		state, want api.PipelineTask_TaskState
		missing     bool
	}{
		{name: "exhausted count", phase: "Failed", state: api.PipelineTask_RUNNING, want: api.PipelineTask_FAILED},
		{name: "policy stops first attempt", phase: "Failed", state: api.PipelineTask_RUNNING, want: api.PipelineTask_FAILED},
		{name: "deadline or deleted pod", phase: "Error", state: api.PipelineTask_RUNNING, want: api.PipelineTask_FAILED},
		{name: "premature cached aggregate", phase: "Failed", state: api.PipelineTask_CACHED, want: api.PipelineTask_FAILED},
		{name: "failure before task creation", phase: "Failed", want: api.PipelineTask_FAILED, missing: true},
		{name: "success before native finalization", phase: "Succeeded", state: api.PipelineTask_RUNNING, want: api.PipelineTask_RUNNING},
		{name: "native failure despite controller success", phase: "Succeeded", state: api.PipelineTask_FAILED, want: api.PipelineTask_FAILED},
		{name: "native success", phase: "Succeeded", state: api.PipelineTask_SUCCEEDED, want: api.PipelineTask_SUCCEEDED},
		{name: "cached success", phase: "Succeeded", state: api.PipelineTask_CACHED, want: api.PipelineTask_CACHED},
		{name: "skipped success", phase: "Succeeded", state: api.PipelineTask_SKIPPED, want: api.PipelineTask_SKIPPED},
		{name: "skipped", phase: "Skipped", state: api.PipelineTask_RUNNING, want: api.PipelineTask_SKIPPED},
		{name: "omitted", phase: "Omitted", state: api.PipelineTask_RUNNING, want: api.PipelineTask_SKIPPED},
		{name: "failure before skip", phase: "Skipped", state: api.PipelineTask_FAILED, want: api.PipelineTask_FAILED},
		{name: "failure before omission", phase: "Omitted", state: api.PipelineTask_FAILED, want: api.PipelineTask_FAILED},
	} {
		t.Run(tc.name, func(t *testing.T) {
			opts, spec := exitStatusOptions()
			opts.ExitTaskStatus = tc.phase
			if !tc.missing {
				opts.Run.Tasks = []*api.PipelineTask{{TaskId: "producer-id", Name: "producer", ParentTaskId: proto.String("parent"), Type: api.PipelineTask_DAG, State: tc.state}}
				if tc.state == api.PipelineTask_FAILED {
					opts.Run.Tasks[0].StatusMetadata = &api.PipelineTask_StatusMetadata{Message: "native failure"}
				}
			}
			snapshot := proto.Clone(opts.Run).(*api.Run)
			value, err := resolveTaskFinalStatus(opts, spec)
			require.NoError(t, err)
			fields := value.GetStructValue().GetFields()
			assert.Equal(t, tc.want.String(), fields["state"].GetStringValue())
			assert.Equal(t, "producer", fields["pipelineTaskName"].GetStringValue())
			code := codes.OK
			if tc.want == api.PipelineTask_FAILED {
				code = codes.Unknown
				assert.NotEmpty(t, fields["error"].GetStructValue().GetFields()["message"].GetStringValue())
				if tc.state == api.PipelineTask_FAILED {
					assert.Equal(t, "native failure", fields["error"].GetStructValue().GetFields()["message"].GetStringValue())
				}
			}
			assert.EqualValues(t, code, fields["error"].GetStructValue().GetFields()["code"].GetNumberValue())
			assert.True(t, proto.Equal(snapshot, opts.Run), "exit status resolution must not reopen or finalize persisted task snapshots")
		})
	}
}

func TestExitTaskFinalStatusScopesFailureToProducerIteration(t *testing.T) {
	opts, spec := exitStatusOptions()
	opts.IterationIndex = 1
	producer := &api.PipelineTask{TaskId: "producer-1", Name: "producer", ParentTaskId: proto.String("parent"), Type: api.PipelineTask_DAG, State: api.PipelineTask_RUNNING, TypeAttributes: &api.PipelineTask_TypeAttributes{IterationIndex: proto.Int64(1)}}
	other := proto.Clone(producer).(*api.PipelineTask)
	other.TaskId = "producer-0"
	other.TypeAttributes.IterationIndex = proto.Int64(0)
	other.StatusMetadata = &api.PipelineTask_StatusMetadata{Message: "wrong iteration"}
	otherParent := proto.Clone(producer).(*api.PipelineTask)
	otherParent.TaskId = "other-parent-producer"
	otherParent.ParentTaskId = proto.String("another-parent")
	otherParent.StatusMetadata = &api.PipelineTask_StatusMetadata{Message: "wrong parent"}
	child := &api.PipelineTask{TaskId: "child", Name: "child", ParentTaskId: proto.String(producer.TaskId), State: api.PipelineTask_RUNNING, Type: api.PipelineTask_RUNTIME, StatusMetadata: &api.PipelineTask_StatusMetadata{Message: "metadata unavailable", CustomProperties: map[string]*structpb.Value{util.DriverRetryGenerationKey: structpb.NewStringValue("0")}}}
	opts.Run.Tasks = []*api.PipelineTask{other, otherParent, producer, child}
	value, err := resolveTaskFinalStatus(opts, spec)
	require.NoError(t, err)
	fields := value.GetStructValue().GetFields()
	assert.Equal(t, "FAILED", fields["state"].GetStringValue())
	assert.Equal(t, "metadata unavailable", fields["error"].GetStructValue().GetFields()["message"].GetStringValue())
	assert.Equal(t, api.PipelineTask_RUNNING, other.State)
	assert.Equal(t, api.PipelineTask_RUNNING, child.State, "resolution is read-only while workflow exit hooks run")
	opts.ExitTaskName = "other-dependency" // A hook fired for another dependency must not override this producer.
	value, err = resolveTaskFinalStatus(opts, spec)
	require.NoError(t, err)
	assert.Equal(t, "RUNNING", value.GetStructValue().GetFields()["state"].GetStringValue())
}

func TestExitTaskFinalStatusFallsBackToNativeState(t *testing.T) {
	for _, phase := range []string{"", "Pending", "Running", "Unknown"} {
		for _, state := range []api.PipelineTask_TaskState{api.PipelineTask_RUNNING, api.PipelineTask_FAILED, api.PipelineTask_SUCCEEDED, api.PipelineTask_CACHED, api.PipelineTask_SKIPPED} {
			t.Run(phase+"/"+state.String(), func(t *testing.T) {
				opts, spec := exitStatusOptions()
				opts.ExitTaskStatus = phase
				opts.Run.Tasks = []*api.PipelineTask{{TaskId: "producer-id", Name: "producer", ParentTaskId: proto.String("parent"), State: state, Type: api.PipelineTask_RUNTIME, StatusMetadata: &api.PipelineTask_StatusMetadata{Message: "native status message"}}}
				snapshot := proto.Clone(opts.Run)
				value, err := resolveTaskFinalStatus(opts, spec)
				require.NoError(t, err)
				fields := value.GetStructValue().GetFields()
				assert.Equal(t, state.String(), fields["state"].GetStringValue())
				assert.Equal(t, "native status message", fields["error"].GetStructValue().GetFields()["message"].GetStringValue())
				assert.True(t, proto.Equal(snapshot, opts.Run))
			})
		}
	}
}
