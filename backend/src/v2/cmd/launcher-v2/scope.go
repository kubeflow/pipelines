package main

import (
	"fmt"

	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"google.golang.org/protobuf/types/known/structpb"
)

func launcherTaskScope(spec *structpb.Struct, parent, task *apiv2.PipelineTask, taskName, executorType string) (util.ScopePath, error) {
	if executorType != "" {
		if task.GetScopePath() == "" {
			return util.ScopePath{}, fmt.Errorf("task %q has no logical pipeline scope", task.GetTaskId())
		}
		return util.ScopePathFromDotNotation(spec, task.GetScopePath())
	}
	return util.ScopePathFromStringPathWithNewTask(spec, parent.GetScopePath(), taskName)
}
