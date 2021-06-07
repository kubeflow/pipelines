// Code generated by go-swagger; DO NOT EDIT.

package pipeline_model

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"strconv"

	strfmt "github.com/go-openapi/strfmt"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/swag"
)

// APIListPipelinesResponse api list pipelines response
// swagger:model apiListPipelinesResponse
type APIListPipelinesResponse struct {

	// The token to list the next page of pipelines.
	NextPageToken string `json:"next_page_token,omitempty"`

	// pipelines
	Pipelines []*APIPipeline `json:"pipelines"`

	// The total number of pipelines for the given query.
	TotalSize int32 `json:"total_size,omitempty"`
}

// Validate validates this api list pipelines response
func (m *APIListPipelinesResponse) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validatePipelines(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *APIListPipelinesResponse) validatePipelines(formats strfmt.Registry) error {

	if swag.IsZero(m.Pipelines) { // not required
		return nil
	}

	for i := 0; i < len(m.Pipelines); i++ {
		if swag.IsZero(m.Pipelines[i]) { // not required
			continue
		}

		if m.Pipelines[i] != nil {
			if err := m.Pipelines[i].Validate(formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("pipelines" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

// MarshalBinary interface implementation
func (m *APIListPipelinesResponse) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *APIListPipelinesResponse) UnmarshalBinary(b []byte) error {
	var res APIListPipelinesResponse
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
