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
	"flag"
	"fmt"
	"os"

	"github.com/golang/glog"
)

// command line arguments
const (
	argumentMlmdUrl                = "mlmd_url"
	argumentMlmdUrlDefault         = "localhost:8080"
	argumentTaskSpec               = "task_spec"
	argumentExecutorSpec           = "executor_spec"
	argumentExecutionName          = "execution_name"
	argumentDriverType             = "driver_type"
	argumentOutputPathExecutionId  = "output_path_execution_id"
	argumentOutputPathContextName  = "output_path_context_name"
	argumentOutputPathParameters   = "output_path_parameters"
	argumentOutputPathPodSpecPatch = "output_path_pod_spec_patch"
	argumentParentContextName      = "parent_context_name"
)

// command line variables
var (
	mlmdUrl                string
	taskSpecJson           string
	executorSpecJson       string
	executionName          string
	driverType             string
	parentContextName      string
	outputPathExecutionId  string
	outputPathContextName  string
	outputPathParameters   string
	outputPathPodSpecPatch string
)

// driver type enum
const (
	driverTypeDag      = "DAG"
	driverTypeExecutor = "EXECUTOR"
)

func initFlags() {
	flag.StringVar(&mlmdUrl, argumentMlmdUrl, argumentMlmdUrlDefault, "URL of MLMD, defaults to localhost:8080")
	flag.StringVar(&taskSpecJson, argumentTaskSpec, "", "PipelineTaskSpec")
	// TODO(Bobgy): add component spec
	flag.StringVar(&executionName, argumentExecutionName, "", "Unique execution name")
	flag.StringVar(&driverType, argumentDriverType, "", fmt.Sprintf("Driver type, can be '%s' or '%s'", driverTypeDag, driverTypeExecutor))
	flag.StringVar(&parentContextName, argumentParentContextName, "", "Name of parent context. Required if not root DAG.")
	flag.StringVar(&outputPathExecutionId, argumentOutputPathExecutionId, "", "Output path where execution ID should be written to")

	// Required when driving a DAG.
	flag.StringVar(&outputPathContextName, argumentOutputPathContextName, "", "Output path where context name should be written to. Required when driver type is DAG.")

	// Required when driving an executor.
	flag.StringVar(&executorSpecJson, argumentExecutorSpec, "", "ExecutorSpec. Required when driver type is EXECUTOR.")
	// TODO(Bobgy): this will not be used most likely, keep it here for now.
	flag.StringVar(&outputPathParameters, argumentOutputPathParameters, "", "Output path where parameters should be written to.")
	flag.StringVar(&outputPathPodSpecPatch, argumentOutputPathPodSpecPatch, "", "Output path where pod spec patch should be written to. Required when driver type is EXECUTOR.")

	flag.Parse()
	glog.Infof("Driver arguments: %v", os.Args)
}

func validateFlagsOrFatal() {
	if driverType == "" {
		glog.Fatalln(argumentDriverType + " is empty.")
	} else if driverType != driverTypeDag && driverType != driverTypeExecutor {
		glog.Fatalf("invalid %s provided: %s. It should be either '%s' or '%s'", argumentDriverType, driverType, driverTypeDag, driverTypeExecutor)
	}
	if executionName == "" {
		glog.Fatalln(argumentExecutionName + " is empty.")
	}
	if taskSpecJson == "" {
		glog.Fatalln(argumentTaskSpec + " is empty.")
	}
	if driverType == driverTypeExecutor && executorSpecJson == "" {
		glog.Fatalln(argumentExecutorSpec + " is empty. It is required for task drivers.")
	}
	if outputPathExecutionId == "" {
		glog.Fatalln(argumentOutputPathExecutionId + " is empty.")
	}
	// Context name is only produced when driving a DAG.
	if driverType == driverTypeDag && outputPathContextName == "" {
		glog.Fatalf("%s is empty. Required when driver type is %s.", argumentOutputPathContextName, driverTypeDag)
	}
	if driverType == driverTypeExecutor {
		// Temporarily commented out, because there's no decision yet, whether
		// drivers need to output parameters or not.

		// Parameters are only produced when driving type is EXECUTOR.
		// if outputPathParameters == "" {
		// 	glog.Fatalf("%s is empty. Required when driver type is %s.", argumentOutputPathParameters, driverTypeExecutor)
		// }
		// Pod spec patch is only produced when driver type is EXECUTOR.
		if outputPathPodSpecPatch == "" {
			glog.Fatalf("%s is empty. Required when driver type is %s.", argumentOutputPathPodSpecPatch, driverTypeExecutor)
		}
	}
}
