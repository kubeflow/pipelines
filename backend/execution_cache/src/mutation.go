package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"

	"k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ArgoWorkflowTemplate string = "workflows.argoproj.io/template"
	ExecutionKey         string = "pipelines.kubeflow.org/execution_cache_key"
	AnnotationPath       string = "/metadata/annotations"
)

const (
	Add OperationType = "add"
)

var (
	podResource = metav1.GroupVersionResource{Version: "v1", Resource: "pods"}
)

// applyPodOutput will check whether the execution has already been run before from MLMD and apply the output into pod.metadata.output
func applyPodOutput(req *v1beta1.AdmissionRequest) ([]patchOperation, error) {
	// This handler should only get called on Pod objects as per the MutatingWebhookConfiguration in the YAML file.
	// However, if (for whatever reason) this gets invoked on an object of a different kind, issue a log message but
	// let the object request pass through otherwise.
	if req.Resource != podResource {
		log.Printf("expect resource to be %q", podResource)
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
	// Generate the executionHashKey based on pod.metadata.annotations.workflows.argoproj.io/template
	if exists {
		hash := sha256.New()
		hash.Write([]byte(template))
		md := hash.Sum(nil)
		executionHashKey = hex.EncodeToString(md)
	}

	annotations[ExecutionKey] = executionHashKey
	// Add executionKey to pod.metadata.annotations
	if len(executionHashKey) != 0 {
		patches = append(patches, patchOperation{
			Op:    Add,
			Path:  AnnotationPath,
			Value: annotations,
		})
	}

	return patches, nil
}
