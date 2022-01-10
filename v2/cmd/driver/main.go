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
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"github.com/kubeflow/pipelines/v2/cacheutils"
	"github.com/kubeflow/pipelines/v2/driver"

	"github.com/golang/glog"
	"github.com/golang/protobuf/jsonpb"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/v2/config"
	"github.com/kubeflow/pipelines/v2/metadata"
)

const (
	driverTypeArg = "type"
)

var (
	// inputs
	driverType        = flag.String(driverTypeArg, "", "task driver type, one of ROOT_DAG, DAG, CONTAINER")
	pipelineName      = flag.String("pipeline_name", "", "pipeline context name")
	runID             = flag.String("run_id", "", "pipeline run uid")
	componentSpecJson = flag.String("component", "{}", "component spec")
	taskSpecJson      = flag.String("task", "", "task spec")
	runtimeConfigJson = flag.String("runtime_config", "", "jobruntime config")
	iterationIndex    = flag.Int("iteration_index", -1, "iteration index, -1 means not an interation")

	// container inputs
	dagExecutionID    = flag.Int64("dag_execution_id", 0, "DAG execution ID")
	containerSpecJson = flag.String("container", "{}", "container spec")

	// config
	mlmdServerAddress = flag.String("mlmd_server_address", "", "MLMD server address")
	mlmdServerPort    = flag.String("mlmd_server_port", "", "MLMD server port")

	// output paths
	executionIDPath    = flag.String("execution_id_path", "", "Exeucution ID output path")
	iterationCountPath = flag.String("iteration_count_path", "", "Iteration Count output path")
	podSpecPatchPath   = flag.String("pod_spec_patch_path", "", "Pod Spec Patch output path")
	// the value stored in the paths will be either 'true' or 'false'
	cachedDecisionPath = flag.String("cached_decision_path", "", "Cached Decision output path")
	conditionPath      = flag.String("condition_path", "", "Condition output path")
)

// func RootDAG(pipelineName string, runID string, component *pipelinespec.ComponentSpec, task *pipelinespec.PipelineTaskSpec, mlmd *metadata.Client) (*Execution, error) {

func main() {
	flag.Parse()
	err := drive()
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
	if *driverType == "" {
		return fmt.Errorf("argument --%s must be specified", driverTypeArg)
	}
	// validation responsibility lives in driver itself, so we do not validate all other args
	return nil
}

func drive() (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("KFP driver: %w", err)
		}
	}()
	ctx := context.Background()
	if err = validate(); err != nil {
		return err
	}
	glog.Infof("input ComponentSpec:%s\n", prettyPrint(*componentSpecJson))
	componentSpec := &pipelinespec.ComponentSpec{}
	if err := jsonpb.UnmarshalString(*componentSpecJson, componentSpec); err != nil {
		return fmt.Errorf("failed to unmarshal component spec, error: %w\ncomponentSpec: %v", err, componentSpecJson)
	}
	var taskSpec *pipelinespec.PipelineTaskSpec
	if *taskSpecJson != "" {
		glog.Infof("input TaskSpec:%s\n", prettyPrint(*taskSpecJson))
		taskSpec = &pipelinespec.PipelineTaskSpec{}
		if err := jsonpb.UnmarshalString(*taskSpecJson, taskSpec); err != nil {
			return fmt.Errorf("failed to unmarshal task spec, error: %w\ntask: %v", err, taskSpecJson)
		}
	}
	glog.Infof("input ContainerSpec:%s\n", prettyPrint(*containerSpecJson))
	containerSpec := &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{}
	if err := jsonpb.UnmarshalString(*containerSpecJson, containerSpec); err != nil {
		return fmt.Errorf("failed to unmarshal container spec, error: %w\ncontainerSpec: %v", err, containerSpecJson)
	}
	var runtimeConfig *pipelinespec.PipelineJob_RuntimeConfig
	if *runtimeConfigJson != "" {
		glog.Infof("input RuntimeConfig:%s\n", prettyPrint(*runtimeConfigJson))
		runtimeConfig = &pipelinespec.PipelineJob_RuntimeConfig{}
		if err := jsonpb.UnmarshalString(*runtimeConfigJson, runtimeConfig); err != nil {
			return fmt.Errorf("failed to unmarshal runtime config, error: %w\nruntimeConfig: %v", err, runtimeConfigJson)
		}
	}
	namespace, err := config.InPodNamespace()
	if err != nil {
		return err
	}
	client, err := newMlmdClient()
	if err != nil {
		return err
	}
	cacheClient, err := cacheutils.NewClient()
	if err != nil {
		return err
	}
	options := driver.Options{
		PipelineName:   *pipelineName,
		RunID:          *runID,
		Namespace:      namespace,
		Component:      componentSpec,
		Task:           taskSpec,
		DAGExecutionID: *dagExecutionID,
		IterationIndex: *iterationIndex,
	}
	var execution *driver.Execution
	var driverErr error
	switch *driverType {
	case "ROOT_DAG":
		options.RuntimeConfig = runtimeConfig
		execution, driverErr = driver.RootDAG(ctx, options, client)
	case "DAG":
		execution, driverErr = driver.DAG(ctx, options, client)
	case "CONTAINER":
		options.Container = containerSpec
		execution, driverErr = driver.Container(ctx, options, client, cacheClient)
	default:
		err = fmt.Errorf("unknown driverType %s", *driverType)
	}
	if driverErr != nil {
		if execution == nil {
			return driverErr
		}
		defer func() {
			// Override error with driver error, because driver error is more important.
			// However, we continue running, because the following code prints debug info that
			// may be helpful for figuring out why this failed.
			err = driverErr
		}()
	}
	if execution.ID != 0 {
		glog.Infof("output execution.ID=%v", execution.ID)
		if *executionIDPath != "" {
			if err = writeFile(*executionIDPath, []byte(fmt.Sprint(execution.ID))); err != nil {
				return fmt.Errorf("failed to write execution ID to file: %w", err)
			}
		}
	}
	if execution.IterationCount != nil {
		if err = writeFile(*iterationCountPath, []byte(fmt.Sprintf("%v", *execution.IterationCount))); err != nil {
			return fmt.Errorf("failed to write iteration count to file: %w", err)
		}
	}
	if execution.Cached != nil {
		if err = writeFile(*cachedDecisionPath, []byte(strconv.FormatBool(*execution.Cached))); err != nil {
			return fmt.Errorf("failed to write cached decision to file: %w", err)
		}
	}
	if execution.Condition != nil {
		if err = writeFile(*conditionPath, []byte(strconv.FormatBool(*execution.Condition))); err != nil {
			return fmt.Errorf("failed to write condition to file: %w", err)
		}
	}
	if execution.PodSpecPatch != "" {
		glog.Infof("output podSpecPatch=\n%s\n", execution.PodSpecPatch)
		if *podSpecPatchPath == "" {
			return fmt.Errorf("--pod_spec_patch_path is required for container executor drivers")
		}
		if err = writeFile(*podSpecPatchPath, []byte(execution.PodSpecPatch)); err != nil {
			return fmt.Errorf("failed to write pod spec patch to file: %w", err)
		}
	}
	if execution.ExecutorInput != nil {
		marshaler := jsonpb.Marshaler{}
		executorInputJSON, err := marshaler.MarshalToString(execution.ExecutorInput)
		if err != nil {
			return fmt.Errorf("failed to marshal ExecutorInput to JSON: %w", err)
		}
		glog.Infof("output ExecutorInput:%s\n", prettyPrint(executorInputJSON))
	}
	return nil
}

func prettyPrint(jsonStr string) string {
	var prettyJSON bytes.Buffer
	err := json.Indent(&prettyJSON, []byte(jsonStr), "", "  ")
	if err != nil {
		return jsonStr
	}
	return string(prettyJSON.Bytes())
}

func writeFile(path string, data []byte) (err error) {
	if path == "" {
		return fmt.Errorf("path is not specified")
	}
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to write to %s: %w", path, err)
		}
	}()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return ioutil.WriteFile(path, data, 0o644)
}

func newMlmdClient() (*metadata.Client, error) {
	mlmdConfig := metadata.DefaultConfig()
	if *mlmdServerAddress != "" && *mlmdServerPort != "" {
		mlmdConfig.Address = *mlmdServerAddress
		mlmdConfig.Port = *mlmdServerPort
	}
	return metadata.NewClient(mlmdConfig.Address, mlmdConfig.Port)
}
