// Copyright 2025 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package model

import apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"

// IOType represents the I/O relationship type
type IOType apiv2beta1.IOType

// ArtifactTask represents the relationship between artifacts and tasks
type ArtifactTask struct {
	UUID        string   `gorm:"column:UUID; not null; primaryKey; type:varchar(191);"`
	ArtifactID  string   `gorm:"column:ArtifactID; not null; type:varchar(191); index:idx_link_artifact_id; uniqueIndex:UniqueLink,priority:1;"`
	TaskID      string   `gorm:"column:TaskID; not null; type:varchar(191); index:idx_link_task_id; uniqueIndex:UniqueLink,priority:2;"`
	Type        IOType   `gorm:"column:Type; not null; uniqueIndex:UniqueLink,priority:3;"`
	RunUUID     string   `gorm:"column:RunUUID; not null; type:varchar(191); index:idx_link_run_id;"`
	Producer    JSONData `gorm:"column:Producer; type:json; default:null;"`
	ArtifactKey string   `gorm:"column:ArtifactKey; not null; type:varchar(191); default:'';"`

	// Relationships
	Artifact Artifact `gorm:"foreignKey:ArtifactID;references:UUID;constraint:fk_artifact_tasks_artifacts,OnDelete:CASCADE,OnUpdate:CASCADE;"`
	Task     Task     `gorm:"foreignKey:TaskID;references:UUID;constraint:fk_artifact_tasks_tasks,OnDelete:CASCADE,OnUpdate:CASCADE;"`
	Run      Run      `gorm:"foreignKey:RunUUID;references:UUID;constraint:fk_artifact_tasks_runs,OnDelete:CASCADE,OnUpdate:CASCADE;"`
}

func (at ArtifactTask) PrimaryKeyColumnName() string {
	return "UUID"
}

func (at ArtifactTask) DefaultSortField() string {
	return "UUID"
}

func (at ArtifactTask) APIToModelFieldMap() map[string]string {
	return artifactTaskAPIToModelFieldMap
}

func (at ArtifactTask) GetModelName() string {
	return "artifact_tasks"
}

func (at ArtifactTask) GetSortByFieldPrefix(s string) string {
	return "artifact_tasks."
}

func (at ArtifactTask) GetKeyFieldPrefix() string {
	return "artifact_tasks."
}

var artifactTaskAPIToModelFieldMap = map[string]string{
	"id":          "UUID",
	"artifact_id": "ArtifactID",
	"task_id":     "TaskID",
	"type":        "Type",
	"run_id":      "RunUUID",
	"producer":    "Producer",
	"key":         "ArtifactKey",
}

func (at ArtifactTask) GetField(name string) (string, bool) {
	if field, ok := artifactTaskAPIToModelFieldMap[name]; ok {
		return field, true
	}
	return "", false
}

func (at ArtifactTask) GetFieldValue(name string) interface{} {
	switch name {
	case "UUID":
		return at.UUID
	case "ArtifactID":
		return at.ArtifactID
	case "TaskID":
		return at.TaskID
	case "Type":
		return at.Type
	case "RunUUID":
		return at.RunUUID
	case "Producer":
		return at.Producer
	case "ArtifactKey":
		return at.ArtifactKey
	default:
		return nil
	}
}
