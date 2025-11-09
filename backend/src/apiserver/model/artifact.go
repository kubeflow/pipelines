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

// Package model contains data models for the KFP API server.
package model

import apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"

type ArtifactType apiv2beta1.Artifact_ArtifactType

// Artifact represents an artifact in the KFP system
type Artifact struct {
	UUID            string       `gorm:"column:UUID; not null; primaryKey; type:varchar(191);"`
	Namespace       string       `gorm:"column:Namespace; not null; type:varchar(63); index:idx_type_namespace,priority:1;"`
	Type            ArtifactType `gorm:"column:Type; default:null; index:idx_type_namespace,priority:2;"`
	URI             *string      `gorm:"column:URI; type:text;"`
	Name            string       `gorm:"column:Name; type:varchar(128); default:null;"`
	Description     string       `gorm:"column:Description; type:text; default:null;"`
	CreatedAtInSec  int64        `gorm:"column:CreatedAtInSec; not null; default:0; index:idx_artifact_created_timestamp;"`
	LastUpdateInSec int64        `gorm:"column:LastUpdateInSec; not null; default:0; index:idx_artifact_last_update_timestamp;"`
	Metadata        JSONData     `gorm:"column:Metadata; type:json; default:null;"`
	// Used primarily for metrics
	NumberValue *float64 `gorm:"column:NumberValue; default:null;"`
}

func (a Artifact) PrimaryKeyColumnName() string {
	return "UUID"
}

func (a Artifact) DefaultSortField() string {
	return "CreatedAtInSec"
}

func (a Artifact) APIToModelFieldMap() map[string]string {
	return artifactAPIToModelFieldMap
}

func (a Artifact) GetModelName() string {
	return "artifacts"
}

func (a Artifact) GetSortByFieldPrefix(s string) string {
	return "artifacts."
}

func (a Artifact) GetKeyFieldPrefix() string {
	return "artifacts."
}

var artifactAPIToModelFieldMap = map[string]string{
	"artifact_id":  "UUID",
	"id":           "UUID",
	"namespace":    "Namespace",
	"type":         "Type",
	"uri":          "URI",
	"name":         "Name",
	"description":  "Description",
	"created_at":   "CreatedAtInSec",
	"last_update":  "LastUpdateInSec",
	"metadata":     "Metadata",
	"number_value": "NumberValue",
}

func (a Artifact) GetField(name string) (string, bool) {
	if field, ok := artifactAPIToModelFieldMap[name]; ok {
		return field, true
	}
	return "", false
}

func (a Artifact) GetFieldValue(name string) interface{} {
	switch name {
	case "UUID":
		return a.UUID
	case "Namespace":
		return a.Namespace
	case "Type":
		return a.Type
	case "URI":
		return a.URI
	case "Name":
		return a.Name
	case "Description":
		return a.Description
	case "CreatedAtInSec":
		return a.CreatedAtInSec
	case "LastUpdateInSec":
		return a.LastUpdateInSec
	case "Metadata":
		return a.Metadata
	case "NumberValue":
		return a.NumberValue
	default:
		return nil
	}
}
