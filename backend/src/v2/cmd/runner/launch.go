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
	"github.com/kubeflow/pipelines/backend/src/v2/component"
	"github.com/kubeflow/pipelines/backend/src/v2/config"
	"github.com/spf13/cobra"
)

var (
	copy              string
	parentDagID       int64
	executorType      string
	executionID       int64
	executorInputJSON string
	componentSpecJSON string
	importerSpecJSON  string
	taskSpecJSON      string
	podName           string
	podUID            string
)

// launchCmd represents the launch command
var launchCmd = &cobra.Command{
	Use:   "launch",
	Short: "A KFPv2 launcher",
	Long:  `Construct importer or container logic for KFPv2 atomic execution`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("launch called")

		err := run()
		if err != nil {
			glog.Exit(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(launchCmd)

	launchCmd.Flags().StringVarP(&copy, "copy", "c", "", "copy this binary to specified destination path")
	launchCmd.Flags().Int64Var(&parentDagID, "parent_dag_id", 0, "parent DAG execution ID")
	launchCmd.Flags().StringVar(&executorType, "executor_type", "container", "The type of the ExecutorSpec")
	launchCmd.Flags().Int64Var(&executionID, "execution_id", 0, "Execution ID of this task.")
	launchCmd.Flags().StringVar(&executorInputJSON, "executor_input", "", "The JSON-encoded ExecutorInput.")
	launchCmd.Flags().StringVar(&componentSpecJSON, "component_spec", "", "The JSON-encoded ComponentSpec.")
	launchCmd.Flags().StringVar(&importerSpecJSON, "importer_spec", "", "The JSON-encoded ImporterSpec.")
	launchCmd.Flags().StringVar(&taskSpecJSON, "task_spec", "", "The JSON-encoded TaskSpec.")
	launchCmd.Flags().StringVar(&podName, "pod_name", "", "Kubernetes Pod name.")
	launchCmd.Flags().StringVar(&podUID, "pod_uid", "", "Kubernetes Pod UID.")
}

func run() error {
	// flag.Parse()
	ctx := context.Background()

	if copy != "" {
		// copy is used to copy this binary to a shared volume
		// this is a special command, ignore all other flags by returning
		// early
		return component.CopyThisBinary(copy)
	}
	namespace, err := config.InPodNamespace()
	if err != nil {
		return err
	}
	launcherV2Opts := &component.LauncherV2Options{
		Namespace:         namespace,
		PodName:           podName,
		PodUID:            podUID,
		MLMDServerAddress: mlmdServerAddress,
		MLMDServerPort:    mlmdServerPort,
		PipelineName:      pipelineName,
		RunID:             runID,
	}

	switch executorType {
	case "importer":
		importerLauncherOpts := &component.ImporterLauncherOptions{
			PipelineName: pipelineName,
			RunID:        runID,
			ParentDagID:  parentDagID,
		}
		importerLauncher, err := component.NewImporterLauncher(ctx, componentSpecJSON, importerSpecJSON, taskSpecJSON, launcherV2Opts, importerLauncherOpts)
		if err != nil {
			return err
		}
		if err := importerLauncher.Execute(ctx); err != nil {
			return err
		}
		return nil
	case "container":
		launcher, err := component.NewLauncherV2(ctx, executionID, executorInputJSON, componentSpecJSON, flag.Args(), launcherV2Opts)
		if err != nil {
			return err
		}
		glog.V(5).Info(launcher.Info())
		if err := launcher.Execute(ctx); err != nil {
			return err
		}

		return nil

	}
	return fmt.Errorf("unsupported executor type %s", executorType)

}
