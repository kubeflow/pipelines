// Copyright 2025 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/config"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

var (
	workflowTaskResultGVR = schema.GroupVersionResource{
		Group:    "argoproj.io",
		Version:  "v1alpha1",
		Resource: "workflowtaskresults",
	}
	workflowGVR = schema.GroupVersionResource{
		Group:    "argoproj.io",
		Version:  "v1alpha1",
		Resource: "workflows",
	}
)

// terminateOnCacheHit signals Argo that this node succeeded and then self-deletes
// the pod. This prevents the main container image (potentially very large) from
// being pulled and started when the driver init container has already confirmed a
// cache hit and published the cached outputs to MLMD.
//
// The mechanism:
//  1. Fetch the Argo Workflow to get its UID (needed for the owner reference on
//     the WorkflowTaskResult, so it gets garbage-collected with the workflow).
//  2. A WorkflowTaskResult CRD is written with phase "Succeeded" and an owner
//     reference to the workflow. Argo's workflow controller watches these resources
//     and marks the workflow node as Succeeded.
//  3. We poll the Argo Workflow resource until the node is actually marked
//     Succeeded. This ensures Argo has processed the result before we delete the
//     pod — preventing a spurious pod retry.
//  4. Once Argo has confirmed the node as Succeeded, the pod self-deletes with
//     zero grace period, aborting any in-progress image pull for the main container
//     and preventing the kfp-launcher init container from starting.
//  5. The process blocks forever after issuing the delete, so that the init
//     container does not exit (which could briefly start the next container before
//     the kubelet tears down the pod).
func terminateOnCacheHit(ctx context.Context) error {
	podName, err := config.InPodName()
	if err != nil {
		return fmt.Errorf("failed to get pod name: %w", err)
	}
	namespace, err := config.InPodNamespace()
	if err != nil {
		return fmt.Errorf("failed to get namespace: %w", err)
	}

	restConfig, err := util.GetKubernetesConfig()
	if err != nil {
		return fmt.Errorf("failed to get kubernetes config: %w", err)
	}
	coreClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}
	dynClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("failed to create dynamic kubernetes client: %w", err)
	}

	// Read the Argo workflow name from pod labels and node ID from pod
	// annotations (both set by Argo at pod creation time).
	pod, err := coreClient.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get pod %s: %w", podName, err)
	}
	workflowName := pod.Labels["workflows.argoproj.io/workflow"]
	if workflowName == "" {
		return fmt.Errorf("pod %s has no workflows.argoproj.io/workflow label", podName)
	}
	nodeID := pod.Annotations["workflows.argoproj.io/node-id"]
	if nodeID == "" {
		return fmt.Errorf("pod %s has no workflows.argoproj.io/node-id annotation", podName)
	}

	// Fetch the workflow to get its UID for the owner reference.
	wfClient := dynClient.Resource(workflowGVR).Namespace(namespace)
	wf, err := wfClient.Get(ctx, workflowName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get workflow %s: %w", workflowName, err)
	}
	workflowUID := wf.GetUID()

	if err := upsertWorkflowTaskResult(ctx, dynClient, podName, namespace, workflowName, workflowUID); err != nil {
		return err
	}
	glog.Infof("Wrote WorkflowTaskResult for pod %s (node %s in workflow %s); polling for Argo to mark node Succeeded.", podName, nodeID, workflowName)

	if err := waitForArgoNodeSucceeded(ctx, dynClient, namespace, workflowName, nodeID); err != nil {
		return fmt.Errorf("timed out waiting for Argo to mark node %s as Succeeded: %w", nodeID, err)
	}
	glog.Infof("Argo confirmed node %s Succeeded; self-deleting pod to abort main container image pull.", nodeID)

	// Force-delete the pod (grace period 0) to immediately abort the main
	// container image pull and prevent the kfp-launcher init container from starting.
	gracePeriod := int64(0)
	if err := coreClient.CoreV1().Pods(namespace).Delete(ctx, podName, metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriod,
	}); err != nil {
		return fmt.Errorf("failed to delete pod %s: %w", podName, err)
	}
	glog.Infof("Deleted pod %s on cache hit; blocking until SIGKILL.", podName)

	// Block forever. The kubelet will SIGKILL this process when it processes the
	// pod deletion. We must not return, because if this init container exits
	// successfully, Kubernetes would start the next init container / main
	// container in the brief window before the kubelet tears down the pod.
	select {}
}

// upsertWorkflowTaskResult creates the WorkflowTaskResult for this pod with
// phase Succeeded. If a result already exists (e.g., from a previous incarnation
// of this pod), it is patched in place. An owner reference to the workflow ensures
// the result is garbage-collected when the workflow is deleted.
func upsertWorkflowTaskResult(ctx context.Context, dynClient dynamic.Interface, podName, namespace, workflowName string, workflowUID types.UID) error {
	isController := true
	result := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "argoproj.io/v1alpha1",
			"kind":       "WorkflowTaskResult",
			"metadata": map[string]interface{}{
				"name":      podName,
				"namespace": namespace,
				"labels": map[string]interface{}{
					"workflows.argoproj.io/workflow": workflowName,
				},
				"ownerReferences": []interface{}{
					map[string]interface{}{
						"apiVersion": "argoproj.io/v1alpha1",
						"kind":       "Workflow",
						"name":       workflowName,
						"uid":        string(workflowUID),
						"controller": isController,
					},
				},
			},
			"phase":   "Succeeded",
			"message": "Cache hit: outputs reused from a previous execution",
			"outputs": map[string]interface{}{
				"exitCode": "0",
			},
		},
	}
	wtrClient := dynClient.Resource(workflowTaskResultGVR).Namespace(namespace)
	_, err := wtrClient.Create(ctx, result, metav1.CreateOptions{})
	if err == nil {
		return nil
	}
	if !k8serrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create WorkflowTaskResult for pod %s: %w", podName, err)
	}
	// Resource already exists — patch it with the Succeeded phase.
	patch := map[string]interface{}{
		"phase":   "Succeeded",
		"message": "Cache hit: outputs reused from a previous execution",
		"outputs": map[string]interface{}{
			"exitCode": "0",
		},
	}
	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("failed to marshal WorkflowTaskResult patch: %w", err)
	}
	if _, err := wtrClient.Patch(ctx, podName, types.MergePatchType, patchBytes, metav1.PatchOptions{}); err != nil {
		return fmt.Errorf("failed to patch WorkflowTaskResult for pod %s: %w", podName, err)
	}
	return nil
}

// waitForArgoNodeSucceeded polls the Argo Workflow until the given node shows
// phase "Succeeded". This confirms that Argo has processed the WorkflowTaskResult
// and will not retry the pod after deletion. Returns an error if the node does not
// reach Succeeded within the timeout.
func waitForArgoNodeSucceeded(ctx context.Context, dynClient dynamic.Interface, namespace, workflowName, nodeID string) error {
	const (
		pollInterval = 500 * time.Millisecond
		maxAttempts  = 40 // 20 seconds total
	)
	wfClient := dynClient.Resource(workflowGVR).Namespace(namespace)
	for attempt := 0; attempt < maxAttempts; attempt++ {
		time.Sleep(pollInterval)
		wf, err := wfClient.Get(ctx, workflowName, metav1.GetOptions{})
		if err != nil {
			glog.Warningf("Failed to get workflow %s (attempt %d): %v", workflowName, attempt, err)
			continue
		}
		phase, _, _ := unstructured.NestedString(wf.Object, "status", "nodes", nodeID, "phase")
		if phase == "Succeeded" {
			return nil
		}
		glog.V(4).Infof("Node %s phase=%q (attempt %d/%d); waiting...", nodeID, phase, attempt+1, maxAttempts)
	}
	return fmt.Errorf("node %s did not reach Succeeded within %v", nodeID, time.Duration(maxAttempts)*pollInterval)
}
