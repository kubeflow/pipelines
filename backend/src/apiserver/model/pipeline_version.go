// Copyright 2019 The Kubeflow Authors
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
	UUID           string `gorm:"column:UUID; not null; primaryKey;type:varchar(191);"`
	CreatedAtInSec int64  `gorm:"column:CreatedAtInSec; not null; index:idx_pipeline_versions_CreatedAtInSec;"`
	// Explicitly specify varchar(127)
	// so that the combined index (PipelineId + Name) does not exceed 767 bytes in utf8mb4,
	// For details on type lengths and index safety, refer to comments in the Pipeline struct.
	Name        string `gorm:"column:Name; not null; type:varchar(127); uniqueIndex:idx_pipelineid_name;"`
	DisplayName string `gorm:"column:DisplayName; not null"`
	// TODO(gkcalat): this is deprecated. Consider removing and adding data migration logic at the server startup.
	Parameters string `gorm:"column:Parameters; not null; type:longtext;"` // deprecated
	// PipelineVersion belongs to Pipeline. If a pipeline with a specific UUID
	// is deleted from Pipeline table, all this pipeline's versions will be
	// deleted from PipelineVersion table.
	// Explicitly specify varchar(64). Refer to Name field comments for details.
	// nolint:staticcheck // [ST1003] Field name matches upstream legacy naming
	PipelineId string                `gorm:"column:PipelineId; not null; index:idx_pipeline_versions_PipelineId; uniqueIndex:idx_pipelineid_name; type:varchar(64)"`
	Pipeline   Pipeline              `gorm:"foreignKey:PipelineId; references:UUID;constraint:pipeline_versions_PipelineId_pipelines_UUID_foreign,OnDelete:CASCADE,OnUpdate:CASCADE"` // This 'belongs to' relation replaces the legacy AddForeignKey constraint previously defined in client_manager.go
	Status     PipelineVersionStatus `gorm:"column:Status; not null;"`
	// Code source url links to the pipeline version's definition in repo.
	CodeSourceUrl   string `gorm:"column:CodeSourceUrl;"`
	Description     string `gorm:"column:Description; type:longtext;"`
	PipelineSpec    string `gorm:"column:PipelineSpec; not null; type:longtext;"`    // Same as common.MaxFileLength (32MB in server). Argo imposes 700kB limit
	PipelineSpecURI string `gorm:"column:PipelineSpecURI; not null; type:longtext;"` // Can store references to ObjectStore files
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
		"id":                  "UUID",        // v1beta1 API
		"pipeline_version_id": "UUID",        // v2beta1 API
		"name":                "Name",        // v1beta1 API
		"display_name":        "DisplayName", // v2beta1 API
		"created_at":          "CreatedAtInSec",
		"status":              "Status",
		"description":         "Description",  // v2beta1 API
		"pipeline_spec":       "PipelineSpec", // v2beta1 API
	}
}

// GetModelName returns table name used as sort field prefix.
func (p *PipelineVersion) GetModelName() string {
	return "pipeline_versions"
}

// TableName overrides GORM's table name inference.
func (PipelineVersion) TableName() string {
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
	case "DisplayName":
		return p.DisplayName
	case "CreatedAtInSec":
		return p.CreatedAtInSec
	case "Status":
		return p.Status
	case "Description":
		return p.Description
	case "CodeSourceUrl":
		return p.CodeSourceUrl
	case "PipelineSpec":
		return p.PipelineSpec
	case "PipelineSpecURI":
		return p.PipelineSpecURI
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
