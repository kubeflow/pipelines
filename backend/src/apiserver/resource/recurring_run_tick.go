// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package resource

import (
	"fmt"
	"time"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/template"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	scheduleutil "github.com/kubeflow/pipelines/backend/src/crd/controller/scheduledworkflow/util"
	scheduledworkflow "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// A preview does not reserve capacity until normal pipeline and account checks pass.
type recurringRunTick struct {
	previousIndex int64
	index         int64
	scheduledAt   int64
	createdAt     int64
	replay        *model.Run
}

func (r *ResourceManager) prepareRecurringRunTick(run *model.Run, owner *scheduledworkflow.ScheduledWorkflow, now int64) (*recurringRunTick, error) {
	job, err := r.jobStore.GetJob(run.RecurringRunId)
	if err != nil {
		return nil, err
	}
	if owner == nil || string(owner.UID) != job.UUID || owner.Name != job.K8SName || owner.Namespace != run.Namespace {
		return nil, util.NewPermissionDeniedError(fmt.Errorf("ScheduledWorkflow identity changed"),
			"Recreate the recurring run through the KFP API")
	}
	if !job.Enabled || !owner.Spec.Enabled {
		return nil, util.NewFailedPreconditionError(fmt.Errorf("the recurring run is disabled"),
			"Enable the recurring run through the KFP API before triggering runs")
	}
	state, err := r.jobStore.GetRecurringRunState(job.UUID)
	if err != nil {
		return nil, err
	}
	existingID, err := r.runStore.GetRunByRecurringRunIDAndDisplayName(job.UUID, run.DisplayName)
	if err != nil {
		return nil, err
	}
	if existingID != "" {
		existing, err := r.runStore.GetRun(existingID)
		if err != nil {
			return nil, err
		}
		run.PipelineVersionId = existing.PipelineVersionId
		return &recurringRunTick{
			index: state.LastRunIndex, scheduledAt: existing.ScheduledAtInSec,
			createdAt: existing.CreatedAtInSec, replay: existing,
		}, nil
	}
	if state.LastRunIndex > 0 && state.RequestKey == run.DisplayName {
		if !state.Pending {
			return nil, util.NewFailedPreconditionError(fmt.Errorf("the scheduled run was already dispatched"),
				"This tick has already executed; wait for the next scheduled tick")
		}
		run.PipelineVersionId = state.PipelineVersionID
		return &recurringRunTick{
			previousIndex: state.LastRunIndex - 1, index: state.LastRunIndex,
			scheduledAt: state.LastScheduledAtInSec, createdAt: state.LastCreatedAtInSec,
		}, nil
	}
	if state.Pending {
		return nil, util.NewFailedPreconditionError(fmt.Errorf("a previous tick is still pending"),
			"Retry the previous scheduled tick before submitting another one")
	}
	schedule, err := template.NewGenericScheduledWorkflow(job)
	if err != nil {
		return nil, err
	}
	// Kubernetes owns creationTimestamp; unlike spec/status it is immutable.
	schedule.CreationTimestamp = owner.CreationTimestamp
	if schedule.CreationTimestamp.IsZero() {
		schedule.CreationTimestamp = metav1.NewTime(time.Unix(job.CreatedAtInSec, 0))
	}
	if state.LastRunIndex > 0 {
		last := metav1.NewTime(time.Unix(state.LastScheduledAtInSec, 0))
		schedule.Status.Trigger.LastTriggeredTime = &last
	}
	location, err := scheduleutil.GetLocation()
	if err != nil {
		return nil, util.Wrap(err, "Failed to resolve the recurring-run timezone")
	}
	scheduledAt, due := scheduleutil.NewScheduledWorkflow(schedule).GetNextScheduledEpoch(0, now, *location)
	if !due {
		return nil, util.NewFailedPreconditionError(fmt.Errorf("no authorized tick is due"),
			"Wait for the next tick allowed by the API-stored recurring-run schedule")
	}
	return &recurringRunTick{
		previousIndex: state.LastRunIndex, index: state.LastRunIndex + 1,
		scheduledAt: scheduledAt, createdAt: now,
	}, nil
}

func (r *ResourceManager) claimRecurringRunTick(run *model.Run, tick *recurringRunTick) error {
	state, err := r.jobStore.ClaimRecurringRun(run.RecurringRunId, run.DisplayName,
		tick.previousIndex, tick.scheduledAt, tick.createdAt, run.PipelineVersionId)
	if err != nil {
		return err
	}
	// A concurrent claimant may have frozen a different CurrentTime/NoCatchup time.
	// Retry preparation so compilation uses precisely that durable claim.
	if state.LastRunIndex != tick.index || state.LastScheduledAtInSec != tick.scheduledAt || state.LastCreatedAtInSec != tick.createdAt || state.PipelineVersionID != run.PipelineVersionId {
		return util.NewUnavailableServerError(fmt.Errorf("the tick was claimed concurrently"),
			"Retry this tick to use its persisted execution inputs")
	}
	return nil
}
