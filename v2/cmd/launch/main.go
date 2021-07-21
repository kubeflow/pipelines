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
package main

import (
	"context"
	"flag"
	"strconv"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/v2/component"
)

var (
	mlmdServerAddress = flag.String("mlmd_server_address", "", "The MLMD gRPC server address.")
	mlmdServerPort    = flag.String("mlmd_server_port", "8080", "The MLMD gRPC server port.")
	runtimeInfoJSON   = flag.String("runtime_info_json", "", "The JSON-encoded RuntimeInfo dictionary.")
	image             = flag.String("container_image", "", "The current container image name.")
	taskName          = flag.String("task_name", "", "The current task name.")
	pipelineName      = flag.String("pipeline_name", "", "The current pipeline name.")
	runID             = flag.String("run_id", "", "The current pipeline run ID.")
	runResource       = flag.String("run_resource", "", "The current pipeline's corresponding Kubernetes resource. e.g. workflows.argoproj.io/workflow-name")
	namespace         = flag.String("namespace", "", "The Kubernetes namespace this Pod belongs to.")
	podName           = flag.String("pod_name", "", "Kubernetes Pod name.")
	podUID            = flag.String("pod_uid", "", "Kubernetes Pod UID.")
	pipelineRoot      = flag.String("pipeline_root", "", "The root output directory in which to store output artifacts.")
	// Use flag.String instead of flag.Bool here to avoid breaking the logic of parser(parseArgs(flag.Args(), rt) in launcher component
	// With flag.Bool, "--enable_caching true" is not valid syntax (https://pkg.go.dev/flag#hdr-Command_line_flag_syntax)
	enableCaching = flag.String("enable_caching", "false", "Enable caching or not")
)

func main() {
	flag.Parse()
	ctx := context.Background()

	enableCachingBool, err := strconv.ParseBool(*enableCaching)
	if err != nil {
		glog.Exitf("Failed to parse enableCaching %s: %v", *enableCaching, err)
	}

	opts := &component.LauncherOptions{
		PipelineName:      *pipelineName,
		RunID:             *runID,
		RunResource:       *runResource,
		Namespace:         *namespace,
		PodName:           *podName,
		PodUID:            *podUID,
		PipelineRoot:      *pipelineRoot,
		TaskName:          *taskName,
		Image:             *image,
		MLMDServerAddress: *mlmdServerAddress,
		MLMDServerPort:    *mlmdServerPort,
		EnableCaching:     enableCachingBool,
	}
	launcher, err := component.NewLauncher(*runtimeInfoJSON, opts)
	if err != nil {
		glog.Exitf("Failed to create component launcher: %v", err)
	}

	if err := launcher.RunComponent(ctx); err != nil {
		glog.Exitf("Failed to execute component: %v", err)
	}
}

// Use WARNING default logging level to facilitate troubleshooting.
func init() {
	flag.Set("logtostderr", "true")
	// Change the WARNING to INFO level for debugging.
	flag.Set("stderrthreshold", "WARNING")
}
