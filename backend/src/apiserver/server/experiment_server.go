package server

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/pkg/errors"
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

	err = CanAccessNamespaceInResourceReferences(s.resourceManager, ctx, request.Experiment.ResourceReferences)
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the request.")
	}

	newExperiment, err := s.resourceManager.CreateExperiment(request.Experiment)
	if err != nil {
		return nil, util.Wrap(err, "Create experiment failed.")
	}
	return ToApiExperiment(newExperiment), nil
}

func (s *ExperimentServer) GetExperiment(ctx context.Context, request *api.GetExperimentRequest) (
	*api.Experiment, error) {
	if !common.IsMultiUserSharedReadMode() {
		err := s.canAccessExperiment(ctx, request.Id)
		if err != nil {
			return nil, util.Wrap(err, "Failed to authorize the request.")
		}
	}

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

	filterContext, err := ValidateFilter(request.ResourceReferenceKey)
	if err != nil {
		return nil, util.Wrap(err, "Validating filter failed.")
	}

	refKey := filterContext.ReferenceKey
	if common.IsMultiUserMode() {
		if refKey == nil || refKey.Type != common.Namespace {
			return nil, util.NewInvalidInputError("Invalid resource references for experiment. ListExperiment requires filtering by namespace.")
		}
		namespace := refKey.ID
		if len(namespace) == 0 {
			return nil, util.NewInvalidInputError("Invalid resource references for experiment. Namespace is empty.")
		}
		err = isAuthorized(s.resourceManager, ctx, namespace)
		if err != nil {
			return nil, util.Wrap(err, "Failed to authorize with API resource references")
		}
	} else {
		if refKey != nil && refKey.Type == common.Namespace && len(refKey.ID) > 0 {
			return nil, util.NewInvalidInputError("In single-user mode, ListExperiment cannot filter by namespace.")
		}
		// In single user mode, apply filter with empty namespace for backward compatibile.
		filterContext = &common.FilterContext{
			ReferenceKey: &common.ReferenceKey{Type: common.Namespace, ID: ""},
		}
	}

	experiments, total_size, nextPageToken, err := s.resourceManager.ListExperiments(filterContext, opts)
	if err != nil {
		return nil, util.Wrap(err, "List experiments failed.")
	}
	return &api.ListExperimentsResponse{
			Experiments:   ToApiExperiments(experiments),
			TotalSize:     int32(total_size),
			NextPageToken: nextPageToken},
		nil
}

func (s *ExperimentServer) DeleteExperiment(ctx context.Context, request *api.DeleteExperimentRequest) (*empty.Empty, error) {
	err := s.canAccessExperiment(ctx, request.Id)
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the request.")
	}

	err = s.resourceManager.DeleteExperiment(request.Id)
	if err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil
}

func ValidateCreateExperimentRequest(request *api.CreateExperimentRequest) error {
	if request.Experiment == nil || request.Experiment.Name == "" {
		return util.NewInvalidInputError("Experiment name is empty. Please specify a valid experiment name.")
	}

	resourceReferences := request.Experiment.GetResourceReferences()
	if common.IsMultiUserMode() {
		if len(resourceReferences) != 1 ||
			resourceReferences[0].Key.Type != api.ResourceType_NAMESPACE ||
			resourceReferences[0].Relationship != api.Relationship_OWNER {
			return util.NewInvalidInputError(
				"Invalid resource references for experiment. Expect one namespace type with owner relationship. Got: %v", resourceReferences)
		}
		namespace := common.GetNamespaceFromAPIResourceReferences(request.Experiment.ResourceReferences)
		if len(namespace) == 0 {
			return util.NewInvalidInputError("Invalid resource references for experiment. Namespace is empty.")
		}
	} else if len(resourceReferences) > 0 {
		return util.NewInvalidInputError("In single-user mode, CreateExperimentRequest shouldn't contain resource references.")
	}
	return nil
}

// TODO(chensun): consider refactoring the code to get rid of double-query of experiment.
func (s *ExperimentServer) canAccessExperiment(ctx context.Context, experimentID string) error {
	if !common.IsMultiUserMode() {
		// Skip authorization if not multi-user mode.
		return nil
	}
	namespace, err := s.resourceManager.GetNamespaceFromExperimentID(experimentID)
	if err != nil {
		return util.Wrap(err, "Failed to authorize with the experiment ID.")
	}
	if len(namespace) == 0 {
		return util.NewInternalServerError(errors.New("Empty namespace"), "The experiment doesn't have a valid namespace.")
	}

	err = isAuthorized(s.resourceManager, ctx, namespace)
	if err != nil {
		return util.Wrap(err, "Failed to authorize with API resource references")
	}
	return nil
}

func (s *ExperimentServer) ArchiveExperiment(ctx context.Context, request *api.ArchiveExperimentRequest) (*empty.Empty, error) {
	err := s.canAccessExperiment(ctx, request.Id)
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the requests.")
	}
	err = s.resourceManager.ArchiveExperiment(request.Id)
	if err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil
}

func (s *ExperimentServer) UnarchiveExperiment(ctx context.Context, request *api.UnarchiveExperimentRequest) (*empty.Empty, error) {
	err := s.canAccessExperiment(ctx, request.Id)
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the requests.")
	}
	err = s.resourceManager.UnarchiveExperiment(request.Id)
	if err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil
}

func NewExperimentServer(resourceManager *resource.ResourceManager) *ExperimentServer {
	return &ExperimentServer{resourceManager: resourceManager}
}
