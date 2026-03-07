// Copyright 2025 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/driver/api"
	"github.com/kubeflow/pipelines/backend/src/v2/cacheutils"
	"github.com/kubeflow/pipelines/backend/src/v2/config"
	"github.com/kubeflow/pipelines/backend/src/v2/driver"
	"google.golang.org/protobuf/encoding/protojson"
)

func ExecutePlugin(w http.ResponseWriter, r *http.Request) {
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			glog.Errorf("Error closing response body: %v", err)
		}
	}(r.Body)

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	args, err := parseDriverRequestArgs(r)
	if err != nil {
		glog.Errorf("Failed to parse driver request args: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if args == nil {
		glog.Errorf("Failed to parse driver request args: nil")
		http.Error(w, "Driver plugin requires at least one argument", http.StatusBadRequest)
		return
	}
	glog.Infof("driver plugin arguments: %v", args)
	execution, err := drive(*args)
	outputs := extractOutputParameters(execution, args.Type)
	if err != nil {
		glog.Errorf("unable to drive execution: %v", err)
		resp := api.DriverResponse{
			Node: api.Node{
				Phase: "Failed",
				Outputs: api.Outputs{
					Parameters: outputs,
				},
				Message: fmt.Sprintf("unable to drive execution: %v", err),
			},
		}
		WriteJSONResponse(w, resp)
		return
	}
	if execution != nil && execution.ExecutorInput != nil {
		executorInputBytes, err := protojson.Marshal(execution.ExecutorInput)
		if err != nil {
			WriteJSONResponse(w, api.DriverResponse{
				Node: api.Node{
					Phase: "Failed",
					Outputs: api.Outputs{
						Parameters: outputs,
					},
					Message: fmt.Sprintf("unable to drive execution: failed to marshal ExecutorInput to JSON: %v", err),
				},
			})
			return
		}
		executorInputJSON := string(executorInputBytes)
		glog.Infof("output ExecutorInput:%s\n", prettyPrint(executorInputJSON))
	}
	resp := api.DriverResponse{
		Node: api.Node{
			Phase: "Succeeded",
			Outputs: api.Outputs{
				Parameters: outputs,
			},
		},
	}
	WriteJSONResponse(w, resp)
}

func parseDriverRequestArgs(r *http.Request) (*api.DriverPluginArgs, error) {
	var body api.DriverRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("failed to parse driver request body: %v", err)
	}
	switch {
	case body.Template == nil:
		return nil, fmt.Errorf("driver request body.Template is empty")
	case body.Template.Plugin == nil:
		return nil, fmt.Errorf("driver request body.Template.Plugin is empty")
	case body.Template.Plugin.DriverPlugin == nil:
		return nil, fmt.Errorf("driver request body.Template.Plugin.DriverPlugin is empty")
	case body.Template.Plugin.DriverPlugin.Args == nil:
		return nil, fmt.Errorf("driver request body.Template.Plugin.Args is empty")
	}
	args := body.Template.Plugin.DriverPlugin.Args
	if err := validate(*args); err != nil {
		return nil, err
	}
	return body.Template.Plugin.DriverPlugin.Args, nil
}

func drive(args api.DriverPluginArgs) (execution *driver.Execution, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("KFP driver: %w", err)
		}
	}()
	ctx := context.Background()

	// Support reading component spec from a file if value starts with @
	// This bypasses exec() argument size limits for large workflows
	if strings.HasPrefix(args.Component, "@") {
		filePath := (args.Component)[1:] // Remove the "@" prefix
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read component spec from file %s: %w", filePath, err)
		}
		args.Component = string(data)
		glog.Infof("Read component spec from file: %s (%d bytes)", filePath, len(data))
	}

	proxy.InitializeConfig(args.HTTPProxy, args.HTTPSProxy, args.NoProxy)

	glog.Infof("input ComponentSpec:%s\n", prettyPrint(args.Component))
	componentSpec := &pipelinespec.ComponentSpec{}
	if err := util.UnmarshalString(args.Component, componentSpec); err != nil {
		return nil, fmt.Errorf("failed to unmarshal component spec, error: %w\ncomponentSpec: %v", err, prettyPrint(args.Component))
	}
	var taskSpec *pipelinespec.PipelineTaskSpec
	if args.Task != "" {
		glog.Infof("input TaskSpec:%s\n", prettyPrint(args.Task))
		taskSpec = &pipelinespec.PipelineTaskSpec{}
		if err := util.UnmarshalString(args.Task, taskSpec); err != nil {
			return nil, fmt.Errorf("failed to unmarshal task spec, error: %w\ntask: %v", err, args.Task)
		}
	}

	containerSpec := &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{}
	if args.Container != "" {
		glog.Infof("input ContainerSpec:%s\n", prettyPrint(args.Container))
		if err := util.UnmarshalString(args.Container, containerSpec); err != nil {
			return nil, fmt.Errorf("failed to unmarshal container spec, error: %w\ncontainerSpec: %v", err, args.Container)
		}
	}
	var runtimeConfig *pipelinespec.PipelineJob_RuntimeConfig
	if args.RuntimeConfig != "" {
		glog.Infof("input RuntimeConfig:%s\n", prettyPrint(args.RuntimeConfig))
		runtimeConfig = &pipelinespec.PipelineJob_RuntimeConfig{}
		if err := util.UnmarshalString(args.RuntimeConfig, runtimeConfig); err != nil {
			return nil, fmt.Errorf("failed to unmarshal runtime config, error: %w\nruntimeConfig: %v", err, args.RuntimeConfig)
		}
	}
	k8sExecCfg, err := parseExecConfigJSON(&args.KubernetesConfig)
	if err != nil {
		return nil, err
	}
	namespace, err := config.InPodNamespace()
	if err != nil {
		return nil, err
	}
	var tlsCfg *tls.Config
	if args.MetadataTLSEnabled {
		tlsCfg, err = util.GetTLSConfig(args.CACertPath)
		if err != nil {
			return nil, fmt.Errorf("unable to drive driver: failed to load TLS configuration: %v", err)
		}
	}
	client, err := newMlmdClient(args.MLMDServerAddress, args.MLMDServerPort, tlsCfg)
	if err != nil {
		return nil, err
	}
	cacheClient, err := cacheutils.NewClient(args.MlPipelineServerAddress, args.MlPipelineServerPort, args.CacheDisabledFlag, tlsCfg)
	if err != nil {
		return nil, err
	}

	dagExecutionID, err := strconv.ParseInt(args.DagExecutionID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse dag execution id, error: %w", err)
	}
	iterationIndex, err := strconv.Atoi(args.IterationIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to parse iteration index, error: %w", err)
	}
	options := driver.Options{
		PipelineName:            args.PipelineName,
		RunID:                   args.RunID,
		RunName:                 args.RunName,
		RunDisplayName:          args.RunDisplayName,
		Namespace:               namespace,
		Component:               componentSpec,
		Task:                    taskSpec,
		DAGExecutionID:          dagExecutionID,
		IterationIndex:          iterationIndex,
		PipelineLogLevel:        args.LogLevel,
		PublishLogs:             args.PublishLogs,
		CacheDisabled:           args.CacheDisabledFlag,
		DriverType:              args.Type,
		TaskName:                args.TaskName,
		MLPipelineServerAddress: args.MlPipelineServerAddress,
		MLPipelineServerPort:    args.MlPipelineServerPort,
		MLPipelineTLSEnabled:    args.MlPipelineTLSEnabled,
		MLMDServerAddress:       args.MLMDServerAddress,
		MLMDServerPort:          args.MLMDServerPort,
		MLMDTLSEnabled:          args.MetadataTLSEnabled,
		CaCertPath:              args.CACertPath,
	}

	var driverErr error
	switch args.Type {
	case RootDag:
		options.RuntimeConfig = runtimeConfig
		execution, driverErr = driver.RootDAG(ctx, options, client)
	case DAG:
		execution, driverErr = driver.DAG(ctx, options, client)
	case CONTAINER:
		options.Container = containerSpec
		options.KubernetesExecutorConfig = k8sExecCfg
		if args.DefaultRunAsUser != nil && *args.DefaultRunAsUser >= 0 {
			options.DefaultRunAsUser = args.DefaultRunAsUser
		}
		if args.DefaultRunAsGroup != nil && *args.DefaultRunAsGroup >= 0 {
			options.DefaultRunAsGroup = args.DefaultRunAsGroup
		}
		if args.DefaultRunAsNonRoot != "" {
			v, err := strconv.ParseBool(args.DefaultRunAsNonRoot)
			if err == nil {
				options.DefaultRunAsNonRoot = &v
			}
		}
		execution, driverErr = driver.Container(ctx, options, client, cacheClient)
	default:
		err = fmt.Errorf("unknown driverType %s", args.Type)
	}
	if driverErr != nil {
		if execution == nil {
			return nil, driverErr
		}
		defer func() {
			// Override error with driver error, because driver error is more important.
			// However, we continue running, because the following code prints debug info that
			// may be helpful for figuring out why this failed.
			err = driverErr
		}()
	}

	return execution, nil
}

func validate(args api.DriverPluginArgs) error {
	switch {
	case args.Type == "":
		return fmt.Errorf("argument type must be specified")
	case args.HTTPProxy == unsetProxyArgValue:
		return fmt.Errorf("argument http_proxy is required but can be an empty value")
	case args.HTTPSProxy == unsetProxyArgValue:
		return fmt.Errorf("argument https_proxy is required but can be an empty value")
	case args.NoProxy == unsetProxyArgValue:
		return fmt.Errorf("argument no_proxy is required but can be an empty value")
	}
	return nil
}

func extractOutputParameters(execution *driver.Execution, driverType string) []api.Parameter {
	if execution == nil {
		return []api.Parameter{}
	}
	var outputs []api.Parameter
	if execution.ID != 0 {
		outputs = append(outputs, api.Parameter{
			Name:  "execution-id",
			Value: fmt.Sprint(execution.ID),
		})
	}
	switch {
	case execution.IterationCount != nil:
		outputs = append(outputs, api.Parameter{
			Name:  "iteration-count",
			Value: fmt.Sprint(*execution.IterationCount),
		})
	case driverType == RootDag:
		outputs = append(outputs, api.Parameter{
			Name:  "iteration-count",
			Value: "0",
		})
	}
	if execution.Cached != nil {
		outputs = append(outputs, api.Parameter{
			Name:  "cached-decision",
			Value: strconv.FormatBool(*execution.Cached),
		})
	}
	if execution.Condition != nil {
		outputs = append(outputs, api.Parameter{
			Name:  "condition",
			Value: strconv.FormatBool(*execution.Condition),
		})
	}
	outputs = append(outputs, api.Parameter{
		Name:  "pod-spec-patch",
		Value: execution.PodSpecPatch,
	})
	return outputs
}

func WriteJSONResponse(w http.ResponseWriter, payload api.DriverResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}
