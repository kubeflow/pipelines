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

package storage

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/crd/kubernetes/v2beta1"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestPipelineVersionWrites_AdmissionRejections(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	t.Cleanup(func() { viper.Set("POD_NAMESPACE", podNamespace) })
	const message = "The spec.pipelineName doesn't map to an existing Pipeline object"
	denial := &k8serrors.StatusError{ErrStatus: metav1.Status{
		Code: 400, Reason: metav1.StatusReasonBadRequest,
		Message: fmt.Sprintf("admission webhook %q denied the request: %s", "pipelineversions.pipelines.kubeflow.org", message),
		Details: &metav1.StatusDetails{Causes: []metav1.StatusCause{{Type: "pipelines.kubeflow.org/PipelineVersionRejected", Message: message}}},
	}}
	testCases := []struct {
		name string
		err  error
		code codes.Code
	}{
		{"KFP input rejection", denial, codes.InvalidArgument},
		{"unmarked bad request", k8serrors.NewBadRequest("another webhook denied the request"), codes.Internal},
		{"webhook dependency failure", k8serrors.NewInternalError(fmt.Errorf("failed to get Pipeline: service unavailable")), codes.Internal},
	}
	for _, operation := range []string{"create", "update"} {
		for _, testCase := range testCases {
			t.Run(operation+"/"+testCase.name, func(t *testing.T) {
				scheme := runtime.NewScheme()
				require.NoError(t, v2beta1.AddToScheme(scheme))
				k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
					&v2beta1.Pipeline{ObjectMeta: metav1.ObjectMeta{UID: "pipeline-id", Name: "my-pipeline", Namespace: "Test"}},
					&v2beta1.PipelineVersion{ObjectMeta: metav1.ObjectMeta{UID: "version-id", Name: "my-pipeline-v1", Namespace: "Test"}},
				).WithInterceptorFuncs(interceptor.Funcs{
					Create: func(context.Context, client.WithWatch, client.Object, ...client.CreateOption) error {
						return testCase.err
					},
					Update: func(context.Context, client.WithWatch, client.Object, ...client.UpdateOption) error {
						return testCase.err
					},
				}).Build()
				store := NewPipelineStoreKubernetes(k8sClient, k8sClient)
				var err error
				if operation == "create" {
					_, err = store.CreatePipelineVersion(&model.PipelineVersion{Name: "v2", PipelineId: "pipeline-id", PipelineSpec: model.LargeText(getBasicPipelineSpecYAML())})
				} else {
					err = store.UpdatePipelineVersionFields("version-id", "renamed", nil)
				}
				var userError *util.UserError
				require.ErrorAs(t, err, &userError)
				assert.Equal(t, testCase.code, userError.ExternalStatusCode())
				if testCase.code == codes.InvalidArgument {
					assert.Equal(t, message, userError.ExternalMessage())
					assert.NotContains(t, userError.GRPCStatus().Message(), "admission webhook")
					assert.Equal(t, 1, strings.Count(userError.GRPCStatus().Message(), message))
				} else {
					assert.Equal(t, "Internal Server Error", userError.ExternalMessage())
				}
			})
		}
	}
}
