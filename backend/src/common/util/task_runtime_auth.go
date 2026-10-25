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
	"fmt"
	"path"
	"reflect"
	"slices"
	"strings"

	wfapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"sigs.k8s.io/yaml"
)

const (
	// TaskTokenPathEnv selects the projected workload credential used for task writes.
	TaskTokenPathEnv = "KFP_TASK_API_TOKEN_PATH"
	// TaskTokenPath is separate from user SDK credentials and /var mounts.
	TaskTokenPath      = "/kfp-task-api/token"
	taskTokenDirectory = "/kfp-task-api"
	taskTokenVolume    = "kfp-task-api-token"
)

// TaskRuntimeAuthConfig describes the currently configured credential-aware runtimes.
// Images and commands are recognition criteria, never replacements for saved runtimes.
type TaskRuntimeAuthConfig struct {
	Audience, DriverImage, LauncherImage string
	DriverCommand, LauncherCommand       []string
}

func taskTokenProjection(audience string) corev1.Volume {
	expiration := int64(3600)
	mode := int32(0644)
	return corev1.Volume{Name: taskTokenVolume, VolumeSource: corev1.VolumeSource{Projected: &corev1.ProjectedVolumeSource{DefaultMode: &mode, Sources: []corev1.VolumeProjection{{ServiceAccountToken: &corev1.ServiceAccountTokenProjection{Audience: audience, ExpirationSeconds: &expiration, Path: "token"}}}}}}
}

func argumentValue(args []string, name string) string {
	for i, arg := range args {
		if arg == name && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

// taskRuntime returns the binary container for a task writer, or nil for a DAG/importer.
func taskRuntime(t *wfapi.Template) (*corev1.Container, error) {
	if t.Script != nil || t.ContainerSet != nil || t.Resource != nil || t.Data != nil || t.HTTP != nil || t.Plugin != nil || t.Suspend != nil {
		return nil, fmt.Errorf("template %q has an unsupported task runtime layout", t.Name)
	}
	if t.Container == nil {
		if t.DAG != nil || t.Steps != nil {
			return nil, nil
		}
		return nil, fmt.Errorf("template %q has an unsupported task runtime layout", t.Name)
	}
	switch t.Metadata.Annotations[AnnotationKeyRuntimeRole] {
	case string(ExecutionRuntimeRoleDriver):
		switch argumentValue(t.Container.Args, "--type") {
		case "CONTAINER":
			return t.Container, nil
		case "{{inputs.parameters.driver-type}}", "DAG", "ROOT_DAG":
			return nil, nil
		}
	case string(ExecutionRuntimeRoleLauncher):
		if argumentValue(t.Container.Args, "--executor_type") == "importer" {
			return nil, nil
		}
		for i := range t.InitContainers {
			init := &t.InitContainers[i].Container
			if init.Name == "kfp-launcher" && reflect.DeepEqual(init.Args, []string{"--copy", "/kfp-launcher/launch"}) {
				mount := func(c *corev1.Container) bool {
					count := 0
					for _, m := range c.VolumeMounts {
						if m.Name == "kfp-launcher" || shadowsDirectory(m.MountPath, "/kfp-launcher") {
							if m.Name != "kfp-launcher" || m.MountPath != "/kfp-launcher" || m.SubPath != "" || m.SubPathExpr != "" {
								return false
							}
							count++
						}
					}
					return count == 1
				}
				if mount(init) && mount(t.Container) && slices.ContainsFunc(t.Volumes, func(v corev1.Volume) bool {
					return v.Name == "kfp-launcher" && v.EmptyDir != nil && reflect.DeepEqual(v.VolumeSource, corev1.VolumeSource{EmptyDir: v.EmptyDir})
				}) {
					return init, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("template %q has an unsupported task runtime layout", t.Name)
}

func shadowsDirectory(mountPath, directory string) bool {
	p := path.Clean(mountPath)
	return p == directory || strings.HasPrefix(directory, p+"/") || strings.HasPrefix(p, directory+"/") || p == "/" || strings.Contains(p, "{{")
}

// ValidateTaskRuntimePodPatch reserves credential fields for the compiler.
// Call after dynamic user/plugin configuration has been resolved, before publishing it.
// These wiring checks prevent accidental shadowing and preserve retry compatibility;
// they do not isolate the token from user code sharing the launcher container.
func ValidateTaskRuntimePodPatch(patch *corev1.PodSpec) error {
	for _, v := range patch.Volumes {
		if v.Name == taskTokenVolume {
			return fmt.Errorf("pod patch overrides reserved task credential volume %q", taskTokenVolume)
		}
	}
	for _, c := range patch.Containers {
		if c.Name != "main" {
			continue
		}
		for _, e := range c.Env {
			if e.Name == TaskTokenPathEnv && !reflect.DeepEqual(e, corev1.EnvVar{Name: TaskTokenPathEnv, Value: TaskTokenPath}) {
				return fmt.Errorf("pod patch overrides reserved task credential environment %q", TaskTokenPathEnv)
			}
		}
		for _, m := range c.VolumeMounts {
			if m.Name == taskTokenVolume || shadowsDirectory(m.MountPath, taskTokenDirectory) {
				return fmt.Errorf("pod patch shadows the reserved task credential mount")
			}
		}
	}
	return nil
}

func ensureTaskToken(t *wfapi.Template, audience string, add bool) (bool, error) {
	volume := taskTokenProjection(audience)
	mount := corev1.VolumeMount{Name: taskTokenVolume, MountPath: taskTokenDirectory, ReadOnly: true}
	env := corev1.EnvVar{Name: TaskTokenPathEnv, Value: TaskTokenPath}
	volumes, mounts, envs := 0, 0, 0
	for _, v := range t.Volumes {
		if v.Name == taskTokenVolume {
			volumes++
			if !reflect.DeepEqual(v, volume) {
				return false, fmt.Errorf("conflicting task credential volume")
			}
		}
	}
	for _, m := range t.Container.VolumeMounts {
		if m.Name == taskTokenVolume || shadowsDirectory(m.MountPath, taskTokenDirectory) {
			mounts++
			if !reflect.DeepEqual(m, mount) {
				return false, fmt.Errorf("conflicting task credential mount")
			}
		}
	}
	for _, e := range t.Container.Env {
		if e.Name == TaskTokenPathEnv {
			envs++
			if !reflect.DeepEqual(e, env) {
				return false, fmt.Errorf("conflicting task credential environment")
			}
		}
	}
	if volumes > 1 || mounts > 1 || envs > 1 {
		return false, fmt.Errorf("duplicate task credential declaration")
	}
	complete := volumes == 1 && mounts == 1 && envs == 1
	if add {
		if volumes == 0 {
			t.Volumes = append(t.Volumes, volume)
		}
		if mounts == 0 {
			t.Container.VolumeMounts = append(t.Container.VolumeMounts, mount)
		}
		if envs == 0 {
			t.Container.Env = append(t.Container.Env, env)
		}
	}
	return complete, nil
}

// EnsureTaskRuntimeCredentials adds only missing credential declarations to recognized
// current runtimes. It preserves completed state and commits nothing on failure.
func (w *Workflow) EnsureTaskRuntimeCredentials(config TaskRuntimeAuthConfig) error {
	if config.Audience == "" {
		return nil
	}
	copy := NewWorkflow(w.DeepCopy())
	if err := copy.checkTaskRuntimeStructure(); err != nil {
		return err
	}
	for i := range copy.Spec.Templates {
		t := &copy.Spec.Templates[i]
		runtime, err := taskRuntime(t)
		if err != nil {
			return err
		}
		if runtime == nil {
			continue
		}
		complete, err := ensureTaskToken(t, config.Audience, false)
		if err != nil {
			return fmt.Errorf("template %q: %w", t.Name, err)
		}
		if !complete {
			image, command := config.DriverImage, config.DriverCommand
			if runtime != t.Container {
				image, command = config.LauncherImage, config.LauncherCommand
			}
			if runtime.Image != image || !reflect.DeepEqual(runtime.Command, command) {
				return fmt.Errorf("template %q uses an older or custom runtime without task credentials", t.Name)
			}
			if _, err := ensureTaskToken(t, config.Audience, true); err != nil {
				return err
			}
		}
	}
	if err := copy.ValidateTaskRuntimeCredentials(config.Audience); err != nil {
		return err
	}
	*w.Workflow = *copy.Workflow
	return nil
}

func (w *Workflow) checkTaskRuntimeStructure() error {
	if w.Status.StoredWorkflowSpec != nil || len(w.Status.StoredTemplates) > 0 {
		return fmt.Errorf("retained external task runtime templates cannot be prepared")
	}
	// Compiler runtimes are top-level templates. Do not silently skip custom
	// inline runtimes when deciding whether a saved workflow can be retried.
	topLevel := map[*wfapi.Template]bool{w.Spec.TemplateDefaults: true}
	for i := range w.Spec.Templates {
		topLevel[&w.Spec.Templates[i]] = true
	}
	if err := w.walkTemplates(func(t *wfapi.Template) error {
		if !topLevel[t] {
			return fmt.Errorf("inline task runtime templates cannot be prepared")
		}
		return nil
	}); err != nil {
		return err
	}
	for _, v := range w.Spec.Volumes {
		if v.Name == taskTokenVolume {
			return fmt.Errorf("workflow overrides reserved task credential volume")
		}
	}
	return nil
}

// ValidateTaskRuntimeCredentials checks the final templates and saved pod patches.
func (w *Workflow) ValidateTaskRuntimeCredentials(audience string) error {
	if audience == "" {
		return nil
	}
	if err := w.checkTaskRuntimeStructure(); err != nil {
		return err
	}
	for i := range w.Spec.Templates {
		t := &w.Spec.Templates[i]
		runtime, err := taskRuntime(t)
		if err != nil {
			return err
		}
		if runtime == nil {
			continue
		}
		complete, err := ensureTaskToken(t, audience, false)
		if err != nil {
			return err
		}
		if !complete {
			return fmt.Errorf("template %q is missing task credentials", t.Name)
		}
		base := corev1.PodSpec{Containers: []corev1.Container{*t.Container.DeepCopy()}, Volumes: t.Volumes}
		base.Containers[0].Name = "main"
		for _, init := range t.InitContainers {
			base.InitContainers = append(base.InitContainers, init.Container)
		}
		if defaults := w.Spec.TemplateDefaults; defaults != nil {
			// Defaults must not shadow reserved fields; template-owned fields otherwise win.
			dp := corev1.PodSpec{Volumes: defaults.Volumes}
			if defaults.Container != nil {
				dp.Containers = []corev1.Container{*defaults.Container.DeepCopy()}
				dp.Containers[0].Name = "main"
			}
			if err := ValidateTaskRuntimePodPatch(&dp); err != nil {
				return err
			}
		}
		// Argo inherits the default only when the template has no patch of
		// its own, then applies workflow followed by effective template patch.
		templatePatch := t.PodSpecPatch
		if templatePatch == "" && w.Spec.TemplateDefaults != nil {
			templatePatch = w.Spec.TemplateDefaults.PodSpecPatch
		}
		patches := []string{w.Spec.PodSpecPatch, templatePatch}
		if err := validateTaskPatches(base, patches, audience, runtime == t.Container); err != nil {
			return fmt.Errorf("template %q: %w", t.Name, err)
		}
		if runtime != t.Container {
			for _, node := range w.Status.Nodes {
				if node.Outputs == nil {
					continue
				}
				for _, p := range node.Outputs.Parameters {
					if p.Name == "pod-spec-patch" && p.Value != nil && p.Value.String() != "" {
						if err := validateTaskPatches(base, append(append([]string{}, patches...), p.Value.String()), audience, false); err != nil {
							return fmt.Errorf("saved driver output conflicts with task credentials: %w", err)
						}
					}
				}
			}
		}
	}
	return nil
}

func validateTaskPatches(base corev1.PodSpec, patches []string, audience string, driver bool) error {
	raw, err := json.Marshal(base)
	if err != nil {
		return err
	}
	for _, p := range patches {
		if strings.TrimSpace(p) == "" || strings.TrimSpace(p) == compilerGeneratedPodSpecPatchTemplate {
			continue
		}
		patch, err := yaml.YAMLToJSON([]byte(p))
		if err != nil {
			return fmt.Errorf("invalid task runtime pod patch: %w", err)
		}
		raw, err = strategicpatch.StrategicMergePatch(raw, patch, corev1.PodSpec{})
		if err != nil {
			return fmt.Errorf("invalid task runtime pod patch: %w", err)
		}
	}
	var pod corev1.PodSpec
	if err := json.Unmarshal(raw, &pod); err != nil {
		return err
	}
	for _, c := range pod.Containers {
		if c.Name != "main" {
			continue
		}
		tmpl := wfapi.Template{Container: &c, Volumes: pod.Volumes}
		complete, err := ensureTaskToken(&tmpl, audience, false)
		if err != nil {
			return err
		}
		if !complete {
			return fmt.Errorf("pod patch removes task credentials")
		}
		if driver && (c.Image != base.Containers[0].Image || !reflect.DeepEqual(c.Command, base.Containers[0].Command)) {
			return fmt.Errorf("pod patch changes the credential-aware driver")
		}
		if !driver {
			// Validate the handoff after merging patches: the init binary can
			// remain intact while main loses access to its shared executable.
			tmpl.Metadata.Annotations = map[string]string{AnnotationKeyRuntimeRole: string(ExecutionRuntimeRoleLauncher)}
			for _, init := range pod.InitContainers {
				tmpl.InitContainers = append(tmpl.InitContainers, wfapi.UserContainer{Container: init})
			}
			if runtime, err := taskRuntime(&tmpl); err != nil || runtime == nil {
				return fmt.Errorf("pod patch changes the launcher shared volume or mounts")
			}
			for _, init := range base.InitContainers {
				if init.Name != "kfp-launcher" {
					continue
				}
				for _, actual := range pod.InitContainers {
					if actual.Name == init.Name && actual.Image == init.Image && reflect.DeepEqual(actual.Command, init.Command) && reflect.DeepEqual(actual.Args, init.Args) && reflect.DeepEqual(actual.VolumeMounts, init.VolumeMounts) {
						return nil
					}
				}
				return fmt.Errorf("pod patch changes the credential-aware launcher")
			}
		}
		return nil
	}
	return fmt.Errorf("pod patch removes the task container")
}
