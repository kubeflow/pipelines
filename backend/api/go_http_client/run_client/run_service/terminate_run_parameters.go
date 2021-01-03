// Code generated by go-swagger; DO NOT EDIT.

package run_service

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

// NewTerminateRunParams creates a new TerminateRunParams object
// with the default values initialized.
func NewTerminateRunParams() *TerminateRunParams {
	var ()
	return &TerminateRunParams{

		timeout: cr.DefaultTimeout,
	}
}

// NewTerminateRunParamsWithTimeout creates a new TerminateRunParams object
// with the default values initialized, and the ability to set a timeout on a request
func NewTerminateRunParamsWithTimeout(timeout time.Duration) *TerminateRunParams {
	var ()
	return &TerminateRunParams{

		timeout: timeout,
	}
}

// NewTerminateRunParamsWithContext creates a new TerminateRunParams object
// with the default values initialized, and the ability to set a context for a request
func NewTerminateRunParamsWithContext(ctx context.Context) *TerminateRunParams {
	var ()
	return &TerminateRunParams{

		Context: ctx,
	}
}

// NewTerminateRunParamsWithHTTPClient creates a new TerminateRunParams object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewTerminateRunParamsWithHTTPClient(client *http.Client) *TerminateRunParams {
	var ()
	return &TerminateRunParams{
		HTTPClient: client,
	}
}

/*TerminateRunParams contains all the parameters to send to the API endpoint
for the terminate run operation typically these are written to a http.Request
*/
type TerminateRunParams struct {

	/*RunID
	  The ID of the run to be terminated.

	*/
	RunID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the terminate run params
func (o *TerminateRunParams) WithTimeout(timeout time.Duration) *TerminateRunParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the terminate run params
func (o *TerminateRunParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the terminate run params
func (o *TerminateRunParams) WithContext(ctx context.Context) *TerminateRunParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the terminate run params
func (o *TerminateRunParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the terminate run params
func (o *TerminateRunParams) WithHTTPClient(client *http.Client) *TerminateRunParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the terminate run params
func (o *TerminateRunParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithRunID adds the runID to the terminate run params
func (o *TerminateRunParams) WithRunID(runID string) *TerminateRunParams {
	o.SetRunID(runID)
	return o
}

// SetRunID adds the runId to the terminate run params
func (o *TerminateRunParams) SetRunID(runID string) {
	o.RunID = runID
}

// WriteToRequest writes these params to a swagger request
func (o *TerminateRunParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param run_id
	if err := r.SetPathParam("run_id", o.RunID); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
