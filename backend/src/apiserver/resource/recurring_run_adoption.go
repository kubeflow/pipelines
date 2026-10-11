// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0

package resource

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/apiserver/template"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"gorm.io/gorm"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

// AdoptLegacyRecurringRuns accepts persisted jobs as the one-time upgrade
// baseline. All schedule writers and controllers must be stopped first. It does
// not authenticate historical ownership or copy execution inputs from a CR.
func (r *ResourceManager) AdoptLegacyRecurringRuns(ctx context.Context) (*model.RecurringRunAdoption, error) {
	if !common.IsMultiUserMode() {
		return nil, fmt.Errorf("legacy recurring-run adoption requires MULTIUSER=true")
	}
	db, err := r.recurringRunAdoptionDB(ctx)
	if err != nil {
		return nil, err
	}
	receipt, err := storage.GetLegacyRecurringRunAdoption(db)
	if err != nil || (receipt != nil && receipt.Ready) {
		return receipt, err
	}
	if receipt == nil {
		candidates, err := r.legacyRecurringRunCandidates(ctx, db)
		if err != nil {
			return nil, err
		}
		receipt, err = storage.ApplyLegacyRecurringRunAdoption(db, candidates, r.time.Now().Unix())
		if err != nil {
			return nil, err
		}
	}
	// SQL and Kubernetes cannot commit atomically. A durable, incomplete receipt
	// blocks ticks while retries finish synchronizing CRs from SQL, never vice versa.
	var ids []string
	if err := json.Unmarshal([]byte(receipt.JobIDs), &ids); err != nil {
		return nil, fmt.Errorf("invalid legacy adoption receipt; restore its recorded job inventory: %w", err)
	}
	if int64(len(ids)) != receipt.AdoptedCount {
		return nil, fmt.Errorf("legacy adoption receipt inventory is incomplete; restore the receipt before retrying")
	}
	for _, id := range ids {
		if err := r.synchronizeAdoptedRecurringRun(ctx, id); err != nil {
			return nil, fmt.Errorf("recurring run %s was adopted but is not synchronized; keep the controller stopped and retry adoption: %w", id, err)
		}
	}
	if err := storage.CompleteLegacyRecurringRunAdoption(db); err != nil {
		return nil, err
	}
	receipt.Ready = true
	return receipt, nil
}

func (r *ResourceManager) recurringRunAdoptionDB(ctx context.Context) (*gorm.DB, error) {
	if r.transferDB == nil {
		return nil, fmt.Errorf("recurring-run adoption database is unavailable")
	}
	db, err := r.transferDB()
	if err != nil {
		return nil, err
	}
	return db.WithContext(ctx), nil
}

func (r *ResourceManager) legacyRecurringRunCandidates(ctx context.Context, db *gorm.DB) ([]storage.RecurringRunAdoptionCandidate, error) {
	transferred, err := r.transferScheduleIDs(ctx)
	if err != nil {
		return nil, err
	}
	inventory, ok := r.jobStore.(interface {
		ListJobsWithoutRecurringRunState(string, uint64) ([]storage.RecurringRunMigrationCandidate, error)
	})
	if !ok {
		return nil, fmt.Errorf("legacy recurring-run inventory is unavailable")
	}
	runInventory, ok := r.runStore.(interface {
		ListRunIDsForRecurringRun(string) ([]string, error)
	})
	if !ok {
		return nil, fmt.Errorf("legacy execution inventory is unavailable")
	}
	var candidates []storage.RecurringRunAdoptionCandidate
	workflows := map[string]util.ExecutionSpecList{}
	for cursor := ""; ; {
		page, err := inventory.ListJobsWithoutRecurringRunState(cursor, 1000)
		if err != nil {
			return nil, err
		}
		if len(page) == 0 {
			return candidates, nil
		}
		for _, entry := range page {
			if transferred[entry.ID] {
				return nil, fmt.Errorf("transferred scheduling state is missing; refusing to reseed progress")
			}
			// GetJob resolves older reference-backed identity columns. Keep the
			// raw row separately for the transaction's unchanged-snapshot check.
			var snapshot model.Job
			if err := db.Where(&model.Job{UUID: entry.ID}).Take(&snapshot).Error; err != nil {
				return nil, err
			}
			job, err := r.jobStore.GetJob(entry.ID)
			if err != nil {
				return nil, err
			}
			if r.IsEmptyNamespace(job.Namespace) {
				return nil, fmt.Errorf("recurring run %s has no stored namespace; resolve its destination before adoption", job.UUID)
			}
			swf, err := r.getScheduledWorkflowClient(job.Namespace).Get(ctx, job.K8SName, metav1.GetOptions{})
			if err != nil {
				return nil, fmt.Errorf("recurring run %s backing CR is unavailable; resolve its identity before adoption: %w", job.UUID, err)
			}
			if _, exists := workflows[job.Namespace]; !exists {
				live, err := r.getWorkflowClient(job.Namespace).List(ctx, metav1.ListOptions{})
				if err != nil {
					return nil, err
				}
				if live == nil {
					return nil, fmt.Errorf("execution inventory unavailable in namespace %s; retry adoption", job.Namespace)
				}
				workflows[job.Namespace] = *live
			}
			var runs []*model.Run
			ids, err := runInventory.ListRunIDsForRecurringRun(job.UUID)
			if err != nil {
				return nil, err
			}
			for _, id := range ids {
				// Resolve reference-backed identity just as normal run reads do.
				run, err := r.runStore.GetRun(id)
				if err != nil {
					return nil, err
				}
				runs = append(runs, run)
			}
			state, err := adoptLegacyRecurringRunProgress(job, swf, runs, workflows[job.Namespace], r.time.Now().Unix())
			if err != nil {
				return nil, fmt.Errorf("recurring run %s cannot be adopted: %w", job.UUID, err)
			}
			candidates = append(candidates, storage.RecurringRunAdoptionCandidate{Job: snapshot, State: *state})
		}
		cursor = page[len(page)-1].ID
	}
}

func (r *ResourceManager) synchronizeAdoptedRecurringRun(ctx context.Context, id string) error {
	job, err := r.jobStore.GetJob(id)
	if err != nil {
		return err
	}
	state, err := r.jobStore.GetRecurringRunState(id)
	if err != nil {
		return err
	}
	canonical, err := template.NewGenericScheduledWorkflow(job)
	if err != nil {
		return err
	}
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		live, err := r.getScheduledWorkflowClient(job.Namespace).Get(ctx, job.K8SName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if live == nil || string(live.UID) != job.UUID || live.Name != job.K8SName || live.Namespace != job.Namespace {
			return fmt.Errorf("backing ScheduledWorkflow identity changed; restore the adopted CR identity")
		}
		original := live
		live = live.DeepCopy()
		live.Spec = canonical.Spec
		// A rerun after online adoption must not acknowledge an in-flight claim.
		if !state.Pending {
			live.Status.Trigger.LastIndex = util.Int64Pointer(state.LastRunIndex)
			live.Status.Trigger.LastTriggeredTime = nil
			if state.LastRunIndex > 0 {
				last := metav1.NewTime(time.Unix(state.LastScheduledAtInSec, 0))
				live.Status.Trigger.LastTriggeredTime = &last
			}
		}
		if equality.Semantic.DeepEqual(original.Spec, live.Spec) && equality.Semantic.DeepEqual(original.Status, live.Status) {
			return nil
		}
		_, err = r.getScheduledWorkflowClient(job.Namespace).Update(ctx, live)
		return err
	})
}

func (r *ResourceManager) requireRecurringRunAdoptionReady(ctx context.Context) error {
	db, err := r.recurringRunAdoptionDB(ctx)
	if err != nil {
		return err
	}
	receipt, err := storage.GetLegacyRecurringRunAdoption(db)
	if err != nil {
		return err
	}
	if receipt != nil && !receipt.Ready {
		return util.NewUnavailableServerError(fmt.Errorf("legacy recurring-run adoption is incomplete"),
			"Keep the scheduled workflow controller stopped and rerun --adopt-legacy-recurring-runs before resuming schedules")
	}
	return nil
}
