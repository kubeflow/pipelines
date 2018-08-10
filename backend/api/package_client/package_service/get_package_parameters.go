// Code generated by go-swagger; DO NOT EDIT.

package package_service

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"net/http"
	"time"

	"golang.org/x/net/context"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	cr "github.com/go-openapi/runtime/client"

	strfmt "github.com/go-openapi/strfmt"
)

// NewGetPackageParams creates a new GetPackageParams object
// with the default values initialized.
func NewGetPackageParams() *GetPackageParams {
	var ()
	return &GetPackageParams{

		timeout: cr.DefaultTimeout,
	}
}

// NewGetPackageParamsWithTimeout creates a new GetPackageParams object
// with the default values initialized, and the ability to set a timeout on a request
func NewGetPackageParamsWithTimeout(timeout time.Duration) *GetPackageParams {
	var ()
	return &GetPackageParams{

		timeout: timeout,
	}
}

// NewGetPackageParamsWithContext creates a new GetPackageParams object
// with the default values initialized, and the ability to set a context for a request
func NewGetPackageParamsWithContext(ctx context.Context) *GetPackageParams {
	var ()
	return &GetPackageParams{

		Context: ctx,
	}
}

// NewGetPackageParamsWithHTTPClient creates a new GetPackageParams object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewGetPackageParamsWithHTTPClient(client *http.Client) *GetPackageParams {
	var ()
	return &GetPackageParams{
		HTTPClient: client,
	}
}

/*GetPackageParams contains all the parameters to send to the API endpoint
for the get package operation typically these are written to a http.Request
*/
type GetPackageParams struct {

	/*ID*/
	ID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the get package params
func (o *GetPackageParams) WithTimeout(timeout time.Duration) *GetPackageParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the get package params
func (o *GetPackageParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the get package params
func (o *GetPackageParams) WithContext(ctx context.Context) *GetPackageParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the get package params
func (o *GetPackageParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the get package params
func (o *GetPackageParams) WithHTTPClient(client *http.Client) *GetPackageParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the get package params
func (o *GetPackageParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithID adds the id to the get package params
func (o *GetPackageParams) WithID(id string) *GetPackageParams {
	o.SetID(id)
	return o
}

// SetID adds the id to the get package params
func (o *GetPackageParams) SetID(id string) {
	o.ID = id
}

// WriteToRequest writes these params to a swagger request
func (o *GetPackageParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

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
