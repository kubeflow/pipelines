// Copyright 2026 The Kubeflow Authors
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

// TaskLogHistory retains only troubleshooting references from a task before a
// manual retry resets it. Deliberately no foreign keys: introducing this table
// must not alter or lock existing task/run tables during schema migration.
// Run deletion removes these rows in the same transaction as the tasks.
type TaskLogHistory struct {
	TaskID                string    `gorm:"column:TaskID;type:varchar(191);primaryKey;not null"`
	BeforeRetryGeneration int64     `gorm:"column:BeforeRetryGeneration;primaryKey;autoIncrement:false;not null"`
	RunUUID               string    `gorm:"column:RunUUID;type:varchar(191);not null;index:idx_task_log_history_run"`
	Snapshot              LargeText `gorm:"column:Snapshot;not null"`
}

func (TaskLogHistory) TableName() string { return "task_log_history" }
