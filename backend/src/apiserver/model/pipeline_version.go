// Copyright 2019-2022 The Kubeflow Authors
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

	"github.com/kubeflow/pipelines/backend/src/common/util"
	"gorm.io/gorm"
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
	gorm.Model
	UUID           string `gorm:"column:UUID; not null; primary_key;unique_index:idx_pipelineid_name;"`
	CreatedAtInSec int64  `gorm:"column:CreatedAtInSec; not null; index"`
	Name           string `gorm:"column:Name; not null;unique_index:idx_pipelineid_name;"`
	// Set size to 65535 so it will be stored as longtext.
	// https://dev.mysql.com/doc/refman/8.0/en/column-count-limit.html
	Parameters string `gorm:"column:Parameters; not null; size:65535"`
	// PipelineVersion belongs to Pipeline. If a pipeline with a specific UUID
	// is deleted from Pipeline table, all this pipeline's versions will be
	// deleted from PipelineVersion table.
	PipelineId string                `gorm:"column:PipelineId; not null;index; unique_index:idx_pipelineid_name;"`
	Pipeline   *Pipeline             `gorm:"foreignKey:PipelineId; constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Status     PipelineVersionStatus `gorm:"column:Status; not null"`
	// Code source url links to the pipeline version's definition in repo.
	CodeSourceUrl   string `gorm:"column:CodeSourceUrl;"`
	Description     string `gorm:"column:Description; not null; size:65535;"`     // Set size to large number so it will be stored as longtext
	PipelineSpec    string `gorm:"column:PipelineSpec; not null; size:33554432;"` // Same as common.MaxFileLength (32MB in server). Argo imposes 700kB limit
	PipelineSpecURI string `gorm:"column:PipelineSpecURI; not null; size:65535;"` // Can store references to ObjectStore files
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
		"id":                  "UUID",
		"pipeline_version_id": "UUID", // Added for KFP v2
		"name":                "Name",
		"display_name":        "Name", // Added for KFP v2
		"created_at":          "CreatedAtInSec",
		"status":              "Status",
		"description":         "Description",  // Added for KFP v2
		"pipeline_spec":       "PipelineSpec", // Added for KFP v2
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

// Returns a value of a field in a PipelineVersion object.
// If not found, checks for proto-style names based on APIToModelFieldMap.
// Returns nil if does not exist.
func (p *PipelineVersion) GetProtoFieldValue(name string) interface{} {
	if val := p.GetFieldValue(name); val == nil {
		newKey, ok := p.APIToModelFieldMap()[name]
		if ok {
			return p.GetFieldValue(newKey)
		} else {
			return nil
		}
	} else {
		return val
	}
}

func (p *PipelineVersion) GetSortByFieldPrefix(name string) string {
	return "pipeline_versions."
}

func (p *PipelineVersion) GetKeyFieldPrefix() string {
	return "pipeline_versions."
}

// Sets a value based on the field's name provided.
// Returns NewInvalidInputError if does not exist.
func (p *PipelineVersion) SetFieldValue(name string, value interface{}) error {
	switch name {
	case "UUID":
		p.UUID = value.(string)
		return nil
	case "Name":
		p.Name = value.(string)
		return nil
	case "CreatedAtInSec":
		p.CreatedAtInSec = value.(int64)
		return nil
	case "Status":
		p.Status = value.(PipelineVersionStatus)
		return nil
	case "Description":
		p.Description = value.(string)
		return nil
	case "CodeSourceUrl":
		p.CodeSourceUrl = value.(string)
		return nil
	case "PipelineSpec":
		p.PipelineSpec = value.(string)
		return nil
	case "PipelineSpecURI":
		p.PipelineSpecURI = value.(string)
		return nil
	default:
		return util.NewInvalidInputError(fmt.Sprintf("Error setting field '%s' to '%v' in PipelineVersion object: not found or not allowed to set.", name, value))
	}
}

// Sets a value based on the field's name provided.
// If not found, checks for proto-style names based on APIToModelFieldMap.
// Returns NewInvalidInputError if does not exist.
func (p *PipelineVersion) SetProtoFieldValue(name string, value interface{}) error {
	if err := p.SetFieldValue(name, value); err != nil {
		newKey, ok := p.APIToModelFieldMap()[name]
		if ok {
			return p.SetFieldValue(newKey, value)
		} else {
			return err
		}
	}
	return nil
}
