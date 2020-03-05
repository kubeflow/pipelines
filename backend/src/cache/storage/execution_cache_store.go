package storage

import (
	"fmt"

	"database/sql"

	sq "github.com/Masterminds/squirrel"
	model "github.com/kubeflow/pipelines/backend/src/cache/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

type ExecutionCacheStoreInterface interface {
	GetExecutionCache(executionCacheKey string) (*model.ExecutionCache, error)
	CreateExecutionCache(*model.ExecutionCache) (*model.ExecutionCache, error)
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
		var executionCacheKey, executionOutput string
		var createdAtInSec int64
		err := rows.Scan(&executionCacheKey, &executionOutput, &createdAtInSec)
		if err != nil {
			return executionCaches, nil
		}
		executionCaches = append(executionCaches, &model.ExecutionCache{
			ExecutionCacheKey: executionCacheKey,
			ExecutionOutput:   executionOutput,
			CreatedAtInSec:    createdAtInSec,
		})
	}
	return executionCaches, nil
}

func (s *ExecutionCacheStore) CreateExecutionCache(executionCache *model.ExecutionCache) (*model.ExecutionCache, error) {
	newExecutionCache := *executionCache
	now := s.time.Now().Unix()

	newExecutionCache.CreatedAtInSec = now
	sql, args, err := sq.
		Insert("execution_caches").
		SetMap(
			sq.Eq{
				"ExecutionCacheKey": newExecutionCache.ExecutionCacheKey,
				"ExecutionOutput":   newExecutionCache.ExecutionOutput,
				"CreatedAtInSec":    newExecutionCache.CreatedAtInSec}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("Failed to create query to insert execution cache to execution cache table")
	}
	_, err = s.db.Exec(sql, args...)
	if err != nil {
		return nil, fmt.Errorf("Failed to create a new execution cache")
	}
	return &newExecutionCache, nil
}

// factory function for execution cache store
func NewExecutionCacheStore(db *DB, time util.TimeInterface) *ExecutionCacheStore {
	return &ExecutionCacheStore{
		db:   db,
		time: time,
	}
}
