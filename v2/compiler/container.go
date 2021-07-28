package compiler

import (
	"fmt"

	wfapi "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/golang/protobuf/jsonpb"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	k8score "k8s.io/api/core/v1"
)

func (c *workflowCompiler) Container(name string, component *pipelinespec.ComponentSpec, container *pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec) error {
	if component == nil {
		return fmt.Errorf("workflowCompiler.Container: component spec must be non-nil")
	}
	marshaler := jsonpb.Marshaler{}
	componentJson, err := marshaler.MarshalToString(component)
	if err != nil {
		return fmt.Errorf("workflowCompiler.Container: marlshaling component spec to proto JSON failed: %w", err)
	}
	driverTask, _ := c.containerDriverTask("driver", componentJson, inputParameter(paramTask), inputParameter(paramDAGContextID), inputParameter(paramDAGExecutionID))
	if err != nil {
		return err
	}
	t := &wfapi.Template{
		Container: &k8score.Container{
			Command: container.Command,
			Args:    container.Args,
			Image:   container.Image,
			// TODO(Bobgy): support resource requests/limits
		},
	}
	// TODO(Bobgy): how can we avoid template name collisions?
	containerTemplateName, err := c.addTemplate(t, name+"-container")
	wrapper := &wfapi.Template{
		Inputs: wfapi.Inputs{
			Parameters: []wfapi.Parameter{
				{Name: paramTask},
				{Name: paramDAGContextID},
				{Name: paramDAGExecutionID},
			},
		},
		DAG: &wfapi.DAGTemplate{
			Tasks: []wfapi.DAGTask{
				*driverTask,
				{Name: "container", Template: containerTemplateName, Dependencies: []string{driverTask.Name}},
			},
		},
	}
	_, err = c.addTemplate(wrapper, name)
	return err
}

type containerDriverOutputs struct {
}

func (c *workflowCompiler) containerDriverTask(name, component, task, dagContextID, dagExecutionID string) (*wfapi.DAGTask, *containerDriverOutputs) {
	return &wfapi.DAGTask{
		Name:     name,
		Template: c.addContainerDriverTemplate(),
		Arguments: wfapi.Arguments{
			Parameters: []wfapi.Parameter{
				{Name: paramComponent, Value: wfapi.AnyStringPtr(component)},
				{Name: paramTask, Value: wfapi.AnyStringPtr(task)},
				{Name: paramDAGContextID, Value: wfapi.AnyStringPtr(dagContextID)},
				{Name: paramDAGExecutionID, Value: wfapi.AnyStringPtr(dagExecutionID)},
			},
		},
	}, &containerDriverOutputs{}
}

func (c *workflowCompiler) addContainerDriverTemplate() string {
	name := "system-container-driver"
	_, ok := c.templates[name]
	if ok {
		return name
	}
	t := &wfapi.Template{
		Name: name,
		Inputs: wfapi.Inputs{
			Parameters: []wfapi.Parameter{
				{Name: paramComponent},
				{Name: paramTask},
				{Name: paramDAGContextID},
				{Name: paramDAGExecutionID},
			},
		},
		Outputs: wfapi.Outputs{
			Parameters: []wfapi.Parameter{
				{Name: paramExecutionID, ValueFrom: &wfapi.ValueFrom{Path: "/tmp/outputs/execution-id"}},
				{Name: paramExecutorInput, ValueFrom: &wfapi.ValueFrom{Path: "/tmp/outputs/executor-input"}},
			},
		},
		Container: &k8score.Container{
			Image:   c.driverImage,
			Command: []string{"/bin/kfp/driver"},
			Args: []string{
				"--type", "CONTAINER",
				"--pipeline_name", c.spec.GetPipelineInfo().GetName(),
				"--run_id", runID(),
				"--dag_context_id", inputValue(paramDAGContextID),
				"--dag_execution_id", inputValue(paramDAGExecutionID),
				"--component", inputValue(paramComponent),
				"--task", inputValue(paramTask),
				"--execution_id_path", outputPath(paramExecutionID),
				"--executor_input_path", outputPath(paramExecutorInput),
			},
		},
	}
	c.templates[name] = t
	c.wf.Spec.Templates = append(c.wf.Spec.Templates, *t)
	return name
}
