// Copyright 2018 The Kubeflow Authors
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
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"google.golang.org/grpc/metadata"
	"os"
	"time"

	api "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	addressTemp = "%s:%s"
)

type PipelineClientInterface interface {
	ReportWorkflow(workflow util.ExecutionSpec) error
	ReportScheduledWorkflow(swf *util.ScheduledWorkflow) error
	ReadArtifact(request *api.ReadArtifactRequest, user string) (*api.ReadArtifactResponse, error)
	ReportRunMetrics(request *api.ReportRunMetricsRequest, user string) (*api.ReportRunMetricsResponse, error)
}

type PipelineClient struct {
	initializeTimeout   time.Duration
	timeout             time.Duration
	reportServiceClient api.ReportServiceClient
	runServiceClient    api.RunServiceClient
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

func (p *PipelineClient) ReportWorkflow(workflow util.ExecutionSpec) error {
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
				workflow.ToStringForStore())
		} else {
			// Retry otherwise
			return util.NewCustomError(err, util.CUSTOM_CODE_TRANSIENT,
				"Error while reporting workflow resource (code: %v, message: %v): %v, %+v",
				statusCode.Code(),
				statusCode.Message(),
				err.Error(),
				workflow.ToStringForStore())
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
func (p *PipelineClient) ReadArtifact(request *api.ReadArtifactRequest, user string) (*api.ReadArtifactResponse, error) {
	pctx := context.Background()
	if user != "" {
		pctx = metadata.AppendToOutgoingContext(pctx, getKubeflowUserIDHeader(),
			getKubeflowUserIDPrefix()+user)
	}
	ctx, cancel := context.WithTimeout(pctx, time.Minute)
	defer cancel()

	response, err := p.runServiceClient.ReadArtifact(ctx, request)
	if err != nil {
		// TODO(hongyes): check NotFound error code before skip the error.
		return nil, nil
	}

	return response, nil
}

// ReportRunMetrics reports run metrics to run service.
func (p *PipelineClient) ReportRunMetrics(request *api.ReportRunMetricsRequest, user string) (*api.ReportRunMetricsResponse, error) {
	pctx := context.Background()
	if user != "" {
		pctx = metadata.AppendToOutgoingContext(pctx, getKubeflowUserIDHeader(),
			getKubeflowUserIDPrefix()+user)
	}
	ctx, cancel := context.WithTimeout(pctx, time.Minute)
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

// TODO use config file & viper and "github.com/kubeflow/pipelines/backend/src/apiserver/common.GetKubeflowUserIDHeader()"
func getKubeflowUserIDHeader() string {
	if value, ok := os.LookupEnv(common.KubeflowUserIDHeader); ok {
		return value
	}
	return common.GoogleIAPUserIdentityHeader
}

// TODO use of viper & viper and "github.com/kubeflow/pipelines/backend/src/apiserver/common.GetKubeflowUserIDPrefix()"
func getKubeflowUserIDPrefix() string {
	if value, ok := os.LookupEnv(common.KubeflowUserIDPrefix); ok {
		return value
	}
	return common.GoogleIAPUserIdentityPrefix
}
