package poc

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

type logTestExecutor struct {
	scriptedExecutor
	text      string
	err       error
	reference ExecutionReference
	follow    bool
	calls     int
}

func (e *logTestExecutor) Logs(_ context.Context, reference ExecutionReference) (ExecutionLogSource, error) {
	return ExecutionLogSource{Identity: reference.Identity, stream: func(ctx context.Context, follow bool, destination io.Writer) error {
		return e.ReadLogs(ctx, reference, follow, destination)
	}}, nil
}

func (e *logTestExecutor) ReadLogs(_ context.Context, reference ExecutionReference, follow bool, dst io.Writer) error {
	e.reference, e.follow = reference, follow
	e.calls++
	if _, err := io.WriteString(dst, e.text); err != nil {
		return err
	}
	return e.err
}

type logArtifactStore struct {
	storage.ArtifactStoreInterface
	artifact *model.Artifact
	calls    int
}

func (s *logArtifactStore) GetArtifact(string) (*model.Artifact, error) {
	s.calls++
	return s.artifact, nil
}

type logArtifactLinks struct {
	storage.ArtifactTaskStoreInterface
	artifactID string
}

func (s *logArtifactLinks) ListArtifactTasks([]*model.FilterContext, *model.IOType, *list.Options) ([]*model.ArtifactTask, int, string, error) {
	return []*model.ArtifactTask{{ArtifactID: s.artifactID, ArtifactKey: "executor-logs"}}, 1, "", nil
}

func activeLogFixture(t *testing.T) (*Coordinator, *queuedRun, *model.Task, *logTestExecutor) {
	t.Helper()
	coordinator, item, preparation := attemptFixture(t)
	item.run.ExecutorType = util.StringPointer("scripted")
	executor := &logTestExecutor{text: "live output\n"}
	coordinator.RegisterExecutor(executor)
	reference := ExecutionReference{
		Executor: "scripted", Version: 1,
		Identity: TaskAttemptIdentity{RunID: item.run.UUID, TaskID: preparation.Task.UUID, AttemptID: "log-attempt"},
	}
	data, err := toJSONData(reference)
	require.NoError(t, err)
	require.NoError(t, coordinator.runtimeStore.Update(&model.RuntimeTask{
		TaskUUID: preparation.Task.UUID, RunUUID: item.run.UUID, AttemptUUID: util.StringPointer("log-attempt"),
		State: model.RuntimeTaskRunning, ExecutionReference: data,
	}, ""))
	return coordinator, item, preparation.Task, executor
}

func TestReadTaskLogUsesTaskIdentityWithoutPods(t *testing.T) {
	coordinator, item, task, executor := activeLogFixture(t)
	var logs bytes.Buffer
	require.NoError(t, coordinator.ReadTaskLog(context.Background(), item.run, task.UUID, true, &logs))
	require.Equal(t, "live output\n", logs.String())
	require.Equal(t, task.UUID, executor.reference.Identity.TaskID)
	require.True(t, executor.follow)
	require.Empty(t, task.Pods)
	wrongRun := *item.run
	wrongRun.UUID = "other-run"
	require.ErrorContains(t, coordinator.ReadTaskLog(context.Background(), &wrongRun, task.UUID, false, io.Discard), "does not belong")
	require.Equal(t, 1, executor.calls)
}

func TestReadTaskLogDoesNotDuplicatePartialStream(t *testing.T) {
	coordinator, item, task, executor := activeLogFixture(t)
	executor.err = errors.New("interrupted stream")
	artifacts := &logArtifactStore{}
	coordinator.artifactStore = artifacts
	coordinator.artifactTaskStore = &logArtifactLinks{artifactID: "logs"}
	var logs bytes.Buffer
	require.ErrorIs(t, coordinator.ReadTaskLog(context.Background(), item.run, task.UUID, true, &logs), executor.err)
	require.Equal(t, "live output\n", logs.String())
	require.Zero(t, artifacts.calls, "a partial stream must not restart from published logs")
}

func TestReadTaskLogUsesPublishedArtifactAfterFinalization(t *testing.T) {
	coordinator, item, preparation := attemptFixture(t)
	item.run.ExecutorType = util.StringPointer("scripted")
	executor := &logTestExecutor{}
	coordinator.RegisterExecutor(executor)
	root := t.TempDir()
	path := filepath.Join(root, "attempt.log")
	require.NoError(t, os.WriteFile(path, []byte("durable output\n"), 0o600))
	original := viper.Get("ObjectStoreConfig.BucketURL")
	viper.Set("ObjectStoreConfig.BucketURL", "file://"+root)
	t.Cleanup(func() { viper.Set("ObjectStoreConfig.BucketURL", original) })
	uri := "file://" + path
	coordinator.artifactStore = &logArtifactStore{artifact: &model.Artifact{
		UUID: "logs", Namespace: item.run.Namespace, URI: &uri,
	}}
	coordinator.artifactTaskStore = &logArtifactLinks{artifactID: "logs"}
	require.NoError(t, coordinator.runtimeStore.FinalizePending(preparation.Task.UUID, &model.Task{
		UUID: preparation.Task.UUID, State: model.TaskStatus(apiv2.PipelineTask_SUCCEEDED),
	}, nil))
	var logs bytes.Buffer
	require.NoError(t, coordinator.ReadTaskLog(context.Background(), item.run, preparation.Task.UUID, false, &logs))
	require.Equal(t, "durable output\n", logs.String())
	require.Zero(t, executor.calls)
}

func TestTaskRetryDoesNotPresentAnEarlierAttemptsLogsAsCurrentOutput(t *testing.T) {
	coordinator, item, task, executor := activeLogFixture(t)
	attempt, err := coordinator.runtimeStore.Get(task.UUID)
	require.NoError(t, err)
	attempt.State, attempt.ExecutionReference = model.RuntimeTaskInfrastructureBackoff, nil
	require.NoError(t, coordinator.runtimeStore.Update(attempt, *attempt.AttemptUUID))
	root := t.TempDir()
	path := filepath.Join(root, "previous-attempt.log")
	require.NoError(t, os.WriteFile(path, []byte("earlier attempt output"), 0o600))
	original := viper.Get("ObjectStoreConfig.BucketURL")
	viper.Set("ObjectStoreConfig.BucketURL", "file://"+root)
	t.Cleanup(func() { viper.Set("ObjectStoreConfig.BucketURL", original) })
	uri := "file://" + path
	artifacts := &logArtifactStore{artifact: &model.Artifact{
		UUID: "previous-attempt", Namespace: item.run.Namespace, URI: &uri,
	}}
	coordinator.artifactStore = artifacts
	coordinator.artifactTaskStore = &logArtifactLinks{artifactID: "previous-attempt"}
	var output bytes.Buffer
	require.Error(t, coordinator.ReadTaskLog(context.Background(), item.run, task.UUID, false, &output))
	require.Empty(t, output.String())
	require.Zero(t, executor.calls)
	require.Zero(t, artifacts.calls, "earlier-attempt links remain artifacts, not the current attempt's live logs")
}

func TestTaskLogFileRootRejectsTraversalAndSymlinkEscapes(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.log")
	require.NoError(t, os.WriteFile(outside, []byte("outside"), 0o600))
	original := viper.Get("ObjectStoreConfig.BucketURL")
	viper.Set("ObjectStoreConfig.BucketURL", "file://"+root)
	t.Cleanup(func() { viper.Set("ObjectStoreConfig.BucketURL", original) })
	require.ErrorContains(t, validateTaskLogURI("file://"+outside), "outside")
	link := filepath.Join(root, "linked.log")
	require.NoError(t, os.Symlink(outside, link))
	require.ErrorContains(t, validateTaskLogURI("file://"+link), "outside")
	require.ErrorContains(t, validateTaskLogURI("file://"+root), "regular file")
}
