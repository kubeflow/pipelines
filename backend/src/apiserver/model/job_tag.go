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

// JobTag stores user-defined metadata for a recurring run.
type JobTag struct {
	JobID    string `gorm:"column:JobId; not null; primaryKey; type:varchar(191);index:idx_job_tags_key_value_id,priority:3"`
	TagKey   string `gorm:"column:TagKey; not null; primaryKey; type:varchar(63);index:idx_job_tags_key_value_id,priority:1"`
	TagValue string `gorm:"column:TagValue; not null; type:varchar(63);index:idx_job_tags_key_value_id,priority:2"`
	Job      Job    `gorm:"foreignKey:JobID; references:UUID; constraint:job_tags_JobId_jobs_UUID_fk,OnDelete:CASCADE,OnUpdate:CASCADE"`
}

// TableName overrides GORM's table name inference.
func (JobTag) TableName() string { return "job_tags" }
