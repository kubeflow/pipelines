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

	"github.com/kubeflow/pipelines/backend/src/cache/model"
	"github.com/kubeflow/pipelines/backend/src/cache/storage"
	"k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ArgoWorkflowTemplate string = "workflows.argoproj.io/template"
	ExecutionKey         string = "pipelines.kubeflow.org/execution_cache_key"
	AnnotationPath       string = "/metadata/annotations"
	ExecutionOutputs     string = "workflows.argoproj.io/outputs"
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

	// Check cache existance
	cachedExecution, err := clientMgr.CacheStore().GetExecutionCache(executionHashKey)
	if err != nil {
		log.Println(err.Error())
		// return nil, nil
	}
	var cachedOutput string
	if cachedExecution != nil {
		cachedOutput = cachedExecution.GetExecutionOutput()
		if cachedOutput != "" {
			annotations[ExecutionOutputs] = cachedOutput
			pod.Spec.Containers = []corev1.Container{}
		}
		log.Println(cachedExecution.ExecutionTemplate)
	}
	log.Println("pod outputs: " + annotations[ExecutionOutputs])
	// executionOutputs, _ := annotations[ExecutionOutputs]
	if cachedExecution == nil {
		executionToCache := model.ExecutionCache{
			ExecutionCacheKey: executionHashKey,
			ExecutionTemplate: template,
			MaxCacheStaleness: -1,
		}
		clientMgr.CacheStore().DeleteExecutionCache(executionHashKey)
		_, err := clientMgr.CacheStore().CreateExecutionCache(&executionToCache)
		if err != nil {
			log.Println(err.Error())
			// return nil, nil
		}
		log.Println("A new cache entry was created.")
	}

	// Add executionKey to pod.metadata.annotations
	patches = append(patches, patchOperation{
		Op:    OperationTypeAdd,
		Path:  AnnotationPath,
		Value: annotations,
	})

	return patches, nil
}
