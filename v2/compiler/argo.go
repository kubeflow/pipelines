package compiler

import (
	"fmt"
	"strings"

	wfapi "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Options struct {
	// optional, use official image if not provided
	LauncherImage string
	// optional
	DriverImage string
	// optional
	PipelineRoot string
	// TODO(Bobgy): add an option -- dev mode, ImagePullPolicy should only be Always in dev mode.
}

func Compile(jobArg *pipelinespec.PipelineJob, opts *Options) (*wfapi.Workflow, error) {
	// clone jobArg, because we don't want to change it
	jobMsg := proto.Clone(jobArg)
	job, ok := jobMsg.(*pipelinespec.PipelineJob)
	if !ok {
		return nil, fmt.Errorf("bug: cloned pipeline job message does not have expected type")
	}
	if job.RuntimeConfig == nil {
		job.RuntimeConfig = &pipelinespec.PipelineJob_RuntimeConfig{}
	}
	if job.GetRuntimeConfig().GetParameterValues() == nil {
		job.RuntimeConfig.ParameterValues = map[string]*structpb.Value{}
	}
	spec, err := getPipelineSpec(job)
	if err != nil {
		return nil, err
	}
	// validation
	if spec.GetPipelineInfo().GetName() == "" {
		return nil, fmt.Errorf("pipelineInfo.name is empty")
	}
	// fill root component default paramters to PipelineJob
	specParams := spec.GetRoot().GetInputDefinitions().GetParameters()
	if specParams != nil {
		for name, param := range specParams {
			_, ok := job.RuntimeConfig.ParameterValues[name]
			if !ok && param.GetDefaultValue() != nil {
				job.RuntimeConfig.ParameterValues[name] = param.GetDefaultValue()
			}
		}
	}
	// initialization
	wf := &wfapi.Workflow{
		TypeMeta: k8smeta.TypeMeta{
			APIVersion: "argoproj.io/v1alpha1",
			Kind:       "Workflow",
		},
		ObjectMeta: k8smeta.ObjectMeta{
			GenerateName: retrieveLastValidString(spec.GetPipelineInfo().GetName()) + "-",
			// Note, uncomment the following during development to view argo inputs/outputs in KFP UI.
			Annotations: map[string]string{
				"pipelines.kubeflow.org/v2_pipeline": "true",
			},
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
		wf:        wf,
		templates: make(map[string]*wfapi.Template),
		// TODO(Bobgy): release process and update the images.
		driverImage:   "gcr.io/ml-pipeline/kfp-driver:latest",
		launcherImage: "gcr.io/ml-pipeline/kfp-launcher-v2:latest",
		job:           job,
		spec:          spec,
	}
	if opts != nil {
		if opts.DriverImage != "" {
			compiler.driverImage = opts.DriverImage
		}
		if opts.LauncherImage != "" {
			compiler.launcherImage = opts.LauncherImage
		}
		if opts.PipelineRoot != "" {
			job.RuntimeConfig.GcsOutputDirectory = opts.PipelineRoot
		}
	}

	// compile
	err = Accept(job, compiler)

	return compiler.wf, err
}

func retrieveLastValidString(s string) string {
	sections := strings.Split(s, "/")
	return sections[len(sections)-1]
}

type workflowCompiler struct {
	// inputs
	job  *pipelinespec.PipelineJob
	spec *pipelinespec.PipelineSpec
	// state
	wf            *wfapi.Workflow
	templates     map[string]*wfapi.Template
	driverImage   string
	launcherImage string
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
	paramContainer      = "container"      // container spec
	paramImporter       = "importer"       // importer spec
	paramRuntimeConfig  = "runtime-config" // job runtime config, pipeline level inputs
	paramDAGExecutionID = "dag-execution-id"
	paramExecutionID    = "execution-id"
	paramIterationCount = "iteration-count"
	paramIterationIndex = "iteration-index"
	paramExecutorInput  = "executor-input"
	paramDriverType     = "driver-type"
	paramCachedDecision = "cached-decision" // indicate hit cache or not
)

func runID() string {
	// KFP API server converts this to KFP run ID.
	return "{{workflow.uid}}"
}

func workflowParameter(name string) string {
	return fmt.Sprintf("{{workflow.parameters.%s}}", name)
}

// In a container template, refer to inputs to the template.
func inputValue(parameter string) string {
	return fmt.Sprintf("{{inputs.parameters.%s}}", parameter)
}

// In a DAG/steps template, refer to inputs to the parent template.
func inputParameter(parameter string) string {
	return fmt.Sprintf("{{inputs.parameters.%s}}", parameter)
}

func outputPath(parameter string) string {
	return fmt.Sprintf("{{outputs.parameters.%s.path}}", parameter)
}

func taskOutputParameter(task string, param string) string {
	return fmt.Sprintf("{{tasks.%s.outputs.parameters.%s}}", task, param)
}

func loopItem() string {
	// https://github.com/argoproj/argo-workflows/blob/13bf15309567ff10ec23b6e5cfed846312d6a1ab/examples/loops-sequence.yaml#L20
	return "{{item}}"
}
