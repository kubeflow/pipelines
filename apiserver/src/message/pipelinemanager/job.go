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

package pipelinemanager

import (
	"time"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

type Job struct {
	Name     string     `json:"name"`
	CreateAt *time.Time `json:"createdAt,omitempty"`
	StartAt  *time.Time `json:"startedAt,omitempty"`
	FinishAt *time.Time `json:"finishedAt,omitempty"`
	Status   string     `json:"status,omitempty"`
}

func ToJob(workflow v1alpha1.Workflow) Job {
	return Job{
		Name:     workflow.ObjectMeta.Name,
		CreateAt: &workflow.ObjectMeta.CreationTimestamp.Time,
		StartAt:  &workflow.Status.StartedAt.Time,
		FinishAt: &workflow.Status.FinishedAt.Time,
		Status:   string(workflow.Status.Phase)}
}
