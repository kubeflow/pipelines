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
	"errors"
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
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

const (
	// ArgoWorkflowLabelKey is the label Argo sets on pods to identify the parent workflow.
	ArgoWorkflowLabelKey = argocommon.LabelKeyWorkflow
	// WorkflowPodIndexName is the name of the pod informer index that maps a
	// workflow (namespace/name) to the pods labeled for it. Looking pods up
	// through the index keeps each check proportional to the workflow's own
	// pods instead of scanning every cached pod in the namespace.
	WorkflowPodIndexName = "workflow"
)

// WorkflowPodIndexFunc indexes a pod under the namespace/name of the workflow
// named by its Argo workflow label. Pods without the label are not indexed.
// The label is user-controlled, so the checker still verifies the controller
// owner reference of every pod returned by the index.
func WorkflowPodIndexFunc(obj interface{}) ([]string, error) {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return nil, fmt.Errorf("expected *corev1.Pod but got %T", obj)
	}
	workflowName, ok := pod.Labels[ArgoWorkflowLabelKey]
	if !ok || workflowName == "" {
		return nil, nil
	}
	return []string{workflowKey(pod.Namespace, workflowName)}, nil
}

// AddWorkflowPodIndex registers the workflow index on the pod informer. It
// must be called before the informer is started.
func AddWorkflowPodIndex(informer cache.SharedIndexInformer) error {
	return informer.AddIndexers(cache.Indexers{WorkflowPodIndexName: WorkflowPodIndexFunc})
}

// StuckPodChecker checks workflow pods for conditions that leave a pod stuck
// with no chance of progress (such as an image that cannot be pulled, or a pod
// no node can fit) and terminates the workflow once a condition has outlasted
// its grace period.
type StuckPodChecker interface {
	// CheckAndTerminate inspects the pods of the running workflow identified by
	// the given metadata and terminates the workflow once a pod has been stuck
	// in one of the checker's conditions for longer than that condition's grace
	// period. Only pods whose controller owner reference matches the workflow
	// name and UID are considered, since pod labels are user-controlled. The
	// termination patch is conditioned on the workflow UID and resource version
	// the decision was made against, so a run that was retried or recreated in
	// the meantime is left alone. Failure tracking is scoped to the workflow UID
	// and its retry-generation annotation, so a retried attempt never inherits
	// the grace period consumed by the attempt it replaces. Before terminating,
	// the stuck pod is re-read from the API server (bypassing the informer
	// cache) and must still exist with the same UID and the same failure, so a
	// stale cached pod from a previous attempt cannot fail the current one even
	// with a zero grace period. A live read showing the pod has recovered also
	// resets that failure's clock, so a later failure of the same kind gets a
	// full grace period of its own.
	CheckAndTerminate(ctx context.Context, workflow *metav1.ObjectMeta) error
	// Forget drops any failure tracking state held for the workflow. Callers
	// should invoke it once a workflow reaches a final state or no longer exists
	// so the checker does not retain state for workflows it will never check again.
	Forget(namespace string, workflowName string)
}

// StuckPodCondition describes one way a workflow pod can be stuck, how to
// detect it on a pod, and how long it may last before the workflow is
// terminated. A checker is built from one or more conditions and shares the
// tracking, live confirmation and termination machinery between them. Create
// conditions with ImagePullCondition and UnschedulableCondition.
type StuckPodCondition struct {
	// name identifies the condition in tracking keys and log messages.
	name string
	// terminationReason is stamped on the terminated workflow under
	// util.AnnotationKeyTerminationReason.
	terminationReason string
	// gracePeriod is how long a failure may last, measured from when the
	// checker first observed it, before the workflow is terminated.
	gracePeriod time.Duration
	// detect returns every failure of this condition on the pod, in a stable
	// order.
	detect func(pod *corev1.Pod) []stuckPodFailure
	// annotations returns extra annotations that describe the failure on the
	// terminated workflow. It may be nil.
	annotations func(failure stuckPodFailure) map[string]string
}

// stuckPodFailure is one detected failure of a condition on a pod.
type stuckPodFailure struct {
	// container names the failing container, or is empty for a failure of the
	// pod as a whole.
	container string
	// subject distinguishes failures of the same container (the image that
	// cannot be pulled). It is part of the tracking key, so a change of subject
	// starts a fresh grace period.
	subject string
	// message is informational only and is not part of the tracking key, so a
	// message that changes between checks does not restart the grace period.
	message string
}

// ImagePullCondition detects containers in ImagePullBackOff or ErrImagePull.
func ImagePullCondition(gracePeriod time.Duration) StuckPodCondition {
	return StuckPodCondition{
		name:              "image pull failure",
		terminationReason: util.TerminationReasonImagePullFailure,
		gracePeriod:       gracePeriod,
		detect: func(pod *corev1.Pod) []stuckPodFailure {
			var failures []stuckPodFailure
			for _, failure := range getImagePullFailures(pod) {
				failures = append(failures, stuckPodFailure{container: failure.container, subject: failure.image})
			}
			return failures
		},
		annotations: func(failure stuckPodFailure) map[string]string {
			return map[string]string{util.AnnotationKeyFailedImage: failure.subject}
		},
	}
}

// UnschedulableCondition detects pods the scheduler has reported as
// unschedulable, for example because they request more CPU or memory than any
// node has. Such a pod stays Pending forever. A pod that is merely waiting for
// a node (the cluster autoscaler may still add one) is also reported, so use a
// grace period long enough to cover node provisioning.
func UnschedulableCondition(gracePeriod time.Duration) StuckPodCondition {
	return StuckPodCondition{
		name:              "unschedulable",
		terminationReason: util.TerminationReasonUnschedulable,
		gracePeriod:       gracePeriod,
		detect:            getUnschedulableFailures,
	}
}

// getUnschedulableFailures reports a Pending pod without a node whose
// PodScheduled condition says the scheduler could not place it. Pods held back
// by scheduling gates are not unschedulable, they are waiting on a controller.
func getUnschedulableFailures(pod *corev1.Pod) []stuckPodFailure {
	if pod.Status.Phase != corev1.PodPending || pod.Spec.NodeName != "" {
		return nil
	}
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodScheduled &&
			condition.Status == corev1.ConditionFalse &&
			condition.Reason == corev1.PodReasonUnschedulable {
			return []stuckPodFailure{{message: condition.Message}}
		}
	}
	return nil
}

// stuckPodChecker checks pods belonging to a workflow for the failures of its
// conditions and terminates the workflow after each condition's grace period.
// It reads pods from a shared informer's indexer, through the workflow index
// registered by AddWorkflowPodIndex, to avoid direct API calls to the
// Kubernetes API server and namespace-wide scans on every check.
//
// A grace period is measured from the moment the checker first observes a
// failure, not from the pod's creation time, so time spent pending or
// initializing does not count against it. A failure that goes away (or a pod
// that is replaced) has its clock reset.
type stuckPodChecker struct {
	podIndexer      cache.Indexer
	getPod          PodGetter
	executionClient util.ExecutionClient
	conditions      []StuckPodCondition
	now             func() time.Time

	mu sync.Mutex
	// failureStart records when each failure was first observed, keyed by
	// workflow (namespace/name). The entry also records the workflow attempt it
	// was observed for so a retry starts a fresh clock.
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
	// failures maps each failure to when it was first observed.
	failures map[stuckPodKey]time.Time
}

// stuckPodKey identifies one failure: a condition on a specific container (or
// the pod as a whole) of a specific pod, plus the condition's subject, such as
// the image that cannot be pulled. Tracking at this granularity means that
// when one container recovers and another starts failing between two checks,
// the new failure starts its own grace period instead of inheriting the
// recovered one's clock.
type stuckPodKey struct {
	condition string
	podUID    types.UID
	container string
	subject   string
}

// PodGetter reads a pod directly from the API server. The checker uses it to
// confirm a cached failure against live state before terminating a workflow.
type PodGetter func(ctx context.Context, namespace, name string) (*corev1.Pod, error)

// ClientsetPodGetter returns a PodGetter backed by the given clientset.
func ClientsetPodGetter(client kubernetes.Interface) PodGetter {
	return func(ctx context.Context, namespace, name string) (*corev1.Pod, error) {
		return client.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	}
}

// NewStuckPodChecker creates a new checker for the given conditions.
// podIndexer is the indexer of a pod shared informer on which
// AddWorkflowPodIndex has been registered, so routine pod lookups are served
// from the local cache by workflow. getPod is only called when a failure has
// outlasted its grace period, to confirm it against the API server before the
// workflow is terminated.
func NewStuckPodChecker(
	podIndexer cache.Indexer,
	getPod PodGetter,
	executionClient util.ExecutionClient,
	conditions ...StuckPodCondition,
) StuckPodChecker {
	return &stuckPodChecker{
		podIndexer:      podIndexer,
		getPod:          getPod,
		executionClient: executionClient,
		conditions:      conditions,
		now:             time.Now,
		failureStart:    make(map[string]*trackedWorkflow),
	}
}

// expiredStuckPod describes a pod whose failure has outlasted its condition's
// grace period.
type expiredStuckPod struct {
	condition *StuckPodCondition
	podName   string
	podUID    types.UID
	failure   stuckPodFailure
	elapsed   time.Duration
	// exitHandler is true when the pod belongs to an exit handler. Such pods
	// are exempt from the workflow deadline, so terminating the workflow
	// requires the Terminate shutdown strategy.
	exitHandler bool
}

// CheckAndTerminate looks up the pods of the given workflow and terminates the
// workflow if any pod has been stuck in one of the checker's conditions longer
// than that condition's grace period (measured from when the failure was first
// observed).
func (c *stuckPodChecker) CheckAndTerminate(ctx context.Context, workflow *metav1.ObjectMeta) error {
	if workflow == nil {
		return fmt.Errorf("workflow metadata is required to check for stuck pods")
	}
	namespace, workflowName := workflow.Namespace, workflow.Name
	pods, err := c.listWorkflowPods(namespace, workflowName)
	if err != nil {
		return err
	}

	expired := c.trackFailures(workflow, pods)
	if expired == nil {
		return nil
	}

	confirmed, err := c.confirmFailureLive(ctx, workflow, expired)
	if err != nil {
		return err
	}
	if !confirmed {
		return nil
	}

	log.Infof("Terminating workflow %s/%s: pod %s has been stuck on %s (container %q, subject %q) for %v, which exceeds the grace period %v",
		namespace, workflowName, expired.podName, expired.condition.name, expired.failure.container, expired.failure.subject,
		expired.elapsed.Round(time.Second), expired.condition.gracePeriod)
	terminated, err := c.terminateWorkflow(ctx, workflow, expired)
	if err != nil {
		return err
	}
	if terminated {
		c.Forget(namespace, workflowName)
	}
	return nil
}

// confirmFailureLive re-reads the expired pod from the API server and reports
// whether the failure the decision was based on is still present on the live
// object. The informer cache can lag: KFP retries delete the previous attempt's
// pods before updating the workflow, and the workflow informer may observe the
// retry while the pod informer still holds the deleted, failing pod. Because
// an in-place retry keeps the workflow UID and the termination patch carries
// the refreshed resource version, nothing else would stop that stale pod from
// failing the new attempt, which is immediate with a zero grace period.
//
// The pod must still exist with the same UID, still be controlled by the
// workflow, still be non-terminal, and still report the same container and
// image failing. A deleted, replaced or terminal pod keeps its tracking entry
// until the cache stops listing it, which is harmless since that pod UID can
// never fail again. A pod that recovered has its entry dropped at once, so a
// container that fails to pull the same image again (a restarted sidecar)
// gets a fresh grace period even if the cache never showed it healthy. Only a
// failed read, which proves nothing, keeps the entry.
func (c *stuckPodChecker) confirmFailureLive(ctx context.Context, workflow *metav1.ObjectMeta, expired *expiredStuckPod) (bool, error) {
	namespace, workflowName, workflowUID := workflow.Namespace, workflow.Name, workflow.UID
	if c.getPod == nil {
		return false, fmt.Errorf("pod getter not configured, cannot confirm %s of pod %s/%s", expired.condition.name, namespace, expired.podName)
	}
	livePod, err := c.getPod(ctx, namespace, expired.podName)
	if apierrors.IsNotFound(err) {
		log.Infof("Discarding %s decision for pod %s/%s: the pod no longer exists", expired.condition.name, namespace, expired.podName)
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to confirm %s of pod %s/%s: %w", expired.condition.name, namespace, expired.podName, err)
	}
	if livePod.UID != expired.podUID {
		log.Infof("Discarding %s decision for pod %s/%s: the cached pod (uid %s) was replaced (live uid %s)",
			expired.condition.name, namespace, expired.podName, expired.podUID, livePod.UID)
		return false, nil
	}
	if !isOwnedByWorkflow(livePod, workflowName, workflowUID) || isPodTerminal(livePod) {
		log.Infof("Discarding %s decision for pod %s/%s: the live pod is terminal or no longer owned by the workflow",
			expired.condition.name, namespace, expired.podName)
		return false, nil
	}
	for _, failure := range expired.condition.detect(livePod) {
		if failure.container == expired.failure.container && failure.subject == expired.failure.subject {
			return true, nil
		}
	}
	log.Infof("Discarding %s decision for pod %s/%s: the live pod no longer reports it (container %q, subject %q); resetting its failure clock",
		expired.condition.name, namespace, expired.podName, expired.failure.container, expired.failure.subject)
	c.forgetFailure(workflow, expired)
	return false, nil
}

// forgetFailure drops the tracking entry of one failure, so a later failure
// of the same kind on the same container of the same pod starts a fresh grace
// period. The entry is only dropped if it still belongs to the
// workflow attempt the decision was made for.
func (c *stuckPodChecker) forgetFailure(workflow *metav1.ObjectMeta, expired *expiredStuckPod) {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := workflowKey(workflow.Namespace, workflow.Name)
	tracked := c.failureStart[key]
	if tracked == nil || tracked.uid != workflow.UID || tracked.retryGeneration != workflow.Annotations[util.AnnotationKeyRetryGeneration] {
		return
	}
	delete(tracked.failures, failureKeyOf(expired.condition, expired.podUID, expired.failure))
	if len(tracked.failures) == 0 {
		delete(c.failureStart, key)
	}
}

// listWorkflowPods returns the cached pods labeled for the workflow, using the
// workflow index so the lookup does not scan the whole namespace.
func (c *stuckPodChecker) listWorkflowPods(namespace, workflowName string) ([]*corev1.Pod, error) {
	objs, err := c.podIndexer.ByIndex(WorkflowPodIndexName, workflowKey(namespace, workflowName))
	if err != nil {
		return nil, fmt.Errorf("failed to look up pods for workflow %s/%s: %w", namespace, workflowName, err)
	}
	pods := make([]*corev1.Pod, 0, len(objs))
	for _, obj := range objs {
		pod, ok := obj.(*corev1.Pod)
		if !ok {
			return nil, fmt.Errorf("pod index for workflow %s/%s returned %T", namespace, workflowName, obj)
		}
		pods = append(pods, pod)
	}
	return pods, nil
}

// Forget drops the failure tracking state for the given workflow.
func (c *stuckPodChecker) Forget(namespace string, workflowName string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.failureStart, workflowKey(namespace, workflowName))
}

// trackFailures updates the failure start times for the workflow's failing
// pods and returns a failure that has outlasted its condition's grace period,
// or nil. When several have expired, one on an exit-handler pod is preferred
// because it needs the stronger termination mechanism.
// Failures that are no longer reported, or pods that are no longer listed,
// have their tracking dropped so a recovered pod starts a fresh grace period
// next time.
// State recorded for a different attempt of the workflow (another UID or
// retry generation) is discarded, so pods left over from the previous attempt
// cannot fail the new one with an already expired clock.
func (c *stuckPodChecker) trackFailures(workflow *metav1.ObjectMeta, pods []*corev1.Pod) *expiredStuckPod {
	c.mu.Lock()
	defer c.mu.Unlock()

	namespace, workflowName, workflowUID := workflow.Namespace, workflow.Name, workflow.UID
	retryGeneration := workflow.Annotations[util.AnnotationKeyRetryGeneration]
	key := workflowKey(namespace, workflowName)
	previous := c.failureStart[key]
	if previous != nil && (previous.uid != workflowUID || previous.retryGeneration != retryGeneration) {
		log.Infof("Workflow %s/%s is a new attempt (uid %q -> %q, retry generation %q -> %q); resetting stuck pod tracking",
			namespace, workflowName, previous.uid, workflowUID, previous.retryGeneration, retryGeneration)
		previous = nil
	}
	current := &trackedWorkflow{uid: workflowUID, retryGeneration: retryGeneration, failures: make(map[stuckPodKey]time.Time)}
	now := c.now()

	var expired *expiredStuckPod
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
		exitHandler := isExitHandlerPod(pod)
		for i := range c.conditions {
			condition := &c.conditions[i]
			for _, failure := range condition.detect(pod) {
				failureKey := failureKeyOf(condition, pod.UID, failure)
				start := now
				if previous != nil {
					if seen, ok := previous.failures[failureKey]; ok {
						start = seen
					}
				}
				current.failures[failureKey] = start

				elapsed := now.Sub(start)
				if elapsed < condition.gracePeriod {
					log.Debugf("Pod %s/%s is stuck on %s (container %q, subject %q) for %v, waiting for grace period (%v)",
						pod.Namespace, pod.Name, condition.name, failure.container, failure.subject, elapsed.Round(time.Second), condition.gracePeriod)
					continue
				}
				if expired == nil || (exitHandler && !expired.exitHandler) {
					expired = &expiredStuckPod{condition: condition, podName: pod.Name, podUID: pod.UID, failure: failure, elapsed: elapsed, exitHandler: exitHandler}
				}
			}
		}
	}

	if len(current.failures) == 0 {
		delete(c.failureStart, key)
	} else {
		c.failureStart[key] = current
	}
	return expired
}

// failureKeyOf builds the tracking key of one failure of a condition on a pod.
func failureKeyOf(condition *StuckPodCondition, podUID types.UID, failure stuckPodFailure) stuckPodKey {
	return stuckPodKey{condition: condition.name, podUID: podUID, container: failure.container, subject: failure.subject}
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

// imagePullFailure describes a container that cannot pull its image.
type imagePullFailure struct {
	container string
	image     string
}

// getImagePullFailures returns every init and main container of the pod that
// is in ImagePullBackOff or ErrImagePull, in status order.
func getImagePullFailures(pod *corev1.Pod) []imagePullFailure {
	var failures []imagePullFailure
	for _, status := range pod.Status.InitContainerStatuses {
		if failure, ok := imagePullFailureFromStatus(status); ok {
			failures = append(failures, failure)
		}
	}
	for _, status := range pod.Status.ContainerStatuses {
		if failure, ok := imagePullFailureFromStatus(status); ok {
			failures = append(failures, failure)
		}
	}
	return failures
}

// imagePullFailureFromStatus reports whether the container is in
// ImagePullBackOff or ErrImagePull state, and if so which image it is failing
// to pull.
func imagePullFailureFromStatus(status corev1.ContainerStatus) (imagePullFailure, bool) {
	if status.State.Waiting != nil {
		reason := status.State.Waiting.Reason
		if reason == "ImagePullBackOff" || reason == "ErrImagePull" {
			return imagePullFailure{container: status.Name, image: status.Image}, true
		}
	}
	return imagePullFailure{}, false
}

// terminateWorkflow terminates an Argo workflow and annotates it with the
// termination reason (and any detail the condition provides, such as the
// failing image) so the reason is visible to users. It returns true when the
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
func (c *stuckPodChecker) terminateWorkflow(ctx context.Context, workflow *metav1.ObjectMeta, expired *expiredStuckPod) (bool, error) {
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
	// failure reason so users can see it in the workflow manifest.
	// The uid and resourceVersion act as preconditions, see above.
	annotations := map[string]interface{}{
		util.AnnotationKeyTerminationReason: expired.condition.terminationReason,
	}
	if expired.condition.annotations != nil {
		for key, value := range expired.condition.annotations(expired.failure) {
			annotations[key] = value
		}
	}
	patch := map[string]interface{}{
		"metadata": map[string]interface{}{
			"uid":             string(workflow.UID),
			"resourceVersion": workflow.ResourceVersion,
			"annotations":     annotations,
		},
	}
	for k, v := range terminatePatch {
		patch[k] = v
	}
	if expired.exitHandler {
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
	if apierrors.IsConflict(err) || isImmutableUIDError(err) {
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

	log.Infof("Successfully terminated workflow %s/%s due to %s", namespace, workflowName, expired.condition.name)
	return true, nil
}

// isImmutableUIDError reports whether the API server rejected the patch
// because its metadata.uid precondition did not match the live workflow, i.e.
// the workflow was recreated. Any other validation failure is a real error:
// treating it as a stale decision would retry the same doomed patch forever
// while only logging at Info level.
func isImmutableUIDError(err error) bool {
	if !apierrors.IsInvalid(err) {
		return false
	}
	var statusErr *apierrors.StatusError
	if !errors.As(err, &statusErr) || statusErr.ErrStatus.Details == nil {
		return false
	}
	for _, cause := range statusErr.ErrStatus.Details.Causes {
		if cause.Field == "metadata.uid" {
			return true
		}
	}
	return false
}
