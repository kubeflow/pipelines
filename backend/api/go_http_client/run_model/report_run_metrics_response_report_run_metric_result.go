// Code generated by go-swagger; DO NOT EDIT.

package run_model

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// ReportRunMetricsResponseReportRunMetricResult report run metrics response report run metric result
//
// swagger:model ReportRunMetricsResponseReportRunMetricResult
type ReportRunMetricsResponseReportRunMetricResult struct {

	// Output. The detailed message of the error of the reporting.
	Message string `json:"message,omitempty"`

	// Output. The name of the metric.
	MetricName string `json:"metric_name,omitempty"`

	// Output. The ID of the node which reports the metric.
	MetricNodeID string `json:"metric_node_id,omitempty"`

	// Output. The status of the metric reporting.
	Status ReportRunMetricsResponseReportRunMetricResultStatus `json:"status,omitempty"`
}

// Validate validates this report run metrics response report run metric result
func (m *ReportRunMetricsResponseReportRunMetricResult) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateStatus(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *ReportRunMetricsResponseReportRunMetricResult) validateStatus(formats strfmt.Registry) error {

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

// MarshalBinary interface implementation
func (m *ReportRunMetricsResponseReportRunMetricResult) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *ReportRunMetricsResponseReportRunMetricResult) UnmarshalBinary(b []byte) error {
	var res ReportRunMetricsResponseReportRunMetricResult
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
