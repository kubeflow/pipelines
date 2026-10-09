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
	"testing"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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
	require.Error(t, RegisterManagedScheduleWriter(context.Background(), handoffClient(deployments, pods), "kubeflow", pods[0].Name))
}
