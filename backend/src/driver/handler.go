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

// Package driver implements an HTTP handler compatible with the Argo Workflows
// executor plugin protocol. It exposes the KFP driver logic as a centralized
// service endpoint within the KFP API server, eliminating the need for a
// per-workflow driver sidecar container.
package driver

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/driver/api"
	"github.com/kubeflow/pipelines/backend/src/v2/cacheutils"
	"github.com/kubeflow/pipelines/backend/src/v2/config"
	"github.com/kubeflow/pipelines/backend/src/v2/driver"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	"github.com/kubeflow/pipelines/backend/src/v2/objectstore"
	"github.com/kubeflow/pipelines/kubernetes_platform/go/kubernetesplatform"
	"google.golang.org/protobuf/encoding/protojson"
	"k8s.io/client-go/kubernetes"
)

const (
	unsetProxyArgValue = "unset"
	// RootDag denotes the root DAG driver type.
	RootDag = "ROOT_DAG"
	// DAG denotes the DAG driver type.
	DAG = "DAG"
	// CONTAINER denotes the container driver type.
	CONTAINER = "CONTAINER"
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

// ExecutePlugin handles POST /api/v1/template.execute requests from Argo Workflows.
// It is invoked by the Argo workflow controller via an Argo HTTP template. The
// request body is a flat JSON object of DriverPluginArgs; the response is a flat
// JSON object (DriverHTTPResponse) whose fields are extracted by the compiler as
// Argo output parameters using jsonpath(response.body, "$.FIELD") expressions.
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
	glog.Infof("Received driver execute request: %v", r)
	args, err := parseDriverRequestArgs(r)
	if err != nil {
		glog.Errorf("Failed to parse driver request args: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	execution, err := drive(*args, args.Namespace)
	if err != nil {
		glog.Errorf("unable to drive execution: %v", err)
		http.Error(w, fmt.Sprintf("driver execution failed: %v", err), http.StatusInternalServerError)
		return
	}
	if execution != nil && execution.ExecutorInput != nil {
		executorInputBytes, err := protojson.Marshal(execution.ExecutorInput)
		if err != nil {
			glog.Errorf("failed to marshal ExecutorInput: %v", err)
			http.Error(w, fmt.Sprintf("failed to marshal ExecutorInput: %v", err), http.StatusInternalServerError)
			return
		}
		glog.Infof("output ExecutorInput:%s\n", prettyPrint(string(executorInputBytes)))
	}
	WriteJSONResponse(w, buildHTTPResponse(execution, args.Type))
}

// parseDriverRequestArgs parses the Argo HTTP template request body, which is a
// flat JSON object of DriverPluginArgs. The namespace is read from the Namespace
// field, which the compiler populates with {{workflow.namespace}}.
//
// The compiler base64-encodes JSON-valued fields (component, task, container,
// kubernetes_config, runtime_config) to avoid Argo's HTTP body substitution
// breaking the enclosing JSON when those values contain quotes. This function
// decodes those fields transparently.
func parseDriverRequestArgs(r *http.Request) (*api.DriverPluginArgs, error) {
	var args api.DriverPluginArgs
	if err := json.NewDecoder(r.Body).Decode(&args); err != nil {
		return nil, fmt.Errorf("failed to parse driver request body: %v", err)
	}
	if err := decodeBase64Fields(&args); err != nil {
		return nil, fmt.Errorf("failed to decode base64 fields: %v", err)
	}
	if err := validate(args); err != nil {
		return nil, err
	}
	return &args, nil
}

// decodeBase64Fields decodes the base64-encoded JSON fields that the compiler
// wraps in {{=base64encode(...)}} expressions to survive Argo HTTP body
// template substitution without JSON-escaping issues.
func decodeBase64Fields(args *api.DriverPluginArgs) error {
	var err error
	if args.Component, err = decodeBase64Field(args.Component); err != nil {
		return fmt.Errorf("component: %w", err)
	}
	if args.Task, err = decodeBase64Field(args.Task); err != nil {
		return fmt.Errorf("task: %w", err)
	}
	if args.Container, err = decodeBase64Field(args.Container); err != nil {
		return fmt.Errorf("container: %w", err)
	}
	if args.KubernetesConfig, err = decodeBase64Field(args.KubernetesConfig); err != nil {
		return fmt.Errorf("kubernetes_config: %w", err)
	}
	if args.RuntimeConfig, err = decodeBase64Field(args.RuntimeConfig); err != nil {
		return fmt.Errorf("runtime_config: %w", err)
	}
	return nil
}

// decodeBase64Field base64-decodes a field value. Returns empty string for
// empty input without error (matching the behaviour of absent optional fields).
func decodeBase64Field(s string) (string, error) {
	if s == "" {
		return "", nil
	}
	decoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

// drive executes the KFP driver logic for the given args and workflow namespace.
//
// namespace is the Kubernetes namespace in which the workflow runs. When running
// centrally in the API server this is supplied from the Argo request body. When
// empty (e.g., in tests or legacy sidecar mode) the pod's own namespace is used
// as a fallback.
func drive(args api.DriverPluginArgs, namespace string) (execution *driver.Execution, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("KFP driver: %w", err)
		}
	}()
	var (
		pipelineRoot     string
		storeSessionInfo string
		outputPathPrefix string
	)
	var pipeline *metadata.Pipeline
	logID := fmt.Sprintf("%d-%v-%v-%v", time.Now().UnixMilli(), args.IterationIndex, args.Type, args.TaskName)
	// Use a temp file for driver logs; the file is uploaded to object storage then
	// deleted. In centralized mode there is no shared Argo volume to write to.
	logFile := filepath.Join(os.TempDir(), fmt.Sprintf("kfp-driver-%s.log", logID))
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
	// Support reading component spec from a file if value starts with @.
	// This bypasses exec() argument size limits for large workflows.
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

	// Resolve namespace: prefer the workflow namespace from the Argo request body
	// (populated in executor plugin address mode). Fall back to the pod's own namespace
	// for backward compatibility with sidecar mode or local testing.
	if namespace == "" {
		namespace, err = config.InPodNamespace()
		if err != nil {
			return nil, err
		}
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
			return nil, err
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

// buildHTTPResponse converts a driver Execution into the flat DriverHTTPResponse
// JSON that Argo extracts output parameters from via jsonpath expressions.
func buildHTTPResponse(execution *driver.Execution, driverType string) api.DriverHTTPResponse {
	resp := api.DriverHTTPResponse{}
	if execution == nil {
		return resp
	}
	if execution.ID != 0 {
		resp.ExecutionID = fmt.Sprint(execution.ID)
	}
	switch {
	case execution.IterationCount != nil:
		resp.IterationCount = fmt.Sprint(*execution.IterationCount)
	case driverType == RootDag:
		resp.IterationCount = "0"
	}
	if execution.Cached != nil {
		resp.CachedDecision = strconv.FormatBool(*execution.Cached)
	}
	if execution.Condition != nil {
		resp.Condition = strconv.FormatBool(*execution.Condition)
	}
	resp.PodSpecPatch = execution.PodSpecPatch
	return resp
}

// WriteJSONResponse writes a DriverHTTPResponse as JSON to the HTTP response writer.
func WriteJSONResponse(w http.ResponseWriter, payload api.DriverHTTPResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

// parseExecConfigJSON parses the Kubernetes executor configuration from a JSON string.
func parseExecConfigJSON(k8sExecConfigJSON *string) (*kubernetesplatform.KubernetesExecutorConfig, error) {
	var k8sExecCfg *kubernetesplatform.KubernetesExecutorConfig
	if *k8sExecConfigJSON != "" {
		k8sExecCfg = &kubernetesplatform.KubernetesExecutorConfig{}
		if err := util.UnmarshalString(*k8sExecConfigJSON, k8sExecCfg); err != nil {
			return nil, fmt.Errorf("failed to unmarshal Kubernetes config, error: %w\nKubernetesConfig: %v", err, k8sExecConfigJSON)
		}
	}
	return k8sExecCfg, nil
}

// prettyPrint formats a JSON string with indentation for readable logging output.
func prettyPrint(jsonStr string) string {
	var prettyJSON bytes.Buffer
	err := json.Indent(&prettyJSON, []byte(jsonStr), "", "  ")
	if err != nil {
		return jsonStr
	}
	return prettyJSON.String()
}

// newMlmdClient creates a new MLMD metadata client with optional TLS.
func newMlmdClient(mlmdServerAddress string, mlmdServerPort string, tlsCfg *tls.Config) (*metadata.Client, error) {
	return metadata.NewClient(mlmdServerAddress, mlmdServerPort, tlsCfg)
}
