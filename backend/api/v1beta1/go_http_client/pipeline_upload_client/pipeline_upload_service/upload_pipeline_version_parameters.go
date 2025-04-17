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

	strfmt "github.com/go-openapi/strfmt"
)

// NewUploadPipelineVersionParams creates a new UploadPipelineVersionParams object
// with the default values initialized.
func NewUploadPipelineVersionParams() *UploadPipelineVersionParams {
	var ()
	return &UploadPipelineVersionParams{

		timeout: cr.DefaultTimeout,
	}
}

// NewUploadPipelineVersionParamsWithTimeout creates a new UploadPipelineVersionParams object
// with the default values initialized, and the ability to set a timeout on a request
func NewUploadPipelineVersionParamsWithTimeout(timeout time.Duration) *UploadPipelineVersionParams {
	var ()
	return &UploadPipelineVersionParams{

		timeout: timeout,
	}
}

// NewUploadPipelineVersionParamsWithContext creates a new UploadPipelineVersionParams object
// with the default values initialized, and the ability to set a context for a request
func NewUploadPipelineVersionParamsWithContext(ctx context.Context) *UploadPipelineVersionParams {
	var ()
	return &UploadPipelineVersionParams{

		Context: ctx,
	}
}

// NewUploadPipelineVersionParamsWithHTTPClient creates a new UploadPipelineVersionParams object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewUploadPipelineVersionParamsWithHTTPClient(client *http.Client) *UploadPipelineVersionParams {
	var ()
	return &UploadPipelineVersionParams{
		HTTPClient: client,
	}
}

/*
UploadPipelineVersionParams contains all the parameters to send to the API endpoint
for the upload pipeline version operation typically these are written to a http.Request
*/
type UploadPipelineVersionParams struct {

	/*Description*/
	Description *string
	/*Name*/
	Name *string
	/*Namespace*/
	Namespace *string
	/*Pipelineid*/
	Pipelineid *string
	/*Uploadfile
	  The pipeline to upload. Maximum size of 32MB is supported.

	*/
	Uploadfile runtime.NamedReadCloser

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the upload pipeline version params
func (o *UploadPipelineVersionParams) WithTimeout(timeout time.Duration) *UploadPipelineVersionParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the upload pipeline version params
func (o *UploadPipelineVersionParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the upload pipeline version params
func (o *UploadPipelineVersionParams) WithContext(ctx context.Context) *UploadPipelineVersionParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the upload pipeline version params
func (o *UploadPipelineVersionParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the upload pipeline version params
func (o *UploadPipelineVersionParams) WithHTTPClient(client *http.Client) *UploadPipelineVersionParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the upload pipeline version params
func (o *UploadPipelineVersionParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithDescription adds the description to the upload pipeline version params
func (o *UploadPipelineVersionParams) WithDescription(description *string) *UploadPipelineVersionParams {
	o.SetDescription(description)
	return o
}

// SetDescription adds the description to the upload pipeline version params
func (o *UploadPipelineVersionParams) SetDescription(description *string) {
	o.Description = description
}

// WithName adds the name to the upload pipeline version params
func (o *UploadPipelineVersionParams) WithName(name *string) *UploadPipelineVersionParams {
	o.SetName(name)
	return o
}

// SetName adds the name to the upload pipeline version params
func (o *UploadPipelineVersionParams) SetName(name *string) {
	o.Name = name
}

// WithNamespace adds the namespace to the upload pipeline version params
func (o *UploadPipelineVersionParams) WithNamespace(namespace *string) *UploadPipelineVersionParams {
	o.SetNamespace(namespace)
	return o
}

// SetNamespace adds the namespace to the upload pipeline version params
func (o *UploadPipelineVersionParams) SetNamespace(namespace *string) {
	o.Namespace = namespace
}

// WithPipelineid adds the pipelineid to the upload pipeline version params
func (o *UploadPipelineVersionParams) WithPipelineid(pipelineid *string) *UploadPipelineVersionParams {
	o.SetPipelineid(pipelineid)
	return o
}

// SetPipelineid adds the pipelineid to the upload pipeline version params
func (o *UploadPipelineVersionParams) SetPipelineid(pipelineid *string) {
	o.Pipelineid = pipelineid
}

// WithUploadfile adds the uploadfile to the upload pipeline version params
func (o *UploadPipelineVersionParams) WithUploadfile(uploadfile runtime.NamedReadCloser) *UploadPipelineVersionParams {
	o.SetUploadfile(uploadfile)
	return o
}

// SetUploadfile adds the uploadfile to the upload pipeline version params
func (o *UploadPipelineVersionParams) SetUploadfile(uploadfile runtime.NamedReadCloser) {
	o.Uploadfile = uploadfile
}

// WriteToRequest writes these params to a swagger request
func (o *UploadPipelineVersionParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

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

	if o.Pipelineid != nil {

		// query param pipelineid
		var qrPipelineid string
		if o.Pipelineid != nil {
			qrPipelineid = *o.Pipelineid
		}
		qPipelineid := qrPipelineid
		if qPipelineid != "" {
			if err := r.SetQueryParam("pipelineid", qPipelineid); err != nil {
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
