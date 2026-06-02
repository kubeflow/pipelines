// Copyright 2025 The Kubeflow Authors
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

package apiclient

import (
	"crypto/tls"
	"fmt"
	"os"

	gc "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// Client provides typed clients for KFP v2beta1 API services used by driver/launcher.
type Client struct {
	Run      gc.RunServiceClient
	Pipeline gc.PipelineServiceClient
	Artifact gc.ArtifactServiceClient
	Conn     *grpc.ClientConn
	Endpoint string
}

// Config holds connection options.
type Config struct {
	// Endpoint in host:port form, e.g. ml-pipeline.kubeflow:8887
	Endpoint string
}

// FromEnv builds a Config from environment with sensible defaults.
// KFP_API_ADDRESS and KFP_API_PORT are used; default is ml-pipeline.kubeflow:8887.
func FromEnv() *Config {
	addr := os.Getenv("KFP_API_ADDRESS")
	port := os.Getenv("KFP_API_PORT")
	endpoint := "ml-pipeline.kubeflow:8887"
	if addr != "" && port != "" {
		endpoint = fmt.Sprintf("%s:%s", addr, port)
	}
	return &Config{Endpoint: endpoint}
}

// New creates a new API client connection.
func New(cfg *Config, tlsCfg *tls.Config) (*Client, error) {
	if cfg == nil || cfg.Endpoint == "" {
		return nil, fmt.Errorf("invalid config: missing endpoint")
	}
	creds := insecure.NewCredentials()
	if tlsCfg != nil {
		creds = credentials.NewTLS(tlsCfg)
	}
	conn, err := grpc.NewClient(
		cfg.Endpoint,
		grpc.WithTransportCredentials(creds),
		grpc.WithUnaryInterceptor(authUnaryInterceptor), // Add auth interceptor for all requests
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to KFP API at %s: %w", cfg.Endpoint, err)
	}
	return &Client{
		Run:      gc.NewRunServiceClient(conn),
		Pipeline: gc.NewPipelineServiceClient(conn),
		Artifact: gc.NewArtifactServiceClient(conn),
		Conn:     conn,
		Endpoint: cfg.Endpoint,
	}, nil
}

// Close closes the underlying gRPC connection.
func (c *Client) Close() error {
	if c == nil || c.Conn == nil {
		return nil
	}
	return c.Conn.Close()
}
