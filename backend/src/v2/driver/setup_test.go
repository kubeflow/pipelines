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
	"time"

	"github.com/google/uuid"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient/kfpapi"
	clientmanager "github.com/kubeflow/pipelines/backend/src/v2/client_manager"
	"github.com/kubeflow/pipelines/backend/src/v2/component"
	"github.com/kubeflow/pipelines/backend/src/v2/driver/common"
	"github.com/kubeflow/pipelines/kubernetes_platform/go/kubernetesplatform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"k8s.io/client-go/kubernetes/fake"
)

const TestPipelineName = "test-pipeline"
const TestNamespace = "test-namespace"

type TestContext struct {
	Run *apiv2beta1.Run
	util.ScopePath
	T             *testing.T
	PipelineSpec  *pipelinespec.PipelineSpec
	RootTask      *apiv2beta1.PipelineTaskDetail
	PlatformSpec  *pipelinespec.PlatformSpec
	ClientManager clientmanager.ClientManagerInterface
	MockAPI       *kfpapi.MockAPI
	MockObjStore  *component.MockObjectStoreClient // Shared across all launchers in this test context
}

// NewTestContextWithRootExecuted creates a new test context with basic configuration
// It will automatically launch a root DAG using the provided input
// and update the scope path.
// When using a Test Context note the following:
//   - Output parameters, artifacts and artifact-tasks are not auto created
//     and must be manually created by using mock launcher calls. This
//     includes outputs for upstream dags, since we expect launcher->api server
//     to handle these.
//   - All other input resolutions are handled automatically.
//   - ScopePath is automatically pushed/popped for container calls.
//   - ScopePath is automatically updated for dags upon entering a dag (push),
//     but not exiting the dag (pop), this should be handled by the caller
//     via tc.ExitDag()
//   - When Updating TextContext, ensure for any driver executions the Run
//     object is refreshed using RefreshRun() otherwise you might use stale data.
func NewTestContextWithRootExecuted(t *testing.T, runtimeConfig *pipelinespec.PipelineJob_RuntimeConfig, pipelinePath string) *TestContext {
	t.Helper()
	proxy.InitializeConfigWithEmptyForTests()
	mockAPI := kfpapi.NewMockAPI()
	tc := &TestContext{
		ClientManager: clientmanager.NewFakeClientManager(fake.NewClientset(), mockAPI),
		MockAPI:       mockAPI,
		MockObjStore:  component.NewMockObjectStoreClient(), // Shared object store
	}

	// Load pipeline spec
	pipelineSpec, platformSpec, err := util.LoadPipelineAndPlatformSpec(pipelinePath)
	require.NoError(t, err)
	require.NotNil(t, platformSpec)
	require.NotNil(t, pipelineSpec)

	// Convert pipelineSpec to structpb.Struct
	pipelineSpecJSON, err := protojson.Marshal(pipelineSpec)
	require.NoError(t, err)
	pipelineSpecStruct := &structpb.Struct{}
	err = protojson.Unmarshal(pipelineSpecJSON, pipelineSpecStruct)
	require.NoError(t, err)

	// Set up scope path
	tc.ScopePath, err = util.NewScopePathFromStruct(pipelineSpecStruct)
	require.NoError(t, err)
	tc.PipelineSpec = pipelineSpec
	tc.PlatformSpec = platformSpec

	// Create a test run
	run := tc.CreateTestRun(t, "test-pipeline")
	require.NotNil(t, run)

	tc.Run = run
	tc.T = t

	// Create a root DAG execution using basic inputs
	_, rootTask := tc.RunRootDag(tc, run, runtimeConfig)
	tc.RootTask = rootTask
	return tc
}

// CreateTestRun creates a test run with basic configuration
func (tc *TestContext) CreateTestRun(t *testing.T, pipelineName string) *apiv2beta1.Run {
	t.Helper()

	// Convert the loaded pipeline spec to structpb.Struct for the run
	// Marshal to JSON and then unmarshal into structpb
	pipelineSpecJSON, err := protojson.Marshal(tc.PipelineSpec)
	require.NoError(t, err)
	pipelineSpecStruct := &structpb.Struct{}
	err = protojson.Unmarshal(pipelineSpecJSON, pipelineSpecStruct)
	require.NoError(t, err)

	uuid, _ := uuid.NewRandom()
	run := &apiv2beta1.Run{
		RunId:          uuid.String(),
		DisplayName:    fmt.Sprintf("test-run-%s-%d", pipelineName, time.Now().Unix()),
		PipelineSource: &apiv2beta1.Run_PipelineSpec{PipelineSpec: pipelineSpecStruct},
		RuntimeConfig:  &apiv2beta1.RuntimeConfig{},
		State:          apiv2beta1.RuntimeState_RUNNING,
	}

	tc.MockAPI.AddRun(run)
	return run
}

// CreateTestTask creates a test task with the given configuration
func (tc *TestContext) CreateTestTask(
	t *testing.T,
	runID,
	taskName string,
	taskType apiv2beta1.PipelineTaskDetail_TaskType,
	inputParams, outputParams []*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter,
) *apiv2beta1.PipelineTaskDetail {
	t.Helper()

	podUUID, _ := uuid.NewRandom()
	task := &apiv2beta1.PipelineTaskDetail{
		Name:        taskName,
		DisplayName: taskName,
		RunId:       runID,
		Type:        taskType,
		State:       apiv2beta1.PipelineTaskDetail_RUNNING,
		Pods: []*apiv2beta1.PipelineTaskDetail_TaskPod{
			{
				Name: fmt.Sprintf("%s-pod", taskName),
				Uid:  podUUID.String(),
				Type: apiv2beta1.PipelineTaskDetail_DRIVER,
			},
		},
		Inputs: &apiv2beta1.PipelineTaskDetail_InputOutputs{
			Parameters: inputParams,
		},
		Outputs: &apiv2beta1.PipelineTaskDetail_InputOutputs{
			Parameters: outputParams,
		},
	}

	createdTask, err := tc.ClientManager.KFPAPIClient().CreateTask(context.Background(), &apiv2beta1.CreateTaskRequest{
		Task: task,
	})
	require.NoError(t, err)
	return createdTask
}

// CreateTestArtifact creates a test artifact with the given configuration
func (tc *TestContext) CreateTestArtifact(t *testing.T, name, artifactType string) *apiv2beta1.Artifact {
	t.Helper()

	artifact := &apiv2beta1.Artifact{
		Name: name,
		Type: apiv2beta1.Artifact_Dataset, // Default type
	}

	// Set specific type if provided
	switch artifactType {
	case "model":
		artifact.Type = apiv2beta1.Artifact_Model
	case "metric":
		artifact.Type = apiv2beta1.Artifact_Metric
	}

	createdArtifact, err := tc.ClientManager.KFPAPIClient().CreateArtifact(context.Background(), &apiv2beta1.CreateArtifactRequest{
		Artifact: artifact,
	})
	require.NoError(t, err)
	return createdArtifact
}

// CreateTestArtifactTask creates an artifact-task relationship
func (tc *TestContext) CreateTestArtifactTask(t *testing.T, artifactID, taskID, runID, key string,
	producer *apiv2beta1.IOProducer, artifactType apiv2beta1.IOType) *apiv2beta1.ArtifactTask {
	t.Helper()

	artifactTask := &apiv2beta1.ArtifactTask{
		ArtifactId: artifactID,
		TaskId:     taskID,
		RunId:      runID,
		Type:       artifactType,
		Producer:   producer,
		Key:        key,
	}

	createdArtifactTask, err := tc.ClientManager.KFPAPIClient().CreateArtifactTask(context.Background(), &apiv2beta1.CreateArtifactTaskRequest{
		ArtifactTask: artifactTask,
	})
	require.NoError(t, err)
	return createdArtifactTask
}

// CreateParameter creates a test parameter with the given name and value
func CreateParameter(value, key string,
	producer *apiv2beta1.IOProducer) *apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter {
	val, _ := structpb.NewValue(value)
	param := &apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
		Value:        val,
		ParameterKey: key,
		Producer:     producer,
	}
	return param
}

// Example test demonstrating the usage including artifact population
func TestTestContext(t *testing.T) {
	// Setup test environment
	testSetup := NewTestContextWithRootExecuted(t, &pipelinespec.PipelineJob_RuntimeConfig{}, "test_data/taskOutputArtifact_test.yaml")
	require.NotNil(t, testSetup)
	assert.NotEmpty(t, testSetup.Run.RunId)

	// Create a test run
	run := testSetup.CreateTestRun(t, "test-pipeline")
	assert.NotNil(t, run)
	assert.NotEmpty(t, run.RunId)
	assert.Equal(t, "primary-pipeline", run.GetPipelineSpec().Fields["pipelineInfo"].GetStructValue().Fields["name"].GetStringValue())

	// Create test tasks
	task1 := testSetup.CreateTestTask(t,
		run.RunId,
		"producer-task",
		apiv2beta1.PipelineTaskDetail_RUNTIME,
		[]*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
			CreateParameter(
				"input1",
				"pipelinechannel--args-generator-op-Output",
				&apiv2beta1.IOProducer{TaskName: "some-task"},
			),
		},
		[]*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
			CreateParameter(
				"output1",
				"msg",
				nil,
			),
			CreateParameter(
				"output2",
				"",
				nil,
			),
		})
	task2 := testSetup.CreateTestTask(t, run.RunId, "consumer-task", apiv2beta1.PipelineTaskDetail_RUNTIME,
		[]*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
			CreateParameter(
				"input4",
				"input4key",
				nil,
			),
		},
		[]*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
			CreateParameter(
				"output3",
				"pipelinechannel--split-ids-Output",
				nil,
			),
		})

	// Create test artifacts
	artifact1 := testSetup.CreateTestArtifact(t, "output-data", "dataset")
	artifact2 := testSetup.CreateTestArtifact(t, "trained-model", "model")

	// Create artifact-task relationships
	// task1 produces artifact1 (output)
	testSetup.CreateTestArtifactTask(t,
		artifact1.ArtifactId, task1.TaskId, run.RunId, "pipelinechannel--loop_parameter-loop-item-1",
		&apiv2beta1.IOProducer{
			TaskName: task1.Name,
		},
		apiv2beta1.IOType_OUTPUT,
	)

	// task2 consumes artifact1 (input)
	testSetup.CreateTestArtifactTask(t,
		artifact1.ArtifactId, task2.TaskId, run.RunId, "pipelinechannel--loop_parameter-loop-item-2",
		&apiv2beta1.IOProducer{
			TaskName: task1.Name,
		},
		apiv2beta1.IOType_COMPONENT_INPUT,
	)
	// task2 produces artifact2 (output)
	testSetup.CreateTestArtifactTask(t,
		artifact2.ArtifactId, task2.TaskId, run.RunId, "pipelinechannel--loop_parameter-loop-item",
		&apiv2beta1.IOProducer{
			TaskName: task2.Name,
		},
		apiv2beta1.IOType_OUTPUT,
	)

	// Test getting run with populated tasks and artifacts
	fullView := apiv2beta1.GetRunRequest_FULL
	populatedRun, err := testSetup.ClientManager.KFPAPIClient().GetRun(context.Background(), &apiv2beta1.GetRunRequest{RunId: run.RunId, View: &fullView})
	require.NoError(t, err)
	assert.NotNil(t, populatedRun)
	assert.Len(t, populatedRun.Tasks, 2)

	// Verify task1 has correct artifacts (1 output)
	var producerTask *apiv2beta1.PipelineTaskDetail
	for _, task := range populatedRun.Tasks {
		if task.Name == "producer-task" {
			producerTask = task
			break
		}
	}
	require.NotNil(t, producerTask)
	assert.Len(t, producerTask.Inputs.Artifacts, 0)  // No input artifacts
	assert.Len(t, producerTask.Outputs.Artifacts, 1) // 1 output artifact

	// Verify task2 has correct artifacts (1 input, 1 output)
	var consumerTask *apiv2beta1.PipelineTaskDetail
	for _, task := range populatedRun.Tasks {
		if task.Name == "consumer-task" {
			consumerTask = task
			break
		}
	}
	require.NotNil(t, consumerTask)
	assert.Len(t, consumerTask.Inputs.Artifacts, 1)  // 1 input artifact
	assert.Len(t, consumerTask.Outputs.Artifacts, 1) // 1 output artifact

	// Verify producer information is correctly set
	inputArtifact := consumerTask.Inputs.Artifacts[0]
	assert.Equal(t, "producer-task", inputArtifact.GetProducer().TaskName)
	assert.Equal(t, "pipelinechannel--loop_parameter-loop-item-2", inputArtifact.GetArtifactKey())
}

func (tc *TestContext) RunRootDag(testSetup *TestContext, run *apiv2beta1.Run, runtimeConfig *pipelinespec.PipelineJob_RuntimeConfig) (*Execution, *apiv2beta1.PipelineTaskDetail) {
	tc.RefreshRun()
	defer tc.RefreshRun()
	err := tc.Push("root")
	require.NoError(tc.T, err)

	opts := common.Options{
		PipelineName:             TestPipelineName,
		Run:                      run,
		Component:                tc.ScopePath.GetLast().GetComponentSpec(),
		ParentTask:               nil,
		IterationIndex:           -1,
		RuntimeConfig:            runtimeConfig,
		Namespace:                TestNamespace,
		Task:                     nil,
		Container:                nil,
		KubernetesExecutorConfig: &kubernetesplatform.KubernetesExecutorConfig{},
		PipelineLogLevel:         "1",
		PublishLogs:              "false",
		CacheDisabled:            false,
		DriverType:               "ROOT_DAG",
		TaskName:                 "", // Empty for root driver
		PodName:                  "system-dag-driver",
		PodUID:                   "some-uid",
		ScopePath:                tc.ScopePath,
	}
	// Execute RootDAG
	execution, err := RootDAG(context.Background(), opts, testSetup.ClientManager)
	require.NoError(tc.T, err)
	require.NotNil(tc.T, execution)

	task, err := tc.ClientManager.KFPAPIClient().GetTask(context.Background(), &apiv2beta1.GetTaskRequest{TaskId: execution.TaskID})
	require.NoError(tc.T, err)
	require.NotNil(tc.T, task)
	require.Equal(tc.T, execution.TaskID, task.TaskId)

	return execution, task
}

func (tc *TestContext) RunDagDriver(
	taskName string,
	parentTask *apiv2beta1.PipelineTaskDetail) (*Execution, *apiv2beta1.PipelineTaskDetail) {
	t := tc.T
	tc.RefreshRun()
	defer tc.RefreshRun()

	err := tc.Push(taskName)
	require.NoError(t, err)
	taskSpec := tc.GetLast().GetTaskSpec()

	opts := tc.setupDagOptions(parentTask, taskSpec, nil)

	execution, err := DAG(context.Background(), opts, tc.ClientManager)
	require.NoError(t, err)
	require.NotNil(t, execution)

	task, err := tc.ClientManager.KFPAPIClient().GetTask(context.Background(), &apiv2beta1.GetTaskRequest{TaskId: execution.TaskID})
	require.NoError(t, err)
	require.NotNil(t, task)
	require.Equal(t, execution.TaskID, task.TaskId)
	require.Equal(t, taskName, task.GetName())

	return execution, task
}

// RunContainerDriver runs a container for the given task.
// If autoUpdateScope is true, the scope path will
// be popped after the container is completed.
func (tc *TestContext) RunContainerDriver(
	taskName string,
	parentTask *apiv2beta1.PipelineTaskDetail,
	iterationIndex *int64,
	autoUpdateScope bool,
) (*Execution, *apiv2beta1.PipelineTaskDetail) {
	tc.RefreshRun()
	defer tc.RefreshRun()

	// Add scope path and pop it once done
	err := tc.Push(taskName)

	if autoUpdateScope {
		defer func() {
			_, ok := tc.Pop()
			require.True(tc.T, ok)
		}()
	}

	require.NoError(tc.T, err)
	taskSpec := tc.GetLast().GetTaskSpec()

	kubernetesExecutorConfig, err := util.LoadKubernetesExecutorConfig(tc.GetLast().GetComponentSpec(), tc.PlatformSpec)
	require.NoError(tc.T, err)
	opts := tc.setupContainerOptions(parentTask, taskSpec, kubernetesExecutorConfig)

	if iterationIndex != nil {
		opts.IterationIndex = int(*iterationIndex)
	}

	execution, err := Container(context.Background(), opts, tc.ClientManager)
	require.NoError(tc.T, err)
	require.NotNil(tc.T, execution)

	task, err := tc.ClientManager.KFPAPIClient().GetTask(context.Background(), &apiv2beta1.GetTaskRequest{TaskId: execution.TaskID})
	require.NoError(tc.T, err)
	require.NotNil(tc.T, task)
	require.Equal(tc.T, execution.TaskID, task.TaskId)
	require.Equal(tc.T, taskName, task.GetName())
	if task.State != apiv2beta1.PipelineTaskDetail_CACHED {
		// In the case of k8s ops like createpvc/deletepvc
		// There are no launchers, so we mark them success
		// within driver.
		require.True(
			tc.T,
			task.State == apiv2beta1.PipelineTaskDetail_RUNNING ||
				task.State == apiv2beta1.PipelineTaskDetail_SUCCEEDED,
			"expected task.Status to be RUNNING or SUCCEEDED, got %v",
			task.State,
		)
	}

	return execution, task
}

func (tc *TestContext) RefreshRun() {
	t := tc.T
	fullView := apiv2beta1.GetRunRequest_FULL
	run, err := tc.ClientManager.KFPAPIClient().GetRun(context.Background(), &apiv2beta1.GetRunRequest{RunId: tc.Run.RunId, View: &fullView})
	require.NoError(t, err)
	tc.Run = run
}

func (tc *TestContext) ExitDag() {
	_, ok := tc.Pop()
	require.True(tc.T, ok)
}

func (tc *TestContext) MockLauncherOutputParameterCreate(
	taskID string,
	parameterKey string,
	value *structpb.Value,
	outputType apiv2beta1.IOType,
	producerTask string,
	producerIteration *int64,
) {
	// Get Task
	task, err := tc.ClientManager.KFPAPIClient().GetTask(context.Background(), &apiv2beta1.GetTaskRequest{TaskId: taskID})
	require.NoError(tc.T, err)
	require.NotNil(tc.T, task)

	newParameter := &apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
		Value:        value,
		Type:         outputType,
		ParameterKey: parameterKey,
		Producer: &apiv2beta1.IOProducer{
			TaskName: producerTask,
		},
	}
	if producerIteration != nil {
		newParameter.Producer.Iteration = producerIteration
	}
	parameters := task.Outputs.Parameters
	parameters = append(parameters, newParameter)
	task.Outputs.Parameters = parameters
	// Update Task via kfpAPI UpdateTask
	task, err = tc.ClientManager.KFPAPIClient().UpdateTask(context.Background(), &apiv2beta1.UpdateTaskRequest{
		TaskId: taskID,
		Task:   task,
	})
	require.NoError(tc.T, err)
	require.NotNil(tc.T, task)

	tc.RefreshRun()
}

// This helper will update a Runtime Tasks inputs with optional values if
// no upstream input was provided.
func (tc *TestContext) MockLauncherDefaultInputParametersUpdate(taskID string, componentSpec *pipelinespec.ComponentSpec) *apiv2beta1.PipelineTaskDetail {
	defer func() { tc.RefreshRun() }()

	// Get Task
	task, err := tc.ClientManager.KFPAPIClient().GetTask(context.Background(), &apiv2beta1.GetTaskRequest{TaskId: taskID})
	require.NoError(tc.T, err)
	require.NotNil(tc.T, task)

	taskInputParameters := task.GetInputs().GetParameters()

	// Find all optional parameters that have default values and add them to the task input parameters
	for key, inputParamSpec := range componentSpec.GetInputDefinitions().GetParameters() {
		if !parameterExistsWithKey(taskInputParameters, key) {
			var value *structpb.Value
			if inputParamSpec.GetDefaultValue() != nil {
				value = inputParamSpec.GetDefaultValue()
			} else {
				require.True(tc.T, inputParamSpec.IsOptional, "Parameter %s is not optional", key)
				continue
			}
			require.NotNil(tc.T, value)
			parameterIO := &apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
				Value:        value,
				Type:         apiv2beta1.IOType_COMPONENT_DEFAULT_INPUT,
				ParameterKey: key,
			}
			taskInputParameters = append(taskInputParameters, parameterIO)
		}
	}
	task.Inputs.Parameters = taskInputParameters
	task, err = tc.ClientManager.KFPAPIClient().UpdateTask(context.Background(), &apiv2beta1.UpdateTaskRequest{
		TaskId: taskID,
		Task:   task,
	})
	require.NoError(tc.T, err)
	require.NotNil(tc.T, task)
	return task
}

func parameterExistsWithKey(parameters []*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter, key string) bool {
	for _, parameter := range parameters {
		if parameter.ParameterKey == key {
			return true
		}
	}
	return false
}

func (tc *TestContext) MockLauncherOutputArtifactCreate(
	taskID string,
	artifactKey string,
	artifactType apiv2beta1.Artifact_ArtifactType,
	outputType apiv2beta1.IOType,
	producerTask string,
	producerIteration *int64,
) string {
	t := tc.T
	artifactID, _ := uuid.NewRandom()
	outputArtifact := &apiv2beta1.Artifact{
		ArtifactId: artifactID.String(),
		Name:       artifactKey,
		Type:       artifactType,
		Uri:        util.StringPointer(fmt.Sprintf("s3://some.location/%s", artifactKey)),
		Namespace:  TestNamespace,
		Metadata: map[string]*structpb.Value{
			"display_name":  structpb.NewStringValue(artifactKey),
			"task_id":       structpb.NewStringValue(taskID),
			"producer_task": structpb.NewStringValue(producerTask),
		},
	}
	createArtifact, err := tc.ClientManager.KFPAPIClient().CreateArtifact(
		context.Background(),
		&apiv2beta1.CreateArtifactRequest{
			Artifact: outputArtifact,
		})
	require.NoError(t, err)
	require.NotNil(t, createArtifact)

	artifactTask := &apiv2beta1.ArtifactTask{
		ArtifactId: artifactID.String(),
		TaskId:     taskID,
		RunId:      tc.Run.GetRunId(),
		Key:        artifactKey,
		Producer:   &apiv2beta1.IOProducer{TaskName: producerTask},
		Type:       outputType,
	}
	if producerIteration != nil {
		artifactTask.Producer.Iteration = producerIteration
	}
	at, err := tc.ClientManager.KFPAPIClient().CreateArtifactTask(
		context.Background(),
		&apiv2beta1.CreateArtifactTaskRequest{
			ArtifactTask: artifactTask,
		})
	require.NoError(t, err)
	require.NotNil(t, at)
	tc.RefreshRun()
	return artifactID.String()
}

func (tc *TestContext) MockLauncherArtifactTaskCreate(
	producerTaskName, taskID, key string,
	artifactID string, producerIteration *int64,
	outputType apiv2beta1.IOType) {
	t := tc.T
	at := &apiv2beta1.ArtifactTask{
		ArtifactId: artifactID,
		TaskId:     taskID,
		RunId:      tc.Run.GetRunId(),
		Key:        key,
		Type:       outputType,
		Producer:   &apiv2beta1.IOProducer{TaskName: producerTaskName},
	}
	if producerIteration != nil {
		at.Producer.Iteration = producerIteration
	}
	result, err := tc.ClientManager.KFPAPIClient().CreateArtifactTask(
		context.Background(),
		&apiv2beta1.CreateArtifactTaskRequest{ArtifactTask: at})
	require.NoError(t, err)
	require.NotNil(t, result)
	tc.RefreshRun()
}

// LauncherExecution holds the result of a launcher execution
type LauncherExecution struct {
	// The launcher instance (with mocks still attached)
	Launcher *component.LauncherV2
	// Mock instances for verification
	MockFS       *component.MockFileSystem
	MockCmd      *component.MockCommandExecutor
	MockObjStore *component.MockObjectStoreClient
	// The task that was executed
	Task *apiv2beta1.PipelineTaskDetail
}

// RunLauncher executes a launcher for the given execution with mocked dependencies.
// This simulates what the launcher would do when executing user code.
// It uses the ExecutorInput that was already prepared by the driver.
//
// Usage:
//
//	execution, _ := tc.RunContainerDriver("task-name", parentTask, nil, true)
//	launcherExec := tc.RunLauncher(execution, map[string][]byte{
//	    "/tmp/outputs/metric": []byte("0.95"),
//	})
//
//	// Verify command was executed
//	assert.Equal(t, 1, launcherExec.MockCmd.CallCount())
//
//	// Verify artifacts were uploaded
//	uploads := launcherExec.MockObjStore.GetUploadCallsForKey("model")
//	assert.Len(t, uploads, 1)
func (tc *TestContext) RunLauncher(execution *Execution, outputFiles map[string][]byte, autoUpdateScope bool) *LauncherExecution {
	t := tc.T
	ctx := context.Background()

	// Get the task that was created by the driver
	task, err := tc.ClientManager.KFPAPIClient().GetTask(ctx, &apiv2beta1.GetTaskRequest{TaskId: execution.TaskID})
	require.NoError(t, err)
	require.NotNil(t, task)

	// Use the ExecutorInput that was already prepared by the driver
	executorInput := execution.ExecutorInput
	require.NotNil(t, executorInput, "ExecutorInput should be set by driver")

	// Get componentSpec and taskSpec from the current scope path
	// The TestContext's ScopePath should already be at the right location after RunContainerDriver
	componentSpec := tc.GetLast().GetComponentSpec()
	taskSpec := tc.GetLast().GetTaskSpec()
	require.NotNil(t, componentSpec, "Component spec not found")
	require.NotNil(t, taskSpec, "Task spec not found")

	// Marshal executor input
	executorInputJSON, err := protojson.Marshal(executorInput)
	require.NoError(t, err)

	// Create launcher options
	var iterPtr *int64
	var parentTaskForLauncher *apiv2beta1.PipelineTaskDetail
	if task.ParentTaskId != nil && *task.ParentTaskId != "" {
		// Get the parent task
		parentTask, err := tc.ClientManager.KFPAPIClient().GetTask(ctx, &apiv2beta1.GetTaskRequest{TaskId: *task.ParentTaskId})
		if err == nil {
			parentTaskForLauncher = parentTask
			// Extract iteration index from the task's type attributes if this is an iteration
			if task.GetTypeAttributes() != nil && task.GetTypeAttributes().IterationIndex != nil {
				iterPtr = task.GetTypeAttributes().IterationIndex
			}
		}
	}

	// Convert PipelineSpec to structpb.Struct
	pipelineSpecJSON, err := protojson.Marshal(tc.PipelineSpec)
	require.NoError(t, err)
	pipelineSpecStruct := &structpb.Struct{}
	err = protojson.Unmarshal(pipelineSpecJSON, pipelineSpecStruct)
	require.NoError(t, err)

	opts := &component.LauncherV2Options{
		Namespace:      TestNamespace,
		PodName:        fmt.Sprintf("%s-pod", task.GetName()),
		PodUID:         uuid.New().String(),
		PipelineName:   TestPipelineName,
		PublishLogs:    "false",
		ComponentSpec:  componentSpec,
		TaskSpec:       taskSpec,
		ScopePath:      tc.ScopePath,
		Run:            tc.Run,
		ParentTask:     parentTaskForLauncher,
		Task:           task,
		IterationIndex: iterPtr,
		PipelineSpec:   pipelineSpecStruct,
	}

	// Create launcher with a dummy command (will be mocked anyway)
	launcher, err := component.NewLauncherV2(
		string(executorInputJSON),
		[]string{"python", "component.py"}, // Default command, will be mocked
		opts,
		tc.ClientManager,
	)
	require.NoError(t, err)

	// Setup mocks
	mockFS := component.NewMockFileSystem()
	mockCmd := component.NewMockCommandExecutor()
	// Use the shared mock object store from TestContext
	mockObjStore := tc.MockObjStore

	// Configure output files
	for path, content := range outputFiles {
		mockFS.SetFileContent(path, content)
	}

	// Set output metadata file if not provided
	if _, exists := outputFiles["/tmp/kfp_outputs/output_metadata.json"]; !exists {
		mockFS.SetFileContent("/tmp/kfp_outputs/output_metadata.json", []byte("{}"))
	}

	// Configure command to succeed by default
	mockCmd.RunError = nil

	// Pre-populate input artifacts in mock object store
	for _, ioArtifact := range task.GetInputs().GetArtifacts() {
		for _, artifact := range ioArtifact.Artifacts {
			if artifact.GetUri() != "" {
				// Simulate artifact already exists in object store
				mockObjStore.SetArtifact(artifact.GetUri(), []byte("input data"))
			}
		}
	}

	// Inject mocks (file system, command executor, object store)
	// Note: KFP API client comes from clientManager which already has MockAPI
	launcher.WithFileSystem(mockFS).
		WithCommandExecutor(mockCmd).
		WithObjectStore(mockObjStore)

	// Execute the launcher using the full Execute() method
	// This will test the complete flow including:
	// - Task output parameter updates
	// - Task status updates to SUCCEEDED
	// - Status propagation up the DAG hierarchy
	err = launcher.Execute(ctx)
	require.NoError(t, err, "Launcher execution failed for task %s", task.GetName())

	require.Equal(t, 1, mockCmd.CallCount())

	// Refresh the run to get updated task data
	tc.RefreshRun()

	// Get updated task
	updatedTask, err := tc.ClientManager.KFPAPIClient().GetTask(ctx, &apiv2beta1.GetTaskRequest{TaskId: execution.TaskID})
	require.NoError(t, err)

	// Pop scope if autoUpdateScope is true
	if autoUpdateScope {
		_, ok := tc.Pop()
		require.True(t, ok, "Failed to pop scope path")
	}

	return &LauncherExecution{
		Launcher:     launcher,
		MockFS:       mockFS,
		MockCmd:      mockCmd,
		MockObjStore: mockObjStore,
		Task:         updatedTask,
	}
}

func (tc *TestContext) setupDagOptions(
	parentTask *apiv2beta1.PipelineTaskDetail,
	taskSpec *pipelinespec.PipelineTaskSpec,
	kubernetesExecutorConfig *kubernetesplatform.KubernetesExecutorConfig,
) common.Options {
	componentSpec := tc.PipelineSpec.Components[taskSpec.ComponentRef.Name]

	ds := tc.PipelineSpec.GetDeploymentSpec()
	platformDeploymentSpec := &pipelinespec.PlatformDeploymentConfig{}

	b, err := protojson.Marshal(ds)
	require.NoError(tc.T, err)
	err = protojson.Unmarshal(b, platformDeploymentSpec)
	require.NoError(tc.T, err)
	assert.NotNil(tc.T, platformDeploymentSpec)

	cs := platformDeploymentSpec.Executors[componentSpec.GetExecutorLabel()]
	containerExecutorSpec := &pipelinespec.PipelineDeploymentConfig_ExecutorSpec{}
	b, err = protojson.Marshal(cs)
	require.NoError(tc.T, err)
	err = protojson.Unmarshal(b, containerExecutorSpec)
	require.NoError(tc.T, err)
	assert.NotNil(tc.T, containerExecutorSpec)

	return common.Options{
		PipelineName:             TestPipelineName,
		Run:                      tc.Run,
		Component:                componentSpec,
		ParentTask:               parentTask,
		IterationIndex:           -1,
		RuntimeConfig:            nil,
		Namespace:                TestNamespace,
		Task:                     taskSpec,
		Container:                nil,
		KubernetesExecutorConfig: kubernetesExecutorConfig,
		RunName:                  "",
		RunDisplayName:           "",
		PipelineLogLevel:         "1",
		PublishLogs:              "false",
		CacheDisabled:            false,
		DriverType:               "DAG",
		TaskName:                 taskSpec.TaskInfo.GetName(),
		PodName:                  "system-dag-driver",
		PodUID:                   "some-uid",
		ScopePath:                tc.ScopePath,
	}
}

func (tc *TestContext) setupContainerOptions(
	parentTask *apiv2beta1.PipelineTaskDetail,
	taskSpec *pipelinespec.PipelineTaskSpec,
	kubernetesExecutorConfig *kubernetesplatform.KubernetesExecutorConfig,
) common.Options {
	componentSpec := tc.PipelineSpec.Components[taskSpec.ComponentRef.Name]

	ds := tc.PipelineSpec.GetDeploymentSpec()
	platformDeploymentSpec := &pipelinespec.PlatformDeploymentConfig{}

	b, err := protojson.Marshal(ds)
	require.NoError(tc.T, err)
	err = protojson.Unmarshal(b, platformDeploymentSpec)
	require.NoError(tc.T, err)
	assert.NotNil(tc.T, platformDeploymentSpec)

	cs := platformDeploymentSpec.Executors[componentSpec.GetExecutorLabel()]
	containerExecutorSpec := &pipelinespec.PipelineDeploymentConfig_ExecutorSpec{}
	b, err = protojson.Marshal(cs)
	require.NoError(tc.T, err)
	err = protojson.Unmarshal(b, containerExecutorSpec)
	require.NoError(tc.T, err)
	assert.NotNil(tc.T, containerExecutorSpec)

	return common.Options{
		PipelineName:             TestPipelineName,
		Run:                      tc.Run,
		Component:                componentSpec,
		ParentTask:               parentTask,
		IterationIndex:           -1,
		RuntimeConfig:            nil,
		Namespace:                TestNamespace,
		Task:                     taskSpec,
		Container:                containerExecutorSpec.GetContainer(),
		KubernetesExecutorConfig: kubernetesExecutorConfig,
		PipelineLogLevel:         "1",
		PublishLogs:              "false",
		CacheDisabled:            false,
		DriverType:               "CONTAINER",
		TaskName:                 taskSpec.TaskInfo.GetName(),
		PodName:                  "system-container-impl",
		PodUID:                   "some-uid",
		ScopePath:                tc.ScopePath,
	}
}

func (tc *TestContext) fetchParameter(key string, params []*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter) *apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter {
	for _, p := range params {
		if key == p.ParameterKey {
			return p
		}
	}
	return nil
}
