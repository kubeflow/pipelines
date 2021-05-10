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

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/v2/component"
)

var (
	mlmdServerAddress = flag.String("mlmd_server_address", "", "The MLMD gRPC server address.")
	mlmdServerPort    = flag.String("mlmd_server_port", "8080", "The MLMD gRPC server port.")
	runtimeInfoJSON   = flag.String("runtime_info_json", "", "The JSON-encoded RuntimeInfo dictionary.")
	containerImage    = flag.String("container_image", "", "The current container image name.")
	taskName          = flag.String("task_name", "", "The current task name.")
	pipelineName      = flag.String("pipeline_name", "", "The current pipeline name.")
	pipelineRunID     = flag.String("pipeline_run_id", "", "The current pipeline run ID.")
	pipelineTaskID    = flag.String("pipeline_task_id", "", "The current pipeline task ID.")
	pipelineRoot      = flag.String("pipeline_root", "minio://mlpipeline/v2/artifacts", "The root output directory in which to store output artifacts.")
)

func main() {
	flag.Parse()
	ctx := context.Background()

	opts := &component.LauncherOptions{
		PipelineName:      *pipelineName,
		PipelineRunID:     *pipelineRunID,
		PipelineTaskID:    *pipelineTaskID,
		PipelineRoot:      *pipelineRoot,
		TaskName:          *taskName,
		ContainerImage:    *containerImage,
		MLMDServerAddress: *mlmdServerAddress,
		MLMDServerPort:    *mlmdServerPort,
	}
	launcher, err := component.NewLauncher(*runtimeInfoJSON, opts)
	if err != nil {
		glog.Exitf("Failed to create component launcher: %v", err)
	}

	if err := launcher.RunComponent(ctx, flag.Args()[0], flag.Args()[1:]...); err != nil {
		glog.Exitf("Failed to execute component: %v", err)
	}
}
