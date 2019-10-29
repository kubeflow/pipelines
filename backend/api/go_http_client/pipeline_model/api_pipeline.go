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

package pipeline_model

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"strconv"

	strfmt "github.com/go-openapi/strfmt"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/swag"
	"github.com/go-openapi/validate"
)

// APIPipeline api pipeline
// swagger:model apiPipeline
type APIPipeline struct {

	// Output. The time this pipeline is created.
	// Format: date-time
	CreatedAt strfmt.DateTime `json:"created_at,omitempty"`

	// Output only. The default version of the pipeline. As of now, the latest
	// version is used as default. (In the future, if desired by customers, we
	// can allow them to set default version.)
	// Read Only: true
	DefaultVersion *APIPipelineVersion `json:"default_version,omitempty"`

	// Optional input field. Describing the purpose of the job.
	Description string `json:"description,omitempty"`

	// In case any error happens retrieving a pipeline field, only pipeline ID
	// and the error message is returned. Client has the flexibility of choosing
	// how to handle error. This is especially useful during listing call.
	Error string `json:"error,omitempty"`

	// Output. Unique pipeline ID. Generated by API server.
	ID string `json:"id,omitempty"`

	// Optional input field. Pipeline name provided by user. If not specified,
	// file name is used as pipeline name.
	Name string `json:"name,omitempty"`

	// Output. The input parameters for this pipeline.
	// TODO(jingzhang36): replace this parameters field with the parameters field
	// inside PipelineVersion when all usage of the former has been changed to use
	// the latter.
	Parameters []*APIParameter `json:"parameters"`

	// The URL to the source of the pipeline. This is required when creating the
	// pipeine through CreatePipeline API.
	// TODO(jingzhang36): replace this url field with the code_source_urls field
	// inside PipelineVersion when all usage of the former has been changed to use
	// the latter.
	URL *APIURL `json:"url,omitempty"`
}

// Validate validates this api pipeline
func (m *APIPipeline) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateCreatedAt(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateDefaultVersion(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateParameters(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateURL(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *APIPipeline) validateCreatedAt(formats strfmt.Registry) error {

	if swag.IsZero(m.CreatedAt) { // not required
		return nil
	}

	if err := validate.FormatOf("created_at", "body", "date-time", m.CreatedAt.String(), formats); err != nil {
		return err
	}

	return nil
}

func (m *APIPipeline) validateDefaultVersion(formats strfmt.Registry) error {

	if swag.IsZero(m.DefaultVersion) { // not required
		return nil
	}

	if m.DefaultVersion != nil {
		if err := m.DefaultVersion.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("default_version")
			}
			return err
		}
	}

	return nil
}

func (m *APIPipeline) validateParameters(formats strfmt.Registry) error {

	if swag.IsZero(m.Parameters) { // not required
		return nil
	}

	for i := 0; i < len(m.Parameters); i++ {
		if swag.IsZero(m.Parameters[i]) { // not required
			continue
		}

		if m.Parameters[i] != nil {
			if err := m.Parameters[i].Validate(formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("parameters" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

func (m *APIPipeline) validateURL(formats strfmt.Registry) error {

	if swag.IsZero(m.URL) { // not required
		return nil
	}

	if m.URL != nil {
		if err := m.URL.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("url")
			}
			return err
		}
	}

	return nil
}

// MarshalBinary interface implementation
func (m *APIPipeline) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *APIPipeline) UnmarshalBinary(b []byte) error {
	var res APIPipeline
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
