// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package resource

import (
	"context"
	"testing"
	"time"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func reconcileTestTask(t *testing.T, state apiv2.PipelineTask_TaskState, taskType apiv2.PipelineTask_TaskType, pods ...*apiv2.PipelineTask_TaskPod) *model.Task {
	t.Helper()
	podSlice, err := model.ProtoSliceToJSONSlice(pods)
	require.NoError(t, err)
	return &model.Task{Name: "oom", Type: model.TaskType(taskType), State: model.TaskStatus(state), Pods: podSlice}
}

func executorPod(name string) *apiv2.PipelineTask_TaskPod {
	return &apiv2.PipelineTask_TaskPod{Name: name, Type: apiv2.PipelineTask_EXECUTOR}
}

func driverPod(name string) *apiv2.PipelineTask_TaskPod {
	return &apiv2.PipelineTask_TaskPod{Name: name, Type: apiv2.PipelineTask_DRIVER}
}

// retryNodes builds the Argo layout of one task: a Retry group whose children are the attempts.
func retryNodes(retryState string, attempts ...util.NodeStatus) map[string]util.NodeStatus {
	nodes := map[string]util.NodeStatus{}
	retry := util.NodeStatus{ID: "retry", Type: "Retry", State: retryState}
	for i, attempt := range attempts {
		key := "attempt-" + string(rune('0'+i))
		attempt.Type = "Pod"
		nodes[key] = attempt
		retry.Children = append(retry.Children, key)
	}
	nodes["retry"] = retry
	return nodes
}

func TestExecutorFailureForTask(t *testing.T) {
	running := apiv2.PipelineTask_RUNNING
	oomMsg := "main: OOMKilled (exit code 137)"

	tests := []struct {
		name     string
		task     *model.Task
		nodes    map[string]util.NodeStatus
		resolved map[string]string
		want     bool
	}{
		{
			name: "executor pod failed and the retry group gave up",
			task: reconcileTestTask(t, running, apiv2.PipelineTask_RUNTIME, executorPod("p1")),
			nodes: retryNodes("Failed",
				util.NodeStatus{ID: "p1", State: "Failed", CreateTime: 10, FinishTime: 20, Message: oomMsg}),
			want: true,
		},
		{
			name: "executor pod errored without a retry group",
			task: reconcileTestTask(t, running, apiv2.PipelineTask_RUNTIME, executorPod("p1")),
			nodes: map[string]util.NodeStatus{
				"p1": {ID: "p1", Type: "Pod", State: "Error", CreateTime: 10, FinishTime: 20},
			},
			want: true,
		},
		{
			name: "only the last attempt counts when an older attempt failed and the newest is still running",
			task: reconcileTestTask(t, running, apiv2.PipelineTask_RUNTIME, executorPod("p1"), executorPod("p2")),
			nodes: retryNodes("Running",
				util.NodeStatus{ID: "p1", State: "Failed", CreateTime: 10, FinishTime: 20, Message: oomMsg},
				util.NodeStatus{ID: "p2", State: "Running", CreateTime: 30}),
			want: false,
		},
		{
			name: "recovered retry: older failure, newest attempt succeeded",
			task: reconcileTestTask(t, running, apiv2.PipelineTask_RUNTIME, executorPod("p1"), executorPod("p2")),
			nodes: retryNodes("Succeeded",
				util.NodeStatus{ID: "p1", State: "Failed", CreateTime: 10, FinishTime: 20, Message: oomMsg},
				util.NodeStatus{ID: "p2", State: "Succeeded", CreateTime: 30, FinishTime: 40}),
			want: false,
		},
		{
			name: "failed attempt that is not the last child of its retry group",
			task: reconcileTestTask(t, running, apiv2.PipelineTask_RUNTIME, executorPod("p1")),
			nodes: retryNodes("Running",
				util.NodeStatus{ID: "p1", State: "Failed", CreateTime: 10, FinishTime: 20},
				util.NodeStatus{ID: "other", State: "Pending", CreateTime: 30}),
			want: false,
		},
		{
			name: "failed attempt while the retry group is still retrying",
			task: reconcileTestTask(t, running, apiv2.PipelineTask_RUNTIME, executorPod("p1")),
			nodes: retryNodes("Running",
				util.NodeStatus{ID: "p1", State: "Failed", CreateTime: 10, FinishTime: 20}),
			want: false,
		},
		{
			name: "driver pod failure is not executor evidence",
			task: reconcileTestTask(t, running, apiv2.PipelineTask_RUNTIME, driverPod("d1")),
			nodes: map[string]util.NodeStatus{
				"d1": {ID: "d1", Type: "Pod", State: "Failed", CreateTime: 10, FinishTime: 20},
			},
			want: false,
		},
		{
			name: "a successful parent does not matter and nothing failed",
			task: reconcileTestTask(t, running, apiv2.PipelineTask_RUNTIME, executorPod("p1")),
			nodes: map[string]util.NodeStatus{
				"dag": {ID: "dag", Type: "DAG", State: "Succeeded", Children: []string{"p1"}},
				"p1":  {ID: "p1", Type: "Pod", State: "Running", CreateTime: 10},
			},
			want: false,
		},
		{
			name: "DAG tasks are never reconciled from a pod",
			task: reconcileTestTask(t, running, apiv2.PipelineTask_DAG, executorPod("p1")),
			nodes: map[string]util.NodeStatus{
				"p1": {ID: "p1", Type: "Pod", State: "Failed", CreateTime: 10, FinishTime: 20},
			},
			want: false,
		},
		{
			name: "task already succeeded is left alone",
			task: reconcileTestTask(t, apiv2.PipelineTask_SUCCEEDED, apiv2.PipelineTask_RUNTIME, executorPod("p1")),
			nodes: map[string]util.NodeStatus{
				"p1": {ID: "p1", Type: "Pod", State: "Failed", CreateTime: 10, FinishTime: 20},
			},
			want: false,
		},
		{
			name:  "no executor pod in the workflow yet",
			task:  reconcileTestTask(t, running, apiv2.PipelineTask_RUNTIME, executorPod("p1")),
			nodes: map[string]util.NodeStatus{},
			want:  false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := executorFailureForTask(tc.task, tc.nodes, tc.resolved)
			assert.Equal(t, tc.want, ok)
			if ok {
				assert.NotEmpty(t, got.message)
				assert.Positive(t, got.finishedAt)
			}
		})
	}
}

func TestExecutorFailureForTask_UsesResolvedMessageAndFinishTime(t *testing.T) {
	task := reconcileTestTask(t, apiv2.PipelineTask_RUNNING, apiv2.PipelineTask_RUNTIME, executorPod("p1"))
	nodes := retryNodes("Failed", util.NodeStatus{ID: "p1", State: "Failed", CreateTime: 10, FinishTime: 99})
	got, ok := executorFailureForTask(task, nodes, map[string]string{"attempt-0": "main: OOMKilled (exit code 137)"})
	require.True(t, ok)
	assert.Equal(t, "main: OOMKilled (exit code 137)", got.message)
	assert.Equal(t, int64(99), got.finishedAt)
}

func TestExecutorFailureForTask_UnfinishedNodeFallsBackToNow(t *testing.T) {
	task := reconcileTestTask(t, apiv2.PipelineTask_RUNNING, apiv2.PipelineTask_RUNTIME, executorPod("p1"))
	// An Argo node that has no finishedAt converts to a large negative Unix time, not zero.
	nodes := map[string]util.NodeStatus{
		"p1": {ID: "p1", Type: "Pod", State: "Error", CreateTime: 10, FinishTime: time.Time{}.Unix()},
	}
	before := time.Now().Unix()
	got, ok := executorFailureForTask(task, nodes, nil)
	require.True(t, ok)
	assert.GreaterOrEqual(t, got.finishedAt, before)
}

func reconcileTestWorkflow(runName, runUUID string, nodes map[string]v1alpha1.NodeStatus) *util.Workflow {
	return util.NewWorkflow(&v1alpha1.Workflow{
		TypeMeta: v1.TypeMeta{APIVersion: "argoproj.io/v1alpha1", Kind: "Workflow"},
		ObjectMeta: v1.ObjectMeta{
			Name:        runName,
			UID:         types.UID(runUUID),
			Namespace:   "ns1",
			Labels:      map[string]string{util.LabelKeyWorkflowRunId: runUUID},
			Annotations: map[string]string{"workflows.argoproj.io/pod-name-format": "v1"},
		},
		Status: v1alpha1.WorkflowStatus{Phase: v1alpha1.WorkflowRunning, Nodes: nodes},
	})
}

func oomedWorkflowNodes() map[string]v1alpha1.NodeStatus {
	started := v1.NewTime(time.Unix(1000, 0))
	finished := v1.NewTime(time.Unix(1060, 0))
	return map[string]v1alpha1.NodeStatus{
		"oom-retry": {
			ID: "oom-retry", Name: "oom-retry", DisplayName: "executor", Type: v1alpha1.NodeTypeRetry,
			Phase: v1alpha1.NodeFailed, Children: []string{"oom-pod"},
		},
		"oom-pod": {
			ID: "oom-pod", Name: "oom-pod", DisplayName: "executor(0)", Type: v1alpha1.NodeTypePod,
			Phase: v1alpha1.NodeFailed, Message: "main: OOMKilled (exit code 137)",
			StartedAt: started, FinishedAt: finished,
		},
	}
}

func TestReportWorkflowResource_ReconcilesRunningTaskWhenPodDied(t *testing.T) {
	store, manager, run := initWithOneTimeRun(t)
	defer store.Close()

	pods, err := model.ProtoSliceToJSONSlice([]*apiv2.PipelineTask_TaskPod{executorPod("oom-pod")})
	require.NoError(t, err)
	task, err := store.TaskStore().CreateTask(&model.Task{
		Namespace: "ns1", RunUUID: run.UUID, Name: "oom", DisplayName: "oom",
		Type: model.TaskType(apiv2.PipelineTask_RUNTIME), State: model.TaskStatus(apiv2.PipelineTask_RUNNING),
		Fingerprint: "fp-oom", Pods: pods, TypeAttrs: model.JSONData{},
		StateHistory: model.JSONSlice{}, InputParameters: model.JSONSlice{}, OutputParameters: model.JSONSlice{},
	})
	require.NoError(t, err)

	workflow := reconcileTestWorkflow(run.K8SName, run.UUID, oomedWorkflowNodes())
	syncWorkflowReportWithFakeCluster(t, store, workflow)
	_, err = manager.ReportWorkflowResource(context.Background(), workflow)
	require.NoError(t, err)

	got, err := manager.GetTask(task.UUID)
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatus(apiv2.PipelineTask_FAILED), got.State)
	assert.Equal(t, int64(1060), got.FinishedInSec)
	meta, err := model.JSONDataToProtoMessage(got.StatusMetadata, func() *apiv2.PipelineTask_StatusMetadata {
		return &apiv2.PipelineTask_StatusMetadata{}
	})
	require.NoError(t, err)
	assert.Equal(t, "main: OOMKilled (exit code 137)", meta.GetMessage())
	require.NotNil(t, got.LifecycleMessage, "lifecycle diagnostics stay attached to the failed attempt")
}

func TestReportWorkflowResource_ReconcileDoesNotOverwriteSucceededTask(t *testing.T) {
	store, manager, run := initWithOneTimeRun(t)
	defer store.Close()

	pods, err := model.ProtoSliceToJSONSlice([]*apiv2.PipelineTask_TaskPod{executorPod("oom-pod")})
	require.NoError(t, err)
	task, err := store.TaskStore().CreateTask(&model.Task{
		Namespace: "ns1", RunUUID: run.UUID, Name: "oom", DisplayName: "oom",
		Type: model.TaskType(apiv2.PipelineTask_RUNTIME), State: model.TaskStatus(apiv2.PipelineTask_SUCCEEDED),
		Fingerprint: "fp-oom-ok", Pods: pods, TypeAttrs: model.JSONData{},
		StateHistory: model.JSONSlice{}, InputParameters: model.JSONSlice{}, OutputParameters: model.JSONSlice{},
	})
	require.NoError(t, err)

	workflow := reconcileTestWorkflow(run.K8SName, run.UUID, oomedWorkflowNodes())
	syncWorkflowReportWithFakeCluster(t, store, workflow)
	_, err = manager.ReportWorkflowResource(context.Background(), workflow)
	require.NoError(t, err)

	got, err := manager.GetTask(task.UUID)
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatus(apiv2.PipelineTask_SUCCEEDED), got.State)
}
