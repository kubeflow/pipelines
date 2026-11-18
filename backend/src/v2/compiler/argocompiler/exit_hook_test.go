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

package argocompiler_test

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	wfapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/compiler/argocompiler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

// exitTask describes an exit task, i.e. one the SDK marks with
// ALL_UPSTREAM_TASKS_COMPLETED. dsl.ExitHandler exit tasks and
// .ignore_upstream_failure() tasks both compile to that trigger strategy.
type exitTask struct {
	name string
	deps []string
}

// compileExitHookPipeline compiles a pipeline with the given plain producers and
// exit tasks, mirroring the IR the SDK emits for them.
func compileExitHookPipeline(t *testing.T, producers []string, exitTasks []exitTask, enableCache bool) *wfapi.Workflow {
	t.Helper()
	proxy.InitializeConfigWithEmptyForTests()

	tasks := map[string]*pipelinespec.PipelineTaskSpec{}
	components := map[string]*pipelinespec.ComponentSpec{}
	executors := map[string]any{}

	addComponent := func(name string) {
		components["comp-"+name] = &pipelinespec.ComponentSpec{
			Implementation: &pipelinespec.ComponentSpec_ExecutorLabel{ExecutorLabel: "exec-" + name},
		}
		executors["exec-"+name] = map[string]any{
			"container": map[string]any{
				"image":   "python:3.11",
				"command": []any{"echo", name},
			},
		}
	}

	for _, p := range producers {
		addComponent(p)
		tasks[p] = &pipelinespec.PipelineTaskSpec{
			TaskInfo:       &pipelinespec.PipelineTaskInfo{Name: p},
			ComponentRef:   &pipelinespec.ComponentRef{Name: "comp-" + p},
			CachingOptions: &pipelinespec.PipelineTaskSpec_CachingOptions{EnableCache: enableCache},
		}
	}
	for _, e := range exitTasks {
		addComponent(e.name)
		tasks[e.name] = &pipelinespec.PipelineTaskSpec{
			TaskInfo:       &pipelinespec.PipelineTaskInfo{Name: e.name},
			ComponentRef:   &pipelinespec.ComponentRef{Name: "comp-" + e.name},
			CachingOptions: &pipelinespec.PipelineTaskSpec_CachingOptions{EnableCache: enableCache},
			DependentTasks: e.deps,
			TriggerPolicy: &pipelinespec.PipelineTaskSpec_TriggerPolicy{
				Strategy: pipelinespec.PipelineTaskSpec_TriggerPolicy_ALL_UPSTREAM_TASKS_COMPLETED,
			},
		}
	}

	deployment, err := structpb.NewStruct(map[string]any{"executors": executors})
	require.NoError(t, err)

	spec := &pipelinespec.PipelineSpec{
		PipelineInfo:   &pipelinespec.PipelineInfo{Name: "exit-hook-test"},
		SchemaVersion:  "2.1.0",
		SdkVersion:     "kfp-2.17.0",
		DeploymentSpec: deployment,
		Components:     components,
		Root: &pipelinespec.ComponentSpec{
			Implementation: &pipelinespec.ComponentSpec_Dag{
				Dag: &pipelinespec.DagSpec{Tasks: tasks},
			},
		},
	}

	raw, err := protojson.Marshal(spec)
	require.NoError(t, err)
	specStruct := &structpb.Struct{}
	require.NoError(t, protojson.Unmarshal(raw, specStruct))

	wf, err := argocompiler.Compile(
		&pipelinespec.PipelineJob{PipelineSpec: specStruct}, nil, &argocompiler.Options{},
	)
	require.NoError(t, err)
	return wf
}

// hookAttachments maps each hooked template to the "<template>/<task>" sites
// that attach it as an exit hook.
func hookAttachments(wf *wfapi.Workflow) map[string][]string {
	attachments := map[string][]string{}
	for _, tmpl := range wf.Spec.Templates {
		if tmpl.DAG == nil {
			continue
		}
		for _, task := range tmpl.DAG.Tasks {
			for event, hook := range task.Hooks {
				attachments[hook.Template] = append(attachments[hook.Template],
					fmt.Sprintf("%s/%s[%s]", tmpl.Name, task.Name, event))
			}
		}
	}
	return attachments
}

// exitHookInvocations maps each "exit-hook-*" template to every site that
// invokes it, whether attached as a lifecycle hook or run as a task of an
// aggregating wrapper. A template invoked from two sites runs twice.
func exitHookInvocations(wf *wfapi.Workflow) map[string][]string {
	invocations := map[string][]string{}
	record := func(template, site string) {
		if strings.HasPrefix(template, "exit-hook-") {
			invocations[template] = append(invocations[template], site)
		}
	}
	for _, tmpl := range wf.Spec.Templates {
		if tmpl.DAG == nil {
			continue
		}
		for _, task := range tmpl.DAG.Tasks {
			record(task.Template, fmt.Sprintf("%s/%s", tmpl.Name, task.Name))
			for event, hook := range task.Hooks {
				record(hook.Template, fmt.Sprintf("%s/%s[%s]", tmpl.Name, task.Name, event))
			}
		}
	}
	return invocations
}

// unreachableTemplates returns templates that nothing in the workflow can reach,
// so a generated exit hook that is never invoked fails loudly.
func unreachableTemplates(wf *wfapi.Workflow) []string {
	reachable := map[string]bool{wf.Spec.Entrypoint: true}
	if wf.Spec.OnExit != "" {
		reachable[wf.Spec.OnExit] = true
	}
	for _, tmpl := range wf.Spec.Templates {
		if tmpl.DAG == nil {
			continue
		}
		for _, task := range tmpl.DAG.Tasks {
			if task.Template != "" {
				reachable[task.Template] = true
			}
			for _, hook := range task.Hooks {
				reachable[hook.Template] = true
			}
		}
	}
	var unreachable []string
	for _, tmpl := range wf.Spec.Templates {
		if !reachable[tmpl.Name] {
			unreachable = append(unreachable, tmpl.Name)
		}
	}
	return unreachable
}

func dagTask(t *testing.T, wf *wfapi.Workflow, tmplName, taskName string) wfapi.DAGTask {
	t.Helper()
	for _, tmpl := range wf.Spec.Templates {
		if tmpl.Name != tmplName || tmpl.DAG == nil {
			continue
		}
		for _, task := range tmpl.DAG.Tasks {
			if task.Name == taskName {
				return task
			}
		}
	}
	t.Fatalf("task %q not found in template %q", taskName, tmplName)
	return wfapi.DAGTask{}
}

// TestExitHookInvokedExactlyOnce locks in the two invariants the exit-hook
// compiler must hold for every pipeline shape:
//
//  1. Every generated exit-hook template is reachable. Keying the upstream ->
//     template mapping by upstream task name used to let a second exit task
//     naming the same upstream replace the first, leaving a template that
//     nothing invoked, so that exit task silently never ran.
//  2. Every exit-hook template is attached exactly once. Argo evaluates an exit
//     hook per task it is attached to, so attaching one template to several
//     upstream tasks ran the exit task once per upstream.
func TestExitHookInvokedExactlyOnce(t *testing.T) {
	tests := []struct {
		name      string
		producers []string
		exitTasks []exitTask
	}{
		{
			name:      "single upstream",
			producers: []string{"produce"},
			exitTasks: []exitTask{{name: "cleanup", deps: []string{"produce"}}},
		},
		{
			name:      "two upstream tasks",
			producers: []string{"produce", "produce-2"},
			exitTasks: []exitTask{{name: "cleanup", deps: []string{"produce", "produce-2"}}},
		},
		{
			name:      "three upstream tasks",
			producers: []string{"produce", "produce-2", "produce-3"},
			exitTasks: []exitTask{{name: "cleanup", deps: []string{"produce", "produce-2", "produce-3"}}},
		},
		{
			name:      "two exit tasks sharing one upstream",
			producers: []string{"produce"},
			exitTasks: []exitTask{
				{name: "cleanup", deps: []string{"produce"}},
				{name: "cleanup-2", deps: []string{"produce"}},
			},
		},
		{
			name:      "three exit tasks sharing one upstream",
			producers: []string{"produce"},
			exitTasks: []exitTask{
				{name: "cleanup", deps: []string{"produce"}},
				{name: "cleanup-2", deps: []string{"produce"}},
				{name: "cleanup-3", deps: []string{"produce"}},
			},
		},
		{
			name:      "exit tasks on different upstream tasks",
			producers: []string{"produce", "produce-2"},
			exitTasks: []exitTask{
				{name: "cleanup", deps: []string{"produce"}},
				{name: "cleanup-2", deps: []string{"produce-2"}},
			},
		},
		{
			name:      "shared upstream mixed with multi upstream",
			producers: []string{"produce", "produce-2"},
			exitTasks: []exitTask{
				{name: "cleanup", deps: []string{"produce"}},
				{name: "cleanup-2", deps: []string{"produce"}},
				{name: "cleanup-3", deps: []string{"produce", "produce-2"}},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			wf := compileExitHookPipeline(t, test.producers, test.exitTasks, true)

			assert.Empty(t, unreachableTemplates(wf),
				"every generated template must be reachable; an unreachable exit hook never runs")

			invocations := exitHookInvocations(wf)
			for _, e := range test.exitTasks {
				tmpl := fmt.Sprintf("exit-hook-root-%s", e.name)
				require.Contains(t, invocations, tmpl,
					"exit task %q produced no reachable invocation site", e.name)
			}
			for tmpl, sites := range invocations {
				assert.Len(t, sites, 1,
					"exit-hook template %q is invoked from %d sites, so the exit task runs once per site: %v",
					tmpl, len(sites), sites)
			}
		})
	}
}

// TestMultiUpstreamExitTaskIsGated checks the structure that makes a
// multi-upstream exit task run once: the hook moves onto a gate task that waits
// for every upstream task to reach a final state, and a companion status task
// keeps an upstream failure visible in the DAG phase.
func TestMultiUpstreamExitTaskIsGated(t *testing.T) {
	wf := compileExitHookPipeline(t,
		[]string{"produce", "produce-2"},
		[]exitTask{{name: "cleanup", deps: []string{"produce", "produce-2"}}},
		true,
	)

	// The upstream tasks themselves must not carry the hook any more.
	for _, producer := range []string{"produce", "produce-2"} {
		assert.Empty(t, dagTask(t, wf, "root", producer).Hooks,
			"upstream task %q must not carry the exit hook, or the exit task runs once per upstream", producer)
	}

	gate := dagTask(t, wf, "root", "cleanup-upstream-gate")
	require.NotNil(t, gate.Hooks.GetExitHook(), "the gate must carry the exit hook")
	assert.Equal(t, "exit-hook-root-cleanup", gate.Hooks.GetExitHook().Template)
	// The gate waits for both upstream tasks, in any final state, so that
	// .ignore_upstream_failure() still runs the exit task when one fails.
	for _, producer := range []string{"produce", "produce-2"} {
		for _, state := range []string{"Succeeded", "Failed", "Errored", "Skipped", "Omitted"} {
			assert.Contains(t, gate.Depends, producer+"."+state)
		}
	}
	assert.Equal(t, 1, strings.Count(gate.Depends, "&&"),
		"the gate must require both upstream tasks, not either one: %q", gate.Depends)

	// The gate succeeds even when an upstream task fails, and depending on the
	// upstream tasks stops them being DAG leaves. Without the status task the
	// DAG would then report success despite the failure.
	status := dagTask(t, wf, "root", "cleanup-upstream-status")
	assert.Equal(t, "produce.Succeeded && produce-2.Succeeded", status.Depends)
	assert.Empty(t, status.Hooks, "the status task must not duplicate the hook")

	// Neither gate task should schedule a pod.
	for _, tmpl := range wf.Spec.Templates {
		if tmpl.Name == gate.Template {
			require.NotNil(t, tmpl.Suspend, "gate template must be a suspend template, not a pod")
			assert.Equal(t, "0s", tmpl.Suspend.Duration)
		}
	}
	assert.Equal(t, gate.Template, status.Template, "both gate tasks share one no-op template")
}

// TestSingleUpstreamExitHookUnchanged keeps the single-upstream shape, which is
// what dsl.ExitHandler and the common .ignore_upstream_failure() case compile
// to, attached directly to its upstream task with no added tasks.
func TestSingleUpstreamExitHookUnchanged(t *testing.T) {
	wf := compileExitHookPipeline(t,
		[]string{"produce"},
		[]exitTask{{name: "cleanup", deps: []string{"produce"}}},
		true,
	)

	produce := dagTask(t, wf, "root", "produce")
	require.NotNil(t, produce.Hooks.GetExitHook())
	assert.Equal(t, "exit-hook-root-cleanup", produce.Hooks.GetExitHook().Template,
		"a single upstream task keeps the hook directly, with no wrapper")

	for _, tmpl := range wf.Spec.Templates {
		assert.NotContains(t, tmpl.Name, "-group", "no aggregation wrapper is needed for one exit task")
		assert.Nil(t, tmpl.Suspend, "no gate is needed for a single upstream task")
	}
	for _, task := range dagTemplate(t, wf, "root").DAG.Tasks {
		assert.NotContains(t, task.Name, "-upstream-gate")
		assert.NotContains(t, task.Name, "-upstream-status")
	}
}

// TestSharedUpstreamExitTemplatesAreAggregated covers two exit tasks naming the
// same upstream task. Argo's lifecycle hooks are keyed by event, so one task can
// hold only one "exit" hook; both exit tasks therefore run through a single
// aggregating template rather than one replacing the other.
func TestSharedUpstreamExitTemplatesAreAggregated(t *testing.T) {
	wf := compileExitHookPipeline(t,
		[]string{"produce"},
		[]exitTask{
			{name: "cleanup", deps: []string{"produce"}},
			{name: "cleanup-2", deps: []string{"produce"}},
		},
		true,
	)

	produce := dagTask(t, wf, "root", "produce")
	require.NotNil(t, produce.Hooks.GetExitHook())
	wrapper := produce.Hooks.GetExitHook().Template
	assert.Equal(t, "exit-hook-root-produce-group", wrapper)

	invoked := map[string]bool{}
	for _, task := range dagTemplate(t, wf, wrapper).DAG.Tasks {
		invoked[task.Template] = true
	}
	assert.True(t, invoked["exit-hook-root-cleanup"], "first exit task must still run")
	assert.True(t, invoked["exit-hook-root-cleanup-2"], "second exit task must still run")
	assert.Empty(t, unreachableTemplates(wf))
}

// TestExitHookGatingIndependentOfCaching checks that running a multi-upstream
// exit task once comes from the generated topology rather than from a cache hit
// suppressing a second execution.
func TestExitHookGatingIndependentOfCaching(t *testing.T) {
	for _, enableCache := range []bool{true, false} {
		t.Run(fmt.Sprintf("enableCache=%v", enableCache), func(t *testing.T) {
			wf := compileExitHookPipeline(t,
				[]string{"produce", "produce-2"},
				[]exitTask{{name: "cleanup", deps: []string{"produce", "produce-2"}}},
				enableCache,
			)
			require.Len(t, exitHookInvocations(wf)["exit-hook-root-cleanup"], 1)
			require.Len(t, hookAttachments(wf)["exit-hook-root-cleanup"], 1)
			assert.Empty(t, dagTask(t, wf, "root", "produce").Hooks)
			assert.Empty(t, dagTask(t, wf, "root", "produce-2").Hooks)
		})
	}
}

// TestCompiledFixturesHaveNoUnreachableExitHooks applies the same invariants to
// the SDK-compiled fixtures, including nested DAGs and dsl.ExitHandler.
func TestCompiledFixturesHaveNoUnreachableExitHooks(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()
	fixtures := []string{
		"failing/pipeline_with_exit_handler.yaml",
		"failing/pipeline_with_multiple_exit_handlers.yaml",
		"pipeline_as_exit_task.yaml",
		"pipeline_with_task_using_ignore_upstream_failure.yaml",
	}
	for _, fixture := range fixtures {
		t.Run(fixture, func(t *testing.T) {
			spec, platform, err := util.LoadPipelineAndPlatformSpec(filepath.Join(
				"../../../../../test_data/sdk_compiled_pipelines/valid", fixture))
			require.NoError(t, err)
			raw, err := protojson.Marshal(spec)
			require.NoError(t, err)
			specStruct := &structpb.Struct{}
			require.NoError(t, protojson.Unmarshal(raw, specStruct))
			wf, err := argocompiler.Compile(&pipelinespec.PipelineJob{PipelineSpec: specStruct},
				platform.GetPlatforms()["kubernetes"], &argocompiler.Options{})
			require.NoError(t, err)

			assert.Empty(t, unreachableTemplates(wf))
			for tmpl, sites := range exitHookInvocations(wf) {
				assert.Len(t, sites, 1, "exit-hook template %q invoked from %d sites: %v",
					tmpl, len(sites), sites)
			}
		})
	}
}

func dagTemplate(t *testing.T, wf *wfapi.Workflow, name string) wfapi.Template {
	t.Helper()
	for _, tmpl := range wf.Spec.Templates {
		if tmpl.Name == name {
			require.NotNil(t, tmpl.DAG, "template %q is not a DAG", name)
			return tmpl
		}
	}
	t.Fatalf("template %q not found", name)
	return wfapi.Template{}
}
