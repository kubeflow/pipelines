// Code generated by go-swagger; DO NOT EDIT.

package job_model

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	strfmt "github.com/go-openapi/strfmt"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/swag"
)

// APIJobDetail api job detail
// swagger:model apiJobDetail
type APIJobDetail struct {

	// job
	Job *APIJob `json:"job,omitempty"`

	// workflow
	Workflow string `json:"workflow,omitempty"`
}

// Validate validates this api job detail
func (m *APIJobDetail) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateJob(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *APIJobDetail) validateJob(formats strfmt.Registry) error {

	if swag.IsZero(m.Job) { // not required
		return nil
	}

	if m.Job != nil {
		if err := m.Job.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("job")
			}
			return err
		}
	}

	return nil
}

// MarshalBinary interface implementation
func (m *APIJobDetail) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *APIJobDetail) UnmarshalBinary(b []byte) error {
	var res APIJobDetail
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
