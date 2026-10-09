// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0

package resource

import (
	"context"
	"fmt"
	"time"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/apiserver/template"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"gorm.io/gorm"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

// RequireRecurringRunAdoptionReady checks only this record. An incomplete
// historical global receipt must not stall unrelated or newly created schedules.
func (r *ResourceManager) RequireRecurringRunAdoptionReady(ctx context.Context, id string) error {
	if _, err := r.jobStore.GetRecurringRunState(id); err != nil {
		return err
	}
	db, err := r.recurringRunAdoptionDB(ctx)
	if err != nil {
		return err
	}
	receipt, err := storage.GetLegacyRecurringRunAdoptionForJob(db, id)
	if err != nil {
		return err
	}
	if receipt != nil && !receipt.Ready {
		return util.NewUnavailableServerError(fmt.Errorf("recurring-run adoption is incomplete"), "The schedule is being synchronized; retry shortly")
	}
	return nil
}

func (r *ResourceManager) ListPendingRecurringRunSynchronizations(ctx context.Context, afterID string, limit uint64) ([]storage.RecurringRunMigrationCandidate, error) {
	db, err := r.recurringRunAdoptionDB(ctx)
	if err != nil {
		return nil, err
	}
	return storage.ListPendingLegacyRecurringRunAdoptions(db, afterID, limit)
}

// SynchronizeRecurringRun is called only after the automatic legacy-writer
// handoff fence. Ordinary APIs remain available while individual records retry.
func (r *ResourceManager) SynchronizeRecurringRun(ctx context.Context, id string) error {
	if !common.IsMultiUserMode() {
		return nil
	}
	db, err := r.recurringRunAdoptionDB(ctx)
	if err != nil {
		return err
	}
	if _, err := r.jobStore.GetRecurringRunState(id); err != nil {
		return err
	}
	// Resolve immutable, reference-backed identity before taking the SQL lock;
	// callback reads no second connection while the transaction owns the job.
	resolved, err := r.jobStore.GetJob(id)
	if err != nil {
		return err
	}
	syncCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return storage.SynchronizeLegacyRecurringRunAdoption(db.WithContext(syncCtx), id, func(raw *model.Job, state *model.RecurringRunState) error {
		job := *raw
		if job.Namespace == "" {
			job.Namespace = resolved.Namespace
		}
		if job.ExperimentId == "" {
			job.ExperimentId = resolved.ExperimentId
		}
		if job.PipelineId == "" {
			job.PipelineId = resolved.PipelineId
		}
		if job.PipelineVersionId == "" {
			job.PipelineVersionId = resolved.PipelineVersionId
		}
		canonical, err := template.NewGenericScheduledWorkflow(&job)
		if err != nil {
			return err
		}
		return retry.RetryOnConflict(retry.DefaultRetry, func() error {
			live, err := r.getScheduledWorkflowClient(job.Namespace).Get(syncCtx, job.K8SName, metav1.GetOptions{})
			// Revocation succeeds even if its CR has already disappeared.
			// Never recreate it; enabling still requires the original CR.
			if !job.Enabled && apierrors.IsNotFound(err) {
				return nil
			}
			if err != nil {
				return err
			}
			if live == nil || string(live.UID) != job.UUID || live.Name != job.K8SName || live.Namespace != job.Namespace {
				return fmt.Errorf("backing ScheduledWorkflow identity changed")
			}
			live = live.DeepCopy()
			live.Spec = canonical.Spec
			// A mode reconciliation must not acknowledge an in-flight claim.
			if !state.Pending {
				live.Status.Trigger.LastIndex = util.Int64Pointer(state.LastRunIndex)
				live.Status.Trigger.LastTriggeredTime = nil
				if state.LastRunIndex > 0 {
					last := metav1.NewTime(time.Unix(state.LastScheduledAtInSec, 0))
					live.Status.Trigger.LastTriggeredTime = &last
				}
			}
			_, err = r.getScheduledWorkflowClient(job.Namespace).Update(syncCtx, live)
			return err
		})
	})
}

// changeAdoptableJobMode authorizes before locking, then verifies the captured
// definition under the same lock used by adoption before changing either store.
func (r *ResourceManager) changeAdoptableJobMode(ctx context.Context, job *model.Job, enabled bool) error {
	db, err := r.recurringRunAdoptionDB(ctx)
	if err != nil {
		return err
	}
	var snapshot model.Job
	if err := db.Where(&model.Job{UUID: job.UUID}).Take(&snapshot).Error; err != nil {
		return err
	}
	namespace := job.Namespace
	if namespace == "" {
		namespace = common.GetPodNamespace()
	}
	if enabled {
		swf, err := r.getScheduledWorkflowClient(namespace).Get(ctx, job.K8SName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if swf == nil || string(swf.UID) != job.UUID {
			return fmt.Errorf("ScheduledWorkflow identity differs from its job")
		}
		if swf.Spec.Workflow != nil && swf.Spec.Workflow.Spec != nil {
			execution, err := util.ScheduleSpecToExecutionSpec(util.ArgoWorkflow, swf.Spec.Workflow)
			if err != nil {
				return err
			}
			allow, err := r.allowsCompilerPodSpecPatch(job.PipelineSpec)
			if err != nil {
				return err
			}
			if err := r.authorizeExecutionServiceAccounts(ctx, execution, allow, namespace, "enable_recurring_run"); err != nil {
				return err
			}
		}
	}

	if err := storage.SetRecurringRunModeForReconciliation(db, snapshot, enabled, r.time.Now().Unix()); err != nil {
		return err
	}
	if r.options != nil && r.options.EnsureRecurringRunSynchronized != nil {
		return r.options.EnsureRecurringRunSynchronized(ctx, job.UUID)
	}
	return r.SynchronizeRecurringRun(ctx, job.UUID)
}

func (r *ResourceManager) recurringRunAdoptionDB(ctx context.Context) (*gorm.DB, error) {
	db, err := r.transferDB()
	if err != nil {
		return nil, err
	}
	return db.WithContext(ctx), nil
}
