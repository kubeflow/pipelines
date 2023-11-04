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
	UUID           string `gorm:"column:UUID; not null; primary_key;"`
	CreatedAtInSec int64  `gorm:"column:CreatedAtInSec; not null;"`
	Name           string `gorm:"column:Name; not null; unique_index:namespace_name;"` // Index improves performance of the List ang Get queries
	Description    string `gorm:"column:Description; size:65535;"`                     // Same as below, set size to large number so it will be stored as longtext
	// TODO(gkcalat): this is deprecated. Consider removing and adding data migration logic at the server startup.
	Parameters string         `gorm:"column:Parameters; size:65535;"`
	Status     PipelineStatus `gorm:"column:Status; not null;"`
	// TODO(gkcalat): this is deprecated. Consider removing and adding data migration logic at the server startup.
	DefaultVersionId string `gorm:"column:DefaultVersionId;"` // deprecated
	Namespace        string `gorm:"column:Namespace; unique_index:namespace_name; size:63;"`
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
	"id":           "UUID", // v1beta1 API
	"pipeline_id":  "UUID", // v2beta1 API
	"name":         "Name", // v1beta1 API
	"display_name": "Name", // v2beta1 API
	"created_at":   "CreatedAtInSec",
	"description":  "Description",
	"namespace":    "Namespace",
}

// APIToModelFieldMap returns a map from API names to field names for model
// Pipeline.
func (p *Pipeline) APIToModelFieldMap() map[string]string {
	return pipelineAPIToModelFieldMap
}

// GetModelName returns table name used as sort field prefix.
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

func (p *Pipeline) GetSortByFieldPrefix(name string) string {
	return "pipelines."
}

func (p *Pipeline) GetKeyFieldPrefix() string {
	return "pipelines."
}
