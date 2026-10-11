// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0

package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"reflect"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/validation"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// LegacyRecurringRunAdoptionID identifies the single pre-2.18 adoption window.
const LegacyRecurringRunAdoptionID = "legacy-2.18"

// RecurringRunAdoptionCandidate pairs a reviewed SQL snapshot with legacy progress.
type RecurringRunAdoptionCandidate struct {
	Job   model.Job
	State model.RecurringRunState
}

// GetLegacyRecurringRunAdoption returns nil when adoption has not started.
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

// ApplyLegacyRecurringRunAdoption inserts all missing scheduling states and the
// global receipt atomically. Once receipted, retries return the recorded result.
func ApplyLegacyRecurringRunAdoption(db *gorm.DB, candidates []RecurringRunAdoptionCandidate, completedAt int64) (*model.RecurringRunAdoption, error) {
	if receipt, err := GetLegacyRecurringRunAdoption(db); err != nil || receipt != nil {
		return receipt, err
	}
	if completedAt <= 0 {
		return nil, fmt.Errorf("adoption completion time must be positive; supply the current Unix time")
	}
	byID := make(map[string]RecurringRunAdoptionCandidate, len(candidates))
	for _, candidate := range candidates {
		if err := validateRecurringRunAdoption(candidate); err != nil {
			return nil, err
		}
		if _, exists := byID[candidate.Job.UUID]; exists {
			return nil, fmt.Errorf("recurring run %s appears twice; rebuild the adoption inventory", candidate.Job.UUID)
		}
		byID[candidate.Job.UUID] = candidate
	}

	var receipt *model.RecurringRunAdoption
	err := db.Transaction(func(tx *gorm.DB) error {
		var err error
		receipt, err = GetLegacyRecurringRunAdoption(tx)
		if err != nil || receipt != nil {
			return err
		}
		states := tx.Model(&model.RecurringRunState{}).Select("JobUUID")
		var missing []model.Job
		// Legacy writers must be stopped before adoption. Current API writers
		// insert jobs and scheduling states together, so they cannot add missing
		// states after this serializable inventory check.
		if err := tx.Where(clause.Expr{SQL: "? NOT IN (?)", Vars: []any{clause.Column{Name: "UUID"}, states}}).
			Order(clause.OrderByColumn{Column: clause.Column{Name: "UUID"}}).
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Find(&missing).Error; err != nil {
			return err
		}
		if len(missing) != len(candidates) {
			return fmt.Errorf("missing scheduling-state inventory changed; rebuild and review the adoption inventory")
		}
		jobIDs := make([]string, 0, len(missing))
		for _, current := range missing {
			candidate, exists := byID[current.UUID]
			if !exists || !sameRecurringRunAdoptionJob(current, candidate.Job) {
				return fmt.Errorf("recurring run %s changed; rebuild and review the adoption inventory", current.UUID)
			}
			sealed, err := GetLegacyRecurringRunAdoptionForJob(tx, current.UUID)
			if err != nil {
				return err
			}
			if sealed != nil {
				return fmt.Errorf("adopted scheduling state is missing; refusing to reseed progress")
			}
			if err := tx.Create(&candidate.State).Error; err != nil {
				return err
			}
			jobIDs = append(jobIDs, current.UUID)
		}
		encodedIDs, err := json.Marshal(jobIDs)
		if err != nil {
			return err
		}
		receipt = &model.RecurringRunAdoption{ID: LegacyRecurringRunAdoptionID, AdoptedCount: int64(len(candidates)), CompletedAt: completedAt, JobIDs: model.LargeText(encodedIDs)}
		return tx.Create(receipt).Error
	}, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		// Another transaction may have won the fixed receipt's unique key.
		if recorded, readErr := GetLegacyRecurringRunAdoption(db); readErr == nil && recorded != nil {
			return recorded, nil
		}
		return nil, err
	}
	return receipt, nil
}

// CompleteLegacyRecurringRunAdoption releases execution after CR synchronization.
func CompleteLegacyRecurringRunAdoption(db *gorm.DB) error {
	result := db.Model(&model.RecurringRunAdoption{}).
		Where(&model.RecurringRunAdoption{ID: LegacyRecurringRunAdoptionID}).
		UpdateColumn("Ready", true)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		// MySQL can report zero changed rows for an already-ready receipt.
		receipt, err := GetLegacyRecurringRunAdoption(db)
		if err != nil {
			return err
		}
		if receipt == nil {
			return fmt.Errorf("legacy adoption receipt is missing; apply adoption before completing it")
		}
	}
	return nil
}

func sameRecurringRunAdoptionJob(current, snapshot model.Job) bool {
	// Reports may update lifecycle status, but cannot change execution inputs.
	current.Conditions, snapshot.Conditions = "", ""
	current.UpdatedAtInSec, snapshot.UpdatedAtInSec = 0, 0
	current.ResourceReferences, snapshot.ResourceReferences = nil, nil
	return reflect.DeepEqual(current, snapshot)
}

func validateRecurringRunAdoption(candidate RecurringRunAdoptionCandidate) error {
	state := candidate.State
	if candidate.Job.UUID == "" || state.JobUUID != candidate.Job.UUID {
		return fmt.Errorf("adoption job and scheduling-state identities must match; rebuild the inventory")
	}
	if state.Pending || state.LastRunIndex < 0 || state.LastRunIndex == math.MaxInt64 ||
		state.LastScheduledAtInSec < 0 || state.LastScheduledAtInSec == math.MaxInt64 ||
		state.LastCreatedAtInSec < 0 || state.LastCreatedAtInSec == math.MaxInt64 {
		return fmt.Errorf("recurring run %s has invalid scheduling progress; review its legacy status", candidate.Job.UUID)
	}
	if state.LastRunIndex == 0 {
		if state.LastScheduledAtInSec != 0 || state.LastCreatedAtInSec != 0 || state.RequestKey != "" || state.LastRunUUID != "" {
			return fmt.Errorf("recurring run %s has progress without a run index; review its legacy status", candidate.Job.UUID)
		}
		return nil
	}
	// An identity-verified retained execution may predate a later acknowledged
	// controller timestamp. Preserve its real creation time, never synthesize it.
	if state.LastCreatedAtInSec == 0 || (state.LastCreatedAtInSec < state.LastScheduledAtInSec && state.LastRunUUID == "") {
		return fmt.Errorf("adoption progress lacks a valid historical creation time")
	}
	if state.LastScheduledAtInSec == 0 {
		return fmt.Errorf("recurring run %s has no scheduled time for its run index; review its legacy status", candidate.Job.UUID)
	}
	return validation.ValidateRecurringRunRequestKey(state.RequestKey)
}
