package server

import (
	"context"

	api "github.com/googleprivate/ml/backend/api/go_client"
	"github.com/googleprivate/ml/backend/src/apiserver/resource"
	"github.com/googleprivate/ml/backend/src/common/util"
)

type ExperimentServer struct {
	resourceManager *resource.ResourceManager
}

func (s *ExperimentServer) CreateExperiment(context.Context, *api.CreateExperimentRequest) (*api.Experiment, error) {
	return nil, util.NewBadRequestError(nil, "Not implemented")
}

func (s *ExperimentServer) GetExperiment(context.Context, *api.GetExperimentRequest) (*api.Experiment, error) {
	return nil, util.NewBadRequestError(nil, "Not implemented")
}

func (s *ExperimentServer) ListExperiment(context.Context, *api.ListExperimentsRequest) (*api.ListExperimentsResponse, error) {
	return nil, util.NewBadRequestError(nil, "Not implemented")
}

func NewExperimentServer(resourceManager *resource.ResourceManager) *ExperimentServer {
	return &ExperimentServer{resourceManager: resourceManager}
}
