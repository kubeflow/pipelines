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

	recurring_run_model "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/recurring_run_model"
)

// NewRecurringRunServiceCreateRecurringRunParams creates a new RecurringRunServiceCreateRecurringRunParams object
// with the default values initialized.
func NewRecurringRunServiceCreateRecurringRunParams() *RecurringRunServiceCreateRecurringRunParams {
	var ()
	return &RecurringRunServiceCreateRecurringRunParams{

		timeout: cr.DefaultTimeout,
	}
}

// NewRecurringRunServiceCreateRecurringRunParamsWithTimeout creates a new RecurringRunServiceCreateRecurringRunParams object
// with the default values initialized, and the ability to set a timeout on a request
func NewRecurringRunServiceCreateRecurringRunParamsWithTimeout(timeout time.Duration) *RecurringRunServiceCreateRecurringRunParams {
	var ()
	return &RecurringRunServiceCreateRecurringRunParams{

		timeout: timeout,
	}
}

// NewRecurringRunServiceCreateRecurringRunParamsWithContext creates a new RecurringRunServiceCreateRecurringRunParams object
// with the default values initialized, and the ability to set a context for a request
func NewRecurringRunServiceCreateRecurringRunParamsWithContext(ctx context.Context) *RecurringRunServiceCreateRecurringRunParams {
	var ()
	return &RecurringRunServiceCreateRecurringRunParams{

		Context: ctx,
	}
}

// NewRecurringRunServiceCreateRecurringRunParamsWithHTTPClient creates a new RecurringRunServiceCreateRecurringRunParams object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewRecurringRunServiceCreateRecurringRunParamsWithHTTPClient(client *http.Client) *RecurringRunServiceCreateRecurringRunParams {
	var ()
	return &RecurringRunServiceCreateRecurringRunParams{
		HTTPClient: client,
	}
}

/*RecurringRunServiceCreateRecurringRunParams contains all the parameters to send to the API endpoint
for the recurring run service create recurring run operation typically these are written to a http.Request
*/
type RecurringRunServiceCreateRecurringRunParams struct {

	/*Body
	  The recurring run to be created.

	*/
	Body *recurring_run_model.V2beta1RecurringRun

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the recurring run service create recurring run params
func (o *RecurringRunServiceCreateRecurringRunParams) WithTimeout(timeout time.Duration) *RecurringRunServiceCreateRecurringRunParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the recurring run service create recurring run params
func (o *RecurringRunServiceCreateRecurringRunParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the recurring run service create recurring run params
func (o *RecurringRunServiceCreateRecurringRunParams) WithContext(ctx context.Context) *RecurringRunServiceCreateRecurringRunParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the recurring run service create recurring run params
func (o *RecurringRunServiceCreateRecurringRunParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the recurring run service create recurring run params
func (o *RecurringRunServiceCreateRecurringRunParams) WithHTTPClient(client *http.Client) *RecurringRunServiceCreateRecurringRunParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the recurring run service create recurring run params
func (o *RecurringRunServiceCreateRecurringRunParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithBody adds the body to the recurring run service create recurring run params
func (o *RecurringRunServiceCreateRecurringRunParams) WithBody(body *recurring_run_model.V2beta1RecurringRun) *RecurringRunServiceCreateRecurringRunParams {
	o.SetBody(body)
	return o
}

// SetBody adds the body to the recurring run service create recurring run params
func (o *RecurringRunServiceCreateRecurringRunParams) SetBody(body *recurring_run_model.V2beta1RecurringRun) {
	o.Body = body
}

// WriteToRequest writes these params to a swagger request
func (o *RecurringRunServiceCreateRecurringRunParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

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
