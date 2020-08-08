// Copyright 2020 Google LLC
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
	"strconv"
	"strings"

	"github.com/kubeflow/pipelines/backend/src/cache/client"
	"github.com/kubeflow/pipelines/backend/src/cache/model"
	"github.com/kubeflow/pipelines/backend/src/cache/storage"
	"k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	KFPCacheEnabledLabelKey   string = "pipelines.kubeflow.org/cache_enabled"
	KFPCacheEnabledLabelValue string = "true"
	KFPCachedLabelKey         string = "pipelines.kubeflow.org/reused_from_cache"
	KFPCachedLabelValue       string = "true"
	ArgoWorkflowNodeName      string = "workflows.argoproj.io/node-name"
	ArgoWorkflowTemplate      string = "workflows.argoproj.io/template"
	ExecutionKey              string = "pipelines.kubeflow.org/execution_cache_key"
	CacheIDLabelKey           string = "pipelines.kubeflow.org/cache_id"
	ArgoWorkflowOutputs       string = "workflows.argoproj.io/outputs"
	MetadataWrittenKey        string = "pipelines.kubeflow.org/metadata_written"
	AnnotationPath            string = "/metadata/annotations"
	LabelPath                 string = "/metadata/labels"
	SpecContainersPath        string = "/spec/containers"
	SpecInitContainersPath    string = "/spec/initContainers"
	TFXPodSuffix              string = "tfx/orchestration/kubeflow/container_entrypoint.py"
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

	var patches []patchOperation
	annotations := pod.ObjectMeta.Annotations
	labels := pod.ObjectMeta.Labels
	template, exists := annotations[ArgoWorkflowTemplate]
	var executionHashKey string
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
	var maxCacheStalenessInSeconds int64 = -1
	maxCacheStaleness, exists := annotations[MaxCacheStalenessKey]
	if exists {
		maxCacheStalenessInSeconds = getMaxCacheStaleness(maxCacheStaleness)
	}

	var cachedExecution *model.ExecutionCache
	cachedExecution, err = clientMgr.CacheStore().GetExecutionCache(executionHashKey, maxCacheStalenessInSeconds)
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

		dummyContainer := corev1.Container{
			Name:    "main",
			Image:   "alpine",
			Command: []string{`echo`, `"This step output is taken from cache."`},
		}
		dummyContainers := []corev1.Container{
			dummyContainer,
		}
		patches = append(patches, patchOperation{
			Op:    OperationTypeReplace,
			Path:  SpecContainersPath,
			Value: dummyContainers,
		})
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
