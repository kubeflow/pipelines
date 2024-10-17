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
	apiclient "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/job_client"
	params "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/job_client/job_service"
	model "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/job_model"
	"github.com/kubeflow/pipelines/backend/src/common/client/api_server"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"golang.org/x/net/context"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
)

type JobInterface interface {
	Create(params *params.JobServiceCreateJobParams) (*model.APIJob, error)
	Get(params *params.JobServiceGetJobParams) (*model.APIJob, error)
	Delete(params *params.JobServiceDeleteJobParams) error
	Enable(params *params.JobServiceEnableJobParams) error
	Disable(params *params.JobServiceDisableJobParams) error
	List(params *params.JobServiceListJobsParams) ([]*model.APIJob, int, string, error)
	ListAll(params *params.JobServiceListJobsParams, maxResultSize int) ([]*model.APIJob, error)
}

type JobClient struct {
	apiClient      *apiclient.Job
	authInfoWriter rt.ClientAuthInfoWriter
}

func NewJobClient(clientConfig clientcmd.ClientConfig, debug bool) (
	*JobClient, error) {

	runtime, err := api_server.NewHTTPRuntime(clientConfig, debug)
	if err != nil {
		return nil, fmt.Errorf("Error occurred when creating job client: %w", err)
	}

	apiClient := apiclient.New(runtime, strfmt.Default)

	// Creating job client
	return &JobClient{
		apiClient: apiClient,
	}, nil
}

func NewKubeflowInClusterJobClient(namespace string, debug bool) (
	*JobClient, error) {

	runtime := api_server.NewKubeflowInClusterHTTPRuntime(namespace, debug)

	apiClient := apiclient.New(runtime, strfmt.Default)

	// Creating job client
	return &JobClient{
		apiClient:      apiClient,
		authInfoWriter: api_server.SATokenVolumeProjectionAuth,
	}, nil
}

func (c *JobClient) Create(parameters *params.JobServiceCreateJobParams) (*model.APIJob,
	error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	response, err := c.apiClient.JobService.JobServiceCreateJob(parameters, c.authInfoWriter)
	if err != nil {
		if defaultError, ok := err.(*params.JobServiceCreateJobDefault); ok {
			err = api_server.CreateErrorFromAPIStatus(defaultError.Payload.Error, defaultError.Payload.Code)
		} else {
			err = api_server.CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return nil, util.NewUserError(err,
			fmt.Sprintf("Failed to create job. Params: '%+v'. Body: '%+v'", parameters, parameters.Body),
			fmt.Sprintf("Failed to create job '%v'", parameters.Body.Name))
	}

	return response.Payload, nil
}

func (c *JobClient) Get(parameters *params.JobServiceGetJobParams) (*model.APIJob,
	error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	response, err := c.apiClient.JobService.JobServiceGetJob(parameters, c.authInfoWriter)
	if err != nil {
		if defaultError, ok := err.(*params.JobServiceGetJobDefault); ok {
			err = api_server.CreateErrorFromAPIStatus(defaultError.Payload.Error, defaultError.Payload.Code)
		} else {
			err = api_server.CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return nil, util.NewUserError(err,
			fmt.Sprintf("Failed to get job. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to get job '%v'", parameters.ID))
	}

	return response.Payload, nil
}

func (c *JobClient) Delete(parameters *params.JobServiceDeleteJobParams) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	_, err := c.apiClient.JobService.JobServiceDeleteJob(parameters, c.authInfoWriter)
	if err != nil {
		if defaultError, ok := err.(*params.JobServiceDeleteJobDefault); ok {
			err = api_server.CreateErrorFromAPIStatus(defaultError.Payload.Error, defaultError.Payload.Code)
		} else {
			err = api_server.CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return util.NewUserError(err,
			fmt.Sprintf("Failed to get job. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to get job '%v'", parameters.ID))
	}

	return nil
}

func (c *JobClient) Enable(parameters *params.JobServiceEnableJobParams) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	_, err := c.apiClient.JobService.JobServiceEnableJob(parameters, c.authInfoWriter)
	if err != nil {
		if defaultError, ok := err.(*params.JobServiceEnableJobDefault); ok {
			err = api_server.CreateErrorFromAPIStatus(defaultError.Payload.Error, defaultError.Payload.Code)
		} else {
			err = api_server.CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return util.NewUserError(err,
			fmt.Sprintf("Failed to enable job. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to enable job '%v'", parameters.ID))
	}

	return nil
}

func (c *JobClient) Disable(parameters *params.JobServiceDisableJobParams) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	_, err := c.apiClient.JobService.JobServiceDisableJob(parameters, c.authInfoWriter)
	if err != nil {
		if defaultError, ok := err.(*params.JobServiceDisableJobDefault); ok {
			err = api_server.CreateErrorFromAPIStatus(defaultError.Payload.Error, defaultError.Payload.Code)
		} else {
			err = api_server.CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return util.NewUserError(err,
			fmt.Sprintf("Failed to disable job. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to disable job '%v'", parameters.ID))
	}

	return nil
}

func (c *JobClient) List(parameters *params.JobServiceListJobsParams) (
	[]*model.APIJob, int, string, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	response, err := c.apiClient.JobService.JobServiceListJobs(parameters, c.authInfoWriter)
	if err != nil {
		if defaultError, ok := err.(*params.JobServiceListJobsDefault); ok {
			err = api_server.CreateErrorFromAPIStatus(defaultError.Payload.Error, defaultError.Payload.Code)
		} else {
			err = api_server.CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return nil, 0, "", util.NewUserError(err,
			fmt.Sprintf("Failed to list jobs. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to list jobs"))
	}

	return response.Payload.Jobs, int(response.Payload.TotalSize), response.Payload.NextPageToken, nil
}

func (c *JobClient) ListAll(parameters *params.JobServiceListJobsParams, maxResultSize int) (
	[]*model.APIJob, error) {
	return listAllForJob(c, parameters, maxResultSize)
}

func listAllForJob(client JobInterface, parameters *params.JobServiceListJobsParams,
	maxResultSize int) ([]*model.APIJob, error) {
	if maxResultSize < 0 {
		maxResultSize = 0
	}

	allResults := make([]*model.APIJob, 0)
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
