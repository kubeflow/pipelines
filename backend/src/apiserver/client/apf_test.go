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

package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	authv1 "k8s.io/api/authentication/v1"
	authzv1 "k8s.io/api/authorization/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func apfTestClient(t *testing.T, host string, qps float64) *kubernetes.Clientset {
	t.Helper()
	t.Setenv("KUBERNETES_SERVICE_HOST", "")
	path := filepath.Join(t.TempDir(), "kubeconfig")
	require.NoError(t, clientcmd.WriteToFile(clientcmdapi.Config{
		Clusters:       map[string]*clientcmdapi.Cluster{"test": {Server: host}},
		AuthInfos:      map[string]*clientcmdapi.AuthInfo{"test": {}},
		Contexts:       map[string]*clientcmdapi.Context{"test": {Cluster: "test", AuthInfo: "test"}},
		CurrentContext: "test",
	}, path))
	t.Setenv("KUBECONFIG", path)
	client, err := getKubernetesClientset(util.ClientParameters{QPS: qps, Burst: 10})
	require.NoError(t, err)
	return client
}

// Exercise KFP construction and the real pinned client-go transport, rather than
// fake clientsets which do not implement rate limiting or Retry-After handling.
func TestAPFClientReviews(t *testing.T) {
	for _, profiling := range []string{"false", "true"} {
		t.Run("profiling="+profiling, func(t *testing.T) {
			t.Setenv("KFP_RUNTIME_PROFILE_TIMINGS", profiling)
			t.Setenv("KUBE_CLIENT_BACKOFF_BASE", "")
			t.Setenv("KUBE_CLIENT_BACKOFF_DURATION", "")
			for _, kind := range []string{"TokenReview", "SubjectAccessReview"} {
				for _, scenario := range []string{"retry-success", "retry-exhausted", "cancel", "deny", "forbidden", "no-retry-header"} {
					t.Run(kind+"/"+scenario, func(t *testing.T) {
						var calls atomic.Int32
						var first atomic.Int64
						ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
						defer cancel()
						server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
							assert.Equal(t, http.MethodPost, r.Method)
							n := calls.Add(1)
							w.Header().Set("Content-Type", "application/json")
							if scenario == "cancel" {
								w.Header().Set("Retry-After", "60")
								w.WriteHeader(http.StatusTooManyRequests)
								w.(http.Flusher).Flush()
								// Cancellation must interrupt either the response or its retry wait.
								cancel()
								return
							}
							if scenario == "retry-exhausted" || scenario == "no-retry-header" || scenario == "retry-success" && n == 1 {
								if scenario != "no-retry-header" {
									w.Header().Set("Retry-After", "0")
								}
								if scenario == "retry-success" {
									w.Header().Set("Retry-After", "1")
									first.Store(time.Now().UnixNano())
								}
								w.WriteHeader(http.StatusTooManyRequests)
								_ = json.NewEncoder(w).Encode(metav1.Status{Status: "Failure", Reason: metav1.StatusReasonTooManyRequests, Code: 429})
								return
							}
							if scenario == "retry-success" {
								assert.GreaterOrEqual(t, time.Since(time.Unix(0, first.Load())), time.Second)
							}
							if scenario == "forbidden" {
								w.WriteHeader(http.StatusForbidden)
								_ = json.NewEncoder(w).Encode(metav1.Status{Status: "Failure", Reason: metav1.StatusReasonForbidden, Code: 403})
								return
							}
							allowed := scenario != "deny"
							if kind == "TokenReview" {
								_ = json.NewEncoder(w).Encode(authv1.TokenReview{TypeMeta: metav1.TypeMeta{APIVersion: "authentication.k8s.io/v1", Kind: kind}, Status: authv1.TokenReviewStatus{Authenticated: allowed}})
							} else {
								_ = json.NewEncoder(w).Encode(authzv1.SubjectAccessReview{TypeMeta: metav1.TypeMeta{APIVersion: "authorization.k8s.io/v1", Kind: kind}, Status: authzv1.SubjectAccessReviewStatus{Allowed: allowed, Denied: !allowed}})
							}
						}))
						defer server.Close()
						client := apfTestClient(t, server.URL, -1)
						require.Nil(t, client.AuthenticationV1().RESTClient().GetRateLimiter())
						require.Nil(t, client.AuthorizationV1().RESTClient().GetRateLimiter())
						var allowed bool
						var err error
						if kind == "TokenReview" {
							var result *authv1.TokenReview
							result, err = client.AuthenticationV1().TokenReviews().Create(ctx, &authv1.TokenReview{Spec: authv1.TokenReviewSpec{Token: "test-only"}}, metav1.CreateOptions{})
							if err == nil {
								allowed = result.Status.Authenticated
							}
						} else {
							var result *authzv1.SubjectAccessReview
							result, err = client.AuthorizationV1().SubjectAccessReviews().Create(ctx, &authzv1.SubjectAccessReview{Spec: authzv1.SubjectAccessReviewSpec{User: "test-only"}}, metav1.CreateOptions{})
							if err == nil {
								allowed = result.Status.Allowed
							}
						}
						switch scenario {
						case "retry-success":
							require.NoError(t, err)
							require.True(t, allowed)
							require.EqualValues(t, 2, calls.Load())
						case "retry-exhausted":
							require.True(t, apierrors.IsTooManyRequests(err), "%v", err)
							require.EqualValues(t, 11, calls.Load())
						case "no-retry-header":
							require.True(t, apierrors.IsTooManyRequests(err), "%v", err)
							require.EqualValues(t, 1, calls.Load())
						case "cancel":
							require.True(t, errors.Is(err, context.Canceled), "%v", err)
							require.EqualValues(t, 1, calls.Load())
						case "deny":
							require.NoError(t, err)
							require.False(t, allowed)
							require.EqualValues(t, 1, calls.Load())
						case "forbidden":
							require.True(t, apierrors.IsForbidden(err), "%v", err)
							require.EqualValues(t, 1, calls.Load())
						}
					})
				}
			}
			// Zero restores client-go defaults; it does not opt in to server throttling.
			client := apfTestClient(t, "http://localhost", 0)
			require.NotNil(t, client.AuthenticationV1().RESTClient().GetRateLimiter())
			require.Equal(t, float32(5), client.AuthenticationV1().RESTClient().GetRateLimiter().QPS())
		})
	}
}
