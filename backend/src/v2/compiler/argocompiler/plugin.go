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

package argocompiler

import (
	"fmt"
	"strings"

	wfapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/driver/driverapi"
	"github.com/kubeflow/pipelines/backend/src/v2/config"
)

func (c *workflowCompiler) driverPluginArgs(driverType string) map[string]interface{} {
	args := map[string]interface{}{
		"type":                       driverType,
		"driver_retry_generation":    "{{workflow.annotations." + util.AnnotationKeyRetryGeneration + "}}",
		"pipeline_name":              c.spec.GetPipelineInfo().GetName(),
		"run_id":                     runID(),
		"run_name":                   runResourceName(),
		"run_display_name":           c.job.DisplayName,
		"namespace":                  "{{workflow.namespace}}",
		"kfp_token_audience":         c.tokenAudienceForRun(runID()),
		"parent_task_id":             inputValue(paramParentDagTaskID),
		"task_name":                  inputValue(paramTaskName),
		"iteration_index":            inputValue(paramIterationIndex),
		"http_proxy":                 proxy.GetConfig().GetHttpProxy(),
		"https_proxy":                proxy.GetConfig().GetHttpsProxy(),
		"no_proxy":                   proxy.GetConfig().GetNoProxy(),
		"ml_pipeline_server_address": config.GetMLPipelineServerConfig().Address,
		"ml_pipeline_server_port":    config.GetMLPipelineServerConfig().Port,
		"cache_disabled":             c.cacheDisabled,
		"log_level":                  pipelineLogLevelArg(),
		"publish_logs":               publishLogsArg(),
		"ml_pipeline_tls_enabled":    c.mlPipelineTLSEnabled,
	}
	for _, env := range mlPipelineAPIClientEnvVars() {
		args[strings.ToLower(env.Name)] = env.Value
	}
	return args
}

// Create the Argo Workflow executor plugin template with parameters.
// See https://argo-workflows.readthedocs.io/en/latest/executor_plugins/
func driverPlugin(params map[string]interface{}) (*wfapi.Plugin, error) {
	value, err := driverapi.EncodePluginValue(params)
	if err != nil {
		return nil, fmt.Errorf("driver plugin creation error: %w", err)
	}
	return &wfapi.Plugin{Object: wfapi.Object{Value: value}}, nil
}

// updateDriverPluginArgs preserves existing compiler-generated request arguments.
func updateDriverPluginArgs(tmpl *wfapi.Template, updates map[string]any) error {
	if tmpl.Plugin == nil {
		return fmt.Errorf("template %q has no driver plugin", tmpl.Name)
	}
	config, owned, err := driverapi.DecodePluginConfig(tmpl.Plugin.Value)
	if err != nil {
		return err
	}
	if !owned {
		return fmt.Errorf("template %q is not a KFP driver plugin", tmpl.Name)
	}
	for key, value := range updates {
		config.Args[key] = value
	}
	plugin, err := driverPlugin(config.Args)
	if err != nil {
		return err
	}
	tmpl.Plugin = plugin
	return nil
}
