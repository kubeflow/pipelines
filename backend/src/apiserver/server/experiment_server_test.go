package server

import (
	"testing"

	"github.com/golang/protobuf/ptypes/timestamp"
	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
)

func TestCreateExperiment(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager)
	server := ExperimentServer{resourceManager: resourceManager}
	experiment := &api.Experiment{Name: "ex1", Description: "first experiment"}

	result, err := server.CreateExperiment(nil, &api.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)
	expectedExperiment := &api.Experiment{
		Id:          resource.DefaultFakeUUID,
		Name:        "ex1",
		Description: "first experiment",
		CreatedAt:   &timestamp.Timestamp{Seconds: 1},
	}
	assert.Equal(t, expectedExperiment, result)
}

func TestCreateExperiment_Failed(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager)
	server := ExperimentServer{resourceManager: resourceManager}
	experiment := &api.Experiment{Name: "ex1", Description: "first experiment"}
	clientManager.DB().Close()
	_, err := server.CreateExperiment(nil, &api.CreateExperimentRequest{Experiment: experiment})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Create experiment failed.")
}

func TestGetExperiment(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager)
	server := ExperimentServer{resourceManager: resourceManager}
	experiment := &api.Experiment{Name: "ex1", Description: "first experiment"}

	createResult, err := server.CreateExperiment(nil, &api.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)
	result, err := server.GetExperiment(nil, &api.GetExperimentRequest{Id: createResult.Id})
	expectedExperiment := &api.Experiment{
		Id:          createResult.Id,
		Name:        "ex1",
		Description: "first experiment",
		CreatedAt:   &timestamp.Timestamp{Seconds: 1},
	}
	assert.Equal(t, expectedExperiment, result)
}

func TestGetExperiment_Failed(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager)
	server := ExperimentServer{resourceManager: resourceManager}
	experiment := &api.Experiment{Name: "ex1", Description: "first experiment"}

	createResult, err := server.CreateExperiment(nil, &api.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)
	clientManager.DB().Close()
	_, err = server.GetExperiment(nil, &api.GetExperimentRequest{Id: createResult.Id})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Get experiment failed.")
}

func TestListExperiment(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager)
	server := ExperimentServer{resourceManager: resourceManager}
	experiment := &api.Experiment{Name: "ex1", Description: "first experiment"}

	createResult, err := server.CreateExperiment(nil, &api.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)
	result, err := server.ListExperiment(nil, &api.ListExperimentsRequest{})
	expectedExperiment := []*api.Experiment{{
		Id:          createResult.Id,
		Name:        "ex1",
		Description: "first experiment",
		CreatedAt:   &timestamp.Timestamp{Seconds: 1},
	}}
	assert.Equal(t, expectedExperiment, result.Experiments)
}

func TestListExperiment_Failed(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager)
	server := ExperimentServer{resourceManager: resourceManager}
	experiment := &api.Experiment{Name: "ex1", Description: "first experiment"}

	_, err := server.CreateExperiment(nil, &api.CreateExperimentRequest{Experiment: experiment})
	assert.Nil(t, err)
	clientManager.DB().Close()
	_, err = server.ListExperiment(nil, &api.ListExperimentsRequest{})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "List experiments failed.")
}

func TestValidateCreateExperimentRequest_EmptyName(t *testing.T) {
	experiment := &api.Experiment{Description: "first experiment"}
	err := ValidateCreateExperimentRequest(&api.CreateExperimentRequest{Experiment: experiment})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "name is empty")
}
