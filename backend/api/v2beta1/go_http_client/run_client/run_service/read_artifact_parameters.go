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

	strfmt "github.com/go-openapi/strfmt"
)

// NewReadArtifactParams creates a new ReadArtifactParams object
// with the default values initialized.
func NewReadArtifactParams() *ReadArtifactParams {
	var ()
	return &ReadArtifactParams{

		timeout: cr.DefaultTimeout,
	}
}

// NewReadArtifactParamsWithTimeout creates a new ReadArtifactParams object
// with the default values initialized, and the ability to set a timeout on a request
func NewReadArtifactParamsWithTimeout(timeout time.Duration) *ReadArtifactParams {
	var ()
	return &ReadArtifactParams{

		timeout: timeout,
	}
}

// NewReadArtifactParamsWithContext creates a new ReadArtifactParams object
// with the default values initialized, and the ability to set a context for a request
func NewReadArtifactParamsWithContext(ctx context.Context) *ReadArtifactParams {
	var ()
	return &ReadArtifactParams{

		Context: ctx,
	}
}

// NewReadArtifactParamsWithHTTPClient creates a new ReadArtifactParams object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewReadArtifactParamsWithHTTPClient(client *http.Client) *ReadArtifactParams {
	var ()
	return &ReadArtifactParams{
		HTTPClient: client,
	}
}

/*ReadArtifactParams contains all the parameters to send to the API endpoint
for the read artifact operation typically these are written to a http.Request
*/
type ReadArtifactParams struct {

	/*ArtifactName
	  Name of the artifact.

	*/
	ArtifactName string
	/*ExperimentID
	  The ID of the parent experiment.

	*/
	ExperimentID string
	/*NodeID
	  ID of the running node.

	*/
	NodeID string
	/*RunID
	  ID of the run.

	*/
	RunID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the read artifact params
func (o *ReadArtifactParams) WithTimeout(timeout time.Duration) *ReadArtifactParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the read artifact params
func (o *ReadArtifactParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the read artifact params
func (o *ReadArtifactParams) WithContext(ctx context.Context) *ReadArtifactParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the read artifact params
func (o *ReadArtifactParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the read artifact params
func (o *ReadArtifactParams) WithHTTPClient(client *http.Client) *ReadArtifactParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the read artifact params
func (o *ReadArtifactParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithArtifactName adds the artifactName to the read artifact params
func (o *ReadArtifactParams) WithArtifactName(artifactName string) *ReadArtifactParams {
	o.SetArtifactName(artifactName)
	return o
}

// SetArtifactName adds the artifactName to the read artifact params
func (o *ReadArtifactParams) SetArtifactName(artifactName string) {
	o.ArtifactName = artifactName
}

// WithExperimentID adds the experimentID to the read artifact params
func (o *ReadArtifactParams) WithExperimentID(experimentID string) *ReadArtifactParams {
	o.SetExperimentID(experimentID)
	return o
}

// SetExperimentID adds the experimentId to the read artifact params
func (o *ReadArtifactParams) SetExperimentID(experimentID string) {
	o.ExperimentID = experimentID
}

// WithNodeID adds the nodeID to the read artifact params
func (o *ReadArtifactParams) WithNodeID(nodeID string) *ReadArtifactParams {
	o.SetNodeID(nodeID)
	return o
}

// SetNodeID adds the nodeId to the read artifact params
func (o *ReadArtifactParams) SetNodeID(nodeID string) {
	o.NodeID = nodeID
}

// WithRunID adds the runID to the read artifact params
func (o *ReadArtifactParams) WithRunID(runID string) *ReadArtifactParams {
	o.SetRunID(runID)
	return o
}

// SetRunID adds the runId to the read artifact params
func (o *ReadArtifactParams) SetRunID(runID string) {
	o.RunID = runID
}

// WriteToRequest writes these params to a swagger request
func (o *ReadArtifactParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param artifact_name
	if err := r.SetPathParam("artifact_name", o.ArtifactName); err != nil {
		return err
	}

	// path param experiment_id
	if err := r.SetPathParam("experiment_id", o.ExperimentID); err != nil {
		return err
	}

	// path param node_id
	if err := r.SetPathParam("node_id", o.NodeID); err != nil {
		return err
	}

	// path param run_id
	if err := r.SetPathParam("run_id", o.RunID); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
