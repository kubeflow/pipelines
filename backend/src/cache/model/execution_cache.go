package model

type ExecutionCache struct {
	ExecutionCacheKey string `gorm:"column:ExecutionCacheKey; not null; primary_key"`
	ExecutionTemplate string `gorm:"column:ExecutionTemplate; not null"`
	ExecutionOutput   string `gorm:"column:ExecutionOutput;"`
	MaxCacheStaleness int64  `gorm:"column:MaxCacheStaleness;"`
	StartedAtInSec    int64  `gorm:"column:StartedAtInSec; not null"`
	EndedAtInSec      int64  `gorm:"column:EndedAtInSec; not null"`
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
	return "executionCaches"
}

func (e *ExecutionCache) GetExecutionOutput() string {
	return e.ExecutionOutput
}

func (e *ExecutionCache) GetExecutionTemplate() string {
	return e.ExecutionTemplate
}

func (e *ExecutionCache) GetMaxCacheStaleness() int64 {
	return e.MaxCacheStaleness
}

func (e *ExecutionCache) GetEndedAtInSec() int64 {
	return e.EndedAtInSec
}
