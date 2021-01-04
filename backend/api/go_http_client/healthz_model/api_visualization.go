// Copyright 2021 Google LLC
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

package healthz_model

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// APIVisualization api visualization
//
// swagger:model apiVisualization
type APIVisualization struct {

	// Variables to be used during generation of a visualization.
	// This should be provided as a JSON string.
	// This is required when creating the pipeline through CreateVisualization
	// API.
	Arguments string `json:"arguments,omitempty"`

	// In case any error happens when generating visualizations, only
	// visualization ID and the error message are returned. Client has the
	// flexibility of choosing how to handle the error.
	Error string `json:"error,omitempty"`

	// Output. Generated visualization html.
	HTML string `json:"html,omitempty"`

	// Path pattern of input data to be used during generation of visualizations.
	// This is required when creating the pipeline through CreateVisualization
	// API.
	Source string `json:"source,omitempty"`

	// type
	Type APIVisualizationType `json:"type,omitempty"`
}

// Validate validates this api visualization
func (m *APIVisualization) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateType(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *APIVisualization) validateType(formats strfmt.Registry) error {

	if swag.IsZero(m.Type) { // not required
		return nil
	}

	if err := m.Type.Validate(formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("type")
		}
		return err
	}

	return nil
}

// MarshalBinary interface implementation
func (m *APIVisualization) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *APIVisualization) UnmarshalBinary(b []byte) error {
	var res APIVisualization
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
