package compiler

import (
	"fmt"

	wfapi "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"google.golang.org/protobuf/encoding/protojson"
	k8score "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (c *workflowCompiler) DAG(name string, componentSpec *pipelinespec.ComponentSpec, dagSpec *pipelinespec.DagSpec) error {
	err := addImplicitDependencies(dagSpec)
	if err != nil {
		return err
	}
	dag := &wfapi.Template{
		Inputs: wfapi.Inputs{
			Parameters: []wfapi.Parameter{
				{Name: paramDAGExecutionID},
			},
		},
		DAG: &wfapi.DAGTemplate{},
	}
	for taskName, kfpTask := range dagSpec.GetTasks() {
		bytes, err := protojson.Marshal(kfpTask)
		if err != nil {
			return fmt.Errorf("DAG: marshaling task %q's spec to proto JSON failed: %w", taskName, err)
		}
		taskJson := string(bytes)
		if kfpTask.GetParameterIterator() != nil && kfpTask.GetArtifactIterator() != nil {
			return fmt.Errorf("DAG: invalid task %q: parameterIterator and artifactIterator cannot be specified at the same time", taskName)
		}
		if kfpTask.GetArtifactIterator() != nil {
			return fmt.Errorf("DAG: artifact iterator not implemented yet")
		}
		// For a normal task, we execute the component's template directly.
		templateName := c.templateName(kfpTask.GetComponentRef().GetName())
		// For iterator task, we need to use argo withSequence to iterate.
		if kfpTask.GetParameterIterator() != nil {
			iterator, err := c.iteratorTemplate(taskName, kfpTask, taskJson)
			if err != nil {
				return fmt.Errorf("DAG: invalid parameter iterator: %w", err)
			}
			iteratorTemplateName, err := c.addTemplate(iterator, name+"-"+taskName)
			if err != nil {
				return fmt.Errorf("DAG: %w", err)
			}
			templateName = iteratorTemplateName
		}
		dag.DAG.Tasks = append(dag.DAG.Tasks, wfapi.DAGTask{
			Name:         kfpTask.GetTaskInfo().GetName(),
			Template:     templateName,
			Dependencies: kfpTask.GetDependentTasks(),
			Arguments: wfapi.Arguments{
				Parameters: []wfapi.Parameter{
					{
						Name:  paramDAGExecutionID,
						Value: wfapi.AnyStringPtr(inputParameter(paramDAGExecutionID)),
					},
					{
						Name:  paramTask,
						Value: wfapi.AnyStringPtr(taskJson),
					},
				},
			},
		})
	}
	// TODO(Bobgy): how can we avoid template name collisions?
	dagName, err := c.addTemplate(dag, name+"-dag")
	if err != nil {
		return fmt.Errorf("DAG: %w", err)
	}
	var runtimeConfig *pipelinespec.PipelineJob_RuntimeConfig
	if name == rootComponentName {
		// runtime config is input to the entire pipeline (root DAG)
		runtimeConfig = c.job.GetRuntimeConfig()
	}
	driverTask, driverOutputs, err := c.dagDriverTask("driver", dagDriverInputs{
		dagExecutionID: inputParameter(paramDAGExecutionID),
		task:           inputParameter(paramTask),
		iterationIndex: inputParameter(paramIterationIndex),
		component:      componentSpec,
		runtimeConfig:  runtimeConfig,
	})
	if err != nil {
		return err
	}
	wrapper := &wfapi.Template{
		Inputs: wfapi.Inputs{
			Parameters: []wfapi.Parameter{
				{Name: paramDAGExecutionID, Default: wfapi.AnyStringPtr("0")},
				{Name: paramTask, Default: wfapi.AnyStringPtr("{}")},
				{Name: paramIterationIndex, Default: wfapi.AnyStringPtr("-1")},
			},
		},
		DAG: &wfapi.DAGTemplate{Tasks: []wfapi.DAGTask{
			*driverTask,
			{
				Name:         "dag",
				Template:     dagName,
				Dependencies: []string{"driver"},
				Arguments: wfapi.Arguments{Parameters: []wfapi.Parameter{
					{Name: paramDAGExecutionID, Value: wfapi.AnyStringPtr(driverOutputs.executionID)},
				}},
			},
		}},
	}
	_, err = c.addTemplate(wrapper, name)
	return err
}

func (c *workflowCompiler) iteratorTemplate(taskName string, task *pipelinespec.PipelineTaskSpec, taskJson string) (tmpl *wfapi.Template, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("generating template for iterator task %q: %w", taskName, err)
		}
	}()
	componentName := task.GetComponentRef().GetName()
	component, ok := c.spec.GetComponents()[componentName]
	if !ok {
		return nil, fmt.Errorf("cannot find component %q in pipeline spec", componentName)
	}
	driverArgoName := "driver"
	driverInputs := dagDriverInputs{
		component:      component,
		dagExecutionID: inputParameter(paramDAGExecutionID),
		task:           inputParameter(paramTask),
	}
	driverArgoTask, driverOutputs, err := c.dagDriverTask(driverArgoName, driverInputs)
	if err != nil {
		return nil, err
	}
	componentTemplateName := c.templateName(task.GetComponentRef().GetName())
	iterationCount := intstr.FromString(driverOutputs.iterationCount)
	tmpl = &wfapi.Template{
		Inputs: wfapi.Inputs{
			Parameters: []wfapi.Parameter{
				{Name: paramDAGExecutionID},
				{Name: paramTask},
			},
		},
		DAG: &wfapi.DAGTemplate{Tasks: []wfapi.DAGTask{
			*driverArgoTask,
			{
				Name:         "iterations",
				Template:     componentTemplateName,
				Dependencies: []string{driverArgoName},
				Arguments: wfapi.Arguments{
					Parameters: []wfapi.Parameter{{
						Name:  paramDAGExecutionID,
						Value: wfapi.AnyStringPtr(driverOutputs.executionID),
					}, {
						Name:  paramTask,
						Value: wfapi.AnyStringPtr(inputParameter(paramTask)),
					}, {
						Name:  paramIterationIndex,
						Value: wfapi.AnyStringPtr(loopItem()),
					}},
				},
				WithSequence: &wfapi.Sequence{Count: &iterationCount},
			},
		}},
	}
	return tmpl, nil
}

type dagDriverOutputs struct {
	executionID    string
	iterationCount string // only returned for iterator DAG drivers
}

type dagDriverInputs struct {
	dagExecutionID string // parent DAG execution ID. optional, the root DAG does not have parent
	component      *pipelinespec.ComponentSpec
	task           string                                  // optional, the root DAG does not have task spec.
	runtimeConfig  *pipelinespec.PipelineJob_RuntimeConfig // optional, only root DAG needs this
	iterationIndex string                                  // optional, iterator passes iteration index to iteration tasks
}

func (c *workflowCompiler) dagDriverTask(name string, inputs dagDriverInputs) (*wfapi.DAGTask, *dagDriverOutputs, error) {
	component := inputs.component
	runtimeConfig := inputs.runtimeConfig
	if inputs.iterationIndex == "" {
		inputs.iterationIndex = "-1"
	}
	if component == nil {
		return nil, nil, fmt.Errorf("dagDriverTask: component must be non-nil")
	}
	componentBytes, err := protojson.Marshal(component)
	if err != nil {
		return nil, nil, fmt.Errorf("dagDriverTask: marlshaling component spec to proto JSON failed: %w", err)
	}
	componentJson := string(componentBytes)
	runtimeConfigJson := "{}"
	if runtimeConfig != nil {
		bytes, err := protojson.Marshal(runtimeConfig)
		if err != nil {
			return nil, nil, fmt.Errorf("dagDriverTask: marshaling runtime config to proto JSON failed: %w", err)
		}
		runtimeConfigJson = string(bytes)
	}
	templateName := c.addDAGDriverTemplate()
	t := &wfapi.DAGTask{
		Name:     name,
		Template: templateName,
		Arguments: wfapi.Arguments{
			Parameters: []wfapi.Parameter{{
				Name:  paramDAGExecutionID,
				Value: wfapi.AnyStringPtr(inputs.dagExecutionID),
			}, {
				Name:  paramComponent,
				Value: wfapi.AnyStringPtr(componentJson),
			}, {
				Name:  paramTask,
				Value: wfapi.AnyStringPtr(inputs.task),
			}, {
				Name:  paramRuntimeConfig,
				Value: wfapi.AnyStringPtr(runtimeConfigJson),
			}, {
				Name:  paramIterationIndex,
				Value: wfapi.AnyStringPtr(inputs.iterationIndex),
			}},
		},
	}
	if runtimeConfig != nil {
		t.Arguments.Parameters = append(t.Arguments.Parameters, wfapi.Parameter{
			Name:  paramDriverType,
			Value: wfapi.AnyStringPtr("ROOT_DAG"),
		})
	}
	return t, &dagDriverOutputs{
		executionID:    taskOutputParameter(name, paramExecutionID),
		iterationCount: taskOutputParameter(name, paramIterationCount),
	}, nil
}

func (c *workflowCompiler) addDAGDriverTemplate() string {
	name := "system-dag-driver"
	_, ok := c.templates[name]
	if ok {
		return name
	}
	t := &wfapi.Template{
		Name: name,
		Inputs: wfapi.Inputs{
			Parameters: []wfapi.Parameter{
				{Name: paramComponent},
				{Name: paramRuntimeConfig},
				{Name: paramTask},
				{Name: paramDAGExecutionID, Default: wfapi.AnyStringPtr("0")},
				{Name: paramIterationIndex, Default: wfapi.AnyStringPtr("-1")},
				{Name: paramDriverType, Default: wfapi.AnyStringPtr("DAG")},
			},
		},
		Outputs: wfapi.Outputs{
			Parameters: []wfapi.Parameter{
				{Name: paramExecutionID, ValueFrom: &wfapi.ValueFrom{Path: "/tmp/outputs/execution-id"}},
				{Name: paramIterationCount, ValueFrom: &wfapi.ValueFrom{Path: "/tmp/outputs/iteration-count", Default: wfapi.AnyStringPtr("0")}},
			},
		},
		Container: &k8score.Container{
			Image:   c.driverImage,
			Command: []string{"driver"},
			Args: []string{
				"--type", inputValue(paramDriverType),
				"--pipeline_name", c.spec.GetPipelineInfo().GetName(),
				"--run_id", runID(),
				"--dag_execution_id", inputValue(paramDAGExecutionID),
				"--component", inputValue(paramComponent),
				"--task", inputValue(paramTask),
				"--runtime_config", inputValue(paramRuntimeConfig),
				"--iteration_index", inputValue(paramIterationIndex),
				"--execution_id_path", outputPath(paramExecutionID),
				"--iteration_count_path", outputPath(paramIterationCount),
			},
			Resources: driverResources,
		},
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
