// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/driver/driverapi"
	"github.com/kubeflow/pipelines/backend/src/v2/client_manager"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func agentPodForAuth() *corev1.Pod {
	controller := true
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "workflow-agent", Namespace: "run-namespace", UID: "agent-uid",
			Labels: map[string]string{util.LabelKeyWorkflowRunId: "run-id"},
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "argoproj.io/v1alpha1", Kind: "Workflow", Name: "workflow",
				UID: "workflow-uid", Controller: &controller,
			}},
		},
		Spec: corev1.PodSpec{ServiceAccountName: "runtime-sa"},
	}
}

func driverArgsForAuth() driverapi.DriverPluginArgs {
	return driverapi.DriverPluginArgs{
		Namespace: "run-namespace", RunID: "run-id", RunName: "workflow",
		MlPipelineServerAddress: "ml-pipeline", MlPipelineServerPort: "8887",
	}
}

func TestDriverAPIClientConfigUsesAgentServiceAccount(t *testing.T) {
	for _, audience := range []string{"", "custom.kfp.example/runs/run-id"} {
		t.Run(audience, func(t *testing.T) {
			pod := agentPodForAuth()
			client := fake.NewSimpleClientset(pod)
			args := driverArgsForAuth()
			args.KFPTokenAudience = audience
			wantAudience := audience
			if wantAudience == "" {
				wantAudience = "pipelines.kubeflow.org/runs/run-id"
			}
			client.PrependReactor("create", "serviceaccounts", func(action k8stesting.Action) (bool, runtime.Object, error) {
				create := action.(k8stesting.CreateActionImpl)
				assert.Equal(t, "token", create.GetSubresource())
				assert.Equal(t, pod.Namespace, create.GetNamespace())
				assert.Equal(t, "runtime-sa", create.Name)
				request := create.GetObject().(*authenticationv1.TokenRequest)
				assert.Equal(t, []string{wantAudience}, request.Spec.Audiences)
				require.NotNil(t, request.Spec.BoundObjectRef)
				assert.Equal(t, pod.Name, request.Spec.BoundObjectRef.Name)
				assert.Equal(t, pod.UID, request.Spec.BoundObjectRef.UID)
				return true, &authenticationv1.TokenRequest{Status: authenticationv1.TokenRequestStatus{
					Token: "test-run-token", ExpirationTimestamp: metav1.NewTime(time.Now().Add(time.Hour)),
				}}, nil
			})

			cfg, actualPod, err := driverAPIClientConfig(context.Background(), client, pod.Namespace, pod.Name, args)
			require.NoError(t, err)
			assert.Equal(t, pod, actualPod)
			assert.Equal(t, "ml-pipeline:8887", cfg.Endpoint)
			require.NotNil(t, cfg.TokenSource)
			token, err := cfg.TokenSource.Token(context.Background())
			require.NoError(t, err)
			assert.Equal(t, "test-run-token", token)
		})
	}
}

func TestDriverAPIClientConfigRejectsUnboundRequests(t *testing.T) {
	for _, tt := range []struct {
		name   string
		mutate func(*corev1.Pod, *driverapi.DriverPluginArgs)
	}{
		{"namespace mismatch", func(_ *corev1.Pod, args *driverapi.DriverPluginArgs) { args.Namespace = "another-namespace" }},
		{"run mismatch", func(_ *corev1.Pod, args *driverapi.DriverPluginArgs) { args.RunID = "another-run" }},
		{"missing run label", func(pod *corev1.Pod, _ *driverapi.DriverPluginArgs) { pod.Labels = nil }},
		{"workflow mismatch", func(_ *corev1.Pod, args *driverapi.DriverPluginArgs) { args.RunName = "another-workflow" }},
		{"missing owner", func(pod *corev1.Pod, _ *driverapi.DriverPluginArgs) { pod.OwnerReferences = nil }},
		{"wrong owner kind", func(pod *corev1.Pod, _ *driverapi.DriverPluginArgs) { pod.OwnerReferences[0].Kind = "Deployment" }},
		{"missing service account", func(pod *corev1.Pod, _ *driverapi.DriverPluginArgs) { pod.Spec.ServiceAccountName = "" }},
		{"missing pod UID", func(pod *corev1.Pod, _ *driverapi.DriverPluginArgs) { pod.UID = "" }},
		{"broad audience", func(_ *corev1.Pod, args *driverapi.DriverPluginArgs) {
			args.KFPTokenAudience = "pipelines.kubeflow.org"
		}},
		{"another run audience", func(_ *corev1.Pod, args *driverapi.DriverPluginArgs) {
			args.KFPTokenAudience = "pipelines.kubeflow.org/runs/other-run"
		}},
		{"missing audience base", func(_ *corev1.Pod, args *driverapi.DriverPluginArgs) { args.KFPTokenAudience = "/runs/run-id" }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			pod, args := agentPodForAuth(), driverArgsForAuth()
			tt.mutate(pod, &args)
			client := fake.NewSimpleClientset(pod)
			cfg, _, err := driverAPIClientConfig(context.Background(), client, pod.Namespace, pod.Name, args)
			require.Error(t, err)
			assert.Nil(t, cfg)
			for _, action := range client.Actions() {
				assert.NotEqual(t, "create", action.GetVerb(), "invalid requests must not mint tokens")
			}
		})
	}
}

func TestDriverAPIClientConfigFailsWhenPodCannotBeRead(t *testing.T) {
	cfg, _, err := driverAPIClientConfig(context.Background(), fake.NewSimpleClientset(), "run-namespace", "missing-pod", driverArgsForAuth())
	require.ErrorContains(t, err, "failed to get executor plugin Pod")
	assert.Nil(t, cfg)
}

type authenticatedRunServer struct {
	go_client.UnimplementedRunServiceServer
}

func (*authenticatedRunServer) GetRun(ctx context.Context, request *go_client.GetRunRequest) (*go_client.Run, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	auth := md.Get("authorization")
	if len(auth) != 1 || auth[0] != "Bearer test-runtime-sa-token" {
		return nil, status.Error(codes.Unauthenticated, "expected runtime service account token")
	}
	return &go_client.Run{RunId: request.RunId, ServiceAccount: "runtime-sa"}, nil
}

func TestDriverAuthenticatesFirstKFPRPCWithRunServiceAccount(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := grpc.NewServer()
	go_client.RegisterRunServiceServer(server, &authenticatedRunServer{})
	t.Cleanup(server.Stop)
	go server.Serve(listener)

	pod := agentPodForAuth()
	k8sClient := fake.NewSimpleClientset(pod)
	issued := 0
	k8sClient.PrependReactor("create", "serviceaccounts", func(action k8stesting.Action) (bool, runtime.Object, error) {
		create := action.(k8stesting.CreateActionImpl)
		if create.Name != "runtime-sa" || create.GetNamespace() != pod.Namespace || create.GetSubresource() != "token" {
			return true, nil, status.Error(codes.PermissionDenied, "wrong token identity")
		}
		issued++
		return true, &authenticationv1.TokenRequest{Status: authenticationv1.TokenRequestStatus{
			Token: "test-runtime-sa-token", ExpirationTimestamp: metav1.NewTime(time.Now().Add(time.Hour)),
		}}, nil
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cfg, _, err := driverAPIClientConfig(ctx, k8sClient, pod.Namespace, pod.Name, driverArgsForAuth())
	require.NoError(t, err)
	cfg.Endpoint = listener.Addr().String()
	manager, err := client_manager.NewClientManager(&client_manager.Options{APIClientConfig: cfg, K8sClient: k8sClient})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close()) })
	assert.Same(t, k8sClient, manager.K8sClient())
	for range 2 {
		run, err := manager.KFPAPIClient().GetRun(ctx, &go_client.GetRunRequest{RunId: "run-id"})
		require.NoError(t, err)
		assert.Equal(t, "runtime-sa", run.ServiceAccount)
	}
	assert.Equal(t, 1, issued, "successive RPCs should reuse the token until refresh")
}

func TestAuthenticatedPluginHandler(t *testing.T) {
	tokenPath := filepath.Join(t.TempDir(), "token")
	require.NoError(t, os.WriteFile(tokenPath, []byte("test-agent-token\n"), 0600))
	for _, header := range []string{"", "Bearer wrong-token", "test-agent-token", "Bearer test-agent-token"} {
		t.Run(header, func(t *testing.T) {
			called := false
			handler, err := authenticatedPluginHandler(tokenPath, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				called = true
				w.WriteHeader(http.StatusNoContent)
			}))
			require.NoError(t, err)
			request := httptest.NewRequest(http.MethodPost, "/api/v1/template.execute", nil)
			request.Header.Set("Authorization", header)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if header == "Bearer test-agent-token" {
				assert.True(t, called)
				assert.Equal(t, http.StatusNoContent, response.Code)
			} else {
				assert.False(t, called)
				assert.Equal(t, http.StatusForbidden, response.Code)
			}
		})
	}
}

func TestAuthenticatedPluginHandlerRequiresToken(t *testing.T) {
	tokenPath := filepath.Join(t.TempDir(), "token")
	_, err := authenticatedPluginHandler(tokenPath, nil)
	require.Error(t, err)
	require.NoError(t, os.WriteFile(tokenPath, []byte(" \n"), 0600))
	_, err = authenticatedPluginHandler(tokenPath, nil)
	require.ErrorContains(t, err, "token is empty")
}
