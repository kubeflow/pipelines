package resource

import (
	"context"
	"testing"

	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	pocruntime "github.com/kubeflow/pipelines/backend/src/v2/runtime/poc"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestManagedRunExplicitRetryReopensCleanedFatalFailure(t *testing.T) {
	enableCoordinatorRuntimeModeForTest(t)
	for key, value := range map[string]any{
		common.V2RuntimeExecutor: pocruntime.ExecutorDocker, common.V2RuntimeAutoExecute: false,
	} {
		previous := viper.Get(key)
		viper.Set(key, value)
		t.Cleanup(func() { viper.Set(key, previous) })
	}
	t.Setenv("KFP_DEFAULT_PIPELINE_ROOT", "file://"+t.TempDir())
	initEnvVars()
	client := NewFakeClientManagerOrFatalV2()
	defer client.Close()
	manager := NewResourceManager(client, &ResourceManagerOptions{CollectMetrics: false})
	experiment, err := manager.CreateExperiment(&model.Experiment{Name: "runtime-retry", Namespace: "ns1"})
	require.NoError(t, err)
	run, err := manager.CreateRun(context.Background(), &model.Run{
		DisplayName: "fatal-publication", ExperimentId: experiment.UUID,
		PipelineSpec: model.PipelineSpec{
			PipelineSpecManifest: model.LargeText(v2SpecHelloWorld),
			RuntimeConfig:        model.RuntimeConfig{Parameters: `{"text":"world"}`},
		},
	})
	require.NoError(t, err)
	provider, ok := manager.taskStore.(interface {
		RuntimeStore() storage.RuntimeStoreInterface
	})
	require.True(t, ok)
	runtime := provider.RuntimeStore()
	tasks, _, _, err := manager.taskStore.ListTasks(&model.FilterContext{
		ReferenceKey: &model.ReferenceKey{Type: model.RunResourceType, ID: run.UUID},
	}, list.EmptyOptions())
	require.NoError(t, err)
	require.NotEmpty(t, tasks)
	for _, task := range tasks {
		require.NoError(t, runtime.FinalizePending(task.UUID, &model.Task{
			UUID: task.UUID, State: model.TaskStatus(apiv2.PipelineTask_FAILED),
			StatusMetadata: model.JSONData{"message": "fatal publication retries exhausted"},
		}, nil))
	}
	sibling, err := manager.taskStore.CreateTask(&model.Task{
		RunUUID: run.UUID, Namespace: run.Namespace, Name: "successful-sibling", ScopePath: "root.successful-sibling",
		State: model.TaskStatus(apiv2.PipelineTask_SUCCEEDED),
	})
	require.NoError(t, err)
	run.State, run.FinishedAtInSec = model.RuntimeStateFailed, 100
	require.NoError(t, manager.runStore.UpdateRun(run))
	require.NoError(t, manager.RetryRun(context.Background(), run.UUID))
	reopened, err := manager.runStore.GetRun(run.UUID, false)
	require.NoError(t, err)
	require.Equal(t, model.RuntimeStatePending, reopened.State)
	require.EqualValues(t, 1, reopened.RetryGeneration)
	require.Zero(t, reopened.FinishedAtInSec)
	for _, task := range tasks {
		pending, err := runtime.Get(task.UUID)
		require.NoError(t, err)
		require.Equal(t, model.RuntimeTaskWaiting, pending.State)
		require.Nil(t, pending.AttemptUUID)
	}
	preserved, err := manager.taskStore.GetTask(sibling.UUID)
	require.NoError(t, err)
	require.Equal(t, model.TaskStatus(apiv2.PipelineTask_SUCCEEDED), preserved.State)
	require.Error(t, manager.RetryRun(context.Background(), run.UUID), "a second request must not reset an already reopened run")
}
