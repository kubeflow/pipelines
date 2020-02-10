package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/api/admission/v1beta1"
)

func TestIsKubeNamespace(t *testing.T) {
	assert.True(t, isKubeNamespace("kube-public"))
	assert.True(t, isKubeNamespace("kube-system"))
	assert.False(t, isKubeNamespace("kube"))
}

func mockAdmitFunc(*v1beta1.AdmissionRequest) ([]patchOperation, error) {
	return nil, nil
}

// func TestAdmitFuncHandler(t *testing.T) {
// 	req, err := http.NewRequest("POST", "/url", nil)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	rr := httptest.NewRecorder()
// 	handler := http.Handler(admitFuncHandler(mockAdmitFunc))

// 	handler.ServeHTTP(rr, req)

// }
