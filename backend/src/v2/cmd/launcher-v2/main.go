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
	"strconv"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/v2/component"
	"github.com/kubeflow/pipelines/backend/src/v2/config"
)

// TODO: use https://github.com/spf13/cobra as a framework to create more complex CLI tools with subcommands.
var (
	copy                           = flag.String("copy", "", "copy this binary to specified destination path")
	pipelineName                   = flag.String("pipeline_name", "", "pipeline context name")
	runID                          = flag.String("run_id", "", "pipeline run uid")
	parentDagID                    = flag.Int64("parent_dag_id", 0, "parent DAG execution ID")
	executorType                   = flag.String("executor_type", "container", "The type of the ExecutorSpec")
	executionID                    = flag.Int64("execution_id", 0, "Execution ID of this task.")
	executorInputJSON              = flag.String("executor_input", "", "The JSON-encoded ExecutorInput.")
	componentSpecJSON              = flag.String("component_spec", "", "The JSON-encoded ComponentSpec.")
	importerSpecJSON               = flag.String("importer_spec", "", "The JSON-encoded ImporterSpec.")
	taskSpecJSON                   = flag.String("task_spec", "", "The JSON-encoded TaskSpec.")
	podName                        = flag.String("pod_name", "", "Kubernetes Pod name.")
	podUID                         = flag.String("pod_uid", "", "Kubernetes Pod UID.")
	mlmdServerAddress              = flag.String("mlmd_server_address", "", "The MLMD gRPC server address.")
	mlmdServerPort                 = flag.String("mlmd_server_port", "8080", "The MLMD gRPC server port.")
	mlPipelineServiceTLSEnabledStr = flag.String("mlPipelineServiceTLSEnabled", "false", "Set to 'true' if mlpipeline api server serves over TLS (default: 'false').")
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

	if *copy != "" {
		// copy is used to copy this binary to a shared volume
		// this is a special command, ignore all other flags by returning
		// early
		return component.CopyThisBinary(*copy)
	}
	namespace, err := config.InPodNamespace()
	if err != nil {
		return err
	}

	mlPipelineServiceTLSEnabled, err := strconv.ParseBool(*mlPipelineServiceTLSEnabledStr)
	if err != nil {
		return err
	}
	launcherV2Opts := &component.LauncherV2Options{
		Namespace:            namespace,
		PodName:              *podName,
		PodUID:               *podUID,
		MLMDServerAddress:    *mlmdServerAddress,
		MLMDServerPort:       *mlmdServerPort,
		PipelineName:         *pipelineName,
		RunID:                *runID,
		MLPipelineTLSEnabled: mlPipelineServiceTLSEnabled,
	}

	switch *executorType {
	case "importer":
		importerLauncherOpts := &component.ImporterLauncherOptions{
			PipelineName: *pipelineName,
			RunID:        *runID,
			ParentDagID:  *parentDagID,
		}
		importerLauncher, err := component.NewImporterLauncher(ctx, *componentSpecJSON, *importerSpecJSON, *taskSpecJSON, launcherV2Opts, importerLauncherOpts)
		if err != nil {
			return err
		}
		if err := importerLauncher.Execute(ctx); err != nil {
			return err
		}
		return nil
	case "container":
		launcher, err := component.NewLauncherV2(ctx, *executionID, *executorInputJSON, *componentSpecJSON, flag.Args(), launcherV2Opts)
		if err != nil {
			return err
		}
		glog.V(5).Info(launcher.Info())
		if err := launcher.Execute(ctx); err != nil {
			return err
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
