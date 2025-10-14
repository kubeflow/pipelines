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
	"github.com/kubeflow/pipelines/backend/src/v2/cacheutils"
	"github.com/kubeflow/pipelines/backend/src/v2/expression"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

func validateContainer(opts Options) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("invalid container driver args: %w", err)
		}
	}()
	if opts.Container == nil {
		return fmt.Errorf("container spec is required")
	}
	return validateNonRoot(opts)
}

func Container(ctx context.Context, opts Options, mlmd *metadata.Client, cacheClient cacheutils.Client) (execution *Execution, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("driver.Container(%s) failed: %w", opts.info(), err)
		}
	}()
	b, _ := json.Marshal(opts)
	glog.V(4).Info("Container opts: ", string(b))
	err = validateContainer(opts)
	if err != nil {
		return nil, err
	}
	var iterationIndex *int
	if opts.IterationIndex >= 0 {
		index := opts.IterationIndex
		iterationIndex = &index
	}
	// TODO(Bobgy): there's no need to pass any parameters, because pipeline
	// and pipeline run context have been created by root DAG driver.
	pipeline, err := mlmd.GetPipeline(ctx, opts.PipelineName, opts.RunID, "", "", "", "")
	if err != nil {
		return nil, err
	}
	dag, err := mlmd.GetDAG(ctx, opts.DAGExecutionID)
	if err != nil {
		return nil, err
	}
	glog.Infof("parent DAG: %+v", dag.Execution)
	expr, err := expression.New()
	if err != nil {
		return nil, err
	}
	inputs, err := resolveInputs(ctx, dag, iterationIndex, pipeline, opts, mlmd, expr)
	if err != nil {
		return nil, err
	}

	executorInput := &pipelinespec.ExecutorInput{
		Inputs: inputs,
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
	// run. It publishes execution details to mlmd in driver and takes care of
	// caching, which are usually done in launcher. We also skip creating the
	// podspecpatch in these cases.
	_, isKubernetesPlatformOp := dummyImages[opts.Container.Image]
	if isKubernetesPlatformOp {
		// To be consistent with other artifacts, the driver registers log
		// artifacts to MLMD and the launcher publishes them to the object
		// store. This pattern does not work for kubernetesPlatformOps because
		// they have no launcher. There's no point in registering logs that
		// won't be published. Consequently, when we know we're dealing with
		// kubernetesPlatformOps, we set publishLogs to "false". We can amend
		// this when we update the driver to publish logs directly.
		opts.PublishLogs = "false"
	}

	if execution.WillTrigger() {
		executorInput.Outputs = provisionOutputs(
			pipeline.GetPipelineRoot(),
			opts.TaskName,
			opts.Component.GetOutputDefinitions(),
			uuid.NewString(),
			opts.PublishLogs,
		)
	}

	ecfg, err := metadata.GenerateExecutionConfig(executorInput)
	if err != nil {
		return execution, err
	}
	ecfg.TaskName = opts.TaskName
	ecfg.DisplayName = opts.Task.GetTaskInfo().GetName()
	ecfg.ExecutionType = metadata.ContainerExecutionTypeName
	ecfg.ParentDagID = dag.Execution.GetID()
	ecfg.IterationIndex = iterationIndex
	ecfg.NotTriggered = !execution.WillTrigger()

	if isKubernetesPlatformOp {
		return execution, kubernetesPlatformOps(ctx, mlmd, cacheClient, execution, ecfg, &opts)
	}

	var inputParams map[string]*structpb.Value

	if opts.KubernetesExecutorConfig != nil {
		inputParams, _, err = dag.Execution.GetParameters()
		if err != nil {
			return nil, fmt.Errorf("failed to fetch input parameters from execution: %w", err)
		}
	}

	if !opts.CacheDisabled {
		// Generate fingerprint and MLMD ID for cache
		// Start by getting the names of the PVCs that need to be mounted.
		pvcNames := []string{}
		if opts.KubernetesExecutorConfig != nil && opts.KubernetesExecutorConfig.GetPvcMount() != nil {
			_, volumes, err := makeVolumeMountPatch(ctx, opts, opts.KubernetesExecutorConfig.GetPvcMount(),
				dag, pipeline, mlmd, inputParams)
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

		fingerPrint, cachedMLMDExecutionID, err := getFingerPrintsAndID(execution, &opts, cacheClient, pvcNames)
		if err != nil {
			return execution, err
		}
		ecfg.CachedMLMDExecutionID = cachedMLMDExecutionID
		ecfg.FingerPrint = fingerPrint
	}

	// TODO(Bobgy): change execution state to pending, because this is driver, execution hasn't started.
	createdExecution, err := mlmd.CreateExecution(ctx, pipeline, ecfg)
	if err != nil {
		return execution, err
	}
	glog.Infof("Created execution: %s", createdExecution)
	execution.ID = createdExecution.GetID()
	if !execution.WillTrigger() {
		return execution, nil
	}

	// Use cache and skip launcher if all contions met:
	// (1) Cache is enabled globally
	// (2) Cache is enabled for the task
	// (3) CachedMLMDExecutionID is non-empty, which means a cache entry exists
	cached := false
	execution.Cached = &cached
	if !opts.CacheDisabled {
		if opts.Task.GetCachingOptions().GetEnableCache() && ecfg.CachedMLMDExecutionID != "" {
			executorOutput, outputArtifacts, err := reuseCachedOutputs(ctx, execution.ExecutorInput, mlmd, ecfg.CachedMLMDExecutionID)
			if err != nil {
				return execution, err
			}
			// TODO(Bobgy): upload output artifacts.
			// TODO(Bobgy): when adding artifacts, we will need execution.pipeline to be non-nil, because we need
			// to publish output artifacts to the context too.
			if err := mlmd.PublishExecution(ctx, createdExecution, executorOutput.GetParameterValues(), outputArtifacts, pb.Execution_CACHED); err != nil {
				return execution, fmt.Errorf("failed to publish cached execution: %w", err)
			}
			glog.Infof("Use cache for task %s", opts.Task.GetTaskInfo().GetName())
			*execution.Cached = true
			return execution, nil
		}
	} else {
		glog.Info("Cache disabled globally at the server level.")
	}

	taskConfig := &TaskConfig{}

	podSpec, err := initPodSpecPatch(
		opts.Container,
		opts.Component,
		executorInput,
		execution.ID,
		opts.PipelineName,
		opts.RunID,
		opts.RunName,
		opts.PipelineLogLevel,
		opts.PublishLogs,
		strconv.FormatBool(opts.CacheDisabled),
		opts.MLPipelineTLSEnabled,
		opts.MLMDServerAddress,
		opts.MLMDServerPort,
		opts.MLMDTLSEnabled,
		opts.CaCertPath,
		taskConfig,
	)
	if err != nil {
		return execution, err
	}
	if opts.KubernetesExecutorConfig != nil {
		err = extendPodSpecPatch(ctx, podSpec, opts, dag, pipeline, mlmd, inputParams, taskConfig)
		if err != nil {
			return execution, err
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

		ecfg.InputParameters = executorInput.Inputs.ParameterValues

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

	podSpecPatchBytes, err := json.Marshal(podSpec)
	if err != nil {
		return execution, fmt.Errorf("JSON marshaling pod spec patch: %w", err)
	}
	execution.PodSpecPatch = string(podSpecPatchBytes)
	return execution, nil
}
