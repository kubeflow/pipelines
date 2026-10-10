// Copyright 2025 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package driverapi provides HTTP DTOs used by the driver server.
package driverapi

import (
	"encoding/json"
	"fmt"
)

// RuntimeArgs contains runtime-only settings passed to driver plugin handlers.
type RuntimeArgs map[string]string

func (r *RuntimeArgs) UnmarshalJSON(value []byte) error {
	if string(value) == "null" {
		return nil
	}
	var runtimeArgsJSON string
	if err := json.Unmarshal(value, &runtimeArgsJSON); err != nil {
		return fmt.Errorf("runtime_args must be a JSON object string")
	}
	if runtimeArgsJSON == "" {
		*r = nil
		return nil
	}
	var runtimeArgs map[string]string
	if err := json.Unmarshal([]byte(runtimeArgsJSON), &runtimeArgs); err != nil {
		return fmt.Errorf("failed to unmarshal runtime_args JSON string: %w", err)
	}
	*r = runtimeArgs
	return nil
}

type DriverPluginArgs struct {
	ParentTaskID                    string      `json:"parent_task_id"`
	Namespace                       string      `json:"namespace"`
	MlPipelineGRPCBackoffBaseDelay  string      `json:"ml_pipeline_grpc_backoff_base_delay,omitempty"`
	MlPipelineGRPCBackoffMultiplier string      `json:"ml_pipeline_grpc_backoff_multiplier,omitempty"`
	MlPipelineGRPCBackoffJitter     string      `json:"ml_pipeline_grpc_backoff_jitter,omitempty"`
	MlPipelineGRPCBackoffMaxDelay   string      `json:"ml_pipeline_grpc_backoff_max_delay,omitempty"`
	MlPipelineGRPCMinConnectTimeout string      `json:"ml_pipeline_grpc_min_connect_timeout,omitempty"`
	IterationIndex                  string      `json:"iteration_index"`
	HTTPProxy                       string      `json:"http_proxy"`
	HTTPSProxy                      string      `json:"https_proxy"`
	NoProxy                         string      `json:"no_proxy"`
	KubernetesConfig                string      `json:"kubernetes_config,omitempty"`
	RuntimeConfig                   string      `json:"runtime_config,omitempty"`
	PipelineName                    string      `json:"pipeline_name"`
	PublishLogs                     string      `json:"publish_logs,omitempty"`
	RunID                           string      `json:"run_id"`
	KFPTokenAudience                string      `json:"kfp_token_audience,omitempty"`
	RunName                         string      `json:"run_name"`
	RunDisplayName                  string      `json:"run_display_name"`
	TaskName                        string      `json:"task_name"`
	Type                            string      `json:"type"`
	CacheDisabledFlag               bool        `json:"cache_disabled"`
	MlPipelineServerAddress         string      `json:"ml_pipeline_server_address"`
	MlPipelineServerPort            string      `json:"ml_pipeline_server_port"`
	MlPipelineTLSEnabled            bool        `json:"ml_pipeline_tls_enabled"`
	LogLevel                        string      `json:"log_level"`
	DefaultRunAsUser                *int64      `json:"default_run_as_user,omitempty"`
	DefaultRunAsGroup               *int64      `json:"default_run_as_group,omitempty"`
	DefaultRunAsNonRoot             string      `json:"default_run_as_non_root,omitempty"`
	DefaultHostUsers                string      `json:"default_host_users,omitempty"`
	RuntimeArgs                     RuntimeArgs `json:"runtime_args,omitempty"`
}

type DriverPlugin struct {
	DriverPlugin *DriverPluginContainer `json:"driver-plugin"`
}

type DriverPluginContainer struct {
	Args *DriverPluginArgs `json:"args"`
}

type DriverTemplate struct {
	Plugin *DriverPlugin `json:"plugin"`
}

type DriverRequest struct {
	Template *DriverTemplate `json:"template"`
}
