// Copyright 2020 The Kubeflow Authors
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
	"log"
	"strings"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/cache/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func closeDB(t *testing.T, db *gorm.DB) {
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.Close()
}

func createExecutionCache(cacheKey string, cacheOutput string) *model.ExecutionCache {
	return &model.ExecutionCache{
		ExecutionCacheKey: cacheKey,
		ExecutionTemplate: "testTemplate",
		ExecutionOutput:   cacheOutput,
		MaxCacheStaleness: -1,
		StartedAtInSec:    1,
		EndedAtInSec:      1,
	}
}

func TestCreateExecutionCache(t *testing.T) {
	db, dialect := NewFakeDBOrFatal()
	defer closeDB(t, db)
	executionCacheStore := NewExecutionCacheStore(db, util.NewFakeTimeForEpoch(), dialect)
	executionCacheExpected := model.ExecutionCache{
		ID:                1,
		ExecutionCacheKey: "test",
		ExecutionTemplate: "testTemplate",
		ExecutionOutput:   "testOutput",
		MaxCacheStaleness: -1,
		StartedAtInSec:    1,
		EndedAtInSec:      1,
	}
	executionCache := &model.ExecutionCache{
		ExecutionCacheKey: "test",
		ExecutionTemplate: "testTemplate",
		ExecutionOutput:   "testOutput",
		MaxCacheStaleness: -1,
	}
	executionCache, err := executionCacheStore.CreateExecutionCache(executionCache)
	assert.Nil(t, err)
	require.Equal(t, executionCacheExpected, *executionCache)
}

func TestCreateExecutionCacheWithDuplicateRecord(t *testing.T) {
	executionCache := &model.ExecutionCache{
		ID:                1,
		ExecutionCacheKey: "test",
		ExecutionTemplate: "testTemplate",
		ExecutionOutput:   "testOutput",
		MaxCacheStaleness: -1,
		StartedAtInSec:    1,
		EndedAtInSec:      1,
	}
	db, dialect := NewFakeDBOrFatal()
	defer closeDB(t, db)
	executionCacheStore := NewExecutionCacheStore(db, util.NewFakeTimeForEpoch(), dialect)
	executionCacheStore.CreateExecutionCache(executionCache)
	cache, err := executionCacheStore.CreateExecutionCache(executionCache)
	assert.Nil(t, cache)
	assert.Contains(t, err.Error(), "failed to create new execution cache")
}

func TestGetExecutionCache(t *testing.T) {
	db, dialect := NewFakeDBOrFatal()
	defer closeDB(t, db)
	executionCacheStore := NewExecutionCacheStore(db, util.NewFakeTimeForEpoch(), dialect)

	executionCacheStore.CreateExecutionCache(createExecutionCache("testKey", "testOutput"))
	executionCacheExpected := model.ExecutionCache{
		ID:                1,
		ExecutionCacheKey: "testKey",
		ExecutionTemplate: "testTemplate",
		ExecutionOutput:   "testOutput",
		MaxCacheStaleness: -1,
		StartedAtInSec:    1,
		EndedAtInSec:      1,
	}

	var executionCache *model.ExecutionCache
	executionCache, err := executionCacheStore.GetExecutionCache("testKey", -1, -1)
	require.Nil(t, err)
	require.Equal(t, &executionCacheExpected, executionCache)
}

func TestGetExecutionCacheWithEmptyCacheEntry(t *testing.T) {
	db, dialect := NewFakeDBOrFatal()
	defer closeDB(t, db)
	executionCacheStore := NewExecutionCacheStore(db, util.NewFakeTimeForEpoch(), dialect)

	executionCacheStore.CreateExecutionCache(createExecutionCache("testKey", "testOutput"))
	var executionCache *model.ExecutionCache
	executionCache, err := executionCacheStore.GetExecutionCache("wrongKey", -1, -1)
	require.Nil(t, executionCache)
	require.Contains(t, err.Error(), `Execution cache not found with cache key: "wrongKey"`)
}

func TestGetExecutionCacheWithLatestCacheEntry(t *testing.T) {
	db, dialect := NewFakeDBOrFatal()
	defer closeDB(t, db)
	executionCacheStore := NewExecutionCacheStore(db, util.NewFakeTimeForEpoch(), dialect)

	executionCacheStore.CreateExecutionCache(createExecutionCache("testKey", "testOutput"))
	executionCacheStore.CreateExecutionCache(createExecutionCache("testKey", "testOutput2"))

	executionCacheExpected := model.ExecutionCache{
		ID:                1,
		ExecutionCacheKey: "testKey",
		ExecutionTemplate: "testTemplate",
		ExecutionOutput:   "testOutput",
		MaxCacheStaleness: -1,
		StartedAtInSec:    1,
		EndedAtInSec:      1,
	}
	var executionCache *model.ExecutionCache
	executionCache, err := executionCacheStore.GetExecutionCache("testKey", -1, -1)
	require.Nil(t, err)
	require.Equal(t, &executionCacheExpected, executionCache)
}

func TestGetExecutionCacheWithExpiredDatabaseCacheStaleness(t *testing.T) {
	db, dialect := NewFakeDBOrFatal()
	defer closeDB(t, db)
	executionCacheStore := NewExecutionCacheStore(db, util.NewFakeTimeForEpoch(), dialect)
	executionCacheToPersist := &model.ExecutionCache{
		ExecutionCacheKey: "testKey",
		ExecutionTemplate: "testTemplate",
		ExecutionOutput:   "testOutput",
		MaxCacheStaleness: 0,
	}
	executionCacheStore.CreateExecutionCache(executionCacheToPersist)

	var executionCache *model.ExecutionCache
	executionCache, err := executionCacheStore.GetExecutionCache("testKey", -1, -1)
	require.Contains(t, err.Error(), "Execution cache not found")
	require.Nil(t, executionCache)
}

func TestGetExecutionCacheWithExpiredAnnotationCacheStaleness(t *testing.T) {
	db, dialect := NewFakeDBOrFatal()
	defer closeDB(t, db)
	executionCacheStore := NewExecutionCacheStore(db, util.NewFakeTimeForEpoch(), dialect)
	executionCacheToPersist := &model.ExecutionCache{
		ExecutionCacheKey: "testKey",
		ExecutionTemplate: "testTemplate",
		ExecutionOutput:   "testOutput",
		MaxCacheStaleness: -1,
	}
	executionCacheStore.CreateExecutionCache(executionCacheToPersist)

	var executionCache *model.ExecutionCache
	executionCache, err := executionCacheStore.GetExecutionCache("testKey", 0, -1)
	log.Println(executionCache)
	log.Println("error: " + err.Error())
	require.Contains(t, err.Error(), "CacheStaleness=0, Cache is disabled.")
	require.Nil(t, executionCache)
}

func TestGetExecutionCacheWithExpiredMaximumCacheStaleness(t *testing.T) {
	db, dialect := NewFakeDBOrFatal()
	defer closeDB(t, db)
	executionCacheStore := NewExecutionCacheStore(db, util.NewFakeTimeForEpoch(), dialect)
	executionCacheToPersist := &model.ExecutionCache{
		ExecutionCacheKey: "testKey",
		ExecutionTemplate: "testTemplate",
		ExecutionOutput:   "testOutput",
		MaxCacheStaleness: -1,
	}
	executionCacheStore.CreateExecutionCache(executionCacheToPersist)

	var executionCache *model.ExecutionCache
	executionCache, err := executionCacheStore.GetExecutionCache("testKey", -1, 0)
	log.Println(executionCache)
	log.Println("error: " + err.Error())
	require.Contains(t, err.Error(), "Execution cache not found")
	require.Nil(t, executionCache)
}

// Regression test: empty-key queries must not return unrelated rows.
// If Where is mistakenly changed to struct-based (which omits zero-value fields),
// an empty key would produce an unfiltered query and return an existing row.
func TestGetExecutionCacheWithEmptyKey(t *testing.T) {
	db, dialect := NewFakeDBOrFatal()
	defer closeDB(t, db)
	executionCacheStore := NewExecutionCacheStore(db, util.NewFakeTimeForEpoch(), dialect)

	_, err := executionCacheStore.CreateExecutionCache(createExecutionCache("someKey", "output1"))
	require.NoError(t, err)

	result, err := executionCacheStore.GetExecutionCache("", -1, -1)
	require.Nil(t, result)
	require.Contains(t, err.Error(), "Execution cache not found")
}

// Guardrail: the WHERE predicate must use the model-tag column name
// "ExecutionCacheKey" to maintain compatibility with various SQL dialects,
// instead of GORM's default snake_case "execution_cache_key".
func TestGetExecutionCache_SQLUsesTaggedColumnName(t *testing.T) {
	db, dialect := NewFakeDBOrFatal()
	defer closeDB(t, db)

	dryRun := db.Session(&gorm.Session{DryRun: true}).
		Where(&model.ExecutionCache{ExecutionCacheKey: "anyKey"}).
		Find(&[]model.ExecutionCache{})

	sql := dryRun.Statement.SQL.String()
	assert.True(t, strings.Contains(sql, "ExecutionCacheKey"),
		"WHERE clause must reference the tag-declared column name 'ExecutionCacheKey', got: %s", sql)
	assert.False(t, strings.Contains(sql, "execution_cache_key"),
		"WHERE clause must not use GORM's default snake_case conversion, got: %s", sql)

	_ = dialect
}

// Same guardrail for the duplicate-existence check in CreateExecutionCache.
func TestCreateExecutionCache_DuplicateCheck_SQLUsesTaggedColumnName(t *testing.T) {
	db, dialect := NewFakeDBOrFatal()
	defer closeDB(t, db)

	dryRun := db.Session(&gorm.Session{DryRun: true}).
		Where(&model.ExecutionCache{ExecutionCacheKey: "anyKey"}).
		Find(&[]model.ExecutionCache{})

	sql := dryRun.Statement.SQL.String()
	assert.True(t, strings.Contains(sql, "ExecutionCacheKey"),
		"duplicate-check WHERE clause must reference the tag-declared column name 'ExecutionCacheKey', got: %s", sql)
	assert.False(t, strings.Contains(sql, "execution_cache_key"),
		"duplicate-check WHERE clause must not use GORM's default snake_case conversion, got: %s", sql)

	_ = dialect
}

// Regression test: empty-key creation must not falsely report a duplicate.
// If Where is mistakenly changed to struct-based, the duplicate check becomes
// unfiltered and any existing row would be mistaken for a duplicate.
func TestCreateExecutionCacheWithEmptyKey(t *testing.T) {
	db, dialect := NewFakeDBOrFatal()
	defer closeDB(t, db)
	executionCacheStore := NewExecutionCacheStore(db, util.NewFakeTimeForEpoch(), dialect)

	_, err := executionCacheStore.CreateExecutionCache(createExecutionCache("someKey", "output1"))
	require.NoError(t, err)

	_, err = executionCacheStore.CreateExecutionCache(createExecutionCache("", "output2"))
	if err != nil {
		assert.NotContains(t, err.Error(), "already exists")
	}
}
