// Copyright 2025 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package driver

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiV2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/client_manager"
	"github.com/kubeflow/pipelines/backend/src/v2/config"
	"github.com/kubeflow/pipelines/backend/src/v2/driver/common"
	"github.com/kubeflow/pipelines/backend/src/v2/driver/resolver"
	"github.com/kubeflow/pipelines/backend/src/v2/expression"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func Container(ctx context.Context, opts common.Options, clientManager client_manager.ClientManagerInterface) (execution *Execution, driverErr error) {
	defer func() {
		if driverErr != nil {
			driverErr = fmt.Errorf("driver.Container(%s) failed: %w", opts.Info(), driverErr)
		}
	}()
	b, driverErr := json.Marshal(opts)
	if driverErr != nil {
		return nil, driverErr
	}
	glog.V(4).Info("Container opts: ", string(b))

	if clientManager == nil {
		return nil, fmt.Errorf("kfpAPI client is nil")
	}
	if opts.TaskName == "" {
		return nil, fmt.Errorf("task name flag is required for Container")
	}
	if opts.ParentTask == nil {
		return nil, fmt.Errorf("parent task is required for Runtime Task")
	}

	var iterationIndex *int
	if opts.IterationIndex >= 0 {
		idx := opts.IterationIndex
		iterationIndex = &idx
	}

	expr, driverErr := expression.New()
	if driverErr != nil {
		return nil, driverErr
	}

	parentTask, driverErr := clientManager.KFPAPIClient().GetTask(ctx, &apiV2beta1.GetTaskRequest{TaskId: opts.ParentTask.GetTaskId()})
	if driverErr != nil {
		return nil, driverErr
	}
	opts.ParentTask = parentTask

	taskToCreate := &apiV2beta1.PipelineTaskDetail{
		Name:         opts.TaskName,
		DisplayName:  opts.Task.GetTaskInfo().GetName(),
		RunId:        opts.Run.GetRunId(),
		Type:         apiV2beta1.PipelineTaskDetail_RUNTIME,
		State:        apiV2beta1.PipelineTaskDetail_RUNNING,
		ParentTaskId: util.StringPointer(opts.ParentTask.TaskId),
		ScopePath:    opts.ScopePath.StringPath(),
		CreateTime:   timestamppb.Now(),
		Pods: []*apiV2beta1.PipelineTaskDetail_TaskPod{
			{
				Name: opts.PodName,
				Uid:  opts.PodUID,
				Type: apiV2beta1.PipelineTaskDetail_DRIVER,
			},
		},
	}

	// Ensure we capture and propagate any errors.
	defer func() {
		if driverErr != nil {
			taskToCreate.State = apiV2beta1.PipelineTaskDetail_FAILED
			taskToCreate.EndTime = timestamppb.Now()
			taskToCreate.StatusMetadata = &apiV2beta1.PipelineTaskDetail_StatusMetadata{
				Message: driverErr.Error(),
			}
			// We encountered an error in driver before we got the chance to create the task.
			if taskToCreate.TaskId == "" {
				_, err := clientManager.KFPAPIClient().CreateTask(ctx, &apiV2beta1.CreateTaskRequest{Task: taskToCreate})
				if err != nil {
					glog.Errorf("Failed to Create task %s: %v", taskToCreate.Name, err)
				}
			} else {
				_, err := clientManager.KFPAPIClient().UpdateTask(ctx, &apiV2beta1.UpdateTaskRequest{Task: taskToCreate})
				if err != nil {
					glog.Errorf("Failed to update task %s: %v", taskToCreate.Name, err)
				}
			}
		}

		fullView := apiV2beta1.GetRunRequest_FULL
		refreshedRun, getRunErr := clientManager.KFPAPIClient().GetRun(ctx, &apiV2beta1.GetRunRequest{RunId: opts.Run.GetRunId(), View: &fullView})
		if getRunErr != nil {
			glog.Errorf("failed to refresh run: %w", getRunErr)
			return
		}
		opts.Run = refreshedRun
		err := clientManager.KFPAPIClient().UpdateStatuses(ctx, opts.Run, opts.ScopePath.GetPipelineSpecStruct(), taskToCreate)
		if err != nil {
			glog.Errorf("Failed to update statuses: %v", err)
			return
		}
	}()

	// Resolve inputs
	inputs, _, driverErr := resolver.ResolveInputs(ctx, opts)
	if driverErr != nil {
		return nil, driverErr
	}

	// Convert inputs to executor inputs.
	executorInput, driverErr := pipelineTaskInputsToExecutorInputs(inputs)
	if driverErr != nil {
		return nil, fmt.Errorf("failed to convert inputs to executor inputs: %w", driverErr)
	}

	execution = &Execution{ExecutorInput: executorInput}
	condition := opts.Task.GetTriggerPolicy().GetCondition()
	if condition != "" {
		willTrigger, err := expr.Condition(executorInput, condition)
		if err != nil {
			return execution, err
		}
		execution.Condition = &willTrigger
	}

	// When the container image is a dummy image, there is no launcher for this
	// task. This happens when this task is created to implement a
	// Kubernetes-specific configuration, i.e., there is no user container to
	// run. It creates a task in the driver and takes care of.
	// caching, which is usually done in launcher. We also skip creating the
	// podspecpatch in these cases.
	_, isKubernetesPlatformOp := dummyImages[opts.Container.Image]
	if isKubernetesPlatformOp {
		// To be consistent with other artifacts, the driver registers log
		// artifacts to KFP and the launcher publishes them to the object
		// store. This pattern does not work for kubernetesPlatformOps because
		// they have no launcher. There's no point in registering logs that
		// won't be published. Consequently, when we know we're dealing with
		// kubernetesPlatformOps, we set publishLogs to "false". We can amend
		// this when we update the driver to publish logs directly.
		opts.PublishLogs = "false"
	}

	// If this is an iteration runtime task, set the iteration index.
	if iterationIndex != nil {
		taskToCreate.TypeAttributes = &apiV2beta1.PipelineTaskDetail_TypeAttributes{IterationIndex: util.Int64Pointer(int64(*iterationIndex))}
	}

	// Handle Kubernetes-specific tasks such as pvc-creation or pvc-deletion
	if isKubernetesPlatformOp {
		return execution, kubernetesPlatformOps(ctx, clientManager, execution, taskToCreate, &opts)
	}

	var inputParams []*apiV2beta1.PipelineTaskDetail_InputOutputs_IOParameter
	if opts.KubernetesExecutorConfig != nil {
		inputParams = parentTask.GetInputs().GetParameters()
		if driverErr != nil {
			return nil, fmt.Errorf("failed to fetch input parameters from task: %w", driverErr)
		}
	}

	// Generate a fingerprint and check if we have a cache hit.
	var fingerPrint string
	var cachedTask *apiV2beta1.PipelineTaskDetail
	if !opts.CacheDisabled {
		// Generate fingerprint
		// Start by getting the names of the PVCs that need to be mounted.
		var pvcNames []string
		if opts.KubernetesExecutorConfig != nil && opts.KubernetesExecutorConfig.GetPvcMount() != nil {
			_, volumes, err := makeVolumeMountPatch(
				opts, opts.KubernetesExecutorConfig.GetPvcMount(),
				inputParams)
			if err != nil {
				return nil, fmt.Errorf("failed to extract volume mount info while generating fingerprint: %w", err)
			}

			for _, volume := range volumes {
				pvcNames = append(pvcNames, volume.Name)
			}
		}

		if needsWorkspaceMount(execution.ExecutorInput) {
			if opts.RunName == "" {
				return execution, fmt.Errorf("failed to generate fingerprint: run name is required when workspace is used")
			}

			pvcNames = append(pvcNames, GetWorkspacePVCName(opts.RunName))
		}

		fingerPrint, cachedTask, driverErr = getFingerPrintsAndID(ctx, execution, clientManager.KFPAPIClient(), &opts, pvcNames)
		if driverErr != nil {
			return execution, driverErr
		}
		taskToCreate.CacheFingerprint = fingerPrint
	}

	// Use cache and skip pvc creation if all conditions met:
	// (1) Cache is enabled globally
	// (2) Cache is enabled for the task
	// (3) We had a cache hit for this Task
	execution.Cached = util.BoolPointer(false)
	if !opts.CacheDisabled {
		if opts.Task.GetCachingOptions().GetEnableCache() && cachedTask != nil {
			taskToCreate.State = apiV2beta1.PipelineTaskDetail_CACHED
			taskToCreate.Outputs = cachedTask.Outputs
			*execution.Cached = true
			createdTask, createErr := clientManager.KFPAPIClient().CreateTask(ctx, &apiV2beta1.CreateTaskRequest{
				Task: taskToCreate,
			})
			if createErr != nil {
				return execution, fmt.Errorf("failed to update task: %w", createErr)
			}

			// Artifacts are not embedded in tasks like parameters, we need to create separate ArtifactTasks for each output.
			var artifactTasks []*apiV2beta1.ArtifactTask
			for _, cachedOutput := range cachedTask.Outputs.Artifacts {
				for _, artifact := range cachedOutput.Artifacts {
					artifactTasks = append(artifactTasks, &apiV2beta1.ArtifactTask{
						ArtifactId: artifact.GetArtifactId(),
						RunId:      createdTask.RunId,
						TaskId:     createdTask.TaskId,
						Type:       cachedOutput.GetType(),
						Producer:   cachedOutput.GetProducer(),
						Key:        cachedOutput.ArtifactKey,
					})
				}
			}
			_, err := clientManager.KFPAPIClient().CreateArtifactTasks(ctx, &apiV2beta1.CreateArtifactTasksBulkRequest{
				ArtifactTasks: artifactTasks,
			})
			if err != nil {
				return execution, fmt.Errorf("failed to create artifact tasks: %w", err)
			}
			execution.TaskID = createdTask.TaskId
			glog.Infof("Cache hit for task %s", opts.TaskName)
			return execution, nil
		}
	} else {
		glog.Info("Cache disabled globally at the server level.")
	}

	taskToCreate, driverErr = handleInputTaskParametersCreation(inputs.Parameters, taskToCreate)
	if driverErr != nil {
		return execution, driverErr
	}

	if !execution.WillTrigger() {
		taskToCreate.State = apiV2beta1.PipelineTaskDetail_SKIPPED
	}

	glog.Infof("Creating task %s in pod %s", opts.TaskName, opts.Namespace)
	createdTask, driverErr := clientManager.KFPAPIClient().CreateTask(ctx, &apiV2beta1.CreateTaskRequest{Task: taskToCreate})
	if driverErr != nil {
		return execution, driverErr
	}
	execution.TaskID = createdTask.TaskId

	// Create ArtifactTasks for each Artifact Input.
	driverErr = handleInputTaskArtifactsCreation(ctx, opts, inputs.Artifacts, createdTask, clientManager.KFPAPIClient())
	if driverErr != nil {
		return execution, driverErr
	}

	// If this Task is a condition branch and the condition was not met, skip it.
	if !execution.WillTrigger() {
		return execution, nil
	}

	// Determine the pipeline root with the pipeline run context.
	// If a user sets a pipeline root at the runtime config, use that.
	// Otherwise, we use the default pipeline root from the launcher config map.
	// If none is set, we use the hardcoded default.
	pipelineRoot, driverErr := config.GetPipelineRootWithPipelineRunContext(
		ctx,
		opts.PipelineName,
		opts.Namespace,
		clientManager.K8sClient(),
		opts.Run)
	if driverErr != nil {
		return execution, fmt.Errorf("failed to get pipeline root: %w", driverErr)
	}

	// Provision Outputs in ExecutorInput
	if execution.WillTrigger() {
		executorInput.Outputs = provisionOutputs(
			pipelineRoot,
			opts.TaskName,
			opts.Component.GetOutputDefinitions(),
			uuid.NewString(),
			opts.PublishLogs,
		)
	}

	// Generate pod spec patch.
	taskConfig := &TaskConfig{}
	podSpec, driverErr := initPodSpecPatch(
		opts.Container,
		opts.Component,
		executorInput,
		execution.TaskID,
		parentTask.GetTaskId(),
		opts.PipelineName,
		opts.Run.GetRunId(),
		opts.RunName,
		opts.PipelineLogLevel,
		opts.PublishLogs,
		strconv.FormatBool(opts.CacheDisabled),
		taskConfig,
		fingerPrint,
		iterationIndex,
		opts.TaskName,
		opts.MLPipelineTLSEnabled,
		opts.CaCertPath,
		opts.MLPipelineServerAddress,
		opts.MLPipelineServerPort,
	)
	if driverErr != nil {
		return execution, driverErr
	}
	if opts.KubernetesExecutorConfig != nil {
		driverErr = extendPodSpecPatch(ctx, podSpec, opts, inputParams, taskConfig)
		if driverErr != nil {
			return execution, driverErr
		}
	}

	// Handle replacing any dsl.TaskConfig inputs with the taskConfig. This is done here because taskConfig is
	// populated by initPodSpecPatch and extendPodSpecPatch.
	taskConfigInputs := map[string]bool{}
	for inputName := range opts.Component.GetInputDefinitions().GetParameters() {
		compParam := opts.Component.GetInputDefinitions().GetParameters()[inputName]
		if compParam != nil && compParam.GetParameterType() == pipelinespec.ParameterType_TASK_CONFIG {
			taskConfigInputs[inputName] = true
		}
	}

	if len(taskConfigInputs) > 0 {
		taskConfigBytes, err := json.Marshal(taskConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal Kubernetes passthrough info: %w", err)
		}

		taskConfigStruct := &structpb.Struct{}
		err = protojson.Unmarshal(taskConfigBytes, taskConfigStruct)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal Kubernetes passthrough info: %w", err)
		}

		for inputName := range taskConfigInputs {
			executorInput.Inputs.ParameterValues[inputName] = &structpb.Value{
				Kind: &structpb.Value_StructValue{StructValue: taskConfigStruct},
			}
		}

		// Overwrite the --executor_input argument in the podSpec container command with the updated executorInput
		executorInputJSON, err := protojson.Marshal(executorInput)
		if err != nil {
			return execution, fmt.Errorf("JSON marshaling executor input: %w", err)
		}
		for index, container := range podSpec.Containers {
			if container.Name == "main" {
				cmd := container.Command
				for i := 0; i < len(cmd)-1; i++ {
					if cmd[i] == "--executor_input" {
						podSpec.Containers[index].Command[i+1] = string(executorInputJSON)
						break
					}
				}
				break
			}
		}
		execution.ExecutorInput = executorInput
	}

	podSpecPatchBytes, driverErr := json.Marshal(podSpec)
	if driverErr != nil {
		return execution, fmt.Errorf("JSON marshaling pod spec patch: %w", driverErr)
	}
	execution.PodSpecPatch = string(podSpecPatchBytes)
	return execution, nil
}

func pipelineTaskInputsToExecutorInputs(inputMetadata *resolver.InputMetadata) (*pipelinespec.ExecutorInput, error) {
	parameters := make(map[string]*structpb.Value)
	artifacts := make(map[string]*pipelinespec.ArtifactList)
	for _, p := range inputMetadata.Parameters {
		if p.ParameterIO.GetValue() == nil {
			return nil, fmt.Errorf("parameter %s has no value", p.Key)
		}
		if p.ParameterIO.GetType() == apiV2beta1.IOType_ITERATOR_INPUT {
			// first check if p.Key is already present in parameters
			if _, ok := parameters[p.Key]; ok {
				// if present, then append to the existing value
				err := addValueToStructPBList(parameters[p.Key], p.ParameterIO.GetValue())
				if err != nil {
					return nil, fmt.Errorf("failed to append value to existing parameter %s: %w", p.Key, err)
				}
			} else {
				parameters[p.Key] = resolver.ToListValue([]*structpb.Value{
					p.ParameterIO.GetValue(),
				})
			}
		} else {
			parameters[p.Key] = p.ParameterIO.GetValue()
		}
	}
	for _, a := range inputMetadata.Artifacts {
		artifactsList, err := convertArtifactsToArtifactList(a.ArtifactIO.GetArtifacts())
		if err != nil {
			return nil, err
		}
		artifacts[a.Key] = artifactsList
	}
	executorInput := &pipelinespec.ExecutorInput{
		Inputs: &pipelinespec.ExecutorInput_Inputs{
			ParameterValues: parameters,
			Artifacts:       artifacts,
		},
	}
	return executorInput, nil
}

func convertArtifactsToArtifactList(artifacts []*apiV2beta1.Artifact) (*pipelinespec.ArtifactList, error) {
	if len(artifacts) == 0 {
		return &pipelinespec.ArtifactList{}, nil
	}

	// Check if all artifacts are metrics
	allMetrics := true
	for _, artifact := range artifacts {
		if artifact.Type != apiV2beta1.Artifact_Metric {
			allMetrics = false
			break
		}
	}

	// If all are metrics and there is multiple, merge into ONE RuntimeArtifact.
	// This is because the KFP sdk expects a single RuntimeArtifact for metrics.
	// for a given key, and it expects the key/value data to be present in the
	// metadata.
	if allMetrics && len(artifacts) > 1 {
		// Merge all metric artifacts into one RuntimeArtifact with combined metadata
		mergedMetadata := make(map[string]*structpb.Value)
		var firstName string
		var firstURI string
		var firstArtifactID string

		for i, artifact := range artifacts {
			// Use first artifact's common fields
			if i == 0 {
				firstName = artifact.GetName()
				firstURI = artifact.GetUri()
				firstArtifactID = artifact.GetArtifactId()
			}

			// Merge metadata fields: each artifact's metadata contains the metric key/value
			if artifact.GetMetadata() != nil {
				for key, value := range artifact.GetMetadata() {
					mergedMetadata[key] = value
				}
			}

			// Also include the NumberValue in metadata if present
			if artifact.NumberValue != nil {
				// The artifact Name is the metric key (e.g., "accuracy", "precision")
				metricKey := artifact.GetName()
				if metricKey != "" {
					mergedMetadata[metricKey] = structpb.NewNumberValue(*artifact.NumberValue)
				}
			}
		}

		// Create single RuntimeArtifact with merged metadata
		mergedRuntimeArtifact := &pipelinespec.RuntimeArtifact{
			Name:       firstName,
			ArtifactId: firstArtifactID,
			Type: &pipelinespec.ArtifactTypeSchema{
				Kind: &pipelinespec.ArtifactTypeSchema_SchemaTitle{
					SchemaTitle: apiV2beta1.Artifact_Metric.String(),
				},
			},
			Metadata: &structpb.Struct{
				Fields: mergedMetadata,
			},
		}
		if firstURI != "" {
			mergedRuntimeArtifact.Uri = firstURI
		}

		return &pipelinespec.ArtifactList{
			Artifacts: []*pipelinespec.RuntimeArtifact{mergedRuntimeArtifact},
		}, nil
	}

	// Non-metrics or single artifact: convert each artifact to RuntimeArtifact (existing behavior)
	var runtimeArtifacts []*pipelinespec.RuntimeArtifact
	for _, artifact := range artifacts {
		runtimeArtifact, err := convertArtifactToRuntimeArtifact(artifact)
		if err != nil {
			return nil, err
		}
		runtimeArtifacts = append(runtimeArtifacts, runtimeArtifact)
	}
	return &pipelinespec.ArtifactList{
		Artifacts: runtimeArtifacts,
	}, nil
}

func convertArtifactToRuntimeArtifact(
	artifact *apiV2beta1.Artifact,
) (*pipelinespec.RuntimeArtifact, error) {
	if artifact.GetName() == "" && artifact.GetUri() == "" {
		return nil, fmt.Errorf("artifact name or uri cannot be empty")
	}
	runtimeArtifact := &pipelinespec.RuntimeArtifact{
		Name:       artifact.GetName(),
		ArtifactId: artifact.GetArtifactId(),
		Type: &pipelinespec.ArtifactTypeSchema{
			Kind: &pipelinespec.ArtifactTypeSchema_SchemaTitle{
				SchemaTitle: artifact.Type.String(),
			},
		},
	}
	if artifact.GetUri() != "" {
		runtimeArtifact.Uri = artifact.GetUri()
	}
	if artifact.GetMetadata() != nil {
		runtimeArtifact.Metadata = &structpb.Struct{
			Fields: artifact.GetMetadata(),
		}
	}
	return runtimeArtifact, nil
}

func addValueToStructPBList(list *structpb.Value, value *structpb.Value) error {
	switch v := list.GetKind().(type) {
	case *structpb.Value_ListValue:
		v.ListValue.Values = append(v.ListValue.Values, value)
		return nil
	default:
		return fmt.Errorf("value of type %T cannot be appended to", v)
	}
}
