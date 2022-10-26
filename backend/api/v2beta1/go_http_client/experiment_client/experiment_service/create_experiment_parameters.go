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

	experiment_model "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/experiment_model"
)

// NewCreateExperimentParams creates a new CreateExperimentParams object
// with the default values initialized.
func NewCreateExperimentParams() *CreateExperimentParams {
	var ()
	return &CreateExperimentParams{

		timeout: cr.DefaultTimeout,
	}
}

// NewCreateExperimentParamsWithTimeout creates a new CreateExperimentParams object
// with the default values initialized, and the ability to set a timeout on a request
func NewCreateExperimentParamsWithTimeout(timeout time.Duration) *CreateExperimentParams {
	var ()
	return &CreateExperimentParams{

		timeout: timeout,
	}
}

// NewCreateExperimentParamsWithContext creates a new CreateExperimentParams object
// with the default values initialized, and the ability to set a context for a request
func NewCreateExperimentParamsWithContext(ctx context.Context) *CreateExperimentParams {
	var ()
	return &CreateExperimentParams{

		Context: ctx,
	}
}

// NewCreateExperimentParamsWithHTTPClient creates a new CreateExperimentParams object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewCreateExperimentParamsWithHTTPClient(client *http.Client) *CreateExperimentParams {
	var ()
	return &CreateExperimentParams{
		HTTPClient: client,
	}
}

/*CreateExperimentParams contains all the parameters to send to the API endpoint
for the create experiment operation typically these are written to a http.Request
*/
type CreateExperimentParams struct {

	/*Body
	  The experiment to be created.

	*/
	Body *experiment_model.V2beat1Experiment

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the create experiment params
func (o *CreateExperimentParams) WithTimeout(timeout time.Duration) *CreateExperimentParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the create experiment params
func (o *CreateExperimentParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the create experiment params
func (o *CreateExperimentParams) WithContext(ctx context.Context) *CreateExperimentParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the create experiment params
func (o *CreateExperimentParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the create experiment params
func (o *CreateExperimentParams) WithHTTPClient(client *http.Client) *CreateExperimentParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the create experiment params
func (o *CreateExperimentParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithBody adds the body to the create experiment params
func (o *CreateExperimentParams) WithBody(body *experiment_model.V2beat1Experiment) *CreateExperimentParams {
	o.SetBody(body)
	return o
}

// SetBody adds the body to the create experiment params
func (o *CreateExperimentParams) SetBody(body *experiment_model.V2beat1Experiment) {
	o.Body = body
}

// WriteToRequest writes these params to a swagger request
func (o *CreateExperimentParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	if o.Body != nil {
		if err := r.SetBodyParam(o.Body); err != nil {
			return err
		}
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
