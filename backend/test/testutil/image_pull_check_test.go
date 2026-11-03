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

package testutil

import (
	"errors"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
)

func imagePullTestPod() corev1.Pod {
	return corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "task", Namespace: "kubeflow", UID: "pod-uid", Labels: map[string]string{"pipeline/runid": "run-id"}},
		Status: corev1.PodStatus{
			Phase: corev1.PodPending,
			ContainerStatuses: []corev1.ContainerStatus{{
				Name: "main", Image: "public.ecr.aws/docker/library/python:3.12",
				State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{
					Reason: "ImagePullBackOff", Message: "Back-off pulling image",
				}},
			}},
		},
	}
}

func TestRunImagePullCheckPersistent(t *testing.T) {
	for _, initContainer := range []bool{false, true} {
		name := "main"
		if initContainer {
			name = "init"
		}
		t.Run(name, func(t *testing.T) {
			pod := imagePullTestPod()
			if initContainer {
				pod.Status.InitContainerStatuses = pod.Status.ContainerStatuses
				pod.Status.ContainerStatuses = nil
			}
			now := time.Unix(1000, 0)
			client := fake.NewClientset()
			client.PrependReactor("list", "pods", func(action clienttesting.Action) (bool, runtime.Object, error) {
				selector := action.(clienttesting.ListAction).GetListRestrictions().Labels.String()
				if selector != "pipeline/runid=run-id" || action.GetNamespace() != "kubeflow" {
					t.Fatalf("unscoped pod lookup: %v", action)
				}
				return true, &corev1.PodList{Items: []corev1.Pod{pod}}, nil
			})
			client.PrependReactor("list", "events", func(action clienttesting.Action) (bool, runtime.Object, error) {
				selector := action.(clienttesting.ListAction).GetListRestrictions().Fields.String()
				if selector != "involvedObject.uid=pod-uid" {
					t.Fatalf("unscoped event lookup: %v", action)
				}
				return true, &corev1.EventList{Items: []corev1.Event{
					{InvolvedObject: corev1.ObjectReference{UID: pod.UID}, Reason: "Failed", Message: "Failed to pull image public.ecr.aws/docker/library/python:3.12: 429 Too Many Requests: Data limit exceeded"},
					{InvolvedObject: corev1.ObjectReference{UID: "another-pod"}, Reason: "Failed", Message: "public.ecr.aws/docker/library/python:3.12: unrelated failure"},
				}}, nil
			})
			check := newRunImagePullCheck(client, "kubeflow", "run-id", func() time.Time { return now })
			if err := check(); err != nil {
				t.Fatalf("first observation must allow recovery: %v", err)
			}
			now = now.Add(imagePullFailureGrace - time.Second)
			if err := check(); err != nil {
				t.Fatalf("failed before grace elapsed: %v", err)
			}
			now = now.Add(time.Second)
			err := check()
			if err == nil {
				t.Fatal("expected persistent image-pull error")
			}
			for _, want := range []string{"kubeflow/task", "container main", "python:3.12", "ImagePullBackOff", "Back-off pulling image", "429 Too Many Requests", "Data limit exceeded"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("missing %q in %v", want, err)
				}
			}
			if strings.Contains(err.Error(), "unrelated failure") {
				t.Errorf("reported another pod's failure: %v", err)
			}
		})
	}
}

func TestRunImagePullCheckObservationResets(t *testing.T) {
	for _, change := range []string{"running", "creating", "gone", "replaced", "container changed", "image changed", "completed", "list error"} {
		t.Run(change, func(t *testing.T) {
			original := imagePullTestPod()
			pod := original.DeepCopy()
			now := time.Unix(1000, 0)
			var listErr error
			client := fake.NewClientset()
			client.PrependReactor("list", "pods", func(clienttesting.Action) (bool, runtime.Object, error) {
				list := &corev1.PodList{}
				if pod != nil {
					list.Items = []corev1.Pod{*pod}
				}
				return true, list, listErr
			})
			check := newRunImagePullCheck(client, "kubeflow", "run-id", func() time.Time { return now })
			if err := check(); err != nil {
				t.Fatal(err)
			}
			now = now.Add(imagePullFailureGrace)
			switch change {
			case "running":
				pod.Status.ContainerStatuses[0].State = corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}
			case "creating":
				pod.Status.ContainerStatuses[0].State.Waiting.Reason = "ContainerCreating"
			case "gone":
				pod = nil
			case "replaced":
				pod.UID = "replacement"
			case "container changed":
				pod.Status.ContainerStatuses[0].Name = "other-container"
			case "image changed":
				pod.Status.ContainerStatuses[0].Image = "different:image"
			case "completed":
				pod.Status.Phase = corev1.PodFailed
			case "list error":
				listErr = errors.New("API unavailable")
			}
			if err := check(); err != nil {
				t.Fatalf("%s must not count as continuous pull failure: %v", change, err)
			}
			pod = original.DeepCopy()
			listErr = nil
			now = now.Add(time.Second)
			if err := check(); err != nil {
				t.Fatalf("new failure must receive a fresh grace period: %v", err)
			}
		})
	}
}

func TestRunImagePullCheckEventsUnavailable(t *testing.T) {
	now := time.Unix(1000, 0)
	pod := imagePullTestPod()
	pod.Status.ContainerStatuses[0].State.Waiting = &corev1.ContainerStateWaiting{Reason: "ErrImagePull", Message: "registry returned 429"}
	client := fake.NewClientset()
	client.PrependReactor("list", "pods", func(clienttesting.Action) (bool, runtime.Object, error) {
		return true, &corev1.PodList{Items: []corev1.Pod{pod}}, nil
	})
	client.PrependReactor("list", "events", func(clienttesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("forbidden")
	})
	check := newRunImagePullCheck(client, "kubeflow", "run-id", func() time.Time { return now })
	if err := check(); err != nil {
		t.Fatal(err)
	}
	pod.Status.ContainerStatuses[0].State.Waiting = &corev1.ContainerStateWaiting{Reason: "ImagePullBackOff", Message: "Back-off pulling image"}
	now = now.Add(imagePullFailureGrace)
	err := check()
	if err == nil || !strings.Contains(err.Error(), "previous ErrImagePull: registry returned 429") || !strings.Contains(err.Error(), "ImagePullBackOff: Back-off pulling image") || !strings.Contains(err.Error(), "events unavailable: forbidden") {
		t.Fatalf("must retain observed pull failure without events: %v", err)
	}
}
