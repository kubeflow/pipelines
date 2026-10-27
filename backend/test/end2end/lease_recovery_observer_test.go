// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package end2end

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
)

func TestReplicaPodObserverRetainsDeletedWorkloads(t *testing.T) {
	events := make(chan watch.Event, 6)
	producer := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "producer", UID: "original"},
		Status:     corev1.PodStatus{Phase: corev1.PodRunning},
	}
	events <- watch.Event{Type: watch.Added, Object: producer.DeepCopy()}
	producer.Status.Phase = corev1.PodSucceeded
	events <- watch.Event{Type: watch.Modified, Object: producer.DeepCopy()}
	events <- watch.Event{Type: watch.Deleted, Object: producer}
	consumer := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "consumer", UID: "consumer"},
		Status:     corev1.PodStatus{Phase: corev1.PodSucceeded},
	}
	events <- watch.Event{Type: watch.Added, Object: consumer}
	events <- watch.Event{Type: watch.Deleted, Object: consumer}

	pods := make(map[types.UID]replicaPodObservation)
	require.NoError(t, recordReplicaPodEvents(events, pods))
	require.Len(t, pods, 2)
	require.Equal(t, corev1.PodSucceeded, pods["original"].Pod.Status.Phase)
	require.True(t, pods["original"].Deleted)
	producer.Status.Phase = corev1.PodFailed
	require.Equal(t, corev1.PodSucceeded, pods["original"].Pod.Status.Phase)

	replayed := producer.DeepCopy()
	replayed.UID = "replayed"
	events <- watch.Event{Type: watch.Added, Object: replayed}
	events <- watch.Event{Type: watch.Deleted, Object: replayed}
	require.NoError(t, recordReplicaPodEvents(events, pods))
	require.Len(t, pods, 3, "a replay must remain visible even after both Pod UIDs are deleted")
}

func TestReplicaPodObserverWaitsForConsumerCleanupBeforeCounting(t *testing.T) {
	events := make(chan watch.Event, 4)
	producer := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "producer", UID: "original"},
		Status:     corev1.PodStatus{Phase: corev1.PodSucceeded},
	}
	consumer := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "consumer", UID: "consumer",
			Labels: map[string]string{"pipelines.kubeflow.org/task_id": "consumer-task"},
		},
		Status: corev1.PodStatus{Phase: corev1.PodRunning},
	}
	pods := make(map[types.UID]replicaPodObservation)
	events <- watch.Event{Type: watch.Deleted, Object: producer}
	events <- watch.Event{Type: watch.Added, Object: consumer}
	require.NoError(t, recordReplicaPodEvents(events, pods))
	require.Len(t, pods, 2)
	require.False(t, replicaConsumerCleanupObserved(pods, "consumer-task"),
		"two UIDs including the consumer are not yet a complete watch")

	replayed := consumer.DeepCopy()
	replayed.UID = "replayed"
	events <- watch.Event{Type: watch.Added, Object: replayed}
	events <- watch.Event{Type: watch.Deleted, Object: replayed}
	require.NoError(t, recordReplicaPodEvents(events, pods))
	require.False(t, replicaConsumerCleanupObserved(pods, "consumer-task"))

	consumer.Status.Phase = corev1.PodSucceeded
	events <- watch.Event{Type: watch.Modified, Object: consumer}
	require.NoError(t, recordReplicaPodEvents(events, pods))
	require.False(t, replicaConsumerCleanupObserved(pods, "consumer-task"),
		"terminal status is not the cleanup/deletion barrier")
	events <- watch.Event{Type: watch.Deleted, Object: consumer}
	require.NoError(t, recordReplicaPodEvents(events, pods))
	require.True(t, replicaConsumerCleanupObserved(pods, "consumer-task"))
	require.Len(t, pods, 3, "the final count must include a replay delivered in a later batch")
}

func TestReplicaPodObserverReportsWatchFailures(t *testing.T) {
	t.Run("server error", func(t *testing.T) {
		events := make(chan watch.Event, 1)
		events <- watch.Event{Type: watch.Error, Object: &metav1.Status{
			Status: metav1.StatusFailure, Message: "expired watch", Code: 410,
		}}
		require.ErrorContains(t, recordReplicaPodEvents(events, make(map[types.UID]replicaPodObservation)), "expired watch")
	})
	t.Run("unexpected resource", func(t *testing.T) {
		events := make(chan watch.Event, 1)
		events <- watch.Event{Type: watch.Added, Object: &corev1.ConfigMap{}}
		require.ErrorContains(t, recordReplicaPodEvents(events, make(map[types.UID]replicaPodObservation)), "instead of a Pod")
	})
	t.Run("closed watch", func(t *testing.T) {
		events := make(chan watch.Event)
		close(events)
		require.ErrorContains(t, recordReplicaPodEvents(events, make(map[types.UID]replicaPodObservation)), "watch closed")
	})
}
