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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// istioSidecarInjectCases covers the environment the installation can hand to the API server.
// Only the exact string "true" opts in; everything else, including a variable that is not set at
// all, must keep upstream behaviour so a typo can never silently put workflow pods in the mesh.
var istioSidecarInjectCases = []struct {
	name        string
	environment *string
	want        string
}{
	{"opt-in true", util.StringPointer("true"), util.AnnotationValueIstioSidecarInjectEnabled},
	{"unset", nil, util.AnnotationValueIstioSidecarInjectDisabled},
	{"empty", util.StringPointer(""), util.AnnotationValueIstioSidecarInjectDisabled},
	{"false", util.StringPointer("false"), util.AnnotationValueIstioSidecarInjectDisabled},
	{"upper case TRUE", util.StringPointer("TRUE"), util.AnnotationValueIstioSidecarInjectDisabled},
	{"padded true", util.StringPointer(" true "), util.AnnotationValueIstioSidecarInjectDisabled},
	{"invalid", util.StringPointer("mesh"), util.AnnotationValueIstioSidecarInjectDisabled},
}

// useProcessEnvironment makes viper resolve settings from the process environment exactly as
// initConfig does in the API server, then applies the case's environment to the process.
func useProcessEnvironment(t *testing.T, environment *string) {
	t.Helper()
	viper.Reset()
	t.Cleanup(viper.Reset)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	viper.AllowEmptyEnv(true)

	// t.Setenv registers the cleanup that restores the original value, including "was unset".
	t.Setenv(WorkflowIstioSidecarInject, "")
	if environment == nil {
		require.NoError(t, os.Unsetenv(WorkflowIstioSidecarInject))
		return
	}
	t.Setenv(WorkflowIstioSidecarInject, *environment)
}

func TestIstioSidecarInjectDefault(t *testing.T) {
	for _, test := range istioSidecarInjectCases {
		t.Run(test.name, func(t *testing.T) {
			useProcessEnvironment(t, test.environment)
			assert.Equal(t, test.want, istioSidecarInjectDefault())
		})
	}
}

// Load a synthetic config.json through the same file discovery as main.initConfig.
// Environment values, including an explicitly empty value, take precedence.
func TestIstioSidecarInjectConfigJSON(t *testing.T) {
	cases := []struct {
		name        string
		value       interface{}
		environment *string
		want        string
	}{
		{"missing", nil, nil, "false"},
		{"empty", "", nil, "false"},
		{"true", "true", nil, "true"},
		{"false", "false", nil, "false"},
		{"wrong case", "TRUE", nil, "false"},
		{"invalid", "mesh", nil, "false"},
		{"JSON boolean true", true, nil, "true"},
		{"environment opts out", "true", util.StringPointer("false"), "false"},
		{"environment opts in", "false", util.StringPointer("true"), "true"},
		{"empty environment overrides file", "true", util.StringPointer(""), "false"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			useProcessEnvironment(t, test.environment)
			directory := t.TempDir()
			config := map[string]interface{}{}
			if test.value != nil {
				config[WorkflowIstioSidecarInject] = test.value
			}
			content, err := json.Marshal(config)
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(filepath.Join(directory, "config.json"), content, 0600))
			viper.SetConfigName("config")
			viper.AddConfigPath(directory)
			require.NoError(t, viper.ReadInConfig())
			assert.Equal(t, test.want, istioSidecarInjectDefault())
			for _, compiler := range istioCompilers {
				for name, value := range istioAnnotationByTemplate(t, compiler.compile(t, helloWorldFixture)) {
					assert.Equal(t, test.want, value, "%s template %q", compiler.name, name)
				}
			}
		})
	}
	t.Run("explicit reload", func(t *testing.T) {
		useProcessEnvironment(t, nil)
		directory := t.TempDir()
		file := filepath.Join(directory, "config.json")
		viper.SetConfigName("config")
		viper.AddConfigPath(directory)
		for _, value := range []string{"true", "false"} {
			content, err := json.Marshal(map[string]string{WorkflowIstioSidecarInject: value})
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(file, content, 0600))
			// main's watcher callback reloads via ReadInConfig; no watcher is started here.
			require.NoError(t, viper.ReadInConfig())
			assert.Equal(t, value, istioSidecarInjectDefault())
		}
	})
}

// istioFixture is a pipeline manifest together with the runtime parameters it needs to compile.
type istioFixture struct {
	path       string
	parameters string
	retry      bool
}

var helloWorldFixture = istioFixture{path: "testdata/hello_world.yaml", parameters: `{"y":"world"}`}

// istioCompilers are the two places a template is compiled and given its injection default.
var istioCompilers = []struct {
	name    string
	compile func(*testing.T, istioFixture) *util.Workflow
}{
	{"run", compileRun},
	{"recurring run", compileRecurringRun},
}

func compileRun(t *testing.T, fixture istioFixture) *util.Workflow {
	t.Helper()
	proxy.InitializeConfigWithEmptyForTests()
	manifest := loadYaml(t, fixture.path)
	pipeline, err := New([]byte(manifest), TemplateOptions{CacheDisabled: true, DefaultWorkspace: defaultPVC})
	require.NoError(t, err)
	if fixture.retry {
		enableIstioFixtureRetries(t, pipeline)
	}

	executionSpec, err := pipeline.RunWorkflow(&model.Run{
		DisplayName: "run1",
		Namespace:   "ns1",
		PipelineSpec: model.PipelineSpec{
			PipelineSpecManifest: model.LargeText(manifest),
			RuntimeConfig:        model.RuntimeConfig{Parameters: model.LargeText(fixture.parameters)},
		},
	}, RunWorkflowOptions{RunID: "run-id", RunAt: 1})
	require.NoError(t, err)
	workflow, ok := executionSpec.(*util.Workflow)
	require.True(t, ok, "expected an Argo workflow, got %T", executionSpec)
	return workflow
}

func compileRecurringRun(t *testing.T, fixture istioFixture) *util.Workflow {
	t.Helper()
	proxy.InitializeConfigWithEmptyForTests()
	manifest := loadYaml(t, fixture.path)
	pipeline, err := New([]byte(manifest), TemplateOptions{CacheDisabled: true, DefaultWorkspace: defaultPVC})
	require.NoError(t, err)
	if fixture.retry {
		enableIstioFixtureRetries(t, pipeline)
	}

	scheduledWorkflow, err := pipeline.ScheduledWorkflow(&model.Job{
		K8SName:        "name1",
		Enabled:        true,
		MaxConcurrency: 1,
		Trigger: model.Trigger{
			CronSchedule: model.CronSchedule{
				CronScheduleStartTimeInSec: util.Int64Pointer(1),
				CronScheduleEndTimeInSec:   util.Int64Pointer(10),
				Cron:                       util.StringPointer("1 * * * *"),
			},
		},
		PipelineSpec: model.PipelineSpec{
			PipelineSpecManifest: model.LargeText(manifest),
			RuntimeConfig:        model.RuntimeConfig{Parameters: model.LargeText(fixture.parameters)},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, scheduledWorkflow.Spec.Workflow)
	spec, ok := scheduledWorkflow.Spec.Workflow.Spec.(string)
	require.True(t, ok, "expected the workflow spec as a JSON string, got %T", scheduledWorkflow.Spec.Workflow.Spec)
	workflow, err := util.NewWorkflowFromScheduleWorkflowSpecBytesJSON([]byte(spec))
	require.NoError(t, err)
	return workflow
}

func enableIstioFixtureRetries(t *testing.T, pipeline Template) {
	t.Helper()
	spec, ok := pipeline.(*V2Spec)
	require.True(t, ok)
	for _, task := range spec.spec.GetRoot().GetDag().GetTasks() {
		task.RetryPolicy = &pipelinespec.PipelineTaskSpec_RetryPolicy{MaxRetryCount: 1, BackoffFactor: 2}
	}
}

// istioAnnotationByTemplate returns the injection annotation of every generated template, failing
// on the first template that has none so a skipped template cannot hide behind the others.
func istioAnnotationByTemplate(t *testing.T, workflow *util.Workflow) map[string]string {
	t.Helper()
	templates := workflow.Spec.Templates
	// hello-world compiles to an entrypoint, a root DAG, drivers and an executor; one template
	// would mean the loop below checks nothing of interest.
	require.Greater(t, len(templates), 1, "compiled workflow should contain several templates")

	annotations := make(map[string]string, len(templates))
	for _, tmpl := range templates {
		value, ok := tmpl.Metadata.Annotations[util.AnnotationKeyIstioSidecarInject]
		require.True(t, ok, "template %q has no %s annotation", tmpl.Name, util.AnnotationKeyIstioSidecarInject)
		annotations[tmpl.Name] = value
	}
	require.Len(t, annotations, len(templates), "template names must be unique")
	return annotations
}

// Compiling both paths fails if either stops reading the configured default.
func TestIstioSidecarInjectDefaultIsAppliedToEveryCompiledTemplate(t *testing.T) {
	for _, test := range istioSidecarInjectCases {
		t.Run(test.name, func(t *testing.T) {
			useProcessEnvironment(t, test.environment)

			run := istioAnnotationByTemplate(t, compileRun(t, helloWorldFixture))
			recurringRun := istioAnnotationByTemplate(t, compileRecurringRun(t, helloWorldFixture))

			for name, value := range run {
				assert.Equal(t, test.want, value, "run template %q", name)
			}
			for name, value := range recurringRun {
				assert.Equal(t, test.want, value, "recurring run template %q", name)
			}
			assert.Equal(t, run, recurringRun, "run and recurring run must annotate the same templates alike")
		})
	}
}
