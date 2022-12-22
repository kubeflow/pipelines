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
	api "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	//"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	//"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	//"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	//"github.com/robfig/cron"
	//authorizationv1 "k8s.io/api/authorization/v1"
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

func (s *RecurringRunServer) CreateRecurringRun(ctx context.Context, request *api.CreateRecurringRunRequest) (*api.RecurringRun, error) {
	return nil, util.NewInvalidInputError("CreateRecurringRun succeed")
}

func (s *RecurringRunServer) GetRecurringRun(ctx context.Context, request *api.GetRecurringRunRequest) (*api.RecurringRun, error) {
	return nil, util.NewInvalidInputError("GetRecurringRun succeed")
}

func (s *RecurringRunServer) ListRecurringRuns(ctx context.Context, request *api.ListRecurringRunsRequest) (*api.ListRecurringRunsResponse, error) {
	return nil, util.NewInvalidInputError("ListRecurringRuns succeed")
}

func (s *RecurringRunServer) EnableRecurringRun(ctx context.Context, request *api.EnableRecurringRunRequest) (*empty.Empty, error) {
	return nil, util.NewInvalidInputError("EnableRecurringRun succeed")
}

func (s *RecurringRunServer) DisableRecurringRun(ctx context.Context, request *api.DisableRecurringRunRequest) (*empty.Empty, error) {
	return nil, util.NewInvalidInputError("DisableRecurringRun succeed")
}

func (s *RecurringRunServer) DeleteRecurringRun(ctx context.Context, request *api.DeleteRecurringRunRequest) (*empty.Empty, error) {
	return &empty.Empty{}, util.NewInvalidInputError("DeleteRecurringRun succeed")
}

func NewRecurringRunServer(resourceManager *resource.ResourceManager, options *RecurringRunServerOptions) *RecurringRunServer {
	return &RecurringRunServer{resourceManager: resourceManager, options: options}
}
