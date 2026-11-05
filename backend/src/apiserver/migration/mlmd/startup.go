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
	"fmt"
)

// CutoverHoldMessage explains why production binaries refuse the legacy upgrade.
const CutoverHoldMessage = "production MLMD migration is disabled pending nonblocking database cutover qualification; keep the installation on KFP 2.18"

// StartupState describes work that must finish before native schema reconciliation.
type StartupState int

const (
	Fresh StartupState = iota
	Native
	MigrationRequired
	CleanupPending
)

type schemaReader interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// ClassifyStartup checks the catalog without changing schema or data.
func ClassifyStartup(ctx context.Context, db schemaReader, dialect string) (StartupState, error) {
	catalog := "DATABASE()"
	parameter := "?"
	if dialect == "pgx" || dialect == "postgres" {
		catalog = "current_schema()"
		parameter = "$1"
	}
	tableExists := func(table string) (bool, error) {
		var count int
		query := "SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=" + catalog + " AND TABLE_NAME=" + parameter
		err := db.QueryRowContext(ctx, query, table).Scan(&count)
		return count != 0, err
	}
	tables := make(map[string]bool)
	for _, table := range []string{"tasks", "run_details", "tasks_pre_migration", "tasks_migration_tmp", "artifacts", "artifact_tasks", "mlmd_migration_state", "mlmd_migration_complete"} {
		present, err := tableExists(table)
		if err != nil {
			return Fresh, fmt.Errorf("classify MLMD migration schema: %w", err)
		}
		tables[table] = present
	}
	backup, staging := tables["tasks_pre_migration"], tables["tasks_migration_tmp"]
	hasProgress := backup || staging || tables["mlmd_migration_state"] || tables["mlmd_migration_complete"]
	if !tables["tasks"] {
		if hasProgress || tables["run_details"] || tables["artifacts"] || tables["artifact_tasks"] {
			return Fresh, fmt.Errorf("tasks table is missing from an existing KFP schema; inspect or restore it before startup")
		}
		return Fresh, nil
	}
	var legacy, native int
	query := "SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=" + catalog + " AND TABLE_NAME='tasks' AND LOWER(COLUMN_NAME)=" + parameter
	if err := db.QueryRowContext(ctx, query, "mlmdexecutionid").Scan(&legacy); err != nil {
		return Fresh, fmt.Errorf("inspect legacy tasks column: %w", err)
	}
	if err := db.QueryRowContext(ctx, query, "typeattrs").Scan(&native); err != nil {
		return Fresh, fmt.Errorf("inspect native tasks column: %w", err)
	}
	if dialect != "mysql" && (legacy != 0 || hasProgress) {
		return Fresh, fmt.Errorf("legacy MLMD migration requires MySQL; PostgreSQL upgrades are unsupported; retain the old installation and database backup")
	}
	if (legacy != 0) == (native != 0) || backup && (legacy != 0 || staging) {
		return Fresh, fmt.Errorf("tasks schema or migration staging state cannot be classified safely; inspect or restore the KFP database")
	}
	bothArtifacts := tables["artifacts"] && tables["artifact_tasks"]
	var validated int
	if tables["mlmd_migration_complete"] {
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM mlmd_migration_complete WHERE ID=1").Scan(&validated); err != nil {
			return Fresh, fmt.Errorf("read MLMD migration completion: %w", err)
		}
		if validated != 0 && (legacy != 0 || staging || !bothArtifacts) {
			return Fresh, fmt.Errorf("MLMD validation marker conflicts with the current schema; inspect or restore the database")
		}
	}
	switch {
	case legacy != 0 || backup && validated == 0:
		return MigrationRequired, nil
	case backup:
		return CleanupPending, nil
	case hasProgress && validated == 0:
		return Fresh, fmt.Errorf("MLMD migration state exists without a validation record or legacy task backup; inspect or restore the database")
	case staging || !bothArtifacts:
		return Fresh, fmt.Errorf("native tasks, artifacts, and artifact_tasks are not all present in a consistent schema")
	default:
		return Native, nil
	}
}

// NeedsMigration reports whether the MySQL migration or its cleanup is pending.
func NeedsMigration(ctx context.Context, db schemaReader) (bool, error) {
	state, err := ClassifyStartup(ctx, db, "mysql")
	return state == MigrationRequired || state == CleanupPending, err
}
