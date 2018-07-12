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

package model

import (
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

// JobStatus is a label for the status of the job
type JobStatus string

const (
	JobCreationPending  JobStatus = "CREATION_PENDING"  /* Waiting for K8s resource to be created */
	JobExecutionPending JobStatus = "EXECUTION_PENDING" /* Waiting for Ks8 workflow to run */
)

// Job metadata of a job.
type Job struct {
	Name             string    `gorm:"column:Name; not null; primary_key"`
	CreatedAtInSec   int64     `gorm:"column:CreatedAtInSec; not null"`
	UpdatedAtInSec   int64     `gorm:"column:UpdatedAtInSec; not null"`
	Status           JobStatus `gorm:"column:Status; not null"`
	ScheduledAtInSec int64     `gorm:"column:ScheduledAtInSec; not null"`
	PipelineID       uint32    `gorm:"column:PipelineID; not null"`
}

// JobDetail a wrapper around both Argo workflow and Job metadata
type JobDetail struct {
	Workflow *v1alpha1.Workflow
	Job      *Job
}

func (j Job) GetValueOfPrimaryKey() string {
	return j.Name
}

func GetJobTablePrimaryKeyColumn() string {
	return "Name"
}
