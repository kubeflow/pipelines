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
	"os"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
	"github.com/kubeflow/pipelines/backend/src/cache/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// The pre-isolation schema, including its indexes, exercises upgrades with data.
type legacyExecutionCache struct {
	ID                int64  `gorm:"column:ID; not null; primaryKey; AUTO_INCREMENT; index:composite_id_idx"`
	ExecutionCacheKey string `gorm:"column:ExecutionCacheKey; not null; index:idx_cache_key;"`
	ExecutionTemplate string `gorm:"column:ExecutionTemplate; not null;"`
	ExecutionOutput   string `gorm:"column:ExecutionOutput; not null;"`
	MaxCacheStaleness int64  `gorm:"column:MaxCacheStaleness; not null;"`
	StartedAtInSec    int64  `gorm:"column:StartedAtInSec; not null; index:composite_id_idx;"`
	EndedAtInSec      int64  `gorm:"column:EndedAtInSec; not null;"`
}

func (legacyExecutionCache) TableName() string { return "execution_caches" }

func TestCacheNamespaceMigrationAndStorage(t *testing.T) {
	for _, backend := range []string{"sqlite", "mysql", "pgx"} {
		t.Run(backend, func(t *testing.T) {
			var driver gorm.Dialector
			switch backend {
			case "mysql":
				dsn := os.Getenv("KFP_CACHE_TEST_MYSQL_DSN")
				if dsn == "" {
					t.Skip("set KFP_CACHE_TEST_MYSQL_DSN to an empty disposable database")
				}
				driver = mysql.New(mysql.Config{DSN: dsn, DefaultStringSize: 255})
			case "pgx":
				dsn := os.Getenv("KFP_CACHE_TEST_POSTGRES_DSN")
				if dsn == "" {
					t.Skip("set KFP_CACHE_TEST_POSTGRES_DSN to an empty disposable database")
				}
				driver = postgres.Open(dsn)
			default:
				driver = sqlite.Open(":memory:")
			}
			db, err := gorm.Open(driver, &gorm.Config{})
			require.NoError(t, err)
			defer closeDB(t, db)
			// Never overwrite an existing cache database when integration tests are opted in.
			require.False(t, db.Migrator().HasTable(&legacyExecutionCache{}), "use an empty disposable database")
			require.NoError(t, db.AutoMigrate(&legacyExecutionCache{}))
			legacy := legacyExecutionCache{ExecutionCacheKey: "shared-key", ExecutionTemplate: "{}", ExecutionOutput: "legacy-output", MaxCacheStaleness: -1, StartedAtInSec: 1, EndedAtInSec: 1}
			require.NoError(t, db.Create(&legacy).Error)
			require.NoError(t, db.AutoMigrate(&model.ExecutionCache{}))
			require.NoError(t, db.AutoMigrate(&model.ExecutionCache{}), "migration must be idempotent")
			require.True(t, db.Migrator().HasIndex(&model.ExecutionCache{}, "idx_cache_namespace_key"))
			var migrated model.ExecutionCache
			require.NoError(t, db.First(&migrated, legacy.ID).Error)
			require.Empty(t, migrated.Namespace, "do not infer ownership for historical rows")
			require.Equal(t, "legacy-output", migrated.ExecutionOutput)

			s := NewExecutionCacheStore(db, util.NewFakeTimeForEpoch(), dialect.NewDBDialect(backend))
			_, err = s.GetExecutionCache("tenant-a", "shared-key", -1, -1)
			require.ErrorIs(t, err, ErrExecutionCacheNotFound, "legacy rows must not satisfy scoped lookups")
			cachedLegacy, err := s.GetLegacyExecutionCache("shared-key", -1, -1)
			require.NoError(t, err)
			require.Equal(t, &migrated, cachedLegacy)
			_, err = s.GetExecutionCache("", "shared-key", -1, -1)
			require.Error(t, err)
			require.NotErrorIs(t, err, ErrExecutionCacheNotFound, "invalid namespaces must not trigger legacy fallback")
			_, err = s.CreateExecutionCache(&model.ExecutionCache{ExecutionCacheKey: "shared-key"})
			require.Error(t, err, "new rows require ownership")
			for _, ns := range []string{"tenant-a", "tenant-b"} {
				row := createExecutionCache("shared-key", ns+"-output")
				row.Namespace = ns
				_, err := s.CreateExecutionCache(row)
				require.NoError(t, err, "same key may exist in different namespaces and alongside legacy rows")
				duplicate := createExecutionCache("shared-key", "replacement")
				duplicate.Namespace = ns
				_, err = s.CreateExecutionCache(duplicate)
				require.Error(t, err, "duplicate detection remains scoped to its namespace")
			}
			for _, ns := range []string{"tenant-a", "tenant-b"} {
				cached, err := s.GetExecutionCache(ns, "shared-key", -1, -1)
				require.NoError(t, err)
				require.Equal(t, ns, cached.Namespace)
				require.Equal(t, ns+"-output", cached.ExecutionOutput)
			}
			_, err = s.GetExecutionCache("tenant-c", "shared-key", -1, -1)
			require.ErrorIs(t, err, ErrExecutionCacheNotFound)
			cachedLegacy, err = s.GetLegacyExecutionCache("shared-key", -1, -1)
			require.NoError(t, err)
			require.Equal(t, &migrated, cachedLegacy, "legacy lookups must exclude namespaced entries with the same key")
		})
	}
}
