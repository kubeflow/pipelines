package model

type ExecutionCache struct {
	ID                int64  `gorm:"column:ID; not null; primary_key; AUTO_INCREMENT"`
	ExecutionCacheKey string `gorm:"column:ExecutionCacheKey; not null; index:idx_cache_key"`
	ExecutionTemplate string `gorm:"column:ExecutionTemplate; not null"`
	ExecutionOutput   string `gorm:"column:ExecutionOutput; not null"`
	MaxCacheStaleness int64  `gorm:"column:MaxCacheStaleness; not null"`
	StartedAtInSec    int64  `gorm:"column:StartedAtInSec; not null"`
	EndedAtInSec      int64  `gorm:"column:EndedAtInSec; not null"`
}

// GetValueOfPrimaryKey returns the value of ExecutionCacheKey.
func (e ExecutionCache) GetValueOfPrimaryKey() int64 {
	return e.ID
}

// GetExecutionCacheTablePrimaryKeyColumn returns the primary key column of ExecutionCache.
func GetExecutionCacheTablePrimaryKeyColumn() string {
	return "ID"
}

// PrimaryKeyColumnName returns the primary key for ExecutionCache.
func (e *ExecutionCache) PrimaryKeyColumnName() string {
	return "ID"
}

// GetModelName returns the name of ExecutionCache.
func (e *ExecutionCache) GetModelName() string {
	return "executionCaches"
}
