package integration

import (
	"github.com/stretchr/testify/assert"
	"testing"

	"github.com/golang/glog"
	params "github.com/kubeflow/pipelines/backend/api/go_http_client/visualization_client/visualization_service"
	"github.com/kubeflow/pipelines/backend/api/go_http_client/visualization_model"
	"github.com/kubeflow/pipelines/backend/src/common/client/api_server"
	"github.com/kubeflow/pipelines/backend/test"
	"github.com/stretchr/testify/suite"
)

type VisualizationApiTest struct {
	suite.Suite
	namespace           string
	visualizationClient *api_server.VisualizationClient
}

// Check the namespace have ML job installed and ready
func (s *VisualizationApiTest) SetupTest() {
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
	s.visualizationClient, err = api_server.NewVisualizationClient(clientConfig, false)
	if err != nil {
		glog.Exitf("Failed to get experiment client. Error: %v", err)
	}
}

func (s *VisualizationApiTest) TestVisualizationAPI() {
	t := s.T()

	/* ---------- Generate custom visualization --------- */
	visualization := &visualization_model.APIVisualization{
		Arguments: `{"code": ["print(2)"]}`,
		Type:      visualization_model.APIVisualizationTypeCUSTOM,
	}
	customVisualization, err := s.visualizationClient.Create(&params.CreateVisualizationParams{
		Body: visualization,
	})
	assert.Nil(t, err)
	assert.NotNil(t, customVisualization.HTML)
}

func TestVisualizationAPI(t *testing.T) {
	suite.Run(t, new(VisualizationApiTest))
}
