// Copyright 2018 The Kubeflow Authors
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

import (
	"database/sql/driver"
	"encoding/json"

	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// JSONSlice represents JSON list data stored in database columns
type JSONSlice []interface{}

func (j *JSONSlice) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements sql.Scanner interface for JSONSlice
func (j *JSONSlice) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, j)
	case string:
		return json.Unmarshal([]byte(v), j)
	default:
		return nil
	}
}

// JSONData represents JSON struct data stored in database columns
type JSONData map[string]interface{}

// Scan implements sql.Scanner interface for JSONData
func (j *JSONData) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, j)
	case string:
		return json.Unmarshal([]byte(v), j)
	default:
		return nil
	}
}

// Value implements driver.Valuer interface for JSONData
func (j *JSONData) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// PodNames represents JSON array of pod names
type PodNames []string

// Scan implements sql.Scanner interface for PodNames
func (p *PodNames) Scan(value interface{}) error {
	if value == nil {
		*p = nil
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, p)
	case string:
		return json.Unmarshal([]byte(v), p)
	default:
		return nil
	}
}

// Value implements driver.Valuer interface for PodNames
func (p *PodNames) Value() (driver.Value, error) {
	if p == nil {
		return nil, nil
	}
	return json.Marshal(p)
}

// TaskArtifactHydrated holds hydrated artifact info per task (not stored in DB)
type TaskArtifactHydrated struct {
	Value    *Artifact
	Producer *IOProducer
	Key      string
	Type     apiv2beta1.IOType
}
type IOProducer struct {
	TaskName  string
	Iteration *int64
}

type TaskType apiv2beta1.PipelineTaskDetail_TaskType
type TaskStatus apiv2beta1.PipelineTaskDetail_TaskState

type Task struct {
	UUID             string     `gorm:"column:UUID; not null; primaryKey; type:varchar(191);"`
	Namespace        string     `gorm:"column:Namespace; not null; type:varchar(63);"`
	RunUUID          string     `gorm:"column:RunUUID; type:varchar(191); not null; index:idx_parent_run,priority:1;"`
	Run              Run        `gorm:"foreignKey:RunUUID;references:UUID;constraint:tasks_RunUUID_run_details_UUID_foreign,OnDelete:CASCADE,OnUpdate:CASCADE;"`
	Pods             JSONSlice  `gorm:"column:pods; not null; type:json;"`
	CreatedAtInSec   int64      `gorm:"column:CreatedAtInSec; not null; index:idx_task_created_timestamp;"`
	StartedInSec     int64      `gorm:"column:StartedInSec; default:0; index:idx_task_started_timestamp;"`
	FinishedInSec    int64      `gorm:"column:FinishedInSec; default:0; index:idx_task_finished_timestamp;"`
	Fingerprint      string     `gorm:"column:Fingerprint; not null; type:varchar(255);"`
	Name             string     `gorm:"column:Name; type:varchar(128); default:null;"`
	DisplayName      string     `gorm:"column:DisplayName; type:varchar(128); default:null;"`
	ParentTaskUUID   *string    `gorm:"column:ParentTaskUUID; type:varchar(191); default:null; index:idx_parent_task_uuid; index:idx_parent_run,priority:2;"`
	ParentTask       *Task      `gorm:"foreignKey:ParentTaskUUID;references:UUID;constraint:fk_tasks_parent_task,OnDelete:CASCADE,OnUpdate:CASCADE;"`
	State            TaskStatus `gorm:"column:State; not null;"`
	StatusMetadata   JSONData   `gorm:"column:StatusMetadata; type:json; default:null;"`
	StateHistory     JSONSlice  `gorm:"column:StateHistory; type:json;"`
	InputParameters  JSONSlice  `gorm:"column:InputParameters; type:json;"`
	OutputParameters JSONSlice  `gorm:"column:OutputParameters; type:json;"`
	Type             TaskType   `gorm:"column:Type; not null; index:idx_task_type;"`
	TypeAttrs        JSONData   `gorm:"column:TypeAttrs; not null; type:json;"`
	ScopePath        string     `gorm:"column:ScopePath; type:text; default:null;"`

	// Transient fields populated during hydration (not stored in DB)
	InputArtifactsHydrated  []TaskArtifactHydrated `gorm:"-"`
	OutputArtifactsHydrated []TaskArtifactHydrated `gorm:"-"`
}

func (t Task) ToString() string {
	task, err := json.Marshal(t)
	if err != nil {
		return ""
	} else {
		return string(task)
	}
}

func (t Task) PrimaryKeyColumnName() string {
	return "UUID"
}

func (t Task) DefaultSortField() string {
	return "CreatedAtInSec"
}

func (t Task) APIToModelFieldMap() map[string]string {
	return taskAPIToModelFieldMap
}

func (t Task) GetModelName() string {
	return "tasks"
}

func (t Task) GetSortByFieldPrefix(s string) string {
	return "tasks."
}

func (t Task) GetKeyFieldPrefix() string {
	return "tasks."
}

var taskAPIToModelFieldMap = map[string]string{
	"name":              "Name",
	"display_name":      "DisplayName",
	"task_id":           "UUID",
	"run_id":            "RunUUID",
	"pods":              "Pods",
	"cache_fingerprint": "Fingerprint",
	"create_time":       "CreatedAtInSec",
	"start_time":        "StartedInSec",
	"end_time":          "FinishedInSec",
	"status":            "State",
	"status_metadata":   "StatusMetadata",
	"state_history":     "StateHistory",
	"type":              "Type",
	"type_attributes":   "TypeAttrs",
	"parent_task_id":    "ParentTaskUUID",
	"inputs":            "InputParameters",
	"outputs":           "OutputParameters",
	"scope_path":        "ScopePath",
}

func (t Task) GetField(name string) (string, bool) {
	if field, ok := taskAPIToModelFieldMap[name]; ok {
		return field, true
	}
	return "", false
}

func (t Task) GetFieldValue(name string) interface{} {
	switch name {
	case "UUID":
		return t.UUID
	case "Namespace":
		return t.Namespace
	case "RunUUID":
		return t.RunUUID
	case "CreatedAtInSec":
		return t.CreatedAtInSec
	case "StartedInSec":
		return t.StartedInSec
	case "FinishedInSec":
		return t.FinishedInSec
	case "Fingerprint":
		return t.Fingerprint
	case "ParentTaskUUID":
		return t.ParentTaskUUID
	case "State":
		return t.State
	case "StatusMetadata":
		return t.StatusMetadata
	case "StateHistory":
		return t.StateHistory
	case "Name":
		return t.Name
	case "DisplayName":
		return t.DisplayName
	case "InputParameters":
		return t.InputParameters
	case "OutputParameters":
		return t.OutputParameters
	case "Type":
		return t.Type
	case "TypeAttrs":
		return t.TypeAttrs
	case "ScopePath":
		return t.ScopePath
	default:
		return nil
	}
}

// ProtoSliceToJSONSlice converts a slice of protobuf messages (e.g., []*MyMsg)
// into a model.JSONSlice (i.e., []interface{}).
func ProtoSliceToJSONSlice[T proto.Message](msgs []T) (JSONSlice, error) {
	out := make(JSONSlice, 0, len(msgs))
	for _, m := range msgs {
		var pm proto.Message = m
		if pm == nil {
			out = append(out, nil)
			continue
		}
		b, err := protojson.Marshal(m)
		if err != nil {
			return nil, err
		}
		var v interface{}
		if err := json.Unmarshal(b, &v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

// ProtoMessageToJSONData marshals a protobuf message into JSONData.
func ProtoMessageToJSONData(msg proto.Message) (JSONData, error) {
	if msg == nil {
		return nil, nil
	}
	b, err := protojson.Marshal(msg)
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// JSONSliceToProtoSlice converts a JSONSlice (i.e., []interface{}) into a slice
// of protobuf messages. Provide a constructor for T so we can allocate a new
// concrete message for each element.
func JSONSliceToProtoSlice[T proto.Message](in JSONSlice, newT func() T) ([]T, error) {
	if in == nil {
		return nil, nil
	}
	out := make([]T, 0, len(in))
	for _, v := range in {
		if v == nil {
			var zero T
			out = append(out, zero)
			continue
		}
		b, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		msg := newT()
		if err := protojson.Unmarshal(b, msg); err != nil {
			return nil, err
		}
		out = append(out, msg)
	}
	return out, nil
}

// JSONDataToProtoMessage unmarshals JSONData into a protobuf message instance.
// Provide a constructor for T so we can allocate a concrete message to fill.
func JSONDataToProtoMessage[T proto.Message](data JSONData, newT func() T) (T, error) {
	var zero T
	if data == nil {
		return zero, nil
	}
	b, err := json.Marshal(data)
	if err != nil {
		return zero, err
	}
	msg := newT()
	if err := protojson.Unmarshal(b, msg); err != nil {
		return zero, err
	}
	return msg, nil
}
