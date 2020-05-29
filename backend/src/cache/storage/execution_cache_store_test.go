// Copyright 2020 Google LLC
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

	"github.com/kubeflow/pipelines/backend/src/cache/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	db := NewFakeDbOrFatal()
	defer db.Close()
	executionCacheStore := NewExecutionCacheStore(db, util.NewFakeTimeForEpoch())
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
	db := NewFakeDbOrFatal()
	defer db.Close()
	executionCacheStore := NewExecutionCacheStore(db, util.NewFakeTimeForEpoch())
	executionCacheStore.CreateExecutionCache(executionCache)
	cache, err := executionCacheStore.CreateExecutionCache(executionCache)
	assert.Nil(t, cache)
	assert.Contains(t, err.Error(), "Failed to create a new execution cache")
}

func TestGetExecutionCache(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	executionCacheStore := NewExecutionCacheStore(db, util.NewFakeTimeForEpoch())

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
	executionCache, err := executionCacheStore.GetExecutionCache("testKey", -1)
	require.Nil(t, err)
	require.Equal(t, &executionCacheExpected, executionCache)
}

func TestGetExecutionCacheWithEmptyCacheEntry(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	executionCacheStore := NewExecutionCacheStore(db, util.NewFakeTimeForEpoch())

	executionCacheStore.CreateExecutionCache(createExecutionCache("testKey", "testOutput"))
	var executionCache *model.ExecutionCache
	executionCache, err := executionCacheStore.GetExecutionCache("wrongKey", -1)
	require.Nil(t, executionCache)
	require.Contains(t, err.Error(), `Execution cache not found with cache key: "wrongKey"`)
}

func TestGetExecutionCacheWithLatestCacheEntry(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	executionCacheStore := NewExecutionCacheStore(db, util.NewFakeTimeForEpoch())

	executionCacheStore.CreateExecutionCache(createExecutionCache("testKey", "testOutput"))
	executionCacheStore.CreateExecutionCache(createExecutionCache("testKey", "testOutput2"))

	executionCacheExpected := model.ExecutionCache{
		ID:                2,
		ExecutionCacheKey: "testKey",
		ExecutionTemplate: "testTemplate",
		ExecutionOutput:   "testOutput2",
		MaxCacheStaleness: -1,
		StartedAtInSec:    2,
		EndedAtInSec:      2,
	}
	var executionCache *model.ExecutionCache
	executionCache, err := executionCacheStore.GetExecutionCache("testKey", -1)
	require.Nil(t, err)
	require.Equal(t, &executionCacheExpected, executionCache)
}

func TestGetExecutionCacheWithExpiredMaxCacheStaleness(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	executionCacheStore := NewExecutionCacheStore(db, util.NewFakeTimeForEpoch())
	executionCacheToPersist := &model.ExecutionCache{
		ExecutionCacheKey: "testKey",
		ExecutionTemplate: "testTemplate",
		ExecutionOutput:   "testOutput",
		MaxCacheStaleness: 0,
	}
	executionCacheStore.CreateExecutionCache(executionCacheToPersist)

	var executionCache *model.ExecutionCache
	executionCache, err := executionCacheStore.GetExecutionCache("testKey", -1)
	require.Contains(t, err.Error(), "Execution cache not found")
	require.Nil(t, executionCache)
}

func TestDeleteExecutionCache(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	executionCacheStore := NewExecutionCacheStore(db, util.NewFakeTimeForEpoch())
	executionCacheStore.CreateExecutionCache(createExecutionCache("testKey", "testOutput"))
	executionCache, err := executionCacheStore.GetExecutionCache("testKey", -1)
	assert.Nil(t, err)
	assert.NotNil(t, executionCache)

	err = executionCacheStore.DeleteExecutionCache("1")
	assert.Nil(t, err)
	_, err = executionCacheStore.GetExecutionCache("testKey", -1)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "not found")
}
