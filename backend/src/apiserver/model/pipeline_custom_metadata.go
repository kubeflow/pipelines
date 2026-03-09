// Copyright 2025 The Kubeflow Authors
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

// PipelineTag stores a single user-defined tag (key-value pair) associated with a pipeline.
// The composite primary key (PipelineId, TagKey) ensures uniqueness per pipeline.
// Both TagKey and TagValue are limited to 20 characters.
type PipelineTag struct {
	// PipelineID references the parent pipeline.
	PipelineID string `gorm:"column:PipelineId; not null; primaryKey; type:varchar(64);"`
	// TagKey is the tag key, limited to 20 characters.
	TagKey string `gorm:"column:TagKey; not null; primaryKey; type:varchar(20);"`
	// TagValue is the tag value, limited to 20 characters.
	TagValue string `gorm:"column:TagValue; type:varchar(20);"`

	// Pipeline establishes the belongs-to relationship for cascading deletes.
	Pipeline Pipeline `gorm:"foreignKey:PipelineID; references:UUID; constraint:pipeline_tags_PipelineId_pipelines_UUID_foreign,OnDelete:CASCADE,OnUpdate:CASCADE"`
}

// TableName overrides GORM's table name inference.
func (PipelineTag) TableName() string {
	return "pipeline_tags"
}

// PipelineVersionTag stores a single user-defined tag (key-value pair) associated with a pipeline version.
// The composite primary key (PipelineVersionId, TagKey) ensures uniqueness per pipeline version.
// Both TagKey and TagValue are limited to 20 characters.
type PipelineVersionTag struct {
	// PipelineVersionID references the parent pipeline version.
	PipelineVersionID string `gorm:"column:PipelineVersionId; not null; primaryKey; type:varchar(191);"`
	// TagKey is the tag key, limited to 20 characters.
	TagKey string `gorm:"column:TagKey; not null; primaryKey; type:varchar(20);"`
	// TagValue is the tag value, limited to 20 characters.
	TagValue string `gorm:"column:TagValue; type:varchar(20);"`

	// PipelineVersion establishes the belongs-to relationship for cascading deletes.
	PipelineVersion PipelineVersion `gorm:"foreignKey:PipelineVersionID; references:UUID; constraint:pv_tags_PipelineVersionId_pv_UUID_fk,OnDelete:CASCADE,OnUpdate:CASCADE"`
}

// TableName overrides GORM's table name inference.
func (PipelineVersionTag) TableName() string {
	return "pipeline_version_tags"
}
