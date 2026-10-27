package poc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
)

type dockerCommand func(context.Context, ...string) (string, string, error)

const dockerRequestLabel = "pipelines.kubeflow.org/execution_data"

type dockerExecutor struct {
	command    dockerCommand
	logCommand dockerLogCommand
}

var (
	_ WorkloadExecutorPlugin = dockerExecutor{}
	_ LostTaskRecoverer      = dockerExecutor{}
)

type dockerExecutionData struct {
	ContainerID          string                                                  `json:"containerId"`
	ContainerName        string                                                  `json:"containerName"`
	ArtifactLocalRoot    string                                                  `json:"artifactLocalRoot,omitempty"`
	ExecutorInput        *pipelinespec.ExecutorInput                             `json:"executorInput,omitempty"`
	OutputParameterTypes map[string]pipelinespec.ParameterType_ParameterTypeEnum `json:"outputParameterTypes,omitempty"`
}

type dockerInspection struct {
	ID     string `json:"Id"`
	Name   string
	Config struct{ Labels map[string]string }
	State  struct {
		Status    string
		Running   bool
		ExitCode  int
		Error     string
		OOMKilled bool
	}
}

func (dockerExecutor) Type() string { return ExecutorDocker }
func (dockerExecutor) Capabilities() ExecutorCapabilities {
	return ExecutorCapabilities{
		RecoverLostTasks: true, SupportsContainers: true,
		SupportedWorkspaceProvisioners: []string{WorkspaceTypeHostPath},
	}
}
func (dockerExecutor) Capacity() ExecutorCapacity { return ExecutorCapacity{} }
func (d dockerExecutor) CheckRequirements(ctx context.Context) error {
	_, stderr, err := d.run(ctx, "info", "--format", "{{.ServerVersion}}")
	if err != nil {
		return fmt.Errorf("docker daemon is unavailable; start Docker: %w: %s", err, stderr)
	}
	return nil
}
func (d dockerExecutor) Start(ctx context.Context) error { return d.CheckRequirements(ctx) }
func (dockerExecutor) Shutdown(context.Context) error    { return nil }

func (d dockerExecutor) run(ctx context.Context, args ...string) (string, string, error) {
	if d.command != nil {
		return d.command(ctx, args...)
	}
	command := exec.CommandContext(ctx, "docker", args...)
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	err := command.Run()
	return stdout.String(), stderr.String(), err
}

func (d dockerExecutor) inspect(ctx context.Context, id string) (*dockerInspection, bool, error) {
	stdout, stderr, err := d.run(ctx, "inspect", "--type", "container", id)
	if err != nil {
		if ctx.Err() != nil {
			return nil, false, ctx.Err()
		}
		if strings.Contains(stderr, "No such container") || strings.Contains(stderr, "No such object") {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("docker inspect failed: %w: %s", err, strings.TrimSpace(stderr))
	}
	var inspections []dockerInspection
	if err := json.Unmarshal([]byte(stdout), &inspections); err != nil {
		return nil, false, err
	}
	if len(inspections) != 1 || inspections[0].ID == "" {
		return nil, false, fmt.Errorf("docker inspect did not return one container with an ID")
	}
	return &inspections[0], true, nil
}

func checkDockerIdentity(container *dockerInspection, identity TaskAttemptIdentity, id string) error {
	if container == nil || (id != "" && container.ID != id) || strings.TrimPrefix(container.Name, "/") != attemptWorkloadName(identity) {
		return fmt.Errorf("docker container ID or name changed; refusing to adopt or stop a replacement")
	}
	return checkAttemptLabels(container.Config.Labels, identity)
}

func dockerExecutionTerminal(container *dockerInspection) bool {
	return container.State.Status == "exited" || container.State.Status == "dead" ||
		(!container.State.Running && container.State.Error != "")
}

func jsonExecutionData(value any) (model.JSONData, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	data := model.JSONData{}
	err = json.Unmarshal(raw, &data)
	return data, err
}

func (d dockerExecutor) Submit(ctx context.Context, request TaskExecutionRequest) (ExecutionReference, error) {
	if err := validateAttempt(request.Identity); err != nil {
		return ExecutionReference{}, err
	}
	if request.Container == nil || request.Container.GetImage() == "" {
		return ExecutionReference{}, fmt.Errorf("docker container image is required")
	}
	name := attemptWorkloadName(request.Identity)
	container, exists, err := d.inspect(ctx, name)
	if err != nil {
		return ExecutionReference{}, err
	}
	if !exists {
		if service := scopedExecutionIO(ctx); service != nil {
			if err := service.Stage(ctx, request); err != nil {
				return ExecutionReference{}, &ExecutionRejectedError{Failure: TaskFailure{
					Class: "infrastructure", Message: fmt.Sprintf("stage Docker task inputs: %v", err),
				}}
			}
		} else if len(request.ExecutorInput.GetInputs().GetArtifacts()) > 0 {
			return ExecutionReference{}, fmt.Errorf("docker input staging requires an attempt-scoped reporting context")
		}
		args, err := dockerCreateArgs(request, name)
		if err != nil {
			return ExecutionReference{}, err
		}
		_, stderr, createErr := d.run(ctx, args...)
		// A concurrent submit or a lost create response may already have created it.
		container, exists, err = d.inspect(ctx, name)
		if err != nil {
			return ExecutionReference{}, err
		}
		if !exists {
			if createErr != nil {
				return ExecutionReference{}, fmt.Errorf("docker create failed: %w: %s", createErr, stderr)
			}
			return ExecutionReference{}, fmt.Errorf("docker created container cannot be found")
		}
	}
	if err := checkDockerIdentity(container, request.Identity, ""); err != nil {
		return ExecutionReference{}, err
	}
	if container.State.Status == "created" && !dockerExecutionTerminal(container) {
		_, stderr, err := d.run(ctx, "start", container.ID)
		if err != nil {
			current, found, inspectErr := d.inspect(ctx, container.ID)
			if inspectErr != nil {
				return ExecutionReference{}, inspectErr
			}
			if !found || checkDockerIdentity(current, request.Identity, container.ID) != nil ||
				(!current.State.Running && !dockerExecutionTerminal(current)) {
				return ExecutionReference{}, fmt.Errorf("docker start failed: %w: %s", err, stderr)
			}
		}
	}
	data, err := jsonExecutionData(dockerExecutionData{
		ContainerID: container.ID, ContainerName: name, ArtifactLocalRoot: request.ArtifactLocalRoot,
		ExecutorInput: request.ExecutorInput, OutputParameterTypes: request.OutputParameterTypes,
	})
	return ExecutionReference{Executor: ExecutorDocker, Version: 1, Identity: request.Identity, Data: data}, err
}

func dockerCreateArgs(request TaskExecutionRequest, name string) ([]string, error) {
	resources, err := dockerResourceArgs(request.Container, request.ExecutorInput)
	if err != nil {
		return nil, err
	}
	environmentArgs, environment, err := dockerEnvironmentArgs(request)
	if err != nil {
		return nil, err
	}
	command, args := containerCommandAndArgs(request.Container)
	var commands []string
	if command != "" {
		commands = []string{command}
	}
	compiledCommand, compiledArgs, err := compileDockerCommandAndArgs(request.ExecutorInput, commands, args)
	if err != nil {
		return nil, err
	}
	if len(compiledCommand) != 0 {
		command = compiledCommand[0]
	}
	args = compiledArgs
	command = expandDockerEnvironment(command, environment)
	for index, value := range args {
		args[index] = expandDockerEnvironment(value, environment)
	}
	dockerArgs := []string{"create", "--name", name}
	durableData, err := json.Marshal(dockerExecutionData{
		ContainerName: name, ArtifactLocalRoot: request.ArtifactLocalRoot,
		ExecutorInput: request.ExecutorInput, OutputParameterTypes: request.OutputParameterTypes,
	})
	if err != nil {
		return nil, err
	}
	dockerArgs = append(dockerArgs, "--label", dockerRequestLabel+"="+string(durableData))
	for _, key := range []string{attemptRunLabel, attemptTaskLabel, attemptIDLabel} {
		dockerArgs = append(dockerArgs, "--label", key+"="+attemptLabels(request.Identity)[key])
	}
	dockerArgs = append(dockerArgs, resources...)
	dockerArgs = append(dockerArgs, environmentArgs...)
	for _, mount := range dockerMounts(&request) {
		dockerArgs = append(dockerArgs, "-v", mount)
	}
	if command != "" {
		dockerArgs = append(dockerArgs, "--entrypoint", command)
	}
	dockerArgs = append(dockerArgs, request.Container.GetImage())
	return append(dockerArgs, args...), nil
}

func parseDockerReference(reference ExecutionReference) (dockerExecutionData, error) {
	data := dockerExecutionData{}
	if reference.Executor != ExecutorDocker || reference.Version != 1 {
		return data, fmt.Errorf("unsupported docker execution reference")
	}
	if err := validateAttempt(reference.Identity); err != nil {
		return data, err
	}
	raw, err := json.Marshal(reference.Data)
	if err == nil {
		err = json.Unmarshal(raw, &data)
	}
	if err != nil {
		return data, err
	}
	if data.ContainerID == "" || data.ContainerName != attemptWorkloadName(reference.Identity) {
		return data, fmt.Errorf("docker execution reference is incomplete or inconsistent")
	}
	return data, nil
}

func (d dockerExecutor) Wait(ctx context.Context, reference ExecutionReference) (TaskExecutionResult, error) {
	if result, resumed, err := resumeExecutorCompletion(ctx, d, reference); resumed {
		return result, err
	}
	data, err := parseDockerReference(reference)
	if err != nil {
		return TaskExecutionResult{}, err
	}
	container, exists, err := d.inspect(ctx, data.ContainerID)
	if err != nil {
		return TaskExecutionResult{}, err
	}
	if !exists {
		return TaskExecutionResult{}, fmt.Errorf("docker execution container is missing")
	}
	if err := checkDockerIdentity(container, reference.Identity, data.ContainerID); err != nil {
		return TaskExecutionResult{}, err
	}
	if container.State.Status == "created" && !dockerExecutionTerminal(container) {
		_, startStderr, startErr := d.run(ctx, "start", data.ContainerID)
		container, exists, err = d.inspect(ctx, data.ContainerID)
		if err != nil {
			return TaskExecutionResult{}, err
		}
		if !exists {
			return TaskExecutionResult{}, fmt.Errorf("recovered docker container disappeared after start")
		}
		if err := checkDockerIdentity(container, reference.Identity, data.ContainerID); err != nil {
			return TaskExecutionResult{}, err
		}
		if startErr != nil && !container.State.Running && !dockerExecutionTerminal(container) {
			return TaskExecutionResult{}, fmt.Errorf("cannot start recovered docker container: %w: %s", startErr, startStderr)
		}
	}
	if container.State.Running || container.State.Status == "restarting" {
		_, stderr, err := d.run(ctx, "wait", data.ContainerID)
		if err != nil {
			return TaskExecutionResult{}, fmt.Errorf("docker wait interrupted: %w: %s", err, stderr)
		}
		container, exists, err = d.inspect(ctx, data.ContainerID)
		if err != nil {
			return TaskExecutionResult{}, err
		}
		if !exists {
			return TaskExecutionResult{}, fmt.Errorf("docker execution container disappeared while waiting")
		}
		if err := checkDockerIdentity(container, reference.Identity, data.ContainerID); err != nil {
			return TaskExecutionResult{}, err
		}
	}
	if !dockerExecutionTerminal(container) {
		return TaskExecutionResult{}, fmt.Errorf("docker execution is not terminal: %s", container.State.Status)
	}
	result, err := d.collectResult(ctx, data, container)
	if err != nil {
		return TaskExecutionResult{}, err
	}
	return completeExecution(ctx, d, reference, result)
}

func (d dockerExecutor) collectResult(ctx context.Context, data dockerExecutionData, container *dockerInspection) (TaskExecutionResult, error) {
	if err := ctx.Err(); err != nil {
		return TaskExecutionResult{}, err
	}
	result := TaskExecutionResult{Outcome: TaskOutcomeSuccess, StatusMetadata: customStatusMetadata("", map[string]any{
		"executor": ExecutorDocker, "containerID": data.ContainerID, "containerName": data.ContainerName,
		"exitCode": container.State.ExitCode,
	})}
	var err error
	if container.State.Error != "" || container.State.OOMKilled || container.State.Status == "dead" {
		result.Outcome, result.Failure = TaskOutcomeError, &TaskFailure{Class: "infrastructure", Message: container.State.Error, Transient: true}
	} else if container.State.ExitCode != 0 {
		result.Outcome, result.Failure = TaskOutcomeFailure, &TaskFailure{Class: "workload", Message: fmt.Sprintf("container exited with code %d", container.State.ExitCode)}
	}
	if result.Outcome == TaskOutcomeSuccess {
		err = withArtifactLocalRootEnv(data.ArtifactLocalRoot, func() error {
			var collectErr error
			result.OutputParameters, collectErr = collectOutputParameters(data.ExecutorInput, data.OutputParameterTypes)
			return collectErr
		})
		if err != nil {
			result.Outcome = TaskOutcomeFailure
			result.Failure = &TaskFailure{Class: "output-contract", Message: err.Error()}
			return result, nil
		}
	}
	return result, err
}

func (d dockerExecutor) Cancel(ctx context.Context, reference ExecutionReference) (CancellationResult, error) {
	if err := d.cancelWorkload(ctx, reference); err != nil {
		return CancellationResult{State: CancellationUnknown}, err
	}
	data, err := parseDockerReference(reference)
	if err != nil {
		return CancellationResult{State: CancellationUnknown}, err
	}
	container, exists, err := d.inspect(ctx, data.ContainerID)
	if err != nil {
		return CancellationResult{State: CancellationUnknown}, err
	}
	neverStarted := exists && container.State.Status == "created"
	if neverStarted {
		reference.Data["canceledBeforeStart"] = true
	}
	return CancellationResult{State: CancellationConfirmed, NeverStarted: neverStarted, Reference: &reference}, nil
}

func (d dockerExecutor) cancelWorkload(ctx context.Context, reference ExecutionReference) error {
	data, err := parseDockerReference(reference)
	if err != nil {
		return err
	}
	container, exists, err := d.inspect(ctx, data.ContainerID)
	if err != nil || !exists {
		return err
	}
	if err := checkDockerIdentity(container, reference.Identity, data.ContainerID); err != nil {
		return err
	}
	if container.State.Running || container.State.Status == "restarting" {
		if _, stderr, err := d.run(ctx, "stop", "--time", "10", data.ContainerID); err != nil {
			current, found, inspectErr := d.inspect(ctx, data.ContainerID)
			if inspectErr != nil {
				return inspectErr
			}
			if !found {
				return nil
			}
			if err := checkDockerIdentity(current, reference.Identity, data.ContainerID); err != nil {
				return err
			}
			if current.State.Running || current.State.Status == "restarting" {
				return fmt.Errorf("docker stop failed: %w: %s", err, stderr)
			}
		}
	}
	container, exists, err = d.inspect(ctx, data.ContainerID)
	if err != nil || !exists {
		return err
	}
	if err := checkDockerIdentity(container, reference.Identity, data.ContainerID); err != nil {
		return err
	}
	if container.State.Running || container.State.Status == "restarting" {
		return fmt.Errorf("docker container has not stopped")
	}
	return nil
}

func (d dockerExecutor) removeContainer(ctx context.Context, id string) error {
	_, stderr, removeErr := d.run(ctx, "rm", id)
	_, exists, err := d.inspect(ctx, id)
	if err != nil {
		return err
	}
	if exists {
		if removeErr != nil {
			return fmt.Errorf("docker remove failed: %w: %s", removeErr, stderr)
		}
		return fmt.Errorf("docker container removal is not confirmed")
	}
	return nil
}

func (d dockerExecutor) Cleanup(ctx context.Context, reference ExecutionReference) error {
	if err := retainCleanupLogs(ctx, d, reference); err != nil {
		return err
	}
	data, err := parseDockerReference(reference)
	if err != nil {
		return err
	}
	container, exists, err := d.inspect(ctx, data.ContainerID)
	if err != nil || !exists {
		return err
	}
	if err := checkDockerIdentity(container, reference.Identity, data.ContainerID); err != nil {
		return err
	}
	canceledBeforeStart, _ := reference.Data["canceledBeforeStart"].(bool)
	if container.State.Running || (!dockerExecutionTerminal(container) && (!canceledBeforeStart || container.State.Status != "created")) {
		return fmt.Errorf("docker cleanup refuses to remove a nonterminal execution")
	}
	return d.removeContainer(ctx, data.ContainerID)
}

func (d dockerExecutor) Recover(ctx context.Context, identity TaskAttemptIdentity, reference *ExecutionReference) (RecoveryResult, error) {
	unknown := RecoveryResult{State: RecoveryUnknown}
	if err := validateAttempt(identity); err != nil {
		return unknown, err
	}
	if reference != nil && reference.Identity == identity {
		if result, resumed, err := resumeExecutorCompletion(ctx, d, *reference); resumed {
			if err != nil {
				return unknown, err
			}
			return RecoveryResult{State: RecoveryCompleted, Reference: reference, Result: &result}, nil
		}
	}
	name := attemptWorkloadName(identity)
	id := name
	if reference != nil {
		if reference.Identity != identity {
			return unknown, fmt.Errorf("recovery identity differs from execution reference")
		}
		data, err := parseDockerReference(*reference)
		if err != nil {
			return unknown, err
		}
		id = data.ContainerID
	}
	container, exists, err := d.inspect(ctx, id)
	if err != nil {
		return unknown, err
	}
	if !exists {
		return RecoveryResult{State: RecoveryStopped}, nil
	}
	expectedID := ""
	if reference != nil {
		expectedID = id
	}
	if err := checkDockerIdentity(container, identity, expectedID); err != nil {
		return unknown, err
	}
	found := reference
	if found == nil {
		durableData := dockerExecutionData{}
		if err := json.Unmarshal([]byte(container.Config.Labels[dockerRequestLabel]), &durableData); err != nil {
			return unknown, fmt.Errorf("docker container is missing durable output metadata: %w", err)
		}
		durableData.ContainerID, durableData.ContainerName = container.ID, name
		data, err := jsonExecutionData(durableData)
		if err != nil {
			return unknown, err
		}
		found = &ExecutionReference{Executor: ExecutorDocker, Version: 1, Identity: identity, Data: data}
	}
	if dockerExecutionTerminal(container) {
		if scopedExecutionIO(ctx) == nil {
			data, err := parseDockerReference(*found)
			if err != nil {
				return unknown, err
			}
			if len(data.ExecutorInput.GetOutputs().GetArtifacts()) > 0 {
				return RecoveryResult{State: RecoveryExisting, Reference: found}, nil
			}
		}
		result, err := d.Wait(ctx, *found)
		if err != nil {
			return unknown, err
		}
		return RecoveryResult{State: RecoveryCompleted, Reference: found, Result: &result}, nil
	}
	switch container.State.Status {
	case "created", "running", "restarting", "paused":
		return RecoveryResult{State: RecoveryExisting, Reference: found}, nil
	default:
		return unknown, fmt.Errorf("cannot establish docker container state: %s", container.State.Status)
	}
}
