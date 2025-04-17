// Code generated by go-swagger; DO NOT EDIT.

package recurring_run_service

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

// NewRecurringRunServiceListRecurringRunsParams creates a new RecurringRunServiceListRecurringRunsParams object
// with the default values initialized.
func NewRecurringRunServiceListRecurringRunsParams() *RecurringRunServiceListRecurringRunsParams {
	var ()
	return &RecurringRunServiceListRecurringRunsParams{

		timeout: cr.DefaultTimeout,
	}
}

// NewRecurringRunServiceListRecurringRunsParamsWithTimeout creates a new RecurringRunServiceListRecurringRunsParams object
// with the default values initialized, and the ability to set a timeout on a request
func NewRecurringRunServiceListRecurringRunsParamsWithTimeout(timeout time.Duration) *RecurringRunServiceListRecurringRunsParams {
	var ()
	return &RecurringRunServiceListRecurringRunsParams{

		timeout: timeout,
	}
}

// NewRecurringRunServiceListRecurringRunsParamsWithContext creates a new RecurringRunServiceListRecurringRunsParams object
// with the default values initialized, and the ability to set a context for a request
func NewRecurringRunServiceListRecurringRunsParamsWithContext(ctx context.Context) *RecurringRunServiceListRecurringRunsParams {
	var ()
	return &RecurringRunServiceListRecurringRunsParams{

		Context: ctx,
	}
}

// NewRecurringRunServiceListRecurringRunsParamsWithHTTPClient creates a new RecurringRunServiceListRecurringRunsParams object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewRecurringRunServiceListRecurringRunsParamsWithHTTPClient(client *http.Client) *RecurringRunServiceListRecurringRunsParams {
	var ()
	return &RecurringRunServiceListRecurringRunsParams{
		HTTPClient: client,
	}
}

/*
RecurringRunServiceListRecurringRunsParams contains all the parameters to send to the API endpoint
for the recurring run service list recurring runs operation typically these are written to a http.Request
*/
type RecurringRunServiceListRecurringRunsParams struct {

	/*ExperimentID
	  The ID of the experiment to be retrieved. If empty, list recurring runs across all experiments.

	*/
	ExperimentID *string
	/*Filter
	  A url-encoded, JSON-serialized Filter protocol buffer (see
	[filter.proto](https://github.com/kubeflow/pipelines/blob/master/backend/api/filter.proto)).

	*/
	Filter *string
	/*Namespace
	  Optional input. The namespace the recurring runs belong to.

	*/
	Namespace *string
	/*PageSize
	  The number of recurring runs to be listed per page. If there are more recurring runs
	than this number, the response message will contain a nextPageToken field you can use
	to fetch the next page.

	*/
	PageSize *int32
	/*PageToken
	  A page token to request the next page of results. The token is acquired
	from the nextPageToken field of the response from the previous
	ListRecurringRuns call or can be omitted when fetching the first page.

	*/
	PageToken *string
	/*SortBy
	  Can be formatted as "field_name", "field_name asc" or "field_name desc".
	Ascending by default.

	*/
	SortBy *string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the recurring run service list recurring runs params
func (o *RecurringRunServiceListRecurringRunsParams) WithTimeout(timeout time.Duration) *RecurringRunServiceListRecurringRunsParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the recurring run service list recurring runs params
func (o *RecurringRunServiceListRecurringRunsParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the recurring run service list recurring runs params
func (o *RecurringRunServiceListRecurringRunsParams) WithContext(ctx context.Context) *RecurringRunServiceListRecurringRunsParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the recurring run service list recurring runs params
func (o *RecurringRunServiceListRecurringRunsParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the recurring run service list recurring runs params
func (o *RecurringRunServiceListRecurringRunsParams) WithHTTPClient(client *http.Client) *RecurringRunServiceListRecurringRunsParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the recurring run service list recurring runs params
func (o *RecurringRunServiceListRecurringRunsParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithExperimentID adds the experimentID to the recurring run service list recurring runs params
func (o *RecurringRunServiceListRecurringRunsParams) WithExperimentID(experimentID *string) *RecurringRunServiceListRecurringRunsParams {
	o.SetExperimentID(experimentID)
	return o
}

// SetExperimentID adds the experimentId to the recurring run service list recurring runs params
func (o *RecurringRunServiceListRecurringRunsParams) SetExperimentID(experimentID *string) {
	o.ExperimentID = experimentID
}

// WithFilter adds the filter to the recurring run service list recurring runs params
func (o *RecurringRunServiceListRecurringRunsParams) WithFilter(filter *string) *RecurringRunServiceListRecurringRunsParams {
	o.SetFilter(filter)
	return o
}

// SetFilter adds the filter to the recurring run service list recurring runs params
func (o *RecurringRunServiceListRecurringRunsParams) SetFilter(filter *string) {
	o.Filter = filter
}

// WithNamespace adds the namespace to the recurring run service list recurring runs params
func (o *RecurringRunServiceListRecurringRunsParams) WithNamespace(namespace *string) *RecurringRunServiceListRecurringRunsParams {
	o.SetNamespace(namespace)
	return o
}

// SetNamespace adds the namespace to the recurring run service list recurring runs params
func (o *RecurringRunServiceListRecurringRunsParams) SetNamespace(namespace *string) {
	o.Namespace = namespace
}

// WithPageSize adds the pageSize to the recurring run service list recurring runs params
func (o *RecurringRunServiceListRecurringRunsParams) WithPageSize(pageSize *int32) *RecurringRunServiceListRecurringRunsParams {
	o.SetPageSize(pageSize)
	return o
}

// SetPageSize adds the pageSize to the recurring run service list recurring runs params
func (o *RecurringRunServiceListRecurringRunsParams) SetPageSize(pageSize *int32) {
	o.PageSize = pageSize
}

// WithPageToken adds the pageToken to the recurring run service list recurring runs params
func (o *RecurringRunServiceListRecurringRunsParams) WithPageToken(pageToken *string) *RecurringRunServiceListRecurringRunsParams {
	o.SetPageToken(pageToken)
	return o
}

// SetPageToken adds the pageToken to the recurring run service list recurring runs params
func (o *RecurringRunServiceListRecurringRunsParams) SetPageToken(pageToken *string) {
	o.PageToken = pageToken
}

// WithSortBy adds the sortBy to the recurring run service list recurring runs params
func (o *RecurringRunServiceListRecurringRunsParams) WithSortBy(sortBy *string) *RecurringRunServiceListRecurringRunsParams {
	o.SetSortBy(sortBy)
	return o
}

// SetSortBy adds the sortBy to the recurring run service list recurring runs params
func (o *RecurringRunServiceListRecurringRunsParams) SetSortBy(sortBy *string) {
	o.SortBy = sortBy
}

// WriteToRequest writes these params to a swagger request
func (o *RecurringRunServiceListRecurringRunsParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	if o.ExperimentID != nil {

		// query param experiment_id
		var qrExperimentID string
		if o.ExperimentID != nil {
			qrExperimentID = *o.ExperimentID
		}
		qExperimentID := qrExperimentID
		if qExperimentID != "" {
			if err := r.SetQueryParam("experiment_id", qExperimentID); err != nil {
				return err
			}
		}

	}

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
