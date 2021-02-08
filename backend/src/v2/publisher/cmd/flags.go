// Copyright 2021 Google LLC
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

package cmd

import (
	"flag"
	"fmt"
	"os"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/v2/common"
)

// command line arguments
const (
	argumentMlmdUrl              = "mlmd_url"
	argumentMlmdUrlDefault       = "localhost:8080"
	argumentComponentOutputsSpec = "component_outputs_spec"
	argumentExecutionId          = "execution_id"
	argumentPublisherType        = "publisher_type"
	argumentInputPathParameters  = "input_path_parameters"
)

// command line variables
var (
	mlmdUrl                  string
	componentOutputsSpecJson string
	executionId              int64
	publisherType            string
	inputPathParameters      string
)

func InitFlags() {
	flag.StringVar(&mlmdUrl, argumentMlmdUrl, argumentMlmdUrlDefault, "URL of MLMD, defaults to localhost:8080")
	flag.StringVar(&componentOutputsSpecJson, argumentComponentOutputsSpec, "", "Component outputs spec")
	flag.Int64Var(&executionId, argumentExecutionId, 0, "Execution ID to publish")
	flag.StringVar(&publisherType, argumentPublisherType, "", fmt.Sprintf("Publisher type, can be '%s' or '%s'", common.PublisherType_DAG, common.PublisherType_EXECUTOR))
	flag.StringVar(&inputPathParameters, argumentInputPathParameters, "", "Input path which contains files corresponding to parameter values")

	flag.Parse()
	glog.Infof("Publisher arguments: %v", os.Args)
}

func ValidateFlagsOrFatal() {
	if publisherType == "" {
		glog.Fatalln(argumentPublisherType + " is not provided.")
	} else if publisherType != common.PublisherType_DAG && publisherType != common.PublisherType_EXECUTOR {
		glog.Fatalf("invalid %s provided: %s. It should be either '%s' or '%s'", argumentPublisherType, publisherType, common.PublisherType_DAG, common.PublisherType_EXECUTOR)
	}
	if executionId == 0 {
		glog.Fatalln(argumentExecutionId + " is not provided.")
	}
	if componentOutputsSpecJson == "" {
		glog.Fatalln(argumentComponentOutputsSpec + " is not provided.")
	}
	if inputPathParameters == "" {
		glog.Fatalln(argumentInputPathParameters + " is not provided.")
	}
}
