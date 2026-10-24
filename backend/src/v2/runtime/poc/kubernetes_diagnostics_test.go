package poc

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestKubernetesFailureRetainsContainerTerminationDiagnostics(t *testing.T) {
	pod := &corev1.Pod{
		Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "main"}}},
		Status: corev1.PodStatus{
			Phase: corev1.PodFailed,
			ContainerStatuses: []corev1.ContainerStatus{{
				Name: "main", State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{
					ExitCode: 1, Reason: "Error", Message: "pip could not download the component dependency",
				}},
			}},
		},
	}
	result, terminal := podTaskResult(pod)
	require.True(t, terminal)
	require.Equal(t, TaskOutcomeFailure, result.Outcome)
	require.Contains(t, result.Failure.Message, "exit code 1")
	require.Contains(t, result.Failure.Message, "pip could not download")
}

func TestCoordinatorPodCapturesLogsWhenLauncherCannotReport(t *testing.T) {
	spec := &corev1.PodSpec{Containers: []corev1.Container{{Name: "main", Image: "python"}}}
	augmentPodSpecForLauncher(spec)
	require.Equal(t, corev1.TerminationMessageFallbackToLogsOnError, spec.Containers[0].TerminationMessagePolicy)
	require.Equal(t, corev1.TerminationMessageFallbackToLogsOnError, spec.InitContainers[0].TerminationMessagePolicy)
}
