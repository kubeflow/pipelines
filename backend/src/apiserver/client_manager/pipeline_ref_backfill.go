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

package clientmanager

import (
	"database/sql"
	"fmt"
	"slices"
	"time"

	"github.com/golang/glog"
	sqldrv "github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"gorm.io/gorm"
)

const pipelineRefBatchSize = 500

var pipelineRefBackfillUpdates = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "pipeline_ref_backfill_updates_total",
	Help: "Committed pipeline reference column updates by the legacy backfill.",
}, []string{"table"})

// backfillPipelineRefsToRunTable repairs legacy rows under the worker's advisory
// lock. A restart repeats the consistent reads and skips committed repairs.
func backfillPipelineRefsToRunTable(db *gorm.DB, dialect sqldrv.DBDialect) error {
	applied, err := migrationApplied(db, backfillPipelineRefsMigration)
	if err != nil || applied {
		return err
	}

	writeBatchSize := pipelineRefWriteBatchSize(db, dialect)
	// Repair the source jobs first so new recurring executions inherit their
	// pipeline IDs. Reports that already read an old job can still commit
	// after this historical sweep; RunStore resolves those rows on read.
	for _, table := range []string{"jobs", "run_details"} {
		// Direct pipeline references take precedence over the version's parent.
		for _, pass := range []pipelineRefPass{
			backfillRefColumnSQL(dialect, table, "PipelineId", model.PipelineResourceType),
			backfillRefColumnSQL(dialect, table, "PipelineVersionId", model.PipelineVersionResourceType),
			backfillPipelineIDFromVersionSQL(dialect, table),
		} {
			if err := pass.run(db, dialect, table, writeBatchSize); err != nil {
				return fmt.Errorf("backfill %s.%s from %s: %w", table, pass.column, pass.source, err)
			}
		}
	}
	_, err = claimMigration(db, backfillPipelineRefsMigration)
	return err
}

// MySQL FULL row events include the original and updated LOBs even when
// only an ID changes. Keep these transactions to one target row on both
// tables while still discovering candidates a page at a time.
func pipelineRefWriteBatchSize(db *gorm.DB, dialect sqldrv.DBDialect) int {
	if dialect.Name() != "mysql" {
		return pipelineRefBatchSize
	}
	var fullImages bool
	if err := db.Raw("SELECT @@GLOBAL.log_bin AND @@SESSION.sql_log_bin AND @@SESSION.binlog_row_image = 'FULL'").Row().Scan(&fullImages); err != nil {
		glog.Warningf("Could not determine MySQL binlog row image, using single-row pipeline reference updates: %v", err)
		return 1
	}
	if fullImages {
		return 1
	}
	return pipelineRefBatchSize
}

type pipelineRefPass struct {
	column    string
	source    string
	selectIDs string
	updateSQL string
}

func (pass pipelineRefPass) run(db *gorm.DB, dialect sqldrv.DBDialect, table string, writeBatchSize int) error {
	start := time.Now()
	cursor := ""
	var batches, updated int64
	for {
		var ids []string
		selectStart := time.Now()
		if err := db.Raw(pass.selectIDs, cursor, pipelineRefBatchSize).Scan(&ids).Error; err != nil {
			return err
		}
		if len(ids) == 0 {
			break
		}
		var pageUpdates int64
		for batch := range slices.Chunk(ids, writeBatchSize) {
			affected, err := pass.update(db, dialect, batch)
			if err != nil {
				return err
			}
			batches++
			pageUpdates += affected
			pipelineRefBackfillUpdates.WithLabelValues(table).Add(float64(affected))
		}
		updated += pageUpdates
		cursor = ids[len(ids)-1]
		glog.V(1).Infof("Pipeline reference backfill table=%s column=%s batches=%d page_rows_updated=%d elapsed=%s", table, pass.column, batches, pageUpdates, time.Since(selectStart))
	}
	glog.Infof("Pipeline reference backfill table=%s column=%s source=%s batches=%d rows_updated=%d elapsed=%s", table, pass.column, pass.source, batches, updated, time.Since(start))
	return nil
}

func (pass pipelineRefPass) update(db *gorm.DB, dialect sqldrv.DBDialect, ids []string) (int64, error) {
	var affected int64
	err := db.Transaction(func(tx *gorm.DB) error {
		if dialect.Name() == "pgx" {
			if err := tx.Exec("SET LOCAL lock_timeout = '2s'").Error; err != nil {
				return err
			}
		}
		result := tx.Exec(pass.updateSQL, ids)
		affected = result.RowsAffected
		return result.Error
	}, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	return affected, err
}

func backfillRefColumnSQL(dialect sqldrv.DBDialect, table, column string, refType model.ResourceType) pipelineRefPass {
	q := dialect.QuoteIdentifier
	resourceType := model.RunResourceType
	if table == "jobs" {
		resourceType = model.JobResourceType
	}
	match := fmt.Sprintf("src.%s = %s.%s AND src.%s = '%s' AND src.%s = '%s'",
		q("ResourceUUID"), table, q("UUID"), q("ResourceType"), resourceType, q("ReferenceType"), refType)
	return newPipelineRefPass(dialect, table, column, "resource_references", "ReferenceUUID", match)
}

func backfillPipelineIDFromVersionSQL(dialect sqldrv.DBDialect, table string) pipelineRefPass {
	q := dialect.QuoteIdentifier
	match := fmt.Sprintf("src.%s = %s.%s AND %s.%s <> ''", q("UUID"), table, q("PipelineVersionId"), table, q("PipelineVersionId"))
	return newPipelineRefPass(dialect, table, "PipelineId", "pipeline_versions", "PipelineId", match)
}

func newPipelineRefPass(dialect sqldrv.DBDialect, table, column, source, sourceColumn, match string) pipelineRefPass {
	q := dialect.QuoteIdentifier
	id := table + "." + q("UUID")
	qualifiedColumn := table + "." + q(column)
	empty := "(" + qualifiedColumn + " IS NULL OR " + qualifiedColumn + " = '')"
	value := "src." + q(sourceColumn)
	nonempty := value + " <> ''"
	lookup := fmt.Sprintf("SELECT %s FROM %s src WHERE %s AND %s", value, source, match, nonempty)
	updateTable := table
	if dialect.Name() == "mysql" {
		// Constrain the locking UPDATE to PK lookups even if the optimizer
		// prefers a scan for a small table or a dense batch.
		updateTable += " FORCE INDEX (PRIMARY)"
	}
	return pipelineRefPass{
		column: column,
		source: source,
		selectIDs: fmt.Sprintf("SELECT %s FROM %s src JOIN %s ON %s WHERE %s > ? AND %s AND %s ORDER BY %s LIMIT ?",
			id, source, table, match, id, empty, nonempty, id),
		updateSQL: fmt.Sprintf("UPDATE %s SET %s = (%s) WHERE %s IN ? AND %s AND EXISTS (%s)",
			updateTable, q(column), lookup, id, empty, lookup),
	}
}
