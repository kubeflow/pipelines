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
	"strconv"
)

// unsetProxyArgValue marks a proxy argument the compiler failed to fill in.
const unsetProxyArgValue = "unset"

// RuntimeArgs contains runtime-only settings passed to driver plugin handlers.
type RuntimeArgs map[string]string

// UnmarshalJSON decodes runtime settings from the plugin argument's JSON string.
func (r *RuntimeArgs) UnmarshalJSON(value []byte) error {
	if string(value) == "null" {
		return nil
	}
	var runtimeArgsJSON string
	if err := json.Unmarshal(value, &runtimeArgsJSON); err != nil {
		return fmt.Errorf("runtime_args must be a JSON object string: %w", err)
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

// DriverPluginArgs contains the arguments for one driver invocation.
type DriverPluginArgs struct {
	ExitTaskName                    string      `json:"exit_task_name,omitempty"`
	ExitTaskStatus                  string      `json:"exit_task_status,omitempty"`
	DriverRetryEnabled              string      `json:"driver_retry_enabled,omitempty"`
	DriverRetryAttempt              string      `json:"driver_retry_attempt,omitempty"`
	DriverRetryMaxCount             string      `json:"driver_retry_max_count,omitempty"`
	DriverRetryGeneration           string      `json:"driver_retry_generation,omitempty"`
	DriverRetryFinalize             bool        `json:"driver_retry_finalize,omitempty"`
	DriverRetryStatus               string      `json:"driver_retry_status,omitempty"`
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
	DriverType                      DriverType  `json:"type"`
	CacheDisabled                   bool        `json:"cache_disabled"`
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

// Verbosity parses log_level as a nonnegative glog verbosity.
func (a DriverPluginArgs) Verbosity() (int, error) {
	level, err := strconv.Atoi(a.LogLevel)
	if err != nil || level < 0 {
		return 0, fmt.Errorf("invalid log_level %q; provide a nonnegative integer verbosity", a.LogLevel)
	}
	return level, nil
}

// PublishDriverLogs parses publish_logs, which also governs driver log upload.
func (a DriverPluginArgs) PublishDriverLogs() (bool, error) {
	publish, err := strconv.ParseBool(a.PublishLogs)
	if err != nil {
		return false, fmt.Errorf("invalid publish_logs %q; provide true or false", a.PublishLogs)
	}
	return publish, nil
}

// validate checks the decoded arguments against the fields present on the wire.
// Required fields may be empty, except namespace, which selects the workflow.
func (a DriverPluginArgs) validate(present map[string]any) error {
	if err := a.DriverType.Validate(); err != nil {
		return err
	}
	for _, name := range a.DriverType.RequiredArgFields() {
		if _, ok := present[name]; !ok {
			return fmt.Errorf("driver argument %q is required for %s drivers; recompile the pipeline with a current API server", name, a.DriverType)
		}
	}
	if a.Namespace == "" {
		return fmt.Errorf("driver argument \"namespace\" is empty; set it to the workflow namespace")
	}
	for name, value := range map[string]string{"http_proxy": a.HTTPProxy, "https_proxy": a.HTTPSProxy, "no_proxy": a.NoProxy} {
		if value == unsetProxyArgValue {
			return fmt.Errorf("driver argument %q is %q; pass the proxy value or an empty string", name, value)
		}
	}
	if _, err := a.Verbosity(); err != nil {
		return err
	}
	_, err := a.PublishDriverLogs()
	return err
}
