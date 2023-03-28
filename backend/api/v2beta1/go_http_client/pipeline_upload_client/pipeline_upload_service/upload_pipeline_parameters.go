// Code generated by go-swagger; DO NOT EDIT.

package pipeline_upload_service

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

// NewUploadPipelineParams creates a new UploadPipelineParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewUploadPipelineParams() *UploadPipelineParams {
	return &UploadPipelineParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewUploadPipelineParamsWithTimeout creates a new UploadPipelineParams object
// with the ability to set a timeout on a request.
func NewUploadPipelineParamsWithTimeout(timeout time.Duration) *UploadPipelineParams {
	return &UploadPipelineParams{
		timeout: timeout,
	}
}

// NewUploadPipelineParamsWithContext creates a new UploadPipelineParams object
// with the ability to set a context for a request.
func NewUploadPipelineParamsWithContext(ctx context.Context) *UploadPipelineParams {
	return &UploadPipelineParams{
		Context: ctx,
	}
}

// NewUploadPipelineParamsWithHTTPClient creates a new UploadPipelineParams object
// with the ability to set a custom HTTPClient for a request.
func NewUploadPipelineParamsWithHTTPClient(client *http.Client) *UploadPipelineParams {
	return &UploadPipelineParams{
		HTTPClient: client,
	}
}

/*
UploadPipelineParams contains all the parameters to send to the API endpoint

	for the upload pipeline operation.

	Typically these are written to a http.Request.
*/
type UploadPipelineParams struct {

	// Description.
	Description *string

	// Name.
	Name *string

	// Namespace.
	Namespace *string

	/* Uploadfile.

	   The pipeline to upload. Maximum size of 32MB is supported.
	*/
	Uploadfile runtime.NamedReadCloser

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the upload pipeline params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *UploadPipelineParams) WithDefaults() *UploadPipelineParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the upload pipeline params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *UploadPipelineParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the upload pipeline params
func (o *UploadPipelineParams) WithTimeout(timeout time.Duration) *UploadPipelineParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the upload pipeline params
func (o *UploadPipelineParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the upload pipeline params
func (o *UploadPipelineParams) WithContext(ctx context.Context) *UploadPipelineParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the upload pipeline params
func (o *UploadPipelineParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the upload pipeline params
func (o *UploadPipelineParams) WithHTTPClient(client *http.Client) *UploadPipelineParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the upload pipeline params
func (o *UploadPipelineParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithDescription adds the description to the upload pipeline params
func (o *UploadPipelineParams) WithDescription(description *string) *UploadPipelineParams {
	o.SetDescription(description)
	return o
}

// SetDescription adds the description to the upload pipeline params
func (o *UploadPipelineParams) SetDescription(description *string) {
	o.Description = description
}

// WithName adds the name to the upload pipeline params
func (o *UploadPipelineParams) WithName(name *string) *UploadPipelineParams {
	o.SetName(name)
	return o
}

// SetName adds the name to the upload pipeline params
func (o *UploadPipelineParams) SetName(name *string) {
	o.Name = name
}

// WithNamespace adds the namespace to the upload pipeline params
func (o *UploadPipelineParams) WithNamespace(namespace *string) *UploadPipelineParams {
	o.SetNamespace(namespace)
	return o
}

// SetNamespace adds the namespace to the upload pipeline params
func (o *UploadPipelineParams) SetNamespace(namespace *string) {
	o.Namespace = namespace
}

// WithUploadfile adds the uploadfile to the upload pipeline params
func (o *UploadPipelineParams) WithUploadfile(uploadfile runtime.NamedReadCloser) *UploadPipelineParams {
	o.SetUploadfile(uploadfile)
	return o
}

// SetUploadfile adds the uploadfile to the upload pipeline params
func (o *UploadPipelineParams) SetUploadfile(uploadfile runtime.NamedReadCloser) {
	o.Uploadfile = uploadfile
}

// WriteToRequest writes these params to a swagger request
func (o *UploadPipelineParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	if o.Description != nil {

		// query param description
		var qrDescription string

		if o.Description != nil {
			qrDescription = *o.Description
		}
		qDescription := qrDescription
		if qDescription != "" {

			if err := r.SetQueryParam("description", qDescription); err != nil {
				return err
			}
		}
	}

	if o.Name != nil {

		// query param name
		var qrName string

		if o.Name != nil {
			qrName = *o.Name
		}
		qName := qrName
		if qName != "" {

			if err := r.SetQueryParam("name", qName); err != nil {
				return err
			}
		}
	}

	if o.Namespace != nil {

		// query param namespace
		var qrNamespace string

		if o.Namespace != nil {
			qrNamespace = *o.Namespace
		}
		qNamespace := qrNamespace
		if qNamespace != "" {

			if err := r.SetQueryParam("namespace", qNamespace); err != nil {
				return err
			}
		}
	}
	// form file param uploadfile
	if err := r.SetFileParam("uploadfile", o.Uploadfile); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
