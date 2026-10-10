// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
)

func handoffFixtures() ([]*appsv1.Deployment, []*corev1.Pod) {
	var deployments []*appsv1.Deployment
	var pods []*corev1.Pod
	for _, name := range scheduleWriterDeployments {
		replicas := int32(1)
		deployments = append(deployments, &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "kubeflow", UID: types.UID(name), ResourceVersion: "1", Generation: 1},
			Spec:       appsv1.DeploymentSpec{Replicas: &replicas},
			Status:     appsv1.DeploymentStatus{ObservedGeneration: 1, Replicas: 1, UpdatedReplicas: 1},
		})
		uid := types.UID(name + "-pod")
		pods = append(pods, &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
			Name: string(uid), Namespace: "kubeflow", UID: uid, ResourceVersion: "1",
			Labels:      map[string]string{"app": name},
			Annotations: map[string]string{scheduleWriterAnnotation: scheduleWriterProtocol + ":" + string(uid)},
		}})
	}
	for _, pod := range pods {
		name := pod.Labels["app"]
		if name == "ml-pipeline" {
			name = "ml-pipeline-api-server"
		}
		pod.Spec.Containers = []corev1.Container{{Name: name, Image: "registry/writer:fixed"}}
		pod.Status.ContainerStatuses = []corev1.ContainerStatus{{Name: name, Image: "registry/writer:fixed", ImageID: "sha256:fixed", ContainerID: "containerd://" + pod.Name, State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}}}
		claim, err := scheduleWriterClaim(pod)
		if err != nil {
			panic(err)
		}
		pod.Annotations[scheduleWriterAnnotation] = claim
	}
	return deployments, pods
}

func handoffClient(deployments []*appsv1.Deployment, pods []*corev1.Pod) *fake.Clientset {
	var objects []runtime.Object
	for _, deployment := range deployments {
		objects = append(objects, deployment)
	}
	for _, pod := range pods {
		objects = append(objects, pod)
	}
	return fake.NewSimpleClientset(objects...)
}

func TestManagedScheduleWritersFence(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func([]*appsv1.Deployment, []*corev1.Pod)
		ready  bool
	}{
		{name: "compatible rollout", ready: true},
		{name: "old API", mutate: func(_ []*appsv1.Deployment, p []*corev1.Pod) { p[0].Annotations = nil }},
		{name: "old controller", mutate: func(_ []*appsv1.Deployment, p []*corev1.Pod) { p[1].Annotations = nil }},
		{name: "stale UID claim", mutate: func(_ []*appsv1.Deployment, p []*corev1.Pod) {
			p[0].Annotations[scheduleWriterAnnotation] = scheduleWriterProtocol + ":old"
		}},
		{name: "terminating compatible Pod", mutate: func(_ []*appsv1.Deployment, p []*corev1.Pod) { now := metav1.Now(); p[1].DeletionTimestamp = &now }},
		{name: "unobserved rollout", mutate: func(d []*appsv1.Deployment, _ []*corev1.Pod) { d[0].Generation++ }},
		{name: "old replicas retained", mutate: func(d []*appsv1.Deployment, _ []*corev1.Pod) { d[1].Status.Replicas++ }},
		{name: "updated replica missing", mutate: func(d []*appsv1.Deployment, _ []*corev1.Pod) { d[0].Status.UpdatedReplicas = 0 }},
		{name: "desired count not met", mutate: func(d []*appsv1.Deployment, _ []*corev1.Pod) { *d[0].Spec.Replicas = 2 }},
		{name: "zero desired with remaining Pod", mutate: func(d []*appsv1.Deployment, _ []*corev1.Pod) {
			*d[1].Spec.Replicas = 0
			d[1].Status.Replicas = 0
			d[1].Status.UpdatedReplicas = 0
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			deployments, pods := handoffFixtures()
			if test.mutate != nil {
				test.mutate(deployments, pods)
			}
			err := ManagedScheduleWritersReady(context.Background(), handoffClient(deployments, pods), "kubeflow")
			if test.ready {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestManagedScheduleWritersEmptyDeploymentAndMixedPods(t *testing.T) {
	deployments, pods := handoffFixtures()
	*deployments[1].Spec.Replicas = 0
	deployments[1].Status.Replicas = 0
	deployments[1].Status.UpdatedReplicas = 0
	require.NoError(t, ManagedScheduleWritersReady(context.Background(), handoffClient(deployments, pods[:1]), "kubeflow"))
	deployments, pods = handoffFixtures()
	old := pods[0].DeepCopy()
	old.Name = "old-api"
	old.UID = "old-api"
	old.Annotations = nil
	require.Error(t, ManagedScheduleWritersReady(context.Background(), handoffClient(deployments, append(pods, old)), "kubeflow"))
	require.Error(t, ManagedScheduleWritersReady(context.Background(), handoffClient(deployments, pods[:1]), "kubeflow"))
}

func TestManagedScheduleWritersRechecksDeployment(t *testing.T) {
	deployments, pods := handoffFixtures()
	client := handoffClient(deployments, pods)
	calls := 0
	client.PrependReactor("get", "deployments", func(action ktesting.Action) (bool, runtime.Object, error) {
		calls++
		if calls == 3 {
			changed := deployments[0].DeepCopy()
			changed.ResourceVersion = "2"
			return true, changed, nil
		}
		return false, nil, nil
	})
	require.ErrorContains(t, ManagedScheduleWritersReady(context.Background(), client, "kubeflow"), "changed during handoff")
}

func TestRegisterManagedScheduleWriterUsesUIDAndVersionTests(t *testing.T) {
	deployments, pods := handoffFixtures()
	pods[0].Annotations = map[string]string{"unrelated": "preserved"}
	client := handoffClient(deployments, pods)
	require.NoError(t, RegisterManagedScheduleWriter(context.Background(), client, "kubeflow", pods[0].Name))
	var operations []map[string]interface{}
	found := false
	for _, action := range client.Actions() {
		if action.GetVerb() == "patch" {
			found = true
			patch := action.(ktesting.PatchAction)
			require.Equal(t, types.JSONPatchType, patch.GetPatchType())
			require.NoError(t, json.Unmarshal(patch.GetPatch(), &operations))
		}
	}
	require.True(t, found)
	require.Equal(t, "test", operations[0]["op"])
	require.Equal(t, "/metadata/uid", operations[0]["path"])
	require.Equal(t, string(pods[0].UID), operations[0]["value"])
	require.Equal(t, "test", operations[1]["op"])
	require.Equal(t, "/metadata/resourceVersion", operations[1]["path"])
	updated, err := client.CoreV1().Pods("kubeflow").Get(context.Background(), pods[0].Name, metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, "preserved", updated.Annotations["unrelated"])
	claim, err := scheduleWriterClaim(pods[0])
	require.NoError(t, err)
	require.Equal(t, claim, updated.Annotations[scheduleWriterAnnotation])
	require.NoError(t, ManagedScheduleWritersReady(context.Background(), client, "kubeflow"))
}

func TestRegisterManagedScheduleWriterRejectsReplacement(t *testing.T) {
	deployments, pods := handoffFixtures()
	pods[0].Annotations = nil
	client := handoffClient(deployments, pods)
	client.PrependReactor("patch", "pods", func(action ktesting.Action) (bool, runtime.Object, error) {
		replacement := pods[0].DeepCopy()
		replacement.UID = "replacement"
		require.NoError(t, client.Tracker().Update(corev1.SchemeGroupVersion.WithResource("pods"), replacement, "kubeflow"))
		return false, nil, nil
	})
	require.Error(t, RegisterManagedScheduleWriter(context.Background(), client, "kubeflow", pods[0].Name))
}

func TestManagedScheduleWriterWaitRegistersBeforeFence(t *testing.T) {
	deployments, pods := handoffFixtures()
	pods[1].Annotations = nil
	client := handoffClient(deployments, pods)
	require.NoError(t, WaitForManagedScheduleWriters(context.Background(), client, "kubeflow", pods[1].Name, nil))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.Error(t, WaitForManagedScheduleWriters(ctx, client, "", "", nil))
}

func TestManagedScheduleWritersRejectsInheritedContainerClaim(t *testing.T) {
	for _, mutation := range []func(*corev1.Pod){
		func(p *corev1.Pod) { p.UID = "replacement-pod" },
		func(p *corev1.Pod) { p.Spec.Containers[0].Image = "registry/writer:old" },
		func(p *corev1.Pod) {
			p.Spec.Containers[0].Image = "registry/writer:old"
			p.Status.ContainerStatuses[0].Image = "registry/writer:old"
		},
		func(p *corev1.Pod) { p.Status.ContainerStatuses[0].ContainerID = "containerd://replacement" },
		func(p *corev1.Pod) { p.Status.ContainerStatuses[0].ImageID = "sha256:replacement" },
		func(p *corev1.Pod) { p.Status.ContainerStatuses[0].RestartCount++ },
		func(p *corev1.Pod) { p.Status.ContainerStatuses[0].State.Running = nil },
	} {
		deployments, pods := handoffFixtures()
		mutation(pods[0])
		require.Error(t, ManagedScheduleWritersReady(context.Background(), handoffClient(deployments, pods), "kubeflow"))
	}
	deployments, pods := handoffFixtures()
	pods[0].Spec.Containers[0].Image = "registry/writer:old"
	client := handoffClient(deployments, pods)
	require.Error(t, ManagedScheduleWritersReady(context.Background(), client, "kubeflow"))
	// A compatible process can attest its current incarnation again. The
	// requested image is hashed, but its name need not match runtime reporting.
	require.NoError(t, RegisterManagedScheduleWriter(context.Background(), client, "kubeflow", pods[0].Name))
	require.NoError(t, ManagedScheduleWritersReady(context.Background(), client, "kubeflow"))
}

func TestManagedScheduleWriterHandoffCachesOnlySuccess(t *testing.T) {
	deployments, pods := handoffFixtures()
	client := handoffClient(deployments, pods)
	ready := NewManagedScheduleWriterHandoff(client, "kubeflow", pods[0].Name)
	fail := true
	client.PrependReactor("get", "deployments", func(ktesting.Action) (bool, runtime.Object, error) {
		if fail {
			return true, nil, apierrors.NewServiceUnavailable("temporary outage")
		}
		return false, nil, nil
	})
	require.Error(t, ready(context.Background()))
	fail = false
	require.NoError(t, ready(context.Background()))
	calls := len(client.Actions())
	fail = true
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); require.NoError(t, ready(context.Background())) }()
	}
	wg.Wait()
	require.Len(t, client.Actions(), calls, "completed handoff must not read Kubernetes again")
	// A replacement process must perform its own handoff.
	require.Error(t, NewManagedScheduleWriterHandoff(client, "kubeflow", pods[0].Name)(context.Background()))
}

func TestManagedScheduleWriterConfigurationFailsWithoutPolling(t *testing.T) {
	for _, test := range []struct {
		name, namespace, pod string
		forbidden            bool
	}{
		{name: "namespace missing", pod: "controller"},
		{name: "name missing", namespace: "kubeflow"},
		{name: "wrong pod name", namespace: "kubeflow", pod: "custom-hostname"},
		{name: "RBAC missing", namespace: "kubeflow", pod: "ml-pipeline-scheduledworkflow-pod", forbidden: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			deployments, pods := handoffFixtures()
			client := handoffClient(deployments, pods)
			if test.forbidden {
				client.PrependReactor("get", "pods", func(ktesting.Action) (bool, runtime.Object, error) {
					return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: "pods"}, test.pod, fmt.Errorf("denied"))
				})
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			err := WaitForManagedScheduleWriters(ctx, client, test.namespace, test.pod, func(error) { t.Error("configuration error entered retry loop") })
			require.Error(t, err)
			require.NoError(t, ctx.Err(), "configuration errors must return before polling timeout")
			require.Contains(t, err.Error(), "POD_NAME")
		})
	}
}

func TestManagedScheduleWriterTransientFailureStillWaits(t *testing.T) {
	deployments, pods := handoffFixtures()
	client := handoffClient(deployments, pods)
	client.PrependReactor("get", "pods", func(ktesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewServiceUnavailable("retry")
	})
	ctx, cancel := context.WithCancel(context.Background())
	waits := 0
	err := WaitForManagedScheduleWriters(ctx, client, "kubeflow", pods[0].Name, func(error) { waits++; cancel() })
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, 1, waits)
}

func TestManagedScheduleWriterRegistrationRaceRetries(t *testing.T) {
	deployments, pods := handoffFixtures()
	pods[0].Annotations = nil
	client := handoffClient(deployments, pods)
	client.PrependReactor("patch", "pods", func(ktesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewInvalid(schema.GroupKind{Kind: "Pod"}, pods[0].Name,
			field.ErrorList{field.Invalid(field.NewPath("metadata", "resourceVersion"), "1", "JSON Patch test failed")})
	})
	ctx, cancel := context.WithCancel(context.Background())
	waits := 0
	err := WaitForManagedScheduleWriters(ctx, client, "kubeflow", pods[0].Name, func(err error) {
		require.True(t, apierrors.IsConflict(err))
		waits++
		cancel()
	})
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, 1, waits)
}

func TestManagedScheduleWriterHandoffCanceledWaiter(t *testing.T) {
	deployments, pods := handoffFixtures()
	client := handoffClient(deployments, pods)
	ready := NewManagedScheduleWriterHandoff(client, "kubeflow", pods[0].Name)
	started, release := make(chan struct{}), make(chan struct{})
	var first sync.Once
	client.PrependReactor("get", "deployments", func(ktesting.Action) (bool, runtime.Object, error) {
		first.Do(func() { close(started); <-release })
		return false, nil, nil
	})
	done := make(chan error, 1)
	go func() { done <- ready(context.Background()) }()
	<-started
	defer func() { close(release); require.NoError(t, <-done) }()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	waiter := make(chan error, 1)
	go func() { waiter <- ready(ctx) }()
	select {
	case err := <-waiter:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Error("canceled caller waited for another caller's Kubernetes request")
	}
}

func TestManagedScheduleWritersIgnoreCompletedPods(t *testing.T) {
	for _, phase := range []corev1.PodPhase{corev1.PodFailed, corev1.PodSucceeded} {
		t.Run(string(phase), func(t *testing.T) {
			deployments, pods := handoffFixtures()
			completed := pods[0].DeepCopy()
			completed.Name = "leftover-terminal-pod"
			completed.UID = "leftover"
			completed.Annotations = nil
			completed.Status.Phase = phase
			completed.Status.Reason = "Evicted"
			completed.Status.ContainerStatuses[0].State = corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{}}
			pods = append(pods, completed)
			client := handoffClient(deployments, pods)
			require.NoError(t, ManagedScheduleWritersReady(context.Background(), client, "kubeflow"))
			require.Error(t, RegisterManagedScheduleWriter(context.Background(), client, "kubeflow", completed.Name))
			// The same leftover is a fence if it is not terminal.
			completed.Status.Phase = corev1.PodPending
			require.Error(t, ManagedScheduleWritersReady(context.Background(), handoffClient(deployments, pods), "kubeflow"))
		})
	}
}

func TestManagedScheduleWriterRuntimeImageNames(t *testing.T) {
	for _, test := range []struct{ name, requested, reported string }{
		{"short image", "kfp-api-server:2.18.0", "docker.io/library/kfp-api-server:2.18.0"},
		{"tag resolved to digest", "mirror.example/kfp:release", "mirror.example/kfp@sha256:resolved"},
		{"digest pin", "ghcr.io/kubeflow/kfp-api-server@sha256:pinned", "runtime-cache/kfp:resolved"},
	} {
		t.Run(test.name, func(t *testing.T) {
			deployments, pods := handoffFixtures()
			pods[0].Annotations = nil
			pods[0].Spec.Containers[0].Image = test.requested
			pods[0].Status.ContainerStatuses[0].Image = test.reported
			client := handoffClient(deployments, pods)
			require.NoError(t, RegisterManagedScheduleWriter(context.Background(), client, "kubeflow", pods[0].Name))
			require.NoError(t, ManagedScheduleWritersReady(context.Background(), client, "kubeflow"))
			pod, err := client.CoreV1().Pods("kubeflow").Get(context.Background(), pods[0].Name, metav1.GetOptions{})
			require.NoError(t, err)
			pod.Status.ContainerStatuses[0].ContainerID = "containerd://replacement"
			_, err = client.CoreV1().Pods("kubeflow").UpdateStatus(context.Background(), pod, metav1.UpdateOptions{})
			require.NoError(t, err)
			require.Error(t, ManagedScheduleWritersReady(context.Background(), client, "kubeflow"))
		})
	}
}

func TestManagedScheduleWriterRegistrationRequiresRuntimeIdentity(t *testing.T) {
	for _, mutate := range []func(*corev1.Pod){
		func(p *corev1.Pod) { p.Status.ContainerStatuses[0].ImageID = "" },
		func(p *corev1.Pod) { p.Status.ContainerStatuses[0].ContainerID = "" },
		func(p *corev1.Pod) { p.Status.ContainerStatuses[0].State.Running = nil },
	} {
		deployments, pods := handoffFixtures()
		pods[0].Annotations = nil
		mutate(pods[0])
		require.Error(t, RegisterManagedScheduleWriter(context.Background(), handoffClient(deployments, pods), "kubeflow", pods[0].Name))
	}
}

func TestManagedScheduleWriterWaitsForMissingDeployment(t *testing.T) {
	for _, missing := range scheduleWriterDeployments {
		t.Run(missing, func(t *testing.T) {
			t.Parallel()
			deployments, pods := handoffFixtures()
			client := handoffClient(deployments, pods)
			deployment, err := client.AppsV1().Deployments("kubeflow").Get(context.Background(), missing, metav1.GetOptions{})
			require.NoError(t, err)
			require.NoError(t, client.AppsV1().Deployments("kubeflow").Delete(context.Background(), missing, metav1.DeleteOptions{}))
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			waits := 0
			err = WaitForManagedScheduleWriters(ctx, client, "kubeflow", pods[1].Name, func(err error) {
				waits++
				require.Contains(t, err.Error(), missing)
				require.Contains(t, err.Error(), "waiting for installation or rollout")
				_, createErr := client.AppsV1().Deployments("kubeflow").Create(ctx, deployment, metav1.CreateOptions{})
				require.NoError(t, createErr)
			})
			require.NoError(t, err, "handoff must recover without restarting the controller")
			require.Equal(t, 1, waits)
		})
	}
}

func TestManagedScheduleWriterDeploymentDisappearsDuringRecheck(t *testing.T) {
	deployments, pods := handoffFixtures()
	client := handoffClient(deployments, pods)
	reads := 0
	client.PrependReactor("get", "deployments", func(action ktesting.Action) (bool, runtime.Object, error) {
		reads++
		if reads == len(scheduleWriterDeployments)+1 {
			name := action.(ktesting.GetAction).GetName()
			return true, nil, apierrors.NewNotFound(schema.GroupResource{Group: "apps", Resource: "deployments"}, name)
		}
		return false, nil, nil
	})
	ctx, cancel := context.WithCancel(context.Background())
	waits := 0
	err := WaitForManagedScheduleWriters(ctx, client, "kubeflow", pods[1].Name, func(err error) {
		require.Contains(t, err.Error(), "disappeared during handoff")
		waits++
		cancel()
	})
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, 1, waits)
}
