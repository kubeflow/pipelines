// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package storage

import (
	"fmt"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
	"github.com/kubeflow/pipelines/backend/src/cache/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// These SQLite tests isolate the schema/read/write compatibility contract; they
// do not substitute for a rolling Deployment test against MySQL. The old model
// and queries below reproduce 2.17.2, commit
// 2511cdbd74cd531c6633f2e094d235082a32917d, from
// backend/src/cache/{model/execution_cache.go,storage/execution_cache_store.go}.
// In particular, the old writer counts rows without scanning them and never
// updates a duplicate, whereas its reader SELECTs * and scans seven columns.
type executionCache217 struct {
	ID                int64  `gorm:"column:ID; not null; primaryKey; AUTO_INCREMENT; index:composite_id_idx"`
	ExecutionCacheKey string `gorm:"column:ExecutionCacheKey; not null; index:idx_cache_key;"`
	ExecutionTemplate string `gorm:"column:ExecutionTemplate; not null;"`
	ExecutionOutput   string `gorm:"column:ExecutionOutput; not null;"`
	MaxCacheStaleness int64  `gorm:"column:MaxCacheStaleness; not null;"`
	StartedAtInSec    int64  `gorm:"column:StartedAtInSec; not null; index:composite_id_idx;"`
	EndedAtInSec      int64  `gorm:"column:EndedAtInSec; not null;"`
}

func (executionCache217) TableName() string { return "execution_caches" }

func insertCache217(db *gorm.DB, key, output string) error {
	rows, err := db.Table("execution_caches").Where("ExecutionCacheKey = ?", key).Rows()
	if err != nil {
		return err
	}
	count := 0
	for rows.Next() {
		count++
	}
	// Close explicitly so the SQLite in-memory test uses the same connection.
	rows.Close()
	if count != 0 {
		return fmt.Errorf("execution cache with key %s already exists", key)
	}
	return db.Create(&executionCache217{
		ExecutionCacheKey: key, ExecutionTemplate: "template", ExecutionOutput: output,
		MaxCacheStaleness: -1, StartedAtInSec: 1, EndedAtInSec: 1,
	}).Error
}

func TestRollingUpgradeExecutionCacheStorage(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	defer closeDB(t, db)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	// An in-memory SQLite database belongs to one connection.
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&executionCache217{}))
	require.NoError(t, insertCache217(db, "legacy-key", "legacy-output"))
	store := NewExecutionCacheStore(db, util.NewFakeTimeForEpoch(), dialect.NewDBDialect("sqlite"))

	// The retained scanRows is the same seven-field reader used by 2.17.2.
	read217 := func(key string) []*model.ExecutionCache {
		rows, err := db.Table("execution_caches").Where("ExecutionCacheKey = ?", key).Rows()
		require.NoError(t, err)
		defer rows.Close()
		entries, err := store.scanRows(rows, -1)
		require.NoError(t, err)
		return entries
	}
	require.Len(t, read217("legacy-key"), 1, "old reader works before schema migration")

	// Fixed startup adds Namespace with an empty default for populated old rows.
	require.NoError(t, db.AutoMigrate(&model.ExecutionCache{}))
	legacy, err := store.GetLegacyExecutionCache("legacy-key", -1, -1)
	require.NoError(t, err)
	require.Empty(t, legacy.Namespace)
	require.Equal(t, "legacy-output", legacy.ExecutionOutput)
	_, err = store.GetExecutionCache("team-a", "legacy-key", -1, -1)
	require.ErrorIs(t, err, ErrExecutionCacheNotFound)
	require.Empty(t, read217("legacy-key"), "eight columns make the old reader miss, not return a different row")

	// An old watcher can see the new pod's scoped key and write it without a
	// namespace. That must neither satisfy a fixed lookup nor block its insert.
	const scopedKey = "scoped-key-from-new-pod"
	require.NoError(t, insertCache217(db, scopedKey, "old-writer-output"))
	_, err = store.GetExecutionCache("team-a", scopedKey, -1, -1)
	require.ErrorIs(t, err, ErrExecutionCacheNotFound)
	fixed := createExecutionCache(scopedKey, "fixed-output")
	fixed.Namespace = "team-a"
	_, err = store.CreateExecutionCache(fixed)
	require.NoError(t, err)
	got, err := store.GetExecutionCache("team-a", scopedKey, -1, -1)
	require.NoError(t, err)
	require.Equal(t, "fixed-output", got.ExecutionOutput)
	_, err = store.GetExecutionCache("team-b", scopedKey, -1, -1)
	require.ErrorIs(t, err, ErrExecutionCacheNotFound)
	legacy, err = store.GetLegacyExecutionCache(scopedKey, -1, -1)
	require.NoError(t, err)
	require.Equal(t, "old-writer-output", legacy.ExecutionOutput)

	// Reverse the race: a scoped row already exists when the old watcher writes.
	const fixedFirstKey = "fixed-first-key"
	fixedFirst := createExecutionCache(fixedFirstKey, "original-fixed-output")
	fixedFirst.Namespace = "team-a"
	_, err = store.CreateExecutionCache(fixedFirst)
	require.NoError(t, err)
	require.ErrorContains(t, insertCache217(db, fixedFirstKey, "replacement-output"), "already exists")
	got, err = store.GetExecutionCache("team-a", fixedFirstKey, -1, -1)
	require.NoError(t, err)
	require.Equal(t, "original-fixed-output", got.ExecutionOutput)
	_, err = store.GetLegacyExecutionCache(fixedFirstKey, -1, -1)
	require.ErrorIs(t, err, ErrExecutionCacheNotFound)
}
