// Copyright 2018 Google LLC
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

package message

import (
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

// Job metadata of a job.
type Job struct {
	*Metadata        `json:",omitempty"`
	Name             string `json:"name" gorm:"not null"`
	ScheduledAtInSec int64  `json:"scheduledAt" gorm:"not null"`
	CreatedAtInSec   int64  `json:"-" gorm:"not null"`
	PipelineID       uint   `json:"-"` /* Foreign key */
}

// JobDetail a detailed view of a Argo job, including templates, job status etc.
type JobDetail struct {
	Workflow *v1alpha1.Workflow `json:"jobDetail"`
	Job      *Job               `json:"metadata"`
}
