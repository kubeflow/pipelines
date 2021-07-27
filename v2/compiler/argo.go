package compiler

import (
	"fmt"

	wfapi "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	k8score "k8s.io/api/core/v1"
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
	compiler := &workflowCompiler{}
	compiler.wf = &wfapi.Workflow{}
	compiler.templates = make(map[string]*wfapi.Template)
	wf := compiler.wf
	wf.APIVersion = "argoproj.io/v1alpha1"
	wf.Kind = "Workflow"
	wf.GenerateName = spec.GetPipelineInfo().GetName() + "-"
	wf.Spec.ServiceAccountName = "pipeline-runner"
	wf.Spec.Entrypoint = rootComponentName

	// compile
	Accept(job, compiler)

	return compiler.wf, nil
}

type workflowCompiler struct {
	wf        *wfapi.Workflow
	templates map[string]*wfapi.Template
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
	return c.addTemplate(t, name)
}
func (c *workflowCompiler) Importer(name string, component *pipelinespec.ComponentSpec, importer *pipelinespec.PipelineDeploymentConfig_ImporterSpec) error {
	return nil
}
func (c *workflowCompiler) Resolver(name string, component *pipelinespec.ComponentSpec, resolver *pipelinespec.PipelineDeploymentConfig_ResolverSpec) error {
	return nil
}
func (c *workflowCompiler) DAG(name string, component *pipelinespec.ComponentSpec, dag *pipelinespec.DagSpec) error {
	t := &wfapi.Template{}
	t.DAG = &wfapi.DAGTemplate{}
	for _, kfpTask := range dag.GetTasks() {
		task := wfapi.DAGTask{}
		componentRef := kfpTask.GetComponentRef().GetName()
		task.Name = kfpTask.GetTaskInfo().GetName()
		task.Template = c.templateName(componentRef)
		t.DAG.Tasks = append(t.DAG.Tasks, task)
	}
	return c.addTemplate(t, name)
}

func (c *workflowCompiler) addTemplate(t *wfapi.Template, name string) error {
	t.Name = c.templateName(name)
	c.wf.Spec.Templates = append(c.wf.Spec.Templates, *t)
	_, ok := c.templates[t.Name]
	if ok {
		return fmt.Errorf("template name=%q added more than once", t.Name)
	}
	c.templates[t.Name] = t
	return nil
}

func (c *workflowCompiler) templateName(componentName string) string {
	// TODO(Bobgy): sanitize component name, because argo template names
	// must be valid Kubernetes resource names.
	return componentName
}
