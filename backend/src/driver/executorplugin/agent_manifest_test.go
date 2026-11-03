// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package executorplugin_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	wf "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"sigs.k8s.io/yaml"
)

func manifestConfig(t *testing.T, path string) corev1.ConfigMap {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("../../../../manifests/kustomize", path))
	require.NoError(t, err)
	var config corev1.ConfigMap
	require.NoError(t, yaml.Unmarshal(data, &config))
	return config
}

func workflowPatch(t *testing.T, config corev1.ConfigMap) string {
	t.Helper()
	var workflow wf.Workflow
	require.NoError(t, yaml.Unmarshal([]byte(config.Data["workflowDefaults"]), &workflow))
	return workflow.Spec.PodSpecPatch
}

// Argo's ApplyPodSpecPatch uses the Kubernetes strategic merge engine over PodSpec.
func applyPodPatch(t *testing.T, pod corev1.PodSpec, patch string) corev1.PodSpec {
	t.Helper()
	original, err := json.Marshal(pod)
	require.NoError(t, err)
	patchJSON, err := yaml.YAMLToJSON([]byte(patch))
	require.NoError(t, err)
	merged, err := strategicpatch.StrategicMergePatch(original, patchJSON, corev1.PodSpec{})
	require.NoError(t, err)
	var result corev1.PodSpec
	require.NoError(t, json.Unmarshal(merged, &result))
	return result
}

func agentContainerSecurity() *corev1.SecurityContext {
	uid, nonRoot, escalation := int64(8737), true, false
	return &corev1.SecurityContext{RunAsUser: &uid, RunAsNonRoot: &nonRoot, AllowPrivilegeEscalation: &escalation}
}

func TestOpenShiftAgentUIDPatch(t *testing.T) {
	config := manifestConfig(t, "env/openshift/base/patches/driver-agent-controller.yaml")
	sidecarConfig := manifestConfig(t, "env/openshift/base/patches/driver-agent-plugin.yaml")
	var sidecar corev1.Container
	require.NoError(t, yaml.Unmarshal([]byte(sidecarConfig.Data["sidecar.container"]), &sidecar))
	for _, agent := range []bool{false, true} {
		uid, nonRoot := int64(8737), true
		pod := corev1.PodSpec{
			SecurityContext: &corev1.PodSecurityContext{RunAsUser: &uid, RunAsNonRoot: &nonRoot},
			InitContainers:  []corev1.Container{{Name: "init", Image: "argoexec", SecurityContext: agentContainerSecurity()}},
			Containers:      []corev1.Container{{Name: "main", Image: "executor", SecurityContext: agentContainerSecurity()}},
		}
		if agent {
			pod.Containers = append(pod.Containers, sidecar)
		}
		patched := applyPodPatch(t, pod, workflowPatch(t, config))
		require.Nil(t, patched.SecurityContext.RunAsUser)
		require.True(t, *patched.SecurityContext.RunAsNonRoot)
		require.Len(t, patched.Containers, len(pod.Containers), "do not add a driver container to workloads")
		require.Len(t, patched.InitContainers, 1)
		for _, container := range append(patched.InitContainers, patched.Containers...) {
			require.NotEmpty(t, container.Image)
			require.Nil(t, container.SecurityContext.RunAsUser, container.Name)
			require.True(t, *container.SecurityContext.RunAsNonRoot)
			require.False(t, *container.SecurityContext.AllowPrivilegeEscalation)
		}
	}
	require.Contains(t, config.Data["initlessPod"], "enabled: false")
}

func TestAgentTrustPatchProjectsOnlyPublicCA(t *testing.T) {
	config := manifestConfig(t, "env/cert-manager/platform-agnostic-standalone-tls/patches/driver-agent-trust.yaml")
	name := "argo-workflows-agent-ca-certificates"
	pod := corev1.PodSpec{
		Volumes:        []corev1.Volume{{Name: name, VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: name}}}},
		InitContainers: []corev1.Container{{Name: "init", VolumeMounts: []corev1.VolumeMount{{Name: name, MountPath: "/etc/ssl/certs", ReadOnly: true}}}},
		Containers:     []corev1.Container{{Name: "main"}, {Name: "driver-plugin", VolumeMounts: []corev1.VolumeMount{{Name: name, MountPath: "/kfp/certs", ReadOnly: true}}}},
	}
	patched := applyPodPatch(t, pod, workflowPatch(t, config))
	require.Equal(t, []corev1.KeyToPath{{Key: "ca.crt", Path: "ca.crt"}}, patched.Volumes[0].Secret.Items)
	// Key selection remains public-only when cert-manager rotates the same Secret.
	for _, ca := range []string{"old-public-ca", "rotated-public-ca"} {
		secret := map[string]string{"ca.crt": ca, "tls.crt": "serving-cert", "tls.key": "private-key"}
		projected := map[string]string{}
		for _, item := range patched.Volumes[0].Secret.Items {
			projected[item.Path] = secret[item.Key]
		}
		require.Equal(t, map[string]string{"ca.crt": ca}, projected)
	}
	for _, container := range append(patched.InitContainers, patched.Containers...) {
		for _, mount := range container.VolumeMounts {
			require.Empty(t, mount.SubPath, "directory mounts must receive Secret rotation")
		}
	}
}
