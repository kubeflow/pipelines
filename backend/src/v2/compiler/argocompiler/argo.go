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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/common/util"

	wfapi "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/compiler"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	k8score "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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
	// optional
	CacheDisabled bool
	// optional
	DefaultWorkspace *k8score.PersistentVolumeClaimSpec
	// optional, default workspace size from API server config
	DefaultWorkspaceSize string
	// TODO(Bobgy): add an option -- dev mode, ImagePullPolicy should only be Always in dev mode.
}

const (
	fallbackWorkspaceSize = "10Gi" // Used if no size is set in apiserver config but workspace is requested
	workspaceVolumeName   = "kfp-workspace"
)

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
	// fill root component default parameters to PipelineJob
	specParams := spec.GetRoot().GetInputDefinitions().GetParameters()
	for name, param := range specParams {
		_, ok := job.RuntimeConfig.ParameterValues[name]
		if !ok {
			if param.GetDefaultValue() != nil {
				job.RuntimeConfig.ParameterValues[name] = param.GetDefaultValue()
			} else if param.IsOptional {
				job.RuntimeConfig.ParameterValues[name] = structpb.NewNullValue()
			}
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
	var volumeClaimTemplates []k8score.PersistentVolumeClaim
	hasPipelineConfig := kubernetesSpec != nil && kubernetesSpec.GetPipelineConfig() != nil
	hasWorkspace := hasPipelineConfig && kubernetesSpec.GetPipelineConfig().GetWorkspace() != nil
	if hasWorkspace {
		workspace := kubernetesSpec.GetPipelineConfig().GetWorkspace()
		if k8sWorkspace := workspace.GetKubernetes(); k8sWorkspace != nil {
			pvc, err := GetWorkspacePVC(workspace, opts)
			if err != nil {
				return nil, err
			}
			volumeClaimTemplates = append(volumeClaimTemplates, pvc)
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
			Arguments: wfapi.Arguments{
				Parameters: []wfapi.Parameter{},
			},
			ServiceAccountName:   common.GetStringConfigWithDefault(common.DefaultPipelineRunnerServiceAccountFlag, common.DefaultPipelineRunnerServiceAccount),
			Entrypoint:           tmplEntrypoint,
			VolumeClaimTemplates: volumeClaimTemplates,
		},
	}

	runAsUser := GetPipelineRunAsUser()
	if runAsUser != nil {
		wf.Spec.SecurityContext = &k8score.PodSecurityContext{RunAsUser: runAsUser}
	}

	c := &workflowCompiler{
		wf:        wf,
		templates: make(map[string]*wfapi.Template),
		// TODO(chensun): release process and update the images.
		launcherImage:   GetLauncherImage(),
		launcherCommand: GetLauncherCommand(),
		driverImage:     GetDriverImage(),
		driverCommand:   GetDriverCommand(),
		job:             job,
		spec:            spec,
		executors:       deploy.GetExecutors(),
	}
	if opts != nil {
		c.cacheDisabled = opts.CacheDisabled
		c.defaultWorkspace = opts.DefaultWorkspace
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
	wf               *wfapi.Workflow
	templates        map[string]*wfapi.Template
	driverImage      string
	driverCommand    []string
	launcherImage    string
	launcherCommand  []string
	cacheDisabled    bool
	defaultWorkspace *k8score.PersistentVolumeClaimSpec
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
	argumentsComponents     = "components-"
	argumentsContainers     = "implementations-"
	argumentsKubernetesSpec = "kubernetes-"
)

func (c *workflowCompiler) saveComponentSpec(name string, spec *pipelinespec.ComponentSpec) error {
	hashedComponent := c.hashComponentContainer(name)

	return c.saveProtoToArguments(argumentsComponents+hashedComponent, spec)
}

// useComponentSpec returns a placeholder we can refer to the component spec
// in argo workflow fields.
func (c *workflowCompiler) useComponentSpec(name string) (string, error) {
	hashedComponent := c.hashComponentContainer(name)

	return c.argumentsPlaceholder(argumentsComponents + hashedComponent)
}

func (c *workflowCompiler) saveComponentImpl(name string, msg proto.Message) error {
	hashedComponent := c.hashComponentContainer(name)

	return c.saveProtoToArguments(argumentsContainers+hashedComponent, msg)
}

func (c *workflowCompiler) useComponentImpl(name string) (string, error) {
	hashedComponent := c.hashComponentContainer(name)

	return c.argumentsPlaceholder(argumentsContainers + hashedComponent)
}

func (c *workflowCompiler) saveKubernetesSpec(name string, spec *structpb.Struct) error {
	return c.saveProtoToArguments(argumentsKubernetesSpec+name, spec)
}

func (c *workflowCompiler) useKubernetesImpl(name string) (string, error) {
	return c.argumentsPlaceholder(argumentsKubernetesSpec + name)
}

// saveProtoToArguments saves a proto message to the workflow arguments. The
// message is serialized to JSON and stored in the workflow arguments and then
// referenced by the workflow templates using AWF templating syntax. The reason
// for storing it in the workflow arguments is because there is a 1-many
// relationship between components and tasks that reference them. The workflow
// arguments allow us to deduplicate the component logic (implementation & spec
// in IR), significantly reducing the size of the argo workflow manifest.
func (c *workflowCompiler) saveProtoToArguments(componentName string, msg proto.Message) error {
	if c == nil {
		return fmt.Errorf("compiler is nil")
	}
	if c.wf.Spec.Arguments.Parameters == nil {
		c.wf.Spec.Arguments = wfapi.Arguments{Parameters: []wfapi.Parameter{}}
	}
	if c.wf.Spec.Arguments.GetParameterByName(componentName) != nil {
		return nil
	}
	json, err := stablyMarshalJSON(msg)
	if err != nil {
		return fmt.Errorf("saving component spec of %q to arguments: %w", componentName, err)
	}
	c.wf.Spec.Arguments.Parameters = append(c.wf.Spec.Arguments.Parameters, wfapi.Parameter{
		Name:  componentName,
		Value: wfapi.AnyStringPtr(json),
	})
	return nil
}

// argumentsPlaceholder checks for the unique component name within the workflow
// arguments and returns a template tag that references the component in the
// workflow arguments.
func (c *workflowCompiler) argumentsPlaceholder(componentName string) (string, error) {
	if c == nil {
		return "", fmt.Errorf("compiler is nil")
	}
	if c.wf.Spec.Arguments.GetParameterByName(componentName) == nil {
		return "", fmt.Errorf("using component spec: failed to find workflow parameter %q", componentName)
	}

	return workflowParameter(componentName), nil
}

// hashComponentContainer serializes and hashes the container field of a given
// component.
func (c *workflowCompiler) hashComponentContainer(componentName string) string {
	log.Debug("componentName: ", componentName)
	// Return early for root component since it has no command and args.
	if componentName == "root" {
		return componentName
	}
	if c.executors != nil { // Don't bother if there are no executors in the pipeline spec.
		// Look up the executorLabel for the component in question.
		executorLabel := c.spec.Components[componentName].GetExecutorLabel()
		log.Debug("executorLabel: ", executorLabel)
		// Iterate through the list of executors.
		for executorName, executorValue := range c.executors {
			log.Debug("executorName: ", executorName)
			// If one of them matches the executorLabel we extracted earlier...
			if executorName == executorLabel {
				// Get the corresponding container.
				container := executorValue.GetContainer()
				if container != nil {
					containerHash, err := hashValue(container)
					if err != nil {
						// Do not bubble up since this is not a breaking error
						// and we can just return the componentName in full.
						log.Debug("Error hashing container: ", err)
					}

					return containerHash
				}
			}
		}
	}

	return componentName
}

// hashValue serializes and hashes a provided value.
func hashValue(value interface{}) (string, error) {
	bytes, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	h.Write([]byte(bytes))

	return hex.EncodeToString(h.Sum(nil)), nil
}

const (
	paramComponent               = "component"      // component spec
	paramTask                    = "task"           // task spec
	paramContainer               = "container"      // container spec
	paramImporter                = "importer"       // importer spec
	paramRuntimeConfig           = "runtime-config" // job runtime config, pipeline level inputs
	paramParentDagID             = "parent-dag-id"
	paramExecutionID             = "execution-id"
	paramIterationCount          = "iteration-count"
	paramIterationIndex          = "iteration-index"
	paramExecutorInput           = "executor-input"
	paramDriverType              = "driver-type"
	paramCachedDecision          = "cached-decision"            // indicate hit cache or not
	paramPodSpecPatch            = "pod-spec-patch"             // a strategic patch merged with the pod spec
	paramCondition               = "condition"                  // condition = false -> skip the task
	paramKubernetesConfig        = "kubernetes-config"          // stores Kubernetes config
	paramRetryMaxCount           = "retry-max-count"            // limit on number of retries
	paramRetryBackOffDuration    = "retry-backoff-duration"     // duration of backoff between retries
	paramRetryBackOffFactor      = "retry-backoff-factor"       // multiplier for backoff duration between retries
	paramRetryBackOffMaxDuration = "retry-backoff-max-duration" // limit on backoff duration between retries
)

func runID() string {
	// KFP API server converts this to KFP run ID.
	return "{{workflow.uid}}"
}

func runResourceName() string {
	// This translates to the Argo Workflow object name.
	return "{{workflow.name}}"
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

// convertStructToPVCSpec converts a protobuf Struct to a PersistentVolumeClaimSpec using JSON marshaling/unmarshalling.
func convertStructToPVCSpec(structVal *structpb.Struct) (*k8score.PersistentVolumeClaimSpec, error) {
	if structVal == nil {
		return nil, nil
	}
	jsonBytes, err := structVal.MarshalJSON()
	if err != nil {
		return nil, err
	}

	// Strict validation: check for unknown fields
	var patchMap map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &patchMap); err != nil {
		return nil, err
	}
	// List of allowed fields in PersistentVolumeClaimSpec
	allowed := map[string]struct{}{
		"accessModes":      {},
		"storageClassName": {},
	}
	var unknown []string
	for k := range patchMap {
		if _, ok := allowed[k]; !ok {
			unknown = append(unknown, k)
		}
	}
	if len(unknown) > 0 {
		return nil, fmt.Errorf("unknown fields in PVC spec patch: %v", unknown)
	}

	var pvcSpec k8score.PersistentVolumeClaimSpec
	if err := json.Unmarshal(jsonBytes, &pvcSpec); err != nil {
		return nil, err
	}
	return &pvcSpec, nil
}

// GetWorkspacePVC constructs a PersistentVolumeClaim for the workspace.
// It uses the default PVC spec (from API server config), and applies any user-specified
// overrides from the pipeline spec, including the requested storage size.
func GetWorkspacePVC(
	workspace *pipelinespec.WorkspaceConfig,
	opts *Options,
) (k8score.PersistentVolumeClaim, error) {
	sizeStr := workspace.GetSize()
	if sizeStr == "" && opts != nil && opts.DefaultWorkspaceSize != "" {
		sizeStr = opts.DefaultWorkspaceSize
	}
	if sizeStr == "" {
		sizeStr = fallbackWorkspaceSize
	}

	k8sWorkspace := workspace.GetKubernetes()
	var pvcSpec k8score.PersistentVolumeClaimSpec
	// If no default workspace spec is configured, users can still use workspaces.
	// This allows workspace usage without requiring default configuration.
	if opts != nil && opts.DefaultWorkspace != nil {
		pvcSpec = *opts.DefaultWorkspace.DeepCopy()
	}
	if k8sWorkspace != nil {
		if k8sWorkspacePVCSpec := k8sWorkspace.GetPvcSpecPatch(); k8sWorkspacePVCSpec != nil {
			userPatch, err := convertStructToPVCSpec(k8sWorkspacePVCSpec)
			if err != nil {
				return k8score.PersistentVolumeClaim{}, err
			}
			pvcSpec = *util.PatchPVCSpec(&pvcSpec, userPatch)
		}
	}

	quantity, err := resource.ParseQuantity(sizeStr)
	if err != nil {
		return k8score.PersistentVolumeClaim{}, fmt.Errorf("invalid size value for workspace PVC: %v", err)
	}
	if quantity.Sign() < 0 {
		return k8score.PersistentVolumeClaim{}, fmt.Errorf("negative size value for workspace PVC: %v", sizeStr)
	}
	if pvcSpec.Resources.Requests == nil {
		pvcSpec.Resources.Requests = make(map[k8score.ResourceName]resource.Quantity)
	}
	pvcSpec.Resources.Requests[k8score.ResourceStorage] = quantity

	return k8score.PersistentVolumeClaim{
		ObjectMeta: k8smeta.ObjectMeta{
			Name: workspaceVolumeName,
		},
		Spec: pvcSpec,
	}, nil
}
