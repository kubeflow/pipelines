// Copyright 2021 The Kubeflow Authors
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

// Package metadata contains types to record/retrieve metadata stored in MLMD
// for individual pipeline steps.
package kfp

import (
	"os"
	"fmt"
	"github.com/golang/glog"
	kfp "github.com/kubeflow/pipelines/v2/third_party/kfp_goclient"
	"google.golang.org/grpc"
)

const (
	// MaxGRPCMessageSize contains max grpc message size supported by the client
	MaxClientGRPCMessageSize = 100 * 1024 * 1024
	// The endpoint uses Kubernetes service DNS name with namespace:
	//https://kubernetes.io/docs/concepts/services-networking/service/#dns
	defaultKFPEndpoint = "ml-pipeline.kubeflow:8888"
)


// Client is an MLMD service client.
type Client struct {
	svc kfp.TaskServiceClient
}

// NewClient creates a Client.
func NewClient() (*Client, error) {
	conn, err := grpc.Dial(kfpDefaultEndpoint(), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(MaxClientGRPCMessageSize)), grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("metadata.NewClient() failed: %w", err)
	}

	return &Client{
		svc: kfp.NewTaskServiceClient(conn),
	}, nil
}

func kfpDefaultEndpoint() string {
	// Discover ml-pipeline in the same namespace by env var.
	// https://kubernetes.io/docs/concepts/services-networking/service/#environment-variables
	kfpHost := os.Getenv("ML_PIPELINE_SERVICE_HOST")
	kfpPort := os.Getenv("ML_PIPELINE_SERVICE_PORT")
	if kfpHost != "" && kfpPort != "" {
		// If there is a ml-pipeline Kubernetes service in the same namespace,
		// ML_PIPELINE_SERVICE_HOST and ML_PIPELINE_SERVICE_PORT env vars should
		// exist by default, so we use it as default.
		return kfpHost + ":" + kfpPort
	}
	// If the env vars do not exist, use default ml-pipeline endpoint `ml-pipeline.kubeflow:8888`.
	glog.Infof("Cannot detect ml-pipeline in the same namespace, default to %s as KFP endpoint.", defaultKFPEndpoint)
	return defaultKFPEndpoint
}
