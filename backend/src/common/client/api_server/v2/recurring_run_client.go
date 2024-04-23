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
	apiclient "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/recurring_run_client"
	params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/recurring_run_client/recurring_run_service"
	model "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/recurring_run_model"
	"github.com/kubeflow/pipelines/backend/src/common/client/api_server"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"golang.org/x/net/context"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
)

type RecurringRunInterface interface {
	Create(params *params.RecurringRunServiceCreateRecurringRunParams) (*model.V2beta1RecurringRun, error)
	Get(params *params.RecurringRunServiceGetRecurringRunParams) (*model.V2beta1RecurringRun, error)
	Delete(params *params.RecurringRunServiceDeleteRecurringRunParams) error
	Enable(params *params.RecurringRunServiceEnableRecurringRunParams) error
	Disable(params *params.RecurringRunServiceDisableRecurringRunParams) error
	List(params *params.RecurringRunServiceListRecurringRunsParams) ([]*model.V2beta1RecurringRun, int, string, error)
	ListAll(params *params.RecurringRunServiceListRecurringRunsParams, maxResultSize int) ([]*model.V2beta1RecurringRun, error)
}

type RecurringRunClient struct {
	apiClient      *apiclient.RecurringRun
	authInfoWriter runtime.ClientAuthInfoWriter
}

func NewRecurringRunClient(clientConfig clientcmd.ClientConfig, debug bool) (
	*RecurringRunClient, error) {

	runtime, err := api_server.NewHTTPRuntime(clientConfig, debug)
	if err != nil {
		return nil, fmt.Errorf("Error occurred when creating job client: %w", err)
	}

	apiClient := apiclient.New(runtime, strfmt.Default)

	// Creating job client
	return &RecurringRunClient{
		apiClient: apiClient,
	}, nil
}

func NewKubeflowInClusterRecurringRunClient(namespace string, debug bool) (
	*RecurringRunClient, error) {

	runtime := api_server.NewKubeflowInClusterHTTPRuntime(namespace, debug)

	apiClient := apiclient.New(runtime, strfmt.Default)

	// Creating job client
	return &RecurringRunClient{
		apiClient:      apiClient,
		authInfoWriter: api_server.SATokenVolumeProjectionAuth,
	}, nil
}

func (c *RecurringRunClient) Create(parameters *params.RecurringRunServiceCreateRecurringRunParams) (*model.V2beta1RecurringRun,
	error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	response, err := c.apiClient.RecurringRunService.RecurringRunServiceCreateRecurringRun(parameters)
	if err != nil {
		return nil, util.NewUserError(err,
			fmt.Sprintf("Failed to create job. Params: '%+v'. Body: '%+v'", parameters, parameters.Body),
			fmt.Sprintf("Failed to create job '%v'", parameters.Body.DisplayName))
	}

	return response.Payload, nil
}

func (c *RecurringRunClient) Get(parameters *params.RecurringRunServiceGetRecurringRunParams) (*model.V2beta1RecurringRun,
	error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	response, err := c.apiClient.RecurringRunService.RecurringRunServiceGetRecurringRun(parameters)
	if err != nil {
		return nil, util.NewUserError(err,
			fmt.Sprintf("Failed to get job. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to get job '%v'", parameters.RecurringRunID))
	}

	return response.Payload, nil
}

func (c *RecurringRunClient) Delete(parameters *params.RecurringRunServiceDeleteRecurringRunParams) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	_, err := c.apiClient.RecurringRunService.RecurringRunServiceDeleteRecurringRun(parameters)
	if err != nil {
		return util.NewUserError(err,
			fmt.Sprintf("Failed to get job. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to get job '%v'", parameters.RecurringRunID))
	}

	return nil
}

func (c *RecurringRunClient) Enable(parameters *params.RecurringRunServiceEnableRecurringRunParams) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	_, err := c.apiClient.RecurringRunService.RecurringRunServiceEnableRecurringRun(parameters)
	if err != nil {
		return util.NewUserError(err,
			fmt.Sprintf("Failed to enable job. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to enable job '%v'", parameters.RecurringRunID))
	}

	return nil
}

func (c *RecurringRunClient) Disable(parameters *params.RecurringRunServiceDisableRecurringRunParams) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	_, err := c.apiClient.RecurringRunService.RecurringRunServiceDisableRecurringRun(parameters)
	if err != nil {
		return util.NewUserError(err,
			fmt.Sprintf("Failed to disable job. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to disable job '%v'", parameters.RecurringRunID))
	}

	return nil
}

func (c *RecurringRunClient) List(parameters *params.RecurringRunServiceListRecurringRunsParams) (
	[]*model.V2beta1RecurringRun, int, string, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	response, err := c.apiClient.RecurringRunService.RecurringRunServiceListRecurringRuns(parameters)
	if err != nil {
		return nil, 0, "", util.NewUserError(err,
			fmt.Sprintf("Failed to list jobs. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to list jobs"))
	}

	return response.Payload.RecurringRuns, int(response.Payload.TotalSize), response.Payload.NextPageToken, nil
}

func (c *RecurringRunClient) ListAll(parameters *params.RecurringRunServiceListRecurringRunsParams, maxResultSize int) (
	[]*model.V2beta1RecurringRun, error) {
	return listAllForJob(c, parameters, maxResultSize)
}

func listAllForJob(client RecurringRunInterface, parameters *params.RecurringRunServiceListRecurringRunsParams,
	maxResultSize int) ([]*model.V2beta1RecurringRun, error) {
	if maxResultSize < 0 {
		maxResultSize = 0
	}

	allResults := make([]*model.V2beta1RecurringRun, 0)
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
