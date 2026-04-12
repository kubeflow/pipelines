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

	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/cacheutils"
	"github.com/kubeflow/pipelines/backend/src/v2/expression"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"google.golang.org/protobuf/types/known/structpb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	// Resolve container command and args: replace {{$.inputs.parameters['x']}} placeholders
	// with the actual resolved values from the executor input. The launcher will further
	// resolve artifact path placeholders at runtime.
	userCommand, err := resolveContainerArgs(opts.Container.GetCommand(), executorInput)
	if err != nil {
		return execution, fmt.Errorf("failed to resolve container command: %w", err)
	}
	userArgs, err := resolveContainerArgs(opts.Container.GetArgs(), executorInput)
	if err != nil {
		return execution, fmt.Errorf("failed to resolve container args: %w", err)
	}
	execution.UserCommand = userCommand
	execution.UserArgs = userArgs

	// Resolve dynamic secret env vars (secretAsEnv entries with taskOutputParameter names).
	if opts.KubernetesExecutorConfig != nil {
		dynamicEnvVars, dynErr := resolveDynamicSecretEnvVars(ctx, dag, pipeline, opts, mlmd, inputParams)
		if dynErr != nil {
			glog.Warningf("failed to resolve dynamic secret env vars: %v", dynErr)
		} else if len(dynamicEnvVars) > 0 {
			execution.DynamicEnvVars = dynamicEnvVars
		}
	}

	return execution, nil
}

// resolveDynamicSecretEnvVars resolves secretAsEnv entries whose secret names come
// from taskOutputParameter (i.e. runtime-resolved). It reads the actual secret
// value from Kubernetes and returns a map of env var name -> value.
func resolveDynamicSecretEnvVars(
	ctx context.Context,
	dag *metadata.DAG,
	pipeline *metadata.Pipeline,
	opts Options,
	mlmd *metadata.Client,
	inputParams map[string]*structpb.Value,
) (map[string]string, error) {
	if opts.KubernetesExecutorConfig == nil {
		return nil, nil
	}
	result := map[string]string{}
	k8sClient, err := createK8sClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s client: %w", err)
	}
	for _, se := range opts.KubernetesExecutorConfig.GetSecretAsEnv() {
		param := se.GetSecretNameParameter()
		if param == nil {
			continue
		}
		// Only handle taskOutputParameter (runtime-resolved) here; static names are
		// already handled at compile time.
		if _, isTask := param.GetKind().(*pipelinespec.TaskInputsSpec_InputParameterSpec_TaskOutputParameter); !isTask {
			continue
		}
		resolvedVal, err := resolveInputParameter(ctx, dag, pipeline, opts, mlmd, param, inputParams)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve dynamic secret name: %w", err)
		}
		secretName := resolvedVal.GetStringValue()
		if secretName == "" {
			continue
		}
		secret, err := k8sClient.CoreV1().Secrets(opts.Namespace).Get(ctx, secretName, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to get secret %q: %w", secretName, err)
		}
		for _, kToEnv := range se.GetKeyToEnv() {
			if val, ok := secret.Data[kToEnv.GetSecretKey()]; ok {
				result[kToEnv.GetEnvVar()] = string(val)
			}
		}
	}
	if len(result) == 0 {
		return nil, nil
	}
	return result, nil
}
