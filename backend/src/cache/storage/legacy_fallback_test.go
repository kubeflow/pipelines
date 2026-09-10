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
	"testing"
	"time"

	"github.com/kubeflow/pipelines/backend/src/cache/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/require"
)

func TestGetLegacyExecutionCacheSelectsLatestLegacyEntry(t *testing.T) {
	db, dialect := NewFakeDBOrFatal()
	defer closeDB(t, db)
	store := NewExecutionCacheStore(db, util.NewFakeTime(time.Unix(100, 0)), dialect)
	for i, namespace := range []string{"", "", "tenant-a", "tenant-b"} {
		row := createExecutionCache("shared-key", "output-"+namespace)
		row.Namespace = namespace
		row.StartedAtInSec = int64(i + 1)
		require.NoError(t, db.Create(row).Error)
	}

	cached, err := store.GetLegacyExecutionCache("shared-key", -1, -1)
	require.NoError(t, err)
	require.Empty(t, cached.Namespace)
	require.Equal(t, int64(2), cached.StartedAtInSec, "newer namespaced entries must not be selected")
	var rowCount int64
	require.NoError(t, db.Model(&model.ExecutionCache{}).Count(&rowCount).Error)
	require.Equal(t, int64(4), rowCount, "legacy reads must not copy or rewrite cache entries")
}

func TestExecutionCacheLookupStalenessAndMissClassification(t *testing.T) {
	for _, namespace := range []string{"default", ""} {
		lookupName := "namespaced"
		if namespace == "" {
			lookupName = "legacy"
		}
		t.Run(lookupName, func(t *testing.T) {
			for _, tc := range []struct {
				name                  string
				key                   string
				rowStaleness          int64
				cacheStaleness        int64
				maximumCacheStaleness int64
				wantMiss              bool
				wantDisabled          bool
			}{
				{name: "valid", key: "key", rowStaleness: -1, cacheStaleness: -1, maximumCacheStaleness: -1},
				{name: "missing key", key: "missing", rowStaleness: -1, cacheStaleness: -1, maximumCacheStaleness: -1, wantMiss: true},
				{name: "empty key", key: "", rowStaleness: -1, cacheStaleness: -1, maximumCacheStaleness: -1, wantMiss: true},
				{name: "expired row", key: "key", rowStaleness: 5, cacheStaleness: -1, maximumCacheStaleness: -1, wantMiss: true},
				{name: "disabled row", key: "key", rowStaleness: 0, cacheStaleness: -1, maximumCacheStaleness: -1, wantMiss: true},
				{name: "expired request", key: "key", rowStaleness: -1, cacheStaleness: 5, maximumCacheStaleness: -1, wantMiss: true},
				{name: "expired maximum", key: "key", rowStaleness: -1, cacheStaleness: -1, maximumCacheStaleness: 5, wantMiss: true},
				{name: "disabled request", key: "key", rowStaleness: -1, cacheStaleness: 0, maximumCacheStaleness: -1, wantDisabled: true},
			} {
				t.Run(tc.name, func(t *testing.T) {
					db, dialect := NewFakeDBOrFatal()
					defer closeDB(t, db)
					store := NewExecutionCacheStore(db, util.NewFakeTime(time.Unix(100, 0)), dialect)
					row := createExecutionCache("key", "output")
					row.Namespace = namespace
					row.MaxCacheStaleness = tc.rowStaleness
					require.NoError(t, db.Create(row).Error)
					lookup := store.GetLegacyExecutionCache
					if namespace != "" {
						lookup = func(key string, staleness, maximumStaleness int64) (*model.ExecutionCache, error) {
							return store.GetExecutionCache(namespace, key, staleness, maximumStaleness)
						}
					}
					cached, err := lookup(tc.key, tc.cacheStaleness, tc.maximumCacheStaleness)
					switch {
					case tc.wantMiss:
						require.Nil(t, cached)
						require.ErrorIs(t, err, ErrExecutionCacheNotFound)
					case tc.wantDisabled:
						require.Nil(t, cached)
						require.ErrorContains(t, err, "Cache is disabled")
						require.NotErrorIs(t, err, ErrExecutionCacheNotFound)
					default:
						require.NoError(t, err)
						require.Equal(t, row, cached)
					}
				})
			}
		})
	}
}

func TestExecutionCacheDatabaseErrorsAreNotMisses(t *testing.T) {
	for _, tc := range []struct {
		name                  string
		maximumCacheStaleness int64
		message               string
	}{
		{name: "query failure", maximumCacheStaleness: -1, message: "failed to get execution cache"},
		{name: "cleanup failure", maximumCacheStaleness: 5, message: "Failed to cleanup old cache entries"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, dialect := NewFakeDBOrFatal()
			defer closeDB(t, db)
			store := NewExecutionCacheStore(db, util.NewFakeTimeForEpoch(), dialect)
			require.NoError(t, db.Migrator().DropTable(&model.ExecutionCache{}))
			cached, err := store.GetExecutionCache("default", "key", -1, tc.maximumCacheStaleness)
			require.Nil(t, cached)
			require.ErrorContains(t, err, tc.message)
			require.NotErrorIs(t, err, ErrExecutionCacheNotFound)
			cached, err = store.GetLegacyExecutionCache("key", -1, tc.maximumCacheStaleness)
			require.Nil(t, cached)
			require.ErrorContains(t, err, tc.message)
			require.NotErrorIs(t, err, ErrExecutionCacheNotFound)
		})
	}
}
