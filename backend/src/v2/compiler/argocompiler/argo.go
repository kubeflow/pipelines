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

package argocompiler

import (
	"fmt"
	"strings"

	wfapi "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/compiler"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	k8score "k8s.io/api/core/v1"
	k8sres "k8s.io/apimachinery/pkg/api/resource"
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

func Compile(jobArg *pipelinespec.PipelineJob, kubernetesSpecArg *pipelinespec.SinglePlatformSpec, opts *Options) (*wfapi.Workflow, error) {
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
	spec, err := compiler.GetPipelineSpec(job)
	if err != nil {
		return nil, err
	}
	// validation
	if spec.GetPipelineInfo().GetName() == "" {
		return nil, fmt.Errorf("pipelineInfo.name is empty")
	}
	deploy, err := compiler.GetDeploymentConfig(spec)
	if err != nil {
		return nil, err
	}
	// fill root component default paramters to PipelineJob
	specParams := spec.GetRoot().GetInputDefinitions().GetParameters()
	for name, param := range specParams {
		_, ok := job.RuntimeConfig.ParameterValues[name]
		if !ok && param.GetDefaultValue() != nil {
			job.RuntimeConfig.ParameterValues[name] = param.GetDefaultValue()
		}
	}

	var kubernetesSpec *pipelinespec.SinglePlatformSpec
	if kubernetesSpecArg != nil {
		// clone kubernetesSpecArg, because we don't want to change it
		kubernetesSpecMsg := proto.Clone(kubernetesSpecArg)
		kubernetesSpec, ok = kubernetesSpecMsg.(*pipelinespec.SinglePlatformSpec)
		if !ok {
			return nil, fmt.Errorf("bug: cloned Kubernetes spec message does not have expected type")
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
			// TODO(Bobgy): figure out what annotations we should use for v2 engine.
			// For now, comment this annotation, so that in KFP UI, it shows argo input/output params/artifacts
			// suitable for debugging.
			//
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
			Entrypoint:         tmplEntrypoint,
		},
	}
	c := &workflowCompiler{
		wf:        wf,
		templates: make(map[string]*wfapi.Template),
		// TODO(chensun): release process and update the images.
		launcherImage: GetLauncherImage(),
		driverImage:   GetDriverImage(),
		job:           job,
		spec:          spec,
		executors:     deploy.GetExecutors(),
	}

	mlPipelineTLSEnabled, err := GetMLPipelineServiceTLSEnabled()
	if err != nil {
		return nil, err
	}
	c.mlPipelineServiceTLSEnabled = mlPipelineTLSEnabled

	if opts != nil {
		if opts.DriverImage != "" {
			c.driverImage = opts.DriverImage
		}
		if opts.LauncherImage != "" {
			c.launcherImage = opts.LauncherImage
		}
		if opts.PipelineRoot != "" {
			job.RuntimeConfig.GcsOutputDirectory = opts.PipelineRoot
		}
	}

	// compile
	err = compiler.Accept(job, kubernetesSpec, c)

	return c.wf, err
}

func retrieveLastValidString(s string) string {
	sections := strings.Split(s, "/")
	return sections[len(sections)-1]
}

type workflowCompiler struct {
	// inputs
	job       *pipelinespec.PipelineJob
	spec      *pipelinespec.PipelineSpec
	executors map[string]*pipelinespec.PipelineDeploymentConfig_ExecutorSpec
	// state
	wf                          *wfapi.Workflow
	templates                   map[string]*wfapi.Template
	driverImage                 string
	launcherImage               string
	mlPipelineServiceTLSEnabled bool
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

// WIP: store component spec, task spec and executor spec in annotations

const (
	annotationComponents     = "pipelines.kubeflow.org/components-"
	annotationContainers     = "pipelines.kubeflow.org/implementations-"
	annotationKubernetesSpec = "pipelines.kubeflow.org/kubernetes-"
)

func (c *workflowCompiler) saveComponentSpec(name string, spec *pipelinespec.ComponentSpec) error {
	return c.saveProtoToAnnotation(annotationComponents+name, spec)
}

// useComponentSpec returns a placeholder we can refer to the component spec
// in argo workflow fields.
func (c *workflowCompiler) useComponentSpec(name string) (string, error) {
	return c.annotationPlaceholder(annotationComponents + name)
}

func (c *workflowCompiler) saveComponentImpl(name string, msg proto.Message) error {
	return c.saveProtoToAnnotation(annotationContainers+name, msg)
}

func (c *workflowCompiler) useComponentImpl(name string) (string, error) {
	return c.annotationPlaceholder(annotationContainers + name)
}

func (c *workflowCompiler) saveKubernetesSpec(name string, spec *structpb.Struct) error {
	return c.saveProtoToAnnotation(annotationKubernetesSpec+name, spec)
}

func (c *workflowCompiler) useKubernetesImpl(name string) (string, error) {
	return c.annotationPlaceholder(annotationKubernetesSpec + name)
}

// TODO(Bobgy): sanitize component name
func (c *workflowCompiler) saveProtoToAnnotation(name string, msg proto.Message) error {
	if c == nil {
		return fmt.Errorf("compiler is nil")
	}
	if c.wf.Annotations == nil {
		c.wf.Annotations = make(map[string]string)
	}
	if _, alreadyExists := c.wf.Annotations[name]; alreadyExists {
		return fmt.Errorf("annotation %q already exists", name)
	}
	json, err := stablyMarshalJSON(msg)
	if err != nil {
		return fmt.Errorf("saving component spec of %q to annotations: %w", name, err)
	}
	// TODO(Bobgy): verify name adheres to Kubernetes annotation restrictions: https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/#syntax-and-character-set
	c.wf.Annotations[name] = json
	return nil
}

func (c *workflowCompiler) annotationPlaceholder(name string) (string, error) {
	if c == nil {
		return "", fmt.Errorf("compiler is nil")
	}
	if _, exists := c.wf.Annotations[name]; !exists {
		return "", fmt.Errorf("using component spec: failed to find annotation %q", name)
	}
	// Reference: https://argoproj.github.io/argo-workflows/variables/
	return fmt.Sprintf("{{workflow.annotations.%s}}", name), nil
}

const (
	paramComponent        = "component"      // component spec
	paramTask             = "task"           // task spec
	paramContainer        = "container"      // container spec
	paramImporter         = "importer"       // importer spec
	paramRuntimeConfig    = "runtime-config" // job runtime config, pipeline level inputs
	paramParentDagID      = "parent-dag-id"
	paramExecutionID      = "execution-id"
	paramIterationCount   = "iteration-count"
	paramIterationIndex   = "iteration-index"
	paramExecutorInput    = "executor-input"
	paramDriverType       = "driver-type"
	paramCachedDecision   = "cached-decision"   // indicate hit cache or not
	paramPodSpecPatch     = "pod-spec-patch"    // a strategic patch merged with the pod spec
	paramCondition        = "condition"         // condition = false -> skip the task
	paramKubernetesConfig = "kubernetes-config" // stores Kubernetes config
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

// Usually drivers should take very minimal amount of CPU and memory, but we
// set a larger limit for extreme cases.
// Note, these are empirical data.
// No need to make this configurable, because we will instead call drivers using argo HTTP templates later.
var driverResources = k8score.ResourceRequirements{
	Limits: map[k8score.ResourceName]k8sres.Quantity{
		k8score.ResourceMemory: k8sres.MustParse("0.5Gi"),
		k8score.ResourceCPU:    k8sres.MustParse("0.5"),
	},
	Requests: map[k8score.ResourceName]k8sres.Quantity{
		k8score.ResourceMemory: k8sres.MustParse("64Mi"),
		k8score.ResourceCPU:    k8sres.MustParse("0.1"),
	},
}

// Launcher only copies the binary into the volume, so it needs minimal resources.
var launcherResources = k8score.ResourceRequirements{
	Limits: map[k8score.ResourceName]k8sres.Quantity{
		k8score.ResourceMemory: k8sres.MustParse("128Mi"),
		k8score.ResourceCPU:    k8sres.MustParse("0.5"),
	},
	Requests: map[k8score.ResourceName]k8sres.Quantity{
		k8score.ResourceCPU: k8sres.MustParse("0.1"),
	},
}

const (
	tmplEntrypoint = "entrypoint"
)

// Here is the collection of all special dummy images that the backend recognizes.
// User need to avoid these image names for their self-defined components.
// These values are in sync with the values in SDK to form a contract between BE and SDK.
// TODO(lingqinggan): clarify these in documentation for KFP V2.
var dummyImages = map[string]bool{
	"argostub/createpvc": true,
	"argostub/deletepvc": true,
}
