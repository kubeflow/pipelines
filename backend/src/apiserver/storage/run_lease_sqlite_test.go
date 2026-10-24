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

package storage

import (
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRunLeaseSQLiteIndependentPools(t *testing.T) {
	path := fmt.Sprintf(".run-lease-pools-%d.sqlite", time.Now().UnixNano())
	t.Cleanup(func() {
		for _, suffix := range []string{"", "-wal", "-shm"} {
			err := os.Remove(path + suffix)
			if !os.IsNotExist(err) {
				require.NoError(t, err)
			}
		}
	})
	dsn := path + "?_journal_mode=WAL&_foreign_keys=on&_busy_timeout=5000&_txlock=immediate&cache=private"
	orm, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, orm.AutoMigrate(model.AllModels()...))
	firstDB, err := orm.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, firstDB.Close()) })
	secondDB, err := sql.Open("sqlite3", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, secondDB.Close()) })
	firstDB.SetMaxOpenConns(4)
	secondDB.SetMaxOpenConns(4)
	d := dialect.NewDBDialect("sqlite")
	executor := "kubernetes"
	_, err = NewRunStore(firstDB, util.NewRealTime(), d).CreateRun(&model.Run{
		UUID: "shared-host-run", RunDetails: model.RunDetails{State: model.RuntimeStateRunning, ExecutorType: &executor},
	})
	require.NoError(t, err)
	first := NewTaskStore(firstDB, util.NewRealTime(), util.NewUUIDGenerator(), d)
	second := NewTaskStore(secondDB, util.NewRealTime(), util.NewUUIDGenerator(), d)
	providers := []RunLeaseStoreInterface{first.RunLeaseStore(), second.RunLeaseStore()}
	tokens := make([]*RunLeaseToken, 12)
	errs := recurringConcurrent(len(tokens), func(i int) error {
		var err error
		tokens[i], err = providers[i%2].Acquire("shared-host-run", fmt.Sprintf("worker-%d", i), time.Minute, false)
		return err
	})
	winner := -1
	for i, err := range errs {
		if err == nil {
			assert.Equal(t, -1, winner)
			winner = i
		} else {
			require.ErrorIs(t, err, ErrRunLeaseConflict)
		}
	}
	require.NotEqual(t, -1, winner)
	old, err := first.LeaseScopedStores(*tokens[winner])
	require.NoError(t, err)
	require.NoError(t, providers[winner%2].Release(*tokens[winner]))
	takeover, err := second.RunLeaseStore().Acquire("shared-host-run", "new-worker", time.Minute, false)
	require.NoError(t, err)
	require.Greater(t, takeover.Generation, tokens[winner].Generation)
	_, err = old.Tasks.CreateTask(&model.Task{RunUUID: "shared-host-run", Name: "stale"})
	require.ErrorIs(t, err, ErrRunLeaseConflict)
	var count int
	require.NoError(t, secondDB.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count))
	assert.Zero(t, count)
}
