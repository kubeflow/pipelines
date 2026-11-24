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
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestBackfillPipelineRefs_MySQLFullRowBinlog(t *testing.T) {
	db, dialect := setupBackfillTestDB(t, "mysql")
	require.NoError(t, db.Create(&model.Experiment{UUID: "exp", Name: "experiment", Namespace: "ns"}).Error)
	insertBackfillCandidates(t, db, 2)
	for _, id := range []string{"job-000000", "job-000001"} {
		require.NoError(t, db.Create(&model.Job{
			UUID: id, DisplayName: id, K8SName: id, Namespace: "ns", ExperimentId: "exp",
		}).Error)
		require.NoError(t, db.Create(&model.ResourceReference{
			ResourceUUID: id, ResourceType: model.JobResourceType, ReferenceUUID: "pipeline",
			ReferenceType: model.PipelineResourceType, Relationship: model.OwnerRelationship,
		}).Error)
	}

	// A server exception on the second primary key reveals the actual commit
	// boundary, including a regression that groups single-row UPDATEs in one
	// transaction. Each row carries a 1 MiB manifest in both FULL row images.
	tables := []string{"jobs", "run_details"}
	for _, table := range tables {
		require.NoError(t, db.Exec("UPDATE "+table+" SET WorkflowSpecManifest = REPEAT('x', 1048576)").Error)
		require.NoError(t, db.Exec("CREATE TRIGGER fail_second_"+table+" BEFORE UPDATE ON "+table+
			" FOR EACH ROW BEGIN IF NEW.UUID LIKE '%000001' THEN SIGNAL SQLSTATE '45000' "+
			"SET MESSAGE_TEXT = 'second wide row'; END IF; END").Error)
	}

	require.NoError(t, db.Connection(func(pinned *gorm.DB) error {
		pinned = pinned.Session(&gorm.Session{})
		var logging bool
		require.NoError(t, pinned.Raw("SELECT @@GLOBAL.log_bin AND @@SESSION.sql_log_bin").Row().Scan(&logging))
		require.True(t, logging, "MySQL integration service must have its binary log enabled")
		require.NoError(t, pinned.Exec("SET SESSION binlog_format = 'ROW', SESSION binlog_row_image = 'FULL'").Error)
		for _, table := range tables {
			err := backfillPipelineRefsToRunTable(pinned, dialect)
			require.ErrorContains(t, err, "second wide row")
			var repaired int
			require.NoError(t, db.Raw("SELECT COUNT(*) FROM "+table+" WHERE PipelineId = 'pipeline'").Scan(&repaired).Error)
			require.Equal(t, 1, repaired, "first wide %s row must already be committed", table)
			applied, err := migrationApplied(db, backfillPipelineRefsMigration)
			require.NoError(t, err)
			require.False(t, applied)
			require.NoError(t, db.Exec("DROP TRIGGER fail_second_"+table).Error)
		}
		return backfillPipelineRefsToRunTable(pinned, dialect)
	}))
	for _, table := range tables {
		var repaired int
		require.NoError(t, db.Raw("SELECT COUNT(*) FROM "+table+" WHERE PipelineId = 'pipeline'").Scan(&repaired).Error)
		require.Equal(t, 2, repaired)
	}
}
