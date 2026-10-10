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

package executorplugin

import (
	"context"
	"fmt"
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
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
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

type recordingWarningHandler struct {
	messages []string
}

func (h *recordingWarningHandler) HandleWarningHeaderWithContext(_ context.Context, _ int, _, message string) {
	h.messages = append(h.messages, message)
}

func TestExecutorPluginWarningHandlerFiltersOnlyLegacyTokenWarning(t *testing.T) {
	delegate := &recordingWarningHandler{}
	handler := executorPluginWarningHandler{delegate: delegate}
	handler.HandleWarningHeaderWithContext(context.Background(), 299, "kubernetes", legacyServiceAccountTokenWarning)
	handler.HandleWarningHeaderWithContext(context.Background(), 299, "kubernetes", "another warning")

	assert.Equal(t, []string{"another warning"}, delegate.messages)
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
	args := driverArgsForAuth()
	args.MlPipelineServerAddress, args.MlPipelineServerPort, err = net.SplitHostPort(listener.Addr().String())
	require.NoError(t, err)
	factory := inClusterClientFactory()
	factory.identity = func() (string, string, error) { return pod.Namespace, pod.Name, nil }
	factory.kubernetesConfig = func() (*rest.Config, error) { return &rest.Config{}, nil }
	factory.kubernetesClient = func(*rest.Config) (kubernetes.Interface, error) { return k8sClient, nil }
	clients, err := factory.newDriverClientManager(ctx, args, "")
	require.NoError(t, err)
	manager := clients.manager
	t.Cleanup(func() { require.NoError(t, manager.Close()) })
	assert.Same(t, k8sClient, manager.K8sClient())
	for range 2 {
		run, err := manager.KFPAPIClient().GetRun(ctx, &go_client.GetRunRequest{RunId: "run-id"})
		require.NoError(t, err)
		assert.Equal(t, "runtime-sa", run.ServiceAccount)
	}
	assert.Equal(t, 1, issued, "successive RPCs should reuse the token until refresh")
}

func TestExecutorPluginIdentity(t *testing.T) {
	namespacePath := filepath.Join(t.TempDir(), "namespace")
	_, _, err := executorPluginIdentity(namespacePath)
	require.ErrorIs(t, err, os.ErrNotExist)
	require.ErrorContains(t, err, "check the service account mount")
	require.NoError(t, os.WriteFile(namespacePath, []byte(" \n"), 0600))
	_, _, err = executorPluginIdentity(namespacePath)
	require.ErrorContains(t, err, "namespace is empty")
	require.NoError(t, os.WriteFile(namespacePath, []byte("run-namespace\n"), 0600))
	t.Setenv("KFP_POD_NAME", "workflow-agent")
	namespace, podName, err := executorPluginIdentity(namespacePath)
	require.NoError(t, err)
	assert.Equal(t, "run-namespace", namespace)
	assert.Equal(t, "workflow-agent", podName)
}

func TestNewDriverClientManagerConfiguration(t *testing.T) {
	pod := agentPodForAuth()
	k8sClient := fake.NewSimpleClientset(pod)
	args := driverArgsForAuth()
	args.MlPipelineTLSEnabled = true
	args.MlPipelineGRPCBackoffJitter = "0"
	manager := &invocationManager{k8s: k8sClient}
	metadataCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		metadataCalls++
		assert.Equal(t, "/apis/argoproj.io/v1alpha1/namespaces/run-namespace/workflows/workflow", request.URL.Path)
		assert.Equal(t, "Bearer bootstrap-token", request.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"apiVersion":"argoproj.io/v1alpha1","kind":"Workflow","metadata":{"name":"workflow","namespace":"run-namespace"}}`))
		assert.NoError(t, err)
	}))
	defer server.Close()
	restConfig := &rest.Config{Host: server.URL, BearerToken: "bootstrap-token"}
	factory := inClusterClientFactory()
	factory.identity = func() (string, string, error) { return pod.Namespace, pod.Name, nil }
	configCalls := 0
	factory.kubernetesConfig = func() (*rest.Config, error) { configCalls++; return restConfig, nil }
	factory.kubernetesClient = func(config *rest.Config) (kubernetes.Interface, error) {
		assert.Same(t, restConfig, config)
		return k8sClient, nil
	}
	factory.manager = func(options *client_manager.Options) (driverClientManager, error) {
		assert.True(t, options.MLPipelineTLSEnabled)
		assert.Equal(t, "/ca/cert", options.CaCertPath)
		assert.Same(t, k8sClient, options.K8sClient)
		require.NotNil(t, options.APIClientConfig.TokenSource)
		assert.Equal(t, "ml-pipeline:8887", options.APIClientConfig.Endpoint)
		assert.Equal(t, "0", options.APIClientConfig.BackoffJitter)
		return manager, nil
	}
	clients, err := factory.newDriverClientManager(context.Background(), args, "/ca/cert")
	require.NoError(t, err)
	assert.Same(t, manager, clients.manager)
	assert.Equal(t, pod, clients.pod)
	assert.Zero(t, metadataCalls, "workflow client is only needed for placeholder fallback")
	metadata, err := clients.workflowMetadata(context.Background(), pod.Namespace, "workflow")
	require.NoError(t, err)
	assert.Equal(t, "workflow", metadata.Name)
	assert.Equal(t, 1, metadataCalls)
	assert.Equal(t, 1, configCalls, "reuse invocation REST configuration for workflow metadata")
}

func TestNewDriverClientManagerFailures(t *testing.T) {
	for _, tc := range []struct {
		name, message string
		mutate        func(*clientFactory, *driverapi.DriverPluginArgs)
		cause         error
	}{
		{"identity", "identity failure", func(f *clientFactory, _ *driverapi.DriverPluginArgs) {
			f.identity = func() (string, string, error) { return "", "", fmt.Errorf("identity failure: %w", assert.AnError) }
		}, assert.AnError},
		{"config", "load executor plugin Kubernetes config", func(f *clientFactory, _ *driverapi.DriverPluginArgs) {
			f.kubernetesConfig = func() (*rest.Config, error) { return nil, assert.AnError }
		}, assert.AnError},
		{"client", "initialize executor plugin Kubernetes client", func(f *clientFactory, _ *driverapi.DriverPluginArgs) {
			f.kubernetesClient = func(*rest.Config) (kubernetes.Interface, error) { return nil, assert.AnError }
		}, assert.AnError},
		{"namespace", "namespace does not match", func(_ *clientFactory, args *driverapi.DriverPluginArgs) { args.Namespace = "other" }, nil},
		{"pod", "failed to get executor plugin Pod", func(f *clientFactory, _ *driverapi.DriverPluginArgs) {
			f.kubernetesClient = func(*rest.Config) (kubernetes.Interface, error) { return fake.NewSimpleClientset(), nil }
		}, nil},
		{"run binding", "run ID does not match", func(_ *clientFactory, args *driverapi.DriverPluginArgs) { args.RunID = "other" }, nil},
		{"manager", "check the KFP endpoint and TLS settings", func(f *clientFactory, _ *driverapi.DriverPluginArgs) {
			f.manager = func(*client_manager.Options) (driverClientManager, error) { return nil, assert.AnError }
		}, assert.AnError},
		{"TLS CA file", "initialize driver API client", func(f *clientFactory, args *driverapi.DriverPluginArgs) {
			args.MlPipelineTLSEnabled = true
			f.manager = inClusterClientFactory().manager
		}, os.ErrNotExist},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pod := agentPodForAuth()
			args := driverArgsForAuth()
			factory := clientFactory{
				identity:         func() (string, string, error) { return pod.Namespace, pod.Name, nil },
				kubernetesConfig: func() (*rest.Config, error) { return &rest.Config{}, nil },
				kubernetesClient: func(*rest.Config) (kubernetes.Interface, error) { return fake.NewSimpleClientset(pod), nil },
				manager: func(*client_manager.Options) (driverClientManager, error) {
					t.Fatal("failed identity must not construct an API client")
					return nil, nil
				},
			}
			tc.mutate(&factory, &args)
			clients, err := factory.newDriverClientManager(context.Background(), args, filepath.Join(t.TempDir(), "missing-ca"))
			require.ErrorContains(t, err, tc.message)
			if tc.cause != nil {
				assert.ErrorIs(t, err, tc.cause)
			}
			assert.Nil(t, clients)
		})
	}
}

func TestExecutorPluginKubernetesConfig(t *testing.T) {
	// Avoid both the developer's kubeconfig and in-cluster discovery.
	t.Setenv("KUBERNETES_SERVICE_HOST", "")
	t.Setenv("KUBERNETES_SERVICE_PORT", "")
	kubeconfigPath := filepath.Join(t.TempDir(), "config")
	t.Setenv("KUBECONFIG", kubeconfigPath)
	require.NoError(t, os.WriteFile(kubeconfigPath, []byte("invalid: ["), 0600))
	_, err := executorPluginKubernetesConfig()
	require.Error(t, err)
	require.NoError(t, os.WriteFile(kubeconfigPath, []byte(`apiVersion: v1
kind: Config
current-context: plugin
contexts:
- name: plugin
  context:
    cluster: cluster
    user: plugin
clusters:
- name: cluster
  cluster:
    server: https://kubernetes.example
users:
- name: plugin
  user:
    token: bootstrap-token
`), 0600))
	cfg, err := executorPluginKubernetesConfig()
	require.NoError(t, err)
	assert.Equal(t, "https://kubernetes.example", cfg.Host)
	assert.Equal(t, "bootstrap-token", cfg.BearerToken)
	assert.IsType(t, executorPluginWarningHandler{}, cfg.WarningHandlerWithContext)
	client, err := inClusterClientFactory().kubernetesClient(cfg)
	require.NoError(t, err)
	require.NotNil(t, client)
}
