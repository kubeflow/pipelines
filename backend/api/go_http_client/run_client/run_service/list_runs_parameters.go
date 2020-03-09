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
	"github.com/go-openapi/swag"

	strfmt "github.com/go-openapi/strfmt"
)

// NewListRunsParams creates a new ListRunsParams object
// with the default values initialized.
func NewListRunsParams() *ListRunsParams {
	var (
		resourceReferenceKeyTypeDefault = string("UNKNOWN_RESOURCE_TYPE")
	)
	return &ListRunsParams{
		ResourceReferenceKeyType: &resourceReferenceKeyTypeDefault,

		timeout: cr.DefaultTimeout,
	}
}

// NewListRunsParamsWithTimeout creates a new ListRunsParams object
// with the default values initialized, and the ability to set a timeout on a request
func NewListRunsParamsWithTimeout(timeout time.Duration) *ListRunsParams {
	var (
		resourceReferenceKeyTypeDefault = string("UNKNOWN_RESOURCE_TYPE")
	)
	return &ListRunsParams{
		ResourceReferenceKeyType: &resourceReferenceKeyTypeDefault,

		timeout: timeout,
	}
}

// NewListRunsParamsWithContext creates a new ListRunsParams object
// with the default values initialized, and the ability to set a context for a request
func NewListRunsParamsWithContext(ctx context.Context) *ListRunsParams {
	var (
		resourceReferenceKeyTypeDefault = string("UNKNOWN_RESOURCE_TYPE")
	)
	return &ListRunsParams{
		ResourceReferenceKeyType: &resourceReferenceKeyTypeDefault,

		Context: ctx,
	}
}

// NewListRunsParamsWithHTTPClient creates a new ListRunsParams object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewListRunsParamsWithHTTPClient(client *http.Client) *ListRunsParams {
	var (
		resourceReferenceKeyTypeDefault = string("UNKNOWN_RESOURCE_TYPE")
	)
	return &ListRunsParams{
		ResourceReferenceKeyType: &resourceReferenceKeyTypeDefault,
		HTTPClient:               client,
	}
}

/*ListRunsParams contains all the parameters to send to the API endpoint
for the list runs operation typically these are written to a http.Request
*/
type ListRunsParams struct {

	/*Filter
	  A url-encoded, JSON-serialized Filter protocol buffer (see
	[filter.proto](https://github.com/kubeflow/pipelines/
	blob/master/backend/api/filter.proto)).

	*/
	Filter *string
	/*PageSize*/
	PageSize *int32
	/*PageToken*/
	PageToken *string
	/*ResourceReferenceKeyID
	  The ID of the resource that referred to.

	*/
	ResourceReferenceKeyID *string
	/*ResourceReferenceKeyType
	  The type of the resource that referred to.

	*/
	ResourceReferenceKeyType *string
	/*SortBy
	  Can be format of "field_name", "field_name asc" or "field_name des"
	(Example, "name asc" or "id des"). Ascending by default.

	*/
	SortBy *string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the list runs params
func (o *ListRunsParams) WithTimeout(timeout time.Duration) *ListRunsParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list runs params
func (o *ListRunsParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list runs params
func (o *ListRunsParams) WithContext(ctx context.Context) *ListRunsParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list runs params
func (o *ListRunsParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list runs params
func (o *ListRunsParams) WithHTTPClient(client *http.Client) *ListRunsParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list runs params
func (o *ListRunsParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithFilter adds the filter to the list runs params
func (o *ListRunsParams) WithFilter(filter *string) *ListRunsParams {
	o.SetFilter(filter)
	return o
}

// SetFilter adds the filter to the list runs params
func (o *ListRunsParams) SetFilter(filter *string) {
	o.Filter = filter
}

// WithPageSize adds the pageSize to the list runs params
func (o *ListRunsParams) WithPageSize(pageSize *int32) *ListRunsParams {
	o.SetPageSize(pageSize)
	return o
}

// SetPageSize adds the pageSize to the list runs params
func (o *ListRunsParams) SetPageSize(pageSize *int32) {
	o.PageSize = pageSize
}

// WithPageToken adds the pageToken to the list runs params
func (o *ListRunsParams) WithPageToken(pageToken *string) *ListRunsParams {
	o.SetPageToken(pageToken)
	return o
}

// SetPageToken adds the pageToken to the list runs params
func (o *ListRunsParams) SetPageToken(pageToken *string) {
	o.PageToken = pageToken
}

// WithResourceReferenceKeyID adds the resourceReferenceKeyID to the list runs params
func (o *ListRunsParams) WithResourceReferenceKeyID(resourceReferenceKeyID *string) *ListRunsParams {
	o.SetResourceReferenceKeyID(resourceReferenceKeyID)
	return o
}

// SetResourceReferenceKeyID adds the resourceReferenceKeyId to the list runs params
func (o *ListRunsParams) SetResourceReferenceKeyID(resourceReferenceKeyID *string) {
	o.ResourceReferenceKeyID = resourceReferenceKeyID
}

// WithResourceReferenceKeyType adds the resourceReferenceKeyType to the list runs params
func (o *ListRunsParams) WithResourceReferenceKeyType(resourceReferenceKeyType *string) *ListRunsParams {
	o.SetResourceReferenceKeyType(resourceReferenceKeyType)
	return o
}

// SetResourceReferenceKeyType adds the resourceReferenceKeyType to the list runs params
func (o *ListRunsParams) SetResourceReferenceKeyType(resourceReferenceKeyType *string) {
	o.ResourceReferenceKeyType = resourceReferenceKeyType
}

// WithSortBy adds the sortBy to the list runs params
func (o *ListRunsParams) WithSortBy(sortBy *string) *ListRunsParams {
	o.SetSortBy(sortBy)
	return o
}

// SetSortBy adds the sortBy to the list runs params
func (o *ListRunsParams) SetSortBy(sortBy *string) {
	o.SortBy = sortBy
}

// WriteToRequest writes these params to a swagger request
func (o *ListRunsParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	if o.Filter != nil {

		// query param filter
		var qrFilter string
		if o.Filter != nil {
			qrFilter = *o.Filter
		}
		qFilter := qrFilter
		if qFilter != "" {
			if err := r.SetQueryParam("filter", qFilter); err != nil {
				return err
			}
		}

	}

	if o.PageSize != nil {

		// query param page_size
		var qrPageSize int32
		if o.PageSize != nil {
			qrPageSize = *o.PageSize
		}
		qPageSize := swag.FormatInt32(qrPageSize)
		if qPageSize != "" {
			if err := r.SetQueryParam("page_size", qPageSize); err != nil {
				return err
			}
		}

	}

	if o.PageToken != nil {

		// query param page_token
		var qrPageToken string
		if o.PageToken != nil {
			qrPageToken = *o.PageToken
		}
		qPageToken := qrPageToken
		if qPageToken != "" {
			if err := r.SetQueryParam("page_token", qPageToken); err != nil {
				return err
			}
		}

	}

	if o.ResourceReferenceKeyID != nil {

		// query param resource_reference_key.id
		var qrResourceReferenceKeyID string
		if o.ResourceReferenceKeyID != nil {
			qrResourceReferenceKeyID = *o.ResourceReferenceKeyID
		}
		qResourceReferenceKeyID := qrResourceReferenceKeyID
		if qResourceReferenceKeyID != "" {
			if err := r.SetQueryParam("resource_reference_key.id", qResourceReferenceKeyID); err != nil {
				return err
			}
		}

	}

	if o.ResourceReferenceKeyType != nil {

		// query param resource_reference_key.type
		var qrResourceReferenceKeyType string
		if o.ResourceReferenceKeyType != nil {
			qrResourceReferenceKeyType = *o.ResourceReferenceKeyType
		}
		qResourceReferenceKeyType := qrResourceReferenceKeyType
		if qResourceReferenceKeyType != "" {
			if err := r.SetQueryParam("resource_reference_key.type", qResourceReferenceKeyType); err != nil {
				return err
			}
		}

	}

	if o.SortBy != nil {

		// query param sort_by
		var qrSortBy string
		if o.SortBy != nil {
			qrSortBy = *o.SortBy
		}
		qSortBy := qrSortBy
		if qSortBy != "" {
			if err := r.SetQueryParam("sort_by", qSortBy); err != nil {
				return err
			}
		}

	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
