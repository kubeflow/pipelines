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
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"google.golang.org/grpc/metadata"

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
	ReadArtifact(request *api.ReadArtifactRequest) (*api.ReadArtifactResponse, error)
	ReportRunMetrics(request *api.ReportRunMetricsRequest) (*api.ReportRunMetricsResponse, error)
}

type PipelineClient struct {
	initializeTimeout   time.Duration
	timeout             time.Duration
	reportServiceClient api.ReportServiceClient
	runServiceClient    api.RunServiceClient
	tokenRefresher      TokenRefresherInterface
	httpClient          *http.Client
	httpBaseURL         string
}

func NewPipelineClient(
	initializeTimeout time.Duration,
	timeout time.Duration,
	tokenRefresher TokenRefresherInterface,
	basePath string,
	mlPipelineServiceName string,
	mlPipelineServiceHttpPort string,
	mlPipelineServiceGRPCPort string,
	tlsCfg *tls.Config) (*PipelineClient, error) {
	httpAddress := fmt.Sprintf(addressTemp, mlPipelineServiceName, mlPipelineServiceHttpPort)
	grpcAddress := fmt.Sprintf(addressTemp, mlPipelineServiceName, mlPipelineServiceGRPCPort)
	scheme := "http"
	var httpClient *http.Client
	if tlsCfg != nil {
		scheme = "https"
		tr := &http.Transport{TLSClientConfig: tlsCfg}
		httpClient = &http.Client{Transport: tr}
	} else {
		httpClient = http.DefaultClient
	}
	healthURL := fmt.Sprintf("%s://%s%s/healthz", scheme, httpAddress, basePath)
	err := util.WaitForAPIAvailable(initializeTimeout, healthURL, httpClient)
	if err != nil {
		return nil, errors.Wrapf(err,
			"Failed to initialize pipeline client. Error: %s", err.Error())
	}
	connection, err := util.GetRPCConnection(grpcAddress, tlsCfg)
	if err != nil {
		return nil, errors.Wrapf(err,
			"Failed to get RPC connection. Error: %s", err.Error())
	}

	return &PipelineClient{
		initializeTimeout:   initializeTimeout,
		timeout:             timeout,
		reportServiceClient: api.NewReportServiceClient(connection),
		tokenRefresher:      tokenRefresher,
		runServiceClient:    api.NewRunServiceClient(connection),
		httpClient:          httpClient,
		httpBaseURL:         fmt.Sprintf("%s://%s%s", scheme, httpAddress, basePath),
	}, nil
}

func (p *PipelineClient) ReportWorkflow(workflow util.ExecutionSpec) error {
	pctx := context.Background()
	pctx = metadata.AppendToOutgoingContext(pctx, "Authorization",
		"Bearer "+p.tokenRefresher.GetToken())

	ctx, cancel := context.WithTimeout(pctx, time.Minute)
	defer cancel()

	_, err := p.reportServiceClient.ReportWorkflowV1(ctx, &api.ReportWorkflowRequest{
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
		} else if statusCode.Code() == codes.Unauthenticated && strings.Contains(err.Error(), "service account token has expired") {
			// If unauthenticated because SA token is expired, re-read/refresh the token and try again
			p.tokenRefresher.RefreshToken()
			return util.NewCustomError(err, util.CUSTOM_CODE_TRANSIENT,
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
	pctx := context.Background()
	pctx = metadata.AppendToOutgoingContext(pctx, "Authorization",
		"Bearer "+p.tokenRefresher.GetToken())

	ctx, cancel := context.WithTimeout(pctx, time.Minute)
	defer cancel()

	_, err := p.reportServiceClient.ReportScheduledWorkflowV1(ctx,
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
		} else if statusCode.Code() == codes.Unauthenticated && strings.Contains(err.Error(), "service account token has expired") {
			// If unauthenticated because SA token is expired, re-read/refresh the token and try again
			p.tokenRefresher.RefreshToken()
			return util.NewCustomError(err, util.CUSTOM_CODE_TRANSIENT,
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

// ReadArtifact reads artifact content from run service using HTTP streaming. If the artifact is not present, returns
// nil response.
func (p *PipelineClient) ReadArtifact(request *api.ReadArtifactRequest) (*api.ReadArtifactResponse, error) {
	// Construct the HTTP streaming endpoint URL
	// Format: /apis/v1beta1/runs/{run_id}/nodes/{node_id}/artifacts/{artifact_name}:stream
	url := fmt.Sprintf("%s/apis/v1beta1/runs/%s/nodes/%s/artifacts/%s:stream",
		p.httpBaseURL, request.RunId, request.NodeId, request.ArtifactName)

	// Create HTTP request with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, util.NewCustomError(err, util.CUSTOM_CODE_PERMANENT,
			"Failed to create HTTP request: %v", err.Error())
	}

	// Add authorization header
	req.Header.Set("Authorization", "Bearer "+p.tokenRefresher.GetToken())

	// Make the HTTP request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		if strings.Contains(err.Error(), "service account token has expired") {
			// If unauthenticated because SA token is expired, re-read/refresh the token and try again
			p.tokenRefresher.RefreshToken()
			return nil, util.NewCustomError(err, util.CUSTOM_CODE_TRANSIENT,
				"Error while reading artifact due to token expiry: %v", err.Error())
		}
		return nil, util.NewCustomError(err, util.CUSTOM_CODE_PERMANENT,
			"Failed to make HTTP request: %v", err.Error())
	}
	defer resp.Body.Close()

	// Handle HTTP status codes
	switch resp.StatusCode {
	case http.StatusOK:
		// Success case - read the artifact data
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, util.NewCustomError(err, util.CUSTOM_CODE_PERMANENT,
				"Failed to read artifact data: %v", err.Error())
		}
		return &api.ReadArtifactResponse{Data: data}, nil

	case http.StatusNotFound:
		// Artifact not found - return nil as per original behavior
		return nil, nil

	case http.StatusUnauthorized:
		// Unauthorized - refresh token and return transient error
		p.tokenRefresher.RefreshToken()
		return nil, util.NewCustomError(fmt.Errorf("unauthorized access"), util.CUSTOM_CODE_TRANSIENT,
			"Unauthorized access to artifact")

	case http.StatusForbidden:
		// Forbidden - permanent error
		return nil, util.NewCustomError(fmt.Errorf("forbidden access"), util.CUSTOM_CODE_PERMANENT,
			"Forbidden access to artifact")

	default:
		// Other errors - treat as permanent
		return nil, util.NewCustomError(fmt.Errorf("HTTP %d", resp.StatusCode), util.CUSTOM_CODE_PERMANENT,
			"Failed to read artifact, HTTP status: %d", resp.StatusCode)
	}
}

// ReportRunMetrics reports run metrics to run service.
func (p *PipelineClient) ReportRunMetrics(request *api.ReportRunMetricsRequest) (*api.ReportRunMetricsResponse, error) {
	pctx := context.Background()
	pctx = metadata.AppendToOutgoingContext(pctx, "Authorization",
		"Bearer "+p.tokenRefresher.GetToken())

	ctx, cancel := context.WithTimeout(pctx, time.Minute)
	defer cancel()

	response, err := p.runServiceClient.ReportRunMetricsV1(ctx, request)
	if err != nil {
		statusCode, _ := status.FromError(err)
		if statusCode.Code() == codes.Unauthenticated && strings.Contains(err.Error(), "service account token has expired") {
			// If unauthenticated because SA token is expired, re-read/refresh the token and try again
			p.tokenRefresher.RefreshToken()
			return nil, util.NewCustomError(err, util.CUSTOM_CODE_TRANSIENT,
				"Error while reporting workflow resource (code: %v, message: %v): %v",
				statusCode.Code(),
				statusCode.Message(),
				err.Error())
		}
		// This call should always succeed unless the run doesn't exist or server is broken. In
		// either cases, the job should retry at a later time.
		return nil, util.NewCustomError(err, util.CUSTOM_CODE_TRANSIENT,
			"Error while reporting metrics (%+v): %+v", request, err)
	}
	return response, nil
}
