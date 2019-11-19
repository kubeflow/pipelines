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

type ExperimentApiTest struct {
	suite.Suite
	namespace        string
	experimentClient *api_server.ExperimentClient
}

// Check the namespace have ML job installed and ready
func (s *ExperimentApiTest) SetupTest() {
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

func (s *ExperimentApiTest) TestExperimentAPI() {
	t := s.T()

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

	/* ---------- Create an experiment with same name. Should fail due to name uniqueness ---------- */
	_, err = s.experimentClient.Create(&params.CreateExperimentParams{Body: experiment})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Please specify a new name")

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

	/* ---------- Verify list experiments works ---------- */
	experiments, totalSize, nextPageToken, err := s.experimentClient.List(&params.ListExperimentParams{})
	assert.Nil(t, err)
	assert.Equal(t, 3, totalSize)
	assert.Equal(t, 3, len(experiments))
	for _, e := range experiments {
		// Sampling one of the experiments and verify the result is expected.
		if e.Name == "training" {
			assert.Equal(t, expectedTrainingExperiment, trainingExperiment)
		}
	}

	/* ---------- Verify list experiments sorted by names ---------- */
	experiments, totalSize, nextPageToken, err = s.experimentClient.List(&params.ListExperimentParams{
		PageSize: util.Int32Pointer(2), SortBy: util.StringPointer("name")})
	assert.Nil(t, err)
	assert.Equal(t, 3, totalSize)
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

	/* ---------- List experiments sort by unsupported field. Should fail. ---------- */
	_, _, _, err = s.experimentClient.List(&params.ListExperimentParams{
		PageSize: util.Int32Pointer(2), SortBy: util.StringPointer("unknownfield")})
	assert.NotNil(t, err)

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
	experiment, err = s.experimentClient.Get(&params.GetExperimentParams{ID: trainingExperiment.ID})
	assert.Nil(t, err)
	assert.Equal(t, expectedTrainingExperiment, experiment)

	/* ---------- Clean up ---------- */
	test.DeleteAllExperiments(s.experimentClient, t)
}

func TestExperimentAPI(t *testing.T) {
	suite.Run(t, new(ExperimentApiTest))
}
