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
	"strings"
	"sync"
	"time"

	workflowregister "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow"
	workflowapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	argocommon "github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
	// CheckAndTerminate inspects the pods of the running workflow identified by
	// the given metadata and terminates the workflow once a pod has been failing
	// to pull an image for longer than the grace period. Only pods whose
	// controller owner reference matches the workflow name and UID are
	// considered, since pod labels are user-controlled. The termination patch
	// is conditioned on the workflow UID and resource version the decision was
	// made against, so a run that was retried or recreated in the meantime is
	// left alone. Failure tracking is scoped to the workflow UID and its
	// retry-generation annotation, so a retried attempt never inherits the
	// grace period consumed by the attempt it replaces.
	CheckAndTerminate(ctx context.Context, workflow *metav1.ObjectMeta) error
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
	// failureStart records when an image pull failure was first observed on
	// each pod, keyed by workflow (namespace/name). The entry also records the
	// workflow attempt it was observed for so a retry starts a fresh clock.
	failureStart map[string]*trackedWorkflow
}

// trackedWorkflow holds the failure tracking state for one attempt of a
// workflow. KFP retries reuse the workflow name, either by updating the
// object in place (same UID, new retry-generation annotation) or by
// recreating it (new UID). In both cases the previous attempt's failing pods
// may still be present in the pod informer cache for a while, so the state is
// discarded whenever the attempt identity changes.
type trackedWorkflow struct {
	uid             types.UID
	retryGeneration string
	// pods maps each failing pod's UID to when its failure was first observed.
	pods map[types.UID]time.Time
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
		failureStart:    make(map[string]*trackedWorkflow),
	}
}

// expiredImagePullFailure describes a pod whose image pull failure has
// outlasted the grace period.
type expiredImagePullFailure struct {
	podName     string
	failedImage string
	elapsed     time.Duration
	// exitHandler is true when the pod belongs to an exit handler. Such pods
	// are exempt from the workflow deadline, so terminating the workflow
	// requires the Terminate shutdown strategy.
	exitHandler bool
}

// CheckAndTerminate lists pods for the given workflow and terminates the workflow
// if any pod has been stuck in ImagePullBackOff or ErrImagePull longer than the
// grace period (measured from when the failure was first observed).
func (c *imagePullFailureChecker) CheckAndTerminate(ctx context.Context, workflow *metav1.ObjectMeta) error {
	if workflow == nil {
		return fmt.Errorf("workflow metadata is required to check for image pull failures")
	}
	namespace, workflowName := workflow.Namespace, workflow.Name
	selector, err := labels.Parse(fmt.Sprintf("%s=%s", ArgoWorkflowLabelKey, workflowName))
	if err != nil {
		return fmt.Errorf("failed to parse label selector for workflow %s/%s: %w", namespace, workflowName, err)
	}

	pods, err := c.podLister.Pods(namespace).List(selector)
	if err != nil {
		return fmt.Errorf("failed to list pods for workflow %s/%s: %w", namespace, workflowName, err)
	}

	expired := c.trackFailures(workflow, pods)
	if expired == nil {
		return nil
	}

	log.Infof("Terminating workflow %s/%s: pod %s has image pull failure for %q (failing for %v exceeds grace period %v)",
		namespace, workflowName, expired.podName, expired.failedImage, expired.elapsed.Round(time.Second), c.gracePeriod)
	terminated, err := c.terminateWorkflow(ctx, workflow, expired.failedImage, expired.exitHandler)
	if err != nil {
		return err
	}
	if terminated {
		c.Forget(namespace, workflowName)
	}
	return nil
}

// Forget drops the failure tracking state for the given workflow.
func (c *imagePullFailureChecker) Forget(namespace string, workflowName string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.failureStart, workflowKey(namespace, workflowName))
}

// trackFailures updates the failure start times for the workflow's pods and
// returns a pod whose failure has outlasted the grace period, or nil. When
// several pods have expired, an exit-handler pod is preferred because it
// needs the stronger termination mechanism.
// Pods that no longer report a failure, or are no longer listed, have their
// tracking dropped so a recovered pod starts a fresh grace period next time.
// State recorded for a different attempt of the workflow (another UID or
// retry generation) is discarded, so pods left over from the previous attempt
// cannot fail the new one with an already expired clock.
func (c *imagePullFailureChecker) trackFailures(workflow *metav1.ObjectMeta, pods []*corev1.Pod) *expiredImagePullFailure {
	c.mu.Lock()
	defer c.mu.Unlock()

	namespace, workflowName, workflowUID := workflow.Namespace, workflow.Name, workflow.UID
	retryGeneration := workflow.Annotations[util.AnnotationKeyRetryGeneration]
	key := workflowKey(namespace, workflowName)
	previous := c.failureStart[key]
	if previous != nil && (previous.uid != workflowUID || previous.retryGeneration != retryGeneration) {
		log.Infof("Workflow %s/%s is a new attempt (uid %q -> %q, retry generation %q -> %q); resetting image pull failure tracking",
			namespace, workflowName, previous.uid, workflowUID, previous.retryGeneration, retryGeneration)
		previous = nil
	}
	current := &trackedWorkflow{uid: workflowUID, retryGeneration: retryGeneration, pods: make(map[types.UID]time.Time)}
	now := c.now()

	var expired *expiredImagePullFailure
	for _, pod := range pods {
		if !isOwnedByWorkflow(pod, workflowName, workflowUID) {
			// Labels are user-controlled; only act on pods the workflow actually owns.
			log.Warnf("Ignoring pod %s/%s labeled for workflow %s: it is not controlled by that workflow",
				pod.Namespace, pod.Name, workflowName)
			continue
		}
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

		start := now
		if previous != nil {
			if seen, ok := previous.pods[pod.UID]; ok {
				start = seen
			}
		}
		current.pods[pod.UID] = start

		elapsed := now.Sub(start)
		if elapsed < c.gracePeriod {
			log.Debugf("Pod %s/%s has image pull failure for %q (failing for %v), waiting for grace period (%v)",
				pod.Namespace, pod.Name, failedImage, elapsed.Round(time.Second), c.gracePeriod)
			continue
		}
		exitHandler := isExitHandlerPod(pod)
		if expired == nil || (exitHandler && !expired.exitHandler) {
			expired = &expiredImagePullFailure{podName: pod.Name, failedImage: failedImage, elapsed: elapsed, exitHandler: exitHandler}
		}
	}

	if len(current.pods) == 0 {
		delete(c.failureStart, key)
	} else {
		c.failureStart[key] = current
	}
	return expired
}

func workflowKey(namespace, workflowName string) string {
	return namespace + "/" + workflowName
}

// isOwnedByWorkflow reports whether the pod's controller owner reference is the
// Argo Workflow with the given name and UID.
func isOwnedByWorkflow(pod *corev1.Pod, workflowName string, workflowUID types.UID) bool {
	owner := metav1.GetControllerOf(pod)
	if owner == nil {
		return false
	}
	group := owner.APIVersion
	if i := strings.Index(group, "/"); i >= 0 {
		group = group[:i]
	}
	return group == workflowregister.Group &&
		owner.Kind == workflowregister.WorkflowKind &&
		owner.Name == workflowName &&
		owner.UID == workflowUID
}

// isExitHandlerPod reports whether Argo created the pod as part of an exit
// handler (an onExit template or an exit lifecycle hook).
func isExitHandlerPod(pod *corev1.Pod) bool {
	return pod.Labels[argocommon.LabelKeyOnExit] == "true"
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
// failing image so the reason is visible to users. It returns true when the
// workflow was terminated and false when the decision was discarded because
// the workflow changed since it was inspected.
//
// The patch sets activeDeadlineSeconds to 0, which is how KFP marks a run as
// terminated. Argo exempts exit-handler pods from the workflow deadline, so
// healthy cleanup handlers still run after an ordinary task is terminated this
// way. When the stuck pod is itself an exit handler, the deadline cannot stop
// it, so the Terminate shutdown strategy is set as well.
//
// The decision is made against informer caches, which can lag behind the API
// server. A user may retry the run in between, and KFP retries reuse the
// workflow name (updating or recreating the object). The patch therefore
// carries the inspected UID and resource version: the API server rejects a
// resource version mismatch with a Conflict and a UID change with an Invalid
// (immutable field) error, so a stale decision can never hit the new attempt.
func (c *imagePullFailureChecker) terminateWorkflow(ctx context.Context, workflow *metav1.ObjectMeta, failedImage string, forceTerminate bool) (bool, error) {
	namespace, workflowName := workflow.Namespace, workflow.Name
	if c.executionClient == nil {
		return false, fmt.Errorf("execution client not configured, cannot terminate workflow %s/%s", namespace, workflowName)
	}
	if workflow.UID == "" || workflow.ResourceVersion == "" {
		return false, fmt.Errorf("workflow %s/%s has no UID or resource version, cannot terminate it safely", namespace, workflowName)
	}

	terminatePatch, ok := util.GetTerminatePatch(util.CurrentExecutionType()).(map[string]interface{})
	if !ok {
		return false, fmt.Errorf("unsupported execution type for termination")
	}

	// Build a merged patch that terminates the workflow and annotates it with the
	// image pull failure reason so users can see it in the workflow manifest.
	// The uid and resourceVersion act as preconditions, see above.
	patch := map[string]interface{}{
		"metadata": map[string]interface{}{
			"uid":             string(workflow.UID),
			"resourceVersion": workflow.ResourceVersion,
			"annotations": map[string]interface{}{
				"pipelines.kubeflow.org/termination-reason": "ImagePullFailure",
				"pipelines.kubeflow.org/failed-image":       failedImage,
			},
		},
	}
	for k, v := range terminatePatch {
		patch[k] = v
	}
	if forceTerminate {
		spec, ok := patch["spec"].(map[string]interface{})
		if !ok {
			spec = map[string]interface{}{}
			patch["spec"] = spec
		}
		spec["shutdown"] = string(workflowapi.ShutdownStrategyTerminate)
	}

	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return false, fmt.Errorf("failed to marshal termination patch: %w", err)
	}

	_, err = c.executionClient.Execution(namespace).Patch(
		ctx, workflowName, types.MergePatchType, patchBytes, metav1.PatchOptions{})
	if apierrors.IsConflict(err) || apierrors.IsInvalid(err) {
		// The workflow was updated, retried or recreated after it was inspected.
		// Keep the tracking state so the next sync re-evaluates the refreshed
		// workflow immediately instead of restarting the grace period.
		log.Infof("Discarding stale termination decision for workflow %s/%s (uid %s, resourceVersion %s): %v",
			namespace, workflowName, workflow.UID, workflow.ResourceVersion, err)
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to patch workflow %s/%s: %w", namespace, workflowName, err)
	}

	log.Infof("Successfully terminated workflow %s/%s due to image pull failure", namespace, workflowName)
	return true, nil
}
