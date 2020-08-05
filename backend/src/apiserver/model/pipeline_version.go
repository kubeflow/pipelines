// Copyright 2019 Google LLC
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
	"fmt"
)

// PipelineVersionStatus a label for the status of the Pipeline.
// This is intend to make pipeline creation and deletion atomic.
type PipelineVersionStatus string

const (
	PipelineVersionCreating PipelineVersionStatus = "CREATING"
	PipelineVersionReady    PipelineVersionStatus = "READY"
	PipelineVersionDeleting PipelineVersionStatus = "DELETING"
)

type PipelineVersion struct {
	UUID           string `gorm:"column:UUID; not null; primary_key"`
	CreatedAtInSec int64  `gorm:"column:CreatedAtInSec; not null; index"`
	Name           string `gorm:"column:Name; not null; unique_index:idx_pipelineid_name"`
	// Set size to 65535 so it will be stored as longtext.
	// https://dev.mysql.com/doc/refman/8.0/en/column-count-limit.html
	Parameters string `gorm:"column:Parameters; not null; size:65535"`
	// PipelineVersion belongs to Pipeline. If a pipeline with a specific UUID
	// is deleted from Pipeline table, all this pipeline's versions will be
	// deleted from PipelineVersion table.
	PipelineId string                `gorm:"column:PipelineId; not null;index; unique_index:idx_pipelineid_name"`
	Status     PipelineVersionStatus `gorm:"column:Status; not null"`
	// Code source url links to the pipeline version's definition in repo.
	CodeSourceUrl string `gorm:"column:CodeSourceUrl;"`
}

func (p PipelineVersion) GetValueOfPrimaryKey() string {
	return fmt.Sprint(p.UUID)
}

// PrimaryKeyColumnName returns the primary key for model PipelineVersion.
func (p *PipelineVersion) PrimaryKeyColumnName() string {
	return "UUID"
}

// DefaultSortField returns the default sorting field for model Pipeline.
func (p *PipelineVersion) DefaultSortField() string {
	return "CreatedAtInSec"
}

// APIToModelFieldMap returns a map from API names to field names for model
// PipelineVersion.
func (p *PipelineVersion) APIToModelFieldMap() map[string]string {
	return map[string]string{
		"id":         "UUID",
		"name":       "Name",
		"created_at": "CreatedAtInSec",
		"status":     "Status",
	}
}

// GetModelName returns table name used as sort field prefix
func (p *PipelineVersion) GetModelName() string {
	return "pipeline_versions"
}

func (p *PipelineVersion) GetField(name string) (string, bool) {
	if field, ok := p.APIToModelFieldMap()[name]; ok {
		return field, true
	}
	return "", false
}

func (p *PipelineVersion) GetFieldValue(name string) interface{} {
	switch name {
	case "UUID":
		return p.UUID
	case "Name":
		return p.Name
	case "CreatedAtInSec":
		return p.CreatedAtInSec
	case "Status":
		return p.Status
	default:
		return nil
	}
}

func (p *PipelineVersion) GetSortByFieldPrefix(name string) string {
	return "pipeline_versions."
}

func (p *PipelineVersion) GetKeyFieldPrefix() string {
	return "pipeline_versions."
}
