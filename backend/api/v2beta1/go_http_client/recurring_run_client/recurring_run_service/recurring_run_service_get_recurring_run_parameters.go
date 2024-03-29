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

	strfmt "github.com/go-openapi/strfmt"
)

// NewRecurringRunServiceGetRecurringRunParams creates a new RecurringRunServiceGetRecurringRunParams object
// with the default values initialized.
func NewRecurringRunServiceGetRecurringRunParams() *RecurringRunServiceGetRecurringRunParams {
	var ()
	return &RecurringRunServiceGetRecurringRunParams{

		timeout: cr.DefaultTimeout,
	}
}

// NewRecurringRunServiceGetRecurringRunParamsWithTimeout creates a new RecurringRunServiceGetRecurringRunParams object
// with the default values initialized, and the ability to set a timeout on a request
func NewRecurringRunServiceGetRecurringRunParamsWithTimeout(timeout time.Duration) *RecurringRunServiceGetRecurringRunParams {
	var ()
	return &RecurringRunServiceGetRecurringRunParams{

		timeout: timeout,
	}
}

// NewRecurringRunServiceGetRecurringRunParamsWithContext creates a new RecurringRunServiceGetRecurringRunParams object
// with the default values initialized, and the ability to set a context for a request
func NewRecurringRunServiceGetRecurringRunParamsWithContext(ctx context.Context) *RecurringRunServiceGetRecurringRunParams {
	var ()
	return &RecurringRunServiceGetRecurringRunParams{

		Context: ctx,
	}
}

// NewRecurringRunServiceGetRecurringRunParamsWithHTTPClient creates a new RecurringRunServiceGetRecurringRunParams object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewRecurringRunServiceGetRecurringRunParamsWithHTTPClient(client *http.Client) *RecurringRunServiceGetRecurringRunParams {
	var ()
	return &RecurringRunServiceGetRecurringRunParams{
		HTTPClient: client,
	}
}

/*RecurringRunServiceGetRecurringRunParams contains all the parameters to send to the API endpoint
for the recurring run service get recurring run operation typically these are written to a http.Request
*/
type RecurringRunServiceGetRecurringRunParams struct {

	/*RecurringRunID
	  The ID of the recurring run to be retrieved.

	*/
	RecurringRunID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the recurring run service get recurring run params
func (o *RecurringRunServiceGetRecurringRunParams) WithTimeout(timeout time.Duration) *RecurringRunServiceGetRecurringRunParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the recurring run service get recurring run params
func (o *RecurringRunServiceGetRecurringRunParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the recurring run service get recurring run params
func (o *RecurringRunServiceGetRecurringRunParams) WithContext(ctx context.Context) *RecurringRunServiceGetRecurringRunParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the recurring run service get recurring run params
func (o *RecurringRunServiceGetRecurringRunParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the recurring run service get recurring run params
func (o *RecurringRunServiceGetRecurringRunParams) WithHTTPClient(client *http.Client) *RecurringRunServiceGetRecurringRunParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the recurring run service get recurring run params
func (o *RecurringRunServiceGetRecurringRunParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithRecurringRunID adds the recurringRunID to the recurring run service get recurring run params
func (o *RecurringRunServiceGetRecurringRunParams) WithRecurringRunID(recurringRunID string) *RecurringRunServiceGetRecurringRunParams {
	o.SetRecurringRunID(recurringRunID)
	return o
}

// SetRecurringRunID adds the recurringRunId to the recurring run service get recurring run params
func (o *RecurringRunServiceGetRecurringRunParams) SetRecurringRunID(recurringRunID string) {
	o.RecurringRunID = recurringRunID
}

// WriteToRequest writes these params to a swagger request
func (o *RecurringRunServiceGetRecurringRunParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param recurring_run_id
	if err := r.SetPathParam("recurring_run_id", o.RecurringRunID); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
