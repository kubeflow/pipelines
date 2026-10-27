// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package component

import (
	"context"
	"errors"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiV2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient"
	"github.com/kubeflow/pipelines/backend/src/v2/client_manager"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
)

type runtimePublicationStore struct {
	ObjectStoreClientInterface
	upload func(context.Context, string, string, string) error
}

func (s *runtimePublicationStore) UploadArtifact(ctx context.Context, local, remote, key string) error {
	return s.upload(ctx, local, remote, key)
}

func newRuntimePublicationLauncher(t *testing.T) *LauncherV2 {
	t.Helper()
	t.Setenv(apiclient.KFPTaskAttemptIDEnvVar, "attempt")
	t.Setenv(apiclient.KFPTaskIDEnvVar, "plugin-task")
	root := t.TempDir()
	outputFile := filepath.Join(root, "executor-output.json")
	artifactFile := filepath.Join(root, "dataset")
	input := &pipelinespec.ExecutorInput{Outputs: &pipelinespec.ExecutorInput_Outputs{
		OutputFile: outputFile,
		Artifacts: map[string]*pipelinespec.ArtifactList{"dataset": {Artifacts: []*pipelinespec.RuntimeArtifact{{
			Uri: "s3://bucket/output/dataset", CustomPath: &artifactFile,
			Type: &pipelinespec.ArtifactTypeSchema{Kind: &pipelinespec.ArtifactTypeSchema_SchemaTitle{SchemaTitle: "system.Artifact"}},
		}}}},
	}}
	launcher := newLauncherForPluginLifecycleTestWithIO(t, &recordingPluginDispatcher{}, nil, input, []byte("{}"))
	launcher.fileSystem.(*MockFileSystem).SetFileContent(outputFile, []byte("{}"))
	launcher.fileSystem.(*MockFileSystem).SetFileContent(artifactFile, []byte("staged component output"))
	return launcher
}

func restartRuntimePublicationLauncher(t *testing.T, previous *LauncherV2) *LauncherV2 {
	t.Helper()
	options := previous.options
	task, err := previous.clientManager.KFPAPIClient().GetTask(context.Background(), &apiV2.GetTaskRequest{
		RunId: options.Run.GetRunId(), TaskId: options.Task.GetTaskId(),
	})
	require.NoError(t, err)
	options.Task = task
	input, err := protojson.Marshal(previous.executorInput)
	require.NoError(t, err)
	launcher, err := NewLauncherV2(string(input), append([]string{previous.command}, previous.args...), &options, previous.clientManager)
	require.NoError(t, err)
	return launcher.WithFileSystem(previous.fileSystem).WithCommandExecutor(previous.cmdExecutor).WithObjectStore(previous.objectStore)
}

func requireRuntimeAPIReport(t *testing.T, launcher *LauncherV2) *RuntimeLauncherReport {
	t.Helper()
	task, err := launcher.clientManager.KFPAPIClient().GetTask(context.Background(), &apiV2.GetTaskRequest{
		RunId: launcher.options.Run.GetRunId(), TaskId: launcher.options.Task.GetTaskId(),
	})
	require.NoError(t, err)
	assert.Equal(t, apiV2.PipelineTask_RUNNING, task.GetState(), "only the framework may publish terminal task state")
	report, err := RuntimeLauncherReportForTask(task)
	require.NoError(t, err)
	require.NotNil(t, report)
	assert.True(t, report.ComputationCompleted)
	return report
}

func TestRuntimePublicationRetriesWithoutReexecutingComputation(t *testing.T) {
	launcher := newRuntimePublicationLauncher(t)
	store := NewMockObjectStoreClient()
	uploads := 0
	launcher.objectStore = &runtimePublicationStore{ObjectStoreClientInterface: store,
		upload: func(ctx context.Context, local, remote, key string) error {
			report := requireRuntimeAPIReport(t, launcher)
			assert.Equal(t, RuntimePhasePublishing, report.Phase)
			assert.False(t, report.PublicationCompleted)
			uploads++
			if uploads == 1 {
				return syscall.ECONNRESET
			}
			return store.UploadArtifact(ctx, local, remote, key)
		}}
	require.NoError(t, launcher.Execute(context.Background()))
	assert.Equal(t, 1, launcher.cmdExecutor.(*MockCommandExecutor).CallCount())
	assert.Equal(t, 2, uploads)
	assert.Equal(t, 1, store.RefreshCalls)
	report := requireRuntimeAPIReport(t, launcher)
	assert.True(t, report.PublicationCompleted)
	assert.Equal(t, RuntimePhasePublished, report.Phase)
	assert.Equal(t, 2, report.PublicationAttempts)
	restarted := restartRuntimePublicationLauncher(t, launcher)
	require.NoError(t, restarted.Execute(context.Background()))
	assert.Equal(t, 1, restarted.cmdExecutor.(*MockCommandExecutor).CallCount())
	assert.Equal(t, 2, uploads)
}

func TestRuntimePublicationResumesAfterObservationLossAndRestart(t *testing.T) {
	launcher := newRuntimePublicationLauncher(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	uploads := 0
	launcher.objectStore = &runtimePublicationStore{ObjectStoreClientInterface: NewMockObjectStoreClient(),
		upload: func(context.Context, string, string, string) error {
			uploads++
			if uploads == 1 {
				cancel()
				return context.Canceled
			}
			return nil
		}}
	require.ErrorIs(t, launcher.Execute(ctx), context.Canceled)
	report := requireRuntimeAPIReport(t, launcher)
	assert.Equal(t, RuntimePhasePublishing, report.Phase)
	assert.Equal(t, 1, report.PublicationAttempts)
	restarted := restartRuntimePublicationLauncher(t, launcher)
	require.NoError(t, restarted.Execute(context.Background()))
	assert.Equal(t, 1, restarted.cmdExecutor.(*MockCommandExecutor).CallCount())
	assert.Equal(t, 2, uploads)
	assert.True(t, requireRuntimeAPIReport(t, restarted).PublicationCompleted)
}

func TestRuntimePublicationResumesFromAPIWhenLocalCheckpointIsUnavailable(t *testing.T) {
	launcher := newRuntimePublicationLauncher(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	uploads := 0
	launcher.objectStore = &runtimePublicationStore{ObjectStoreClientInterface: NewMockObjectStoreClient(),
		upload: func(context.Context, string, string, string) error {
			uploads++
			if uploads == 1 {
				cancel()
				return context.Canceled
			}
			return nil
		}}
	require.ErrorIs(t, launcher.Execute(ctx), context.Canceled)
	fresh := NewMockFileSystem()
	outputFile := launcher.executorInput.GetOutputs().GetOutputFile()
	metadata, err := launcher.fileSystem.ReadFile(outputFile)
	require.NoError(t, err)
	fresh.SetFileContent(outputFile, metadata)
	artifact := launcher.executorInput.GetOutputs().GetArtifacts()["dataset"].Artifacts[0]
	staging, err := launcher.fileSystem.ReadFile(artifact.GetCustomPath())
	require.NoError(t, err)
	fresh.SetFileContent(artifact.GetCustomPath(), staging)
	launcher.fileSystem = fresh
	restarted := restartRuntimePublicationLauncher(t, launcher)
	require.NoError(t, restarted.Execute(context.Background()))
	assert.Equal(t, 1, restarted.cmdExecutor.(*MockCommandExecutor).CallCount())
	assert.Equal(t, 2, uploads)
	assert.True(t, requireRuntimeAPIReport(t, restarted).PublicationCompleted)
}

func TestRuntimePublicationExhaustionNeverReexecutesComputation(t *testing.T) {
	launcher := newRuntimePublicationLauncher(t)
	store := NewMockObjectStoreClient()
	store.UploadError = syscall.ECONNRESET
	launcher.objectStore = store
	require.ErrorContains(t, launcher.Execute(context.Background()), "publication failed after successful computation")
	assert.Equal(t, RuntimePublicationMaxAttempts, len(store.UploadCalls))
	assert.Equal(t, 1, launcher.cmdExecutor.(*MockCommandExecutor).CallCount())
	report := requireRuntimeAPIReport(t, launcher)
	assert.Equal(t, RuntimePhasePublicationFailed, report.Phase)
	assert.Equal(t, "publication", report.FailureClass)
	assert.False(t, report.PublicationCompleted)
	restarted := restartRuntimePublicationLauncher(t, launcher)
	require.ErrorContains(t, restarted.Execute(context.Background()), "retained publication failure")
	assert.Equal(t, 1, restarted.cmdExecutor.(*MockCommandExecutor).CallCount())
	assert.Equal(t, RuntimePublicationMaxAttempts, len(store.UploadCalls))
	artifact := restarted.executorInput.GetOutputs().GetArtifacts()["dataset"].Artifacts[0]
	bytes, err := restarted.fileSystem.ReadFile(artifact.GetCustomPath())
	require.NoError(t, err)
	assert.Equal(t, "staged component output", string(bytes))
	termination, err := restarted.fileSystem.ReadFile(RuntimeLauncherTerminationPath)
	require.NoError(t, err)
	summary, err := ParseRuntimeLauncherReport(string(termination))
	require.NoError(t, err)
	assert.Equal(t, "publication", summary.FailureClass)
	assert.True(t, summary.ComputationCompleted)
	assert.Empty(t, summary.ExecutorInputJSON)
	assert.Empty(t, summary.ExecutorOutputJSON)
	assert.Less(t, len(termination), 4096)
}

func TestRuntimePublicationExhaustedAttemptCannotReplayComponent(t *testing.T) {
	launcher := newRuntimePublicationLauncher(t)
	store := NewMockObjectStoreClient()
	store.UploadError = syscall.ECONNRESET
	launcher.objectStore = store
	require.Error(t, launcher.Execute(context.Background()))
	restarted := restartRuntimePublicationLauncher(t, launcher)
	require.Error(t, restarted.Execute(context.Background()))
	assert.Equal(t, 1, restarted.cmdExecutor.(*MockCommandExecutor).CallCount(),
		"publication exhaustion and launcher restart must never rerun successful computation")
}

func TestRuntimePublicationCheckpointFailureDoesNotUploadOrReplay(t *testing.T) {
	launcher := newRuntimePublicationLauncher(t)
	store := NewMockObjectStoreClient()
	launcher.objectStore = store
	api := &finalizationFailureAPI{API: launcher.clientManager.KFPAPIClient(), updateTaskErr: errors.New("checkpoint API unavailable")}
	launcher.clientManager = client_manager.NewFakeClientManager(launcher.clientManager.K8sClient(), api)
	require.ErrorContains(t, launcher.Execute(context.Background()), "checkpoint API unavailable")
	assert.Empty(t, store.UploadCalls, "publication must not start before a durable computation API report")
	assert.Equal(t, 1, launcher.cmdExecutor.(*MockCommandExecutor).CallCount())
	api.updateTaskErr = nil
	restarted := restartRuntimePublicationLauncher(t, launcher)
	require.ErrorContains(t, restarted.Execute(context.Background()), "retained publication failure")
	assert.Equal(t, 1, restarted.cmdExecutor.(*MockCommandExecutor).CallCount())
}

func TestRuntimePublicationLocalCheckpointFailureStillPersistsComputation(t *testing.T) {
	launcher := newRuntimePublicationLauncher(t)
	store := NewMockObjectStoreClient()
	launcher.objectStore = store
	launcher.fileSystem.(*MockFileSystem).WriteFileError = errors.New("checkpoint volume is full")
	require.ErrorContains(t, launcher.Execute(context.Background()), "checkpoint volume is full")
	assert.Empty(t, store.UploadCalls)
	report := requireRuntimeAPIReport(t, launcher)
	assert.Equal(t, RuntimePhasePublicationFailed, report.Phase)
	assert.Equal(t, "publication", report.FailureClass)
	assert.Equal(t, 1, launcher.cmdExecutor.(*MockCommandExecutor).CallCount())
	launcher.fileSystem.(*MockFileSystem).WriteFileError = nil
	restarted := restartRuntimePublicationLauncher(t, launcher)
	require.ErrorContains(t, restarted.Execute(context.Background()), "retained publication failure")
	assert.Equal(t, 1, restarted.cmdExecutor.(*MockCommandExecutor).CallCount())
}
