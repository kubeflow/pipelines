// Copyright 2021-2023 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
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
	argoclient "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient/kfpapi"
	"github.com/kubeflow/pipelines/backend/src/v2/client_manager"
	"github.com/kubeflow/pipelines/backend/src/v2/compiler"
	drivercommon "github.com/kubeflow/pipelines/backend/src/v2/driver/common"
	"github.com/spf13/viper"
	"google.golang.org/protobuf/encoding/protojson"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/driver"
	"github.com/kubeflow/pipelines/kubernetes_platform/go/kubernetesplatform"

	_ "github.com/kubeflow/pipelines/backend/src/v2/common/plugins/all"
)

const (
	driverTypeArg                         = "type"
	httpProxyArg                          = "http_proxy"
	httpsProxyArg                         = "https_proxy"
	noProxyArg                            = "no_proxy"
	unsetProxyArgValue                    = "unset"
	ROOT_DAG                              = "ROOT_DAG" //nolint
	DAG                                   = "DAG"
	CONTAINER                             = "CONTAINER"
	pipelineJobCreateTimeUTCPlaceholder   = "{{$.pipeline_job_create_time_utc}}"
	pipelineJobScheduleTimeUTCPlaceholder = "{{$.pipeline_job_schedule_time_utc}}"
)

var (
	// inputs
	driverType        = flag.String(driverTypeArg, "", "task driver type, one of ROOT_DAG, DAG, CONTAINER")
	pipelineName      = flag.String("pipeline_name", "", "pipeline context name")
	runID             = flag.String("run_id", "", "pipeline run uid")
	runName           = flag.String("run_name", "", "pipeline run name (Kubernetes object name)")
	runDisplayName    = flag.String("run_display_name", "", "pipeline run display name")
	runtimeConfigJSON = flag.String("runtime_config", "", "jobruntime config")
	iterationIndex    = flag.Int("iteration_index", -1, "iteration index, -1 means not an interation")
	taskName          = flag.String("task_name", "", "original task name, used for proper input resolution in the container/dag driver")
	namespaceFlag     = flag.String("namespace", "", "Kubernetes namespace for runtime operations.")

	// container inputs
	parentTaskID      = flag.String("parent_task_id", "", "Parent PipelineTask ID")
	k8sExecConfigJson = flag.String("kubernetes_config", "{}", "kubernetes executor config")

	// config
	mlPipelineServerAddress = flag.String("ml_pipeline_server_address", "ml-pipeline", "The name of the ML pipeline API server address.")
	mlPipelineServerPort    = flag.String("ml_pipeline_server_port", "8887", "The port of the ML pipeline API server.")

	// output paths
	parentTaskIDPath   = flag.String("parent_task_id_path", "", "Parent Task ID output path")
	iterationCountPath = flag.String("iteration_count_path", "", "Iteration Count output path")
	podSpecPatchPath   = flag.String("pod_spec_patch_path", "", "Pod Spec Patch output path")
	// the value stored in the paths will be either 'true' or 'false'
	cachedDecisionPath = flag.String("cached_decision_path", "", "Cached Decision output path")
	conditionPath      = flag.String("condition_path", "", "Condition output path")
	logLevel           = flag.String("log_level", "1", "The verbosity level to log.")

	// proxy
	httpProxy            = flag.String(httpProxyArg, unsetProxyArgValue, "The proxy for HTTP connections.")
	httpsProxy           = flag.String(httpsProxyArg, unsetProxyArgValue, "The proxy for HTTPS connections.")
	noProxy              = flag.String(noProxyArg, unsetProxyArgValue, "Addresses that should ignore the proxy.")
	publishLogs          = flag.String("publish_logs", "true", "Whether to publish component logs to the object store")
	cacheDisabledFlag    = flag.Bool("cache_disabled", false, "Disable cache globally.")
	mlPipelineTLSEnabled = flag.Bool("ml_pipeline_tls_enabled", false, "Set to true if mlpipeline API server serves over TLS.")
	caCertPath           = flag.String("ca_cert_path", "", "The path to the CA certificate to trust on connections to the ML pipeline API server and metadata server.")
	defaultRunAsUser     = flag.Int64("default_run_as_user", -1, "Admin-configured default runAsUser for user containers. -1 means not set.")
	defaultRunAsGroup    = flag.Int64("default_run_as_group", -1, "Admin-configured default runAsGroup for user containers. -1 means not set.")
	defaultRunAsNonRoot  = flag.String("default_run_as_non_root", "", "Admin-configured default runAsNonRoot for user containers. Empty means not set.")
	defaultHostUsers     = flag.String("default_host_users", "", "Administrator-configured default hostUsers for user workload pods. Empty means not set. Set to false to run pods in a dedicated Linux user namespace.")
)

func main() {
	flag.Parse()
	initConfig()

	glog.Infof("Setting log level to: '%s'", *logLevel)
	err := flag.Set("v", *logLevel)
	if err != nil {
		glog.Warningf("Failed to set log level: %s", err.Error())
	}

	err = drive()
	if err != nil {
		glog.Exitf("Failed to execute driver: %v", err)
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
	if *httpProxy == unsetProxyArgValue {
		return fmt.Errorf("argument --%s is required but can be an empty value", httpProxyArg)
	}
	if *httpsProxy == unsetProxyArgValue {
		return fmt.Errorf("argument --%s is required but can be an empty value", httpsProxyArg)
	}
	if *noProxy == unsetProxyArgValue {
		return fmt.Errorf("argument --%s is required but can be an empty value", noProxyArg)
	}
	// validation responsibility lives in driver itself, so we do not validate all other args
	return nil
}

// getCurrentWorkflowMetadata returns metadata for the Argo Workflow backing the
// current run.
//
// The compiler can safely pass workflow creation time directly via
// {{workflow.creationTimestamp}}, but recurring-run schedule time is stored in
// the workflowEpoch label and that label is absent for ad hoc runs. Referencing
// the label directly from the compiled template causes Argo to reject manual
// runs before the driver starts, so the driver resolves the label at runtime
// from the Workflow object instead. The driver already receives --run_name as
// {{workflow.name}}, so it can read the Workflow directly without first looking
// up its own Pod.
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

// getPipelineJobTimePlaceholderUsage reports whether the current driver can
// resolve pipeline job time placeholders from task input runtime values.
//
// Only DAG and CONTAINER drivers call resolveInputs for the current task, and
// the resolver substitutes pipeline job time placeholders only when a task
// input parameter is a runtime-value constant matching the placeholder.
func getPipelineJobTimePlaceholderUsage(
	driverType string,
	taskSpec *pipelinespec.PipelineTaskSpec,
) pipelineJobTimePlaceholderUsage {
	usage := pipelineJobTimePlaceholderUsage{}
	if driverType == ROOT_DAG || taskSpec == nil {
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

// getWorkflowMetadataForPipelineJobTimes loads Workflow metadata only when the
// current driver still needs unresolved pipeline job time placeholders. If
// create time is already available and only schedule time still needs runtime
// metadata, lookup is best-effort so clusters without Workflow read RBAC still
// resolve schedule time by falling back to create time.
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

// resolvePipelineJobScheduleTimeUTCFromWorkflow returns the exact recurring-run
// schedule time when workflowEpoch is present and otherwise falls back to the
// workflow creation time for manual runs.
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

// resolvePipelineJobTimes normalizes the placeholder inputs into the UTC values
// consumed by driver.Options. Schedule time may come from the compiled flag
// when explicitly provided, or from workflow metadata when manual runs would
// otherwise have no workflowEpoch label to resolve.
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

func drive() (err error) {
	ctx := context.Background()

	// Initialize connection to the KFP API server
	clientManagerOptions := &client_manager.Options{
		MLPipelineTLSEnabled:    *mlPipelineTLSEnabled,
		CaCertPath:              *caCertPath,
		MLPipelineServerAddress: *mlPipelineServerAddress,
		MLPipelineServerPort:    *mlPipelineServerPort,
	}
	clientManager, err := client_manager.NewClientManager(clientManagerOptions)
	if err != nil {
		return err
	}
	glog.Infof("Initialized Client Manager.")

	if err = validate(); err != nil {
		return err
	}

	proxy.InitializeConfig(*httpProxy, *httpsProxy, *noProxy)
	var runtimeConfig *pipelinespec.PipelineJob_RuntimeConfig
	if *runtimeConfigJSON != "" {
		glog.Infof("input RuntimeConfig:%s\n", prettyPrint(*runtimeConfigJSON))
		runtimeConfig = &pipelinespec.PipelineJob_RuntimeConfig{}
		if err := util.UnmarshalString(*runtimeConfigJSON, runtimeConfig); err != nil {
			return fmt.Errorf("failed to unmarshal runtime config, error: %w\nruntimeConfig: %v", err, runtimeConfigJSON)
		}
	}

	k8sExecCfg, err := parseExecConfigJson(k8sExecConfigJson)
	if err != nil {
		return err
	}

	namespace, err := resolveNamespace(*namespaceFlag)
	if err != nil {
		return err
	}

	podName := os.Getenv("KFP_POD_NAME")
	podUID := os.Getenv("KFP_POD_UID")
	if podUID == "" || podName == "" {
		return fmt.Errorf("KFP_POD_UID and KFP_POD_NAME environment variables must be set")
	}

	if runID == nil {
		return fmt.Errorf("argument --run_id must be specified")
	}
	fullView := go_client.GetRunRequest_FULL
	run, err := clientManager.KFPAPIClient().GetRun(ctx, &go_client.GetRunRequest{RunId: *runID, View: &fullView})
	if err != nil {
		return err
	}

	var parentTask *go_client.PipelineTask
	if *parentTaskID != "" {
		parentTask, err = clientManager.KFPAPIClient().GetTask(ctx, &go_client.GetTaskRequest{
			TaskId: *parentTaskID,
			RunId:  *runID,
		})
		if err != nil {
			return err
		}
	}

	// The driver now resolves component and task specs from the run's pipeline spec
	// plus the current scope path. We still require an explicit task name here for
	// non-root executions so the scope path can be extended to the active task.
	var resolvedTaskName string
	if *driverType != ROOT_DAG {
		if *taskName != "" {
			resolvedTaskName = *taskName
		} else {
			return fmt.Errorf("task name for non Root dag could not be resolved")
		}
	}

	scopePath, err := buildScopePath(ctx, run, parentTask, resolvedTaskName, clientManager.KFPAPIClient())
	if err != nil || scopePath == nil {
		return fmt.Errorf("failed to build scope path: %w", err)
	}
	componentSpec, taskSpec, containerSpec, err := resolveDriverSpecs(scopePath, *driverType)
	if err != nil {
		return fmt.Errorf("failed to resolve specs from scope path: %w", err)
	}

	createTimeUTC := ""
	if createdAt := run.GetCreatedAt(); createdAt != nil {
		createTimeUTC = createdAt.AsTime().UTC().Format(time.RFC3339)
	}
	scheduleTimeEpochSeconds := ""
	if scheduledAt := run.GetScheduledAt(); scheduledAt != nil {
		scheduleTimeEpochSeconds = strconv.FormatInt(scheduledAt.AsTime().Unix(), 10)
	}

	placeholderUsage := getPipelineJobTimePlaceholderUsage(*driverType, taskSpec)
	workflowMeta, err := getWorkflowMetadataForPipelineJobTimes(
		ctx,
		namespace,
		*runName,
		placeholderUsage,
		createTimeUTC,
		scheduleTimeEpochSeconds,
		getCurrentWorkflowMetadata,
	)
	if err != nil {
		return err
	}
	resolvedPipelineJobCreateTimeUTC, resolvedPipelineJobScheduleTimeUTC, err := resolvePipelineJobTimes(
		createTimeUTC,
		scheduleTimeEpochSeconds,
		workflowMeta,
	)
	if err != nil {
		return err
	}
	options := drivercommon.Options{
		PipelineName:               *pipelineName,
		Run:                        run,
		RunName:                    *runName,
		RunDisplayName:             *runDisplayName,
		Namespace:                  namespace,
		Component:                  componentSpec,
		Task:                       taskSpec,
		IterationIndex:             *iterationIndex,
		PipelineLogLevel:           *logLevel,
		PublishLogs:                *publishLogs,
		CacheDisabled:              *cacheDisabledFlag,
		DriverType:                 *driverType,
		TaskName:                   resolvedTaskName,
		ParentTask:                 parentTask,
		PodName:                    podName,
		PodUID:                     podUID,
		ScopePath:                  *scopePath,
		MLPipelineServerAddress:    *mlPipelineServerAddress,
		MLPipelineServerPort:       *mlPipelineServerPort,
		MLPipelineTLSEnabled:       *mlPipelineTLSEnabled,
		PipelineJobCreateTimeUTC:   resolvedPipelineJobCreateTimeUTC,
		PipelineJobScheduleTimeUTC: resolvedPipelineJobScheduleTimeUTC,
		CaCertPath:                 *caCertPath,
	}
	var execution *driver.Execution
	switch *driverType {
	case ROOT_DAG:
		options.RuntimeConfig = runtimeConfig
		execution, err = driver.RootDAG(ctx, options, clientManager)
	case DAG:
		execution, err = driver.DAG(ctx, options, clientManager)
	case CONTAINER:
		options.Container = containerSpec
		options.KubernetesExecutorConfig = k8sExecCfg
		if *defaultRunAsUser >= 0 {
			options.DefaultRunAsUser = defaultRunAsUser
		}
		if *defaultRunAsGroup >= 0 {
			options.DefaultRunAsGroup = defaultRunAsGroup
		}
		if *defaultRunAsNonRoot != "" {
			v, err := strconv.ParseBool(*defaultRunAsNonRoot)
			if err == nil {
				options.DefaultRunAsNonRoot = &v
			}
		}
		if *defaultHostUsers != "" {
			if _, err := strconv.ParseBool(*defaultHostUsers); err != nil {
				return fmt.Errorf("invalid --default_host_users value %q: %w", *defaultHostUsers, err)
			}
		}
		execution, err = driver.Container(ctx, options, clientManager)
	default:
		err = fmt.Errorf("unknown driverType %s", *driverType)
	}
	if err != nil {
		return fmt.Errorf("failed to execute driver: %w", err)
	}
	if execution == nil {
		return fmt.Errorf("driver execution is nil")
	}

	executionPaths := &TaskPaths{
		TaskID:         *parentTaskIDPath,
		IterationCount: *iterationCountPath,
		CachedDecision: *cachedDecisionPath,
		Condition:      *conditionPath,
		PodSpecPatch:   *podSpecPatchPath,
	}

	return handleExecution(execution, *driverType, executionPaths)
}

func resolveNamespace(explicitNamespace string) (string, error) {
	if explicitNamespace == "" {
		return "", fmt.Errorf("argument --namespace must be specified")
	}
	return explicitNamespace, nil
}

func parseExecConfigJson(k8sExecConfigJson *string) (*kubernetesplatform.KubernetesExecutorConfig, error) {
	var k8sExecCfg *kubernetesplatform.KubernetesExecutorConfig
	if *k8sExecConfigJson != "" {
		glog.Infof("input kubernetesConfig:%s\n", prettyPrint(*k8sExecConfigJson))
		k8sExecCfg = &kubernetesplatform.KubernetesExecutorConfig{}
		if err := util.UnmarshalString(*k8sExecConfigJson, k8sExecCfg); err != nil {
			return nil, fmt.Errorf("failed to unmarshal Kubernetes config, error: %w\nKubernetesConfig: %v", err, k8sExecConfigJson)
		}
	}
	return k8sExecCfg, nil
}

func handleExecution(execution *driver.Execution, driverType string, executionPaths *TaskPaths) error {
	if execution.TaskID == "" {
		return fmt.Errorf("execution.TaskID is empty")
	}
	glog.Infof("output execution.ID=%v", execution.TaskID)
	if executionPaths.TaskID != "" {
		if err := writeFile(executionPaths.TaskID, []byte(fmt.Sprint(execution.TaskID))); err != nil {
			return fmt.Errorf("failed to write execution ID to file: %w", err)
		}
	}

	if execution.IterationCount != nil {
		if err := writeFile(executionPaths.IterationCount, []byte(fmt.Sprintf("%v", *execution.IterationCount))); err != nil {
			return fmt.Errorf("failed to write iteration count to file: %w", err)
		}
	} else {
		if driverType == ROOT_DAG || driverType == DAG {
			if err := writeFile(executionPaths.IterationCount, []byte("0")); err != nil {
				return fmt.Errorf("failed to write iteration count to file: %w", err)
			}
		}
	}
	if execution.Cached != nil {
		if err := writeFile(executionPaths.CachedDecision, []byte(strconv.FormatBool(*execution.Cached))); err != nil {
			return fmt.Errorf("failed to write cached decision to file: %w", err)
		}
	}
	if execution.Condition != nil {
		if err := writeFile(executionPaths.Condition, []byte(strconv.FormatBool(*execution.Condition))); err != nil {
			return fmt.Errorf("failed to write condition to file: %w", err)
		}
	} else {
		// nil is a valid value for Condition
		if driverType == ROOT_DAG || driverType == DAG || driverType == CONTAINER {
			if err := writeFile(executionPaths.Condition, []byte("nil")); err != nil {
				return fmt.Errorf("failed to write condition to file: %w", err)
			}
		}
	}
	if execution.PodSpecPatch != "" {
		glog.Infof("output podSpecPatch=\n%s\n", execution.PodSpecPatch)
		if executionPaths.PodSpecPatch == "" {
			return fmt.Errorf("--pod_spec_patch_path is required for container executor drivers")
		}
		if err := writeFile(executionPaths.PodSpecPatch, []byte(execution.PodSpecPatch)); err != nil {
			return fmt.Errorf("failed to write pod spec patch to file: %w", err)
		}
	}
	if execution.ExecutorInput != nil {
		executorInputBytes, err := protojson.Marshal(execution.ExecutorInput)
		if err != nil {
			return fmt.Errorf("failed to marshal ExecutorInput to JSON: %w", err)
		}
		executorInputJSON := string(executorInputBytes)
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
	return prettyJSON.String()
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
	return os.WriteFile(path, data, 0o644)
}

// buildScopePath builds a ScopePath from the run, parentTask and taskName.
func buildScopePath(
	ctx context.Context,
	run *go_client.Run,
	parentTask *go_client.PipelineTask,
	taskName string,
	kfpAPI kfpapi.API) (*util.ScopePath, error) {
	pipelineSpecStruct, err := kfpAPI.FetchPipelineSpecFromRun(ctx, run)
	if err != nil {
		return nil, err
	}
	var scopePath util.ScopePath
	if driverType == nil {
		return nil, fmt.Errorf("argument --%s must be specified", driverTypeArg)
	}
	if *driverType == ROOT_DAG {
		scopePath, err = util.NewScopePathFromStruct(pipelineSpecStruct)
		if err != nil {
			return nil, err
		}
		err = scopePath.Push("root")
		if err != nil {
			return nil, err
		}
	} else {
		if taskName == "" {
			return nil, fmt.Errorf("task name must be specified for non-root drivers")
		}
		scopePath, err = util.ScopePathFromStringPathWithNewTask(
			pipelineSpecStruct,
			parentTask.GetScopePath(),
			taskName,
		)
		if err != nil {
			return nil, err
		}
	}
	return &scopePath, nil
}

func resolveDriverSpecs(
	scopePath *util.ScopePath,
	driverType string,
) (*pipelinespec.ComponentSpec, *pipelinespec.PipelineTaskSpec, *pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec, error) {
	return resolveDriverSpecsFromScopePath(scopePath, driverType)
}

type specSourceUnavailableError struct {
	message string
}

func (e specSourceUnavailableError) Error() string {
	return e.message
}

func unavailableSpec(message string) error {
	return specSourceUnavailableError{message: message}
}

func resolveDriverSpecsFromScopePath(
	scopePath *util.ScopePath,
	driverType string,
) (*pipelinespec.ComponentSpec, *pipelinespec.PipelineTaskSpec, *pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec, error) {
	if scopePath == nil || scopePath.GetLast() == nil {
		return nil, nil, nil, unavailableSpec("scope path is empty")
	}

	componentSpec := scopePath.GetLast().GetComponentSpec()
	if componentSpec == nil {
		return nil, nil, nil, unavailableSpec("component spec not found")
	}

	var taskSpec *pipelinespec.PipelineTaskSpec
	if driverType != ROOT_DAG {
		taskSpec = scopePath.GetLast().GetTaskSpec()
		if taskSpec == nil {
			return nil, nil, nil, unavailableSpec("task spec not found")
		}
	}

	if err := validateDriverComponentKinds(driverType, componentSpec); err != nil {
		return nil, nil, nil, err
	}

	var containerSpec *pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec
	if driverType == CONTAINER {
		var err error
		containerSpec, err = loadContainerSpec(componentSpec, scopePath.GetPipelineSpec())
		if err != nil {
			return nil, nil, nil, err
		}
	}

	return componentSpec, taskSpec, containerSpec, nil
}

func validateDriverComponentKinds(driverType string, componentSpec *pipelinespec.ComponentSpec) error {
	switch driverType {
	case ROOT_DAG:
		if componentSpec.GetDag() == nil {
			return fmt.Errorf("root driver requires a DAG root component")
		}
	case DAG:
		if componentSpec.GetDag() == nil {
			return fmt.Errorf("dag driver requires a DAG component")
		}
	case CONTAINER:
		if componentSpec.GetExecutorLabel() == "" {
			return fmt.Errorf("container driver requires an executor-label component")
		}
	default:
		return fmt.Errorf("unknown driver type %q", driverType)
	}
	return nil
}

func loadContainerSpec(
	componentSpec *pipelinespec.ComponentSpec,
	pipelineSpec *pipelinespec.PipelineSpec,
) (*pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec, error) {
	if componentSpec == nil {
		return nil, unavailableSpec("component spec is nil")
	}
	if pipelineSpec == nil {
		return nil, unavailableSpec("pipeline spec is nil")
	}

	executorLabel := componentSpec.GetExecutorLabel()
	if executorLabel == "" {
		return nil, fmt.Errorf("component executor label is empty")
	}

	if pipelineSpec.GetDeploymentSpec() == nil {
		return nil, unavailableSpec("pipeline deployment spec is missing")
	}

	deploymentConfig, err := compiler.GetDeploymentConfig(pipelineSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal deployment spec: %w", err)
	}

	executor, ok := deploymentConfig.GetExecutors()[executorLabel]
	if !ok || executor == nil {
		return nil, unavailableSpec(fmt.Sprintf("container executor %q not found in deployment spec", executorLabel))
	}
	containerSpec := executor.GetContainer()
	if containerSpec == nil {
		return nil, fmt.Errorf("executor %q does not contain a container spec", executorLabel)
	}
	return containerSpec, nil
}

func initConfig() {
	viper.AutomaticEnv()
}
