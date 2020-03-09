package storage

import (
	"fmt"
	"log"

	"database/sql"

	sq "github.com/Masterminds/squirrel"
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
	sql, args, err := sq.Select("*").From("execution_caches").Where(sq.Eq{"executionCacheKey": executionCacheKey}).Limit(1).ToSql()
	if err != nil {
		return nil, fmt.Errorf("Failed to get execution cache: %q", executionCacheKey)
	}
	r, err := s.db.Query(sql, args...)
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
		var maxCacheStaleness, startedAtInSec, endedAtInSec int64
		err := rows.Scan(
			&executionCacheKey,
			&executionTemplate,
			&executionOutput,
			&maxCacheStaleness,
			&startedAtInSec,
			&endedAtInSec)
		if err != nil {
			return executionCaches, nil
		}
		executionCaches = append(executionCaches, &model.ExecutionCache{
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
	newExecutionCache := *executionCache
	now := s.time.Now().Unix()

	newExecutionCache.StartedAtInSec = now
	newExecutionCache.MaxCacheStaleness = -1
	sql, args, err := sq.
		Insert("execution_caches").
		SetMap(
			sq.Eq{
				"ExecutionCacheKey": newExecutionCache.ExecutionCacheKey,
				"ExecutionTemplate": newExecutionCache.ExecutionTemplate,
				"ExecutionOutput":   "testoutput",
				"MaxCacheStaleness": newExecutionCache.MaxCacheStaleness,
				"StartedAtInSec":    newExecutionCache.StartedAtInSec,
				"EndedAtInSec":      newExecutionCache.StartedAtInSec}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("Failed to create query to insert execution cache to execution cache table")
	}
	_, err = s.db.Exec(sql, args...)
	if err != nil {
		log.Println(err.Error())
		return nil, fmt.Errorf("Failed to create a new execution cache")
	}
	return &newExecutionCache, nil
}

func (s *ExecutionCacheStore) DeleteExecutionCache(executionCacheKey string) error {
	sql, args, err := sq.Delete("execution_caches").Where(sq.Eq{"ExecutionCacheKey": executionCacheKey}).ToSql()
	if err != nil {
		return err
	}
	_, err = s.db.Exec(sql, args...)
	if err != nil {
		log.Println(err.Error())
		return fmt.Errorf("Failed to delete cache entry")
	}
	// _, err = tx.Exec(sql, args...)
	// if err != nil {
	// 	tx.Rollback()
	// 	return fmt.Errorf("Failed to delete cache entry %s from table", executionCacheKey)
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
