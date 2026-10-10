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
	"crypto/subtle"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/driver/driverapi"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient"
	"github.com/kubeflow/pipelines/backend/src/v2/client_manager"
	"github.com/kubeflow/pipelines/backend/src/v2/config"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	// DefaultTokenPath is the bearer-token mount provided by Argo.
	DefaultTokenPath                 = "/var/run/argo/token"
	podNamespacePath                 = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
	legacyServiceAccountTokenWarning = "Use tokens from the TokenRequest API or manually created secret-based tokens instead of auto-generated secret-based tokens."
)

type executorPluginWarningHandler struct {
	delegate rest.WarningHandlerWithContext
}

func (h executorPluginWarningHandler) HandleWarningHeaderWithContext(ctx context.Context, code int, agent, message string) {
	// Argo Workflows v4.1.2 can give an executor plugin a separate Kubernetes
	// identity only by mounting its legacy ServiceAccount token Secret. The API
	// server returns this warning for every request made with that token, so drop
	// only the expected warning and preserve every other Kubernetes warning.
	if message == legacyServiceAccountTokenWarning {
		return
	}
	h.delegate.HandleWarningHeaderWithContext(ctx, code, agent, message)
}

func executorPluginKubernetesConfig() (*rest.Config, error) {
	restConfig, err := util.GetKubernetesConfig()
	if err != nil {
		return nil, err
	}
	restConfig.WarningHandlerWithContext = executorPluginWarningHandler{delegate: rest.WarningLogger{}}
	return restConfig, nil
}

func authenticatedPluginHandler(tokenPath string, next http.Handler) (http.Handler, error) {
	token, err := os.ReadFile(tokenPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read executor plugin authentication token: check the Argo token mount: %w", err)
	}
	expectedToken := strings.TrimSpace(string(token))
	if expectedToken == "" {
		return nil, fmt.Errorf("executor plugin authentication token is empty: check the Argo token mount")
	}
	expectedHeader := []byte("Bearer " + expectedToken)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), expectedHeader) != 1 {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	}), nil
}

type driverClientManager interface {
	client_manager.ClientManagerInterface
	Close() error
}

type invocationClients struct {
	manager          driverClientManager
	pod              *corev1.Pod
	workflowMetadata workflowMetadataGetter
}

// clientFactory isolates in-cluster identity and client construction, without
// changing process-wide state for concurrent invocations.
type clientFactory struct {
	identity         func() (namespace, podName string, err error)
	kubernetesConfig func() (*rest.Config, error)
	kubernetesClient func(*rest.Config) (kubernetes.Interface, error)
	manager          func(*client_manager.Options) (driverClientManager, error)
}

func inClusterClientFactory() clientFactory {
	return clientFactory{
		identity:         func() (string, string, error) { return executorPluginIdentity(podNamespacePath) },
		kubernetesConfig: executorPluginKubernetesConfig,
		kubernetesClient: func(cfg *rest.Config) (kubernetes.Interface, error) { return kubernetes.NewForConfig(cfg) },
		manager: func(opts *client_manager.Options) (driverClientManager, error) {
			return client_manager.NewClientManager(opts)
		},
	}
}

func executorPluginIdentity(namespacePath string) (namespace, podName string, err error) {
	namespaceBytes, err := os.ReadFile(namespacePath)
	if err != nil {
		return "", "", fmt.Errorf("cannot read executor plugin namespace: check the service account mount: %w", err)
	}
	namespace = strings.TrimSpace(string(namespaceBytes))
	if namespace == "" {
		return "", "", fmt.Errorf("executor plugin namespace is empty: check the service account mount")
	}
	podName, err = config.InPodName()
	if err != nil {
		return "", "", fmt.Errorf("cannot determine executor plugin Pod name: check the KFP_POD_NAME environment variable: %w", err)
	}
	return namespace, podName, nil
}

func (f clientFactory) newDriverClientManager(ctx context.Context, args driverapi.DriverPluginArgs, caCertPath string) (*invocationClients, error) {
	podNamespace, podName, err := f.identity()
	if err != nil {
		return nil, err
	}
	restConfig, err := f.kubernetesConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load executor plugin Kubernetes config: check the service account mount: %w", err)
	}
	k8sClient, err := f.kubernetesClient(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize executor plugin Kubernetes client: %w", err)
	}
	apiConfig, pod, err := driverAPIClientConfig(ctx, k8sClient, podNamespace, podName, args)
	if err != nil {
		return nil, err
	}
	manager, err := f.manager(&client_manager.Options{
		MLPipelineTLSEnabled: args.MlPipelineTLSEnabled,
		CaCertPath:           caCertPath,
		APIClientConfig:      apiConfig,
		K8sClient:            k8sClient,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize driver API client: check the KFP endpoint and TLS settings: %w", err)
	}
	return &invocationClients{
		manager: manager, pod: pod,
		workflowMetadata: func(ctx context.Context, namespace, workflowName string) (*metav1.ObjectMeta, error) {
			return getCurrentWorkflowMetadata(ctx, restConfig, namespace, workflowName)
		},
	}, nil
}

func validatedAgentPod(ctx context.Context, k8sClient kubernetes.Interface, podNamespace, podName string, args driverapi.DriverPluginArgs) (*corev1.Pod, error) {
	if podNamespace == "" || podName == "" || args.Namespace != podNamespace {
		return nil, fmt.Errorf("driver request namespace does not match its agent Pod: use the workflow namespace")
	}
	pod, err := k8sClient.CoreV1().Pods(podNamespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get executor plugin Pod: check the plugin service account's pods/get permission: %w", err)
	}
	// Request fields select work, but cannot select the identity used to mint tokens.
	if args.RunID == "" || pod.Labels[util.LabelKeyWorkflowRunId] != args.RunID {
		return nil, fmt.Errorf("driver request run ID does not match its agent Pod: use the Pod's pipeline/runid")
	}
	owner := metav1.GetControllerOf(pod)
	if owner == nil || owner.APIVersion != "argoproj.io/v1alpha1" || owner.Kind != "Workflow" || owner.Name != args.RunName {
		return nil, fmt.Errorf("driver request workflow does not own its agent Pod: use the owning workflow name")
	}
	return pod, nil
}

func driverAPIClientConfig(ctx context.Context, k8sClient kubernetes.Interface, podNamespace, podName string, args driverapi.DriverPluginArgs) (*apiclient.Config, *corev1.Pod, error) {
	pod, err := validatedAgentPod(ctx, k8sClient, podNamespace, podName, args)
	if err != nil {
		return nil, nil, err
	}
	audience := args.KFPTokenAudience
	if audience == "" {
		audience = common.DefaultTokenReviewAudience + common.TokenAudienceRunPrefix + args.RunID
	}
	runSuffix := common.TokenAudienceRunPrefix + args.RunID
	if !strings.HasSuffix(audience, runSuffix) || audience == runSuffix {
		return nil, nil, fmt.Errorf("KFP token audience is not bound to the agent Pod's run: use <KFP audience>/runs/<run ID>")
	}
	source, err := apiclient.NewServiceAccountTokenSource(k8sClient.CoreV1().ServiceAccounts(podNamespace),
		pod.Spec.ServiceAccountName, audience, pod.Name, string(pod.UID))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to configure run-bound token source: check the agent Pod service account and UID: %w", err)
	}
	apiConfig := apiClientConfig(args)
	apiConfig.TokenSource = source
	return apiConfig, pod, nil
}

// apiClientConfig builds settings for this invocation without changing process env.
func apiClientConfig(args driverapi.DriverPluginArgs) *apiclient.Config {
	cfg := apiclient.FromEnvWithEndpointOverride(args.MlPipelineServerAddress, args.MlPipelineServerPort)
	if args.MlPipelineGRPCBackoffBaseDelay != "" {
		cfg.BackoffBaseDelay = args.MlPipelineGRPCBackoffBaseDelay
	}
	if args.MlPipelineGRPCBackoffMultiplier != "" {
		cfg.BackoffMultiplier = args.MlPipelineGRPCBackoffMultiplier
	}
	if args.MlPipelineGRPCBackoffJitter != "" {
		cfg.BackoffJitter = args.MlPipelineGRPCBackoffJitter
	}
	if args.MlPipelineGRPCBackoffMaxDelay != "" {
		cfg.BackoffMaxDelay = args.MlPipelineGRPCBackoffMaxDelay
	}
	if args.MlPipelineGRPCMinConnectTimeout != "" {
		cfg.MinConnectTimeout = args.MlPipelineGRPCMinConnectTimeout
	}
	return cfg
}
