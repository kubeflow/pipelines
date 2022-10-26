// Code generated by go-swagger; DO NOT EDIT.

package pipeline_model

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	strfmt "github.com/go-openapi/strfmt"

	"github.com/go-openapi/swag"
)

// V1beta1GetTemplateResponse v1beta1 get template response
// swagger:model v1beta1GetTemplateResponse
type V1beta1GetTemplateResponse struct {

	// The template of the pipeline specified in a GetTemplate request, or of a
	// pipeline version specified in a GetPipelinesVersionTemplate request.
	Template string `json:"template,omitempty"`
}

// Validate validates this v1beta1 get template response
func (m *V1beta1GetTemplateResponse) Validate(formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *V1beta1GetTemplateResponse) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *V1beta1GetTemplateResponse) UnmarshalBinary(b []byte) error {
	var res V1beta1GetTemplateResponse
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
