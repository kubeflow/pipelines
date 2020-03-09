// Copyright 2020 Google LLC
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

package pipeline_upload_model

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"strconv"

	strfmt "github.com/go-openapi/strfmt"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/swag"
	"github.com/go-openapi/validate"
)

// APIPipelineVersion api pipeline version
// swagger:model apiPipelineVersion
type APIPipelineVersion struct {

	// Input. Optional. Pipeline version code source.
	CodeSourceURL string `json:"code_source_url,omitempty"`

	// Output. The time this pipeline version is created.
	// Format: date-time
	CreatedAt strfmt.DateTime `json:"created_at,omitempty"`

	// Output. Unique version ID. Generated by API server.
	ID string `json:"id,omitempty"`

	// Optional input field. Version name provided by user.
	Name string `json:"name,omitempty"`

	// Input. Required. Pipeline version package url.
	// Whe calling CreatePipelineVersion API method, need to provide one package
	// file location.
	PackageURL *APIURL `json:"package_url,omitempty"`

	// Output. The input parameters for this pipeline.
	Parameters []*APIParameter `json:"parameters"`

	// Input. Required. E.g., specify which pipeline this pipeline version belongs
	// to.
	ResourceReferences []*APIResourceReference `json:"resource_references"`
}

// Validate validates this api pipeline version
func (m *APIPipelineVersion) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateCreatedAt(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validatePackageURL(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateParameters(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateResourceReferences(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *APIPipelineVersion) validateCreatedAt(formats strfmt.Registry) error {

	if swag.IsZero(m.CreatedAt) { // not required
		return nil
	}

	if err := validate.FormatOf("created_at", "body", "date-time", m.CreatedAt.String(), formats); err != nil {
		return err
	}

	return nil
}

func (m *APIPipelineVersion) validatePackageURL(formats strfmt.Registry) error {

	if swag.IsZero(m.PackageURL) { // not required
		return nil
	}

	if m.PackageURL != nil {
		if err := m.PackageURL.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("package_url")
			}
			return err
		}
	}

	return nil
}

func (m *APIPipelineVersion) validateParameters(formats strfmt.Registry) error {

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

func (m *APIPipelineVersion) validateResourceReferences(formats strfmt.Registry) error {

	if swag.IsZero(m.ResourceReferences) { // not required
		return nil
	}

	for i := 0; i < len(m.ResourceReferences); i++ {
		if swag.IsZero(m.ResourceReferences[i]) { // not required
			continue
		}

		if m.ResourceReferences[i] != nil {
			if err := m.ResourceReferences[i].Validate(formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("resource_references" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

// MarshalBinary interface implementation
func (m *APIPipelineVersion) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *APIPipelineVersion) UnmarshalBinary(b []byte) error {
	var res APIPipelineVersion
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
