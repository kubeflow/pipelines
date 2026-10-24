package poc

import (
	"database/sql"
	"testing"

	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/require"
)

func TestFailedRunCleanupFinalizesOnlyConfirmedUndispatchedTasks(t *testing.T) {
	coordinator, item, preparation := attemptFixture(t)
	require.NoError(t, coordinator.updateRunState(item.run.UUID, model.RuntimeStateFailed, 1))
	item.run.State = model.RuntimeStateFailed
	pending, err := coordinator.runtimeStore.Get(preparation.Task.UUID)
	require.NoError(t, err)
	require.NoError(t, coordinator.cleanupNeverDispatchedTask(item.run, pending))
	_, err = coordinator.runtimeStore.Get(preparation.Task.UUID)
	require.ErrorIs(t, err, sql.ErrNoRows)
	task, err := coordinator.taskStore.GetTask(preparation.Task.UUID)
	require.NoError(t, err)
	require.Equal(t, model.TaskStatus(apiv2.PipelineTask_SKIPPED), task.State)
	run, err := coordinator.runStore.GetRun(item.run.UUID, false)
	require.NoError(t, err)
	require.Equal(t, model.RuntimeStateFailed, run.State)
	fakeAttempt := "possibly-submitted"
	pending.AttemptUUID = &fakeAttempt
	require.ErrorContains(t, coordinator.cleanupNeverDispatchedTask(item.run, pending), "never-dispatched")
}
