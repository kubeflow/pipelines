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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	argoclient "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned"
	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/driver/driverapi"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient/kfpapi"
	"github.com/kubeflow/pipelines/backend/src/v2/client_manager"
	"github.com/kubeflow/pipelines/backend/src/v2/common/plugins"
	drivercommon "github.com/kubeflow/pipelines/backend/src/v2/driver/common"

	// Import plugin packages for side effects so their init() functions register factories.
	_ "github.com/kubeflow/pipelines/backend/src/v2/common/plugins/all"
	"github.com/kubeflow/pipelines/backend/src/v2/config"
	"github.com/kubeflow/pipelines/backend/src/v2/driver"
	"github.com/kubeflow/pipelines/backend/src/v2/objectstore"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	pipelineJobCreateTimeUTCPlaceholder   = "{{$.pipeline_job_create_time_utc}}"
	pipelineJobScheduleTimeUTCPlaceholder = "{{$.pipeline_job_schedule_time_utc}}"
	caCertPathEnvVar                      = "CA_CERT_PATH"
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
	RunID            string
	KFPAPI           kfpapi.API
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
	glog.Info("Received request to execute driver plugin")
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
		glog.Infof("output ExecutorInput: %d bytes", len(executorInputJSON))
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
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read driver request body: %v", err)
	}
	var body rawDriverRequest
	if err := json.Unmarshal(bodyBytes, &body); err != nil {
		return nil, fmt.Errorf("failed to parse driver request body: %v", err)
	}
	switch {
	case body.Template == nil:
		return nil, fmt.Errorf("driver request body.Template is empty")
	case body.Template.Plugin == nil:
		return nil, fmt.Errorf("driver request body.Template.Plugin is empty")
	case body.Template.Plugin.DriverPlugin == nil:
		return nil, fmt.Errorf("driver request body.Template.Plugin.DriverPlugin is empty")
	case len(body.Template.Plugin.DriverPlugin.Args) == 0 || string(body.Template.Plugin.DriverPlugin.Args) == "null":
		return nil, fmt.Errorf("driver request body.Template.Plugin.Args is empty")
	}
	var args driverapi.DriverPluginArgs
	if err := json.Unmarshal(body.Template.Plugin.DriverPlugin.Args, &args); err != nil {
		return nil, fmt.Errorf("failed to parse driver request args: %v", err)
	}
	var argFields map[string]json.RawMessage
	if err := json.Unmarshal(body.Template.Plugin.DriverPlugin.Args, &argFields); err != nil {
		return nil, fmt.Errorf("failed to parse driver request args as object: %v", err)
	}
	if err := validate(args, argFields); err != nil {
		return nil, err
	}
	return &args, nil
}

type rawDriverRequest struct {
	Template *rawDriverTemplate `json:"template"`
}

type rawDriverTemplate struct {
	Plugin *rawDriverPlugin `json:"plugin"`
}

type rawDriverPlugin struct {
	DriverPlugin *rawDriverPluginContainer `json:"driver-plugin"`
}

type rawDriverPluginContainer struct {
	Args json.RawMessage `json:"args"`
}

func getCurrentWorkflowMetadata(ctx context.Context, namespace string, workflowName string) (*metav1.ObjectMeta, error) {
	if workflowName == "" {
		return nil, fmt.Errorf("workflow name is empty")
	}
	restConfig, err := util.GetKubernetesConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize kubernetes config for workflow metadata: %w", err)
	}
	argoClient, err := argoclient.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize argo client for workflow metadata: %w", err)
	}
	workflow, err := argoClient.ArgoprojV1alpha1().Workflows(namespace).Get(ctx, workflowName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve workflow %q: %w", workflowName, err)
	}
	return &workflow.ObjectMeta, nil
}

type workflowMetadataGetter func(ctx context.Context, namespace string, workflowName string) (*metav1.ObjectMeta, error)

type pipelineJobTimePlaceholderUsage struct {
	needsCreateTime   bool
	needsScheduleTime bool
}

func getPipelineJobTimePlaceholderUsage(
	driverType string,
	taskSpec *pipelinespec.PipelineTaskSpec,
) pipelineJobTimePlaceholderUsage {
	usage := pipelineJobTimePlaceholderUsage{}
	if driverType == RootDag || taskSpec == nil {
		return usage
	}
	for _, inputParamSpec := range taskSpec.GetInputs().GetParameters() {
		runtimeValue := inputParamSpec.GetRuntimeValue()
		if runtimeValue == nil {
			continue
		}
		constant := runtimeValue.GetConstant()
		if constant == nil {
			continue
		}
		switch constant.GetStringValue() {
		case pipelineJobCreateTimeUTCPlaceholder:
			usage.needsCreateTime = true
		case pipelineJobScheduleTimeUTCPlaceholder:
			usage.needsScheduleTime = true
		}
		if usage.needsCreateTime && usage.needsScheduleTime {
			return usage
		}
	}
	return usage
}

func getWorkflowMetadataForPipelineJobTimes(
	ctx context.Context,
	namespace string,
	workflowName string,
	placeholderUsage pipelineJobTimePlaceholderUsage,
	createTimeUTC string,
	scheduleTimeEpochSeconds string,
	getMetadata workflowMetadataGetter,
) (*metav1.ObjectMeta, error) {
	needsCreateTimeMetadata := placeholderUsage.needsCreateTime && createTimeUTC == ""
	needsScheduleTimeMetadata := placeholderUsage.needsScheduleTime && scheduleTimeEpochSeconds == ""
	if !needsCreateTimeMetadata && !needsScheduleTimeMetadata {
		return nil, nil
	}
	workflowMeta, err := getMetadata(ctx, namespace, workflowName)
	if err != nil {
		if !needsCreateTimeMetadata && needsScheduleTimeMetadata && createTimeUTC != "" {
			glog.Warningf(
				"Failed to retrieve workflow metadata for pipeline job schedule time for workflow %q, falling back to create time: %v",
				workflowName,
				err,
			)
			return nil, nil
		}
		return nil, err
	}
	return workflowMeta, nil
}

func resolvePipelineJobScheduleTimeUTCFromWorkflow(
	workflowMeta *metav1.ObjectMeta,
	fallbackCreateTimeUTC string,
) string {
	if workflowMeta == nil {
		return fallbackCreateTimeUTC
	}
	createTimeUTC := fallbackCreateTimeUTC
	if createTimeUTC == "" {
		createTimeUTC = workflowMeta.CreationTimestamp.Time.UTC().Format(time.RFC3339)
	}
	value, ok := workflowMeta.Labels[util.LabelKeyWorkflowEpoch]
	if !ok {
		return createTimeUTC
	}
	scheduledEpochSeconds, err := util.RetrieveInt64FromLabel(value)
	if err != nil {
		return createTimeUTC
	}
	return time.Unix(scheduledEpochSeconds, 0).UTC().Format(time.RFC3339)
}

func resolvePipelineJobTimes(
	createTimeUTC string,
	scheduleTimeEpochSeconds string,
	workflowMeta *metav1.ObjectMeta,
) (string, string, error) {
	if createTimeUTC == "" && workflowMeta != nil {
		createTimeUTC = workflowMeta.CreationTimestamp.Time.UTC().Format(time.RFC3339)
	}
	if scheduleTimeEpochSeconds == "" {
		return createTimeUTC, resolvePipelineJobScheduleTimeUTCFromWorkflow(workflowMeta, createTimeUTC), nil
	}
	scheduleTimeEpoch, err := strconv.ParseInt(scheduleTimeEpochSeconds, 10, 64)
	if err != nil {
		return "", "", fmt.Errorf("invalid pipeline job schedule time epoch seconds %q: %w", scheduleTimeEpochSeconds, err)
	}
	return createTimeUTC, time.Unix(scheduleTimeEpoch, 0).UTC().Format(time.RFC3339), nil
}

func drive(args driverapi.DriverPluginArgs) (execution *driver.Execution, err error) {
	var clientManager *client_manager.ClientManager
	defer func() {
		if clientManager != nil {
			_ = clientManager.Close()
		}
	}()
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
	logID := uuid.NewString()
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
				RunID:            args.RunID,
				KFPAPI:           clientManager.KFPAPIClient(),
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

	log.Infof("driver invocation: type=%s run_id=%s task_name=%s", args.Type, args.RunID, args.TaskName)
	namespace, err = resolveNamespace(args.Namespace)
	if err != nil {
		return nil, err
	}
	iterationIndex, err := strconv.Atoi(args.IterationIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to parse iteration index: %w", err)
	}

	caCertPath := os.Getenv(caCertPathEnvVar)
	clientManager, pod, err := newDriverClientManager(ctx, args, caCertPath)
	if err != nil {
		return nil, err
	}
	podName, podUID := pod.Name, string(pod.UID)

	var runtimeConfig *pipelinespec.PipelineJob_RuntimeConfig
	if args.RuntimeConfig != "" {
		runtimeConfig = &pipelinespec.PipelineJob_RuntimeConfig{}
		if err := util.UnmarshalString(args.RuntimeConfig, runtimeConfig); err != nil {
			return nil, fmt.Errorf("failed to unmarshal runtime config")
		}
	}
	k8sExecCfg, err := parseExecConfigJSON(&args.KubernetesConfig)
	if err != nil {
		return nil, err
	}

	fullView := go_client.GetRunRequest_FULL
	run, err := clientManager.KFPAPIClient().GetRun(ctx, &go_client.GetRunRequest{RunId: args.RunID, View: &fullView})
	if err != nil {
		return nil, err
	}
	var parentTask *go_client.PipelineTask
	if args.ParentTaskID != "" {
		parentTask, err = clientManager.KFPAPIClient().GetTask(ctx, &go_client.GetTaskRequest{TaskId: args.ParentTaskID, RunId: args.RunID})
		if err != nil {
			return nil, err
		}
	}
	scopePath, err := buildScopePath(ctx, run, parentTask, args.TaskName, args.Type, clientManager.KFPAPIClient())
	if err != nil {
		return nil, fmt.Errorf("failed to build scope path: %w", err)
	}
	componentSpec, taskSpec, containerSpec, err := resolveDriverSpecs(scopePath, args.Type)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve specs from scope path: %w", err)
	}

	createTimeUTC := ""
	if createdAt := run.GetCreatedAt(); createdAt != nil {
		createTimeUTC = createdAt.AsTime().UTC().Format(time.RFC3339)
	}
	scheduleTimeEpochSeconds := ""
	if scheduledAt := run.GetScheduledAt(); scheduledAt != nil {
		scheduleTimeEpochSeconds = strconv.FormatInt(scheduledAt.AsTime().Unix(), 10)
	}
	placeholderUsage := getPipelineJobTimePlaceholderUsage(args.Type, taskSpec)
	workflowMeta, err := getWorkflowMetadataForPipelineJobTimes(ctx, namespace, args.RunName,
		placeholderUsage, createTimeUTC, scheduleTimeEpochSeconds, getCurrentWorkflowMetadata)
	if err != nil {
		return nil, err
	}
	createTime, scheduleTime, err := resolvePipelineJobTimes(createTimeUTC, scheduleTimeEpochSeconds, workflowMeta)
	if err != nil {
		return nil, err
	}
	pluginDispatcher, err := plugins.GetPluginDispatcherWithRuntimeArgs(args.RuntimeArgs)
	if err != nil {
		log.Errorf("Failed to initialize plugin dispatcher: %v", err)
		pluginDispatcher = plugins.NoOpDispatcher{}
	}
	options := drivercommon.Options{
		PipelineName: args.PipelineName, Run: run, RunName: args.RunName, RunDisplayName: args.RunDisplayName,
		Namespace: namespace, Component: componentSpec, Task: taskSpec, ParentTask: parentTask, ScopePath: *scopePath,
		IterationIndex: iterationIndex, PipelineLogLevel: args.LogLevel, PublishLogs: args.PublishLogs,
		CacheDisabled: args.CacheDisabledFlag, DriverType: args.Type, TaskName: args.TaskName,
		PodName: podName, PodUID: podUID,
		MLPipelineServerAddress: args.MlPipelineServerAddress, MLPipelineServerPort: args.MlPipelineServerPort,
		MLPipelineTLSEnabled: args.MlPipelineTLSEnabled, CaCertPath: caCertPath,
		PipelineJobCreateTimeUTC: createTime, PipelineJobScheduleTimeUTC: scheduleTime,
		PluginDispatcher: pluginDispatcher, ProxyConfig: proxy.NewConfig(args.HTTPProxy, args.HTTPSProxy, args.NoProxy),
	}

	// Resolve the same root/session as the native runtime for the log upload.
	// Log storage failures remain nonfatal to task execution.
	pipelineRoot, err = config.GetPipelineRootWithPipelineRunContext(ctx, args.PipelineName, namespace, clientManager.K8sClient(), run)
	if err == nil {
		var launcherConfig *config.Config
		launcherConfig, err = config.LoadLauncherConfig(ctx, clientManager.K8sClient(), namespace)
		if err == nil {
			var session objectstore.SessionInfo
			session, err = launcherConfig.GetStoreSessionInfo(pipelineRoot)
			if err == nil {
				var sessionJSON []byte
				sessionJSON, err = json.Marshal(session)
				storeSessionInfo = string(sessionJSON)
			}
		}
	}
	if err != nil {
		log.Errorf("Failed to initialize driver log storage: %v", err)
		pipelineRoot = ""
	}

	switch args.Type {
	case RootDag:
		options.RuntimeConfig = runtimeConfig
		execution, err = driver.RootDAG(ctx, options, clientManager)
	case DAG:
		execution, err = driver.DAG(ctx, options, clientManager)
	case CONTAINER:
		options.Container = containerSpec
		options.KubernetesExecutorConfig = k8sExecCfg
		options.DefaultRunAsUser = args.DefaultRunAsUser
		options.DefaultRunAsGroup = args.DefaultRunAsGroup
		options.DefaultRunAsNonRoot, err = parseOptionalBoolFlag("default_run_as_non_root", args.DefaultRunAsNonRoot)
		if err != nil {
			return nil, err
		}
		options.DefaultHostUsers, err = parseOptionalBoolFlag("default_host_users", args.DefaultHostUsers)
		if err != nil {
			return nil, err
		}
		outputPathPrefix = uuid.NewString()
		options.OutputPathPrefix = outputPathPrefix
		execution, err = driver.Container(ctx, options, clientManager)
	default:
		err = fmt.Errorf("unknown driverType %s", args.Type)
	}
	if err != nil {
		log.Errorf("driver execution failed: %v", err)
	}
	return execution, err
}

// apiClientConfig builds settings for this invocation without changing process env.
func apiClientConfig(args driverapi.DriverPluginArgs) *apiclient.Config {
	cfg := apiclient.FromEnvWithEndpointOverride(args.MlPipelineServerAddress, args.MlPipelineServerPort)
	if args.MlPipelineGRPCBackoffBaseDelay != "" {
		cfg.BackoffBaseDelay = args.MlPipelineGRPCBackoffBaseDelay
	}
	if args.MlPipelineGRPCBackoffMultiplier != "" {
		cfg.BackoffMultiplier = args.MlPipelineGRPCBackoffMultiplier
	}
	if args.MlPipelineGRPCBackoffJitter != "" {
		cfg.BackoffJitter = args.MlPipelineGRPCBackoffJitter
	}
	if args.MlPipelineGRPCBackoffMaxDelay != "" {
		cfg.BackoffMaxDelay = args.MlPipelineGRPCBackoffMaxDelay
	}
	if args.MlPipelineGRPCMinConnectTimeout != "" {
		cfg.MinConnectTimeout = args.MlPipelineGRPCMinConnectTimeout
	}
	return cfg
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
		var session objectstore.SessionInfo
		if err := json.Unmarshal([]byte(logContext.StoreSessionInfo), &session); err != nil {
			return fmt.Errorf("failed to get session info from store: %v", err)
		}
		bucketConfig, err := objectstore.ParseBucketPathToConfig(logContext.PipelineRoot)
		if err != nil {
			return fmt.Errorf("failed to parse bucket config: %v", err)
		}
		bucket, err := objectstore.OpenBucket(ctx, k8sClient, logContext.Namespace, bucketConfig, &session)
		if err != nil {
			return fmt.Errorf("failed to open bucket: %v", err)
		}
		defer bucket.Close()
		key := fmt.Sprintf("driver/%s-logs", logContext.LogID)
		if logContext.Execution != nil && logContext.OutputPathPrefix != "" {
			key = fmt.Sprintf("%s/%s/driver-logs", logContext.Task, logContext.OutputPathPrefix)
		}
		glog.Infof("Uploading log key: %s ...", key)
		err = objectstore.UploadBlob(ctx, bucket, logContext.LocalPath, key)
		if err != nil {
			return fmt.Errorf("failed to upload log: %v", err)
		}
		if logContext.Execution != nil && logContext.Execution.TaskID != "" && logContext.KFPAPI != nil {
			uri := util.GenerateOutputURI(logContext.PipelineRoot, []string{key}, false)
			return registerDriverLog(ctx, logContext.KFPAPI, logContext.RunID, logContext.Execution.TaskID, uri, logContext.StoreSessionInfo)
		}
	}
	return nil
}

// registerDriverLog preserves plugin properties and status messages while adding
// the uploaded log location. Only status metadata is updated, never task state.
func registerDriverLog(ctx context.Context, api kfpapi.API, runID, taskID, uri, session string) error {
	task, err := api.GetTask(ctx, &go_client.GetTaskRequest{RunId: runID, TaskId: taskID})
	if err != nil {
		return fmt.Errorf("failed to read task for driver log: %w", err)
	}
	metadata := &go_client.PipelineTask_StatusMetadata{}
	if task.GetStatusMetadata() != nil {
		metadata = proto.Clone(task.GetStatusMetadata()).(*go_client.PipelineTask_StatusMetadata)
	}
	if metadata.CustomProperties == nil {
		metadata.CustomProperties = map[string]*structpb.Value{}
	}
	metadata.CustomProperties["driver_logs_uri"] = structpb.NewStringValue(uri)
	metadata.CustomProperties["store_session_info"] = structpb.NewStringValue(session)
	_, err = api.UpdateTask(ctx, &go_client.UpdateTaskRequest{
		RunId: runID, TaskId: taskID,
		Task: &go_client.PipelineTask{RunId: runID, TaskId: taskID, StatusMetadata: metadata},
	})
	if err != nil {
		return fmt.Errorf("failed to register driver log: %w", err)
	}
	return nil
}

var commonRequiredDriverArgFields = []string{
	"type", "pipeline_name", "run_id", "run_name", "run_display_name",
	"parent_task_id", "task_name", "namespace", "iteration_index",
	"ml_pipeline_server_address", "ml_pipeline_server_port", "log_level", "publish_logs",
	"cache_disabled", "ml_pipeline_tls_enabled", "http_proxy", "https_proxy", "no_proxy",
}

func requiredDriverArgFields(driverType string) ([]string, error) {
	required := append([]string{}, commonRequiredDriverArgFields...)
	switch driverType {
	case RootDag:
		required = append(required, "runtime_config")
	case DAG:
	case CONTAINER:
		required = append(required, "kubernetes_config")
	default:
		return nil, fmt.Errorf("unknown driver type %q, must be one of %s, %s, %s", driverType, RootDag, DAG, CONTAINER)
	}
	return required, nil
}

func validate(args driverapi.DriverPluginArgs, argFields map[string]json.RawMessage) error {
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
	required, err := requiredDriverArgFields(args.Type)
	if err != nil {
		return err
	}
	for _, name := range required {
		if _, ok := argFields[name]; !ok {
			return fmt.Errorf("--%s is required for %s but was not provided", name, args.Type)
		}
	}
	return nil
}

func podSpecPatchLogMessage(podSpecPatch string) string {
	return fmt.Sprintf("output podSpecPatch: %d bytes", len(podSpecPatch))
}

func kubernetesConfigLogMessage(kubernetesConfig string) string {
	return fmt.Sprintf("input kubernetesConfig: %d bytes", len(kubernetesConfig))
}

func extractOutputParameters(execution *driver.Execution, driverType string) []driverapi.Parameter {
	if execution == nil {
		return []driverapi.Parameter{}
	}
	var outputs []driverapi.Parameter
	if execution.TaskID != "" {
		outputs = append(outputs, driverapi.Parameter{
			Name:  "task-id",
			Value: execution.TaskID,
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
		// nil is a valid value for Condition
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
	} else {
		outputs = append(outputs, driverapi.Parameter{
			Name:  "pod-spec-patch",
			Value: "",
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
