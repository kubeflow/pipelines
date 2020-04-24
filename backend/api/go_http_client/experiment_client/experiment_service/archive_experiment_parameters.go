// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
)

// NewArchiveExperimentParams creates a new ArchiveExperimentParams object
// with the default values initialized.
func NewArchiveExperimentParams() *ArchiveExperimentParams {
	var ()
	return &ArchiveExperimentParams{

		timeout: cr.DefaultTimeout,
	}
}

// NewArchiveExperimentParamsWithTimeout creates a new ArchiveExperimentParams object
// with the default values initialized, and the ability to set a timeout on a request
func NewArchiveExperimentParamsWithTimeout(timeout time.Duration) *ArchiveExperimentParams {
	var ()
	return &ArchiveExperimentParams{

		timeout: timeout,
	}
}

// NewArchiveExperimentParamsWithContext creates a new ArchiveExperimentParams object
// with the default values initialized, and the ability to set a context for a request
func NewArchiveExperimentParamsWithContext(ctx context.Context) *ArchiveExperimentParams {
	var ()
	return &ArchiveExperimentParams{

		Context: ctx,
	}
}

// NewArchiveExperimentParamsWithHTTPClient creates a new ArchiveExperimentParams object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewArchiveExperimentParamsWithHTTPClient(client *http.Client) *ArchiveExperimentParams {
	var ()
	return &ArchiveExperimentParams{
		HTTPClient: client,
	}
}

/*ArchiveExperimentParams contains all the parameters to send to the API endpoint
for the archive experiment operation typically these are written to a http.Request
*/
type ArchiveExperimentParams struct {

	/*ID*/
	ID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the archive experiment params
func (o *ArchiveExperimentParams) WithTimeout(timeout time.Duration) *ArchiveExperimentParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the archive experiment params
func (o *ArchiveExperimentParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the archive experiment params
func (o *ArchiveExperimentParams) WithContext(ctx context.Context) *ArchiveExperimentParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the archive experiment params
func (o *ArchiveExperimentParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the archive experiment params
func (o *ArchiveExperimentParams) WithHTTPClient(client *http.Client) *ArchiveExperimentParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the archive experiment params
func (o *ArchiveExperimentParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithID adds the id to the archive experiment params
func (o *ArchiveExperimentParams) WithID(id string) *ArchiveExperimentParams {
	o.SetID(id)
	return o
}

// SetID adds the id to the archive experiment params
func (o *ArchiveExperimentParams) SetID(id string) {
	o.ID = id
}

// WriteToRequest writes these params to a swagger request
func (o *ArchiveExperimentParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

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
