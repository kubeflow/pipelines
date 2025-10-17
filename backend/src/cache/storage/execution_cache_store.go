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
	"database/sql"
	"fmt"
	"log"
	"strconv"

	"github.com/kubeflow/pipelines/backend/src/cache/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

type ExecutionCacheStoreInterface interface {
	GetExecutionCache(executionCacheKey string, cacheStaleness int64, maximumCacheStaleness int64) (*model.ExecutionCache, error)
	CreateExecutionCache(*model.ExecutionCache) (*model.ExecutionCache, error)
}

type ExecutionCacheStore struct {
	db   *DB
	time util.TimeInterface
}

func (s *ExecutionCacheStore) GetExecutionCache(executionCacheKey string, cacheStaleness int64, maximumCacheStaleness int64) (*model.ExecutionCache, error) {
	rowsAffected, err := s.cleanDatabase(maximumCacheStaleness)
	log.Printf("Number of deleted rows: %d", rowsAffected)
	if err != nil {
		return nil, fmt.Errorf("Failed to cleanup old cache entries: %s", err)
	}
	if cacheStaleness == 0 {
		return nil, fmt.Errorf("CacheStaleness=0, Cache is disabled.")
	}
	r, err := s.db.Table("execution_caches").Where("ExecutionCacheKey = ?", executionCacheKey).Rows()
	if err != nil {
		return nil, fmt.Errorf("Failed to get execution cache: %q, err: %v", executionCacheKey, err)
	}
	defer r.Close()
	executionCaches, err := s.scanRows(r, cacheStaleness)
	if err != nil {
		return nil, fmt.Errorf("Failed to scan rows on execution cache: %q, err: %v", executionCacheKey, err)
	}
	if len(executionCaches) == 0 {
		return nil, fmt.Errorf("Execution cache not found with cache key: %q", executionCacheKey)
	}
	latestCache, err := getLatestCacheEntry(executionCaches)
	if err != nil {
		return nil, err
	}
	return latestCache, nil
}

func (s *ExecutionCacheStore) cleanDatabase(maximumCacheStaleness int64) (int64, error) {
	// Expire any entry for any pipeline that is
	// older than os.LookupEnv("MAXIMUM_CACHE_STALENESS")
	if maximumCacheStaleness < 0 {
		return 0, nil
	}

	log.Printf("Cleaning cache entries older than maximumCacheStaleness=%d", maximumCacheStaleness)

	cutoffTime := s.time.Now().UTC().Unix() - maximumCacheStaleness
	result := s.db.Exec(
		"DELETE FROM execution_caches WHERE StartedAtInSec < ?",
		cutoffTime)

	if result.Error != nil {
		return 0, fmt.Errorf("database cache expiration cleanup failed: %v", result.Error)
	}

	return result.RowsAffected, nil
}
func (s *ExecutionCacheStore) scanRows(rows *sql.Rows, podCacheStaleness int64) ([]*model.ExecutionCache, error) {
	var executionCaches []*model.ExecutionCache
	for rows.Next() {
		var executionCacheKey, executionTemplate, executionOutput string
		var id, maxCacheStaleness, startedAtInSec, endedAtInSec int64
		err := rows.Scan(
			&id,
			&executionCacheKey,
			&executionTemplate,
			&executionOutput,
			&maxCacheStaleness,
			&startedAtInSec,
			&endedAtInSec)
		if err != nil {
			return executionCaches, nil
		}
		log.Println("Get id: " + strconv.FormatInt(id, 10))
		log.Println("Get template: " + executionTemplate)
		// maxCacheStaleness comes from the database entry.
		// podCacheStaleness is computed from the pods annotation and environment variables.
		if (maxCacheStaleness < 0 || s.time.Now().UTC().Unix()-startedAtInSec <= maxCacheStaleness) &&
			(podCacheStaleness < 0 || s.time.Now().UTC().Unix()-startedAtInSec <= podCacheStaleness) {
			executionCaches = append(executionCaches, &model.ExecutionCache{
				ID:                id,
				ExecutionCacheKey: executionCacheKey,
				ExecutionTemplate: executionTemplate,
				ExecutionOutput:   executionOutput,
				MaxCacheStaleness: maxCacheStaleness,
				StartedAtInSec:    startedAtInSec,
				EndedAtInSec:      endedAtInSec,
			})
		}
	}
	return executionCaches, nil
}

// Demo version will return the latest cache entry within same cache key. MaxCacheStaleness will
// be taken into consideration in the future.
func getLatestCacheEntry(executionCaches []*model.ExecutionCache) (*model.ExecutionCache, error) {
	var latestCacheEntry *model.ExecutionCache
	var maxStartedAtInSec int64
	for _, cache := range executionCaches {
		if cache.StartedAtInSec >= maxStartedAtInSec {
			latestCacheEntry = cache
			maxStartedAtInSec = cache.StartedAtInSec
		}
	}
	if latestCacheEntry == nil {
		return nil, fmt.Errorf("No cache entry found.")
	}
	return latestCacheEntry, nil
}

func (s *ExecutionCacheStore) CreateExecutionCache(executionCache *model.ExecutionCache) (*model.ExecutionCache, error) {
	log.Printf("checking for existing row with cache key: %s before insertion", executionCache.ExecutionCacheKey)

	r, err := s.db.Table("execution_caches").Where("ExecutionCacheKey = ?", executionCache.ExecutionCacheKey).Rows()
	if err != nil {
		log.Printf("Failed to get execution cache with key: %s, err: %v", executionCache.ExecutionCacheKey, err)
		return nil, err
	}

	rowCount := 0

	for r.Next() {
		rowCount++
	}
	log.Printf("number of rows returned for existing rows check: %d", rowCount)

	if rowCount == 0 {
		log.Printf("creating new exec cache row for key: %s", executionCache.ExecutionCacheKey)
		newExecutionCache := *executionCache
		now := s.time.Now().UTC().Unix()

		newExecutionCache.StartedAtInSec = now
		newExecutionCache.EndedAtInSec = now

		// GORM v2 removed NewRecord(); it was unreliable as it only checked whether the primary key was 0 / "" / nil

		var rowInsert model.ExecutionCache
		d := s.db.Create(&newExecutionCache).Scan(&rowInsert)
		if d.Error != nil {
			return nil, fmt.Errorf("failed to create new execution cache: %w", d.Error)
		}

		log.Printf("cache entry created successfully with key: %s, template: %v, row id: %d", executionCache.ExecutionCacheKey, executionCache.ExecutionTemplate, rowInsert.ID)
		return &rowInsert, nil
	} else {
		// return an error to prevent returning non-inserted user-provided structs and align with the unit test
		return nil, fmt.Errorf("execution cache with key %s already exists, failed to create new execution cache", executionCache.ExecutionCacheKey)
	}
}

// factory function for execution cache store
func NewExecutionCacheStore(db *DB, time util.TimeInterface) *ExecutionCacheStore {
	return &ExecutionCacheStore{
		db:   db,
		time: time,
	}
}
