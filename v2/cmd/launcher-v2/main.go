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

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/v2/component"
)

var (
	copy              = flag.String("copy", "", "copy this binary to specified destination path")
	executionID       = flag.Int64("execution_id", 0, "Execution ID of this task.")
	executorInputJSON = flag.String("executor_input", "", "The JSON-encoded ExecutorInput.")
	namespace         = flag.String("namespace", "", "The Kubernetes namespace this Pod belongs to.")
	podName           = flag.String("pod_name", "", "Kubernetes Pod name.")
	podUID            = flag.String("pod_uid", "", "Kubernetes Pod UID.")
	pipelineRoot      = flag.String("pipeline_root", "", "The root output directory in which to store output artifacts.")
	mlmdServerAddress = flag.String("mlmd_server_address", "", "The MLMD gRPC server address.")
	mlmdServerPort    = flag.String("mlmd_server_port", "8080", "The MLMD gRPC server port.")
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

	opts := &component.LauncherV2Options{
		Namespace:         *namespace,
		PodName:           *podName,
		PodUID:            *podUID,
		PipelineRoot:      *pipelineRoot,
		MLMDServerAddress: *mlmdServerAddress,
		MLMDServerPort:    *mlmdServerPort,
	}
	launcher, err := component.NewLauncherV2(*executionID, *executorInputJSON, flag.Args(), opts)
	if err != nil {
		return err
	}
	if err := launcher.Execute(ctx); err != nil {
		return err
	}
	return nil
}

// Use WARNING default logging level to facilitate troubleshooting.
func init() {
	flag.Set("logtostderr", "true")
	// Change the WARNING to INFO level for debugging.
	flag.Set("stderrthreshold", "WARNING")
}
