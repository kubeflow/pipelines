// Code generated by go-swagger; DO NOT EDIT.

package experiment_model

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	strfmt "github.com/go-openapi/strfmt"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/swag"
)

// APIResourceReference api resource reference
// swagger:model apiResourceReference
type APIResourceReference struct {

	// key
	Key *APIResourceKey `json:"key,omitempty"`

	// The name of the resource that referred to.
	Name string `json:"name,omitempty"`

	// Required field. The relationship from referred resource to the object.
	Relationship APIRelationship `json:"relationship,omitempty"`
}

// Validate validates this api resource reference
func (m *APIResourceReference) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateKey(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateRelationship(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *APIResourceReference) validateKey(formats strfmt.Registry) error {

	if swag.IsZero(m.Key) { // not required
		return nil
	}

	if m.Key != nil {
		if err := m.Key.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("key")
			}
			return err
		}
	}

	return nil
}

func (m *APIResourceReference) validateRelationship(formats strfmt.Registry) error {

	if swag.IsZero(m.Relationship) { // not required
		return nil
	}

	if err := m.Relationship.Validate(formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("relationship")
		}
		return err
	}

	return nil
}

// MarshalBinary interface implementation
func (m *APIResourceReference) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *APIResourceReference) UnmarshalBinary(b []byte) error {
	var res APIResourceReference
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
