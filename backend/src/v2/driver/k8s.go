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
	"encoding/json"
	"errors"
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
	taskConfig *TaskConfig,
) error {
	kubernetesExecutorConfig := opts.KubernetesExecutorConfig

	setOnTaskConfig, setOnPod := getTaskConfigOptions(opts.Component)

	// Always set setOnTaskConfig to an empty map if taskConfig is nil to avoid nil pointer dereference.
	if taskConfig == nil {
		setOnTaskConfig = map[pipelinespec.TaskConfigPassthroughType_TaskConfigPassthroughTypeEnum]bool{}
	}

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

		if setOnTaskConfig[pipelinespec.TaskConfigPassthroughType_KUBERNETES_VOLUMES] {
			taskConfig.VolumeMounts = append(taskConfig.VolumeMounts, volumeMounts...)
			taskConfig.Volumes = append(taskConfig.Volumes, volumes...)
		}

		if setOnPod[pipelinespec.TaskConfigPassthroughType_KUBERNETES_VOLUMES] {
			podSpec.Volumes = append(podSpec.Volumes, volumes...)
			// We assume that the user container always gets executed first within a pod.
			podSpec.Containers[0].VolumeMounts = append(podSpec.Containers[0].VolumeMounts, volumeMounts...)
		}
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
		var nodeSelector map[string]string
		// skipNodeSelector marks when the node selector input resolved to a null optional
		// value. In that case we avoid appending an empty selector to the pod spec.
		skipNodeSelector := false
		if kubernetesExecutorConfig.GetNodeSelector().GetNodeSelectorJson() != nil {
			err := resolveK8sJsonParameter(ctx, opts, dag, pipeline, mlmd,
				kubernetesExecutorConfig.GetNodeSelector().GetNodeSelectorJson(), inputParams, &nodeSelector)
			if err != nil {
				if errors.Is(err, ErrResolvedParameterNull) {
					skipNodeSelector = true
				} else {
					return fmt.Errorf("failed to resolve node selector: %w", err)
				}
			}
		} else {
			nodeSelector = kubernetesExecutorConfig.GetNodeSelector().GetLabels()
		}

		if !skipNodeSelector {
			if setOnTaskConfig[pipelinespec.TaskConfigPassthroughType_KUBERNETES_NODE_SELECTOR] {
				taskConfig.NodeSelector = nodeSelector
			}

			if setOnPod[pipelinespec.TaskConfigPassthroughType_KUBERNETES_NODE_SELECTOR] {
				podSpec.NodeSelector = nodeSelector
			}
		}
	}

	if tolerations := kubernetesExecutorConfig.GetTolerations(); tolerations != nil {
		var k8sTolerations []k8score.Toleration

		glog.Infof("Tolerations passed: %+v", tolerations)

		for _, toleration := range tolerations {
			if toleration != nil {
				k8sToleration := &k8score.Toleration{}
				if toleration.TolerationJson != nil {
					resolvedParam, err := resolveInputParameter(ctx, dag, pipeline, opts, mlmd,
						toleration.GetTolerationJson(), inputParams)
					if err != nil {
						if errors.Is(err, ErrResolvedParameterNull) {
							continue // Skip applying the patch for this null/optional parameter
						}
						return fmt.Errorf("failed to resolve toleration: %w", err)
					}

					// TolerationJson can be either a single toleration or list of tolerations
					// the field accepts both, and in both cases the tolerations are appended
					// to the total executor pod toleration list.
					var paramJSON []byte
					isSingleToleration := resolvedParam.GetStructValue() != nil
					isListToleration := resolvedParam.GetListValue() != nil
					if isSingleToleration {
						structVal := resolvedParam.GetStructValue()
						if structVal != nil && len(structVal.Fields) > 0 {
							paramJSON, err = resolvedParam.GetStructValue().MarshalJSON()
							if err != nil {
								return err
							}
							var singleToleration k8score.Toleration
							if err = json.Unmarshal(paramJSON, &singleToleration); err != nil {
								return fmt.Errorf("failed to marshal single toleration to json: %w", err)
							}
							k8sTolerations = append(k8sTolerations, singleToleration)
						} else {
							glog.V(4).Info("encountered empty tolerations struct, ignoring.")
						}
					} else if isListToleration {
						listVal := resolvedParam.GetListValue()
						if listVal != nil && len(listVal.Values) > 0 {
							paramJSON, err = resolvedParam.GetListValue().MarshalJSON()
							if err != nil {
								return err
							}
							var k8sTolerationsList []k8score.Toleration
							if err = json.Unmarshal(paramJSON, &k8sTolerationsList); err != nil {
								return fmt.Errorf("failed to marshal list toleration to json: %w", err)
							}
							k8sTolerations = append(k8sTolerations, k8sTolerationsList...)
						} else {
							glog.V(4).Info("encountered empty tolerations list, ignoring.")
						}
					} else {
						return fmt.Errorf("encountered unexpected toleration proto value, must be either struct or list type")
					}
				} else {
					k8sToleration.Key = toleration.Key
					k8sToleration.Operator = k8score.TolerationOperator(toleration.Operator)
					k8sToleration.Value = toleration.Value
					k8sToleration.Effect = k8score.TaintEffect(toleration.Effect)
					k8sToleration.TolerationSeconds = toleration.TolerationSeconds
					k8sTolerations = append(k8sTolerations, *k8sToleration)
				}

			}
		}
		if setOnTaskConfig[pipelinespec.TaskConfigPassthroughType_KUBERNETES_TOLERATIONS] {
			taskConfig.Tolerations = k8sTolerations
		}

		if setOnPod[pipelinespec.TaskConfigPassthroughType_KUBERNETES_TOLERATIONS] {
			podSpec.Tolerations = k8sTolerations
		}
	}

	// Get secret mount information
	for _, secretAsVolume := range kubernetesExecutorConfig.GetSecretAsVolume() {
		var secretName string
		if secretAsVolume.SecretNameParameter != nil {
			resolvedSecretName, err := resolveInputParameterStr(ctx, dag, pipeline, opts, mlmd,
				secretAsVolume.SecretNameParameter, inputParams)
			if err != nil {
				if errors.Is(err, ErrResolvedParameterNull) {
					continue
				}
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

		if setOnTaskConfig[pipelinespec.TaskConfigPassthroughType_KUBERNETES_VOLUMES] {
			taskConfig.Volumes = append(taskConfig.Volumes, secretVolume)
			taskConfig.VolumeMounts = append(taskConfig.VolumeMounts, secretVolumeMount)
		}

		if setOnPod[pipelinespec.TaskConfigPassthroughType_KUBERNETES_VOLUMES] {
			podSpec.Volumes = append(podSpec.Volumes, secretVolume)
			podSpec.Containers[0].VolumeMounts = append(podSpec.Containers[0].VolumeMounts, secretVolumeMount)
		}
	}

	// Get secret env information
	for _, secretAsEnv := range kubernetesExecutorConfig.GetSecretAsEnv() {
		for _, keyToEnv := range secretAsEnv.GetKeyToEnv() {
			secretKeySelector := &k8score.SecretKeySelector{
				Key: keyToEnv.GetSecretKey(),
			}

			// Set Optional field when explicitly provided (true or false), leave nil when not specified
			if secretAsEnv.Optional != nil {
				secretKeySelector.Optional = secretAsEnv.Optional
			}

			secretEnvVar := k8score.EnvVar{
				Name: keyToEnv.GetEnvVar(),
				ValueFrom: &k8score.EnvVarSource{
					SecretKeyRef: secretKeySelector,
				},
			}

			var secretName string
			if secretAsEnv.SecretNameParameter != nil {
				resolvedSecretName, err := resolveInputParameterStr(ctx, dag, pipeline, opts, mlmd,
					secretAsEnv.SecretNameParameter, inputParams)
				if err != nil {
					if errors.Is(err, ErrResolvedParameterNull) {
						continue
					}
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

			if setOnPod[pipelinespec.TaskConfigPassthroughType_ENV] {
				podSpec.Containers[0].Env = append(podSpec.Containers[0].Env, secretEnvVar)
			}

			if setOnTaskConfig[pipelinespec.TaskConfigPassthroughType_ENV] {
				taskConfig.Env = append(taskConfig.Env, secretEnvVar)
			}
		}
	}

	// Get config map mount information
	for _, configMapAsVolume := range kubernetesExecutorConfig.GetConfigMapAsVolume() {
		var configMapName string
		if configMapAsVolume.ConfigMapNameParameter != nil {
			resolvedConfigMapName, err := resolveInputParameterStr(ctx, dag, pipeline, opts, mlmd,
				configMapAsVolume.ConfigMapNameParameter, inputParams)
			if err != nil {
				if errors.Is(err, ErrResolvedParameterNull) {
					continue
				}
				return fmt.Errorf("failed to resolve configmap name: %w", err)
			}
			configMapName = resolvedConfigMapName.GetStringValue()
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

		if setOnPod[pipelinespec.TaskConfigPassthroughType_KUBERNETES_VOLUMES] {
			podSpec.Volumes = append(podSpec.Volumes, configMapVolume)
			podSpec.Containers[0].VolumeMounts = append(podSpec.Containers[0].VolumeMounts, configMapVolumeMount)
		}

		if setOnTaskConfig[pipelinespec.TaskConfigPassthroughType_KUBERNETES_VOLUMES] {
			taskConfig.Volumes = append(taskConfig.Volumes, configMapVolume)
			taskConfig.VolumeMounts = append(taskConfig.VolumeMounts, configMapVolumeMount)
		}
	}

	// Get config map env information
	for _, configMapAsEnv := range kubernetesExecutorConfig.GetConfigMapAsEnv() {
		for _, keyToEnv := range configMapAsEnv.GetKeyToEnv() {
			configMapKeySelector := &k8score.ConfigMapKeySelector{
				Key: keyToEnv.GetConfigMapKey(),
			}

			// Set Optional field when explicitly provided (true or false), leave nil when not specified
			if configMapAsEnv.Optional != nil {
				configMapKeySelector.Optional = configMapAsEnv.Optional
			}

			configMapEnvVar := k8score.EnvVar{
				Name: keyToEnv.GetEnvVar(),
				ValueFrom: &k8score.EnvVarSource{
					ConfigMapKeyRef: configMapKeySelector,
				},
			}

			var configMapName string
			if configMapAsEnv.ConfigMapNameParameter != nil {
				resolvedConfigMapName, err := resolveInputParameterStr(ctx, dag, pipeline, opts, mlmd,
					configMapAsEnv.ConfigMapNameParameter, inputParams)
				if err != nil {
					if errors.Is(err, ErrResolvedParameterNull) {
						continue
					}
					return fmt.Errorf("failed to resolve configmap name: %w", err)
				}
				configMapName = resolvedConfigMapName.GetStringValue()
			} else if configMapAsEnv.ConfigMapName != "" {
				configMapName = configMapAsEnv.ConfigMapName
			} else {
				return fmt.Errorf("missing either ConfigMapName or ConfigNameParameter for " +
					"configmap environment variable in executor config")
			}

			configMapEnvVar.ValueFrom.ConfigMapKeyRef.LocalObjectReference.Name = configMapName

			if setOnPod[pipelinespec.TaskConfigPassthroughType_ENV] {
				podSpec.Containers[0].Env = append(podSpec.Containers[0].Env, configMapEnvVar)
			}

			if setOnTaskConfig[pipelinespec.TaskConfigPassthroughType_ENV] {
				taskConfig.Env = append(taskConfig.Env, configMapEnvVar)
			}
		}
	}

	// Get image pull secret information
	for _, imagePullSecret := range kubernetesExecutorConfig.GetImagePullSecret() {
		var secretName string
		if imagePullSecret.SecretNameParameter != nil {
			resolvedSecretName, err := resolveInputParameterStr(ctx, dag, pipeline, opts, mlmd,
				imagePullSecret.SecretNameParameter, inputParams)
			if err != nil {
				if errors.Is(err, ErrResolvedParameterNull) {
					continue
				}
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

		if setOnTaskConfig[pipelinespec.TaskConfigPassthroughType_ENV] {
			taskConfig.Env = append(taskConfig.Env, fieldPathEnvVar)
		}

		if setOnPod[pipelinespec.TaskConfigPassthroughType_ENV] {
			podSpec.Containers[0].Env = append(podSpec.Containers[0].Env, fieldPathEnvVar)
		}
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

		// Ephemeral volumes don't apply for passthrough.
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

		// EmptyDirMounts don't apply for passthrough.
		podSpec.Volumes = append(podSpec.Volumes, emptyDirVolume)
		podSpec.Containers[0].VolumeMounts = append(podSpec.Containers[0].VolumeMounts, emptyDirVolumeMount)
	}

	// Get node affinity information
	if nodeAffinityTerms := kubernetesExecutorConfig.GetNodeAffinity(); len(nodeAffinityTerms) > 0 {
		var requiredTerms []k8score.NodeSelectorTerm
		var preferredTerms []k8score.PreferredSchedulingTerm

		for i, nodeAffinityTerm := range nodeAffinityTerms {
			if nodeAffinityTerm.GetNodeAffinityJson() == nil &&
				len(nodeAffinityTerm.GetMatchExpressions()) == 0 &&
				len(nodeAffinityTerm.GetMatchFields()) == 0 {
				glog.Warningf("NodeAffinityTerm %d is empty, skipping", i)
				continue
			}
			if nodeAffinityTerm.GetNodeAffinityJson() != nil {
				var k8sNodeAffinity json.RawMessage
				err := resolveK8sJsonParameter(ctx, opts, dag, pipeline, mlmd,
					nodeAffinityTerm.GetNodeAffinityJson(), inputParams, &k8sNodeAffinity)
				if err != nil {
					if errors.Is(err, ErrResolvedParameterNull) {
						continue
					}
					return fmt.Errorf("failed to resolve node affinity json: %w", err)
				}

				var nodeAffinity k8score.NodeAffinity
				if err := json.Unmarshal(k8sNodeAffinity, &nodeAffinity); err != nil {
					return fmt.Errorf("failed to unmarshal node affinity json: %w", err)
				}

				if nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution != nil {
					requiredTerms = append(requiredTerms, nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms...)
				}
				preferredTerms = append(preferredTerms, nodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution...)
			} else {
				nodeSelectorTerm := k8score.NodeSelectorTerm{}

				for _, expr := range nodeAffinityTerm.GetMatchExpressions() {
					nodeSelectorRequirement := k8score.NodeSelectorRequirement{
						Key:      expr.GetKey(),
						Operator: k8score.NodeSelectorOperator(expr.GetOperator()),
						Values:   expr.GetValues(),
					}
					nodeSelectorTerm.MatchExpressions = append(nodeSelectorTerm.MatchExpressions, nodeSelectorRequirement)
				}

				for _, field := range nodeAffinityTerm.GetMatchFields() {
					nodeSelectorRequirement := k8score.NodeSelectorRequirement{
						Key:      field.GetKey(),
						Operator: k8score.NodeSelectorOperator(field.GetOperator()),
						Values:   field.GetValues(),
					}
					nodeSelectorTerm.MatchFields = append(nodeSelectorTerm.MatchFields, nodeSelectorRequirement)
				}

				if nodeAffinityTerm.Weight != nil {
					preferredTerms = append(preferredTerms, k8score.PreferredSchedulingTerm{
						Weight:     *nodeAffinityTerm.Weight,
						Preference: nodeSelectorTerm,
					})
					glog.V(4).Infof("Added preferred node affinity: %+v", nodeSelectorTerm)
				} else {
					requiredTerms = append(requiredTerms, nodeSelectorTerm)
					glog.V(4).Infof("Added required node affinity: %+v", nodeSelectorTerm)
				}

			}
		}

		if len(requiredTerms) > 0 || len(preferredTerms) > 0 {
			k8sNodeAffinity := &k8score.NodeAffinity{}
			if len(requiredTerms) > 0 {
				k8sNodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = &k8score.NodeSelector{
					NodeSelectorTerms: requiredTerms,
				}
			}
			if len(preferredTerms) > 0 {
				k8sNodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution = preferredTerms
			}

			if setOnTaskConfig[pipelinespec.TaskConfigPassthroughType_KUBERNETES_AFFINITY] {
				if taskConfig.Affinity == nil {
					taskConfig.Affinity = &k8score.Affinity{}
				}

				taskConfig.Affinity.NodeAffinity = k8sNodeAffinity
			}

			if setOnPod[pipelinespec.TaskConfigPassthroughType_KUBERNETES_AFFINITY] {
				if podSpec.Affinity == nil {
					podSpec.Affinity = &k8score.Affinity{}
				}

				podSpec.Affinity.NodeAffinity = k8sNodeAffinity
			}
		}
	}

	return nil
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
