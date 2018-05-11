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

// PipelineStatus a label for the status of the Pipeline.
// This is intend to make pipeline creation and deletion atomic.
type PipelineStatus string

const (
	PipelineCreating PipelineStatus = "CREATING"
	PipelineReady    PipelineStatus = "READY"
	PipelineDeleting PipelineStatus = "DELETING"
)

type Pipeline struct {
	ID             uint32 `gorm:"primary_key"`
	CreatedAtInSec int64  `gorm:"not null"`
	UpdatedAtInSec int64  `gorm:"not null"`
	Name           string `gorm:"not null"`
	Description    string
	PackageId      uint32         `gorm:"not null"`
	Schedule       string         `gorm:"not null"`
	Enabled        bool           `gorm:"not null"`
	EnabledAtInSec int64          `gorm:"not null"`
	Parameters     []Parameter    `gorm:"polymorphic:Owner;"`
	Status         PipelineStatus `gorm:"not null"`
}
