// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ktesting "k8s.io/client-go/testing"
)

func renameWriter(deployment *appsv1.Deployment, pod *corev1.Pod, name string) {
	deployment.Name = name
	deployment.UID = types.UID(name)
	pod.OwnerReferences[0].Name = name + "-rs"
	pod.OwnerReferences[0].UID = types.UID(name + "-rs")
}

func TestScheduleWriterAutomaticDiscovery(t *testing.T) {
	for _, scenario := range []string{"standard", "renamed", "zero replicas", "old API", "old controller", "terminating", "missing API", "missing controller", "fake label", "wrong owner UID", "missing container", "lost template label", "conflicting role hints", "extra unowned pod", "old renamed peer", "retired renamed peer", "drifted terminating pod", "unrelated pod"} {
		t.Run(scenario, func(t *testing.T) {
			t.Setenv(scheduleWritersEnv, "")
			deployments, pods := handoffFixtures()
			renameWriter(deployments[0], pods[0], "custom-api")
			renameWriter(deployments[1], pods[1], "custom-controller")
			switch scenario {
			case "standard":
				deployments, pods = handoffFixtures()
			case "zero replicas":
				for _, deployment := range deployments {
					*deployment.Spec.Replicas = 0
					deployment.Status.Replicas = 0
					deployment.Status.UpdatedReplicas = 0
				}
				pods = nil
			case "old API":
				pods[0].Annotations = nil
			case "old controller":
				pods[1].Annotations = nil
			case "terminating":
				now := metav1.Now()
				pods[0].DeletionTimestamp = &now
			case "missing API":
				deployments = deployments[1:]
				pods = pods[1:]
			case "missing controller":
				deployments = deployments[:1]
				pods = pods[:1]
			case "fake label":
				pods[0].OwnerReferences = nil
			case "wrong owner UID":
				pods[0].OwnerReferences[0].UID = "fake"
			case "missing container":
				deployments[0].Spec.Template.Spec.Containers[0].Name = "changed-container"
			case "lost template label":
				deployments[0].Spec.Template.Labels["app"] = "changed-label"
			case "conflicting role hints":
				deployments[0].Labels = map[string]string{"app": "ml-pipeline-scheduledworkflow"}
			case "drifted terminating pod":
				extra := pods[0].DeepCopy()
				extra.Name = "old-app-drift"
				extra.UID = "old-app-drift"
				extra.Labels["app"] = "drifted"
				extra.Annotations = nil
				now := metav1.Now()
				extra.DeletionTimestamp = &now
				pods = append(pods, extra)
			case "unrelated pod":
				pods = append(pods, &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "unrelated", Namespace: "kubeflow", Labels: map[string]string{"app": "unrelated"}}, Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "unrelated"}}}})
			case "extra unowned pod":
				extra := pods[0].DeepCopy()
				extra.Name = "unowned-old-api"
				extra.OwnerReferences = nil
				pods = append(pods, extra)
			case "old renamed peer", "retired renamed peer":
				oldDeployment, oldPod := deployments[0].DeepCopy(), pods[0].DeepCopy()
				renameWriter(oldDeployment, oldPod, "previous-api")
				oldPod.Name = "previous-api-pod"
				oldPod.UID = "previous-api-pod"
				oldPod.Annotations = nil
				if scenario == "retired renamed peer" {
					*oldDeployment.Spec.Replicas = 0
					oldDeployment.Status.Replicas = 0
					oldDeployment.Status.UpdatedReplicas = 0
				} else {
					pods = append(pods, oldPod)
				}
				deployments = append(deployments, oldDeployment)
			}
			err := ManagedScheduleWritersReady(context.Background(), handoffClient(deployments, pods), "kubeflow")
			switch scenario {
			case "standard", "renamed", "zero replicas", "retired renamed peer", "unrelated pod":
				require.NoError(t, err)
			default:
				require.Error(t, err)
			}
			if scenario == "missing API" || scenario == "missing controller" {
				require.False(t, apierrors.IsBadRequest(err), "missing peer must wait for installation")
			}
		})
	}
}

func TestScheduleWriterDiscoveryDoesNotHideMalformedOldWriter(t *testing.T) {
	for _, hint := range []string{"name", "metadata", "selector", "expression", "container"} {
		t.Run(hint, func(t *testing.T) {
			t.Setenv(scheduleWritersEnv, "")
			deployments, pods := handoffFixtures()
			renameWriter(deployments[0], pods[0], "custom-api")
			old := deployments[0].DeepCopy()
			old.Name = "malformed-old-api"
			old.UID = "malformed-old-api"
			if hint != "container" {
				old.Spec.Template.Spec.Containers[0].Name = "unknown"
			}
			old.Spec.Template.Labels = map[string]string{"app": "unknown"}
			old.Spec.Selector = &metav1.LabelSelector{MatchLabels: map[string]string{"app": "unknown"}}
			switch hint {
			case "name":
				old.Name = "ml-pipeline"
			case "metadata":
				old.Labels = map[string]string{"app": "ml-pipeline"}
			case "selector":
				old.Spec.Selector.MatchLabels["app"] = "ml-pipeline"
			case "expression":
				old.Spec.Selector = &metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{{Key: "app", Operator: metav1.LabelSelectorOpIn, Values: []string{"ml-pipeline"}}}}
			}
			_, _, err := discoverScheduleWriters(context.Background(), handoffClient(append(deployments, old), pods), "kubeflow")
			require.ErrorContains(t, err, "inconsistent schedule-writer")
			require.ErrorContains(t, err, scheduleWritersEnv)
		})
	}
}

func TestScheduleWriterDiscoveryRechecksInventory(t *testing.T) {
	for _, change := range []string{"added", "removed", "replaced", "updated"} {
		t.Run(change, func(t *testing.T) {
			t.Setenv(scheduleWritersEnv, "")
			deployments, pods := handoffFixtures()
			client := handoffClient(deployments, pods)
			calls := 0
			client.PrependReactor("list", "deployments", func(action ktesting.Action) (bool, runtime.Object, error) {
				calls++
				if calls != 2 {
					return false, nil, nil
				}
				changed := deployments[0].DeepCopy()
				switch change {
				case "added":
					changed.Name = "new-api-peer"
					changed.UID = "new-api-peer"
					require.NoError(t, client.Tracker().Create(appsv1.SchemeGroupVersion.WithResource("deployments"), changed, "kubeflow"))
				case "removed":
					require.NoError(t, client.Tracker().Delete(appsv1.SchemeGroupVersion.WithResource("deployments"), changed.Namespace, changed.Name))
				case "replaced":
					changed.UID = "replacement"
					require.NoError(t, client.Tracker().Update(appsv1.SchemeGroupVersion.WithResource("deployments"), changed, "kubeflow"))
				case "updated":
					changed.ResourceVersion = "2"
					require.NoError(t, client.Tracker().Update(appsv1.SchemeGroupVersion.WithResource("deployments"), changed, "kubeflow"))
				}
				return false, nil, nil
			})
			require.Error(t, ManagedScheduleWritersReady(context.Background(), client, "kubeflow"))
			require.Equal(t, 2, calls)
		})
	}
}

func TestScheduleWriterExplicitSharedRoleLabel(t *testing.T) {
	t.Setenv(scheduleWritersEnv, "")
	deployments, pods := handoffFixtures()
	config := defaultScheduleWriters()
	old := deployments[0].DeepCopy()
	old.Name = "previous-api"
	old.UID = "previous-api"
	*old.Spec.Replicas = 0
	old.Status.Replicas = 0
	old.Status.UpdatedReplicas = 0
	config = append(config, scheduleWriterIdentity{Role: "api", Deployment: old.Name, App: "ml-pipeline", Container: "ml-pipeline-api-server"})
	setScheduleWriters(t, config)
	for _, pod := range pods {
		claim, err := scheduleWriterClaim(pod)
		require.NoError(t, err)
		pod.Annotations[scheduleWriterAnnotation] = claim
	}
	require.NoError(t, ManagedScheduleWritersReady(context.Background(), handoffClient(append(deployments, old), pods), "kubeflow"))
}

func TestScheduleWriterMissingPeerReplicaSetWaits(t *testing.T) {
	t.Setenv(scheduleWritersEnv, "")
	deployments, pods := handoffFixtures()
	client := handoffClient(deployments, pods)
	require.NoError(t, client.AppsV1().ReplicaSets("kubeflow").Delete(context.Background(), pods[1].OwnerReferences[0].Name, metav1.DeleteOptions{}))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	waited := false
	err := WaitForManagedScheduleWriters(ctx, client, "kubeflow", pods[0].Name, func(err error) {
		require.ErrorContains(t, err, "missing ReplicaSet")
		require.False(t, apierrors.IsNotFound(err))
		waited = true
		cancel()
	})
	require.True(t, waited, "peer ReplicaSet deletion is a rollout wait, not a fatal self identity failure")
	require.ErrorIs(t, err, context.Canceled)
}
