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

// NewExperimentServiceDeleteExperimentV1Params creates a new ExperimentServiceDeleteExperimentV1Params object
// with the default values initialized.
func NewExperimentServiceDeleteExperimentV1Params() *ExperimentServiceDeleteExperimentV1Params {
	var ()
	return &ExperimentServiceDeleteExperimentV1Params{

		timeout: cr.DefaultTimeout,
	}
}

// NewExperimentServiceDeleteExperimentV1ParamsWithTimeout creates a new ExperimentServiceDeleteExperimentV1Params object
// with the default values initialized, and the ability to set a timeout on a request
func NewExperimentServiceDeleteExperimentV1ParamsWithTimeout(timeout time.Duration) *ExperimentServiceDeleteExperimentV1Params {
	var ()
	return &ExperimentServiceDeleteExperimentV1Params{

		timeout: timeout,
	}
}

// NewExperimentServiceDeleteExperimentV1ParamsWithContext creates a new ExperimentServiceDeleteExperimentV1Params object
// with the default values initialized, and the ability to set a context for a request
func NewExperimentServiceDeleteExperimentV1ParamsWithContext(ctx context.Context) *ExperimentServiceDeleteExperimentV1Params {
	var ()
	return &ExperimentServiceDeleteExperimentV1Params{

		Context: ctx,
	}
}

// NewExperimentServiceDeleteExperimentV1ParamsWithHTTPClient creates a new ExperimentServiceDeleteExperimentV1Params object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewExperimentServiceDeleteExperimentV1ParamsWithHTTPClient(client *http.Client) *ExperimentServiceDeleteExperimentV1Params {
	var ()
	return &ExperimentServiceDeleteExperimentV1Params{
		HTTPClient: client,
	}
}

/*ExperimentServiceDeleteExperimentV1Params contains all the parameters to send to the API endpoint
for the experiment service delete experiment v1 operation typically these are written to a http.Request
*/
type ExperimentServiceDeleteExperimentV1Params struct {

	/*ID
	  The ID of the experiment to be deleted.

	*/
	ID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the experiment service delete experiment v1 params
func (o *ExperimentServiceDeleteExperimentV1Params) WithTimeout(timeout time.Duration) *ExperimentServiceDeleteExperimentV1Params {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the experiment service delete experiment v1 params
func (o *ExperimentServiceDeleteExperimentV1Params) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the experiment service delete experiment v1 params
func (o *ExperimentServiceDeleteExperimentV1Params) WithContext(ctx context.Context) *ExperimentServiceDeleteExperimentV1Params {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the experiment service delete experiment v1 params
func (o *ExperimentServiceDeleteExperimentV1Params) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the experiment service delete experiment v1 params
func (o *ExperimentServiceDeleteExperimentV1Params) WithHTTPClient(client *http.Client) *ExperimentServiceDeleteExperimentV1Params {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the experiment service delete experiment v1 params
func (o *ExperimentServiceDeleteExperimentV1Params) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithID adds the id to the experiment service delete experiment v1 params
func (o *ExperimentServiceDeleteExperimentV1Params) WithID(id string) *ExperimentServiceDeleteExperimentV1Params {
	o.SetID(id)
	return o
}

// SetID adds the id to the experiment service delete experiment v1 params
func (o *ExperimentServiceDeleteExperimentV1Params) SetID(id string) {
	o.ID = id
}

// WriteToRequest writes these params to a swagger request
func (o *ExperimentServiceDeleteExperimentV1Params) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param id
	if err := r.SetPathParam("id", o.ID); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
