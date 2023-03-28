// Code generated by go-swagger; DO NOT EDIT.

package recurring_run_model

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"
	"encoding/json"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/validate"
)

// V2beta1RecurringRunStatus Output. The status of the recurring run.
//
// swagger:model v2beta1RecurringRunStatus
type V2beta1RecurringRunStatus string

func NewV2beta1RecurringRunStatus(value V2beta1RecurringRunStatus) *V2beta1RecurringRunStatus {
	return &value
}

// Pointer returns a pointer to a freshly-allocated V2beta1RecurringRunStatus.
func (m V2beta1RecurringRunStatus) Pointer() *V2beta1RecurringRunStatus {
	return &m
}

const (

	// V2beta1RecurringRunStatusSTATUSUNSPECIFIED captures enum value "STATUS_UNSPECIFIED"
	V2beta1RecurringRunStatusSTATUSUNSPECIFIED V2beta1RecurringRunStatus = "STATUS_UNSPECIFIED"

	// V2beta1RecurringRunStatusENABLED captures enum value "ENABLED"
	V2beta1RecurringRunStatusENABLED V2beta1RecurringRunStatus = "ENABLED"

	// V2beta1RecurringRunStatusDISABLED captures enum value "DISABLED"
	V2beta1RecurringRunStatusDISABLED V2beta1RecurringRunStatus = "DISABLED"
)

// for schema
var v2beta1RecurringRunStatusEnum []interface{}

func init() {
	var res []V2beta1RecurringRunStatus
	if err := json.Unmarshal([]byte(`["STATUS_UNSPECIFIED","ENABLED","DISABLED"]`), &res); err != nil {
		panic(err)
	}
	for _, v := range res {
		v2beta1RecurringRunStatusEnum = append(v2beta1RecurringRunStatusEnum, v)
	}
}

func (m V2beta1RecurringRunStatus) validateV2beta1RecurringRunStatusEnum(path, location string, value V2beta1RecurringRunStatus) error {
	if err := validate.EnumCase(path, location, value, v2beta1RecurringRunStatusEnum, true); err != nil {
		return err
	}
	return nil
}

// Validate validates this v2beta1 recurring run status
func (m V2beta1RecurringRunStatus) Validate(formats strfmt.Registry) error {
	var res []error

	// value enum
	if err := m.validateV2beta1RecurringRunStatusEnum("", "body", m); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

// ContextValidate validates this v2beta1 recurring run status based on context it is used
func (m V2beta1RecurringRunStatus) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}
