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
	"time"

	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/driver/driverapi"
	"github.com/kubeflow/pipelines/backend/src/v2/cacheutils"
	"github.com/kubeflow/pipelines/backend/src/v2/config"
	"github.com/kubeflow/pipelines/backend/src/v2/driver"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	"github.com/kubeflow/pipelines/backend/src/v2/objectstore"
	"google.golang.org/protobuf/encoding/protojson"
	"k8s.io/client-go/kubernetes"
)

type driverLogArtifactContext struct {
	Execution        *driver.Execution
	Task             string
	LocalPath        string
	OutputPathPrefix string
	Namespace        string
	PipelineRoot     string
	StoreSessionInfo string
	LogID            string
}

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
	glog.Infof("Received request to execute plugin: %v", r)
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
	execution, err := drive(*args)
	outputs := extractOutputParameters(execution, args.Type)
	if err != nil {
		glog.Errorf("unable to drive execution: %v", err)
		resp := driverapi.DriverResponse{
			Node: driverapi.Node{
				Phase: "Failed",
				Outputs: driverapi.Outputs{
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
			WriteJSONResponse(w, driverapi.DriverResponse{
				Node: driverapi.Node{
					Phase: "Failed",
					Outputs: driverapi.Outputs{
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
	resp := driverapi.DriverResponse{
		Node: driverapi.Node{
			Phase: "Succeeded",
			Outputs: driverapi.Outputs{
				Parameters: outputs,
			},
		},
	}
	WriteJSONResponse(w, resp)
}

func parseDriverRequestArgs(r *http.Request) (*driverapi.DriverPluginArgs, error) {
	var body driverapi.DriverRequest
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

func drive(args driverapi.DriverPluginArgs) (execution *driver.Execution, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("KFP driver: %w", err)
		}
	}()
	var (
		pipelineRoot     string
		storeSessionInfo string
		namespace        string
		outputPathPrefix string
	)
	var pipeline *metadata.Pipeline
	logID := fmt.Sprintf("%d-%v-%v-%v", time.Now().UnixMilli(), args.IterationIndex, args.Type, args.TaskName)
	logDir := "/kfp/log"
	logFile := fmt.Sprintf("%s/%s.log", logDir, logID)
	ctx, f, err := util.WithLogger(context.Background(), logFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create driver logger: %v", err)
	}
	defer func() {
		removeErr := os.Remove(logFile)
		if removeErr != nil {
			glog.Errorf("Failed to remove processed log file: %v", removeErr)
		}
	}()
	defer func() {
		if pipelineRoot != "" {
			logContext := &driverLogArtifactContext{
				Execution:        execution,
				Task:             args.TaskName,
				LocalPath:        logFile,
				LogID:            logID,
				Namespace:        namespace,
				PipelineRoot:     pipelineRoot,
				StoreSessionInfo: storeSessionInfo,
				OutputPathPrefix: outputPathPrefix,
			}
			uploadErr := uploadDriverLogArtifact(ctx, logContext)
			if uploadErr != nil {
				glog.Errorf("Failed to upload driver-logs artifact: %v", uploadErr)
			}
		}
	}()
	defer func() {
		if f != nil {
			closeErr := f.Close()
			if closeErr != nil {
				glog.Errorf("Failed to close file: %v", closeErr)
			}
		}
	}()

	log := util.GetLoggerFrom(ctx)

	log.Infof("driver plugin arguments: %v", args)
	// Support reading component spec from a file if value starts with @
	// This bypasses exec() argument size limits for large workflows
	if strings.HasPrefix(args.Component, "@") {
		filePath := (args.Component)[1:] // Remove the "@" prefix
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read component spec from file %s: %w", filePath, err)
		}
		args.Component = string(data)
		log.Infof("Read component spec from file: %s (%d bytes)", filePath, len(data))
	}

	proxy.InitializeConfig(args.HTTPProxy, args.HTTPSProxy, args.NoProxy)

	log.Infof("input ComponentSpec:%s\n", prettyPrint(args.Component))
	componentSpec := &pipelinespec.ComponentSpec{}
	if err := util.UnmarshalString(args.Component, componentSpec); err != nil {
		return nil, fmt.Errorf("failed to unmarshal component spec, error: %w\ncomponentSpec: %v", err, prettyPrint(args.Component))
	}
	var taskSpec *pipelinespec.PipelineTaskSpec
	if args.Task != "" {
		log.Infof("input TaskSpec:%s\n", prettyPrint(args.Task))
		taskSpec = &pipelinespec.PipelineTaskSpec{}
		if err := util.UnmarshalString(args.Task, taskSpec); err != nil {
			return nil, fmt.Errorf("failed to unmarshal task spec, error: %w\ntask: %v", err, args.Task)
		}
	}

	containerSpec := &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{}
	if args.Container != "" {
		log.Infof("input ContainerSpec:%s\n", prettyPrint(args.Container))
		if err := util.UnmarshalString(args.Container, containerSpec); err != nil {
			return nil, fmt.Errorf("failed to unmarshal container spec, error: %w\ncontainerSpec: %v", err, args.Container)
		}
	}
	var runtimeConfig *pipelinespec.PipelineJob_RuntimeConfig
	if args.RuntimeConfig != "" {
		log.Infof("input RuntimeConfig:%s\n", prettyPrint(args.RuntimeConfig))
		runtimeConfig = &pipelinespec.PipelineJob_RuntimeConfig{}
		if err := util.UnmarshalString(args.RuntimeConfig, runtimeConfig); err != nil {
			return nil, fmt.Errorf("failed to unmarshal runtime config, error: %w\nruntimeConfig: %v", err, args.RuntimeConfig)
		}
	}
	if args.KubernetesConfig != "" {
		log.Infof("input kubernetesConfig:%s\n", prettyPrint(args.KubernetesConfig))
	}
	k8sExecCfg, err := parseExecConfigJSON(&args.KubernetesConfig)
	if err != nil {
		return nil, err
	}
	namespace, err = config.InPodNamespace()
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
		execution, pipeline, driverErr = driver.RootDAG(ctx, options, client)
		if driverErr != nil {
			return nil, driverErr
		}
		pipelineRoot = pipeline.GetPipelineRoot()
		storeSessionInfo = pipeline.GetStoreSessionInfo()
	case DAG:
		pipeline, driverErr = client.GetPipeline(ctx, options.PipelineName, options.RunID, "", "", "", "")
		if driverErr != nil {
			return nil, driverErr
		}
		pipelineRoot = pipeline.GetPipelineRoot()
		storeSessionInfo = pipeline.GetStoreSessionInfo()
		execution, driverErr = driver.DAG(ctx, pipeline, options, client)
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
		pipeline, driverErr = client.GetPipeline(ctx, options.PipelineName, options.RunID, "", "", "", "")
		if driverErr != nil {
			return nil, driverErr
		}
		pipelineRoot = pipeline.GetPipelineRoot()
		storeSessionInfo = pipeline.GetStoreSessionInfo()
		outputPathPrefix = uuid.NewString()
		execution, driverErr = driver.Container(ctx, pipeline, options, client, cacheClient, outputPathPrefix)
	default:
		err = fmt.Errorf("unknown driverType %s", args.Type)
	}
	if driverErr != nil {
		log.Errorf("driver execution failed with error: %v", driverErr)
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

func uploadDriverLogArtifact(ctx context.Context, logContext *driverLogArtifactContext) error {
	if logContext == nil {
		return fmt.Errorf("logContext is nil")
	}
	if logContext.PipelineRoot != "" {
		restConfig, err := util.GetKubernetesConfig()
		if err != nil {
			return fmt.Errorf("failed to get kubernetes config: %v", err)
		}
		k8sClient, err := kubernetes.NewForConfig(restConfig)
		if err != nil {
			return fmt.Errorf("failed to initialize kubernetes client set: %w", err)
		}
		session, err := objectstore.GetSessionInfoFromString(logContext.StoreSessionInfo)
		if err != nil {
			return fmt.Errorf("failed to get session info from store: %v", err)
		}
		bucketConfig, err := objectstore.ParseBucketConfig(logContext.PipelineRoot, session)
		if err != nil {
			return fmt.Errorf("failed to parse bucket config: %v", err)
		}
		bucket, err := objectstore.OpenBucket(ctx, k8sClient, logContext.Namespace, bucketConfig)
		if err != nil {
			return fmt.Errorf("failed to open bucket: %v", err)
		}
		key := fmt.Sprintf("driver/%s-logs", logContext.LogID)
		if logContext.Execution != nil && logContext.OutputPathPrefix != "" {
			key = fmt.Sprintf("%s/%s/driver-logs", logContext.Task, logContext.OutputPathPrefix)
		}
		glog.Infof("Uploading log key: %s ...", key)
		err = objectstore.UploadBlob(ctx, bucket, logContext.LocalPath, key)
		if err != nil {
			return fmt.Errorf("failed to upload log: %v", err)
		}
	}
	return nil
}

func validate(args driverapi.DriverPluginArgs) error {
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

func extractOutputParameters(execution *driver.Execution, driverType string) []driverapi.Parameter {
	if execution == nil {
		return []driverapi.Parameter{}
	}
	var outputs []driverapi.Parameter
	if execution.ID != 0 {
		outputs = append(outputs, driverapi.Parameter{
			Name:  "execution-id",
			Value: fmt.Sprint(execution.ID),
		})
	}
	if execution.IterationCount != nil {
		outputs = append(outputs, driverapi.Parameter{
			Name:  "iteration-count",
			Value: fmt.Sprint(*execution.IterationCount),
		})
	} else if driverType == RootDag || driverType == DAG {
		outputs = append(outputs, driverapi.Parameter{
			Name:  "iteration-count",
			Value: "0",
		})
	}
	if execution.Cached != nil {
		outputs = append(outputs, driverapi.Parameter{
			Name:  "cached-decision",
			Value: strconv.FormatBool(*execution.Cached),
		})
	}
	if execution.Condition != nil {
		outputs = append(outputs, driverapi.Parameter{
			Name:  "condition",
			Value: strconv.FormatBool(*execution.Condition),
		})
	} else if driverType == DAG || driverType == RootDag || driverType == CONTAINER {
		outputs = append(outputs, driverapi.Parameter{
			Name:  "condition",
			Value: "nil",
		})
	}
	if execution.PodSpecPatch != "" {
		outputs = append(outputs, driverapi.Parameter{
			Name:  "pod-spec-patch",
			Value: execution.PodSpecPatch,
		})
	}
	return outputs
}

func WriteJSONResponse(w http.ResponseWriter, payload driverapi.DriverResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}
