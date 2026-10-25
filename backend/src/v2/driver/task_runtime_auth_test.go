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

package driver

import (
	"testing"

	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestContainerTaskCredentialPatchCollision(t *testing.T) {
	t.Setenv(util.TaskTokenPathEnv, util.TaskTokenPath)
	pod := &corev1.PodSpec{Containers: []corev1.Container{{Name: "main", Image: "user-image", Command: []string{"user-command"}, Env: []corev1.EnvVar{{Name: util.TaskTokenPathEnv, Value: "/user/token"}}}}}
	_, err := marshalTaskPodSpecPatch(pod)
	require.Error(t, err)
	pod.Containers[0].Env = []corev1.EnvVar{{Name: util.TaskTokenPathEnv, Value: util.TaskTokenPath}, {Name: "KF_PIPELINES_SA_TOKEN_PATH", Value: "/custom/sdk/token"}}
	raw, err := marshalTaskPodSpecPatch(pod)
	require.NoError(t, err)
	require.Contains(t, string(raw), "/custom/sdk/token")
	pod.Containers[0].Env = nil
	raw, err = marshalTaskPodSpecPatch(pod)
	require.NoError(t, err)
	require.Contains(t, string(raw), "user-image")
	require.Contains(t, string(raw), "user-command")
	pod.Containers[0].Env = []corev1.EnvVar{{Name: util.TaskTokenPathEnv, Value: "/user/token"}}
	t.Setenv(util.TaskTokenPathEnv, "")
	_, err = marshalTaskPodSpecPatch(pod)
	require.NoError(t, err)
}
