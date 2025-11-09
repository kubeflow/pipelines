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
	copy                 = flag.String("copy", "", "copy this binary to specified destination path")
	pipelineName         = flag.String("pipeline_name", "", "pipeline context name")
	runID                = flag.String("run_id", "", "pipeline run uid")
	taskID               = flag.String("task_id", "", "pipeline task id (PipelineTaskDetail.task_id)")
	parentTaskID         = flag.String("parent_task_id", "", "Parent PipelineTask ID")
	executorType         = flag.String("executor_type", "container", "The type of the ExecutorSpec")
	executorInputJSON    = flag.String("executor_input", "", "The JSON-encoded ExecutorInput.")
	taskName             = flag.String("task_name", "", "The name of the task.")
	importerSpecJSON     = flag.String("importer_spec", "", "The JSON-encoded ImporterSpec.")
	podName              = flag.String("pod_name", "", "Kubernetes Pod name.")
	podUID               = flag.String("pod_uid", "", "Kubernetes Pod UID.")
	mlPipelineServerAddress = flag.String("ml_pipeline_server_address", "ml-pipeline.kubeflow", "The name of the ML pipeline API server address.")
	mlPipelineServerPort    = flag.String("ml_pipeline_server_port", "8887", "The port of the ML pipeline API server.")
	logLevel             = flag.String("log_level", "1", "The verbosity level to log.")
	publishLogs          = flag.String("publish_logs", "true", "Whether to publish component logs to the object store")
	cacheDisabledFlag    = flag.Bool("cache_disabled", false, "Disable cache globally.")
	fingerPrint          = flag.String("fingerprint", "", "The fingerprint of the pipeline executor.")
	iterationIndex       = flag.Int("iteration_index", -1, "iteration index, -1 means not an interation")
	caCertPath           = flag.String("ca_cert_path", "", "The path to the CA certificate to trust on connections to the ML pipeline API server and metadata server.")
	mlPipelineTLSEnabled = flag.Bool("ml_pipeline_tls_enabled", false, "Set to true if mlpipeline API server serves over TLS.")
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
		return component.CopyThisBinary(*copy)
	}
	namespace := os.Getenv("NAMESPACE")
	if namespace == "" {
		return fmt.Errorf("NAMESPACE environment variable must be set")
	}

	// Create a client manager
	clientOptions := &client_manager.Options{
		MLPipelineTLSEnabled: *mlPipelineTLSEnabled,
		CaCertPath:           *caCertPath,
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
	if parentTaskID == nil || *parentTaskID == "" {
		return fmt.Errorf("parent task id is nil or empty")
	}
	parentTask, err := kfpAPI.GetTask(ctx, &go_client.GetTaskRequest{TaskId: *parentTaskID})
	if err != nil {
		return fmt.Errorf("failed to get parent task: %w", err)
	}

	// Build scope path
	pipelineSpecStruct, err := kfpAPI.FetchPipelineSpecFromRun(ctx, pipelineRun)
	if err != nil {
		return err
	}
	var scopePath util.ScopePath
	scopePath, err = util.ScopePathFromStringPathWithNewTask(
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
		Namespace:         namespace,
		PodName:           *podName,
		PodUID:            *podUID,
		MLPipelineServerAddress: *mlPipelineServerAddress,
		MLPipelineServerPort:    *mlPipelineServerPort,
		PipelineName:      *pipelineName,
		Run:               pipelineRun,
		ParentTask:        parentTask,
		PublishLogs:       *publishLogs,
		CacheDisabled:     *cacheDisabledFlag,
		CachedFingerprint: *fingerPrint,
		ComponentSpec:     componentSpec,
		TaskSpec:          taskSpec,
		ScopePath:         scopePath,
		PipelineSpec:      pipelineSpecStruct,
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
		if taskID != nil && *taskID != "" {
			task, err := kfpAPI.GetTask(ctx, &go_client.GetTaskRequest{TaskId: *taskID})
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

// Use WARNING default logging level to facilitate troubleshooting.
func init() {
	flag.Set("logtostderr", "true")
	// Change the WARNING to INFO level for debugging.
	flag.Set("stderrthreshold", "WARNING")
}
