package poc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
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

type canceledLogExecutor struct {
	logTestExecutor
	canceled   bool
	cleanupErr error
	cleanups   int
}

func (e *canceledLogExecutor) Cancel(context.Context, ExecutionReference) error {
	e.canceled = true
	return nil
}
func (e *canceledLogExecutor) Cleanup(context.Context, ExecutionReference) error {
	e.cleanups++
	return e.cleanupErr
}

func cancellationLogFixture(t *testing.T, executorType string) (*Coordinator, *model.Run, *model.Task, TaskExecutionRequest) {
	t.Helper()
	root, bucket := t.TempDir(), t.TempDir()
	originalRoot, originalBucket := viper.Get(common.V2RuntimeWorkspaceRoot), viper.Get("ObjectStoreConfig.BucketURL")
	viper.Set(common.V2RuntimeWorkspaceRoot, root)
	viper.Set("ObjectStoreConfig.BucketURL", "file://"+bucket)
	t.Setenv("KFP_DEFAULT_PIPELINE_ROOT", "file://"+bucket)
	t.Cleanup(func() {
		viper.Set(common.V2RuntimeWorkspaceRoot, originalRoot)
		viper.Set("ObjectStoreConfig.BucketURL", originalBucket)
	})
	db, dialect, err := storage.NewFakeDB()
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	clock, uuid := util.NewFakeTimeForEpoch(), util.NewUUIDGenerator()
	runs := storage.NewRunStore(db, clock, dialect)
	tasks := storage.NewTaskStore(db, clock, uuid, dialect)
	artifacts := storage.NewArtifactStore(db, clock, uuid, dialect)
	links := storage.NewArtifactTaskStore(db, uuid, dialect)
	coordinator := NewCoordinator(runs, tasks, artifacts, links, clock, nil, nil, false)
	manifest, err := NewRuntimeManifest(executorType).ToJSON()
	require.NoError(t, err)
	run, err := runs.CreateRun(&model.Run{
		UUID: "canceled-log-run", Namespace: "ns", K8SName: "canceled-log-run",
		RunDetails: model.RunDetails{
			State: model.RuntimeStateRunning, ExecutorType: util.StringPointer(executorType),
			PipelineRuntimeManifest: model.LargeText(manifest),
		},
	})
	require.NoError(t, err)
	task, err := coordinator.createTask(run, nil, "task", "task", apiv2.PipelineTask_RUNTIME, "root.task", nil)
	require.NoError(t, err)
	identity := TaskAttemptIdentity{RunID: run.UUID, TaskID: task.UUID, AttemptID: "canceled-log-attempt"}
	logURI := "file://" + filepath.Join(bucket, "canceled.log")
	request := TaskExecutionRequest{
		Identity: identity, TaskName: task.Name,
		ArtifactLocalRoot: filepath.Join(root, run.UUID, "artifact-local", identity.AttemptID),
		ExecutorInput: &pipelinespec.ExecutorInput{Outputs: &pipelinespec.ExecutorInput_Outputs{
			Artifacts: map[string]*pipelinespec.ArtifactList{"executor-logs": {Artifacts: []*pipelinespec.RuntimeArtifact{{Uri: logURI}}}},
		}},
	}
	return coordinator, run, task, request
}

func TestCanceledAttemptPublishesLogsBeforeCleanupAndFinalization(t *testing.T) {
	coordinator, run, task, request := cancellationLogFixture(t, "scripted")
	executor := &canceledLogExecutor{
		logTestExecutor: logTestExecutor{text: "complete canceled logs\n"},
		cleanupErr:      fmt.Errorf("temporary cleanup failure"),
	}
	coordinator.RegisterExecutor(executor)
	identity := request.Identity
	requestData, err := toJSONData(request)
	require.NoError(t, err)
	referenceData, err := toJSONData(ExecutionReference{Executor: "scripted", Version: 1, Identity: identity})
	require.NoError(t, err)
	require.NoError(t, coordinator.runtimeStore.Create(&model.RuntimeTask{
		TaskUUID: task.UUID, RunUUID: run.UUID, AttemptUUID: util.StringPointer(identity.AttemptID),
		State: model.RuntimeTaskRunning, ExecutionRequest: requestData, ExecutionReference: referenceData,
	}))
	require.NoError(t, coordinator.updateRunState(run.UUID, model.RuntimeStateCancelling, 0))
	err = coordinator.cancelRuntimeAttempts(context.Background(), run.UUID)
	var pending *unresolvedAttempt
	require.True(t, errors.As(err, &pending))
	require.True(t, executor.canceled)
	attempt, err := coordinator.runtimeStore.Get(task.UUID)
	require.NoError(t, err)
	require.Equal(t, model.RuntimeTaskCancelFinalizing, attempt.State)
	executor.cleanupErr = nil
	require.NoError(t, coordinator.cancelRuntimeAttempts(context.Background(), run.UUID))
	require.Equal(t, 1, executor.calls, "cleanup recovery must reuse durable canceled-attempt logs")
	require.Equal(t, 2, executor.cleanups)
	stored, err := coordinator.taskStore.GetTask(task.UUID)
	require.NoError(t, err)
	require.Equal(t, model.TaskStatus(apiv2.PipelineTask_SKIPPED), stored.State)
	var output bytes.Buffer
	require.NoError(t, coordinator.ReadTaskLog(context.Background(), run, task.UUID, false, &output))
	require.Equal(t, "complete canceled logs\n", output.String())
}

func TestDockerCancellationCoordinatorRetainsLogsUntilPublication(t *testing.T) {
	coordinator, run, task, request := cancellationLogFixture(t, ExecutorDocker)
	engine := &fakeDockerEngine{}
	executor := dockerExecutor{
		command: engine.run,
		logCommand: func(ctx context.Context, destination io.Writer, args ...string) error {
			stdout, stderr, err := engine.run(ctx, args...)
			if err != nil {
				return err
			}
			_, err = io.WriteString(destination, stdout+stderr)
			return err
		},
	}
	coordinator.RegisterExecutor(executor)
	request.Container = &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{Image: "test", Command: []string{"echo", "logs"}}
	reference, err := executor.Submit(context.Background(), request)
	require.NoError(t, err)
	requestData, err := toJSONData(request)
	require.NoError(t, err)
	referenceData, err := toJSONData(reference)
	require.NoError(t, err)
	require.NoError(t, coordinator.runtimeStore.Create(&model.RuntimeTask{
		TaskUUID: task.UUID, RunUUID: run.UUID, AttemptUUID: util.StringPointer(request.Identity.AttemptID),
		State: model.RuntimeTaskRunning, ExecutionRequest: requestData, ExecutionReference: referenceData,
	}))
	require.NoError(t, coordinator.updateRunState(run.UUID, model.RuntimeStateCancelling, 0))
	require.NoError(t, coordinator.cancelRuntimeAttempts(context.Background(), run.UUID))
	stored, err := coordinator.taskStore.GetTask(task.UUID)
	require.NoError(t, err)
	properties, ok := stored.StatusMetadata["customProperties"].(map[string]interface{})
	require.True(t, ok)
	require.NotContains(t, properties, "logsUnavailable", "canceled logs must remain readable until publication")
	require.NotContains(t, properties, "logPublicationFailure", "canceled logs must be durably published")
	var logs bytes.Buffer
	require.NoError(t, coordinator.ReadTaskLog(context.Background(), run, task.UUID, false, &logs))
	require.Contains(t, logs.String(), "durable stdout")
	require.Contains(t, logs.String(), "durable stderr")
	require.Equal(t, 1, engine.stopCount)
	require.Equal(t, 1, engine.removeCount, "only post-publication cleanup should remove the container")
	require.Error(t, executor.ReadLogs(context.Background(), reference, false, io.Discard), "the completed cleanup should release the backend log source")
}
