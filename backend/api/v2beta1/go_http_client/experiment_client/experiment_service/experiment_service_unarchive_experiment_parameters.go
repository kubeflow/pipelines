// Code generated by go-swagger; DO NOT EDIT.

package experiment_service

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

// NewExperimentServiceUnarchiveExperimentParams creates a new ExperimentServiceUnarchiveExperimentParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewExperimentServiceUnarchiveExperimentParams() *ExperimentServiceUnarchiveExperimentParams {
	return &ExperimentServiceUnarchiveExperimentParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewExperimentServiceUnarchiveExperimentParamsWithTimeout creates a new ExperimentServiceUnarchiveExperimentParams object
// with the ability to set a timeout on a request.
func NewExperimentServiceUnarchiveExperimentParamsWithTimeout(timeout time.Duration) *ExperimentServiceUnarchiveExperimentParams {
	return &ExperimentServiceUnarchiveExperimentParams{
		timeout: timeout,
	}
}

// NewExperimentServiceUnarchiveExperimentParamsWithContext creates a new ExperimentServiceUnarchiveExperimentParams object
// with the ability to set a context for a request.
func NewExperimentServiceUnarchiveExperimentParamsWithContext(ctx context.Context) *ExperimentServiceUnarchiveExperimentParams {
	return &ExperimentServiceUnarchiveExperimentParams{
		Context: ctx,
	}
}

// NewExperimentServiceUnarchiveExperimentParamsWithHTTPClient creates a new ExperimentServiceUnarchiveExperimentParams object
// with the ability to set a custom HTTPClient for a request.
func NewExperimentServiceUnarchiveExperimentParamsWithHTTPClient(client *http.Client) *ExperimentServiceUnarchiveExperimentParams {
	return &ExperimentServiceUnarchiveExperimentParams{
		HTTPClient: client,
	}
}

/*
ExperimentServiceUnarchiveExperimentParams contains all the parameters to send to the API endpoint

	for the experiment service unarchive experiment operation.

	Typically these are written to a http.Request.
*/
type ExperimentServiceUnarchiveExperimentParams struct {

	/* ExperimentID.

	   The ID of the experiment to be restored.
	*/
	ExperimentID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the experiment service unarchive experiment params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ExperimentServiceUnarchiveExperimentParams) WithDefaults() *ExperimentServiceUnarchiveExperimentParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the experiment service unarchive experiment params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ExperimentServiceUnarchiveExperimentParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the experiment service unarchive experiment params
func (o *ExperimentServiceUnarchiveExperimentParams) WithTimeout(timeout time.Duration) *ExperimentServiceUnarchiveExperimentParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the experiment service unarchive experiment params
func (o *ExperimentServiceUnarchiveExperimentParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the experiment service unarchive experiment params
func (o *ExperimentServiceUnarchiveExperimentParams) WithContext(ctx context.Context) *ExperimentServiceUnarchiveExperimentParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the experiment service unarchive experiment params
func (o *ExperimentServiceUnarchiveExperimentParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the experiment service unarchive experiment params
func (o *ExperimentServiceUnarchiveExperimentParams) WithHTTPClient(client *http.Client) *ExperimentServiceUnarchiveExperimentParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the experiment service unarchive experiment params
func (o *ExperimentServiceUnarchiveExperimentParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithExperimentID adds the experimentID to the experiment service unarchive experiment params
func (o *ExperimentServiceUnarchiveExperimentParams) WithExperimentID(experimentID string) *ExperimentServiceUnarchiveExperimentParams {
	o.SetExperimentID(experimentID)
	return o
}

// SetExperimentID adds the experimentId to the experiment service unarchive experiment params
func (o *ExperimentServiceUnarchiveExperimentParams) SetExperimentID(experimentID string) {
	o.ExperimentID = experimentID
}

// WriteToRequest writes these params to a swagger request
func (o *ExperimentServiceUnarchiveExperimentParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param experimentId
	if err := r.SetPathParam("experimentId", o.ExperimentID); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
