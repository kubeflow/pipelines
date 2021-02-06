// Copyright 2018 Google LLC
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

package client

import (
	"context"
	"fmt"
	"time"

	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	addressTemp = "%s:%s"
)

type PipelineClientInterface interface {
	ReportWorkflow(workflow *util.Workflow) error
	ReportScheduledWorkflow(swf *util.ScheduledWorkflow) error
	ReadArtifact(request *api.ReadArtifactRequest) (*api.ReadArtifactResponse, error)
	ReportRunMetrics(request *api.ReportRunMetricsRequest) (*api.ReportRunMetricsResponse, error)
}

type PipelineClient struct {
	initializeTimeout     time.Duration
	timeout               time.Duration
	basePath              string
	mlPipelineServiceName string
	mlPipelineServicePort string
	reportServiceClient   api.ReportServiceClient
	runServiceClient      api.RunServiceClient
}

func NewPipelineClient(
	initializeTimeout time.Duration,
	timeout time.Duration,
	basePath string,
	mlPipelineServiceName string,
	mlPipelineServiceHttpPort string,
	mlPipelineServiceGRPCPort string) (*PipelineClient, error) {
	httpAddress := fmt.Sprintf(addressTemp, mlPipelineServiceName, mlPipelineServiceHttpPort)
	grpcAddress := fmt.Sprintf(addressTemp, mlPipelineServiceName, mlPipelineServiceGRPCPort)
	err := util.WaitForAPIAvailable(initializeTimeout, basePath, httpAddress)
	if err != nil {
		return nil, errors.Wrapf(err,
			"Failed to initialize pipeline client. Error: %s", err.Error())
	}
	connection, err := util.GetRpcConnection(grpcAddress)
	if err != nil {
		return nil, errors.Wrapf(err,
			"Failed to get RPC connection. Error: %s", err.Error())
	}

	return &PipelineClient{
		initializeTimeout:   initializeTimeout,
		timeout:             timeout,
		reportServiceClient: api.NewReportServiceClient(connection),
		runServiceClient:    api.NewRunServiceClient(connection),
	}, nil
}

func (p *PipelineClient) ReportWorkflow(workflow *util.Workflow) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	_, err := p.reportServiceClient.ReportWorkflow(ctx, &api.ReportWorkflowRequest{
		Workflow: workflow.ToStringForStore(),
	})

	if err != nil {
		statusCode, _ := status.FromError(err)
		if statusCode.Code() == codes.InvalidArgument || statusCode.Code() == codes.NotFound {
			// Do not retry if either:
			// * there is something wrong with the workflow
			// * the workflow has been deleted by someone else
			return util.NewCustomError(err, util.CUSTOM_CODE_PERMANENT,
				"Error while reporting workflow resource (code: %v, message: %v): %v, %+v",
				statusCode.Code(),
				statusCode.Message(),
				err.Error(),
				workflow.Workflow)
		} else {
			// Retry otherwise
			return util.NewCustomError(err, util.CUSTOM_CODE_TRANSIENT,
				"Error while reporting workflow resource (code: %v, message: %v): %v, %+v",
				statusCode.Code(),
				statusCode.Message(),
				err.Error(),
				workflow.Workflow)
		}
	}
	return nil
}

func (p *PipelineClient) ReportScheduledWorkflow(swf *util.ScheduledWorkflow) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	_, err := p.reportServiceClient.ReportScheduledWorkflow(ctx,
		&api.ReportScheduledWorkflowRequest{
			ScheduledWorkflow: swf.ToStringForStore(),
		})

	if err != nil {
		statusCode, _ := status.FromError(err)
		if statusCode.Code() == codes.InvalidArgument {
			// Do not retry if there is something wrong with the workflow
			return util.NewCustomError(err, util.CUSTOM_CODE_PERMANENT,
				"Error while reporting workflow resource (code: %v, message: %v): %v, %+v",
				statusCode.Code(),
				statusCode.Message(),
				err.Error(),
				swf.ScheduledWorkflow)
		} else {
			// Retry otherwise
			return util.NewCustomError(err, util.CUSTOM_CODE_TRANSIENT,
				"Error while reporting workflow resource (code: %v, message: %v): %v, %+v",
				statusCode.Code(),
				statusCode.Message(),
				err.Error(),
				swf.ScheduledWorkflow)
		}
	}
	return nil
}

// ReadArtifact reads artifact content from run service. If the artifact is not present, returns
// nil response.
func (p *PipelineClient) ReadArtifact(request *api.ReadArtifactRequest) (*api.ReadArtifactResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	response, err := p.runServiceClient.ReadArtifact(ctx, request)
	if err != nil {
		// TODO(hongyes): check NotFound error code before skip the error.
		return nil, nil
	}

	return response, nil
}

// ReportRunMetrics reports run metrics to run service.
func (p *PipelineClient) ReportRunMetrics(request *api.ReportRunMetricsRequest) (*api.ReportRunMetricsResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	response, err := p.runServiceClient.ReportRunMetrics(ctx, request)
	if err != nil {
		// This call should always succeed unless the run doesn't exist or server is broken. In
		// either cases, the job should retry at a later time.
		return nil, util.NewCustomError(err, util.CUSTOM_CODE_TRANSIENT,
			"Error while reporting metrics (%+v): %+v", request, err)
	}
	return response, nil
}
