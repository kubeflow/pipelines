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
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func setScheduleWriters(t *testing.T, config scheduleWriterConfiguration) {
	t.Helper()
	data, err := json.Marshal(config)
	require.NoError(t, err)
	t.Setenv(scheduleWritersEnv, string(data))
}

func TestScheduleWriterConfigurationValidation(t *testing.T) {
	for _, raw := range []string{"[]", "null", "{}", "[] []", `[ {"role":"api", "deployment":"api", "app":"api", "container":"api", "typo":true} ]`} {
		t.Run(raw, func(t *testing.T) {
			t.Setenv(scheduleWritersEnv, raw)
			_, err := configuredScheduleWriters()
			require.Error(t, err)
		})
	}
	for _, test := range []struct {
		name   string
		modify func(scheduleWriterConfiguration) scheduleWriterConfiguration
	}{
		{"missing controller", func(c scheduleWriterConfiguration) scheduleWriterConfiguration { return c[:1] }},
		{"invalid role", func(c scheduleWriterConfiguration) scheduleWriterConfiguration { c[0].Role = "other"; return c }},
		{"duplicate deployment", func(c scheduleWriterConfiguration) scheduleWriterConfiguration {
			c[1].Deployment = c[0].Deployment
			return c
		}},
		{"duplicate label", func(c scheduleWriterConfiguration) scheduleWriterConfiguration { c[1].App = c[0].App; return c }},
		{"empty container", func(c scheduleWriterConfiguration) scheduleWriterConfiguration { c[0].Container = ""; return c }},
		{"empty label", func(c scheduleWriterConfiguration) scheduleWriterConfiguration { c[0].App = ""; return c }},
		{"invalid name", func(c scheduleWriterConfiguration) scheduleWriterConfiguration {
			c[0].Deployment = "../wrong"
			return c
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			setScheduleWriters(t, test.modify(defaultScheduleWriters()))
			_, err := configuredScheduleWriters()
			require.Error(t, err)
		})
	}
}

func TestCustomScheduleWriterHandoff(t *testing.T) {
	for _, scenario := range []string{"ready", "old writer", "terminating writer", "label-only pod", "wrong deployment owner", "wrong replicaset uid", "selector mismatch", "config disagreement", "missing peer", "retired empty", "retired writer"} {
		t.Run(scenario, func(t *testing.T) {
			t.Setenv(scheduleWritersEnv, "")
			deployments, pods := handoffFixtures()
			config := defaultScheduleWriters()
			var replicasets []*appsv1.ReplicaSet
			for i := range config {
				config[i].Deployment = "custom-" + config[i].Role
				config[i].App = "writer-" + config[i].Role
				config[i].Container = "process-" + config[i].Role
				deployments[i].Name = config[i].Deployment
				deployments[i].Spec.Selector = &metav1.LabelSelector{MatchLabels: map[string]string{"app": config[i].App}}
				pods[i].Labels["app"] = config[i].App
				pods[i].Spec.Containers[0].Name = config[i].Container
				pods[i].Status.ContainerStatuses[0].Name = config[i].Container
				controller := true
				rs := &appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{Name: config[i].Deployment + "-rs", Namespace: "kubeflow", UID: types.UID(config[i].Deployment + "-rs"), OwnerReferences: []metav1.OwnerReference{{APIVersion: "apps/v1", Kind: "Deployment", Name: deployments[i].Name, UID: deployments[i].UID, Controller: &controller}}}}
				pods[i].OwnerReferences = []metav1.OwnerReference{{APIVersion: "apps/v1", Kind: "ReplicaSet", Name: rs.Name, UID: rs.UID, Controller: &controller}}
				replicasets = append(replicasets, rs)
			}
			if scenario == "retired empty" || scenario == "retired writer" {
				config = append(config, scheduleWriterIdentity{Role: "api", Deployment: "retiring-api", App: "retiring-api", Container: "old-api"})
				zero := int32(0)
				retired := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "retiring-api", Namespace: "kubeflow", UID: "retiring-api", ResourceVersion: "1", Generation: 1}, Spec: appsv1.DeploymentSpec{Replicas: &zero}, Status: appsv1.DeploymentStatus{ObservedGeneration: 1}}
				deployments = append(deployments, retired)
			}
			setScheduleWriters(t, config)
			for _, pod := range pods {
				claim, err := scheduleWriterClaim(pod)
				require.NoError(t, err)
				pod.Annotations[scheduleWriterAnnotation] = claim
			}
			switch scenario {
			case "retired writer":
				old := pods[0].DeepCopy()
				old.Name = "retiring-api-pod"
				old.UID = "retiring-api-pod"
				old.Labels["app"] = "retiring-api"
				old.Annotations = nil
				pods = append(pods, old)
			case "old writer":
				pods[0].Annotations = nil
			case "terminating writer":
				now := metav1.Now()
				pods[0].DeletionTimestamp = &now
			case "label-only pod":
				pods[0].OwnerReferences = nil
			case "wrong deployment owner":
				replicasets[0].OwnerReferences[0].UID = "other"
			case "wrong replicaset uid":
				replicasets[0].UID = "replacement"
			case "selector mismatch":
				deployments[0].Spec.Selector.MatchLabels["extra"] = "required"
			case "config disagreement":
				config[1].Container = "different"
				setScheduleWriters(t, config)
			case "missing peer":
				deployments = deployments[:1]
			}
			client := handoffClient(deployments, pods)
			for _, rs := range replicasets {
				_, err := client.AppsV1().ReplicaSets("kubeflow").Create(context.Background(), rs, metav1.CreateOptions{})
				require.NoError(t, err)
			}
			err := ManagedScheduleWritersReady(context.Background(), client, "kubeflow")
			if scenario == "ready" || scenario == "retired empty" {
				require.NoError(t, err)
				require.NoError(t, RegisterManagedScheduleWriter(context.Background(), client, "kubeflow", pods[0].Name))
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestScheduleWriterConfigurationOrderPreservesClaim(t *testing.T) {
	t.Setenv(scheduleWritersEnv, "")
	_, pods := handoffFixtures()
	original, err := scheduleWriterClaim(pods[0])
	require.NoError(t, err)
	config := defaultScheduleWriters()
	config[0], config[1] = config[1], config[0]
	setScheduleWriters(t, config)
	explicit, err := scheduleWriterClaim(pods[0])
	require.NoError(t, err)
	require.Equal(t, original, explicit, "explicit default config must preserve existing fixed-writer protocol")
}
