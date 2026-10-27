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
	"strings"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunLeaseSchemaActualEntrypointAndRepeat(t *testing.T) {
	db, trace := runtimeSchemaSQLite(t)
	require.NoError(t, autoMigrate(db))
	require.NoError(t, db.Migrator().DropTable(&model.RunLease{}))
	require.NoError(t, db.Migrator().DropIndex(&model.Run{}, runLeaseDiscoveryIndex))
	trace.reset()
	require.NoError(t, autoMigrate(db))
	require.True(t, db.Migrator().HasTable(&model.RunLease{}))
	require.NoError(t, validateRunLeaseSchema(db))
	require.NoError(t, requireRuntimeIndex(db, &model.Run{}, runLeaseDiscoveryIndex, "ExecutorType,State,CreatedAtInSec,UUID", false))
	for _, query := range trace.queries() {
		assert.False(t, strings.HasPrefix(query, "ALTER TABLE"), "additive lease upgrade must not alter populated tables: %s", query)
	}
	trace.reset()
	require.NoError(t, autoMigrate(db))
	for _, query := range trace.queries() {
		assert.False(t, strings.HasPrefix(query, "CREATE TABLE"), query)
		assert.False(t, strings.HasPrefix(query, "CREATE INDEX"), query)
		assert.False(t, strings.HasPrefix(query, "ALTER TABLE"), query)
	}
}

func TestRunLeaseSchemaRefusesImplicitRepair(t *testing.T) {
	t.Run("missing lease index", func(t *testing.T) {
		db, _ := runtimeSchemaSQLite(t)
		require.NoError(t, autoMigrate(db))
		require.NoError(t, db.Migrator().DropIndex(&model.RunLease{}, "idx_run_leases_expiry_run"))
		require.ErrorContains(t, autoMigrate(db), "refusing implicit schema repair")
		assert.False(t, db.Migrator().HasIndex(&model.RunLease{}, "idx_run_leases_expiry_run"))
	})
	t.Run("incompatible discovery index", func(t *testing.T) {
		db, _ := runtimeSchemaSQLite(t)
		require.NoError(t, autoMigrate(db))
		require.NoError(t, db.Migrator().DropIndex(&model.Run{}, runLeaseDiscoveryIndex))
		require.NoError(t, db.Exec("CREATE INDEX "+runLeaseDiscoveryIndex+" ON run_details (UUID)").Error)
		require.ErrorContains(t, autoMigrate(db), "refusing implicit schema repair")
		assert.True(t, db.Migrator().HasIndex(&model.Run{}, runLeaseDiscoveryIndex), "do not drop another process's index")
	})
	t.Run("foreign key must cascade", func(t *testing.T) {
		db, _ := runtimeSchemaSQLite(t)
		require.NoError(t, autoMigrate(db))
		var createSQL string
		require.NoError(t, db.Raw("SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'run_leases'").Scan(&createSQL).Error)
		require.NoError(t, db.Migrator().DropTable(&model.RunLease{}))
		require.NoError(t, db.Exec(strings.ReplaceAll(createSQL, "ON DELETE CASCADE", "ON DELETE RESTRICT")).Error)
		require.NoError(t, db.Exec("CREATE INDEX idx_run_leases_expiry_run ON run_leases (ExpiresAt, RunUUID)").Error)
		require.ErrorContains(t, autoMigrate(db), "must cascade")
	})
}
