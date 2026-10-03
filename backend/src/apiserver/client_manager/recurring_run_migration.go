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

// Package clientmanager initializes and owns API-server clients and stores.
package clientmanager

import (
	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
)

type recurringRunMigrationStore interface {
	ListJobsWithoutRecurringRunState(afterID string, limit uint64) ([]storage.RecurringRunMigrationCandidate, error)
}

func reportRecurringRunMigration(store recurringRunMigrationStore, warnf func(string, ...any)) error {
	const pageSize = 100
	afterID, count := "", 0
	for {
		candidates, err := store.ListJobsWithoutRecurringRunState(afterID, pageSize)
		if err != nil {
			return err
		}
		for _, candidate := range candidates {
			warnf("recurring_run_migration action=recreate id=%q namespace=%q scheduledworkflow=%q enabled=%t: no trusted scheduling state; review inputs and recreate through the KFP API; see docs/operator-guides/scheduled-service-accounts.md", candidate.ID, candidate.Namespace, candidate.Name, candidate.Enabled)
			afterID = candidate.ID
			count++
		}
		if len(candidates) < pageSize {
			break
		}
	}
	if count > 0 {
		warnf("recurring_run_migration affected_jobs=%d: existing recurring runs require review and recreation before their next execution", count)
	} else {
		glog.Info("recurring_run_migration affected_jobs=0")
	}
	return nil
}
