package test

import (
	"context"
	"testing"

	"time"

	"github.com/golang/glog"
	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
)

type ExperimentApiTest struct {
	suite.Suite
	namespace        string
	conn             *grpc.ClientConn
	experimentClient api.ExperimentServiceClient
}

// Check the namespace have ML job installed and ready
func (s *ExperimentApiTest) SetupTest() {
	err := waitForReady(*namespace, *initializeTimeout)
	if err != nil {
		glog.Exitf("Failed to initialize test. Error: %s", err.Error())
	}
	s.namespace = *namespace
	s.conn, err = getRpcConnection(s.namespace)
	if err != nil {
		glog.Exitf("Failed to get RPC connection. Error: %s", err.Error())
	}
	s.experimentClient = api.NewExperimentServiceClient(s.conn)
}

func (s *ExperimentApiTest) TestExperimentAPI() {
	t := s.T()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	/* ---------- Verify no experiment exist ---------- */
	listExperimentResponse, err := s.experimentClient.ListExperiment(ctx, &api.ListExperimentsRequest{})
	assert.Nil(t, err)
	assert.True(t, len(listExperimentResponse.Experiments) == 0)

	/* ---------- Create a new experiment ---------- */
	requestStartTime := time.Now().Unix()
	experiment := &api.Experiment{Name: "training", Description: "my first experiment"}
	trainingExperiment, err := s.experimentClient.CreateExperiment(ctx, &api.CreateExperimentRequest{
		Experiment: experiment,
	})
	assert.Nil(t, err)
	assert.True(t, trainingExperiment.CreatedAt.GetSeconds() >= requestStartTime)
	expectedTrainingExperiment := &api.Experiment{
		Id: trainingExperiment.Id, Name: experiment.Name,
		Description: experiment.Description, CreatedAt: trainingExperiment.CreatedAt}
	assert.Equal(t, expectedTrainingExperiment, trainingExperiment)

	/* ---------- Create an experiment with same name. Should fail due to name uniqueness ---------- */
	_, err = s.experimentClient.CreateExperiment(ctx, &api.CreateExperimentRequest{Experiment: experiment})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Please specify a new name")

	/* ---------- Create a few more new experiment ---------- */
	// 1 second interval. This ensures they can be sorted by create time in expected order.
	time.Sleep(1 * time.Second)
	experiment = &api.Experiment{Name: "prediction", Description: "my second experiment"}
	predictionExperiment, err := s.experimentClient.CreateExperiment(ctx, &api.CreateExperimentRequest{
		Experiment: experiment,
	})
	time.Sleep(1 * time.Second)
	experiment = &api.Experiment{Name: "moonshot", Description: "my second experiment"}
	moonshotExperiment, err := s.experimentClient.CreateExperiment(ctx, &api.CreateExperimentRequest{
		Experiment: experiment,
	})
	assert.Nil(t, err)

	/* ---------- Verify list experiments works ---------- */
	listExperimentResponse, err = s.experimentClient.ListExperiment(ctx, &api.ListExperimentsRequest{})
	assert.Nil(t, err)
	assert.Equal(t, 3, len(listExperimentResponse.Experiments))
	for _, e := range listExperimentResponse.Experiments {
		// Sampling one of the experiments and verify the result is expected.
		if e.Name == "training" {
			assert.Equal(t, expectedTrainingExperiment, trainingExperiment)
		}
	}

	/* ---------- Verify list experiments sorted by names ---------- */
	listFirstPageExperimentResponse, err := s.experimentClient.ListExperiment(ctx, &api.ListExperimentsRequest{PageSize: 2, SortBy: "name"})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(listFirstPageExperimentResponse.Experiments))
	assert.Equal(t, "moonshot", listFirstPageExperimentResponse.Experiments[0].Name)
	assert.Equal(t, "prediction", listFirstPageExperimentResponse.Experiments[1].Name)
	assert.NotEmpty(t, listFirstPageExperimentResponse.NextPageToken)

	listSecondPageExperimentResponse, err := s.experimentClient.ListExperiment(ctx, &api.ListExperimentsRequest{PageToken: listFirstPageExperimentResponse.NextPageToken, PageSize: 2, SortBy: "name"})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(listSecondPageExperimentResponse.Experiments))
	assert.Equal(t, "training", listSecondPageExperimentResponse.Experiments[0].Name)
	assert.Empty(t, listSecondPageExperimentResponse.NextPageToken)

	/* ---------- Verify list experiments sorted by creation time ---------- */
	listFirstPageExperimentResponse, err = s.experimentClient.ListExperiment(ctx, &api.ListExperimentsRequest{PageSize: 2, SortBy: "created_at"})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(listFirstPageExperimentResponse.Experiments))
	assert.Equal(t, "training", listFirstPageExperimentResponse.Experiments[0].Name)
	assert.Equal(t, "prediction", listFirstPageExperimentResponse.Experiments[1].Name)
	assert.NotEmpty(t, listFirstPageExperimentResponse.NextPageToken)

	listSecondPageExperimentResponse, err = s.experimentClient.ListExperiment(ctx, &api.ListExperimentsRequest{PageToken: listFirstPageExperimentResponse.NextPageToken, PageSize: 2, SortBy: "created_at"})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(listSecondPageExperimentResponse.Experiments))
	assert.Equal(t, "moonshot", listSecondPageExperimentResponse.Experiments[0].Name)
	assert.Empty(t, listSecondPageExperimentResponse.NextPageToken)

	/* ---------- List experiments sort by unsupported description field. Should fail. ---------- */
	_, err = s.experimentClient.ListExperiment(ctx, &api.ListExperimentsRequest{PageSize: 2, SortBy: "description"})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "InvalidArgument")

	/* ---------- List experiments sorted by names descend order ---------- */
	listFirstPageExperimentResponse, err = s.experimentClient.ListExperiment(ctx, &api.ListExperimentsRequest{PageSize: 2, SortBy: "name desc"})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(listFirstPageExperimentResponse.Experiments))
	assert.Equal(t, "training", listFirstPageExperimentResponse.Experiments[0].Name)
	assert.Equal(t, "prediction", listFirstPageExperimentResponse.Experiments[1].Name)
	assert.NotEmpty(t, listFirstPageExperimentResponse.NextPageToken)

	listSecondPageExperimentResponse, err = s.experimentClient.ListExperiment(ctx, &api.ListExperimentsRequest{PageToken: listFirstPageExperimentResponse.NextPageToken, PageSize: 2, SortBy: "name desc"})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(listSecondPageExperimentResponse.Experiments))
	assert.Equal(t, "moonshot", listSecondPageExperimentResponse.Experiments[0].Name)
	assert.Empty(t, listSecondPageExperimentResponse.NextPageToken)

	/* ---------- Verify get experiment works ---------- */
	experiment, err = s.experimentClient.GetExperiment(ctx, &api.GetExperimentRequest{Id: trainingExperiment.Id})
	assert.Nil(t, err)
	assert.Equal(t, expectedTrainingExperiment, experiment)

	/* ---------- Clean up ---------- */
	_, err = s.experimentClient.DeleteExperiment(ctx, &api.DeleteExperimentRequest{Id: trainingExperiment.Id})
	assert.Nil(t, err)
	_, err = s.experimentClient.DeleteExperiment(ctx, &api.DeleteExperimentRequest{Id: predictionExperiment.Id})
	assert.Nil(t, err)
	_, err = s.experimentClient.DeleteExperiment(ctx, &api.DeleteExperimentRequest{Id: moonshotExperiment.Id})
	assert.Nil(t, err)
}

func (s *ExperimentApiTest) TearDownTest() {
	s.conn.Close()
}

func TestExperimentAPI(t *testing.T) {
	suite.Run(t, new(ExperimentApiTest))
}
