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
	"path/filepath"
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
	"sigs.k8s.io/yaml"
)

// exitHookInParallelForIR is the IR for:
//
//	with dsl.ParallelFor(['A', 'B']) as item:
//	    p = produce(x=item)
//	    cleanup(msg=item).after(p).ignore_upstream_failure()
//
// .ignore_upstream_failure() sets ALL_UPSTREAM_TASKS_COMPLETED, which the Argo
// compiler turns into an exit lifecycle hook on the producer's task.
const exitHookInParallelForIR = `
components:
  comp-cleanup:
    executorLabel: exec-cleanup
    inputDefinitions:
      parameters:
        msg:
          parameterType: STRING
  comp-for-loop-1:
    dag:
      tasks:
        cleanup:
          componentRef:
            name: comp-cleanup
          dependentTasks:
          - produce
          inputs:
            parameters:
              msg:
                componentInputParameter: pipelinechannel--loop-item-param-1
          taskInfo:
            name: cleanup
          triggerPolicy:
            strategy: ALL_UPSTREAM_TASKS_COMPLETED
        produce:
          componentRef:
            name: comp-produce
          inputs:
            parameters:
              x:
                componentInputParameter: pipelinechannel--loop-item-param-1
          taskInfo:
            name: produce
    inputDefinitions:
      parameters:
        pipelinechannel--loop-item-param-1:
          parameterType: STRING
  comp-produce:
    executorLabel: exec-produce
    inputDefinitions:
      parameters:
        x:
          parameterType: STRING
deploymentSpec:
  executors:
    exec-cleanup:
      container:
        image: busybox
    exec-produce:
      container:
        image: busybox
pipelineInfo:
  name: exit-hook-in-parallelfor
root:
  dag:
    tasks:
      for-loop-1:
        componentRef:
          name: comp-for-loop-1
        parameterIterator:
          itemInput: pipelinechannel--loop-item-param-1
          items:
            raw: '["A", "B"]'
        taskInfo:
          name: for-loop-1
schemaVersion: 2.1.0
sdkVersion: kfp-2.17.0
`

// exitHookInNestedParallelForIR nests the loop above inside an outer
// ParallelFor, so the hook must pick up the inner loop's iteration index.
const exitHookInNestedParallelForIR = `
components:
  comp-cleanup:
    executorLabel: exec-cleanup
    inputDefinitions:
      parameters:
        msg:
          parameterType: STRING
  comp-for-loop-outer:
    dag:
      tasks:
        for-loop-1:
          componentRef:
            name: comp-for-loop-1
          parameterIterator:
            itemInput: pipelinechannel--loop-item-param-1
            items:
              raw: '["1", "2"]'
          taskInfo:
            name: for-loop-1
    inputDefinitions:
      parameters:
        pipelinechannel--loop-item-param-outer:
          parameterType: STRING
  comp-for-loop-1:
    dag:
      tasks:
        cleanup:
          componentRef:
            name: comp-cleanup
          dependentTasks:
          - produce
          inputs:
            parameters:
              msg:
                componentInputParameter: pipelinechannel--loop-item-param-1
          taskInfo:
            name: cleanup
          triggerPolicy:
            strategy: ALL_UPSTREAM_TASKS_COMPLETED
        produce:
          componentRef:
            name: comp-produce
          inputs:
            parameters:
              x:
                componentInputParameter: pipelinechannel--loop-item-param-1
          taskInfo:
            name: produce
    inputDefinitions:
      parameters:
        pipelinechannel--loop-item-param-1:
          parameterType: STRING
  comp-produce:
    executorLabel: exec-produce
    inputDefinitions:
      parameters:
        x:
          parameterType: STRING
deploymentSpec:
  executors:
    exec-cleanup:
      container:
        image: busybox
    exec-produce:
      container:
        image: busybox
pipelineInfo:
  name: exit-hook-nested-parallelfor
root:
  dag:
    tasks:
      for-loop-outer:
        componentRef:
          name: comp-for-loop-outer
        parameterIterator:
          itemInput: pipelinechannel--loop-item-param-outer
          items:
            raw: '["A", "B"]'
        taskInfo:
          name: for-loop-outer
schemaVersion: 2.1.0
sdkVersion: kfp-2.17.0
`

const paramIterationIndexName = "iteration-index"

func compileIRString(t *testing.T, ir string) map[string]wfapi.Template {
	t.Helper()
	jsonSpec, err := yaml.YAMLToJSON([]byte(ir))
	require.NoError(t, err)
	spec := &pipelinespec.PipelineSpec{}
	require.NoError(t, protojson.Unmarshal(jsonSpec, spec))
	specJSON, err := protojson.Marshal(spec)
	require.NoError(t, err)
	specStruct := &structpb.Struct{}
	require.NoError(t, protojson.Unmarshal(specJSON, specStruct))

	workflow, err := argocompiler.Compile(
		&pipelinespec.PipelineJob{PipelineSpec: specStruct},
		nil,
		&argocompiler.Options{},
	)
	require.NoError(t, err)
	return templatesByName(workflow)
}

func templatesByName(workflow *wfapi.Workflow) map[string]wfapi.Template {
	templates := make(map[string]wfapi.Template, len(workflow.Spec.Templates))
	for _, template := range workflow.Spec.Templates {
		templates[template.Name] = template
	}
	return templates
}

func findParameter(parameters []wfapi.Parameter, name string) *wfapi.Parameter {
	for i := range parameters {
		if parameters[i].Name == name {
			return &parameters[i]
		}
	}
	return nil
}

func exitHookOf(t *testing.T, template wfapi.Template, taskName string) wfapi.LifecycleHook {
	t.Helper()
	for _, task := range template.DAG.Tasks {
		if task.Name != taskName {
			continue
		}
		hook, found := task.Hooks[wfapi.ExitLifecycleEvent]
		require.True(t, found, "task %q in template %q has no exit hook", taskName, template.Name)
		return hook
	}
	t.Fatalf("task %q not found in template %q", taskName, template.Name)
	return wfapi.LifecycleHook{}
}

// An exit hook attached inside a loop body runs once per iteration, so its
// driver needs the enclosing iteration index to resolve iteration-scoped
// inputs. Without it the driver falls back to "-1" and resolves against the
// wrong iteration.
func TestCompileExitHookInParallelFor_PropagatesIterationIndex(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()
	templates := compileIRString(t, exitHookInParallelForIR)

	loopBody, found := templates["comp-for-loop-1"]
	require.True(t, found, "loop body template missing")

	hook := exitHookOf(t, loopBody, "produce")
	assert.Equal(t, "exit-hook-comp-for-loop-1-cleanup", hook.Template)
	hookArg := findParameter(hook.Arguments.Parameters, paramIterationIndexName)
	require.NotNil(t, hookArg, "exit hook must forward iteration-index")
	require.NotNil(t, hookArg.Value)
	assert.Equal(t, "{{inputs.parameters.iteration-index}}", hookArg.Value.String())

	hookTemplate, found := templates[hook.Template]
	require.True(t, found, "exit hook template missing")
	hookInput := findParameter(hookTemplate.Inputs.Parameters, paramIterationIndexName)
	require.NotNil(t, hookInput, "exit hook template must declare iteration-index")
	require.NotNil(t, hookInput.Default)
	assert.Equal(t, "-1", hookInput.Default.String())

	// The driver inside the hook is what resolves inputs, so it must receive
	// the index rather than fall back to the template default.
	var driverFound bool
	for _, task := range hookTemplate.DAG.Tasks {
		if task.Template != "system-container-driver" {
			continue
		}
		driverFound = true
		arg := findParameter(task.Arguments.Parameters, paramIterationIndexName)
		require.NotNil(t, arg, "hook driver task %q must receive iteration-index", task.Name)
		require.NotNil(t, arg.Value)
		assert.Equal(t, "{{inputs.parameters.iteration-index}}", arg.Value.String())
	}
	require.True(t, driverFound, "exit hook template has no container driver task")

	// Ordinary tasks in the same loop body keep their existing propagation.
	for _, task := range loopBody.DAG.Tasks {
		if task.Template != "system-container-driver" {
			continue
		}
		arg := findParameter(task.Arguments.Parameters, paramIterationIndexName)
		require.NotNil(t, arg, "loop task %q lost iteration-index", task.Name)
	}

	// Templates outside any loop must not gain the parameter.
	for _, name := range []string{"root", "comp-for-loop-1-for-loop-1-iterator"} {
		template, found := templates[name]
		require.True(t, found, "template %q missing", name)
		assert.Nil(t, findParameter(template.Inputs.Parameters, paramIterationIndexName),
			"template %q should not declare iteration-index", name)
	}

	assertNoDuplicateParameters(t, templates)
}

// In a nested loop the hook lives in the inner loop body, so it must receive
// the inner iteration index. The outer loop's propagation must not overwrite
// the inner loop's {{item}} binding.
func TestCompileExitHookInNestedParallelFor_UsesInnerIterationIndex(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()
	templates := compileIRString(t, exitHookInNestedParallelForIR)

	innerBody, found := templates["comp-for-loop-1"]
	require.True(t, found, "inner loop body template missing")
	hook := exitHookOf(t, innerBody, "produce")
	hookArg := findParameter(hook.Arguments.Parameters, paramIterationIndexName)
	require.NotNil(t, hookArg, "nested exit hook must forward iteration-index")
	require.NotNil(t, hookArg.Value)
	assert.Equal(t, "{{inputs.parameters.iteration-index}}", hookArg.Value.String())

	// comp-for-loop-1 is instantiated by the inner iterator, which binds
	// iteration-index to {{item}}. That binding is the inner index, so the hook
	// expression above resolves to the inner iteration and not the outer one.
	innerIterator, found := templates["comp-for-loop-1-for-loop-1-iterator"]
	require.True(t, found, "inner iterator template missing")
	var iterationsTaskChecked bool
	for _, task := range innerIterator.DAG.Tasks {
		if task.Template != "comp-for-loop-1" {
			continue
		}
		iterationsTaskChecked = true
		arg := findParameter(task.Arguments.Parameters, paramIterationIndexName)
		require.NotNil(t, arg)
		require.NotNil(t, arg.Value)
		assert.Equal(t, "{{item}}", arg.Value.String(),
			"inner loop body must keep its own {{item}} binding")
	}
	require.True(t, iterationsTaskChecked, "inner iterations task not found")

	// The inner loop's coordinator driver still receives the outer index, which
	// is what makes the inner parent-dag-task-id distinct per outer iteration.
	for _, task := range innerIterator.DAG.Tasks {
		if task.Template != "system-dag-driver" {
			continue
		}
		arg := findParameter(task.Arguments.Parameters, paramIterationIndexName)
		require.NotNil(t, arg, "inner coordinator driver must receive the outer index")
		require.NotNil(t, arg.Value)
		assert.Equal(t, "{{inputs.parameters.iteration-index}}", arg.Value.String())
	}

	assertNoDuplicateParameters(t, templates)
}

// A root-level dsl.ExitHandler is not inside a loop, so its enclosing template
// declares no iteration-index. Forwarding the parameter there would produce an
// unresolvable placeholder at submission time.
func TestCompileRootExitHandler_OmitsIterationIndex(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()
	pipelineSpec, platformSpec, err := util.LoadPipelineAndPlatformSpec(filepath.Join(
		"../../../../../test_data/sdk_compiled_pipelines/valid/failing/pipeline_with_exit_handler.yaml"))
	require.NoError(t, err)
	pipelineJSON, err := protojson.Marshal(pipelineSpec)
	require.NoError(t, err)
	pipelineStruct := &structpb.Struct{}
	require.NoError(t, protojson.Unmarshal(pipelineJSON, pipelineStruct))
	workflow, err := argocompiler.Compile(
		&pipelinespec.PipelineJob{PipelineSpec: pipelineStruct},
		platformSpec.GetPlatforms()["kubernetes"],
		&argocompiler.Options{},
	)
	require.NoError(t, err)
	templates := templatesByName(workflow)

	var hookTemplates []string
	for name, template := range templates {
		if template.DAG == nil {
			continue
		}
		for _, task := range template.DAG.Tasks {
			hook, found := task.Hooks[wfapi.ExitLifecycleEvent]
			if !found {
				continue
			}
			hookTemplates = append(hookTemplates, hook.Template)
			assert.Nil(t, findParameter(hook.Arguments.Parameters, paramIterationIndexName),
				"root-level exit hook on %q/%q must not forward iteration-index", name, task.Name)
			assert.Nil(t, findParameter(template.Inputs.Parameters, paramIterationIndexName),
				"template %q enclosing a root-level hook must not declare iteration-index", name)
		}
	}
	require.NotEmpty(t, hookTemplates, "fixture should compile at least one exit hook")

	for _, name := range hookTemplates {
		template, found := templates[name]
		require.True(t, found, "exit hook template %q missing", name)
		assert.Nil(t, findParameter(template.Inputs.Parameters, paramIterationIndexName),
			"root-level exit hook template %q must not declare iteration-index", name)
	}

	assertNoDuplicateParameters(t, templates)
}

// Templates are shared and traversed more than once, so parameter insertion
// must stay idempotent.
func assertNoDuplicateParameters(t *testing.T, templates map[string]wfapi.Template) {
	t.Helper()
	for name, template := range templates {
		seen := make(map[string]bool, len(template.Inputs.Parameters))
		for _, parameter := range template.Inputs.Parameters {
			require.False(t, seen[parameter.Name],
				"template %q declares input parameter %q twice", name, parameter.Name)
			seen[parameter.Name] = true
		}
		if template.DAG == nil {
			continue
		}
		for _, task := range template.DAG.Tasks {
			seenArgs := make(map[string]bool, len(task.Arguments.Parameters))
			for _, parameter := range task.Arguments.Parameters {
				require.False(t, seenArgs[parameter.Name],
					"task %q/%q passes argument %q twice", name, task.Name, parameter.Name)
				seenArgs[parameter.Name] = true
			}
			for event, hook := range task.Hooks {
				seenHookArgs := make(map[string]bool, len(hook.Arguments.Parameters))
				for _, parameter := range hook.Arguments.Parameters {
					require.False(t, seenHookArgs[parameter.Name],
						"hook %q on %q/%q passes argument %q twice", event, name, task.Name, parameter.Name)
					seenHookArgs[parameter.Name] = true
				}
			}
		}
	}
}
