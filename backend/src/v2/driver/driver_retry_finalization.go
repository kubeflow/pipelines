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

package driver

import (
	"context"
	"fmt"

	api "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/v2/client_manager"
	"github.com/kubeflow/pipelines/backend/src/v2/driver/common"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func persistDriverRetryFailure(ctx context.Context, opts common.Options, manager client_manager.ClientManagerInterface, task *api.PipelineTask, failure error) error {
	setDriverRetryOwner(task, opts)
	finalAttempt := opts.DriverRetryAttempt == opts.DriverRetryMaxCount
	if !finalAttempt {
		task.State = api.PipelineTask_RUNNING
		task.EndTime = nil
	} else if task.GetState() != api.PipelineTask_CACHED && task.GetState() != api.PipelineTask_SKIPPED && task.GetState() != api.PipelineTask_SUCCEEDED {
		// A final driver error must not leave its task or parent DAG running
		// while unrelated branches delay the terminal workflow report.
		task.State = api.PipelineTask_FAILED
		task.EndTime = timestamppb.Now()
	}
	driverRecoveryMetadata(task).Message = failure.Error()
	updated, err := updateDriverTask(ctx, manager.KFPAPIClient(), task)
	if err != nil || !finalAttempt {
		return err
	}
	// Propagate only after the original attempt's fenced write succeeds.
	fullView := api.GetRunRequest_FULL
	run, err := manager.KFPAPIClient().GetRun(ctx, &api.GetRunRequest{RunId: task.GetRunId(), View: &fullView})
	if err != nil {
		return fmt.Errorf("failed to refresh run after final driver failure: %w", err)
	}
	if err := manager.KFPAPIClient().UpdateStatuses(ctx, run, opts.ScopePath.GetPipelineSpecStruct(), updated); err != nil {
		return fmt.Errorf("failed to propagate final driver status: %w", err)
	}
	return nil
}
