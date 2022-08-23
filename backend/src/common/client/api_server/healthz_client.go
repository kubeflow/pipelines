package api_server

import (
	"fmt"

	"github.com/go-openapi/strfmt"
	apiclient "github.com/kubeflow/pipelines/backend/api/go_http_client/healthz_client"
	params "github.com/kubeflow/pipelines/backend/api/go_http_client/healthz_client/healthz_service"
	model "github.com/kubeflow/pipelines/backend/api/go_http_client/healthz_model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"k8s.io/client-go/tools/clientcmd"
)

type HealthzInterface interface {
	GetHealthz() (*params.GetHealthzOK, error)
}

type HealthzClient struct {
	apiClient *apiclient.Healthz
}

func NewHealthzClient(clientConfig clientcmd.ClientConfig, debug bool) (*HealthzClient, error) {
	runtime, err := NewHTTPRuntime(clientConfig, debug)
	if err != nil {
		return nil, err
	}

	apiClient := apiclient.New(runtime, strfmt.Default)

	// Creating upload client
	return &HealthzClient{
		apiClient: apiClient,
	}, nil
}

func (c *HealthzClient) GetHealthz() (*model.APIGetHealthzResponse, error) {
	parameters := params.NewGetHealthzParamsWithTimeout(apiServerDefaultTimeout)
	response, err := c.apiClient.HealthzService.GetHealthz(parameters, PassThroughAuth)
	if err != nil {
		if defaultError, ok := err.(*params.GetHealthzDefault); ok {
			err = CreateErrorFromAPIStatus(defaultError.Payload.Error, defaultError.Payload.Code)
		} else {
			err = CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return nil, util.NewUserError(err,
			fmt.Sprintf("Failed to get Healthz. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to get Healthz. Params: '%+v'", parameters))

	}
	return response.Payload, nil
}
