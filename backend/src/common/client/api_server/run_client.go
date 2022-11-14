package api_server

import (
	"fmt"

	workflowapi "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	apiclient "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/run_client"
	params "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/run_client/run_service"
	model "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/run_model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"golang.org/x/net/context"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
)

type RunInterface interface {
	Archive(params *params.ArchiveRunV1Params) error
	Get(params *params.GetRunV1Params) (*model.V1beta1RunDetail, *workflowapi.Workflow, error)
	List(params *params.ListRunsV1Params) ([]*model.V1beta1Run, int, string, error)
	ListAll(params *params.ListRunsV1Params, maxResultSize int) ([]*model.V1beta1Run, error)
	Unarchive(params *params.UnarchiveRunV1Params) error
	Terminate(params *params.TerminateRunV1Params) error
}

type RunClient struct {
	apiClient      *apiclient.Run
	authInfoWriter runtime.ClientAuthInfoWriter
}

func NewRunClient(clientConfig clientcmd.ClientConfig, debug bool) (
	*RunClient, error) {

	runtime, err := NewHTTPRuntime(clientConfig, debug)
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

	runtime := NewKubeflowInClusterHTTPRuntime(namespace, debug)

	apiClient := apiclient.New(runtime, strfmt.Default)

	// Creating run client
	return &RunClient{
		apiClient:      apiClient,
		authInfoWriter: SATokenVolumeProjectionAuth,
	}, nil
}

func (c *RunClient) Create(parameters *params.CreateRunV1Params) (*model.V1beta1RunDetail,
	*workflowapi.Workflow, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), apiServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	response, err := c.apiClient.RunService.CreateRunV1(parameters, c.authInfoWriter)
	if err != nil {
		if defaultError, ok := err.(*params.GetRunV1Default); ok {
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

func (c *RunClient) Get(parameters *params.GetRunV1Params) (*model.V1beta1RunDetail,
	*workflowapi.Workflow, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), apiServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	response, err := c.apiClient.RunService.GetRunV1(parameters, c.authInfoWriter)
	if err != nil {
		if defaultError, ok := err.(*params.GetRunV1Default); ok {
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

func (c *RunClient) Archive(parameters *params.ArchiveRunV1Params) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), apiServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	_, err := c.apiClient.RunService.ArchiveRunV1(parameters, c.authInfoWriter)

	if err != nil {
		if defaultError, ok := err.(*params.ListRunsV1Default); ok {
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

func (c *RunClient) Unarchive(parameters *params.UnarchiveRunV1Params) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), apiServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	_, err := c.apiClient.RunService.UnarchiveRunV1(parameters, c.authInfoWriter)

	if err != nil {
		if defaultError, ok := err.(*params.ListRunsV1Default); ok {
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

func (c *RunClient) Delete(parameters *params.DeleteRunV1Params) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), apiServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	_, err := c.apiClient.RunService.DeleteRunV1(parameters, c.authInfoWriter)

	if err != nil {
		if defaultError, ok := err.(*params.ListRunsV1Default); ok {
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

func (c *RunClient) List(parameters *params.ListRunsV1Params) (
	[]*model.V1beta1Run, int, string, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), apiServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	response, err := c.apiClient.RunService.ListRunsV1(parameters, c.authInfoWriter)

	if err != nil {
		if defaultError, ok := err.(*params.ListRunsV1Default); ok {
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

func (c *RunClient) ListAll(parameters *params.ListRunsV1Params, maxResultSize int) (
	[]*model.V1beta1Run, error) {
	return listAllForRun(c, parameters, maxResultSize)
}

func listAllForRun(client RunInterface, parameters *params.ListRunsV1Params, maxResultSize int) (
	[]*model.V1beta1Run, error) {
	if maxResultSize < 0 {
		maxResultSize = 0
	}

	allResults := make([]*model.V1beta1Run, 0)
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

func (c *RunClient) Terminate(parameters *params.TerminateRunV1Params) error {
	ctx, cancel := context.WithTimeout(context.Background(), apiServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	_, err := c.apiClient.RunService.TerminateRunV1(parameters, c.authInfoWriter)
	if err != nil {
		return util.NewUserError(err,
			fmt.Sprintf("Failed to terminate run. Params: %+v", parameters),
			fmt.Sprintf("Failed to terminate run %v", parameters.RunID))
	}
	return nil
}
