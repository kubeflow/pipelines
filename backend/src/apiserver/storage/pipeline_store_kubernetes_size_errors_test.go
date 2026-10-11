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

func TestPipelineVersionWrites_KubernetesSizeErrors(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	t.Cleanup(func() { viper.Set("POD_NAMESPACE", podNamespace) })

	testCases := []struct {
		name string
		err  error
		code codes.Code
	}{
		{"etcd request limit", &k8serrors.StatusError{ErrStatus: metav1.Status{Code: 500, Message: "etcdserver: request is too large"}}, codes.InvalidArgument},
		{"etcd client send limit", &k8serrors.StatusError{ErrStatus: metav1.Status{Code: 500, Message: "rpc error: code = ResourceExhausted desc = trying to send message larger than max (2097703 vs. 2097152)"}}, codes.InvalidArgument},
		{"API server body limit", k8serrors.NewRequestEntityTooLargeError("limit is 3145728"), codes.InvalidArgument},
		{"untyped error with size text", fmt.Errorf("etcdserver: request is too large"), codes.Internal},
		{"bad request with size text", k8serrors.NewBadRequest("etcdserver: request is too large"), codes.Internal},
		{"receive limit", k8serrors.NewInternalError(fmt.Errorf("received message larger than max")), codes.Internal},
		{"unrelated server error", k8serrors.NewInternalError(fmt.Errorf("storage unavailable")), codes.Internal},
		{"unknown write outcome", fmt.Errorf("http2: client connection lost"), codes.Internal},
	}
	for _, operation := range []string{"create", "update"} {
		for _, testCase := range testCases {
			t.Run(operation+"/"+testCase.name, func(t *testing.T) {
				scheme := runtime.NewScheme()
				require.NoError(t, v2beta1.AddToScheme(scheme))
				pipeline := &v2beta1.Pipeline{ObjectMeta: metav1.ObjectMeta{UID: "pipeline-id", Name: "my-pipeline", Namespace: "Test"}}
				version := &v2beta1.PipelineVersion{ObjectMeta: metav1.ObjectMeta{UID: "version-id", Name: "my-pipeline-v1", Namespace: "Test"}}
				k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(pipeline, version).
					WithInterceptorFuncs(interceptor.Funcs{
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
					err = store.UpdatePipelineVersionFields("version-id", strings.Repeat("n", 16<<10), nil)
				}
				var userError *util.UserError
				require.ErrorAs(t, err, &userError)
				assert.Equal(t, testCase.code, userError.ExternalStatusCode())
				if testCase.code == codes.InvalidArgument {
					assert.Contains(t, userError.ExternalMessage(), "too large")
					assert.Contains(t, userError.ExternalMessage(), "Kubernetes")
					assert.Contains(t, userError.ExternalMessage(), "Reduce")
					if operation == "create" {
						assert.Contains(t, userError.ExternalMessage(), "compiled pipeline")
					} else {
						assert.Contains(t, userError.ExternalMessage(), "display name")
						assert.Contains(t, userError.ExternalMessage(), "tags")
					}
					assert.Equal(t, 1, strings.Count(userError.GRPCStatus().Message(), testCase.err.Error()))
				} else {
					assert.Equal(t, "Internal Server Error", userError.ExternalMessage())
				}
			})
		}
	}
}

func TestCreatePipelineAndPipelineVersion_SizeRejectionRecovery(t *testing.T) {
	podNamespace := viper.Get("POD_NAMESPACE")
	viper.Set("POD_NAMESPACE", "Test")
	t.Cleanup(func() { viper.Set("POD_NAMESPACE", podNamespace) })

	testCases := []struct {
		name      string
		err       error
		persisted bool
	}{
		{"etcd request limit", &k8serrors.StatusError{ErrStatus: metav1.Status{Code: 500, Message: "etcdserver: request is too large"}}, false},
		{"etcd client send limit", &k8serrors.StatusError{ErrStatus: metav1.Status{Code: 500, Message: "rpc error: code = ResourceExhausted desc = trying to send message larger than max (2097703 vs. 2097152)"}}, false},
		{"API server body limit", k8serrors.NewRequestEntityTooLargeError("limit is 3145728"), false},
		{"persisted before lost response", fmt.Errorf("http2: client connection lost"), true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			require.NoError(t, v2beta1.AddToScheme(scheme))
			failVersion := true
			k8sClient := fake.NewClientBuilder().WithScheme(scheme).
				WithInterceptorFuncs(interceptor.Funcs{
					Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
						if _, ok := obj.(*v2beta1.Pipeline); ok {
							obj.SetUID("pipeline-id")
						} else {
							obj.SetUID("version-id")
							if failVersion {
								if testCase.persisted {
									require.NoError(t, c.Create(ctx, obj, opts...))
								}
								return testCase.err
							}
						}
						return c.Create(ctx, obj, opts...)
					},
				}).Build()
			store := NewPipelineStoreKubernetes(k8sClient, k8sClient)
			_, _, err := store.CreatePipelineAndPipelineVersion(
				&model.Pipeline{Name: "large-pipeline", Namespace: "Test"},
				&model.PipelineVersion{Name: "v1", PipelineSpec: model.LargeText(getBasicPipelineSpecYAML())})
			var userError *util.UserError
			require.ErrorAs(t, err, &userError)
			pipeline, err := store.GetPipeline("pipeline-id")
			require.NoError(t, err)
			assert.Equal(t, "large-pipeline", pipeline.Name)
			if testCase.persisted {
				assert.Equal(t, codes.Internal, userError.ExternalStatusCode())
				assert.Equal(t, "Internal Server Error", userError.ExternalMessage())
			} else {
				assert.Equal(t, codes.InvalidArgument, userError.ExternalStatusCode())
				assert.Contains(t, userError.ExternalMessage(), "pipeline-id")
				assert.Contains(t, userError.ExternalMessage(), "large-pipeline")
				assert.Contains(t, userError.ExternalMessage(), "Upload the corrected version")
				assert.Equal(t, 1, strings.Count(userError.GRPCStatus().Message(), testCase.err.Error()))
				assert.Equal(t, 1, strings.Count(userError.GRPCStatus().Message(), "Kubernetes rejected"))
				_, err = store.GetPipelineVersionByName("pipeline-id", "v1")
				assert.True(t, util.IsUserErrorCodeMatch(err, codes.NotFound), "a rejected version must not have been persisted")
				failVersion = false
				_, err = store.CreatePipelineVersion(&model.PipelineVersion{Name: "v1", PipelineId: "pipeline-id", PipelineSpec: model.LargeText(getBasicPipelineSpecYAML())})
				require.NoError(t, err)
			}
			version, err := store.GetPipelineVersionByName("pipeline-id", "v1")
			require.NoError(t, err)
			assert.Equal(t, "pipeline-id", version.PipelineId)
		})
	}
}
