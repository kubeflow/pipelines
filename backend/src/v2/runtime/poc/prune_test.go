package poc

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

type pruneExecutor struct {
	fakeExecutorLifecycle
	cleanups int
	err      error
}

func (*pruneExecutor) Type() string { return ExecutorDocker }
func (*pruneExecutor) Start(context.Context) error {
	panic("cleanup must not start a worker")
}
func (*pruneExecutor) Submit(context.Context, TaskExecutionRequest) (ExecutionReference, error) {
	panic("cleanup must not submit tasks")
}
func (*pruneExecutor) Wait(context.Context, ExecutionReference) (TaskExecutionResult, error) {
	panic("cleanup must not observe or publish a new result")
}
func (*pruneExecutor) Cancel(context.Context, ExecutionReference) (CancellationResult, error) {
	panic("cleanup must not terminate workloads")
}
func (e *pruneExecutor) Cleanup(context.Context, ExecutionReference) error {
	e.cleanups++
	return e.err
}

func pruneFixture(t *testing.T) (*Coordinator, *model.Run, *model.RuntimeTask, *pruneExecutor) {
	t.Helper()
	root := t.TempDir()
	for key, value := range map[string]string{
		common.V2RuntimeExecutor: ExecutorDocker, common.V2RuntimeWorkspaceRoot: root,
	} {
		previous := viper.Get(key)
		viper.Set(key, value)
		t.Cleanup(func() { viper.Set(key, previous) })
	}
	db, dialect, err := storage.NewFakeDB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	db.SetMaxOpenConns(1)
	clock := util.NewFakeTimeForEpoch()
	runs := storage.NewRunStore(db, clock, dialect)
	tasks := storage.NewTaskStore(db, clock, util.NewUUIDGenerator(), dialect)
	manifest := NewRuntimeManifest(ExecutorDocker)
	manifest.Tasks = []RuntimeTask{{TaskID: "prune-task"}}
	manifest.Workspace = &WorkspaceHandle{Version: 1, Type: WorkspaceTypeHostPath, Metadata: map[string]string{
		"runID": "prune-run", "path": filepath.Join(root, "prune-run", "workspace"),
	}}
	raw, err := manifest.ToJSON()
	require.NoError(t, err)
	run, err := runs.CreateRun(&model.Run{
		UUID: "prune-run", Namespace: "ns", K8SName: "prune-run",
		RunDetails: model.RunDetails{
			State: model.RuntimeStateRunning, PipelineRuntimeManifest: model.LargeText(raw),
			ExecutorType: util.StringPointer(ExecutorDocker),
		},
	})
	require.NoError(t, err)
	c := NewCoordinator(runs, tasks, nil, nil, clock, nil, nil, false)
	task, err := c.runtimeStore.CreateTask(&model.Task{
		UUID: "prune-task", RunUUID: run.UUID, Namespace: run.Namespace,
		Name: "task", ScopePath: "root/task", State: model.TaskStatus(apiv2.PipelineTask_RUNNING),
	})
	require.NoError(t, err)
	manifest.Tasks = []RuntimeTask{{TaskID: task.UUID}}
	raw, err = manifest.ToJSON()
	require.NoError(t, err)
	run.PipelineRuntimeManifest = model.LargeText(raw)
	require.NoError(t, runs.UpdateRun(run))
	identity := TaskAttemptIdentity{RunID: run.UUID, TaskID: task.UUID, AttemptID: "prune-attempt", Namespace: run.Namespace}
	request, err := toJSONData(TaskExecutionRequest{
		Identity: identity, Namespace: run.Namespace,
		ExecutorInput: &pipelinespec.ExecutorInput{Outputs: &pipelinespec.ExecutorInput_Outputs{
			Artifacts: map[string]*pipelinespec.ArtifactList{
				"executor-logs": {Artifacts: []*pipelinespec.RuntimeArtifact{{Uri: "file://" + filepath.Join(root, "published-logs")}}},
			},
		}},
	})
	require.NoError(t, err)
	completion, err := toJSONData(executorCompletion{
		Terminal: true, LogsRetained: true, Result: &TaskExecutionResult{Outcome: TaskOutcomeSuccess},
	})
	require.NoError(t, err)
	reference, err := toJSONData(ExecutionReference{
		Executor: ExecutorDocker, Version: 1, Identity: identity, Data: model.JSONData{executorCompletionKey: completion},
	})
	require.NoError(t, err)
	attempt := &model.RuntimeTask{
		TaskUUID: task.UUID, RunUUID: run.UUID, AttemptUUID: util.StringPointer(identity.AttemptID),
		State: model.RuntimeTaskFinalizing, ExecutionRequest: request, ExecutionReference: reference,
	}
	require.NoError(t, c.runtimeStore.Update(attempt, ""))
	require.NoError(t, os.MkdirAll(manifest.Workspace.Metadata["path"], 0o700))
	executor := &pruneExecutor{}
	c.RegisterExecutor(executor)
	return c, run, attempt, executor
}

func TestPruneSkipsActiveRunsWithoutCanceling(t *testing.T) {
	c, _, attempt, executor := pruneFixture(t)
	result, err := c.Prune(context.Background(), PruneOptions{})
	require.NoError(t, err)
	require.Equal(t, 1, result.RunsSkipped)
	require.Zero(t, executor.cleanups)
	_, err = c.runtimeStore.Get(attempt.TaskUUID)
	require.NoError(t, err)
}

func TestPruneFinalizesOnlyAfterCleanupAndPreservesHistory(t *testing.T) {
	c, run, attempt, executor := pruneFixture(t)
	require.NoError(t, c.updateRunState(run.UUID, model.RuntimeStateSucceeded, 123))
	executor.err = errors.New("backend removal unconfirmed")
	result, err := c.Prune(context.Background(), PruneOptions{})
	require.ErrorContains(t, err, "backend removal unconfirmed")
	require.Equal(t, 1, result.Failures)
	_, err = c.runtimeStore.Get(attempt.TaskUUID)
	require.NoError(t, err)
	require.DirExists(t, filepath.Join(WorkspaceRoot(), run.UUID, "workspace"))
	executor.err = nil
	published := filepath.Join(WorkspaceRoot(), "published-artifact-and-log")
	require.NoError(t, os.WriteFile(published, []byte("durable output"), 0o600))
	result, err = c.Prune(context.Background(), PruneOptions{KeepWorkspaces: true})
	require.NoError(t, err)
	require.Equal(t, 1, result.AttemptsFinalized)
	require.Equal(t, 1, result.WorkspacesRetained)
	require.DirExists(t, filepath.Join(WorkspaceRoot(), run.UUID, "workspace"))
	_, err = c.runtimeStore.Get(attempt.TaskUUID)
	require.ErrorIs(t, err, sql.ErrNoRows)
	task, err := c.taskStore.GetTask(attempt.TaskUUID)
	require.NoError(t, err)
	require.Equal(t, model.TaskStatus(apiv2.PipelineTask_SUCCEEDED), task.State)
	result, err = c.Prune(context.Background(), PruneOptions{})
	require.NoError(t, err)
	require.Equal(t, 1, result.RunsCleaned)
	require.NoDirExists(t, filepath.Join(WorkspaceRoot(), run.UUID, "workspace"))
	current, err := c.runStore.GetRun(run.UUID, false)
	require.NoError(t, err)
	require.Equal(t, model.RuntimeStateSucceeded, current.State)
	require.EqualValues(t, 123, current.FinishedAtInSec)
	require.NotEmpty(t, current.PipelineRuntimeManifest, "keep historical workspace identity")
	content, err := os.ReadFile(published)
	require.NoError(t, err)
	require.Equal(t, "durable output", string(content))
}

func TestPruneRetainsUnconfirmedResultsLogsAndIdentity(t *testing.T) {
	for _, field := range []string{"logs", "outcome", "identity", "phase"} {
		t.Run(field, func(t *testing.T) {
			c, run, attempt, executor := pruneFixture(t)
			var reference ExecutionReference
			require.NoError(t, fromJSONData(attempt.ExecutionReference, &reference))
			completion, err := loadExecutorCompletion(reference)
			require.NoError(t, err)
			switch field {
			case "logs":
				completion.LogsRetained = false
			case "outcome":
				completion.Terminal = false
			case "identity":
				reference.Identity.AttemptID = "another-attempt"
			case "phase":
				attempt.State = model.RuntimeTaskRunning
			}
			data, err := toJSONData(completion)
			require.NoError(t, err)
			reference.Data[executorCompletionKey] = data
			attempt.ExecutionReference, err = toJSONData(reference)
			require.NoError(t, err)
			require.NoError(t, c.runtimeStore.Update(attempt, *attempt.AttemptUUID))
			require.NoError(t, c.updateRunState(run.UUID, model.RuntimeStateSucceeded, 123))
			_, err = c.Prune(context.Background(), PruneOptions{})
			require.Error(t, err)
			require.Zero(t, executor.cleanups)
			_, err = c.runtimeStore.Get(attempt.TaskUUID)
			require.NoError(t, err)
			require.DirExists(t, filepath.Join(WorkspaceRoot(), run.UUID, "workspace"))
		})
	}
}

func TestPruneRefusesAnotherWorker(t *testing.T) {
	c, _, _, executor := pruneFixture(t)
	other := &Coordinator{}
	require.NoError(t, other.lockInstallation())
	defer other.workerLock.Close()
	_, err := c.Prune(context.Background(), PruneOptions{})
	require.ErrorContains(t, err, "another worker")
	require.Zero(t, executor.cleanups)
}

func TestPruneBuiltinDockerRefusesLiveWorkloadDespiteTerminalCheckpoint(t *testing.T) {
	c, run, attempt, _ := pruneFixture(t)
	var request TaskExecutionRequest
	require.NoError(t, fromJSONData(attempt.ExecutionRequest, &request))
	engine := &fakeDockerEngine{}
	executor := dockerExecutor{command: engine.run}
	submission := request
	submission.ExecutorInput = nil
	submission.Container = &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{Image: "test-image"}
	reference, err := executor.Submit(context.Background(), submission)
	require.NoError(t, err)
	var original ExecutionReference
	require.NoError(t, fromJSONData(attempt.ExecutionReference, &original))
	reference.Data[executorCompletionKey] = original.Data[executorCompletionKey]
	attempt.ExecutionReference, err = toJSONData(reference)
	require.NoError(t, err)
	require.NoError(t, c.runtimeStore.Update(attempt, *attempt.AttemptUUID))
	c.RegisterExecutor(executor)
	require.NoError(t, c.updateRunState(run.UUID, model.RuntimeStateSucceeded, 123))
	_, err = c.Prune(context.Background(), PruneOptions{})
	require.ErrorContains(t, err, "nonterminal")
	require.Zero(t, engine.stopCount)
	require.Zero(t, engine.removeCount)
	_, err = c.runtimeStore.Get(attempt.TaskUUID)
	require.NoError(t, err)
	engine.container.State.Status, engine.container.State.Running = "exited", false
	_, err = c.Prune(context.Background(), PruneOptions{})
	require.NoError(t, err)
	require.Equal(t, 1, engine.removeCount)
	require.Zero(t, engine.stopCount)
}

type pruneKubernetesExecutor struct{ pruneExecutor }

func (*pruneKubernetesExecutor) Type() string { return ExecutorKubernetes }

type pruneLeaseStore struct {
	storage.RunLeaseStoreInterface
	checkErr error
}

func (s pruneLeaseStore) Check(token storage.RunLeaseToken) error {
	if s.checkErr != nil {
		return s.checkErr
	}
	return s.RunLeaseStoreInterface.Check(token)
}

func TestPruneKubernetesUsesLeaseFences(t *testing.T) {
	for _, failure := range []string{"held", "lost"} {
		t.Run(failure, func(t *testing.T) {
			c, run, attempt, _ := pruneFixture(t)
			viper.Set(common.V2RuntimeExecutor, ExecutorKubernetes)
			manifest, err := ParseManifest(string(run.PipelineRuntimeManifest))
			require.NoError(t, err)
			manifest.Executor = ExecutorKubernetes
			raw, err := manifest.ToJSON()
			require.NoError(t, err)
			run.PipelineRuntimeManifest = model.LargeText(raw)
			run.ExecutorType = util.StringPointer(ExecutorKubernetes)
			require.NoError(t, c.runStore.UpdateRun(run))
			var reference ExecutionReference
			require.NoError(t, fromJSONData(attempt.ExecutionReference, &reference))
			reference.Executor = ExecutorKubernetes
			attempt.ExecutionReference, err = toJSONData(reference)
			require.NoError(t, err)
			require.NoError(t, c.runtimeStore.Update(attempt, *attempt.AttemptUUID))
			require.NoError(t, c.updateRunState(run.UUID, model.RuntimeStateSucceeded, 123))
			executor := &pruneKubernetesExecutor{}
			c.RegisterExecutor(executor)
			store := c.leaseStore
			if failure == "held" {
				token, err := store.Acquire(run.UUID, "other-worker", c.leaseDuration, true)
				require.NoError(t, err)
				defer store.Release(*token)
			} else {
				c.leaseStore = pruneLeaseStore{RunLeaseStoreInterface: store, checkErr: storage.ErrRunLeaseConflict}
			}
			_, err = c.Prune(context.Background(), PruneOptions{KeepWorkspaces: true})
			require.ErrorIs(t, err, storage.ErrRunLeaseConflict)
			require.Zero(t, executor.cleanups)
			_, err = c.runtimeStore.Get(attempt.TaskUUID)
			require.NoError(t, err)
			require.DirExists(t, manifest.Workspace.Metadata["path"])
			if failure == "lost" {
				c.leaseStore = store
				result, err := c.Prune(context.Background(), PruneOptions{KeepWorkspaces: true})
				require.NoError(t, err)
				require.Equal(t, 1, result.AttemptsFinalized)
				require.Equal(t, 1, executor.cleanups)
			}
		})
	}
}

func TestPruneCanceledAttemptRequiresDurableLogCheckpoint(t *testing.T) {
	for _, retained := range []bool{false, true} {
		t.Run(fmt.Sprint(retained), func(t *testing.T) {
			c, run, attempt, executor := pruneFixture(t)
			var request TaskExecutionRequest
			require.NoError(t, fromJSONData(attempt.ExecutionRequest, &request))
			var reference ExecutionReference
			require.NoError(t, fromJSONData(attempt.ExecutionReference, &reference))
			completion, err := toJSONData(executorCompletion{})
			require.NoError(t, err)
			reference.Data[executorCompletionKey] = completion
			attempt.ExecutionReference, err = toJSONData(reference)
			require.NoError(t, err)
			attempt.State = model.RuntimeTaskCancelFinalizing
			properties := map[string]any{"attemptID": *attempt.AttemptUUID}
			if retained {
				properties["attemptLogURI"] = request.ExecutorInput.Outputs.Artifacts["executor-logs"].Artifacts[0].Uri
			}

			require.NoError(t, c.runtimeStore.CheckpointCancellation(attempt, &model.Task{
				UUID: attempt.TaskUUID, State: model.TaskStatus(apiv2.PipelineTask_RUNNING),
				StatusMetadata: customStatusMetadata("", properties),
			}))
			require.NoError(t, c.updateRunState(run.UUID, model.RuntimeStateCanceled, 123))
			_, err = c.Prune(context.Background(), PruneOptions{})
			if retained {
				require.NoError(t, err)
				require.Equal(t, 1, executor.cleanups)
				task, err := c.taskStore.GetTask(attempt.TaskUUID)
				require.NoError(t, err)
				require.Equal(t, model.TaskStatus(apiv2.PipelineTask_SKIPPED), task.State)
			} else {
				require.ErrorContains(t, err, "durable logs")
				require.Zero(t, executor.cleanups)
			}
			current, err := c.runStore.GetRun(run.UUID, false)
			require.NoError(t, err)
			require.Equal(t, model.RuntimeStateCanceled, current.State)
		})
	}
}

func TestPruneContinuesAfterAnUnresolvedRun(t *testing.T) {
	c, _, _, executor := pruneFixture(t)
	require.NoError(t, c.updateRunState("prune-run", model.RuntimeStateSucceeded, 123))
	executor.err = errors.New("retained execution")
	raw, err := NewRuntimeManifest(ExecutorDocker).ToJSON()
	require.NoError(t, err)
	_, err = c.runStore.CreateRun(&model.Run{
		UUID: "z-clean-run", K8SName: "z-clean-run", Namespace: "ns",
		RunDetails: model.RunDetails{
			State: model.RuntimeStateSucceeded, PipelineRuntimeManifest: model.LargeText(raw),
			ExecutorType: util.StringPointer(ExecutorDocker),
		},
	})
	require.NoError(t, err)
	result, err := c.Prune(context.Background(), PruneOptions{})
	require.ErrorContains(t, err, "retained execution")
	require.Equal(t, 2, result.RunsExamined)
	require.Equal(t, 1, result.Failures)
	require.Equal(t, 1, result.RunsCleaned)
}
