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

package util

import (
	"encoding/json"
	"path"
	"testing"

	wfapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func taskAuthFixture() (*Workflow, TaskRuntimeAuthConfig) {
	config := TaskRuntimeAuthConfig{Audience: "pipelines", DriverImage: "driver:patched", LauncherImage: "launcher:patched", DriverCommand: []string{"driver"}, LauncherCommand: []string{"launcher-v2"}}
	w := NewWorkflow(&wfapi.Workflow{Spec: wfapi.WorkflowSpec{ServiceAccountName: "custom-runner", Templates: []wfapi.Template{
		{Name: "driver", Metadata: wfapi.Metadata{Annotations: map[string]string{AnnotationKeyRuntimeRole: "driver"}}, Container: &corev1.Container{Image: config.DriverImage, Command: config.DriverCommand, Args: []string{"--type", "CONTAINER"}}},
		{Name: "executor", Metadata: wfapi.Metadata{Annotations: map[string]string{AnnotationKeyRuntimeRole: "launcher"}}, Container: &corev1.Container{Image: "user-image", Command: []string{"user-command"}, VolumeMounts: []corev1.VolumeMount{{Name: "kfp-launcher", MountPath: "/kfp-launcher"}}}, InitContainers: []wfapi.UserContainer{{Container: corev1.Container{Name: "kfp-launcher", Image: config.LauncherImage, Command: config.LauncherCommand, Args: []string{"--copy", "/kfp-launcher/launch"}, VolumeMounts: []corev1.VolumeMount{{Name: "kfp-launcher", MountPath: "/kfp-launcher"}}}}}, Volumes: []corev1.Volume{{Name: "kfp-launcher", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}}}, PodSpecPatch: "{{inputs.parameters.pod-spec-patch}}"},
	}}})
	return w, config
}

func TestTaskRuntimeCredentials(t *testing.T) {
	w, config := taskAuthFixture()
	original := w.DeepCopy()
	require.NoError(t, w.EnsureTaskRuntimeCredentials(config))
	require.NoError(t, w.ValidateTaskRuntimeCredentials(config.Audience))
	require.Equal(t, original.Spec.ServiceAccountName, w.Spec.ServiceAccountName)
	for i, tmpl := range w.Spec.Templates {
		require.Equal(t, original.Spec.Templates[i].Container.Image, tmpl.Container.Image)
		require.Equal(t, original.Spec.Templates[i].Container.Command, tmpl.Container.Command)
		require.Contains(t, tmpl.Container.Env, corev1.EnvVar{Name: TaskTokenPathEnv, Value: TaskTokenPath})
		require.Contains(t, tmpl.Container.VolumeMounts, corev1.VolumeMount{Name: "kfp-task-api-token", MountPath: "/kfp-task-api", ReadOnly: true})
		v := tmpl.Volumes[len(tmpl.Volumes)-1]
		token := v.Projected.Sources[0].ServiceAccountToken
		require.Equal(t, "pipelines", token.Audience)
		require.Equal(t, int64(3600), *token.ExpirationSeconds)
		require.Equal(t, "token", token.Path)
	}
	prepared := w.DeepCopy()
	require.NoError(t, w.EnsureTaskRuntimeCredentials(config))
	require.Equal(t, prepared, w.Workflow)
	w.Spec.Templates[0].Container.Image = "custom:already-authenticated"
	require.NoError(t, w.EnsureTaskRuntimeCredentials(config))
	require.Equal(t, "custom:already-authenticated", w.Spec.Templates[0].Container.Image)
}

func TestTaskRuntimeCredentialsRejectBeforeMutation(t *testing.T) {
	for name, change := range map[string]func(*Workflow){
		"old runtime": func(w *Workflow) { w.Spec.Templates[1].InitContainers[0].Image = "launcher:old" },
		"token env": func(w *Workflow) {
			w.Spec.Templates[1].Container.Env = []corev1.EnvVar{{Name: TaskTokenPathEnv, Value: "/user/token"}}
		},
		"shadow mount": func(w *Workflow) {
			w.Spec.Templates[1].Container.VolumeMounts = append(w.Spec.Templates[1].Container.VolumeMounts, corev1.VolumeMount{Name: "user", MountPath: "/kfp-task-api/token"})
		},
		"stored template": func(w *Workflow) { w.Status.StoredTemplates = map[string]wfapi.Template{"old": w.Spec.Templates[1]} },
		"saved output": func(w *Workflow) {
			w.Status.Nodes = wfapi.Nodes{"driver": wfapi.NodeStatus{Outputs: &wfapi.Outputs{Parameters: []wfapi.Parameter{{Name: "pod-spec-patch", Value: wfapi.AnyStringPtr(`{"containers":[{"name":"main","env":[{"name":"KFP_TASK_API_TOKEN_PATH","value":"/other"}]}]}`)}}}}}
		},
		"literal deletion": func(w *Workflow) {
			w.Spec.PodSpecPatch = `{"containers":[{"name":"main","volumeMounts":[{"$patch":"replace"}]}]}`
		},
	} {
		t.Run(name, func(t *testing.T) {
			w, config := taskAuthFixture()
			change(w)
			before := w.DeepCopy()
			require.Error(t, w.EnsureTaskRuntimeCredentials(config))
			require.Equal(t, before, w.Workflow)
		})
	}
	w, config := taskAuthFixture()
	config.Audience = ""
	before := w.DeepCopy()
	require.NoError(t, w.EnsureTaskRuntimeCredentials(config))
	require.Equal(t, before, w.Workflow)
}

func TestTaskRuntimePodPatch(t *testing.T) {
	require.NoError(t, ValidateTaskRuntimePodPatch(&corev1.PodSpec{Containers: []corev1.Container{{Name: "main", Image: "user", Command: []string{"/kfp-launcher/launch", "--execution_id", "42"}}}}))
	require.Error(t, ValidateTaskRuntimePodPatch(&corev1.PodSpec{Containers: []corev1.Container{{Name: "main", Env: []corev1.EnvVar{{Name: TaskTokenPathEnv, Value: "bad"}}}}}))
	require.Error(t, ValidateTaskRuntimePodPatch(&corev1.PodSpec{Volumes: []corev1.Volume{{Name: "kfp-task-api-token"}}}))
	require.Error(t, ValidateTaskRuntimePodPatch(&corev1.PodSpec{Containers: []corev1.Container{{Name: "main", VolumeMounts: []corev1.VolumeMount{{Name: "x", MountPath: "/kfp-task-api"}}}}}))
}

func TestTaskRuntimeCredentialsInlineRejected(t *testing.T) {
	for _, steps := range []bool{false, true} {
		w, config := taskAuthFixture()
		inline := w.Spec.Templates[0].DeepCopy()
		inline.Container.Image = "driver:unsupported"
		root := wfapi.Template{Name: "root"}
		if steps {
			root.Steps = []wfapi.ParallelSteps{{Steps: []wfapi.WorkflowStep{{Name: "inline-driver", Inline: inline}}}}
		} else {
			root.DAG = &wfapi.DAGTemplate{Tasks: []wfapi.DAGTask{{Name: "inline-driver", Inline: inline}}}
		}
		w.Spec.Templates = []wfapi.Template{root}
		before := w.DeepCopy()
		require.ErrorContains(t, w.EnsureTaskRuntimeCredentials(config), "inline")
		require.Equal(t, before, w.Workflow)
		require.ErrorContains(t, w.ValidateTaskRuntimeCredentials(config.Audience), "inline")
	}
}

func TestTaskRuntimeCredentialsUnsupportedExecutable(t *testing.T) {
	for name, change := range map[string]func(*wfapi.Template){
		"script":                     func(t *wfapi.Template) { t.Script = &wfapi.ScriptTemplate{Container: *t.Container}; t.Container = nil },
		"container set":              func(t *wfapi.Template) { t.ContainerSet = &wfapi.ContainerSetTemplate{}; t.Container = nil },
		"resource":                   func(t *wfapi.Template) { t.Resource = &wfapi.ResourceTemplate{Action: "create"}; t.Container = nil },
		"mixed script and container": func(t *wfapi.Template) { t.Script = &wfapi.ScriptTemplate{Container: *t.Container} },
		"empty":                      func(t *wfapi.Template) { t.Container = nil },
	} {
		t.Run(name, func(t *testing.T) {
			w, config := taskAuthFixture()
			change(&w.Spec.Templates[0])
			before := w.DeepCopy()
			require.ErrorContains(t, w.EnsureTaskRuntimeCredentials(config), "unsupported task runtime")
			require.Equal(t, before, w.Workflow)
			require.ErrorContains(t, w.ValidateTaskRuntimeCredentials(config.Audience), "unsupported task runtime")
		})
	}
}

func TestTaskRuntimeCredentialsPreserveSDKCredentials(t *testing.T) {
	for _, tokenPath := range []string{"/var/run/secrets/kubeflow/pipelines/token", "/custom/sdk/token"} {
		t.Run(tokenPath, func(t *testing.T) {
			w, config := taskAuthFixture()
			expiration := int64(7200)
			sdkEnv := corev1.EnvVar{Name: "KF_PIPELINES_SA_TOKEN_PATH", Value: tokenPath}
			sdkMount := corev1.VolumeMount{Name: "volume-kf-pipeline-token", MountPath: path.Dir(tokenPath), ReadOnly: true}
			sdkVolume := corev1.Volume{Name: "volume-kf-pipeline-token", VolumeSource: corev1.VolumeSource{Projected: &corev1.ProjectedVolumeSource{Sources: []corev1.VolumeProjection{{ServiceAccountToken: &corev1.ServiceAccountTokenProjection{Audience: config.Audience, ExpirationSeconds: &expiration, Path: "token"}}}}}}
			for i := range w.Spec.Templates {
				tmpl := &w.Spec.Templates[i]
				tmpl.Container.Env = append(tmpl.Container.Env, sdkEnv)
				tmpl.Container.VolumeMounts = append(tmpl.Container.VolumeMounts, sdkMount)
				tmpl.Volumes = append(tmpl.Volumes, sdkVolume)
			}
			require.NoError(t, w.EnsureTaskRuntimeCredentials(config))
			require.NoError(t, w.ValidateTaskRuntimeCredentials(config.Audience))
			for _, tmpl := range w.Spec.Templates {
				require.Contains(t, tmpl.Container.Env, sdkEnv)
				require.Contains(t, tmpl.Container.VolumeMounts, sdkMount)
				require.Contains(t, tmpl.Volumes, sdkVolume)
			}
			sdkPatch := &corev1.PodSpec{Containers: []corev1.Container{{Name: "main", Env: []corev1.EnvVar{sdkEnv}, VolumeMounts: []corev1.VolumeMount{sdkMount}}}, Volumes: []corev1.Volume{sdkVolume}}
			require.NoError(t, ValidateTaskRuntimePodPatch(sdkPatch))
			raw, err := json.Marshal(sdkPatch)
			require.NoError(t, err)
			w.Spec.Templates[1].PodSpecPatch = string(raw)
			require.NoError(t, w.ValidateTaskRuntimeCredentials(config.Audience))
		})
	}
}

func TestTaskRuntimePodPatchIdenticalPath(t *testing.T) {
	require.NoError(t, ValidateTaskRuntimePodPatch(&corev1.PodSpec{Containers: []corev1.Container{{Name: "main", Env: []corev1.EnvVar{{Name: TaskTokenPathEnv, Value: TaskTokenPath}}}}}))
}

func TestTaskRuntimeCredentialsAllowUserVarMount(t *testing.T) {
	w, config := taskAuthFixture()
	mount := corev1.VolumeMount{Name: "user-var", MountPath: "/var"}
	w.Spec.Templates[1].Container.VolumeMounts = append(w.Spec.Templates[1].Container.VolumeMounts, mount)
	w.Spec.Templates[1].Volumes = append(w.Spec.Templates[1].Volumes, corev1.Volume{Name: "user-var", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}})
	require.NoError(t, w.EnsureTaskRuntimeCredentials(config))
	require.Contains(t, w.Spec.Templates[1].Container.VolumeMounts, mount)
	require.NoError(t, ValidateTaskRuntimePodPatch(&corev1.PodSpec{Containers: []corev1.Container{{Name: "main", VolumeMounts: []corev1.VolumeMount{mount}}}}))
}

func TestTaskRuntimeCredentialsRejectAlteredProjection(t *testing.T) {
	for name, change := range map[string]func(*wfapi.Template){
		"wrong audience": func(t *wfapi.Template) { t.Volumes[0].Projected.Sources[0].ServiceAccountToken.Audience = "other" },
		"static secret": func(t *wfapi.Template) {
			t.Volumes[0].VolumeSource = corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: "other"}}
		},
		"subpath":        func(t *wfapi.Template) { t.Container.VolumeMounts[0].SubPath = "token" },
		"writable mount": func(t *wfapi.Template) { t.Container.VolumeMounts[0].ReadOnly = false },
		"env from secret": func(t *wfapi.Template) {
			t.Container.Env[0] = corev1.EnvVar{Name: TaskTokenPathEnv, ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "other"}, Key: "path"}}}
		},
		"duplicate env": func(t *wfapi.Template) { t.Container.Env = append(t.Container.Env, t.Container.Env[0]) },
	} {
		t.Run(name, func(t *testing.T) {
			w, config := taskAuthFixture()
			require.NoError(t, w.EnsureTaskRuntimeCredentials(config))
			change(&w.Spec.Templates[0])
			before := w.DeepCopy()
			require.Error(t, w.EnsureTaskRuntimeCredentials(config))
			require.Equal(t, before, w.Workflow)
			require.Error(t, w.ValidateTaskRuntimeCredentials(config.Audience))
		})
	}
}

func TestTaskRuntimeCredentialsImageVersionBoundary(t *testing.T) {
	w, config := taskAuthFixture()
	require.NoError(t, w.EnsureTaskRuntimeCredentials(config))
	prepared := w.DeepCopy()
	config.DriverImage = "driver:next"
	config.LauncherImage = "launcher:next"
	require.NoError(t, w.EnsureTaskRuntimeCredentials(config))
	require.Equal(t, prepared, w.Workflow)
	old, _ := taskAuthFixture()
	before := old.DeepCopy()
	require.Error(t, old.EnsureTaskRuntimeCredentials(config))
	require.Equal(t, before, old.Workflow)
}

func TestTaskRuntimeCredentialsDefaultPatchPrecedence(t *testing.T) {
	good := `{"containers":[{"name":"main","env":[{"name":"KFP_TASK_API_TOKEN_PATH","value":"/kfp-task-api/token"}]}]}`
	bad := `{"containers":[{"name":"main","env":[{"name":"KFP_TASK_API_TOKEN_PATH","value":"/missing/token"}]}]}`
	for _, tc := range []struct {
		name, defaults, workflow, template string
		reject                             bool
	}{
		{name: "inherited default wins", defaults: bad, workflow: good, reject: true},
		{name: "inherited default repairs workflow", defaults: good, workflow: bad},
		{name: "own patch replaces default", defaults: bad, template: `{}`},
		{name: "own patch overrides inherited repair", defaults: good, workflow: bad, template: `{}`, reject: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, config := taskAuthFixture()
			w.Spec.Templates = w.Spec.Templates[:1]
			w.Spec.TemplateDefaults = &wfapi.Template{PodSpecPatch: tc.defaults}
			w.Spec.PodSpecPatch = tc.workflow
			w.Spec.Templates[0].PodSpecPatch = tc.template
			before := w.DeepCopy()
			err := w.EnsureTaskRuntimeCredentials(config)
			if tc.reject {
				require.Error(t, err)
				require.Equal(t, before, w.Workflow)
			} else {
				require.NoError(t, err)
				require.NoError(t, w.ValidateTaskRuntimeCredentials(config.Audience))
			}
		})
	}
}

func TestTaskRuntimeCredentialsLauncherHandoff(t *testing.T) {
	for name, patch := range map[string]string{
		"remove main mount":     `{"containers":[{"name":"main","volumeMounts":[{"mountPath":"/kfp-launcher","$patch":"delete"}]}]}`,
		"replace main mount":    `{"containers":[{"name":"main","volumeMounts":[{"mountPath":"/kfp-launcher","name":"other"}]}]}`,
		"shadow executable":     `{"containers":[{"name":"main","volumeMounts":[{"mountPath":"/kfp-launcher/launch","name":"other"}]}]}`,
		"remove shared volume":  `{"volumes":[{"name":"kfp-launcher","$patch":"delete"}]}`,
		"replace shared volume": `{"volumes":[{"name":"kfp-launcher","$patch":"replace","secret":{"secretName":"other"}}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			w, config := taskAuthFixture()
			w.Status.Nodes = wfapi.Nodes{"completed-driver": wfapi.NodeStatus{Outputs: &wfapi.Outputs{Parameters: []wfapi.Parameter{{Name: "pod-spec-patch", Value: wfapi.AnyStringPtr(patch)}}}}}
			before := w.DeepCopy()
			require.Error(t, w.EnsureTaskRuntimeCredentials(config))
			require.Equal(t, before, w.Workflow)
		})
	}
}
