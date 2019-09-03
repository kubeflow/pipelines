package integration

import (
	"testing"

	"time"

	"github.com/golang/glog"
	params "github.com/kubeflow/pipelines/backend/api/go_http_client/experiment_client/experiment_service"
	"github.com/kubeflow/pipelines/backend/api/go_http_client/experiment_model"
	"github.com/kubeflow/pipelines/backend/src/common/client/api_server"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// Methods are organized into two types: "prepare" and "verify".
// "prepare" tests setup resources before upgrade
// "verify" tests verifies resources are expected after upgrade
type UpgradeTests struct {
	suite.Suite
	namespace        string
	experimentClient *api_server.ExperimentClient
}

// Check the namespace have ML job installed and ready
func (s *UpgradeTests) SetupSuite() {
	// Integration tests also run these tests to first ensure they work, so that
	// when integration tests pass and upgrade tests fail, we know for sure
	// upgrade process went wrong somehow.
	if !(*runIntegrationTests || *runUpgradeTests) {
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

func (s *UpgradeTests) TestPrepareExperiments() {
	t := s.T()

	/* ---------- Clean up ---------- */
	test.DeleteAllExperiments(s.experimentClient, t)

	/* ---------- Create a new experiment ---------- */
	experiment := &experiment_model.APIExperiment{Name: "training", Description: "my first experiment"}
	_, err := s.experimentClient.Create(&params.CreateExperimentParams{
		Body: experiment,
	})
	assert.Nil(t, err)

	/* ---------- Create a few more new experiment ---------- */
	// This ensures they can be sorted by create time in expected order.
	time.Sleep(1 * time.Second)
	experiment = &experiment_model.APIExperiment{Name: "prediction", Description: "my second experiment"}
	_, err = s.experimentClient.Create(&params.CreateExperimentParams{
		Body: experiment,
	})
	assert.Nil(t, err)

	time.Sleep(1 * time.Second)
	experiment = &experiment_model.APIExperiment{Name: "moonshot", Description: "my third experiment"}
	_, err = s.experimentClient.Create(&params.CreateExperimentParams{
		Body: experiment,
	})
	assert.Nil(t, err)
}

func (s *UpgradeTests) TestVerifyExperiments() {
	t := s.T()

	/* ---------- Verify list experiments sorted by creation time ---------- */
	experiments, _, _, err := s.experimentClient.List(
		&params.ListExperimentParams{SortBy: util.StringPointer("created_at")})
	assert.Nil(t, err)
	// after upgrade, default experiment may be inserted, but the oldest 3
	// experiments should be the ones created in this test
	assert.True(t, len(experiments) >= 3)

	assert.Equal(t, "training", experiments[0].Name)
	assert.Equal(t, "my first experiment", experiments[0].Description)
	assert.NotEmpty(t, experiments[0].ID)
	assert.NotEmpty(t, experiments[0].CreatedAt)

	assert.Equal(t, "prediction", experiments[1].Name)
	assert.Equal(t, "my second experiment", experiments[1].Description)
	assert.NotEmpty(t, experiments[1].ID)
	assert.NotEmpty(t, experiments[1].CreatedAt)

	assert.Equal(t, "moonshot", experiments[2].Name)
	assert.Equal(t, "my third experiment", experiments[2].Description)
	assert.NotEmpty(t, experiments[2].ID)
	assert.NotEmpty(t, experiments[2].CreatedAt)
}

func TestUpgrade(t *testing.T) {
	suite.Run(t, new(UpgradeTests))
}
