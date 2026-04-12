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
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/v2/client_manager"
	"github.com/kubeflow/pipelines/backend/src/v2/component"
	"github.com/kubeflow/pipelines/backend/src/v2/config"
)

// TODO: use https://github.com/spf13/cobra as a framework to create more complex CLI tools with subcommands.
var (
	copy                    = flag.String("copy", "", "copy this binary to specified destination path")
	pipelineName            = flag.String("pipeline_name", "", "pipeline context name")
	runID                   = flag.String("run_id", "", "pipeline run uid")
	parentDagID             = flag.Int64("parent_dag_id", 0, "parent DAG execution ID")
	executorType            = flag.String("executor_type", "container", "The type of the ExecutorSpec")
	executionID             = flag.Int64("execution_id", 0, "Execution ID of this task.")
	executorInputJSON       = flag.String("executor_input", "", "The JSON-encoded ExecutorInput.")
	componentSpecJSON       = flag.String("component_spec", "", "The JSON-encoded ComponentSpec.")
	importerSpecJSON        = flag.String("importer_spec", "", "The JSON-encoded ImporterSpec.")
	taskSpecJSON            = flag.String("task_spec", "", "The JSON-encoded TaskSpec.")
	podName                 = flag.String("pod_name", "", "Kubernetes Pod name.")
	podUID                  = flag.String("pod_uid", "", "Kubernetes Pod UID.")
	mlPipelineServerAddress = flag.String("ml_pipeline_server_address", "ml-pipeline.kubeflow", "The name of the ML pipeline API server address.")
	mlPipelineServerPort    = flag.String("ml_pipeline_server_port", "8887", "The port of the ML pipeline API server.")
	mlmdServerAddress       = flag.String("mlmd_server_address", "", "The MLMD gRPC server address.")
	mlmdServerPort          = flag.String("mlmd_server_port", "8080", "The MLMD gRPC server port.")
	logLevel                = flag.String("log_level", "1", "The verbosity level to log.")
	publishLogs             = flag.String("publish_logs", "true", "Whether to publish component logs to the object store")
	cacheDisabledFlag       = flag.Bool("cache_disabled", false, "Disable cache globally.")
	caCertPath              = flag.String("ca_cert_path", "", "The path to the CA certificate to trust on connections to the ML pipeline API server and metadata server.")
	mlPipelineTLSEnabled    = flag.Bool("ml_pipeline_tls_enabled", false, "Set to true if mlpipeline API server serves over TLS.")
	metadataTLSEnabled      = flag.Bool("metadata_tls_enabled", false, "Set to true if MLMD serves over TLS.")
	// driverOutputsDir is set in init-container mode. When non-empty, the launcher reads
	// execution-id, executor-input, user-command, user-args, cached-decision and condition
	// from files in this directory instead of from flags.
	driverOutputsDir = flag.String("driver_outputs_dir", "", "Directory to read driver outputs from (init-container driver mode).")
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
	namespace, err := config.InPodNamespace()
	if err != nil {
		return err
	}

	launcherV2Opts := &component.LauncherV2Options{
		Namespace:               namespace,
		PodName:                 *podName,
		PodUID:                  *podUID,
		MLPipelineServerAddress: *mlPipelineServerAddress,
		MLPipelineServerPort:    *mlPipelineServerPort,
		MLMDServerAddress:       *mlmdServerAddress,
		MLMDServerPort:          *mlmdServerPort,
		PipelineName:            *pipelineName,
		RunID:                   *runID,
		PublishLogs:             *publishLogs,
		CacheDisabled:           *cacheDisabledFlag,
		MLPipelineTLSEnabled:    *mlPipelineTLSEnabled,
		MLMDTLSEnabled:          *metadataTLSEnabled,
		CaCertPath:              *caCertPath,
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
		clientOptions := &client_manager.Options{
			MLPipelineServerAddress: launcherV2Opts.MLPipelineServerAddress,
			MLPipelineServerPort:    launcherV2Opts.MLPipelineServerPort,
			MLMDServerAddress:       launcherV2Opts.MLMDServerAddress,
			MLMDServerPort:          launcherV2Opts.MLMDServerPort,
			CacheDisabled:           launcherV2Opts.CacheDisabled,
			MLMDTLSEnabled:          launcherV2Opts.MLMDTLSEnabled,
			CaCertPath:              launcherV2Opts.CaCertPath,
		}
		clientManager, err := client_manager.NewClientManager(clientOptions)
		if err != nil {
			return err
		}

		var (
			resolvedExecutionID   int64
			resolvedExecutorInput string
			resolvedCmdArgs       []string
		)

		if *driverOutputsDir != "" {
			// Init-container mode: read outputs written by the driver init container.
			execID, execInput, userCmd, userArgs, cached, conditionMet, readErr := readDriverOutputs(*driverOutputsDir)
			if readErr != nil {
				return readErr
			}
			if cached {
				glog.Infof("Cache hit; exiting without running user container")
				return nil
			}
			if !conditionMet {
				glog.Infof("Condition not met; exiting without running user container")
				return nil
			}
			resolvedExecutionID = execID
			resolvedExecutorInput = execInput
			resolvedCmdArgs = append(resolvedCmdArgs, userCmd...)
			resolvedCmdArgs = append(resolvedCmdArgs, userArgs...)
		} else {
			// Legacy mode: execution-id and executor-input come from flags, user
			// command+args come from arguments after --.
			resolvedExecutionID = *executionID
			resolvedExecutorInput = *executorInputJSON
			resolvedCmdArgs = flag.Args()
		}

		launcher, err := component.NewLauncherV2(ctx, resolvedExecutionID, resolvedExecutorInput, *componentSpecJSON, resolvedCmdArgs, launcherV2Opts, clientManager)
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

// readDriverOutputs reads the output files written by the driver init container
// from dir (the shared emptyDir volume mounted at driverOutputsMountPath).
func readDriverOutputs(dir string) (
	executionID int64,
	executorInput string,
	userCommand, userArgs []string,
	cached bool,
	conditionMet bool,
	err error,
) {
	readFile := func(name string) (string, error) {
		data, e := os.ReadFile(filepath.Join(dir, name))
		if e != nil {
			return "", fmt.Errorf("failed to read driver output %s: %w", name, e)
		}
		return strings.TrimSpace(string(data)), nil
	}

	execIDStr, err := readFile("execution-id")
	if err != nil {
		return
	}
	executionID, err = strconv.ParseInt(execIDStr, 10, 64)
	if err != nil {
		err = fmt.Errorf("failed to parse execution ID %q: %w", execIDStr, err)
		return
	}

	executorInput, err = readFile("executor-input")
	if err != nil {
		return
	}

	userCmdJSON, err := readFile("user-command")
	if err != nil {
		return
	}
	if err = json.Unmarshal([]byte(userCmdJSON), &userCommand); err != nil {
		err = fmt.Errorf("failed to parse user-command: %w", err)
		return
	}

	userArgsJSON, err := readFile("user-args")
	if err != nil {
		return
	}
	if err = json.Unmarshal([]byte(userArgsJSON), &userArgs); err != nil {
		err = fmt.Errorf("failed to parse user-args: %w", err)
		return
	}

	cachedStr, err := readFile("cached-decision")
	if err != nil {
		return
	}
	cached, err = strconv.ParseBool(cachedStr)
	if err != nil {
		err = fmt.Errorf("failed to parse cached-decision %q: %w", cachedStr, err)
		return
	}

	condStr, err := readFile("condition")
	if err != nil {
		return
	}
	// "nil" means unconditional (always run); "true" means run; "false" means skip.
	if condStr == "nil" {
		conditionMet = true
	} else {
		conditionMet, err = strconv.ParseBool(condStr)
		if err != nil {
			err = fmt.Errorf("failed to parse condition %q: %w", condStr, err)
			return
		}
	}
	return
}

// Use WARNING default logging level to facilitate troubleshooting.
func init() {
	flag.Set("logtostderr", "true")
	// Change the WARNING to INFO level for debugging.
	flag.Set("stderrthreshold", "WARNING")
}
