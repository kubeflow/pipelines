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
	"fmt"
	"sync"

	workflowapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	workflowclient "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned"
	workflowinformers "github.com/argoproj/argo-workflows/v4/pkg/client/informers/externalversions"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/driver/driverapi"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// agentClients is the executor plugin's own identity: the Argo agent Pod it
// runs in, the Workflow controlling that Pod, and clients using the plugin's
// ServiceAccount. The Pod's name, UID, run label and owner never change, so
// they are read once per process and later requests validate against the copy.
type agentClients struct {
	kube     kubernetes.Interface
	argo     workflowclient.Interface
	pod      *corev1.Pod
	workflow metav1.OwnerReference
}

type agentLoader func(context.Context) (*agentClients, error)

// cachedAgent retries bootstrap until it succeeds, then reuses the result.
func cachedAgent(load agentLoader) agentLoader {
	var mu sync.Mutex
	var cached *agentClients
	return func(ctx context.Context) (*agentClients, error) {
		mu.Lock()
		defer mu.Unlock()
		if cached == nil {
			clients, err := load(ctx)
			if err != nil {
				return nil, err
			}
			cached = clients
		}
		return cached, nil
	}
}

func inClusterAgent(ctx context.Context) (*agentClients, error) {
	namespace, podName, err := executorPluginIdentity(podNamespacePath)
	if err != nil {
		return nil, err
	}
	restConfig, err := executorPluginKubernetesConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load executor plugin Kubernetes config; check the service account mount: %w", err)
	}
	kube, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize executor plugin Kubernetes client; check the service account mount: %w", err)
	}
	argo, err := workflowclient.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize executor plugin Argo client; check the service account mount: %w", err)
	}
	return loadAgent(ctx, kube, argo, namespace, podName)
}

func loadAgent(ctx context.Context, kube kubernetes.Interface, argo workflowclient.Interface, namespace, podName string) (*agentClients, error) {
	if namespace == "" || podName == "" {
		return nil, fmt.Errorf("executor plugin identity is incomplete; set KFP_POD_NAME and mount the service account namespace")
	}
	pod, err := kube.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get executor plugin Pod %s/%s; grant the plugin service account pods/get: %w", namespace, podName, err)
	}
	owner := metav1.GetControllerOf(pod)
	if owner == nil || owner.APIVersion != "argoproj.io/v1alpha1" || owner.Kind != "Workflow" || owner.UID == "" {
		return nil, fmt.Errorf("executor plugin Pod %s/%s is not controlled by an Argo Workflow; run the driver plugin only as an Argo agent sidecar", namespace, podName)
	}
	return &agentClients{kube: kube, argo: argo, pod: pod, workflow: *owner}, nil
}

// validateRequest ensures a request only selects work in this agent's Workflow.
// Request fields select work, but cannot select the identity used to mint tokens.
func (a *agentClients) validateRequest(args driverapi.DriverPluginArgs) error {
	if args.Namespace != a.pod.Namespace {
		return fmt.Errorf("driver request namespace %q does not match the agent Pod namespace %q; use the workflow namespace", args.Namespace, a.pod.Namespace)
	}
	if args.RunID == "" || a.pod.Labels[util.LabelKeyWorkflowRunId] != args.RunID {
		return fmt.Errorf("driver request run ID %q does not match the agent Pod's %s label; use the owning run", args.RunID, util.LabelKeyWorkflowRunId)
	}
	if args.RunName != a.workflow.Name {
		return fmt.Errorf("driver request workflow %q does not own the agent Pod (owner %q); use the owning Workflow name", args.RunName, a.workflow.Name)
	}
	return nil
}

// tasksetSource returns the agent Workflow's current WorkflowTaskSet.
type tasksetSource func(context.Context, *agentClients) (*workflowapi.WorkflowTaskSet, error)

// watchedTaskset serves the agent Workflow's taskset from an informer cache
// bound to the service lifetime, so Argo's one-second polls do not each read
// the API server. Waiting for the initial sync is bounded by the request ctx.
func watchedTaskset(serviceCtx context.Context) tasksetSource {
	var once sync.Once
	var informer cache.SharedIndexInformer
	return func(ctx context.Context, agent *agentClients) (*workflowapi.WorkflowTaskSet, error) {
		namespace, name := agent.pod.Namespace, agent.workflow.Name
		once.Do(func() {
			factory := workflowinformers.NewSharedInformerFactoryWithOptions(agent.argo, 0,
				workflowinformers.WithNamespace(namespace),
				workflowinformers.WithTweakListOptions(func(options *metav1.ListOptions) {
					options.FieldSelector = fields.OneTermEqualSelector("metadata.name", name).String()
				}))
			informer = factory.Argoproj().V1alpha1().WorkflowTaskSets().Informer()
			factory.Start(serviceCtx.Done())
		})
		if !cache.WaitForCacheSync(ctx.Done(), informer.HasSynced) {
			return nil, fmt.Errorf("WorkflowTaskSet cache for %s/%s is not synced; grant the plugin service account workflowtasksets list and watch", namespace, name)
		}
		object, exists, err := informer.GetStore().GetByKey(namespace + "/" + name)
		if err != nil {
			return nil, fmt.Errorf("failed to read cached WorkflowTaskSet %s/%s: %w", namespace, name, err)
		}
		if !exists {
			return nil, fmt.Errorf("WorkflowTaskSet %s/%s does not exist yet; Argo creates it when the first plugin task starts, retry", namespace, name)
		}
		return object.(*workflowapi.WorkflowTaskSet), nil
	}
}
