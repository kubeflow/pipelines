// Code generated by go-swagger; DO NOT EDIT.

package recurring_run_model

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	strfmt "github.com/go-openapi/strfmt"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/swag"
	"github.com/go-openapi/validate"
)

// V2beta1RecurringRun v2beta1 recurring run
// swagger:model v2beta1RecurringRun
type V2beta1RecurringRun struct {

	// Output. The time this recurring run was created.
	// Format: date-time
	CreatedAt strfmt.DateTime `json:"created_at,omitempty"`

	// Optional input field. Describes the purpose of the recurring run.
	Description string `json:"description,omitempty"`

	// Required input field. Recurring run name provided by user. Not unique.
	DisplayName string `json:"display_name,omitempty"`

	// In case any error happens retrieving a recurring run field, only recurring run ID
	// and the error message is returned. Client has the flexibility of choosing
	// how to handle the error. This is especially useful during listing call.
	Error *GooglerpcStatus `json:"error,omitempty"`

	// ID of the parent experiment this recurring run belongs to.
	ExperimentID string `json:"experiment_id,omitempty"`

	// Required input field.
	// Specifies how many runs can be executed concurrently. Range [1-10].
	MaxConcurrency int64 `json:"max_concurrency,omitempty,string"`

	// mode
	Mode RecurringRunMode `json:"mode,omitempty"`

	// TODO (gkclat): consider removing this field if it can be obtained from the parent experiment.
	// Output only. Namespace this recurring run belongs to. Derived from the parent experiment.
	// Read Only: true
	Namespace string `json:"namespace,omitempty"`

	// Optional input field. Whether the recurring run should catch up if behind schedule.
	// If true, the recurring run will only schedule the latest interval if behind schedule.
	// If false, the recurring run will catch up on each past interval.
	NoCatchup bool `json:"no_catchup,omitempty"`

	// The pipeline spec.
	PipelineSpec interface{} `json:"pipeline_spec,omitempty"`

	// The ID of the pipeline version used for creating runs.
	PipelineVersionID string `json:"pipeline_version_id,omitempty"`

	// Reference to a pipeline version containing pipeline_id and pipeline_version_id.
	PipelineVersionReference *V2beta1PipelineVersionReference `json:"pipeline_version_reference,omitempty"`

	// Output. Unique run ID generated by API server.
	RecurringRunID string `json:"recurring_run_id,omitempty"`

	// Runtime config of the pipeline.
	RuntimeConfig *V2beta1RuntimeConfig `json:"runtime_config,omitempty"`

	// Optional input field. Specifies which Kubernetes service account this recurring run uses.
	ServiceAccount string `json:"service_account,omitempty"`

	// status
	Status V2beta1RecurringRunStatus `json:"status,omitempty"`

	// Required input field.
	// Specifies how a run is triggered. Support cron mode or periodic mode.
	Trigger *V2beta1Trigger `json:"trigger,omitempty"`

	// Output. The last time this recurring run was updated.
	// Format: date-time
	UpdatedAt strfmt.DateTime `json:"updated_at,omitempty"`
}

// Validate validates this v2beta1 recurring run
func (m *V2beta1RecurringRun) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateCreatedAt(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateError(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateMode(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validatePipelineVersionReference(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateRuntimeConfig(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateStatus(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateTrigger(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateUpdatedAt(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *V2beta1RecurringRun) validateCreatedAt(formats strfmt.Registry) error {

	if swag.IsZero(m.CreatedAt) { // not required
		return nil
	}

	if err := validate.FormatOf("created_at", "body", "date-time", m.CreatedAt.String(), formats); err != nil {
		return err
	}

	return nil
}

func (m *V2beta1RecurringRun) validateError(formats strfmt.Registry) error {

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

func (m *V2beta1RecurringRun) validateMode(formats strfmt.Registry) error {

	if swag.IsZero(m.Mode) { // not required
		return nil
	}

	if err := m.Mode.Validate(formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("mode")
		}
		return err
	}

	return nil
}

func (m *V2beta1RecurringRun) validatePipelineVersionReference(formats strfmt.Registry) error {

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

func (m *V2beta1RecurringRun) validateRuntimeConfig(formats strfmt.Registry) error {

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

func (m *V2beta1RecurringRun) validateStatus(formats strfmt.Registry) error {

	if swag.IsZero(m.Status) { // not required
		return nil
	}

	if err := m.Status.Validate(formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("status")
		}
		return err
	}

	return nil
}

func (m *V2beta1RecurringRun) validateTrigger(formats strfmt.Registry) error {

	if swag.IsZero(m.Trigger) { // not required
		return nil
	}

	if m.Trigger != nil {
		if err := m.Trigger.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("trigger")
			}
			return err
		}
	}

	return nil
}

func (m *V2beta1RecurringRun) validateUpdatedAt(formats strfmt.Registry) error {

	if swag.IsZero(m.UpdatedAt) { // not required
		return nil
	}

	if err := validate.FormatOf("updated_at", "body", "date-time", m.UpdatedAt.String(), formats); err != nil {
		return err
	}

	return nil
}

// MarshalBinary interface implementation
func (m *V2beta1RecurringRun) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *V2beta1RecurringRun) UnmarshalBinary(b []byte) error {
	var res V2beta1RecurringRun
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
