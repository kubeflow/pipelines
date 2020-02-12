package main

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	fakePod = corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"workflows.argoproj.io/node-name": "test_name",
				"workflows.argoproj.io/template":  "test_template",
			},
		},
	}
	fakeAdmissionRequest = v1beta1.AdmissionRequest{
		UID: "test-12345",
		Kind: metav1.GroupVersionKind{
			Group:   "group",
			Version: "v1",
			Kind:    "k8s",
		},
		Resource: metav1.GroupVersionResource{
			Version:  "v1",
			Resource: "pods",
		},
		SubResource: "subresource",
		Name:        "test",
		Namespace:   "default",
		Operation:   "test",
		Object: runtime.RawExtension{
			Raw: EncodePod(&fakePod),
		},
	}
)

func GetFakeAdmissionRequestFromPod(pod *corev1.Pod) *v1beta1.AdmissionRequest {
	fakeRequest := fakeAdmissionRequest
	fakeRequest.Object.Raw = EncodePod(pod)
	return &fakeRequest
}

func EncodePod(pod *corev1.Pod) []byte {
	reqBodyBytes := new(bytes.Buffer)
	json.NewEncoder(reqBodyBytes).Encode(*pod)

	return reqBodyBytes.Bytes()
}

func TestApplyPodOutputWithErrorPodResource(t *testing.T) {
	mockAdmissionRequest := &v1beta1.AdmissionRequest{
		Resource: metav1.GroupVersionResource{
			Version: "wrong", Resource: "wrong",
		},
	}
	patchOperations, err := applyPodOutput(mockAdmissionRequest)
	assert.Nil(t, patchOperations)
	assert.Nil(t, err)
}

func TestApplyPodOutputWithDecodeError(t *testing.T) {
	invalidAdmissionRequest := fakeAdmissionRequest
	invalidAdmissionRequest.Object.Raw = []byte{5, 5}
	patchOperation, err := applyPodOutput(&invalidAdmissionRequest)
	assert.Nil(t, patchOperation)
	assert.Contains(t, err.Error(), "could not deserialize pod object")
}

func TestApplyPodOutputWithNoArgoPod(t *testing.T) {
	notArgoPod := fakePod
	delete(notArgoPod.GetAnnotations(), ArgoWorkflowNodeName)
	patchOperation, err := applyPodOutput(GetFakeAdmissionRequestFromPod(&notArgoPod))
	assert.Nil(t, patchOperation)
	assert.Nil(t, err)
}

func TestApplyPodOutput(t *testing.T) {
	patchOperation, err := applyPodOutput(&fakeAdmissionRequest)
	assert.Nil(t, err)
	require.NotNil(t, patchOperation)
	require.Equal(t, 1, len(patchOperation))
	require.Equal(t, patchOperation[0].Op, OperationType("add"))
}
