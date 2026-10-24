package poc

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func imagePullAttempt(t *testing.T) (kubernetesExecutor, ExecutionReference, *stoppingPodClient) {
	t.Helper()
	client := &stoppingPodClient{recordingPodClient{pods: map[string]*corev1.Pod{}}}
	executor := kubernetesExecutor{coreClient: &recordingCoreClient{podClient: client}}
	reference, err := executor.Submit(context.Background(), TaskExecutionRequest{
		Identity:  TaskAttemptIdentity{RunID: "run", TaskID: "task", AttemptID: "attempt"},
		Namespace: "ns", Pod: &corev1.Pod{Spec: corev1.PodSpec{Containers: []corev1.Container{
			{Name: "main", Image: "registry.invalid/missing:image"},
		}}},
	})
	require.NoError(t, err)
	pod := client.pods[attemptWorkloadName(reference.Identity)]
	pod.Spec.NodeName = "worker"
	pod.Status.Phase = corev1.PodPending
	pod.Status.ContainerStatuses = []corev1.ContainerStatus{{
		Name: "main", Image: "registry.invalid/missing:image",
		State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "ErrImagePull"}},
	}}
	return executor, reference, client
}

func TestKubernetesImagePullFailureTerminatesOwnedAttempt(t *testing.T) {
	t.Setenv("IMAGE_PULL_FAILURE_HANDLING_ENABLED", "true")
	t.Setenv("IMAGE_PULL_FAILURE_GRACE_PERIOD_SEC", "0")
	executor, reference, client := imagePullAttempt(t)
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()
	result, err := executor.Wait(ctx, reference)
	require.NoError(t, err)
	require.Equal(t, TaskOutcomeFailure, result.Outcome)
	require.NotNil(t, result.Failure)
	require.Equal(t, "image-pull", result.Failure.Class)
	require.Contains(t, result.Failure.Message, "registry.invalid/missing:image")
	require.Equal(t, corev1.PodFailed, client.pods[attemptWorkloadName(reference.Identity)].Status.Phase)
	var logs bytes.Buffer
	require.NoError(t, executor.ReadLogs(context.Background(), reference, false, &logs))
	require.Contains(t, logs.String(), "No container output was produced")
	require.Contains(t, logs.String(), "registry.invalid/missing:image")
}

func TestKubernetesImagePullPolicyValidation(t *testing.T) {
	for _, test := range []struct {
		name    string
		enabled string
		grace   string
		wantErr string
	}{
		{name: "disabled"},
		{name: "default grace", enabled: "true"},
		{name: "immediate", enabled: "true", grace: "0"},
		{name: "invalid toggle", enabled: "perhaps", wantErr: "must be a boolean"},
		{name: "negative", enabled: "true", grace: "-1", wantErr: "nonnegative integer"},
		{name: "fractional", enabled: "true", grace: "0.5", wantErr: "nonnegative integer"},
		{name: "overflow", enabled: "true", grace: "9223372036854775807", wantErr: "supported duration"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("IMAGE_PULL_FAILURE_HANDLING_ENABLED", test.enabled)
			t.Setenv("IMAGE_PULL_FAILURE_GRACE_PERIOD_SEC", test.grace)
			policy, err := kubernetesImagePullPolicy()
			if test.wantErr != "" {
				require.ErrorContains(t, err, test.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.enabled == "true", policy.enabled)
			if test.grace == "0" {
				require.Zero(t, policy.grace)
			} else {
				require.Equal(t, 5*time.Minute, policy.grace)
			}
		})
	}
}

func TestKubernetesImagePullClockTracksContainersAndResetsAfterRecovery(t *testing.T) {
	_, reference, client := imagePullAttempt(t)
	pod := client.pods[attemptWorkloadName(reference.Identity)]
	now := time.Now().UTC()
	state, expired, changed := observeKubernetesImagePulls(kubernetesImagePullState{}, pod, now, time.Minute)
	require.False(t, expired)
	require.True(t, changed)
	state, expired, changed = observeKubernetesImagePulls(state, pod, now.Add(time.Minute-time.Nanosecond), time.Minute)
	require.False(t, expired, "the full configured grace period must elapse")
	require.False(t, changed)
	pod.Status.ContainerStatuses[0].State.Waiting.Reason = "ImagePullBackOff"
	state, expired, _ = observeKubernetesImagePulls(state, pod, now.Add(time.Minute), time.Minute)
	require.True(t, expired, "switching pull-error reasons must not restart the same image's clock")
	pod.Status.ContainerStatuses[0].Image = "replacement:image"
	state, expired, changed = observeKubernetesImagePulls(state, pod, now.Add(2*time.Minute), time.Minute)
	require.False(t, expired, "a replacement image must not inherit the previous image's elapsed grace")
	require.True(t, changed)

	pod.Status.ContainerStatuses = nil
	state, expired, changed = observeKubernetesImagePulls(state, pod, now.Add(3*time.Minute), time.Minute)
	require.False(t, expired)
	require.True(t, changed)
	require.Empty(t, state.Observed)
	pod.Status.InitContainerStatuses = []corev1.ContainerStatus{{
		Name: "init", Image: "another:image",
		State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "ErrImagePull"}},
	}}
	state, expired, changed = observeKubernetesImagePulls(state, pod, now.Add(4*time.Minute), time.Minute)
	require.False(t, expired, "a different container/image gets its own full grace period")
	require.True(t, changed)
	require.Len(t, state.Observed, 1)
}

func TestKubernetesImagePullRecoveryPreservesObservationCheckpoint(t *testing.T) {
	executor, reference, _ := imagePullAttempt(t)
	observed := time.Now().UTC().Add(-time.Minute)
	key := "main\x00registry.invalid/missing:image"
	service := &recordingExecutionIO{}
	ctx := completionContext(service)
	require.NoError(t, saveKubernetesImagePullState(ctx, reference, kubernetesImagePullState{
		Observed: map[string]time.Time{key: observed},
	}))
	recovered, err := executor.Recover(ctx, reference.Identity, &service.lastReference)
	require.NoError(t, err)
	require.Equal(t, RecoveryExisting, recovered.State)
	state, err := loadKubernetesImagePullState(*recovered.Reference)
	require.NoError(t, err)
	require.True(t, state.Observed[key].Equal(observed), "owner takeover must not restart the grace period")
}

func TestKubernetesImagePullDecisionSurvivesRecovery(t *testing.T) {
	t.Setenv("IMAGE_PULL_FAILURE_HANDLING_ENABLED", "false")
	for _, missing := range []bool{false, true} {
		t.Run(map[bool]string{false: "pending Pod", true: "missing Pod"}[missing], func(t *testing.T) {
			executor, reference, client := imagePullAttempt(t)
			state := kubernetesImagePullState{
				Container: "main", Image: "registry.invalid/missing:image",
				Failure: &TaskFailure{Class: "image-pull", Message: "recorded image-pull failure"},
			}
			require.NoError(t, saveKubernetesImagePullState(context.Background(), reference, state))
			if missing {
				delete(client.pods, attemptWorkloadName(reference.Identity))
			}
			service := &recordingExecutionIO{}
			recovered, err := executor.Recover(completionContext(service), reference.Identity, &reference)
			require.NoError(t, err)
			require.Equal(t, RecoveryCompleted, recovered.State)
			require.Equal(t, "image-pull", recovered.Result.Failure.Class)
			require.Equal(t, "recorded image-pull failure", recovered.Result.Failure.Message)
		})
	}
}

func TestKubernetesImagePullRequiresOwnershipBeforeTermination(t *testing.T) {
	t.Setenv("IMAGE_PULL_FAILURE_HANDLING_ENABLED", "true")
	t.Setenv("IMAGE_PULL_FAILURE_GRACE_PERIOD_SEC", "0")
	executor, reference, client := imagePullAttempt(t)
	service := &recordingExecutionIO{checkpointErr: errors.New("ownership lost")}
	_, err := executor.Wait(completionContext(service), reference)
	require.ErrorIs(t, err, service.checkpointErr)
	pod := client.pods[attemptWorkloadName(reference.Identity)]
	require.Nil(t, pod.Spec.ActiveDeadlineSeconds)
	require.Equal(t, corev1.PodPending, pod.Status.Phase)
}

func TestKubernetesImagePullRechecksRecoveryBeforeTermination(t *testing.T) {
	executor, reference, client := imagePullAttempt(t)
	pod := client.pods[attemptWorkloadName(reference.Identity)]
	stale := pod.DeepCopy()
	pod.Status.Phase = corev1.PodRunning
	pod.Status.ContainerStatuses = nil
	failure, err := executor.checkImagePullFailure(context.Background(), reference, stale, imagePullPolicy{enabled: true})
	require.NoError(t, err)
	require.Nil(t, failure)
	state, err := loadKubernetesImagePullState(reference)
	require.NoError(t, err)
	require.Empty(t, state.Observed, "a recovered container must get a fresh clock if pulling fails again")
	require.Nil(t, pod.Spec.ActiveDeadlineSeconds)
}

func TestKubernetesImagePullDiagnosticDoesNotReplaceStartedContainerLogs(t *testing.T) {
	_, reference, client := imagePullAttempt(t)
	pod := client.pods[attemptWorkloadName(reference.Identity)]
	pod.Status.Phase = corev1.PodFailed
	require.True(t, imagePullMainNeverStarted(pod))
	pod.Status.ContainerStatuses[0].ContainerID = "containerd://started"
	require.False(t, imagePullMainNeverStarted(pod))
	pod.Status.ContainerStatuses[0].ContainerID = ""
	pod.Status.ContainerStatuses[0].RestartCount = 1
	require.False(t, imagePullMainNeverStarted(pod))
	pod.Status.ContainerStatuses = nil
	require.False(t, imagePullMainNeverStarted(pod), "missing status is not proof of no output")
}

func TestKubernetesImagePullPolicyDisabledPreservesPendingWork(t *testing.T) {
	t.Setenv("IMAGE_PULL_FAILURE_HANDLING_ENABLED", "false")
	t.Setenv("IMAGE_PULL_FAILURE_GRACE_PERIOD_SEC", "0")
	executor, reference, client := imagePullAttempt(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err := executor.Wait(ctx, reference)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	pod := client.pods[attemptWorkloadName(reference.Identity)]
	require.Nil(t, pod.Spec.ActiveDeadlineSeconds)
	require.Equal(t, corev1.PodPending, pod.Status.Phase)
}

func TestKubernetesImagePullCheckpointRejectsInvalidState(t *testing.T) {
	for _, state := range []kubernetesImagePullState{
		{Observed: map[string]time.Time{"main": {}}},
		{Failure: &TaskFailure{Class: "image-pull"}},
		{Container: "main", Image: "image", Failure: &TaskFailure{Class: "other"}},
	} {
		_, reference, _ := imagePullAttempt(t)
		require.NoError(t, saveKubernetesImagePullState(context.Background(), reference, state))
		_, err := loadKubernetesImagePullState(reference)
		require.Error(t, err)
	}
}
