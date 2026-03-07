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

// Package api provides HTTP DTOs used by the driver server.
package api

type DriverPluginArgs struct {
	CachedDecisionPath      string `json:"cached_decision_path"`
	Component               string `json:"component,omitempty"`
	Container               string `json:"container,omitempty"`
	DagExecutionID          string `json:"dag_execution_id"`
	IterationIndex          string `json:"iteration_index"`
	HTTPProxy               string `json:"http_proxy"`
	HTTPSProxy              string `json:"https_proxy"`
	NoProxy                 string `json:"no_proxy"`
	KubernetesConfig        string `json:"kubernetes_config,omitempty"`
	RuntimeConfig           string `json:"runtime_config,omitempty"`
	PipelineName            string `json:"pipeline_name"`
	PublishLogs             string `json:"publish_logs,omitempty"`
	RunID                   string `json:"run_id"`
	RunName                 string `json:"run_name"`
	RunDisplayName          string `json:"run_display_name"`
	TaskName                string `json:"task_name"`
	Task                    string `json:"task"`
	Type                    string `json:"type"`
	CacheDisabledFlag       bool   `json:"cache_disabled"`
	ExecutionIDPath         string `json:"execution_id_path"`
	IterationCountPath      string `json:"iteration_count_path"`
	ConditionPath           string `json:"condition_path"`
	PodSpecPatchPath        string `json:"pod_spec_patch_path"`
	MLMDServerAddress       string `json:"mlmd_server_address"`
	MLMDServerPort          string `json:"mlmd_server_port"`
	MlPipelineServerAddress string `json:"ml_pipeline_server_address"`
	MlPipelineServerPort    string `json:"ml_pipeline_server_port"`
	MlPipelineTLSEnabled    bool   `json:"ml_pipeline_tls_enabled"`
	MetadataTLSEnabled      bool   `json:"metadata_tls_enabled"`
	CACertPath              string `json:"ca_cert_path"`
	LogLevel                string `json:"log_level"`
	DefaultRunAsUser        *int64 `json:"default_run_as_user,omitempty"`
	DefaultRunAsGroup       *int64 `json:"default_run_as_group,omitempty"`
	DefaultRunAsNonRoot     string `json:"default_run_as_non_root,omitempty"`
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
