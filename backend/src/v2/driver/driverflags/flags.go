// Copyright 2026 The Kubeflow Authors
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

// Package driverflags registers the shared CLI FlagSet used by the driver
// binary and by compiler tests that validate generated driver arguments.
package driverflags

import "flag"

const (
	DriverTypeArg = "type"
	HTTPProxyArg  = "http_proxy"
	HTTPSProxyArg = "https_proxy"
	NoProxyArg    = "no_proxy"
)

// Values stores pointers returned by flag registration so callers can continue
// reading parsed driver arguments through shared state.
type Values struct {
	DriverType              *string
	PipelineName            *string
	RunID                   *string
	RunName                 *string
	RunDisplayName          *string
	RuntimeConfigJSON       *string
	IterationIndex          *int
	TaskName                *string
	Namespace               *string
	ParentTaskID            *string
	KubernetesConfigJSON    *string
	MLPipelineServerAddress *string
	MLPipelineServerPort    *string
	ParentTaskIDPath        *string
	IterationCountPath      *string
	PodSpecPatchPath        *string
	CachedDecisionPath      *string
	ConditionPath           *string
	LogLevel                *string
	HTTPProxy               *string
	HTTPSProxy              *string
	NoProxy                 *string
	PublishLogs             *string
	CacheDisabled           *bool
	MLPipelineTLSEnabled    *bool
	CACertPath              *string
	DefaultRunAsUser        *int64
	DefaultRunAsGroup       *int64
	DefaultRunAsNonRoot     *string
	DefaultHostUsers        *string
}

// RegisterDriverFlags registers the driver CLI flags on the provided flag set.
func RegisterDriverFlags(fs *flag.FlagSet) *Values {
	return &Values{
		DriverType:              fs.String(DriverTypeArg, "", "task driver type, one of ROOT_DAG, DAG, CONTAINER"),
		PipelineName:            fs.String("pipeline_name", "", "pipeline context name"),
		RunID:                   fs.String("run_id", "", "pipeline run uid"),
		RunName:                 fs.String("run_name", "", "pipeline run name (Kubernetes object name)"),
		RunDisplayName:          fs.String("run_display_name", "", "pipeline run display name"),
		RuntimeConfigJSON:       fs.String("runtime_config", "", "jobruntime config"),
		IterationIndex:          fs.Int("iteration_index", -1, "iteration index, -1 means not an interation"),
		TaskName:                fs.String("task_name", "", "original task name, used for proper input resolution in the container/dag driver"),
		Namespace:               fs.String("namespace", "", "Kubernetes namespace for runtime operations."),
		ParentTaskID:            fs.String("parent_task_id", "", "Parent PipelineTask ID"),
		KubernetesConfigJSON:    fs.String("kubernetes_config", "{}", "kubernetes executor config"),
		MLPipelineServerAddress: fs.String("ml_pipeline_server_address", "ml-pipeline", "The name of the ML pipeline API server address."),
		MLPipelineServerPort:    fs.String("ml_pipeline_server_port", "8887", "The port of the ML pipeline API server."),
		ParentTaskIDPath:        fs.String("parent_task_id_path", "", "Parent Task ID output path"),
		IterationCountPath:      fs.String("iteration_count_path", "", "Iteration Count output path"),
		PodSpecPatchPath:        fs.String("pod_spec_patch_path", "", "Pod Spec Patch output path"),
		CachedDecisionPath:      fs.String("cached_decision_path", "", "Cached Decision output path"),
		ConditionPath:           fs.String("condition_path", "", "Condition output path"),
		LogLevel:                fs.String("log_level", "1", "The verbosity level to log."),
		HTTPProxy:               fs.String(HTTPProxyArg, "", "The proxy for HTTP connections."),
		HTTPSProxy:              fs.String(HTTPSProxyArg, "", "The proxy for HTTPS connections."),
		NoProxy:                 fs.String(NoProxyArg, "", "Addresses that should ignore the proxy."),
		PublishLogs:             fs.String("publish_logs", "true", "Whether to publish component logs to the object store"),
		CacheDisabled:           fs.Bool("cache_disabled", false, "Disable cache globally."),
		MLPipelineTLSEnabled:    fs.Bool("ml_pipeline_tls_enabled", false, "Set to true if mlpipeline API server serves over TLS."),
		CACertPath:              fs.String("ca_cert_path", "", "The path to the CA certificate to trust on connections to the ML pipeline API server and metadata server."),
		DefaultRunAsUser:        fs.Int64("default_run_as_user", -1, "Admin-configured default runAsUser for user containers. -1 means not set."),
		DefaultRunAsGroup:       fs.Int64("default_run_as_group", -1, "Admin-configured default runAsGroup for user containers. -1 means not set."),
		DefaultRunAsNonRoot:     fs.String("default_run_as_non_root", "", "Admin-configured default runAsNonRoot for user containers. Empty means not set."),
		DefaultHostUsers:        fs.String("default_host_users", "", "Administrator-configured default hostUsers for user workload pods. Empty means not set. Set to false to run pods in a dedicated Linux user namespace."),
	}
}
