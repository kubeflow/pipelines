// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

package storage

import (
	"encoding/json"
	"fmt"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PrepareRecurringRunStartupRepairs records bounded, durable repair work before
// the startup scan advances. Only existing trusted SQL state is eligible; this
// never imports progress or recreates a deleted schedule. A failed Kubernetes
// update remains visible to the normal pending reconciliation scan.
func PrepareRecurringRunStartupRepairs(db *gorm.DB, afterID string, limit uint64, now int64) ([]RecurringRunMigrationCandidate, error) {
	if limit == 0 || limit > 1000 {
		return nil, fmt.Errorf("repair page size must be between 1 and 1000")
	}
	var result []RecurringRunMigrationCandidate
	err := db.Transaction(func(tx *gorm.DB) error {
		var jobs []model.Job
		q := tx.Statement.Quote
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where(q("UUID")+" > ?", afterID).Order(q("UUID")).Limit(int(limit)).Find(&jobs).Error; err != nil {
			return err
		}
		for _, job := range jobs {
			result = append(result, RecurringRunMigrationCandidate{ID: job.UUID, Namespace: job.Namespace, Name: job.K8SName, Enabled: job.Enabled})
			var count int64
			if err := tx.Model(&model.RecurringRunState{}).Where(&model.RecurringRunState{JobUUID: job.UUID}).Count(&count).Error; err != nil {
				return err
			}
			if count == 0 {
				continue
			}
			var existing int64
			if err := tx.Model(&model.RecurringRunAdoption{}).Where(&model.RecurringRunAdoption{ID: legacyRecurringRunRecordPrefix + job.UUID}).Count(&existing).Error; err != nil {
				return err
			}
			if existing == 0 {
				// A per-record seal must not override invalid global provenance.
				global, err := GetLegacyRecurringRunAdoption(tx)
				if err != nil {
					return err
				}
				if _, err := legacyAdoptionIDs(global); err != nil {
					return err
				}
			}
			ids, err := json.Marshal([]string{job.UUID})
			if err != nil {
				return err
			}
			receipt := model.RecurringRunAdoption{ID: legacyRecurringRunRecordPrefix + job.UUID, AdoptedCount: 1, CompletedAt: now, JobIDs: model.LargeText(ids)}
			// Existing inventory is retained, including malformed inventory that must
			// fail validation rather than be silently replaced by startup repair.
			if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "ID"}}, DoUpdates: clause.Assignments(map[string]any{"Ready": false})}).Create(&receipt).Error; err != nil {
				return err
			}
		}
		return nil
	})
	return result, err
}
