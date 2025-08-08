// Copyright 2018 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"context"

	"google.golang.org/protobuf/types/known/emptypb"

	apiv1beta1 "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	authorizationv1 "k8s.io/api/authorization/v1"
)

// Metric variables. Please prefix the metric names with experiment_server_.
var (
	// Used to calculate the request rate.
	createExperimentRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "experiment_server_create_requests",
		Help: "The total number of CreateExperiment requests",
	})

	getExperimentRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "experiment_server_get_requests",
		Help: "The total number of GetExperiment requests",
	})

	listExperimentsV1Requests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "experiment_server_list_requests",
		Help: "The total number of ListExperimentsV1 requests",
	})

	deleteExperimentRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "experiment_server_delete_requests",
		Help: "The total number of DeleteExperiment requests",
	})

	archiveExperimentRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "experiment_server_archive_requests",
		Help: "The total number of ArchiveExperiment requests",
	})

	unarchiveExperimentRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "experiment_server_unarchive_requests",
		Help: "The total number of UnarchiveExperiment requests",
	})

	// TODO(jingzhang36): error count and success count.

	experimentCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "experiment_server_run_count",
		Help: "The current number of experiments in Kubeflow Pipelines instance",
	})
)

type ExperimentServerOptions struct {
	CollectMetrics bool
}

// BaseExperimentServer wraps ExperimentServer and ExperimentServerV1
// to enable method sharing. It can be removed once ExperimentServerV1
// is removed.
type BaseExperimentServer struct {
	resourceManager *resource.ResourceManager
	options         *ExperimentServerOptions
}

type ExperimentServer struct {
	*BaseExperimentServer
	apiv2beta1.UnimplementedExperimentServiceServer
}

type ExperimentServerV1 struct {
	*BaseExperimentServer
	apiv1beta1.UnimplementedExperimentServiceServer
}

func (s *BaseExperimentServer) createExperiment(ctx context.Context, experiment *model.Experiment) (*model.Experiment, error) {
	experiment.Namespace = s.resourceManager.ReplaceNamespace(experiment.Namespace)
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Namespace: experiment.Namespace,
		Verb:      common.RbacResourceVerbCreate,
		Name:      experiment.Name,
	}
	err := s.canAccessExperiment(ctx, "", resourceAttributes)
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the request")
	}

	return s.resourceManager.CreateExperiment(experiment)
}

func (s *ExperimentServerV1) CreateExperimentV1(ctx context.Context, request *apiv1beta1.CreateExperimentRequest) (
	*apiv1beta1.Experiment, error,
) {
	if s.options.CollectMetrics {
		createExperimentRequests.Inc()
	}

	modelExperiment, err := toModelExperiment(request.GetExperiment())
	if err != nil {
		return nil, util.Wrap(err, "[ExperimentServer]: Failed to create a v1beta1 experiment due to conversion error")
	}

	newExperiment, err := s.createExperiment(ctx, modelExperiment)
	if err != nil {
		return nil, util.Wrap(err, "Failed to create a v1beta1 experiment")
	}

	if s.options.CollectMetrics {
		experimentCount.Inc()
	}

	apiExperiment := toApiExperimentV1(newExperiment)
	if apiExperiment == nil {
		return nil, util.NewInternalServerError(errors.New("Failed to convert internal experiment representation to its API counterpart"), "Failed to create v1beta1 experiment")
	}
	return apiExperiment, nil
}

func (s *ExperimentServer) CreateExperiment(ctx context.Context, request *apiv2beta1.CreateExperimentRequest) (
	*apiv2beta1.Experiment, error,
) {
	if s.options.CollectMetrics {
		createExperimentRequests.Inc()
	}

	modelExperiment, err := toModelExperiment(request.GetExperiment())
	if err != nil {
		return nil, util.Wrap(err, "[ExperimentServer]: Failed to create a experiment due to conversion error")
	}

	newExperiment, err := s.createExperiment(ctx, modelExperiment)
	if err != nil {
		return nil, util.Wrap(err, "Failed to create a experiment")
	}

	if s.options.CollectMetrics {
		experimentCount.Inc()
	}

	apiExperiment := toApiExperiment(newExperiment)
	if apiExperiment == nil {
		return nil, util.NewInternalServerError(errors.New("Failed to convert internal experiment representation to its API counterpart"), "Failed to create experiment")
	}
	return apiExperiment, nil
}

func (s *BaseExperimentServer) getExperiment(ctx context.Context, experimentId string) (*model.Experiment, error) {
	err := s.canAccessExperiment(ctx, experimentId, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbGet})
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the request")
	}
	return s.resourceManager.GetExperiment(experimentId)
}

func (s *ExperimentServerV1) GetExperimentV1(ctx context.Context, request *apiv1beta1.GetExperimentRequest) (
	*apiv1beta1.Experiment, error,
) {
	if s.options.CollectMetrics {
		getExperimentRequests.Inc()
	}

	experiment, err := s.getExperiment(ctx, request.GetId())
	if err != nil {
		return nil, util.Wrap(err, "Failed to fetch v1beta1 experiment")
	}

	apiExperiment := toApiExperimentV1(experiment)
	if apiExperiment == nil {
		return nil, util.NewInternalServerError(errors.New("Failed to convert internal experiment representation to its v1beta1 API counterpart"), "Failed to fetch v1beta1 experiment")
	}
	return apiExperiment, nil
}

func (s *ExperimentServer) GetExperiment(ctx context.Context, request *apiv2beta1.GetExperimentRequest) (
	*apiv2beta1.Experiment, error,
) {
	if s.options.CollectMetrics {
		getExperimentRequests.Inc()
	}

	experiment, err := s.getExperiment(ctx, request.GetExperimentId())
	if err != nil {
		return nil, util.Wrap(err, "Failed to fetch experiment")
	}

	apiExperiment := toApiExperiment(experiment)
	if apiExperiment == nil {
		return nil, util.NewInternalServerError(errors.New("Failed to convert internal experiment representation to its API counterpart"), "Failed to fetch experiment")
	}
	return apiExperiment, nil
}

func (s *BaseExperimentServer) listExperiments(ctx context.Context, pageToken string, pageSize int32, sortBy string, opts *list.Options, namespace string) ([]*model.Experiment, int32, string, error) {
	namespace = s.resourceManager.ReplaceNamespace(namespace)
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Namespace: namespace,
		Verb:      common.RbacResourceVerbList,
	}
	err := s.canAccessExperiment(ctx, "", resourceAttributes)
	if err != nil {
		return nil, 0, "", util.Wrap(err, "Failed to authorize with API")
	}
	filterContext := &model.FilterContext{
		ReferenceKey: &model.ReferenceKey{Type: model.NamespaceResourceType, ID: namespace},
	}

	experiments, totalSize, nextPageToken, err := s.resourceManager.ListExperiments(filterContext, opts)
	if err != nil {
		return nil, 0, "", util.Wrap(err, "List experiments failed")
	}
	return experiments, int32(totalSize), nextPageToken, nil
}

func (s *ExperimentServerV1) ListExperimentsV1(ctx context.Context, request *apiv1beta1.ListExperimentsRequest) (
	*apiv1beta1.ListExperimentsResponse, error,
) {
	if s.options.CollectMetrics {
		listExperimentsV1Requests.Inc()
	}

	filterContext, err := validateFilterV1(request.ResourceReferenceKey)
	if err != nil {
		return nil, util.Wrap(err, "Validating v1beta1 filter failed")
	}
	namespace := ""
	if filterContext.ReferenceKey != nil {
		if filterContext.ReferenceKey.Type == model.NamespaceResourceType {
			namespace = filterContext.ReferenceKey.ID
		} else {
			return nil, util.NewInvalidInputError("Failed to list v1beta1 experiment due to invalid resource reference key. It must be of type 'Namespace' and contain an existing or empty namespace, but you provided %v of type %v", filterContext.ReferenceKey.ID, filterContext.ReferenceKey.Type)
		}
	}

	opts, err := validatedListOptions(&model.Experiment{}, request.GetPageToken(), int(request.GetPageSize()), request.GetSortBy(), request.GetFilter(), "v1beta1")
	if err != nil {
		return nil, util.Wrap(err, "Failed to create list options")
	}

	experiments, totalSize, nextPageToken, err := s.listExperiments(
		ctx,
		request.GetPageToken(),
		request.GetPageSize(),
		request.GetSortBy(),
		opts,
		namespace,
	)
	if err != nil {
		return nil, util.Wrap(err, "List v1beta1 experiments failed")
	}
	return &apiv1beta1.ListExperimentsResponse{
		Experiments:   toApiExperimentsV1(experiments),
		TotalSize:     totalSize,
		NextPageToken: nextPageToken,
	}, nil
}

func (s *ExperimentServer) ListExperiments(ctx context.Context, request *apiv2beta1.ListExperimentsRequest) (
	*apiv2beta1.ListExperimentsResponse, error,
) {
	if s.options.CollectMetrics {
		listExperimentsV1Requests.Inc()
	}

	opts, err := validatedListOptions(&model.Experiment{}, request.GetPageToken(), int(request.GetPageSize()), request.GetSortBy(), request.GetFilter(), "v2beta1")
	if err != nil {
		return nil, util.Wrap(err, "Failed to create list options")
	}

	experiments, totalSize, nextPageToken, err := s.listExperiments(ctx, request.GetPageToken(), request.GetPageSize(), request.GetSortBy(), opts, request.GetNamespace())
	if err != nil {
		return nil, util.Wrap(err, "List experiments failed")
	}
	return &apiv2beta1.ListExperimentsResponse{
		Experiments:   toApiExperiments(experiments),
		TotalSize:     totalSize,
		NextPageToken: nextPageToken,
	}, nil
}

func (s *BaseExperimentServer) deleteExperiment(ctx context.Context, experimentId string) error {
	err := s.canAccessExperiment(ctx, experimentId, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbDelete})
	if err != nil {
		return util.Wrap(err, "Failed to authorize the request")
	}
	return s.resourceManager.DeleteExperiment(experimentId)
}

func (s *ExperimentServerV1) DeleteExperimentV1(ctx context.Context, request *apiv1beta1.DeleteExperimentRequest) (*emptypb.Empty, error) {
	if s.options.CollectMetrics {
		deleteExperimentRequests.Inc()
	}

	if err := s.deleteExperiment(ctx, request.GetId()); err != nil {
		return nil, util.Wrap(err, "Failed to delete v1beta1 experiment")
	}

	if s.options.CollectMetrics {
		experimentCount.Dec()
	}
	return &emptypb.Empty{}, nil
}

func (s *ExperimentServer) DeleteExperiment(ctx context.Context, request *apiv2beta1.DeleteExperimentRequest) (*emptypb.Empty, error) {
	if s.options.CollectMetrics {
		deleteExperimentRequests.Inc()
	}

	if err := s.deleteExperiment(ctx, request.GetExperimentId()); err != nil {
		return nil, util.Wrap(err, "Failed to delete experiment")
	}

	if s.options.CollectMetrics {
		experimentCount.Dec()
	}
	return &emptypb.Empty{}, nil
}

// TODO(chensun): consider refactoring the code to get rid of double-query of experiment.
func (s *BaseExperimentServer) canAccessExperiment(ctx context.Context, experimentID string, resourceAttributes *authorizationv1.ResourceAttributes) error {
	if !common.IsMultiUserMode() {
		// Skip authorization if not multi-user mode.
		return nil
	}
	if experimentID != "" {
		experiment, err := s.resourceManager.GetExperiment(experimentID)
		if err != nil {
			return util.Wrap(err, "Failed to authorize with the experiment ID")
		}
		resourceAttributes.Namespace = experiment.Namespace
		if resourceAttributes.Name == "" {
			resourceAttributes.Name = experiment.Name
		}
	}
	if s.resourceManager.IsEmptyNamespace(resourceAttributes.Namespace) {
		return util.NewInvalidInputError("An experiment cannot have an empty namespace in multi-user mode")
	}
	resourceAttributes.Group = common.RbacPipelinesGroup
	resourceAttributes.Version = common.RbacPipelinesVersion
	resourceAttributes.Resource = common.RbacResourceTypeExperiments

	err := s.resourceManager.IsAuthorized(ctx, resourceAttributes)
	if err != nil {
		return util.Wrap(err, "Failed to authorize with API")
	}
	return nil
}

func (s *BaseExperimentServer) archiveExperiment(ctx context.Context, experimentId string) error {
	err := s.canAccessExperiment(ctx, experimentId, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbArchive})
	if err != nil {
		return util.Wrap(err, "Failed to authorize the request")
	}
	return s.resourceManager.ArchiveExperiment(ctx, experimentId)
}

func (s *ExperimentServerV1) ArchiveExperimentV1(ctx context.Context, request *apiv1beta1.ArchiveExperimentRequest) (*emptypb.Empty, error) {
	if s.options.CollectMetrics {
		archiveExperimentRequests.Inc()
	}
	if err := s.archiveExperiment(ctx, request.GetId()); err != nil {
		return nil, util.Wrap(err, "Failed to archive v1beta1 experiment")
	}
	return &emptypb.Empty{}, nil
}

func (s *ExperimentServer) ArchiveExperiment(ctx context.Context, request *apiv2beta1.ArchiveExperimentRequest) (*emptypb.Empty, error) {
	if s.options.CollectMetrics {
		archiveExperimentRequests.Inc()
	}

	if err := s.archiveExperiment(ctx, request.GetExperimentId()); err != nil {
		return nil, util.Wrap(err, "Failed to archive experiment")
	}
	return &emptypb.Empty{}, nil
}

func (s *BaseExperimentServer) unarchiveExperiment(ctx context.Context, experimentId string) error {
	err := s.canAccessExperiment(ctx, experimentId, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbUnarchive})
	if err != nil {
		return util.Wrap(err, "Failed to authorize the request")
	}
	return s.resourceManager.UnarchiveExperiment(experimentId)
}

func (s *ExperimentServerV1) UnarchiveExperimentV1(ctx context.Context, request *apiv1beta1.UnarchiveExperimentRequest) (*emptypb.Empty, error) {
	if s.options.CollectMetrics {
		unarchiveExperimentRequests.Inc()
	}

	if err := s.unarchiveExperiment(ctx, request.GetId()); err != nil {
		return nil, util.Wrap(err, "Failed to unarchive v1beta1 experiment")
	}
	return &emptypb.Empty{}, nil
}

func (s *ExperimentServer) UnarchiveExperiment(ctx context.Context, request *apiv2beta1.UnarchiveExperimentRequest) (*emptypb.Empty, error) {
	if s.options.CollectMetrics {
		unarchiveExperimentRequests.Inc()
	}

	if err := s.unarchiveExperiment(ctx, request.GetExperimentId()); err != nil {
		return nil, util.Wrap(err, "Failed to unarchive experiment")
	}
	return &emptypb.Empty{}, nil
}

func NewExperimentServer(resourceManager *resource.ResourceManager, options *ExperimentServerOptions) *ExperimentServer {
	return &ExperimentServer{
		BaseExperimentServer: &BaseExperimentServer{
			resourceManager: resourceManager,
			options:         options,
		},
	}
}

func NewExperimentServerV1(resourceManager *resource.ResourceManager, options *ExperimentServerOptions) *ExperimentServerV1 {
	return &ExperimentServerV1{
		BaseExperimentServer: &BaseExperimentServer{
			resourceManager: resourceManager,
			options:         options,
		},
	}
}
