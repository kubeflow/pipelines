package poc

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/v2/component"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeDockerEngine struct {
	container          *dockerInspection
	createCount        int
	startCount         int
	stopCount          int
	removeCount        int
	waitErr            error
	inspectErr         error
	exitCode           int
	startError         string
	lostCreateResponse bool
	createArgs         []string
}

func (f *fakeDockerEngine) run(ctx context.Context, args ...string) (string, string, error) {
	if err := ctx.Err(); err != nil {
		return "", "", err
	}
	switch args[0] {
	case "info":
		return "28", "", nil
	case "inspect":
		if f.inspectErr != nil {
			return "", "daemon unavailable", f.inspectErr
		}
		if f.container == nil {
			return "", "Error: No such container", errors.New("missing")
		}
		raw, err := json.Marshal([]*dockerInspection{f.container})
		return string(raw), "", err
	case "create":
		f.createCount++
		f.createArgs = append([]string(nil), args...)
		f.container = &dockerInspection{ID: "container-uid", Name: "/" + args[2]}
		f.container.Config.Labels = map[string]string{}
		for index, arg := range args {
			if arg == "--label" {
				key, value, _ := strings.Cut(args[index+1], "=")
				f.container.Config.Labels[key] = value
			}
		}
		f.container.State.Status = "created"
		if f.lostCreateResponse {
			return "", "response lost", errors.New("connection lost")
		}
		return f.container.ID, "", nil
	case "start":
		f.startCount++
		if f.startError != "" {
			f.container.State.Error = f.startError
			return "", f.startError, errors.New("container start failed")
		}
		f.container.State.Status, f.container.State.Running = "running", true
		return f.container.ID, "", nil
	case "wait":
		if f.waitErr != nil {
			return "", "", f.waitErr
		}
		f.container.State.Status, f.container.State.Running, f.container.State.ExitCode = "exited", false, f.exitCode
		return "0", "", nil
	case "logs":
		return "durable stdout", "durable stderr", nil
	case "stop":
		f.stopCount++
		f.container.State.Status, f.container.State.Running = "exited", false
		return f.container.ID, "", nil
	case "rm":
		f.removeCount++
		f.container = nil
		return "container-uid", "", nil
	default:
		return "", "", errors.New("unexpected docker command")
	}
}

func dockerTestRequest(t *testing.T) TaskExecutionRequest {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	return TaskExecutionRequest{
		Identity:  TaskAttemptIdentity{RunID: "run", TaskID: "task", AttemptID: "attempt"},
		Container: &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{Image: "python:3.11", Command: []string{"python"}, Args: []string{"-c", "print('hello')"}},
	}
}

func TestDockerExecutorDurableLifecycle(t *testing.T) {
	engine := &fakeDockerEngine{lostCreateResponse: true}
	executor := dockerExecutor{command: engine.run}
	require.NoError(t, executor.Start(context.Background()))
	request := dockerTestRequest(t)
	outputPath := filepath.Join(t.TempDir(), "result")
	request.ExecutorInput = &pipelinespec.ExecutorInput{Outputs: &pipelinespec.ExecutorInput_Outputs{
		Parameters: map[string]*pipelinespec.ExecutorInput_OutputParameter{"result": {OutputFile: outputPath}},
		Artifacts: map[string]*pipelinespec.ArtifactList{
			"executor-logs": {Artifacts: []*pipelinespec.RuntimeArtifact{{
				Uri: "file://" + outputPath + ".logs", CustomPath: ptr(outputPath + ".logs"),
				Type: &pipelinespec.ArtifactTypeSchema{Kind: &pipelinespec.ArtifactTypeSchema_SchemaTitle{
					SchemaTitle: "system.Artifact",
				}, SchemaVersion: "0.0.1"},
			}}},
		},
	}}
	request.OutputParameterTypes = map[string]pipelinespec.ParameterType_ParameterTypeEnum{"result": pipelinespec.ParameterType_STRING}
	reference, err := executor.Submit(context.Background(), request)
	require.NoError(t, err)
	secondReference, err := executor.Submit(context.Background(), request)
	require.NoError(t, err)
	assert.Equal(t, reference, secondReference)
	assert.Equal(t, 1, engine.createCount)
	assert.Equal(t, 1, engine.startCount)
	assert.NotContains(t, engine.createArgs, "--rm")
	assert.Contains(t, engine.createArgs, attemptWorkloadName(request.Identity))
	require.NoError(t, os.WriteFile(outputPath, []byte("hello"), 0o600))
	raw, err := json.Marshal(reference)
	require.NoError(t, err)
	reference = ExecutionReference{}
	require.NoError(t, json.Unmarshal(raw, &reference))
	result, err := executor.Wait(context.Background(), reference)
	require.NoError(t, err)
	assert.Equal(t, TaskOutcomeSuccess, result.Outcome)
	assert.Equal(t, "hello", result.OutputParameters["result"].GetStringValue())
	assert.Contains(t, result.StatusMetadata, "customProperties")
	assert.NotNil(t, engine.container, "completed execution must remain recoverable until publication")
	recovery, err := executor.Recover(context.Background(), request.Identity, nil)
	require.NoError(t, err)
	assert.Equal(t, RecoveryCompleted, recovery.State)
	assert.Equal(t, "hello", recovery.Result.OutputParameters["result"].GetStringValue())
	assert.Equal(t, reference, *recovery.Reference)
	require.NoError(t, executor.Cleanup(context.Background(), reference))
	require.NoError(t, executor.Cleanup(context.Background(), reference))
	assert.Equal(t, 1, engine.removeCount)
	assert.Zero(t, engine.stopCount)
	assert.FileExists(t, outputPath, "execution cleanup must not remove outputs awaiting publication")
}

func TestDockerExecutorCancellationAndRecovery(t *testing.T) {
	engine := &fakeDockerEngine{}
	executor := dockerExecutor{command: engine.run}
	request := dockerTestRequest(t)
	reference, err := executor.Submit(context.Background(), request)
	require.NoError(t, err)
	require.ErrorContains(t, executor.Cleanup(context.Background(), reference), "nonterminal")
	assert.Zero(t, engine.removeCount)
	recovery, err := executor.Recover(context.Background(), request.Identity, nil)
	require.NoError(t, err)
	assert.Equal(t, RecoveryExisting, recovery.State)
	engine.waitErr = context.Canceled
	result, err := executor.Wait(context.Background(), reference)
	require.ErrorIs(t, err, context.Canceled)
	assert.Empty(t, result.Outcome)
	assert.Zero(t, engine.stopCount)
	assert.Zero(t, engine.removeCount)
	require.NoError(t, executor.Cancel(context.Background(), reference))
	assert.Equal(t, 1, engine.stopCount)
	assert.Zero(t, engine.removeCount, "cancellation must retain durable logs until publication")
	recovery, err = executor.Recover(context.Background(), request.Identity, &reference)
	require.NoError(t, err)
	assert.Equal(t, RecoveryCompleted, recovery.State)
	require.NoError(t, executor.Cleanup(context.Background(), reference))
	assert.Equal(t, 1, engine.removeCount)
	recovery, err = executor.Recover(context.Background(), request.Identity, &reference)
	require.NoError(t, err)
	assert.Equal(t, RecoveryStopped, recovery.State)
	engine.inspectErr = errors.New("disconnected")
	recovery, err = executor.Recover(context.Background(), request.Identity, nil)
	require.Error(t, err)
	assert.Equal(t, RecoveryUnknown, recovery.State)
	assert.Equal(t, 1, engine.createCount)
}

func TestDockerExecutorRefusesForeignContainer(t *testing.T) {
	engine := &fakeDockerEngine{}
	executor := dockerExecutor{command: engine.run}
	request := dockerTestRequest(t)
	reference, err := executor.Submit(context.Background(), request)
	require.NoError(t, err)
	engine.container.Config.Labels[attemptIDLabel] = "another-attempt"
	_, err = executor.Submit(context.Background(), request)
	require.ErrorContains(t, err, "identity mismatch")
	require.ErrorContains(t, executor.Cancel(context.Background(), reference), "identity mismatch")
	recovery, err := executor.Recover(context.Background(), request.Identity, &reference)
	require.Error(t, err)
	assert.Equal(t, RecoveryUnknown, recovery.State)
	assert.Zero(t, engine.stopCount)
	assert.Zero(t, engine.removeCount)
}

func TestDockerExecutorInvalidTypedOutputIsTerminal(t *testing.T) {
	engine := &fakeDockerEngine{}
	executor := dockerExecutor{command: engine.run}
	request := dockerTestRequest(t)
	output := filepath.Join(t.TempDir(), "boolean-output")
	require.NoError(t, os.WriteFile(output, []byte("not-a-boolean"), 0o600))
	request.ExecutorInput = &pipelinespec.ExecutorInput{Outputs: &pipelinespec.ExecutorInput_Outputs{
		Parameters: map[string]*pipelinespec.ExecutorInput_OutputParameter{"flag": {OutputFile: output}},
	}}
	request.OutputParameterTypes = map[string]pipelinespec.ParameterType_ParameterTypeEnum{"flag": pipelinespec.ParameterType_BOOLEAN}
	reference, err := executor.Submit(context.Background(), request)
	require.NoError(t, err)
	result, err := executor.Wait(context.Background(), reference)
	require.NoError(t, err, "confirmed terminal output-contract violations are not observer errors")
	assert.Equal(t, TaskOutcomeError, result.Outcome)
	require.NotNil(t, result.Failure)
	assert.Equal(t, "output-contract", result.Failure.Class)
	assert.False(t, result.Failure.Transient)
	recovery, err := executor.Recover(context.Background(), request.Identity, nil)
	require.NoError(t, err)
	assert.Equal(t, RecoveryCompleted, recovery.State)
	assert.Equal(t, TaskOutcomeError, recovery.Result.Outcome)
	assert.Equal(t, "output-contract", recovery.Result.Failure.Class)
	assert.Equal(t, 1, engine.createCount)
	assert.Equal(t, 1, engine.startCount)
}

func TestDockerExecutorReportsWorkloadFailure(t *testing.T) {
	engine := &fakeDockerEngine{exitCode: 2}
	executor := dockerExecutor{command: engine.run}
	reference, err := executor.Submit(context.Background(), dockerTestRequest(t))
	require.NoError(t, err)
	result, err := executor.Wait(context.Background(), reference)
	require.NoError(t, err)
	assert.Equal(t, TaskOutcomeFailure, result.Outcome)
	assert.Equal(t, "workload", result.Failure.Class)
	assert.NotNil(t, engine.container)
}

func TestDockerExecutorRefusesRecoveryWithoutDurableOutputMetadata(t *testing.T) {
	engine := &fakeDockerEngine{}
	executor := dockerExecutor{command: engine.run}
	request := dockerTestRequest(t)
	_, err := executor.Submit(context.Background(), request)
	require.NoError(t, err)
	delete(engine.container.Config.Labels, dockerRequestLabel)
	recovery, err := executor.Recover(context.Background(), request.Identity, nil)
	require.ErrorContains(t, err, "durable output metadata")
	assert.Equal(t, RecoveryUnknown, recovery.State)
	assert.Equal(t, 1, engine.createCount)
}

func TestDockerExecutorRecoversDurableContainerStartFailure(t *testing.T) {
	engine := &fakeDockerEngine{startError: "OCI runtime cannot start the container"}
	executor := dockerExecutor{command: engine.run}
	request := dockerTestRequest(t)
	reference, err := executor.Submit(context.Background(), request)
	require.NoError(t, err, "a durable backend start error must retain its reference for terminal classification")
	result, err := executor.Wait(context.Background(), reference)
	require.NoError(t, err)
	assert.Equal(t, TaskOutcomeError, result.Outcome)
	assert.Contains(t, result.Failure.Message, engine.startError)
	recovery, err := executor.Recover(context.Background(), request.Identity, nil)
	require.NoError(t, err)
	assert.Equal(t, RecoveryCompleted, recovery.State)
	assert.Equal(t, TaskOutcomeError, recovery.Result.Outcome)
	assert.Equal(t, 1, engine.startCount, "recovery must not restart a conclusively failed attempt")
	require.NoError(t, executor.Cleanup(context.Background(), reference))
}

func TestDockerExecutorRecoversCreateBeforeStart(t *testing.T) {
	engine := &fakeDockerEngine{}
	executor := dockerExecutor{command: engine.run}
	request := dockerTestRequest(t)
	_, err := executor.Submit(context.Background(), request)
	require.NoError(t, err)
	engine.container.State.Status, engine.container.State.Running = "created", false
	engine.startCount = 0
	recovery, err := executor.Recover(context.Background(), request.Identity, nil)
	require.NoError(t, err)
	assert.Equal(t, RecoveryExisting, recovery.State)
	result, err := executor.Wait(context.Background(), *recovery.Reference)
	require.NoError(t, err)
	assert.Equal(t, TaskOutcomeSuccess, result.Outcome)
	assert.Equal(t, 1, engine.createCount, "recovery must start the existing container, never create another")
	assert.Equal(t, 1, engine.startCount)
}

func TestDockerExecutorRejectsChangedContainerID(t *testing.T) {
	engine := &fakeDockerEngine{}
	executor := dockerExecutor{command: engine.run}
	request := dockerTestRequest(t)
	reference, err := executor.Submit(context.Background(), request)
	require.NoError(t, err)
	engine.container.ID = "replacement"
	_, err = executor.Wait(context.Background(), reference)
	require.ErrorContains(t, err, "ID")
	require.ErrorContains(t, executor.Cancel(context.Background(), reference), "ID")
	require.ErrorContains(t, executor.Cleanup(context.Background(), reference), "ID")
	assert.Zero(t, engine.stopCount)
	assert.Zero(t, engine.removeCount)
}

func TestPrepareDockerTaskStagesBeforeSubmitWithoutResettingReattachedOutputs(t *testing.T) {
	root := t.TempDir()
	inputPath := filepath.Join(root, "input")
	stagedInputPath := filepath.Join(root, "minio", "bucket", "input")
	outputPath := filepath.Join(root, "output")
	store := component.NewMockObjectStoreClient()
	store.DownloadFunc = func(_ context.Context, uri, path, key string) error {
		assert.Equal(t, "minio://bucket/input", uri)
		assert.Equal(t, stagedInputPath, path)
		assert.Equal(t, "input", key)
		return os.WriteFile(path, []byte("downloaded"), 0o600)
	}
	preparation := &TaskPreparationRequest{
		Run:               &model.Run{UUID: "run"},
		Container:         &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{Image: "python"},
		ArtifactLocalRoot: root, ObjectStore: store,
		ExecutorInput: &pipelinespec.ExecutorInput{
			Inputs: &pipelinespec.ExecutorInput_Inputs{Artifacts: map[string]*pipelinespec.ArtifactList{
				"input": {Artifacts: []*pipelinespec.RuntimeArtifact{{Uri: "minio://bucket/input", CustomPath: &inputPath}}},
			}},
			Outputs: &pipelinespec.ExecutorInput_Outputs{
				Parameters: map[string]*pipelinespec.ExecutorInput_OutputParameter{"output": {OutputFile: outputPath}},
				Artifacts: map[string]*pipelinespec.ArtifactList{
					"artifact": {Artifacts: []*pipelinespec.RuntimeArtifact{{Uri: "minio://bucket/output"}}},
				},
			},
		},
	}
	require.NoError(t, os.WriteFile(outputPath, []byte("stale"), 0o600))
	require.NoError(t, PrepareDockerTask(context.Background(), preparation))
	assert.Len(t, store.DownloadCalls, 1)
	assert.NoFileExists(t, outputPath)
	assert.Equal(t, "minio://bucket/input", preparation.ExecutorInput.GetInputs().GetArtifacts()["input"].Artifacts[0].GetUri())
	assert.Equal(t, stagedInputPath, preparation.ExecutorInput.GetInputs().GetArtifacts()["input"].Artifacts[0].GetCustomPath())
	assert.Equal(t, filepath.Join(root, "minio", "bucket", "output"), preparation.ExecutorInput.GetOutputs().GetArtifacts()["artifact"].Artifacts[0].GetCustomPath())
	require.NoError(t, os.WriteFile(outputPath, []byte("durable output"), 0o600))
	request := dockerTestRequest(t)
	request.ExecutorInput, request.ArtifactLocalRoot = preparation.ExecutorInput, root
	engine := &fakeDockerEngine{}
	executor := dockerExecutor{command: engine.run}
	_, err := executor.Submit(context.Background(), request)
	require.NoError(t, err)
	_, err = executor.Submit(context.Background(), request)
	require.NoError(t, err)
	content, err := os.ReadFile(outputPath)
	require.NoError(t, err)
	assert.Equal(t, "durable output", string(content))
	assert.Len(t, store.DownloadCalls, 1)
	assert.Equal(t, 1, engine.createCount)
}
