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
	"sync"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/driver/driverapi"
	"github.com/kubeflow/pipelines/backend/src/driver/driverinputs"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient"
	"github.com/kubeflow/pipelines/backend/src/v2/client_manager"
	"github.com/kubeflow/pipelines/backend/src/v2/config"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

const (
	// DefaultTokenPath is the bearer-token mount provided by Argo.
	DefaultTokenPath                 = "/var/run/argo/token"
	podNamespacePath                 = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
	legacyServiceAccountTokenWarning = "Use tokens from the TokenRequest API or manually created secret-based tokens instead of auto-generated secret-based tokens."
	executorPluginClientQPS          = 100
	executorPluginClientBurst        = 200
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
	// One plugin process drives every task of a workflow, and each invocation
	// makes several API calls. client-go's 5 QPS default serialized 120
	// concurrent drivers into 20-second waits per request in scale tests.
	restConfig.QPS = executorPluginClientQPS
	restConfig.Burst = executorPluginClientBurst
	return restConfig, nil
}

func authenticatedPluginHandler(tokenPath string, next http.Handler) (http.Handler, error) {
	token, err := os.ReadFile(tokenPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read executor plugin authentication token; check the Argo token mount: %w", err)
	}
	expectedToken := strings.TrimSpace(string(token))
	if expectedToken == "" {
		return nil, fmt.Errorf("executor plugin authentication token is empty; check the Argo token mount")
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
	workflowMetadata driverinputs.WorkflowMetadataGetter
}

// clientFactory builds per-invocation KFP API clients on top of the shared
// agent identity, without changing process-wide state.
type clientFactory struct {
	agent   agentLoader
	manager func(*client_manager.Options) (driverClientManager, error)
	// tokens shares one run-bound token source across invocations; nil mints
	// a fresh source, and therefore a fresh TokenRequest, per invocation.
	tokens *tokenSourceCache
}

func inClusterClientFactory(agent agentLoader) clientFactory {
	return clientFactory{
		agent: agent,
		manager: func(opts *client_manager.Options) (driverClientManager, error) {
			return client_manager.NewClientManager(opts)
		},
		tokens: &tokenSourceCache{sources: map[string]apiclient.TokenSource{}},
	}
}

// tokenSourceCache keys token sources by service account, audience and Pod
// UID. Every task of a workflow shares all three, so one TokenRequest serves
// the whole agent process and the source refreshes itself before expiry.
type tokenSourceCache struct {
	mu      sync.Mutex
	sources map[string]apiclient.TokenSource
}

func (c *tokenSourceCache) get(key string, create func() (apiclient.TokenSource, error)) (apiclient.TokenSource, error) {
	if c == nil {
		return create()
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if source, ok := c.sources[key]; ok {
		return source, nil
	}
	source, err := create()
	if err != nil {
		return nil, err
	}
	c.sources[key] = source
	return source, nil
}

func executorPluginIdentity(namespacePath string) (namespace, podName string, err error) {
	namespaceBytes, err := os.ReadFile(namespacePath)
	if err != nil {
		return "", "", fmt.Errorf("cannot read executor plugin namespace; check the service account mount: %w", err)
	}
	namespace = strings.TrimSpace(string(namespaceBytes))
	if namespace == "" {
		return "", "", fmt.Errorf("executor plugin namespace is empty; check the service account mount")
	}
	podName, err = config.InPodName()
	if err != nil {
		return "", "", fmt.Errorf("cannot determine executor plugin Pod name; check the KFP_POD_NAME environment variable: %w", err)
	}
	return namespace, podName, nil
}

func (f clientFactory) newDriverClientManager(ctx context.Context, args driverapi.DriverPluginArgs, caCertPath string) (*invocationClients, error) {
	retry, err := parseDriverRetry(args)
	if err != nil {
		return nil, err
	}
	agent, err := f.agent(ctx)
	if err != nil {
		return nil, err
	}
	if err := agent.validateRequest(args); err != nil {
		return nil, err
	}
	apiConfig, err := driverAPIClientConfig(agent, args, f.tokens)
	if err != nil {
		return nil, err
	}
	manager, err := f.manager(&client_manager.Options{
		MLPipelineTLSEnabled:  args.MlPipelineTLSEnabled,
		CaCertPath:            caCertPath,
		APIClientConfig:       apiConfig,
		K8sClient:             agent.kube,
		DriverRetryGeneration: &retry.generation,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize driver API client; check the KFP endpoint and TLS settings: %w", err)
	}
	return &invocationClients{
		manager: manager, pod: agent.pod,
		workflowMetadata: func(ctx context.Context, namespace, workflowName string) (*metav1.ObjectMeta, error) {
			return driverinputs.CurrentWorkflowMetadata(ctx, agent.argo, namespace, workflowName)
		},
	}, nil
}

// driverAPIClientConfig binds the KFP API token to the agent Pod's run.
func driverAPIClientConfig(agent *agentClients, args driverapi.DriverPluginArgs, tokens *tokenSourceCache) (*apiclient.Config, error) {
	audience := args.KFPTokenAudience
	if audience == "" {
		audience = common.DefaultTokenReviewAudience + common.TokenAudienceRunPrefix + args.RunID
	}
	runSuffix := common.TokenAudienceRunPrefix + args.RunID
	if !strings.HasSuffix(audience, runSuffix) || audience == runSuffix {
		return nil, fmt.Errorf("KFP token audience %q is not bound to the agent Pod's run; use <KFP audience>/runs/<run ID>", audience)
	}
	pod := agent.pod
	source, err := tokens.get(pod.Spec.ServiceAccountName+"\x00"+audience+"\x00"+string(pod.UID), func() (apiclient.TokenSource, error) {
		return apiclient.NewServiceAccountTokenSource(agent.kube.CoreV1().ServiceAccounts(pod.Namespace),
			pod.Spec.ServiceAccountName, audience, pod.Name, string(pod.UID))
	})
	if err != nil {
		return nil, fmt.Errorf("failed to configure run-bound token source; check the agent Pod service account and UID: %w", err)
	}
	apiConfig := apiClientConfig(args)
	apiConfig.TokenSource = source
	return apiConfig, nil
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
