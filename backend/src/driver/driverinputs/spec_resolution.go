// Copyright 2021-2023 The Kubeflow Authors
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

// Package driverinputs resolves what a driver invocation needs from the run:
// component, task and container specs, the scope path, and pipeline job times.
package driverinputs

import (
	"context"
	"fmt"
	"strconv"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/driver/driverapi"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient/kfpapi"
	"github.com/kubeflow/pipelines/backend/src/v2/compiler"
	"github.com/kubeflow/pipelines/kubernetes_platform/go/kubernetesplatform"
)

// ParseOptionalBool parses an optional boolean request argument.
// Empty value means unset (nil). Invalid values return an error.
func ParseOptionalBool(flagName, value string) (*bool, error) {
	if value == "" {
		return nil, nil
	}
	v, err := strconv.ParseBool(value)
	if err != nil {
		return nil, fmt.Errorf("invalid %s value %q; use true or false: %w", flagName, value, err)
	}
	return &v, nil
}

// BuildScopePath builds a ScopePath from the run, parentTask and taskName.
func BuildScopePath(
	ctx context.Context,
	run *go_client.Run,
	parentTask *go_client.PipelineTask,
	taskName string,
	driverType driverapi.DriverType,
	kfpAPI kfpapi.API) (*util.ScopePath, error) {
	pipelineSpecStruct, err := kfpAPI.FetchPipelineSpecFromRun(ctx, run)
	if err != nil {
		return nil, err
	}
	var scopePath util.ScopePath
	if driverType == driverapi.RootDAGDriver {
		scopePath, err = util.NewScopePathFromStruct(pipelineSpecStruct)
		if err != nil {
			return nil, err
		}
		err = scopePath.Push("root")
		if err != nil {
			return nil, err
		}
	} else {
		if taskName == "" {
			return nil, fmt.Errorf("task name is empty for a %s driver; set task_name to the task being driven", driverType)
		}
		scopePath, err = util.ScopePathFromStringPathWithNewTask(
			pipelineSpecStruct,
			parentTask.GetScopePath(),
			taskName,
		)
		if err != nil {
			return nil, err
		}
	}
	return &scopePath, nil
}

type specSourceUnavailableError struct {
	message string
}

func (e specSourceUnavailableError) Error() string {
	return e.message
}

func unavailableSpec(message string) error {
	return specSourceUnavailableError{message: message}
}

// ResolveSpecsFromScopePath returns the component, task and container specs the
// driver type needs; the task spec is nil for the root DAG.
func ResolveSpecsFromScopePath(
	scopePath *util.ScopePath,
	driverType driverapi.DriverType,
) (*pipelinespec.ComponentSpec, *pipelinespec.PipelineTaskSpec, *pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec, error) {
	if scopePath == nil || scopePath.GetLast() == nil {
		return nil, nil, nil, unavailableSpec("scope path is empty; check that the parent task's scope path is set")
	}

	componentSpec := scopePath.GetLast().GetComponentSpec()
	if componentSpec == nil {
		return nil, nil, nil, unavailableSpec("component spec not found in the pipeline spec; check the task's componentRef")
	}

	var taskSpec *pipelinespec.PipelineTaskSpec
	if driverType != driverapi.RootDAGDriver {
		taskSpec = scopePath.GetLast().GetTaskSpec()
		if taskSpec == nil {
			return nil, nil, nil, unavailableSpec("task spec not found in the parent DAG; check the task name")
		}
	}

	if err := validateDriverComponentKinds(driverType, componentSpec); err != nil {
		return nil, nil, nil, err
	}

	var containerSpec *pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec
	if driverType == driverapi.ContainerDriver {
		var err error
		containerSpec, err = loadContainerSpec(componentSpec, scopePath.GetPipelineSpec())
		if err != nil {
			return nil, nil, nil, err
		}
	}

	return componentSpec, taskSpec, containerSpec, nil
}

func validateDriverComponentKinds(driverType driverapi.DriverType, componentSpec *pipelinespec.ComponentSpec) error {
	if err := driverType.Validate(); err != nil {
		return err
	}
	if driverType.IsDAG() && componentSpec.GetDag() == nil {
		return fmt.Errorf("%s driver requires a DAG component; check that the task's component is a DAG", driverType)
	}
	if !driverType.IsDAG() && componentSpec.GetExecutorLabel() == "" {
		return fmt.Errorf("%s driver requires a component with an executor label; check that the task's component is a container", driverType)
	}
	return nil
}

func loadContainerSpec(
	componentSpec *pipelinespec.ComponentSpec,
	pipelineSpec *pipelinespec.PipelineSpec,
) (*pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec, error) {
	if componentSpec == nil {
		return nil, unavailableSpec("component spec is nil; check the task's componentRef")
	}
	if pipelineSpec == nil {
		return nil, unavailableSpec("pipeline spec is nil; check that the run has a pipeline spec")
	}

	executorLabel := componentSpec.GetExecutorLabel()
	if executorLabel == "" {
		return nil, fmt.Errorf("component executor label is empty; check the component's executorLabel")
	}

	if pipelineSpec.GetDeploymentSpec() == nil {
		return nil, unavailableSpec("pipeline deployment spec is missing; recompile the pipeline")
	}

	deploymentConfig, err := compiler.GetDeploymentConfig(pipelineSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal deployment spec; recompile the pipeline: %w", err)
	}

	executor, ok := deploymentConfig.GetExecutors()[executorLabel]
	if !ok || executor == nil {
		return nil, unavailableSpec(fmt.Sprintf("container executor %q not found in deployment spec; check the component's executorLabel", executorLabel))
	}
	containerSpec := executor.GetContainer()
	if containerSpec == nil {
		return nil, fmt.Errorf("executor %q does not contain a container spec; check the deployment spec", executorLabel)
	}
	return containerSpec, nil
}

// ParseKubernetesExecutorConfig decodes the optional kubernetes_config argument.
func ParseKubernetesExecutorConfig(k8sExecConfigJSON *string) (*kubernetesplatform.KubernetesExecutorConfig, error) {
	var k8sExecCfg *kubernetesplatform.KubernetesExecutorConfig
	if *k8sExecConfigJSON != "" {
		k8sExecCfg = &kubernetesplatform.KubernetesExecutorConfig{}
		if err := util.UnmarshalString(*k8sExecConfigJSON, k8sExecCfg); err != nil {
			return nil, kubernetesConfigParseError{cause: err}
		}
	}
	return k8sExecCfg, nil
}

// Keep configuration contents out of HTTP errors and logs while retaining the
// decoding cause for callers that inspect the error chain.
type kubernetesConfigParseError struct{ cause error }

func (e kubernetesConfigParseError) Error() string {
	return "failed to unmarshal Kubernetes config; provide valid KubernetesExecutorConfig JSON"
}
func (e kubernetesConfigParseError) Unwrap() error { return e.cause }
