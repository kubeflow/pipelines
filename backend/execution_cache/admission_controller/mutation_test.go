package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
