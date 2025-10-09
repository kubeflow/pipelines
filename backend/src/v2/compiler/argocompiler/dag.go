// Copyright 2021-2023 The Kubeflow Authors
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
package argocompiler

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"

	wfapi "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/v2/compiler"
	k8score "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (c *workflowCompiler) DAG(name string, componentSpec *pipelinespec.ComponentSpec, dagSpec *pipelinespec.DagSpec) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("compiling DAG %q: %w", name, err)
		}
	}()
	err = addImplicitDependencies(dagSpec)
	if err != nil {
		return err
	}
	err = c.saveComponentSpec(name, componentSpec)
	if err != nil {
		return err
	}
	tasks := dagSpec.GetTasks()
	// Iterate through tasks in deterministic order to facilitate testing.
	// Note, order doesn't affect compiler with real effect right now.
	// In the future, we may consider using topology sort when building local
	// executor that runs on pipeline spec directly.
	keys := make([]string, 0, len(tasks))
	for key := range tasks {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	taskToExitTemplate := map[string]string{}

	// First process exit tasks since those need to be set as lifecycle hooks on the exit handler sub DAG.
	for _, taskName := range keys {
		kfpTask := dagSpec.GetTasks()[taskName]
		if kfpTask.GetParameterIterator() != nil && kfpTask.GetArtifactIterator() != nil {
			return fmt.Errorf("invalid task %q: parameterIterator and artifactIterator cannot be specified at the same time", taskName)
		}
		if kfpTask.GetArtifactIterator() != nil {
			return fmt.Errorf("artifact iterator not implemented yet")
		}

		if kfpTask.GetTriggerPolicy().GetStrategy().String() != "ALL_UPSTREAM_TASKS_COMPLETED" {
			// Skip tasks that aren't exit tasks.
			continue
		}

		tasks, err := c.task(taskName, kfpTask, taskInputs{
			parentDagID: inputParameter(paramParentDagID),
		})
		if err != nil {
			return err
		}

		// Generate the template name in the format of "exit-hook-<DAG name>-<task name>"
		// (e.g. exit-hook-root-print-op).
		name := fmt.Sprintf("exit-hook-%s-%s", name, taskName)

		deps := kfpTask.GetDependentTasks()

		for _, dep := range deps {
			taskToExitTemplate[dep] = name
		}

		exitDag := &wfapi.Template{
			Name: name,
			Inputs: wfapi.Inputs{
				Parameters: []wfapi.Parameter{
					{Name: paramParentDagID},
				},
			},
			DAG: &wfapi.DAGTemplate{
				Tasks: tasks,
			},
		}

		_, err = c.addTemplate(exitDag, name)
		if err != nil {
			return fmt.Errorf("DAG: %w", err)
		}
	}

	dag := &wfapi.Template{
		Name: c.templateName(name),
		Inputs: wfapi.Inputs{
			Parameters: []wfapi.Parameter{
				{Name: paramParentDagID},
			},
		},
		DAG: &wfapi.DAGTemplate{},
	}

	for _, taskName := range keys {
		kfpTask := dagSpec.GetTasks()[taskName]
		if kfpTask.GetTriggerPolicy().GetStrategy().String() == "ALL_UPSTREAM_TASKS_COMPLETED" {
			// Skip already processed exit tasks.
			continue
		}

		exitTemplate := taskToExitTemplate[taskName]

		tasks, err := c.task(
			taskName, kfpTask, taskInputs{parentDagID: inputParameter(paramParentDagID), exitTemplate: exitTemplate},
		)
		if err != nil {
			return err
		}
		dag.DAG.Tasks = append(dag.DAG.Tasks, tasks...)
	}

	// The compilation should fail before this point, but add it as an extra precaution to guard against an orphaned
	// exit task.
	if len(dag.DAG.Tasks) == 0 {
		return fmt.Errorf("DAG %s must contain one or more non-exit tasks", name)
	}

	_, err = c.addTemplate(dag, name)
	if err != nil {
		return fmt.Errorf("DAG: %w", err)
	}
	if name == compiler.RootComponentName {
		// Create template for entrypoint
		// TODO(Bobgy): consider moving the logic below into c.task()
		// runtime config is input to the entire pipeline (root DAG)
		runtimeConfig := c.job.GetRuntimeConfig()
		driverTaskName := name + "-driver"
		componentSpecPlaceholder, err := c.useComponentSpec(name)
		if err != nil {
			return err
		}
		driver, driverOutputs, err := c.dagDriverTask(driverTaskName, dagDriverInputs{
			component:     componentSpecPlaceholder,
			runtimeConfig: runtimeConfig,
		})
		if err != nil {
			return err
		}
		dag := c.dagTask("root", name, dagInputs{
			parentDagID: driverOutputs.executionID,
		})
		dag.Depends = depends([]string{driverTaskName})
		entrypoint := &wfapi.Template{
			Name: tmplEntrypoint,
			DAG: &wfapi.DAGTemplate{
				Tasks: []wfapi.DAGTask{*driver, *dag},
			},
		}
		_, err = c.addTemplate(entrypoint, tmplEntrypoint)
		if err != nil {
			return err
		}
	}
	return nil
}

type dagInputs struct {
	// placeholder for parent DAG execution ID
	parentDagID string
	// if provided along with exitTemplate, this will be provided as the parent-dag-id input to the Argo Workflow exit
	// lifecycle hook.
	hookParentDagID string
	// this will be provided as the parent-dag-id input to the Argo Workflow exit lifecycle hook.
	exitTemplate string
	condition    string
}

// dagTask generates task for a DAG component.
// name: task name
// componentName: DAG component name
func (c *workflowCompiler) dagTask(name string, componentName string, inputs dagInputs) *wfapi.DAGTask {
	task := &wfapi.DAGTask{
		Name:     name,
		Template: c.templateName(componentName),
		Arguments: wfapi.Arguments{Parameters: []wfapi.Parameter{
			{Name: paramParentDagID, Value: wfapi.AnyStringPtr(inputs.parentDagID)},
			{Name: paramCondition, Value: wfapi.AnyStringPtr(inputs.condition)},
		}},
	}

	addExitTask(task, inputs.exitTemplate, inputs.hookParentDagID)

	return task
}

type taskInputs struct {
	parentDagID    string
	iterationIndex string
	// if provided, this will be the template the Argo Workflow exit lifecycle hook will execute.
	exitTemplate string
}

// parentDagID: placeholder for parent DAG execution ID
func (c *workflowCompiler) task(name string, task *pipelinespec.PipelineTaskSpec, inputs taskInputs) (tasks []wfapi.DAGTask, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("compiling task %q: %w", name, err)
		}
	}()
	componentName := task.GetComponentRef().GetName()
	componentSpec, found := c.spec.Components[componentName]
	if !found {
		return nil, fmt.Errorf("component spec for %q not found", componentName)
	}
	componentSpecPlaceholder, err := c.useComponentSpec(componentName)
	if err != nil {
		return nil, err
	}
	taskSpecJson, err := stablyMarshalJSON(task)
	if err != nil {
		return nil, err
	}
	// For iterator task, we need to use argo withSequence to iterate.
	isIteratorTask := inputs.iterationIndex == "" &&
		(task.GetParameterIterator() != nil || task.GetArtifactIterator() != nil)
	if isIteratorTask {
		return c.iteratorTask(name, task, taskSpecJson, inputs.parentDagID)
	}
	switch impl := componentSpec.GetImplementation().(type) {
	case *pipelinespec.ComponentSpec_Dag:
		driverTaskName := name + "-driver"
		driver, driverOutputs, err := c.dagDriverTask(driverTaskName, dagDriverInputs{
			parentDagID:    inputs.parentDagID,
			component:      componentSpecPlaceholder,
			task:           taskSpecJson,
			iterationIndex: inputs.iterationIndex,
			taskName:       name,
		})
		if err != nil {
			return nil, err
		}
		// iterations belong to a sub-DAG, no need to add dependent tasks
		// Also skip adding dependencies when it's an exit hook
		if inputs.iterationIndex == "" && task.GetTriggerPolicy().GetStrategy().String() != "ALL_UPSTREAM_TASKS_COMPLETED" {
			driver.Depends = depends(task.GetDependentTasks())
		}
		dag := c.dagTask(name, componentName, dagInputs{
			parentDagID:     driverOutputs.executionID,
			exitTemplate:    inputs.exitTemplate,
			hookParentDagID: inputs.parentDagID,
			condition:       driverOutputs.condition,
		})
		dag.Depends = depends([]string{driverTaskName})
		if task.GetTriggerPolicy().GetCondition() != "" {
			dag.When = driverOutputs.condition + " != false"
		}
		return []wfapi.DAGTask{*driver, *dag}, nil
	case *pipelinespec.ComponentSpec_ExecutorLabel:
		executor, found := c.executors[impl.ExecutorLabel]
		if !found {
			return nil, fmt.Errorf("executor with label %q not found", impl.ExecutorLabel)
		}
		switch e := executor.GetSpec().(type) {
		case *pipelinespec.PipelineDeploymentConfig_ExecutorSpec_Container:
			containerPlaceholder, err := c.useComponentImpl(componentName)
			if err != nil {
				return nil, err
			}
			driverTaskName := name + "-driver"
			// The following call will return an empty string for tasks without kubernetes-specific annotation.
			kubernetesConfigPlaceholder, _ := c.useKubernetesImpl(componentName)
			driver, driverOutputs := c.containerDriverTask(driverTaskName, containerDriverInputs{
				component:        componentSpecPlaceholder,
				task:             taskSpecJson,
				container:        containerPlaceholder,
				parentDagID:      inputs.parentDagID,
				iterationIndex:   inputs.iterationIndex,
				kubernetesConfig: kubernetesConfigPlaceholder,
				taskName:         name,
			})
			if task.GetTriggerPolicy().GetCondition() == "" {
				driverOutputs.condition = ""
			}
			// iterations belong to a sub-DAG, no need to add dependent tasks
			// Also skip adding dependencies when it's an exit hook
			if inputs.iterationIndex == "" && task.GetTriggerPolicy().GetStrategy().String() != "ALL_UPSTREAM_TASKS_COMPLETED" {
				driver.Depends = depends(task.GetDependentTasks())
			}

			// When using a dummy image, this means this task is for Kubernetes configs.
			// In this case skip executor(launcher).
			if dummyImages[e.Container.GetImage()] {
				driver.Name = name
				return []wfapi.DAGTask{*driver}, nil
			}
			executor, err := c.containerExecutorTask(name, containerExecutorInputs{
				podSpecPatch:    driverOutputs.podSpecPatch,
				cachedDecision:  driverOutputs.cached,
				condition:       driverOutputs.condition,
				exitTemplate:    inputs.exitTemplate,
				hookParentDagID: inputs.parentDagID,
			}, task)
			if err != nil {
				return nil, fmt.Errorf("error creating executor for %q: %v", name, err)
			}
			executor.Depends = depends([]string{driverTaskName})
			return []wfapi.DAGTask{*driver, *executor}, nil
		case *pipelinespec.PipelineDeploymentConfig_ExecutorSpec_Importer:
			if task.GetTriggerPolicy().GetCondition() != "" {
				// Note, because importer task has only one container which runs both the driver and importer,
				// it's impossible to add a when condition based on driver outputs.
				return nil, fmt.Errorf("triggerPolicy.condition on importer task is not supported")
			}
			importer, err := c.importerTask(name, task, taskSpecJson, inputs.parentDagID, e.Importer.GetDownloadToWorkspace())
			if err != nil {
				return nil, err
			}
			return []wfapi.DAGTask{*importer}, nil
		case *pipelinespec.PipelineDeploymentConfig_ExecutorSpec_Resolver:
			return nil, fmt.Errorf("resolver executors not implemented")
		case *pipelinespec.PipelineDeploymentConfig_ExecutorSpec_CustomJob:
			return nil, fmt.Errorf("custom job executors is Google Cloud only, it's not supported")
		default:
			return nil, fmt.Errorf("unknown executor spec type: %T", e)
		}
	default:
		return nil, fmt.Errorf("unsupported component implementation kind: %T", impl)
	}
}

func (c *workflowCompiler) iteratorTask(name string, task *pipelinespec.PipelineTaskSpec, taskJson string, parentDagID string) (tasks []wfapi.DAGTask, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("iterator task: %w", err)
		}
	}()
	componentName := task.GetComponentRef().GetName()
	// Set up Loop Control Template
	iteratorTasks, err := c.iterationItemTask("iteration", task, taskJson, parentDagID)
	if err != nil {
		return nil, err
	}
	loopTmpl := &wfapi.Template{
		Inputs: wfapi.Inputs{
			Parameters: []wfapi.Parameter{
				{Name: paramParentDagID},
			},
		},
		DAG: &wfapi.DAGTemplate{
			Tasks: iteratorTasks,
		},
	}
	parallelismLimit := int64(task.GetIteratorPolicy().GetParallelismLimit())
	if parallelismLimit > 0 {
		loopTmpl.Parallelism = &parallelismLimit
	}

	loopTmplName, err := c.addTemplate(loopTmpl, fmt.Sprintf("%s-%s-iterator", componentName, name))
	if err != nil {
		return nil, err
	}

	tasks = []wfapi.DAGTask{
		{
			Name:     name,
			Template: loopTmplName,
			Depends:  depends(task.GetDependentTasks()),
			Arguments: wfapi.Arguments{
				Parameters: []wfapi.Parameter{
					{
						Name:  paramParentDagID,
						Value: wfapi.AnyStringPtr(parentDagID),
					},
				},
			},
		},
	}
	return tasks, nil
}

func (c *workflowCompiler) iterationItemTask(name string, task *pipelinespec.PipelineTaskSpec, taskJson string, parentDagID string) (tasks []wfapi.DAGTask, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("iterationItem task: %w", err)
		}
	}()
	componentName := task.GetComponentRef().GetName()
	componentSpecPlaceholder, err := c.useComponentSpec(componentName)
	if err != nil {
		return nil, err
	}

	// Set up Iteration (Single  Task) Template
	driverArgoName := name + "-driver"
	driverInputs := dagDriverInputs{
		component:   componentSpecPlaceholder,
		parentDagID: parentDagID,
		task:        taskJson, // TODO(Bobgy): avoid duplicating task JSON twice in the template.
	}
	driver, driverOutputs, err := c.dagDriverTask(driverArgoName, driverInputs)
	if err != nil {
		return nil, err
	}

	iterationCount := intstr.FromString(driverOutputs.iterationCount)
	iterationTasks, err := c.task(
		"iteration-item",
		task,
		taskInputs{
			parentDagID:    inputParameter(paramParentDagID),
			iterationIndex: inputParameter(paramIterationIndex),
		},
	)
	if err != nil {
		return nil, err
	}
	iterationsTmpl := &wfapi.Template{
		Inputs: wfapi.Inputs{
			Parameters: []wfapi.Parameter{
				{Name: paramParentDagID},
				{Name: paramIterationIndex},
			},
		},
		DAG: &wfapi.DAGTemplate{
			Tasks: iterationTasks,
		},
	}
	iterationsTmplName, err := c.addTemplate(iterationsTmpl, componentName+"-"+name)
	if err != nil {
		return nil, err
	}
	when := ""
	if task.GetTriggerPolicy().GetCondition() != "" {
		when = driverOutputs.condition + " != false"
	}

	iteratorTasks := []wfapi.DAGTask{
		*driver,
		{
			Name:     name + "-iterations",
			Template: iterationsTmplName,
			Depends:  depends([]string{driverArgoName}),
			When:     when,
			Arguments: wfapi.Arguments{
				Parameters: []wfapi.Parameter{{
					Name:  paramParentDagID,
					Value: wfapi.AnyStringPtr(driverOutputs.executionID),
				}, {
					Name:  paramIterationIndex,
					Value: wfapi.AnyStringPtr(loopItem()),
				}},
			},
			WithSequence: &wfapi.Sequence{Count: &iterationCount},
		},
	}
	return iteratorTasks, nil
}

type dagDriverOutputs struct {
	executionID    string
	iterationCount string // only returned for iterator DAG drivers
	condition      string // if false, the DAG is skipped
}

type dagDriverInputs struct {
	parentDagID    string                                  // parent DAG execution ID. optional, the root DAG does not have parent
	component      string                                  // input placeholder for component spec
	task           string                                  // optional, the root DAG does not have task spec.
	taskName       string                                  // optional, the name of the task, used for input resolving
	runtimeConfig  *pipelinespec.PipelineJob_RuntimeConfig // optional, only root DAG needs this
	iterationIndex string                                  // optional, iterator passes iteration index to iteration tasks
}

func (c *workflowCompiler) dagDriverTask(name string, inputs dagDriverInputs) (*wfapi.DAGTask, *dagDriverOutputs, error) {
	if inputs.component == "" {
		return nil, nil, fmt.Errorf("dagDriverTask: component must be non-nil")
	}
	params := []wfapi.Parameter{{
		Name:  paramComponent,
		Value: wfapi.AnyStringPtr(inputs.component),
	}}
	if inputs.iterationIndex != "" {
		params = append(params, wfapi.Parameter{
			Name:  paramIterationIndex,
			Value: wfapi.AnyStringPtr(inputs.iterationIndex),
		})
	}
	if inputs.parentDagID != "" {
		params = append(params, wfapi.Parameter{
			Name:  paramParentDagID,
			Value: wfapi.AnyStringPtr(inputs.parentDagID),
		})
	}
	if inputs.runtimeConfig != nil {
		runtimeConfigJson, err := stablyMarshalJSON(inputs.runtimeConfig)
		if err != nil {
			return nil, nil, fmt.Errorf("dagDriverTask: marshaling runtime config to proto JSON failed: %w", err)
		}
		params = append(params, wfapi.Parameter{
			Name:  paramRuntimeConfig,
			Value: wfapi.AnyStringPtr(runtimeConfigJson),
		}, wfapi.Parameter{
			Name:  paramDriverType,
			Value: wfapi.AnyStringPtr("ROOT_DAG"),
		})
	}
	if inputs.task != "" {
		params = append(params, wfapi.Parameter{
			Name:  paramTask,
			Value: wfapi.AnyStringPtr(inputs.task),
		})
	}

	if inputs.taskName != "" && inputs.taskName != "iteration-item" {
		params = append(params, wfapi.Parameter{
			Name:  paramTaskName,
			Value: wfapi.AnyStringPtr(inputs.taskName),
		})
	}
	t := &wfapi.DAGTask{
		Name:     name,
		Template: c.addDAGDriverTemplate(),
		Arguments: wfapi.Arguments{
			Parameters: params,
		},
	}
	return t, &dagDriverOutputs{
		executionID:    taskOutputParameter(name, paramExecutionID),
		iterationCount: taskOutputParameter(name, paramIterationCount),
		condition:      taskOutputParameter(name, paramCondition),
	}, nil
}

func (c *workflowCompiler) addDAGDriverTemplate() string {
	name := "system-dag-driver"
	_, ok := c.templates[name]
	if ok {
		return name
	}

	args := []string{
		"--type", inputValue(paramDriverType),
		"--pipeline_name", c.spec.GetPipelineInfo().GetName(),
		"--run_id", runID(),
		"--run_name", runResourceName(),
		"--run_display_name", c.job.DisplayName,
		"--dag_execution_id", inputValue(paramParentDagID),
		"--component", inputValue(paramComponent),
		"--task", inputValue(paramTask),
		"--task_name", inputValue(paramTaskName),
		"--runtime_config", inputValue(paramRuntimeConfig),
		"--iteration_index", inputValue(paramIterationIndex),
		"--execution_id_path", outputPath(paramExecutionID),
		"--iteration_count_path", outputPath(paramIterationCount),
		"--condition_path", outputPath(paramCondition),
		"--http_proxy", proxy.GetConfig().GetHttpProxy(),
		"--https_proxy", proxy.GetConfig().GetHttpsProxy(),
		"--no_proxy", proxy.GetConfig().GetNoProxy(),
	}
	if c.cacheDisabled {
		args = append(args, "--cache_disabled")
	}
	if c.mlPipelineTLSEnabled {
		args = append(args, "--ml_pipeline_tls_enabled")
	}
	if common.GetMetadataTLSEnabled() {
		args = append(args, "--metadata_tls_enabled")
	}

	setCABundle := false
	if common.GetCaBundleSecretName() != "" && (c.mlPipelineTLSEnabled || common.GetMetadataTLSEnabled()) {
		args = append(args, "--ca_cert_path", common.TLSCertCAPath)
		setCABundle = true
	}

	if value, ok := os.LookupEnv(PipelineLogLevelEnvVar); ok {
		args = append(args, "--log_level", value)
	}
	if value, ok := os.LookupEnv(PublishLogsEnvVar); ok {
		args = append(args, "--publish_logs", value)
	}

	t := &wfapi.Template{
		Name: name,
		Inputs: wfapi.Inputs{
			Parameters: []wfapi.Parameter{
				{Name: paramComponent}, // Required.
				{Name: paramRuntimeConfig, Default: wfapi.AnyStringPtr("")},
				{Name: paramTask, Default: wfapi.AnyStringPtr("")},
				{Name: paramTaskName, Default: wfapi.AnyStringPtr("")},
				{Name: paramParentDagID, Default: wfapi.AnyStringPtr("0")},
				{Name: paramIterationIndex, Default: wfapi.AnyStringPtr("-1")},
				{Name: paramDriverType, Default: wfapi.AnyStringPtr("DAG")},
			},
		},
		Outputs: wfapi.Outputs{
			Parameters: []wfapi.Parameter{
				{Name: paramExecutionID, ValueFrom: &wfapi.ValueFrom{Path: "/tmp/outputs/execution-id"}},
				{Name: paramIterationCount, ValueFrom: &wfapi.ValueFrom{Path: "/tmp/outputs/iteration-count", Default: wfapi.AnyStringPtr("0")}},
				{Name: paramCondition, ValueFrom: &wfapi.ValueFrom{Path: "/tmp/outputs/condition", Default: wfapi.AnyStringPtr("true")}},
			},
		},
		Container: &k8score.Container{
			Image:     c.driverImage,
			Command:   c.driverCommand,
			Args:      args,
			Resources: driverResources,
			Env:       proxy.GetConfig().GetEnvVars(),
		},
	}
	// If TLS is enabled (apiserver or metadata), add the custom CA bundle to the DAG driver template.
	if setCABundle {
		ConfigureCustomCABundle(t)
	}
	c.templates[name] = t
	c.wf.Spec.Templates = append(c.wf.Spec.Templates, *t)
	return name
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
