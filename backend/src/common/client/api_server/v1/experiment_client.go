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
	apiclient "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/experiment_client"
	params "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/experiment_client/experiment_service"
	model "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/experiment_model"
	"github.com/kubeflow/pipelines/backend/src/common/client/api_server"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"golang.org/x/net/context"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
)

type ExperimentInterface interface {
	Create(params *params.CreateExperimentV1Params) (*model.APIExperiment, error)
	Get(params *params.GetExperimentV1Params) (*model.APIExperiment, error)
	List(params *params.ListExperimentsV1Params) ([]*model.APIExperiment, int, string, error)
	ListAll(params *params.ListExperimentsV1Params, maxResultSize int) ([]*model.APIExperiment, error)
	Archive(params *params.ArchiveExperimentV1Params) error
	Unarchive(params *params.UnarchiveExperimentV1Params) error
}

type ExperimentClient struct {
	apiClient      *apiclient.Experiment
	authInfoWriter rt.ClientAuthInfoWriter
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
		apiClient:      apiClient,
		authInfoWriter: api_server.PassThroughAuth,
	}, nil
}

func NewKubeflowInClusterExperimentClient(namespace string, debug bool) (
	*ExperimentClient, error) {

	runtime := api_server.NewKubeflowInClusterHTTPRuntime(namespace, debug)

	apiClient := apiclient.New(runtime, strfmt.Default)

	// Creating experiment client
	return &ExperimentClient{
		apiClient:      apiClient,
		authInfoWriter: api_server.SATokenVolumeProjectionAuth,
	}, nil
}

func (c *ExperimentClient) Create(parameters *params.CreateExperimentV1Params) (*model.APIExperiment,
	error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	response, err := c.apiClient.ExperimentService.CreateExperimentV1(parameters, c.authInfoWriter)
	if err != nil {
		if defaultError, ok := err.(*params.CreateExperimentV1Default); ok {
			err = api_server.CreateErrorFromAPIStatus(defaultError.Payload.Error, defaultError.Payload.Code)
		} else {
			err = api_server.CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return nil, util.NewUserError(err,
			fmt.Sprintf("Failed to create experiment. Params: '%+v'. Body: '%+v'", parameters, parameters.Body),
			fmt.Sprintf("Failed to create experiment '%v'", parameters.Body.Name))
	}

	return response.Payload, nil
}

func (c *ExperimentClient) Get(parameters *params.GetExperimentV1Params) (*model.APIExperiment,
	error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	response, err := c.apiClient.ExperimentService.GetExperimentV1(parameters, c.authInfoWriter)
	if err != nil {
		if defaultError, ok := err.(*params.GetExperimentV1Default); ok {
			err = api_server.CreateErrorFromAPIStatus(defaultError.Payload.Error, defaultError.Payload.Code)
		} else {
			err = api_server.CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return nil, util.NewUserError(err,
			fmt.Sprintf("Failed to get experiment. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to get experiment '%v'", parameters.ID))
	}

	return response.Payload, nil
}

func (c *ExperimentClient) List(parameters *params.ListExperimentsV1Params) (
	[]*model.APIExperiment, int, string, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	response, err := c.apiClient.ExperimentService.ListExperimentsV1(parameters, c.authInfoWriter)
	if err != nil {
		if defaultError, ok := err.(*params.ListExperimentsV1Default); ok {
			err = api_server.CreateErrorFromAPIStatus(defaultError.Payload.Error, defaultError.Payload.Code)
		} else {
			err = api_server.CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return nil, 0, "", util.NewUserError(err,
			fmt.Sprintf("Failed to list experiments. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to list experiments"))
	}

	return response.Payload.Experiments, int(response.Payload.TotalSize), response.Payload.NextPageToken, nil
}

func (c *ExperimentClient) Delete(parameters *params.DeleteExperimentV1Params) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	_, err := c.apiClient.ExperimentService.DeleteExperimentV1(parameters, c.authInfoWriter)
	if err != nil {
		if defaultError, ok := err.(*params.DeleteExperimentV1Default); ok {
			err = api_server.CreateErrorFromAPIStatus(defaultError.Payload.Error, defaultError.Payload.Code)
		} else {
			err = api_server.CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return util.NewUserError(err,
			fmt.Sprintf("Failed to delete experiments. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to delete experiment"))
	}

	return nil
}

func (c *ExperimentClient) ListAll(parameters *params.ListExperimentsV1Params, maxResultSize int) (
	[]*model.APIExperiment, error) {
	return listAllForExperiment(c, parameters, maxResultSize)
}

func listAllForExperiment(client ExperimentInterface, parameters *params.ListExperimentsV1Params,
	maxResultSize int) ([]*model.APIExperiment, error) {
	if maxResultSize < 0 {
		maxResultSize = 0
	}

	allResults := make([]*model.APIExperiment, 0)
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

func (c *ExperimentClient) Archive(parameters *params.ArchiveExperimentV1Params) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	_, err := c.apiClient.ExperimentService.ArchiveExperimentV1(parameters, c.authInfoWriter)

	if err != nil {
		if defaultError, ok := err.(*params.ArchiveExperimentV1Default); ok {
			err = api_server.CreateErrorFromAPIStatus(defaultError.Payload.Error, defaultError.Payload.Code)
		} else {
			err = api_server.CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return util.NewUserError(err,
			fmt.Sprintf("Failed to archive experiments. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to archive experiments"))
	}

	return nil
}

func (c *ExperimentClient) Unarchive(parameters *params.UnarchiveExperimentV1Params) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	_, err := c.apiClient.ExperimentService.UnarchiveExperimentV1(parameters, c.authInfoWriter)

	if err != nil {
		if defaultError, ok := err.(*params.UnarchiveExperimentV1Default); ok {
			err = api_server.CreateErrorFromAPIStatus(defaultError.Payload.Error, defaultError.Payload.Code)
		} else {
			err = api_server.CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return util.NewUserError(err,
			fmt.Sprintf("Failed to unarchive experiments. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to unarchive experiments"))
	}

	return nil
}
