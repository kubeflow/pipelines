package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/kubeflow/pipelines/backend/src/cache/client"
	"github.com/kubeflow/pipelines/backend/src/cache/model"
	"github.com/peterhellberg/duration"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
)

const (
	ArgoWorkflowTemplateEnvKey string = "ARGO_TEMPLATE"
	ArgoCompleteLabelKey       string = "workflows.argoproj.io/completed"
	MetadataExecutionIDKey     string = "pipelines.kubeflow.org/metadata_execution_id"
	MaxCacheStalenessKey       string = "pipelines.kubeflow.org/max_cache_staleness"
)

func WatchPods(ctx context.Context, namespaceToWatch string, clientManager ClientManagerInterface) {
	k8sCore := clientManager.KubernetesCoreClient()

	for {
		listOptions := metav1.ListOptions{
			Watch:         true,
			LabelSelector: CacheIDLabelKey,
		}
		watcher, err := k8sCore.PodClient(namespaceToWatch).Watch(ctx, listOptions)

		if err != nil {
			log.Printf("%s", "Watcher error:"+err.Error())
		}

		for event := range watcher.ResultChan() {
			if event.Type == watch.Error {
				continue
			}
			pod, ok := event.Object.(*corev1.Pod)
			if !ok {
				continue
			}
			log.Printf("%s", (*pod).GetName())

			if err := cacheCompletedPod(ctx, pod, clientManager); err != nil {
				log.Printf("%s", err.Error())
			}
		}
	}
}

// cacheCompletedPod validates a watched pod's cache identity before publishing its outputs.
func cacheCompletedPod(ctx context.Context, pod *corev1.Pod, clientManager ClientManagerInterface) error {
	if !isKFPCacheEnabled(pod) || isTFXPod(pod) || isV2Pod(pod) || !isPodCompletedAndSucceeded(pod) {
		log.Printf("Pod %s is not eligible for a cache write.", pod.Name)
		return nil
	}

	if isCacheWriten(pod.Labels) {
		return nil
	}

	executionTemplate, exists := getArgoTemplate(pod)
	if !exists {
		return nil
	}
	executionKey, err := generateCacheKeyFromTemplate(executionTemplate, pod.Namespace)
	if err != nil {
		return fmt.Errorf("cannot determine cache identity for pod %s/%s: %w", pod.Namespace, pod.Name, err)
	}
	// The cache-key annotation is mutable and must not select another tenant's row.
	if pod.Annotations[ExecutionKey] != executionKey {
		return fmt.Errorf("skipping cache write for pod %s/%s: execution cache key does not match namespace and template", pod.Namespace, pod.Name)
	}

	executionOutput := pod.Annotations[ArgoWorkflowOutputs]

	executionOutputMap := make(map[string]interface{})
	executionOutputMap[ArgoWorkflowOutputs] = executionOutput
	executionOutputMap[MetadataExecutionIDKey] = pod.Labels[MetadataExecutionIDKey]
	executionOutputJSON, _ := json.Marshal(executionOutputMap)

	executionstaleness, exists := pod.Annotations[MaxCacheStalenessKey]
	var cacheStalenessInSeconds int64 = -1
	if exists {
		cacheStalenessInSeconds = stalenessToSeconds(executionstaleness)
	}

	var maximumCacheStalenessInSeconds int64 = -1
	maximumCacheStaleness, exists := os.LookupEnv("MAXIMUM_CACHE_STALENESS")
	if exists {
		log.Printf("maximumCacheStaleness: %s", maximumCacheStaleness)
		maximumCacheStalenessInSeconds = stalenessToSeconds(maximumCacheStaleness)
		log.Printf("maximumCacheStalenessInSeconds: %d", maximumCacheStalenessInSeconds)
	}
	if maximumCacheStalenessInSeconds >= 0 && cacheStalenessInSeconds > maximumCacheStalenessInSeconds {
		cacheStalenessInSeconds = maximumCacheStalenessInSeconds
	}
	log.Printf("Creating cachedb entry with cacheStalenessInSeconds: %d", cacheStalenessInSeconds)

	executionToPersist := model.ExecutionCache{
		Namespace:         pod.Namespace,
		ExecutionCacheKey: executionKey,
		ExecutionTemplate: executionTemplate,
		ExecutionOutput:   string(executionOutputJSON),
		MaxCacheStaleness: cacheStalenessInSeconds,
	}

	cacheEntryCreated, err := clientManager.CacheStore().CreateExecutionCache(&executionToPersist)
	if err != nil {
		return fmt.Errorf("unable to create cache entry: %w", err)
	}
	return patchCacheID(ctx, clientManager.KubernetesCoreClient(), pod, cacheEntryCreated.ID)
}

func isPodCompletedAndSucceeded(pod *corev1.Pod) bool {
	return pod.ObjectMeta.Labels[ArgoCompleteLabelKey] == "true" && pod.Status.Phase == corev1.PodSucceeded
}

func isCacheWriten(labels map[string]string) bool {
	cacheID := labels[CacheIDLabelKey]
	return cacheID != ""
}

func patchCacheID(ctx context.Context, k8sCore client.KubernetesCoreInterface, podToPatch *corev1.Pod, id int64) error {
	labels := podToPatch.ObjectMeta.Labels
	labels[CacheIDLabelKey] = strconv.FormatInt(id, 10)
	log.Println(id)
	var patchOps []patchOperation
	patchOps = append(patchOps, patchOperation{
		Op:    OperationTypeAdd,
		Path:  LabelPath,
		Value: labels,
	})
	patchBytes, err := json.Marshal(patchOps)
	if err != nil {
		return fmt.Errorf("Unable to patch cache_id to pod: %s", podToPatch.ObjectMeta.Name)
	}
	_, err = k8sCore.PodClient(podToPatch.Namespace).Patch(ctx, podToPatch.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		return err
	}
	log.Printf("Cache id patched.")
	return nil
}

// Convert RFC3339 Duration(Eg. "P1DT30H4S") to int64 seconds.
func stalenessToSeconds(staleness string) int64 {
	var seconds int64 = -1
	if d, err := duration.Parse(staleness); err == nil {
		seconds = int64(d / time.Second)
	}
	return seconds
}

// Get Argo workflow template from container env.
func getArgoTemplate(pod *corev1.Pod) (string, bool) {
	if len(pod.Spec.Containers) == 0 {
		return "", false
	}
	for _, env := range pod.Spec.Containers[0].Env {
		if ArgoWorkflowTemplateEnvKey == env.Name {
			return env.Value, true
		}
	}
	return "", false
}
