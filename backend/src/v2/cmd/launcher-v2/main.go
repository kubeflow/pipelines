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
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/client_manager"
	"github.com/kubeflow/pipelines/backend/src/v2/component"
	"google.golang.org/protobuf/encoding/protojson"
)

var (
	copy                     = flag.String("copy", "", "copy this binary to specified destination path")
	copyTokenTo              = flag.String("copy_token_to", "", "copy the projected KFP token to the specified destination path")
	pipelineName             = flag.String("pipeline_name", "", "pipeline context name")
	runID                    = flag.String("run_id", "", "pipeline run uid")
	taskID                   = flag.String("task_id", "", "pipeline task id (PipelineTask.task_id)")
	legacyTaskID             = flag.String("execution_id", "", "Legacy alias for task id")
	parentTaskID             = flag.String("parent_task_id", "", "Parent PipelineTask ID")
	legacyParentTaskID       = flag.String("parent_dag_id", "", "Legacy alias for parent task id")
	executorType             = flag.String("executor_type", "container", "The type of the ExecutorSpec")
	executorInputJSON        = flag.String("executor_input", "", "The JSON-encoded ExecutorInput.")
	taskName                 = flag.String("task_name", "", "The name of the task.")
	legacyTaskSpecJSON       = flag.String("task_spec", "", "Legacy task spec override")
	legacyComponentSpecJSON  = flag.String("component_spec", "", "Legacy component spec override")
	importerSpecJSON         = flag.String("importer_spec", "", "The JSON-encoded ImporterSpec.")
	podName                  = flag.String("pod_name", "", "Kubernetes Pod name.")
	podUID                   = flag.String("pod_uid", "", "Kubernetes Pod UID.")
	mlPipelineServerAddress  = flag.String("ml_pipeline_server_address", "ml-pipeline.kubeflow", "The name of the ML pipeline API server address.")
	mlPipelineServerPort     = flag.String("ml_pipeline_server_port", "8887", "The port of the ML pipeline API server.")
	legacyMLMDServerAddress  = flag.String("mlmd_server_address", "", "Legacy no-op MLMD server address")
	legacyMLMDServerPort     = flag.String("mlmd_server_port", "", "Legacy no-op MLMD server port")
	logLevel                 = flag.String("log_level", "1", "The verbosity level to log.")
	publishLogs              = flag.String("publish_logs", "true", "Whether to publish component logs to the object store")
	cacheDisabledFlag        = flag.Bool("cache_disabled", false, "Disable cache globally.")
	fingerPrint              = flag.String("fingerprint", "", "The fingerprint of the pipeline executor.")
	iterationIndex           = flag.Int("iteration_index", -1, "iteration index, -1 means not an interation")
	caCertPath               = flag.String("ca_cert_path", "", "The path to the CA certificate to trust on connections to the ML pipeline API server and metadata server.")
	mlPipelineTLSEnabled     = flag.Bool("ml_pipeline_tls_enabled", false, "Set to true if mlpipeline API server serves over TLS.")
	legacyMetadataTLSEnabled = flag.Bool("metadata_tls_enabled", false, "Legacy no-op metadata TLS flag")
)

func main() {
	err := run()
	if err != nil {
		glog.Exit(err)
	}
}

func run() error {
	flag.Parse()
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
		return component.CopyThisBinaryAndToken(*copy, *copyTokenTo)
	}
	namespace, err := resolveNamespace()
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
	resolvedParentTaskID := resolveStringFlag(parentTaskID, legacyParentTaskID)
	if resolvedParentTaskID == "" {
		return fmt.Errorf("parent task id is nil or empty")
	}
	parentTask, err := kfpAPI.GetTask(ctx, &go_client.GetTaskRequest{
		TaskId: resolvedParentTaskID,
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
	var scopePath util.ScopePath
	resolvedTaskName, err := resolveTaskName(*taskName, *legacyTaskSpecJSON)
	if err != nil {
		return err
	}
	scopePath, err = util.ScopePathFromStringPathWithNewTask(
		pipelineSpecStruct,
		parentTask.GetScopePath(),
		resolvedTaskName,
	)
	if err != nil {
		return fmt.Errorf("failed to build scope path: %w", err)
	}

	componentSpec := scopePath.GetLast().GetComponentSpec()
	taskSpec := scopePath.GetLast().GetTaskSpec()
	if componentSpec == nil && *legacyComponentSpecJSON != "" {
		componentSpec = &pipelinespec.ComponentSpec{}
		if err := protojson.Unmarshal([]byte(*legacyComponentSpecJSON), componentSpec); err != nil {
			return fmt.Errorf("failed to unmarshal legacy component spec: %w", err)
		}
	}
	if taskSpec == nil && *legacyTaskSpecJSON != "" {
		taskSpec = &pipelinespec.PipelineTaskSpec{}
		if err := protojson.Unmarshal([]byte(*legacyTaskSpecJSON), taskSpec); err != nil {
			return fmt.Errorf("failed to unmarshal legacy task spec: %w", err)
		}
	}

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
		resolvedTaskID := resolveStringFlag(taskID, legacyTaskID)
		if resolvedTaskID != "" {
			task, err := kfpAPI.GetTask(ctx, &go_client.GetTaskRequest{
				TaskId: resolvedTaskID,
				RunId:  *runID,
			})
			if err != nil {
				return fmt.Errorf("failed to get task: %w", err)
			}
			launcherV2Opts.Task = task
		} else {
			return fmt.Errorf("task id is nil or empty")
		}
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

func resolveNamespace() (string, error) {
	if namespace := os.Getenv("NAMESPACE"); namespace != "" {
		return namespace, nil
	}
	if namespace := os.Getenv("POD_NAMESPACE"); namespace != "" {
		return namespace, nil
	}
	const serviceAccountNamespacePath = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
	namespaceBytes, err := os.ReadFile(serviceAccountNamespacePath)
	if err != nil {
		return "", fmt.Errorf("NAMESPACE environment variable must be set")
	}
	return string(bytes.TrimSpace(namespaceBytes)), nil
}

func resolveStringFlag(primary, legacy *string) string {
	if primary != nil && *primary != "" {
		return *primary
	}
	if legacy != nil {
		return *legacy
	}
	return ""
}

func resolveTaskName(primaryTaskName, rawTaskSpec string) (string, error) {
	if primaryTaskName != "" {
		return primaryTaskName, nil
	}
	if rawTaskSpec == "" {
		return "", fmt.Errorf("task name is nil or empty")
	}
	taskSpec := &pipelinespec.PipelineTaskSpec{}
	if err := protojson.Unmarshal([]byte(rawTaskSpec), taskSpec); err != nil {
		return "", fmt.Errorf("failed to unmarshal legacy task spec: %w", err)
	}
	if taskSpec.GetTaskInfo().GetName() == "" {
		return "", fmt.Errorf("task name is nil or empty")
	}
	return taskSpec.GetTaskInfo().GetName(), nil
}

// Use WARNING default logging level to facilitate troubleshooting.
func init() {
	flag.Set("logtostderr", "true")
	// Change the WARNING to INFO level for debugging.
	flag.Set("stderrthreshold", "WARNING")
}
