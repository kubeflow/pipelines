// Copyright 2025 The Kubeflow Authors
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
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/cacheutils"
	"github.com/kubeflow/pipelines/backend/src/v2/component"
	"github.com/kubeflow/pipelines/backend/src/v2/config"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	"github.com/kubeflow/pipelines/kubernetes_platform/go/kubernetesplatform"
	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"google.golang.org/protobuf/types/known/structpb"
	k8score "k8s.io/api/core/v1"
	k8sres "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var accessModeMap = map[string]k8score.PersistentVolumeAccessMode{
	"ReadWriteOnce":    k8score.ReadWriteOnce,
	"ReadOnlyMany":     k8score.ReadOnlyMany,
	"ReadWriteMany":    k8score.ReadWriteMany,
	"ReadWriteOncePod": k8score.ReadWriteOncePod,
}

var dummyImages = map[string]string{
	"argostub/createpvc": "create PVC",
	"argostub/deletepvc": "delete PVC",
}

// kubernetesPlatformOps() carries out the Kubernetes-specific operations, such as create PVC,
// delete PVC, etc. In these operations we skip the launcher due to there being no user container.
// It also prepublishes and publishes the execution, which are usually done in the launcher.
func kubernetesPlatformOps(
	ctx context.Context,
	mlmd *metadata.Client,
	cacheClient cacheutils.Client,
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

// GetWorkspacePVCName gets the name of the workspace PVC for a given run name. runName is the resolved Argo Workflows
// variable of {{workflow.name}}
func GetWorkspacePVCName(runName string) string {
	return fmt.Sprintf("%s-%s", runName, component.WorkspaceVolumeName)
}

// execution is passed by value because we make changes to it to generate  fingerprint
func createPVC(
	ctx context.Context,
	k8sClient kubernetes.Interface,
	execution Execution,
	opts *Options,
	cacheClient cacheutils.Client,
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
	pvcAnnotations := make(map[string]string)
	if pvcAnnotationsInput, ok := inputs.ParameterValues["annotations"]; ok && pvcAnnotationsInput != nil {
		for key, val := range pvcAnnotationsInput.GetStructValue().GetFields() {
			pvcAnnotations[key] = val.GetStringValue()
		}
	}

	// Optional input: volume_name
	var volumeName string
	if volumeNameInput, ok := inputs.ParameterValues["volume_name"]; ok && volumeNameInput != nil {
		volumeName = volumeNameInput.GetStringValue()
	}

	// Get execution fingerprint and MLMD ID for caching
	// If pvcName includes a randomly generated UUID, it is added in the execution input as a key-value pair for this purpose only
	// The original execution is not changed.
	fingerPrint, cachedMLMDExecutionID, err := getFingerPrintsAndID(&execution, opts, cacheClient, nil)
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
	// (1) Cache is enabled globally
	// (2) Cache is enabled for the task
	// (3) CachedMLMDExecutionID is non-empty, which means a cache entry exists
	cached := false
	execution.Cached = &cached
	if !opts.CacheDisabled && opts.Task.GetCachingOptions().GetEnableCache() && ecfg.CachedMLMDExecutionID != "" {
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
	if !opts.CacheDisabled && opts.Task.GetCachingOptions().GetEnableCache() {
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
	cacheClient cacheutils.Client,
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
	fingerPrint, cachedMLMDExecutionID, err := getFingerPrintsAndID(&execution, opts, cacheClient, nil)
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
	// (1) Cache is enabled globally
	// (2) Cache is enabled for the task
	// (3) CachedMLMDExecutionID is non-empty, which means a cache entry exists
	cached := false
	execution.Cached = &cached
	if !opts.CacheDisabled && opts.Task.GetCachingOptions().GetEnableCache() && ecfg.CachedMLMDExecutionID != "" {
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

	// Create a cache entry
	if !opts.CacheDisabled && opts.Task.GetCachingOptions().GetEnableCache() && ecfg.CachedMLMDExecutionID != "" {
		err = createCache(ctx, createdExecution, opts, taskStartedTime, fingerPrint, cacheClient)
		if err != nil {
			return createdExecution, pb.Execution_FAILED, fmt.Errorf("failed to create cache entry for delete pvc: %w", err)
		}
	}

	return createdExecution, pb.Execution_COMPLETE, nil
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
			SubPath:   pvcMount.GetSubPath(),
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
