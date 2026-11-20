// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mlmd

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
)

// DiskBudget records operator measurements for the two database hosts. MySQL
// provides table sizes, but a connection cannot measure free filesystem space.
type DiskBudget struct {
	AvailableBytes    uint64 `json:"available_bytes"`
	MLMDDatabaseBytes uint64 `json:"mlmd_database_bytes"`
}

type CapacityReport struct {
	DiskBudget
	LegacyTasksBytes uint64 `json:"legacy_tasks_bytes"`
	RequiredBytes    uint64 `json:"required_bytes"`
}

// CheckDiskCapacity checks the disk budget without changing either database.
func CheckDiskCapacity(ctx context.Context, database *sql.DB, budget DiskBudget) (CapacityReport, error) {
	return checkDiskCapacity(ctx, database, budget)
}

func checkDiskCapacity(ctx context.Context, database interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, budget DiskBudget) (CapacityReport, error) {
	report := CapacityReport{DiskBudget: budget}
	// The legacy rows live in tasks until the atomic rename, then in
	// tasks_pre_migration for the remainder of an interrupted upgrade.
	query := `SELECT COALESCE(t.DATA_LENGTH, 0) + COALESCE(t.INDEX_LENGTH, 0),
		t.TABLE_NAME = 'tasks_pre_migration' OR EXISTS (
			SELECT 1 FROM information_schema.COLUMNS c
			WHERE c.TABLE_SCHEMA = t.TABLE_SCHEMA AND c.TABLE_NAME = t.TABLE_NAME
			AND c.COLUMN_NAME = 'MLMDExecutionID')
		FROM information_schema.TABLES t
		WHERE t.TABLE_SCHEMA = DATABASE() AND t.TABLE_NAME IN ('tasks', 'tasks_pre_migration')
		ORDER BY t.TABLE_NAME = 'tasks_pre_migration' DESC LIMIT 1`
	var legacy bool
	err := database.QueryRowContext(ctx, query).Scan(&report.LegacyTasksBytes, &legacy)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return report, fmt.Errorf("cannot check migration disk capacity: KFP tasks table is missing")
	case err != nil:
		return report, fmt.Errorf("read MySQL table size for the disk capacity check: %w", err)
	case !legacy:
		report.LegacyTasksBytes = 0
		return report, nil
	case budget.AvailableBytes == 0 || budget.MLMDDatabaseBytes == 0:
		return report, fmt.Errorf("measure the free bytes on the KFP MySQL data volume and the size of the MLMD database before migrating")
	case budget.MLMDDatabaseBytes > (math.MaxUint64-report.LegacyTasksBytes)/2:
		return report, fmt.Errorf("migration disk estimate exceeds the supported byte range")
	}

	// Reserve the estimated MLMD import footprint plus space for the
	// tasks copy and its indexes. Use the same conservative estimate on retry.
	report.RequiredBytes = 2*budget.MLMDDatabaseBytes + report.LegacyTasksBytes
	if budget.AvailableBytes < report.RequiredBytes {
		return report, fmt.Errorf("insufficient MySQL disk capacity: %d bytes available, at least %d bytes required", budget.AvailableBytes, report.RequiredBytes)
	}
	return report, nil
}
