package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/golang/glog"
	params "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/visualization_client/visualization_service"
	"github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/visualization_model"
	"github.com/kubeflow/pipelines/backend/src/common/client/api_server"
	"github.com/kubeflow/pipelines/backend/test"
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
		err := test.WaitForReady(*namespace, *initializeTimeout)
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
	visualization := &visualization_model.V1beta1Visualization{
		Arguments: `{"code": ["print(2)"]}`,
		Type:      visualization_model.V1beta1VisualizationTypeCUSTOM,
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
