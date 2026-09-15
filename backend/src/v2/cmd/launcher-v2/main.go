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

// Launcher command for Kubeflow Pipelines v2.
package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/client_manager"
	"github.com/kubeflow/pipelines/backend/src/v2/component"
	"github.com/spf13/viper"
	"google.golang.org/protobuf/encoding/protojson"
)

var (
	copy                    = flag.String("copy", "", "copy this binary to specified destination path")
	pipelineName            = flag.String("pipeline_name", "", "pipeline context name")
	runID                   = flag.String("run_id", "", "pipeline run uid")
	taskID                  = flag.String("task_id", "", "pipeline task id (PipelineTask.task_id)")
	parentTaskID            = flag.String("parent_task_id", "", "Parent PipelineTask ID")
	executorType            = flag.String("executor_type", "container", "The type of the ExecutorSpec")
	executorInputJSON       = flag.String("executor_input", "", "The JSON-encoded ExecutorInput.")
	taskName                = flag.String("task_name", "", "The name of the task.")
	importerSpecJSON        = flag.String("importer_spec", "", "The JSON-encoded ImporterSpec.")
	namespaceFlag           = flag.String("namespace", "", "Kubernetes namespace for runtime operations.")
	podName                 = flag.String("pod_name", "", "Kubernetes Pod name.")
	podUID                  = flag.String("pod_uid", "", "Kubernetes Pod UID.")
	mlPipelineServerAddress = flag.String("ml_pipeline_server_address", "ml-pipeline.kubeflow", "The name of the ML pipeline API server address.")
	mlPipelineServerPort    = flag.String("ml_pipeline_server_port", "8887", "The port of the ML pipeline API server.")
	logLevel                = flag.String("log_level", "1", "The verbosity level to log.")
	publishLogs             = flag.String("publish_logs", "true", "Whether to publish component logs to the object store")
	cacheDisabledFlag       = flag.Bool("cache_disabled", false, "Disable cache globally.")
	fingerPrint             = flag.String("fingerprint", "", "The fingerprint of the pipeline executor.")
	iterationIndex          = flag.Int("iteration_index", -1, "iteration index, -1 means not an interation")
	caCertPath              = flag.String("ca_cert_path", "", "The path to the CA certificate to trust on connections to the ML pipeline API server and metadata server.")
	mlPipelineTLSEnabled    = flag.Bool("ml_pipeline_tls_enabled", false, "Set to true if mlpipeline API server serves over TLS.")
)

// Required flags the driver/compiler must always pass to the launcher, grouped
// by executor type, making the implicit contract fail-fast instead of silently
// falling back to defaults. A flag is required for an executor type when the
// driver/compiler always emits it for that type; only copy (a special mode
// selector that short-circuits before validation) stays optional.
var (
	commonRequiredLauncherFlags = []string{
		"executor_type",
		"pipeline_name",
		"run_id",
		"namespace",
		"pod_name",
		"pod_uid",
		"log_level",
		"publish_logs",
		"cache_disabled",
		"ml_pipeline_tls_enabled",
		"ca_cert_path",
	}
	containerRequiredLauncherFlags = []string{
		"task_id",
		"parent_task_id",
		"executor_input",
		"ml_pipeline_server_address",
		"ml_pipeline_server_port",
		"fingerprint",
		"task_name",
	}
	importerRequiredLauncherFlags = []string{
		"task_name",
		"importer_spec",
		"parent_task_id",
		"iteration_index",
	}
)

// collectProvidedFlags returns the flags explicitly set on the command line.
// flag.Visit reports only flags that were provided, not those left at default.
func collectProvidedFlags() map[string]bool {
	provided := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) {
		provided[f.Name] = true
	})
	return provided
}

func requiredLauncherFlags(executorType string) ([]string, error) {
	required := append([]string{}, commonRequiredLauncherFlags...)
	switch executorType {
	case "container":
		required = append(required, containerRequiredLauncherFlags...)
	case "importer":
		required = append(required, importerRequiredLauncherFlags...)
	default:
		return nil, fmt.Errorf("unsupported executor type %q, must be one of container, importer", executorType)
	}
	return required, nil
}

func validateLauncherFlags(provided map[string]bool, executorType string) error {
	required, err := requiredLauncherFlags(executorType)
	if err != nil {
		return err
	}
	for _, name := range required {
		if !provided[name] {
			return fmt.Errorf("--%s is required for %s executor but was not provided", name, executorType)
		}
	}
	return nil
}

func main() {
	err := run()
	if err != nil {
		glog.Exit(err)
	}
}

func run() error {
	flag.Parse()
	providedFlags := collectProvidedFlags()
	ctx := context.Background()

	glog.Infof("Setting log level to: '%s'", *logLevel)
	err := flag.Set("v", *logLevel)
	if err != nil {
		glog.Warningf("Failed to set log level: %s", err.Error())
	}

	if *copy != "" {
		// copy is used to copy this binary to a shared volume
		// this is a special command, ignore all other flags by returning
		// early
		return component.CopyThisBinary(*copy)
	}
	if err := validateLauncherFlags(providedFlags, *executorType); err != nil {
		return err
	}
	namespace, err := resolveNamespace(*namespaceFlag)
	if err != nil {
		return err
	}

	// Create a client manager
	clientOptions := &client_manager.Options{
		MLPipelineTLSEnabled:    *mlPipelineTLSEnabled,
		CaCertPath:              *caCertPath,
		MLPipelineServerAddress: *mlPipelineServerAddress,
		MLPipelineServerPort:    *mlPipelineServerPort,
	}

	clientManager, err := client_manager.NewClientManager(clientOptions)
	if err != nil {
		return fmt.Errorf("failed to create client manager: %w", err)
	}

	// Fetch Run
	kfpAPI := clientManager.KFPAPIClient()
	fullView := go_client.GetRunRequest_FULL
	pipelineRun, err := kfpAPI.GetRun(ctx, &go_client.GetRunRequest{RunId: *runID, View: &fullView})
	if err != nil {
		return fmt.Errorf("failed to get run: %w", err)
	}

	// Fetch Parent Task
	if *parentTaskID == "" {
		return fmt.Errorf("parent task id is nil or empty")
	}
	parentTask, err := kfpAPI.GetTask(ctx, &go_client.GetTaskRequest{
		TaskId: *parentTaskID,
		RunId:  *runID,
	})
	if err != nil {
		return fmt.Errorf("failed to get parent task: %w", err)
	}

	// Build scope path
	pipelineSpecStruct, err := kfpAPI.FetchPipelineSpecFromRun(ctx, pipelineRun)
	if err != nil {
		return err
	}
	if *taskName == "" {
		return fmt.Errorf("task name is nil or empty")
	}
	scopePath, err := util.ScopePathFromStringPathWithNewTask(
		pipelineSpecStruct,
		parentTask.GetScopePath(),
		*taskName,
	)
	if err != nil {
		return fmt.Errorf("failed to build scope path: %w", err)
	}

	componentSpec := scopePath.GetLast().GetComponentSpec()
	taskSpec := scopePath.GetLast().GetTaskSpec()

	launcherV2Opts := &component.LauncherV2Options{
		Namespace:               namespace,
		PodName:                 *podName,
		PodUID:                  *podUID,
		MLPipelineServerAddress: *mlPipelineServerAddress,
		MLPipelineServerPort:    *mlPipelineServerPort,
		CaCertPath:              *caCertPath,
		PipelineName:            *pipelineName,
		Run:                     pipelineRun,
		ParentTask:              parentTask,
		PublishLogs:             *publishLogs,
		CacheDisabled:           *cacheDisabledFlag,
		CachedFingerprint:       *fingerPrint,
		ComponentSpec:           componentSpec,
		TaskSpec:                taskSpec,
		ScopePath:               scopePath,
		PipelineSpec:            pipelineSpecStruct,
	}

	if iterationIndex != nil && *iterationIndex > -1 {
		launcherV2Opts.IterationIndex = util.Int64Pointer(int64(*iterationIndex))
	}

	switch *executorType {
	case "importer":
		if importerSpecJSON == nil || *importerSpecJSON == "" {
			return fmt.Errorf("importer spec is nil or empty")
		}
		importerSpec := &pipelinespec.PipelineDeploymentConfig_ImporterSpec{}
		err = protojson.Unmarshal([]byte(*importerSpecJSON), importerSpec)
		if err != nil {
			return fmt.Errorf("failed to unmarshal importer spec: %w", err)
		}
		launcherV2Opts.ImporterSpec = importerSpec
		importerLauncher, err := component.NewImporterLauncher(
			launcherV2Opts,
			clientManager,
		)
		if err != nil {
			return fmt.Errorf("failed to create importer launcher: %w", err)
		}
		if err := importerLauncher.Execute(ctx); err != nil {
			return fmt.Errorf("failed to execute importer launcher: %w", err)
		}
		return nil
	case "container":
		// Container task should have a pre-existing task created by the Driver
		if *taskID == "" {
			return fmt.Errorf("task id is nil or empty")
		}
		task, err := kfpAPI.GetTask(ctx, &go_client.GetTaskRequest{
			TaskId: *taskID,
			RunId:  *runID,
		})
		if err != nil {
			return fmt.Errorf("failed to get task: %w", err)
		}
		launcherV2Opts.Task = task
		launcher, err := component.NewLauncherV2(
			*executorInputJSON,
			flag.Args(),
			launcherV2Opts,
			clientManager,
		)
		if err != nil {
			return fmt.Errorf("failed to create launcher: %w", err)
		}
		glog.V(5).Info(launcher.Info())
		if err := launcher.Execute(ctx); err != nil {
			return fmt.Errorf("failed to execute launcher: %w", err)
		}
		return nil

	}
	return fmt.Errorf("unsupported executor type %s", *executorType)

}

func resolveNamespace(explicitNamespace string) (string, error) {
	if explicitNamespace == "" {
		return "", fmt.Errorf("argument --namespace must be specified")
	}
	return explicitNamespace, nil
}

// Use WARNING default logging level to facilitate troubleshooting.
func init() {
	flag.Set("logtostderr", "true")
	// Change the WARNING to INFO level for debugging.
	flag.Set("stderrthreshold", "WARNING")
	viper.AutomaticEnv()
}
