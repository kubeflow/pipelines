// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package model

// Runtime task phases are separate from the task's API-visible state.
const (
	RuntimeTaskWaiting           = "WAITING"
	RuntimeTaskReady             = "READY"
	RuntimeTaskSubmitting        = "SUBMITTING"
	RuntimeTaskRunning           = "RUNNING"
	RuntimeTaskPublishing        = "PUBLISHING"
	RuntimeTaskFailed            = "FAILED"
	RuntimeTaskBackoff           = "BACKOFF"
	RuntimeTaskCanceling         = "CANCELING"
	RuntimeTaskFinalizing        = "finalizing"
	RuntimeTaskFailureFinalizing = "failure-finalizing"
	RuntimeTaskCancelFinalizing  = "cancel-finalizing"
)

// RuntimeTask durably records scheduling and the current execution attempt.
// Finalization removes this row, not the public task or its diagnostics.
type RuntimeTask struct {
	TaskUUID    string  `gorm:"column:TaskUUID;type:varchar(191);not null;primaryKey"`
	RunUUID     string  `gorm:"column:RunUUID;type:varchar(191);not null;index:idx_runtime_tasks_run"`
	AttemptUUID *string `gorm:"column:AttemptUUID;type:varchar(191);uniqueIndex:idx_runtime_tasks_attempt"`
	State       string  `gorm:"column:State;type:varchar(32);not null"`
	RetryCount  int     `gorm:"column:RetryCount;not null;default:0"`
	// Publication retries do not spend the workload's computation retry budget.
	PublicationRetryCount int      `gorm:"column:PublicationRetryCount;not null;default:0"`
	NextEligibleAt        *int64   `gorm:"column:NextEligibleAt"`
	ExecutionRequest      JSONData `gorm:"column:ExecutionRequest;type:json"`
	ExecutionReference    JSONData `gorm:"column:ExecutionReference;type:json"`

	Task Task `gorm:"foreignKey:TaskUUID,RunUUID;references:UUID,RunUUID;constraint:fk_runtime_tasks_task_run,OnDelete:CASCADE,OnUpdate:CASCADE"`
	Run  Run  `gorm:"foreignKey:RunUUID;references:UUID;constraint:fk_runtime_tasks_run,OnDelete:CASCADE,OnUpdate:CASCADE"`
}

// TableName returns the runtime coordination table.
func (RuntimeTask) TableName() string {
	return "runtime_tasks"
}
