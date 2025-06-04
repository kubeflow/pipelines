// Copyright 2021-2023 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package compiler is the backend compiler package for Kubeflow Pipelines v2.
//
// KFP pipeline DSL in python are first compiled by KFP SDK (the frontend compiler)
// to pipeline spec in JSON format. KFP SDK / frontend compiler is not part of
// this package.
// Then, the backend compiler (this package) compiles pipeline spec into argo
// workflow spec, so that it can be run.
package compiler

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/golang/protobuf/jsonpb"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"google.golang.org/protobuf/types/known/structpb"
)

// Visitor interface is called when each component is visited.
// The specific method called depends on the component's type.
type Visitor interface {
	Container(name string, component *pipelinespec.ComponentSpec, container *pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec) error
	Importer(name string, component *pipelinespec.ComponentSpec, importer *pipelinespec.PipelineDeploymentConfig_ImporterSpec) error
	Resolver(name string, component *pipelinespec.ComponentSpec, resolver *pipelinespec.PipelineDeploymentConfig_ResolverSpec) error
	DAG(name string, component *pipelinespec.ComponentSpec, dag *pipelinespec.DagSpec) error
	AddKubernetesSpec(name string, kubernetesSpec *structpb.Struct) error
}

const (
	RootComponentName = "root"
)

// Accept a pipeline spec and a visitor, iterate through components and call
// corresponding visitor methods for each component.
//
// Iteration rules:
// * Components are visited in "bottom-up" order -- leaf container components are
// visited first, then DAG components. When a DAG component is visited, it's
// guaranteed that all the components used in it have already been visited.
// * Each component is visited exactly once.
func Accept(job *pipelinespec.PipelineJob, kubernetesSpec *pipelinespec.SinglePlatformSpec, v Visitor) error {
	if job == nil {
		return nil
	}
	// TODO(Bobgy): reserve root as a keyword that cannot be user component names
	spec, err := GetPipelineSpec(job)
	if err != nil {
		return err
	}
	deploy, err := GetDeploymentConfig(spec)
	if err != nil {
		return err
	}
	state := &pipelineDFS{
		spec:           spec,
		deploy:         deploy,
		kubernetesSpec: kubernetesSpec,
		visitor:        v,
		visited:        make(map[string]bool),
	}
	return state.dfs(RootComponentName, spec.GetRoot(), nil)
}

type pipelineDFS struct {
	spec           *pipelinespec.PipelineSpec
	deploy         *pipelinespec.PipelineDeploymentConfig
	kubernetesSpec *pipelinespec.SinglePlatformSpec
	visitor        Visitor
	// Records which DAG components are visited, map key is component name.
	visited map[string]bool
}

func (state *pipelineDFS) dfs(name string, component *pipelinespec.ComponentSpec, componentTask *pipelinespec.PipelineTaskSpec) error {
	// each component is only visited once
	// TODO(Bobgy): return an error when circular reference detected
	if state.visited[name] {
		return nil
	}
	state.visited[name] = true
	if component == nil {
		return nil
	}
	if state == nil {
		return fmt.Errorf("dfs: unexpected value state=nil")
	}
	componentError := func(err error) error {
		return fmt.Errorf("error processing component name=%q: %w", name, err)
	}
	executorLabel := component.GetExecutorLabel()
	if executorLabel != "" {
		executor, ok := state.deploy.GetExecutors()[executorLabel]
		if !ok {
			return componentError(fmt.Errorf("executor(label=%q) not found in deployment config", executorLabel))
		}

		// Add kubernetes spec to annotation
		if state.kubernetesSpec != nil && state.kubernetesSpec.DeploymentSpec != nil {
			kubernetesExecSpec, ok := state.kubernetesSpec.DeploymentSpec.Executors[executorLabel]
			if ok {
				err := state.visitor.AddKubernetesSpec(name, kubernetesExecSpec)
				if err != nil {
					return componentError(fmt.Errorf("failed to add Kubernetes spec for %s: %w", name, err))
				}
			}
		}

		container := executor.GetContainer()
		if container != nil {
			return state.visitor.Container(name, component, container)
		}
		importer := executor.GetImporter()
		if importer != nil {
			return state.visitor.Importer(name, component, importer)
		}

		return componentError(fmt.Errorf("executor(label=%q): non-container and non-importer executor not implemented", executorLabel))
	}
	dag := component.GetDag()
	if dag == nil { // impl can only be executor or dag
		return componentError(fmt.Errorf("unknown component implementation: %s", component))
	}
	tasks := dag.GetTasks()
	// Iterate through tasks in deterministic order to facilitate testing.
	// Note, order doesn't affect compiler with real effect right now.
	// In the future, we may consider using topology sort when building local
	// executor that runs on pipeline spec directly.
	keys := make([]string, 0, len(tasks))
	for key := range tasks {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		task, ok := tasks[key]
		if !ok {
			return componentError(fmt.Errorf("this is a bug: cannot find key %q in tasks", key))
		}
		refName := task.GetComponentRef().GetName()
		if refName == "" {
			return componentError(fmt.Errorf("component ref name is empty for task name=%q", task.GetTaskInfo().GetName()))
		}
		subComponent, ok := state.spec.Components[refName]
		if !ok {
			return componentError(fmt.Errorf("cannot find component ref name=%q", refName))
		}
		if task.GetRetryPolicy() == nil && componentTask != nil {
			task.RetryPolicy = componentTask.GetRetryPolicy()
		}
		err := state.dfs(refName, subComponent, task)
		if err != nil {
			return err
		}
	}
	// process tasks before DAG component, so that all sub-tasks are already
	// ready by the time the DAG component is visited.
	return state.visitor.DAG(name, component, dag)
}

func GetDeploymentConfig(spec *pipelinespec.PipelineSpec) (*pipelinespec.PipelineDeploymentConfig, error) {
	marshaler := jsonpb.Marshaler{}
	buffer := new(bytes.Buffer)
	if err := marshaler.Marshal(buffer, spec.GetDeploymentSpec()); err != nil {
		return nil, err
	}
	deploymentConfig := &pipelinespec.PipelineDeploymentConfig{}
	// Allow unknown '@type' field in the json message.
	unmarshaler := jsonpb.Unmarshaler{AllowUnknownFields: true}
	if err := unmarshaler.Unmarshal(buffer, deploymentConfig); err != nil {
		return nil, err
	}
	return deploymentConfig, nil
}

func GetPipelineSpec(job *pipelinespec.PipelineJob) (*pipelinespec.PipelineSpec, error) {
	// TODO(Bobgy): can we avoid this marshal to string step?
	marshaler := jsonpb.Marshaler{}
	json, err := marshaler.MarshalToString(job.GetPipelineSpec())
	if err != nil {
		return nil, fmt.Errorf("failed marshal pipeline spec to json: %w", err)
	}
	spec := &pipelinespec.PipelineSpec{}
	if err := jsonpb.UnmarshalString(json, spec); err != nil {
		return nil, fmt.Errorf("failed to parse pipeline spec: %v", err)
	}
	return spec, nil
}
