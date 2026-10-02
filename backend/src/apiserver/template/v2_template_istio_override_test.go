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

package template

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	argologging "github.com/argoproj/argo-workflows/v4/util/logging"
	argocommon "github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	istioInjectKey            = util.AnnotationKeyIstioSidecarInject
	podRoleLabelKey           = "pipelines.kubeflow.org/pod-role"
	templateNameAnnotationKey = "pipelines.kubeflow.org/template-name"

	containerDriverTemplate = "system-container-driver"
	dagDriverTemplate       = "system-dag-driver"
	importerTemplate        = "system-importer"
)

var (
	podMetadataFixture = istioFixture{path: "testdata/pipeline_with_istio_pod_metadata.yaml"}
	importerFixture    = istioFixture{path: "../../../../test_data/sdk_compiled_pipelines/valid/pipeline_with_importer.yaml"}

	// argoTestContext carries the logger Argo's template substitution expects.
	argoTestContext = argologging.NewSlogLogger(argologging.Info, argologging.Text).NewBackgroundContext()
)

// istioFallbackModes are the two installation settings that matter for a task or driver that
// does not author its own injection value: the opt-in and the upstream default.
var istioFallbackModes = []struct {
	name        string
	environment *string
	want        string
}{
	{"opt-in true", util.StringPointer("true"), util.AnnotationValueIstioSidecarInjectEnabled},
	{"default false", nil, util.AnnotationValueIstioSidecarInjectDisabled},
}

// taskPodMetadata is what pipeline_with_istio_pod_metadata.yaml authors for each root task. The
// tasks differ in injection value and in how many unrelated annotations and labels they carry,
// so each compiles to its own executor template.
var taskPodMetadata = []struct {
	task        string
	annotations map[string]string
	labels      map[string]string
}{
	{
		task:        "injected",
		annotations: map[string]string{istioInjectKey: "true", "team": "ml"},
		// Istio's injector, not KFP, resolves a label that contradicts the annotation.
		labels: map[string]string{istioInjectKey: "false", "tier": "gold"},
	},
	{
		task:        "opted-out",
		annotations: map[string]string{istioInjectKey: "false", "owner": "alice", "cost-center": "42"},
	},
	{
		task:        "default",
		annotations: map[string]string{"note": "plain"},
		labels:      map[string]string{"app": "demo", "env": "test"},
	},
}

type driverPodConfig struct {
	labels      map[string]string
	annotations map[string]string
}

var driverPodConfigScenarios = []struct {
	name   string
	config driverPodConfig
}{
	{"no driver config", driverPodConfig{}},
	{"unrelated metadata only", driverPodConfig{
		labels:      map[string]string{"driver-tier": "infra"},
		annotations: map[string]string{"driver-note": "x", "driver-owner": "platform"},
	}},
	{"annotation opts in", driverPodConfig{
		labels:      map[string]string{"driver-tier": "infra"},
		annotations: map[string]string{istioInjectKey: "true", "driver-note": "x"},
	}},
	{"annotation opts out", driverPodConfig{
		labels:      map[string]string{"driver-tier": "infra"},
		annotations: map[string]string{istioInjectKey: "false", "driver-note": "x"},
	}},
	{"label contradicts annotation", driverPodConfig{
		labels:      map[string]string{istioInjectKey: "true", "driver-tier": "infra"},
		annotations: map[string]string{istioInjectKey: "false", "driver-note": "x"},
	}},
	{"label alone contradicts the fallback", driverPodConfig{
		labels: map[string]string{istioInjectKey: "true"},
	}},
	{"label opts out without an annotation", driverPodConfig{
		labels: map[string]string{istioInjectKey: "false"},
	}},
	{"explicit empty annotation", driverPodConfig{
		annotations: map[string]string{istioInjectKey: "", "driver-note": "x"},
	}},
}

// useDriverPodConfig loads DRIVER_POD_LABELS and DRIVER_POD_ANNOTATIONS the way the API server does
// at startup. The cache lives in package common and cannot be reset directly, so cleanup loads the
// configuration that was in effect before the test back through the same path.
func useDriverPodConfig(t *testing.T, config driverPodConfig) {
	t.Helper()
	var previous driverPodConfig
	if saved := common.GetDriverPodConfig(); saved != nil {
		previous = driverPodConfig{labels: saved.Labels, annotations: saved.Annotations}
	}

	viper.Set(common.DriverPodLabels, mapAsJSON(t, config.labels))
	viper.Set(common.DriverPodAnnotations, mapAsJSON(t, config.annotations))
	require.NoError(t, common.InitDriverPodConfig())
	t.Cleanup(func() {
		viper.Reset()
		viper.Set(common.DriverPodLabels, mapAsJSON(t, previous.labels))
		viper.Set(common.DriverPodAnnotations, mapAsJSON(t, previous.annotations))
		require.NoError(t, common.InitDriverPodConfig())
		viper.Reset()
	})

	loaded := common.GetDriverPodConfig()
	if len(config.labels) == 0 && len(config.annotations) == 0 {
		require.Nil(t, loaded, "an empty driver pod configuration must read as unset")
		return
	}
	require.NotNil(t, loaded, "driver pod configuration was not loaded")
	require.Equal(t, config.labels, loaded.Labels)
	require.Equal(t, config.annotations, loaded.Annotations)
}

func mapAsJSON(t *testing.T, entries map[string]string) string {
	t.Helper()
	if len(entries) == 0 {
		return "{}"
	}
	encoded, err := json.Marshal(entries)
	require.NoError(t, err)
	return string(encoded)
}

func mergeMetadata(layers ...map[string]string) map[string]string {
	merged := map[string]string{}
	for _, layer := range layers {
		for key, value := range layer {
			merged[key] = value
		}
	}
	return merged
}

// withInjectDefault returns annotations with the installation default filled in when the
// annotations do not author their own injection value.
func withInjectDefault(annotations map[string]string, defaultValue string) map[string]string {
	merged := mergeMetadata(annotations)
	if _, authored := merged[istioInjectKey]; !authored {
		merged[istioInjectKey] = defaultValue
	}
	return merged
}

func findTemplate(workflow *util.Workflow, name string) *v1alpha1.Template {
	for index := range workflow.Spec.Templates {
		if workflow.Spec.Templates[index].Name == name {
			return &workflow.Spec.Templates[index]
		}
	}
	return nil
}

func requireTemplate(t *testing.T, workflow *util.Workflow, name string) *v1alpha1.Template {
	t.Helper()
	tmpl := findTemplate(workflow, name)
	require.NotNil(t, tmpl, "compiled workflow has no template %q", name)
	return tmpl
}

func requireDAGTask(t *testing.T, tmpl *v1alpha1.Template, name string) *v1alpha1.DAGTask {
	t.Helper()
	require.NotNil(t, tmpl.DAG, "template %q is not a DAG", tmpl.Name)
	for index := range tmpl.DAG.Tasks {
		if tmpl.DAG.Tasks[index].Name == name {
			return &tmpl.DAG.Tasks[index]
		}
	}
	require.Failf(t, "missing DAG task", "template %q has no task %q", tmpl.Name, name)
	return nil
}

func hasPlaceholder(metadata v1alpha1.Metadata) bool {
	for _, entries := range []map[string]string{metadata.Annotations, metadata.Labels} {
		for key, value := range entries {
			if strings.Contains(key, "{{") || strings.Contains(value, "{{") {
				return true
			}
		}
	}
	return false
}

// resolveWithArgo substitutes a template's input parameters with Argo's own ProcessArgs, which is
// what the workflow controller calls before it creates the pod.
func resolveWithArgo(t *testing.T, tmpl *v1alpha1.Template, arguments *v1alpha1.Arguments) *v1alpha1.Template {
	t.Helper()
	resolved, err := argocommon.ProcessArgs(argoTestContext, tmpl, arguments,
		argocommon.Parameters{}, argocommon.Parameters{}, false, "", nil)
	require.NoError(t, err, "template %q", tmpl.Name)
	return resolved
}

// resolveTaskPodTemplate follows a root task through the two substitutions the controller performs:
// the task's arguments fill the executor wrapper's inputs, and the wrapper's inner task passes them
// on to the template that becomes the pod. The compiler writes a task's own pod metadata as
// "{{inputs.parameters...}}" keys and values, so only the substituted template shows what the pod gets.
func resolveTaskPodTemplate(t *testing.T, workflow *util.Workflow, task string) (raw, resolved *v1alpha1.Template) {
	t.Helper()
	rootTask := requireDAGTask(t, requireTemplate(t, workflow, "root"), task)
	wrapper := requireTemplate(t, workflow, rootTask.Template)
	resolvedWrapper := resolveWithArgo(t, wrapper, &rootTask.Arguments)

	executorTask := requireDAGTask(t, resolvedWrapper, "executor")
	raw = requireTemplate(t, workflow, executorTask.Template)
	return raw, resolveWithArgo(t, raw, &executorTask.Arguments)
}

// assertTaskPodMetadata checks the pod metadata of the root tasks after Argo substitution. A task
// keeps what it authored, including a label that contradicts its annotation, and only a task that
// authored no injection value receives the installation default.
func assertTaskPodMetadata(t *testing.T, workflow *util.Workflow, defaultValue string) {
	t.Helper()
	for _, want := range taskPodMetadata {
		raw, resolved := resolveTaskPodTemplate(t, workflow, want.task)

		// The compiler's fallback sits on the literal key next to the task's parameterised keys.
		require.True(t, hasPlaceholder(raw.Metadata), "task %q: expected parameterised pod metadata", want.task)
		assert.Equal(t, defaultValue, raw.Metadata.Annotations[istioInjectKey], "task %q: compiled fallback", want.task)
		require.False(t, hasPlaceholder(resolved.Metadata), "task %q: substitution left a placeholder", want.task)

		assert.Equal(t,
			withInjectDefault(mergeMetadata(map[string]string{
				util.AnnotationKeyRuntimeRole: string(util.ExecutionRuntimeRoleLauncher),
				templateNameAnnotationKey:     resolved.Name,
			}, want.annotations), defaultValue),
			resolved.Metadata.Annotations, "task %q: annotations", want.task)
		assert.Equal(t,
			mergeMetadata(map[string]string{podRoleLabelKey: "container-executor"}, want.labels),
			resolved.Metadata.Labels, "task %q: labels", want.task)
	}
}

// assertSystemTemplateMetadata checks every template that is not authored by a task. Driver
// templates take the admin configured metadata and keep any injection value it carries over the
// fallback; no other template receives driver configuration, and all of them carry the default.
func assertSystemTemplateMetadata(t *testing.T, workflow *util.Workflow, defaultValue string, config driverPodConfig) {
	t.Helper()
	driverRoles := map[string]string{containerDriverTemplate: "container-driver", dagDriverTemplate: "dag-driver"}
	seenDrivers := map[string]bool{}

	for _, tmpl := range workflow.Spec.Templates {
		if role, isDriver := driverRoles[tmpl.Name]; isDriver {
			seenDrivers[tmpl.Name] = true
			assert.Equal(t,
				withInjectDefault(mergeMetadata(map[string]string{
					util.AnnotationKeyRuntimeRole: string(util.ExecutionRuntimeRoleDriver),
					templateNameAnnotationKey:     tmpl.Name,
				}, config.annotations), defaultValue),
				tmpl.Metadata.Annotations, "driver %q: annotations", tmpl.Name)
			assert.Equal(t,
				mergeMetadata(map[string]string{podRoleLabelKey: role}, config.labels),
				tmpl.Metadata.Labels, "driver %q: labels", tmpl.Name)
			continue
		}

		if tmpl.Name == importerTemplate || tmpl.Name == importerTemplate+"-workspace" {
			assert.Equal(t, map[string]string{
				util.AnnotationKeyRuntimeRole: string(util.ExecutionRuntimeRoleLauncher),
				templateNameAnnotationKey:     tmpl.Name,
				istioInjectKey:                defaultValue,
			}, tmpl.Metadata.Annotations, "importer annotations")
			assert.Equal(t, map[string]string{podRoleLabelKey: "importer"}, tmpl.Metadata.Labels, "importer labels")
			continue
		}

		assert.Equal(t, defaultValue, tmpl.Metadata.Annotations[istioInjectKey], "template %q: injection default", tmpl.Name)
		for key := range config.annotations {
			if key != istioInjectKey {
				assert.NotContains(t, tmpl.Metadata.Annotations, key, "template %q must not receive driver annotations", tmpl.Name)
			}
		}
		for key := range config.labels {
			assert.NotContains(t, tmpl.Metadata.Labels, key, "template %q must not receive driver labels", tmpl.Name)
		}
	}

	assert.Contains(t, seenDrivers, containerDriverTemplate)
	assert.Contains(t, seenDrivers, dagDriverTemplate)
}

// A task that authors its own injection value must keep it under either installation default, and
// a task that does not must receive the default. The test resolves the compiled templates with
// Argo itself because the task's value only reaches the pod through parameter substitution.
func TestIstioSidecarInjectDefaultYieldsToTaskPodMetadata(t *testing.T) {
	for _, mode := range istioFallbackModes {
		for _, compiler := range istioCompilers {
			t.Run(mode.name+"/"+compiler.name, func(t *testing.T) {
				useProcessEnvironment(t, mode.environment)
				useDriverPodConfig(t, driverPodConfig{})

				workflow := compiler.compile(t, podMetadataFixture)

				assertTaskPodMetadata(t, workflow, mode.want)
				assertSystemTemplateMetadata(t, workflow, mode.want, driverPodConfig{})
			})
		}
	}
}

// Driver pods are the only ones that take admin configured metadata. An injection value in that
// configuration wins over the installation default, a label that contradicts it is left alone for
// the injector to resolve, and nothing configured for drivers reaches executors or the importer.
func TestIstioSidecarInjectDefaultYieldsToDriverPodConfig(t *testing.T) {
	for _, fixture := range []struct {
		name         string
		fixture      istioFixture
		hasTasks     bool
		importerName string
	}{
		{name: "task pod metadata pipeline", fixture: podMetadataFixture, hasTasks: true},
		{name: "importer pipeline", fixture: importerFixture, importerName: importerTemplate},
		{name: "workspace importer pipeline", fixture: istioFixture{path: "../../../../test_data/sdk_compiled_pipelines/valid/critical/pipeline_with_importer_workspace.yaml"}, importerName: importerTemplate + "-workspace"},
	} {
		for _, mode := range istioFallbackModes {
			for _, scenario := range driverPodConfigScenarios {
				for _, compiler := range istioCompilers {
					t.Run(fixture.name+"/"+mode.name+"/"+scenario.name+"/"+compiler.name, func(t *testing.T) {
						useProcessEnvironment(t, mode.environment)
						useDriverPodConfig(t, scenario.config)

						workflow := compiler.compile(t, fixture.fixture)

						assertSystemTemplateMetadata(t, workflow, mode.want, scenario.config)
						if fixture.hasTasks {
							assertTaskPodMetadata(t, workflow, mode.want)
						}
						if fixture.importerName != "" {
							requireTemplate(t, workflow, fixture.importerName)
						}
					})
				}
			}
		}
	}
}

func TestIstioSidecarInjectPreservesRetryTaskMetadata(t *testing.T) {
	fixture := podMetadataFixture
	fixture.retry = true
	for _, mode := range istioFallbackModes {
		for _, compiler := range istioCompilers {
			t.Run(mode.name+"/"+compiler.name, func(t *testing.T) {
				useProcessEnvironment(t, mode.environment)
				useDriverPodConfig(t, driverPodConfig{})
				workflow := compiler.compile(t, fixture)
				for _, task := range taskPodMetadata {
					raw, _ := resolveTaskPodTemplate(t, workflow, task.task)
					require.NotNil(t, raw.RetryStrategy, "task %q must exercise retry compilation", task.task)
					require.Contains(t, raw.Name, "retry-")
				}
				assertTaskPodMetadata(t, workflow, mode.want)
				assertSystemTemplateMetadata(t, workflow, mode.want, driverPodConfig{})
			})
		}
	}
}
