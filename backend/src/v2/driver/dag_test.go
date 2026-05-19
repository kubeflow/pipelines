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
	"fmt"
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/json"
)

func TestRootDagComponentInputs(t *testing.T) {
	runtimeConfig := &pipelinespec.PipelineJob_RuntimeConfig{
		ParameterValues: map[string]*structpb.Value{
			"string_input": structpb.NewStringValue("test-input1"),
			"number_input": structpb.NewNumberValue(42.5),
			"bool_input":   structpb.NewBoolValue(true),
			"null_input":   structpb.NewNullValue(),
			"list_input": structpb.NewListValue(&structpb.ListValue{Values: []*structpb.Value{
				structpb.NewStringValue("value1"),
				structpb.NewNumberValue(42),
				structpb.NewBoolValue(true),
			}}),
			"map_input": structpb.NewStructValue(&structpb.Struct{
				Fields: map[string]*structpb.Value{
					"key1": structpb.NewStringValue("value1"),
					"key2": structpb.NewNumberValue(42),
					"key3": structpb.NewListValue(&structpb.ListValue{
						Values: []*structpb.Value{
							structpb.NewStringValue("nested1"),
							structpb.NewStringValue("nested2"),
						},
					}),
				},
			}),
		},
	}

	tc := NewTestContextWithRootExecuted(t, runtimeConfig, "test_data/taskOutputArtifact_test.yaml")
	task := tc.RootTask
	require.NotNil(t, task.Inputs)
	require.NotEmpty(t, task.Inputs.Parameters)

	// Verify parameter values
	paramMap := make(map[string]*structpb.Value)
	for _, param := range task.Inputs.Parameters {
		paramMap[param.GetParameterKey()] = param.Value
	}

	assert.Equal(t, "test-input1", paramMap["string_input"].GetStringValue())
	assert.Equal(t, 42.5, paramMap["number_input"].GetNumberValue())
	assert.Equal(t, true, paramMap["bool_input"].GetBoolValue())
	assert.NotNil(t, paramMap["null_input"].GetNullValue())
	assert.Len(t, paramMap["list_input"].GetListValue().Values, 3)
	assert.NotNil(t, paramMap["map_input"].GetStructValue())
	assert.Len(t, paramMap["map_input"].GetStructValue().Fields, 3)
}

func TestLoopArtifactPassing(t *testing.T) {
	tc := NewTestContextWithRootExecuted(
		t,
		&pipelinespec.PipelineJob_RuntimeConfig{},
		"test_data/loop_collected_raw_Iterator.yaml",
	)
	parentTask := tc.RootTask

	// Run Dag on the First Task
	secondaryPipelineExecution, secondaryPipelineTask := tc.RunDagDriver("secondary-pipeline", parentTask)
	require.Nil(t, secondaryPipelineExecution.ExecutorInput.Outputs)
	require.Equal(t, apiv2beta1.PipelineTaskDetail_RUNNING, secondaryPipelineTask.State)

	// Refresh Parent Task - The parent task should be the secondary pipeline task for "create-dataset"
	parentTask = secondaryPipelineTask

	// Now we'll run the subtasks in the secondary pipeline, one of which is a loop of 3 iterations

	// Run the Downstream Task that will use the output artifact
	createDataSetExecution, _ := tc.RunContainerDriver("create-dataset", parentTask, nil, false)
	// Expect the output artifact to be created
	require.NotNil(t, createDataSetExecution.ExecutorInput.Outputs)
	require.NotNil(t, createDataSetExecution.ExecutorInput.Outputs.Artifacts)
	require.Equal(t, 1, len(createDataSetExecution.ExecutorInput.Outputs.Artifacts))
	require.Contains(t, createDataSetExecution.ExecutorInput.Outputs.Artifacts, "output_dataset")
	require.Equal(t, "output_dataset", createDataSetExecution.ExecutorInput.Outputs.Artifacts["output_dataset"].GetArtifacts()[0].Name)

	// Run the actual launcher with mocks to simulate component execution
	launcherExec := tc.RunLauncher(createDataSetExecution, map[string][]byte{"/tmp/kfp_outputs/output_metadata.json": []byte("{}")}, true)
	// Get the artifact ID that was created
	require.Len(t, launcherExec.Task.Outputs.Artifacts, 1)
	createDataSetOutputArtifactID := launcherExec.Task.Outputs.Artifacts[0].Artifacts[0].ArtifactId

	// Run the Loop Task - note that parentTask for for-loop-2 remains as secondary-pipeline
	loopExecution, loopTask := tc.RunDagDriver("for-loop-2", parentTask)
	require.Nil(t, secondaryPipelineExecution.ExecutorInput.Outputs)
	require.NotZero(t, len(loopTask.Inputs.Parameters))
	// Expect loop task to have resolved its input parameter
	require.Equal(t, "pipelinechannel--loop-item-param-1", loopTask.Inputs.Parameters[0].ParameterKey)
	// Expect the artifact output of create-dataset as input to for-loop-2
	require.Equal(t, len(loopTask.Inputs.Artifacts), 1)

	// The parent task should be "for-loop-2" for the iterations at first depth
	parentTask = loopTask

	// Perform the iteration calls
	for index, paramID := range []string{"1", "2", "3"} {
		// Run the "process-dataset" Container Task with iteration index
		processExecution, _ := tc.RunContainerDriver("process-dataset", parentTask, util.Int64Pointer(int64(index)), false)
		require.NotNil(t, processExecution.ExecutorInput.Outputs)
		require.NotNil(t, processExecution.ExecutorInput.Inputs.Artifacts["input_dataset"])
		require.Equal(t, 1, len(processExecution.ExecutorInput.Inputs.Artifacts["input_dataset"].GetArtifacts()))
		require.Equal(t, processExecution.ExecutorInput.Inputs.Artifacts["input_dataset"].GetArtifacts()[0].ArtifactId, createDataSetOutputArtifactID)
		require.NotNil(t, processExecution.ExecutorInput.Inputs.ParameterValues["model_id_in"])
		require.Equal(t, processExecution.ExecutorInput.Inputs.ParameterValues["model_id_in"].GetStringValue(), paramID)

		// Run the actual launcher for process-dataset
		// The launcher will automatically propagate outputs up the DAG hierarchy
		processLauncherExec := tc.RunLauncher(processExecution, map[string][]byte{"/tmp/kfp_outputs/output_metadata.json": []byte("{}")}, true)
		// Get the artifact ID that was created
		require.Len(t, processLauncherExec.Task.Outputs.Artifacts, 1)
		processDataSetArtifactID := processLauncherExec.Task.Outputs.Artifacts[0].Artifacts[0].ArtifactId

		// Verify that the launcher automatically propagated the output artifact to the for-loop-2 task
		loopTask, err := tc.ClientManager.KFPAPIClient().GetTask(context.Background(), &apiv2beta1.GetTaskRequest{TaskId: loopExecution.TaskID})
		require.NoError(t, err)
		require.NotNil(t, loopTask.Outputs)
		require.Equal(t, len(loopTask.Outputs.Artifacts), index+1, "Loop task should have %d artifacts after iteration %d", index+1, index)

		// Verify that the launcher also propagated the output artifact up to the secondary-pipeline task
		secondaryPipelineTask, err = tc.ClientManager.KFPAPIClient().GetTask(context.Background(), &apiv2beta1.GetTaskRequest{TaskId: secondaryPipelineExecution.TaskID})
		require.NoError(t, err)
		require.NotNil(t, secondaryPipelineTask.Outputs)
		require.Equal(t, index+1, len(secondaryPipelineTask.Outputs.Artifacts), "Secondary pipeline task should have %d artifacts after iteration %d", index+1, index)

		// Run the next iteration component
		analyzeExecution, _ := tc.RunContainerDriver("analyze-artifact", parentTask, util.Int64Pointer(int64(index)), false)
		require.NotNil(t, createDataSetExecution.ExecutorInput.Outputs)
		require.NotNil(t, analyzeExecution.ExecutorInput.Outputs)
		require.NotNil(t, analyzeExecution.ExecutorInput.Inputs.Artifacts["analyze_artifact_input"])
		require.Equal(t, 1, len(analyzeExecution.ExecutorInput.Inputs.Artifacts["analyze_artifact_input"].GetArtifacts()))
		require.Equal(t, analyzeExecution.ExecutorInput.Inputs.Artifacts["analyze_artifact_input"].GetArtifacts()[0].ArtifactId, processDataSetArtifactID)

		// Run the actual launcher for analyze-artifact
		_ = tc.RunLauncher(analyzeExecution, map[string][]byte{"/tmp/kfp_outputs/output_metadata.json": []byte("{}")}, true)
	}

	tasks, err := tc.ClientManager.KFPAPIClient().ListTasks(context.Background(), &apiv2beta1.ListTasksRequest{
		ParentFilter: &apiv2beta1.ListTasksRequest_ParentId{ParentId: loopExecution.TaskID},
	})
	require.NoError(t, err)
	require.NotNil(t, tasks)
	// Expect 3 tasks for analyze-artifact + 3 tasks for process-dataset
	require.Equal(t, 6, len(tasks.Tasks))

	// Expect the 3 artifacts from process-task to have been collected by the for-loop-2 task
	forLoopTask, err := tc.ClientManager.KFPAPIClient().GetTask(context.Background(), &apiv2beta1.GetTaskRequest{TaskId: loopExecution.TaskID})
	require.NoError(t, err)
	require.Equal(t, 3, len(forLoopTask.Outputs.Artifacts))

	// Verify producer attribution for loop task artifacts
	// The producer should be the immediate child task from the loop's perspective (process-dataset),
	// not the original runtime task that created the artifact
	for i, artifactIO := range forLoopTask.Outputs.Artifacts {
		require.NotNil(t, artifactIO.Producer, "Loop task artifact %d should have a producer", i)
		require.Equal(t, "process-dataset", artifactIO.Producer.TaskName,
			"Loop task artifact %d producer should be 'process-dataset' (immediate child)", i)
		require.NotNil(t, artifactIO.Producer.Iteration,
			"Loop task artifact %d should have iteration index preserved", i)
	}

	// Run "analyze_artifact_list" in "secondary_pipeline"
	// Move up a parent
	parentTask = secondaryPipelineTask
	tc.ExitDag()

	analyzeArtifactListExecution, analyzeArtifactListTask := tc.RunContainerDriver("analyze-artifact-list", parentTask, nil, false)
	require.NotNil(t, analyzeArtifactListExecution.ExecutorInput.Outputs)
	require.NotNil(t, analyzeArtifactListExecution.ExecutorInput.Inputs.Artifacts["artifact_list_input"])
	require.Equal(t, 3, len(analyzeArtifactListExecution.ExecutorInput.Inputs.Artifacts["artifact_list_input"].GetArtifacts()))

	artifactListLauncher := tc.RunLauncher(analyzeArtifactListExecution, map[string][]byte{"/tmp/kfp_outputs/output_metadata.json": []byte("{}")}, true)
	require.NotNil(t, artifactListLauncher.Task)
	_, err = tc.ClientManager.KFPAPIClient().GetTask(context.Background(), &apiv2beta1.GetTaskRequest{TaskId: analyzeArtifactListTask.TaskId})
	require.NoError(t, err)
	// Primary Pipeline tests

	// Expect the 3 artifacts from process-task to have been collected by the secondary-pipeline task
	secondaryPipelineTask, err = tc.ClientManager.KFPAPIClient().GetTask(context.Background(), &apiv2beta1.GetTaskRequest{TaskId: secondaryPipelineExecution.TaskID})
	require.NoError(t, err)
	require.Equal(t, 3, len(secondaryPipelineTask.Outputs.Artifacts))

	// Verify producer attribution for secondary-pipeline task artifacts
	// The producer should be the immediate child from secondary-pipeline's perspective (for-loop-2),
	// NOT the original runtime task (process-dataset) that created the artifact
	// This demonstrates that producer attribution "resets" at each propagation level
	for i, artifactIO := range secondaryPipelineTask.Outputs.Artifacts {
		require.NotNil(t, artifactIO.Producer, "Secondary pipeline artifact %d should have a producer", i)
		require.Equal(t, "for-loop-2", artifactIO.Producer.TaskName,
			"Secondary pipeline artifact %d producer should be 'for-loop-2' (immediate child from secondary-pipeline's perspective)", i)
		require.Nil(t, artifactIO.Producer.Iteration,
			"Secondary pipeline artifact %d should not have iteration index preserved from process-dataset", i)
	}

	// Move up a parent
	parentTask = tc.RootTask
	tc.ExitDag()

	// Not to be confused with the "analyze-artifact-list" task in secondary pipeline,
	// this is the "analyze-artifact-list" task in the primary pipeline
	analyzeArtifactListOuterExecution, _ := tc.RunContainerDriver("analyze-artifact-list", parentTask, nil, false)
	require.NotNil(t, analyzeArtifactListExecution.ExecutorInput.Outputs)
	require.NotNil(t, analyzeArtifactListOuterExecution.ExecutorInput.Outputs)
	require.NotNil(t, analyzeArtifactListOuterExecution.ExecutorInput.Inputs.Artifacts["artifact_list_input"])
	require.Equal(t, 3, len(analyzeArtifactListOuterExecution.ExecutorInput.Inputs.Artifacts["artifact_list_input"].GetArtifacts()))

	_ = tc.RunLauncher(analyzeArtifactListOuterExecution, map[string][]byte{"/tmp/kfp_outputs/output_metadata.json": []byte("{}")}, true)

	// Refresh Run so it has the new tasks
	tc.RefreshRun()

	// primary_pipeline()		 x 1  (root)
	// secondary_pipeline()      x 1  (dag)
	//   create_dataset()        x 1  (runtime)
	//   for_loop_1()            x 1  (loop)
	//     process_dataset()     x 3  (runtime)
	//	   analyze_artifact()    x 3  (runtime)
	//   analyze_artifact_list() x 1  (runtime)
	// analyze_artifact_list()   x 1  (runtime)
	require.Equal(t, 12, len(tc.Run.Tasks))
}

// TestParameterInputIterator will test parameter Input Iterator
// and parameter collection from output of a task in a loop
func TestParameterInputIterator(t *testing.T) {
	tc := NewTestContextWithRootExecuted(
		t,
		&pipelinespec.PipelineJob_RuntimeConfig{},
		"test_data/loop_collected_InputParameter_Iterator.yaml",
	)
	// Execute full pipeline
	parentTask := tc.RootTask
	_, secondaryPipelineTask := tc.RunDagDriver("secondary-pipeline", parentTask)
	parentTask = secondaryPipelineTask

	splitIDsExecution, _ := tc.RunContainerDriver("split-ids", parentTask, nil, false)

	// Get the output parameter file path
	outputParamPath := splitIDsExecution.ExecutorInput.Outputs.Parameters["Output"].OutputFile

	splitIDsLauncher := tc.RunLauncher(splitIDsExecution, map[string][]byte{
		"/tmp/kfp_outputs/output_metadata.json": []byte("{}"),
		outputParamPath:                         []byte(`["1", "2", "3"]`),
	}, true)
	loopExecution, loopTask := tc.RunDagDriver("for-loop-1", parentTask)
	parentTask = loopTask
	require.NotNil(t, loopExecution)
	require.NotNil(t, loopExecution.IterationCount)
	require.Equal(t, 3, *loopExecution.IterationCount)

	for index := range []string{"1", "2", "3"} {
		index64 := util.Int64Pointer(int64(index))
		createFileExecution, _ := tc.RunContainerDriver("create-file", parentTask, index64, false)
		_ = tc.RunLauncher(createFileExecution, map[string][]byte{
			"/tmp/kfp_outputs/output_metadata.json": []byte("{}"),
		}, true)

		// Run next task
		readSingleFileExecution, _ := tc.RunContainerDriver("read-single-file", parentTask, index64, false)
		readSingleFileOutputPath := readSingleFileExecution.ExecutorInput.Outputs.Parameters["Output"].OutputFile
		_ = tc.RunLauncher(readSingleFileExecution, map[string][]byte{
			"/tmp/kfp_outputs/output_metadata.json": []byte("{}"),
			readSingleFileOutputPath:                []byte(fmt.Sprintf("file-%d", index)),
		}, true)
		_ = splitIDsLauncher
	}

	tc.ExitDag()
	parentTask = secondaryPipelineTask

	// Check what parameters the for-loop-1 task has after all iterations
	refreshedLoopTask, err := tc.ClientManager.KFPAPIClient().GetTask(context.Background(), &apiv2beta1.GetTaskRequest{TaskId: loopTask.TaskId})
	require.NoError(t, err)
	require.NotNil(t, refreshedLoopTask.Outputs)
	require.Equal(t, 3, len(refreshedLoopTask.Outputs.Parameters))

	// Verify producer attribution for loop task parameters
	// The producer should be the immediate child task from the loop's perspective (read-single-file),
	// not the original runtime task that created the parameter
	for i, param := range refreshedLoopTask.Outputs.Parameters {
		require.NotNil(t, param.Producer, "Loop task parameter %d should have a producer", i)
		require.Equal(t, "read-single-file", param.Producer.TaskName,
			"Loop task parameter %d producer should be 'read-single-file' (immediate child)", i)
		require.NotNil(t, param.Producer.Iteration,
			"Loop task parameter %d should have iteration index preserved", i)
	}

	readValuesExecution, _ := tc.RunContainerDriver("read-values", parentTask, nil, false)
	readValuesOutputPath := readValuesExecution.ExecutorInput.Outputs.Parameters["Output"].OutputFile

	_ = tc.RunLauncher(readValuesExecution, map[string][]byte{
		"/tmp/kfp_outputs/output_metadata.json": []byte("{}"),
		readValuesOutputPath:                    []byte("files read"),
	}, true)

	tc.ExitDag()
	parentTask = tc.RootTask

	readValuesExecution2, _ := tc.RunContainerDriver("read-values", parentTask, nil, false)
	readValuesOutputPath2 := readValuesExecution2.ExecutorInput.Outputs.Parameters["Output"].OutputFile

	_ = tc.RunLauncher(readValuesExecution2, map[string][]byte{
		"/tmp/kfp_outputs/output_metadata.json": []byte("{}"),
		readValuesOutputPath2:                   []byte("files read"),
	}, true)

	task, err := tc.ClientManager.KFPAPIClient().GetTask(context.Background(), &apiv2beta1.GetTaskRequest{TaskId: secondaryPipelineTask.GetTaskId()})
	require.NoError(t, err)
	require.NotNil(t, task.Outputs)
	require.Equal(t, 3, len(task.Outputs.Parameters))

	// Verify producer attribution for secondary-pipeline task parameters
	// The producer should be the immediate child from secondary-pipeline's perspective (for-loop-1),
	// NOT the original runtime task (read-single-file) that created the parameter
	// This demonstrates that producer attribution "resets" at each propagation level
	var collectOutputs []string
	for i, params := range task.Outputs.Parameters {
		collectOutputs = append(collectOutputs, params.GetValue().GetStringValue())
		require.Equal(t, apiv2beta1.IOType_ITERATOR_OUTPUT, params.GetType())

		// Verify producer is the immediate child task (for-loop-1)
		require.NotNil(t, params.Producer, "Secondary pipeline parameter %d should have a producer", i)
		require.Equal(t, "for-loop-1", params.Producer.TaskName,
			"Secondary pipeline parameter %d producer should be 'for-loop-1' (immediate child from secondary-pipeline's perspective)", i)
		require.Nil(t, params.Producer.Iteration,
			"Secondary pipeline parameter %d shouldn't propagate read-single-file", i)
	}
	require.Equal(t, []string{"file-0", "file-1", "file-2"}, collectOutputs)
}

func TestNestedDag(t *testing.T) {
	tc := NewTestContextWithRootExecuted(t, &pipelinespec.PipelineJob_RuntimeConfig{}, "test_data/nested_naming_conflicts.yaml")
	parentTask := tc.RootTask

	aExecution, _ := tc.RunContainerDriver("a", parentTask, nil, false)
	aLauncher := tc.RunLauncher(aExecution, map[string][]byte{"/tmp/kfp_outputs/output_metadata.json": []byte("{}")}, true)
	_ = aLauncher.Task

	_, pipelineBTask := tc.RunDagDriver("pipeline-b", parentTask)
	parentTask = pipelineBTask

	nestedAExecution, _ := tc.RunContainerDriver("a", parentTask, nil, false)
	nestedALauncher := tc.RunLauncher(nestedAExecution, map[string][]byte{"/tmp/kfp_outputs/output_metadata.json": []byte("{}")}, true)
	_ = nestedALauncher.Task

	nestedBExecution, _ := tc.RunContainerDriver("b", parentTask, nil, false)
	nestedBLauncher := tc.RunLauncher(nestedBExecution, map[string][]byte{"/tmp/kfp_outputs/output_metadata.json": []byte("{}")}, true)
	_ = nestedBLauncher.Task

	_, pipelineCTask := tc.RunDagDriver("pipeline-c", parentTask)
	parentTask = pipelineCTask

	nestedNestedAExecution, _ := tc.RunContainerDriver("a", parentTask, nil, false)
	nestedNestedALauncher := tc.RunLauncher(nestedNestedAExecution, map[string][]byte{"/tmp/kfp_outputs/output_metadata.json": []byte("{}")}, true)
	_ = nestedNestedALauncher.Task

	nestedNestedBExecution, _ := tc.RunContainerDriver("b", parentTask, nil, false)
	nestedNestedBLauncher := tc.RunLauncher(nestedNestedBExecution, map[string][]byte{"/tmp/kfp_outputs/output_metadata.json": []byte("{}")}, true)
	nestedNestedBTask := nestedNestedBLauncher.Task

	cExecution, _ := tc.RunContainerDriver("c", parentTask, nil, false)

	// Run the launcher for task c which will create outputs and propagate them up
	cLauncherExec := tc.RunLauncher(cExecution, map[string][]byte{"/tmp/kfp_outputs/output_metadata.json": []byte("{}")}, true)
	cTask := cLauncherExec.Task

	tc.ExitDag()
	tc.ExitDag()
	parentTask = tc.RootTask

	_, _ = tc.RunContainerDriver("verify", parentTask, nil, true)

	var err error

	// Get the artifact ID from cTask's output
	require.NotNil(t, cTask.Outputs)
	require.Equal(t, 1, len(cTask.Outputs.Artifacts))
	cTaskArtifactID := cTask.Outputs.Artifacts[0].GetArtifacts()[0].GetArtifactId()

	// Confirm that the artifact passed to "verify" task came from task_c
	// by checking that pipeline-b has the same artifact ID in its outputs (propagated from c)
	pipelineBTask, err = tc.ClientManager.KFPAPIClient().GetTask(context.Background(), &apiv2beta1.GetTaskRequest{TaskId: pipelineBTask.GetTaskId()})
	require.NoError(t, err)
	require.NotNil(t, pipelineBTask.Outputs)
	require.Equal(t, 1, len(pipelineBTask.Outputs.Artifacts))
	require.Equal(t, cTaskArtifactID, pipelineBTask.Outputs.Artifacts[0].GetArtifacts()[0].GetArtifactId(),
		"pipeline-b's output artifact should be the same artifact produced by task c")

	// Get the artifact ID from nestedNestedBTask's output
	require.NotNil(t, nestedNestedBTask.Outputs)
	require.Equal(t, 1, len(nestedNestedBTask.Outputs.Artifacts))
	nestedNestedBArtifactID := nestedNestedBTask.Outputs.Artifacts[0].GetArtifacts()[0].GetArtifactId()

	// Confirm that the artifact passed to cTask came from the nestedNestedBtask
	// I.e the b() task that ran in pipeline-c and not in pipeline-b
	cTask, err = tc.ClientManager.KFPAPIClient().GetTask(context.Background(), &apiv2beta1.GetTaskRequest{TaskId: cTask.GetTaskId()})
	require.NoError(t, err)
	require.NotNil(t, cTask.Inputs)
	require.Equal(t, 1, len(cTask.Inputs.Artifacts))
	require.Equal(t, nestedNestedBArtifactID, cTask.Inputs.Artifacts[0].GetArtifacts()[0].GetArtifactId(),
		"cTask's input artifact should be from nested-nested-b, not nested-b")
}

func TestParameterTaskOutput(t *testing.T) {
	tc := NewTestContextWithRootExecuted(t, &pipelinespec.PipelineJob_RuntimeConfig{}, "test_data/taskOutputParameter_test.yaml")
	parentTask := tc.RootTask

	// Run driver and launcher for create-dataset
	cdExecution, _ := tc.RunContainerDriver("create-dataset", parentTask, nil, false)
	cdOutputPath := cdExecution.ExecutorInput.Outputs.Parameters["output_parameter_path"].OutputFile
	_ = tc.RunLauncher(cdExecution, map[string][]byte{
		"/tmp/kfp_outputs/output_metadata.json": []byte("{}"),
		cdOutputPath:                            []byte("10.0"),
	}, true)

	// Run driver and launcher for process-dataset
	pdExecution, _ := tc.RunContainerDriver("process-dataset", parentTask, nil, false)
	pdOutputPath := pdExecution.ExecutorInput.Outputs.Parameters["output_int"].OutputFile
	_ = tc.RunLauncher(pdExecution, map[string][]byte{
		"/tmp/kfp_outputs/output_metadata.json": []byte("{}"),
		pdOutputPath:                            []byte("100"),
	}, true)

	// Run driver and launcher for analyze-artifact
	analyzeArtifactExecution, _ := tc.RunContainerDriver("analyze-artifact", parentTask, nil, false)
	analyzeOutputPath := analyzeArtifactExecution.ExecutorInput.Outputs.Parameters["output_opinion"].OutputFile
	_ = tc.RunLauncher(analyzeArtifactExecution, map[string][]byte{
		"/tmp/kfp_outputs/output_metadata.json": []byte("{}"),
		analyzeOutputPath:                       []byte("true"),
	}, true)
}

func TestOneOf(t *testing.T) {
	tc := NewTestContextWithRootExecuted(t, &pipelinespec.PipelineJob_RuntimeConfig{}, "test_data/oneof_simple.yaml")
	parentTask := tc.RootTask
	require.NotNil(t, parentTask)

	// Run secondary pipeline
	_, secondaryPipelineTask := tc.RunDagDriver("secondary-pipeline", parentTask)
	parentTask = secondaryPipelineTask

	// Run create_dataset()
	createDatasetExecution, _ := tc.RunContainerDriver("create-dataset", parentTask, nil, false)

	// Get the output parameter file path
	conditionOutPath := createDatasetExecution.ExecutorInput.Outputs.Parameters["condition_out"].OutputFile

	_ = tc.RunLauncher(createDatasetExecution, map[string][]byte{
		"/tmp/kfp_outputs/output_metadata.json": []byte("{}"),
		conditionOutPath:                        []byte("second"),
	}, true)

	// Run ConditionBranch
	_, conditionBranch1Task := tc.RunDagDriver("condition-branches-1", parentTask)

	// Expect this condition to not be met
	condition2Execution, _ := tc.RunDagDriver("condition-2", conditionBranch1Task)
	require.NotNil(t, condition2Execution.Condition)
	require.False(t, *condition2Execution.Condition)

	tc.ExitDag()

	// Expect this condition to not be met
	condition4Execution, _ := tc.RunDagDriver("condition-4", conditionBranch1Task)
	require.NotNil(t, condition4Execution.Condition)
	require.False(t, *condition4Execution.Condition)

	tc.ExitDag()

	// Expect this condition to pass since output of
	// create-dataset == "second"
	condition3Execution, condition3Task := tc.RunDagDriver("condition-3", conditionBranch1Task)
	require.NotNil(t, condition3Execution.Condition)
	require.True(t, *condition3Execution.Condition)

	parentTask = condition3Task
	giveAnimal1Execution, _ := tc.RunContainerDriver("give-animal-2", parentTask, nil, false)
	_ = tc.RunLauncher(giveAnimal1Execution, map[string][]byte{
		"/tmp/kfp_outputs/output_metadata.json": []byte("{}"),
	}, true)

	analyzeAnimal1Execution, _ := tc.RunContainerDriver("analyze-animal", parentTask, nil, false)

	// Run the launcher for analyze-animal which will create outputs and propagate them up the DAG hierarchy
	tc.RunLauncher(analyzeAnimal1Execution, map[string][]byte{"/tmp/kfp_outputs/output_metadata.json": []byte("{}")}, true)

	tc.ExitDag()
	tc.ExitDag()
	tc.ExitDag()
	parentTask = tc.RootTask

	_, _ = tc.RunContainerDriver("check-animal", parentTask, nil, true)
}

func TestFinalStatus(t *testing.T) {
	tc := NewTestContextWithRootExecuted(
		t,
		&pipelinespec.PipelineJob_RuntimeConfig{},
		"test_data/pipeline_with_input_status_state.yaml",
	)
	parentTask := tc.RootTask
	require.NotNil(t, parentTask)

	_, exitHandler1Task := tc.RunDagDriver("exit-handler-1", parentTask)
	parentTask = exitHandler1Task

	_, _ = tc.RunContainerDriver("some-task", parentTask, nil, true)

	tc.ExitDag()
	parentTask = tc.RootTask

	_, echoStateTask := tc.RunContainerDriver("echo-state", parentTask, nil, true)
	require.Len(t, echoStateTask.Inputs.GetParameters(), 1)
	inputFinalStatusParam := echoStateTask.Inputs.GetParameters()[0]
	require.NotNil(t, inputFinalStatusParam)
	// Mock library doesn't update dag statuses, in production we would expect
	// this to say "SUCCEEDED" once it's done running
	require.Equal(t, "RUNNING", inputFinalStatusParam.GetValue().GetStructValue().Fields["state"].GetStringValue())
	require.Equal(t, "exit-handler-1", inputFinalStatusParam.GetValue().GetStructValue().Fields["pipelineTaskName"].GetStringValue())
}

func TestWithCaching(t *testing.T) {
	tc := NewTestContextWithRootExecuted(
		t,
		&pipelinespec.PipelineJob_RuntimeConfig{},
		"test_data/cache_test.yaml",
	)
	parentTask := tc.RootTask
	require.NotNil(t, parentTask)

	// Run create-dataset driver and launcher
	createDatasetExecution, _ := tc.RunContainerDriver("create-dataset", parentTask, nil, false)
	createDatasetLauncher := tc.RunLauncher(createDatasetExecution, map[string][]byte{"/tmp/kfp_outputs/output_metadata.json": []byte("{}")}, true)
	require.Len(t, createDatasetLauncher.Task.Outputs.Artifacts, 1)

	// First run of process-dataset - should not be cached
	processDatasetExecution, _ := tc.RunContainerDriver("process-dataset", parentTask, nil, false)
	processDatasetLauncher := tc.RunLauncher(processDatasetExecution, map[string][]byte{"/tmp/kfp_outputs/output_metadata.json": []byte("{}")}, true)
	require.NotNil(t, processDatasetExecution.Cached)
	require.False(t, *processDatasetExecution.Cached)
	require.Equal(t, apiv2beta1.PipelineTaskDetail_SUCCEEDED, processDatasetLauncher.Task.GetState())
	require.NotEmpty(t, processDatasetExecution.PodSpecPatch)

	// Second run of process-dataset - should be cached
	processDatasetExecution2, processDatasetTask2 := tc.RunContainerDriver("process-dataset", parentTask, nil, true)
	require.NotNil(t, processDatasetExecution2.Cached)
	require.True(t, *processDatasetExecution2.Cached)
	require.Equal(t, apiv2beta1.PipelineTaskDetail_CACHED, processDatasetTask2.GetState())
	require.Empty(t, processDatasetExecution2.PodSpecPatch)
}

func TestOptionalFields(t *testing.T) {
	// The API Server will populate runtime config with
	// the defaults in the root InputDefinition is they are
	// not user overridden. We mock this here.
	runtimeInputs := &pipelinespec.PipelineJob_RuntimeConfig{
		ParameterValues: map[string]*structpb.Value{
			"input_str4": structpb.NewNullValue(),
			"input_str5": structpb.NewStringValue("Some pipeline default"),
			"input_str6": structpb.NewNullValue(),
		},
	}

	tc := NewTestContextWithRootExecuted(
		t, runtimeInputs,
		"test_data/component_with_optional_inputs.yaml",
	)
	parentTask := tc.RootTask
	require.NotNil(t, parentTask)

	execution, _ := tc.RunContainerDriver("component-op", parentTask, nil, false)
	require.NotNil(t, execution)

	// Run launcher which will automatically add default parameters to the task
	// via addDefaultParametersToTask() for any parameters that have defaults
	// but weren't explicitly provided
	launcherExec := tc.RunLauncher(execution, map[string][]byte{
		"/tmp/kfp_outputs/output_metadata.json": []byte("{}"),
	}, true)

	params := launcherExec.Task.Inputs.GetParameters()
	require.GreaterOrEqual(t, len(params), 0)

	p := tc.fetchParameter("input_str1", params)
	require.NotNil(t, p)

	p = tc.fetchParameter("input_str2", params)
	require.NotNil(t, p)

	p = tc.fetchParameter("input_str3", params)
	require.Nil(t, p)

	p = tc.fetchParameter("input_str4_from_pipeline", params)
	require.NotNil(t, p)

	p = tc.fetchParameter("input_str5_from_pipeline", params)
	require.NotNil(t, p)

	p = tc.fetchParameter("input_str6_from_pipeline", params)
	require.Nil(t, p)

	p = tc.fetchParameter("input_bool1", params)
	require.NotNil(t, p)

	p = tc.fetchParameter("input_bool2", params)
	require.Nil(t, p)

	p = tc.fetchParameter("input_dict", params)
	require.NotNil(t, p)

	p = tc.fetchParameter("input_list", params)
	require.NotNil(t, p)

	p = tc.fetchParameter("input_int", params)
	require.NotNil(t, p)
}

func TestK8SPlatform(t *testing.T) {
	nodeAffinity := structpb.NewStructValue(&structpb.Struct{
		Fields: map[string]*structpb.Value{
			"requiredDuringSchedulingIgnoredDuringExecution": structpb.NewStructValue(&structpb.Struct{
				Fields: map[string]*structpb.Value{
					"nodeSelectorTerms": structpb.NewListValue(&structpb.ListValue{
						Values: []*structpb.Value{
							structpb.NewStructValue(&structpb.Struct{
								Fields: map[string]*structpb.Value{
									"matchExpressions": structpb.NewListValue(&structpb.ListValue{
										Values: []*structpb.Value{
											structpb.NewStructValue(&structpb.Struct{
												Fields: map[string]*structpb.Value{
													"key":      structpb.NewStringValue("kubernetes.io/os"),
													"operator": structpb.NewStringValue("In"),
													"values": structpb.NewListValue(&structpb.ListValue{
														Values: []*structpb.Value{
															structpb.NewStringValue("linux"),
														},
													}),
												},
											}),
										},
									}),
								},
							}),
						},
					}),
				},
			}),
		},
	})

	// The API Server will populate runtime config with
	// the defaults in the root InputDefinition is they are
	// not user overridden. We mock this here.
	runtimeInputs := &pipelinespec.PipelineJob_RuntimeConfig{
		ParameterValues: map[string]*structpb.Value{
			"configmap_parm":              structpb.NewStringValue("cfg-2"),
			"container_image":             structpb.NewStringValue("python:3.7-alpine"),
			"cpu_limit":                   structpb.NewStringValue("200m"),
			"default_node_affinity_input": nodeAffinity,
			"empty_dir_mnt_path":          structpb.NewStringValue("/empty_dir/path"),
			"field_path":                  structpb.NewStringValue("spec.serviceAccountName"),
			"memory_limit":                structpb.NewStringValue("50Mi"),
			"node_selector_input": structpb.NewStructValue(&structpb.Struct{
				Fields: map[string]*structpb.Value{
					"kubernetes.io/os": structpb.NewStringValue("linux"),
				},
			}),
			"pull_secret_1":         structpb.NewStringValue("pull-secret-1"),
			"pull_secret_2":         structpb.NewStringValue("pull-secret-2"),
			"pull_secret_3":         structpb.NewStringValue("pull-secret-3"),
			"pvc_name_suffix_input": structpb.NewStringValue("-pvc-1"),
			"secret_param":          structpb.NewStringValue("secret-2"),
			"tolerations_dict_input": structpb.NewStructValue(&structpb.Struct{
				Fields: map[string]*structpb.Value{
					"effect":   structpb.NewStringValue("NoSchedule"),
					"key":      structpb.NewStringValue("some_foo_key6"),
					"operator": structpb.NewStringValue("Equal"),
					"value":    structpb.NewStringValue("value3"),
				},
			}),
			"tolerations_list_input": structpb.NewListValue(&structpb.ListValue{
				Values: []*structpb.Value{
					structpb.NewStructValue(&structpb.Struct{
						Fields: map[string]*structpb.Value{
							"effect":   structpb.NewStringValue("NoSchedule"),
							"key":      structpb.NewStringValue("some_foo_key4"),
							"operator": structpb.NewStringValue("Equal"),
							"value":    structpb.NewStringValue("value2"),
						},
					}),
					structpb.NewStructValue(&structpb.Struct{
						Fields: map[string]*structpb.Value{
							"effect":   structpb.NewStringValue("NoExecute"),
							"key":      structpb.NewStringValue("some_foo_key5"),
							"operator": structpb.NewStringValue("Exists"),
						},
					}),
				},
			}),
		},
	}

	tc := NewTestContextWithRootExecuted(
		t, runtimeInputs,
		"test_data/k8s_parameters.yaml",
	)
	parentTask := tc.RootTask
	require.NotNil(t, parentTask)

	// Execute all the preliminary tasks that will feed Task Output Parameters to the
	// Assert tasks (and secondary pipeline)

	// Run cfg-name-generator
	cfgNameGenExecution, _ := tc.RunContainerDriver("cfg-name-generator", parentTask, nil, false)
	cfgNameGenOutputPath := cfgNameGenExecution.ExecutorInput.Outputs.Parameters["some_output"].OutputFile
	_ = tc.RunLauncher(cfgNameGenExecution, map[string][]byte{
		"/tmp/kfp_outputs/output_metadata.json": []byte("{}"),
		cfgNameGenOutputPath:                    []byte("cfg-3"),
	}, true)

	// Run get-access-mode
	getAccessModeExecution, _ := tc.RunContainerDriver("get-access-mode", parentTask, nil, false)
	accessModeOutputPath := getAccessModeExecution.ExecutorInput.Outputs.Parameters["access_mode"].OutputFile
	_ = tc.RunLauncher(getAccessModeExecution, map[string][]byte{
		"/tmp/kfp_outputs/output_metadata.json": []byte("{}"),
		accessModeOutputPath:                    []byte("[\"ReadWriteOnce\"]"),
	}, true)

	// Run get-node-affinity
	getNodeAffinityExecution, _ := tc.RunContainerDriver("get-node-affinity", parentTask, nil, false)
	nodeAffinityOutputPath := getNodeAffinityExecution.ExecutorInput.Outputs.Parameters["node_affinity"].OutputFile
	// Serialize nodeAffinity to JSON
	nodeAffinityJSON, err := json.Marshal(nodeAffinity.GetStructValue())
	require.NoError(t, err)
	_ = tc.RunLauncher(getNodeAffinityExecution, map[string][]byte{
		"/tmp/kfp_outputs/output_metadata.json": []byte("{}"),
		nodeAffinityOutputPath:                  nodeAffinityJSON,
	}, true)

	// Run secret-name-generator
	secretNameGenExecution, _ := tc.RunContainerDriver("secret-name-generator", parentTask, nil, false)
	secretNameGenOutputPath := secretNameGenExecution.ExecutorInput.Outputs.Parameters["some_output"].OutputFile
	_ = tc.RunLauncher(secretNameGenExecution, map[string][]byte{
		"/tmp/kfp_outputs/output_metadata.json": []byte("{}"),
		secretNameGenOutputPath:                 []byte("secret-3"),
	}, true)

	// Run generate-requests-resources
	generateRequestExecution, _ := tc.RunContainerDriver("generate-requests-resources", parentTask, nil, false)
	cpuRequestOutputPath := generateRequestExecution.ExecutorInput.Outputs.Parameters["cpu_request_out"].OutputFile
	memoryRequestOutputPath := generateRequestExecution.ExecutorInput.Outputs.Parameters["memory_request_out"].OutputFile
	_ = tc.RunLauncher(generateRequestExecution, map[string][]byte{
		"/tmp/kfp_outputs/output_metadata.json": []byte("{}"),
		cpuRequestOutputPath:                    []byte("100m"),
		memoryRequestOutputPath:                 []byte("50Mi"),
	}, true)

	// Run create-pvc task since it depended on get-access-mode
	// There is no launcher for this task, we expect the output
	// parameter to be created by the driver call

	// Create a mock Kubernetes client for PVC operations
	_, createPvcTask := tc.RunContainerDriver("createpvc", parentTask, nil, true)
	require.NotNil(t, createPvcTask.Outputs)
	// CreatePvc always has one output, which is the pvc name
	require.Len(t, createPvcTask.Outputs.GetParameters(), 1)
	require.Equal(t, createPvcTask.Outputs.GetParameters()[0].ParameterKey, "name")

	// Note we don't need to mock the parameter output for k8s tasks like createpvc since
	// there is no launcher for them.

	executorInput, assertValuesTask := tc.RunContainerDriver("assert-values", parentTask, nil, true)
	require.NotNil(t, assertValuesTask.Outputs)
	require.NotNil(t, executorInput)

	podSpecString := executorInput.PodSpecPatch
	require.NotEmpty(t, podSpecString)

	podSpec := &v1.PodSpec{}
	err = json.Unmarshal([]byte(podSpecString), podSpec)
	require.NoError(t, err)

	// Check that pod spec values were correctly set
	require.Equal(t, "python:3.7-alpine", podSpec.Containers[0].Image)
	require.Contains(t, podSpec.NodeSelector, "kubernetes.io/arch")
	require.Equal(t, "amd64", podSpec.NodeSelector["kubernetes.io/arch"])
	require.Len(t, podSpec.Containers, 1)

	// The volumes are: pvc, secret, and cfg-map volumes
	require.Len(t, podSpec.Volumes, 3)
	require.Len(t, podSpec.Containers[0].VolumeMounts, 3)

	// Verify all volumes are present and configured correctly
	// Volume 0: PVC volume
	volume := podSpec.Volumes[0]
	require.Contains(t, volume.Name, "-pvc-1")
	require.Contains(t, volume.PersistentVolumeClaim.ClaimName, "-pvc-1")

	// Volume 1: Secret volume
	volume = podSpec.Volumes[1]
	require.Equal(t, "secret-2", volume.Name)
	require.Equal(t, "secret-2", volume.Secret.SecretName)
	require.False(t, *volume.Secret.Optional)

	// Volume 2: ConfigMap volume
	volume = podSpec.Volumes[2]
	require.Equal(t, "cfg-2", volume.Name)
	require.Equal(t, "cfg-2", volume.ConfigMap.Name)
	require.False(t, *volume.ConfigMap.Optional)

	// Node affinity
	require.NotNil(t, podSpec.Affinity)
	require.NotNil(t, podSpec.Affinity.NodeAffinity)
	require.NotNil(t, podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution)
	require.Len(t, podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms, 1)
	require.Len(t, podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions, 1)
	require.Equal(t, "kubernetes.io/os", podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Key)
	require.Equal(t, "In", string(podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Operator))
	require.Equal(t, []string{"linux"}, podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values)

	// Image pull secrets
	require.Len(t, podSpec.ImagePullSecrets, 6)
	expectedPullSecrets := []string{"pull-secret-1", "pull-secret-2", "pull-secret-1", "pull-secret-2", "pull-secret-3", "pull-secret-4"}
	for i, secret := range podSpec.ImagePullSecrets {
		require.Equal(t, expectedPullSecrets[i], secret.Name)
	}

	// Environment variables
	require.Len(t, podSpec.Containers[0].Env, 11)
	expectedEnvVars := []v1.EnvVar{
		{
			Name: "KFP_POD_NAME",
			ValueFrom: &v1.EnvVarSource{
				FieldRef: &v1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		},
		{
			Name: "KFP_POD_UID",
			ValueFrom: &v1.EnvVarSource{
				FieldRef: &v1.ObjectFieldSelector{
					FieldPath: "metadata.uid",
				},
			},
		},
		{
			Name: "NAMESPACE",
			ValueFrom: &v1.EnvVarSource{
				FieldRef: &v1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		},
		{
			Name: "SECRET_KEY_1",
			ValueFrom: &v1.EnvVarSource{
				SecretKeyRef: &v1.SecretKeySelector{
					LocalObjectReference: v1.LocalObjectReference{Name: "secret-1"},
					Key:                  "secretKey1",
					Optional:             &[]bool{false}[0],
				},
			},
		},
		{
			Name: "SECRET_KEY_2",
			ValueFrom: &v1.EnvVarSource{
				SecretKeyRef: &v1.SecretKeySelector{
					LocalObjectReference: v1.LocalObjectReference{Name: "secret-1"},
					Key:                  "secretKey2",
					Optional:             &[]bool{false}[0],
				},
			},
		},
		{
			Name: "SECRET_KEY_3",
			ValueFrom: &v1.EnvVarSource{
				SecretKeyRef: &v1.SecretKeySelector{
					LocalObjectReference: v1.LocalObjectReference{Name: "secret-2"},
					Key:                  "secretKey3",
					Optional:             &[]bool{false}[0],
				},
			},
		},
		{
			Name: "SECRET_KEY_4",
			ValueFrom: &v1.EnvVarSource{
				SecretKeyRef: &v1.SecretKeySelector{
					LocalObjectReference: v1.LocalObjectReference{Name: "secret-3"},
					Key:                  "secretKey4",
					Optional:             &[]bool{false}[0],
				},
			},
		},
		{
			Name: "CFG_KEY_1",
			ValueFrom: &v1.EnvVarSource{
				ConfigMapKeyRef: &v1.ConfigMapKeySelector{
					LocalObjectReference: v1.LocalObjectReference{Name: "cfg-1"},
					Key:                  "cfgKey1",
					Optional:             &[]bool{false}[0],
				},
			},
		},
		{
			Name: "CFG_KEY_2",
			ValueFrom: &v1.EnvVarSource{
				ConfigMapKeyRef: &v1.ConfigMapKeySelector{
					LocalObjectReference: v1.LocalObjectReference{Name: "cfg-1"},
					Key:                  "cfgKey2",
					Optional:             &[]bool{false}[0],
				},
			},
		},
		{
			Name: "CFG_KEY_3",
			ValueFrom: &v1.EnvVarSource{
				ConfigMapKeyRef: &v1.ConfigMapKeySelector{
					LocalObjectReference: v1.LocalObjectReference{Name: "cfg-2"},
					Key:                  "cfgKey3",
					Optional:             &[]bool{false}[0],
				},
			},
		},
		{
			Name: "CFG_KEY_4",
			ValueFrom: &v1.EnvVarSource{
				ConfigMapKeyRef: &v1.ConfigMapKeySelector{
					LocalObjectReference: v1.LocalObjectReference{Name: "cfg-3"},
					Key:                  "cfgKey4",
					Optional:             &[]bool{false}[0],
				},
			},
		},
	}
	for i, env := range podSpec.Containers[0].Env {
		require.Equal(t, expectedEnvVars[i].Name, env.Name)
		require.Equal(t, expectedEnvVars[i].ValueFrom, env.ValueFrom)
	}

	// Resource limits and requests
	require.Equal(t, "200m", podSpec.Containers[0].Resources.Limits.Cpu().String())
	require.Equal(t, "50Mi", podSpec.Containers[0].Resources.Limits.Memory().String())
	require.Equal(t, "100m", podSpec.Containers[0].Resources.Requests.Cpu().String())
	require.Equal(t, "50Mi", podSpec.Containers[0].Resources.Requests.Memory().String())

	// Tolerations
	require.Len(t, podSpec.Tolerations, 6)
	expectedTolerations := []v1.Toleration{
		{
			Key:      "some_foo_key1",
			Operator: "Equal",
			Value:    "value1",
			Effect:   "NoSchedule",
		},
		{
			Key:      "some_foo_key2",
			Operator: "Exists",
			Effect:   "NoExecute",
		},
		{
			Key:      "some_foo_key3",
			Operator: "Equal",
			Value:    "value1",
			Effect:   "NoSchedule",
		},
		{
			Key:      "some_foo_key6",
			Operator: "Equal",
			Value:    "value3",
			Effect:   "NoSchedule",
		},
		{
			Key:      "some_foo_key4",
			Operator: "Equal",
			Value:    "value2",
			Effect:   "NoSchedule",
		},
		{
			Key:      "some_foo_key5",
			Operator: "Exists",
			Effect:   "NoExecute",
		},
	}
	for i, toleration := range podSpec.Tolerations {
		require.Equal(t, expectedTolerations[i].Key, toleration.Key)
		require.Equal(t, expectedTolerations[i].Operator, toleration.Operator)
		require.Equal(t, expectedTolerations[i].Value, toleration.Value)
		require.Equal(t, expectedTolerations[i].Effect, toleration.Effect)
	}

}

// This test creates a DAG with a single task that uses a component with inputs
// and runtime constants. The test verifies that the inputs are correctly passed
// to the Runtime Task.
func TestContainerComponentInputsAndRuntimeConstants(t *testing.T) {
	// Create a root DAG execution using basic inputs
	runtimeInputs := &pipelinespec.PipelineJob_RuntimeConfig{
		ParameterValues: map[string]*structpb.Value{
			"name_in":      structpb.NewStringValue("some_name"),
			"number_in":    structpb.NewNumberValue(1.0),
			"threshold_in": structpb.NewNumberValue(0.1),
			"active_in":    structpb.NewBoolValue(false),
		},
	}

	tc := NewTestContextWithRootExecuted(t, runtimeInputs, "test_data/componentInput.yaml")

	// Run driver for process-inputs
	processInputsExecution, processInputsTask := tc.RunContainerDriver("process-inputs", tc.RootTask, nil, false)
	require.NotNil(t, processInputsExecution.ExecutorInput.Outputs)

	// Verify input parameters from driver
	params := processInputsTask.Inputs.GetParameters()
	require.Equal(t, apiv2beta1.IOType_COMPONENT_INPUT, tc.fetchParameter("name", params).GetType())
	require.Equal(t, apiv2beta1.IOType_COMPONENT_INPUT, tc.fetchParameter("number", params).GetType())
	require.Equal(t, apiv2beta1.IOType_COMPONENT_INPUT, tc.fetchParameter("active", params).GetType())
	require.Equal(t, apiv2beta1.IOType_COMPONENT_INPUT, tc.fetchParameter("threshold", params).GetType())
	require.Equal(t, apiv2beta1.IOType_RUNTIME_VALUE_INPUT, tc.fetchParameter("a_runtime_string", params).GetType())
	require.Equal(t, apiv2beta1.IOType_RUNTIME_VALUE_INPUT, tc.fetchParameter("a_runtime_number", params).GetType())
	require.Equal(t, apiv2beta1.IOType_RUNTIME_VALUE_INPUT, tc.fetchParameter("a_runtime_bool", params).GetType())

	require.Equal(t, processInputsExecution.TaskID, processInputsTask.TaskId)
	require.Equal(t, processInputsExecution.ExecutorInput.Inputs.ParameterValues["name"].GetStringValue(), "some_name")
	require.Equal(t, processInputsExecution.ExecutorInput.Inputs.ParameterValues["number"].GetNumberValue(), 1.0)
	require.Equal(t, processInputsExecution.ExecutorInput.Inputs.ParameterValues["threshold"].GetNumberValue(), 0.1)
	require.Equal(t, processInputsExecution.ExecutorInput.Inputs.ParameterValues["active"].GetBoolValue(), false)
	require.Equal(t, processInputsExecution.ExecutorInput.Inputs.ParameterValues["a_runtime_string"].GetStringValue(), "foo")
	require.Equal(t, processInputsExecution.ExecutorInput.Inputs.ParameterValues["a_runtime_number"].GetNumberValue(), 10.0)
	require.Equal(t, processInputsExecution.ExecutorInput.Inputs.ParameterValues["a_runtime_bool"].GetBoolValue(), true)

	// Mock a Launcher run by updating the task with output data
	// This test is checking artifact metadata which the mock sets explicitly
	launcherExec := tc.RunLauncher(processInputsExecution, map[string][]byte{"/tmp/kfp_outputs/output_metadata.json": []byte("{}")}, true)
	require.Len(t, launcherExec.Task.Outputs.Artifacts, 1)

	// Run driver for analyze-inputs
	analyzeInputsExecution, _ := tc.RunContainerDriver("analyze-inputs", tc.RootTask, nil, false)
	require.NotNil(t, analyzeInputsExecution.ExecutorInput.Outputs)
	require.Equal(t, 1, len(analyzeInputsExecution.ExecutorInput.Inputs.Artifacts["input_text"].Artifacts))
	launcherExec = tc.RunLauncher(analyzeInputsExecution, map[string][]byte{"/tmp/kfp_outputs/output_metadata.json": []byte("{}")}, true)
	require.Len(t, launcherExec.Task.Outputs.Artifacts, 0)

	// Verify Executor Input has the correct artifact
	artifact := analyzeInputsExecution.ExecutorInput.Inputs.Artifacts["input_text"].Artifacts[0]
	require.Equal(t, apiv2beta1.Artifact_Dataset.String(), artifact.Type.GetSchemaTitle())
	require.Equal(t, "output_text", artifact.Name)
}

func TestNestedPipelineOptionalInputChildLevel(t *testing.T) {
	// This test validates that when a DAG task (nested pipeline) has optional inputs with defaults,
	// and the parent only provides some of those inputs, the child tasks still receive the defaults
	// for the inputs that weren't provided.
	tc := NewTestContextWithRootExecuted(
		t,
		&pipelinespec.PipelineJob_RuntimeConfig{},
		"test_data/nested_pipeline_opt_input_child_level_compiled.yaml",
	)
	parentTask := tc.RootTask

	// Run the nested pipeline driver - it should receive 3 inputs from root and use defaults for the other 3
	nestedPipelineExecution, nestedPipelineTask := tc.RunDagDriver("nested-pipeline", parentTask)
	require.NotNil(t, nestedPipelineExecution)
	require.NotNil(t, nestedPipelineTask)
	require.Equal(t, apiv2beta1.PipelineTaskDetail_RUNNING, nestedPipelineTask.State)

	// The nested pipeline task should have ALL 6 inputs (3 from parent + 3 defaults)
	require.NotNil(t, nestedPipelineTask.Inputs)
	require.NotNil(t, nestedPipelineTask.Inputs.Parameters)

	inputParams := make(map[string]*structpb.Value)
	for _, param := range nestedPipelineTask.Inputs.Parameters {
		inputParams[param.ParameterKey] = param.Value
	}

	// Verify all 6 parameters are present
	require.Contains(t, inputParams, "nestedInputBool1", "nestedInputBool1 should be present")
	require.Contains(t, inputParams, "nestedInputBool2", "nestedInputBool2 should be present (from default)")
	require.Contains(t, inputParams, "nestedInputInt1", "nestedInputInt1 should be present")
	require.Contains(t, inputParams, "nestedInputInt2", "nestedInputInt2 should be present (from default)")
	require.Contains(t, inputParams, "nestedInputStr1", "nestedInputStr1 should be present")
	require.Contains(t, inputParams, "nestedInputStr2", "nestedInputStr2 should be present (from default)")

	// Verify the values are correct
	// From parent (root pipeline)
	require.Equal(t, true, inputParams["nestedInputBool1"].GetBoolValue(),
		"nestedInputBool1 should be true (from parent)")
	require.Equal(t, 1.0, inputParams["nestedInputInt1"].GetNumberValue(),
		"nestedInputInt1 should be 1.0 (from parent)")
	require.Equal(t, "Input - pipeline", inputParams["nestedInputStr1"].GetStringValue(),
		"nestedInputStr1 should be 'Input - pipeline' (from parent)")

	// From defaults (not provided by parent)
	require.Equal(t, false, inputParams["nestedInputBool2"].GetBoolValue(),
		"nestedInputBool2 should be false (from default)")
	require.Equal(t, 0.0, inputParams["nestedInputInt2"].GetNumberValue(),
		"nestedInputInt2 should be 0.0 (from default)")
	require.Equal(t, "Input 2 - nested pipeline", inputParams["nestedInputStr2"].GetStringValue(),
		"nestedInputStr2 should be 'Input 2 - nested pipeline' (from default)")
}
