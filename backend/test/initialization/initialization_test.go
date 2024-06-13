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

package initialization

import (
	"testing"

	"github.com/golang/glog"
	params "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/experiment_client/experiment_service"
	api_server "github.com/kubeflow/pipelines/backend/src/common/client/api_server/v1"
	"github.com/kubeflow/pipelines/backend/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type InitializationTest struct {
	suite.Suite
	namespace        string
	experimentClient *api_server.ExperimentClient
}

// Check the namespace have ML job installed and ready
func (s *InitializationTest) SetupTest() {
	if !*runIntegrationTests {
		s.T().SkipNow()
		return
	}

	err := test.WaitForReady(*initializeTimeout)
	if err != nil {
		glog.Exitf("Failed to initialize test. Error: %v", err)
	}
	s.namespace = *namespace
	clientConfig := test.GetClientConfig(*namespace)
	s.experimentClient, err = api_server.NewExperimentClient(clientConfig, false)
	if err != nil {
		glog.Exitf("Failed to get experiment client. Error: %v", err)
	}
}

func (s *InitializationTest) TestInitialization() {
	t := s.T()

	/* ---------- Verify that only the default experiment exists ---------- */
	experiments, totalSize, _, err := s.experimentClient.List(&params.ExperimentServiceListExperimentsV1Params{})
	assert.Nil(t, err)
	assert.Equal(t, 1, totalSize)
	assert.True(t, len(experiments) == 1)
	assert.Equal(t, "Default", experiments[0].Name)

	/* ---------- Clean up ---------- */
	test.DeleteAllExperiments(s.experimentClient, "", t)
}

func TestInitialization(t *testing.T) {
	suite.Run(t, new(InitializationTest))
}
