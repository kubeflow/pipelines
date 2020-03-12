package storage

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"

	model "github.com/kubeflow/pipelines/backend/src/cache/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

type ExecutionCacheStoreInterface interface {
	GetExecutionCache(executionCacheKey string) (*model.ExecutionCache, error)
	CreateExecutionCache(*model.ExecutionCache) (*model.ExecutionCache, error)
	DeleteExecutionCache(executionCacheKey string) error
}

type ExecutionCacheStore struct {
	db   *DB
	time util.TimeInterface
}

func (s *ExecutionCacheStore) GetExecutionCache(executionCacheKey string) (*model.ExecutionCache, error) {
	r, err := s.db.Table("execution_caches").Where("ExecutionCacheKey = ?", executionCacheKey).Rows()

	if err != nil {
		return nil, fmt.Errorf("Failed to get execution cache: %q", executionCacheKey)
	}
	defer r.Close()
	executionCaches, err := s.scanRows(r)

	if err != nil || len(executionCaches) > 1 {
		return nil, fmt.Errorf("Failed to get execution cache: %q", executionCacheKey)
	}
	if len(executionCaches) == 0 {
		return nil, fmt.Errorf("Execution cache not found with cache key: %q", executionCacheKey)
	}
	return executionCaches[0], nil
}

func (s *ExecutionCacheStore) scanRows(rows *sql.Rows) ([]*model.ExecutionCache, error) {
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
	return executionCaches, nil
}

func (s *ExecutionCacheStore) CreateExecutionCache(executionCache *model.ExecutionCache) (*model.ExecutionCache, error) {
	log.Println("Input cache: " + executionCache.ExecutionCacheKey)
	newExecutionCache := *executionCache
	log.Println("Nex cache key: " + newExecutionCache.ExecutionCacheKey)
	now := s.time.Now().UTC().Unix()

	newExecutionCache.StartedAtInSec = now
	newExecutionCache.MaxCacheStaleness = -1

	ok := s.db.NewRecord(newExecutionCache)
	if !ok {
		return nil, fmt.Errorf("Failed to create a new execution cache")
	}
	var rowInsert model.ExecutionCache
	d := s.db.Create(&newExecutionCache).Scan(&rowInsert)
	if d.Error != nil {
		return nil, d.Error
	}
	log.Println("Cache entry created with cache key: " + newExecutionCache.ExecutionCacheKey)
	log.Println(newExecutionCache.ExecutionTemplate)
	log.Println(rowInsert.ID)
	return &rowInsert, nil
}

func (s *ExecutionCacheStore) DeleteExecutionCache(executionCacheID string) error {
	// _, err = s.db.Exec(sql, args...)
	s.db.Delete(&model.ExecutionCache{}, "ID = ?", executionCacheID)
	// if err != nil {
	// 	log.Println(err.Error())
	// 	return fmt.Errorf("Failed to delete cache entry")
	// }

	return nil
}

// factory function for execution cache store
func NewExecutionCacheStore(db *DB, time util.TimeInterface) *ExecutionCacheStore {
	return &ExecutionCacheStore{
		db:   db,
		time: time,
	}
}
