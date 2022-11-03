// Code generated by go-swagger; DO NOT EDIT.

package experiment_service

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"github.com/go-openapi/runtime"

	strfmt "github.com/go-openapi/strfmt"
)

// New creates a new experiment service API client.
func New(transport runtime.ClientTransport, formats strfmt.Registry) *Client {
	return &Client{transport: transport, formats: formats}
}

/*
Client for experiment service API
*/
type Client struct {
	transport runtime.ClientTransport
	formats   strfmt.Registry
}

/*
ArchiveExperimentV1 archives an experiment and the experiment s runs and jobs
*/
func (a *Client) ArchiveExperimentV1(params *ArchiveExperimentV1Params, authInfo runtime.ClientAuthInfoWriter) (*ArchiveExperimentV1OK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewArchiveExperimentV1Params()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "ArchiveExperimentV1",
		Method:             "POST",
		PathPattern:        "/apis/v1beta1/experiments/{id}:archive",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &ArchiveExperimentV1Reader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	return result.(*ArchiveExperimentV1OK), nil

}

/*
CreateExperimentV1 creates a new experiment
*/
func (a *Client) CreateExperimentV1(params *CreateExperimentV1Params, authInfo runtime.ClientAuthInfoWriter) (*CreateExperimentV1OK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewCreateExperimentV1Params()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "CreateExperimentV1",
		Method:             "POST",
		PathPattern:        "/apis/v1beta1/experiments",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &CreateExperimentV1Reader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	return result.(*CreateExperimentV1OK), nil

}

/*
DeleteExperimentV1 deletes an experiment without deleting the experiment s runs and jobs to avoid unexpected behaviors delete an experiment s runs and jobs before deleting the experiment
*/
func (a *Client) DeleteExperimentV1(params *DeleteExperimentV1Params, authInfo runtime.ClientAuthInfoWriter) (*DeleteExperimentV1OK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewDeleteExperimentV1Params()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "DeleteExperimentV1",
		Method:             "DELETE",
		PathPattern:        "/apis/v1beta1/experiments/{id}",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &DeleteExperimentV1Reader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	return result.(*DeleteExperimentV1OK), nil

}

/*
GetExperimentV1 finds a specific experiment by ID
*/
func (a *Client) GetExperimentV1(params *GetExperimentV1Params, authInfo runtime.ClientAuthInfoWriter) (*GetExperimentV1OK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewGetExperimentV1Params()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "GetExperimentV1",
		Method:             "GET",
		PathPattern:        "/apis/v1beta1/experiments/{id}",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &GetExperimentV1Reader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	return result.(*GetExperimentV1OK), nil

}

/*
ListExperimentV1 finds all experiments supports pagination and sorting on certain fields
*/
func (a *Client) ListExperimentV1(params *ListExperimentV1Params, authInfo runtime.ClientAuthInfoWriter) (*ListExperimentV1OK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewListExperimentV1Params()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "ListExperimentV1",
		Method:             "GET",
		PathPattern:        "/apis/v1beta1/experiments",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &ListExperimentV1Reader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	return result.(*ListExperimentV1OK), nil

}

/*
UnarchiveExperimentV1 restores an archived experiment the experiment s archived runs and jobs will stay archived
*/
func (a *Client) UnarchiveExperimentV1(params *UnarchiveExperimentV1Params, authInfo runtime.ClientAuthInfoWriter) (*UnarchiveExperimentV1OK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewUnarchiveExperimentV1Params()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "UnarchiveExperimentV1",
		Method:             "POST",
		PathPattern:        "/apis/v1beta1/experiments/{id}:unarchive",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &UnarchiveExperimentV1Reader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	return result.(*UnarchiveExperimentV1OK), nil

}

// SetTransport changes the transport on the client
func (a *Client) SetTransport(transport runtime.ClientTransport) {
	a.transport = transport
}
