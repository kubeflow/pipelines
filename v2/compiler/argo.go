package compiler

import (
	"fmt"

	wfapi "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/golang/protobuf/jsonpb"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	k8score "k8s.io/api/core/v1"
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Compile(job *pipelinespec.PipelineJob) (*wfapi.Workflow, error) {
	spec, err := getPipelineSpec(job)
	if err != nil {
		return nil, err
	}
	// validation
	if spec.GetPipelineInfo().GetName() == "" {
		return nil, fmt.Errorf("pipelineInfo.name is empty")
	}

	// initialization
	wf := &wfapi.Workflow{
		TypeMeta: k8smeta.TypeMeta{
			APIVersion: "argoproj.io/v1alpha1",
			Kind:       "Workflow",
		},
		ObjectMeta: k8smeta.ObjectMeta{
			GenerateName: spec.GetPipelineInfo().GetName() + "-",
		},
		Spec: wfapi.WorkflowSpec{
			ServiceAccountName: "pipeline-runner",
			Entrypoint:         rootComponentName,
		},
	}
	compiler := &workflowCompiler{
		wf:          wf,
		templates:   make(map[string]*wfapi.Template),
		driverImage: "gcr.io/gongyuan-dev/dev/kfp-driver:latest",
		job:         job,
		spec:        spec,
	}

	// compile
	Accept(job, compiler)

	return compiler.wf, nil
}

type workflowCompiler struct {
	// inputs
	job  *pipelinespec.PipelineJob
	spec *pipelinespec.PipelineSpec
	// state
	wf          *wfapi.Workflow
	templates   map[string]*wfapi.Template
	driverImage string
}

func (c *workflowCompiler) Container(name string, component *pipelinespec.ComponentSpec, container *pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec) error {
	// TODO(Bobgy): sanitize template names
	t := &wfapi.Template{}
	t.Container = &k8score.Container{
		Command: container.Command,
		Args:    container.Args,
		Image:   container.Image,
		// TODO(Bobgy): support resource requests/limits
	}
	_, err := c.addTemplate(t, name)
	return err
}
func (c *workflowCompiler) Importer(name string, component *pipelinespec.ComponentSpec, importer *pipelinespec.PipelineDeploymentConfig_ImporterSpec) error {
	return nil
}
func (c *workflowCompiler) Resolver(name string, component *pipelinespec.ComponentSpec, resolver *pipelinespec.PipelineDeploymentConfig_ResolverSpec) error {
	return nil
}
func (c *workflowCompiler) DAG(name string, componentSpec *pipelinespec.ComponentSpec, dagSpec *pipelinespec.DagSpec) error {
	dag := &wfapi.Template{}
	dag.DAG = &wfapi.DAGTemplate{}
	for _, kfpTask := range dagSpec.GetTasks() {
		task := wfapi.DAGTask{}
		componentRef := kfpTask.GetComponentRef().GetName()
		task.Name = kfpTask.GetTaskInfo().GetName()
		task.Template = c.templateName(componentRef)
		dag.DAG.Tasks = append(dag.DAG.Tasks, task)
	}
	// TODO(Bobgy): how can we avoid template name collisions?
	dagName, err := c.addTemplate(dag, name+"-dag")
	wrapper := &wfapi.Template{}
	task := &pipelinespec.PipelineTaskSpec{}
	driverTask, err := c.dagDriverTask(componentSpec, task)
	if err != nil {
		return err
	}
	driverTask.Name = "driver"
	wrapper.DAG = &wfapi.DAGTemplate{
		Tasks: []wfapi.DAGTask{
			*driverTask,
			{Name: "dag", Template: dagName, Dependencies: []string{"driver"}},
		},
	}
	c.addTemplate(wrapper, name)
	return err
}

var errAlreadyExists = fmt.Errorf("template already exists")

func (c *workflowCompiler) addTemplate(t *wfapi.Template, name string) (string, error) {
	t.Name = c.templateName(name)
	_, ok := c.templates[t.Name]
	if ok {
		return "", fmt.Errorf("template name=%q: %w", t.Name, errAlreadyExists)
	}
	c.wf.Spec.Templates = append(c.wf.Spec.Templates, *t)
	c.templates[t.Name] = t
	return t.Name, nil
}

const (
	paramComponent   = "component" // component spec
	paramTask        = "task"      // task spec
	paramExecutionID = "execution-id"
	paramContextID   = "context-id"
)

func (c *workflowCompiler) dagDriverTask(component *pipelinespec.ComponentSpec, task *pipelinespec.PipelineTaskSpec) (*wfapi.DAGTask, error) {
	if component == nil || task == nil {
		return nil, fmt.Errorf("dagDriverTask: component and task must be non-nil")
	}
	marshaler := jsonpb.Marshaler{}
	componentJson, err := marshaler.MarshalToString(component)
	if err != nil {
		return nil, fmt.Errorf("dagDriverTask: marlshaling component spec to proto JSON failed: %w", err)
	}
	taskJson, err := marshaler.MarshalToString(task)
	if err != nil {
		return nil, fmt.Errorf("dagDriverTask: marshaling task spec to proto JSON failed: %w", err)
	}
	templateName := c.addDAGDriverTemplate()
	t := &wfapi.DAGTask{
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
			},
		},
	}
	return t, nil
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
				{Name: paramTask},
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
			Command: []string{"/bin/kfp/driver"},
			Args: []string{
				"--pipeline_name", c.spec.GetPipelineInfo().GetName(),
				"--run_id", runID(),
				"--component", inputValue(paramComponent),
				"--task", inputValue(paramTask),
				"--execution_id_path", outputPath(paramExecutionID),
				"--context_id_path", outputPath(paramContextID),
			},
		},
	}
	c.templates[name] = t
	c.wf.Spec.Templates = append(c.wf.Spec.Templates, *t)
	return name
}

func runID() string {
	// KFP API server converts this to KFP run ID.
	return "{{workflow.uid}}"
}

func workflowParameter(name string) string {
	return fmt.Sprintf("{{workflow.parameters.%s}}", name)
}

func inputValue(parameter string) string {
	return fmt.Sprintf("{{inputs.parameters.%s}}", parameter)
}

func outputPath(parameter string) string {
	return fmt.Sprintf("{{outputs.parameters.%s.path}}", parameter)
}

func (c *workflowCompiler) templateName(componentName string) string {
	// TODO(Bobgy): sanitize component name, because argo template names
	// must be valid Kubernetes resource names.
	return componentName
}
