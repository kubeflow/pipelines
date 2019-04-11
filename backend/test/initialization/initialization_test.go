package initialization

import (
	"testing"

	"github.com/golang/glog"
	params "github.com/kubeflow/pipelines/backend/api/go_http_client/experiment_client/experiment_service"
	"github.com/kubeflow/pipelines/backend/src/common/client/api_server"
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

	err := test.WaitForReady(*namespace, *initializeTimeout)
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
	experiments, totalSize, _, err := s.experimentClient.List(&params.ListExperimentParams{})
	assert.Nil(t, err)
	assert.Equal(t, 1, totalSize)
	assert.True(t, len(experiments) == 1)
	assert.Equal(t, "Default", experiments[0].Name)

	/* ---------- Clean up ---------- */
	test.DeleteAllExperiments(s.experimentClient, t)
}

func TestInitialization(t *testing.T) {
	suite.Run(t, new(InitializationTest))
}
