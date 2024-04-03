// Copyright 2018-2023 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api_server_v2

import (
	"fmt"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	apiclient "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_client"
	params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_client/pipeline_service"
	model "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_model"
	"github.com/kubeflow/pipelines/backend/src/common/client/api_server"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"golang.org/x/net/context"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
)

type PipelineInterface interface {
	Create(params *params.PipelineServiceCreatePipelineParams) (*model.V2beta1Pipeline, error)
	CreatePipelineAndVersion(params *params.PipelineServiceCreatePipelineAndVersionParams) (*model.V2beta1Pipeline, error)
	Get(params *params.PipelineServiceGetPipelineParams) (*model.V2beta1Pipeline, error)
	Delete(params *params.PipelineServiceDeletePipelineParams) error
	//GetTemplate(params *params.GetTemplateParams) (template.Template, error)
	List(params *params.PipelineServiceListPipelinesParams) ([]*model.V2beta1Pipeline, int, string, error)
	ListAll(params *params.PipelineServiceListPipelinesParams, maxResultSize int) (
		[]*model.V2beta1Pipeline, error)
	// UpdateDefaultVersion(params *params.UpdatePipelineDefaultVersionParams) error
}

type PipelineClient struct {
	apiClient      *apiclient.Pipeline
	authInfoWriter runtime.ClientAuthInfoWriter
}

func NewPipelineClient(clientConfig clientcmd.ClientConfig, debug bool) (
	*PipelineClient, error) {

	runtime, err := api_server.NewHTTPRuntime(clientConfig, debug)
	if err != nil {
		return nil, fmt.Errorf("Error occurred when creating pipeline client: %w", err)
	}

	apiClient := apiclient.New(runtime, strfmt.Default)

	// Creating pipeline client
	return &PipelineClient{
		apiClient: apiClient,
	}, nil
}

func NewKubeflowInClusterPipelineClient(namespace string, debug bool) (
	*PipelineClient, error) {

	runtime := api_server.NewKubeflowInClusterHTTPRuntime(namespace, debug)

	apiClient := apiclient.New(runtime, strfmt.Default)

	// Creating pipeline client
	return &PipelineClient{
		apiClient:      apiClient,
		authInfoWriter: api_server.SATokenVolumeProjectionAuth,
	}, nil
}

func (c *PipelineClient) Create(parameters *params.PipelineServiceCreatePipelineParams) (*model.V2beta1Pipeline,
	error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()

	parameters.Context = ctx
	response, err := c.apiClient.PipelineService.PipelineServiceCreatePipeline(parameters, c.authInfoWriter)
	if err != nil {
		if defaultError, ok := err.(*params.PipelineServiceCreatePipelineDefault); ok {
			err = api_server.CreateErrorFromAPIStatus(defaultError.Payload.Message, defaultError.Payload.Code)
		} else {
			err = api_server.CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return nil, util.NewUserError(err,
			fmt.Sprintf("Failed to create pipeline. Params: '%v'", parameters),
			fmt.Sprintf("Failed to create pipeline '%v'", parameters.Body.DisplayName))
	}

	return response.Payload, nil
}

func (c *PipelineClient) CreatePipelineAndVersion(parameters *params.PipelineServiceCreatePipelineAndVersionParams) (*model.V2beta1Pipeline,
	error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()

	parameters.Context = ctx
	response, err := c.apiClient.PipelineService.PipelineServiceCreatePipelineAndVersion(parameters, c.authInfoWriter)
	if err != nil {
		if defaultError, ok := err.(*params.PipelineServiceCreatePipelineAndVersionDefault); ok {
			err = api_server.CreateErrorFromAPIStatus(defaultError.Payload.Message, defaultError.Payload.Code)
		} else {
			err = api_server.CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return nil, util.NewUserError(err,
			fmt.Sprintf("Failed to create pipeline and version. Params: '%v'", parameters),
			fmt.Sprintf("Failed to create pipeline and version '%v'", parameters.Body.Pipeline.DisplayName))
	}

	return response.Payload, nil
}

func (c *PipelineClient) Get(parameters *params.PipelineServiceGetPipelineParams) (*model.V2beta1Pipeline,
	error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	response, err := c.apiClient.PipelineService.PipelineServiceGetPipeline(parameters, c.authInfoWriter)
	if err != nil {
		if defaultError, ok := err.(*params.PipelineServiceGetPipelineDefault); ok {
			err = api_server.CreateErrorFromAPIStatus(defaultError.Payload.Message, defaultError.Payload.Code)
		} else {
			err = api_server.CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return nil, util.NewUserError(err,
			fmt.Sprintf("Failed to get pipeline. Params: '%v'", parameters),
			fmt.Sprintf("Failed to get pipeline '%v'", parameters.PipelineID))
	}

	return response.Payload, nil
}

func (c *PipelineClient) Delete(parameters *params.PipelineServiceDeletePipelineParams) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	_, err := c.apiClient.PipelineService.PipelineServiceDeletePipeline(parameters, c.authInfoWriter)
	if err != nil {
		if defaultError, ok := err.(*params.PipelineServiceDeletePipelineDefault); ok {
			err = api_server.CreateErrorFromAPIStatus(defaultError.Payload.Message, defaultError.Payload.Code)
		} else {
			err = api_server.CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return util.NewUserError(err,
			fmt.Sprintf("Failed to delete pipeline. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to delete pipeline '%v'", parameters.PipelineID))
	}

	return nil
}

func (c *PipelineClient) DeletePipelineVersion(parameters *params.PipelineServiceDeletePipelineVersionParams) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	_, err := c.apiClient.PipelineService.PipelineServiceDeletePipelineVersion(parameters, c.authInfoWriter)
	if err != nil {
		if defaultError, ok := err.(*params.PipelineServiceDeletePipelineVersionDefault); ok {
			err = api_server.CreateErrorFromAPIStatus(defaultError.Payload.Message, defaultError.Payload.Code)
		} else {
			err = api_server.CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return util.NewUserError(err,
			fmt.Sprintf("Failed to delete pipeline version. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to delete pipeline version '%v'", parameters.PipelineVersionID))
	}
	return nil
}

func (c *PipelineClient) List(parameters *params.PipelineServiceListPipelinesParams) (
	[]*model.V2beta1Pipeline, int, string, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	response, err := c.apiClient.PipelineService.PipelineServiceListPipelines(parameters, c.authInfoWriter)
	if err != nil {
		if defaultError, ok := err.(*params.PipelineServiceListPipelinesDefault); ok {
			err = api_server.CreateErrorFromAPIStatus(defaultError.Payload.Message, defaultError.Payload.Code)
		} else {
			err = api_server.CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return nil, 0, "", util.NewUserError(err,
			fmt.Sprintf("Failed to list pipelines. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to list pipelines"))
	}

	return response.Payload.Pipelines, int(response.Payload.TotalSize), response.Payload.NextPageToken, nil
}

func (c *PipelineClient) ListAll(parameters *params.PipelineServiceListPipelinesParams, maxResultSize int) (
	[]*model.V2beta1Pipeline, error) {
	return listAllForPipeline(c, parameters, maxResultSize)
}

func listAllForPipeline(client PipelineInterface, parameters *params.PipelineServiceListPipelinesParams,
	maxResultSize int) ([]*model.V2beta1Pipeline, error) {
	if maxResultSize < 0 {
		maxResultSize = 0
	}

	allResults := make([]*model.V2beta1Pipeline, 0)
	firstCall := true
	for (firstCall || (parameters.PageToken != nil && *parameters.PageToken != "")) &&
		(len(allResults) < maxResultSize) {
		results, _, pageToken, err := client.List(parameters)
		if err != nil {
			return nil, err
		}
		allResults = append(allResults, results...)
		parameters.PageToken = util.StringPointer(pageToken)
		firstCall = false
	}
	if len(allResults) > maxResultSize {
		allResults = allResults[0:maxResultSize]
	}

	return allResults, nil
}

func (c *PipelineClient) CreatePipelineVersion(parameters *params.PipelineServiceCreatePipelineVersionParams) (*model.V2beta1PipelineVersion,
	error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()

	parameters.Context = ctx
	response, err := c.apiClient.PipelineService.PipelineServiceCreatePipelineVersion(parameters, c.authInfoWriter)
	if err != nil {
		if defaultError, ok := err.(*params.PipelineServiceCreatePipelineVersionDefault); ok {
			err = api_server.CreateErrorFromAPIStatus(defaultError.Payload.Message, defaultError.Payload.Code)
		} else {
			err = api_server.CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return nil, util.NewUserError(err,
			fmt.Sprintf("Failed to create pipeline version. Params: '%v'", parameters),
			fmt.Sprintf("Failed to create pipeline version for pipeline: '%v'", parameters.PipelineID))
	}

	return response.Payload, nil
}

func (c *PipelineClient) ListPipelineVersions(parameters *params.PipelineServiceListPipelineVersionsParams) (
	[]*model.V2beta1PipelineVersion, int, string, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	response, err := c.apiClient.PipelineService.PipelineServiceListPipelineVersions(parameters, c.authInfoWriter)
	if err != nil {
		if defaultError, ok := err.(*params.PipelineServiceListPipelineVersionsDefault); ok {
			err = api_server.CreateErrorFromAPIStatus(defaultError.Payload.Message, defaultError.Payload.Code)
		} else {
			err = api_server.CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return nil, 0, "", util.NewUserError(err,
			fmt.Sprintf("Failed to list pipeline versions. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to list pipeline versions"))
	}

	return response.Payload.PipelineVersions, int(response.Payload.TotalSize), response.Payload.NextPageToken, nil
}

func (c *PipelineClient) GetPipelineVersion(parameters *params.PipelineServiceGetPipelineVersionParams) (*model.V2beta1PipelineVersion,
	error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	response, err := c.apiClient.PipelineService.PipelineServiceGetPipelineVersion(parameters, c.authInfoWriter)
	if err != nil {
		if defaultError, ok := err.(*params.PipelineServiceGetPipelineVersionDefault); ok {
			err = api_server.CreateErrorFromAPIStatus(defaultError.Payload.Message, defaultError.Payload.Code)
		} else {
			err = api_server.CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return nil, util.NewUserError(err,
			fmt.Sprintf("Failed to get pipeline version. Params: '%v'", parameters),
			fmt.Sprintf("Failed to get pipeline version '%v'", parameters.PipelineVersionID))
	}

	return response.Payload, nil
}
