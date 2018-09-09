package api_server

import (
	"fmt"

	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	apiclient "github.com/googleprivate/ml/backend/api/job_client"
	params "github.com/googleprivate/ml/backend/api/job_client/job_service"
	model "github.com/googleprivate/ml/backend/api/job_model"
	"github.com/googleprivate/ml/backend/src/common/util"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type JobInterface interface {
	Create(params *params.CreateJobParams) (*model.APIJob, error)
	Get(params *params.GetJobParams) (*model.APIJob, error)
	Delete(params *params.DeleteJobParams) error
	Enable(params *params.EnableJobParams) error
	Disable(params *params.DisableJobParams) error
	List(params *params.ListJobsParams) ([]*model.APIJob, string, error)
	ListAll(params *params.ListJobsParams, maxResultSize int) ([]*model.APIJob, error)
	ListRuns(params *params.ListJobRunsParams) ([]*model.APIRun, string, error)
	ListAllRuns(params *params.ListJobRunsParams, maxResultSize int) ([]*model.APIRun, error)
}

type JobClient struct {
	apiClient *apiclient.Job
}

func NewJobClient(clientConfig clientcmd.ClientConfig) (
	*JobClient, error) {
	// Creating k8 client
	k8Client, config, namespace, err := util.GetKubernetesClientFromClientConfig(clientConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "Error while creating K8 client")
	}

	// Create API client
	httpClient := k8Client.RESTClient().(*rest.RESTClient).Client
	masterIPAndPort := util.ExtractMasterIPAndPort(config)
	apiClient := apiclient.New(httptransport.NewWithClient(masterIPAndPort,
		fmt.Sprintf(apiServerBasePath, namespace), nil, httpClient), strfmt.Default)

	// Creating upload client
	return &JobClient{
		apiClient: apiClient,
	}, nil
}

func (c *JobClient) Create(parameters *params.CreateJobParams) (*model.APIJob,
	error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), apiServerDefaultTimeout)
	defer cancel()

	// Make service all
	parameters.Context = ctx
	response, err := c.apiClient.JobService.CreateJob(parameters)

	if err != nil {
		if defaultError, ok := err.(*params.CreateJobDefault); ok {
			err = fmt.Errorf(defaultError.Payload.Error)
		}

		return nil, util.NewUserError(err,
			fmt.Sprintf("Failed to create job. Params: %+v. Body: %+v", parameters, parameters.Body),
			fmt.Sprintf("Failed to create job %v", parameters.Body.Name))
	}

	return response.Payload, nil
}

func (c *JobClient) Get(parameters *params.GetJobParams) (*model.APIJob,
	error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), apiServerDefaultTimeout)
	defer cancel()

	// Make service all
	parameters.Context = ctx
	response, err := c.apiClient.JobService.GetJob(parameters)
	if err != nil {
		if defaultError, ok := err.(*params.GetJobDefault); ok {
			err = fmt.Errorf(defaultError.Payload.Error)
		}

		return nil, util.NewUserError(err,
			fmt.Sprintf("Failed to get job. Params: %+v", parameters),
			fmt.Sprintf("Failed to get job %v", parameters.ID))
	}

	return response.Payload, nil
}

func (c *JobClient) Delete(parameters *params.DeleteJobParams) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), apiServerDefaultTimeout)
	defer cancel()

	// Make service all
	parameters.Context = ctx
	_, err := c.apiClient.JobService.DeleteJob(parameters)
	if err != nil {
		if defaultError, ok := err.(*params.DeleteJobDefault); ok {
			err = fmt.Errorf(defaultError.Payload.Error)
		}

		return util.NewUserError(err,
			fmt.Sprintf("Failed to get job. Params: %+v", parameters),
			fmt.Sprintf("Failed to get job %v", parameters.ID))
	}

	return nil
}

func (c *JobClient) Enable(parameters *params.EnableJobParams) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), apiServerDefaultTimeout)
	defer cancel()

	// Make service all
	parameters.Context = ctx
	_, err := c.apiClient.JobService.EnableJob(parameters)
	if err != nil {
		if defaultError, ok := err.(*params.EnableJobDefault); ok {
			err = fmt.Errorf(defaultError.Payload.Error)
		}

		return util.NewUserError(err,
			fmt.Sprintf("Failed to enable job. Params: %+v", parameters),
			fmt.Sprintf("Failed to enable job %v", parameters.ID))
	}

	return nil
}

func (c *JobClient) Disable(parameters *params.DisableJobParams) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), apiServerDefaultTimeout)
	defer cancel()

	// Make service all
	parameters.Context = ctx
	_, err := c.apiClient.JobService.DisableJob(parameters)
	if err != nil {
		if defaultError, ok := err.(*params.DisableJobDefault); ok {
			err = fmt.Errorf(defaultError.Payload.Error)
		}

		return util.NewUserError(err,
			fmt.Sprintf("Failed to disable job. Params: %+v", parameters),
			fmt.Sprintf("Failed to disable job %v", parameters.ID))
	}

	return nil
}

func (c *JobClient) List(parameters *params.ListJobsParams) (
	[]*model.APIJob, string, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), apiServerDefaultTimeout)
	defer cancel()

	// Make service all
	parameters.Context = ctx
	response, err := c.apiClient.JobService.ListJobs(parameters)
	if err != nil {
		if defaultError, ok := err.(*params.ListJobsDefault); ok {
			err = fmt.Errorf(defaultError.Payload.Error)
		}

		return nil, "", util.NewUserError(err,
			fmt.Sprintf("Failed to list jobs. Params: %+v", parameters),
			fmt.Sprintf("Failed to list jobs"))
	}

	return response.Payload.Jobs, response.Payload.NextPageToken, nil
}

func (c *JobClient) ListAll(parameters *params.ListJobsParams, maxResultSize int) (
	[]*model.APIJob, error) {
	return listAllForJob(c, parameters, maxResultSize)
}

func listAllForJob(client JobInterface, parameters *params.ListJobsParams,
	maxResultSize int) ([]*model.APIJob, error) {
	if maxResultSize < 0 {
		maxResultSize = 0
	}

	allResults := make([]*model.APIJob, 0)
	firstCall := true
	for (firstCall || (parameters.PageToken != nil && *parameters.PageToken != "")) &&
		(len(allResults) < maxResultSize) {
		results, pageToken, err := client.List(parameters)
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

func (c *JobClient) ListRuns(parameters *params.ListJobRunsParams) (
	[]*model.APIRun, string, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), apiServerDefaultTimeout)
	defer cancel()

	// Make service all
	parameters.Context = ctx
	response, err := c.apiClient.JobService.ListJobRuns(parameters)
	if err != nil {
		if defaultError, ok := err.(*params.ListJobRunsDefault); ok {
			err = fmt.Errorf(defaultError.Payload.Error)
		}

		return nil, "", util.NewUserError(err,
			fmt.Sprintf("Failed to list runs for job '%v'. Params: %+v", parameters.JobID, parameters),
			fmt.Sprintf("Failed to list runs for job '%v'", parameters.JobID))
	}

	return response.Payload.Runs, response.Payload.NextPageToken, nil
}

func (c *JobClient) ListAllRuns(parameters *params.ListJobRunsParams, maxResultSize int) (
	[]*model.APIRun, error) {
	return listAllForJobRuns(c, parameters, maxResultSize)
}

func listAllForJobRuns(client JobInterface, parameters *params.ListJobRunsParams, maxResultSize int) (
	[]*model.APIRun, error) {
	if maxResultSize < 0 {
		maxResultSize = 0
	}

	allResults := make([]*model.APIRun, 0)
	firstCall := true
	for (firstCall || (parameters.PageToken != nil && *parameters.PageToken != "")) &&
		(len(allResults) < maxResultSize) {
		results, pageToken, err := client.ListRuns(parameters)
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
