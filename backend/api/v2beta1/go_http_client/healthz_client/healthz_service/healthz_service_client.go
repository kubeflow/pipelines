// Code generated by go-swagger; DO NOT EDIT.

package healthz_service

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
)

// New creates a new healthz service API client.
func New(transport runtime.ClientTransport, formats strfmt.Registry) ClientService {
	return &Client{transport: transport, formats: formats}
}

/*
Client for healthz service API
*/
type Client struct {
	transport runtime.ClientTransport
	formats   strfmt.Registry
}

// ClientOption is the option for Client methods
type ClientOption func(*runtime.ClientOperation)

// ClientService is the interface for Client methods
type ClientService interface {
	HealthzServiceGetHealthz(params *HealthzServiceGetHealthzParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*HealthzServiceGetHealthzOK, error)

	SetTransport(transport runtime.ClientTransport)
}

/*
HealthzServiceGetHealthz gets healthz data
*/
func (a *Client) HealthzServiceGetHealthz(params *HealthzServiceGetHealthzParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*HealthzServiceGetHealthzOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewHealthzServiceGetHealthzParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "HealthzService_GetHealthz",
		Method:             "GET",
		PathPattern:        "/apis/v2beta1/healthz",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http"},
		Params:             params,
		Reader:             &HealthzServiceGetHealthzReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	}
	for _, opt := range opts {
		opt(op)
	}

	result, err := a.transport.Submit(op)
	if err != nil {
		return nil, err
	}
	success, ok := result.(*HealthzServiceGetHealthzOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*HealthzServiceGetHealthzDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

// SetTransport changes the transport on the client
func (a *Client) SetTransport(transport runtime.ClientTransport) {
	a.transport = transport
}
