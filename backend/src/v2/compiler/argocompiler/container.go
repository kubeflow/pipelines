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
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/kubeflow/pipelines/backend/src/v2/config"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"
	"k8s.io/apimachinery/pkg/api/resource"
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
	volumeNameKFPLauncher   = "kfp-launcher"
	volumeNameCABundle      = "ca-bundle"
	volumeNameDriverOutputs = "kfp-driver-outputs"
	driverOutputsMountPath  = "/tmp/kfp-driver-outputs"
	LauncherImageEnvVar     = "V2_LAUNCHER_IMAGE"
	DriverImageEnvVar       = "V2_DRIVER_IMAGE"
	// DefaultLauncherImage & DefaultDriverImage are set as latest here
	// but are overridden by environment variables set via k8s manifests.
	// For releases, the manifest will have the correct release version set.
	// this is to avoid hardcoding releases in code here.
	DefaultLauncherImage     = "ghcr.io/kubeflow/kfp-launcher:latest"
	LauncherCommandEnvVar    = "V2_LAUNCHER_COMMAND"
	DefaultLauncherCommand   = "launcher-v2"
	DefaultDriverImage       = "ghcr.io/kubeflow/kfp-driver:latest"
	DefaultDriverCommand     = "driver"
	DriverCommandEnvVar      = "V2_DRIVER_COMMAND"
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

// containerExecutorInputs holds the arguments passed to the executor pod template.
// Unlike the old design, there is no separate driver pod; the driver runs as an
// init container so all context is forwarded directly to a single DAG task.
type containerExecutorInputs struct {
	component        string
	task             string
	taskName         string
	container        string
	parentDagID      string
	iterationIndex   string
	kubernetesConfig string
	// image is the user container image (static value known at compile time).
	image string
	// exitTemplate and hookParentDagID support Argo lifecycle exit hooks.
	exitTemplate    string
	hookParentDagID string
}

// containerDriverInputs is used only for dummy-image tasks (e.g. createPVC/deletePVC)
// which still run as a standalone driver pod with no executor.
type containerDriverInputs struct {
	component        string
	task             string
	taskName         string
	container        string
	parentDagID      string
	iterationIndex   string
	kubernetesConfig string
}

// containerDriverTask creates the legacy standalone driver DAG task used only
// for dummy-image tasks (argostub/createpvc, argostub/deletepvc).
func (c *workflowCompiler) containerDriverTask(name string, inputs containerDriverInputs) (*wfapi.DAGTask, *struct{}) {
	dagTask := &wfapi.DAGTask{
		Name:     name,
		Template: c.addContainerDriverTemplate(),
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
		dagTask.Arguments.Parameters = append(dagTask.Arguments.Parameters,
			wfapi.Parameter{Name: paramIterationIndex, Value: wfapi.AnyStringPtr(inputs.iterationIndex)})
	}
	if inputs.kubernetesConfig != "" {
		dagTask.Arguments.Parameters = append(dagTask.Arguments.Parameters,
			wfapi.Parameter{Name: paramKubernetesConfig, Value: wfapi.AnyStringPtr(inputs.kubernetesConfig)})
	}
	return dagTask, nil
}

// addContainerDriverTemplate creates the standalone driver pod template used
// only for dummy-image tasks (e.g. createPVC, deletePVC).
func (c *workflowCompiler) addContainerDriverTemplate() string {
	name := "system-container-driver"
	if _, ok := c.templates[name]; ok {
		return name
	}

	args := []string{
		"--type", "CONTAINER",
		"--pipeline_name", c.spec.GetPipelineInfo().GetName(),
		"--run_id", runID(),
		"--run_name", runResourceName(),
		"--run_display_name", c.job.DisplayName,
		"--dag_execution_id", inputValue(paramParentDagID),
		"--component", inputValue(paramComponent),
		"--task", inputValue(paramTask),
		"--task_name", inputValue(paramTaskName),
		"--container", inputValue(paramContainer),
		"--iteration_index", inputValue(paramIterationIndex),
		"--cached_decision_path", outputPath(paramCachedDecision),
		"--pod_spec_patch_path", outputPath(paramPodSpecPatch),
		"--condition_path", outputPath(paramCondition),
		"--kubernetes_config", inputValue(paramKubernetesConfig),
		"--http_proxy", proxy.GetConfig().GetHttpProxy(),
		"--https_proxy", proxy.GetConfig().GetHttpsProxy(),
		"--no_proxy", proxy.GetConfig().GetNoProxy(),
		"--ml_pipeline_server_address", config.GetMLPipelineServerConfig().Address,
		"--ml_pipeline_server_port", config.GetMLPipelineServerConfig().Port,
		"--mlmd_server_address", metadata.GetMetadataConfig().Address,
		"--mlmd_server_port", metadata.GetMetadataConfig().Port,
	}
	if c.cacheDisabled {
		args = append(args, "--cache_disabled")
	}
	if c.mlPipelineTLSEnabled {
		args = append(args, "--ml_pipeline_tls_enabled")
	}
	if common.GetMetadataTLSEnabled() {
		args = append(args, "--metadata_tls_enabled")
	}
	setCABundle := common.GetCaBundleSecretName() != "" || common.GetCaBundleConfigMapName() != ""
	if setCABundle {
		args = append(args, "--ca_cert_path", common.CustomCaCertPath)
	}
	if value, ok := os.LookupEnv(PipelineLogLevelEnvVar); ok {
		args = append(args, "--log_level", value)
	}
	if value, ok := os.LookupEnv(PublishLogsEnvVar); ok {
		args = append(args, "--publish_logs", value)
	}
	if c.defaultRunAsUser != nil {
		args = append(args, "--default_run_as_user", strconv.FormatInt(*c.defaultRunAsUser, 10))
	}
	if c.defaultRunAsGroup != nil {
		args = append(args, "--default_run_as_group", strconv.FormatInt(*c.defaultRunAsGroup, 10))
	}
	if c.defaultRunAsNonRoot != nil {
		args = append(args, "--default_run_as_non_root", strconv.FormatBool(*c.defaultRunAsNonRoot))
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
				{Name: paramPodSpecPatch, ValueFrom: &wfapi.ValueFrom{Path: "/tmp/outputs/pod-spec-patch", Default: wfapi.AnyStringPtr("")}},
				{Name: paramCachedDecision, Default: wfapi.AnyStringPtr("false"), ValueFrom: &wfapi.ValueFrom{Path: "/tmp/outputs/cached-decision", Default: wfapi.AnyStringPtr("false")}},
				{Name: paramCondition, ValueFrom: &wfapi.ValueFrom{Path: "/tmp/outputs/condition", Default: wfapi.AnyStringPtr("true")}},
			},
		},
		Container: &k8score.Container{
			Image:     c.driverImage,
			Command:   c.driverCommand,
			Args:      args,
			Resources: driverResources,
			Env:       append(proxy.GetConfig().GetEnvVars(), commonEnvs...),
		},
	}
	applySecurityContextToTemplate(template)
	if setCABundle {
		ConfigureCustomCABundle(template)
	}
	c.templates[name] = template
	c.wf.Spec.Templates = append(c.wf.Spec.Templates, *template)
	return name
}

func GetLauncherImage() string {
	launcherImage := os.Getenv(LauncherImageEnvVar)
	if launcherImage == "" {
		launcherImage = DefaultLauncherImage
	}
	return launcherImage
}

func GetDriverImage() string {
	driverImage := os.Getenv(DriverImageEnvVar)
	if driverImage == "" {
		driverImage = DefaultDriverImage
	}
	return driverImage
}

func GetDriverCommand() []string {
	driverCommand := os.Getenv(DriverCommandEnvVar)
	if driverCommand == "" {
		driverCommand = DefaultDriverCommand
	}
	return strings.Split(driverCommand, " ")
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

// containerExecutorTask returns an Argo DAGTask that runs a single executor pod
// with the driver as an init container. No separate driver DAG task is created.
func (c *workflowCompiler) containerExecutorTask(name string, inputs containerExecutorInputs, task *pipelinespec.PipelineTaskSpec) (*wfapi.DAGTask, error) {
	var refName string
	if componentRef := task.GetComponentRef(); componentRef != nil {
		refName = componentRef.Name
	} else {
		return nil, fmt.Errorf("component reference is nil")
	}

	// Retrieve static Kubernetes settings for compile-time embedding.
	k8sExecCfg := &kubernetesplatform.KubernetesExecutorConfig{}
	kubernetesConfigParam := c.wf.Spec.Arguments.GetParameterByName(argumentsKubernetesSpec + refName)
	if kubernetesConfigParam != nil && kubernetesConfigParam.Value != nil {
		if err := protojson.Unmarshal([]byte(*kubernetesConfigParam.Value), k8sExecCfg); err != nil {
			return nil, fmt.Errorf("failed to unmarshal kubernetes config: %v", err)
		}
	}

	params := []wfapi.Parameter{
		{Name: paramComponent, Value: wfapi.AnyStringPtr(inputs.component)},
		{Name: paramTask, Value: wfapi.AnyStringPtr(inputs.task)},
		{Name: paramContainer, Value: wfapi.AnyStringPtr(inputs.container)},
		{Name: paramTaskName, Value: wfapi.AnyStringPtr(inputs.taskName)},
		{Name: paramParentDagID, Value: wfapi.AnyStringPtr(inputs.parentDagID)},
		{Name: paramImage, Value: wfapi.AnyStringPtr(inputs.image)},
	}
	if inputs.iterationIndex != "" {
		params = append(params, wfapi.Parameter{
			Name:  paramIterationIndex,
			Value: wfapi.AnyStringPtr(inputs.iterationIndex),
		})
	}
	if inputs.kubernetesConfig != "" {
		params = append(params, wfapi.Parameter{
			Name:  paramKubernetesConfig,
			Value: wfapi.AnyStringPtr(inputs.kubernetesConfig),
		})
	}
	params = append(params,
		append(
			c.getPodMetadataParameters(k8sExecCfg.GetPodMetadata(), true),
			c.getTaskRetryParametersWithValues(task)...)...)

	dagTask := &wfapi.DAGTask{
		Name:     name,
		Template: c.addContainerExecutorTemplate(task, k8sExecCfg),
		Arguments: wfapi.Arguments{
			Parameters: params,
		},
	}

	addExitTask(dagTask, inputs.exitTemplate, inputs.hookParentDagID)
	return dagTask, nil
}

// addContainerExecutorTemplate creates the executor template for a container task.
// The driver runs as the first init container and communicates with the launcher
// (main container) via a shared emptyDir volume at driverOutputsMountPath.
// All static Kubernetes platform settings are applied at compile time.
func (c *workflowCompiler) addContainerExecutorTemplate(task *pipelinespec.PipelineTaskSpec, k8sExecCfg *kubernetesplatform.KubernetesExecutorConfig) string {
	nameContainerImpl := "system-container-impl"
	taskRetrySpec := task.GetRetryPolicy()
	refName := task.GetComponentRef().GetName()
	if taskRetrySpec != nil {
		compSpec := c.spec.Components[refName]
		if compSpec != nil && compSpec.GetDag() == nil {
			nameContainerImpl = "retry-" + nameContainerImpl
		}
	}
	podMetadata := k8sExecCfg.GetPodMetadata()
	if podMetadata != nil {
		numAnnotations := "0"
		numLabels := "0"
		if podMetadata.GetAnnotations() != nil {
			numAnnotations = strconv.Itoa(len(podMetadata.GetAnnotations()))
		}
		if podMetadata.GetLabels() != nil {
			numLabels = strconv.Itoa(len(podMetadata.GetLabels()))
		}
		nameContainerImpl = "metadata-" + numAnnotations + "-" + numLabels + "-" + nameContainerImpl
	}
	// Static K8s settings (volumes, mounts, node selector, tolerations, etc.) are
	// embedded directly in the template, so tasks with different configs need
	// distinct template names.
	if hasStaticK8sSettings(k8sExecCfg) {
		nameContainerImpl = "k8s-" + refName + "-" + nameContainerImpl
	}

	if _, ok := c.templates[nameContainerImpl]; ok {
		return nameContainerImpl
	}

	// Build driver init container args. These mirror the old standalone driver
	// template args, replacing the three output path flags with --driver_outputs_dir.
	driverArgs := []string{
		"--type", "CONTAINER",
		"--pipeline_name", c.spec.GetPipelineInfo().GetName(),
		"--run_id", runID(),
		"--run_name", runResourceName(),
		"--run_display_name", c.job.DisplayName,
		"--dag_execution_id", inputValue(paramParentDagID),
		"--component", inputValue(paramComponent),
		"--task", inputValue(paramTask),
		"--task_name", inputValue(paramTaskName),
		"--container", inputValue(paramContainer),
		"--iteration_index", inputValue(paramIterationIndex),
		"--kubernetes_config", inputValue(paramKubernetesConfig),
		"--driver_outputs_dir", driverOutputsMountPath,
		"--http_proxy", proxy.GetConfig().GetHttpProxy(),
		"--https_proxy", proxy.GetConfig().GetHttpsProxy(),
		"--no_proxy", proxy.GetConfig().GetNoProxy(),
		"--ml_pipeline_server_address", config.GetMLPipelineServerConfig().Address,
		"--ml_pipeline_server_port", config.GetMLPipelineServerConfig().Port,
		"--mlmd_server_address", metadata.GetMetadataConfig().Address,
		"--mlmd_server_port", metadata.GetMetadataConfig().Port,
	}
	if c.cacheDisabled {
		driverArgs = append(driverArgs, "--cache_disabled")
	}
	if c.mlPipelineTLSEnabled {
		driverArgs = append(driverArgs, "--ml_pipeline_tls_enabled")
	}
	if common.GetMetadataTLSEnabled() {
		driverArgs = append(driverArgs, "--metadata_tls_enabled")
	}
	setCABundle := common.GetCaBundleSecretName() != "" || common.GetCaBundleConfigMapName() != ""
	if setCABundle {
		driverArgs = append(driverArgs, "--ca_cert_path", common.CustomCaCertPath)
	}
	if value, ok := os.LookupEnv(PipelineLogLevelEnvVar); ok {
		driverArgs = append(driverArgs, "--log_level", value)
	}
	if value, ok := os.LookupEnv(PublishLogsEnvVar); ok {
		driverArgs = append(driverArgs, "--publish_logs", value)
	}
	if c.defaultRunAsUser != nil {
		driverArgs = append(driverArgs, "--default_run_as_user", strconv.FormatInt(*c.defaultRunAsUser, 10))
	}
	if c.defaultRunAsGroup != nil {
		driverArgs = append(driverArgs, "--default_run_as_group", strconv.FormatInt(*c.defaultRunAsGroup, 10))
	}
	if c.defaultRunAsNonRoot != nil {
		driverArgs = append(driverArgs, "--default_run_as_non_root", strconv.FormatBool(*c.defaultRunAsNonRoot))
	}

	// Launcher init container: copies the launcher binary to the shared launcher volume.
	launcherCopyArgs := []string{"--copy", component.KFPLauncherPath}
	if c.cacheDisabled {
		launcherCopyArgs = append(launcherCopyArgs, "--cache_disabled")
	}
	if value, ok := os.LookupEnv(PipelineLogLevelEnvVar); ok {
		launcherCopyArgs = append(launcherCopyArgs, "--log_level", value)
	}
	if value, ok := os.LookupEnv(PublishLogsEnvVar); ok {
		launcherCopyArgs = append(launcherCopyArgs, "--publish_logs", value)
	}

	// Main container command. The launcher reads execution-id, executor-input,
	// user-command, user-args, cached-decision and condition from driverOutputsMountPath.
	launcherCmd := []string{
		component.KFPLauncherPath,
		"--pipeline_name", c.spec.GetPipelineInfo().GetName(),
		"--run_id", runID(),
		"--driver_outputs_dir", driverOutputsMountPath,
		"--component_spec", inputValue(paramComponent),
		"--pod_name", fmt.Sprintf("$(%s)", component.EnvPodName),
		"--pod_uid", fmt.Sprintf("$(%s)", component.EnvPodUID),
		"--ml_pipeline_server_address", config.GetMLPipelineServerConfig().Address,
		"--ml_pipeline_server_port", config.GetMLPipelineServerConfig().Port,
		"--mlmd_server_address", metadata.GetMetadataConfig().Address,
		"--mlmd_server_port", metadata.GetMetadataConfig().Port,
	}
	if c.mlPipelineTLSEnabled {
		launcherCmd = append(launcherCmd, "--ml_pipeline_tls_enabled")
	}
	if common.GetMetadataTLSEnabled() {
		launcherCmd = append(launcherCmd, "--metadata_tls_enabled")
	}
	if setCABundle {
		launcherCmd = append(launcherCmd, "--ca_cert_path", common.CustomCaCertPath)
	}
	if c.cacheDisabled {
		launcherCmd = append(launcherCmd, "--cache_disabled")
	}
	if value, ok := os.LookupEnv(PipelineLogLevelEnvVar); ok {
		launcherCmd = append(launcherCmd, "--log_level", value)
	}
	if value, ok := os.LookupEnv(PublishLogsEnvVar); ok {
		launcherCmd = append(launcherCmd, "--publish_logs", value)
	}

	// Base volumes: driver outputs, launcher binary, and artifact scratch dirs.
	volumes := []k8score.Volume{
		{Name: volumeNameDriverOutputs, VolumeSource: k8score.VolumeSource{EmptyDir: &k8score.EmptyDirVolumeSource{}}},
		{Name: volumeNameKFPLauncher, VolumeSource: k8score.VolumeSource{EmptyDir: &k8score.EmptyDirVolumeSource{}}},
		{Name: gcsScratchName, VolumeSource: k8score.VolumeSource{EmptyDir: &k8score.EmptyDirVolumeSource{}}},
		{Name: s3ScratchName, VolumeSource: k8score.VolumeSource{EmptyDir: &k8score.EmptyDirVolumeSource{}}},
		{Name: minioScratchName, VolumeSource: k8score.VolumeSource{EmptyDir: &k8score.EmptyDirVolumeSource{}}},
		{Name: dotLocalScratchName, VolumeSource: k8score.VolumeSource{EmptyDir: &k8score.EmptyDirVolumeSource{}}},
		{Name: dotCacheScratchName, VolumeSource: k8score.VolumeSource{EmptyDir: &k8score.EmptyDirVolumeSource{}}},
		{Name: dotConfigScratchName, VolumeSource: k8score.VolumeSource{EmptyDir: &k8score.EmptyDirVolumeSource{}}},
	}

	// Main container volume mounts.
	mainVolumeMounts := []k8score.VolumeMount{
		{Name: volumeNameDriverOutputs, MountPath: driverOutputsMountPath},
		{Name: volumeNameKFPLauncher, MountPath: component.VolumePathKFPLauncher},
		{Name: gcsScratchName, MountPath: gcsScratchLocation},
		{Name: s3ScratchName, MountPath: s3ScratchLocation},
		{Name: minioScratchName, MountPath: minioScratchLocation},
		{Name: dotLocalScratchName, MountPath: dotLocalScratchLocation},
		{Name: dotCacheScratchName, MountPath: dotCacheScratchLocation},
		{Name: dotConfigScratchName, MountPath: dotConfigScratchLocation},
	}

	executor := &wfapi.Template{
		Name: nameContainerImpl,
		Inputs: wfapi.Inputs{
			Parameters: append(
				[]wfapi.Parameter{
					{Name: paramComponent},
					{Name: paramTask},
					{Name: paramContainer},
					{Name: paramTaskName},
					{Name: paramParentDagID},
					{Name: paramIterationIndex, Default: wfapi.AnyStringPtr("-1")},
					{Name: paramKubernetesConfig, Default: wfapi.AnyStringPtr("")},
					// image is passed at call time so templates can be shared across tasks
					// with the same K8s config but different container images.
					{Name: paramImage},
				},
				append(
					c.getPodMetadataParameters(k8sExecCfg.GetPodMetadata(), false),
					c.getTaskRetryParameters(task)...)...),
		},
		Volumes: volumes,
		InitContainers: []wfapi.UserContainer{
			// First init container: driver resolves inputs and writes outputs to the shared volume.
			{
				Container: k8score.Container{
					Name:      "kfp-driver",
					Image:     c.driverImage,
					Command:   c.driverCommand,
					Args:      driverArgs,
					Resources: driverResources,
					Env:       append(proxy.GetConfig().GetEnvVars(), commonEnvs...),
					VolumeMounts: []k8score.VolumeMount{
						{Name: volumeNameDriverOutputs, MountPath: driverOutputsMountPath},
					},
				},
			},
			// Second init container: copies the launcher binary to the shared launcher volume.
			{
				Container: k8score.Container{
					Name:      "kfp-launcher",
					Image:     c.launcherImage,
					Command:   c.launcherCommand,
					Args:      launcherCopyArgs,
					Resources: launcherResources,
					VolumeMounts: []k8score.VolumeMount{
						{Name: volumeNameKFPLauncher, MountPath: component.VolumePathKFPLauncher},
					},
				},
			},
		},
		Container: &k8score.Container{
			// Image is parameterized so tasks with the same K8s config can share this template.
			Image:        inputValue(paramImage),
			Command:      launcherCmd,
			EnvFrom:      []k8score.EnvFromSource{metadataEnvFrom},
			Env:          commonEnvs,
			VolumeMounts: mainVolumeMounts,
		},
	}

	// Apply static Kubernetes platform settings (volumes, mounts, node selector, etc.)
	// directly to the template — these were previously applied at runtime via pod-spec-patch.
	// Build a resolver for ComponentInputParameter PVC names: look up the value from the
	// run's runtime config first, then fall back to pipeline spec default values.
	resolveParam := func(paramName string) (string, bool) {
		if pv := c.job.GetRuntimeConfig().GetParameterValues(); pv != nil {
			if v, ok := pv[paramName]; ok {
				if s := v.GetStringValue(); s != "" {
					return s, true
				}
			}
		}
		if params := c.spec.GetRoot().GetInputDefinitions().GetParameters(); params != nil {
			if p, ok := params[paramName]; ok {
				if dv := p.GetDefaultValue(); dv != nil {
					if s := dv.GetStringValue(); s != "" {
						return s, true
					}
				}
			}
		}
		return "", false
	}
	applyStaticK8sConfig(executor, k8sExecCfg, resolveParam)

	if setCABundle {
		ConfigureCustomCABundle(executor)
		// The driver init container also needs the CA bundle for TLS calls to MLMD/API server.
		for _, vol := range executor.Volumes {
			if vol.Name == volumeNameCABundle {
				executor.InitContainers[0].VolumeMounts = append(
					executor.InitContainers[0].VolumeMounts,
					k8score.VolumeMount{Name: volumeNameCABundle, MountPath: common.CABundleDir},
				)
				break
			}
		}
	}

	applySecurityContextToExecutorTemplate(executor, c.defaultRunAsUser, c.defaultRunAsGroup, c.defaultRunAsNonRoot)

	if taskRetrySpec != nil {
		executor.RetryStrategy = c.getTaskRetryStrategyFromInput(
			inputParameter(paramRetryMaxCount),
			inputParameter(paramRetryBackOffDuration),
			inputParameter(paramRetryBackOffFactor),
			inputParameter(paramRetryBackOffMaxDuration))
		executor.Container.Env = append(executor.Container.Env, retryIndexEnv)
	}

	if k8sExecCfg.GetPodMetadata() != nil {
		extendPodMetadata(&executor.Metadata, k8sExecCfg)
	}

	c.templates[nameContainerImpl] = executor
	c.wf.Spec.Templates = append(c.wf.Spec.Templates, *executor)
	return nameContainerImpl
}

// hasStaticK8sSettings reports whether cfg contains any settings that will be
// embedded directly in the executor template (i.e. beyond PodMetadata which is
// handled via parameterized metadata parameters).
func hasStaticK8sSettings(cfg *kubernetesplatform.KubernetesExecutorConfig) bool {
	if cfg == nil {
		return false
	}
	return len(cfg.GetSecretAsVolume()) > 0 ||
		len(cfg.GetSecretAsEnv()) > 0 ||
		len(cfg.GetConfigMapAsVolume()) > 0 ||
		len(cfg.GetConfigMapAsEnv()) > 0 ||
		len(cfg.GetPvcMount()) > 0 ||
		len(cfg.GetFieldPathAsEnv()) > 0 ||
		len(cfg.GetEmptyDirMounts()) > 0 ||
		len(cfg.GetGenericEphemeralVolume()) > 0 ||
		len(cfg.GetTolerations()) > 0 ||
		cfg.GetNodeSelector() != nil ||
		cfg.GetNodeAffinity() != nil ||
		cfg.GetPodAffinity() != nil ||
		cfg.GetEnabledSharedMemory() != nil ||
		cfg.GetActiveDeadlineSeconds() > 0 ||
		cfg.GetImagePullPolicy() != "" ||
		len(cfg.GetImagePullSecret()) > 0 ||
		cfg.GetSecurityContext() != nil
}

// applyStaticK8sConfig embeds static Kubernetes platform settings from cfg into
// the executor template at compile time. Only settings with statically-known
// values are applied; settings that require runtime parameter resolution
// (e.g. PVC names from task output parameters, TolerationJson) are skipped with
// a log warning.
//
// resolveParam resolves a ComponentInputParameter name to its string value. It
// should first check the run's RuntimeConfig.ParameterValues, then the pipeline
// spec's root-level parameter defaults. Passing nil is treated as "no resolver"
// (all ComponentInputParameter PVC references will be skipped).
func applyStaticK8sConfig(tmpl *wfapi.Template, cfg *kubernetesplatform.KubernetesExecutorConfig, resolveParam func(name string) (string, bool)) {
	if cfg == nil || tmpl.Container == nil {
		return
	}

	// Collect pod-level fields that require a strategic merge patch.
	podSpec := map[string]interface{}{}

	// Image pull policy (container-level).
	if p := cfg.GetImagePullPolicy(); p != "" {
		tmpl.Container.ImagePullPolicy = k8score.PullPolicy(p)
	}

	// Image pull secrets (pod-level).
	if secrets := cfg.GetImagePullSecret(); len(secrets) > 0 {
		refs := make([]map[string]interface{}, 0, len(secrets))
		for _, s := range secrets {
			name := resolveStringParam(s.GetSecretNameParameter(), resolveParam)
			if name == "" {
				name = s.GetSecretName() //nolint:staticcheck
			}
			refs = append(refs, map[string]interface{}{"name": name})
		}
		podSpec["imagePullSecrets"] = refs
	}

	// Node selector — static labels map only; NodeSelectorJson is skipped.
	if ns := cfg.GetNodeSelector(); ns != nil {
		if ns.GetNodeSelectorJson() != nil {
			glog.Warningf("NodeSelectorJson is not supported with init-container driver; skipping")
		} else if labels := ns.GetLabels(); len(labels) > 0 {
			podSpec["nodeSelector"] = labels
		}
	}

	// Tolerations — static fields only; TolerationJson is skipped.
	if tols := cfg.GetTolerations(); len(tols) > 0 {
		tolList := make([]map[string]interface{}, 0, len(tols))
		for _, t := range tols {
			if t.GetTolerationJson() != nil {
				glog.Warningf("TolerationJson is not supported with init-container driver; skipping")
				continue
			}
			tol := map[string]interface{}{}
			if t.Key != "" {
				tol["key"] = t.Key
			}
			if t.Operator != "" {
				tol["operator"] = t.Operator
			}
			if t.Value != "" {
				tol["value"] = t.Value
			}
			if t.Effect != "" {
				tol["effect"] = t.Effect
			}
			if t.TolerationSeconds != nil {
				tol["tolerationSeconds"] = *t.TolerationSeconds
			}
			tolList = append(tolList, tol)
		}
		if len(tolList) > 0 {
			podSpec["tolerations"] = tolList
		}
	}

	// Active deadline.
	if ads := cfg.GetActiveDeadlineSeconds(); ads > 0 {
		podSpec["activeDeadlineSeconds"] = ads
	}

	if len(podSpec) > 0 {
		patch := map[string]interface{}{"spec": podSpec}
		patchBytes, err := json.Marshal(patch)
		if err == nil {
			tmpl.PodSpecPatch = string(patchBytes)
		}
	}

	// Secret volumes.
	for _, sv := range cfg.GetSecretAsVolume() {
		secretName := resolveStringParam(sv.GetSecretNameParameter(), resolveParam)
		if secretName == "" {
			secretName = sv.GetSecretName() //nolint:staticcheck
		}
		if secretName == "" {
			glog.Warningf("SecretAsVolume with empty/dynamic secretName is not supported with init-container driver; skipping")
			continue
		}
		optional := sv.Optional != nil && *sv.Optional
		tmpl.Volumes = append(tmpl.Volumes, k8score.Volume{
			Name: secretName,
			VolumeSource: k8score.VolumeSource{
				Secret: &k8score.SecretVolumeSource{
					SecretName: secretName,
					Optional:   &optional,
				},
			},
		})
		tmpl.Container.VolumeMounts = append(tmpl.Container.VolumeMounts, k8score.VolumeMount{
			Name:      secretName,
			MountPath: sv.GetMountPath(),
		})
	}

	// Secret env vars.
	for _, se := range cfg.GetSecretAsEnv() {
		secretName := resolveStringParam(se.GetSecretNameParameter(), resolveParam)
		if secretName == "" {
			secretName = se.GetSecretName() //nolint:staticcheck
		}
		if secretName == "" {
			glog.Warningf("SecretAsEnv with empty/dynamic secretName is not supported with init-container driver; skipping")
			continue
		}
		for _, kToEnv := range se.GetKeyToEnv() {
			tmpl.Container.Env = append(tmpl.Container.Env, k8score.EnvVar{
				Name: kToEnv.GetEnvVar(),
				ValueFrom: &k8score.EnvVarSource{
					SecretKeyRef: &k8score.SecretKeySelector{
						LocalObjectReference: k8score.LocalObjectReference{Name: secretName},
						Key:                  kToEnv.GetSecretKey(),
						Optional:             se.Optional,
					},
				},
			})
		}
	}

	// ConfigMap volumes.
	for _, cv := range cfg.GetConfigMapAsVolume() {
		configMapName := resolveStringParam(cv.GetConfigMapNameParameter(), resolveParam)
		if configMapName == "" {
			configMapName = cv.GetConfigMapName() //nolint:staticcheck
		}
		if configMapName == "" {
			continue
		}
		optional := cv.Optional != nil && *cv.Optional
		tmpl.Volumes = append(tmpl.Volumes, k8score.Volume{
			Name: configMapName,
			VolumeSource: k8score.VolumeSource{
				ConfigMap: &k8score.ConfigMapVolumeSource{
					LocalObjectReference: k8score.LocalObjectReference{Name: configMapName},
					Optional:             &optional,
				},
			},
		})
		tmpl.Container.VolumeMounts = append(tmpl.Container.VolumeMounts, k8score.VolumeMount{
			Name:      configMapName,
			MountPath: cv.GetMountPath(),
		})
	}

	// ConfigMap env vars.
	for _, ce := range cfg.GetConfigMapAsEnv() {
		configMapName := resolveStringParam(ce.GetConfigMapNameParameter(), resolveParam)
		if configMapName == "" {
			configMapName = ce.GetConfigMapName() //nolint:staticcheck
		}
		for _, kToEnv := range ce.GetKeyToEnv() {
			tmpl.Container.Env = append(tmpl.Container.Env, k8score.EnvVar{
				Name: kToEnv.GetEnvVar(),
				ValueFrom: &k8score.EnvVarSource{
					ConfigMapKeyRef: &k8score.ConfigMapKeySelector{
						LocalObjectReference: k8score.LocalObjectReference{Name: configMapName},
						Key:                  kToEnv.GetConfigMapKey(),
					},
				},
			})
		}
	}

	// FieldPath env vars.
	for _, fpe := range cfg.GetFieldPathAsEnv() {
		tmpl.Container.Env = append(tmpl.Container.Env, k8score.EnvVar{
			Name: fpe.GetName(),
			ValueFrom: &k8score.EnvVarSource{
				FieldRef: &k8score.ObjectFieldSelector{
					FieldPath: fpe.GetFieldPath(),
				},
			},
		})
	}

	// EmptyDir mounts.
	for _, edm := range cfg.GetEmptyDirMounts() {
		tmpl.Volumes = append(tmpl.Volumes, k8score.Volume{
			Name:         edm.GetVolumeName(),
			VolumeSource: k8score.VolumeSource{EmptyDir: &k8score.EmptyDirVolumeSource{}},
		})
		tmpl.Container.VolumeMounts = append(tmpl.Container.VolumeMounts, k8score.VolumeMount{
			Name:      edm.GetVolumeName(),
			MountPath: edm.GetMountPath(),
		})
	}

	// Shared memory (emptyDir with memory medium).
	if shm := cfg.GetEnabledSharedMemory(); shm != nil {
		shmName := shm.GetVolumeName()
		if shmName == "" {
			shmName = "dshm"
		}
		emptyDir := &k8score.EmptyDirVolumeSource{Medium: k8score.StorageMediumMemory}
		if sizeStr := shm.GetSize(); sizeStr != "" {
			if q, err := resource.ParseQuantity(sizeStr); err == nil {
				emptyDir.SizeLimit = &q
			}
		}
		tmpl.Volumes = append(tmpl.Volumes, k8score.Volume{
			Name:         shmName,
			VolumeSource: k8score.VolumeSource{EmptyDir: emptyDir},
		})
		tmpl.Container.VolumeMounts = append(tmpl.Container.VolumeMounts, k8score.VolumeMount{
			Name:      shmName,
			MountPath: "/dev/shm",
		})
	}

	// PVC mounts — constant and component-input-parameter names are supported;
	// TaskOutputParameter names are skipped since they require runtime resolution.
	for _, pm := range cfg.GetPvcMount() {
		pvcName := resolvePvcName(pm, resolveParam)
		if pvcName == "" {
			continue
		}
		mount := k8score.VolumeMount{
			Name:      pvcName,
			MountPath: pm.GetMountPath(),
		}
		if pm.GetSubPath() != "" {
			mount.SubPath = pm.GetSubPath()
		}
		tmpl.Volumes = append(tmpl.Volumes, k8score.Volume{
			Name: pvcName,
			VolumeSource: k8score.VolumeSource{
				PersistentVolumeClaim: &k8score.PersistentVolumeClaimVolumeSource{
					ClaimName: pvcName,
				},
			},
		})
		tmpl.Container.VolumeMounts = append(tmpl.Container.VolumeMounts, mount)
	}
}

// resolveStringParam extracts a string value from a TaskInputsSpec_InputParameterSpec.
// It handles RuntimeValue constants and ComponentInputParameter (via resolveParam).
// TaskOutputParameter is not supported at compile time and returns "".
func resolveStringParam(param *pipelinespec.TaskInputsSpec_InputParameterSpec, resolveParam func(name string) (string, bool)) string {
	if param == nil {
		return ""
	}
	switch kind := param.GetKind().(type) {
	case *pipelinespec.TaskInputsSpec_InputParameterSpec_RuntimeValue:
		if c := kind.RuntimeValue.GetConstant(); c != nil {
			return c.GetStringValue()
		}
	case *pipelinespec.TaskInputsSpec_InputParameterSpec_ComponentInputParameter:
		if resolveParam != nil {
			if val, ok := resolveParam(kind.ComponentInputParameter); ok {
				return val
			}
		}
	}
	return ""
}

// resolvePvcName extracts the PVC claim name from a PvcMount, resolving
// ComponentInputParameter references via resolveParam.
//
// Resolution priority:
//  1. PvcNameParameter.RuntimeValue.Constant  – compile-time constant via new field
//  2. PvcNameParameter.ComponentInputParameter – resolved via resolveParam
//  3. Deprecated PvcReference.Constant         – compile-time constant string
//  4. Deprecated PvcReference.ComponentInputParameter – resolved via resolveParam
//
// TaskOutputParameter references are always skipped: their values are only known
// at runtime (they come from another task's output) and cannot be embedded at
// compile time.
//
// Returns "" if the name cannot be determined; callers should skip the mount.
func resolvePvcName(pm *kubernetesplatform.PvcMount, resolveParam func(name string) (string, bool)) string {
	if np := pm.GetPvcNameParameter(); np != nil {
		switch kind := np.GetKind().(type) {
		case *pipelinespec.TaskInputsSpec_InputParameterSpec_RuntimeValue:
			// Constant value supplied via the new RuntimeValue field.
			rv := kind.RuntimeValue
			if c := rv.GetConstant(); c != nil {
				return c.GetStringValue()
			}
			// Also handle the deprecated ConstantValue sub-field for backward compatibility.
			if c := rv.GetConstantValue(); c != nil { //nolint:staticcheck
				return c.GetStringValue()
			}
		case *pipelinespec.TaskInputsSpec_InputParameterSpec_ComponentInputParameter:
			paramName := kind.ComponentInputParameter
			if paramName == "" {
				return ""
			}
			if resolveParam != nil {
				if val, ok := resolveParam(paramName); ok {
					return val
				}
			}
			glog.Warningf("PvcMount: ComponentInputParameter %q could not be resolved at compile time; skipping", paramName)
			return ""
		case *pipelinespec.TaskInputsSpec_InputParameterSpec_TaskOutputParameter:
			glog.Warningf("PvcMount: TaskOutputParameter PVC names are not supported with the init-container driver; skipping")
			return ""
		}
		return ""
	}

	// Deprecated PvcReference oneof fields (backward compatibility with older pipeline specs).
	if name := pm.GetConstant(); name != "" { //nolint:staticcheck
		return name
	}
	if paramName := pm.GetComponentInputParameter(); paramName != "" { //nolint:staticcheck
		if resolveParam != nil {
			if val, ok := resolveParam(paramName); ok {
				return val
			}
		}
		glog.Warningf("PvcMount: ComponentInputParameter %q could not be resolved at compile time; skipping", paramName)
		return ""
	}
	if pm.GetTaskOutputParameter() != nil { //nolint:staticcheck
		glog.Warningf("PvcMount: TaskOutputParameter PVC names are not supported with the init-container driver; skipping")
		return ""
	}
	return ""
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
