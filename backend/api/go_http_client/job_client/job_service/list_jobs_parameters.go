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
	"github.com/go-openapi/swag"

	strfmt "github.com/go-openapi/strfmt"
)

// NewListJobsParams creates a new ListJobsParams object
// with the default values initialized.
func NewListJobsParams() *ListJobsParams {
	var (
		resourceReferenceKeyTypeDefault = string("UNKNOWN_RESOURCE_TYPE")
	)
	return &ListJobsParams{
		ResourceReferenceKeyType: &resourceReferenceKeyTypeDefault,

		timeout: cr.DefaultTimeout,
	}
}

// NewListJobsParamsWithTimeout creates a new ListJobsParams object
// with the default values initialized, and the ability to set a timeout on a request
func NewListJobsParamsWithTimeout(timeout time.Duration) *ListJobsParams {
	var (
		resourceReferenceKeyTypeDefault = string("UNKNOWN_RESOURCE_TYPE")
	)
	return &ListJobsParams{
		ResourceReferenceKeyType: &resourceReferenceKeyTypeDefault,

		timeout: timeout,
	}
}

// NewListJobsParamsWithContext creates a new ListJobsParams object
// with the default values initialized, and the ability to set a context for a request
func NewListJobsParamsWithContext(ctx context.Context) *ListJobsParams {
	var (
		resourceReferenceKeyTypeDefault = string("UNKNOWN_RESOURCE_TYPE")
	)
	return &ListJobsParams{
		ResourceReferenceKeyType: &resourceReferenceKeyTypeDefault,

		Context: ctx,
	}
}

// NewListJobsParamsWithHTTPClient creates a new ListJobsParams object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewListJobsParamsWithHTTPClient(client *http.Client) *ListJobsParams {
	var (
		resourceReferenceKeyTypeDefault = string("UNKNOWN_RESOURCE_TYPE")
	)
	return &ListJobsParams{
		ResourceReferenceKeyType: &resourceReferenceKeyTypeDefault,
		HTTPClient:               client,
	}
}

/*
ListJobsParams contains all the parameters to send to the API endpoint
for the list jobs operation typically these are written to a http.Request
*/
type ListJobsParams struct {

	/*Filter
	  A url-encoded, JSON-serialized Filter protocol buffer (see
	[filter.proto](https://github.com/kubeflow/pipelines/blob/master/backend/api/filter.proto)).

	*/
	Filter *string
	/*PageSize
	  The number of jobs to be listed per page. If there are more jobs than this
	number, the response message will contain a nextPageToken field you can use
	to fetch the next page.

	*/
	PageSize *int32
	/*PageToken
	  A page token to request the next page of results. The token is acquried
	from the nextPageToken field of the response from the previous
	ListJobs call or can be omitted when fetching the first page.

	*/
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
	  Can be format of "field_name", "field_name asc" or "field_name desc".
	Ascending by default.

	*/
	SortBy *string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the list jobs params
func (o *ListJobsParams) WithTimeout(timeout time.Duration) *ListJobsParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list jobs params
func (o *ListJobsParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list jobs params
func (o *ListJobsParams) WithContext(ctx context.Context) *ListJobsParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list jobs params
func (o *ListJobsParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list jobs params
func (o *ListJobsParams) WithHTTPClient(client *http.Client) *ListJobsParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list jobs params
func (o *ListJobsParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithFilter adds the filter to the list jobs params
func (o *ListJobsParams) WithFilter(filter *string) *ListJobsParams {
	o.SetFilter(filter)
	return o
}

// SetFilter adds the filter to the list jobs params
func (o *ListJobsParams) SetFilter(filter *string) {
	o.Filter = filter
}

// WithPageSize adds the pageSize to the list jobs params
func (o *ListJobsParams) WithPageSize(pageSize *int32) *ListJobsParams {
	o.SetPageSize(pageSize)
	return o
}

// SetPageSize adds the pageSize to the list jobs params
func (o *ListJobsParams) SetPageSize(pageSize *int32) {
	o.PageSize = pageSize
}

// WithPageToken adds the pageToken to the list jobs params
func (o *ListJobsParams) WithPageToken(pageToken *string) *ListJobsParams {
	o.SetPageToken(pageToken)
	return o
}

// SetPageToken adds the pageToken to the list jobs params
func (o *ListJobsParams) SetPageToken(pageToken *string) {
	o.PageToken = pageToken
}

// WithResourceReferenceKeyID adds the resourceReferenceKeyID to the list jobs params
func (o *ListJobsParams) WithResourceReferenceKeyID(resourceReferenceKeyID *string) *ListJobsParams {
	o.SetResourceReferenceKeyID(resourceReferenceKeyID)
	return o
}

// SetResourceReferenceKeyID adds the resourceReferenceKeyId to the list jobs params
func (o *ListJobsParams) SetResourceReferenceKeyID(resourceReferenceKeyID *string) {
	o.ResourceReferenceKeyID = resourceReferenceKeyID
}

// WithResourceReferenceKeyType adds the resourceReferenceKeyType to the list jobs params
func (o *ListJobsParams) WithResourceReferenceKeyType(resourceReferenceKeyType *string) *ListJobsParams {
	o.SetResourceReferenceKeyType(resourceReferenceKeyType)
	return o
}

// SetResourceReferenceKeyType adds the resourceReferenceKeyType to the list jobs params
func (o *ListJobsParams) SetResourceReferenceKeyType(resourceReferenceKeyType *string) {
	o.ResourceReferenceKeyType = resourceReferenceKeyType
}

// WithSortBy adds the sortBy to the list jobs params
func (o *ListJobsParams) WithSortBy(sortBy *string) *ListJobsParams {
	o.SetSortBy(sortBy)
	return o
}

// SetSortBy adds the sortBy to the list jobs params
func (o *ListJobsParams) SetSortBy(sortBy *string) {
	o.SortBy = sortBy
}

// WriteToRequest writes these params to a swagger request
func (o *ListJobsParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

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
