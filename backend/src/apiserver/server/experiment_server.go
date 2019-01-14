package server

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

type ExperimentServer struct {
	resourceManager *resource.ResourceManager
}

func (s *ExperimentServer) CreateExperiment(ctx context.Context, request *api.CreateExperimentRequest) (
	*api.Experiment, error) {
	err := ValidateCreateExperimentRequest(request)
	if err != nil {
		return nil, util.Wrap(err, "Validate experiment request failed.")
	}
	newExperiment, err := s.resourceManager.CreateExperiment(ToModelExperiment(request.Experiment))
	if err != nil {
		return nil, util.Wrap(err, "Create experiment failed.")
	}
	return ToApiExperiment(newExperiment), nil
}

func (s *ExperimentServer) GetExperiment(ctx context.Context, request *api.GetExperimentRequest) (
	*api.Experiment, error) {
	experiment, err := s.resourceManager.GetExperiment(request.Id)
	if err != nil {
		return nil, util.Wrap(err, "Get experiment failed.")
	}
	return ToApiExperiment(experiment), nil
}

func (s *ExperimentServer) ListExperiment(ctx context.Context, request *api.ListExperimentsRequest) (
	*api.ListExperimentsResponse, error) {
	opts, err := validatedListOptions(&model.Experiment{}, request.PageToken, int(request.PageSize), request.SortBy, request.Filter)

	if err != nil {
		return nil, util.Wrap(err, "Failed to create list options")
	}

	experiments, nextPageToken, err := s.resourceManager.ListExperiments(opts)
	if err != nil {
		return nil, util.Wrap(err, "List experiments failed.")
	}
	return &api.ListExperimentsResponse{
			Experiments:   ToApiExperiments(experiments),
			NextPageToken: nextPageToken},
		nil
}

func (s *ExperimentServer) DeleteExperiment(ctx context.Context, request *api.DeleteExperimentRequest) (*empty.Empty, error) {
	err := s.resourceManager.DeleteExperiment(request.Id)
	if err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil
}

func ValidateCreateExperimentRequest(request *api.CreateExperimentRequest) error {
	if request.Experiment == nil || request.Experiment.Name == "" {
		return util.NewInvalidInputError("Experiment name is empty. Please specify a valid experiment name.")
	}
	return nil
}

func NewExperimentServer(resourceManager *resource.ResourceManager) *ExperimentServer {
	return &ExperimentServer{resourceManager: resourceManager}
}
