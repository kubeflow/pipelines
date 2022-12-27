// Copyright 2022 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	//"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/robfig/cron"
	authorizationv1 "k8s.io/api/authorization/v1"
)

// Metric variables. Please prefix the metric names with recurring_run_server_.
var (
	// Used to calculate the request rate.
	createRecurringRunRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "recurring_run_server_create_requests",
		Help: "The total number of CreateRecurringRun requests",
	})

	getRecurringRunRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "recurring_run_server_get_requests",
		Help: "The total number of GetReccurringRun requests",
	})

	listRecurringRunRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "recurring_run_server_list_requests",
		Help: "The total number of ListRecurringRuns requests",
	})

	deleteRecurringRunRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "recurring_run_server_delete_requests",
		Help: "The total number of DeleteRecurringRun requests",
	})

	disableRecurringRunRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "recurring_run_server_disable_requests",
		Help: "The total number of DisableRecurringRun requests",
	})

	enableRecurringRunRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "recurring_run_server_enable_requests",
		Help: "The total number of EnableRecurringRun requests",
	})

	// TODO(jingzhang36): error count and success count.

	recurringRunCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "recurring_run_server_job_count",
		Help: "The current number of recurring runs in Kubeflow Pipelines instance",
	})
)

type RecurringRunServerOptions struct {
	CollectMetrics bool
}

type RecurringRunServer struct {
	resourceManager *resource.ResourceManager
	options         *RecurringRunServerOptions
}

func (s *RecurringRunServer) CreateRecurringRun(ctx context.Context, request *apiv2beta1.CreateRecurringRunRequest) (*apiv2beta1.RecurringRun, error) {
	// For metric purposes. Count how many times this function has been called.
	if s.options.CollectMetrics {
		createRecurringRunRequests.Inc()
	}

	// Validate the request.
	err := s.validateCreateRecurringRunRequest(request)
	if err != nil {
		return nil, util.Wrap(err, "Validate create job request failed")
	}

	// Check authorization in multi-user mode.
	if common.IsMultiUserMode() {
		if request.RecurringRun.Namespace == "" {
			return nil, util.NewInvalidInputError("Recurring run has no namespace.")
		}
		resourceAttributes := &authorizationv1.ResourceAttributes{
			Namespace: request.RecurringRun.Namespace,
			Verb:      common.RbacResourceVerbCreate,
			Name:      request.RecurringRun.DisplayName,
		}
		err = s.canAccessRecurringRun(ctx, "", resourceAttributes)
		if err != nil {
			return nil, util.Wrap(err, "Failed to authorize the request")
		}
	}

	// Send request to resource manager to create this recurring run.
	newRecurringRun, err := s.resourceManager.CreateJob(ctx, request.RecurringRun)
	if err != nil {
		return nil, err
	}

	// For metric purposes. Count how many recurring runs have been created.
	if s.options.CollectMetrics {
		recurringRunCount.Inc()
	}

	return ToApiRecurringRun(newRecurringRun), nil

}

func (s *RecurringRunServer) GetRecurringRun(ctx context.Context, request *apiv2beta1.GetRecurringRunRequest) (*apiv2beta1.RecurringRun, error) {
	return nil, util.NewInvalidInputError("GetRecurringRun succeed")
}

func (s *RecurringRunServer) ListRecurringRuns(ctx context.Context, request *apiv2beta1.ListRecurringRunsRequest) (*apiv2beta1.ListRecurringRunsResponse, error) {
	return nil, util.NewInvalidInputError("ListRecurringRuns succeed")
}

func (s *RecurringRunServer) EnableRecurringRun(ctx context.Context, request *apiv2beta1.EnableRecurringRunRequest) (*empty.Empty, error) {
	return nil, util.NewInvalidInputError("EnableRecurringRun succeed")
}

func (s *RecurringRunServer) DisableRecurringRun(ctx context.Context, request *apiv2beta1.DisableRecurringRunRequest) (*empty.Empty, error) {
	return nil, util.NewInvalidInputError("DisableRecurringRun succeed")
}

func (s *RecurringRunServer) DeleteRecurringRun(ctx context.Context, request *apiv2beta1.DeleteRecurringRunRequest) (*empty.Empty, error) {
	return &empty.Empty{}, util.NewInvalidInputError("DeleteRecurringRun succeed")
}

func NewRecurringRunServer(resourceManager *resource.ResourceManager, options *RecurringRunServerOptions) *RecurringRunServer {
	return &RecurringRunServer{resourceManager: resourceManager, options: options}
}

func (s *RecurringRunServer) validateCreateRecurringRunRequest(request *apiv2beta1.CreateRecurringRunRequest) error {
	recurringRun := request.RecurringRun

	// Validate the content of PipelineSource
	if err := ValidatePipelineSource(s.resourceManager, recurringRun); err != nil {
		return err
	}

	// Validate RuntimeConfig
	if err := validateRuntimeConfig(recurringRun.GetRuntimeConfig()); err != nil {
		return err
	}

	// Validate the value of MaxConcurrency is in range [1, 10]
	if recurringRun.MaxConcurrency > 10 || recurringRun.MaxConcurrency < 1 {
		return util.NewInvalidInputError("The max concurrency of the recurring run is out of range. Support 1-10. Received %v.", recurringRun.MaxConcurrency)
	}

	// Validate the cron schedule.
	if recurringRun.Trigger != nil && recurringRun.Trigger.GetCronSchedule() != nil {
		if _, err := cron.Parse(recurringRun.Trigger.GetCronSchedule().Cron); err != nil {
			return util.NewInvalidInputError(
				"Schedule cron is not a supported format(https://godoc.org/github.com/robfig/cron). Error: %v", err)
		}
	}

	// Validate the periodic schedule
	if recurringRun.Trigger != nil && recurringRun.Trigger.GetPeriodicSchedule() != nil {
		periodicScheduleInterval := recurringRun.Trigger.GetPeriodicSchedule().IntervalSecond
		if periodicScheduleInterval < 1 {
			return util.NewInvalidInputError(
				"Found invalid period schedule interval %v. Set at interval to least 1 second.", periodicScheduleInterval)
		}
	}
	return nil
}

func (s *RecurringRunServer) canAccessRecurringRun(ctx context.Context, recurringRunID string, resourceAttributes *authorizationv1.ResourceAttributes) error {
	if common.IsMultiUserMode() == false {
		// Skip authorization if not multi-user mode.
		return nil
	}

	if len(recurringRunID) > 0 {
		job, err := s.resourceManager.GetJob(recurringRunID)
		if err != nil {
			return util.Wrap(err, "Failed to authorize with the job ID.")
		}
		if len(resourceAttributes.Namespace) == 0 {
			if len(job.Namespace) == 0 {
				return util.NewInternalServerError(
					errors.New("Empty namespace"),
					"The job doesn't have a valid namespace.",
				)
			}
			resourceAttributes.Namespace = job.Namespace
		}
		if len(resourceAttributes.Name) == 0 {
			resourceAttributes.Name = job.Name
		}
	}

	resourceAttributes.Group = common.RbacPipelinesGroup
	resourceAttributes.Version = common.RbacPipelinesVersion
	resourceAttributes.Resource = common.RbacResourceTypeJobs

	err := isAuthorized(s.resourceManager, ctx, resourceAttributes)
	if err != nil {
		return util.Wrap(err, "Failed to authorize with API")
	}
	return nil
}
