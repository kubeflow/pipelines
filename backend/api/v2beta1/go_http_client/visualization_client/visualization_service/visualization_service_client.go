// Code generated by go-swagger; DO NOT EDIT.

package visualization_service

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
)

// New creates a new visualization service API client.
func New(transport runtime.ClientTransport, formats strfmt.Registry) ClientService {
	return &Client{transport: transport, formats: formats}
}

/*
Client for visualization service API
*/
type Client struct {
	transport runtime.ClientTransport
	formats   strfmt.Registry
}

// ClientOption is the option for Client methods
type ClientOption func(*runtime.ClientOperation)

// ClientService is the interface for Client methods
type ClientService interface {
	VisualizationServiceCreateVisualizationV1(params *VisualizationServiceCreateVisualizationV1Params, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*VisualizationServiceCreateVisualizationV1OK, error)

	SetTransport(transport runtime.ClientTransport)
}

/*
VisualizationServiceCreateVisualizationV1 visualization service create visualization v1 API
*/
func (a *Client) VisualizationServiceCreateVisualizationV1(params *VisualizationServiceCreateVisualizationV1Params, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*VisualizationServiceCreateVisualizationV1OK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewVisualizationServiceCreateVisualizationV1Params()
	}
	op := &runtime.ClientOperation{
		ID:                 "VisualizationService_CreateVisualizationV1",
		Method:             "POST",
		PathPattern:        "/apis/v2beta1/visualizations/{namespace}",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http"},
		Params:             params,
		Reader:             &VisualizationServiceCreateVisualizationV1Reader{formats: a.formats},
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
	success, ok := result.(*VisualizationServiceCreateVisualizationV1OK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*VisualizationServiceCreateVisualizationV1Default)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

// SetTransport changes the transport on the client
func (a *Client) SetTransport(transport runtime.ClientTransport) {
	a.transport = transport
}
