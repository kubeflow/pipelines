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

package model

type ExecutionCache struct {
	ID                int64  `gorm:"column:ID; not null; primaryKey; AUTO_INCREMENT; index:composite_id_idx"`
	ExecutionCacheKey string `gorm:"column:ExecutionCacheKey; not null; index:idx_cache_key;"`
	ExecutionTemplate string `gorm:"column:ExecutionTemplate; not null;"`
	ExecutionOutput   string `gorm:"column:ExecutionOutput; not null;"`
	MaxCacheStaleness int64  `gorm:"column:MaxCacheStaleness; not null;"`
	StartedAtInSec    int64  `gorm:"column:StartedAtInSec; not null; index:composite_id_idx;"`
	EndedAtInSec      int64  `gorm:"column:EndedAtInSec; not null;"`
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
