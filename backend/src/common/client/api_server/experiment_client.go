package api_server

import (
	"fmt"

	"github.com/go-openapi/strfmt"
	apiclient "github.com/kubeflow/pipelines/backend/api/go_http_client/experiment_client"
	params "github.com/kubeflow/pipelines/backend/api/go_http_client/experiment_client/experiment_service"
	model "github.com/kubeflow/pipelines/backend/api/go_http_client/experiment_model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"golang.org/x/net/context"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
)

type ExperimentInterface interface {
	Create(params *params.CreateExperimentParams) (*model.APIExperiment, error)
	Get(params *params.GetExperimentParams) (*model.APIExperiment, error)
	List(params *params.ListExperimentParams) ([]*model.APIExperiment, int, string, error)
	ListAll(params *params.ListExperimentParams, maxResultSize int) ([]*model.APIExperiment, error)
	Archive(params *params.ArchiveExperimentParams) error
	Unarchive(params *params.UnarchiveExperimentParams) error
}

type ExperimentClient struct {
	apiClient *apiclient.Experiment
}

func NewExperimentClient(clientConfig clientcmd.ClientConfig, debug bool) (
	*ExperimentClient, error) {

	runtime, err := NewHTTPRuntime(clientConfig, debug)
	if err != nil {
		return nil, err
	}

	apiClient := apiclient.New(runtime, strfmt.Default)

	// Creating upload client
	return &ExperimentClient{
		apiClient: apiClient,
	}, nil
}

func (c *ExperimentClient) Create(parameters *params.CreateExperimentParams) (*model.APIExperiment,
	error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), apiServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	response, err := c.apiClient.ExperimentService.CreateExperiment(parameters, PassThroughAuth)
	if err != nil {
		if defaultError, ok := err.(*params.CreateExperimentDefault); ok {
			err = CreateErrorFromAPIStatus(defaultError.Payload.Error, defaultError.Payload.Code)
		} else {
			err = CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return nil, util.NewUserError(err,
			fmt.Sprintf("Failed to create experiment. Params: '%+v'. Body: '%+v'", parameters, parameters.Body),
			fmt.Sprintf("Failed to create experiment '%v'", parameters.Body.Name))
	}

	return response.Payload, nil
}

func (c *ExperimentClient) Get(parameters *params.GetExperimentParams) (*model.APIExperiment,
	error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), apiServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	response, err := c.apiClient.ExperimentService.GetExperiment(parameters, PassThroughAuth)
	if err != nil {
		if defaultError, ok := err.(*params.GetExperimentDefault); ok {
			err = CreateErrorFromAPIStatus(defaultError.Payload.Error, defaultError.Payload.Code)
		} else {
			err = CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return nil, util.NewUserError(err,
			fmt.Sprintf("Failed to get experiment. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to get experiment '%v'", parameters.ID))
	}

	return response.Payload, nil
}

func (c *ExperimentClient) List(parameters *params.ListExperimentParams) (
	[]*model.APIExperiment, int, string, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), apiServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	response, err := c.apiClient.ExperimentService.ListExperiment(parameters, PassThroughAuth)
	if err != nil {
		if defaultError, ok := err.(*params.ListExperimentDefault); ok {
			err = CreateErrorFromAPIStatus(defaultError.Payload.Error, defaultError.Payload.Code)
		} else {
			err = CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return nil, 0, "", util.NewUserError(err,
			fmt.Sprintf("Failed to list experiments. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to list experiments"))
	}

	return response.Payload.Experiments, int(response.Payload.TotalSize), response.Payload.NextPageToken, nil
}

func (c *ExperimentClient) Delete(parameters *params.DeleteExperimentParams) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), apiServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	_, err := c.apiClient.ExperimentService.DeleteExperiment(parameters, PassThroughAuth)
	if err != nil {
		if defaultError, ok := err.(*params.DeleteExperimentDefault); ok {
			err = CreateErrorFromAPIStatus(defaultError.Payload.Error, defaultError.Payload.Code)
		} else {
			err = CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return util.NewUserError(err,
			fmt.Sprintf("Failed to delete experiments. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to delete experiment"))
	}

	return nil
}

func (c *ExperimentClient) ListAll(parameters *params.ListExperimentParams, maxResultSize int) (
	[]*model.APIExperiment, error) {
	return listAllForExperiment(c, parameters, maxResultSize)
}

func listAllForExperiment(client ExperimentInterface, parameters *params.ListExperimentParams,
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

func (c *ExperimentClient) Archive(parameters *params.ArchiveExperimentParams) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), apiServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	_, err := c.apiClient.ExperimentService.ArchiveExperiment(parameters, PassThroughAuth)

	if err != nil {
		if defaultError, ok := err.(*params.ArchiveExperimentDefault); ok {
			err = CreateErrorFromAPIStatus(defaultError.Payload.Error, defaultError.Payload.Code)
		} else {
			err = CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return util.NewUserError(err,
			fmt.Sprintf("Failed to archive experiments. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to archive experiments"))
	}

	return nil
}

func (c *ExperimentClient) Unarchive(parameters *params.UnarchiveExperimentParams) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), apiServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	_, err := c.apiClient.ExperimentService.UnarchiveExperiment(parameters, PassThroughAuth)

	if err != nil {
		if defaultError, ok := err.(*params.UnarchiveExperimentDefault); ok {
			err = CreateErrorFromAPIStatus(defaultError.Payload.Error, defaultError.Payload.Code)
		} else {
			err = CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return util.NewUserError(err,
			fmt.Sprintf("Failed to unarchive experiments. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to unarchive experiments"))
	}

	return nil
}
