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

package integration

import (
	"testing"

	"github.com/golang/glog"
	params "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/visualization_client/visualization_service"
	"github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/visualization_model"
	api_server "github.com/kubeflow/pipelines/backend/src/common/client/api_server/v1"
	"github.com/kubeflow/pipelines/backend/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type VisualizationApiTest struct {
	suite.Suite
	namespace           string
	resourceNamespace   string
	visualizationClient *api_server.VisualizationClient
}

// Check the namespace have ML job installed and ready
func (s *VisualizationApiTest) SetupTest() {
	if !*runIntegrationTests {
		s.T().SkipNow()
		return
	}

	if !*isDevMode {
		err := test.WaitForReady(*initializeTimeout)
		if err != nil {
			glog.Exitf("Failed to initialize test. Error: %v", err)
		}
	}
	s.namespace = *namespace

	var newVisualizationClient func() (*api_server.VisualizationClient, error)

	if *isKubeflowMode {
		s.resourceNamespace = *resourceNamespace

		newVisualizationClient = func() (*api_server.VisualizationClient, error) {
			return api_server.NewKubeflowInClusterVisualizationClient(s.namespace, *isDebugMode)
		}
	} else {
		clientConfig := test.GetClientConfig(*namespace)

		newVisualizationClient = func() (*api_server.VisualizationClient, error) {
			return api_server.NewVisualizationClient(clientConfig, *isDebugMode)
		}
	}

	var err error
	s.visualizationClient, err = newVisualizationClient()
	if err != nil {
		glog.Exitf("Failed to get visualization client. Error: %v", err)
	}
}

func (s *VisualizationApiTest) TestVisualizationAPI() {
	t := s.T()

	/* ---------- Generate custom visualization --------- */
	visualization := &visualization_model.APIVisualization{
		Arguments: `{"code": ["print(2)"]}`,
		Type:      visualization_model.APIVisualizationTypeCUSTOM,
	}
	customVisualization, err := s.visualizationClient.Create(&params.VisualizationServiceCreateVisualizationV1Params{
		Body: visualization,
	})
	assert.Nil(t, err)
	assert.NotNil(t, customVisualization.HTML)
}

func TestVisualizationAPI(t *testing.T) {
	suite.Run(t, new(VisualizationApiTest))
}
