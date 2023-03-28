// Code generated by go-swagger; DO NOT EDIT.

package recurring_run_service

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"
	"net/http"
	"time"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	cr "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
)

// NewRecurringRunServiceEnableRecurringRunParams creates a new RecurringRunServiceEnableRecurringRunParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewRecurringRunServiceEnableRecurringRunParams() *RecurringRunServiceEnableRecurringRunParams {
	return &RecurringRunServiceEnableRecurringRunParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewRecurringRunServiceEnableRecurringRunParamsWithTimeout creates a new RecurringRunServiceEnableRecurringRunParams object
// with the ability to set a timeout on a request.
func NewRecurringRunServiceEnableRecurringRunParamsWithTimeout(timeout time.Duration) *RecurringRunServiceEnableRecurringRunParams {
	return &RecurringRunServiceEnableRecurringRunParams{
		timeout: timeout,
	}
}

// NewRecurringRunServiceEnableRecurringRunParamsWithContext creates a new RecurringRunServiceEnableRecurringRunParams object
// with the ability to set a context for a request.
func NewRecurringRunServiceEnableRecurringRunParamsWithContext(ctx context.Context) *RecurringRunServiceEnableRecurringRunParams {
	return &RecurringRunServiceEnableRecurringRunParams{
		Context: ctx,
	}
}

// NewRecurringRunServiceEnableRecurringRunParamsWithHTTPClient creates a new RecurringRunServiceEnableRecurringRunParams object
// with the ability to set a custom HTTPClient for a request.
func NewRecurringRunServiceEnableRecurringRunParamsWithHTTPClient(client *http.Client) *RecurringRunServiceEnableRecurringRunParams {
	return &RecurringRunServiceEnableRecurringRunParams{
		HTTPClient: client,
	}
}

/*
RecurringRunServiceEnableRecurringRunParams contains all the parameters to send to the API endpoint

	for the recurring run service enable recurring run operation.

	Typically these are written to a http.Request.
*/
type RecurringRunServiceEnableRecurringRunParams struct {

	/* RecurringRunID.

	   The ID of the recurring runs to be enabled.
	*/
	RecurringRunID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the recurring run service enable recurring run params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *RecurringRunServiceEnableRecurringRunParams) WithDefaults() *RecurringRunServiceEnableRecurringRunParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the recurring run service enable recurring run params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *RecurringRunServiceEnableRecurringRunParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the recurring run service enable recurring run params
func (o *RecurringRunServiceEnableRecurringRunParams) WithTimeout(timeout time.Duration) *RecurringRunServiceEnableRecurringRunParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the recurring run service enable recurring run params
func (o *RecurringRunServiceEnableRecurringRunParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the recurring run service enable recurring run params
func (o *RecurringRunServiceEnableRecurringRunParams) WithContext(ctx context.Context) *RecurringRunServiceEnableRecurringRunParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the recurring run service enable recurring run params
func (o *RecurringRunServiceEnableRecurringRunParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the recurring run service enable recurring run params
func (o *RecurringRunServiceEnableRecurringRunParams) WithHTTPClient(client *http.Client) *RecurringRunServiceEnableRecurringRunParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the recurring run service enable recurring run params
func (o *RecurringRunServiceEnableRecurringRunParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithRecurringRunID adds the recurringRunID to the recurring run service enable recurring run params
func (o *RecurringRunServiceEnableRecurringRunParams) WithRecurringRunID(recurringRunID string) *RecurringRunServiceEnableRecurringRunParams {
	o.SetRecurringRunID(recurringRunID)
	return o
}

// SetRecurringRunID adds the recurringRunId to the recurring run service enable recurring run params
func (o *RecurringRunServiceEnableRecurringRunParams) SetRecurringRunID(recurringRunID string) {
	o.RecurringRunID = recurringRunID
}

// WriteToRequest writes these params to a swagger request
func (o *RecurringRunServiceEnableRecurringRunParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param recurringRunId
	if err := r.SetPathParam("recurringRunId", o.RecurringRunID); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
