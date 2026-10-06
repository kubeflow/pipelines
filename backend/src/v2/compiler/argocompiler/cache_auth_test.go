// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package argocompiler

import (
	"encoding/json"
	"testing"

	wfapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
	k8score "k8s.io/api/core/v1"
)

func TestCompileCacheCredentialsModes(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()
	var spec map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(`{"pipelineInfo":{"name":"cache-auth"},"root":{"dag":{"tasks":{"task":{"taskInfo":{"name":"task"},"componentRef":{"name":"comp"}}}}},"components":{"comp":{"executorLabel":"exec"}},"deploymentSpec":{"executors":{"exec":{"container":{"image":"busybox","command":["true"]}}}},"schemaVersion":"2.1.0","sdkVersion":"kfp-2.17.0"}`), &spec))
	ps, err := structpb.NewStruct(spec)
	require.NoError(t, err)
	for _, tc := range []struct {
		name            string
		multi, disabled bool
		volumes         int
	}{
		{"multi-user", true, false, 1}, {"single-user", false, false, 0}, {"disabled", true, true, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			viper.Set(common.MultiUserMode, tc.multi)
			defer viper.Set(common.MultiUserMode, false)
			wf, err := Compile(&pipelinespec.PipelineJob{PipelineSpec: ps}, nil, &Options{CacheDisabled: tc.disabled})
			require.NoError(t, err)
			var found int
			for _, volume := range wf.Spec.Volumes {
				if volume.Name == "kfp-cache-api-token" {
					found++
				}
			}
			require.Equal(t, tc.volumes, found)
			if found == 1 {
				var containers int
				for _, tmpl := range wf.Spec.Templates {
					if tmpl.Container != nil {
						containers++
						require.Contains(t, tmpl.Container.VolumeMounts, k8score.VolumeMount{Name: "kfp-cache-api-token", MountPath: "/var/run/secrets/kubeflow/cache-api", ReadOnly: true})
					}
				}
				require.GreaterOrEqual(t, containers, 2)
			}
		})
	}
}

func TestCacheAPICredentials(t *testing.T) {
	viper.Set(common.TokenReviewAudience, "custom.kfp")
	defer viper.Set(common.TokenReviewAudience, nil)
	wf := &wfapi.Workflow{Spec: wfapi.WorkflowSpec{Templates: []wfapi.Template{
		{Name: "driver", Container: &k8score.Container{VolumeMounts: []k8score.VolumeMount{{Name: "sdk-token", MountPath: "/var/run/secrets/kubeflow/pipelines"}}, Env: []k8score.EnvVar{{Name: "KF_PIPELINES_SA_TOKEN_PATH", Value: "/custom/sdk-token"}}}},
		{Name: "launcher", Container: &k8score.Container{}},
		{Name: "dag", DAG: &wfapi.DAGTemplate{}},
	}}}
	require.NoError(t, addCacheAPICredentials(wf))
	require.NoError(t, addCacheAPICredentials(wf), "credential injection must be idempotent")
	require.Len(t, wf.Spec.Volumes, 1)
	projection := wf.Spec.Volumes[0].Projected.Sources[0].ServiceAccountToken
	require.Equal(t, "custom.kfp", projection.Audience)
	require.Equal(t, "token", projection.Path)
	require.Equal(t, k8score.VolumeMount{Name: "sdk-token", MountPath: "/var/run/secrets/kubeflow/pipelines"}, wf.Spec.Templates[0].Container.VolumeMounts[0])
	require.Equal(t, "/custom/sdk-token", wf.Spec.Templates[0].Container.Env[0].Value)
	require.Len(t, wf.Spec.Templates[0].Container.VolumeMounts, 2)
	require.Equal(t, []k8score.VolumeMount{{Name: "kfp-cache-api-token", MountPath: "/var/run/secrets/kubeflow/cache-api", ReadOnly: true}}, wf.Spec.Templates[1].Container.VolumeMounts)
	wf.Spec.Volumes[0].Projected.Sources[0].ServiceAccountToken.Audience = "wrong-audience"
	require.Error(t, addCacheAPICredentials(wf))
	wf.Spec.Volumes = nil
	wf.Spec.Templates = []wfapi.Template{{Name: "shadow", Container: &k8score.Container{}, Volumes: []k8score.Volume{{Name: "kfp-cache-api-token", VolumeSource: k8score.VolumeSource{EmptyDir: &k8score.EmptyDirVolumeSource{}}}}}}
	require.ErrorContains(t, addCacheAPICredentials(wf), "template shadow reserved volume")
	wf.Spec.Volumes[0].Name = "custom-cache-token"
	wf.Spec.Templates[0].Volumes[0].Name = "custom-cache-token"
	wf.Spec.Templates[0].Container.VolumeMounts = []k8score.VolumeMount{{Name: "custom-cache-token", MountPath: "/var/run/secrets/kubeflow/cache-api"}}
	require.ErrorContains(t, addCacheAPICredentials(wf), "template shadow mount")
}
