package server

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	api "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
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

	listExperimentRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "experiment_server_list_requests",
		Help: "The total number of ListExperiments requests",
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

type ExperimentServer struct {
	resourceManager *resource.ResourceManager
	options         *ExperimentServerOptions
}

func (s *ExperimentServer) CreateExperiment(ctx context.Context, request *api.CreateExperimentRequest) (
	*api.Experiment, error) {
	if s.options.CollectMetrics {
		createExperimentRequests.Inc()
	}

	err := ValidateCreateExperimentRequest(request)
	if err != nil {
		return nil, util.Wrap(err, "Validate experiment request failed.")
	}

	resourceAttributes := &authorizationv1.ResourceAttributes{
		Namespace: common.GetNamespaceFromAPIResourceReferences(request.Experiment.ResourceReferences),
		Verb:      common.RbacResourceVerbCreate,
		Name:      request.Experiment.Name,
	}
	err = s.canAccessExperiment(ctx, "", resourceAttributes)
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the request")
	}

	newExperiment, err := s.resourceManager.CreateExperiment(request.Experiment)
	if err != nil {
		return nil, util.Wrap(err, "Create experiment failed.")
	}

	if s.options.CollectMetrics {
		experimentCount.Inc()
	}
	return ToApiExperiment(newExperiment), nil
}
