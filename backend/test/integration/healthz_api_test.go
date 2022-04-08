package integration

import (
	"testing"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/common/client/api_server"
	"github.com/kubeflow/pipelines/backend/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type HealthzApiTest struct {
	suite.Suite
	namespace     string
	healthzClient *api_server.HealthzClient
}

// Check the namespace have ML job installed and ready
func (s *HealthzApiTest) SetupTest() {
	if !*runIntegrationTests {
		s.T().SkipNow()
		return
	}

	if !*isDevMode {
		err := test.WaitForReady(*namespace, *initializeTimeout)
		if err != nil {
			glog.Exitf("Failed to initialize test. Error: %v", err)
		}
	}
	s.namespace = *namespace
	clientConfig := test.GetClientConfig(*namespace)
	var err error
	s.healthzClient, err = api_server.NewHealthzClient(clientConfig, false)
	if err != nil {
		glog.Exitf("Failed to get healthz client. Error: %v", err)
	}
	s.cleanUp()
}

func (s *HealthzApiTest) TearDownSuite() {
	if *runIntegrationTests {
		if !*isDevMode {
			s.cleanUp()
		}
	}
}

func (s *HealthzApiTest) cleanUp() {
}

func (s *HealthzApiTest) TestHealthzAPI() {
	t := s.T()

	/* ---------- Verify healthz response ---------- */
	healthzResp, err := s.healthzClient.GetHealthz()
	assert.Nil(t, err)
	assert.NotNil(t, healthzResp)
}

func TestHealthzAPI(t *testing.T) {
	suite.Run(t, new(HealthzApiTest))
}
