// Code generated by go-swagger; DO NOT EDIT.

package pipeline_service

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

// NewDeletePipelineVersionParams creates a new DeletePipelineVersionParams object
// with the default values initialized.
func NewDeletePipelineVersionParams() *DeletePipelineVersionParams {
	var ()
	return &DeletePipelineVersionParams{

		timeout: cr.DefaultTimeout,
	}
}

// NewDeletePipelineVersionParamsWithTimeout creates a new DeletePipelineVersionParams object
// with the default values initialized, and the ability to set a timeout on a request
func NewDeletePipelineVersionParamsWithTimeout(timeout time.Duration) *DeletePipelineVersionParams {
	var ()
	return &DeletePipelineVersionParams{

		timeout: timeout,
	}
}

// NewDeletePipelineVersionParamsWithContext creates a new DeletePipelineVersionParams object
// with the default values initialized, and the ability to set a context for a request
func NewDeletePipelineVersionParamsWithContext(ctx context.Context) *DeletePipelineVersionParams {
	var ()
	return &DeletePipelineVersionParams{

		Context: ctx,
	}
}

// NewDeletePipelineVersionParamsWithHTTPClient creates a new DeletePipelineVersionParams object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewDeletePipelineVersionParamsWithHTTPClient(client *http.Client) *DeletePipelineVersionParams {
	var ()
	return &DeletePipelineVersionParams{
		HTTPClient: client,
	}
}

/*DeletePipelineVersionParams contains all the parameters to send to the API endpoint
for the delete pipeline version operation typically these are written to a http.Request
*/
type DeletePipelineVersionParams struct {

	/*VersionID
	  The ID of the pipeline version to be deleted.

	*/
	VersionID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the delete pipeline version params
func (o *DeletePipelineVersionParams) WithTimeout(timeout time.Duration) *DeletePipelineVersionParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the delete pipeline version params
func (o *DeletePipelineVersionParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the delete pipeline version params
func (o *DeletePipelineVersionParams) WithContext(ctx context.Context) *DeletePipelineVersionParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the delete pipeline version params
func (o *DeletePipelineVersionParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the delete pipeline version params
func (o *DeletePipelineVersionParams) WithHTTPClient(client *http.Client) *DeletePipelineVersionParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the delete pipeline version params
func (o *DeletePipelineVersionParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithVersionID adds the versionID to the delete pipeline version params
func (o *DeletePipelineVersionParams) WithVersionID(versionID string) *DeletePipelineVersionParams {
	o.SetVersionID(versionID)
	return o
}

// SetVersionID adds the versionId to the delete pipeline version params
func (o *DeletePipelineVersionParams) SetVersionID(versionID string) {
	o.VersionID = versionID
}

// WriteToRequest writes these params to a swagger request
func (o *DeletePipelineVersionParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param version_id
	if err := r.SetPathParam("version_id", o.VersionID); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
