// Copyright 2023 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package component

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient/kfpapi"
	"github.com/kubeflow/pipelines/backend/src/v2/client_manager"
	"github.com/kubeflow/pipelines/backend/src/v2/common/plugins"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/structpb"
	"k8s.io/client-go/kubernetes/fake"
)

var addNumbersComponent = &pipelinespec.ComponentSpec{
	Implementation: &pipelinespec.ComponentSpec_ExecutorLabel{ExecutorLabel: "add"},
	InputDefinitions: &pipelinespec.ComponentInputsSpec{
		Parameters: map[string]*pipelinespec.ComponentInputsSpec_ParameterSpec{
			"a": {ParameterType: pipelinespec.ParameterType_NUMBER_INTEGER, DefaultValue: structpb.NewNumberValue(5)},
			"b": {ParameterType: pipelinespec.ParameterType_NUMBER_INTEGER},
		},
	},
	OutputDefinitions: &pipelinespec.ComponentOutputsSpec{
		Parameters: map[string]*pipelinespec.ComponentOutputsSpec_ParameterSpec{
			"Output": {ParameterType: pipelinespec.ParameterType_NUMBER_INTEGER},
		},
	},
}

type finalizationFailureAPI struct {
	kfpapi.API
	updateTasksBulkErr error
	updateTaskErr      error
	getRunErr          error
	updateStatusesErr  error
}

func (api *finalizationFailureAPI) UpdateTasksBulk(
	ctx context.Context,
	req *apiv2beta1.UpdateTasksBulkRequest,
) (*apiv2beta1.UpdateTasksBulkResponse, error) {
	if api.updateTasksBulkErr != nil {
		return nil, api.updateTasksBulkErr
	}
	return api.API.UpdateTasksBulk(ctx, req)
}

func (api *finalizationFailureAPI) UpdateTask(
	ctx context.Context,
	req *apiv2beta1.UpdateTaskRequest,
) (*apiv2beta1.PipelineTask, error) {
	if api.updateTaskErr != nil {
		return nil, api.updateTaskErr
	}
	return api.API.UpdateTask(ctx, req)
}

func (api *finalizationFailureAPI) GetRun(
	ctx context.Context,
	req *apiv2beta1.GetRunRequest,
) (*apiv2beta1.Run, error) {
	if api.getRunErr != nil {
		return nil, api.getRunErr
	}
	return api.API.GetRun(ctx, req)
}

func (api *finalizationFailureAPI) UpdateStatuses(
	ctx context.Context,
	run *apiv2beta1.Run,
	pipelineSpec *structpb.Struct,
	currentTask *apiv2beta1.PipelineTask,
) error {
	if api.updateStatusesErr != nil {
		return api.updateStatusesErr
	}
	return api.API.UpdateStatuses(ctx, run, pipelineSpec, currentTask)
}

func TestFinalizeExecutionReturnsPersistenceFailures(t *testing.T) {
	tests := []struct {
		name               string
		updateTasksBulkErr error
		updateTaskErr      error
		getRunErr          error
		updateStatusesErr  error
		expectedErrors     []string
		// expectNotSucceeded asserts the persisted task is no longer SUCCEEDED
		// after a finalization failure that still managed to force a FAILED update.
		expectNotSucceeded bool
	}{
		{
			name:               "batch flush",
			updateTasksBulkErr: errors.New("flush failed"),
			expectedErrors:     []string{"failed to flush batch updates", "flush failed"},
			expectNotSucceeded: true,
		},
		{
			name:               "batch flush and fallback update",
			updateTasksBulkErr: errors.New("flush failed"),
			updateTaskErr:      errors.New("fallback failed"),
			expectedErrors:     []string{"flush failed", "failed to persist task", "fallback failed"},
		},
		{
			name:               "run refresh",
			getRunErr:          errors.New("refresh failed"),
			expectedErrors:     []string{"failed to refresh run", "refresh failed"},
			expectNotSucceeded: true,
		},
		{
			name:               "status propagation",
			updateStatusesErr:  errors.New("propagation failed"),
			expectedErrors:     []string{"failed to update statuses", "propagation failed"},
			expectNotSucceeded: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			baseAPI := kfpapi.NewMockAPI()
			run := &apiv2beta1.Run{
				RunId: "run",
				PipelineSource: &apiv2beta1.Run_PipelineSpec{
					PipelineSpec: &structpb.Struct{},
				},
			}
			baseAPI.AddRun(run)
			task := &apiv2beta1.PipelineTask{
				TaskId: "task",
				RunId:  run.GetRunId(),
				State:  apiv2beta1.PipelineTask_SUCCEEDED,
			}
			_, err := baseAPI.CreateTask(context.Background(), &apiv2beta1.CreateTaskRequest{
				Task:  task,
				RunId: run.GetRunId(),
			})
			require.NoError(t, err)

			failingAPI := &finalizationFailureAPI{
				API:                baseAPI,
				updateTasksBulkErr: test.updateTasksBulkErr,
				updateTaskErr:      test.updateTaskErr,
				getRunErr:          test.getRunErr,
				updateStatusesErr:  test.updateStatusesErr,
			}
			launcher := &LauncherV2{
				options: LauncherV2Options{
					Run:  run,
					Task: task,
				},
				clientManager: client_manager.NewFakeClientManager(fake.NewSimpleClientset(), failingAPI),
				pipelineSpec:  &structpb.Struct{},
				batchUpdater:  NewBatchUpdater(),
			}

			err = launcher.finalizeExecution(context.Background(), nil)

			require.Error(t, err)
			for _, expectedError := range test.expectedErrors {
				assert.Contains(t, err.Error(), expectedError)
			}
			if test.expectNotSucceeded {
				persistedTask, getErr := baseAPI.GetTask(context.Background(), &apiv2beta1.GetTaskRequest{
					TaskId: task.GetTaskId(),
					RunId:  run.GetRunId(),
				})
				require.NoError(t, getErr)
				assert.NotEqual(t, apiv2beta1.PipelineTask_SUCCEEDED, persistedTask.GetState())
				assert.Equal(t, apiv2beta1.PipelineTask_FAILED, persistedTask.GetState())
			}
		})
	}
}

func TestPropagateOutputsUpDAGForTask_UsesExplicitDependencies(t *testing.T) {
	pipelineSpec := &pipelinespec.PipelineSpec{
		Root: &pipelinespec.ComponentSpec{
			Implementation: &pipelinespec.ComponentSpec_Dag{
				Dag: &pipelinespec.DagSpec{
					Tasks: map[string]*pipelinespec.PipelineTaskSpec{
						"worker": {
							TaskInfo:     &pipelinespec.PipelineTaskInfo{Name: "worker"},
							ComponentRef: &pipelinespec.ComponentRef{Name: "worker-comp"},
						},
					},
					Outputs: &pipelinespec.DagOutputsSpec{
						Parameters: map[string]*pipelinespec.DagOutputsSpec_DagOutputParameterSpec{
							"pipeline-output": {
								Kind: &pipelinespec.DagOutputsSpec_DagOutputParameterSpec_ValueFromParameter{
									ValueFromParameter: &pipelinespec.DagOutputsSpec_ParameterSelectorSpec{
										ProducerSubtask:    "worker",
										OutputParameterKey: "result",
									},
								},
							},
						},
					},
				},
			},
			OutputDefinitions: &pipelinespec.ComponentOutputsSpec{
				Parameters: map[string]*pipelinespec.ComponentOutputsSpec_ParameterSpec{
					"pipeline-output": {ParameterType: pipelinespec.ParameterType_STRING},
				},
			},
		},
		Components: map[string]*pipelinespec.ComponentSpec{
			"worker-comp": {
				Implementation: &pipelinespec.ComponentSpec_ExecutorLabel{ExecutorLabel: "worker"},
				OutputDefinitions: &pipelinespec.ComponentOutputsSpec{
					Parameters: map[string]*pipelinespec.ComponentOutputsSpec_ParameterSpec{
						"result": {ParameterType: pipelinespec.ParameterType_STRING},
					},
				},
			},
		},
	}
	pipelineSpecStruct, err := pipelineSpecToStruct(t, pipelineSpec)
	require.NoError(t, err)
	scopePath, err := util.ScopePathFromStringPathWithNewTask(pipelineSpecStruct, "root", "worker")
	require.NoError(t, err)

	run := &apiv2beta1.Run{RunId: "run-id"}
	rootTask := &apiv2beta1.PipelineTask{
		TaskId:    "root-task",
		RunId:     run.GetRunId(),
		Name:      "root",
		State:     apiv2beta1.PipelineTask_RUNNING,
		Type:      apiv2beta1.PipelineTask_DAG,
		ScopePath: "root",
	}
	childTask := &apiv2beta1.PipelineTask{
		TaskId:    "worker-task",
		RunId:     run.GetRunId(),
		Name:      "worker",
		State:     apiv2beta1.PipelineTask_SUCCEEDED,
		Type:      apiv2beta1.PipelineTask_RUNTIME,
		ScopePath: scopePath.DotNotation(),
		Outputs: &apiv2beta1.PipelineTask_InputOutputs{
			Parameters: []*apiv2beta1.PipelineTask_InputOutputs_IOParameter{{
				ParameterKey: "result",
				Value:        structpb.NewStringValue("done"),
				Type:         apiv2beta1.IOType_OUTPUT,
				Producer:     &apiv2beta1.IOProducer{TaskName: "worker"},
			}},
		},
	}

	mockAPI := kfpapi.NewMockAPI()
	mockAPI.AddRun(run)
	_, err = mockAPI.CreateTask(context.Background(), &apiv2beta1.CreateTaskRequest{
		RunId: run.GetRunId(),
		Task:  rootTask,
	})
	require.NoError(t, err)
	_, err = mockAPI.CreateTask(context.Background(), &apiv2beta1.CreateTaskRequest{
		RunId: run.GetRunId(),
		Task:  childTask,
	})
	require.NoError(t, err)

	clientManager := client_manager.NewFakeClientManager(fake.NewClientset(), mockAPI)
	err = PropagateOutputsUpDAGForTask(context.Background(), OutputPropagationOptions{
		Run:          run,
		Task:         childTask,
		ParentTask:   rootTask,
		ScopePath:    scopePath,
		PipelineSpec: pipelineSpecStruct,
	}, clientManager)
	require.NoError(t, err)

	updatedRootTask, err := mockAPI.GetTask(context.Background(), &apiv2beta1.GetTaskRequest{
		TaskId: rootTask.GetTaskId(),
		RunId:  run.GetRunId(),
	})
	require.NoError(t, err)
	require.Len(t, updatedRootTask.GetOutputs().GetParameters(), 1)

	outputParam := updatedRootTask.GetOutputs().GetParameters()[0]
	assert.Equal(t, "pipeline-output", outputParam.GetParameterKey())
	assert.Equal(t, "done", outputParam.GetValue().GetStringValue())
	assert.Equal(t, apiv2beta1.IOType_OUTPUT, outputParam.GetType())
	require.NotNil(t, outputParam.GetProducer())
	assert.Equal(t, "worker", outputParam.GetProducer().GetTaskName())
}

func TestPropagateOutputsUpDAGForTask_OmitsLinksPersistedByPriorAttempt(t *testing.T) {
	pipelineSpec := &pipelinespec.PipelineSpec{
		Root: &pipelinespec.ComponentSpec{
			Implementation: &pipelinespec.ComponentSpec_Dag{
				Dag: &pipelinespec.DagSpec{
					Tasks: map[string]*pipelinespec.PipelineTaskSpec{
						"worker": {
							TaskInfo:     &pipelinespec.PipelineTaskInfo{Name: "worker"},
							ComponentRef: &pipelinespec.ComponentRef{Name: "worker-comp"},
						},
					},
					Outputs: &pipelinespec.DagOutputsSpec{
						Artifacts: map[string]*pipelinespec.DagOutputsSpec_DagOutputArtifactSpec{
							"pipeline-model": {
								ArtifactSelectors: []*pipelinespec.DagOutputsSpec_ArtifactSelectorSpec{{
									ProducerSubtask:   "worker",
									OutputArtifactKey: "model",
								}},
							},
						},
					},
				},
			},
			OutputDefinitions: &pipelinespec.ComponentOutputsSpec{
				Artifacts: map[string]*pipelinespec.ComponentOutputsSpec_ArtifactSpec{
					"pipeline-model": {IsArtifactList: true},
				},
			},
		},
		Components: map[string]*pipelinespec.ComponentSpec{
			"worker-comp": {
				Implementation: &pipelinespec.ComponentSpec_ExecutorLabel{ExecutorLabel: "worker"},
				OutputDefinitions: &pipelinespec.ComponentOutputsSpec{
					Artifacts: map[string]*pipelinespec.ComponentOutputsSpec_ArtifactSpec{
						"model": {IsArtifactList: true},
					},
				},
			},
		},
	}
	pipelineSpecStruct, err := pipelineSpecToStruct(t, pipelineSpec)
	require.NoError(t, err)
	scopePath, err := util.ScopePathFromStringPathWithNewTask(pipelineSpecStruct, "root", "worker")
	require.NoError(t, err)

	run := &apiv2beta1.Run{RunId: "run-retry"}
	parentTask := &apiv2beta1.PipelineTask{
		TaskId:    "root-task",
		RunId:     run.GetRunId(),
		Name:      "root",
		State:     apiv2beta1.PipelineTask_RUNNING,
		Type:      apiv2beta1.PipelineTask_DAG,
		ScopePath: "root",
	}
	childTask := &apiv2beta1.PipelineTask{
		TaskId:       "worker-task",
		RunId:        run.GetRunId(),
		Name:         "worker",
		State:        apiv2beta1.PipelineTask_SUCCEEDED,
		Type:         apiv2beta1.PipelineTask_RUNTIME,
		ScopePath:    scopePath.DotNotation(),
		ParentTaskId: util.StringPointer(parentTask.GetTaskId()),
	}

	baseAPI := kfpapi.NewMockAPI()
	mockAPI := &uniqueLinkEnforcingMockAPI{MockAPI: baseAPI, seen: map[string]struct{}{}}
	mockAPI.AddRun(run)
	_, err = mockAPI.CreateTask(context.Background(), &apiv2beta1.CreateTaskRequest{
		RunId: run.GetRunId(),
		Task:  parentTask,
	})
	require.NoError(t, err)
	_, err = mockAPI.CreateTask(context.Background(), &apiv2beta1.CreateTaskRequest{
		RunId: run.GetRunId(),
		Task:  childTask,
	})
	require.NoError(t, err)

	const (
		priorArtifactID   = "artifact-prior-attempt"
		currentArtifactID = "artifact-current-attempt"
	)
	for _, artifactID := range []string{priorArtifactID, currentArtifactID} {
		_, err = mockAPI.CreateArtifact(context.Background(), &apiv2beta1.CreateArtifactRequest{
			Artifact:    &apiv2beta1.Artifact{ArtifactId: artifactID, Name: "model", Uri: util.StringPointer("s3://bucket/" + artifactID)},
			TaskId:      childTask.GetTaskId(),
			RunId:       run.GetRunId(),
			ProducerKey: "model",
		})
		require.NoError(t, err)
	}
	_, err = mockAPI.CreateArtifactTasks(context.Background(), &apiv2beta1.CreateArtifactTasksBulkRequest{
		ArtifactTasks: []*apiv2beta1.ArtifactTask{{
			ArtifactId: priorArtifactID,
			TaskId:     parentTask.GetTaskId(),
			RunId:      run.GetRunId(),
			Key:        "pipeline-model",
			Type:       apiv2beta1.IOType_OUTPUT,
			Producer:   &apiv2beta1.IOProducer{TaskName: "worker"},
		}},
	})
	require.NoError(t, err)

	clientManager := client_manager.NewFakeClientManager(fake.NewClientset(), mockAPI)
	err = PropagateOutputsUpDAGForTask(context.Background(), OutputPropagationOptions{
		Run:          run,
		Task:         childTask,
		ParentTask:   parentTask,
		ScopePath:    scopePath,
		PipelineSpec: pipelineSpecStruct,
	}, clientManager)
	require.NoError(t, err)

	updatedParent, err := mockAPI.GetTask(context.Background(), &apiv2beta1.GetTaskRequest{
		TaskId: parentTask.GetTaskId(),
		RunId:  run.GetRunId(),
	})
	require.NoError(t, err)
	require.Len(t, updatedParent.GetOutputs().GetArtifacts(), 1)
	artifacts := updatedParent.GetOutputs().GetArtifacts()[0].GetArtifacts()
	require.Len(t, artifacts, 2)
	assert.ElementsMatch(t, []string{priorArtifactID, currentArtifactID}, []string{
		artifacts[0].GetArtifactId(), artifacts[1].GetArtifactId(),
	})
}

type transientArtifactUploadError struct {
	message string
}

func (e transientArtifactUploadError) Error() string {
	return e.message
}

func (e transientArtifactUploadError) Temporary() bool {
	return true
}

func (e transientArtifactUploadError) Timeout() bool {
	return false
}

// Example_launcherV2WithMocks demonstrates how to test LauncherV2.Execute with all dependencies mocked.
// This example shows the complete pattern for component-level testing.
func TestExample_launcherV2WithMocks(t *testing.T) {
	// Step 1: Create mock KFP API
	mockAPI := kfpapi.NewMockAPI()

	// Step 2: Create test run and task
	runID := "test-run-123"
	taskID := "test-task-456"

	run := &apiv2beta1.Run{
		RunId:       runID,
		DisplayName: "test-run",
		State:       apiv2beta1.RuntimeState_RUNNING,
		PipelineSource: &apiv2beta1.Run_PipelineSpec{
			PipelineSpec: &structpb.Struct{},
		},
		Tasks: []*apiv2beta1.PipelineTask{},
	}
	mockAPI.AddRun(run)

	task := &apiv2beta1.PipelineTask{
		TaskId:  taskID,
		RunId:   runID,
		Name:    "test-task",
		State:   apiv2beta1.PipelineTask_RUNNING,
		Type:    apiv2beta1.PipelineTask_RUNTIME,
		Inputs:  &apiv2beta1.PipelineTask_InputOutputs{},
		Outputs: &apiv2beta1.PipelineTask_InputOutputs{},
	}

	// Step 3: Create executor input with inputs and outputs
	executorInput := &pipelinespec.ExecutorInput{
		Inputs: &pipelinespec.ExecutorInput_Inputs{
			ParameterValues: map[string]*structpb.Value{
				"input_param": structpb.NewStringValue("test_value"),
			},
			Artifacts: map[string]*pipelinespec.ArtifactList{
				"input_data": {
					Artifacts: []*pipelinespec.RuntimeArtifact{
						{
							Name: "dataset",
							Uri:  "s3://bucket/input/data.csv",
							Type: &pipelinespec.ArtifactTypeSchema{
								Kind: &pipelinespec.ArtifactTypeSchema_SchemaTitle{
									SchemaTitle: "system.Dataset",
								},
							},
						},
					},
				},
			},
		},
		Outputs: &pipelinespec.ExecutorInput_Outputs{
			Parameters: map[string]*pipelinespec.ExecutorInput_OutputParameter{
				"output_metric": {
					OutputFile: "/tmp/outputs/output_metric",
				},
			},
			Artifacts: map[string]*pipelinespec.ArtifactList{
				"model": {
					Artifacts: []*pipelinespec.RuntimeArtifact{
						{
							Name: "trained-model",
							Uri:  "s3://bucket/output/model.pkl",
							Type: &pipelinespec.ArtifactTypeSchema{
								Kind: &pipelinespec.ArtifactTypeSchema_SchemaTitle{
									SchemaTitle: "system.Model",
								},
							},
						},
					},
				},
			},
			OutputFile: "/tmp/kfp_outputs/output_metadata.json",
		},
	}

	executorInputJSON, _ := protojson.Marshal(executorInput)

	// Step 4: Create component spec
	componentSpec := &pipelinespec.ComponentSpec{
		InputDefinitions: &pipelinespec.ComponentInputsSpec{
			Parameters: map[string]*pipelinespec.ComponentInputsSpec_ParameterSpec{
				"input_param": {
					ParameterType: pipelinespec.ParameterType_STRING,
				},
			},
		},
		OutputDefinitions: &pipelinespec.ComponentOutputsSpec{
			Parameters: map[string]*pipelinespec.ComponentOutputsSpec_ParameterSpec{
				"output_metric": {
					ParameterType: pipelinespec.ParameterType_NUMBER_DOUBLE,
				},
			},
		},
	}

	// Step 5: Create task spec
	taskSpec := &pipelinespec.PipelineTaskSpec{
		TaskInfo: &pipelinespec.PipelineTaskInfo{
			Name: "train-model",
		},
	}

	// Step 6: Create launcher options
	opts := &LauncherV2Options{
		Namespace:     "default",
		PodName:       "train-model-pod",
		PodUID:        "pod-uid-123",
		PipelineName:  "training-pipeline",
		PublishLogs:   "false",
		ComponentSpec: componentSpec,
		TaskSpec:      taskSpec,
		ScopePath:     util.ScopePath{},
		Run:           run,
		Task:          task,
		PipelineSpec:  &structpb.Struct{},
	}

	// Step 7: Create launcher with client manager
	clientManager := client_manager.NewFakeClientManager(fake.NewClientset(), mockAPI)
	launcher, err := NewLauncherV2(
		string(executorInputJSON),
		[]string{"python", "train.py", "--data", "{{$.inputs.artifacts['input_data'].path}}"},
		opts,
		clientManager,
	)
	require.NoError(t, err)

	// Step 8: Setup mocks for dependencies
	mockFS := NewMockFileSystem()
	mockCmd := NewMockCommandExecutor()
	mockObjStore := NewMockObjectStoreClient()

	// Configure file system with output data
	mockFS.SetFileContent("/tmp/outputs/output_metric", []byte("0.95"))
	mockFS.SetFileContent("/tmp/kfp_outputs/output_metadata.json", []byte("{}"))

	// Configure object store with input data
	mockObjStore.SetArtifact("s3://bucket/input/data.csv", []byte("col1,col2\n1,2\n"))

	// Configure command executor to succeed
	mockCmd.RunError = nil

	// Step 9: Inject mocks into launcher
	launcher.WithFileSystem(mockFS).
		WithCommandExecutor(mockCmd).
		WithObjectStore(mockObjStore)

	// Step 10: Execute the launcher's internal execute method
	ctx := context.Background()
	executorOutput, err := launcher.execute(ctx, "python", []string{"train.py"})
	require.NotNil(t, executorOutput)
	if err != nil {
		panic(err)
	}

	// Output: Test passed - launcher executed successfully with mocked dependencies
	println("Test passed - launcher executed successfully with mocked dependencies")
}

// TestLauncherV2_ArtifactHandling demonstrates testing artifact download and upload
func TestLauncherV2_ArtifactHandling(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockObjStore := NewMockObjectStoreClient()

	// Simulate pre-existing input artifact
	mockObjStore.SetArtifact("s3://bucket/input/dataset.csv", []byte("training,data"))

	// Test download
	err := mockObjStore.DownloadArtifact(ctx, "s3://bucket/input/dataset.csv", "/local/dataset.csv", "input_data")
	require.NoError(t, err)

	// Verify download was called with correct parameters
	assert.Len(t, mockObjStore.DownloadCalls, 1)
	assert.Equal(t, "input_data", mockObjStore.DownloadCalls[0].ArtifactKey)
	assert.Equal(t, "s3://bucket/input/dataset.csv", mockObjStore.DownloadCalls[0].RemoteURI)
	assert.Equal(t, "/local/dataset.csv", mockObjStore.DownloadCalls[0].LocalPath)

	// Test upload
	err = mockObjStore.UploadArtifact(ctx, "/local/model.pkl", "s3://bucket/output/model.pkl", "model_output")
	require.NoError(t, err)

	// Verify upload was called
	assert.Len(t, mockObjStore.UploadCalls, 1)
	assert.Equal(t, "model_output", mockObjStore.UploadCalls[0].ArtifactKey)

	// Verify artifact can be queried
	modelUploads := mockObjStore.GetUploadCallsForKey("model_output")
	assert.Len(t, modelUploads, 1)
	assert.Equal(t, "s3://bucket/output/model.pkl", modelUploads[0].RemoteURI)
}

// TestLauncherV2_CommandExecution demonstrates testing command execution
func TestLauncherV2_CommandExecution(t *testing.T) {
	mockCmd := NewMockCommandExecutor()

	// Setup custom behavior to write to stdout
	mockCmd.RunFunc = func(ctx context.Context, cmd string, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
		// Simulate successful execution
		stdout.Write([]byte("Training completed successfully\n"))
		stdout.Write([]byte("Accuracy: 0.95\n"))
		return nil
	}

	// Execute command
	ctx := context.Background()
	var stdout, stderr bytes.Buffer
	err := mockCmd.Run(ctx, "python", []string{"train.py"}, nil, &stdout, &stderr)

	// Verify
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Training completed successfully")
	assert.Contains(t, stdout.String(), "Accuracy: 0.95")

	// Verify command was called correctly
	assert.Equal(t, 1, mockCmd.CallCount())
	assert.Equal(t, "python", mockCmd.RunCalls[0].Cmd)
	assert.Equal(t, []string{"train.py"}, mockCmd.RunCalls[0].Args)
}

// TestLauncherV2_FileSystemOperations demonstrates testing file system operations
func TestLauncherV2_FileSystemOperations(t *testing.T) {
	mockFS := NewMockFileSystem()

	// Test directory creation
	err := mockFS.MkdirAll("/tmp/outputs", 0755)
	require.NoError(t, err)

	// Test file writing
	err = mockFS.WriteFile("/tmp/outputs/metrics.json", []byte(`{"accuracy": 0.95}`), 0644)
	require.NoError(t, err)

	// Test file reading
	content, err := mockFS.ReadFile("/tmp/outputs/metrics.json")
	require.NoError(t, err)
	assert.Equal(t, `{"accuracy": 0.95}`, string(content))

	// Verify all operations were tracked
	assert.Len(t, mockFS.MkdirAllCalls, 1)
	assert.Equal(t, "/tmp/outputs", mockFS.MkdirAllCalls[0].Path)

	assert.Len(t, mockFS.WriteFileCalls, 1)
	assert.Equal(t, "/tmp/outputs/metrics.json", mockFS.WriteFileCalls[0].Name)

	assert.Len(t, mockFS.ReadFileCalls, 1)
	assert.Equal(t, "/tmp/outputs/metrics.json", mockFS.ReadFileCalls[0])
}

// TestLauncherV2_TaskStatusUpdates demonstrates testing KFP API task updates
func TestLauncherV2_TaskStatusUpdates(t *testing.T) {
	// Create mock API
	mockAPI := kfpapi.NewMockAPI()

	// Create test run
	run := &apiv2beta1.Run{
		RunId:       "run-123",
		DisplayName: "test-run",
		State:       apiv2beta1.RuntimeState_RUNNING,
		PipelineSource: &apiv2beta1.Run_PipelineSpec{
			PipelineSpec: &structpb.Struct{},
		},
	}
	mockAPI.AddRun(run)

	// Create test task
	task := &apiv2beta1.PipelineTask{
		TaskId: "task-456",
		RunId:  "run-123",
		Name:   "test-task",
		State:  apiv2beta1.PipelineTask_RUNNING,
	}
	_, err := mockAPI.CreateTask(context.Background(), &apiv2beta1.CreateTaskRequest{Task: task, RunId: task.GetRunId()})
	require.NoError(t, err)

	// Update task status
	task.State = apiv2beta1.PipelineTask_SUCCEEDED
	_, err = mockAPI.UpdateTask(context.Background(), &apiv2beta1.UpdateTaskRequest{
		TaskId: "task-456",
		Task:   task,
	})
	require.NoError(t, err)

	// Verify task was updated
	updatedTask, err := mockAPI.GetTask(context.Background(), &apiv2beta1.GetTaskRequest{TaskId: "task-456"})
	require.NoError(t, err)
	assert.Equal(t, apiv2beta1.PipelineTask_SUCCEEDED, updatedTask.State)
}

// Tests that launcher correctly executes the user component and successfully writes output parameters to file.
func Test_execute_Parameters(t *testing.T) {
	tests := []struct {
		name          string
		executorInput *pipelinespec.ExecutorInput
		executorArgs  []string
		wantErr       bool
	}{
		{
			"happy pass",
			&pipelinespec.ExecutorInput{
				Inputs: &pipelinespec.ExecutorInput_Inputs{
					ParameterValues: map[string]*structpb.Value{"a": structpb.NewNumberValue(1), "b": structpb.NewNumberValue(2)},
				},
			},
			[]string{"-c", "test {{$.inputs.parameters['a']}} -eq 1 || exit 1\ntest {{$.inputs.parameters['b']}} -eq 2 || exit 1"},
			false,
		},
		{
			"use default value",
			&pipelinespec.ExecutorInput{
				Inputs: &pipelinespec.ExecutorInput_Inputs{
					ParameterValues: map[string]*structpb.Value{"b": structpb.NewNumberValue(2)},
				},
			},
			[]string{"-c", "test {{$.inputs.parameters['a']}} -eq 5 || exit 1\ntest {{$.inputs.parameters['b']}} -eq 2 || exit 1"},
			false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Setup executor input with outputs section
			test.executorInput.Outputs = &pipelinespec.ExecutorInput_Outputs{
				OutputFile: "/tmp/kfp_outputs/output_metadata.json",
			}

			// Marshal executor input
			executorInputJSON, err := protojson.Marshal(test.executorInput)
			assert.Nil(t, err)

			// Create mock dependencies
			mockAPI := kfpapi.NewMockAPI()
			clientManager := client_manager.NewFakeClientManager(fake.NewClientset(), mockAPI)

			// Create test run and task
			run := &apiv2beta1.Run{
				RunId:       "test-run",
				DisplayName: "test-run",
				State:       apiv2beta1.RuntimeState_RUNNING,
				PipelineSource: &apiv2beta1.Run_PipelineSpec{
					PipelineSpec: &structpb.Struct{},
				},
			}
			mockAPI.AddRun(run)

			task := &apiv2beta1.PipelineTask{
				TaskId:  "test-task",
				RunId:   "test-run",
				Name:    "test-task",
				State:   apiv2beta1.PipelineTask_RUNNING,
				Inputs:  &apiv2beta1.PipelineTask_InputOutputs{},
				Outputs: &apiv2beta1.PipelineTask_InputOutputs{},
			}

			// Create launcher options
			opts := &LauncherV2Options{
				Namespace:     "namespace",
				PodName:       "test-pod",
				PodUID:        "test-uid",
				PipelineName:  "test-pipeline",
				ComponentSpec: addNumbersComponent,
				Run:           run,
				Task:          task,
				PipelineSpec:  &structpb.Struct{},
			}

			// Create launcher
			launcher, err := NewLauncherV2(
				string(executorInputJSON),
				append([]string{"sh"}, test.executorArgs...),
				opts,
				clientManager,
			)
			assert.Nil(t, err)

			// Setup mocks
			mockFS := NewMockFileSystem()
			mockCmd := NewMockCommandExecutor()
			mockObjStore := NewMockObjectStoreClient()

			mockFS.SetFileContent("/tmp/kfp_outputs/output_metadata.json", []byte("{}"))
			mockCmd.RunError = nil

			launcher.WithFileSystem(mockFS).
				WithCommandExecutor(mockCmd).
				WithObjectStore(mockObjStore)

			// Execute
			_, err = launcher.execute(context.Background(), "sh", test.executorArgs)

			if test.wantErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func Test_getPlaceholders_WorkspaceArtifactPath(t *testing.T) {
	execIn := &pipelinespec.ExecutorInput{
		Inputs: &pipelinespec.ExecutorInput_Inputs{
			Artifacts: map[string]*pipelinespec.ArtifactList{
				"data": {
					Artifacts: []*pipelinespec.RuntimeArtifact{
						{Uri: "minio://mlpipeline/sample/sample.txt", Metadata: &structpb.Struct{Fields: map[string]*structpb.Value{"_kfp_workspace": structpb.NewBoolValue(true)}}},
					},
				},
			},
		},
	}
	ph, err := getPlaceholders(execIn)
	if err != nil {
		t.Fatalf("getPlaceholders error: %v", err)
	}
	actual := ph["{{$.inputs.artifacts['data'].path}}"]
	expected := filepath.Join(WorkspaceMountPath, ".artifacts", "minio", "mlpipeline", "sample", "sample.txt")
	if actual != expected {
		t.Fatalf("placeholder path mismatch: actual=%q expected=%q", actual, expected)
	}
}

func Test_getPlaceholders_CustomPath(t *testing.T) {
	customPath := "/mnt/custom/output/file"
	execIn := &pipelinespec.ExecutorInput{
		Outputs: &pipelinespec.ExecutorInput_Outputs{
			Artifacts: map[string]*pipelinespec.ArtifactList{
				"model": {
					Artifacts: []*pipelinespec.RuntimeArtifact{
						{
							Uri:        "s3://bucket/path/to/model",
							CustomPath: &customPath,
						},
					},
				},
			},
		},
	}
	ph, err := getPlaceholders(execIn)
	require.NoError(t, err)
	assert.Equal(t, "s3://bucket/path/to/model", ph["{{$.outputs.artifacts['model'].uri}}"])
	assert.Equal(t, customPath, ph["{{$.outputs.artifacts['model'].path}}"])
}

func Test_prepareOutputFolders_CustomPath(t *testing.T) {
	customPath := "/mnt/custom/output/file"
	execIn := &pipelinespec.ExecutorInput{
		Outputs: &pipelinespec.ExecutorInput_Outputs{
			Artifacts: map[string]*pipelinespec.ArtifactList{
				"model": {
					Artifacts: []*pipelinespec.RuntimeArtifact{
						{
							Uri:        "s3://bucket/path/to/model",
							CustomPath: &customPath,
						},
					},
				},
			},
		},
	}
	mockFS := NewMockFileSystem()
	launcher := &LauncherV2{fileSystem: mockFS}
	require.NoError(t, launcher.prepareOutputFolders(execIn))
	require.Len(t, mockFS.MkdirAllCalls, 1)
	assert.Equal(t, filepath.Dir(customPath), mockFS.MkdirAllCalls[0].Path)
}

func Test_executorInput_compileCmdAndArgs(t *testing.T) {
	executorInputJSON := `{
		"inputs": {
			"parameterValues": {
				"config": {
					"category_ids": "{{$.inputs.parameters['pipelinechannel--category_ids']}}",
					"dump_filename": "{{$.inputs.parameters['pipelinechannel--dump_filename']}}",
					"sphinx_host": "{{$.inputs.parameters['pipelinechannel--sphinx_host']}}",
					"sphinx_port": "{{$.inputs.parameters['pipelinechannel--sphinx_port']}}"
				},
				"pipelinechannel--category_ids": "116",
				"pipelinechannel--dump_filename": "dump_filename_test.txt",
				"pipelinechannel--sphinx_host": "sphinx-default-host.ru",
				"pipelinechannel--sphinx_port": 9312
			}
		},
		"outputs": {
			"artifacts": {
				"dataset": {
					"artifacts": [{
						"type": {
							"schemaTitle": "system.Dataset",
							"schemaVersion": "0.0.1"
						},
						"uri": "s3://aviflow-stage-kfp-artifacts/debug-component-pipeline/ae02034e-bd96-4b8a-a06b-55c99fe9eccb/sayhello/c98ac032-2448-4637-bf37-3ad1e13a112c/dataset"
					}]
				}
			},
			"outputFile": "/tmp/kfp_outputs/output_metadata.json"
		}
	}`

	executorInput := &pipelinespec.ExecutorInput{}
	err := protojson.Unmarshal([]byte(executorInputJSON), executorInput)

	assert.NoError(t, err)

	cmd := "sh"
	args := []string{
		"--executor_input", "{{$}}",
		"--function_to_execute", "sayHello",
	}
	_, args, err = compileCmdAndArgs(executorInput, cmd, args)
	assert.NoError(t, err)

	var actualExecutorInput string
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "--executor_input" {
			actualExecutorInput = args[i+1]
			break
		}
	}
	assert.NotEmpty(t, actualExecutorInput, "--executor_input not found")

	var parsed map[string]any
	err = json.Unmarshal([]byte(actualExecutorInput), &parsed)
	assert.NoError(t, err)

	inputs := parsed["inputs"].(map[string]any)
	paramValues := inputs["parameterValues"].(map[string]any)
	config := paramValues["config"].(map[string]any)

	assert.Equal(t, "116", config["category_ids"])
	assert.Equal(t, "dump_filename_test.txt", config["dump_filename"])
	assert.Equal(t, "sphinx-default-host.ru", config["sphinx_host"])
	assert.Equal(t, "9312", config["sphinx_port"])
}

func Test_compileCmdAndArgs_ReplacesCommandAndComplexArgsPlaceholders(t *testing.T) {
	executorInput := &pipelinespec.ExecutorInput{
		Inputs: &pipelinespec.ExecutorInput_Inputs{
			ParameterValues: map[string]*structpb.Value{
				"entrypoint": structpb.NewStringValue("python"),
				"list_arg": structpb.NewListValue(&structpb.ListValue{
					Values: []*structpb.Value{
						structpb.NewStringValue("a"),
						structpb.NewStringValue("b"),
					},
				}),
				"struct_arg": structpb.NewStructValue(&structpb.Struct{
					Fields: map[string]*structpb.Value{
						"alpha": structpb.NewStringValue("beta"),
					},
				}),
			},
		},
	}

	cmd, args, err := compileCmdAndArgs(
		executorInput,
		"{{$.inputs.parameters['entrypoint']}}",
		[]string{
			"--items={{$.inputs.parameters['list_arg']}}",
			"--config={{$.inputs.parameters['struct_arg']}}",
		},
	)
	require.NoError(t, err)
	assert.Equal(t, "python", cmd)
	assert.Equal(t, []string{
		`--items=["a","b"]`,
		`--config={"alpha":"beta"}`,
	}, args)
}

// Tests executeV2 flow including parameter collection, artifact uploads, and task updates
func Test_executeV2(t *testing.T) {
	// Create component spec with input/output parameters and artifacts
	componentSpec := &pipelinespec.ComponentSpec{
		InputDefinitions: &pipelinespec.ComponentInputsSpec{
			Parameters: map[string]*pipelinespec.ComponentInputsSpec_ParameterSpec{
				"input_param": {
					ParameterType: pipelinespec.ParameterType_STRING,
				},
				"optional_param": {
					ParameterType: pipelinespec.ParameterType_NUMBER_INTEGER,
					DefaultValue:  structpb.NewNumberValue(42),
				},
			},
		},
		OutputDefinitions: &pipelinespec.ComponentOutputsSpec{
			Parameters: map[string]*pipelinespec.ComponentOutputsSpec_ParameterSpec{
				"output_metric": {
					ParameterType: pipelinespec.ParameterType_NUMBER_DOUBLE,
				},
				"output_message": {
					ParameterType: pipelinespec.ParameterType_STRING,
				},
			},
			Artifacts: map[string]*pipelinespec.ComponentOutputsSpec_ArtifactSpec{
				"model": {
					ArtifactType: &pipelinespec.ArtifactTypeSchema{
						Kind: &pipelinespec.ArtifactTypeSchema_SchemaTitle{
							SchemaTitle: "system.Model",
						},
					},
				},
			},
		},
	}

	// Create executor input with parameters (intentionally omitting optional_param to test defaults)
	executorInput := &pipelinespec.ExecutorInput{
		Inputs: &pipelinespec.ExecutorInput_Inputs{
			ParameterValues: map[string]*structpb.Value{
				"input_param": structpb.NewStringValue("test_value"),
			},
		},
		Outputs: &pipelinespec.ExecutorInput_Outputs{
			Parameters: map[string]*pipelinespec.ExecutorInput_OutputParameter{
				"output_metric": {
					OutputFile: "/tmp/outputs/output_metric",
				},
				"output_message": {
					OutputFile: "/tmp/outputs/output_message",
				},
			},
			Artifacts: map[string]*pipelinespec.ArtifactList{
				"model": {
					Artifacts: []*pipelinespec.RuntimeArtifact{
						{
							Name: "trained-model",
							Uri:  "s3://bucket/output/model.pkl",
							Type: &pipelinespec.ArtifactTypeSchema{
								Kind: &pipelinespec.ArtifactTypeSchema_SchemaTitle{
									SchemaTitle: "system.Model",
								},
							},
						},
					},
				},
			},
			OutputFile: "/tmp/kfp_outputs/output_metadata.json",
		},
	}

	executorInputJSON, err := protojson.Marshal(executorInput)
	assert.NoError(t, err)

	// Create mock dependencies
	mockAPI := kfpapi.NewMockAPI()
	clientManager := client_manager.NewFakeClientManager(fake.NewClientset(), mockAPI)

	// Create test run
	run := &apiv2beta1.Run{
		RunId:       "test-run-123",
		DisplayName: "test-run",
		State:       apiv2beta1.RuntimeState_RUNNING,
		PipelineSource: &apiv2beta1.Run_PipelineSpec{
			PipelineSpec: &structpb.Struct{},
		},
		Tasks: []*apiv2beta1.PipelineTask{},
	}
	mockAPI.AddRun(run)

	// Create test task with canonical Name distinct from TaskInfo DisplayName.
	const (
		canonicalTaskName = "train-model"
		displayTaskName   = "Train Model Display"
	)
	task := &apiv2beta1.PipelineTask{
		TaskId:  "test-task-456",
		RunId:   "test-run-123",
		Name:    canonicalTaskName,
		State:   apiv2beta1.PipelineTask_RUNNING,
		Type:    apiv2beta1.PipelineTask_RUNTIME,
		Inputs:  &apiv2beta1.PipelineTask_InputOutputs{},
		Outputs: &apiv2beta1.PipelineTask_InputOutputs{},
	}

	// Add task to mock API so it can be updated during execution
	_, err = mockAPI.CreateTask(context.Background(), &apiv2beta1.CreateTaskRequest{Task: task, RunId: task.GetRunId()})
	assert.NoError(t, err)

	// Create task spec with a display name that differs from the canonical DAG key.
	taskSpec := &pipelinespec.PipelineTaskSpec{
		TaskInfo: &pipelinespec.PipelineTaskInfo{
			Name: displayTaskName,
		},
	}

	// Create launcher options
	opts := &LauncherV2Options{
		Namespace:     "default",
		PodName:       "train-model-pod",
		PodUID:        "pod-uid-123",
		PipelineName:  "training-pipeline",
		ComponentSpec: componentSpec,
		TaskSpec:      taskSpec,
		Run:           run,
		Task:          task,
		PipelineSpec:  &structpb.Struct{},
	}

	// Create launcher
	launcher, err := NewLauncherV2(
		string(executorInputJSON),
		[]string{"python", "train.py"},
		opts,
		clientManager,
	)
	assert.NoError(t, err)

	// Setup mocks
	mockFS := NewMockFileSystem()
	mockCmd := NewMockCommandExecutor()
	mockObjStore := NewMockObjectStoreClient()

	// Configure file system with output parameter values
	mockFS.SetFileContent("/tmp/outputs/output_metric", []byte("0.95"))
	mockFS.SetFileContent("/tmp/outputs/output_message", []byte("Training completed successfully"))
	mockFS.SetFileContent("/tmp/kfp_outputs/output_metadata.json", []byte("{}"))

	// Configure command executor to succeed
	mockCmd.RunError = nil

	// Inject mocks
	launcher.WithFileSystem(mockFS).
		WithCommandExecutor(mockCmd).
		WithObjectStore(mockObjStore)

	// Execute executeV2 via ExecuteForTesting
	ctx := context.Background()
	executorOutput, err := launcher.ExecuteForTesting(ctx)

	// Verify execution succeeded
	assert.NoError(t, err)
	assert.NotNil(t, executorOutput)

	// Verify output parameters were collected
	assert.Contains(t, executorOutput.ParameterValues, "output_metric")
	assert.Contains(t, executorOutput.ParameterValues, "output_message")
	assert.Equal(t, 0.95, executorOutput.ParameterValues["output_metric"].GetNumberValue())
	assert.Equal(t, "Training completed successfully", executorOutput.ParameterValues["output_message"].GetStringValue())

	// Verify IOProducer.TaskName uses canonical task Name, not TaskInfo DisplayName.
	updatedTask, err := mockAPI.GetTask(ctx, &apiv2beta1.GetTaskRequest{TaskId: task.GetTaskId(), RunId: task.GetRunId()})
	require.NoError(t, err)
	require.NotEmpty(t, updatedTask.GetOutputs().GetParameters())
	for _, param := range updatedTask.GetOutputs().GetParameters() {
		require.NotNil(t, param.GetProducer())
		assert.Equal(t, canonicalTaskName, param.GetProducer().GetTaskName(),
			"IOProducer.TaskName must be canonical Name, not DisplayName %q", displayTaskName)
		assert.NotEqual(t, displayTaskName, param.GetProducer().GetTaskName())
	}

	// Verify artifact was uploaded to object store
	assert.True(t, mockObjStore.WasUploaded("s3://bucket/output/model.pkl"), "Expected model artifact to be uploaded")

	// Verify batch updater queued artifact creation and task updates
	metrics := launcher.batchUpdater.GetMetrics()
	assert.Greater(t, metrics["queued_artifacts"], 0, "Expected artifacts to be queued for creation")
	assert.Greater(t, metrics["queued_task_updates"], 0, "Expected task updates to be queued")
}

func Test_executeV2_FailsWhenDeclaredOutputArtifactFileIsMissing(t *testing.T) {
	componentSpec := &pipelinespec.ComponentSpec{
		OutputDefinitions: &pipelinespec.ComponentOutputsSpec{
			Artifacts: map[string]*pipelinespec.ComponentOutputsSpec_ArtifactSpec{
				"model": {
					ArtifactType: &pipelinespec.ArtifactTypeSchema{
						Kind: &pipelinespec.ArtifactTypeSchema_SchemaTitle{
							SchemaTitle: "system.Model",
						},
					},
				},
			},
		},
	}
	executorInput := &pipelinespec.ExecutorInput{
		Outputs: &pipelinespec.ExecutorInput_Outputs{
			Artifacts: map[string]*pipelinespec.ArtifactList{
				"model": {
					Artifacts: []*pipelinespec.RuntimeArtifact{
						{
							Name: "trained-model",
							Uri:  "s3://bucket/output/model.pkl",
							Type: &pipelinespec.ArtifactTypeSchema{
								Kind: &pipelinespec.ArtifactTypeSchema_SchemaTitle{
									SchemaTitle: "system.Model",
								},
							},
						},
					},
				},
			},
			OutputFile: "/tmp/kfp_outputs/output_metadata.json",
		},
	}
	executorInputJSON, err := protojson.Marshal(executorInput)
	require.NoError(t, err)

	mockAPI := kfpapi.NewMockAPI()
	clientManager := client_manager.NewFakeClientManager(fake.NewClientset(), mockAPI)
	run := &apiv2beta1.Run{
		RunId: "test-run-123",
		PipelineSource: &apiv2beta1.Run_PipelineSpec{
			PipelineSpec: &structpb.Struct{},
		},
	}
	mockAPI.AddRun(run)
	task := &apiv2beta1.PipelineTask{
		TaskId:  "test-task-456",
		RunId:   "test-run-123",
		Name:    "train-model",
		State:   apiv2beta1.PipelineTask_RUNNING,
		Type:    apiv2beta1.PipelineTask_RUNTIME,
		Inputs:  &apiv2beta1.PipelineTask_InputOutputs{},
		Outputs: &apiv2beta1.PipelineTask_InputOutputs{},
	}
	_, err = mockAPI.CreateTask(context.Background(), &apiv2beta1.CreateTaskRequest{Task: task, RunId: task.GetRunId()})
	require.NoError(t, err)

	launcher, err := NewLauncherV2(
		string(executorInputJSON),
		[]string{"python", "train.py"},
		&LauncherV2Options{
			Namespace:     "default",
			PodName:       "train-model-pod",
			PodUID:        "pod-uid-123",
			PipelineName:  "training-pipeline",
			ComponentSpec: componentSpec,
			TaskSpec:      &pipelinespec.PipelineTaskSpec{TaskInfo: &pipelinespec.PipelineTaskInfo{Name: "train-model"}},
			Run:           run,
			Task:          task,
			PipelineSpec:  &structpb.Struct{},
		},
		clientManager,
	)
	require.NoError(t, err)

	mockFS := NewMockFileSystem()
	mockFS.SetFileContent("/tmp/kfp_outputs/output_metadata.json", []byte("{}"))
	mockCmd := NewMockCommandExecutor()
	mockObjStore := NewMockObjectStoreClient()
	mockObjStore.UploadError = os.ErrNotExist
	launcher.WithFileSystem(mockFS).WithCommandExecutor(mockCmd).WithObjectStore(mockObjStore)

	_, err = launcher.ExecuteForTesting(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "declared output artifact \"model\" is missing")
}

func TestUploadOutputArtifacts_SkipsUnsupportedURIsWithoutUploading(t *testing.T) {
	launcher := &LauncherV2{
		executorInput: &pipelinespec.ExecutorInput{
			Outputs: &pipelinespec.ExecutorInput_Outputs{
				Artifacts: map[string]*pipelinespec.ArtifactList{
					"model": {
						Artifacts: []*pipelinespec.RuntimeArtifact{{
							Name: "trained-model",
							Uri:  "unsupported://bucket/output/model.pkl",
							Type: &pipelinespec.ArtifactTypeSchema{
								Kind: &pipelinespec.ArtifactTypeSchema_SchemaTitle{SchemaTitle: "system.Model"},
							},
						}},
					},
				},
			},
		},
		options: LauncherV2Options{
			Namespace: "default",
			Run:       &apiv2beta1.Run{RunId: "run-1"},
			Task:      &apiv2beta1.PipelineTask{TaskId: "task-1"},
		},
		batchUpdater: NewBatchUpdater(),
		objectStore:  NewMockObjectStoreClient(),
	}

	err := launcher.uploadOutputArtifacts(context.Background(), &pipelinespec.ExecutorOutput{
		Artifacts: map[string]*pipelinespec.ArtifactList{},
	})
	require.NoError(t, err)
	assert.Empty(t, launcher.objectStore.(*MockObjectStoreClient).UploadCalls)
	assert.Zero(t, launcher.batchUpdater.GetMetrics()["queued_artifacts"])
}

func TestUploadOutputArtifactsWithRetry_RetriesTransientUploadFailures(t *testing.T) {
	mockObjectStore := NewMockObjectStoreClient()
	mockObjectStore.UploadErrors = []error{
		transientArtifactUploadError{message: "temporary upload failure"},
		nil,
	}
	launcher := &LauncherV2{
		executorInput: &pipelinespec.ExecutorInput{
			Outputs: &pipelinespec.ExecutorInput_Outputs{
				Artifacts: map[string]*pipelinespec.ArtifactList{
					"model": {
						Artifacts: []*pipelinespec.RuntimeArtifact{{
							Name: "trained-model",
							Uri:  "s3://bucket/output/model.pkl",
							Type: &pipelinespec.ArtifactTypeSchema{
								Kind: &pipelinespec.ArtifactTypeSchema_SchemaTitle{SchemaTitle: "system.Model"},
							},
						}},
					},
				},
			},
		},
		options: LauncherV2Options{
			Namespace: "default",
			Run:       &apiv2beta1.Run{RunId: "run-1"},
			Task:      &apiv2beta1.PipelineTask{TaskId: "task-1"},
		},
		batchUpdater: NewBatchUpdater(),
		objectStore:  mockObjectStore,
	}

	err := launcher.uploadOutputArtifactsWithRetry(context.Background(), &pipelinespec.ExecutorOutput{
		Artifacts: map[string]*pipelinespec.ArtifactList{},
	})
	require.NoError(t, err)
	require.Len(t, mockObjectStore.UploadCalls, 2)
	assert.Equal(t, 1, mockObjectStore.RefreshCalls)
	require.Len(t, launcher.batchUpdater.artifacts, 1)
	assert.Equal(t, "s3://bucket/output/model.pkl", *launcher.batchUpdater.artifacts[0].request.Artifact.Uri)
}

func TestUploadOutputArtifactsWithRetry_DoesNotRetryNonTransientFailures(t *testing.T) {
	mockObjectStore := NewMockObjectStoreClient()
	mockObjectStore.UploadError = errors.New("permanent upload failure")
	launcher := &LauncherV2{
		executorInput: &pipelinespec.ExecutorInput{
			Outputs: &pipelinespec.ExecutorInput_Outputs{
				Artifacts: map[string]*pipelinespec.ArtifactList{
					"model": {
						Artifacts: []*pipelinespec.RuntimeArtifact{{
							Name: "trained-model",
							Uri:  "s3://bucket/output/model.pkl",
							Type: &pipelinespec.ArtifactTypeSchema{
								Kind: &pipelinespec.ArtifactTypeSchema_SchemaTitle{SchemaTitle: "system.Model"},
							},
						}},
					},
				},
			},
		},
		options: LauncherV2Options{
			Namespace: "default",
			Run:       &apiv2beta1.Run{RunId: "run-1"},
			Task:      &apiv2beta1.PipelineTask{TaskId: "task-1"},
		},
		batchUpdater: NewBatchUpdater(),
		objectStore:  mockObjectStore,
	}

	err := launcher.uploadOutputArtifactsWithRetry(context.Background(), &pipelinespec.ExecutorOutput{
		Artifacts: map[string]*pipelinespec.ArtifactList{},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "permanent upload failure")
	require.Len(t, mockObjectStore.UploadCalls, 1)
	assert.Equal(t, 0, mockObjectStore.RefreshCalls)
	assert.Empty(t, launcher.batchUpdater.artifacts)
}

func TestUploadOutputArtifacts_PreservesArtifactListOutputs(t *testing.T) {
	launcher := &LauncherV2{
		executorInput: &pipelinespec.ExecutorInput{
			Outputs: &pipelinespec.ExecutorInput_Outputs{
				Artifacts: map[string]*pipelinespec.ArtifactList{
					"models": {
						Artifacts: []*pipelinespec.RuntimeArtifact{
							{
								Name: "model-0",
								Uri:  "s3://bucket/output/model-0",
								Type: &pipelinespec.ArtifactTypeSchema{
									Kind: &pipelinespec.ArtifactTypeSchema_SchemaTitle{SchemaTitle: "system.Model"},
								},
							},
							{
								Name: "model-1",
								Uri:  "s3://bucket/output/model-1",
								Type: &pipelinespec.ArtifactTypeSchema{
									Kind: &pipelinespec.ArtifactTypeSchema_SchemaTitle{SchemaTitle: "system.Model"},
								},
							},
						},
					},
				},
			},
		},
		options: LauncherV2Options{
			Namespace: "default",
			Run:       &apiv2beta1.Run{RunId: "run-1"},
			Task:      &apiv2beta1.PipelineTask{TaskId: "task-1"},
		},
		batchUpdater: NewBatchUpdater(),
		objectStore:  NewMockObjectStoreClient(),
	}

	metadataZero, err := structpb.NewStruct(map[string]interface{}{"id": "zero"})
	require.NoError(t, err)
	metadataOne, err := structpb.NewStruct(map[string]interface{}{"id": "one"})
	require.NoError(t, err)

	err = launcher.uploadOutputArtifacts(context.Background(), &pipelinespec.ExecutorOutput{
		Artifacts: map[string]*pipelinespec.ArtifactList{
			"models": {
				Artifacts: []*pipelinespec.RuntimeArtifact{
					{Uri: "s3://bucket/output/model-0", Metadata: metadataZero},
					{Uri: "s3://bucket/output/model-1", Metadata: metadataOne},
				},
			},
		},
	})
	require.NoError(t, err)

	mockObjectStore := launcher.objectStore.(*MockObjectStoreClient)
	require.Len(t, mockObjectStore.UploadCalls, 2)
	assert.Equal(t, "s3://bucket/output/model-0", mockObjectStore.UploadCalls[0].RemoteURI)
	assert.Equal(t, "s3://bucket/output/model-1", mockObjectStore.UploadCalls[1].RemoteURI)
	require.Len(t, launcher.batchUpdater.artifacts, 2)
	assert.Equal(t, "zero", launcher.batchUpdater.artifacts[0].request.Artifact.Metadata["id"].GetStringValue())
	assert.Equal(t, "one", launcher.batchUpdater.artifacts[1].request.Artifact.Metadata["id"].GetStringValue())
}

func TestUploadOutputArtifacts_RegistersOCIOutputs(t *testing.T) {
	launcher := &LauncherV2{
		executorInput: &pipelinespec.ExecutorInput{
			Outputs: &pipelinespec.ExecutorInput_Outputs{
				Artifacts: map[string]*pipelinespec.ArtifactList{
					"model": {
						Artifacts: []*pipelinespec.RuntimeArtifact{{
							Name: "trained-model",
							Uri:  "oci://registry.domain.local/org/repo:v1.0",
							Type: &pipelinespec.ArtifactTypeSchema{
								Kind: &pipelinespec.ArtifactTypeSchema_SchemaTitle{SchemaTitle: "system.Model"},
							},
						}},
					},
				},
			},
		},
		options: LauncherV2Options{
			Namespace: "default",
			Run:       &apiv2beta1.Run{RunId: "run-1"},
			Task:      &apiv2beta1.PipelineTask{TaskId: "task-1"},
		},
		batchUpdater: NewBatchUpdater(),
		objectStore:  NewMockObjectStoreClient(),
	}

	err := launcher.uploadOutputArtifacts(context.Background(), &pipelinespec.ExecutorOutput{
		Artifacts: map[string]*pipelinespec.ArtifactList{},
	})
	require.NoError(t, err)
	assert.Empty(t, launcher.objectStore.(*MockObjectStoreClient).UploadCalls)
	require.Len(t, launcher.batchUpdater.artifacts, 1)
	assert.Equal(t, "oci://registry.domain.local/org/repo:v1.0", *launcher.batchUpdater.artifacts[0].request.Artifact.Uri)
}

func TestUploadOutputArtifacts_PreservesCustomSchemaTitle(t *testing.T) {
	launcher := &LauncherV2{
		executorInput: &pipelinespec.ExecutorInput{
			Outputs: &pipelinespec.ExecutorInput_Outputs{
				Artifacts: map[string]*pipelinespec.ArtifactList{
					"vertex-model": {
						Artifacts: []*pipelinespec.RuntimeArtifact{{
							Name: "vertex-model",
							Uri:  "s3://bucket/output/model",
							Type: &pipelinespec.ArtifactTypeSchema{
								Kind: &pipelinespec.ArtifactTypeSchema_SchemaTitle{SchemaTitle: "google.VertexModel"},
							},
						}},
					},
				},
			},
		},
		options: LauncherV2Options{
			Namespace: "default",
			Run:       &apiv2beta1.Run{RunId: "run-1"},
			Task:      &apiv2beta1.PipelineTask{TaskId: "task-1"},
		},
		batchUpdater: NewBatchUpdater(),
		objectStore:  NewMockObjectStoreClient(),
	}

	err := launcher.uploadOutputArtifacts(context.Background(), &pipelinespec.ExecutorOutput{
		Artifacts: map[string]*pipelinespec.ArtifactList{},
	})
	require.NoError(t, err)
	require.Len(t, launcher.batchUpdater.artifacts, 1)
	artifact := launcher.batchUpdater.artifacts[0].request.Artifact
	assert.Equal(t, apiv2beta1.Artifact_Artifact, artifact.GetType())
	require.NotNil(t, artifact.GetMetadata())
	assert.Equal(t, "google.VertexModel", artifact.GetMetadata()[artifactSchemaTitleMetadataKey].GetStringValue())
}

func TestUploadOutputArtifacts_DoesNotLetExecutorLogsOverwriteRetryQualifiedURI(t *testing.T) {
	launcher := &LauncherV2{
		executorInput: &pipelinespec.ExecutorInput{
			Outputs: &pipelinespec.ExecutorInput_Outputs{
				Artifacts: map[string]*pipelinespec.ArtifactList{
					"executor-logs": {
						Artifacts: []*pipelinespec.RuntimeArtifact{{
							Name: "executor-logs",
							Uri:  "minio://bucket/logs/executor-logs-2",
							Type: &pipelinespec.ArtifactTypeSchema{
								Kind: &pipelinespec.ArtifactTypeSchema_SchemaTitle{SchemaTitle: "system.Artifact"},
							},
						}},
					},
				},
			},
		},
		options: LauncherV2Options{
			Namespace: "default",
			Run:       &apiv2beta1.Run{RunId: "run-1"},
			Task:      &apiv2beta1.PipelineTask{TaskId: "task-1"},
		},
		batchUpdater: NewBatchUpdater(),
		objectStore:  NewMockObjectStoreClient(),
	}

	err := launcher.uploadOutputArtifacts(context.Background(), &pipelinespec.ExecutorOutput{
		Artifacts: map[string]*pipelinespec.ArtifactList{
			"executor-logs": {
				Artifacts: []*pipelinespec.RuntimeArtifact{{
					Uri: "minio://bucket/logs/executor-logs",
				}},
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, launcher.batchUpdater.artifacts, 1)
	assert.Equal(t, "minio://bucket/logs/executor-logs-2", *launcher.batchUpdater.artifacts[0].request.Artifact.Uri)
}

func TestUploadExecutorLogsArtifact_RetriesWithSessionRefresh(t *testing.T) {
	mockObjectStore := NewMockObjectStoreClient()
	mockObjectStore.UploadErrors = []error{
		transientArtifactUploadError{message: "temporary log upload failure"},
		nil,
	}
	mockAPI := kfpapi.NewMockAPI()
	launcher := &LauncherV2{
		executorInput: &pipelinespec.ExecutorInput{
			Outputs: &pipelinespec.ExecutorInput_Outputs{
				Artifacts: map[string]*pipelinespec.ArtifactList{
					"executor-logs": {
						Artifacts: []*pipelinespec.RuntimeArtifact{{
							Name: "executor-logs",
							Uri:  "minio://bucket/logs/executor-logs-0",
							Type: &pipelinespec.ArtifactTypeSchema{
								Kind: &pipelinespec.ArtifactTypeSchema_SchemaTitle{SchemaTitle: "system.Artifact"},
							},
						}},
					},
				},
			},
		},
		options: LauncherV2Options{
			Namespace:   "default",
			PublishLogs: "true",
			Run:         &apiv2beta1.Run{RunId: "run-1"},
			Task:        &apiv2beta1.PipelineTask{TaskId: "task-1"},
		},
		clientManager: client_manager.NewFakeClientManager(fake.NewClientset(), mockAPI),
		batchUpdater:  NewBatchUpdater(),
		objectStore:   mockObjectStore,
	}

	err := launcher.uploadExecutorLogsArtifact(context.Background())
	require.NoError(t, err)
	require.Len(t, mockObjectStore.UploadCalls, 2)
	assert.Equal(t, 1, mockObjectStore.RefreshCalls)
}

func Test_get_log_Writer(t *testing.T) {
	old := osCreateFunc
	defer func() { osCreateFunc = old }()

	osCreateFunc = func(name string) (*os.File, error) {
		tmpdir := t.TempDir()
		file, _ := os.CreateTemp(tmpdir, "*")
		return file, nil
	}

	tests := []struct {
		name        string
		artifacts   map[string]*pipelinespec.ArtifactList
		multiWriter bool
	}{
		{
			"single writer - no key logs",
			map[string]*pipelinespec.ArtifactList{
				"notLog": {},
			},
			false,
		},
		{
			"single writer - key log has empty list",
			map[string]*pipelinespec.ArtifactList{
				"logs": {
					Artifacts: []*pipelinespec.RuntimeArtifact{},
				},
			},
			false,
		},
		{
			"single writer - malformed uri",
			map[string]*pipelinespec.ArtifactList{
				"logs": {
					Artifacts: []*pipelinespec.RuntimeArtifact{
						{
							Uri: "",
						},
					},
				},
			},
			false,
		},
		{
			"multiwriter",
			map[string]*pipelinespec.ArtifactList{
				"executor-logs": {
					Artifacts: []*pipelinespec.RuntimeArtifact{
						{
							Uri: "minio://testinguri",
						},
					},
				},
			},
			true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			writer := getLogWriter(test.artifacts)
			if test.multiWriter == false {
				assert.Equal(t, os.Stdout, writer)
			} else {
				assert.IsType(t, io.MultiWriter(), writer)
			}
		})
	}
}

func TestQualifyExecutorLogsForRetry_UsesRetryEnv(t *testing.T) {
	t.Setenv(EnvRetryIndex, "2")
	executorInput := &pipelinespec.ExecutorInput{
		Outputs: &pipelinespec.ExecutorInput_Outputs{
			Artifacts: map[string]*pipelinespec.ArtifactList{
				"executor-logs": {
					Artifacts: []*pipelinespec.RuntimeArtifact{{
						Uri: "minio://bucket/logs/executor-logs",
					}},
				},
			},
		},
	}

	qualifyExecutorLogsForRetry(context.Background(), executorInput, "true", "", nil, "")
	require.Equal(t, "minio://bucket/logs/executor-logs-2", executorInput.GetOutputs().GetArtifacts()["executor-logs"].Artifacts[0].GetUri())
}

// Tests happy and unhappy paths for constructing a new LauncherV2
func Test_NewLauncherV2(t *testing.T) {
	var testCmdArgs = []string{"sh", "-c", "echo \"hello world\""}

	mockAPI := kfpapi.NewMockAPI()
	var testLauncherV2Deps = client_manager.NewFakeClientManager(
		fake.NewSimpleClientset(),
		mockAPI,
	)

	var testValidLauncherV2Opts = LauncherV2Options{
		Namespace:    "my-namespace",
		PodName:      "my-pod",
		PodUID:       "abcd",
		PipelineName: "test-pipeline",
		PipelineSpec: &structpb.Struct{},
	}

	type args struct {
		executorInputJSON string
		cmdArgs           []string
		opts              LauncherV2Options
		cm                client_manager.ClientManagerInterface
	}
	tests := []struct {
		name        string
		args        *args
		expectedErr error
	}{
		{
			name: "happy path",
			args: &args{
				executorInputJSON: "{}",
				cmdArgs:           testCmdArgs,
				opts:              testValidLauncherV2Opts,
				cm:                testLauncherV2Deps,
			},
			expectedErr: nil,
		},
		{
			name: "invalid executorInput",
			args: &args{
				executorInputJSON: "{",
				cmdArgs:           testCmdArgs,
				opts:              testValidLauncherV2Opts,
				cm:                testLauncherV2Deps,
			},
			expectedErr: errors.New("unexpected EOF"),
		},
		{
			name: "missing cmdArgs",
			args: &args{
				executorInputJSON: "{}",
				cmdArgs:           []string{},
				opts:              testValidLauncherV2Opts,
				cm:                testLauncherV2Deps,
			},
			expectedErr: errors.New("command and arguments are empty"),
		},
		{
			name: "invalid opts",
			args: &args{
				executorInputJSON: "{}",
				cmdArgs:           testCmdArgs,
				opts:              LauncherV2Options{},
				cm:                testLauncherV2Deps,
			},
			expectedErr: errors.New("invalid launcher options: must specify Namespace"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			args := test.args
			_, err := NewLauncherV2(args.executorInputJSON, args.cmdArgs, &args.opts, args.cm)
			if test.expectedErr != nil {
				assert.ErrorContains(t, err, test.expectedErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_retrieve_artifact_path(t *testing.T) {
	customPath := "/var/lib/kubelet/pods/pod-uid/volumes/kubernetes.io~csi/pvc-uuid/mount"
	tests := []struct {
		name         string
		artifact     *pipelinespec.RuntimeArtifact
		expectedPath string
	}{
		{
			"Artifact with no custom path",
			&pipelinespec.RuntimeArtifact{
				Uri: "gs://bucket/path/to/artifact",
			},
			"/gcs/bucket/path/to/artifact",
		},
		{
			"Artifact with custom path",
			&pipelinespec.RuntimeArtifact{
				Uri:        "gs://bucket/path/to/artifact",
				CustomPath: &customPath,
			},
			customPath,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path, err := retrieveArtifactPath(test.artifact)
			assert.Nil(t, err)
			assert.Equal(t, path, test.expectedPath)
		})
	}
}

// recordingPluginDispatcher records OnTaskEnd calls for launcher lifecycle tests.
type recordingPluginDispatcher struct {
	plugins.NoOpDispatcher
	endCount      int
	endStates     []apiv2beta1.PipelineTask_TaskState
	endMetrics    []map[string]float64
	endParameters []map[string]interface{}
	endErr        error
}

func (r *recordingPluginDispatcher) OnTaskEnd(_ context.Context, taskInfo *plugins.TaskInfo) error {
	r.endCount++
	if taskInfo != nil {
		r.endStates = append(r.endStates, taskInfo.RunStatus)
		r.endMetrics = append(r.endMetrics, taskInfo.ScalarMetrics)
		r.endParameters = append(r.endParameters, taskInfo.Parameters)
	}
	return r.endErr
}

func pipelineSpecStructForLauncherPluginTest(t *testing.T) *structpb.Struct {
	t.Helper()
	pipelineSpec := &pipelinespec.PipelineSpec{
		PipelineInfo: &pipelinespec.PipelineInfo{Name: "plugin-lifecycle"},
		Root: &pipelinespec.ComponentSpec{
			Implementation: &pipelinespec.ComponentSpec_Dag{
				Dag: &pipelinespec.DagSpec{
					Tasks: map[string]*pipelinespec.PipelineTaskSpec{},
				},
			},
		},
		SchemaVersion: "2.1.0",
	}
	pipelineSpecJSON, err := protojson.Marshal(pipelineSpec)
	require.NoError(t, err)
	pipelineSpecStruct := &structpb.Struct{}
	require.NoError(t, protojson.Unmarshal(pipelineSpecJSON, pipelineSpecStruct))
	return pipelineSpecStruct
}

func newLauncherForPluginLifecycleTest(
	t *testing.T,
	recorder *recordingPluginDispatcher,
	cmdErr error,
) *LauncherV2 {
	t.Helper()
	return newLauncherForPluginLifecycleTestWithIO(t, recorder, cmdErr, nil, []byte("{}"))
}

func newLauncherForPluginLifecycleTestWithIO(
	t *testing.T,
	recorder *recordingPluginDispatcher,
	cmdErr error,
	executorInput *pipelinespec.ExecutorInput,
	outputMetadata []byte,
) *LauncherV2 {
	t.Helper()

	if executorInput == nil {
		executorInput = &pipelinespec.ExecutorInput{
			Outputs: &pipelinespec.ExecutorInput_Outputs{
				OutputFile: "/tmp/kfp_outputs/output_metadata.json",
			},
		}
	}
	executorInputJSON, err := protojson.Marshal(executorInput)
	require.NoError(t, err)

	mockAPI := kfpapi.NewMockAPI()
	clientManager := client_manager.NewFakeClientManager(fake.NewClientset(), mockAPI)

	run := &apiv2beta1.Run{
		RunId: "plugin-run",
		PipelineSource: &apiv2beta1.Run_PipelineSpec{
			PipelineSpec: pipelineSpecStructForLauncherPluginTest(t),
		},
	}
	mockAPI.AddRun(run)

	task := &apiv2beta1.PipelineTask{
		TaskId: "plugin-task",
		RunId:  run.GetRunId(),
		Name:   "plugin-task",
		State:  apiv2beta1.PipelineTask_RUNNING,
		Type:   apiv2beta1.PipelineTask_RUNTIME,
	}
	_, err = mockAPI.CreateTask(context.Background(), &apiv2beta1.CreateTaskRequest{
		Task: task, RunId: task.GetRunId(),
	})
	require.NoError(t, err)

	opts := &LauncherV2Options{
		Namespace:        "default",
		PodName:          "plugin-pod",
		PodUID:           "plugin-pod-uid",
		PipelineName:     "plugin-lifecycle",
		ComponentSpec:    &pipelinespec.ComponentSpec{},
		TaskSpec:         &pipelinespec.PipelineTaskSpec{TaskInfo: &pipelinespec.PipelineTaskInfo{Name: "plugin-task"}},
		Run:              run,
		Task:             task,
		PipelineSpec:     pipelineSpecStructForLauncherPluginTest(t),
		PluginDispatcher: recorder,
	}

	launcher, err := NewLauncherV2(string(executorInputJSON), []string{"sh", "-c", "true"}, opts, clientManager)
	require.NoError(t, err)

	mockFS := NewMockFileSystem()
	mockFS.SetFileContent("/tmp/kfp_outputs/output_metadata.json", outputMetadata)
	mockCmd := NewMockCommandExecutor()
	mockCmd.RunError = cmdErr
	launcher.WithFileSystem(mockFS).
		WithCommandExecutor(mockCmd).
		WithObjectStore(NewMockObjectStoreClient())
	return launcher
}

func TestLauncherV2_PluginLifecycle_SuccessfulExecuteEndsOnceSucceeded(t *testing.T) {
	recorder := &recordingPluginDispatcher{}
	launcher := newLauncherForPluginLifecycleTest(t, recorder, nil)

	err := launcher.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, recorder.endCount, "launcher must close the plugin task exactly once after driver handoff")
	require.Len(t, recorder.endStates, 1)
	assert.Equal(t, apiv2beta1.PipelineTask_SUCCEEDED, recorder.endStates[0])
}

func TestLauncherV2_PluginLifecycle_OnTaskEndReceivesMetricsAndInputParams(t *testing.T) {
	recorder := &recordingPluginDispatcher{}
	executorInput := &pipelinespec.ExecutorInput{
		Inputs: &pipelinespec.ExecutorInput_Inputs{
			ParameterValues: map[string]*structpb.Value{
				"learning_rate": structpb.NewNumberValue(0.01),
				"model_name":    structpb.NewStringValue("resnet"),
			},
		},
		Outputs: &pipelinespec.ExecutorInput_Outputs{
			OutputFile: "/tmp/kfp_outputs/output_metadata.json",
			Artifacts: map[string]*pipelinespec.ArtifactList{
				"metrics": {
					Artifacts: []*pipelinespec.RuntimeArtifact{
						{
							Name: "metrics",
							Uri:  "s3://bucket/output/metrics",
							Type: &pipelinespec.ArtifactTypeSchema{
								Kind: &pipelinespec.ArtifactTypeSchema_SchemaTitle{SchemaTitle: "system.Metrics"},
							},
						},
					},
				},
			},
		},
	}
	// SDK-shaped runtime payload omits type; declared system.Metrics must still drive extraction.
	// Output parameter names deliberately differ from input names.
	outputMetadata := []byte(`{
		"artifacts": {
			"metrics": {
				"artifacts": [{
					"name": "metrics",
					"metadata": {"accuracy": 0.97, "loss": 0.03}
				}]
			}
		},
		"parameterValues": {
			"accuracy": 0.99
		}
	}`)

	launcher := newLauncherForPluginLifecycleTestWithIO(t, recorder, nil, executorInput, outputMetadata)
	err := launcher.Execute(context.Background())
	require.NoError(t, err)

	assert.Equal(t, 1, recorder.endCount)
	require.Len(t, recorder.endMetrics, 1)
	require.Len(t, recorder.endParameters, 1)
	assert.Equal(t, map[string]float64{"accuracy": 0.97, "loss": 0.03}, recorder.endMetrics[0])
	assert.Equal(t, map[string]interface{}{
		"learning_rate": 0.01,
		"model_name":    "resnet",
	}, recorder.endParameters[0], "OnTaskEnd must receive task input parameters, not executor outputs")
	assert.NotContains(t, recorder.endParameters[0], "accuracy")
}

func TestLauncherV2_PluginLifecycle_OnTaskEndErrorPreservesExecutionError(t *testing.T) {
	recorder := &recordingPluginDispatcher{endErr: errors.New("plugin end failed")}
	launcher := newLauncherForPluginLifecycleTest(t, recorder, errors.New("component crashed"))

	err := launcher.Execute(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "component crashed", "original execution error must be preserved when end hook also fails")
	assert.NotContains(t, err.Error(), "plugin end failed")
	assert.Equal(t, 1, recorder.endCount)
	require.Len(t, recorder.endStates, 1)
	assert.Equal(t, apiv2beta1.PipelineTask_FAILED, recorder.endStates[0])
}

func TestLauncherV2_PluginLifecycle_FinalizationFailureEndsPluginAsFailed(t *testing.T) {
	recorder := &recordingPluginDispatcher{}
	launcher := newLauncherForPluginLifecycleTest(t, recorder, nil)

	failingAPI := &finalizationFailureAPI{
		API:               launcher.clientManager.KFPAPIClient(),
		updateStatusesErr: errors.New("propagation failed"),
	}
	launcher.clientManager = client_manager.NewFakeClientManager(
		launcher.clientManager.K8sClient(),
		failingAPI,
	)

	err := launcher.Execute(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update statuses")
	assert.Equal(t, 1, recorder.endCount)
	require.Len(t, recorder.endStates, 1)
	assert.Equal(t, apiv2beta1.PipelineTask_FAILED, recorder.endStates[0],
		"OnTaskEnd must run after finalization and observe FAILED, not SUCCEEDED")

	persisted, getErr := failingAPI.GetTask(context.Background(), &apiv2beta1.GetTaskRequest{
		TaskId: launcher.options.Task.GetTaskId(),
		RunId:  launcher.options.Task.GetRunId(),
	})
	require.NoError(t, getErr)
	assert.Equal(t, apiv2beta1.PipelineTask_FAILED, persisted.GetState())
}

func TestScalarMetricsFromExecutorOutput_UsesDeclaredTypeForSDKShapedRuntime(t *testing.T) {
	declared := &pipelinespec.ExecutorInput_Outputs{
		Artifacts: map[string]*pipelinespec.ArtifactList{
			"metrics": {
				Artifacts: []*pipelinespec.RuntimeArtifact{
					{
						Name: "metrics",
						Type: &pipelinespec.ArtifactTypeSchema{
							Kind: &pipelinespec.ArtifactTypeSchema_SchemaTitle{SchemaTitle: "system.Metrics"},
						},
					},
				},
			},
		},
	}
	// SDK runtime payloads intentionally omit type.
	runtime := &pipelinespec.ExecutorOutput{
		Artifacts: map[string]*pipelinespec.ArtifactList{
			"metrics": {
				Artifacts: []*pipelinespec.RuntimeArtifact{
					{
						Name: "metrics",
						Metadata: &structpb.Struct{Fields: map[string]*structpb.Value{
							"accuracy": structpb.NewNumberValue(0.97),
							"loss":     structpb.NewNumberValue(0.03),
						}},
					},
				},
			},
		},
	}

	got := scalarMetricsFromExecutorOutput(declared, runtime)
	require.Equal(t, map[string]float64{"accuracy": 0.97, "loss": 0.03}, got)

	// Without declared type, metrics must not be extracted from untyped runtime.
	assert.Nil(t, scalarMetricsFromExecutorOutput(nil, runtime))
}

func TestParameterValuesToInterfaces_UsesInputMap(t *testing.T) {
	inputs := map[string]*structpb.Value{
		"learning_rate": structpb.NewNumberValue(0.01),
		"model_name":    structpb.NewStringValue("resnet"),
	}
	outputs := map[string]*structpb.Value{
		"accuracy": structpb.NewNumberValue(0.99),
	}

	gotInputs := parameterValuesToInterfaces(inputs)
	require.Equal(t, map[string]interface{}{
		"learning_rate": 0.01,
		"model_name":    "resnet",
	}, gotInputs)

	gotOutputs := parameterValuesToInterfaces(outputs)
	require.Equal(t, map[string]interface{}{"accuracy": 0.99}, gotOutputs)
	assert.NotEqual(t, gotInputs, gotOutputs)
}
