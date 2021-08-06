package compiler

import (
	"fmt"

	wfapi "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/golang/protobuf/jsonpb"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	k8score "k8s.io/api/core/v1"
)

func (c *workflowCompiler) DAG(name string, componentSpec *pipelinespec.ComponentSpec, dagSpec *pipelinespec.DagSpec) error {
	if name != "root" {
		return fmt.Errorf("SubDAG not implemented yet")
	}
	err := addImplicitDependencies(dagSpec)
	if err != nil {
		return err
	}
	dag := &wfapi.Template{
		Inputs: wfapi.Inputs{
			Parameters: []wfapi.Parameter{
				{Name: paramDAGContextID},
				{Name: paramDAGExecutionID},
			},
		},
		DAG: &wfapi.DAGTemplate{},
	}
	for _, kfpTask := range dagSpec.GetTasks() {
		marshaler := jsonpb.Marshaler{}
		taskJson, err := marshaler.MarshalToString(kfpTask)
		if err != nil {
			return fmt.Errorf("DAG: marshaling task spec to proto JSON failed: %w", err)
		}
		dag.DAG.Tasks = append(dag.DAG.Tasks, wfapi.DAGTask{
			Name:         kfpTask.GetTaskInfo().GetName(),
			Template:     c.templateName(kfpTask.GetComponentRef().GetName()),
			Dependencies: kfpTask.GetDependentTasks(),
			Arguments: wfapi.Arguments{
				Parameters: []wfapi.Parameter{
					{
						Name:  paramDAGContextID,
						Value: wfapi.AnyStringPtr(inputParameter(paramDAGContextID)),
					},
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
	task := &pipelinespec.PipelineTaskSpec{}
	var runtimeConfig *pipelinespec.PipelineJob_RuntimeConfig
	if name == "root" {
		// runtime config is input to the entire pipeline (root DAG)
		runtimeConfig = c.job.GetRuntimeConfig()
	}
	driverTask, outputs, err := c.dagDriverTask("driver", componentSpec, task, runtimeConfig)
	if err != nil {
		return err
	}
	wrapper := &wfapi.Template{}
	wrapper.DAG = &wfapi.DAGTemplate{
		Tasks: []wfapi.DAGTask{
			*driverTask,
			{
				Name: "dag", Template: dagName, Dependencies: []string{"driver"},
				Arguments: wfapi.Arguments{
					Parameters: []wfapi.Parameter{
						{Name: paramDAGExecutionID, Value: wfapi.AnyStringPtr(outputs.executionID)},
						{Name: paramDAGContextID, Value: wfapi.AnyStringPtr(outputs.contextID)},
					},
				},
			},
		},
	}
	_, err = c.addTemplate(wrapper, name)
	return err
}

type dagDriverOutputs struct {
	contextID, executionID string
}

func (c *workflowCompiler) dagDriverTask(name string, component *pipelinespec.ComponentSpec, task *pipelinespec.PipelineTaskSpec, runtimeConfig *pipelinespec.PipelineJob_RuntimeConfig) (*wfapi.DAGTask, *dagDriverOutputs, error) {
	if component == nil {
		return nil, nil, fmt.Errorf("dagDriverTask: component must be non-nil")
	}
	marshaler := jsonpb.Marshaler{}
	componentJson, err := marshaler.MarshalToString(component)
	if err != nil {
		return nil, nil, fmt.Errorf("dagDriverTask: marlshaling component spec to proto JSON failed: %w", err)
	}
	taskJson := "{}"
	if task != nil {
		taskJson, err = marshaler.MarshalToString(task)
		if err != nil {
			return nil, nil, fmt.Errorf("dagDriverTask: marshaling task spec to proto JSON failed: %w", err)
		}
	}
	runtimeConfigJson := "{}"
	if runtimeConfig != nil {
		runtimeConfigJson, err = marshaler.MarshalToString(runtimeConfig)
		if err != nil {
			return nil, nil, fmt.Errorf("dagDriverTask: marshaling runtime config to proto JSON failed: %w", err)
		}
	}
	templateName := c.addDAGDriverTemplate()
	t := &wfapi.DAGTask{
		Name:     name,
		Template: templateName,
		Arguments: wfapi.Arguments{
			Parameters: []wfapi.Parameter{
				{
					Name:  paramComponent,
					Value: wfapi.AnyStringPtr(componentJson),
				},
				{
					Name:  paramTask,
					Value: wfapi.AnyStringPtr(taskJson),
				},
				{
					Name:  paramRuntimeConfig,
					Value: wfapi.AnyStringPtr(runtimeConfigJson),
				},
			},
		},
	}
	return t, &dagDriverOutputs{
		contextID:   taskOutputParameter(name, paramContextID),
		executionID: taskOutputParameter(name, paramExecutionID),
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
			},
		},
		Outputs: wfapi.Outputs{
			Parameters: []wfapi.Parameter{
				{Name: paramExecutionID, ValueFrom: &wfapi.ValueFrom{Path: "/tmp/outputs/execution-id"}},
				{Name: paramContextID, ValueFrom: &wfapi.ValueFrom{Path: "/tmp/outputs/context-id"}},
			},
		},
		Container: &k8score.Container{
			Image:   c.driverImage,
			Command: []string{"driver"},
			Args: []string{
				"--type", "ROOT_DAG",
				"--pipeline_name", c.spec.GetPipelineInfo().GetName(),
				"--run_id", runID(),
				"--component", inputValue(paramComponent),
				"--runtime_config", inputValue(paramRuntimeConfig),
				"--execution_id_path", outputPath(paramExecutionID),
				"--context_id_path", outputPath(paramContextID),
			},
		},
	}
	c.templates[name] = t
	c.wf.Spec.Templates = append(c.wf.Spec.Templates, *t)
	return name
}

func addImplicitDependencies(dagSpec *pipelinespec.DagSpec) error {
	for _, task := range dagSpec.GetTasks() {
		// TODO(Bobgy): add implicit dependencies introduced by artifacts
		for _, input := range task.GetInputs().GetParameters() {
			wrap := func(err error) error {
				return fmt.Errorf("failed to add implicit deps: %w", err)
			}
			switch input.Kind.(type) {
			case *pipelinespec.TaskInputsSpec_InputParameterSpec_TaskOutputParameter:
				producer := input.GetTaskOutputParameter().GetProducerTask()
				_, ok := dagSpec.GetTasks()[producer]
				if !ok {
					return wrap(fmt.Errorf("unknown producer task %q in DAG", producer))
				}
				if task.DependentTasks == nil {
					task.DependentTasks = make([]string, 0)
				}
				// add the dependency if it's not already added
				found := false
				for _, dep := range task.DependentTasks {
					if dep == producer {
						found = true
					}
				}
				if !found {
					task.DependentTasks = append(task.DependentTasks, producer)
				}
			case *pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus_:
				return wrap(fmt.Errorf("task final status not supported yet"))
			default:
				// other input types do not introduce implicit dependencies
			}
		}
	}
	return nil
}
