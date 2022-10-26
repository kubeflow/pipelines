package api_server

import (
	"fmt"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	apiclient "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/job_client"
	params "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/job_client/job_service"
	model "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/job_model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"golang.org/x/net/context"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
)

type JobInterface interface {
	Create(params *params.CreateJobParams) (*model.V1beta1Job, error)
	Get(params *params.GetJobParams) (*model.V1beta1Job, error)
	Delete(params *params.DeleteJobParams) error
	Enable(params *params.EnableJobParams) error
	Disable(params *params.DisableJobParams) error
	List(params *params.ListJobsParams) ([]*model.V1beta1Job, int, string, error)
	ListAll(params *params.ListJobsParams, maxResultSize int) ([]*model.V1beta1Job, error)
}

type JobClient struct {
	apiClient      *apiclient.Job
	authInfoWriter runtime.ClientAuthInfoWriter
}

func NewJobClient(clientConfig clientcmd.ClientConfig, debug bool) (
	*JobClient, error) {

	runtime, err := NewHTTPRuntime(clientConfig, debug)
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

	runtime := NewKubeflowInClusterHTTPRuntime(namespace, debug)

	apiClient := apiclient.New(runtime, strfmt.Default)

	// Creating job client
	return &JobClient{
		apiClient:      apiClient,
		authInfoWriter: SATokenVolumeProjectionAuth,
	}, nil
}

func (c *JobClient) Create(parameters *params.CreateJobParams) (*model.V1beta1Job,
	error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), apiServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	response, err := c.apiClient.JobService.CreateJob(parameters, c.authInfoWriter)
	if err != nil {
		if defaultError, ok := err.(*params.CreateJobDefault); ok {
			err = CreateErrorFromAPIStatus(defaultError.Payload.Error, defaultError.Payload.Code)
		} else {
			err = CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return nil, util.NewUserError(err,
			fmt.Sprintf("Failed to create job. Params: '%+v'. Body: '%+v'", parameters, parameters.Body),
			fmt.Sprintf("Failed to create job '%v'", parameters.Body.Name))
	}

	return response.Payload, nil
}

func (c *JobClient) Get(parameters *params.GetJobParams) (*model.V1beta1Job,
	error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), apiServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	response, err := c.apiClient.JobService.GetJob(parameters, c.authInfoWriter)
	if err != nil {
		if defaultError, ok := err.(*params.GetJobDefault); ok {
			err = CreateErrorFromAPIStatus(defaultError.Payload.Error, defaultError.Payload.Code)
		} else {
			err = CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return nil, util.NewUserError(err,
			fmt.Sprintf("Failed to get job. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to get job '%v'", parameters.ID))
	}

	return response.Payload, nil
}

func (c *JobClient) Delete(parameters *params.DeleteJobParams) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), apiServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	_, err := c.apiClient.JobService.DeleteJob(parameters, c.authInfoWriter)
	if err != nil {
		if defaultError, ok := err.(*params.DeleteJobDefault); ok {
			err = CreateErrorFromAPIStatus(defaultError.Payload.Error, defaultError.Payload.Code)
		} else {
			err = CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return util.NewUserError(err,
			fmt.Sprintf("Failed to get job. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to get job '%v'", parameters.ID))
	}

	return nil
}

func (c *JobClient) Enable(parameters *params.EnableJobParams) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), apiServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	_, err := c.apiClient.JobService.EnableJob(parameters, c.authInfoWriter)
	if err != nil {
		if defaultError, ok := err.(*params.EnableJobDefault); ok {
			err = CreateErrorFromAPIStatus(defaultError.Payload.Error, defaultError.Payload.Code)
		} else {
			err = CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return util.NewUserError(err,
			fmt.Sprintf("Failed to enable job. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to enable job '%v'", parameters.ID))
	}

	return nil
}

func (c *JobClient) Disable(parameters *params.DisableJobParams) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), apiServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	_, err := c.apiClient.JobService.DisableJob(parameters, c.authInfoWriter)
	if err != nil {
		if defaultError, ok := err.(*params.DisableJobDefault); ok {
			err = CreateErrorFromAPIStatus(defaultError.Payload.Error, defaultError.Payload.Code)
		} else {
			err = CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return util.NewUserError(err,
			fmt.Sprintf("Failed to disable job. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to disable job '%v'", parameters.ID))
	}

	return nil
}

func (c *JobClient) List(parameters *params.ListJobsParams) (
	[]*model.V1beta1Job, int, string, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), apiServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	response, err := c.apiClient.JobService.ListJobs(parameters, c.authInfoWriter)
	if err != nil {
		if defaultError, ok := err.(*params.ListJobsDefault); ok {
			err = CreateErrorFromAPIStatus(defaultError.Payload.Error, defaultError.Payload.Code)
		} else {
			err = CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return nil, 0, "", util.NewUserError(err,
			fmt.Sprintf("Failed to list jobs. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to list jobs"))
	}

	return response.Payload.Jobs, int(response.Payload.TotalSize), response.Payload.NextPageToken, nil
}

func (c *JobClient) ListAll(parameters *params.ListJobsParams, maxResultSize int) (
	[]*model.V1beta1Job, error) {
	return listAllForJob(c, parameters, maxResultSize)
}

func listAllForJob(client JobInterface, parameters *params.ListJobsParams,
	maxResultSize int) ([]*model.V1beta1Job, error) {
	if maxResultSize < 0 {
		maxResultSize = 0
	}

	allResults := make([]*model.V1beta1Job, 0)
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
