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

// Package validation provides validation logic for API server fields,
// including field length checks based on logical maximums defined for migration safety
// and backend schema compatibility.
package validation

import (
	"fmt"
	"reflect"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

// ColLenSpec describes the logical max length we enforce during migration.
// Model is the Go struct used by GORM; Field is the Go struct field name (NOT the DB column name).
// Max is the allowed character length after upgrade.
type ColLenSpec struct {
	Model interface{}
	Field string // Go struct field name
	Max   int
}

var LengthSpecs = []ColLenSpec{
	{Model: &model.DefaultExperiment{}, Field: "DefaultExperimentId", Max: 191},

	{Model: &model.Experiment{}, Field: "UUID", Max: 191},
	{Model: &model.Experiment{}, Field: "Name", Max: 128},
	{Model: &model.Experiment{}, Field: "Namespace", Max: 63},

	{Model: &model.Job{}, Field: "UUID", Max: 191},

	{Model: &model.PipelineVersion{}, Field: "UUID", Max: 191},
	{Model: &model.PipelineVersion{}, Field: "Name", Max: 127},
	{Model: &model.PipelineVersion{}, Field: "PipelineId", Max: 64},

	{Model: &model.Pipeline{}, Field: "UUID", Max: 64},
	{Model: &model.Pipeline{}, Field: "Name", Max: 128},
	{Model: &model.Pipeline{}, Field: "Namespace", Max: 63}, // gorm v1 has this size limit

	{Model: &model.ResourceReference{}, Field: "ResourceUUID", Max: 191},
	{Model: &model.ResourceReference{}, Field: "ReferenceUUID", Max: 191},

	{Model: &model.Run{}, Field: "UUID", Max: 191},
	{Model: &model.Run{}, Field: "Namespace", Max: 63},
	{Model: &model.Run{}, Field: "ExperimentId", Max: 64},
	{Model: &model.Run{}, Field: "Conditions", Max: 125},

	{Model: &model.RunMetric{}, Field: "RunUUID", Max: 191},
	{Model: &model.RunMetric{}, Field: "NodeID", Max: 191},
	{Model: &model.RunMetric{}, Field: "Name", Max: 191},

	{Model: &model.Task{}, Field: "UUID", Max: 191},
	// Note: struct field is RunId, column is RunUUID.
	{Model: &model.Task{}, Field: "RunId", Max: 191},
}

var fieldMaxLenMap map[string]int

func init() {
	fieldMaxLenMap = make(map[string]int, len(LengthSpecs))
	for _, spec := range LengthSpecs {
		typ := reflect.TypeOf(spec.Model).Elem().Name()
		fieldMaxLenMap[typ+"."+spec.Field] = spec.Max
	}
}

func getMaxLength(modelName, field string) (int, bool) {
	v, ok := fieldMaxLenMap[modelName+"."+field]
	return v, ok
}

// ValidateFieldLength validates a single string field against LengthSpecs.
// Primarily used in pipeline_upload_server for request-level validation of raw input values,
// especially for fields like Namespace that may be modified before model construction.
// Use this for targeted, field-specific validation. For full-model validation,
// prefer ValidateModel().
func ValidateFieldLength(modelName, fieldName, value string) error {
	maxLen, ok := getMaxLength(modelName, fieldName)
	if !ok {
		return util.NewInternalServerError(
			fmt.Errorf("length spec missing for %s.%s", modelName, fieldName),
			"Length spec missing for %s.%s", modelName, fieldName,
		)
	}

	if len(value) > maxLen {
		return util.NewInvalidInputError("%s.%s length cannot exceed %d", modelName, fieldName, maxLen)
	}
	return nil
}

// ValidateModel dynamically validates all fields of the given model that are defined in LengthSpecs.
// It uses reflection to extract field values and compare against the max length.
// Returns the first encountered InvalidInputError or InternalServerError.
func ValidateModel(model interface{}) error {
	val := reflect.ValueOf(model)
	if val.Kind() != reflect.Ptr || val.IsNil() {
		return util.NewInternalServerError(
			fmt.Errorf("model must be a non-nil pointer"),
			"Model passed to ValidateModel must be a non-nil pointer",
		)
	}

	elem := val.Elem()
	typ := elem.Type()
	modelName := typ.Name()

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		valueField := elem.Field(i)

		// Only validate string fields
		if field.Type.Kind() != reflect.String {
			continue
		}

		maxLen, ok := getMaxLength(modelName, field.Name)
		if !ok {
			continue
		}

		strVal := valueField.String()
		if len(strVal) > maxLen {
			return util.NewInvalidInputError("%s.%s length cannot exceed %d", modelName, field.Name, maxLen)
		}
	}

	return nil
}
