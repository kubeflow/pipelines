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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/objectstore"

	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/google/uuid"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	api "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/v2/cacheutils"
	"github.com/kubeflow/pipelines/backend/src/v2/component"
	"github.com/kubeflow/pipelines/backend/src/v2/config"
	"github.com/kubeflow/pipelines/backend/src/v2/expression"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	"github.com/kubeflow/pipelines/kubernetes_platform/go/kubernetesplatform"
	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	k8score "k8s.io/api/core/v1"
	k8sres "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var dummyImages = map[string]string{
	"argostub/createpvc": "create PVC",
	"argostub/deletepvc": "delete PVC",
}

var ErrResolvedParameterNull = errors.New("the resolved input parameter is null")

// TODO(capri-xiyue): Move driver to component package
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
}

// Identifying information used for error messages
func (o Options) info() string {
	msg := fmt.Sprintf("pipelineName=%v, runID=%v", o.PipelineName, o.RunID)
	if o.Task.GetTaskInfo().GetName() != "" {
		msg = msg + fmt.Sprintf(", task=%q", o.Task.GetTaskInfo().GetName())
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

func RootDAG(ctx context.Context, opts Options, mlmd *metadata.Client) (execution *Execution, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("driver.RootDAG(%s) failed: %w", opts.info(), err)
		}
	}()
	b, _ := json.Marshal(opts)
	glog.V(4).Info("RootDAG opts: ", string(b))
	err = validateRootDAG(opts)
	if err != nil {
		return nil, err
	}
	// TODO(v2): in pipeline spec, rename GCS output directory to pipeline root.
	pipelineRoot := opts.RuntimeConfig.GetGcsOutputDirectory()

	restConfig, err := util.GetKubernetesConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize kubernetes client: %w", err)
	}
	k8sClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize kubernetes client set: %w", err)
	}
	cfg, err := config.FromConfigMap(ctx, k8sClient, opts.Namespace)
	if err != nil {
		return nil, err
	}

	storeSessionInfo := objectstore.SessionInfo{}
	if pipelineRoot != "" {
		glog.Infof("PipelineRoot=%q", pipelineRoot)
	} else {
		pipelineRoot = cfg.DefaultPipelineRoot()
		glog.Infof("PipelineRoot=%q from default config", pipelineRoot)
	}
	storeSessionInfo, err = cfg.GetStoreSessionInfo(pipelineRoot)
	if err != nil {
		return nil, err
	}
	storeSessionInfoJSON, err := json.Marshal(storeSessionInfo)
	if err != nil {
		return nil, err
	}
	storeSessionInfoStr := string(storeSessionInfoJSON)
	// TODO(Bobgy): fill in run resource.
	pipeline, err := mlmd.GetPipeline(ctx, opts.PipelineName, opts.RunID, opts.Namespace, "run-resource", pipelineRoot, storeSessionInfoStr)
	if err != nil {
		return nil, err
	}

	executorInput := &pipelinespec.ExecutorInput{
		Inputs: &pipelinespec.ExecutorInput_Inputs{
			ParameterValues: opts.RuntimeConfig.GetParameterValues(),
		},
	}
	// TODO(Bobgy): validate executorInput matches component spec types
	ecfg, err := metadata.GenerateExecutionConfig(executorInput)
	if err != nil {
		return nil, err
	}
	ecfg.ExecutionType = metadata.DagExecutionTypeName
	ecfg.Name = fmt.Sprintf("run/%s", opts.RunID)
	exec, err := mlmd.CreateExecution(ctx, pipeline, ecfg)
	if err != nil {
		return nil, err
	}
	glog.Infof("Created execution: %s", exec)
	// No need to return ExecutorInput, because tasks in the DAG will resolve
	// needed info from MLMD.
	return &Execution{ID: exec.GetID()}, nil
}

func validateRootDAG(opts Options) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("invalid root DAG driver args: %w", err)
		}
	}()
	if opts.PipelineName == "" {
		return fmt.Errorf("pipeline name is required")
	}
	if opts.RunID == "" {
		return fmt.Errorf("KFP run ID is required")
	}
	if opts.Component == nil {
		return fmt.Errorf("component spec is required")
	}
	if opts.RuntimeConfig == nil {
		return fmt.Errorf("runtime config is required")
	}
	if opts.Namespace == "" {
		return fmt.Errorf("namespace is required")
	}
	if opts.Task.GetTaskInfo().GetName() != "" {
		return fmt.Errorf("task spec is unnecessary")
	}
	if opts.DAGExecutionID != 0 {
		return fmt.Errorf("DAG execution ID is unnecessary")
	}
	if opts.Container != nil {
		return fmt.Errorf("container spec is unnecessary")
	}
	if opts.IterationIndex >= 0 {
		return fmt.Errorf("iteration index is unnecessary")
	}
	return nil
}

func Container(ctx context.Context, opts Options, mlmd *metadata.Client, cacheClient *cacheutils.Client) (execution *Execution, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("driver.Container(%s) failed: %w", opts.info(), err)
		}
	}()
	b, _ := json.Marshal(opts)
	glog.V(4).Info("Container opts: ", string(b))
	err = validateContainer(opts)
	if err != nil {
		return nil, err
	}
	var iterationIndex *int
	if opts.IterationIndex >= 0 {
		index := opts.IterationIndex
		iterationIndex = &index
	}
	// TODO(Bobgy): there's no need to pass any parameters, because pipeline
	// and pipeline run context have been created by root DAG driver.
	pipeline, err := mlmd.GetPipeline(ctx, opts.PipelineName, opts.RunID, "", "", "", "")
	if err != nil {
		return nil, err
	}
	dag, err := mlmd.GetDAG(ctx, opts.DAGExecutionID)
	if err != nil {
		return nil, err
	}
	glog.Infof("parent DAG: %+v", dag.Execution)
	expr, err := expression.New()
	if err != nil {
		return nil, err
	}
	inputs, err := resolveInputs(ctx, dag, iterationIndex, pipeline, opts, mlmd, expr)
	if err != nil {
		return nil, err
	}

	executorInput := &pipelinespec.ExecutorInput{
		Inputs: inputs,
	}
	execution = &Execution{ExecutorInput: executorInput}
	condition := opts.Task.GetTriggerPolicy().GetCondition()
	if condition != "" {
		willTrigger, err := expr.Condition(executorInput, condition)
		if err != nil {
			return execution, err
		}
		execution.Condition = &willTrigger
	}

	// When the container image is a dummy image, there is no launcher for this
	// task. This happens when this task is created to implement a
	// Kubernetes-specific configuration, i.e., there is no user container to
	// run. It publishes execution details to mlmd in driver and takes care of
	// caching, which are usually done in launcher. We also skip creating the
	// podspecpatch in these cases.
	_, isKubernetesPlatformOp := dummyImages[opts.Container.Image]
	if isKubernetesPlatformOp {
		// To be consistent with other artifacts, the driver registers log
		// artifacts to MLMD and the launcher publishes them to the object
		// store. This pattern does not work for kubernetesPlatformOps because
		// they have no launcher. There's no point in registering logs that
		// won't be published. Consequently, when we know we're dealing with
		// kubernetesPlatformOps, we set publishLogs to "false". We can amend
		// this when we update the driver to publish logs directly.
		opts.PublishLogs = "false"
	}

	if execution.WillTrigger() {
		executorInput.Outputs = provisionOutputs(
			pipeline.GetPipelineRoot(),
			opts.Task.GetTaskInfo().GetName(),
			opts.Component.GetOutputDefinitions(),
			uuid.NewString(),
			opts.PublishLogs,
		)
	}

	ecfg, err := metadata.GenerateExecutionConfig(executorInput)
	if err != nil {
		return execution, err
	}
	ecfg.TaskName = opts.Task.GetTaskInfo().GetName()
	ecfg.ExecutionType = metadata.ContainerExecutionTypeName
	ecfg.ParentDagID = dag.Execution.GetID()
	ecfg.IterationIndex = iterationIndex
	ecfg.NotTriggered = !execution.WillTrigger()

	if isKubernetesPlatformOp {
		return execution, kubernetesPlatformOps(ctx, mlmd, cacheClient, execution, ecfg, &opts)
	}

	// Generate fingerprint and MLMD ID for cache
	fingerPrint, cachedMLMDExecutionID, err := getFingerPrintsAndID(execution, &opts, cacheClient)
	if err != nil {
		return execution, err
	}
	ecfg.CachedMLMDExecutionID = cachedMLMDExecutionID
	ecfg.FingerPrint = fingerPrint

	// TODO(Bobgy): change execution state to pending, because this is driver, execution hasn't started.
	createdExecution, err := mlmd.CreateExecution(ctx, pipeline, ecfg)

	if err != nil {
		return execution, err
	}
	glog.Infof("Created execution: %s", createdExecution)
	execution.ID = createdExecution.GetID()
	if !execution.WillTrigger() {
		return execution, nil
	}

	// Use cache and skip launcher if all contions met:
	// (1) Cache is enabled
	// (2) CachedMLMDExecutionID is non-empty, which means a cache entry exists
	cached := false
	execution.Cached = &cached
	if opts.Task.GetCachingOptions().GetEnableCache() && ecfg.CachedMLMDExecutionID != "" {
		executorOutput, outputArtifacts, err := reuseCachedOutputs(ctx, execution.ExecutorInput, mlmd, ecfg.CachedMLMDExecutionID)
		if err != nil {
			return execution, err
		}
		// TODO(Bobgy): upload output artifacts.
		// TODO(Bobgy): when adding artifacts, we will need execution.pipeline to be non-nil, because we need
		// to publish output artifacts to the context too.
		if err := mlmd.PublishExecution(ctx, createdExecution, executorOutput.GetParameterValues(), outputArtifacts, pb.Execution_CACHED); err != nil {
			return execution, fmt.Errorf("failed to publish cached execution: %w", err)
		}
		glog.Infof("Use cache for task %s", opts.Task.GetTaskInfo().GetName())
		*execution.Cached = true
		return execution, nil
	}

	podSpec, err := initPodSpecPatch(
		opts.Container,
		opts.Component,
		executorInput,
		execution.ID,
		opts.PipelineName,
		opts.RunID,
		opts.PipelineLogLevel,
		opts.PublishLogs,
	)
	if err != nil {
		return execution, err
	}
	if opts.KubernetesExecutorConfig != nil {
		inputParams, _, err := dag.Execution.GetParameters()
		if err != nil {
			return nil, fmt.Errorf("failed to fetch input parameters from execution: %w", err)
		}
		err = extendPodSpecPatch(ctx, podSpec, opts, dag, pipeline, mlmd, inputParams)
		if err != nil {
			return execution, err
		}
	}
	podSpecPatchBytes, err := json.Marshal(podSpec)
	if err != nil {
		return execution, fmt.Errorf("JSON marshaling pod spec patch: %w", err)
	}
	execution.PodSpecPatch = string(podSpecPatchBytes)
	return execution, nil
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

		// Following the convention of downloadArtifacts in the launcher to only look at the first in the list.
		inputArtifact := artifactList.Artifacts[0]

		// This should ideally verify that this is also a model input artifact, but this metadata doesn't seem to
		// be set on inputArtifact.
		if !strings.HasPrefix(inputArtifact.Uri, "oci://") {
			continue
		}

		modelcarArtifacts[name] = inputArtifact
		modelcarArtifactNames = append(modelcarArtifactNames, name)
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

// Extends the PodSpec to include Kubernetes-specific executor config.
// inputParams is a map of the input parameter name to a resolvable value.
func extendPodSpecPatch(
	ctx context.Context,
	podSpec *k8score.PodSpec,
	opts Options,
	dag *metadata.DAG,
	pipeline *metadata.Pipeline,
	mlmd *metadata.Client,
	inputParams map[string]*structpb.Value,
) error {
	kubernetesExecutorConfig := opts.KubernetesExecutorConfig

	// Return an error if the podSpec has no user container.
	if len(podSpec.Containers) == 0 {
		return fmt.Errorf("failed to patch the pod with kubernetes-specific config due to missing user container: %v", podSpec)
	}

	// Get volume mount information
	if kubernetesExecutorConfig.GetPvcMount() != nil {
		volumeMounts, volumes, err := makeVolumeMountPatch(ctx, opts, kubernetesExecutorConfig.GetPvcMount(),
			dag, pipeline, mlmd, inputParams)
		if err != nil {
			return fmt.Errorf("failed to extract volume mount info: %w", err)
		}
		podSpec.Volumes = append(podSpec.Volumes, volumes...)
		// We assume that the user container always gets executed first within a pod.
		podSpec.Containers[0].VolumeMounts = append(podSpec.Containers[0].VolumeMounts, volumeMounts...)
	}

	// Get image pull policy
	pullPolicy := kubernetesExecutorConfig.GetImagePullPolicy()
	if pullPolicy != "" {
		policies := []string{"Always", "Never", "IfNotPresent"}
		found := false
		for _, value := range policies {
			if value == pullPolicy {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("unsupported value: %s. ImagePullPolicy should be one of 'Always', 'Never' or 'IfNotPresent'", pullPolicy)
		}
		// We assume that the user container always gets executed first within a pod.
		podSpec.Containers[0].ImagePullPolicy = k8score.PullPolicy(pullPolicy)
	}

	// Get node selector information
	if kubernetesExecutorConfig.GetNodeSelector() != nil {
		if kubernetesExecutorConfig.GetNodeSelector().GetNodeSelectorJson() != nil {
			var nodeSelector map[string]string
			err := resolveK8sJsonParameter(ctx, opts, dag, pipeline, mlmd,
				kubernetesExecutorConfig.GetNodeSelector().GetNodeSelectorJson(), inputParams, &nodeSelector)
			if err != nil {
				return fmt.Errorf("failed to resolve node selector: %w", err)
			}
			podSpec.NodeSelector = nodeSelector
		} else {
			podSpec.NodeSelector = kubernetesExecutorConfig.GetNodeSelector().GetLabels()
		}
	}

	if tolerations := kubernetesExecutorConfig.GetTolerations(); tolerations != nil {
		var k8sTolerations []k8score.Toleration

		glog.Infof("Tolerations passed: %+v", tolerations)

		for _, toleration := range tolerations {
			if toleration != nil {
				k8sToleration := &k8score.Toleration{}
				if toleration.TolerationJson != nil {
					err := resolveK8sJsonParameter(ctx, opts, dag, pipeline, mlmd,
						toleration.GetTolerationJson(), inputParams, k8sToleration)
					if err != nil {
						return fmt.Errorf("failed to resolve toleration: %w", err)
					}
				} else {
					k8sToleration.Key = toleration.Key
					k8sToleration.Operator = k8score.TolerationOperator(toleration.Operator)
					k8sToleration.Value = toleration.Value
					k8sToleration.Effect = k8score.TaintEffect(toleration.Effect)
					k8sToleration.TolerationSeconds = toleration.TolerationSeconds
				}
				k8sTolerations = append(k8sTolerations, *k8sToleration)
			}
		}
		podSpec.Tolerations = k8sTolerations
	}

	// Get secret mount information
	for _, secretAsVolume := range kubernetesExecutorConfig.GetSecretAsVolume() {
		var secretName string
		if secretAsVolume.SecretNameParameter != nil {
			resolvedSecretName, err := resolveInputParameterStr(ctx, dag, pipeline, opts, mlmd,
				secretAsVolume.SecretNameParameter, inputParams)
			if err != nil {
				return fmt.Errorf("failed to resolve secret name: %w", err)
			}
			secretName = resolvedSecretName.GetStringValue()
		} else if secretAsVolume.SecretName != "" {
			secretName = secretAsVolume.SecretName
		} else {
			return fmt.Errorf("missing either SecretName or SecretNameParameter for secret volume in executor config")
		}

		optional := secretAsVolume.Optional != nil && *secretAsVolume.Optional
		secretVolume := k8score.Volume{
			Name: secretName,
			VolumeSource: k8score.VolumeSource{
				Secret: &k8score.SecretVolumeSource{
					SecretName: secretName,
					Optional:   &optional,
				},
			},
		}
		secretVolumeMount := k8score.VolumeMount{
			Name:      secretName,
			MountPath: secretAsVolume.GetMountPath(),
		}
		podSpec.Volumes = append(podSpec.Volumes, secretVolume)
		podSpec.Containers[0].VolumeMounts = append(podSpec.Containers[0].VolumeMounts, secretVolumeMount)
	}

	// Get secret env information
	for _, secretAsEnv := range kubernetesExecutorConfig.GetSecretAsEnv() {
		for _, keyToEnv := range secretAsEnv.GetKeyToEnv() {
			secretEnvVar := k8score.EnvVar{
				Name: keyToEnv.GetEnvVar(),
				ValueFrom: &k8score.EnvVarSource{
					SecretKeyRef: &k8score.SecretKeySelector{
						Key: keyToEnv.GetSecretKey(),
					},
				},
			}

			var secretName string
			if secretAsEnv.SecretNameParameter != nil {
				resolvedSecretName, err := resolveInputParameterStr(ctx, dag, pipeline, opts, mlmd,
					secretAsEnv.SecretNameParameter, inputParams)
				if err != nil {
					return fmt.Errorf("failed to resolve secret name: %w", err)
				}
				secretName = resolvedSecretName.GetStringValue()
			} else if secretAsEnv.SecretName != "" {
				secretName = secretAsEnv.SecretName
			} else {
				return fmt.Errorf("missing either SecretName or SecretNameParameter for " +
					"secret environment variable in executor config")
			}

			secretEnvVar.ValueFrom.SecretKeyRef.LocalObjectReference.Name = secretName
			podSpec.Containers[0].Env = append(podSpec.Containers[0].Env, secretEnvVar)
		}
	}

	// Get config map mount information
	for _, configMapAsVolume := range kubernetesExecutorConfig.GetConfigMapAsVolume() {
		var configMapName string
		if configMapAsVolume.ConfigMapNameParameter != nil {
			resolvedSecretName, err := resolveInputParameterStr(ctx, dag, pipeline, opts, mlmd,
				configMapAsVolume.ConfigMapNameParameter, inputParams)
			if err != nil {
				return fmt.Errorf("failed to resolve configmap name: %w", err)
			}
			configMapName = resolvedSecretName.GetStringValue()
		} else if configMapAsVolume.ConfigMapName != "" {
			configMapName = configMapAsVolume.ConfigMapName
		} else {
			return fmt.Errorf("missing either ConfigMapName or ConfigNameParameter for config volume in executor config")
		}

		optional := configMapAsVolume.Optional != nil && *configMapAsVolume.Optional
		configMapVolume := k8score.Volume{
			Name: configMapName,
			VolumeSource: k8score.VolumeSource{
				ConfigMap: &k8score.ConfigMapVolumeSource{
					LocalObjectReference: k8score.LocalObjectReference{
						Name: configMapName,
					},
					Optional: &optional,
				},
			},
		}
		configMapVolumeMount := k8score.VolumeMount{
			Name:      configMapName,
			MountPath: configMapAsVolume.GetMountPath(),
		}
		podSpec.Volumes = append(podSpec.Volumes, configMapVolume)
		podSpec.Containers[0].VolumeMounts = append(podSpec.Containers[0].VolumeMounts, configMapVolumeMount)
	}

	// Get config map env information
	for _, configMapAsEnv := range kubernetesExecutorConfig.GetConfigMapAsEnv() {
		for _, keyToEnv := range configMapAsEnv.GetKeyToEnv() {
			configMapEnvVar := k8score.EnvVar{
				Name: keyToEnv.GetEnvVar(),
				ValueFrom: &k8score.EnvVarSource{
					ConfigMapKeyRef: &k8score.ConfigMapKeySelector{
						Key: keyToEnv.GetConfigMapKey(),
					},
				},
			}

			var configMapName string
			if configMapAsEnv.ConfigMapNameParameter != nil {
				resolvedSecretName, err := resolveInputParameterStr(ctx, dag, pipeline, opts, mlmd,
					configMapAsEnv.ConfigMapNameParameter, inputParams)
				if err != nil {
					return fmt.Errorf("failed to resolve configmap name: %w", err)
				}
				configMapName = resolvedSecretName.GetStringValue()
			} else if configMapAsEnv.ConfigMapName != "" {
				configMapName = configMapAsEnv.ConfigMapName
			} else {
				return fmt.Errorf("missing either ConfigMapName or ConfigNameParameter for " +
					"configmap environment variable in executor config")
			}

			configMapEnvVar.ValueFrom.ConfigMapKeyRef.LocalObjectReference.Name = configMapName
			podSpec.Containers[0].Env = append(podSpec.Containers[0].Env, configMapEnvVar)
		}
	}

	// Get image pull secret information
	for _, imagePullSecret := range kubernetesExecutorConfig.GetImagePullSecret() {
		var secretName string
		if imagePullSecret.SecretNameParameter != nil {
			resolvedSecretName, err := resolveInputParameterStr(ctx, dag, pipeline, opts, mlmd,
				imagePullSecret.SecretNameParameter, inputParams)
			if err != nil {
				return fmt.Errorf("failed to resolve image pull secret name: %w", err)
			}
			secretName = resolvedSecretName.GetStringValue()
		} else if imagePullSecret.SecretName != "" {
			secretName = imagePullSecret.SecretName
		} else {
			return fmt.Errorf("missing either SecretName or SecretNameParameter " +
				"for image pull secret in executor config")
		}

		podSpec.ImagePullSecrets = append(
			podSpec.ImagePullSecrets,
			k8score.LocalObjectReference{
				Name: secretName,
			},
		)
	}

	// Get Kubernetes FieldPath Env information
	for _, fieldPathAsEnv := range kubernetesExecutorConfig.GetFieldPathAsEnv() {
		fieldPathEnvVar := k8score.EnvVar{
			Name: fieldPathAsEnv.GetName(),
			ValueFrom: &k8score.EnvVarSource{
				FieldRef: &k8score.ObjectFieldSelector{
					FieldPath: fieldPathAsEnv.GetFieldPath(),
				},
			},
		}
		podSpec.Containers[0].Env = append(podSpec.Containers[0].Env, fieldPathEnvVar)
	}

	// Get container timeout information
	timeout := kubernetesExecutorConfig.GetActiveDeadlineSeconds()
	if timeout > 0 {
		podSpec.ActiveDeadlineSeconds = &timeout
	}

	// Get Pod Generic Ephemeral volume information
	for _, ephemeralVolumeSpec := range kubernetesExecutorConfig.GetGenericEphemeralVolume() {
		var accessModes []k8score.PersistentVolumeAccessMode
		for _, value := range ephemeralVolumeSpec.GetAccessModes() {
			accessModes = append(accessModes, accessModeMap[value])
		}
		var storageClassName *string
		storageClassName = nil
		if !ephemeralVolumeSpec.GetDefaultStorageClass() {
			_storageClassName := ephemeralVolumeSpec.GetStorageClassName()
			storageClassName = &_storageClassName
		}
		ephemeralVolume := k8score.Volume{
			Name: ephemeralVolumeSpec.GetVolumeName(),
			VolumeSource: k8score.VolumeSource{
				Ephemeral: &k8score.EphemeralVolumeSource{
					VolumeClaimTemplate: &k8score.PersistentVolumeClaimTemplate{
						ObjectMeta: metav1.ObjectMeta{
							Labels:      ephemeralVolumeSpec.GetMetadata().GetLabels(),
							Annotations: ephemeralVolumeSpec.GetMetadata().GetAnnotations(),
						},
						Spec: k8score.PersistentVolumeClaimSpec{
							AccessModes: accessModes,
							Resources: k8score.VolumeResourceRequirements{
								Requests: k8score.ResourceList{
									k8score.ResourceStorage: k8sres.MustParse(ephemeralVolumeSpec.GetSize()),
								},
							},
							StorageClassName: storageClassName,
						},
					},
				},
			},
		}
		ephemeralVolumeMount := k8score.VolumeMount{
			Name:      ephemeralVolumeSpec.GetVolumeName(),
			MountPath: ephemeralVolumeSpec.GetMountPath(),
		}
		podSpec.Volumes = append(podSpec.Volumes, ephemeralVolume)
		podSpec.Containers[0].VolumeMounts = append(podSpec.Containers[0].VolumeMounts, ephemeralVolumeMount)
	}

	// EmptyDirMounts
	for _, emptyDirVolumeSpec := range kubernetesExecutorConfig.GetEmptyDirMounts() {
		var sizeLimitResource *k8sres.Quantity
		if emptyDirVolumeSpec.GetSizeLimit() != "" {
			r := k8sres.MustParse(emptyDirVolumeSpec.GetSizeLimit())
			sizeLimitResource = &r
		}

		emptyDirVolume := k8score.Volume{
			Name: emptyDirVolumeSpec.GetVolumeName(),
			VolumeSource: k8score.VolumeSource{
				EmptyDir: &k8score.EmptyDirVolumeSource{
					Medium:    k8score.StorageMedium(emptyDirVolumeSpec.GetMedium()),
					SizeLimit: sizeLimitResource,
				},
			},
		}
		emptyDirVolumeMount := k8score.VolumeMount{
			Name:      emptyDirVolumeSpec.GetVolumeName(),
			MountPath: emptyDirVolumeSpec.GetMountPath(),
		}

		podSpec.Volumes = append(podSpec.Volumes, emptyDirVolume)
		podSpec.Containers[0].VolumeMounts = append(podSpec.Containers[0].VolumeMounts, emptyDirVolumeMount)
	}

	return nil
}

// TODO(Bobgy): merge DAG driver and container driver, because they are very similar.
func DAG(ctx context.Context, opts Options, mlmd *metadata.Client) (execution *Execution, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("driver.DAG(%s) failed: %w", opts.info(), err)
		}
	}()
	b, _ := json.Marshal(opts)
	glog.V(4).Info("DAG opts: ", string(b))
	err = validateDAG(opts)
	if err != nil {
		return nil, err
	}
	var iterationIndex *int
	if opts.IterationIndex >= 0 {
		index := opts.IterationIndex
		iterationIndex = &index
	}
	// TODO(Bobgy): there's no need to pass any parameters, because pipeline
	// and pipeline run context have been created by root DAG driver.
	pipeline, err := mlmd.GetPipeline(ctx, opts.PipelineName, opts.RunID, "", "", "", "")
	if err != nil {
		return nil, err
	}
	dag, err := mlmd.GetDAG(ctx, opts.DAGExecutionID)
	if err != nil {
		return nil, err
	}
	glog.Infof("parent DAG: %+v", dag.Execution)
	expr, err := expression.New()
	if err != nil {
		return nil, err
	}
	inputs, err := resolveInputs(ctx, dag, iterationIndex, pipeline, opts, mlmd, expr)
	if err != nil {
		return nil, err
	}
	executorInput := &pipelinespec.ExecutorInput{
		Inputs: inputs,
	}
	glog.Infof("executorInput value: %+v", executorInput)
	execution = &Execution{ExecutorInput: executorInput}
	condition := opts.Task.GetTriggerPolicy().GetCondition()
	if condition != "" {
		willTrigger, err := expr.Condition(executorInput, condition)
		if err != nil {
			return execution, err
		}
		execution.Condition = &willTrigger
	}
	ecfg, err := metadata.GenerateExecutionConfig(executorInput)
	if err != nil {
		return execution, err
	}
	ecfg.TaskName = opts.Task.GetTaskInfo().GetName()
	ecfg.ExecutionType = metadata.DagExecutionTypeName
	ecfg.ParentDagID = dag.Execution.GetID()
	ecfg.IterationIndex = iterationIndex
	ecfg.NotTriggered = !execution.WillTrigger()

	// Handle writing output parameters to MLMD.
	ecfg.OutputParameters = opts.Component.GetDag().GetOutputs().GetParameters()
	glog.V(4).Info("outputParameters: ", ecfg.OutputParameters)

	// Handle writing output artifacts to MLMD.
	ecfg.OutputArtifacts = opts.Component.GetDag().GetOutputs().GetArtifacts()
	glog.V(4).Info("outputArtifacts: ", ecfg.OutputArtifacts)

	totalDagTasks := len(opts.Component.GetDag().GetTasks())
	ecfg.TotalDagTasks = &totalDagTasks
	glog.V(4).Info("totalDagTasks: ", *ecfg.TotalDagTasks)

	if opts.Task.GetArtifactIterator() != nil {
		return execution, fmt.Errorf("ArtifactIterator is not implemented")
	}
	isIterator := opts.Task.GetParameterIterator() != nil && opts.IterationIndex < 0
	// Fan out iterations
	if execution.WillTrigger() && isIterator {
		iterator := opts.Task.GetParameterIterator()
		report := func(err error) error {
			return fmt.Errorf("iterating on item input %q failed: %w", iterator.GetItemInput(), err)
		}
		// Check the items type of parameterIterator:
		// It can be "inputParameter" or "Raw"
		var value *structpb.Value
		switch iterator.GetItems().GetKind().(type) {
		case *pipelinespec.ParameterIteratorSpec_ItemsSpec_InputParameter:
			var ok bool
			value, ok = executorInput.GetInputs().GetParameterValues()[iterator.GetItems().GetInputParameter()]
			if !ok {
				return execution, report(fmt.Errorf("cannot find input parameter"))
			}
		case *pipelinespec.ParameterIteratorSpec_ItemsSpec_Raw:
			value_raw := iterator.GetItems().GetRaw()
			var unmarshalled_raw interface{}
			err = json.Unmarshal([]byte(value_raw), &unmarshalled_raw)
			if err != nil {
				return execution, fmt.Errorf("error unmarshall raw string: %q", err)
			}
			value, err = structpb.NewValue(unmarshalled_raw)
			if err != nil {
				return execution, fmt.Errorf("error converting unmarshalled raw string into protobuf Value type: %q", err)
			}
			// Add the raw input to the executor input
			execution.ExecutorInput.Inputs.ParameterValues[iterator.GetItemInput()] = value
		default:
			return execution, fmt.Errorf("cannot find parameter iterator")
		}
		items, err := getItems(value)
		if err != nil {
			return execution, report(err)
		}
		count := len(items)
		ecfg.IterationCount = &count
		execution.IterationCount = &count
	}

	glog.V(4).Info("pipeline: ", pipeline)
	b, _ = json.Marshal(*ecfg)
	glog.V(4).Info("ecfg: ", string(b))
	glog.V(4).Infof("dag: %v", dag)

	// TODO(Bobgy): change execution state to pending, because this is driver, execution hasn't started.
	createdExecution, err := mlmd.CreateExecution(ctx, pipeline, ecfg)
	if err != nil {
		return execution, err
	}
	glog.Infof("Created execution: %s", createdExecution)
	execution.ID = createdExecution.GetID()
	return execution, nil
}

// Get iteration items from a structpb.Value.
// Return value may be
// * a list of JSON serializable structs
// * a list of structpb.Value
func getItems(value *structpb.Value) (items []*structpb.Value, err error) {
	switch v := value.GetKind().(type) {
	case *structpb.Value_ListValue:
		return v.ListValue.GetValues(), nil
	case *structpb.Value_StringValue:
		listValue := structpb.Value{}
		if err = listValue.UnmarshalJSON([]byte(v.StringValue)); err != nil {
			return nil, err
		}
		return listValue.GetListValue().GetValues(), nil
	default:
		return nil, fmt.Errorf("value of type %T cannot be iterated", v)
	}
}

func reuseCachedOutputs(ctx context.Context, executorInput *pipelinespec.ExecutorInput, mlmd *metadata.Client, cachedMLMDExecutionID string) (*pipelinespec.ExecutorOutput, []*metadata.OutputArtifact, error) {
	cachedMLMDExecutionIDInt64, err := strconv.ParseInt(cachedMLMDExecutionID, 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("failure while transfering cachedMLMDExecutionID %s from string to int64: %w", cachedMLMDExecutionID, err)
	}
	execution, err := mlmd.GetExecution(ctx, cachedMLMDExecutionIDInt64)
	if err != nil {
		return nil, nil, fmt.Errorf("failure while getting execution of cachedMLMDExecutionID %v: %w", cachedMLMDExecutionIDInt64, err)
	}
	executorOutput := &pipelinespec.ExecutorOutput{
		Artifacts: map[string]*pipelinespec.ArtifactList{},
	}
	_, outputs, err := execution.GetParameters()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to collect output parameters from cache: %w", err)
	}
	executorOutput.ParameterValues = outputs
	outputArtifacts, err := collectOutputArtifactMetadataFromCache(ctx, executorInput, cachedMLMDExecutionIDInt64, mlmd)
	if err != nil {
		return nil, nil, fmt.Errorf("failed collect output artifact metadata from cache: %w", err)
	}
	return executorOutput, outputArtifacts, nil
}

func collectOutputArtifactMetadataFromCache(ctx context.Context, executorInput *pipelinespec.ExecutorInput, cachedMLMDExecutionID int64, mlmd *metadata.Client) ([]*metadata.OutputArtifact, error) {
	outputArtifacts, err := mlmd.GetOutputArtifactsByExecutionId(ctx, cachedMLMDExecutionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get MLMDOutputArtifactsByName by executionId %v: %w", cachedMLMDExecutionID, err)
	}

	// Register artifacts with MLMD.
	registeredMLMDArtifacts := make([]*metadata.OutputArtifact, 0, len(executorInput.GetOutputs().GetArtifacts()))
	for name, artifactList := range executorInput.GetOutputs().GetArtifacts() {
		if len(artifactList.Artifacts) == 0 {
			continue
		}
		artifact := artifactList.Artifacts[0]
		outputArtifact, ok := outputArtifacts[name]
		if !ok {
			return nil, fmt.Errorf("unable to find artifact with name %v in mlmd output artifacts", name)
		}
		outputArtifact.Schema = artifact.GetType().GetInstanceSchema()
		registeredMLMDArtifacts = append(registeredMLMDArtifacts, outputArtifact)
	}
	return registeredMLMDArtifacts, nil

}

func getFingerPrint(opts Options, executorInput *pipelinespec.ExecutorInput) (string, error) {
	outputParametersTypeMap := make(map[string]string)
	for outputParamName, outputParamSpec := range opts.Component.GetOutputDefinitions().GetParameters() {
		outputParametersTypeMap[outputParamName] = outputParamSpec.GetParameterType().String()
	}
	userCmdArgs := make([]string, 0, len(opts.Container.Command)+len(opts.Container.Args))
	userCmdArgs = append(userCmdArgs, opts.Container.Command...)
	userCmdArgs = append(userCmdArgs, opts.Container.Args...)

	cacheKey, err := cacheutils.GenerateCacheKey(executorInput.GetInputs(), executorInput.GetOutputs(), outputParametersTypeMap, userCmdArgs, opts.Container.Image)
	if err != nil {
		return "", fmt.Errorf("failure while generating CacheKey: %w", err)
	}
	fingerPrint, err := cacheutils.GenerateFingerPrint(cacheKey)
	return fingerPrint, err
}

func validateContainer(opts Options) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("invalid container driver args: %w", err)
		}
	}()
	if opts.Container == nil {
		return fmt.Errorf("container spec is required")
	}
	return validateNonRoot(opts)
}

func validateDAG(opts Options) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("invalid DAG driver args: %w", err)
		}
	}()
	if opts.Container != nil {
		return fmt.Errorf("container spec is unnecessary")
	}
	return validateNonRoot(opts)
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

func resolveInputs(
	ctx context.Context,
	dag *metadata.DAG,
	iterationIndex *int,
	pipeline *metadata.Pipeline,
	opts Options,
	mlmd *metadata.Client,
	expr *expression.Expr,
) (inputs *pipelinespec.ExecutorInput_Inputs, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to resolve inputs: %w", err)
		}
	}()

	task := opts.Task
	inputsSpec := opts.Component.GetInputDefinitions()

	glog.V(4).Infof("dag: %v", dag)
	glog.V(4).Infof("task: %v", task)
	inputParams, _, err := dag.Execution.GetParameters()
	if err != nil {
		return nil, err
	}
	inputArtifacts, err := mlmd.GetInputArtifactsByExecutionID(ctx, dag.Execution.GetID())
	if err != nil {
		return nil, err
	}
	glog.Infof("parent DAG input parameters: %+v, artifacts: %+v", inputParams, inputArtifacts)
	inputs = &pipelinespec.ExecutorInput_Inputs{
		ParameterValues: make(map[string]*structpb.Value),
		Artifacts:       make(map[string]*pipelinespec.ArtifactList),
	}
	isIterationDriver := iterationIndex != nil

	handleParameterExpressionSelector := func() error {
		for name, paramSpec := range task.GetInputs().GetParameters() {
			var selector string
			if selector = paramSpec.GetParameterExpressionSelector(); selector == "" {
				continue
			}
			wrap := func(e error) error {
				return fmt.Errorf("resolving parameter %q: evaluation of parameter expression selector %q failed: %w", name, selector, e)
			}
			value, ok := inputs.ParameterValues[name]
			if !ok {
				return wrap(fmt.Errorf("value not found in inputs"))
			}
			selected, err := expr.Select(value, selector)
			if err != nil {
				return wrap(err)
			}
			inputs.ParameterValues[name] = selected
		}
		return nil
	}
	handleParamTypeValidationAndConversion := func() error {
		// TODO(Bobgy): verify whether there are inputs not in the inputs spec.
		for name, spec := range inputsSpec.GetParameters() {
			if task.GetParameterIterator() != nil {
				if !isIterationDriver && task.GetParameterIterator().GetItemInput() == name {
					// It's expected that an iterator does not have iteration item input parameter,
					// because only iterations get the item input parameter.
					continue
				}
				if isIterationDriver && task.GetParameterIterator().GetItems().GetInputParameter() == name {
					// It's expected that an iteration does not have iteration items input parameter,
					// because only the iterator has it.
					continue
				}
			}
			value, hasValue := inputs.GetParameterValues()[name]

			// Handle when parameter does not have input value
			if !hasValue && !inputsSpec.GetParameters()[name].GetIsOptional() {
				// When parameter is not optional and there is no input value, first check if there is a default value,
				// if there is a default value, use it as the value of the parameter.
				// if there is no default value, report error.
				if inputsSpec.GetParameters()[name].GetDefaultValue() == nil {
					return fmt.Errorf("neither value nor default value provided for non-optional parameter %q", name)
				}
			} else if !hasValue && inputsSpec.GetParameters()[name].GetIsOptional() {
				// When parameter is optional and there is no input value, value comes from default value.
				// But we don't pass the default value here. They are resolved internally within the component.
				// Note: in the past the backend passed the default values into the component. This is a behavior change.
				// See discussion: https://github.com/kubeflow/pipelines/pull/8765#discussion_r1119477085
				continue
			}

			switch spec.GetParameterType() {
			case pipelinespec.ParameterType_STRING:
				_, isValueString := value.GetKind().(*structpb.Value_StringValue)
				if !isValueString {
					// TODO(Bobgy): discuss whether we want to allow auto type conversion
					// all parameter types can be consumed as JSON string
					text, err := metadata.PbValueToText(value)
					if err != nil {
						return fmt.Errorf("converting input parameter %q to string: %w", name, err)
					}
					inputs.GetParameterValues()[name] = structpb.NewStringValue(text)
				}
			default:
				typeMismatch := func(actual string) error {
					return fmt.Errorf("input parameter %q type mismatch: expect %s, got %s", name, spec.GetParameterType(), actual)
				}
				switch v := value.GetKind().(type) {
				case *structpb.Value_NullValue:
					return fmt.Errorf("got null for input parameter %q", name)
				case *structpb.Value_StringValue:
					// TODO(Bobgy): consider whether we support parsing string as JSON for any other types.
					if spec.GetParameterType() != pipelinespec.ParameterType_STRING {
						return typeMismatch("string")
					}
				case *structpb.Value_NumberValue:
					if spec.GetParameterType() != pipelinespec.ParameterType_NUMBER_DOUBLE && spec.GetParameterType() != pipelinespec.ParameterType_NUMBER_INTEGER {
						return typeMismatch("number")
					}
				case *structpb.Value_BoolValue:
					if spec.GetParameterType() != pipelinespec.ParameterType_BOOLEAN {
						return typeMismatch("bool")
					}
				case *structpb.Value_ListValue:
					if spec.GetParameterType() != pipelinespec.ParameterType_LIST {
						return typeMismatch("list")
					}
				case *structpb.Value_StructValue:
					if spec.GetParameterType() != pipelinespec.ParameterType_STRUCT {
						return typeMismatch("struct")
					}
				default:
					return fmt.Errorf("parameter %s has unknown protobuf.Value type: %T", name, v)
				}
			}
		}
		return nil
	}
	// this function has many branches, so it's hard to add more postprocess steps
	// TODO(Bobgy): consider splitting this function into several sub functions
	defer func() {
		if err == nil {
			err = handleParameterExpressionSelector()
		}
		if err == nil {
			err = handleParamTypeValidationAndConversion()
		}
	}()
	// resolve input parameters
	if isIterationDriver {
		// resolve inputs for iteration driver is very different
		artifacts, err := mlmd.GetInputArtifactsByExecutionID(ctx, dag.Execution.GetID())
		if err != nil {
			return nil, err
		}
		inputs.ParameterValues = inputParams
		inputs.Artifacts = artifacts
		switch {
		case task.GetArtifactIterator() != nil:
			return nil, fmt.Errorf("artifact iterator not implemented yet")
		case task.GetParameterIterator() != nil:
			var itemsInput string
			if task.GetParameterIterator().GetItems().GetInputParameter() != "" {
				// input comes from outside the component
				itemsInput = task.GetParameterIterator().GetItems().GetInputParameter()
			} else if task.GetParameterIterator().GetItemInput() != "" {
				// input comes from static input
				itemsInput = task.GetParameterIterator().GetItemInput()
			} else {
				return nil, fmt.Errorf("cannot retrieve parameter iterator")
			}
			items, err := getItems(inputs.ParameterValues[itemsInput])
			if err != nil {
				return nil, err
			}
			if *iterationIndex >= len(items) {
				return nil, fmt.Errorf("bug: %v items found, but getting index %v", len(items), *iterationIndex)
			}
			delete(inputs.ParameterValues, itemsInput)
			inputs.ParameterValues[task.GetParameterIterator().GetItemInput()] = items[*iterationIndex]
		default:
			return nil, fmt.Errorf("bug: iteration_index>=0, but task iterator is empty")
		}
		return inputs, nil
	}

	// Handle parameters.
	for name, paramSpec := range task.GetInputs().GetParameters() {
		v, err := resolveInputParameter(ctx, dag, pipeline, opts, mlmd, paramSpec, inputParams)
		if err != nil {
			if !errors.Is(err, ErrResolvedParameterNull) {
				return nil, err
			}

			componentParam, ok := opts.Component.GetInputDefinitions().GetParameters()[name]
			if ok && componentParam != nil && componentParam.IsOptional {
				// If the resolved paramter was null and the component input parameter is optional, just skip setting
				// it and the launcher will handle defaults.
				continue
			}

			return nil, err
		}

		inputs.ParameterValues[name] = v
	}

	// Handle artifacts.
	for name, artifactSpec := range task.GetInputs().GetArtifacts() {
		v, err := resolveInputArtifact(ctx, dag, pipeline, mlmd, name, artifactSpec, inputArtifacts, task)
		if err != nil {
			return nil, err
		}
		inputs.Artifacts[name] = v
	}
	// TODO(Bobgy): validate executor inputs match component inputs definition
	return inputs, nil
}

// resolveInputParameter resolves an InputParameterSpec
// using a given input context via InputParams. ErrResolvedParameterNull is returned if paramSpec
// is a component input parameter and parameter resolves to a null value (i.e. an optional pipeline input with no
// default). The caller can decide if this is allowed in that context.
func resolveInputParameter(
	ctx context.Context,
	dag *metadata.DAG,
	pipeline *metadata.Pipeline,
	opts Options,
	mlmd *metadata.Client,
	paramSpec *pipelinespec.TaskInputsSpec_InputParameterSpec,
	inputParams map[string]*structpb.Value,
) (*structpb.Value, error) {
	glog.V(4).Infof("paramSpec: %v", paramSpec)
	paramError := func(err error) error {
		return fmt.Errorf("resolving input parameter with spec %s: %w", paramSpec, err)
	}
	switch t := paramSpec.Kind.(type) {
	case *pipelinespec.TaskInputsSpec_InputParameterSpec_ComponentInputParameter:
		componentInput := paramSpec.GetComponentInputParameter()
		if componentInput == "" {
			return nil, paramError(fmt.Errorf("empty component input"))
		}
		v, ok := inputParams[componentInput]
		if !ok {
			return nil, paramError(fmt.Errorf("parent DAG does not have input parameter %s", componentInput))
		}

		if _, isNullValue := v.GetKind().(*structpb.Value_NullValue); isNullValue {
			// Null values are only allowed for optional pipeline input parameters with no values. The caller has this
			// context to know if this is allowed.
			return nil, fmt.Errorf("%w: %s", ErrResolvedParameterNull, componentInput)
		}

		return v, nil

	// This is the case where the input comes from the output of an upstream task.
	case *pipelinespec.TaskInputsSpec_InputParameterSpec_TaskOutputParameter:
		cfg := resolveUpstreamOutputsConfig{
			ctx:       ctx,
			paramSpec: paramSpec,
			dag:       dag,
			pipeline:  pipeline,
			mlmd:      mlmd,
			err:       paramError,
		}
		v, err := resolveUpstreamParameters(cfg)
		if err != nil {
			return nil, err
		}
		return v, nil
	case *pipelinespec.TaskInputsSpec_InputParameterSpec_RuntimeValue:
		runtimeValue := paramSpec.GetRuntimeValue()
		switch t := runtimeValue.Value.(type) {
		case *pipelinespec.ValueOrRuntimeParameter_Constant:
			val := runtimeValue.GetConstant()
			var v *structpb.Value
			switch val.GetStringValue() {
			case "{{$.pipeline_job_name}}":
				v = structpb.NewStringValue(opts.RunDisplayName)
			case "{{$.pipeline_job_resource_name}}":
				v = structpb.NewStringValue(opts.RunName)
			case "{{$.pipeline_job_uuid}}":
				v = structpb.NewStringValue(opts.RunID)
			case "{{$.pipeline_task_name}}":
				v = structpb.NewStringValue(opts.Task.GetTaskInfo().GetName())
			case "{{$.pipeline_task_uuid}}":
				v = structpb.NewStringValue(fmt.Sprintf("%d", opts.DAGExecutionID))
			default:
				v = val
			}

			return v, nil
		default:
			return nil, paramError(fmt.Errorf("param runtime value spec of type %T not implemented", t))
		}
	// TODO(Bobgy): implement the following cases
	// case *pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus_:
	default:
		return nil, paramError(fmt.Errorf("parameter spec of type %T not implemented yet", t))
	}
}

// resolveInputParameterStr is like resolveInputParameter but returns an error if the resolved value is not a non-empty
// string.
func resolveInputParameterStr(
	ctx context.Context,
	dag *metadata.DAG,
	pipeline *metadata.Pipeline,
	opts Options,
	mlmd *metadata.Client,
	paramSpec *pipelinespec.TaskInputsSpec_InputParameterSpec,
	inputParams map[string]*structpb.Value,
) (*structpb.Value, error) {
	val, err := resolveInputParameter(ctx, dag, pipeline, opts, mlmd, paramSpec, inputParams)
	if err != nil {
		return nil, err
	}

	if typedVal, ok := val.GetKind().(*structpb.Value_StringValue); ok && typedVal != nil {
		if typedVal.StringValue == "" {
			return nil, fmt.Errorf("resolving input parameter with spec %s. Expected a non-empty string.", paramSpec)
		}
	} else {
		return nil, fmt.Errorf("resolving input parameter with spec %s. Expected a string but got: %T", paramSpec, val.GetKind())
	}

	return val, nil
}

// resolveInputArtifact resolves an InputArtifactSpec
// using a given input context via inputArtifacts.
func resolveInputArtifact(
	ctx context.Context,
	dag *metadata.DAG,
	pipeline *metadata.Pipeline,
	mlmd *metadata.Client,
	name string,
	artifactSpec *pipelinespec.TaskInputsSpec_InputArtifactSpec,
	inputArtifacts map[string]*pipelinespec.ArtifactList,
	task *pipelinespec.PipelineTaskSpec,
) (*pipelinespec.ArtifactList, error) {
	glog.V(4).Infof("inputs: %#v", task.GetInputs())
	glog.V(4).Infof("artifacts: %#v", task.GetInputs().GetArtifacts())
	artifactError := func(err error) error {
		return fmt.Errorf("failed to resolve input artifact %s with spec %s: %w", name, artifactSpec, err)
	}
	switch t := artifactSpec.Kind.(type) {
	case *pipelinespec.TaskInputsSpec_InputArtifactSpec_ComponentInputArtifact:
		inputArtifactName := artifactSpec.GetComponentInputArtifact()
		if inputArtifactName == "" {
			return nil, artifactError(fmt.Errorf("component input artifact key is empty"))
		}
		v, ok := inputArtifacts[inputArtifactName]
		if !ok {
			return nil, artifactError(fmt.Errorf("parent DAG does not have input artifact %s", inputArtifactName))
		}
		return v, nil
	case *pipelinespec.TaskInputsSpec_InputArtifactSpec_TaskOutputArtifact:
		cfg := resolveUpstreamOutputsConfig{
			ctx:          ctx,
			artifactSpec: artifactSpec,
			dag:          dag,
			pipeline:     pipeline,
			mlmd:         mlmd,
			err:          artifactError,
		}
		artifacts, err := resolveUpstreamArtifacts(cfg)
		if err != nil {
			return nil, err
		}
		return artifacts, nil
	default:
		return nil, artifactError(fmt.Errorf("artifact spec of type %T not implemented yet", t))
	}
}

// getDAGTasks is a recursive function that returns a map of all tasks across all DAGs in the context of nested DAGs.
func getDAGTasks(
	ctx context.Context,
	dag *metadata.DAG,
	pipeline *metadata.Pipeline,
	mlmd *metadata.Client,
	flattenedTasks map[string]*metadata.Execution,
) (map[string]*metadata.Execution, error) {
	if flattenedTasks == nil {
		flattenedTasks = make(map[string]*metadata.Execution)
	}
	currentExecutionTasks, err := mlmd.GetExecutionsInDAG(ctx, dag, pipeline, true)
	if err != nil {
		return nil, err
	}
	for k, v := range currentExecutionTasks {
		flattenedTasks[k] = v
	}
	for _, v := range currentExecutionTasks {

		if v.GetExecution().GetType() == "system.DAGExecution" {
			// Iteration count is only applied when using ParallelFor, and in
			// that scenario you're guaranteed to have redundant task names even
			// within a single DAG, which results in an error when
			// mlmd.GetExecutionsInDAG is called. ParallelFor outputs should be
			// handled with dsl.Collected.
			_, ok := v.GetExecution().GetCustomProperties()["iteration_count"]
			if ok {
				glog.Infof("Found a ParallelFor task, %v. Skipping it.", v.TaskName())
				continue
			}
			glog.V(4).Infof("Found a task, %v, with an execution type of system.DAGExecution. Adding its tasks to the task list.", v.TaskName())
			subDAG, err := mlmd.GetDAG(ctx, v.GetExecution().GetId())
			if err != nil {
				return nil, err
			}
			// Pass the subDAG into a recursive call to getDAGTasks and update
			// tasks to include the subDAG's tasks.
			flattenedTasks, err = getDAGTasks(ctx, subDAG, pipeline, mlmd, flattenedTasks)
			if err != nil {
				return nil, err
			}
		}
	}

	return flattenedTasks, nil
}

// resolveUpstreamOutputsConfig is just a config struct used to store the input
// parameters of the resolveUpstreamParameters and resolveUpstreamArtifacts
// functions.
type resolveUpstreamOutputsConfig struct {
	ctx          context.Context
	paramSpec    *pipelinespec.TaskInputsSpec_InputParameterSpec
	artifactSpec *pipelinespec.TaskInputsSpec_InputArtifactSpec
	dag          *metadata.DAG
	pipeline     *metadata.Pipeline
	mlmd         *metadata.Client
	err          func(error) error
}

// resolveUpstreamParameters resolves input parameters that come from upstream
// tasks. These tasks can be components/containers, which is relatively
// straightforward, or DAGs, in which case, we need to traverse the graph until
// we arrive at a component/container (since there can be n nested DAGs).
func resolveUpstreamParameters(cfg resolveUpstreamOutputsConfig) (*structpb.Value, error) {
	taskOutput := cfg.paramSpec.GetTaskOutputParameter()
	glog.V(4).Info("taskOutput: ", taskOutput)
	producerTaskName := taskOutput.GetProducerTask()
	if producerTaskName == "" {
		return nil, cfg.err(fmt.Errorf("producerTaskName is empty"))
	}
	outputParameterKey := taskOutput.GetOutputParameterKey()
	if outputParameterKey == "" {
		return nil, cfg.err(fmt.Errorf("output parameter key is empty"))
	}

	// Get a list of tasks for the current DAG first.
	// The reason we use gatDAGTasks instead of mlmd.GetExecutionsInDAG is because the latter does not handle
	// task name collisions in the map which results in a bunch of unhandled edge cases and test failures.
	tasks, err := getDAGTasks(cfg.ctx, cfg.dag, cfg.pipeline, cfg.mlmd, nil)
	if err != nil {
		return nil, cfg.err(err)
	}

	producer, ok := tasks[producerTaskName]
	if !ok {
		return nil, cfg.err(fmt.Errorf("producer task, %v, not in tasks", producerTaskName))
	}
	glog.V(4).Info("producer: ", producer)
	glog.V(4).Infof("tasks: %#v", tasks)
	currentTask := producer
	// Continue looping until we reach a sub-task that is NOT a DAG.
	for {
		glog.V(4).Info("currentTask: ", currentTask.TaskName())
		// If the current task is a DAG:
		if *currentTask.GetExecution().Type == "system.DAGExecution" {
			// Since currentTask is a DAG, we need to deserialize its
			// output parameter map so that we can look up its
			// corresponding producer sub-task, reassign currentTask,
			// and iterate through this loop again.
			outputParametersCustomProperty, ok := currentTask.GetExecution().GetCustomProperties()["parameter_producer_task"]
			if !ok {
				return nil, cfg.err(fmt.Errorf("task, %v, does not have a parameter_producer_task custom property", currentTask.TaskName()))
			}
			glog.V(4).Infof("outputParametersCustomProperty: %#v", outputParametersCustomProperty)

			dagOutputParametersMap := make(map[string]*pipelinespec.DagOutputsSpec_DagOutputParameterSpec)
			glog.V(4).Infof("outputParametersCustomProperty: %v", outputParametersCustomProperty.GetStructValue())

			for name, value := range outputParametersCustomProperty.GetStructValue().GetFields() {
				outputSpec := &pipelinespec.DagOutputsSpec_DagOutputParameterSpec{}
				err := protojson.Unmarshal([]byte(value.GetStringValue()), outputSpec)
				if err != nil {
					return nil, err
				}
				dagOutputParametersMap[name] = outputSpec
			}

			glog.V(4).Infof("Deserialized dagOutputParametersMap: %v", dagOutputParametersMap)

			// Support for the 2 DagOutputParameterSpec types:
			// ValueFromParameter & ValueFromOneof
			var subTaskName string
			switch dagOutputParametersMap[outputParameterKey].Kind.(type) {
			case *pipelinespec.DagOutputsSpec_DagOutputParameterSpec_ValueFromParameter:
				subTaskName = dagOutputParametersMap[outputParameterKey].GetValueFromParameter().GetProducerSubtask()
				outputParameterKey = dagOutputParametersMap[outputParameterKey].GetValueFromParameter().GetOutputParameterKey()
			case *pipelinespec.DagOutputsSpec_DagOutputParameterSpec_ValueFromOneof:
				// When OneOf is specified in a pipeline, the output of only 1 task is consumed even though there may be more than 1 task output set. In this case we will attempt to grab the first successful task output.
				paramSelectors := dagOutputParametersMap[outputParameterKey].GetValueFromOneof().GetParameterSelectors()
				glog.V(4).Infof("paramSelectors: %v", paramSelectors)
				// Since we have the tasks map, we can iterate through the parameterSelectors if the ProducerSubTask is not present in the task map and then assign the new OutputParameterKey only if it exists.
				successfulOneOfTask := false
				for !successfulOneOfTask {
					for _, paramSelector := range paramSelectors {
						subTaskName = paramSelector.GetProducerSubtask()
						glog.V(4).Infof("subTaskName from paramSelector: %v", subTaskName)
						glog.V(4).Infof("outputParameterKey from paramSelector: %v", paramSelector.GetOutputParameterKey())
						if subTask, ok := tasks[subTaskName]; ok {
							subTaskState := subTask.GetExecution().LastKnownState.String()
							glog.V(4).Infof("subTask: %w , subTaskState: %v", subTaskName, subTaskState)
							if subTaskState == "CACHED" || subTaskState == "COMPLETE" {

								outputParameterKey = paramSelector.GetOutputParameterKey()
								successfulOneOfTask = true
								break
							}
						}
					}
					if !successfulOneOfTask {
						return nil, cfg.err(fmt.Errorf("processing OneOf: No successful task found"))
					}
				}
			}
			glog.V(4).Infof("SubTaskName from outputParams: %v", subTaskName)
			glog.V(4).Infof("OutputParameterKey from outputParams: %v", outputParameterKey)
			if subTaskName == "" {
				return nil, cfg.err(fmt.Errorf("producer_subtask not in outputParams"))
			}
			glog.V(4).Infof(
				"Overriding currentTask, %v, output with currentTask's producer_subtask, %v, output.",
				currentTask.TaskName(),
				subTaskName,
			)
			currentTask, ok = tasks[subTaskName]
			if !ok {
				return nil, cfg.err(fmt.Errorf("subTaskName, %v, not in tasks", subTaskName))
			}
		} else {
			_, outputParametersCustomProperty, err := currentTask.GetParameters()
			if err != nil {
				return nil, err
			}
			// Base case
			return outputParametersCustomProperty[outputParameterKey], nil
		}
	}
}

// resolveUpstreamArtifacts resolves input artifacts that come from upstream
// tasks. These tasks can be components/containers, which is relatively
// straightforward, or DAGs, in which case, we need to traverse the graph until
// we arrive at a component/container (since there can be n nested DAGs).
func resolveUpstreamArtifacts(cfg resolveUpstreamOutputsConfig) (*pipelinespec.ArtifactList, error) {
	glog.V(4).Infof("artifactSpec: %#v", cfg.artifactSpec)
	taskOutput := cfg.artifactSpec.GetTaskOutputArtifact()
	if taskOutput.GetProducerTask() == "" {
		return nil, cfg.err(fmt.Errorf("producer task is empty"))
	}
	if taskOutput.GetOutputArtifactKey() == "" {
		cfg.err(fmt.Errorf("output artifact key is empty"))
	}
	tasks, err := getDAGTasks(cfg.ctx, cfg.dag, cfg.pipeline, cfg.mlmd, nil)
	if err != nil {
		cfg.err(err)
	}

	producer, ok := tasks[taskOutput.GetProducerTask()]
	if !ok {
		cfg.err(
			fmt.Errorf("cannot find producer task %q", taskOutput.GetProducerTask()),
		)
	}
	glog.V(4).Info("producer: ", producer)
	currentTask := producer
	outputArtifactKey := taskOutput.GetOutputArtifactKey()

	// Continue looping until we reach a sub-task that is NOT a DAG.
	for {
		glog.V(4).Info("currentTask: ", currentTask.TaskName())
		// If the current task is a DAG:
		if *currentTask.GetExecution().Type == "system.DAGExecution" {
			// Get the sub-task.
			outputArtifactsCustomProperty := currentTask.GetExecution().GetCustomProperties()["artifact_producer_task"]
			// Deserialize the output artifacts.
			var outputArtifacts map[string]*pipelinespec.DagOutputsSpec_DagOutputArtifactSpec
			err := json.Unmarshal([]byte(outputArtifactsCustomProperty.GetStringValue()), &outputArtifacts)
			if err != nil {
				return nil, err
			}
			glog.V(4).Infof("Deserialized outputArtifacts: %v", outputArtifacts)
			// Adding support for multiple output artifacts
			var subTaskName string
			artifactSelectors := outputArtifacts[outputArtifactKey].GetArtifactSelectors()

			for _, v := range artifactSelectors {
				glog.V(4).Infof("v: %v", v)
				glog.V(4).Infof("v.ProducerSubtask: %v", v.ProducerSubtask)
				glog.V(4).Infof("v.OutputArtifactKey: %v", v.OutputArtifactKey)
				subTaskName = v.ProducerSubtask
				outputArtifactKey = v.OutputArtifactKey
			}
			// If the sub-task is a DAG, reassign currentTask and run
			// through the loop again.
			currentTask = tasks[subTaskName]
			// }
		} else {
			// Base case, currentTask is a container, not a DAG.
			outputs, err := cfg.mlmd.GetOutputArtifactsByExecutionId(cfg.ctx, currentTask.GetID())
			if err != nil {
				cfg.err(err)
			}
			glog.V(4).Infof("outputs: %#v", outputs)
			artifact, ok := outputs[outputArtifactKey]
			if !ok {
				cfg.err(
					fmt.Errorf(
						"cannot find output artifact key %q in producer task %q",
						taskOutput.GetOutputArtifactKey(),
						taskOutput.GetProducerTask(),
					),
				)
			}
			runtimeArtifact, err := artifact.ToRuntimeArtifact()
			if err != nil {
				cfg.err(err)
			}
			// Base case
			return &pipelinespec.ArtifactList{
				Artifacts: []*pipelinespec.RuntimeArtifact{runtimeArtifact},
			}, nil
		}
	}
}

// provisionOuutputs prepares output references that will get saved to MLMD.
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

var accessModeMap = map[string]k8score.PersistentVolumeAccessMode{
	"ReadWriteOnce":    k8score.ReadWriteOnce,
	"ReadOnlyMany":     k8score.ReadOnlyMany,
	"ReadWriteMany":    k8score.ReadWriteMany,
	"ReadWriteOncePod": k8score.ReadWriteOncePod,
}

// kubernetesPlatformOps() carries out the Kubernetes-specific operations, such as create PVC,
// delete PVC, etc. In these operations we skip the launcher due to there being no user container.
// It also prepublishes and publishes the execution, which are usually done in the launcher.
func kubernetesPlatformOps(
	ctx context.Context,
	mlmd *metadata.Client,
	cacheClient *cacheutils.Client,
	execution *Execution,
	ecfg *metadata.ExecutionConfig,
	opts *Options,
) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to %s and publish execution %s: %w", dummyImages[opts.Container.Image], opts.Task.GetTaskInfo().GetName(), err)
		}
	}()
	// If we cannot create Kubernetes client, we cannot publish this execution
	k8sClient, err := createK8sClient()
	if err != nil {
		return fmt.Errorf("cannot generate k8s clientset: %w", err)
	}

	var outputParameters map[string]*structpb.Value
	var createdExecution *metadata.Execution
	status := pb.Execution_FAILED
	var pvcName string
	defer func() {
		// We publish the execution, no matter this operartion succeeds or not
		perr := publishDriverExecution(k8sClient, mlmd, ctx, createdExecution, outputParameters, nil, status)
		if perr != nil && err != nil {
			err = fmt.Errorf("failed to publish driver execution: %s. Also failed the Kubernetes platform operation: %s", perr.Error(), err.Error())
		} else if perr != nil {
			err = fmt.Errorf("failed to publish driver execution: %w", perr)
		}
	}()

	switch opts.Container.Image {
	case "argostub/createpvc":
		pvcName, createdExecution, status, err = createPVC(ctx, k8sClient, *execution, opts, cacheClient, mlmd, ecfg)
		if err != nil {
			return err
		}
		outputParameters = map[string]*structpb.Value{
			"name": structpb.NewStringValue(pvcName),
		}
	case "argostub/deletepvc":
		if createdExecution, status, err = deletePVC(ctx, k8sClient, *execution, opts, cacheClient, mlmd, ecfg); err != nil {
			return err
		}
	default:
		err = fmt.Errorf("unknown image name %s for Kubernetes-specific operations", opts.Container.Image)
		return err
	}
	return nil
}

// Usually we publish the execution in launcher, but for Kubernetes-specific operations,
// we skip the launcher. So this function is only used in these special cases.
func publishDriverExecution(
	k8sClient *kubernetes.Clientset,
	mlmd *metadata.Client,
	ctx context.Context,
	execution *metadata.Execution,
	outputParameters map[string]*structpb.Value,
	outputArtifacts []*metadata.OutputArtifact,
	status pb.Execution_State,
) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to publish driver execution %s: %w", execution.TaskName(), err)
		}
	}()
	namespace, err := config.InPodNamespace()
	if err != nil {
		return fmt.Errorf("error getting namespace: %w", err)
	}

	podName, err := config.InPodName()
	if err != nil {
		return fmt.Errorf("error getting pod name: %w", err)
	}

	pod, err := k8sClient.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error retrieving info for pod %s: %w", podName, err)
	}

	ecfg := &metadata.ExecutionConfig{
		PodName:   podName,
		PodUID:    string(pod.UID),
		Namespace: namespace,
	}
	if _, err := mlmd.PrePublishExecution(ctx, execution, ecfg); err != nil {
		return fmt.Errorf("failed to prepublish: %w", err)
	}
	if err = mlmd.PublishExecution(ctx, execution, outputParameters, outputArtifacts, status); err != nil {
		return fmt.Errorf("failed to publish: %w", err)
	}
	glog.Infof("Published execution of Kubernetes platform task %s.", execution.TaskName())
	return nil
}

// execution is passed by value because we make changes to it to generate  fingerprint
func createPVC(
	ctx context.Context,
	k8sClient kubernetes.Interface,
	execution Execution,
	opts *Options,
	cacheClient *cacheutils.Client,
	mlmd *metadata.Client,
	ecfg *metadata.ExecutionConfig,
) (pvcName string, createdExecution *metadata.Execution, status pb.Execution_State, err error) {
	// Create execution regardless the operation succeeds or not
	defer func() {
		if createdExecution == nil {
			pipeline, err := mlmd.GetPipeline(ctx, opts.PipelineName, opts.RunID, "", "", "", "")
			if err != nil {
				return
			}
			createdExecution, err = mlmd.CreateExecution(ctx, pipeline, ecfg)
		}
	}()

	taskStartedTime := time.Now().Unix()

	inputs := execution.ExecutorInput.Inputs
	glog.Infof("Input parameter values: %+v", inputs.ParameterValues)

	// Required input: access_modes
	accessModeInput, ok := inputs.ParameterValues["access_modes"]
	if !ok || accessModeInput == nil {
		return "", createdExecution, pb.Execution_FAILED, fmt.Errorf("failed to create pvc: parameter access_modes not provided")
	}
	var accessModes []k8score.PersistentVolumeAccessMode
	for _, value := range accessModeInput.GetListValue().GetValues() {
		accessModes = append(accessModes, accessModeMap[value.GetStringValue()])
	}

	// Optional input: pvc_name and pvc_name_suffix
	// Can only provide at most one of these two parameters.
	// If neither is provided, PVC name is a randomly generated UUID.
	pvcNameSuffixInput := inputs.ParameterValues["pvc_name_suffix"]
	pvcNameInput := inputs.ParameterValues["pvc_name"]
	if pvcNameInput.GetStringValue() != "" && pvcNameSuffixInput.GetStringValue() != "" {
		return "", createdExecution, pb.Execution_FAILED, fmt.Errorf("failed to create pvc: at most one of pvc_name and pvc_name_suffix can be non-empty")
	} else if pvcNameSuffixInput.GetStringValue() != "" {
		pvcName = uuid.NewString() + pvcNameSuffixInput.GetStringValue()
		// Add pvcName to the executor input for fingerprint generation
		execution.ExecutorInput.Inputs.ParameterValues[pvcName] = structpb.NewStringValue(pvcName)
	} else if pvcNameInput.GetStringValue() != "" {
		pvcName = pvcNameInput.GetStringValue()
	} else {
		pvcName = uuid.NewString()
		// Add pvcName to the executor input for fingerprint generation
		execution.ExecutorInput.Inputs.ParameterValues[pvcName] = structpb.NewStringValue(pvcName)
	}

	// Required input: size
	volumeSizeInput, ok := inputs.ParameterValues["size"]
	if !ok || volumeSizeInput == nil {
		return "", createdExecution, pb.Execution_FAILED, fmt.Errorf("failed to create pvc: parameter volumeSize not provided")
	}

	// Optional input: storage_class_name
	// When not provided, use default value `standard`
	storageClassNameInput, ok := inputs.ParameterValues["storage_class_name"]
	var storageClassName string
	if !ok {
		storageClassName = "standard"
	} else {
		storageClassName = storageClassNameInput.GetStringValue()
	}

	// Optional input: annotations
	pvcAnnotationsInput := inputs.ParameterValues["annotations"]
	pvcAnnotations := make(map[string]string)
	for key, val := range pvcAnnotationsInput.GetStructValue().AsMap() {
		typedVal := val.(structpb.Value)
		pvcAnnotations[key] = typedVal.GetStringValue()
	}

	// Optional input: volume_name
	volumeNameInput := inputs.ParameterValues["volume_name"]
	volumeName := volumeNameInput.GetStringValue()

	// Get execution fingerprint and MLMD ID for caching
	// If pvcName includes a randomly generated UUID, it is added in the execution input as a key-value pair for this purpose only
	// The original execution is not changed.
	fingerPrint, cachedMLMDExecutionID, err := getFingerPrintsAndID(&execution, opts, cacheClient)
	if err != nil {
		return "", createdExecution, pb.Execution_FAILED, err
	}
	ecfg.CachedMLMDExecutionID = cachedMLMDExecutionID
	ecfg.FingerPrint = fingerPrint

	pipeline, err := mlmd.GetPipeline(ctx, opts.PipelineName, opts.RunID, "", "", "", "")
	if err != nil {
		return "", createdExecution, pb.Execution_FAILED, fmt.Errorf("error getting pipeline from MLMD: %w", err)
	}

	// Create execution in MLMD
	// TODO(Bobgy): change execution state to pending, because this is driver, execution hasn't started.
	createdExecution, err = mlmd.CreateExecution(ctx, pipeline, ecfg)
	if err != nil {
		return "", createdExecution, pb.Execution_FAILED, fmt.Errorf("error creating MLMD execution for createpvc: %w", err)
	}
	glog.Infof("Created execution: %s", createdExecution)
	execution.ID = createdExecution.GetID()
	if !execution.WillTrigger() {
		return "", createdExecution, pb.Execution_COMPLETE, nil
	}

	// Use cache and skip createpvc if all conditions met:
	// (1) Cache is enabled
	// (2) CachedMLMDExecutionID is non-empty, which means a cache entry exists
	cached := false
	execution.Cached = &cached
	if opts.Task.GetCachingOptions().GetEnableCache() && ecfg.CachedMLMDExecutionID != "" {
		executorOutput, outputArtifacts, err := reuseCachedOutputs(ctx, execution.ExecutorInput, mlmd, ecfg.CachedMLMDExecutionID)
		if err != nil {
			return "", createdExecution, pb.Execution_FAILED, err
		}
		// TODO(Bobgy): upload output artifacts.
		// TODO(Bobgy): when adding artifacts, we will need execution.pipeline to be non-nil, because we need
		// to publish output artifacts to the context too.
		if err := mlmd.PublishExecution(ctx, createdExecution, executorOutput.GetParameterValues(), outputArtifacts, pb.Execution_CACHED); err != nil {
			return "", createdExecution, pb.Execution_FAILED, fmt.Errorf("failed to publish cached execution: %w", err)
		}
		*execution.Cached = true
		return pvcName, createdExecution, pb.Execution_CACHED, nil
	}

	// Create a PersistentVolumeClaim object
	pvc := &k8score.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:        pvcName,
			Annotations: pvcAnnotations,
		},
		Spec: k8score.PersistentVolumeClaimSpec{
			AccessModes: accessModes,
			Resources: k8score.VolumeResourceRequirements{
				Requests: k8score.ResourceList{
					k8score.ResourceStorage: k8sres.MustParse(volumeSizeInput.GetStringValue()),
				},
			},
			StorageClassName: &storageClassName,
			VolumeName:       volumeName,
		},
	}

	// Create the PVC in the cluster
	createdPVC, err := k8sClient.CoreV1().PersistentVolumeClaims(opts.Namespace).Create(context.Background(), pvc, metav1.CreateOptions{})
	if err != nil {
		return "", createdExecution, pb.Execution_FAILED, fmt.Errorf("failed to create pvc: %w", err)
	}
	glog.Infof("Created PVC %s\n", createdPVC.ObjectMeta.Name)

	// Create a cache entry
	/*
		id := createdExecution.GetID()
		if id == 0 {
			return "", createdExecution, pb.Execution_FAILED, fmt.Errorf("failed to get id from createdExecution")
		}
	*/
	if opts.Task.GetCachingOptions().GetEnableCache() {
		err = createCache(ctx, createdExecution, opts, taskStartedTime, fingerPrint, cacheClient)
		if err != nil {
			return "", createdExecution, pb.Execution_FAILED, fmt.Errorf("failed to create cache entry for create pvc: %w", err)
		}
	}

	return createdPVC.ObjectMeta.Name, createdExecution, pb.Execution_COMPLETE, nil
}

func deletePVC(
	ctx context.Context,
	k8sClient kubernetes.Interface,
	execution Execution,
	opts *Options,
	cacheClient *cacheutils.Client,
	mlmd *metadata.Client,
	ecfg *metadata.ExecutionConfig,
) (createdExecution *metadata.Execution, status pb.Execution_State, err error) {

	// Create execution regardless the operation succeeds or not
	defer func() {
		if createdExecution == nil {
			pipeline, err := mlmd.GetPipeline(ctx, opts.PipelineName, opts.RunID, "", "", "", "")
			if err != nil {
				return
			}
			createdExecution, err = mlmd.CreateExecution(ctx, pipeline, ecfg)
		}
	}()

	taskStartedTime := time.Now().Unix()

	inputs := execution.ExecutorInput.Inputs
	glog.Infof("Input parameter values: %+v", inputs.ParameterValues)

	// Required input: pvc_name
	pvcNameInput, ok := inputs.ParameterValues["pvc_name"]
	if !ok || pvcNameInput == nil {
		return createdExecution, pb.Execution_FAILED, fmt.Errorf("failed to delete pvc: required parameter pvc_name not provided")
	}
	pvcName := pvcNameInput.GetStringValue()

	// Get execution fingerprint and MLMD ID for caching
	// If pvcName includes a randomly generated UUID, it is added in the execution input as a key-value pair for this purpose only
	// The original execution is not changed.
	fingerPrint, cachedMLMDExecutionID, err := getFingerPrintsAndID(&execution, opts, cacheClient)
	if err != nil {
		return createdExecution, pb.Execution_FAILED, err
	}
	ecfg.CachedMLMDExecutionID = cachedMLMDExecutionID
	ecfg.FingerPrint = fingerPrint

	pipeline, err := mlmd.GetPipeline(ctx, opts.PipelineName, opts.RunID, "", "", "", "")
	if err != nil {
		return createdExecution, pb.Execution_FAILED, fmt.Errorf("error getting pipeline from MLMD: %w", err)
	}

	// Create execution in MLMD
	// TODO(Bobgy): change execution state to pending, because this is driver, execution hasn't started.
	createdExecution, err = mlmd.CreateExecution(ctx, pipeline, ecfg)
	if err != nil {
		return createdExecution, pb.Execution_FAILED, fmt.Errorf("error creating MLMD execution for createpvc: %w", err)
	}
	glog.Infof("Created execution: %s", createdExecution)
	execution.ID = createdExecution.GetID()
	if !execution.WillTrigger() {
		return createdExecution, pb.Execution_COMPLETE, nil
	}

	// Use cache and skip createpvc if all conditions met:
	// (1) Cache is enabled
	// (2) CachedMLMDExecutionID is non-empty, which means a cache entry exists
	cached := false
	execution.Cached = &cached
	if opts.Task.GetCachingOptions().GetEnableCache() && ecfg.CachedMLMDExecutionID != "" {
		executorOutput, outputArtifacts, err := reuseCachedOutputs(ctx, execution.ExecutorInput, mlmd, ecfg.CachedMLMDExecutionID)
		if err != nil {
			return createdExecution, pb.Execution_FAILED, err
		}
		// TODO(Bobgy): upload output artifacts.
		// TODO(Bobgy): when adding artifacts, we will need execution.pipeline to be non-nil, because we need
		// to publish output artifacts to the context too.
		if err := mlmd.PublishExecution(ctx, createdExecution, executorOutput.GetParameterValues(), outputArtifacts, pb.Execution_CACHED); err != nil {
			return createdExecution, pb.Execution_FAILED, fmt.Errorf("failed to publish cached execution: %w", err)
		}
		*execution.Cached = true
		return createdExecution, pb.Execution_CACHED, nil
	}

	// Get the PVC you want to delete, verify that it exists.
	_, err = k8sClient.CoreV1().PersistentVolumeClaims(opts.Namespace).Get(context.TODO(), pvcName, metav1.GetOptions{})
	if err != nil {
		return createdExecution, pb.Execution_FAILED, fmt.Errorf("failed to delete pvc %s: cannot find pvc: %v", pvcName, err)
	}

	// Delete the PVC.
	err = k8sClient.CoreV1().PersistentVolumeClaims(opts.Namespace).Delete(context.TODO(), pvcName, metav1.DeleteOptions{})
	if err != nil {
		return createdExecution, pb.Execution_FAILED, fmt.Errorf("failed to delete pvc %s: %v", pvcName, err)
	}

	glog.Infof("Deleted PVC %s\n", pvcName)

	/*
		// Create a cache entry
		id := createdExecution.GetID()
		if id == 0 {
			return createdExecution, pb.Execution_FAILED, fmt.Errorf("failed to get id from createdExecution")
		}
	*/
	if opts.Task.GetCachingOptions().GetEnableCache() && ecfg.CachedMLMDExecutionID != "" {
		err = createCache(ctx, createdExecution, opts, taskStartedTime, fingerPrint, cacheClient)
		if err != nil {
			return createdExecution, pb.Execution_FAILED, fmt.Errorf("failed to create cache entry for delete pvc: %w", err)
		}
	}

	return createdExecution, pb.Execution_COMPLETE, nil
}

func createK8sClient() (*kubernetes.Clientset, error) {
	// Initialize Kubernetes client set
	restConfig, err := util.GetKubernetesConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize kubernetes client: %w", err)
	}
	k8sClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize kubernetes client set: %w", err)
	}
	return k8sClient, nil
}

func makeVolumeMountPatch(
	ctx context.Context,
	opts Options,
	pvcMounts []*kubernetesplatform.PvcMount,
	dag *metadata.DAG,
	pipeline *metadata.Pipeline,
	mlmd *metadata.Client,
	inputParams map[string]*structpb.Value,
) ([]k8score.VolumeMount, []k8score.Volume, error) {
	if pvcMounts == nil {
		return nil, nil, nil
	}
	var volumeMounts []k8score.VolumeMount
	var volumes []k8score.Volume
	for _, pvcMount := range pvcMounts {
		var pvcNameParameter *pipelinespec.TaskInputsSpec_InputParameterSpec
		if pvcMount.PvcNameParameter != nil {
			pvcNameParameter = pvcMount.PvcNameParameter
		} else { // Support deprecated fields
			if pvcMount.GetConstant() != "" {
				pvcNameParameter = inputParamConstant(pvcMount.GetConstant())
			} else if pvcMount.GetTaskOutputParameter() != nil {
				pvcNameParameter = inputParamTaskOutput(
					pvcMount.GetTaskOutputParameter().GetProducerTask(),
					pvcMount.GetTaskOutputParameter().GetOutputParameterKey(),
				)
			} else if pvcMount.GetComponentInputParameter() != "" {
				pvcNameParameter = inputParamComponent(pvcMount.GetComponentInputParameter())
			} else {
				return nil, nil, fmt.Errorf("failed to make podSpecPatch: volume mount: volume name not provided")
			}
		}

		resolvedPvcName, err := resolveInputParameterStr(ctx, dag, pipeline, opts, mlmd,
			pvcNameParameter, inputParams)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to resolve pvc name: %w", err)
		}
		pvcName := resolvedPvcName.GetStringValue()

		pvcMountPath := pvcMount.GetMountPath()
		if pvcName == "" || pvcMountPath == "" {
			return nil, nil, fmt.Errorf("failed to mount volume, missing mountpath or pvc name")
		}
		volumeMount := k8score.VolumeMount{
			Name:      pvcName,
			MountPath: pvcMountPath,
		}
		volume := k8score.Volume{
			Name: pvcName,
			VolumeSource: k8score.VolumeSource{
				PersistentVolumeClaim: &k8score.PersistentVolumeClaimVolumeSource{ClaimName: pvcName},
			},
		}
		volumeMounts = append(volumeMounts, volumeMount)
		volumes = append(volumes, volume)
	}
	return volumeMounts, volumes, nil
}

func getFingerPrintsAndID(execution *Execution, opts *Options, cacheClient *cacheutils.Client) (string, string, error) {
	if execution.WillTrigger() && opts.Task.GetCachingOptions().GetEnableCache() {
		glog.Infof("Task {%s} enables cache", opts.Task.GetTaskInfo().GetName())
		fingerPrint, err := getFingerPrint(*opts, execution.ExecutorInput)
		if err != nil {
			return "", "", fmt.Errorf("failure while getting fingerPrint: %w", err)
		}
		cachedMLMDExecutionID, err := cacheClient.GetExecutionCache(fingerPrint, "pipeline/"+opts.PipelineName, opts.Namespace)
		if err != nil {
			return "", "", fmt.Errorf("failure while getting executionCache: %w", err)
		}
		return fingerPrint, cachedMLMDExecutionID, nil
	} else {
		return "", "", nil
	}
}

func createCache(
	ctx context.Context,
	execution *metadata.Execution,
	opts *Options,
	taskStartedTime int64,
	fingerPrint string,
	cacheClient *cacheutils.Client,
) error {
	id := execution.GetID()
	if id == 0 {
		return fmt.Errorf("failed to get id from createdExecution")
	}
	task := &api.Task{
		//TODO how to differentiate between shared pipeline and namespaced pipeline
		PipelineName:    "pipeline/" + opts.PipelineName,
		Namespace:       opts.Namespace,
		RunId:           opts.RunID,
		MlmdExecutionID: strconv.FormatInt(id, 10),
		CreatedAt:       &timestamp.Timestamp{Seconds: taskStartedTime},
		FinishedAt:      &timestamp.Timestamp{Seconds: time.Now().Unix()},
		Fingerprint:     fingerPrint,
	}
	err := cacheClient.CreateExecutionCache(ctx, task)
	if err != nil {
		return err
	}
	glog.Infof("Created cache entry.")
	return nil
}
