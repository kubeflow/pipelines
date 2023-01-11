// Copyright 2018 The Kubeflow Authors
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
)

// PipelineStatus a label for the status of the Pipeline.
// This is intend to make pipeline creation and deletion atomic.
type PipelineStatus string

const (
	PipelineCreating PipelineStatus = "CREATING"
	PipelineReady    PipelineStatus = "READY"
	PipelineDeleting PipelineStatus = "DELETING"

	NoNamespace string = "-"
)

type Pipeline struct {
	// gorm.Model
	UUID           string `gorm:"column:UUID; not null; primary_key;"`
	CreatedAtInSec int64  `gorm:"column:CreatedAtInSec; not null;"`
	Name           string `gorm:"column:Name; not null; unique_index:namespace_name;"` // Index improves performance of the List ang Get queries
	Description    string `gorm:"column:Description; not null; size:65535;"`           // Same as below, set size to large number so it will be stored as longtext
	// TODO(gkcalat): this is deprecated. Consider removing and adding data migration logic at the server startup.
	Parameters string         `gorm:"column:Parameters; size:65535; default='';"`
	Status     PipelineStatus `gorm:"column:Status; not null;"`
	// TODO(gkcalat): this is deprecated. Consider removing and adding data migration logic at the server startup.
	DefaultVersionId string `gorm:"column:DefaultVersionId; default='';"` // deprecated
	Namespace        string `gorm:"column:Namespace; unique_index:namespace_name; size:63; default:'';"`
}

func (p Pipeline) GetValueOfPrimaryKey() string {
	return fmt.Sprint(p.UUID)
}

func GetPipelineTablePrimaryKeyColumn() string {
	return "UUID"
}

// PrimaryKeyColumnName returns the primary key for model Pipeline.
func (p *Pipeline) PrimaryKeyColumnName() string {
	return "UUID"
}

// DefaultSortField returns the default sorting field for model Pipeline.
func (p *Pipeline) DefaultSortField() string {
	return "CreatedAtInSec"
}

var pipelineAPIToModelFieldMap = map[string]string{
	"id":           "UUID",
	"pipeline_id":  "UUID", // Added for KFP v2
	"name":         "Name",
	"display_name": "Name", // Added for KFP v2
	"created_at":   "CreatedAtInSec",
	"description":  "Description",
	"namespace":    "Namespace",
}

// APIToModelFieldMap returns a map from API names to field names for model
// Pipeline.
func (p *Pipeline) APIToModelFieldMap() map[string]string {
	return pipelineAPIToModelFieldMap
}

// GetModelName returns table name used as sort field prefix
func (p *Pipeline) GetModelName() string {
	return "pipelines"
}

func (p *Pipeline) GetField(name string) (string, bool) {
	if field, ok := pipelineAPIToModelFieldMap[name]; ok {
		return field, true
	}
	return "", false
}

func (p *Pipeline) GetFieldValue(name string) interface{} {
	switch name {
	case "UUID":
		return p.UUID
	case "Name":
		return p.Name
	case "CreatedAtInSec":
		return p.CreatedAtInSec
	case "Description":
		return p.Description
	case "Namespace":
		return p.Namespace
	default:
		return nil
	}
}

// Returns a value of a field in a Pipeline object.
// If not found, checks for proto-style names based on APIToModelFieldMap.
// Returns nil if does not exist.
func (p *Pipeline) GetProtoFieldValue(name string) interface{} {
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

func (p *Pipeline) GetSortByFieldPrefix(name string) string {
	return "pipelines."
}

func (p *Pipeline) GetKeyFieldPrefix() string {
	return "pipelines."
}

// Sets a value based on the field's name provided.
// Returns NewInvalidInputError if does not exist.
func (p *Pipeline) SetFieldValue(name string, value interface{}) error {
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
	case "Description":
		p.Description = value.(string)
		return nil
	case "Namespace":
		p.Namespace = value.(string)
		return nil
	default:
		return util.NewInvalidInputError(fmt.Sprintf("Error setting field '%s' to '%v' in Pipeline object: not found or not allowed to set.", name, value))
	}
}

// Sets a value based on the field's name provided.
// If not found, checks for proto-style names based on APIToModelFieldMap.
// Returns NewInvalidInputError if does not exist.
func (p *Pipeline) SetProtoFieldValue(name string, value interface{}) error {
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
