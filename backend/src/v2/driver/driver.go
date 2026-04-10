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
	"path/filepath"
	"strings"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/component"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	"github.com/kubeflow/pipelines/kubernetes_platform/go/kubernetesplatform"
	"google.golang.org/protobuf/types/known/structpb"
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

	// optional, required only if the {{$.pipeline_job_resource_name}} placeholder is used or the run uses a workspace
	RunName string
	// optional, required only if the {{$.pipeline_job_name}} placeholder is used
	RunDisplayName string

	PipelineLogLevel string

	PublishLogs string

	CacheDisabled bool

	DriverType string

	TaskName string // the original name of the task, used for input resolution

	// set to true if ml pipeline server is serving over tls
	MLPipelineTLSEnabled bool

	// set to true if metadata server is serving over tls
	MLMDTLSEnabled bool

	MLPipelineServerAddress string

	MLPipelineServerPort string

	MLMDServerAddress string

	MLMDServerPort string

	CaCertPath string

	PipelineJobCreateTimeUTC string

	PipelineJobScheduleTimeUTC string

	// Admin-configured default runAsUser for user containers. Nil means not set.
	DefaultRunAsUser *int64
	// Admin-configured default runAsGroup for user containers. Nil means not set.
	DefaultRunAsGroup *int64
	// Admin-configured default runAsNonRoot for user containers. Nil means not set.
	DefaultRunAsNonRoot *bool
}

// TaskConfig needs to stay aligned with the TaskConfig in the SDK.
type TaskConfig struct {
	Affinity     *k8score.Affinity            `json:"affinity"`
	Tolerations  []k8score.Toleration         `json:"tolerations"`
	NodeSelector map[string]string            `json:"nodeSelector"`
	Env          []k8score.EnvVar             `json:"env"`
	Volumes      []k8score.Volume             `json:"volumes"`
	VolumeMounts []k8score.VolumeMount        `json:"volumeMounts"`
	Resources    k8score.ResourceRequirements `json:"resources"`
}

// Identifying information used for error messages
func (o Options) info() string {
	msg := fmt.Sprintf("pipelineName=%v, runID=%v", o.PipelineName, o.RunID)
	if o.Task.GetTaskInfo().GetName() != "" {
		msg = msg + fmt.Sprintf(", taskDisplayName=%q", o.Task.GetTaskInfo().GetName())
	}
	if o.TaskName != "" {
		msg = msg + fmt.Sprintf(", taskName=%q", o.TaskName)
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
	Cached *bool
	// UserCommand and UserArgs hold the resolved user container command and args.
	// They are written to the shared emptyDir volume for the launcher to read.
	UserCommand []string
	UserArgs    []string
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

// getTaskConfigOptions inspects the component's task config passthroughs and returns two maps:
// 1) fields enabled for passthrough
// 2) fields that should apply to the task pod
//
// If the component does not specify a passthrough, then all fields apply to the task pod and no fields are passthrough
// enabled.
func getTaskConfigOptions(
	componentSpec *pipelinespec.ComponentSpec,
) (map[pipelinespec.TaskConfigPassthroughType_TaskConfigPassthroughTypeEnum]bool,
	map[pipelinespec.TaskConfigPassthroughType_TaskConfigPassthroughTypeEnum]bool,
) {
	passthroughEnabled := map[pipelinespec.TaskConfigPassthroughType_TaskConfigPassthroughTypeEnum]bool{}
	// setOnTask contains all possible fields even if they are not in the passthrough list.
	setOnPod := map[pipelinespec.TaskConfigPassthroughType_TaskConfigPassthroughTypeEnum]bool{
		pipelinespec.TaskConfigPassthroughType_RESOURCES:                true,
		pipelinespec.TaskConfigPassthroughType_ENV:                      true,
		pipelinespec.TaskConfigPassthroughType_KUBERNETES_AFFINITY:      true,
		pipelinespec.TaskConfigPassthroughType_KUBERNETES_TOLERATIONS:   true,
		pipelinespec.TaskConfigPassthroughType_KUBERNETES_NODE_SELECTOR: true,
		pipelinespec.TaskConfigPassthroughType_KUBERNETES_VOLUMES:       true,
	}

	if componentSpec == nil {
		return passthroughEnabled, setOnPod
	}

	// If the component specifies a passthrough, then we don't set fields on the pod unless apply_to_task
	// is true.
	if len(componentSpec.GetTaskConfigPassthroughs()) != 0 {
		for field := range setOnPod {
			passthroughEnabled[field] = false
		}
	}

	for _, pt := range componentSpec.GetTaskConfigPassthroughs() {
		field := pt.GetField()
		passthroughEnabled[field] = true
		setOnPod[field] = pt.GetApplyToTask()
	}

	return passthroughEnabled, setOnPod
}

// needsWorkspaceMount checks if the component needs workspace mounting based on input parameters and artifacts.
func needsWorkspaceMount(executorInput *pipelinespec.ExecutorInput) bool {
	for _, param := range executorInput.GetInputs().GetParameterValues() {
		if strVal, ok := param.GetKind().(*structpb.Value_StringValue); ok {
			if strings.Contains(strVal.StringValue, "{{$.workspace_path}}") {
				return true
			}
			if strings.Contains(strVal.StringValue, component.WorkspaceMountPath) {
				return true
			}
		}
	}
	for _, artifactList := range executorInput.GetInputs().GetArtifacts() {
		if len(artifactList.Artifacts) == 0 {
			continue
		}
		artifact := artifactList.Artifacts[0]
		if artifact.Metadata != nil {
			if workspaceVal, ok := artifact.Metadata.Fields["_kfp_workspace"]; ok {
				if boolVal, ok := workspaceVal.GetKind().(*structpb.Value_BoolValue); ok && boolVal.BoolValue {
					return true
				}
			}
		}
	}
	return false
}

// getWorkspaceMount returns the workspace volume and volume mount for a given PVC name.
func getWorkspaceMount(pvcName string) (k8score.Volume, k8score.VolumeMount) {
	return k8score.Volume{
			Name: component.WorkspaceVolumeName,
			VolumeSource: k8score.VolumeSource{
				PersistentVolumeClaim: &k8score.PersistentVolumeClaimVolumeSource{
					ClaimName: pvcName,
				},
			},
		}, k8score.VolumeMount{
			Name:      component.WorkspaceVolumeName,
			MountPath: component.WorkspaceMountPath,
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

	// Compute a task-root remote URI that will serve as the base for all
	// output artifacts and the executor output file. This enables Pythonic
	// artifacts (dsl.get_uri) by allowing the SDK to infer the task root from
	// the executor output file's directory (set below) and convert it back to
	// a remote URI at runtime.
	taskRootRemote := metadata.GenerateOutputURI(pipelineRoot, []string{taskName, outputURISalt}, false)

	// Set per-artifact output URIs under the task root.
	for name, artifact := range artifacts {
		outputs.Artifacts[name] = &pipelinespec.ArtifactList{
			Artifacts: []*pipelinespec.RuntimeArtifact{
				{
					// Required by Pythonic artifacts to avoid a key error in the SDK.
					Name: name,
					// Do not preserve the query string for output artifacts, as otherwise
					// they'd appear in file and artifact names.
					Uri:      metadata.GenerateOutputURI(taskRootRemote, []string{name}, false),
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

	// Place the executor output file under localTaskRoot to enable Pythonic artifacts. The SDK's pythonic artifact
	// runtime derives CONTAINER_TASK_ROOT from the directory of OutputFile to use it in dsl.get_uri.
	if localTaskRoot, err := component.LocalPathForURI(taskRootRemote); err == nil {
		outputs.OutputFile = filepath.Join(localTaskRoot, "output_metadata.json")
	} else {
		// Fallback to legacy path if the pipeline root scheme is not recognized.
		outputs.OutputFile = component.OutputMetadataFilepath
	}

	return outputs
}

func validateVolumeMounts(podSpec *k8score.PodSpec) error {
	// Validate that no user volume mounts conflict with the workspace mount path or volume name
	for _, container := range podSpec.Containers {
		for _, mount := range container.VolumeMounts {
			if strings.HasPrefix(mount.MountPath, component.WorkspaceMountPath) {
				return fmt.Errorf("user volume mount at %s conflicts with workspace mount at %s", mount.MountPath, component.WorkspaceMountPath)
			}

			if mount.Name == component.WorkspaceVolumeName {
				return fmt.Errorf("user volume mount name %s conflicts with workspace volume name %s", mount.Name, component.WorkspaceVolumeName)
			}
		}
	}

	return nil
}
