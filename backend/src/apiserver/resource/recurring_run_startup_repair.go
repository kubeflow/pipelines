// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

package resource

import (
	"context"

	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
)

// PrepareRecurringRunStartupRepairs is called by the worker only after writer
// handoff. Persisting the page first makes failed repairs survive API restarts.
func (r *ResourceManager) PrepareRecurringRunStartupRepairs(ctx context.Context, afterID string, limit uint64) ([]storage.RecurringRunMigrationCandidate, error) {
	db, err := r.recurringRunAdoptionDB(ctx)
	if err != nil {
		return nil, err
	}
	return storage.PrepareRecurringRunStartupRepairs(db, afterID, limit, r.time.Now().Unix())
}
