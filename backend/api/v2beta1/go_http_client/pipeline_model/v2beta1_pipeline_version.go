// Code generated by go-swagger; DO NOT EDIT.

package pipeline_model

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	strfmt "github.com/go-openapi/strfmt"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/swag"
	"github.com/go-openapi/validate"
)

// V2beta1PipelineVersion v2beta1 pipeline version
// swagger:model v2beta1PipelineVersion
type V2beta1PipelineVersion struct {

	// Output. Creation time of the pipeline version.
	// Format: date-time
	CreatedAt strfmt.DateTime `json:"created_at,omitempty"`

	// Optional input field. Short description of the pipeline version.
	Description string `json:"description,omitempty"`

	// Required input field. Pipeline version name provided by user.
	DisplayName string `json:"display_name,omitempty"`

	// In case any error happens retrieving a pipeline version field, only
	// pipeline ID, pipeline version ID, and the error message are returned.
	// Client has the flexibility of choosing how to handle the error.
	// This is especially useful during List() calls.
	Error *GooglerpcStatus `json:"error,omitempty"`

	// Required input field. Pipeline version package url.
	// When calling CreatePipelineVersion, one needs to provide
	// one package file location.
	PackageURL *V2beta1URL `json:"package_url,omitempty"`

	// Required input field. Unique ID of the parent pipeline.
	PipelineID string `json:"pipeline_id,omitempty"`

	// Required input field. Specifies the pipeline spec for the pipeline version.
	PipelineSpec *ProtobufStruct `json:"pipeline_spec,omitempty"`

	// Output. Unique pipeline version ID. Generated by API server.
	PipelineVersionID string `json:"pipeline_version_id,omitempty"`
}

// Validate validates this v2beta1 pipeline version
func (m *V2beta1PipelineVersion) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateCreatedAt(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateError(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validatePackageURL(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validatePipelineSpec(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *V2beta1PipelineVersion) validateCreatedAt(formats strfmt.Registry) error {

	if swag.IsZero(m.CreatedAt) { // not required
		return nil
	}

	if err := validate.FormatOf("created_at", "body", "date-time", m.CreatedAt.String(), formats); err != nil {
		return err
	}

	return nil
}

func (m *V2beta1PipelineVersion) validateError(formats strfmt.Registry) error {

	if swag.IsZero(m.Error) { // not required
		return nil
	}

	if m.Error != nil {
		if err := m.Error.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("error")
			}
			return err
		}
	}

	return nil
}

func (m *V2beta1PipelineVersion) validatePackageURL(formats strfmt.Registry) error {

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

func (m *V2beta1PipelineVersion) validatePipelineSpec(formats strfmt.Registry) error {

	if swag.IsZero(m.PipelineSpec) { // not required
		return nil
	}

	if m.PipelineSpec != nil {
		if err := m.PipelineSpec.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("pipeline_spec")
			}
			return err
		}
	}

	return nil
}

// MarshalBinary interface implementation
func (m *V2beta1PipelineVersion) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *V2beta1PipelineVersion) UnmarshalBinary(b []byte) error {
	var res V2beta1PipelineVersion
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
