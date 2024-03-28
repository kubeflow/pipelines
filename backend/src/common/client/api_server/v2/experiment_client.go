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

	"github.com/go-openapi/strfmt"
	apiclient "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/experiment_client"
	params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/experiment_client/experiment_service"
	model "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/experiment_model"
	"github.com/kubeflow/pipelines/backend/src/common/client/api_server"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"golang.org/x/net/context"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
)

type ExperimentInterface interface {
	Create(params *params.ExperimentServiceCreateExperimentParams) (*model.V2beta1Experiment, error)
	Get(params *params.ExperimentServiceGetExperimentParams) (*model.V2beta1Experiment, error)
	List(params *params.ExperimentServiceListExperimentsParams) ([]*model.V2beta1Experiment, int, string, error)
	ListAll(params *params.ExperimentServiceListExperimentsParams, maxResultSize int) ([]*model.V2beta1Experiment, error)
	Archive(params *params.ExperimentServiceArchiveExperimentParams) error
	Unarchive(params *params.ExperimentServiceUnarchiveExperimentParams) error
}

type ExperimentClient struct {
	apiClient *apiclient.Experiment
}

func NewExperimentClient(clientConfig clientcmd.ClientConfig, debug bool) (
	*ExperimentClient, error) {

	runtime, err := api_server.NewHTTPRuntime(clientConfig, debug)
	if err != nil {
		return nil, fmt.Errorf("Error occurred when creating experiment client: %w", err)
	}

	apiClient := apiclient.New(runtime, strfmt.Default)

	// Creating experiment client
	return &ExperimentClient{
		apiClient: apiClient,
	}, nil
}

func NewKubeflowInClusterExperimentClient(namespace string, debug bool) (
	*ExperimentClient, error) {

	runtime := api_server.NewKubeflowInClusterHTTPRuntime(namespace, debug)

	apiClient := apiclient.New(runtime, strfmt.Default)

	// Creating experiment client
	return &ExperimentClient{
		apiClient: apiClient,
	}, nil
}

func (c *ExperimentClient) Create(parameters *params.ExperimentServiceCreateExperimentParams) (*model.V2beta1Experiment,
	error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	response, err := c.apiClient.ExperimentService.ExperimentServiceCreateExperiment(parameters)
	if err != nil {
		return nil, util.NewUserError(err,
			fmt.Sprintf("Failed to create experiment. Params: '%+v'. Body: '%+v'", parameters, parameters.Body),
			fmt.Sprintf("Failed to create experiment '%v'", parameters.Body.DisplayName))
	}

	return response.Payload, nil
}

func (c *ExperimentClient) Get(parameters *params.ExperimentServiceGetExperimentParams) (*model.V2beta1Experiment,
	error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	response, err := c.apiClient.ExperimentService.ExperimentServiceGetExperiment(parameters)
	if err != nil {
		return nil, util.NewUserError(err,
			fmt.Sprintf("Failed to get experiment. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to get experiment '%v'", parameters.ExperimentID))
	}

	return response.Payload, nil
}

func (c *ExperimentClient) List(parameters *params.ExperimentServiceListExperimentsParams) (
	[]*model.V2beta1Experiment, int, string, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	response, err := c.apiClient.ExperimentService.ExperimentServiceListExperiments(parameters)
	if err != nil {
		return nil, 0, "", util.NewUserError(err,
			fmt.Sprintf("Failed to list experiments. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to list experiments"))
	}

	return response.Payload.Experiments, int(response.Payload.TotalSize), response.Payload.NextPageToken, nil
}

func (c *ExperimentClient) Delete(parameters *params.ExperimentServiceDeleteExperimentParams) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	_, err := c.apiClient.ExperimentService.ExperimentServiceDeleteExperiment(parameters)
	if err != nil {
		return util.NewUserError(err,
			fmt.Sprintf("Failed to delete experiments. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to delete experiment"))
	}

	return nil
}

func (c *ExperimentClient) ListAll(parameters *params.ExperimentServiceListExperimentsParams, maxResultSize int) (
	[]*model.V2beta1Experiment, error) {
	return listAllForExperiment(c, parameters, maxResultSize)
}

func listAllForExperiment(client ExperimentInterface, parameters *params.ExperimentServiceListExperimentsParams,
	maxResultSize int) ([]*model.V2beta1Experiment, error) {
	if maxResultSize < 0 {
		maxResultSize = 0
	}

	allResults := make([]*model.V2beta1Experiment, 0)
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

func (c *ExperimentClient) Archive(parameters *params.ExperimentServiceArchiveExperimentParams) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	_, err := c.apiClient.ExperimentService.ExperimentServiceArchiveExperiment(parameters)

	if err != nil {
		return util.NewUserError(err,
			fmt.Sprintf("Failed to archive experiments. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to archive experiments"))
	}

	return nil
}

func (c *ExperimentClient) Unarchive(parameters *params.ExperimentServiceUnarchiveExperimentParams) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	_, err := c.apiClient.ExperimentService.ExperimentServiceUnarchiveExperiment(parameters)

	if err != nil {
		return util.NewUserError(err,
			fmt.Sprintf("Failed to unarchive experiments. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to unarchive experiments"))
	}

	return nil
}
