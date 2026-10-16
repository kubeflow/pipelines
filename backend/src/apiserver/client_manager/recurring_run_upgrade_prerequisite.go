// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0

package clientmanager

import (
	"encoding/json"
	"fmt"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const recurringRunAdoptionSourceID = "legacy-2.18"

// requireCompletedRecurringRunAdoption checks the source database without
// creating tables or changing progress. Runtime receipt checks remain necessary:
// a 2.18 replica can still change a schedule after this startup snapshot.
// Valid pending receipts are recoverable CR synchronization, not missing SQL
// adoption. Rejecting them here would prevent the new API from registering and
// completing the rollout that the synchronization worker waits for.
func requireCompletedRecurringRunAdoption(db *gorm.DB) error {
	if err := checkCompletedRecurringRunAdoption(db); err != nil {
		return fmt.Errorf("3.0 recurring-run upgrade prerequisite failed: %w; finish automatic SQL-state adoption on the fixed 2.18 release before upgrading to 3.0 (including disabled schedules)", err)
	}
	return nil
}

func checkCompletedRecurringRunAdoption(db *gorm.DB) error {
	// GetTables reports catalog failures; HasTable alone can mistake an error for
	// a fresh installation and accidentally permit migrations.
	tables, err := db.Migrator().GetTables()
	if err != nil {
		return fmt.Errorf("cannot inspect source schema: %w", err)
	}
	present := make(map[string]bool, len(tables))
	for _, table := range tables {
		present[table] = true
	}
	if !present["jobs"] {
		return nil
	}
	var ids []string
	page := func(after string, ids *[]string) error {
		query := db.Model(&model.Job{}).Order(clause.OrderByColumn{Column: clause.Column{Name: "UUID"}}).Limit(100)
		if after != "" {
			query = query.Where(clause.Gt{Column: clause.Column{Name: "UUID"}, Value: after})
		}
		return query.Pluck("UUID", ids).Error
	}
	if err := page("", &ids); err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	if !present["recurring_run_states"] {
		return fmt.Errorf("existing recurring runs have no API-owned scheduling state schema")
	}
	var globalRecords []model.RecurringRunAdoption
	if present["recurring_run_adoptions"] {
		if err := db.Select("ID", "AdoptedCount", "JobIDs", "Ready").Where(map[string]any{"ID": recurringRunAdoptionSourceID}).Find(&globalRecords).Error; err != nil {
			return err
		}
	}
	var globalIDs map[string]bool
	for len(ids) > 0 {
		var stateIDs []string
		if err := db.Model(&model.RecurringRunState{}).Where(map[string]any{"JobUUID": ids}).Pluck("JobUUID", &stateIDs).Error; err != nil {
			return err
		}
		states := make(map[string]bool, len(stateIDs))
		for _, id := range stateIDs {
			states[id] = true
		}
		receiptIDs := make([]string, 0, len(ids))
		for _, id := range ids {
			receiptIDs = append(receiptIDs, recurringRunAdoptionSourceID+":"+id)
		}
		var records []model.RecurringRunAdoption
		// Explicit columns also reject an incomplete receipt schema before migrations.
		if present["recurring_run_adoptions"] {
			if err := db.Select("ID", "AdoptedCount", "JobIDs", "Ready").Where(map[string]any{"ID": receiptIDs}).Find(&records).Error; err != nil {
				return err
			}
		}
		receipts := make(map[string]model.RecurringRunAdoption, len(records))
		for _, receipt := range records {
			receipts[receipt.ID] = receipt
		}
		for _, id := range ids {
			if !states[id] {
				return fmt.Errorf("recurring run %q is missing scheduling state", id)
			}
			receipt, found := receipts[recurringRunAdoptionSourceID+":"+id]
			if found {
				inventory, err := recurringRunReceiptInventory(receipt)
				if err != nil {
					return fmt.Errorf("recurring run %q: %w", id, err)
				}
				if len(inventory) != 1 || !inventory[id] {
					return fmt.Errorf("recurring run %q has an invalid adoption receipt", id)
				}
				continue
			}
			// Native API-owned schedules can predate adoption receipts.
			if len(globalRecords) == 0 {
				continue
			}
			if globalIDs == nil {
				globalIDs, err = recurringRunReceiptInventory(globalRecords[0])
				if err != nil {
					return err
				}
			}
			if !globalIDs[id] {
				continue
			}
		}
		after := ids[len(ids)-1]
		ids = nil
		if err := page(after, &ids); err != nil {
			return err
		}
	}
	return nil
}

func recurringRunReceiptInventory(receipt model.RecurringRunAdoption) (map[string]bool, error) {
	var ids []string
	if err := json.Unmarshal([]byte(receipt.JobIDs), &ids); err != nil {
		return nil, fmt.Errorf("invalid adoption inventory")
	}
	if int64(len(ids)) != receipt.AdoptedCount {
		return nil, fmt.Errorf("incomplete adoption inventory")
	}
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		if id == "" || seen[id] {
			return nil, fmt.Errorf("invalid adoption inventory")
		}
		seen[id] = true
	}
	return seen, nil
}
