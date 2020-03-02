package model

type ExecutionCache struct {
	ExecutionCacheKey string `gorm:"column:ExecutionCacheKey; not null; primary_key"`
	ExecutionOutput   string `gorm:"column:ExecutionOutput; not null"`
	CreatedAtInSec    int64  `gorm:"column:CreatedAtInSec; not null"`
}

// GetValueOfPrimaryKey returns the value of ExecutionCacheKey.
func (e ExecutionCache) GetValueOfPrimaryKey() string {
	return e.ExecutionCacheKey
}

// GetExecutionCacheTablePrimaryKeyColumn returns the primary key column of ExecutionCache.
func GetExecutionCacheTablePrimaryKeyColumn() string {
	return "ExecutionCacheKey"
}

// PrimaryKeyColumnName returns the primary key for ExecutionCache.
func (e *ExecutionCache) PrimaryKeyColumnName() string {
	return "ExecutionCacheKey"
}

// GetModelName returns the name of ExecutionCache.
func (e *ExecutionCache) GetModelName() string {
	return "executionCache"
}
