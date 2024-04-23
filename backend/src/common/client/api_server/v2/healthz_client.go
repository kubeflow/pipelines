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
	apiclient "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/healthz_client"
	params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/healthz_client/healthz_service"
	model "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/healthz_model"
	"github.com/kubeflow/pipelines/backend/src/common/client/api_server"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"k8s.io/client-go/tools/clientcmd"
)

type HealthzInterface interface {
	GetHealthz() (*params.HealthzServiceGetHealthzOK, error)
}

type HealthzClient struct {
	apiClient *apiclient.Healthz
}

func NewHealthzClient(clientConfig clientcmd.ClientConfig, debug bool) (*HealthzClient, error) {
	runtime, err := api_server.NewHTTPRuntime(clientConfig, debug)
	if err != nil {
		return nil, err
	}

	apiClient := apiclient.New(runtime, strfmt.Default)

	// Creating upload client
	return &HealthzClient{
		apiClient: apiClient,
	}, nil
}

func (c *HealthzClient) GetHealthz() (*model.V2beta1GetHealthzResponse, error) {
	parameters := params.NewHealthzServiceGetHealthzParamsWithTimeout(api_server.APIServerDefaultTimeout)
	response, err := c.apiClient.HealthzService.HealthzServiceGetHealthz(parameters, api_server.PassThroughAuth)
	if err != nil {
		if defaultError, ok := err.(*params.HealthzServiceGetHealthzDefault); ok {
			err = api_server.CreateErrorFromAPIStatus(defaultError.Payload.Message, defaultError.Payload.Code)
		} else {
			err = api_server.CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return nil, util.NewUserError(err,
			fmt.Sprintf("Failed to get Healthz. Params: '%+v'", parameters),
			fmt.Sprintf("Failed to get Healthz. Params: '%+v'", parameters))

	}
	return response.Payload, nil
}
