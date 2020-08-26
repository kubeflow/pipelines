package server

import (
	"encoding/json"
	"fmt"
	"log"
	"reflect"
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
	ArgoCompleteLabelKey   string = "workflows.argoproj.io/completed"
	MetadataExecutionIDKey string = "pipelines.kubeflow.org/metadata_execution_id"
	MaxCacheStalenessKey   string = "pipelines.kubeflow.org/max_cache_staleness"
)

func WatchPods(namespaceToWatch string, clientManager ClientManagerInterface) {
	k8sCore := clientManager.KubernetesCoreClient()

	for {
		listOptions := metav1.ListOptions{
			Watch:         true,
			LabelSelector: CacheIDLabelKey,
		}
		watcher, err := k8sCore.PodClient(namespaceToWatch).Watch(listOptions)

		if err != nil {
			log.Printf("Watcher error:" + err.Error())
		}

		for event := range watcher.ResultChan() {
			pod := reflect.ValueOf(event.Object).Interface().(*corev1.Pod)
			if event.Type == watch.Error {
				continue
			}
			log.Printf((*pod).GetName())

			if !isPodCompletedAndSucceeded(pod) {
				log.Printf("Pod %s is not completed or not in successful status.", pod.ObjectMeta.Name)
				continue
			}

			if isCacheWriten(pod.ObjectMeta.Labels) {
				continue
			}

			executionKey, exists := pod.ObjectMeta.Annotations[ExecutionKey]
			if !exists {
				continue
			}

			executionOutput, exists := pod.ObjectMeta.Annotations[ArgoWorkflowOutputs]

			executionOutputMap := make(map[string]interface{})
			executionOutputMap[ArgoWorkflowOutputs] = executionOutput
			executionOutputMap[MetadataExecutionIDKey] = pod.ObjectMeta.Labels[MetadataExecutionIDKey]
			executionOutputJSON, _ := json.Marshal(executionOutputMap)

			executionMaxCacheStaleness, exists := pod.ObjectMeta.Annotations[MaxCacheStalenessKey]
			var maxCacheStalenessInSeconds int64 = -1
			if exists {
				maxCacheStalenessInSeconds = getMaxCacheStaleness(executionMaxCacheStaleness)
			}

			executionTemplate := pod.ObjectMeta.Annotations[ArgoWorkflowTemplate]
			executionToPersist := model.ExecutionCache{
				ExecutionCacheKey: executionKey,
				ExecutionTemplate: executionTemplate,
				ExecutionOutput:   string(executionOutputJSON),
				MaxCacheStaleness: maxCacheStalenessInSeconds,
			}

			cacheEntryCreated, err := clientManager.CacheStore().CreateExecutionCache(&executionToPersist)
			if err != nil {
				log.Println("Unable to create cache entry.")
				continue
			}
			err = patchCacheID(k8sCore, pod, namespaceToWatch, cacheEntryCreated.ID)
			if err != nil {
				log.Printf(err.Error())
			}
		}
	}
}

func isPodCompletedAndSucceeded(pod *corev1.Pod) bool {
	return pod.ObjectMeta.Labels[ArgoCompleteLabelKey] == "true" && pod.Status.Phase == corev1.PodSucceeded
}

func isCacheWriten(labels map[string]string) bool {
	cacheID := labels[CacheIDLabelKey]
	return cacheID != ""
}

func patchCacheID(k8sCore client.KubernetesCoreInterface, podToPatch *corev1.Pod, namespaceToWatch string, id int64) error {
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
	_, err = k8sCore.PodClient(namespaceToWatch).Patch(podToPatch.ObjectMeta.Name, types.JSONPatchType, patchBytes)
	if err != nil {
		return err
	}
	log.Printf("Cache id patched.")
	return nil
}

// Convert RFC3339 Duration(Eg. "P1DT30H4S") to int64 seconds.
func getMaxCacheStaleness(maxCacheStaleness string) int64 {
	var seconds int64 = -1
	if d, err := duration.Parse(maxCacheStaleness); err == nil {
		seconds = int64(d / time.Second)
	}
	return seconds
}
