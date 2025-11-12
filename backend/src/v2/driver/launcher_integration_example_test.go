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

package driver

import (
	"context"
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExample_SingleTask demonstrates the basic pattern for testing
// driver and launcher together for a single task using Execute().
func TestExample_SingleTask(t *testing.T) {
	// Step 1: Create test context with root DAG already executed
	tc := NewTestContextWithRootExecuted(
		t,
		&pipelinespec.PipelineJob_RuntimeConfig{},
		"test_data/taskOutputArtifact_test.yaml",
	)

	// Step 2: Run driver for a task
	execution, _ := tc.RunContainerDriver(
		"create-dataset",
		tc.RootTask,
		nil,   // not in a loop
		false, // don't auto-update scope yet
	)

	// Verify driver created proper ExecutorInput
	require.NotNil(t, execution.ExecutorInput)
	require.NotNil(t, execution.ExecutorInput.Outputs)

	// Step 3: Run launcher using Execute() which tests the full flow including:
	// - Task output parameter updates via KFP API
	// - Task status updates to SUCCEEDED
	// - Status propagation up the DAG hierarchy
	launcherExec := tc.RunLauncher(execution, map[string][]byte{
		"/tmp/kfp_outputs/output_metadata.json": []byte("{}"),
	}, false)

	// Verify launcher executed successfully
	require.Equal(t, 1, launcherExec.MockCmd.CallCount(),
		"Launcher should have executed the component command once")

	// Verify task status was updated to SUCCEEDED
	require.Equal(t, apiv2beta1.PipelineTaskDetail_SUCCEEDED, launcherExec.Task.State,
		"Task should be marked as SUCCEEDED")

	// Clean up scope
	_, ok := tc.Pop()
	require.True(t, ok)
}

// TestExample_SimpleArtifactPassing demonstrates the basic pattern for testing
// driver and launcher together with artifact passing between tasks.
func TestExample_SimpleArtifactPassing(t *testing.T) {
	// Step 1: Create test context with root DAG already executed
	tc := NewTestContextWithRootExecuted(
		t,
		&pipelinespec.PipelineJob_RuntimeConfig{},
		"test_data/taskOutputArtifact_test.yaml",
	)

	// Step 2: Run driver for producer task
	// The driver will create a task and prepare ExecutorInput with output artifact specification
	producerExecution, _ := tc.RunContainerDriver(
		"create-dataset",
		tc.RootTask,
		nil,   // not in a loop
		false, // don't auto-update scope yet - we need it for RunLauncher
	)

	// Verify driver created proper ExecutorInput
	require.NotNil(t, producerExecution.ExecutorInput)
	require.NotNil(t, producerExecution.ExecutorInput.Outputs)
	require.Contains(t, producerExecution.ExecutorInput.Outputs.Artifacts, "output_dataset")

	// Step 3: Run launcher to simulate actual component execution
	// This uses the ExecutorInput prepared by the driver and simulates:
	// - Downloading input artifacts
	// - Executing the component command
	// - Collecting output parameters
	// - Uploading output artifacts
	producerLauncherExec := tc.RunLauncher(producerExecution, map[string][]byte{
		// Provide the output metadata file that the component would write
		"/tmp/kfp_outputs/output_metadata.json": []byte("{}"),
	}, false)

	// Verify launcher executed successfully
	require.Equal(t, 1, producerLauncherExec.MockCmd.CallCount(),
		"Launcher should have executed the component command once")

	// Verify launcher created output artifact
	require.Len(t, producerLauncherExec.Task.Outputs.Artifacts, 1,
		"Task should have one output artifact")

	// Get the artifact ID for later verification
	outputArtifactID := producerLauncherExec.Task.Outputs.Artifacts[0].Artifacts[0].ArtifactId
	require.NotEmpty(t, outputArtifactID)

	// Verify launcher uploaded the artifact to object store
	uploads := producerLauncherExec.MockObjStore.GetUploadCallsForKey("output_dataset")
	require.Len(t, uploads, 1, "Launcher should have uploaded one artifact")

	// Now we're done with the producer task, pop its scope
	_, ok := tc.Pop()
	require.True(t, ok)

	// Step 4: Run driver for consumer task
	// The driver should resolve the input artifact from the producer task
	consumerExecution, _ := tc.RunContainerDriver(
		"process-dataset",
		tc.RootTask,
		nil,
		false, // don't auto-update scope yet
	)

	// Verify driver resolved input artifact from producer
	require.Contains(t, consumerExecution.ExecutorInput.Inputs.Artifacts, "input_dataset")
	inputArtifacts := consumerExecution.ExecutorInput.Inputs.Artifacts["input_dataset"].Artifacts
	require.Len(t, inputArtifacts, 1, "Should have one input artifact")
	require.Equal(t, outputArtifactID, inputArtifacts[0].ArtifactId,
		"Input artifact should be the output artifact from producer")

	// Step 5: Run launcher for consumer task
	consumerLauncherExec := tc.RunLauncher(consumerExecution, map[string][]byte{
		"/tmp/kfp_outputs/output_metadata.json": []byte("{}"),
	}, false)

	// Verify launcher downloaded the input artifact
	require.Len(t, consumerLauncherExec.MockObjStore.DownloadCalls, 1,
		"Launcher should have downloaded one artifact")
	assert.Equal(t, "input_dataset", consumerLauncherExec.MockObjStore.DownloadCalls[0].ArtifactKey,
		"Downloaded artifact should be the input_dataset")

	// Verify launcher executed the consumer command
	require.Equal(t, 1, consumerLauncherExec.MockCmd.CallCount(),
		"Launcher should have executed the component command once")

	// Clean up scope
	_, ok = tc.Pop()
	require.True(t, ok)
}

// TestExample_ParameterPassing demonstrates testing parameter passing between tasks.
func TestExample_ParameterPassing(t *testing.T) {
	tc := NewTestContextWithRootExecuted(
		t,
		&pipelinespec.PipelineJob_RuntimeConfig{},
		"test_data/taskOutputParameter_test.yaml",
	)

	// Step 1: Run driver for producer task
	producerExecution, _ := tc.RunContainerDriver("create-dataset", tc.RootTask, nil, false)

	// Verify driver prepared output parameter specification
	require.Contains(t, producerExecution.ExecutorInput.Outputs.Parameters, "output_parameter_path")

	// Step 2: Run launcher and mock output parameter creation
	// Get the dynamic output parameter file path from ExecutorInput
	outputParamPath := producerExecution.ExecutorInput.Outputs.Parameters["output_parameter_path"].OutputFile

	producerLauncherExec := tc.RunLauncher(producerExecution, map[string][]byte{
		"/tmp/kfp_outputs/output_metadata.json": []byte("{}"),
		// Simulate the component writing output parameter to file
		outputParamPath: []byte("10.0"),
	}, false)

	// Verify launcher executed
	require.Equal(t, 1, producerLauncherExec.MockCmd.CallCount())

	// Execute() now automatically collects output parameters from files and uploads to API.
	// Verify the output parameter was created by the launcher
	require.Len(t, producerLauncherExec.Task.Outputs.Parameters, 1)
	outputParam := producerLauncherExec.Task.Outputs.Parameters[0]
	require.Equal(t, "output_parameter_path", outputParam.ParameterKey)
	// The parameter type is STRING in the component spec, so it's parsed as a string
	require.Equal(t, "10.0", outputParam.Value.GetStringValue())

	// Clean up producer scope
	_, ok := tc.Pop()
	require.True(t, ok)

	// Refresh the run to get the updated producer task outputs
	tc.RefreshRun()

	// Step 3: Run driver for consumer task
	consumerExecution, _ := tc.RunContainerDriver("process-dataset", tc.RootTask, nil, false)

	// Verify driver resolved input parameter from producer
	require.Contains(t, consumerExecution.ExecutorInput.Inputs.ParameterValues, "input_dataset")
	// The parameter value is passed as a string from the producer
	inputValue := consumerExecution.ExecutorInput.Inputs.ParameterValues["input_dataset"]
	require.NotNil(t, inputValue)
	// It should be a string value "10.0"
	require.Equal(t, "10.0", inputValue.GetStringValue())

	// Step 4: Run launcher for consumer
	// The consumer also has output parameters we need to provide
	consumerOutputPath := consumerExecution.ExecutorInput.Outputs.Parameters["output_int"].OutputFile
	consumerLauncherExec := tc.RunLauncher(consumerExecution, map[string][]byte{
		"/tmp/kfp_outputs/output_metadata.json": []byte("{}"),
		consumerOutputPath:                      []byte("42"),
	}, false)

	// Verify launcher executed with resolved parameter
	require.Equal(t, 1, consumerLauncherExec.MockCmd.CallCount())

	// Clean up consumer scope
	_, ok = tc.Pop()
	require.True(t, ok)
}

// TestExample_LoopIteration demonstrates testing loop iterations with artifact collection.
func TestExample_LoopIteration(t *testing.T) {
	tc := NewTestContextWithRootExecuted(
		t,
		&pipelinespec.PipelineJob_RuntimeConfig{},
		"test_data/loop_collected_raw_Iterator.yaml",
	)

	// Step 1: Enter the secondary pipeline
	_, secondaryPipelineTask := tc.RunDagDriver("secondary-pipeline", tc.RootTask)

	// Step 2: Run the task that produces input for the loop
	producerExecution, _ := tc.RunContainerDriver("create-dataset", secondaryPipelineTask, nil, false)

	// Run launcher for producer
	producerLauncherExec := tc.RunLauncher(producerExecution, map[string][]byte{
		"/tmp/kfp_outputs/output_metadata.json": []byte("{}"),
	}, false)
	require.Equal(t, 1, producerLauncherExec.MockCmd.CallCount())

	// Get producer's output artifact ID
	producerArtifactID := producerLauncherExec.Task.Outputs.Artifacts[0].Artifacts[0].ArtifactId

	// Clean up producer scope
	_, ok := tc.Pop()
	require.True(t, ok)

	// Step 3: Run loop DAG driver
	loopExecution, loopTask := tc.RunDagDriver("for-loop-2", secondaryPipelineTask)

	// Verify loop received input artifact from producer
	require.Len(t, loopTask.Inputs.Artifacts, 1)

	// Step 4: Run loop iterations
	for index := 0; index < 3; index++ {
		// Run driver for this iteration
		iterExecution, _ := tc.RunContainerDriver(
			"process-dataset",
			loopTask,
			&[]int64{int64(index)}[0], // iteration index
			false,                     // don't auto-update scope
		)

		// Verify driver resolved input artifact (same for all iterations)
		require.Contains(t, iterExecution.ExecutorInput.Inputs.Artifacts, "input_dataset")
		inputArtifact := iterExecution.ExecutorInput.Inputs.Artifacts["input_dataset"].Artifacts[0]
		require.Equal(t, producerArtifactID, inputArtifact.ArtifactId)

		// Run launcher for iteration
		iterLauncherExec := tc.RunLauncher(iterExecution, map[string][]byte{
			"/tmp/kfp_outputs/output_metadata.json": []byte("{}"),
		}, false)

		// Verify launcher executed
		require.Equal(t, 1, iterLauncherExec.MockCmd.CallCount())

		// Clean up iteration scope
		_, ok = tc.Pop()
		require.True(t, ok)
	}

	// Step 5: Exit loop DAG
	tc.ExitDag()

	// Verify loop collected all iteration outputs
	loopTask, err := tc.ClientManager.KFPAPIClient().GetTask(
		context.Background(),
		&apiv2beta1.GetTaskRequest{TaskId: loopExecution.TaskID},
	)
	require.NoError(t, err)
	require.Len(t, loopTask.Outputs.Artifacts, 3,
		"Loop should have collected outputs from 3 iterations")
}

// TestExample_CustomCommandOutput demonstrates testing with custom command output verification.
func TestExample_CustomCommandOutput(t *testing.T) {
	tc := NewTestContextWithRootExecuted(
		t,
		&pipelinespec.PipelineJob_RuntimeConfig{},
		"test_data/taskOutputArtifact_test.yaml",
	)

	// Run driver
	execution, _ := tc.RunContainerDriver("create-dataset", tc.RootTask, nil, false)

	// Run launcher
	launcherExec := tc.RunLauncher(execution, map[string][]byte{
		"/tmp/kfp_outputs/output_metadata.json": []byte("{}"),
	}, false)

	// Verify the command that was executed
	require.Equal(t, 1, launcherExec.MockCmd.CallCount())

	// You can verify the exact command and args if needed
	cmdCall := launcherExec.MockCmd.RunCalls[0]
	assert.NotEmpty(t, cmdCall.Cmd, "Command should not be empty")

	// Verify file system operations
	// The launcher reads the output metadata file if it exists
	// In our case, we provided an empty JSON file
	assert.GreaterOrEqual(t, len(launcherExec.MockFS.ReadFileCalls), 0,
		"Launcher may have read files during execution")

	// Verify object store operations
	// Check that the artifact was uploaded with correct URI
	uploads := launcherExec.MockObjStore.GetUploadCallsForKey("output_dataset")
	require.Len(t, uploads, 1)
	// Artifact URI should use a valid object store scheme (minio://, s3://, gs://, etc.)
	assert.Regexp(t, `^(minio|s3|gs)://`, uploads[0].RemoteURI, "Artifact URI should use a valid object store scheme")

	// Clean up scope
	_, ok := tc.Pop()
	require.True(t, ok)
}
