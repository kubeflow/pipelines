package component

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_podFailureFields_waitingReason_ImagePullBackOff(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "p", Namespace: "ns"},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "main",
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{
							Reason:  "ImagePullBackOff",
							Message: "Back-off pulling image",
						},
					},
				},
			},
		},
	}

	fields := podFailureFields(pod)
	require.NotNil(t, fields)
	require.Equal(t, "p", fields["podName"].GetStringValue())
	require.Equal(t, "ns", fields["namespace"].GetStringValue())
	require.Equal(t, "main", fields["containerName"].GetStringValue())
	require.Equal(t, "ImagePullBackOff", fields["reason"].GetStringValue())
	require.Equal(t, "Back-off pulling image", fields["kubernetesMessage"].GetStringValue())
}

func Test_podFailureFields_terminatedReason_OOMKilled(t *testing.T) {
	exitCode := int32(137)
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "p", Namespace: "ns"},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "main",
					State: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							Reason:   "OOMKilled",
							Message:  "Container killed due to memory",
							ExitCode: exitCode,
						},
					},
				},
			},
		},
	}

	fields := podFailureFields(pod)
	require.NotNil(t, fields)
	require.Equal(t, "main", fields["containerName"].GetStringValue())
	require.Equal(t, "OOMKilled", fields["reason"].GetStringValue())
	require.Equal(t, float64(137), fields["containerExitCode"].GetNumberValue())
	require.Equal(t, "Container killed due to memory", fields["kubernetesMessage"].GetStringValue())
}

