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
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/kubeflow/pipelines/backend/src/v2/config"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"
	"k8s.io/apimachinery/pkg/util/intstr"

	wfapi "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/v2/component"
	"github.com/kubeflow/pipelines/kubernetes_platform/go/kubernetesplatform"
	k8score "k8s.io/api/core/v1"
)

const (
	volumeNameKFPLauncher = "kfp-launcher"
	LauncherImageEnvVar   = "V2_LAUNCHER_IMAGE"
	// DefaultLauncherImage & DefaultDriverImage are set as latest here
	// but are overridden by environment variables set via k8s manifests.
	// For releases, the manifest will have the correct release version set.
	// this is to avoid hardcoding releases in code here.
	DefaultLauncherImage     = "ghcr.io/kubeflow/kfp-launcher:latest"
	LauncherCommandEnvVar    = "V2_LAUNCHER_COMMAND"
	DefaultLauncherCommand   = "launcher-v2"
	PipelineRunAsUserEnvVar  = "PIPELINE_RUN_AS_USER"
	PipelineLogLevelEnvVar   = "PIPELINE_LOG_LEVEL"
	PublishLogsEnvVar        = "PUBLISH_LOGS"
	gcsScratchLocation       = "/gcs"
	gcsScratchName           = "gcs-scratch"
	s3ScratchLocation        = "/s3"
	s3ScratchName            = "s3-scratch"
	minioScratchLocation     = "/minio"
	minioScratchName         = "minio-scratch"
	dotLocalScratchLocation  = "/.local"
	dotLocalScratchName      = "dot-local-scratch"
	dotCacheScratchLocation  = "/.cache"
	dotCacheScratchName      = "dot-cache-scratch"
	dotConfigScratchLocation = "/.config"
	dotConfigScratchName     = "dot-config-scratch"
)

func (c *workflowCompiler) Container(name string, component *pipelinespec.ComponentSpec, container *pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec) error {
	err := c.saveComponentSpec(name, component)
	if err != nil {
		return err
	}
	err = c.saveComponentImpl(name, container)
	if err != nil {
		return err
	}
	return nil
}

type containerDriverOutputs struct {
	podSpecPatch string
	cached       string
	condition    string
}

type containerDriverInputs struct {
	component        string
	task             string
	taskName         string // preserve the original task name for input resolving
	container        string
	parentDagID      string
	iterationIndex   string // optional, when this is an iteration task
	kubernetesConfig string // optional, used when Kubernetes config is not empty
}

func GetLauncherImage() string {
	launcherImage := os.Getenv(LauncherImageEnvVar)
	if launcherImage == "" {
		launcherImage = DefaultLauncherImage
	}
	return launcherImage
}

func GetLauncherCommand() []string {
	launcherCommand := os.Getenv(LauncherCommandEnvVar)
	if launcherCommand == "" {
		launcherCommand = DefaultLauncherCommand
	}
	return strings.Split(launcherCommand, " ")
}

func GetPipelineRunAsUser() *int64 {
	runAsUserStr := os.Getenv(PipelineRunAsUserEnvVar)
	if runAsUserStr == "" {
		return nil
	}

	runAsUser, err := strconv.ParseInt(runAsUserStr, 10, 64)
	if err != nil {
		glog.Error(
			"Failed to parse the %s environment variable with value %s as an int64: %v",
			PipelineRunAsUserEnvVar, runAsUserStr, err,
		)

		return nil
	}

	return &runAsUser
}

func (c *workflowCompiler) containerDriverTask(name string, inputs containerDriverInputs) (*wfapi.DAGTask, *containerDriverOutputs, error) {
	template, err := c.addContainerDriverTemplate()
	if err != nil {
		return nil, nil, err
	}
	dagTask := &wfapi.DAGTask{
		Name:     name,
		Template: template,
		Arguments: wfapi.Arguments{
			Parameters: []wfapi.Parameter{
				{Name: paramComponent, Value: wfapi.AnyStringPtr(inputs.component)},
				{Name: paramTask, Value: wfapi.AnyStringPtr(inputs.task)},
				{Name: paramContainer, Value: wfapi.AnyStringPtr(inputs.container)},
				{Name: paramTaskName, Value: wfapi.AnyStringPtr(inputs.taskName)},
				{Name: paramParentDagID, Value: wfapi.AnyStringPtr(inputs.parentDagID)},
			},
		},
	}
	if inputs.iterationIndex != "" {
		dagTask.Arguments.Parameters = append(
			dagTask.Arguments.Parameters,
			wfapi.Parameter{Name: paramIterationIndex, Value: wfapi.AnyStringPtr(inputs.iterationIndex)},
		)
	}
	if inputs.kubernetesConfig != "" {
		dagTask.Arguments.Parameters = append(
			dagTask.Arguments.Parameters,
			wfapi.Parameter{Name: paramKubernetesConfig, Value: wfapi.AnyStringPtr(inputs.kubernetesConfig)},
		)
	}
	outputs := &containerDriverOutputs{
		podSpecPatch: taskOutputParameter(name, paramPodSpecPatch),
		cached:       taskOutputParameter(name, paramCachedDecision),
		condition:    taskOutputParameter(name, paramCondition),
	}
	return dagTask, outputs, nil
}

// Create the Argo Workflow executor plugin template for the container driver.
// See https://argo-workflows.readthedocs.io/en/latest/executor_plugins/
func (c *workflowCompiler) addContainerDriverTemplate() (string, error) {
	name := "system-container-driver"
	_, ok := c.templates[name]
	if ok {
		return name, nil
	}

	args := map[string]interface{}{
		"type":                       "CONTAINER",
		"pipeline_name":              c.spec.GetPipelineInfo().GetName(),
		"run_id":                     runID(),
		"run_name":                   runResourceName(),
		"run_display_name":           c.job.DisplayName,
		"dag_execution_id":           inputValue(paramParentDagID),
		"component":                  inputValue(paramComponent),
		"task":                       inputValue(paramTask),
		"task_name":                  inputValue(paramTaskName),
		"container":                  inputValue(paramContainer),
		"iteration_index":            inputValue(paramIterationIndex),
		"cached_decision_path":       outputPath(paramCachedDecision),
		"pod_spec_patch_path":        outputPath(paramPodSpecPatch),
		"condition_path":             outputPath(paramCondition),
		"kubernetes_config":          inputValue(paramKubernetesConfig),
		"http_proxy":                 proxy.GetConfig().GetHttpProxy(),
		"https_proxy":                proxy.GetConfig().GetHttpsProxy(),
		"no_proxy":                   proxy.GetConfig().GetNoProxy(),
		"ml_pipeline_server_address": config.GetMLPipelineServerConfig().Address,
		"ml_pipeline_server_port":    config.GetMLPipelineServerConfig().Port,
		"mlmd_server_address":        metadata.GetMetadataConfig().Address,
		"mlmd_server_port":           metadata.GetMetadataConfig().Port,
		"cache_disabled":             c.cacheDisabled,
		"ml_pipeline_tls_enabled":    c.mlPipelineTLSEnabled,
		"metadata_tls_enabled":       common.GetMetadataTLSEnabled(),
	}

	// If CABUNDLE_SECRET_NAME or CABUNDLE_CONFIGMAP_NAME is set, add ca_cert_path arg to container driver.
	if common.GetCaBundleSecretName() != "" || common.GetCaBundleConfigMapName() != "" {
		args["ca_cert_path"] = common.CustomCaCertPath
	}

	if value, ok := os.LookupEnv(PipelineLogLevelEnvVar); ok {
		args["log_level"] = value
	}
	if value, ok := os.LookupEnv(PublishLogsEnvVar); ok {
		args["publish_logs"] = value
	}
	if c.defaultRunAsUser != nil {
		args["default_run_as_user"] = strconv.FormatInt(*c.defaultRunAsUser, 10)
	}
	if c.defaultRunAsGroup != nil {
		args["default_run_as_group"] = strconv.FormatInt(*c.defaultRunAsGroup, 10)
	}
	if c.defaultRunAsNonRoot != nil {
		args["default_run_as_non_root"] = strconv.FormatBool(*c.defaultRunAsNonRoot)
	}

	containerDriverPlugin, err := driverPlugin(args)
	if err != nil {
		return name, fmt.Errorf("failed to add container driver plugin: %v", err)
	}

	template := &wfapi.Template{
		Name: name,
		Inputs: wfapi.Inputs{
			Parameters: []wfapi.Parameter{
				{Name: paramComponent},
				{Name: paramTask},
				{Name: paramContainer},
				{Name: paramTaskName},
				{Name: paramParentDagID},
				{Name: paramIterationIndex, Default: wfapi.AnyStringPtr("-1")},
				{Name: paramKubernetesConfig, Default: wfapi.AnyStringPtr("")},
			},
		},
		Outputs: wfapi.Outputs{
			Parameters: []wfapi.Parameter{
				{Name: paramPodSpecPatch, ValueFrom: &wfapi.ValueFrom{JSONPath: "$.pod-spec-patch", Default: wfapi.AnyStringPtr("")}},
				{Name: paramCachedDecision, Default: wfapi.AnyStringPtr("false"), ValueFrom: &wfapi.ValueFrom{JSONPath: "$.cached-decision", Default: wfapi.AnyStringPtr("false")}},
				{Name: paramCondition, ValueFrom: &wfapi.ValueFrom{JSONPath: "$.condition", Default: wfapi.AnyStringPtr("true")}},
			},
		},
		Plugin: containerDriverPlugin,
	}
	applySecurityContextToTemplate(template)
	c.templates[name] = template
	c.wf.Spec.Templates = append(c.wf.Spec.Templates, *template)
	return name, err
}

type containerExecutorInputs struct {
	// a strategic patch of pod spec merged with runtime Pod spec.
	podSpecPatch string
	// if true, the container will be cached.
	cachedDecision string
	// if false, the container will be skipped.
	condition string
	// if provided, this will be the template the Argo Workflow exit lifecycle hook will execute.
	exitTemplate string
	// this will be provided as the parent-dag-id input to the Argo Workflow exit lifecycle hook.
	hookParentDagID string
}

// containerExecutorTask returns an argo workflows DAGTask.
// name: argo workflows DAG task name
// The other arguments are argo workflows task parameters, they can be either a
// string or a placeholder.
func (c *workflowCompiler) containerExecutorTask(name string, inputs containerExecutorInputs, task *pipelinespec.PipelineTaskSpec) (*wfapi.DAGTask, error) {
	when := ""
	if inputs.condition != "" {
		when = inputs.condition + " != false"
	}
	var refName string
	if componentRef := task.GetComponentRef(); componentRef != nil {
		refName = componentRef.Name
	} else {
		return nil, fmt.Errorf("component reference is nil")
	}

	// Retrieve pod metadata defined in the Kubernetes Spec, if any
	kubernetesConfigParam := c.wf.Spec.Arguments.GetParameterByName(argumentsKubernetesSpec + refName)
	k8sExecCfg := &kubernetesplatform.KubernetesExecutorConfig{}
	if kubernetesConfigParam != nil && kubernetesConfigParam.Value != nil {
		if err := protojson.Unmarshal([]byte((*kubernetesConfigParam.Value)), k8sExecCfg); err != nil {
			return nil, fmt.Errorf("failed to unmarshal kubernetes config: %v", err)
		}
	}
	dagTask := &wfapi.DAGTask{
		Name:     name,
		Template: c.addContainerExecutorTemplate(task, k8sExecCfg),
		When:     when,
		Arguments: wfapi.Arguments{
			Parameters: append(
				[]wfapi.Parameter{
					{
						Name:  paramPodSpecPatch,
						Value: wfapi.AnyStringPtr(inputs.podSpecPatch),
					},
					{
						Name:    paramCachedDecision,
						Value:   wfapi.AnyStringPtr(inputs.cachedDecision),
						Default: wfapi.AnyStringPtr("false"),
					},
				},
				append(
					c.getPodMetadataParameters(k8sExecCfg.GetPodMetadata(), true),
					c.getTaskRetryParametersWithValues(task)...)...),
		},
	}

	addExitTask(dagTask, inputs.exitTemplate, inputs.hookParentDagID)

	return dagTask, nil
}

// addContainerExecutorTemplate adds a generic container executor template for
// any container component task.
// During runtime, it's expected that pod-spec-patch will specify command, args
// and resources etc, that are different for different tasks.
func (c *workflowCompiler) addContainerExecutorTemplate(task *pipelinespec.PipelineTaskSpec, k8sExecCfg *kubernetesplatform.KubernetesExecutorConfig) string {
	// container template is parent of container implementation template
	nameContainerExecutor := "system-container-executor"
	nameContainerImpl := "system-container-impl"
	taskRetrySpec := task.GetRetryPolicy()
	refName := task.GetComponentRef().GetName()
	if taskRetrySpec != nil {
		task := c.spec.Components[refName]
		// retry-system-container-executor/impl is used if retry is set at the component level (if task is not a DAG)
		if task != nil && task.GetDag() == nil {
			nameContainerExecutor = "retry-" + nameContainerExecutor
			nameContainerImpl = "retry-" + nameContainerImpl
		}
	}
	podMetadata := k8sExecCfg.GetPodMetadata()
	// TODO (agoins): This approach was used because Argo Workflows does not currently supporting patching Metadata: https://github.com/argoproj/argo-workflows/issues/14661
	// if pod metadata is set, create a template name including the numbers of annotations and labels.
	if podMetadata != nil {
		numAnnotations := "0"
		numLabels := "0"
		if podMetadata.GetAnnotations() != nil {
			numAnnotations = strconv.Itoa(len(k8sExecCfg.GetPodMetadata().GetAnnotations()))
		}
		if podMetadata.GetLabels() != nil {
			numLabels = strconv.Itoa(len(k8sExecCfg.GetPodMetadata().GetLabels()))
		}
		nameContainerExecutor = "metadata-" + numAnnotations + "-" + numLabels + "-" + nameContainerExecutor
		nameContainerImpl = "metadata-" + numAnnotations + "-" + numLabels + "-" + nameContainerImpl
	}
	_, ok := c.templates[nameContainerExecutor]
	if ok {
		return nameContainerExecutor
	}
	container := &wfapi.Template{
		Name: nameContainerExecutor,
		Inputs: wfapi.Inputs{
			Parameters: append(
				[]wfapi.Parameter{
					{
						Name: paramPodSpecPatch,
					},
					{
						Name:    paramCachedDecision,
						Default: wfapi.AnyStringPtr("false"),
					},
				},
				append(
					c.getPodMetadataParameters(k8sExecCfg.GetPodMetadata(), false),
					c.addParameterDefault(c.getTaskRetryParameters(task), "0")...)...),
		},
		DAG: &wfapi.DAGTemplate{
			Tasks: []wfapi.DAGTask{{
				Name:     "executor",
				Template: nameContainerImpl,
				Arguments: wfapi.Arguments{
					Parameters: append(
						[]wfapi.Parameter{
							{
								Name:  paramPodSpecPatch,
								Value: wfapi.AnyStringPtr(inputParameter(paramPodSpecPatch)),
							},
						},
						append(
							c.addParameterInputPath(c.getPodMetadataParameters(k8sExecCfg.GetPodMetadata(), false)),
							c.addParameterInputPath(c.getTaskRetryParameters(task))...)...),
				},
				// When cached decision is true, the container
				// implementation template will be skipped, but
				// container executor template is still considered
				// to have succeeded.
				// This makes sure downstream tasks are not skipped.
				When: inputParameter(paramCachedDecision) + " != true",
			}},
		},
	}
	c.templates[nameContainerExecutor] = container

	args := []string{
		"--copy", component.KFPLauncherPath,
	}
	if c.cacheDisabled {
		args = append(args, "--cache_disabled")
	}
	if value, ok := os.LookupEnv(PipelineLogLevelEnvVar); ok {
		args = append(args, "--log_level", value)
	}
	if value, ok := os.LookupEnv(PublishLogsEnvVar); ok {
		args = append(args, "--publish_logs", value)
	}
	executor := &wfapi.Template{
		Name: nameContainerImpl,
		Inputs: wfapi.Inputs{
			Parameters: append(
				[]wfapi.Parameter{
					{
						Name: paramPodSpecPatch,
					},
				},
				append(
					c.getPodMetadataParameters(k8sExecCfg.GetPodMetadata(), false),
					c.getTaskRetryParameters(task)...)...),
		},
		// PodSpecPatch input param is where actual image, command and
		// args come from. It is treated as a strategic merge patch on
		// top of the Pod spec.
		PodSpecPatch: inputValue(paramPodSpecPatch),
		Volumes: []k8score.Volume{
			{
				Name: volumeNameKFPLauncher,
				VolumeSource: k8score.VolumeSource{
					EmptyDir: &k8score.EmptyDirVolumeSource{},
				},
			},
			{
				Name: gcsScratchName,
				VolumeSource: k8score.VolumeSource{
					EmptyDir: &k8score.EmptyDirVolumeSource{},
				},
			},
			{
				Name: s3ScratchName,
				VolumeSource: k8score.VolumeSource{
					EmptyDir: &k8score.EmptyDirVolumeSource{},
				},
			},
			{
				Name: minioScratchName,
				VolumeSource: k8score.VolumeSource{
					EmptyDir: &k8score.EmptyDirVolumeSource{},
				},
			},
			{
				Name: dotLocalScratchName,
				VolumeSource: k8score.VolumeSource{
					EmptyDir: &k8score.EmptyDirVolumeSource{},
				},
			},
			{
				Name: dotCacheScratchName,
				VolumeSource: k8score.VolumeSource{
					EmptyDir: &k8score.EmptyDirVolumeSource{},
				},
			},
			{
				Name: dotConfigScratchName,
				VolumeSource: k8score.VolumeSource{
					EmptyDir: &k8score.EmptyDirVolumeSource{},
				},
			},
		},
		InitContainers: []wfapi.UserContainer{{
			Container: k8score.Container{
				Name:    "kfp-launcher",
				Image:   c.launcherImage,
				Command: c.launcherCommand,
				Args:    args,
				VolumeMounts: []k8score.VolumeMount{
					{
						Name:      volumeNameKFPLauncher,
						MountPath: component.VolumePathKFPLauncher,
					},
				},
				Resources: launcherResources,
			},
		}},
		Container: &k8score.Container{
			// The placeholder image and command should always be
			// overridden in podSpecPatch.
			// In case we have a bug, the placeholder image is kept
			// in gcr.io/ml-pipeline, so that we are sure the image
			// never exists.
			// These are added to pass argo workflows linting.
			Image:   "gcr.io/ml-pipeline/should-be-overridden-during-runtime",
			Command: []string{"should-be-overridden-during-runtime"},
			VolumeMounts: []k8score.VolumeMount{
				{
					Name:      volumeNameKFPLauncher,
					MountPath: component.VolumePathKFPLauncher,
				},
				{
					Name:      gcsScratchName,
					MountPath: gcsScratchLocation,
				},
				{
					Name:      s3ScratchName,
					MountPath: s3ScratchLocation,
				},
				{
					Name:      minioScratchName,
					MountPath: minioScratchLocation,
				},
				{
					Name:      dotLocalScratchName,
					MountPath: dotLocalScratchLocation,
				},
				{
					Name:      dotCacheScratchName,
					MountPath: dotCacheScratchLocation,
				},
				{
					Name:      dotConfigScratchName,
					MountPath: dotConfigScratchLocation,
				},
			},
			EnvFrom: []k8score.EnvFromSource{metadataEnvFrom},
			Env:     commonEnvs,
		},
	}
	// If CABUNDLE_SECRET_NAME or CABUNDLE_CONFIGMAP_NAME is set, add the custom CA bundle to the executor.
	if common.GetCaBundleSecretName() != "" || common.GetCaBundleConfigMapName() != "" {
		ConfigureCustomCABundle(executor)
	}
	applySecurityContextToExecutorTemplate(executor, c.defaultRunAsUser, c.defaultRunAsGroup, c.defaultRunAsNonRoot)

	// If retry policy is set, add retryStrategy to executor
	if taskRetrySpec != nil {
		executor.RetryStrategy = c.getTaskRetryStrategyFromInput(inputParameter(paramRetryMaxCount),
			inputParameter(paramRetryBackOffDuration),
			inputParameter(paramRetryBackOffFactor),
			inputParameter(paramRetryBackOffMaxDuration))
	}
	// Update pod metadata if it defined in the Kubernetes Spec
	if k8sExecCfg.GetPodMetadata() != nil {
		extendPodMetadata(&executor.Metadata, k8sExecCfg)
	}

	c.templates[nameContainerImpl] = executor
	c.wf.Spec.Templates = append(c.wf.Spec.Templates, *container, *executor)
	return nameContainerExecutor
}

func (c *workflowCompiler) getTaskRetryParameters(task *pipelinespec.PipelineTaskSpec) []wfapi.Parameter {
	if task != nil && task.RetryPolicy != nil {
		// Create slice of parameters with "Name" field set.
		parameters := []wfapi.Parameter{
			{Name: paramRetryMaxCount},
			{Name: paramRetryBackOffDuration},
			{Name: paramRetryBackOffFactor},
			{Name: paramRetryBackOffMaxDuration},
		}
		return parameters
	} else {
		return nil
	}
}

func (c *workflowCompiler) getTaskRetryParametersWithValues(task *pipelinespec.PipelineTaskSpec) []wfapi.Parameter {
	retryPolicy := task.GetRetryPolicy()
	if retryPolicy == nil {
		return []wfapi.Parameter{}
	}
	// Create slice of all non-nil retryPolicy fields.
	parameters := append(
		[]wfapi.Parameter{},
		wfapi.Parameter{
			Name:  paramRetryMaxCount,
			Value: wfapi.AnyStringPtr(retryPolicy.MaxRetryCount),
		})
	if retryPolicy.GetBackoffDuration() != nil {
		retryBackOffDuration := wfapi.AnyStringPtr(retryPolicy.GetBackoffDuration().Seconds)
		parameters = append(
			parameters,
			wfapi.Parameter{
				Name:  paramRetryBackOffDuration,
				Value: retryBackOffDuration})
	}
	parameters = append(
		parameters,
		wfapi.Parameter{
			Name:  paramRetryBackOffFactor,
			Value: wfapi.AnyStringPtr(retryPolicy.GetBackoffFactor()),
		})
	if retryPolicy.GetBackoffMaxDuration() != nil {
		retryBackOffMaxDuration := wfapi.AnyStringPtr(retryPolicy.GetBackoffMaxDuration().Seconds)
		parameters = append(
			parameters,
			wfapi.Parameter{
				Name:  paramRetryBackOffMaxDuration,
				Value: retryBackOffMaxDuration})
	}
	return parameters
}

func (c *workflowCompiler) addParameterDefault(parameters []wfapi.Parameter, defaultValue string) []wfapi.Parameter {
	// Set the "Default" field of each parameter in input slice with input defaultValue.
	for i := range parameters {
		parameters[i].Default = wfapi.AnyStringPtr(defaultValue)
	}
	return parameters
}

func (c *workflowCompiler) addParameterInputPath(parameters []wfapi.Parameter) []wfapi.Parameter {
	// Format "Value" field of each parameter in input slice as '{{inputs.parameters.[parameter name]}}'
	for i := range parameters {
		parameters[i].Value = wfapi.AnyStringPtr(inputParameter(parameters[i].Name))
	}
	return parameters
}

func (c *workflowCompiler) getTaskRetryStrategyFromInput(maxCount string, backOffDuration string, backOffFactor string,
	backOffMaxDuration string) *wfapi.RetryStrategy {
	backoff := &wfapi.Backoff{
		Factor: &intstr.IntOrString{
			Type:   intstr.String,
			StrVal: backOffFactor,
		},
		MaxDuration: backOffMaxDuration,
		Duration:    backOffDuration,
	}

	return &wfapi.RetryStrategy{
		Limit: &intstr.IntOrString{
			Type:   intstr.String,
			StrVal: maxCount,
		},
		Backoff: backoff,
	}
}

// Return pod metadata parameters for annotations and labels, with corresponding key/value vals, if applicable.
// If no parameters found, returns empty slice.
func (c *workflowCompiler) getPodMetadataParameters(podMetadata *kubernetesplatform.PodMetadata, includeVal bool) []wfapi.Parameter {
	parameters := []wfapi.Parameter{}
	if podMetadata != nil && podMetadata.GetAnnotations() != nil {
		annotations := podMetadata.GetAnnotations()
		parameters = append(parameters, c.formatPodMetadataParameters(annotations, includeVal, paramPodAnnotationKey, paramPodAnnotationVal)...)
	}
	if podMetadata != nil && podMetadata.GetLabels() != nil {
		labels := podMetadata.GetLabels()
		parameters = append(parameters, c.formatPodMetadataParameters(labels, includeVal, paramPodLabelKey, paramPodLabelVal)...)
	}
	return parameters
}

// Return slice of formatted parameters. If includeVal is set to true, the corresponding values for the metadata key
// and value are included. If input metadata map is empty, empty slice is returned.
func (c *workflowCompiler) formatPodMetadataParameters(metadata map[string]string, includeVal bool, paramPodMetadataKey string, paramPodMetadataVal string) []wfapi.Parameter {
	var parameters []wfapi.Parameter
	// sort metadata alphabetically to ensure metadata are processed in identical order across all runs for a pipeline.
	sortedMetadataKeys := make([]string, 0, len(metadata))
	for entry := range metadata {
		sortedMetadataKeys = append(sortedMetadataKeys, entry)
	}
	sort.Strings(sortedMetadataKeys)
	// paramPodMetadataKey/value parameters are numbered when more than one annotation/label is set.
	count := 1
	key := paramPodMetadataKey
	val := paramPodMetadataVal
	for _, metadataKey := range sortedMetadataKeys {
		if len(sortedMetadataKeys) > 1 {
			key = paramPodMetadataKey + "-" + strconv.Itoa(count)
			val = paramPodMetadataVal + "-" + strconv.Itoa(count)
		}
		if includeVal {
			parameters = append(parameters,
				[]wfapi.Parameter{
					{
						Name:  key,
						Value: wfapi.AnyStringPtr(metadataKey),
					},
					{
						Name:  val,
						Value: wfapi.AnyStringPtr(metadata[metadataKey]),
					}}...)
		} else {
			parameters = append(parameters, []wfapi.Parameter{{Name: key}, {Name: val}}...)
		}
		count++
	}
	return parameters
}

// Extends the PodMetadata to PodMetadata input parameters.
// Although the current podMetadata object is always empty, this function
// doesn't overwrite the existing podMetadata because for security reasons
// the existing podMetadata should have higher privilege than the user definition.
func extendPodMetadata(
	podMetadata *wfapi.Metadata,
	kubernetesExecutorConfig *kubernetesplatform.KubernetesExecutorConfig,
) {
	// Get pod metadata information
	if kubernetesExecutorConfig.GetPodMetadata() != nil {
		labels := kubernetesExecutorConfig.GetPodMetadata().GetLabels()
		if labels != nil {
			count := 1
			for range labels {
				key := paramPodLabelKey
				val := paramPodLabelVal
				if len(labels) > 1 {
					key = key + "-" + strconv.Itoa(count)
					val = val + "-" + strconv.Itoa(count)
				}
				if podMetadata.Labels == nil {
					podMetadata.Labels = map[string]string{inputParameter(key): inputParameter(val)}
				} else {
					podMetadata.Labels = extendMetadataMap(podMetadata.Labels, map[string]string{inputParameter(key): inputParameter(val)})
				}
				count++
			}
		}
		annotations := kubernetesExecutorConfig.GetPodMetadata().GetAnnotations()
		if annotations != nil {
			count := 1
			for range annotations {
				key := paramPodAnnotationKey
				val := paramPodAnnotationVal
				if len(annotations) > 1 {
					key = key + "-" + strconv.Itoa(count)
					val = val + "-" + strconv.Itoa(count)
				}
				if podMetadata.Annotations == nil {
					podMetadata.Annotations = map[string]string{inputParameter(key): inputParameter(val)}
				} else {
					podMetadata.Annotations = extendMetadataMap(podMetadata.Annotations, map[string]string{inputParameter(key): inputParameter(val)})
				}
				count++
			}
		}
	}
}

// Extends metadata map values, highPriorityMap should overwrites lowPriorityMap values
// The original Map inputs should have higher priority since its defined by admin
// TODO: Use maps.Copy after moving to go 1.21+
func extendMetadataMap(
	highPriorityMap map[string]string,
	lowPriorityMap map[string]string,
) map[string]string {
	for k, v := range highPriorityMap {
		lowPriorityMap[k] = v
	}
	return lowPriorityMap
}
