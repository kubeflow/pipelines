// Code generated by go-swagger; DO NOT EDIT.

package pipeline_service

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"github.com/go-openapi/runtime"

	strfmt "github.com/go-openapi/strfmt"
)

// New creates a new pipeline service API client.
func New(transport runtime.ClientTransport, formats strfmt.Registry) *Client {
	return &Client{transport: transport, formats: formats}
}

/*
Client for pipeline service API
*/
type Client struct {
	transport runtime.ClientTransport
	formats   strfmt.Registry
}

/*
CreatePipeline creates a pipeline
*/
func (a *Client) CreatePipeline(params *CreatePipelineParams, authInfo runtime.ClientAuthInfoWriter) (*CreatePipelineOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewCreatePipelineParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "CreatePipeline",
		Method:             "POST",
		PathPattern:        "/apis/v1beta1/pipelines",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http"},
		Params:             params,
		Reader:             &CreatePipelineReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	success, ok := result.(*CreatePipelineOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*CreatePipelineDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
CreatePipelineVersion adds a pipeline version to the specified pipeline
*/
func (a *Client) CreatePipelineVersion(params *CreatePipelineVersionParams, authInfo runtime.ClientAuthInfoWriter) (*CreatePipelineVersionOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewCreatePipelineVersionParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "CreatePipelineVersion",
		Method:             "POST",
		PathPattern:        "/apis/v1beta1/pipeline_versions",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http"},
		Params:             params,
		Reader:             &CreatePipelineVersionReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	success, ok := result.(*CreatePipelineVersionOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*CreatePipelineVersionDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
DeletePipeline deletes a pipeline and its pipeline versions
*/
func (a *Client) DeletePipeline(params *DeletePipelineParams, authInfo runtime.ClientAuthInfoWriter) (*DeletePipelineOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewDeletePipelineParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "DeletePipeline",
		Method:             "DELETE",
		PathPattern:        "/apis/v1beta1/pipelines/{id}",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http"},
		Params:             params,
		Reader:             &DeletePipelineReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	success, ok := result.(*DeletePipelineOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*DeletePipelineDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
DeletePipelineVersion deletes a pipeline version by pipeline version ID if the deleted pipeline version is the default pipeline version the pipeline s default version changes to the pipeline s most recent pipeline version if there are no remaining pipeline versions the pipeline will have no default version examines the run service api ipynb notebook to learn more about creating a run using a pipeline version https github com kubeflow pipelines blob master tools benchmarks run service api ipynb
*/
func (a *Client) DeletePipelineVersion(params *DeletePipelineVersionParams, authInfo runtime.ClientAuthInfoWriter) (*DeletePipelineVersionOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewDeletePipelineVersionParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "DeletePipelineVersion",
		Method:             "DELETE",
		PathPattern:        "/apis/v1beta1/pipeline_versions/{version_id}",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http"},
		Params:             params,
		Reader:             &DeletePipelineVersionReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	success, ok := result.(*DeletePipelineVersionOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*DeletePipelineVersionDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
GetPipeline finds a specific pipeline by ID
*/
func (a *Client) GetPipeline(params *GetPipelineParams, authInfo runtime.ClientAuthInfoWriter) (*GetPipelineOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewGetPipelineParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "GetPipeline",
		Method:             "GET",
		PathPattern:        "/apis/v1beta1/pipelines/{id}",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http"},
		Params:             params,
		Reader:             &GetPipelineReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	success, ok := result.(*GetPipelineOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*GetPipelineDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
GetPipelineVersion gets a pipeline version by pipeline version ID
*/
func (a *Client) GetPipelineVersion(params *GetPipelineVersionParams, authInfo runtime.ClientAuthInfoWriter) (*GetPipelineVersionOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewGetPipelineVersionParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "GetPipelineVersion",
		Method:             "GET",
		PathPattern:        "/apis/v1beta1/pipeline_versions/{version_id}",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http"},
		Params:             params,
		Reader:             &GetPipelineVersionReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	success, ok := result.(*GetPipelineVersionOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*GetPipelineVersionDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
GetPipelineVersionTemplate returns a y a m l template that contains the specified pipeline version s description parameters and metadata
*/
func (a *Client) GetPipelineVersionTemplate(params *GetPipelineVersionTemplateParams, authInfo runtime.ClientAuthInfoWriter) (*GetPipelineVersionTemplateOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewGetPipelineVersionTemplateParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "GetPipelineVersionTemplate",
		Method:             "GET",
		PathPattern:        "/apis/v1beta1/pipeline_versions/{version_id}/templates",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http"},
		Params:             params,
		Reader:             &GetPipelineVersionTemplateReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	success, ok := result.(*GetPipelineVersionTemplateOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*GetPipelineVersionTemplateDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
GetTemplate returns a single y a m l template that contains the description parameters and metadata associated with the pipeline provided
*/
func (a *Client) GetTemplate(params *GetTemplateParams, authInfo runtime.ClientAuthInfoWriter) (*GetTemplateOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewGetTemplateParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "GetTemplate",
		Method:             "GET",
		PathPattern:        "/apis/v1beta1/pipelines/{id}/templates",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http"},
		Params:             params,
		Reader:             &GetTemplateReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	success, ok := result.(*GetTemplateOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*GetTemplateDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
ListPipelineVersions lists all pipeline versions of a given pipeline
*/
func (a *Client) ListPipelineVersions(params *ListPipelineVersionsParams, authInfo runtime.ClientAuthInfoWriter) (*ListPipelineVersionsOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewListPipelineVersionsParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "ListPipelineVersions",
		Method:             "GET",
		PathPattern:        "/apis/v1beta1/pipeline_versions",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http"},
		Params:             params,
		Reader:             &ListPipelineVersionsReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	success, ok := result.(*ListPipelineVersionsOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*ListPipelineVersionsDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
ListPipelines finds all pipelines
*/
func (a *Client) ListPipelines(params *ListPipelinesParams, authInfo runtime.ClientAuthInfoWriter) (*ListPipelinesOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewListPipelinesParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "ListPipelines",
		Method:             "GET",
		PathPattern:        "/apis/v1beta1/pipelines",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http"},
		Params:             params,
		Reader:             &ListPipelinesReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	success, ok := result.(*ListPipelinesOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*ListPipelinesDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
UpdatePipelineDefaultVersion updates the default pipeline version of a specific pipeline
*/
func (a *Client) UpdatePipelineDefaultVersion(params *UpdatePipelineDefaultVersionParams, authInfo runtime.ClientAuthInfoWriter) (*UpdatePipelineDefaultVersionOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewUpdatePipelineDefaultVersionParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "UpdatePipelineDefaultVersion",
		Method:             "POST",
		PathPattern:        "/apis/v1beta1/pipelines/{pipeline_id}/default_version/{version_id}",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http"},
		Params:             params,
		Reader:             &UpdatePipelineDefaultVersionReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	success, ok := result.(*UpdatePipelineDefaultVersionOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*UpdatePipelineDefaultVersionDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

// SetTransport changes the transport on the client
func (a *Client) SetTransport(transport runtime.ClientTransport) {
	a.transport = transport
}
