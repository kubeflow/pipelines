// Copyright 2025 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package worker implements persistence workers that sync Kubernetes resources
// to the Kubeflow Pipelines database.
package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	workflowapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	corelisters "k8s.io/client-go/listers/core/v1"
)

const (
	// ArgoWorkflowLabelKey is the label Argo sets on pods to identify the parent workflow.
	ArgoWorkflowLabelKey = "workflows.argoproj.io/workflow"
)

// ImagePullFailureChecker checks workflow pods for image pull failures
// and terminates the workflow if the grace period has elapsed.
type ImagePullFailureChecker interface {
	// CheckAndTerminate inspects the pods of a running workflow and terminates
	// the workflow once a pod has been failing to pull an image for longer than
	// the grace period.
	CheckAndTerminate(ctx context.Context, namespace string, workflowName string) error
	// Forget drops any failure tracking state held for the workflow. Callers
	// should invoke it once a workflow reaches a final state or no longer exists
	// so the checker does not retain state for workflows it will never check again.
	Forget(namespace string, workflowName string)
}

// imagePullFailureChecker checks pods belonging to a workflow for image pull
// failures and terminates the workflow after a configurable grace period.
// It uses a pod lister backed by a shared informer to avoid direct API calls
// to the Kubernetes API server on every check.
//
// The grace period is measured from the moment the checker first observes the
// image pull failure on a pod, not from the pod's creation time, so time spent
// pending or initializing does not count against the pull. A pod that recovers
// (or is replaced) has its failure clock reset.
type imagePullFailureChecker struct {
	podLister       corelisters.PodLister
	executionClient util.ExecutionClient
	gracePeriod     time.Duration
	now             func() time.Time

	mu sync.Mutex
	// failureStart records when an image pull failure was first observed on a
	// pod, keyed by workflow (namespace/name) and then by pod UID.
	failureStart map[string]map[types.UID]time.Time
}

// NewImagePullFailureChecker creates a new checker. The podLister should be
// backed by a shared informer so that pod lookups are served from a local
// cache rather than making API calls to the Kubernetes API server.
func NewImagePullFailureChecker(
	podLister corelisters.PodLister,
	executionClient util.ExecutionClient,
	gracePeriod time.Duration,
) ImagePullFailureChecker {
	return &imagePullFailureChecker{
		podLister:       podLister,
		executionClient: executionClient,
		gracePeriod:     gracePeriod,
		now:             time.Now,
		failureStart:    make(map[string]map[types.UID]time.Time),
	}
}

// expiredImagePullFailure describes a pod whose image pull failure has
// outlasted the grace period.
type expiredImagePullFailure struct {
	podName     string
	failedImage string
	elapsed     time.Duration
}

// CheckAndTerminate lists pods for the given workflow and terminates the workflow
// if any pod has been stuck in ImagePullBackOff or ErrImagePull longer than the
// grace period (measured from when the failure was first observed).
func (c *imagePullFailureChecker) CheckAndTerminate(ctx context.Context, namespace string, workflowName string) error {
	selector, err := labels.Parse(fmt.Sprintf("%s=%s", ArgoWorkflowLabelKey, workflowName))
	if err != nil {
		return fmt.Errorf("failed to parse label selector for workflow %s/%s: %w", namespace, workflowName, err)
	}

	pods, err := c.podLister.Pods(namespace).List(selector)
	if err != nil {
		return fmt.Errorf("failed to list pods for workflow %s/%s: %w", namespace, workflowName, err)
	}

	expired := c.trackFailures(namespace, workflowName, pods)
	if expired == nil {
		return nil
	}

	log.Infof("Terminating workflow %s/%s: pod %s has image pull failure for %q (failing for %v exceeds grace period %v)",
		namespace, workflowName, expired.podName, expired.failedImage, expired.elapsed.Round(time.Second), c.gracePeriod)
	if err := c.terminateWorkflow(ctx, namespace, workflowName, expired.failedImage); err != nil {
		return err
	}
	c.Forget(namespace, workflowName)
	return nil
}

// Forget drops the failure tracking state for the given workflow.
func (c *imagePullFailureChecker) Forget(namespace string, workflowName string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.failureStart, workflowKey(namespace, workflowName))
}

// trackFailures updates the failure start times for the workflow's pods and
// returns the first pod whose failure has outlasted the grace period, or nil.
// Pods that no longer report a failure, or are no longer listed, have their
// tracking dropped so a recovered pod starts a fresh grace period next time.
func (c *imagePullFailureChecker) trackFailures(namespace, workflowName string, pods []*corev1.Pod) *expiredImagePullFailure {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := workflowKey(namespace, workflowName)
	previous := c.failureStart[key]
	current := make(map[types.UID]time.Time)
	now := c.now()

	var expired *expiredImagePullFailure
	for _, pod := range pods {
		if isPodTerminal(pod) {
			// A retained terminal pod (for example one killed by a task deadline
			// while an init container was still pulling) must not fail a
			// workflow whose retry is progressing.
			continue
		}
		failedImage := getImagePullFailure(pod)
		if failedImage == "" {
			continue
		}

		start, seen := previous[pod.UID]
		if !seen {
			start = now
		}
		current[pod.UID] = start

		elapsed := now.Sub(start)
		if elapsed < c.gracePeriod {
			log.Debugf("Pod %s/%s has image pull failure for %q (failing for %v), waiting for grace period (%v)",
				pod.Namespace, pod.Name, failedImage, elapsed.Round(time.Second), c.gracePeriod)
			continue
		}
		if expired == nil {
			expired = &expiredImagePullFailure{podName: pod.Name, failedImage: failedImage, elapsed: elapsed}
		}
	}

	if len(current) == 0 {
		delete(c.failureStart, key)
	} else {
		c.failureStart[key] = current
	}
	return expired
}

func workflowKey(namespace, workflowName string) string {
	return namespace + "/" + workflowName
}

// isPodTerminal reports whether the pod has finished running or is being deleted.
// Such pods can still carry a stale image pull failure in their container
// statuses, but they no longer block the workflow.
func isPodTerminal(pod *corev1.Pod) bool {
	if pod.DeletionTimestamp != nil {
		return true
	}
	return pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed
}

// getImagePullFailure checks if any container in the pod has an image pull failure.
// Returns the failed image name, or empty string if no failure is found.
func getImagePullFailure(pod *corev1.Pod) string {
	for _, status := range pod.Status.InitContainerStatuses {
		if image := imagePullFailureFromStatus(status); image != "" {
			return image
		}
	}
	for _, status := range pod.Status.ContainerStatuses {
		if image := imagePullFailureFromStatus(status); image != "" {
			return image
		}
	}
	return ""
}

// imagePullFailureFromStatus returns the image name if the container is in
// ImagePullBackOff or ErrImagePull state, empty string otherwise.
func imagePullFailureFromStatus(status corev1.ContainerStatus) string {
	if status.State.Waiting != nil {
		reason := status.State.Waiting.Reason
		if reason == "ImagePullBackOff" || reason == "ErrImagePull" {
			return status.Image
		}
	}
	return ""
}

// terminateWorkflow terminates an Argo workflow and annotates it with the
// failing image so the reason is visible to users.
//
// The patch sets activeDeadlineSeconds to 0, which is how KFP marks a run as
// terminated, and additionally sets the Terminate shutdown strategy. Argo
// exempts exit-handler pods from the workflow deadline, so without the shutdown
// strategy an exit handler whose image cannot be pulled would keep the
// workflow running forever.
func (c *imagePullFailureChecker) terminateWorkflow(ctx context.Context, namespace, workflowName, failedImage string) error {
	if c.executionClient == nil {
		return fmt.Errorf("execution client not configured, cannot terminate workflow %s/%s", namespace, workflowName)
	}

	terminatePatch, ok := util.GetTerminatePatch(util.CurrentExecutionType()).(map[string]interface{})
	if !ok {
		return fmt.Errorf("unsupported execution type for termination")
	}

	// Build a merged patch that terminates the workflow and annotates it with the
	// image pull failure reason so users can see it in the workflow manifest.
	patch := map[string]interface{}{
		"metadata": map[string]interface{}{
			"annotations": map[string]interface{}{
				"pipelines.kubeflow.org/termination-reason": "ImagePullFailure",
				"pipelines.kubeflow.org/failed-image":       failedImage,
			},
		},
	}
	for k, v := range terminatePatch {
		patch[k] = v
	}
	spec, ok := patch["spec"].(map[string]interface{})
	if !ok {
		spec = map[string]interface{}{}
		patch["spec"] = spec
	}
	spec["shutdown"] = string(workflowapi.ShutdownStrategyTerminate)

	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("failed to marshal termination patch: %w", err)
	}

	_, err = c.executionClient.Execution(namespace).Patch(
		ctx, workflowName, types.MergePatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("failed to patch workflow %s/%s: %w", namespace, workflowName, err)
	}

	log.Infof("Successfully terminated workflow %s/%s due to image pull failure", namespace, workflowName)
	return nil
}
