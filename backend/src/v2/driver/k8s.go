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

	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiV2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/client_manager"
	"github.com/kubeflow/pipelines/backend/src/v2/component"
	"github.com/kubeflow/pipelines/backend/src/v2/driver/common"
	"github.com/kubeflow/pipelines/backend/src/v2/driver/resolver"
	"github.com/kubeflow/pipelines/kubernetes_platform/go/kubernetesplatform"
	"google.golang.org/protobuf/types/known/structpb"
	k8score "k8s.io/api/core/v1"
	k8sres "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
func kubernetesPlatformOps(ctx context.Context, clientManager client_manager.ClientManagerInterface, execution *Execution, taskToCreate *apiV2beta1.PipelineTaskDetail, opts *common.Options) (err error) {
	switch opts.Container.Image {
	case "argostub/createpvc":
		err = createPVCTask(ctx, clientManager, execution, opts, taskToCreate)
		if err != nil {
			return err
		}
	case "argostub/deletepvc":
		if err = deletePVCTask(ctx, clientManager, execution, opts, taskToCreate); err != nil {
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
	opts common.Options,
	inputParams []*apiV2beta1.PipelineTaskDetail_InputOutputs_IOParameter,
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
		volumeMounts, volumes, err := makeVolumeMountPatch(opts, kubernetesExecutorConfig.GetPvcMount(), inputParams)
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
			err := resolver.ResolveK8sJSONParameter(
				opts, kubernetesExecutorConfig.GetNodeSelector().GetNodeSelectorJson(),
				inputParams, &nodeSelector)
			if err != nil {
				if errors.Is(err, resolver.ErrResolvedParameterNull) {
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
					resolvedParam, _, err := resolver.ResolveInputParameter(opts, toleration.GetTolerationJson(), inputParams)
					if err != nil {
						if errors.Is(err, resolver.ErrResolvedParameterNull) {
							continue // Skip applying the patch for this null/optional parameter
						}
						return fmt.Errorf("failed to resolve toleration: %w", err)
					}

					// TolerationJson can be either a single toleration or list of tolerations
					// the field accepts both, and in both cases the tolerations are appended
					// to the total executor pod toleration list.
					var paramJSON []byte
					isSingleToleration := resolvedParam.GetValue().GetStructValue() != nil
					isListToleration := resolvedParam.GetValue().GetListValue() != nil
					if isSingleToleration {
						structVal := resolvedParam.GetValue().GetStructValue()
						if structVal != nil && len(structVal.Fields) > 0 {
							paramJSON, err = resolvedParam.GetValue().GetStructValue().MarshalJSON()
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
						listVal := resolvedParam.GetValue().GetListValue()
						if listVal != nil && len(listVal.Values) > 0 {
							paramJSON, err = resolvedParam.GetValue().GetListValue().MarshalJSON()
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
			resolvedSecretName, err := resolver.ResolveInputParameterStr(opts, secretAsVolume.SecretNameParameter, inputParams)
			if err != nil {
				if errors.Is(err, resolver.ErrResolvedParameterNull) {
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
				resolvedSecretName, err := resolver.ResolveInputParameterStr(opts, secretAsEnv.SecretNameParameter, inputParams)
				if err != nil {
					if errors.Is(err, resolver.ErrResolvedParameterNull) {
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

			secretEnvVar.ValueFrom.SecretKeyRef.Name = secretName

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
			resolvedConfigMapName, err := resolver.ResolveInputParameterStr(opts,
				configMapAsVolume.ConfigMapNameParameter, inputParams)
			if err != nil {
				if errors.Is(err, resolver.ErrResolvedParameterNull) {
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
				resolvedConfigMapName, err := resolver.ResolveInputParameterStr(opts,
					configMapAsEnv.ConfigMapNameParameter, inputParams)
				if err != nil {
					if errors.Is(err, resolver.ErrResolvedParameterNull) {
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

			configMapEnvVar.ValueFrom.ConfigMapKeyRef.Name = configMapName

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
			resolvedSecretName, err := resolver.ResolveInputParameterStr(opts,
				imagePullSecret.SecretNameParameter, inputParams)
			if err != nil {
				if errors.Is(err, resolver.ErrResolvedParameterNull) {
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
				err := resolver.ResolveK8sJSONParameter(
					opts,
					nodeAffinityTerm.GetNodeAffinityJson(),
					inputParams,
					&k8sNodeAffinity,
				)
				if err != nil {
					if errors.Is(err, resolver.ErrResolvedParameterNull) {
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

// execution is passed by pointer so we can update TaskID for the defer function
func createPVCTask(
	ctx context.Context,
	clientManager client_manager.ClientManagerInterface,
	execution *Execution,
	opts *common.Options,
	taskToCreate *apiV2beta1.PipelineTaskDetail,
) (err error) {
	taskCreated := false

	// Ensure that we update the final task state after creation, or if we fail the procedure
	defer func() {
		if err != nil {
			taskToCreate.State = apiV2beta1.PipelineTaskDetail_FAILED
			taskToCreate.StatusMetadata = &apiV2beta1.PipelineTaskDetail_StatusMetadata{
				Message: err.Error(),
			}
		} else {
			// K8s ops drivers do not have executors, we can mark them completed at the driver stage.
			taskToCreate.State = apiV2beta1.PipelineTaskDetail_SUCCEEDED
		}
		if taskCreated {
			_, updateErr := clientManager.KFPAPIClient().UpdateTask(ctx, &apiV2beta1.UpdateTaskRequest{
				TaskId: execution.TaskID,
				Task:   taskToCreate,
			})
			if updateErr != nil {
				err = errors.Join(err, fmt.Errorf("failed to update task: %w", updateErr))
			}
		} else {
			_, createErr := clientManager.KFPAPIClient().CreateTask(ctx, &apiV2beta1.CreateTaskRequest{
				Task: taskToCreate,
			})
			if createErr != nil {
				err = errors.Join(err, fmt.Errorf("failed to create task: %w", createErr))
			}
		}
		// Do not need to propagate statuses, this will be handled in the defer for Container().
	}()

	inputs := execution.ExecutorInput.Inputs
	glog.Infof("Input parameter values: %+v", inputs.ParameterValues)

	// Required input: access_modes
	accessModeInput, ok := inputs.ParameterValues["access_modes"]
	if !ok || accessModeInput == nil {
		err = fmt.Errorf("failed to create pvc: parameter access_modes not provided")
		return err
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
	var pvcName string

	if pvcNameInput.GetStringValue() != "" && pvcNameSuffixInput.GetStringValue() != "" {
		err = fmt.Errorf("failed to create pvc: at most one of pvc_name and pvc_name_suffix can be non-empty")
		return err
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

	if taskToCreate.Outputs == nil {
		taskToCreate.Outputs = &apiV2beta1.PipelineTaskDetail_InputOutputs{
			Parameters: make([]*apiV2beta1.PipelineTaskDetail_InputOutputs_IOParameter, 0),
		}
	}
	if taskToCreate.Outputs.Parameters == nil {
		taskToCreate.Outputs.Parameters = make([]*apiV2beta1.PipelineTaskDetail_InputOutputs_IOParameter, 0)
	}
	taskToCreate.Outputs.Parameters = append(
		taskToCreate.Outputs.Parameters,
		&apiV2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
			Value:        execution.ExecutorInput.Inputs.ParameterValues[pvcName],
			ParameterKey: "name", // create-pvc output parameter is always "name"
			Type:         apiV2beta1.IOType_OUTPUT,
			Producer: &apiV2beta1.IOProducer{
				TaskName: opts.Task.GetTaskInfo().GetName(),
			},
		})

	// Size is required input.
	volumeSizeInput, ok := inputs.ParameterValues["size"]
	if !ok || volumeSizeInput == nil {
		err = fmt.Errorf("failed to create pvc: parameter volumeSize not provided")
		return err
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

	// Create Initial Task. We will update the status later if
	// anything fails, or the task successfully completes.
	taskToCreate.State = apiV2beta1.PipelineTaskDetail_RUNNING
	task, err := clientManager.KFPAPIClient().CreateTask(ctx, &apiV2beta1.CreateTaskRequest{
		Task: taskToCreate,
	})
	if err != nil {
		err = fmt.Errorf("failed to create task: %w", err)
		return err
	}
	glog.Infof("Created Task: %s", task.TaskId)
	taskCreated = true
	execution.TaskID = task.TaskId
	if !execution.WillTrigger() {
		taskToCreate.State = apiV2beta1.PipelineTaskDetail_SKIPPED
		glog.Infof("Condition not met, skipping task %s", task.TaskId)
		return nil
	}

	// Use cache and skip pvc creation if all conditions met:
	// (1) Cache is enabled globally
	// (2) Cache is enabled for the task
	// (3) We had a cache hit for this Task
	fingerPrint, cachedTask, err := getFingerPrintsAndID(ctx, execution, clientManager.KFPAPIClient(), opts, nil)
	if err != nil {
		return err
	}
	taskToCreate.CacheFingerprint = fingerPrint
	execution.Cached = util.BoolPointer(false)
	if !opts.CacheDisabled && opts.Task.GetCachingOptions().GetEnableCache() && cachedTask != nil {
		taskToCreate.State = apiV2beta1.PipelineTaskDetail_CACHED
		taskToCreate.Outputs = cachedTask.Outputs
		*execution.Cached = true
		return nil
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
	createdPVC, err := clientManager.K8sClient().CoreV1().PersistentVolumeClaims(opts.Namespace).Create(context.Background(), pvc, metav1.CreateOptions{})
	if err != nil {
		err = fmt.Errorf("failed to create pvc: %w", err)
		return err
	}
	glog.Infof("Created PVC %s\n", createdPVC.Name)
	taskToCreate.State = apiV2beta1.PipelineTaskDetail_SUCCEEDED
	return nil
}

func deletePVCTask(
	ctx context.Context,
	clientManager client_manager.ClientManagerInterface,
	execution *Execution,
	opts *common.Options,
	taskToCreate *apiV2beta1.PipelineTaskDetail,
) (err error) {
	taskCreated := false

	// Ensure that we update the final task state after creation, or if we fail the procedure
	defer func() {
		if err != nil {
			taskToCreate.State = apiV2beta1.PipelineTaskDetail_FAILED
			taskToCreate.StatusMetadata = &apiV2beta1.PipelineTaskDetail_StatusMetadata{
				Message: err.Error(),
			}
		} else {
			// K8s ops drivers do not have executors, we can mark them completed at the driver stage.
			taskToCreate.State = apiV2beta1.PipelineTaskDetail_SUCCEEDED
		}
		if taskCreated {
			_, updateErr := clientManager.KFPAPIClient().UpdateTask(ctx, &apiV2beta1.UpdateTaskRequest{
				TaskId: execution.TaskID,
				Task:   taskToCreate,
			})
			if updateErr != nil {
				err = errors.Join(err, fmt.Errorf("failed to update task: %w", updateErr))
			}
		} else {
			_, createErr := clientManager.KFPAPIClient().CreateTask(ctx, &apiV2beta1.CreateTaskRequest{
				Task: taskToCreate,
			})
			if createErr != nil {
				err = errors.Join(err, fmt.Errorf("failed to create task: %w", createErr))
			}
		}
		// Do not need to propagate statuses, this will be handled in the defer for Container().
	}()

	inputs := execution.ExecutorInput.Inputs
	glog.Infof("Input parameter values: %+v", inputs.ParameterValues)

	// Required input: pvc_name
	pvcNameInput, ok := inputs.ParameterValues["pvc_name"]
	if !ok || pvcNameInput == nil {
		err = fmt.Errorf("failed to delete pvc: required parameter pvc_name not provided")
		return err
	}
	pvcName := pvcNameInput.GetStringValue()

	// Create Initial Task. We will update the status later if
	// anything fails, or the task successfully completes.
	taskToCreate.State = apiV2beta1.PipelineTaskDetail_RUNNING
	task, err := clientManager.KFPAPIClient().CreateTask(ctx, &apiV2beta1.CreateTaskRequest{
		Task: taskToCreate,
	})
	if err != nil {
		err = fmt.Errorf("failed to create task: %w", err)
		return err
	}
	glog.Infof("Created Task: %s", task.TaskId)
	taskCreated = true
	execution.TaskID = task.TaskId
	if !execution.WillTrigger() {
		taskToCreate.State = apiV2beta1.PipelineTaskDetail_SKIPPED
		glog.Infof("Condition not met, skipping task %s", task.TaskId)
		return nil
	}

	// Use cache and skip pvc creation if all conditions met:
	// (1) Cache is enabled globally
	// (2) Cache is enabled for the task
	// (3) We had a cache hit for this Task
	fingerPrint, cachedTask, err := getFingerPrintsAndID(ctx, execution, clientManager.KFPAPIClient(), opts, nil)
	if err != nil {
		return err
	}
	taskToCreate.CacheFingerprint = fingerPrint
	execution.Cached = util.BoolPointer(false)
	if !opts.CacheDisabled && opts.Task.GetCachingOptions().GetEnableCache() && cachedTask != nil {
		taskToCreate.State = apiV2beta1.PipelineTaskDetail_CACHED
		taskToCreate.Outputs = cachedTask.Outputs
		*execution.Cached = true
		return nil
	}

	// Get the PVC you want to delete, verify that it exists.
	_, err = clientManager.K8sClient().CoreV1().PersistentVolumeClaims(opts.Namespace).Get(context.TODO(), pvcName, metav1.GetOptions{})
	if err != nil {
		err = fmt.Errorf("failed to delete pvc %s: cannot find pvc: %v", pvcName, err)
		return err
	}

	// Delete the PVC.
	err = clientManager.K8sClient().CoreV1().PersistentVolumeClaims(opts.Namespace).Delete(context.TODO(), pvcName, metav1.DeleteOptions{})
	if err != nil {
		err = fmt.Errorf("failed to delete pvc %s: %v", pvcName, err)
		return err
	}

	return nil
}

func makeVolumeMountPatch(
	opts common.Options,
	pvcMounts []*kubernetesplatform.PvcMount,
	inputParams []*apiV2beta1.PipelineTaskDetail_InputOutputs_IOParameter,
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
				pvcNameParameter = common.InputParamConstant(pvcMount.GetConstant())
			} else if pvcMount.GetTaskOutputParameter() != nil {
				pvcNameParameter = common.InputParamTaskOutput(
					pvcMount.GetTaskOutputParameter().GetProducerTask(),
					pvcMount.GetTaskOutputParameter().GetOutputParameterKey(),
				)
			} else if pvcMount.GetComponentInputParameter() != "" {
				pvcNameParameter = common.InputParamComponent(pvcMount.GetComponentInputParameter())
			} else {
				return nil, nil, fmt.Errorf("failed to make podSpecPatch: volume mount: volume name not provided")
			}
		}

		resolvedPvcName, err := resolver.ResolveInputParameterStr(opts, pvcNameParameter, inputParams)
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
