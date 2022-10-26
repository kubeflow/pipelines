package api_server

import (
	"fmt"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	apiclient "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/visualization_client"
	params "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/visualization_client/visualization_service"
	model "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/visualization_model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"golang.org/x/net/context"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
)

type VisualizationInterface interface {
	Create(params *params.CreateVisualizationParams) (*model.V1beta1Visualization, error)
}

type VisualizationClient struct {
	apiClient      *apiclient.Visualization
	authInfoWriter runtime.ClientAuthInfoWriter
}

func NewVisualizationClient(clientConfig clientcmd.ClientConfig, debug bool) (
	*VisualizationClient, error) {

	runtime, err := NewHTTPRuntime(clientConfig, debug)
	if err != nil {
		return nil, fmt.Errorf("Error occurred when creating visualization client: %w", err)
	}

	apiClient := apiclient.New(runtime, strfmt.Default)

	// Creating upload client
	return &VisualizationClient{
		apiClient: apiClient,
	}, nil
}

func NewKubeflowInClusterVisualizationClient(namespace string, debug bool) (
	*VisualizationClient, error) {

	runtime := NewKubeflowInClusterHTTPRuntime(namespace, debug)

	apiClient := apiclient.New(runtime, strfmt.Default)

	// Creating upload client
	return &VisualizationClient{
		apiClient:      apiClient,
		authInfoWriter: SATokenVolumeProjectionAuth,
	}, nil
}

func (c *VisualizationClient) Create(parameters *params.CreateVisualizationParams) (*model.V1beta1Visualization,
	error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), apiServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	response, err := c.apiClient.VisualizationService.CreateVisualization(parameters, PassThroughAuth)
	if err != nil {
		if defaultError, ok := err.(*params.CreateVisualizationDefault); ok {
			err = CreateErrorFromAPIStatus(defaultError.Payload.Error, defaultError.Payload.Code)
		} else {
			err = CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return nil, util.NewUserError(err,
			fmt.Sprintf("Failed to create visualizaiton. Params: '%+v'. Body: '%+v'", parameters, parameters.Body),
			fmt.Sprintf("Failed to create visualization '%v'", parameters.Body.Type))
	}

	return response.Payload, nil
}
