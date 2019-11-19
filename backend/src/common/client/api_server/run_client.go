package api_server

import (
	"fmt"

	workflowapi "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/go-openapi/strfmt"
	apiclient "github.com/kubeflow/pipelines/backend/api/go_http_client/run_client"
	params "github.com/kubeflow/pipelines/backend/api/go_http_client/run_client/run_service"
	model "github.com/kubeflow/pipelines/backend/api/go_http_client/run_model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"golang.org/x/net/context"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
)

type RunInterface interface {
	Archive(params *params.ArchiveRunParams) error
	Get(params *params.GetRunParams) (*model.APIRunDetail, *workflowapi.Workflow, error)
	List(params *params.ListRunsParams) ([]*model.APIRun, int, string, error)
	ListAll(params *params.ListRunsParams, maxResultSize int) ([]*model.APIRun, error)
	Unarchive(params *params.UnarchiveRunParams) error
	Terminate(params *params.TerminateRunParams) error
}

type RunClient struct {
	apiClient *apiclient.Run
}

func NewRunClient(clientConfig clientcmd.ClientConfig, debug bool) (
	*RunClient, error) {

	runtime, err := NewHTTPRuntime(clientConfig, debug)
	if err != nil {
		return nil, err
	}

	apiClient := apiclient.New(runtime, strfmt.Default)

	// Creating upload client
	return &RunClient{
		apiClient: apiClient,
	}, nil
}

func (c *RunClient) Create(parameters *params.CreateRunParams) (*model.APIRunDetail,
	*workflowapi.Workflow, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), apiServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	response, err := c.apiClient.RunService.CreateRun(parameters, PassThroughAuth)
	if err != nil {
		if defaultError, ok := err.(*params.GetRunDefault); ok {
			err = CreateErrorFromAPIStatus(defaultError.Payload.Error, defaultError.Payload.Code)
		} else {
			err = CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return nil, nil, util.NewUserError(err,
			fmt.Sprintf("Failed to create run. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to create run '%v'", parameters.Body.Name))
	}

	// Unmarshal response
	var workflow workflowapi.Workflow
	err = yaml.Unmarshal([]byte(response.Payload.PipelineRuntime.WorkflowManifest), &workflow)
	if err != nil {
		return nil, nil, util.NewUserError(err,
			fmt.Sprintf("Failed to unmarshal reponse. Params: %+v. Response: %s", parameters,
				response.Payload.PipelineRuntime.WorkflowManifest),
			fmt.Sprintf("Failed to unmarshal reponse"))
	}

	return response.Payload, &workflow, nil
}

func (c *RunClient) Get(parameters *params.GetRunParams) (*model.APIRunDetail,
	*workflowapi.Workflow, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), apiServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	response, err := c.apiClient.RunService.GetRun(parameters, PassThroughAuth)
	if err != nil {
		if defaultError, ok := err.(*params.GetRunDefault); ok {
			err = CreateErrorFromAPIStatus(defaultError.Payload.Error, defaultError.Payload.Code)
		} else {
			err = CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return nil, nil, util.NewUserError(err,
			fmt.Sprintf("Failed to get run. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to get run '%v'", parameters.RunID))
	}

	// Unmarshal response
	var workflow workflowapi.Workflow
	err = yaml.Unmarshal([]byte(response.Payload.PipelineRuntime.WorkflowManifest), &workflow)
	if err != nil {
		return nil, nil, util.NewUserError(err,
			fmt.Sprintf("Failed to unmarshal reponse. Params: %+v. Response: %s", parameters,
				response.Payload.PipelineRuntime.WorkflowManifest),
			fmt.Sprintf("Failed to unmarshal reponse"))
	}

	return response.Payload, &workflow, nil
}

func (c *RunClient) Archive(parameters *params.ArchiveRunParams) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), apiServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	_, err := c.apiClient.RunService.ArchiveRun(parameters, PassThroughAuth)

	if err != nil {
		if defaultError, ok := err.(*params.ListRunsDefault); ok {
			err = CreateErrorFromAPIStatus(defaultError.Payload.Error, defaultError.Payload.Code)
		} else {
			err = CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return util.NewUserError(err,
			fmt.Sprintf("Failed to archive runs. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to archive runs"))
	}

	return nil
}

func (c *RunClient) Unarchive(parameters *params.UnarchiveRunParams) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), apiServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	_, err := c.apiClient.RunService.UnarchiveRun(parameters, PassThroughAuth)

	if err != nil {
		if defaultError, ok := err.(*params.ListRunsDefault); ok {
			err = CreateErrorFromAPIStatus(defaultError.Payload.Error, defaultError.Payload.Code)
		} else {
			err = CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return util.NewUserError(err,
			fmt.Sprintf("Failed to unarchive runs. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to unarchive runs"))
	}

	return nil
}

func (c *RunClient) Delete(parameters *params.DeleteRunParams) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), apiServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	_, err := c.apiClient.RunService.DeleteRun(parameters, PassThroughAuth)

	if err != nil {
		if defaultError, ok := err.(*params.ListRunsDefault); ok {
			err = CreateErrorFromAPIStatus(defaultError.Payload.Error, defaultError.Payload.Code)
		} else {
			err = CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return util.NewUserError(err,
			fmt.Sprintf("Failed to delete runs. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to delete runs"))
	}

	return nil
}

func (c *RunClient) List(parameters *params.ListRunsParams) (
	[]*model.APIRun, int, string, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), apiServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	response, err := c.apiClient.RunService.ListRuns(parameters, PassThroughAuth)

	if err != nil {
		if defaultError, ok := err.(*params.ListRunsDefault); ok {
			err = CreateErrorFromAPIStatus(defaultError.Payload.Error, defaultError.Payload.Code)
		} else {
			err = CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return nil, 0, "", util.NewUserError(err,
			fmt.Sprintf("Failed to list runs. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to list runs"))
	}

	return response.Payload.Runs, int(response.Payload.TotalSize), response.Payload.NextPageToken, nil
}

func (c *RunClient) ListAll(parameters *params.ListRunsParams, maxResultSize int) (
	[]*model.APIRun, error) {
	return listAllForRun(c, parameters, maxResultSize)
}

func listAllForRun(client RunInterface, parameters *params.ListRunsParams, maxResultSize int) (
	[]*model.APIRun, error) {
	if maxResultSize < 0 {
		maxResultSize = 0
	}

	allResults := make([]*model.APIRun, 0)
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

func (c *RunClient) Terminate(parameters *params.TerminateRunParams) error {
	ctx, cancel := context.WithTimeout(context.Background(), apiServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	_, err := c.apiClient.RunService.TerminateRun(parameters, PassThroughAuth)
	if err != nil {
		return util.NewUserError(err,
			fmt.Sprintf("Failed to terminate run. Params: %+v", parameters),
			fmt.Sprintf("Failed to terminate run %v", parameters.RunID))
	}
	return nil
}
