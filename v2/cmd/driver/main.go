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
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/golang/glog"
	"github.com/golang/protobuf/jsonpb"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/v2/driver"
	"github.com/kubeflow/pipelines/v2/metadata"
)

const (
	pipelineNameArg      = "pipeline_name"
	runIDArg             = "run_id"
	componentArg         = "component"
	taskArg              = "task"
	mlmdServerAddressArg = "mlmd_server_address"
	mlmdServerPortArg    = "mlmd_server_port"
	executionIDPathArg   = "execution_id_path"
	contextIDPathArg     = "context_id_path"
)

var (
	// inputs
	pipelineName      = flag.String(pipelineNameArg, "", "pipeline context name")
	runID             = flag.String(runIDArg, "", "pipeline run uid")
	componentSpecJson = flag.String(componentArg, "", "component spec")
	taskSpecJson      = flag.String(taskArg, "", "task spec")
	// config
	mlmdServerAddress = flag.String(mlmdServerAddressArg, "", "MLMD server address")
	mlmdServerPort    = flag.String(mlmdServerPortArg, "", "MLMD server port")
	// output paths
	executionIDPath = flag.String(executionIDPathArg, "", "Exeucution ID output path")
	contextIDPath   = flag.String(contextIDPathArg, "", "Context ID output path")
)

// func RootDAG(pipelineName string, runID string, component *pipelinespec.ComponentSpec, task *pipelinespec.PipelineTaskSpec, mlmd *metadata.Client) (*Execution, error) {

func main() {
	flag.Parse()
	err := validate()
	if err != nil {
		glog.Exitf("%v", err)
	}
	err = drive()
	if err != nil {
		glog.Exitf("%v", err)
	}
}

// Use WARNING default logging level to facilitate troubleshooting.
func init() {
	flag.Set("logtostderr", "true")
	// Change the WARNING to INFO level for debugging.
	flag.Set("stderrthreshold", "WARNING")
}

func validate() error {
	if err := notEmpty(pipelineNameArg, pipelineName); err != nil {
		return err
	}
	if err := notEmpty(runIDArg, runID); err != nil {
		return err
	}
	if err := notEmpty(componentArg, componentSpecJson); err != nil {
		return err
	}
	if err := notEmpty(taskArg, taskSpecJson); err != nil {
		return err
	}
	if err := notEmpty(executionIDPathArg, executionIDPath); err != nil {
		return err
	}
	if err := notEmpty(contextIDPathArg, contextIDPath); err != nil {
		return err
	}
	// mlmdServerAddress and mlmdServerPort can be empty
	return nil
}

func drive() error {
	componentSpec := &pipelinespec.ComponentSpec{}
	if err := jsonpb.UnmarshalString(*componentSpecJson, componentSpec); err != nil {
		return fmt.Errorf("Failed to unmarshal component spec, error: %w, job: %v", err, componentSpecJson)
	}
	taskSpec := &pipelinespec.PipelineTaskSpec{}
	if err := jsonpb.UnmarshalString(*taskSpecJson, taskSpec); err != nil {
		return fmt.Errorf("Failed to unmarshal task spec, error: %w, task: %v", err, taskSpecJson)
	}
	client, err := newMlmdClient()
	if err != nil {
		return err
	}
	execution, err := driver.RootDAG(*pipelineName, *runID, componentSpec, taskSpec, client)
	if err != nil {
		return err
	}
	err = os.MkdirAll(filepath.Dir(*executionIDPath), 0o755)
	if err == nil { // no error
		err = ioutil.WriteFile(*executionIDPath, []byte(fmt.Sprint(execution.ID)), 0o644)
	}
	if err != nil {
		return fmt.Errorf("Failed to write execution ID to %q: %w", *executionIDPath, err)
	}
	err = os.MkdirAll(filepath.Dir(*contextIDPath), 0o755)
	if err == nil { // no error
		err = ioutil.WriteFile(*contextIDPath, []byte(fmt.Sprint(execution.Context)), 0o644)
	}
	if err != nil {
		return fmt.Errorf("Failed to write context ID to %q: %w", *contextIDPath, err)
	}
	return nil
}

func notEmpty(arg string, value *string) error {
	if value == nil || *value == "" {
		return argErr(arg, "must be non empty")
	}
	return nil
}

func argErr(arg, msg string) error {
	return fmt.Errorf("argument --%s: %s", arg, msg)
}

func newMlmdClient() (*metadata.Client, error) {
	mlmdConfig := metadata.DefaultConfig()
	if *mlmdServerAddress != "" && *mlmdServerPort != "" {
		mlmdConfig.Address = *mlmdServerAddress
		mlmdConfig.Port = *mlmdServerPort
	}
	return metadata.NewClient(mlmdConfig.Address, mlmdConfig.Port)
}
