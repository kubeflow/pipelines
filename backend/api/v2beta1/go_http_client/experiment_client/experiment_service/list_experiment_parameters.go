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
	"github.com/go-openapi/swag"

	strfmt "github.com/go-openapi/strfmt"
)

// NewListExperimentParams creates a new ListExperimentParams object
// with the default values initialized.
func NewListExperimentParams() *ListExperimentParams {
	var ()
	return &ListExperimentParams{

		timeout: cr.DefaultTimeout,
	}
}

// NewListExperimentParamsWithTimeout creates a new ListExperimentParams object
// with the default values initialized, and the ability to set a timeout on a request
func NewListExperimentParamsWithTimeout(timeout time.Duration) *ListExperimentParams {
	var ()
	return &ListExperimentParams{

		timeout: timeout,
	}
}

// NewListExperimentParamsWithContext creates a new ListExperimentParams object
// with the default values initialized, and the ability to set a context for a request
func NewListExperimentParamsWithContext(ctx context.Context) *ListExperimentParams {
	var ()
	return &ListExperimentParams{

		Context: ctx,
	}
}

// NewListExperimentParamsWithHTTPClient creates a new ListExperimentParams object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewListExperimentParamsWithHTTPClient(client *http.Client) *ListExperimentParams {
	var ()
	return &ListExperimentParams{
		HTTPClient: client,
	}
}

/*ListExperimentParams contains all the parameters to send to the API endpoint
for the list experiment operation typically these are written to a http.Request
*/
type ListExperimentParams struct {

	/*Filter
	  A url-encoded, JSON-serialized Filter protocol buffer (see
	[filter.proto](https://github.com/kubeflow/pipelines/blob/master/backend/api/filter.proto)).

	*/
	Filter *string
	/*Namespace
	  Which namespace to filter the experiments on.

	*/
	Namespace *string
	/*PageSize
	  The number of experiments to be listed per page. If there are more
	experiments than this number, the response message will contain a
	nextPageToken field you can use to fetch the next page.

	*/
	PageSize *int32
	/*PageToken
	  A page token to request the next page of results. The token is acquried
	from the nextPageToken field of the response from the previous
	ListExperiment call or can be omitted when fetching the first page.

	*/
	PageToken *string
	/*SortBy
	  Can be format of "field_name", "field_name asc" or "field_name desc"
	Ascending by default.

	*/
	SortBy *string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the list experiment params
func (o *ListExperimentParams) WithTimeout(timeout time.Duration) *ListExperimentParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list experiment params
func (o *ListExperimentParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list experiment params
func (o *ListExperimentParams) WithContext(ctx context.Context) *ListExperimentParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list experiment params
func (o *ListExperimentParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list experiment params
func (o *ListExperimentParams) WithHTTPClient(client *http.Client) *ListExperimentParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list experiment params
func (o *ListExperimentParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithFilter adds the filter to the list experiment params
func (o *ListExperimentParams) WithFilter(filter *string) *ListExperimentParams {
	o.SetFilter(filter)
	return o
}

// SetFilter adds the filter to the list experiment params
func (o *ListExperimentParams) SetFilter(filter *string) {
	o.Filter = filter
}

// WithNamespace adds the namespace to the list experiment params
func (o *ListExperimentParams) WithNamespace(namespace *string) *ListExperimentParams {
	o.SetNamespace(namespace)
	return o
}

// SetNamespace adds the namespace to the list experiment params
func (o *ListExperimentParams) SetNamespace(namespace *string) {
	o.Namespace = namespace
}

// WithPageSize adds the pageSize to the list experiment params
func (o *ListExperimentParams) WithPageSize(pageSize *int32) *ListExperimentParams {
	o.SetPageSize(pageSize)
	return o
}

// SetPageSize adds the pageSize to the list experiment params
func (o *ListExperimentParams) SetPageSize(pageSize *int32) {
	o.PageSize = pageSize
}

// WithPageToken adds the pageToken to the list experiment params
func (o *ListExperimentParams) WithPageToken(pageToken *string) *ListExperimentParams {
	o.SetPageToken(pageToken)
	return o
}

// SetPageToken adds the pageToken to the list experiment params
func (o *ListExperimentParams) SetPageToken(pageToken *string) {
	o.PageToken = pageToken
}

// WithSortBy adds the sortBy to the list experiment params
func (o *ListExperimentParams) WithSortBy(sortBy *string) *ListExperimentParams {
	o.SetSortBy(sortBy)
	return o
}

// SetSortBy adds the sortBy to the list experiment params
func (o *ListExperimentParams) SetSortBy(sortBy *string) {
	o.SortBy = sortBy
}

// WriteToRequest writes these params to a swagger request
func (o *ListExperimentParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

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
