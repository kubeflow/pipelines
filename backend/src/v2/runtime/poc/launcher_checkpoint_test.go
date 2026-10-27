package poc

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/component"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubernetesfake "k8s.io/client-go/kubernetes/fake"
)

func checkpointLauncherReport(t *testing.T, coordinator *Coordinator, taskID, attemptID string, published bool) {
	t.Helper()
	phase := component.RuntimePhaseComputed
	if published {
		phase = component.RuntimePhasePublished
	}
	raw, err := json.Marshal(&component.RuntimeLauncherReport{
		Version: 1, AttemptID: attemptID, ComputationCompleted: true,
		PublicationCompleted: published, Phase: phase, PublicationAttempts: 1,
		ExecutorOutputJSON: `{"parameterValues":{"value":9}}`,
	})
	require.NoError(t, err)
	_, err = coordinator.taskStore.UpdateTask(&model.Task{
		UUID:           taskID,
		StatusMetadata: customStatusMetadata("", map[string]any{component.RuntimeLauncherReportKey: string(raw)}),
	})
	require.NoError(t, err)
}

func TestLauncherCheckpointMustMatchCurrentAttempt(t *testing.T) {
	coordinator, _, preparation := attemptFixture(t)
	checkpointLauncherReport(t, coordinator, preparation.Task.UUID, "old-attempt", true)
	task, err := coordinator.taskStore.GetTask(preparation.Task.UUID)
	require.NoError(t, err)
	result, err := reconcileLauncherCheckpoint(task, "current-attempt", &TaskExecutionResult{
		Outcome: TaskOutcomeError, Failure: &TaskFailure{Class: "interrupted"},
	})
	require.NoError(t, err)
	require.Equal(t, "interrupted", result.Failure.Class)
	checkpointLauncherReport(t, coordinator, preparation.Task.UUID, "current-attempt", true)
	task, err = coordinator.taskStore.GetTask(preparation.Task.UUID)
	require.NoError(t, err)
	result, err = reconcileLauncherCheckpoint(task, "current-attempt", result)
	require.NoError(t, err)
	require.Equal(t, TaskOutcomeSuccess, result.Outcome)
	require.Nil(t, result.Failure)
	require.Equal(t, float64(9), result.OutputParameters["value"].GetNumberValue())
}

func TestMissingPodDoesNotReplayCheckpointedSuccessfulComputation(t *testing.T) {
	coordinator, item, preparation := attemptFixture(t)
	checkpointLauncherReport(t, coordinator, preparation.Task.UUID, "computed-attempt", false)
	identity := TaskAttemptIdentity{
		RunID: item.run.UUID, TaskID: preparation.Task.UUID, AttemptID: "computed-attempt", Namespace: item.run.Namespace,
	}
	request, err := toJSONData(TaskExecutionRequest{Identity: identity, ExecutorInput: preparation.ExecutorInput})
	require.NoError(t, err)
	reference, err := toJSONData(podReference(identity, &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
		Name: attemptWorkloadName(identity), Namespace: item.run.Namespace, UID: "missing-pod-uid",
	}}))
	require.NoError(t, err)
	require.NoError(t, coordinator.runtimeStore.Update(&model.RuntimeTask{
		TaskUUID: preparation.Task.UUID, RunUUID: item.run.UUID,
		AttemptUUID: util.StringPointer(identity.AttemptID), State: model.RuntimeTaskRunning,
		ExecutionRequest: request, ExecutionReference: reference,
	}, ""))
	client := kubernetesfake.NewSimpleClientset()
	executor := kubernetesExecutor{coreClient: &fakeKubernetesCoreWithClientset{clientSet: client}}
	_, err = coordinator.executeAttempt(context.Background(), item, executor, preparation, &pipelinespec.PipelineTaskSpec{
		RetryPolicy: &pipelinespec.PipelineTaskSpec_RetryPolicy{MaxRetryCount: 3, Policy: pipelinespec.PipelineTaskSpec_RetryPolicy_POLICY_ALWAYS},
	}, false)
	require.ErrorContains(t, err, "publication")
	for _, action := range client.Actions() {
		require.NotEqual(t, "create", action.GetVerb(), "known successful computation must never be resubmitted")
	}
	task, err := coordinator.taskStore.GetTask(preparation.Task.UUID)
	require.NoError(t, err)
	require.Equal(t, model.TaskStatus(apiv2.PipelineTask_FAILED), task.State)
	_, err = coordinator.runtimeStore.Get(task.UUID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}
