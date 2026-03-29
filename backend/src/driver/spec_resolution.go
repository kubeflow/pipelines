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
package main

import (
	"context"
	"fmt"
	"strconv"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient/kfpapi"
	"github.com/kubeflow/pipelines/backend/src/v2/compiler"
)

// parseOptionalBoolFlag parses an optional boolean request argument.
// Empty value means unset (nil). Invalid values return an error.
func parseOptionalBoolFlag(flagName, value string) (*bool, error) {
	if value == "" {
		return nil, nil
	}
	v, err := strconv.ParseBool(value)
	if err != nil {
		return nil, fmt.Errorf("invalid %s value %q: %w", flagName, value, err)
	}
	return &v, nil
}

func resolveNamespace(explicitNamespace string) (string, error) {
	if explicitNamespace == "" {
		return "", fmt.Errorf("argument --namespace must be specified")
	}
	return explicitNamespace, nil
}

// buildScopePath builds a ScopePath from the run, parentTask and taskName.
func buildScopePath(
	ctx context.Context,
	run *go_client.Run,
	parentTask *go_client.PipelineTask,
	taskName string,
	driverType string,
	kfpAPI kfpapi.API) (*util.ScopePath, error) {
	pipelineSpecStruct, err := kfpAPI.FetchPipelineSpecFromRun(ctx, run)
	if err != nil {
		return nil, err
	}
	var scopePath util.ScopePath
	if driverType == RootDag {
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
			return nil, fmt.Errorf("task name must be specified for non-root drivers")
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

func resolveDriverSpecs(
	scopePath *util.ScopePath,
	driverType string,
) (*pipelinespec.ComponentSpec, *pipelinespec.PipelineTaskSpec, *pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec, error) {
	return resolveDriverSpecsFromScopePath(scopePath, driverType)
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

func resolveDriverSpecsFromScopePath(
	scopePath *util.ScopePath,
	driverType string,
) (*pipelinespec.ComponentSpec, *pipelinespec.PipelineTaskSpec, *pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec, error) {
	if scopePath == nil || scopePath.GetLast() == nil {
		return nil, nil, nil, unavailableSpec("scope path is empty")
	}

	componentSpec := scopePath.GetLast().GetComponentSpec()
	if componentSpec == nil {
		return nil, nil, nil, unavailableSpec("component spec not found")
	}

	var taskSpec *pipelinespec.PipelineTaskSpec
	if driverType != RootDag {
		taskSpec = scopePath.GetLast().GetTaskSpec()
		if taskSpec == nil {
			return nil, nil, nil, unavailableSpec("task spec not found")
		}
	}

	if err := validateDriverComponentKinds(driverType, componentSpec); err != nil {
		return nil, nil, nil, err
	}

	var containerSpec *pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec
	if driverType == CONTAINER {
		var err error
		containerSpec, err = loadContainerSpec(componentSpec, scopePath.GetPipelineSpec())
		if err != nil {
			return nil, nil, nil, err
		}
	}

	return componentSpec, taskSpec, containerSpec, nil
}

func validateDriverComponentKinds(driverType string, componentSpec *pipelinespec.ComponentSpec) error {
	switch driverType {
	case RootDag:
		if componentSpec.GetDag() == nil {
			return fmt.Errorf("root driver requires a DAG root component")
		}
	case DAG:
		if componentSpec.GetDag() == nil {
			return fmt.Errorf("dag driver requires a DAG component")
		}
	case CONTAINER:
		if componentSpec.GetExecutorLabel() == "" {
			return fmt.Errorf("container driver requires an executor-label component")
		}
	default:
		return fmt.Errorf("unknown driver type %q", driverType)
	}
	return nil
}

func loadContainerSpec(
	componentSpec *pipelinespec.ComponentSpec,
	pipelineSpec *pipelinespec.PipelineSpec,
) (*pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec, error) {
	if componentSpec == nil {
		return nil, unavailableSpec("component spec is nil")
	}
	if pipelineSpec == nil {
		return nil, unavailableSpec("pipeline spec is nil")
	}

	executorLabel := componentSpec.GetExecutorLabel()
	if executorLabel == "" {
		return nil, fmt.Errorf("component executor label is empty")
	}

	if pipelineSpec.GetDeploymentSpec() == nil {
		return nil, unavailableSpec("pipeline deployment spec is missing")
	}

	deploymentConfig, err := compiler.GetDeploymentConfig(pipelineSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal deployment spec: %w", err)
	}

	executor, ok := deploymentConfig.GetExecutors()[executorLabel]
	if !ok || executor == nil {
		return nil, unavailableSpec(fmt.Sprintf("container executor %q not found in deployment spec", executorLabel))
	}
	containerSpec := executor.GetContainer()
	if containerSpec == nil {
		return nil, fmt.Errorf("executor %q does not contain a container spec", executorLabel)
	}
	return containerSpec, nil
}
