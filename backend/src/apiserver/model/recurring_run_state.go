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

// RecurringRunState stores API-owned scheduling progress. A pending claim keeps
// its index and timestamps until workflow creation has been persisted.
type RecurringRunState struct {
	JobUUID              string `gorm:"column:JobUUID; not null; primaryKey; type:varchar(191);"`
	RequestKey           string `gorm:"column:RequestKey; not null; type:text;"`
	PipelineVersionID    string `gorm:"column:PipelineVersionID; not null; type:varchar(191);"`
	LastRunIndex         int64  `gorm:"column:LastRunIndex; not null; default:0;"`
	LastScheduledAtInSec int64  `gorm:"column:LastScheduledAtInSec; not null; default:0;"`
	LastCreatedAtInSec   int64  `gorm:"column:LastCreatedAtInSec; not null; default:0;"`
	Pending              bool   `gorm:"column:Pending; not null; default:false;"`
}

func (RecurringRunState) TableName() string {
	return "recurring_run_states"
}
