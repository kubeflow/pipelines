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
	"fmt"
	"strconv"
	"time"

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
	"github.com/kubeflow/pipelines/backend/src/v2/objectstore"
	"github.com/kubeflow/pipelines/kubernetes_platform/go/kubernetesplatform"
	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	k8score "k8s.io/api/core/v1"
	k8sres "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var dummyImages = map[string]string{
	"argostub/createpvc": "create PVC",
	"argostub/deletepvc": "delete PVC",
}

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

	// set to true if ml pipeline server is serving over tls
	MLPipelineTLSEnabled bool

	MLMDServerAddress string

	MLMDServerPort string

	// set to true if MLMD server is serving over tls
	MLMDTLSEnabled bool

	CaCertPath string
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
	err = validateRootDAG(opts)
	if err != nil {
		return nil, err
	}
	// TODO(v2): in pipeline spec, rename GCS output directory to pipeline root.
	pipelineRoot := opts.RuntimeConfig.GetGcsOutputDirectory()

	restConfig, err := rest.InClusterConfig()
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
	inputs, err := resolveInputs(ctx, dag, iterationIndex, pipeline, opts.Task, opts.Component.GetInputDefinitions(), mlmd, expr)
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
	if execution.WillTrigger() {
		executorInput.Outputs = provisionOutputs(pipeline.GetPipelineRoot(), opts.Task.GetTaskInfo().GetName(), opts.Component.GetOutputDefinitions())
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

	// When the container image is a dummy image, there is no launcher for this task.
	// This happens when this task is created to implement a Kubernetes-specific configuration, i.e.,
	// there is no user container to run.
	// It publishes execution details to mlmd in driver and takes care of caching, which are usually done in launcher.
	// We also skip creating the podspecpatch in these cases.
	if _, ok := dummyImages[opts.Container.Image]; ok {
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
		executorOutput, outputArtifacts, err := reuseCachedOutputs(ctx, execution.ExecutorInput, opts.Component.GetOutputDefinitions(), mlmd, ecfg.CachedMLMDExecutionID)
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

	podSpec, err := initPodSpecPatch(opts.Container, opts.Component, executorInput, execution.ID, opts.PipelineName, opts.RunID, opts.MLPipelineTLSEnabled, opts.MLMDServerAddress, opts.MLMDServerPort, opts.MLMDTLSEnabled, opts.CaCertPath)
	if err != nil {
		return execution, err
	}
	if opts.KubernetesExecutorConfig != nil {
		dagTasks, err := mlmd.GetExecutionsInDAG(ctx, dag, pipeline)
		if err != nil {
			return execution, err
		}
		err = extendPodSpecPatch(podSpec, opts.KubernetesExecutorConfig, dag, dagTasks)
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
	mlPipelineTLSEnabled bool,
	mlmdServerAddress string,
	mlmdServerPort string,
	mlmdTLSEnabled bool,
	caCertPath string,
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

	userCmdArgs := make([]string, 0, len(container.Command)+len(container.Args))
	userCmdArgs = append(userCmdArgs, container.Command...)
	userCmdArgs = append(userCmdArgs, container.Args...)
	launcherCmd := []string{
		// TODO(Bobgy): workaround argo emissary executor bug, after we upgrade to an argo version with the bug fix, we can remove the following line.
		// Reference: https://github.com/argoproj/argo-workflows/issues/7406
		"/var/run/argo/argoexec", "emissary", "--",
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
		"--mlmd_server_address", mlmdServerAddress,
		"--mlmd_server_port", mlmdServerPort,
		"--metadataTLSEnabled", fmt.Sprintf("%v", mlmdTLSEnabled),
		"--mlPipelineServiceTLSEnabled",
		fmt.Sprintf("%v", mlPipelineTLSEnabled),
		"--ca_cert_path", caCertPath,
		"--", // separater before user command and args
	}
	res := k8score.ResourceRequirements{
		Limits:   map[k8score.ResourceName]k8sres.Quantity{},
		Requests: map[k8score.ResourceName]k8sres.Quantity{},
	}
	memoryLimit := container.GetResources().GetMemoryLimit()
	if memoryLimit != 0 {
		q, err := k8sres.ParseQuantity(fmt.Sprintf("%vG", memoryLimit))
		if err != nil {
			return nil, fmt.Errorf("failed to init podSpecPatch: %w", err)
		}
		res.Limits[k8score.ResourceMemory] = q
	}
	memoryRequest := container.GetResources().GetMemoryRequest()
	if memoryRequest != 0 {
		q, err := k8sres.ParseQuantity(fmt.Sprintf("%vG", memoryRequest))
		if err != nil {
			return nil, err
		}
		res.Requests[k8score.ResourceMemory] = q
	}
	cpuLimit := container.GetResources().GetCpuLimit()
	if cpuLimit != 0 {
		q, err := k8sres.ParseQuantity(fmt.Sprintf("%v", cpuLimit))
		if err != nil {
			return nil, fmt.Errorf("failed to init podSpecPatch: %w", err)
		}
		res.Limits[k8score.ResourceCPU] = q
	}
	cpuRequest := container.GetResources().GetCpuRequest()
	if cpuRequest != 0 {
		q, err := k8sres.ParseQuantity(fmt.Sprintf("%v", cpuRequest))
		if err != nil {
			return nil, err
		}
		res.Requests[k8score.ResourceCPU] = q
	}
	accelerator := container.GetResources().GetAccelerator()
	if accelerator != nil {
		if accelerator.GetType() != "" && accelerator.GetCount() > 0 {
			q, err := k8sres.ParseQuantity(fmt.Sprintf("%v", accelerator.GetCount()))
			if err != nil {
				return nil, fmt.Errorf("failed to init podSpecPatch: %w", err)
			}
			res.Limits[k8score.ResourceName(accelerator.GetType())] = q
		}
	}
	podSpec := &k8score.PodSpec{
		Containers: []k8score.Container{{
			Name:      "main", // argo task user container is always called "main"
			Command:   launcherCmd,
			Args:      userCmdArgs,
			Image:     container.Image,
			Resources: res,
			Env:       userEnvVar,
		}},
	}
	return podSpec, nil
}

// Extends the PodSpec to include Kubernetes-specific executor config.
func extendPodSpecPatch(
	podSpec *k8score.PodSpec,
	kubernetesExecutorConfig *kubernetesplatform.KubernetesExecutorConfig,
	dag *metadata.DAG,
	dagTasks map[string]*metadata.Execution,
) error {
	// Return an error if the podSpec has no user container.
	if len(podSpec.Containers) == 0 {
		return fmt.Errorf("failed to patch the pod with kubernetes-specific config due to missing user container: %v", podSpec)
	}
	// Get volume mount information
	if kubernetesExecutorConfig.GetPvcMount() != nil {
		volumeMounts, volumes, err := makeVolumeMountPatch(kubernetesExecutorConfig.GetPvcMount(), dag, dagTasks)
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
		podSpec.NodeSelector = kubernetesExecutorConfig.GetNodeSelector().GetLabels()
	}

	if tolerations := kubernetesExecutorConfig.GetTolerations(); tolerations != nil {
		var k8sTolerations []k8score.Toleration

		glog.Infof("Tolerations passed: %+v", tolerations)

		for _, toleration := range tolerations {
			if toleration != nil {
				k8sToleration := k8score.Toleration{
					Key:               toleration.Key,
					Operator:          k8score.TolerationOperator(toleration.Operator),
					Value:             toleration.Value,
					Effect:            k8score.TaintEffect(toleration.Effect),
					TolerationSeconds: toleration.TolerationSeconds,
				}

				k8sTolerations = append(k8sTolerations, k8sToleration)
			}
		}

		podSpec.Tolerations = k8sTolerations
	}

	// Get secret mount information
	for _, secretAsVolume := range kubernetesExecutorConfig.GetSecretAsVolume() {
		optional := secretAsVolume.Optional != nil && *secretAsVolume.Optional
		secretVolume := k8score.Volume{
			Name: secretAsVolume.GetSecretName(),
			VolumeSource: k8score.VolumeSource{
				Secret: &k8score.SecretVolumeSource{SecretName: secretAsVolume.GetSecretName(), Optional: &optional},
			},
		}
		secretVolumeMount := k8score.VolumeMount{
			Name:      secretAsVolume.GetSecretName(),
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
			secretEnvVar.ValueFrom.SecretKeyRef.LocalObjectReference.Name = secretAsEnv.GetSecretName()
			podSpec.Containers[0].Env = append(podSpec.Containers[0].Env, secretEnvVar)
		}
	}

	// Get config map mount information
	for _, configMapAsVolume := range kubernetesExecutorConfig.GetConfigMapAsVolume() {
		optional := configMapAsVolume.Optional != nil && *configMapAsVolume.Optional
		configMapVolume := k8score.Volume{
			Name: configMapAsVolume.GetConfigMapName(),
			VolumeSource: k8score.VolumeSource{
				ConfigMap: &k8score.ConfigMapVolumeSource{
					LocalObjectReference: k8score.LocalObjectReference{Name: configMapAsVolume.GetConfigMapName()}, Optional: &optional},
			},
		}
		configMapVolumeMount := k8score.VolumeMount{
			Name:      configMapAsVolume.GetConfigMapName(),
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
			configMapEnvVar.ValueFrom.ConfigMapKeyRef.LocalObjectReference.Name = configMapAsEnv.GetConfigMapName()
			podSpec.Containers[0].Env = append(podSpec.Containers[0].Env, configMapEnvVar)
		}
	}

	// Get image pull secret information
	for _, imagePullSecret := range kubernetesExecutorConfig.GetImagePullSecret() {
		podSpec.ImagePullSecrets = append(podSpec.ImagePullSecrets, k8score.LocalObjectReference{Name: imagePullSecret.GetSecretName()})
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
							Resources: k8score.ResourceRequirements{
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
	return nil
}

// TODO(Bobgy): merge DAG driver and container driver, because they are very similar.
func DAG(ctx context.Context, opts Options, mlmd *metadata.Client) (execution *Execution, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("driver.DAG(%s) failed: %w", opts.info(), err)
		}
	}()
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
	inputs, err := resolveInputs(ctx, dag, iterationIndex, pipeline, opts.Task, opts.Component.GetInputDefinitions(), mlmd, expr)
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

func reuseCachedOutputs(ctx context.Context, executorInput *pipelinespec.ExecutorInput, outputDefinitions *pipelinespec.ComponentOutputsSpec, mlmd *metadata.Client, cachedMLMDExecutionID string) (*pipelinespec.ExecutorOutput, []*metadata.OutputArtifact, error) {
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

func resolveInputs(ctx context.Context, dag *metadata.DAG, iterationIndex *int, pipeline *metadata.Pipeline, task *pipelinespec.PipelineTaskSpec, inputsSpec *pipelinespec.ComponentInputsSpec, mlmd *metadata.Client, expr *expression.Expr) (inputs *pipelinespec.ExecutorInput_Inputs, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to resolve inputs: %w", err)
		}
	}()
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
	// get executions in context on demand
	var tasksCache map[string]*metadata.Execution
	getDAGTasks := func() (map[string]*metadata.Execution, error) {
		if tasksCache != nil {
			return tasksCache, nil
		}
		tasks, err := mlmd.GetExecutionsInDAG(ctx, dag, pipeline)
		if err != nil {
			return nil, err
		}
		tasksCache = tasks
		return tasks, nil
	}
	for name, paramSpec := range task.GetInputs().GetParameters() {
		paramError := func(err error) error {
			return fmt.Errorf("resolving input parameter %s with spec %s: %w", name, paramSpec, err)
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
			inputs.ParameterValues[name] = v

		case *pipelinespec.TaskInputsSpec_InputParameterSpec_TaskOutputParameter:
			taskOutput := paramSpec.GetTaskOutputParameter()
			if taskOutput.GetProducerTask() == "" {
				return nil, paramError(fmt.Errorf("producer task is empty"))
			}
			if taskOutput.GetOutputParameterKey() == "" {
				return nil, paramError(fmt.Errorf("output parameter key is empty"))
			}
			tasks, err := getDAGTasks()
			if err != nil {
				return nil, paramError(err)
			}
			producer, ok := tasks[taskOutput.GetProducerTask()]
			if !ok {
				return nil, paramError(fmt.Errorf("cannot find producer task %q", taskOutput.GetProducerTask()))
			}
			_, outputs, err := producer.GetParameters()
			if err != nil {
				return nil, paramError(fmt.Errorf("get producer output parameters: %w", err))
			}
			param, ok := outputs[taskOutput.GetOutputParameterKey()]
			if !ok {
				return nil, paramError(fmt.Errorf("cannot find output parameter key %q in producer task %q", taskOutput.GetOutputParameterKey(), taskOutput.GetProducerTask()))
			}
			inputs.ParameterValues[name] = param
		case *pipelinespec.TaskInputsSpec_InputParameterSpec_RuntimeValue:
			runtimeValue := paramSpec.GetRuntimeValue()
			switch t := runtimeValue.Value.(type) {
			case *pipelinespec.ValueOrRuntimeParameter_Constant:
				inputs.ParameterValues[name] = runtimeValue.GetConstant()
			default:
				return nil, paramError(fmt.Errorf("param runtime value spec of type %T not implemented", t))
			}

		// TODO(Bobgy): implement the following cases
		// case *pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus_:
		default:
			return nil, paramError(fmt.Errorf("parameter spec of type %T not implemented yet", t))
		}
	}
	for name, artifactSpec := range task.GetInputs().GetArtifacts() {
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
			inputs.Artifacts[name] = v

		case *pipelinespec.TaskInputsSpec_InputArtifactSpec_TaskOutputArtifact:
			taskOutput := artifactSpec.GetTaskOutputArtifact()
			if taskOutput.GetProducerTask() == "" {
				return nil, artifactError(fmt.Errorf("producer task is empty"))
			}
			if taskOutput.GetOutputArtifactKey() == "" {
				return nil, artifactError(fmt.Errorf("output artifact key is empty"))
			}
			tasks, err := getDAGTasks()
			if err != nil {
				return nil, artifactError(err)
			}
			producer, ok := tasks[taskOutput.GetProducerTask()]
			if !ok {
				return nil, artifactError(fmt.Errorf("cannot find producer task %q", taskOutput.GetProducerTask()))
			}
			// TODO(Bobgy): cache results
			outputs, err := mlmd.GetOutputArtifactsByExecutionId(ctx, producer.GetID())
			if err != nil {
				return nil, artifactError(err)
			}
			artifact, ok := outputs[taskOutput.GetOutputArtifactKey()]
			if !ok {
				return nil, artifactError(fmt.Errorf("cannot find output artifact key %q in producer task %q", taskOutput.GetOutputArtifactKey(), taskOutput.GetProducerTask()))
			}
			runtimeArtifact, err := artifact.ToRuntimeArtifact()
			if err != nil {
				return nil, artifactError(err)
			}
			inputs.Artifacts[name] = &pipelinespec.ArtifactList{
				Artifacts: []*pipelinespec.RuntimeArtifact{runtimeArtifact},
			}
		default:
			return nil, artifactError(fmt.Errorf("artifact spec of type %T not implemented yet", t))
		}
	}
	// TODO(Bobgy): validate executor inputs match component inputs definition
	return inputs, nil
}

func provisionOutputs(pipelineRoot, taskName string, outputsSpec *pipelinespec.ComponentOutputsSpec) *pipelinespec.ExecutorInput_Outputs {
	outputs := &pipelinespec.ExecutorInput_Outputs{
		Artifacts:  make(map[string]*pipelinespec.ArtifactList),
		Parameters: make(map[string]*pipelinespec.ExecutorInput_OutputParameter),
		OutputFile: component.OutputMetadataFilepath,
	}
	for name, artifact := range outputsSpec.GetArtifacts() {
		outputs.Artifacts[name] = &pipelinespec.ArtifactList{
			Artifacts: []*pipelinespec.RuntimeArtifact{
				{
					// Do not preserve the query string for output artifacts, as otherwise
					// they'd appear in file and artifact names.
					Uri:      metadata.GenerateOutputURI(pipelineRoot, []string{taskName, name}, false),
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

	// Requied input: access_modes
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
		executorOutput, outputArtifacts, err := reuseCachedOutputs(ctx, execution.ExecutorInput, opts.Component.GetOutputDefinitions(), mlmd, ecfg.CachedMLMDExecutionID)
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
			Resources: k8score.ResourceRequirements{
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
	err = createCache(ctx, createdExecution, opts, taskStartedTime, fingerPrint, cacheClient)
	if err != nil {
		return "", createdExecution, pb.Execution_FAILED, fmt.Errorf("failed to create cache entrty for create pvc: %w", err)
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
		executorOutput, outputArtifacts, err := reuseCachedOutputs(ctx, execution.ExecutorInput, opts.Component.GetOutputDefinitions(), mlmd, ecfg.CachedMLMDExecutionID)
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
	err = createCache(ctx, createdExecution, opts, taskStartedTime, fingerPrint, cacheClient)
	if err != nil {
		return createdExecution, pb.Execution_FAILED, fmt.Errorf("failed to create cache entrty for delete pvc: %w", err)
	}

	return createdExecution, pb.Execution_COMPLETE, nil
}

func createK8sClient() (*kubernetes.Clientset, error) {
	// Initialize Kubernetes client set
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize kubernetes client: %w", err)
	}
	k8sClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize kubernetes client set: %w", err)
	}
	return k8sClient, nil
}

func makeVolumeMountPatch(pvcMount []*kubernetesplatform.PvcMount, dag *metadata.DAG, dagTasks map[string]*metadata.Execution) ([]k8score.VolumeMount, []k8score.Volume, error) {
	if pvcMount == nil {
		return nil, nil, nil
	}
	var volumeMounts []k8score.VolumeMount
	var volumes []k8score.Volume
	for _, vmc := range pvcMount {
		// Find mount path
		if vmc.GetMountPath() == "" {
			return nil, nil, fmt.Errorf("failed to make podSpecPatch: volume mount: volume mount path not provided")
		}
		volumeMount := k8score.VolumeMount{
			MountPath: vmc.GetMountPath(),
		}
		volume := k8score.Volume{}

		// Volume name may come from three different sources:
		// 1) A constant
		// 2) As a task output parameter
		// 3) As a component input parameter
		if vmc.GetConstant() != "" {
			volumeMount.Name = vmc.GetConstant()
			volume.Name = vmc.GetConstant()
			volume.PersistentVolumeClaim = &k8score.PersistentVolumeClaimVolumeSource{ClaimName: vmc.GetConstant()}
		} else if vmc.GetTaskOutputParameter() != nil {
			if vmc.GetTaskOutputParameter().GetProducerTask() == "" {
				return nil, nil, fmt.Errorf("failed to make podSpecPatch: volume mount: producer task empty")
			}
			if vmc.GetTaskOutputParameter().GetOutputParameterKey() == "" {
				return nil, nil, fmt.Errorf("failed to make podSpecPatch: volume mount: OutputParameterKey")
			}
			producer, ok := dagTasks[vmc.GetTaskOutputParameter().GetProducerTask()]
			if !ok {
				return nil, nil, fmt.Errorf("failed to make podSpecPatch: volume mount: cannot find producer task %s", vmc.GetTaskOutputParameter().GetProducerTask())
			}
			_, outputs, err := producer.GetParameters()
			if err != nil {
				return nil, nil, fmt.Errorf("failed to make podSpecPatch: volume mount: cannot get producer output: %w", err)
			}
			pvcName, ok := outputs[vmc.GetTaskOutputParameter().GetOutputParameterKey()]
			if !ok {
				return nil, nil, fmt.Errorf("failed to make podSpecPatch: volume mount: cannot find output parameter %s from producer task %s", vmc.GetTaskOutputParameter().GetOutputParameterKey(), vmc.GetTaskOutputParameter().GetProducerTask())
			}
			volumeMount.Name = pvcName.GetStringValue()
			volume.Name = pvcName.GetStringValue()
			volume.PersistentVolumeClaim = &k8score.PersistentVolumeClaimVolumeSource{ClaimName: pvcName.GetStringValue()}
		} else if vmc.GetComponentInputParameter() != "" {
			inputParams, _, err := dag.Execution.GetParameters()
			if err != nil {
				return nil, nil, fmt.Errorf("failed to make podSpecPatch: volume mount: error getting input parameters")
			}
			glog.Infof("parent DAG input parameters %+v", inputParams)
			pvcName, ok := inputParams[vmc.GetComponentInputParameter()]
			if !ok {
				return nil, nil, fmt.Errorf("failed to make podSpecPatch: volume mount:component input parameters %s doesn't exist", vmc.GetComponentInputParameter())
			}
			volumeMount.Name = pvcName.GetStringValue()
			volume.Name = pvcName.GetStringValue()
			volume.PersistentVolumeClaim = &k8score.PersistentVolumeClaimVolumeSource{ClaimName: pvcName.GetStringValue()}
		} else {
			return nil, nil, fmt.Errorf("failed to make podSpecPatch: volume mount: volume name not provided")
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
		fmt.Errorf("failed to get id from createdExecution")
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
