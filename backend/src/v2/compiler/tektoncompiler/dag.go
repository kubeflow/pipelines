// Copyright 2023 The Kubeflow Authors
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

package tektoncompiler

import (
	"encoding/json"
	"fmt"
	"strings"

	pipelineloopapi "github.com/kubeflow/kfp-tekton/tekton-catalog/pipeline-loops/pkg/apis/pipelineloop/v1alpha1"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/compiler"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func (c *pipelinerunCompiler) LoopDAG(taskName, compRef string, task *pipelinespec.PipelineTaskSpec, componentSpec *pipelinespec.ComponentSpec, dagSpec *pipelinespec.DagSpec) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("compiling DAG %q: %w", taskName, err)
		}
	}()

	err = c.saveComponentSpec(compRef, componentSpec)
	if err != nil {
		return err
	}

	// create a PipelineLoop and push to stack
	c.PushLoop(c.createPipelineLoop())

	newTaskName := taskName + "-loop"
	if err := c.addDagTask(newTaskName, compRef, task, true); err != nil {
		return err
	}

	// add the Loop DAG into DAG Stack
	c.PushDagStack(newTaskName)
	return nil
}

func (c *pipelinerunCompiler) EmbedLoopDAG(taskName, compRef string, task *pipelinespec.PipelineTaskSpec, componentSpec *pipelinespec.ComponentSpec, dagSpec *pipelinespec.DagSpec) (err error) {
	loop := c.PopLoop()

	// inject parallelism if it exists
	parallel := task.GetIteratorPolicy().GetParallelismLimit()
	if parallel > 0 {
		loop.Spec.Parallelism = int(parallel)
	}

	raw, err := json.Marshal(loop.Spec)
	if err != nil {
		return fmt.Errorf("unable to Marshal pipelineSpec:%v", err)
	}

	componentSpecStr, err := c.useComponentSpec(compRef)
	if err != nil {
		return err
	}

	taskSpecJson, err := stablyMarshalJSON(task)
	if err != nil {
		return err
	}

	pipelinelooptask := pipelineapi.PipelineTask{
		Name: taskName + subfixPipelineLoop,
		Params: []pipelineapi.Param{
			{Name: paramParentDagID, Value: pipelineapi.ParamValue{
				Type: "string", StringVal: taskOutputParameter(getDAGDriverTaskName(taskName), paramExecutionID)}},
			{Name: "from", Value: pipelineapi.ParamValue{Type: "string", StringVal: "0"}},
			{Name: "step", Value: pipelineapi.ParamValue{Type: "string", StringVal: "1"}},
			{Name: "to", Value: pipelineapi.ParamValue{Type: "string", StringVal: taskOutputParameter(getDAGDriverTaskName(taskName), paramIterationCount)}},
			// "--type"
			{
				Name:  paramNameType,
				Value: pipelineapi.ParamValue{Type: "string", StringVal: "DAG"},
			},
			// "--pipeline-name"
			{
				Name:  paramNamePipelineName,
				Value: pipelineapi.ParamValue{Type: "string", StringVal: c.spec.GetPipelineInfo().GetName()},
			},
			// "--run-id"
			{
				Name:  paramRunId,
				Value: pipelineapi.ParamValue{Type: "string", StringVal: runID()},
			},
			// "--dag-execution-id"
			{
				Name:  paramNameDagExecutionId,
				Value: pipelineapi.ParamValue{Type: "string", StringVal: taskOutputParameter(getDAGDriverTaskName(taskName), paramExecutionID)},
			},
			// "--component"
			{
				Name:  paramComponent,
				Value: pipelineapi.ParamValue{Type: "string", StringVal: componentSpecStr},
			},
			// "--task"
			{
				Name:  paramTask,
				Value: pipelineapi.ParamValue{Type: "string", StringVal: taskSpecJson},
			},
			// "--runtime-config"
			{
				Name:  paramNameRuntimeConfig,
				Value: pipelineapi.ParamValue{Type: "string", StringVal: ""},
			},
			// "--mlmd-server-address"
			{
				Name:  paramNameMLMDServerHost,
				Value: pipelineapi.ParamValue{Type: "string", StringVal: GetMLMDHost()},
			},
			// "--mlmd_server_port"
			{
				Name:  paramNameMLMDServerPort,
				Value: pipelineapi.ParamValue{Type: "string", StringVal: GetMLMDPort()},
			},
		},
		TaskSpec: &pipelineapi.EmbeddedTask{
			TypeMeta: runtime.TypeMeta{
				Kind:       kindPipelineLoop,
				APIVersion: pipelineloopapi.SchemeGroupVersion.String(),
			},
			Spec: runtime.RawExtension{
				Raw: raw,
			},
		},
		RunAfter: task.GetDependentTasks(),
	}

	c.addPipelineTask(&pipelinelooptask)

	c.PopDagStack()

	return nil
}

func (c *pipelinerunCompiler) DAG(taskName, compRef string, task *pipelinespec.PipelineTaskSpec, componentSpec *pipelinespec.ComponentSpec, dagSpec *pipelinespec.DagSpec) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("compiling DAG %q: %w", taskName, err)
		}
	}()

	// DAG with iteration already generate the compoentSpec string
	if task.GetIterator() == nil {
		err = c.saveComponentSpec(compRef, componentSpec)
		if err != nil {
			return err
		}
	}

	if err := c.addDagTask(taskName, compRef, task, false); err != nil {
		return err
	}

	if err := c.addDagPubTask(taskName, dagSpec, c.spec, false, task.GetIterator() != nil); err != nil {
		return err
	}
	return nil
}

func (c *pipelinerunCompiler) addDagTask(name, compRef string, task *pipelinespec.PipelineTaskSpec, loopDag bool) error {
	driverTaskName := getDAGDriverTaskName(name)
	componentSpecStr, err := c.useComponentSpec(compRef)
	if err != nil {
		return err
	}

	inputs := dagDriverInputs{
		component: componentSpecStr,
	}

	if name == compiler.RootComponentName {
		// runtime config is input to the entire pipeline (root DAG)
		inputs.runtimeConfig = c.job.GetRuntimeConfig()
	} else {
		//sub-dag and task shall not be nil
		if task == nil {
			return fmt.Errorf("invalid sub-dag")
		}
		taskSpecJson, err := stablyMarshalJSON(task)
		if err != nil {
			return err
		}
		inputs.task = taskSpecJson
		inputs.deps = task.GetDependentTasks()
		inputs.parentDagID = c.CurrentDag()
		inputs.inLoopDag = c.HasLoopName(c.CurrentDag())
	}

	if loopDag {
		inputs.iterationIndex = inputValue(paramIterationIndex)
		inputs.loopDag = true
		c.AddLoopName(name)
	} else {
		driver, err := c.dagDriverTask(driverTaskName, &inputs)
		if err != nil {
			return err
		}
		c.addPipelineTask(driver)
	}
	return nil
}

func (c *pipelinerunCompiler) addDagPubTask(name string, dagSpec *pipelinespec.DagSpec, pipelineSpec *pipelinespec.PipelineSpec, inLoopDag, loopDag bool) error {

	if c.exithandler != nil && name == compiler.RootComponentName {
		// this dag-pub only depends on the exit handler task, lets find out its name
		exithandlerTask := ""
		for name, task := range dagSpec.GetTasks() {
			if task.GetTriggerPolicy().GetStrategy() == pipelinespec.PipelineTaskSpec_TriggerPolicy_ALL_UPSTREAM_TASKS_COMPLETED {
				exithandlerTask = name
				break
			}
		}
		pubdriver, err := c.dagPubDriverTask(getDAGPubTaskName(name), &pubDagDriverInputs{
			deps: []string{exithandlerTask}, parentDagID: name, inLoopDag: inLoopDag})
		if err != nil {
			return err
		}
		// Add root dag pub to exithandler's pipelinerun to make sure it will be executed
		c.addExitHandlerTask(pubdriver)
	} else {
		leaves := getLeafNodes(dagSpec, c.spec)
		if loopDag {
			leaves = []string{name + subfixPipelineLoop}
		}
		pubdriver, err := c.dagPubDriverTask(getDAGPubTaskName(name), &pubDagDriverInputs{
			deps: leaves, parentDagID: name, inLoopDag: inLoopDag})
		if err != nil {
			return err
		}
		c.addPipelineTask(pubdriver)
	}
	return nil
}

func (c *pipelinerunCompiler) createPipelineLoop() *pipelineloopapi.PipelineLoop {
	return &pipelineloopapi.PipelineLoop{
		TypeMeta: k8smeta.TypeMeta{
			Kind:       kindPipelineLoop,
			APIVersion: pipelineloopapi.SchemeGroupVersion.String(),
		},
		Spec: pipelineloopapi.PipelineLoopSpec{
			PipelineSpec: &pipelineapi.PipelineSpec{
				Params: []pipelineapi.ParamSpec{
					{Name: paramNameDagExecutionId, Type: "string"},
					{Name: paramIterationIndex, Type: "string"},
				}},
			IterateNumeric: paramIterationIndex,
		},
	}
}

func (c *pipelinerunCompiler) dagDriverTask(
	name string,
	inputs *dagDriverInputs,
) (*pipelineapi.PipelineTask, error) {
	if inputs == nil || len(inputs.component) == 0 {
		return nil, fmt.Errorf("dagDriverTask: component must be non-nil")
	}
	runtimeConfigJson := ""
	if inputs.runtimeConfig != nil {
		rtStr, err := stablyMarshalJSON(inputs.runtimeConfig)
		if err != nil {
			return nil, fmt.Errorf("dagDriverTask: marshaling runtime config to proto JSON failed: %w", err)
		}
		runtimeConfigJson = rtStr
	}

	t := &pipelineapi.PipelineTask{
		Name: name,
		TaskRef: &pipelineapi.TaskRef{
			APIVersion: "custom.tekton.dev/v1alpha1",
			Kind:       "KFPTask",
		},
		Params: []pipelineapi.Param{
			// "--type"
			{
				Name:  paramNameType,
				Value: pipelineapi.ParamValue{Type: "string", StringVal: inputs.getDagType()},
			},
			// "--pipeline-name"
			{
				Name:  paramNamePipelineName,
				Value: pipelineapi.ParamValue{Type: "string", StringVal: c.spec.GetPipelineInfo().GetName()},
			},
			// "--run-id"
			{
				Name:  paramRunId,
				Value: pipelineapi.ParamValue{Type: "string", StringVal: runID()},
			},
			// "--dag-execution-id"
			{
				Name:  paramNameDagExecutionId,
				Value: pipelineapi.ParamValue{Type: "string", StringVal: inputs.getParentDagID(c.ExitHandlerScope())},
			},
			// "--component"
			{
				Name:  paramComponent,
				Value: pipelineapi.ParamValue{Type: "string", StringVal: inputs.component},
			},
			// "--task"
			{
				Name:  paramTask,
				Value: pipelineapi.ParamValue{Type: "string", StringVal: inputs.task},
			},
			// "--runtime-config"
			{
				Name:  paramNameRuntimeConfig,
				Value: pipelineapi.ParamValue{Type: "string", StringVal: runtimeConfigJson},
			},
			// "--iteration-index"
			{
				Name:  paramNameIterationIndex,
				Value: pipelineapi.ParamValue{Type: "string", StringVal: inputs.getIterationIndex()},
			},
			// "--mlmd-server-address"
			{
				Name:  paramNameMLMDServerHost,
				Value: pipelineapi.ParamValue{Type: "string", StringVal: GetMLMDHost()},
			},
			// "--mlmd_server_port"
			{
				Name:  paramNameMLMDServerPort,
				Value: pipelineapi.ParamValue{Type: "string", StringVal: GetMLMDPort()},
			},
			// produce the following outputs:
			// - execution-id
			// - iteration-count
			// - condition
		},
	}
	if len(inputs.deps) > 0 && !(c.ExitHandlerScope() && inputs.parentDagID == compiler.RootComponentName) && !inputs.loopDag {
		t.RunAfter = inputs.deps
	}

	return t, nil
}

func (c *pipelinerunCompiler) dagPubDriverTask(
	name string,
	inputs *pubDagDriverInputs,
) (*pipelineapi.PipelineTask, error) {

	rootDagPub := c.exithandler != nil && inputs.parentDagID == compiler.RootComponentName
	t := &pipelineapi.PipelineTask{
		Name: name,
		TaskRef: &pipelineapi.TaskRef{
			APIVersion: "custom.tekton.dev/v1alpha1",
			Kind:       "KFPTask",
		},
		Params: []pipelineapi.Param{
			// "--type"
			{
				Name:  paramNameType,
				Value: pipelineapi.ParamValue{Type: "string", StringVal: inputs.getDagType()},
			},
			// "--pipeline-name"
			{
				Name:  paramNamePipelineName,
				Value: pipelineapi.ParamValue{Type: "string", StringVal: c.spec.GetPipelineInfo().GetName()},
			},
			// "--run-id"
			{
				Name:  paramRunId,
				Value: pipelineapi.ParamValue{Type: "string", StringVal: runID()},
			},
			// "--dag-execution-id"
			{
				Name:  paramNameDagExecutionId,
				Value: pipelineapi.ParamValue{Type: "string", StringVal: inputs.getParentDagID(c.ExitHandlerScope() || rootDagPub)},
			},
			// "--mlmd-server-address"
			{
				Name:  paramNameMLMDServerHost,
				Value: pipelineapi.ParamValue{Type: "string", StringVal: GetMLMDHost()},
			},
			// "--mlmd-server-port"
			{
				Name:  paramNameMLMDServerPort,
				Value: pipelineapi.ParamValue{Type: "string", StringVal: GetMLMDPort()},
			},
		},
	}
	if len(inputs.deps) > 0 {
		t.RunAfter = inputs.deps
	}
	return t, nil
}

type dagDriverInputs struct {
	parentDagID    string                                  // parent DAG execution ID. optional, the root DAG does not have parent
	component      string                                  // input placeholder for component spec
	task           string                                  // optional, the root DAG does not have task spec.
	runtimeConfig  *pipelinespec.PipelineJob_RuntimeConfig // optional, only root DAG needs this
	iterationIndex string                                  // optional, iterator passes iteration index to iteration tasks
	deps           []string
	loopDag        bool
	inLoopDag      bool
}

type pubDagDriverInputs struct {
	parentDagID string
	deps        []string
	inLoopDag   bool
}

func (i *pubDagDriverInputs) getDagType() string {
	return "DAG_PUB"
}

// pubDagDriverInputs getParentDagID returns the parent node of the DAG publisher
// which should always be the DAG driver DAG ID. However, exit handler doesn't
// have driver so it's point to the root DAG ID instead.
func (i *pubDagDriverInputs) getParentDagID(isExitHandler bool) string {
	if isExitHandler && i.parentDagID == compiler.RootComponentName {
		return fmt.Sprintf("$(params.%s)", paramParentDagID)
	} else {
		return taskOutputParameter(getDAGDriverTaskName(i.parentDagID), paramExecutionID)
	}
}

func (i *dagDriverInputs) getParentDagID(isExitHandler bool) string {
	if i.parentDagID == "" {
		return "0"
	}
	if isExitHandler && i.parentDagID == compiler.RootComponentName {
		return fmt.Sprintf("$(params.%s)", paramParentDagID)
	} else if i.loopDag || i.inLoopDag {
		return fmt.Sprintf("$(params.%s)", paramNameDagExecutionId)
	} else {
		return taskOutputParameter(getDAGDriverTaskName(i.parentDagID), paramExecutionID)
	}
}

func (i *dagDriverInputs) getDagType() string {
	if i.runtimeConfig != nil {
		return "ROOT_DAG"
	}
	return "DAG"
}

func (i *dagDriverInputs) getIterationIndex() string {
	if i.iterationIndex == "" {
		return "-1"
	}
	return i.iterationIndex
}

func addImplicitDependencies(dagSpec *pipelinespec.DagSpec) error {
	for _, task := range dagSpec.GetTasks() {
		wrap := func(err error) error {
			return fmt.Errorf("failed to add implicit deps: %w", err)
		}
		addDep := func(producer string) error {
			if _, ok := dagSpec.GetTasks()[producer]; !ok {
				return fmt.Errorf("unknown producer task %q in DAG", producer)
			}
			if task.DependentTasks == nil {
				task.DependentTasks = make([]string, 0)
			}
			// add the dependency if it's not already added
			found := false
			for _, dep := range task.DependentTasks {
				if dep == producer {
					found = true
					break
				}
			}
			if !found {
				task.DependentTasks = append(task.DependentTasks, producer)
			}
			return nil
		}
		for _, input := range task.GetInputs().GetParameters() {
			switch input.GetKind().(type) {
			case *pipelinespec.TaskInputsSpec_InputParameterSpec_TaskOutputParameter:
				if err := addDep(input.GetTaskOutputParameter().GetProducerTask()); err != nil {
					return wrap(err)
				}
			case *pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus_:
				return wrap(fmt.Errorf("task final status not supported yet"))
			default:
				// other parameter input types do not introduce implicit dependencies
			}
		}
		for _, input := range task.GetInputs().GetArtifacts() {
			switch input.GetKind().(type) {
			case *pipelinespec.TaskInputsSpec_InputArtifactSpec_TaskOutputArtifact:
				if err := addDep(input.GetTaskOutputArtifact().GetProducerTask()); err != nil {
					return wrap(err)
				}
			default:
				// other artifact input types do not introduce implicit dependencies
			}
		}
	}
	return nil
}

// depends builds an enhanced depends string for argo.
// Argo DAG normal dependencies run even when upstream tasks are skipped, which
// is not what we want. Using enhanced depends, we can be strict that upstream
// tasks must be succeeded.
// https://argoproj.github.io/argo-workflows/enhanced-depends-logic/
func depends(deps []string) string {
	if len(deps) == 0 {
		return ""
	}
	var builder strings.Builder
	for index, dep := range deps {
		if index > 0 {
			builder.WriteString(" && ")
		}
		builder.WriteString(dep)
		builder.WriteString(".Succeeded")
	}
	return builder.String()
}

// Exit handler task happens no matter the state of the upstream tasks
func depends_exit_handler(deps []string) string {
	if len(deps) == 0 {
		return ""
	}
	var builder strings.Builder
	for index, dep := range deps {
		if index > 0 {
			builder.WriteString(" || ")
		}
		for inner_index, task_status := range []string{".Succeeded", ".Skipped", ".Failed", ".Errored"} {
			if inner_index > 0 {
				builder.WriteString(" || ")
			}
			builder.WriteString(dep)
			builder.WriteString(task_status)
		}
	}
	return builder.String()
}
