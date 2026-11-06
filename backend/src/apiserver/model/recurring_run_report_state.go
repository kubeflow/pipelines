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

// RecurringRunReportState orders ScheduledWorkflow specifications independently
// of scheduling progress. It is populated lazily, without rewriting existing jobs.
// There is deliberately no foreign key or association that migrates the jobs table.
type RecurringRunReportState struct {
	JobUUID    string `gorm:"column:JobUUID;primaryKey;type:varchar(191)"`
	Generation int64  `gorm:"column:Generation;not null"`
}

func (RecurringRunReportState) TableName() string {
	return "recurring_run_report_states"
}
