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

// NewUnarchiveRunParams creates a new UnarchiveRunParams object
// with the default values initialized.
func NewUnarchiveRunParams() *UnarchiveRunParams {
	var ()
	return &UnarchiveRunParams{

		timeout: cr.DefaultTimeout,
	}
}

// NewUnarchiveRunParamsWithTimeout creates a new UnarchiveRunParams object
// with the default values initialized, and the ability to set a timeout on a request
func NewUnarchiveRunParamsWithTimeout(timeout time.Duration) *UnarchiveRunParams {
	var ()
	return &UnarchiveRunParams{

		timeout: timeout,
	}
}

// NewUnarchiveRunParamsWithContext creates a new UnarchiveRunParams object
// with the default values initialized, and the ability to set a context for a request
func NewUnarchiveRunParamsWithContext(ctx context.Context) *UnarchiveRunParams {
	var ()
	return &UnarchiveRunParams{

		Context: ctx,
	}
}

// NewUnarchiveRunParamsWithHTTPClient creates a new UnarchiveRunParams object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewUnarchiveRunParamsWithHTTPClient(client *http.Client) *UnarchiveRunParams {
	var ()
	return &UnarchiveRunParams{
		HTTPClient: client,
	}
}

/*UnarchiveRunParams contains all the parameters to send to the API endpoint
for the unarchive run operation typically these are written to a http.Request
*/
type UnarchiveRunParams struct {

	/*ID
	  The ID of the run to be restored.

	*/
	ID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the unarchive run params
func (o *UnarchiveRunParams) WithTimeout(timeout time.Duration) *UnarchiveRunParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the unarchive run params
func (o *UnarchiveRunParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the unarchive run params
func (o *UnarchiveRunParams) WithContext(ctx context.Context) *UnarchiveRunParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the unarchive run params
func (o *UnarchiveRunParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the unarchive run params
func (o *UnarchiveRunParams) WithHTTPClient(client *http.Client) *UnarchiveRunParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the unarchive run params
func (o *UnarchiveRunParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithID adds the id to the unarchive run params
func (o *UnarchiveRunParams) WithID(id string) *UnarchiveRunParams {
	o.SetID(id)
	return o
}

// SetID adds the id to the unarchive run params
func (o *UnarchiveRunParams) SetID(id string) {
	o.ID = id
}

// WriteToRequest writes these params to a swagger request
func (o *UnarchiveRunParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

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
