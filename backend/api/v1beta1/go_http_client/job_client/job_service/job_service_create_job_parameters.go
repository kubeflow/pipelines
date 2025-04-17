// Code generated by go-swagger; DO NOT EDIT.

package job_service

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

	job_model "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/job_model"
)

// NewJobServiceCreateJobParams creates a new JobServiceCreateJobParams object
// with the default values initialized.
func NewJobServiceCreateJobParams() *JobServiceCreateJobParams {
	var ()
	return &JobServiceCreateJobParams{

		timeout: cr.DefaultTimeout,
	}
}

// NewJobServiceCreateJobParamsWithTimeout creates a new JobServiceCreateJobParams object
// with the default values initialized, and the ability to set a timeout on a request
func NewJobServiceCreateJobParamsWithTimeout(timeout time.Duration) *JobServiceCreateJobParams {
	var ()
	return &JobServiceCreateJobParams{

		timeout: timeout,
	}
}

// NewJobServiceCreateJobParamsWithContext creates a new JobServiceCreateJobParams object
// with the default values initialized, and the ability to set a context for a request
func NewJobServiceCreateJobParamsWithContext(ctx context.Context) *JobServiceCreateJobParams {
	var ()
	return &JobServiceCreateJobParams{

		Context: ctx,
	}
}

// NewJobServiceCreateJobParamsWithHTTPClient creates a new JobServiceCreateJobParams object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewJobServiceCreateJobParamsWithHTTPClient(client *http.Client) *JobServiceCreateJobParams {
	var ()
	return &JobServiceCreateJobParams{
		HTTPClient: client,
	}
}

/*
JobServiceCreateJobParams contains all the parameters to send to the API endpoint
for the job service create job operation typically these are written to a http.Request
*/
type JobServiceCreateJobParams struct {

	/*Body
	  The job to be created

	*/
	Body *job_model.APIJob

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the job service create job params
func (o *JobServiceCreateJobParams) WithTimeout(timeout time.Duration) *JobServiceCreateJobParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the job service create job params
func (o *JobServiceCreateJobParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the job service create job params
func (o *JobServiceCreateJobParams) WithContext(ctx context.Context) *JobServiceCreateJobParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the job service create job params
func (o *JobServiceCreateJobParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the job service create job params
func (o *JobServiceCreateJobParams) WithHTTPClient(client *http.Client) *JobServiceCreateJobParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the job service create job params
func (o *JobServiceCreateJobParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithBody adds the body to the job service create job params
func (o *JobServiceCreateJobParams) WithBody(body *job_model.APIJob) *JobServiceCreateJobParams {
	o.SetBody(body)
	return o
}

// SetBody adds the body to the job service create job params
func (o *JobServiceCreateJobParams) SetBody(body *job_model.APIJob) {
	o.Body = body
}

// WriteToRequest writes these params to a swagger request
func (o *JobServiceCreateJobParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

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
