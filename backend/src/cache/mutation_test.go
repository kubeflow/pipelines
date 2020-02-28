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
	fakePod = &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				ArgoWorkflowNodeName: "test_node",
				ArgoWorkflowTemplate: "test_template",
				KFPAnnotation:        "test_kfp",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				corev1.Container{
					Name:    "main",
					Image:   "test_image",
					Command: []string{"python"},
				},
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
			Raw: EncodePod(fakePod),
		},
	}
)

func EncodePod(pod *corev1.Pod) []byte {
	reqBodyBytes := new(bytes.Buffer)
	json.NewEncoder(reqBodyBytes).Encode(*pod)

	return reqBodyBytes.Bytes()
}

func GetFakeRequestFromPod(pod *corev1.Pod) *v1beta1.AdmissionRequest {
	fakeRequest := fakeAdmissionRequest
	fakeRequest.Object.Raw = EncodePod(pod)
	return &fakeRequest
}

func TestMutatePodIfCachedWithErrorPodResource(t *testing.T) {
	mockAdmissionRequest := &v1beta1.AdmissionRequest{
		Resource: metav1.GroupVersionResource{
			Version: "wrong", Resource: "wrong",
		},
	}
	patchOperations, err := mutatePodIfCached(mockAdmissionRequest)
	assert.Nil(t, patchOperations)
	assert.Nil(t, err)
}

func TestMutatePodIfCachedWithDecodeError(t *testing.T) {
	invalidAdmissionRequest := fakeAdmissionRequest
	invalidAdmissionRequest.Object.Raw = []byte{5, 5}
	patchOperation, err := mutatePodIfCached(&invalidAdmissionRequest)
	assert.Nil(t, patchOperation)
	assert.Contains(t, err.Error(), "could not deserialize pod object")
}

func TestMutatePodIfCachedWithNonKFPPod(t *testing.T) {
	nonKFPPod := *fakePod
	delete(nonKFPPod.Annotations, KFPAnnotation)
	patchOperation, err := mutatePodIfCached(GetFakeRequestFromPod(&nonKFPPod))
	assert.Nil(t, patchOperation)
	assert.Nil(t, err)
}

func TestMutatePodIfCachedWithNonArgoPod(t *testing.T) {
	nonArgoPod := *fakePod
	delete(nonArgoPod.Annotations, ArgoWorkflowNodeName)
	patchOperation, err := mutatePodIfCached(GetFakeRequestFromPod(&nonArgoPod))
	assert.Nil(t, patchOperation)
	assert.Nil(t, err)
}

func TestMutatePodIfCachedWithTFXPod(t *testing.T) {
	tfxPod := *fakePod
	mainContainerCommand := append(tfxPod.Spec.Containers[0].Command, "/tfx-src/"+TFXPodSuffix)
	tfxPod.Spec.Containers[0].Command = mainContainerCommand
	patchOperation, err := mutatePodIfCached(GetFakeRequestFromPod(&tfxPod))
	assert.Nil(t, patchOperation)
	assert.Nil(t, err)
}

func TestMutatePodIfCached(t *testing.T) {
	patchOperation, err := mutatePodIfCached(&fakeAdmissionRequest)
	assert.Nil(t, err)
	require.NotNil(t, patchOperation)
	require.Equal(t, 1, len(patchOperation))
	require.Equal(t, patchOperation[0].Op, OperationTypeAdd)
}
