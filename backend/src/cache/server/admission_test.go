// Copyright 2020 The Kubeflow Authors
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
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/common/util"
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

var fakeClientManager = NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())

func fakeAdmitFunc(req *v1beta1.AdmissionRequest, clientMgr ClientManagerInterface) ([]patchOperation, error) {
	operation := patchOperation{
		Op:    OperationTypeAdd,
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
		t.Fatalf("failed to send fake request, err: %v", err)
	}

	rr := httptest.NewRecorder()
	patchOperations, err := doServeAdmitFunc(rr, req, fakeAdmitFunc, fakeClientManager)
	require.NotNil(t, patchOperations)
	assert.Nil(t, err)
}

func TestDoServeAdmitFuncWithInvalidHttpMethod(t *testing.T) {
	req, _ := http.NewRequest("Get", "", nil)
	rr := httptest.NewRecorder()
	patchOperations, err := doServeAdmitFunc(rr, req, fakeAdmitFunc, fakeClientManager)
	assert.Nil(t, patchOperations)
	assert.Contains(t, err.Error(), "Invalid method")
}

func TestDoServeAdmitFuncWithInvalidContentType(t *testing.T) {
	req, err := http.NewRequest("POST", "/url", strings.NewReader(""))
	rr := httptest.NewRecorder()
	patchOperations, err := doServeAdmitFunc(rr, req, fakeAdmitFunc, fakeClientManager)
	assert.Nil(t, patchOperations)
	assert.Contains(t, err.Error(), "Unsupported content type")
}

func TestDoServeAdmitFuncWithInvalidRequestBody(t *testing.T) {
	req, err := http.NewRequest("POST", "/url", strings.NewReader("invalid"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	patchOperations, err := doServeAdmitFunc(rr, req, fakeAdmitFunc, fakeClientManager)
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
	patchOperations, err := doServeAdmitFunc(rr, req, fakeAdmitFunc, fakeClientManager)
	assert.Nil(t, patchOperations)
	assert.Contains(t, err.Error(), "Malformed admission review request: request body is nil")
}

type failingAdmissionBody struct{}

func (failingAdmissionBody) Read([]byte) (int, error) { return 0, errors.New("read failed") }
func (failingAdmissionBody) Close() error             { return nil }

func TestServeAdmitFuncBodyLimit(t *testing.T) {
	body, err := json.Marshal(fakeAdmissionReview)
	require.NoError(t, err)
	for _, tc := range []struct {
		name   string
		size   int
		status int
		called bool
	}{
		{"at limit", 32 << 20, http.StatusOK, true},
		{"over limit", (32 << 20) + 1, http.StatusRequestEntityTooLarge, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", io.MultiReader(strings.NewReader(string(body)), strings.NewReader(strings.Repeat(" ", tc.size-len(body)))))
			req.Header.Set(ContentType, JsonContentType)
			recorder := httptest.NewRecorder()
			called := false
			serveAdmitFunc(recorder, req, func(*v1beta1.AdmissionRequest, ClientManagerInterface) ([]patchOperation, error) {
				called = true
				return nil, nil
			}, fakeClientManager)
			require.Equal(t, tc.status, recorder.Code)
			require.Equal(t, tc.called, called)
		})
	}
	t.Run("read error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Body = failingAdmissionBody{}
		req.Header.Set(ContentType, JsonContentType)
		recorder := httptest.NewRecorder()
		serveAdmitFunc(recorder, req, fakeAdmitFunc, fakeClientManager)
		require.Equal(t, http.StatusBadRequest, recorder.Code)
	})
}
