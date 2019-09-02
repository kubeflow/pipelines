package integration

import (
	"fmt"
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

type ExperimentsTest struct {
	suite.Suite
	namespace        string
	experimentClient *api_server.ExperimentClient
}

// Check the namespace have ML job installed and ready
func (s *ExperimentsTest) SetupSuite() {
	fmt.Println("SetupSuite started")

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

func (s *ExperimentsTest) TestPrepareExperiments() {
	fmt.Println("upgrade prepare test start")
	t := s.T()

	/* ---------- Clean up ---------- */
	test.DeleteAllExperiments(s.experimentClient, t)

	/* ---------- Verify no experiment exist ---------- */
	experiments, totalSize, _, err := s.experimentClient.List(&params.ListExperimentParams{})
	assert.Nil(t, err)
	assert.Equal(t, 0, totalSize)
	assert.True(t, len(experiments) == 0)

	/* ---------- Create a new experiment ---------- */
	experiment := &experiment_model.APIExperiment{Name: "training", Description: "my first experiment"}
	trainingExperiment, err := s.experimentClient.Create(&params.CreateExperimentParams{
		Body: experiment,
	})
	assert.Nil(t, err)
	expectedTrainingExperiment := &experiment_model.APIExperiment{
		ID: trainingExperiment.ID, Name: experiment.Name,
		Description: experiment.Description, CreatedAt: trainingExperiment.CreatedAt}
	assert.Equal(t, expectedTrainingExperiment, trainingExperiment)

	/* ---------- Create a few more new experiment ---------- */
	// 1 second interval. This ensures they can be sorted by create time in expected order.
	time.Sleep(1 * time.Second)
	experiment = &experiment_model.APIExperiment{Name: "prediction", Description: "my second experiment"}
	_, err = s.experimentClient.Create(&params.CreateExperimentParams{
		Body: experiment,
	})
	time.Sleep(1 * time.Second)
	experiment = &experiment_model.APIExperiment{Name: "moonshot", Description: "my second experiment"}
	_, err = s.experimentClient.Create(&params.CreateExperimentParams{
		Body: experiment,
	})
	assert.Nil(t, err)
}

func (s *ExperimentsTest) TestVerifyExperiments() {
	fmt.Println("verify start")
	t := s.T()

	/* ---------- Verify list experiments works ---------- */
	experiments, totalSize, nextPageToken, err := s.experimentClient.List(&params.ListExperimentParams{})
	assert.Nil(t, err)
	assert.Equal(t, 3, totalSize)
	assert.Equal(t, 3, len(experiments))
	var trainingExperiment *experiment_model.APIExperiment
	for _, e := range experiments {
		// Sampling one of the experiments and verify the result is expected.
		if e.Name == "training" {
			trainingExperiment = e
		}
	}
	assert.NotNil(t, trainingExperiment)
	assert.Equal(t, trainingExperiment.Description, "my first experiment")

	/* ---------- Verify list experiments sorted by names ---------- */
	experiments, totalSize, nextPageToken, err = s.experimentClient.List(&params.ListExperimentParams{
		PageSize: util.Int32Pointer(2), SortBy: util.StringPointer("name")})
	assert.Nil(t, err)
	assert.Equal(t, 3, totalSize)
	fmt.Println("len = ", len(experiments))
	assert.Equal(t, 2, len(experiments))
	assert.Equal(t, "moonshot", experiments[0].Name)
	assert.Equal(t, "prediction", experiments[1].Name)
	assert.NotEmpty(t, nextPageToken)

	experiments, totalSize, nextPageToken, err = s.experimentClient.List(&params.ListExperimentParams{
		PageToken: util.StringPointer(nextPageToken), PageSize: util.Int32Pointer(2), SortBy: util.StringPointer("name")})
	assert.Nil(t, err)
	assert.Equal(t, 3, totalSize)
	assert.Equal(t, 1, len(experiments))
	assert.Equal(t, "training", experiments[0].Name)
	assert.Empty(t, nextPageToken)

	/* ---------- Verify list experiments sorted by creation time ---------- */
	experiments, totalSize, nextPageToken, err = s.experimentClient.List(&params.ListExperimentParams{
		PageSize: util.Int32Pointer(2), SortBy: util.StringPointer("created_at")})
	assert.Nil(t, err)
	assert.Equal(t, 3, totalSize)
	assert.Equal(t, 2, len(experiments))
	assert.Equal(t, "training", experiments[0].Name)
	assert.Equal(t, "prediction", experiments[1].Name)
	assert.NotEmpty(t, nextPageToken)

	experiments, totalSize, nextPageToken, err = s.experimentClient.List(&params.ListExperimentParams{
		PageToken: util.StringPointer(nextPageToken), PageSize: util.Int32Pointer(2), SortBy: util.StringPointer("created_at")})
	assert.Nil(t, err)
	assert.Equal(t, 3, totalSize)
	assert.Equal(t, 1, len(experiments))
	assert.Equal(t, "moonshot", experiments[0].Name)
	assert.Empty(t, nextPageToken)

	/* ---------- List experiments sorted by names descend order ---------- */
	experiments, totalSize, nextPageToken, err = s.experimentClient.List(&params.ListExperimentParams{
		PageSize: util.Int32Pointer(2), SortBy: util.StringPointer("name desc")})
	assert.Nil(t, err)
	assert.Equal(t, 3, totalSize)
	assert.Equal(t, 2, len(experiments))
	assert.Equal(t, "training", experiments[0].Name)
	assert.Equal(t, "prediction", experiments[1].Name)
	assert.NotEmpty(t, nextPageToken)

	experiments, totalSize, nextPageToken, err = s.experimentClient.List(&params.ListExperimentParams{
		PageToken: util.StringPointer(nextPageToken), PageSize: util.Int32Pointer(2), SortBy: util.StringPointer("name desc")})
	assert.Nil(t, err)
	assert.Equal(t, 3, totalSize)
	assert.Equal(t, 1, len(experiments))
	assert.Equal(t, "moonshot", experiments[0].Name)
	assert.Empty(t, nextPageToken)

	/* ---------- Verify get experiment works ---------- */
	experiment, err := s.experimentClient.Get(&params.GetExperimentParams{ID: trainingExperiment.ID})
	assert.Nil(t, err)
	assert.Equal(t, trainingExperiment, experiment)
}

func TestExperiments(t *testing.T) {
	suite.Run(t, new(ExperimentsTest))
}
