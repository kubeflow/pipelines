// Code generated by go-swagger; DO NOT EDIT.

package run_model

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"strconv"

	strfmt "github.com/go-openapi/strfmt"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/swag"
	"github.com/go-openapi/validate"
)

// V2beta1Run v2beta1 run
// swagger:model v2beta1Run
type V2beta1Run struct {

	// Output. Creation time of the run.
	// Format: date-time
	CreatedAt strfmt.DateTime `json:"created_at,omitempty"`

	// Optional input. Short description of the run.
	Description string `json:"description,omitempty"`

	// Required input. Name provided by user,
	// or auto generated if run is created by a recurring run.
	DisplayName string `json:"display_name,omitempty"`

	// In case any error happens retrieving a run field, only run ID
	// and the error message is returned. Client has the flexibility of choosing
	// how to handle the error. This is especially useful during listing call.
	Error *GooglerpcStatus `json:"error,omitempty"`

	// Input. ID of the parent experiment.
	// The default experiment ID will be used if this is not specified.
	ExperimentID string `json:"experiment_id,omitempty"`

	// Output. Completion of the run.
	// Format: date-time
	FinishedAt strfmt.DateTime `json:"finished_at,omitempty"`

	// Pipeline spec.
	PipelineSpec interface{} `json:"pipeline_spec,omitempty"`

	// This field is Deprecated. The pipeline version id is under pipeline_version_reference for v2.
	PipelineVersionID string `json:"pipeline_version_id,omitempty"`

	// Reference to a pipeline containing pipeline_id and optionally the pipeline_version_id.
	PipelineVersionReference *V2beta1PipelineVersionReference `json:"pipeline_version_reference,omitempty"`

	// ID of the recurring run that triggered this run.
	RecurringRunID string `json:"recurring_run_id,omitempty"`

	// Output. Runtime details of a run.
	RunDetails *V2beta1RunDetails `json:"run_details,omitempty"`

	// Output. Unique run ID. Generated by API server.
	RunID string `json:"run_id,omitempty"`

	// Required input. Runtime config of the run.
	RuntimeConfig *V2beta1RuntimeConfig `json:"runtime_config,omitempty"`

	// Output. When this run is scheduled to start. This could be different from
	// created_at. For example, if a run is from a backfilling job that was supposed
	// to run 2 month ago, the created_at will be 2 month behind scheduled_at.
	// Format: date-time
	ScheduledAt strfmt.DateTime `json:"scheduled_at,omitempty"`

	// Optional input. Specifies which kubernetes service account is used.
	ServiceAccount string `json:"service_account,omitempty"`

	// Output. Runtime state of a run.
	State V2beta1RuntimeState `json:"state,omitempty"`

	// Output. A sequence of run statuses. This field keeps a record
	// of state transitions.
	StateHistory []*V2beta1RuntimeStatus `json:"state_history"`

	// Output. Specifies whether this run is in archived or available mode.
	StorageState V2beta1RunStorageState `json:"storage_state,omitempty"`
}

// Validate validates this v2beta1 run
func (m *V2beta1Run) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateCreatedAt(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateError(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateFinishedAt(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validatePipelineVersionReference(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateRunDetails(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateRuntimeConfig(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateScheduledAt(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateState(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateStateHistory(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateStorageState(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *V2beta1Run) validateCreatedAt(formats strfmt.Registry) error {

	if swag.IsZero(m.CreatedAt) { // not required
		return nil
	}

	if err := validate.FormatOf("created_at", "body", "date-time", m.CreatedAt.String(), formats); err != nil {
		return err
	}

	return nil
}

func (m *V2beta1Run) validateError(formats strfmt.Registry) error {

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

func (m *V2beta1Run) validateFinishedAt(formats strfmt.Registry) error {

	if swag.IsZero(m.FinishedAt) { // not required
		return nil
	}

	if err := validate.FormatOf("finished_at", "body", "date-time", m.FinishedAt.String(), formats); err != nil {
		return err
	}

	return nil
}

func (m *V2beta1Run) validatePipelineVersionReference(formats strfmt.Registry) error {

	if swag.IsZero(m.PipelineVersionReference) { // not required
		return nil
	}

	if m.PipelineVersionReference != nil {
		if err := m.PipelineVersionReference.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("pipeline_version_reference")
			}
			return err
		}
	}

	return nil
}

func (m *V2beta1Run) validateRunDetails(formats strfmt.Registry) error {

	if swag.IsZero(m.RunDetails) { // not required
		return nil
	}

	if m.RunDetails != nil {
		if err := m.RunDetails.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("run_details")
			}
			return err
		}
	}

	return nil
}

func (m *V2beta1Run) validateRuntimeConfig(formats strfmt.Registry) error {

	if swag.IsZero(m.RuntimeConfig) { // not required
		return nil
	}

	if m.RuntimeConfig != nil {
		if err := m.RuntimeConfig.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("runtime_config")
			}
			return err
		}
	}

	return nil
}

func (m *V2beta1Run) validateScheduledAt(formats strfmt.Registry) error {

	if swag.IsZero(m.ScheduledAt) { // not required
		return nil
	}

	if err := validate.FormatOf("scheduled_at", "body", "date-time", m.ScheduledAt.String(), formats); err != nil {
		return err
	}

	return nil
}

func (m *V2beta1Run) validateState(formats strfmt.Registry) error {

	if swag.IsZero(m.State) { // not required
		return nil
	}

	if err := m.State.Validate(formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("state")
		}
		return err
	}

	return nil
}

func (m *V2beta1Run) validateStateHistory(formats strfmt.Registry) error {

	if swag.IsZero(m.StateHistory) { // not required
		return nil
	}

	for i := 0; i < len(m.StateHistory); i++ {
		if swag.IsZero(m.StateHistory[i]) { // not required
			continue
		}

		if m.StateHistory[i] != nil {
			if err := m.StateHistory[i].Validate(formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("state_history" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

func (m *V2beta1Run) validateStorageState(formats strfmt.Registry) error {

	if swag.IsZero(m.StorageState) { // not required
		return nil
	}

	if err := m.StorageState.Validate(formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("storage_state")
		}
		return err
	}

	return nil
}

// MarshalBinary interface implementation
func (m *V2beta1Run) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *V2beta1Run) UnmarshalBinary(b []byte) error {
	var res V2beta1Run
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
