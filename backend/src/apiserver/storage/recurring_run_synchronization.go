// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0

package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"

	sq "github.com/Masterminds/squirrel"
	"github.com/kubeflow/pipelines/backend/src/common/util"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const legacyRecurringRunRecordPrefix = LegacyRecurringRunAdoptionID + ":"

func legacyAdoptionIDs(receipt *model.RecurringRunAdoption) ([]string, error) {
	if receipt == nil {
		return nil, nil
	}
	var ids []string
	if err := json.Unmarshal([]byte(receipt.JobIDs), &ids); err != nil {
		return nil, fmt.Errorf("invalid legacy adoption inventory; restore the adoption receipt from trusted inventory or backup before retrying")
	}
	seen := make(map[string]bool)
	for _, id := range ids {
		if id == "" || seen[id] {
			return nil, fmt.Errorf("invalid legacy adoption inventory; restore the adoption receipt from trusted inventory or backup before retrying")
		}
		seen[id] = true
	}
	if int64(len(ids)) != receipt.AdoptedCount {
		return nil, fmt.Errorf("incomplete legacy adoption inventory; restore the adoption receipt from trusted inventory or backup before retrying")
	}
	return ids, nil
}

// GetLegacyRecurringRunAdoptionForJob prefers the record receipt; the old global
// receipt seals only its listed records and never freezes later inventory.
func GetLegacyRecurringRunAdoptionForJob(db *gorm.DB, id string) (*model.RecurringRunAdoption, error) {
	var receipt model.RecurringRunAdoption
	result := db.Where(&model.RecurringRunAdoption{ID: legacyRecurringRunRecordPrefix + id}).Limit(1).Find(&receipt)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected != 0 {
		ids, err := legacyAdoptionIDs(&receipt)
		if err != nil {
			return nil, err
		}
		if len(ids) != 1 || ids[0] != id {
			return nil, fmt.Errorf("invalid scheduling receipt inventory")
		}
		return &receipt, nil
	}
	global, err := GetLegacyRecurringRunAdoption(db)
	if err != nil {
		return nil, err
	}
	ids, err := legacyAdoptionIDs(global)
	if err != nil {
		return nil, err
	}
	for _, recorded := range ids {
		if recorded == id {
			return global, nil
		}
	}
	return nil, nil
}

func lockLegacyAdoptionJob(tx *gorm.DB, id string) (*model.Job, error) {
	var job model.Job
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where(&model.Job{UUID: id}).Take(&job).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

func legacyAdoptionState(tx *gorm.DB, id string) (*model.RecurringRunState, error) {
	var state model.RecurringRunState
	result := tx.Where(&model.RecurringRunState{JobUUID: id}).Limit(1).Find(&state)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}
	return &state, nil
}

// SynchronizeLegacyRecurringRunAdoption holds the job lock across the bounded
// Kubernetes update. Mode changes must use the same lock across their CR write.
// A failed CR update leaves the durable receipt pending; retries never reread CR
// progress into SQL or reset an advanced scheduling state.
func SynchronizeLegacyRecurringRunAdoption(db *gorm.DB, id string, synchronize func(*model.Job, *model.RecurringRunState) error) error {
	return db.Transaction(func(tx *gorm.DB) error {
		job, err := lockLegacyAdoptionJob(tx, id)
		if err != nil {
			return err
		}
		receipt, err := GetLegacyRecurringRunAdoptionForJob(tx, id)
		if err != nil {
			return err
		}
		state, err := legacyAdoptionState(tx, id)
		if err != nil {
			return err
		}
		if state == nil {
			return fmt.Errorf("adopted scheduling state is missing; refusing to reseed progress")
		}
		if receipt == nil || receipt.Ready {
			return nil
		}
		if err := synchronize(job, state); err != nil {
			return err
		}
		if receipt.ID == LegacyRecurringRunAdoptionID {
			ids, err := json.Marshal([]string{id})
			if err != nil {
				return err
			}
			return tx.Create(&model.RecurringRunAdoption{ID: legacyRecurringRunRecordPrefix + id, AdoptedCount: 1, CompletedAt: receipt.CompletedAt, JobIDs: model.LargeText(ids), Ready: true}).Error
		}
		return tx.Model(&model.RecurringRunAdoption{}).Where(&model.RecurringRunAdoption{ID: receipt.ID}).UpdateColumn("Ready", true).Error
	})
}

// ListPendingLegacyRecurringRunAdoptions includes late legacy inserts and records
// interrupted between SQL commit and CR synchronization, including disabled jobs.
func ListPendingLegacyRecurringRunAdoptions(db *gorm.DB, afterID string, limit uint64) ([]RecurringRunMigrationCandidate, error) {
	if limit == 0 || limit > 1000 {
		return nil, fmt.Errorf("adoption page size must be between 1 and 1000")
	}
	q := db.Statement.Quote
	jobID, stateID, receiptID := q("jobs.UUID"), q("recurring_run_states.JobUUID"), q("recurring_run_adoptions.ID")
	recordID := "? || " + jobID
	if db.Name() == "mysql" {
		recordID = "CONCAT(?, " + jobID + ")"
	}
	query := db.Model(&model.Job{}).
		Joins("LEFT JOIN "+q("recurring_run_states")+" ON "+stateID+" = "+jobID).
		Joins("LEFT JOIN "+q("recurring_run_adoptions")+" ON "+receiptID+" = "+recordID, legacyRecurringRunRecordPrefix).
		Where(jobID+" > ?", afterID)
	condition := stateID + " IS NULL OR " + q("recurring_run_adoptions.Ready") + " = ?"
	args := []any{false}
	// A pending global receipt makes unsealed jobs potential candidates. Do not
	// decode or bind its entire inventory for every page. Per-record recovery
	// validates membership and provenance before mutating either store; returning
	// nonmembers here also preserves keyset progress through a full candidate page.
	pendingGlobal := db.Table("recurring_run_adoptions AS global_adoption").Select("1").
		Where(q("global_adoption.ID")+" = ? AND "+q("global_adoption.Ready")+" = ?", LegacyRecurringRunAdoptionID, false)
	condition += " OR (" + receiptID + " IS NULL AND EXISTS (?))"
	args = append(args, pendingGlobal)
	var jobs []model.Job
	if err := query.Where("("+condition+")", args...).Select(q("jobs") + ".*").Order(jobID).Limit(int(limit)).Find(&jobs).Error; err != nil {
		return nil, err
	}
	result := make([]RecurringRunMigrationCandidate, 0, len(jobs))
	for _, job := range jobs {
		result = append(result, RecurringRunMigrationCandidate{ID: job.UUID, Namespace: job.Namespace, Name: job.K8SName, Enabled: job.Enabled})
	}
	return result, nil
}

// SetRecurringRunModeForReconciliation commits desired SQL authorization before
// any CR mutation. Existing state gets a durable pending reconciliation receipt;
// missing-state legacy jobs remain discoverable without premature sealing.
func SetRecurringRunModeForReconciliation(db *gorm.DB, snapshot model.Job, enabled bool, now int64) error {
	return db.Transaction(func(tx *gorm.DB) error {
		current, err := lockLegacyAdoptionJob(tx, snapshot.UUID)
		if err != nil {
			return err
		}
		_, err = GetLegacyRecurringRunAdoptionForJob(tx, current.UUID)
		if err != nil {
			return err
		}
		snapshot.Enabled = current.Enabled
		if !sameRecurringRunAdoptionJob(*current, snapshot) {
			return fmt.Errorf("recurring run changed during mode authorization; retry")
		}
		if err := tx.Model(&model.Job{}).Where(&model.Job{UUID: current.UUID}).Updates(map[string]any{"Enabled": enabled, "UpdatedAtInSec": now}).Error; err != nil {
			return err
		}
		state, err := legacyAdoptionState(tx, current.UUID)
		if err != nil {
			return err
		}
		if state == nil {
			return fmt.Errorf("complete recurring-run adoption in KFP 2.18 before upgrading")
		}
		ids, err := json.Marshal([]string{current.UUID})
		if err != nil {
			return err
		}
		record := model.RecurringRunAdoption{ID: legacyRecurringRunRecordPrefix + current.UUID, AdoptedCount: 1, CompletedAt: now, JobIDs: model.LargeText(ids)}
		return tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "ID"}}, DoUpdates: clause.Assignments(map[string]any{"Ready": false})}).Create(&record).Error
	})
}

func (s *JobStore) requireAdoptionReadyForClaim(tx *sql.Tx, id string) error {
	q := s.dbDialect.QuoteIdentifier
	query, args, err := s.dbDialect.QueryBuilder().Select(q("ID"), q("AdoptedCount"), q("JobIDs"), q("Ready")).From(q("recurring_run_adoptions")).Where(sq.Eq{q("ID"): []string{LegacyRecurringRunAdoptionID, legacyRecurringRunRecordPrefix + id}}).ToSql()
	if err != nil {
		return err
	}
	rows, err := tx.Query(query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	var global *model.RecurringRunAdoption
	var record *model.RecurringRunAdoption
	for rows.Next() {
		var receipt model.RecurringRunAdoption
		if err := rows.Scan(&receipt.ID, &receipt.AdoptedCount, &receipt.JobIDs, &receipt.Ready); err != nil {
			return err
		}
		if receipt.ID == LegacyRecurringRunAdoptionID {
			global = &receipt
		} else {
			record = &receipt
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if record != nil {
		ids, err := legacyAdoptionIDs(record)
		if err != nil {
			return err
		}
		if len(ids) != 1 || ids[0] != id {
			return fmt.Errorf("invalid scheduling receipt inventory")
		}
		if record.Ready {
			return nil
		}
		return util.NewUnavailableServerError(fmt.Errorf("adoption synchronization is pending"), "Retry the schedule after synchronization")
	}
	if global != nil {
		ids, err := legacyAdoptionIDs(global)
		if err != nil {
			return err
		}
		for _, jobID := range ids {
			if jobID == id {
				if global.Ready {
					return nil
				}
				return util.NewUnavailableServerError(fmt.Errorf("adoption synchronization is pending"), "Retry the schedule after synchronization")
			}
		}
	}
	return nil
}

const LegacyRecurringRunAdoptionID = "legacy-2.18"

func GetLegacyRecurringRunAdoption(db *gorm.DB) (*model.RecurringRunAdoption, error) {
	var receipt model.RecurringRunAdoption
	result := db.Where(&model.RecurringRunAdoption{ID: LegacyRecurringRunAdoptionID}).Limit(1).Find(&receipt)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}
	return &receipt, nil
}
func sameRecurringRunAdoptionJob(current, snapshot model.Job) bool {
	current.Conditions, snapshot.Conditions = "", ""
	current.UpdatedAtInSec, snapshot.UpdatedAtInSec = 0, 0
	current.ResourceReferences, snapshot.ResourceReferences = nil, nil
	return reflect.DeepEqual(current, snapshot)
}
