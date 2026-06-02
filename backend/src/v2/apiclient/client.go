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
	"strconv"
	"time"

	gc "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	KFPAPIAddressEnvVar               = "KFP_API_ADDRESS"
	KFPAPIPortEnvVar                  = "KFP_API_PORT"
	KFPAPIGRPCBackoffBaseDelayEnvVar  = "ML_PIPELINE_GRPC_BACKOFF_BASE_DELAY"
	KFPAPIGRPCBackoffMultiplierEnvVar = "ML_PIPELINE_GRPC_BACKOFF_MULTIPLIER"
	KFPAPIGRPCBackoffJitterEnvVar     = "ML_PIPELINE_GRPC_BACKOFF_JITTER"
	KFPAPIGRPCBackoffMaxDelayEnvVar   = "ML_PIPELINE_GRPC_BACKOFF_MAX_DELAY"
	KFPAPIGRPCMinConnectTimeoutEnvVar = "ML_PIPELINE_GRPC_MIN_CONNECT_TIMEOUT"
	defaultMinConnectTimeout          = 20 * time.Second
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
	// Optional gRPC connection backoff settings propagated from the API server.
	BackoffBaseDelay  string
	BackoffMultiplier string
	BackoffJitter     string
	BackoffMaxDelay   string
	MinConnectTimeout string
}

// FromEnv builds a Config from environment with sensible defaults.
// KFP_API_ADDRESS and KFP_API_PORT are used; default is ml-pipeline.kubeflow:8887.
func FromEnv() *Config {
	return FromEnvWithEndpointOverride("", "")
}

// FromEnvWithEndpointOverride builds a Config from an explicit endpoint override,
// then falls back to environment variables and finally to the default service endpoint.
func FromEnvWithEndpointOverride(address, port string) *Config {
	addr := os.Getenv(KFPAPIAddressEnvVar)
	envPort := os.Getenv(KFPAPIPortEnvVar)
	endpoint := "ml-pipeline.kubeflow:8887"
	switch {
	case address != "" && port != "":
		endpoint = fmt.Sprintf("%s:%s", address, port)
	case addr != "" && envPort != "":
		endpoint = fmt.Sprintf("%s:%s", addr, envPort)
	}
	return &Config{
		Endpoint:          endpoint,
		BackoffBaseDelay:  os.Getenv(KFPAPIGRPCBackoffBaseDelayEnvVar),
		BackoffMultiplier: os.Getenv(KFPAPIGRPCBackoffMultiplierEnvVar),
		BackoffJitter:     os.Getenv(KFPAPIGRPCBackoffJitterEnvVar),
		BackoffMaxDelay:   os.Getenv(KFPAPIGRPCBackoffMaxDelayEnvVar),
		MinConnectTimeout: os.Getenv(KFPAPIGRPCMinConnectTimeoutEnvVar),
	}
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
	dialOptions := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
		grpc.WithUnaryInterceptor(authUnaryInterceptor), // Add auth interceptor for all requests
	}
	connectParams, err := cfg.connectParams()
	if err != nil {
		return nil, fmt.Errorf("invalid gRPC connect params: %w", err)
	}
	if connectParams != nil {
		dialOptions = append(dialOptions, grpc.WithConnectParams(*connectParams))
	}
	conn, err := grpc.NewClient(
		cfg.Endpoint,
		dialOptions...,
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

func (c *Config) connectParams() (*grpc.ConnectParams, error) {
	if c == nil {
		return nil, nil
	}
	if c.BackoffBaseDelay == "" &&
		c.BackoffMultiplier == "" &&
		c.BackoffJitter == "" &&
		c.BackoffMaxDelay == "" &&
		c.MinConnectTimeout == "" {
		return nil, nil
	}

	params := &grpc.ConnectParams{
		Backoff:           backoff.DefaultConfig,
		MinConnectTimeout: defaultMinConnectTimeout,
	}
	if c.BackoffBaseDelay != "" {
		baseDelay, err := time.ParseDuration(c.BackoffBaseDelay)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", KFPAPIGRPCBackoffBaseDelayEnvVar, err)
		}
		params.Backoff.BaseDelay = baseDelay
	}
	if c.BackoffMultiplier != "" {
		multiplier, err := strconv.ParseFloat(c.BackoffMultiplier, 64)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", KFPAPIGRPCBackoffMultiplierEnvVar, err)
		}
		params.Backoff.Multiplier = multiplier
	}
	if c.BackoffJitter != "" {
		jitter, err := strconv.ParseFloat(c.BackoffJitter, 64)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", KFPAPIGRPCBackoffJitterEnvVar, err)
		}
		params.Backoff.Jitter = jitter
	}
	if c.BackoffMaxDelay != "" {
		maxDelay, err := time.ParseDuration(c.BackoffMaxDelay)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", KFPAPIGRPCBackoffMaxDelayEnvVar, err)
		}
		params.Backoff.MaxDelay = maxDelay
	}
	if c.MinConnectTimeout != "" {
		minConnectTimeout, err := time.ParseDuration(c.MinConnectTimeout)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", KFPAPIGRPCMinConnectTimeoutEnvVar, err)
		}
		params.MinConnectTimeout = minConnectTimeout
	}
	return params, nil
}
