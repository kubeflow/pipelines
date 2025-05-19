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
	UUID           string `gorm:"column:UUID; not null; primaryKey;type:varchar(64);"`
	CreatedAtInSec int64  `gorm:"column:CreatedAtInSec; not null;"`
	// Name is limited to varchar(128) to ensure the composite index (Namespace, Name)
	// stays within MySQL’s 767-byte prefix limit for indexable columns.
	// MySQL uses utf8mb4 encoding by default, where each character can take up to 4 bytes.
	// In the worst case: 63 (namespace) * 4 + 128 (name) * 4 = 764 bytes ≤ 767 bytes.
	// https://dev.mysql.com/doc/refman/8.4/en/column-indexes.html
	// Even though Namespace rarely uses its full 63-character capacity in practice,
	// MySQL calculates index length based on declared size, not actual content.
	// Therefore, keeping Name at varchar(128) is a safe upper bound.
	Name        string `gorm:"column:Name; not null; uniqueIndex:namespace_name; type:varchar(128);"` // Index improves performance of the List and Get queries
	DisplayName string `gorm:"column:DisplayName; not null"`
	Description string `gorm:"column:Description; type:longtext; not null"`
	// TODO(gkcalat): this is deprecated. Consider removing and adding data migration logic at the server startup.
	Parameters string         `gorm:"column:Parameters; type:longtext;"`
	Status     PipelineStatus `gorm:"column:Status; not null;"`
	// TODO(gkcalat): this is deprecated. Consider removing and adding data migration logic at the server startup.
	DefaultVersionId string `gorm:"column:DefaultVersionId;"` // deprecated
	// Namespace is restricted to varchar(63) due to Kubernetes' naming constraints:
	// https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#dns-label-names
	Namespace string `gorm:"column:Namespace; uniqueIndex:namespace_name; type:varchar(63);"`
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
	"id":           "UUID",        // v1beta1 API
	"pipeline_id":  "UUID",        // v2beta1 API
	"name":         "Name",        // v1beta1 API
	"display_name": "DisplayName", // v2beta1 API
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

// TableName overrides GORM's table name inference.
func (Pipeline) TableName() string {
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
	case "DisplayName":
		return p.DisplayName
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
