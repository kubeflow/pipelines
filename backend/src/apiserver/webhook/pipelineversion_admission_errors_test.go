// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	k8sapi "github.com/kubeflow/pipelines/backend/src/crd/kubernetes/v2beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionv1 "k8s.io/api/admission/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestPipelineVersionWebhook_AdmissionErrorContract(t *testing.T) {
	pipelineWebhook, validSpec := setupPipelineWebhookTest(t)
	version := &k8sapi.PipelineVersion{
		TypeMeta:   metav1.TypeMeta{APIVersion: k8sapi.GroupVersion.String(), Kind: "PipelineVersion"},
		ObjectMeta: metav1.ObjectMeta{Name: "test-pipeline-v1", Namespace: "default"},
		Spec:       k8sapi.PipelineVersionSpec{PipelineName: "test-pipeline", PipelineSpec: k8sapi.IRSpec{Value: validSpec}},
	}
	invalid := version.DeepCopy()
	invalid.Spec.PipelineSpec.Value = nil
	missingParent := version.DeepCopy()
	missingParent.Spec.PipelineName = "missing"
	immutableUpdate := version.DeepCopy()
	immutableUpdate.Generation = version.Generation + 1
	immutableUpdate.Spec.Description = "changed immutable description"
	mutableUpdate := version.DeepCopy()
	mutableUpdate.Generation = version.Generation + 1
	mutableUpdate.Spec.DisplayName = "renamed"
	mutableUpdate.Spec.Tags = map[string]string{"stage": "test"}
	brokenClient := func(err error) client.Client {
		return fake.NewClientBuilder().WithScheme(scheme).WithInterceptorFuncs(interceptor.Funcs{
			Get: func(context.Context, client.WithWatch, client.ObjectKey, client.Object, ...client.GetOption) error {
				return err
			},
		}).Build()
	}
	dependencyFailure := fmt.Errorf("Kubernetes API unavailable")
	testCases := []struct {
		name     string
		version  *k8sapi.PipelineVersion
		previous *k8sapi.PipelineVersion
		mutating bool
		cached   client.Client
		direct   client.Client
		code     int32
	}{
		{"invalid spec", invalid, nil, false, pipelineWebhook.Client, pipelineWebhook.Client, 400},
		{"missing Pipeline", missingParent, nil, true, pipelineWebhook.Client, pipelineWebhook.Client, 400},
		{"immutable update", immutableUpdate, version, false, pipelineWebhook.Client, pipelineWebhook.Client, 400},
		{"mutable update", mutableUpdate, version, false, pipelineWebhook.Client, pipelineWebhook.Client, 200},
		{"cached lookup failure", version, nil, true, brokenClient(dependencyFailure), pipelineWebhook.Client, 500},
		{"direct lookup failure", version, nil, true, brokenClient(apierrors.NewNotFound(k8sapi.GroupVersion.WithResource("pipelines").GroupResource(), "test-pipeline")), brokenClient(dependencyFailure), 500},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			validating, mutating, err := NewPipelineVersionWebhook(testCase.cached, testCase.direct)
			require.NoError(t, err)
			handler := validating
			if testCase.mutating {
				handler = mutating
			}
			raw, err := json.Marshal(testCase.version)
			require.NoError(t, err)
			operation := admissionv1.Create
			var previous []byte
			if testCase.previous != nil {
				operation = admissionv1.Update
				previous, err = json.Marshal(testCase.previous)
				require.NoError(t, err)
			}
			body, err := json.Marshal(admissionv1.AdmissionReview{
				TypeMeta: metav1.TypeMeta{APIVersion: "admission.k8s.io/v1", Kind: "AdmissionReview"},
				Request: &admissionv1.AdmissionRequest{
					UID: "request-id", Operation: operation, Namespace: "default",
					Kind:      metav1.GroupVersionKind{Group: k8sapi.GroupVersion.Group, Version: k8sapi.GroupVersion.Version, Kind: "PipelineVersion"},
					Resource:  metav1.GroupVersionResource{Group: k8sapi.GroupVersion.Group, Version: k8sapi.GroupVersion.Version, Resource: "pipelineversions"},
					Object:    runtime.RawExtension{Raw: raw},
					OldObject: runtime.RawExtension{Raw: previous},
				},
			})
			require.NoError(t, err)
			request := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			var review admissionv1.AdmissionReview
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &review))
			require.NotNil(t, review.Response)
			assert.Equal(t, testCase.code == 200, review.Response.Allowed)
			require.NotNil(t, review.Response.Result)
			assert.Equal(t, testCase.code, review.Response.Result.Code)
			// Use the literal cause type to pin the wire contract independently of the production constant.
			cause, ok := apierrors.StatusCause(&apierrors.StatusError{ErrStatus: *review.Response.Result}, "pipelines.kubeflow.org/PipelineVersionRejected")
			if testCase.code == 400 {
				require.True(t, ok)
				assert.Equal(t, review.Response.Result.Message, cause.Message)
			} else {
				assert.False(t, ok)
			}
		})
	}
}
