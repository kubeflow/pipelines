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

import "fmt"

// PipelineStatus a label for the status of the Pipeline.
// This is intend to make pipeline creation and deletion atomic.
type PipelineStatus string

const (
	PipelineCreating PipelineStatus = "CREATING"
	PipelineReady    PipelineStatus = "READY"
	PipelineDeleting PipelineStatus = "DELETING"
)

type Pipeline struct {
	ID             uint32         `gorm:"column:ID; primary_key"`
	CreatedAtInSec int64          `gorm:"column:CreatedAtInSec; not null"`
	UpdatedAtInSec int64          `gorm:"column:UpdatedAtInSec; not null"`
	Name           string         `gorm:"column:Name; not null"`
	Description    string         `gorm:"column:Description"`
	PackageId      uint32         `gorm:"column:PackageId; not null"`
	Schedule       string         `gorm:"column:Schedule; not null"`
	Enabled        bool           `gorm:"column:Enabled; not null"`
	EnabledAtInSec int64          `gorm:"column:EnabledAtInSec; not null"`
	Parameters     string         `gorm:"column:Parameters"` /* Json format argo.v1alpha1.parameter */
	Status         PipelineStatus `gorm:"column:Status; not null"`
}

func (p Pipeline) GetValueOfPrimaryKey() string {
	return fmt.Sprint(p.ID)
}

func GetPipelineTablePrimaryKeyColumn() string {
	return "ID"
}
