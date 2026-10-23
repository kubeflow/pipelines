// Copyright 2025 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
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
	"fmt"
	"io"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient/kfpapi"
	"github.com/kubeflow/pipelines/backend/src/v2/client_manager"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
	"k8s.io/client-go/kubernetes/fake"
)

// newStageTestLauncher builds a launcher wired to a mock API with a single
// RUNNING task already created, ready for stage-reporting tests.
func newStageTestLauncher(t *testing.T) (*LauncherV2, *kfpapi.MockAPI, *apiv2.PipelineTask) {
	t.Helper()
	mockAPI := kfpapi.NewMockAPI()
	run := &apiv2.Run{RunId: "run-id"}
	mockAPI.AddRun(run)
	task := &apiv2.PipelineTask{
		TaskId: "task-id",
		RunId:  run.GetRunId(),
		Name:   "trainer",
		State:  apiv2.PipelineTask_RUNNING,
		Type:   apiv2.PipelineTask_RUNTIME,
	}
	_, err := mockAPI.CreateTask(context.Background(), &apiv2.CreateTaskRequest{
		RunId: run.GetRunId(),
		Task:  task,
	})
	require.NoError(t, err)

	launcher := &LauncherV2{
		options:       LauncherV2Options{Run: run, Task: task},
		clientManager: client_manager.NewFakeClientManager(fake.NewSimpleClientset(), mockAPI),
		batchUpdater:  NewBatchUpdater(),
	}
	return launcher, mockAPI, task
}

func taskStages(t *testing.T, task *apiv2.PipelineTask) map[string]*structpb.Value {
	t.Helper()
	return task.GetStatusMetadata().GetCustomProperties()[stagesCustomPropertyKey].GetStructValue().GetFields()
}

func TestReadTaskStages_ValidJSONL(t *testing.T) {
	launcher, mockAPI, task := newStageTestLauncher(t)
	r, w, err := os.Pipe()
	require.NoError(t, err)
	_, err = w.WriteString(`{"stage":"load","message":"Loaded 50k rows","state":"COMPLETED"}` + "\n")
	require.NoError(t, err)
	_, err = w.WriteString(`{"stage":"train","message":"Epoch 7/50","state":"RUNNING","progress":0.14}` + "\n")
	require.NoError(t, err)
	require.NoError(t, w.Close())

	launcher.readTaskStages(context.Background(), r)
	require.NoError(t, r.Close())

	stages := taskStages(t, task)
	require.Len(t, stages, 2)

	load := stages["load"].GetStructValue().GetFields()
	assert.Equal(t, "Loaded 50k rows", load["message"].GetStringValue())
	assert.Equal(t, "COMPLETED", load["state"].GetStringValue())

	train := stages["train"].GetStructValue().GetFields()
	assert.Equal(t, "RUNNING", train["state"].GetStringValue())
	assert.InDelta(t, 0.14, train["progress"].GetNumberValue(), 1e-9)
	// The stage name is the map key and must not be duplicated in the detail.
	_, hasStageField := train[stageNameField]
	assert.False(t, hasStageField)

	// The accumulated stages were persisted through UpdateTask for live visibility.
	persisted, err := mockAPI.GetTask(context.Background(), &apiv2.GetTaskRequest{
		TaskId: task.GetTaskId(),
		RunId:  task.GetRunId(),
	})
	require.NoError(t, err)
	assert.Len(t, taskStages(t, persisted), 2)
}

func TestReadTaskStages_DiscardsMalformedAndUnnamedLines(t *testing.T) {
	launcher, _, task := newStageTestLauncher(t)
	r, w, err := os.Pipe()
	require.NoError(t, err)
	lines := []string{
		`not json at all`,
		`{"message":"missing stage field","state":"RUNNING"}`,
		`{"stage":"","message":"empty stage name"}`,
		`{"stage":123,"message":"stage is not a string"}`,
		``, // blank line is ignored
		`{"stage":"ok","message":"good","state":"RUNNING"}`,
	}
	for _, line := range lines {
		_, err = w.WriteString(line + "\n")
		require.NoError(t, err)
	}
	require.NoError(t, w.Close())

	launcher.readTaskStages(context.Background(), r)
	require.NoError(t, r.Close())

	stages := taskStages(t, task)
	require.Len(t, stages, 1)
	require.Contains(t, stages, "ok")
}

func TestReadTaskStages_UpsertsStageByName(t *testing.T) {
	launcher, _, task := newStageTestLauncher(t)
	r, w, err := os.Pipe()
	require.NoError(t, err)
	for i, state := range []string{"RUNNING", "RUNNING", "COMPLETED"} {
		_, err = w.WriteString(fmt.Sprintf(`{"stage":"train","message":"step %d","state":%q}`+"\n", i, state))
		require.NoError(t, err)
	}
	require.NoError(t, w.Close())

	launcher.readTaskStages(context.Background(), r)
	require.NoError(t, r.Close())

	stages := taskStages(t, task)
	require.Len(t, stages, 1)
	train := stages["train"].GetStructValue().GetFields()
	assert.Equal(t, "COMPLETED", train["state"].GetStringValue())
	assert.Equal(t, "step 2", train["message"].GetStringValue())
}

func TestReadTaskStages_PreservesExistingCustomProperties(t *testing.T) {
	launcher, _, task := newStageTestLauncher(t)
	task.StatusMetadata = &apiv2.PipelineTask_StatusMetadata{
		CustomProperties: map[string]*structpb.Value{
			"driver_logs_uri": structpb.NewStringValue("s3://root/driver-logs.txt"),
		},
	}
	r, w, err := os.Pipe()
	require.NoError(t, err)
	_, err = w.WriteString(`{"stage":"load","message":"Loaded","state":"COMPLETED"}` + "\n")
	require.NoError(t, err)
	require.NoError(t, w.Close())

	launcher.readTaskStages(context.Background(), r)
	require.NoError(t, r.Close())

	props := task.GetStatusMetadata().GetCustomProperties()
	assert.Equal(t, "s3://root/driver-logs.txt", props["driver_logs_uri"].GetStringValue())
	require.Contains(t, props, stagesCustomPropertyKey)
	assert.Contains(t, props[stagesCustomPropertyKey].GetStructValue().GetFields(), "load")
}

func TestReadTaskStages_ReturnsOnEOFWithNoData(t *testing.T) {
	launcher, _, task := newStageTestLauncher(t)
	r, w, err := os.Pipe()
	require.NoError(t, err)
	require.NoError(t, w.Close()) // immediate EOF

	done := make(chan struct{})
	go func() {
		launcher.readTaskStages(context.Background(), r)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("readTaskStages did not return on EOF")
	}
	require.NoError(t, r.Close())
	assert.Nil(t, task.GetStatusMetadata().GetCustomProperties()[stagesCustomPropertyKey])
}

// TestReadTaskStages_DrainsBufferedLinesAfterClose mirrors the launcher's
// production lifecycle: the reader runs concurrently while lines are written,
// and must drain every buffered line after the write-end is closed.
func TestReadTaskStages_DrainsBufferedLinesAfterClose(t *testing.T) {
	launcher, _, task := newStageTestLauncher(t)
	r, w, err := os.Pipe()
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		launcher.readTaskStages(context.Background(), r)
	}()

	const count = 25
	for i := 0; i < count; i++ {
		_, err = w.WriteString(fmt.Sprintf(`{"stage":"s%d","message":"m","state":"COMPLETED"}`+"\n", i))
		require.NoError(t, err)
	}
	require.NoError(t, w.Close())
	wg.Wait()
	require.NoError(t, r.Close())

	assert.Len(t, taskStages(t, task), count)
}

// TestReadTaskStages_StagesSurviveFailedFinalize verifies the failure path: once
// the reader has accumulated stages onto the in-memory task, a failed
// finalizeExecution adds the error message in place without discarding them.
func TestReadTaskStages_StagesSurviveFailedFinalize(t *testing.T) {
	launcher, mockAPI, task := newStageTestLauncher(t)
	// A minimal, parseable pipeline spec lets finalizeExecution's status
	// propagation run its happy path for this childless root task.
	spec, err := structpb.NewStruct(map[string]interface{}{"root": map[string]interface{}{}})
	require.NoError(t, err)
	launcher.pipelineSpec = spec
	task.ParentTaskId = nil

	r, w, err := os.Pipe()
	require.NoError(t, err)
	_, err = w.WriteString(`{"stage":"load","message":"Loaded","state":"COMPLETED"}` + "\n")
	require.NoError(t, err)
	_, err = w.WriteString(`{"stage":"train","message":"Crashed at epoch 3","state":"FAILED"}` + "\n")
	require.NoError(t, err)
	require.NoError(t, w.Close())

	launcher.readTaskStages(context.Background(), r)
	require.NoError(t, r.Close())

	finalizeErr := launcher.finalizeExecution(context.Background(), errors.New("boom"))
	require.Error(t, finalizeErr)
	assert.Contains(t, finalizeErr.Error(), "boom")

	persisted, err := mockAPI.GetTask(context.Background(), &apiv2.GetTaskRequest{
		TaskId: task.GetTaskId(),
		RunId:  task.GetRunId(),
	})
	require.NoError(t, err)
	assert.Equal(t, apiv2.PipelineTask_FAILED, persisted.GetState())
	assert.Contains(t, persisted.GetStatusMetadata().GetMessage(), "boom")
	stages := taskStages(t, persisted)
	require.Len(t, stages, 2)
	assert.Equal(t, "FAILED", stages["train"].GetStructValue().GetFields()["state"].GetStringValue())
}

// TestExecute_WiresStageReportingPipe exercises execute() end to end: the command
// executor receives the write-end as its single ExtraFiles entry (FD 3 in a real
// child) and anything written there is read back onto the task's StatusMetadata.
func TestExecute_WiresStageReportingPipe(t *testing.T) {
	launcher, _, task := newStageTestLauncher(t)
	launcher.fileSystem = NewMockFileSystem()
	launcher.objectStore = NewMockObjectStoreClient()
	launcher.executorInput = &pipelinespec.ExecutorInput{
		Outputs: &pipelinespec.ExecutorInput_Outputs{OutputFile: "/tmp/kfp_outputs/output_metadata.json"},
	}

	mockCmd := NewMockCommandExecutor()
	mockCmd.RunFunc = func(ctx context.Context, cmd string, args []string, stdin io.Reader, stdout, stderr io.Writer, extraFiles []*os.File) error {
		require.Len(t, extraFiles, 1)
		_, err := extraFiles[0].WriteString(`{"stage":"train","message":"Epoch 1","state":"RUNNING"}` + "\n")
		require.NoError(t, err)
		return nil
	}
	launcher.cmdExecutor = mockCmd

	_, err := launcher.execute(context.Background(), "python", []string{"train.py"})
	require.NoError(t, err)

	stages := taskStages(t, task)
	require.Contains(t, stages, "train")
	assert.Equal(t, "Epoch 1", stages["train"].GetStructValue().GetFields()["message"].GetStringValue())
}
