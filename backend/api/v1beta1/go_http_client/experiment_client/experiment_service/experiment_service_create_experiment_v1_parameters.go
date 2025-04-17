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

	experiment_model "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/experiment_model"
)

// NewExperimentServiceCreateExperimentV1Params creates a new ExperimentServiceCreateExperimentV1Params object
// with the default values initialized.
func NewExperimentServiceCreateExperimentV1Params() *ExperimentServiceCreateExperimentV1Params {
	var ()
	return &ExperimentServiceCreateExperimentV1Params{

		timeout: cr.DefaultTimeout,
	}
}

// NewExperimentServiceCreateExperimentV1ParamsWithTimeout creates a new ExperimentServiceCreateExperimentV1Params object
// with the default values initialized, and the ability to set a timeout on a request
func NewExperimentServiceCreateExperimentV1ParamsWithTimeout(timeout time.Duration) *ExperimentServiceCreateExperimentV1Params {
	var ()
	return &ExperimentServiceCreateExperimentV1Params{

		timeout: timeout,
	}
}

// NewExperimentServiceCreateExperimentV1ParamsWithContext creates a new ExperimentServiceCreateExperimentV1Params object
// with the default values initialized, and the ability to set a context for a request
func NewExperimentServiceCreateExperimentV1ParamsWithContext(ctx context.Context) *ExperimentServiceCreateExperimentV1Params {
	var ()
	return &ExperimentServiceCreateExperimentV1Params{

		Context: ctx,
	}
}

// NewExperimentServiceCreateExperimentV1ParamsWithHTTPClient creates a new ExperimentServiceCreateExperimentV1Params object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewExperimentServiceCreateExperimentV1ParamsWithHTTPClient(client *http.Client) *ExperimentServiceCreateExperimentV1Params {
	var ()
	return &ExperimentServiceCreateExperimentV1Params{
		HTTPClient: client,
	}
}

/*
ExperimentServiceCreateExperimentV1Params contains all the parameters to send to the API endpoint
for the experiment service create experiment v1 operation typically these are written to a http.Request
*/
type ExperimentServiceCreateExperimentV1Params struct {

	/*Body
	  The experiment to be created.

	*/
	Body *experiment_model.APIExperiment

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the experiment service create experiment v1 params
func (o *ExperimentServiceCreateExperimentV1Params) WithTimeout(timeout time.Duration) *ExperimentServiceCreateExperimentV1Params {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the experiment service create experiment v1 params
func (o *ExperimentServiceCreateExperimentV1Params) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the experiment service create experiment v1 params
func (o *ExperimentServiceCreateExperimentV1Params) WithContext(ctx context.Context) *ExperimentServiceCreateExperimentV1Params {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the experiment service create experiment v1 params
func (o *ExperimentServiceCreateExperimentV1Params) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the experiment service create experiment v1 params
func (o *ExperimentServiceCreateExperimentV1Params) WithHTTPClient(client *http.Client) *ExperimentServiceCreateExperimentV1Params {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the experiment service create experiment v1 params
func (o *ExperimentServiceCreateExperimentV1Params) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithBody adds the body to the experiment service create experiment v1 params
func (o *ExperimentServiceCreateExperimentV1Params) WithBody(body *experiment_model.APIExperiment) *ExperimentServiceCreateExperimentV1Params {
	o.SetBody(body)
	return o
}

// SetBody adds the body to the experiment service create experiment v1 params
func (o *ExperimentServiceCreateExperimentV1Params) SetBody(body *experiment_model.APIExperiment) {
	o.Body = body
}

// WriteToRequest writes these params to a swagger request
func (o *ExperimentServiceCreateExperimentV1Params) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

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
