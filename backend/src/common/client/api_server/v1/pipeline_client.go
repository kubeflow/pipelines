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

package api_server

import (
	"fmt"

	rt "github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	apiclient "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/pipeline_client"
	params "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/pipeline_client/pipeline_service"
	model "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/pipeline_model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/template"
	"github.com/kubeflow/pipelines/backend/src/common/client/api_server"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"golang.org/x/net/context"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
)

type PipelineInterface interface {
	Create(params *params.CreatePipelineV1Params) (*model.APIPipeline, error)
	Get(params *params.GetPipelineV1Params) (*model.APIPipeline, error)
	Delete(params *params.DeletePipelineV1Params) error
	GetTemplate(params *params.GetTemplateParams) (template.Template, error)
	List(params *params.ListPipelinesV1Params) ([]*model.APIPipeline, int, string, error)
	ListAll(params *params.ListPipelinesV1Params, maxResultSize int) (
		[]*model.APIPipeline, error)
	UpdateDefaultVersion(params *params.UpdatePipelineDefaultVersionV1Params) error
}

type PipelineClient struct {
	apiClient      *apiclient.Pipeline
	authInfoWriter rt.ClientAuthInfoWriter
}

func (c *PipelineClient) UpdateDefaultVersion(parameters *params.UpdatePipelineDefaultVersionV1Params) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()
	// Make service call
	parameters.Context = ctx
	_, err := c.apiClient.PipelineService.UpdatePipelineDefaultVersionV1(parameters, c.authInfoWriter)
	if err != nil {
		if defaultError, ok := err.(*params.GetPipelineV1Default); ok {
			err = api_server.CreateErrorFromAPIStatus(defaultError.Payload.Error, defaultError.Payload.Code)
		} else {
			err = api_server.CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return util.NewUserError(err,
			fmt.Sprintf("Failed to update pipeline. Params: '%v'", parameters),
			fmt.Sprintf("Failed to update pipeline '%v'", parameters.PipelineID))
	}

	return nil
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

func (c *PipelineClient) Create(parameters *params.CreatePipelineV1Params) (*model.APIPipeline,
	error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()

	parameters.Context = ctx
	response, err := c.apiClient.PipelineService.CreatePipelineV1(parameters, c.authInfoWriter)
	if err != nil {
		if defaultError, ok := err.(*params.CreatePipelineV1Default); ok {
			err = api_server.CreateErrorFromAPIStatus(defaultError.Payload.Error, defaultError.Payload.Code)
		} else {
			err = api_server.CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return nil, util.NewUserError(err,
			fmt.Sprintf("Failed to create pipeline. Params: '%v'", parameters),
			fmt.Sprintf("Failed to create pipeline from URL '%v'", parameters.Body.URL.PipelineURL))
	}

	return response.Payload, nil
}

func (c *PipelineClient) Get(parameters *params.GetPipelineV1Params) (*model.APIPipeline,
	error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	response, err := c.apiClient.PipelineService.GetPipelineV1(parameters, c.authInfoWriter)
	if err != nil {
		if defaultError, ok := err.(*params.GetPipelineV1Default); ok {
			err = api_server.CreateErrorFromAPIStatus(defaultError.Payload.Error, defaultError.Payload.Code)
		} else {
			err = api_server.CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return nil, util.NewUserError(err,
			fmt.Sprintf("Failed to get pipeline. Params: '%v'", parameters),
			fmt.Sprintf("Failed to get pipeline '%v'", parameters.ID))
	}

	return response.Payload, nil
}

func (c *PipelineClient) Delete(parameters *params.DeletePipelineV1Params) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	_, err := c.apiClient.PipelineService.DeletePipelineV1(parameters, c.authInfoWriter)
	if err != nil {
		if defaultError, ok := err.(*params.DeletePipelineV1Default); ok {
			err = api_server.CreateErrorFromAPIStatus(defaultError.Payload.Error, defaultError.Payload.Code)
		} else {
			err = api_server.CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return util.NewUserError(err,
			fmt.Sprintf("Failed to delete pipeline. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to delete pipeline '%v'", parameters.ID))
	}

	return nil
}

func (c *PipelineClient) DeletePipelineVersion(parameters *params.DeletePipelineVersionV1Params) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	_, err := c.apiClient.PipelineService.DeletePipelineVersionV1(parameters, c.authInfoWriter)
	if err != nil {
		if defaultError, ok := err.(*params.DeletePipelineVersionV1Default); ok {
			err = api_server.CreateErrorFromAPIStatus(defaultError.Payload.Error, defaultError.Payload.Code)
		} else {
			err = api_server.CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return util.NewUserError(err,
			fmt.Sprintf("Failed to delete pipeline version. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to delete pipeline version '%v'", parameters.VersionID))
	}
	return nil
}

func (c *PipelineClient) GetTemplate(parameters *params.GetTemplateParams) (template.Template, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	response, err := c.apiClient.PipelineService.GetTemplate(parameters, c.authInfoWriter)
	if err != nil {
		if defaultError, ok := err.(*params.GetTemplateDefault); ok {
			err = api_server.CreateErrorFromAPIStatus(defaultError.Payload.Error, defaultError.Payload.Code)
		} else {
			err = api_server.CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return nil, util.NewUserError(err,
			fmt.Sprintf("Failed to get template. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to get template for pipeline '%v'", parameters.ID))
	}

	// Unmarshal response
	return template.New([]byte(response.Payload.Template))
}

func (c *PipelineClient) List(parameters *params.ListPipelinesV1Params) (
	[]*model.APIPipeline, int, string, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	response, err := c.apiClient.PipelineService.ListPipelinesV1(parameters, c.authInfoWriter)
	if err != nil {
		if defaultError, ok := err.(*params.ListPipelinesV1Default); ok {
			err = api_server.CreateErrorFromAPIStatus(defaultError.Payload.Error, defaultError.Payload.Code)
		} else {
			err = api_server.CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return nil, 0, "", util.NewUserError(err,
			fmt.Sprintf("Failed to list pipelines. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to list pipelines"))
	}

	return response.Payload.Pipelines, int(response.Payload.TotalSize), response.Payload.NextPageToken, nil
}

func (c *PipelineClient) ListAll(parameters *params.ListPipelinesV1Params, maxResultSize int) (
	[]*model.APIPipeline, error) {
	return listAllForPipeline(c, parameters, maxResultSize)
}

func listAllForPipeline(client PipelineInterface, parameters *params.ListPipelinesV1Params,
	maxResultSize int) ([]*model.APIPipeline, error) {
	if maxResultSize < 0 {
		maxResultSize = 0
	}

	allResults := make([]*model.APIPipeline, 0)
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

func (c *PipelineClient) CreatePipelineVersion(parameters *params.CreatePipelineVersionV1Params) (*model.APIPipelineVersion,
	error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()

	parameters.Context = ctx
	response, err := c.apiClient.PipelineService.CreatePipelineVersionV1(parameters, c.authInfoWriter)
	if err != nil {
		if defaultError, ok := err.(*params.CreatePipelineVersionV1Default); ok {
			err = api_server.CreateErrorFromAPIStatus(defaultError.Payload.Error, defaultError.Payload.Code)
		} else {
			err = api_server.CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return nil, util.NewUserError(err,
			fmt.Sprintf("Failed to create pipeline version. Params: '%v'", parameters),
			fmt.Sprintf("Failed to create pipeline version from URL '%v'", parameters.Body.PackageURL.PipelineURL))
	}

	return response.Payload, nil
}

func (c *PipelineClient) ListPipelineVersions(parameters *params.ListPipelineVersionsV1Params) (
	[]*model.APIPipelineVersion, int, string, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	response, err := c.apiClient.PipelineService.ListPipelineVersionsV1(parameters, c.authInfoWriter)
	if err != nil {
		if defaultError, ok := err.(*params.ListPipelineVersionsV1Default); ok {
			err = api_server.CreateErrorFromAPIStatus(defaultError.Payload.Error, defaultError.Payload.Code)
		} else {
			err = api_server.CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return nil, 0, "", util.NewUserError(err,
			fmt.Sprintf("Failed to list pipeline versions. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to list pipeline versions"))
	}

	return response.Payload.Versions, int(response.Payload.TotalSize), response.Payload.NextPageToken, nil
}

func (c *PipelineClient) GetPipelineVersion(parameters *params.GetPipelineVersionV1Params) (*model.APIPipelineVersion,
	error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	response, err := c.apiClient.PipelineService.GetPipelineVersionV1(parameters, c.authInfoWriter)
	if err != nil {
		if defaultError, ok := err.(*params.GetPipelineVersionV1Default); ok {
			err = api_server.CreateErrorFromAPIStatus(defaultError.Payload.Error, defaultError.Payload.Code)
		} else {
			err = api_server.CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return nil, util.NewUserError(err,
			fmt.Sprintf("Failed to get pipeline version. Params: '%v'", parameters),
			fmt.Sprintf("Failed to get pipeline version '%v'", parameters.VersionID))
	}

	return response.Payload, nil
}

func (c *PipelineClient) GetPipelineVersionTemplate(parameters *params.GetPipelineVersionTemplateParams) (
	template.Template, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	response, err := c.apiClient.PipelineService.GetPipelineVersionTemplate(parameters, c.authInfoWriter)
	if err != nil {
		if defaultError, ok := err.(*params.GetPipelineVersionTemplateDefault); ok {
			err = api_server.CreateErrorFromAPIStatus(defaultError.Payload.Error, defaultError.Payload.Code)
		} else {
			err = api_server.CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return nil, util.NewUserError(err,
			fmt.Sprintf("Failed to get template. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to get template for pipeline version '%v'", parameters.VersionID))
	}

	// Unmarshal response
	return template.New([]byte(response.Payload.Template))
}
