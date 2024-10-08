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
	apiclient "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_client"
	params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_client/run_service"
	model "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_model"
	"github.com/kubeflow/pipelines/backend/src/common/client/api_server"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"golang.org/x/net/context"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
)

type RunInterface interface {
	Archive(params *params.RunServiceArchiveRunParams) error
	Create(params *params.RunServiceCreateRunParams) (*model.V2beta1Run, error)
	Get(params *params.RunServiceGetRunParams) (*model.V2beta1Run, error)
	List(params *params.RunServiceListRunsParams) ([]*model.V2beta1Run, int, string, error)
	ListAll(params *params.RunServiceListRunsParams, maxResultSize int) ([]*model.V2beta1Run, error)
	Unarchive(params *params.RunServiceUnarchiveRunParams) error
	Terminate(params *params.RunServiceTerminateRunParams) error
}

type RunClient struct {
	apiClient      *apiclient.Run
	authInfoWriter runtime.ClientAuthInfoWriter
}

func NewRunClient(clientConfig clientcmd.ClientConfig, debug bool) (
	*RunClient, error) {

	runtime, err := api_server.NewHTTPRuntime(clientConfig, debug)
	if err != nil {
		return nil, fmt.Errorf("Error occurred when creating run client: %w", err)
	}

	apiClient := apiclient.New(runtime, strfmt.Default)

	// Creating run client
	return &RunClient{
		apiClient: apiClient,
	}, nil
}

func NewKubeflowInClusterRunClient(namespace string, debug bool) (
	*RunClient, error) {

	runtime := api_server.NewKubeflowInClusterHTTPRuntime(namespace, debug)

	apiClient := apiclient.New(runtime, strfmt.Default)

	// Creating run client
	return &RunClient{
		apiClient:      apiClient,
		authInfoWriter: api_server.SATokenVolumeProjectionAuth,
	}, nil
}

func (c *RunClient) Create(parameters *params.RunServiceCreateRunParams) (*model.V2beta1Run, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	response, err := c.apiClient.RunService.RunServiceCreateRun(parameters, c.authInfoWriter)
	if err != nil {
		if defaultError, ok := err.(*params.RunServiceGetRunDefault); ok {
			err = api_server.CreateErrorFromAPIStatus(defaultError.Payload.Message, defaultError.Payload.Code)
		} else {
			err = api_server.CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return nil, util.NewUserError(err,
			fmt.Sprintf("Failed to create run. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to create run '%v'", parameters.Body.DisplayName))
	}

	return response.Payload, nil
}

func (c *RunClient) Get(parameters *params.RunServiceGetRunParams) (*model.V2beta1Run, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	response, err := c.apiClient.RunService.RunServiceGetRun(parameters, c.authInfoWriter)
	if err != nil {
		if defaultError, ok := err.(*params.RunServiceGetRunDefault); ok {
			err = api_server.CreateErrorFromAPIStatus(defaultError.Payload.Message, defaultError.Payload.Code)
		} else {
			err = api_server.CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return nil, util.NewUserError(err,
			fmt.Sprintf("Failed to get run. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to get run '%v'", parameters.RunID))
	}

	return response.Payload, nil
}

func (c *RunClient) Archive(parameters *params.RunServiceArchiveRunParams) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	_, err := c.apiClient.RunService.RunServiceArchiveRun(parameters, c.authInfoWriter)

	if err != nil {
		if defaultError, ok := err.(*params.RunServiceListRunsDefault); ok {
			err = api_server.CreateErrorFromAPIStatus(defaultError.Payload.Message, defaultError.Payload.Code)
		} else {
			err = api_server.CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return util.NewUserError(err,
			fmt.Sprintf("Failed to archive runs. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to archive runs"))
	}

	return nil
}

func (c *RunClient) Unarchive(parameters *params.RunServiceUnarchiveRunParams) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	_, err := c.apiClient.RunService.RunServiceUnarchiveRun(parameters, c.authInfoWriter)

	if err != nil {
		if defaultError, ok := err.(*params.RunServiceListRunsDefault); ok {
			err = api_server.CreateErrorFromAPIStatus(defaultError.Payload.Message, defaultError.Payload.Code)
		} else {
			err = api_server.CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return util.NewUserError(err,
			fmt.Sprintf("Failed to unarchive runs. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to unarchive runs"))
	}

	return nil
}

func (c *RunClient) Delete(parameters *params.RunServiceDeleteRunParams) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	_, err := c.apiClient.RunService.RunServiceDeleteRun(parameters, c.authInfoWriter)

	if err != nil {
		if defaultError, ok := err.(*params.RunServiceListRunsDefault); ok {
			err = api_server.CreateErrorFromAPIStatus(defaultError.Payload.Message, defaultError.Payload.Code)
		} else {
			err = api_server.CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return util.NewUserError(err,
			fmt.Sprintf("Failed to delete runs. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to delete runs"))
	}

	return nil
}

func (c *RunClient) List(parameters *params.RunServiceListRunsParams) (
	[]*model.V2beta1Run, int, string, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	response, err := c.apiClient.RunService.RunServiceListRuns(parameters, c.authInfoWriter)

	if err != nil {
		if defaultError, ok := err.(*params.RunServiceListRunsDefault); ok {
			err = api_server.CreateErrorFromAPIStatus(defaultError.Payload.Message, defaultError.Payload.Code)
		} else {
			err = api_server.CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return nil, 0, "", util.NewUserError(err,
			fmt.Sprintf("Failed to list runs. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to list runs"))
	}

	return response.Payload.Runs, int(response.Payload.TotalSize), response.Payload.NextPageToken, nil
}

func (c *RunClient) ListAll(parameters *params.RunServiceListRunsParams, maxResultSize int) (
	[]*model.V2beta1Run, error) {
	return listAllForRun(c, parameters, maxResultSize)
}

func listAllForRun(client RunInterface, parameters *params.RunServiceListRunsParams, maxResultSize int) (
	[]*model.V2beta1Run, error) {
	if maxResultSize < 0 {
		maxResultSize = 0
	}

	allResults := make([]*model.V2beta1Run, 0)
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

func (c *RunClient) Terminate(parameters *params.RunServiceTerminateRunParams) error {
	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	_, err := c.apiClient.RunService.RunServiceTerminateRun(parameters, c.authInfoWriter)
	if err != nil {
		return util.NewUserError(err,
			fmt.Sprintf("Failed to terminate run. Params: %+v", parameters),
			fmt.Sprintf("Failed to terminate run %v", parameters.RunID))
	}
	return nil
}
