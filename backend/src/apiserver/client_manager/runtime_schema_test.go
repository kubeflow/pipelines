// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package clientmanager

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type runtimeSchemaTrace struct {
	logger.Interface
	mutex      sync.Mutex
	statements []string
}

func (trace *runtimeSchemaTrace) Trace(_ context.Context, _ time.Time, sql func() (string, int64), _ error) {
	statement, _ := sql()
	trace.mutex.Lock()
	defer trace.mutex.Unlock()
	trace.statements = append(trace.statements, statement)
}

func (trace *runtimeSchemaTrace) reset() {
	trace.mutex.Lock()
	defer trace.mutex.Unlock()
	trace.statements = nil
}

func (trace *runtimeSchemaTrace) queries() []string {
	trace.mutex.Lock()
	defer trace.mutex.Unlock()
	return append([]string{}, trace.statements...)
}

func runtimeSchemaSQLite(t *testing.T) (*gorm.DB, *runtimeSchemaTrace) {
	t.Helper()
	trace := &runtimeSchemaTrace{Interface: logger.Default.LogMode(logger.Silent)}
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: trace})
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	pool.SetMaxOpenConns(1)
	t.Cleanup(func() { require.NoError(t, pool.Close()) })
	return db, trace
}

func runtimePriorSchema(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, autoMigrate(db))
	require.NoError(t, db.Migrator().DropTable(&model.RunLease{}))
	require.NoError(t, db.Migrator().DropIndex(&model.Run{}, runLeaseDiscoveryIndex))
	require.NoError(t, db.Migrator().DropTable(&model.RuntimeTask{}))
	for _, column := range runtimeRunColumns {
		require.NoError(t, db.Exec("ALTER TABLE "+runtimeSchemaQuote(db, "run_details")+" DROP COLUMN "+runtimeSchemaQuote(db, column.name)).Error)
	}
}

func TestRuntimeSchemaActualEntrypointUpgradeAndRepeatedStartup(t *testing.T) {
	db, trace := runtimeSchemaSQLite(t)
	runtimePriorSchema(t, db)
	longManifest := strings.Repeat("pipeline-data-", 1024)
	require.NoError(t, db.Exec(`INSERT INTO run_details
		(UUID, ExperimentUUID, DisplayName, Name, Namespace, ServiceAccount, Description, Conditions, StorageState,
		PipelineId, PipelineName, PipelineSpecManifest, WorkflowSpecManifest, Parameters,
		CreatedAtInSec, PipelineRuntimeManifest, WorkflowRuntimeManifest)
		VALUES ('run', '', '', '', '', '', '', 'RUNNING', 'AVAILABLE', '', '', '', '', '', 1, ?, ?)`, longManifest, longManifest).Error)
	trace.reset()
	require.NoError(t, autoMigrate(db))
	var manifest string
	require.NoError(t, db.Raw(`SELECT PipelineRuntimeManifest FROM run_details WHERE UUID = 'run'`).Scan(&manifest).Error)
	assert.Equal(t, longManifest, manifest)
	var run model.Run
	require.NoError(t, db.First(&run, map[string]any{"UUID": "run"}).Error)
	assert.Nil(t, run.ExecutorType)
	assert.Nil(t, run.WorkspaceProvisioner)
	assert.Nil(t, run.WorkspaceReference)
	additions := 0
	for _, query := range trace.queries() {
		if strings.HasPrefix(query, "ALTER TABLE `run_details`") {
			additions++
			assert.Contains(t, query, "ADD COLUMN")
			assert.Contains(t, query, " NULL")
			assert.NotContains(t, query, "DEFAULT")
		}
		assert.NotContains(t, query, "run_details__temp")
	}
	assert.Equal(t, 3, additions)
	require.True(t, db.Migrator().HasTable(&model.RuntimeTask{}))
	trace.reset()
	require.NoError(t, autoMigrate(db))
	for _, query := range trace.queries() {
		assert.False(t, strings.HasPrefix(query, "ALTER TABLE"), query)
		assert.False(t, strings.HasPrefix(query, "CREATE TABLE"), query)
	}
}

func TestRuntimeSchemaRefusesImplicitRepair(t *testing.T) {
	t.Run("incompatible nullable run column", func(t *testing.T) {
		db, trace := runtimeSchemaSQLite(t)
		runtimePriorSchema(t, db)
		require.NoError(t, db.Exec("ALTER TABLE run_details ADD COLUMN ExecutorType TEXT NULL").Error)
		trace.reset()
		require.ErrorContains(t, autoMigrate(db), "refusing implicit type conversion")
		for _, query := range trace.queries() {
			assert.False(t, strings.HasPrefix(query, "ALTER TABLE"), query)
		}
	})
	t.Run("missing parent candidate key", func(t *testing.T) {
		db, _ := runtimeSchemaSQLite(t)
		require.NoError(t, autoMigrate(db))
		require.NoError(t, db.Migrator().DropIndex(&model.Task{}, "idx_tasks_uuid_run"))
		require.ErrorContains(t, autoMigrate(db), "separately verified online index migration")
	})
	t.Run("missing runtime index", func(t *testing.T) {
		db, _ := runtimeSchemaSQLite(t)
		require.NoError(t, autoMigrate(db))
		require.NoError(t, db.Migrator().DropIndex(&model.RuntimeTask{}, "idx_runtime_tasks_attempt"))
		require.ErrorContains(t, autoMigrate(db), "refusing implicit schema repair")
		assert.False(t, db.Migrator().HasIndex(&model.RuntimeTask{}, "idx_runtime_tasks_attempt"))
	})
}
