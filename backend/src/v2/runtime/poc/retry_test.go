package poc

import (
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestManagedRetryPreservesIndependentSuccessfulTasks(t *testing.T) {
	tasks := []*model.Task{
		{UUID: "root", ScopePath: "root", State: model.TaskStatus(apiv2.PipelineTask_FAILED)},
		{UUID: "failed", ParentTaskUUID: util.StringPointer("root"), ScopePath: "root.failed", State: model.TaskStatus(apiv2.PipelineTask_FAILED)},
		{UUID: "success", ParentTaskUUID: util.StringPointer("root"), ScopePath: "root.success", State: model.TaskStatus(apiv2.PipelineTask_SUCCEEDED)},
		{UUID: "cached", ParentTaskUUID: util.StringPointer("root"), ScopePath: "root.cached", State: model.TaskStatus(apiv2.PipelineTask_CACHED)},
		{UUID: "exit", ParentTaskUUID: util.StringPointer("root"), ScopePath: "root.exit", State: model.TaskStatus(apiv2.PipelineTask_SUCCEEDED)},
		{UUID: "skipped", ParentTaskUUID: util.StringPointer("root"), ScopePath: "root.skipped", State: model.TaskStatus(apiv2.PipelineTask_SKIPPED)},
		{UUID: "downstream-dag", ParentTaskUUID: util.StringPointer("root"), ScopePath: "root.downstream-dag", State: model.TaskStatus(apiv2.PipelineTask_SUCCEEDED)},
		{UUID: "downstream-child", ParentTaskUUID: util.StringPointer("downstream-dag"), ScopePath: "root.downstream-dag.child", State: model.TaskStatus(apiv2.PipelineTask_SUCCEEDED)},
	}
	dependencies := map[string][]string{
		"exit": {"failed"}, "skipped": {"failed"}, "downstream-dag": {"failed"},
	}
	require.ElementsMatch(t, []string{"root", "failed", "exit", "skipped", "downstream-dag", "downstream-child"}, tasksForManagedRetry(tasks, dependencies, false))
	require.Len(t, tasksForManagedRetry(tasks, dependencies, true), len(tasks), "workspace data must be reconstructed rather than assuming deleted files survived")
}

func TestManagedRetryReconstructsDependenciesForEachMaterializedIteration(t *testing.T) {
	task := func(id, parent, name, scope string, state apiv2.PipelineTask_TaskState) *model.Task {
		result := &model.Task{UUID: id, Name: name, ScopePath: scope, State: model.TaskStatus(state)}
		if parent != "" {
			result.ParentTaskUUID = &parent
		}
		return result
	}
	tasks := []*model.Task{
		task("root", "", "ROOT", "root", apiv2.PipelineTask_FAILED),
		task("loop", "root", "loop", "root.loop", apiv2.PipelineTask_FAILED),
		task("iteration-0", "loop", "loop", "root.loop", apiv2.PipelineTask_FAILED),
		task("iteration-1", "loop", "loop", "root.loop", apiv2.PipelineTask_SUCCEEDED),
		task("a0", "iteration-0", "a", "root.loop.a", apiv2.PipelineTask_FAILED),
		task("b0", "iteration-0", "b", "root.loop.b", apiv2.PipelineTask_SUCCEEDED),
		task("a1", "iteration-1", "a", "root.loop.a", apiv2.PipelineTask_SUCCEEDED),
		task("b1", "iteration-1", "b", "root.loop.b", apiv2.PipelineTask_SUCCEEDED),
	}
	spec := &pipelinespec.PipelineSpec{}
	require.NoError(t, protojson.Unmarshal([]byte(`{
		"root":{"dag":{"tasks":{"loop":{"componentRef":{"name":"body"},"parameterIterator":{"itemInput":"item","items":{"raw":"[0,1]"}}}}}},
		"components":{
			"body":{"dag":{"tasks":{
				"a":{"componentRef":{"name":"leaf"}},
				"b":{"componentRef":{"name":"leaf"},"dependentTasks":["a"],"triggerPolicy":{"strategy":"ALL_UPSTREAM_TASKS_COMPLETED"}}
			}}},
			"leaf":{"executorLabel":"executor"}
		}
	}`), spec))
	dependencies, err := materializedRetryDependencies(tasks, spec)
	require.NoError(t, err)
	require.Equal(t, []string{"a0"}, dependencies["b0"])
	require.Equal(t, []string{"a1"}, dependencies["b1"])
	require.ElementsMatch(t, []string{"root", "loop", "iteration-0", "a0", "b0"}, tasksForManagedRetry(tasks, dependencies, false),
		"an invalidated downstream iteration task must rerun, while the independent successful iteration is preserved")
}
