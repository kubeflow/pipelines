// Code generated by go-swagger; DO NOT EDIT.

package run_model

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	strfmt "github.com/go-openapi/strfmt"

	"github.com/go-openapi/swag"
)

// V2beta1PipelineVersionReference Reference to an existing pipeline version.
// swagger:model v2beta1PipelineVersionReference
type V2beta1PipelineVersionReference struct {

	// Input. Required. Unique ID of the parent pipeline.
	PipelineID string `json:"pipeline_id,omitempty"`

	// Input. Optional. Unique ID of an existing pipeline version. If unset, the latest pipeline version is used.
	PipelineVersionID string `json:"pipeline_version_id,omitempty"`
}

// Validate validates this v2beta1 pipeline version reference
func (m *V2beta1PipelineVersionReference) Validate(formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *V2beta1PipelineVersionReference) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *V2beta1PipelineVersionReference) UnmarshalBinary(b []byte) error {
	var res V2beta1PipelineVersionReference
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
