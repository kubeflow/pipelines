// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package executorplugin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	workflowapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	workflowclient "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned"
	"github.com/kubeflow/pipelines/backend/src/driver/driverapi"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type attemptResolver func(context.Context, driverapi.DriverPluginArgs) (string, error)

func inClusterAttemptResolver() attemptResolver {
	var mu sync.Mutex
	var resolver attemptResolver
	return func(ctx context.Context, args driverapi.DriverPluginArgs) (string, error) {
		mu.Lock()
		if resolver == nil {
			// Retry bootstrap failures; only reuse successfully configured clients.
			namespace, pod, err := executorPluginIdentity(podNamespacePath)
			if err != nil {
				mu.Unlock()
				return "", err
			}
			cfg, err := executorPluginKubernetesConfig()
			if err != nil {
				mu.Unlock()
				return "", err
			}
			kube, err := kubernetes.NewForConfig(cfg)
			if err != nil {
				mu.Unlock()
				return "", err
			}
			argo, err := workflowclient.NewForConfig(cfg)
			if err != nil {
				mu.Unlock()
				return "", err
			}
			resolver = tasksetAttemptResolver(kube, argo, namespace, pod)
		}
		resolve := resolver
		mu.Unlock()
		return resolve(ctx, args)
	}
}

func tasksetAttemptResolver(kube kubernetes.Interface, argo workflowclient.Interface, namespace, podName string) attemptResolver {
	return func(ctx context.Context, args driverapi.DriverPluginArgs) (string, error) {
		pod, err := validatedAgentPod(ctx, kube, namespace, podName, args)
		if err != nil {
			return "", err
		}
		owner := metav1.GetControllerOf(pod)
		if owner.UID == "" {
			return "", fmt.Errorf("agent Workflow owner UID is empty")
		}
		taskset, err := argo.ArgoprojV1alpha1().WorkflowTaskSets(namespace).Get(ctx, args.RunName, metav1.GetOptions{})
		if err != nil {
			return "", err
		}
		if taskset.Namespace != namespace || taskset.Name != owner.Name {
			return "", fmt.Errorf("taskset identity does not match agent Workflow")
		}
		owned := false
		for _, ref := range taskset.OwnerReferences {
			if ref.APIVersion == owner.APIVersion && ref.Kind == owner.Kind && ref.Name == owner.Name && ref.UID == owner.UID {
				owned = true
			}
		}
		if !owned {
			return "", fmt.Errorf("taskset owner does not match agent Workflow UID")
		}
		var match string
		for id, tmpl := range taskset.Spec.Tasks {
			if tmpl.Plugin == nil {
				continue
			}
			encoded, err := json.Marshal(map[string]any{"template": tmpl})
			if err != nil {
				return "", err
			}
			request, err := http.NewRequestWithContext(ctx, http.MethodPost, "/", strings.NewReader(string(encoded)))
			if err != nil {
				return "", err
			}
			candidate, err := parseDriverRequestArgs(request)
			if err != nil {
				return "", fmt.Errorf("invalid taskset driver template: %w", err)
			}
			if candidate == nil || argsDigest(*candidate) != argsDigest(args) {
				continue
			}
			phase := taskset.Status.Nodes[id].Phase
			if phase == workflowapi.NodeSucceeded || phase == workflowapi.NodeFailed || phase == workflowapi.NodeError || phase == workflowapi.NodeSkipped || phase == workflowapi.NodeOmitted {
				continue
			}
			if match != "" || id == "" {
				return "", fmt.Errorf("ambiguous active driver taskset match")
			}
			match = id
		}
		if match == "" {
			return "", fmt.Errorf("no active driver taskset match")
		}
		return string(owner.UID) + "/" + match, nil
	}
}
