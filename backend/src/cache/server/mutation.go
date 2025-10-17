// Copyright 2020 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/kubeflow/pipelines/backend/src/cache/client"
	"github.com/kubeflow/pipelines/backend/src/cache/model"
	"github.com/kubeflow/pipelines/backend/src/cache/storage"
	"k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	KFPCacheEnabledLabelKey    string = "pipelines.kubeflow.org/cache_enabled"
	KFPCacheEnabledLabelValue  string = "true"
	KFPCachedLabelKey          string = "pipelines.kubeflow.org/reused_from_cache"
	KFPCachedLabelValue        string = "true"
	ArgoWorkflowNodeName       string = "workflows.argoproj.io/node-name"
	ExecutionKey               string = "pipelines.kubeflow.org/execution_cache_key"
	CacheIDLabelKey            string = "pipelines.kubeflow.org/cache_id"
	ArgoWorkflowOutputs        string = "workflows.argoproj.io/outputs"
	MetadataWrittenKey         string = "pipelines.kubeflow.org/metadata_written"
	AnnotationPath             string = "/metadata/annotations"
	LabelPath                  string = "/metadata/labels"
	SpecContainersPath         string = "/spec/containers"
	SpecInitContainersPath     string = "/spec/initContainers"
	TFXPodSuffix               string = "tfx/orchestration/kubeflow/container_entrypoint.py"
	SdkTypeLabel               string = "pipelines.kubeflow.org/pipeline-sdk-type"
	TfxSdkTypeLabel            string = "tfx"
	V2ComponentAnnotationKey   string = "pipelines.kubeflow.org/v2_component"
	V2ComponentAnnotationValue string = "true"
)

var (
	podResource = metav1.GroupVersionResource{Version: "v1", Resource: "pods"}
)

type ClientManagerInterface interface {
	CacheStore() storage.ExecutionCacheStoreInterface
	KubernetesCoreClient() client.KubernetesCoreInterface
}

// MutatePodIfCached will check whether the execution has already been run before from MLMD and apply the output into pod.metadata.output
func MutatePodIfCached(req *v1beta1.AdmissionRequest, clientMgr ClientManagerInterface) ([]patchOperation, error) {
	// This handler should only get called on Pod objects as per the MutatingWebhookConfiguration in the YAML file.
	// However, if (for whatever reason) this gets invoked on an object of a different kind, issue a log message but
	// let the object request pass through otherwise.
	if req.Resource != podResource {
		log.Printf("Expect resource to be %q, but found %q", podResource, req.Resource)
		return nil, nil
	}

	// Parse the Pod object.
	raw := req.Object.Raw
	pod := corev1.Pod{}
	if _, _, err := universalDeserializer.Decode(raw, nil, &pod); err != nil {
		return nil, fmt.Errorf("could not deserialize pod object: %v", err)
	}

	// Pod filtering to only cache KFP argo pods except TFX pods
	// TODO: Switch to objectSelector once Kubernetes 1.15 hits the GKE stable channel. See
	// https://github.com/kubernetes/kubernetes/pull/78505
	// https://cloud.google.com/kubernetes-engine/docs/release-notes-stable
	if !isKFPCacheEnabled(&pod) {
		log.Printf("This pod %s does not enable cache.", pod.ObjectMeta.Name)
		return nil, nil
	}

	if isTFXPod(&pod) {
		log.Printf("This pod %s is created by tfx pipelines.", pod.ObjectMeta.Name)
		return nil, nil
	}

	if isV2Pod(&pod) {
		// KFP v2 handles caching by its driver.
		log.Printf("This pod %s is created by KFP v2 pipelines.", pod.ObjectMeta.Name)
		return nil, nil
	}

	var patches []patchOperation
	annotations := pod.ObjectMeta.Annotations
	labels := pod.ObjectMeta.Labels

	template, exists := getArgoTemplate(&pod)
	if !exists {
		return patches, nil
	}

	// Generate the executionHashKey based on pod.metadata.annotations.workflows.argoproj.io/template
	executionHashKey, err := generateCacheKeyFromTemplate(template)
	log.Println(executionHashKey)
	if err != nil {
		log.Printf("Unable to generate cache key for pod %s : %s", pod.ObjectMeta.Name, err.Error())
		return patches, nil
	}

	annotations[ExecutionKey] = executionHashKey
	labels[CacheIDLabelKey] = ""

	var cacheStalenessInSeconds int64 = -1
	var userCacheStalenessInSeconds int64 = -1
	var defaultCacheStalenessInSeconds int64 = -1
	var maximumCacheStalenessInSeconds int64 = -1

	userCacheStaleness, exists := annotations[MaxCacheStalenessKey]
	if exists {
		userCacheStalenessInSeconds = stalenessToSeconds(userCacheStaleness)
	}
	defaultCacheStaleness, exists := os.LookupEnv("DEFAULT_CACHE_STALENESS")
	if exists {
		defaultCacheStalenessInSeconds = stalenessToSeconds(defaultCacheStaleness)
	}
	maximumCacheStaleness, exists := os.LookupEnv("MAXIMUM_CACHE_STALENESS")
	if exists {
		maximumCacheStalenessInSeconds = stalenessToSeconds(maximumCacheStaleness)
	}

	if userCacheStalenessInSeconds < 0 {
		cacheStalenessInSeconds = defaultCacheStalenessInSeconds
	} else {
		cacheStalenessInSeconds = userCacheStalenessInSeconds
	}
	if maximumCacheStalenessInSeconds >= 0 && cacheStalenessInSeconds > maximumCacheStalenessInSeconds {
		cacheStalenessInSeconds = maximumCacheStalenessInSeconds
	}
	log.Printf("cacheStalenessInSeconds: %d", cacheStalenessInSeconds)

	var cachedExecution *model.ExecutionCache
	cachedExecution, err = clientMgr.CacheStore().GetExecutionCache(executionHashKey, cacheStalenessInSeconds, maximumCacheStalenessInSeconds)
	if err != nil {
		log.Println(err.Error())
	}
	// Found cached execution, add cached output and cache_id and replace container images.
	if cachedExecution != nil {
		log.Println("Cached output: " + cachedExecution.ExecutionOutput)

		annotations[ArgoWorkflowOutputs] = getValueFromSerializedMap(cachedExecution.ExecutionOutput, ArgoWorkflowOutputs)
		labels[CacheIDLabelKey] = strconv.FormatInt(cachedExecution.ID, 10)
		labels[KFPCachedLabelKey] = KFPCachedLabelValue // This label indicates the pod is taken from cache.

		// These labels cache results for metadata-writer.
		labels[MetadataExecutionIDKey] = getValueFromSerializedMap(cachedExecution.ExecutionOutput, MetadataExecutionIDKey)
		labels[MetadataWrittenKey] = "true"

		// Image selected from Google Container Register(gcr) for it small size, gcr since there
		// is not image pull rate limit. For more info see issue: https://github.com/kubeflow/pipelines/issues/4099
		image := "ghcr.io/containerd/busybox"
		if v, ok := os.LookupEnv("CACHE_IMAGE"); ok {
			image = v
		}
		dummyContainer := corev1.Container{
			Name:    "main",
			Image:   image,
			Command: []string{`echo`, `"This step output is taken from cache."`},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("0.01"),
					corev1.ResourceMemory: resource.MustParse("16Mi"),
				},
			},
		}
		dummyContainers := []corev1.Container{
			dummyContainer,
		}
		patches = append(patches, patchOperation{
			Op:    OperationTypeReplace,
			Path:  SpecContainersPath,
			Value: dummyContainers,
		})
		node_restrictions, err := getEnvBool("CACHE_NODE_RESTRICTIONS")
		if err != nil {
			return nil, err
		}
		if !node_restrictions {
			if pod.Spec.Affinity != nil {
				patches = append(patches, patchOperation{
					Op:   OperationTypeRemove,
					Path: "/spec/affinity",
				})
			}
			if pod.Spec.NodeSelector != nil {
				patches = append(patches, patchOperation{
					Op:   OperationTypeRemove,
					Path: "/spec/nodeSelector",
				})
			}
		}
		if pod.Spec.InitContainers != nil || len(pod.Spec.InitContainers) != 0 {
			patches = append(patches, patchOperation{
				Op:   OperationTypeRemove,
				Path: SpecInitContainersPath,
			})
		}
	}

	// Add executionKey to pod.metadata.annotations
	patches = append(patches, patchOperation{
		Op:    OperationTypeAdd,
		Path:  AnnotationPath,
		Value: annotations,
	})

	// Add cache_id label key
	patches = append(patches, patchOperation{
		Op:    OperationTypeAdd,
		Path:  LabelPath,
		Value: labels,
	})

	return patches, nil
}

// intersectStructureWithSkeleton recursively intersects two maps
// nil values in the skeleton map mean that the whole value (which can also be a map) should be kept.
func intersectStructureWithSkeleton(src map[string]interface{}, skeleton map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for key, skeletonValue := range skeleton {
		if value, ok := src[key]; ok {
			if skeletonValue == nil {
				result[key] = value
			} else {
				result[key] = intersectStructureWithSkeleton(value.(map[string]interface{}), skeletonValue.(map[string]interface{}))
			}
		}
	}
	return result
}

func generateCacheKeyFromTemplate(template string) (string, error) {
	var templateMap map[string]interface{}
	b := []byte(template)
	err := json.Unmarshal(b, &templateMap)
	if err != nil {
		return "", err
	}

	// Selectively copying parts of the template that should affect the cache
	templateSkeleton := map[string]interface{}{
		"container": map[string]interface{}{
			"image":        nil,
			"command":      nil,
			"args":         nil,
			"env":          nil,
			"volumeMounts": nil,
		},
		"inputs":         nil,
		"outputs":        nil, // Output artifact/parameter names and paths are important and need to be considered. We can include the whole section since Argo does not seem to put the artifact s3 specifications here, leaving them in archiveLocation.
		"volumes":        nil,
		"initContainers": nil,
		"sidecars":       nil,
	}
	cacheKeyMap := intersectStructureWithSkeleton(templateMap, templateSkeleton)

	b, err = json.Marshal(cacheKeyMap)
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	hash.Write(b)
	md := hash.Sum(nil)
	executionHashKey := hex.EncodeToString(md)

	return executionHashKey, nil
}

func getValueFromSerializedMap(serializedMap string, key string) string {
	var outputMap map[string]interface{}
	b := []byte(serializedMap)
	err := json.Unmarshal(b, &outputMap)
	if err != nil {
		return ""
	}

	value, exist := outputMap[key].(string)
	if !exist || value == "" {
		return ""
	}
	return value
}

func isKFPCacheEnabled(pod *corev1.Pod) bool {
	cacheEnabled, exists := pod.ObjectMeta.Labels[KFPCacheEnabledLabelKey]
	if !exists {
		log.Printf("This pod %s is not created by KFP.", pod.ObjectMeta.Name)
		return false
	}
	return cacheEnabled == KFPCacheEnabledLabelValue
}

func isTFXPod(pod *corev1.Pod) bool {
	// The label defaults to 'tfx', but is overridable.
	// Official tfx templates override the value to 'tfx-template', so
	// we loosely match the word 'tfx'.
	if strings.Contains(pod.Labels[SdkTypeLabel], TfxSdkTypeLabel) {
		return true
	}
	containers := pod.Spec.Containers
	if containers == nil || len(containers) == 0 {
		log.Printf("This pod container does not exist.")
		return true
	}
	var mainContainers []corev1.Container
	for _, c := range containers {
		if c.Name != "" && c.Name == "main" {
			mainContainers = append(mainContainers, c)
		}
	}
	if len(mainContainers) != 1 {
		return false
	}
	mainContainer := mainContainers[0]
	return len(mainContainer.Command) != 0 && strings.HasSuffix(mainContainer.Command[len(mainContainer.Command)-1], TFXPodSuffix)
}

func isV2Pod(pod *corev1.Pod) bool {
	return pod.Annotations[V2ComponentAnnotationKey] == V2ComponentAnnotationValue
}

func getEnvBool(key string) (bool, error) {
	v, ok := os.LookupEnv("CACHE_NODE_RESTRICTIONS")
	if !ok {
		return false, nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, err
	}
	return b, nil
}
