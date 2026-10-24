package poc

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	commonutil "github.com/kubeflow/pipelines/backend/src/common/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const kubernetesImagePullKey = "imagePull"

type imagePullPolicy struct {
	enabled bool
	grace   time.Duration
}

type kubernetesImagePullState struct {
	Observed  map[string]time.Time `json:"observed,omitempty"`
	Container string               `json:"container,omitempty"`
	Image     string               `json:"image,omitempty"`
	Failure   *TaskFailure         `json:"failure,omitempty"`
}

func kubernetesImagePullPolicy() (imagePullPolicy, error) {
	policy := imagePullPolicy{grace: 5 * time.Minute}
	if value := os.Getenv("IMAGE_PULL_FAILURE_HANDLING_ENABLED"); value != "" {
		enabled, err := strconv.ParseBool(value)
		if err != nil {
			return policy, fmt.Errorf("IMAGE_PULL_FAILURE_HANDLING_ENABLED must be a boolean: %w", err)
		}
		policy.enabled = enabled
	}
	if !policy.enabled {
		return policy, nil
	}
	if value := os.Getenv("IMAGE_PULL_FAILURE_GRACE_PERIOD_SEC"); value != "" {
		seconds, err := strconv.ParseInt(value, 10, 64)
		if err != nil || seconds < 0 {
			return policy, fmt.Errorf("IMAGE_PULL_FAILURE_GRACE_PERIOD_SEC must be a nonnegative integer")
		}
		grace, err := time.ParseDuration(value + "s")
		if err != nil {
			return policy, fmt.Errorf("IMAGE_PULL_FAILURE_GRACE_PERIOD_SEC exceeds the supported duration: %w", err)
		}
		policy.grace = grace
	}
	return policy, nil
}

func loadKubernetesImagePullState(reference ExecutionReference) (kubernetesImagePullState, error) {
	var state kubernetesImagePullState
	if value, ok := reference.Data[kubernetesImagePullKey]; ok {
		data, err := toJSONData(value)
		if err != nil {
			return state, err
		}
		if err := fromJSONData(data, &state); err != nil {
			return state, err
		}
	}
	for _, observed := range state.Observed {
		if observed.IsZero() {
			return state, fmt.Errorf("image-pull checkpoint contains an invalid observation time")
		}
	}
	if state.Failure != nil && (state.Container == "" || state.Image == "" || state.Failure.Class != "image-pull") {
		return state, fmt.Errorf("image-pull failure checkpoint is incomplete")
	}
	return state, nil
}

func saveKubernetesImagePullState(ctx context.Context, reference ExecutionReference, state kubernetesImagePullState) error {
	data, err := toJSONData(state)
	if err != nil {
		return err
	}
	reference.Data[kubernetesImagePullKey] = data
	if service := scopedExecutionIO(ctx); service != nil {
		return service.Checkpoint(ctx, reference)
	}
	return nil
}

func observeKubernetesImagePulls(state kubernetesImagePullState, pod *corev1.Pod, now time.Time, grace time.Duration) (kubernetesImagePullState, bool, bool) {
	observed := make(map[string]time.Time)
	changed, expired := false, false
	statuses := append(append([]corev1.ContainerStatus{}, pod.Status.InitContainerStatuses...), pod.Status.ContainerStatuses...)
	for _, status := range statuses {
		if pod.DeletionTimestamp != nil || !commonutil.IsImagePullFailure(status) {
			continue
		}
		key := status.Name + "\x00" + status.Image
		start, exists := state.Observed[key]
		if !exists {
			start, changed = now, true
		}
		observed[key] = start
		if !expired && now.Sub(start) >= grace {
			state.Container, state.Image, expired = status.Name, status.Image, true
		}
	}
	changed = changed || len(observed) != len(state.Observed)
	state.Observed = observed
	return state, expired, changed
}

func (k kubernetesExecutor) checkImagePullFailure(ctx context.Context, reference ExecutionReference, pod *corev1.Pod, policy imagePullPolicy) (*kubernetesImagePullState, error) {
	state, err := loadKubernetesImagePullState(reference)
	if err != nil {
		return nil, err
	}
	state, expired, changed := observeKubernetesImagePulls(state, pod, time.Now(), policy.grace)
	if changed {
		if err := saveKubernetesImagePullState(ctx, reference, state); err != nil {
			return nil, err
		}
	}
	if !expired {
		return nil, nil
	}
	// Recheck live state before making the durable termination decision.
	current, err := k.coreClient.PodClient(pod.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	data, err := parsePodReference(reference)
	if err != nil {
		return nil, err
	}
	if err := checkReferencedPod(current, reference, data); err != nil {
		return nil, err
	}
	if _, terminal := podTaskResult(current); terminal {
		return nil, nil
	}
	state, expired, changed = observeKubernetesImagePulls(state, current, time.Now(), policy.grace)
	if changed {
		if err := saveKubernetesImagePullState(ctx, reference, state); err != nil {
			return nil, err
		}
	}
	if !expired {
		return nil, nil
	}
	state.Failure = &TaskFailure{
		Class: "image-pull",
		Message: fmt.Sprintf("ImagePullFailure: container %q could not pull image %q within the %s grace period",
			state.Container, state.Image, policy.grace),
	}
	return &state, nil
}

func (k kubernetesExecutor) finishImagePullFailure(ctx context.Context, reference ExecutionReference, state kubernetesImagePullState) (TaskExecutionResult, error) {
	if _, err := parsePodReference(reference); err != nil {
		return TaskExecutionResult{}, err
	}
	// Checkpoint the decision before stopping the Pod so a replacement worker
	// finishes the same failure instead of restarting its grace period.
	if err := saveKubernetesImagePullState(ctx, reference, state); err != nil {
		return TaskExecutionResult{}, err
	}
	canceled, err := k.Cancel(ctx, reference)
	if err != nil {
		return TaskExecutionResult{}, err
	}
	if canceled.State != CancellationConfirmed {
		return TaskExecutionResult{}, fmt.Errorf("image-pull failure termination is not confirmed")
	}
	if canceled.Reference != nil {
		reference = *canceled.Reference
	}
	return completeExecution(ctx, k, reference, TaskExecutionResult{
		Outcome: TaskOutcomeFailure, Failure: state.Failure,
		StatusMetadata: customStatusMetadata(state.Failure.Message, map[string]any{
			"terminationReason": commonutil.TerminationReasonImagePullFailure,
			"failedImage":       state.Image,
			"failedContainer":   state.Container,
		}),
	})
}

func imagePullMainNeverStarted(pod *corev1.Pod) bool {
	if pod.Status.Phase != corev1.PodFailed || len(pod.Spec.Containers) == 0 {
		return false
	}
	for _, status := range pod.Status.ContainerStatuses {
		if status.Name != pod.Spec.Containers[0].Name {
			continue
		}
		if status.ContainerID != "" || status.RestartCount != 0 ||
			(status.Started != nil && *status.Started) || status.LastTerminationState.Running != nil ||
			status.LastTerminationState.Terminated != nil || status.State.Running != nil {
			return false
		}
		return status.State.Waiting != nil ||
			(status.State.Terminated != nil && status.State.Terminated.StartedAt.IsZero())
	}
	return false
}
