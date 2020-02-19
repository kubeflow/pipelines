package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	fakeAdmissionReview = v1beta1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			Kind:       "pods",
			APIVersion: "v1",
		},
		Request: &v1beta1.AdmissionRequest{
			UID: "123",
		},
		Response: &v1beta1.AdmissionResponse{
			UID: "123",
		},
	}
)

func fakeAdmitFunc(req *v1beta1.AdmissionRequest) ([]patchOperation, error) {
	operation := patchOperation{
		Op:    "add",
		Path:  "test",
		Value: "test",
	}
	return []patchOperation{operation}, nil
}

func TestIsKubeNamespace(t *testing.T) {
	assert.True(t, isKubeNamespace("kube-public"))
	assert.True(t, isKubeNamespace("kube-system"))
	assert.False(t, isKubeNamespace("kube"))
}

func TestDoServeAdmitFunc(t *testing.T) {
	body, err := json.Marshal(fakeAdmissionReview)
	req, err := http.NewRequest("POST", "/url", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")

	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	patchOperations, err := doServeAdmitFunc(rr, req, fakeAdmitFunc)
	require.NotNil(t, patchOperations)
	assert.Nil(t, err)
}

func TestDoServeAdmitFuncWithInvalidHttpMethod(t *testing.T) {
	req, _ := http.NewRequest("Get", "", nil)
	rr := httptest.NewRecorder()
	patchOperations, err := doServeAdmitFunc(rr, req, fakeAdmitFunc)
	assert.Nil(t, patchOperations)
	assert.Contains(t, err.Error(), "Invalid method")
}

func TestDoServeAdmitFuncWithInvalidContentType(t *testing.T) {
	req, err := http.NewRequest("POST", "/url", strings.NewReader(""))
	rr := httptest.NewRecorder()
	patchOperations, err := doServeAdmitFunc(rr, req, fakeAdmitFunc)
	assert.Nil(t, patchOperations)
	assert.Contains(t, err.Error(), "Unsupported content type")
}

func TestDoServeAdmitFuncWithInvalidRequestBody(t *testing.T) {
	req, err := http.NewRequest("POST", "/url", strings.NewReader("invalid"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	patchOperations, err := doServeAdmitFunc(rr, req, fakeAdmitFunc)
	assert.Nil(t, patchOperations)
	assert.Contains(t, err.Error(), "Could not deserialize request")
}

func TestDoServeAdmitFuncWithEmptyAdmissionRequest(t *testing.T) {
	invalidRequest := fakeAdmissionReview
	invalidRequest.Request = nil
	body, _ := json.Marshal(invalidRequest)
	req, _ := http.NewRequest("POST", "/url", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	patchOperations, err := doServeAdmitFunc(rr, req, fakeAdmitFunc)
	assert.Nil(t, patchOperations)
	assert.Contains(t, err.Error(), "Malformed admission review request: request body is nil")
}
