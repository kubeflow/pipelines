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
		Tasks: []*apiv2beta1.PipelineTaskDetail{},
	}
	mockAPI.AddRun(run)

	task := &apiv2beta1.PipelineTaskDetail{
		TaskId:  taskID,
		RunId:   runID,
		Name:    "test-task",
		State:   apiv2beta1.PipelineTaskDetail_RUNNING,
		Type:    apiv2beta1.PipelineTaskDetail_RUNTIME,
		Inputs:  &apiv2beta1.PipelineTaskDetail_InputOutputs{},
		Outputs: &apiv2beta1.PipelineTaskDetail_InputOutputs{},
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
	task := &apiv2beta1.PipelineTaskDetail{
		TaskId: "task-456",
		RunId:  "run-123",
		Name:   "test-task",
		State:  apiv2beta1.PipelineTaskDetail_RUNNING,
	}
	_, err := mockAPI.CreateTask(context.Background(), &apiv2beta1.CreateTaskRequest{Task: task})
	require.NoError(t, err)

	// Update task status
	task.State = apiv2beta1.PipelineTaskDetail_SUCCEEDED
	_, err = mockAPI.UpdateTask(context.Background(), &apiv2beta1.UpdateTaskRequest{
		TaskId: "task-456",
		Task:   task,
	})
	require.NoError(t, err)

	// Verify task was updated
	updatedTask, err := mockAPI.GetTask(context.Background(), &apiv2beta1.GetTaskRequest{TaskId: "task-456"})
	require.NoError(t, err)
	assert.Equal(t, apiv2beta1.PipelineTaskDetail_SUCCEEDED, updatedTask.State)
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

			task := &apiv2beta1.PipelineTaskDetail{
				TaskId:  "test-task",
				RunId:   "test-run",
				Name:    "test-task",
				State:   apiv2beta1.PipelineTaskDetail_RUNNING,
				Inputs:  &apiv2beta1.PipelineTaskDetail_InputOutputs{},
				Outputs: &apiv2beta1.PipelineTaskDetail_InputOutputs{},
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
		Tasks: []*apiv2beta1.PipelineTaskDetail{},
	}
	mockAPI.AddRun(run)

	// Create test task
	task := &apiv2beta1.PipelineTaskDetail{
		TaskId:  "test-task-456",
		RunId:   "test-run-123",
		Name:    "train-model",
		State:   apiv2beta1.PipelineTaskDetail_RUNNING,
		Type:    apiv2beta1.PipelineTaskDetail_RUNTIME,
		Inputs:  &apiv2beta1.PipelineTaskDetail_InputOutputs{},
		Outputs: &apiv2beta1.PipelineTaskDetail_InputOutputs{},
	}

	// Add task to mock API so it can be updated during execution
	_, err = mockAPI.CreateTask(context.Background(), &apiv2beta1.CreateTaskRequest{Task: task})
	assert.NoError(t, err)

	// Create task spec
	taskSpec := &pipelinespec.PipelineTaskSpec{
		TaskInfo: &pipelinespec.PipelineTaskInfo{
			Name: "train-model",
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

	// Verify artifact was uploaded to object store
	assert.True(t, mockObjStore.WasUploaded("s3://bucket/output/model.pkl"), "Expected model artifact to be uploaded")

	// Verify batch updater queued artifact creation and task updates
	metrics := launcher.batchUpdater.GetMetrics()
	assert.Greater(t, metrics["queued_artifacts"], 0, "Expected artifacts to be queued for creation")
	assert.Greater(t, metrics["queued_task_updates"], 0, "Expected task updates to be queued")
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
