package compiler

import (
	"fmt"

	wfapi "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
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
			// TODO(Bobgy): uncomment the following. Temporarily commented for development purposes.
			// Annotations: map[string]string{
			// 	"pipelines.kubeflow.org/v2_pipeline": "true",
			// },
		},
		Spec: wfapi.WorkflowSpec{
			PodMetadata: &wfapi.Metadata{
				Annotations: map[string]string{
					"pipelines.kubeflow.org/v2_component": "true",
				},
				Labels: map[string]string{
					"pipelines.kubeflow.org/v2_component": "true",
				},
			},
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

func (c *workflowCompiler) Importer(name string, component *pipelinespec.ComponentSpec, importer *pipelinespec.PipelineDeploymentConfig_ImporterSpec) error {
	return fmt.Errorf("importer not implemented yet")
}
func (c *workflowCompiler) Resolver(name string, component *pipelinespec.ComponentSpec, resolver *pipelinespec.PipelineDeploymentConfig_ResolverSpec) error {
	return fmt.Errorf("resolver not implemented yet")
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

func (c *workflowCompiler) templateName(componentName string) string {
	// TODO(Bobgy): sanitize component name, because argo template names
	// must be valid Kubernetes resource names.
	return componentName
}

const (
	paramComponent      = "component"      // component spec
	paramTask           = "task"           // task spec
	paramRuntimeConfig  = "runtime-config" // job runtime config, pipeline level inputs
	paramDAGContextID   = "dag-context-id"
	paramDAGExecutionID = "dag-execution-id"
	paramExecutionID    = "execution-id"
	paramContextID      = "context-id"
	paramExecutorInput  = "executor-input"
)

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

func inputParameter(parameter string) string {
	return fmt.Sprintf("{{inputs.parameters.%s}}", parameter)
}

func outputPath(parameter string) string {
	return fmt.Sprintf("{{outputs.parameters.%s.path}}", parameter)
}

func taskOutputParameter(task string, param string) string {
	return fmt.Sprintf("{{tasks.%s.outputs.parameters.%s}}", task, param)
}
