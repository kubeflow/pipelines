// Copyright 2019 Google LLC
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

// Code generated by go-swagger; DO NOT EDIT.

package run_model

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	strfmt "github.com/go-openapi/strfmt"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/swag"
)

// APIRunDetail api run detail
// swagger:model apiRunDetail
type APIRunDetail struct {

	// pipeline runtime
	PipelineRuntime *APIPipelineRuntime `json:"pipeline_runtime,omitempty"`

	// run
	Run *APIRun `json:"run,omitempty"`
}

// Validate validates this api run detail
func (m *APIRunDetail) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validatePipelineRuntime(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateRun(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *APIRunDetail) validatePipelineRuntime(formats strfmt.Registry) error {

	if swag.IsZero(m.PipelineRuntime) { // not required
		return nil
	}

	if m.PipelineRuntime != nil {
		if err := m.PipelineRuntime.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("pipeline_runtime")
			}
			return err
		}
	}

	return nil
}

func (m *APIRunDetail) validateRun(formats strfmt.Registry) error {

	if swag.IsZero(m.Run) { // not required
		return nil
	}

	if m.Run != nil {
		if err := m.Run.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("run")
			}
			return err
		}
	}

	return nil
}

// MarshalBinary interface implementation
func (m *APIRunDetail) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *APIRunDetail) UnmarshalBinary(b []byte) error {
	var res APIRunDetail
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
