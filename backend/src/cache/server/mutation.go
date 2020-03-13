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
	"fmt"
	"log"
	"strings"

	"github.com/kubeflow/pipelines/backend/src/cache/model"
	"github.com/kubeflow/pipelines/backend/src/cache/storage"
	"k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	KFPAnnotation        string = "pipelines.kubeflow.org"
	ArgoWorkflowNodeName string = "workflows.argoproj.io/node-name"
	ArgoWorkflowTemplate string = "workflows.argoproj.io/template"
	ExecutionKey         string = "pipelines.kubeflow.org/execution_cache_key"
	CacheIDLabelKey      string = "pipelines.kubeflow.org/cache_id"
	AnnotationPath       string = "/metadata/annotations"
	LabelPath            string = "/metadata/labels"
	ExecutionOutputs     string = "workflows.argoproj.io/outputs"
	TFXPodSuffix         string = "tfx/orchestration/kubeflow/container_entrypoint.py"
)

var (
	podResource = metav1.GroupVersionResource{Version: "v1", Resource: "pods"}
)

type ClientManagerInterface interface {
	CacheStore() storage.ExecutionCacheStoreInterface
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
	if !isValidPod(&pod) {
		log.Printf("This pod %s is not a valid pod.", pod.ObjectMeta.Name)
		return nil, nil
	}

	var patches []patchOperation
	annotations := pod.ObjectMeta.Annotations
	template, exists := annotations[ArgoWorkflowTemplate]
	var executionHashKey string
	if !exists {
		return patches, nil
	}

	// Generate the executionHashKey based on pod.metadata.annotations.workflows.argoproj.io/template
	hash := sha256.New()
	hash.Write([]byte(template))
	md := hash.Sum(nil)
	executionHashKey = hex.EncodeToString(md)

	annotations[ExecutionKey] = executionHashKey

	testExecution := model.ExecutionCache{
		ExecutionCacheKey: "123abceeeeeeeeeeee",
		ExecutionTemplate: "testTemplate",
		ExecutionOutput:   "testoutput",
		MaxCacheStaleness: -1,
		StartedAtInSec:    -1,
		EndedAtInSec:      -1,
	}
	log.Println("Origin: " + testExecution.ExecutionTemplate)
	var executionCreated *model.ExecutionCache
	executionCreated, err := clientMgr.CacheStore().CreateExecutionCache(&testExecution)
	if err != nil {
		log.Println(err.Error())
	}
	log.Println("Created: " + (*executionCreated).ExecutionTemplate)
	var getExectuion *model.ExecutionCache
	getExectuion, err = clientMgr.CacheStore().GetExecutionCache("123abceeeeeeeeeeee")
	if err != nil {
		log.Printf(err.Error())
	}
	log.Println(getExectuion)

	cachedExecution, err := clientMgr.CacheStore().GetExecutionCache(executionHashKey)
	if err != nil {
		log.Println(err.Error())
	}
	if cachedExecution != nil {
		log.Println("Cached output: " + cachedExecution.ExecutionOutput)
	}

	// Add executionKey to pod.metadata.annotations
	patches = append(patches, patchOperation{
		Op:    OperationTypeAdd,
		Path:  AnnotationPath,
		Value: annotations,
	})

	// Add cache_id label
	labels := pod.ObjectMeta.Labels
	_, exists = labels[CacheIDLabelKey]
	if !exists {
		labels[CacheIDLabelKey] = ""
		patches = append(patches, patchOperation{
			Op:    OperationTypeAdd,
			Path:  LabelPath,
			Value: labels,
		})
	}

	return patches, nil
}

func isValidPod(pod *corev1.Pod) bool {
	annotations := pod.ObjectMeta.Annotations
	if annotations == nil || len(annotations) == 0 {
		log.Printf("The annotation of this pod %s is empty.", pod.ObjectMeta.Name)
		return false
	}
	if !isKFPArgoPod(&annotations, pod.ObjectMeta.Name) {
		log.Printf("This pod %s is not created by KFP.", pod.ObjectMeta.Name)
		return false
	}
	containers := pod.Spec.Containers
	if containers != nil && len(containers) != 0 && isTFXPod(&containers) {
		log.Printf("This pod %s is created by TFX pipelines.", pod.ObjectMeta.Name)
		return false
	}
	return true
}

func isKFPArgoPod(annotations *map[string]string, podName string) bool {
	// is argo pod or not
	if _, exists := (*annotations)[ArgoWorkflowNodeName]; !exists {
		log.Printf("This pod %s is not created by Argo.", podName)
		return false
	}
	// is KFP pod or not
	for k := range *annotations {
		if strings.Contains(k, KFPAnnotation) {
			return true
		}
	}
	return false
}

func isTFXPod(containers *[]corev1.Container) bool {
	var mainContainers []corev1.Container
	for _, c := range *containers {
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
