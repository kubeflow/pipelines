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

type Experiment struct {
	UUID                  string       `gorm:"column:UUID; not null; primary_key;"`
	Name                  string       `gorm:"column:Name; not null; unique_index:idx_name_namespace;"`
	Description           string       `gorm:"column:Description; not null;"`
	CreatedAtInSec        int64        `gorm:"column:CreatedAtInSec; not null;"`
	LastRunCreatedAtInSec int64        `gorm:"column:LastRunCreatedAtInSec; not null;"`
	Namespace             string       `gorm:"column:Namespace; not null; unique_index:idx_name_namespace;"`
	StorageState          StorageState `gorm:"column:StorageState; not null;"`
}

// Note: Experiment.StorageState can have values: "STORAGE_STATE_UNSPECIFIED", "AVAILABLE" or "ARCHIVED"

func (e Experiment) GetValueOfPrimaryKey() string {
	return e.UUID
}

func GetExperimentTablePrimaryKeyColumn() string {
	return "UUID"
}

// PrimaryKeyColumnName returns the primary key for model Experiment.
func (e *Experiment) PrimaryKeyColumnName() string {
	return "UUID"
}

// DefaultSortField returns the default sorting field for model Experiment.
func (e *Experiment) DefaultSortField() string {
	return "CreatedAtInSec"
}

var experimentAPIToModelFieldMap = map[string]string{
	"id":                  "UUID", // v1beta1 API
	"experiment_id":       "UUID", // v2beta1 API
	"name":                "Name", // v1beta1 API
	"display_name":        "Name", // v2beta1 API
	"created_at":          "CreatedAtInSec",
	"last_run_created_at": "LastRunCreatedAtInSec", // v2beta1 API
	"description":         "Description",
	"namespace":           "Namespace", // v2beta1 API
	"storage_state":       "StorageState",
}

// APIToModelFieldMap returns a map from API names to field names for model
// Experiment.
func (e *Experiment) APIToModelFieldMap() map[string]string {
	return experimentAPIToModelFieldMap
}

// GetModelName returns table name used as sort field prefix.
func (e *Experiment) GetModelName() string {
	return "experiments"
}

func (e *Experiment) GetField(name string) (string, bool) {
	if field, ok := experimentAPIToModelFieldMap[name]; ok {
		return field, true
	}
	return "", false
}

func (e *Experiment) GetFieldValue(name string) interface{} {
	switch name {
	case "UUID":
		return e.UUID
	case "Name":
		return e.Name
	case "CreatedAtInSec":
		return e.CreatedAtInSec
	case "LastRunCreatedAtInSec":
		return e.LastRunCreatedAtInSec
	case "Description":
		return e.Description
	case "Namespace":
		return e.Namespace
	case "StorageState":
		return e.StorageState
	default:
		return nil
	}
}

func (e *Experiment) GetSortByFieldPrefix(name string) string {
	return "experiments."
}

func (e *Experiment) GetKeyFieldPrefix() string {
	return "experiments."
}
