// Copyright 2021-2023 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package driver

import (
	"fmt"
	"slices"
	"strings"

	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/component"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	"github.com/kubeflow/pipelines/kubernetes_platform/go/kubernetesplatform"
	"google.golang.org/protobuf/encoding/protojson"
	k8score "k8s.io/api/core/v1"
	k8sres "k8s.io/apimachinery/pkg/api/resource"
)

// Driver options
type Options struct {
	// required, pipeline context name
	PipelineName string
	// required, KFP run ID
	RunID string
	// required, Component spec
	Component *pipelinespec.ComponentSpec
	// optional, iteration index. -1 means not an iteration.
	IterationIndex int

	// optional, required only by root DAG driver
	RuntimeConfig *pipelinespec.PipelineJob_RuntimeConfig
	Namespace     string

	// optional, required by non-root drivers
	Task           *pipelinespec.PipelineTaskSpec
	DAGExecutionID int64

	// optional, required only by container driver
	Container *pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec

	// optional, allows to specify kubernetes-specific executor config
	KubernetesExecutorConfig *kubernetesplatform.KubernetesExecutorConfig

	// optional, required only if the {{$.pipeline_job_resource_name}} placeholder is used
	RunName string
	// optional, required only if the {{$.pipeline_job_name}} placeholder is used
	RunDisplayName string

	PipelineLogLevel string

	PublishLogs string

	CacheDisabled bool

	DriverType string
}

// Identifying information used for error messages
func (o Options) info() string {
	msg := fmt.Sprintf("pipelineName=%v, runID=%v", o.PipelineName, o.RunID)
	if o.Task.GetTaskInfo().GetName() != "" {
		msg = msg + fmt.Sprintf(", taskDisplayName=%q", o.Task.GetTaskInfo().GetName())
	}
	if o.Task.GetTaskInfo().GetTaskName() != "" {
		msg = msg + fmt.Sprintf(", taskName=%q", o.Task.GetTaskInfo().GetTaskName())
	}
	if o.Task.GetComponentRef().GetName() != "" {
		msg = msg + fmt.Sprintf(", component=%q", o.Task.GetComponentRef().GetName())
	}
	if o.DAGExecutionID != 0 {
		msg = msg + fmt.Sprintf(", dagExecutionID=%v", o.DAGExecutionID)
	}
	if o.IterationIndex >= 0 {
		msg = msg + fmt.Sprintf(", iterationIndex=%v", o.IterationIndex)
	}
	if o.RuntimeConfig != nil {
		msg = msg + ", runtimeConfig" // this only means runtimeConfig is not empty
	}
	if o.Component.GetImplementation() != nil {
		msg = msg + ", componentSpec" // this only means componentSpec is not empty
	}
	if o.KubernetesExecutorConfig != nil {
		msg = msg + ", KubernetesExecutorConfig" // this only means KubernetesExecutorConfig is not empty
	}
	return msg
}

type Execution struct {
	ID             int64
	ExecutorInput  *pipelinespec.ExecutorInput
	IterationCount *int  // number of iterations, -1 means not an iterator
	Condition      *bool // true -> trigger the task, false -> not trigger the task, nil -> the task is unconditional

	// only specified when this is a Container execution
	Cached       *bool
	PodSpecPatch string
}

func (e *Execution) WillTrigger() bool {
	if e == nil || e.Condition == nil {
		return true
	}
	return *e.Condition
}

// getPodResource will accept the new field that accepts placeholders (e.g. resourceMemoryLimit) and the old float64
// field (e.g. memoryLimit) and return the resolved value as a Quantity. If the returned Quantity is nil, it was not set
// by the user. If the new field is set, the old field is ignored.
func getPodResource(
	new string, old float64, executorInput *pipelinespec.ExecutorInput, oldFmtStr string,
) (*k8sres.Quantity, error) {
	var resolved string

	if new != "" {
		var err error

		resolved, err = resolvePodSpecInputRuntimeParameter(new, executorInput)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve executor input when retrieving pod resource: %w", err)
		}
	} else if old != 0 {
		resolved = fmt.Sprintf(oldFmtStr, old)
	} else {
		return nil, nil
	}

	q, err := k8sres.ParseQuantity(resolved)
	if err != nil {
		return nil, err
	}

	return &q, nil
}

// initPodSpecPatch generates a strategic merge patch for pod spec, it is merged
// to container base template generated in compiler/container.go. Therefore, only
// dynamic values are patched here. The volume mounts / configmap mounts are
// defined in compiler, because they are static.
func initPodSpecPatch(
	container *pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec,
	componentSpec *pipelinespec.ComponentSpec,
	executorInput *pipelinespec.ExecutorInput,
	executionID int64,
	pipelineName string,
	runID string,
	pipelineLogLevel string,
	publishLogs string,
	cacheDisabled string,
) (*k8score.PodSpec, error) {
	executorInputJSON, err := protojson.Marshal(executorInput)
	if err != nil {
		return nil, fmt.Errorf("failed to init podSpecPatch: %w", err)
	}
	componentJSON, err := protojson.Marshal(componentSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to init podSpecPatch: %w", err)
	}

	// Convert environment variables
	userEnvVar := make([]k8score.EnvVar, 0)
	for _, envVar := range container.GetEnv() {
		userEnvVar = append(userEnvVar, k8score.EnvVar{Name: envVar.GetName(), Value: envVar.GetValue()})
	}

	userEnvVar = append(userEnvVar, proxy.GetConfig().GetEnvVars()...)

	userCmdArgs := make([]string, 0, len(container.Command)+len(container.Args))
	userCmdArgs = append(userCmdArgs, container.Command...)
	userCmdArgs = append(userCmdArgs, container.Args...)
	launcherCmd := []string{
		component.KFPLauncherPath,
		// TODO(Bobgy): no need to pass pipeline_name and run_id, these info can be fetched via pipeline context and pipeline run context which have been created by root DAG driver.
		"--pipeline_name", pipelineName,
		"--run_id", runID,
		"--execution_id", fmt.Sprintf("%v", executionID),
		"--executor_input", string(executorInputJSON),
		"--component_spec", string(componentJSON),
		"--pod_name",
		fmt.Sprintf("$(%s)", component.EnvPodName),
		"--pod_uid",
		fmt.Sprintf("$(%s)", component.EnvPodUID),
		"--mlmd_server_address",
		fmt.Sprintf("$(%s)", component.EnvMetadataHost),
		"--mlmd_server_port",
		fmt.Sprintf("$(%s)", component.EnvMetadataPort),
		"--publish_logs", publishLogs,
	}
	if cacheDisabled == "true" {
		launcherCmd = append(launcherCmd, "--cache_disabled")
	}
	if pipelineLogLevel != "1" {
		// Add log level to user code launcher if not default (set to 1)
		launcherCmd = append(launcherCmd, "--log_level", pipelineLogLevel)
	}
	if publishLogs == "true" {
		launcherCmd = append(launcherCmd, "--publish_logs", publishLogs)
	}
	launcherCmd = append(launcherCmd, "--") // separater before user command and args
	res := k8score.ResourceRequirements{
		Limits:   map[k8score.ResourceName]k8sres.Quantity{},
		Requests: map[k8score.ResourceName]k8sres.Quantity{},
	}

	memoryLimit, err := getPodResource(
		container.GetResources().GetResourceMemoryLimit(),
		container.GetResources().GetMemoryLimit(),
		executorInput,
		"%vG",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to init podSpecPatch: %w", err)
	}
	if memoryLimit != nil {
		res.Limits[k8score.ResourceMemory] = *memoryLimit
	}

	memoryRequest, err := getPodResource(
		container.GetResources().GetResourceMemoryRequest(),
		container.GetResources().GetMemoryRequest(),
		executorInput,
		"%vG",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to init podSpecPatch: %w", err)
	}
	if memoryRequest != nil {
		res.Requests[k8score.ResourceMemory] = *memoryRequest
	}

	cpuLimit, err := getPodResource(
		container.GetResources().GetResourceCpuLimit(),
		container.GetResources().GetCpuLimit(),
		executorInput,
		"%v",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to init podSpecPatch: %w", err)
	}
	if cpuLimit != nil {
		res.Limits[k8score.ResourceCPU] = *cpuLimit
	}

	cpuRequest, err := getPodResource(
		container.GetResources().GetResourceCpuRequest(),
		container.GetResources().GetCpuRequest(),
		executorInput,
		"%v",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to init podSpecPatch: %w", err)
	}
	if cpuRequest != nil {
		res.Requests[k8score.ResourceCPU] = *cpuRequest
	}

	accelerator := container.GetResources().GetAccelerator()
	if accelerator != nil {
		var acceleratorType string
		if accelerator.GetResourceType() != "" {
			acceleratorType, err = resolvePodSpecInputRuntimeParameter(accelerator.GetResourceType(), executorInput)
			if err != nil {
				return nil, fmt.Errorf("failed to init podSpecPatch: %w", err)
			}
		} else if accelerator.GetType() != "" {
			acceleratorType = accelerator.GetType()
		}

		var acceleratorCount string

		if accelerator.GetResourceCount() != "" {
			var err error

			acceleratorCount, err = resolvePodSpecInputRuntimeParameter(accelerator.GetResourceCount(), executorInput)
			if err != nil {
				return nil, fmt.Errorf("failed to init podSpecPatch: %w", err)
			}
		} else if accelerator.Count > 0 {
			acceleratorCount = fmt.Sprintf("%v", accelerator.GetCount())
		}

		if acceleratorType != "" && acceleratorCount != "" {
			q, err := k8sres.ParseQuantity(acceleratorCount)
			if err != nil {
				return nil, fmt.Errorf("failed to init podSpecPatch: %w", err)
			}
			res.Limits[k8score.ResourceName(acceleratorType)] = q
		}
	}

	containerImage, err := resolvePodSpecInputRuntimeParameter(container.Image, executorInput)
	if err != nil {
		return nil, fmt.Errorf("failed to init podSpecPatch: %w", err)
	}
	podSpec := &k8score.PodSpec{
		Containers: []k8score.Container{{
			Name:      "main", // argo task user container is always called "main"
			Command:   launcherCmd,
			Args:      userCmdArgs,
			Image:     containerImage,
			Resources: res,
			Env:       userEnvVar,
		}},
	}

	addModelcarsToPodSpec(executorInput.GetInputs().GetArtifacts(), userEnvVar, podSpec)

	return podSpec, nil
}

// addModelcarsToPodSpec will patch the pod spec if there are any input artifacts in the Modelcar format.
// Much of this logic is based on KServe:
// https://github.com/kserve/kserve/blob/v0.14.1/pkg/webhook/admission/pod/storage_initializer_injector.go#L131
func addModelcarsToPodSpec(
	artifacts map[string]*pipelinespec.ArtifactList,
	userEnvVar []k8score.EnvVar,
	podSpec *k8score.PodSpec,
) {
	// We need to add Modelcar containers and volumes in a deterministic order so that we can have stable naming of
	// containers and volumes. The approach taken is sorting by input artifact name and then leveraging the index
	// as a suffix to Modelcar containers and volumes added to the pod spec. The artifact name cannot be directly used
	// as it may not be a compatible Kubernetes object name.
	modelcarArtifacts := map[string]*pipelinespec.RuntimeArtifact{}
	modelcarArtifactNames := []string{}

	for name, artifactList := range artifacts {
		if len(artifactList.Artifacts) == 0 {
			continue
		}
		// Following the convention of downloadArtifacts in the launcher to look at the entire list.
		for index, artifact := range artifactList.Artifacts {
			inputArtifact := artifact

			// This should ideally verify that this is also a model input artifact, but this metadata doesn't seem to
			// be set on inputArtifact.
			if !strings.HasPrefix(inputArtifact.Uri, "oci://") {
				continue
			}

			artifactName := fmt.Sprintf("%s-%d", name, index)
			modelcarArtifacts[artifactName] = inputArtifact
			modelcarArtifactNames = append(modelcarArtifactNames, artifactName)
		}
	}

	slices.Sort(modelcarArtifactNames)

	for i, name := range modelcarArtifactNames {
		inputArtifact := modelcarArtifacts[name]

		localPath, err := component.LocalPathForURI(inputArtifact.Uri)
		if err != nil {
			continue
		}

		// If there is at least one Modelcar image, then shareProcessNamespace must be enabled.
		trueVal := true
		podSpec.ShareProcessNamespace = &trueVal

		image := strings.TrimPrefix(inputArtifact.Uri, "oci://")

		podSpec.InitContainers = append(
			podSpec.InitContainers,
			k8score.Container{
				Name:  fmt.Sprintf("oci-prepull-%d", i),
				Image: image,
				Command: []string{
					"sh",
					"-c",
					// Check that the expected models directory exists
					// Taken from KServe:
					// https://github.com/kserve/kserve/blob/v0.14.1/pkg/webhook/admission/pod/storage_initializer_injector.go#L732
					"echo 'Pre-fetching modelcar " + image + ": ' && [ -d /models ] && " +
						"[ \"$$(ls -A /models)\" ] && echo 'OK ... Prefetched and valid (/models exists)' || " +
						"(echo 'NOK ... Prefetched but modelcar is invalid (/models does not exist or is empty)' && " +
						" exit 1)",
				},
				Env:                      userEnvVar,
				TerminationMessagePolicy: k8score.TerminationMessageFallbackToLogsOnError,
			},
		)

		volumeName := fmt.Sprintf("oci-%d", i)

		podSpec.Volumes = append(
			podSpec.Volumes,
			k8score.Volume{
				Name: volumeName,
				VolumeSource: k8score.VolumeSource{
					EmptyDir: &k8score.EmptyDirVolumeSource{},
				},
			},
		)

		mountPath := strings.TrimSuffix(localPath, "/models")

		emptyDirVolumeMount := k8score.VolumeMount{
			Name:      volumeName,
			MountPath: mountPath,
			SubPath:   strings.TrimPrefix(mountPath, "/oci/"),
		}

		podSpec.Containers[0].VolumeMounts = append(podSpec.Containers[0].VolumeMounts, emptyDirVolumeMount)

		podSpec.Containers = append(
			podSpec.Containers,
			k8score.Container{
				Name:            fmt.Sprintf("oci-%d", i),
				Image:           image,
				ImagePullPolicy: "IfNotPresent",
				Env:             userEnvVar,
				VolumeMounts:    []k8score.VolumeMount{emptyDirVolumeMount},
				Command: []string{
					"sh",
					"-c",
					// $$$$ gets escaped by YAML to $$, which is the current PID
					// This container will sleep until the main container finishes execution and
					// communicates its exit via a file creation, at which point this container
					// will then also exit.
					// This approach is taken instead of having the main container send a SIGHUP to the
					// sleep process to avoid the need for the SYS_PTRACE capability which is not always available
					// depending on the security context restrictions.
					// This approach is inspired by KServe:
					// https://github.com/kserve/kserve/blob/v0.14.1/pkg/webhook/admission/pod/storage_initializer_injector.go#L732
					fmt.Sprintf(
						"ln -s /proc/$$$$/root/models \"%s\" && "+
							"echo \"Running Modelcar container...\" && "+
							"until [ -f \"%s/launcher-complete\" ]; do sleep 1; done",
						localPath, mountPath,
					),
				},
				TerminationMessagePolicy: k8score.TerminationMessageFallbackToLogsOnError,
			},
		)
	}
}

func validateNonRoot(opts Options) error {
	if opts.PipelineName == "" {
		return fmt.Errorf("pipeline name is required")
	}
	if opts.RunID == "" {
		return fmt.Errorf("KFP run ID is required")
	}
	if opts.Component == nil {
		return fmt.Errorf("component spec is required")
	}
	if opts.Task.GetTaskInfo().GetName() == "" {
		return fmt.Errorf("task spec is required")
	}
	if opts.RuntimeConfig != nil {
		return fmt.Errorf("runtime config is unnecessary")
	}
	if opts.DAGExecutionID == 0 {
		return fmt.Errorf("DAG execution ID is required")
	}
	return nil
}

// provisionOutputs prepares output references that will get saved to MLMD.
func provisionOutputs(
	pipelineRoot,
	taskName string,
	outputsSpec *pipelinespec.ComponentOutputsSpec,
	outputURISalt string,
	publishOutput string,
) *pipelinespec.ExecutorInput_Outputs {
	outputs := &pipelinespec.ExecutorInput_Outputs{
		Artifacts:  make(map[string]*pipelinespec.ArtifactList),
		Parameters: make(map[string]*pipelinespec.ExecutorInput_OutputParameter),
		OutputFile: component.OutputMetadataFilepath,
	}
	artifacts := outputsSpec.GetArtifacts()

	// TODO: Check if there's a more idiomatic way to handle this.
	if publishOutput == "true" {
		// Add a placeholder for a log artifact that will be written to by the
		// subsequent executor.
		if artifacts == nil {
			artifacts = make(map[string]*pipelinespec.ComponentOutputsSpec_ArtifactSpec)
		}
		artifacts["executor-logs"] = &pipelinespec.ComponentOutputsSpec_ArtifactSpec{
			ArtifactType: &pipelinespec.ArtifactTypeSchema{
				Kind: &pipelinespec.ArtifactTypeSchema_SchemaTitle{
					SchemaTitle: "system.Artifact",
				},
			},
		}
	}

	for name, artifact := range artifacts {
		outputs.Artifacts[name] = &pipelinespec.ArtifactList{
			Artifacts: []*pipelinespec.RuntimeArtifact{
				{
					// Do not preserve the query string for output artifacts, as otherwise
					// they'd appear in file and artifact names.
					Uri:      metadata.GenerateOutputURI(pipelineRoot, []string{taskName, outputURISalt, name}, false),
					Type:     artifact.GetArtifactType(),
					Metadata: artifact.GetMetadata(),
				},
			},
		}
	}

	for name := range outputsSpec.GetParameters() {
		outputs.Parameters[name] = &pipelinespec.ExecutorInput_OutputParameter{
			OutputFile: fmt.Sprintf("/tmp/kfp/outputs/%s", name),
		}
	}

	return outputs
}
