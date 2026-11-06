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

// NeedsMigration classifies the database before GORM can reconcile its schema.
// A native installation without MLMD migration state is a fresh installation.
func NeedsMigration(ctx context.Context, db *sql.DB) (bool, error) {
	var tables, legacyColumns, nativeColumns, temporaryTables, nativeTables, stateTables, completeRows int
	queries := []struct {
		query string
		out   *int
	}{
		{"SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='tasks'", &tables},
		{"SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='tasks' AND COLUMN_NAME='MLMDExecutionID'", &legacyColumns},
		{"SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='tasks' AND COLUMN_NAME='TypeAttrs'", &nativeColumns},
		{"SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME IN ('tasks_pre_migration', 'tasks_migration_tmp')", &temporaryTables},
		{"SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME IN ('artifacts', 'artifact_tasks')", &nativeTables},
		{"SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='mlmd_migration_complete'", &stateTables},
	}
	for _, item := range queries {
		if err := db.QueryRowContext(ctx, item.query).Scan(item.out); err != nil {
			return false, fmt.Errorf("classify MLMD migration schema: %w", err)
		}
	}
	if tables == 0 {
		return false, nil
	}
	if legacyColumns > 0 || temporaryTables > 0 {
		return true, nil
	}
	if nativeColumns == 0 {
		return false, fmt.Errorf("tasks has neither the qualified legacy nor native schema; inspect the database before startup")
	}
	if stateTables > 0 {
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM mlmd_migration_complete WHERE ID=1").Scan(&completeRows); err != nil {
			return false, fmt.Errorf("read MLMD migration completion: %w", err)
		}
		return completeRows == 0 || nativeTables != 2, nil
	}
	var cutoffTables int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='mlmd_migration_state'").Scan(&cutoffTables); err != nil {
		return false, err
	}
	return cutoffTables > 0, nil
}
