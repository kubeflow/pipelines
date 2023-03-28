// Code generated by go-swagger; DO NOT EDIT.

package run_model

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// V2beta1PipelineTaskExecutorDetail Runtime information of a pipeline task executor.
//
// swagger:model v2beta1PipelineTaskExecutorDetail
type V2beta1PipelineTaskExecutorDetail struct {

	// The names of the previously failed job for the main container
	// executions. The list includes the all attempts in chronological order.
	FailedMainJobs []string `json:"failed_main_jobs"`

	// The names of the previously failed job for the
	// pre-caching-check container executions. This job will be available if the
	// Run.pipeline_spec specifies the `pre_caching_check` hook in
	// the lifecycle events.
	// The list includes the all attempts in chronological order.
	FailedPreCachingCheckJobs []string `json:"failed_pre_caching_check_jobs"`

	// The name of the job for the main container execution.
	MainJob string `json:"main_job,omitempty"`

	// The name of the job for the pre-caching-check container
	// execution. This job will be available if the
	// Run.pipeline_spec specifies the `pre_caching_check` hook in
	// the lifecycle events.
	PreCachingCheckJob string `json:"pre_caching_check_job,omitempty"`
}

// Validate validates this v2beta1 pipeline task executor detail
func (m *V2beta1PipelineTaskExecutorDetail) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this v2beta1 pipeline task executor detail based on context it is used
func (m *V2beta1PipelineTaskExecutorDetail) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *V2beta1PipelineTaskExecutorDetail) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *V2beta1PipelineTaskExecutorDetail) UnmarshalBinary(b []byte) error {
	var res V2beta1PipelineTaskExecutorDetail
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
