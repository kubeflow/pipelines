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

	strfmt "github.com/go-openapi/strfmt"
)

// NewExperimentServiceDeleteExperimentParams creates a new ExperimentServiceDeleteExperimentParams object
// with the default values initialized.
func NewExperimentServiceDeleteExperimentParams() *ExperimentServiceDeleteExperimentParams {
	var ()
	return &ExperimentServiceDeleteExperimentParams{

		timeout: cr.DefaultTimeout,
	}
}

// NewExperimentServiceDeleteExperimentParamsWithTimeout creates a new ExperimentServiceDeleteExperimentParams object
// with the default values initialized, and the ability to set a timeout on a request
func NewExperimentServiceDeleteExperimentParamsWithTimeout(timeout time.Duration) *ExperimentServiceDeleteExperimentParams {
	var ()
	return &ExperimentServiceDeleteExperimentParams{

		timeout: timeout,
	}
}

// NewExperimentServiceDeleteExperimentParamsWithContext creates a new ExperimentServiceDeleteExperimentParams object
// with the default values initialized, and the ability to set a context for a request
func NewExperimentServiceDeleteExperimentParamsWithContext(ctx context.Context) *ExperimentServiceDeleteExperimentParams {
	var ()
	return &ExperimentServiceDeleteExperimentParams{

		Context: ctx,
	}
}

// NewExperimentServiceDeleteExperimentParamsWithHTTPClient creates a new ExperimentServiceDeleteExperimentParams object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewExperimentServiceDeleteExperimentParamsWithHTTPClient(client *http.Client) *ExperimentServiceDeleteExperimentParams {
	var ()
	return &ExperimentServiceDeleteExperimentParams{
		HTTPClient: client,
	}
}

/*ExperimentServiceDeleteExperimentParams contains all the parameters to send to the API endpoint
for the experiment service delete experiment operation typically these are written to a http.Request
*/
type ExperimentServiceDeleteExperimentParams struct {

	/*ExperimentID
	  The ID of the experiment to be deleted.

	*/
	ExperimentID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the experiment service delete experiment params
func (o *ExperimentServiceDeleteExperimentParams) WithTimeout(timeout time.Duration) *ExperimentServiceDeleteExperimentParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the experiment service delete experiment params
func (o *ExperimentServiceDeleteExperimentParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the experiment service delete experiment params
func (o *ExperimentServiceDeleteExperimentParams) WithContext(ctx context.Context) *ExperimentServiceDeleteExperimentParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the experiment service delete experiment params
func (o *ExperimentServiceDeleteExperimentParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the experiment service delete experiment params
func (o *ExperimentServiceDeleteExperimentParams) WithHTTPClient(client *http.Client) *ExperimentServiceDeleteExperimentParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the experiment service delete experiment params
func (o *ExperimentServiceDeleteExperimentParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithExperimentID adds the experimentID to the experiment service delete experiment params
func (o *ExperimentServiceDeleteExperimentParams) WithExperimentID(experimentID string) *ExperimentServiceDeleteExperimentParams {
	o.SetExperimentID(experimentID)
	return o
}

// SetExperimentID adds the experimentId to the experiment service delete experiment params
func (o *ExperimentServiceDeleteExperimentParams) SetExperimentID(experimentID string) {
	o.ExperimentID = experimentID
}

// WriteToRequest writes these params to a swagger request
func (o *ExperimentServiceDeleteExperimentParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param experiment_id
	if err := r.SetPathParam("experiment_id", o.ExperimentID); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
