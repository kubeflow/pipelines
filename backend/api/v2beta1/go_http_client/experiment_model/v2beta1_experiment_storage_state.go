// Code generated by go-swagger; DO NOT EDIT.

package experiment_model

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"
	"encoding/json"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/validate"
)

// V2beta1ExperimentStorageState Describes whether an entity is available or archived.
//
//   - STORAGE_STATE_UNSPECIFIED: Default state. This state in not used
//   - AVAILABLE: Entity is available.
//   - ARCHIVED: Entity is archived.
//
// swagger:model v2beta1ExperimentStorageState
type V2beta1ExperimentStorageState string

func NewV2beta1ExperimentStorageState(value V2beta1ExperimentStorageState) *V2beta1ExperimentStorageState {
	return &value
}

// Pointer returns a pointer to a freshly-allocated V2beta1ExperimentStorageState.
func (m V2beta1ExperimentStorageState) Pointer() *V2beta1ExperimentStorageState {
	return &m
}

const (

	// V2beta1ExperimentStorageStateSTORAGESTATEUNSPECIFIED captures enum value "STORAGE_STATE_UNSPECIFIED"
	V2beta1ExperimentStorageStateSTORAGESTATEUNSPECIFIED V2beta1ExperimentStorageState = "STORAGE_STATE_UNSPECIFIED"

	// V2beta1ExperimentStorageStateAVAILABLE captures enum value "AVAILABLE"
	V2beta1ExperimentStorageStateAVAILABLE V2beta1ExperimentStorageState = "AVAILABLE"

	// V2beta1ExperimentStorageStateARCHIVED captures enum value "ARCHIVED"
	V2beta1ExperimentStorageStateARCHIVED V2beta1ExperimentStorageState = "ARCHIVED"
)

// for schema
var v2beta1ExperimentStorageStateEnum []interface{}

func init() {
	var res []V2beta1ExperimentStorageState
	if err := json.Unmarshal([]byte(`["STORAGE_STATE_UNSPECIFIED","AVAILABLE","ARCHIVED"]`), &res); err != nil {
		panic(err)
	}
	for _, v := range res {
		v2beta1ExperimentStorageStateEnum = append(v2beta1ExperimentStorageStateEnum, v)
	}
}

func (m V2beta1ExperimentStorageState) validateV2beta1ExperimentStorageStateEnum(path, location string, value V2beta1ExperimentStorageState) error {
	if err := validate.EnumCase(path, location, value, v2beta1ExperimentStorageStateEnum, true); err != nil {
		return err
	}
	return nil
}

// Validate validates this v2beta1 experiment storage state
func (m V2beta1ExperimentStorageState) Validate(formats strfmt.Registry) error {
	var res []error

	// value enum
	if err := m.validateV2beta1ExperimentStorageStateEnum("", "body", m); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

// ContextValidate validates this v2beta1 experiment storage state based on context it is used
func (m V2beta1ExperimentStorageState) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}
