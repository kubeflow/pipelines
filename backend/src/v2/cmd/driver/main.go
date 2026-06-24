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
	"time"

	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient/kfpapi"
	"github.com/kubeflow/pipelines/backend/src/v2/client_manager"
	"github.com/kubeflow/pipelines/backend/src/v2/driver/common"
	"google.golang.org/protobuf/encoding/protojson"

	"os"
	"path/filepath"
	"strconv"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/driver"
	"github.com/kubeflow/pipelines/kubernetes_platform/go/kubernetesplatform"
)

const (
	driverTypeArg      = "type"
	httpProxyArg       = "http_proxy"
	httpsProxyArg      = "https_proxy"
	noProxyArg         = "no_proxy"
	unsetProxyArgValue = "unset"
	ROOT_DAG           = "ROOT_DAG"
	DAG                = "DAG"
	CONTAINER          = "CONTAINER"
)

var (
	// inputs
	driverType        = flag.String(driverTypeArg, "", "task driver type, one of ROOT_DAG, DAG, CONTAINER")
	pipelineName      = flag.String("pipeline_name", "", "pipeline context name")
	runID             = flag.String("run_id", "", "pipeline run uid")
	runName           = flag.String("run_name", "", "pipeline run name (Kubernetes object name)")
	runDisplayName    = flag.String("run_display_name", "", "pipeline run display name")
	componentSpecJson = flag.String("component", "", "legacy component spec override")
	taskSpecJson      = flag.String("task", "", "task spec")
	runtimeConfigJson = flag.String("runtime_config", "", "jobruntime config")
	iterationIndex    = flag.Int("iteration_index", -1, "iteration index, -1 means not an interation")
	taskName          = flag.String("task_name", "", "original task name, used for proper input resolution in the container/dag driver")

	// container inputs
	parentTaskID       = flag.String("parent_task_id", "", "Parent PipelineTask ID")
	legacyParentTaskID = flag.String("dag_execution_id", "", "Legacy alias for parent task id")
	containerSpecJson  = flag.String("container", "", "legacy container spec override")
	k8sExecConfigJson  = flag.String("kubernetes_config", "{}", "kubernetes executor config")

	// config
	mlPipelineServerAddress = flag.String("ml_pipeline_server_address", "ml-pipeline", "The name of the ML pipeline API server address.")
	mlPipelineServerPort    = flag.String("ml_pipeline_server_port", "8887", "The port of the ML pipeline API server.")

	// output paths
	parentTaskIDPath       = flag.String("parent_task_id_path", "", "Parent Task ID output path")
	legacyParentTaskIDPath = flag.String("execution_id_path", "", "Legacy alias for parent task id output path")
	iterationCountPath     = flag.String("iteration_count_path", "", "Iteration Count output path")
	podSpecPatchPath       = flag.String("pod_spec_patch_path", "", "Pod Spec Patch output path")
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
)

func main() {
	flag.Parse()

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
	if *runtimeConfigJson != "" {
		glog.Infof("input RuntimeConfig:%s\n", prettyPrint(*runtimeConfigJson))
		runtimeConfig = &pipelinespec.PipelineJob_RuntimeConfig{}
		if err := util.UnmarshalString(*runtimeConfigJson, runtimeConfig); err != nil {
			return fmt.Errorf("failed to unmarshal runtime config, error: %w\nruntimeConfig: %v", err, runtimeConfigJson)
		}
	}

	k8sExecCfg, err := parseExecConfigJson(k8sExecConfigJson)
	if err != nil {
		return err
	}

	namespace, err := resolveNamespace()
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
	resolvedParentTaskID := resolveStringFlag(parentTaskID, legacyParentTaskID)
	if resolvedParentTaskID != "" {
		parentTask, err = clientManager.KFPAPIClient().GetTask(ctx, &go_client.GetTaskRequest{
			TaskId: resolvedParentTaskID,
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
	componentSpec, taskSpec, containerSpec, err := resolveDriverSpecs(
		scopePath,
		*driverType,
		*componentSpecJson,
		*taskSpecJson,
		*containerSpecJson,
	)
	if err != nil {
		return fmt.Errorf("failed to resolve specs from scope path: %w", err)
	}

	options := common.Options{
		PipelineName:            *pipelineName,
		Run:                     run,
		RunName:                 *runName,
		RunDisplayName:          *runDisplayName,
		Namespace:               namespace,
		Component:               componentSpec,
		Task:                    taskSpec,
		IterationIndex:          *iterationIndex,
		PipelineLogLevel:        *logLevel,
		PublishLogs:             *publishLogs,
		CacheDisabled:           *cacheDisabledFlag,
		DriverType:              *driverType,
		TaskName:                resolvedTaskName,
		ParentTask:              parentTask,
		PodName:                 podName,
		PodUID:                  podUID,
		ScopePath:               *scopePath,
		MLPipelineServerAddress: *mlPipelineServerAddress,
		MLPipelineServerPort:    *mlPipelineServerPort,
		MLPipelineTLSEnabled:    *mlPipelineTLSEnabled,
		CaCertPath:              *caCertPath,
	}
	if createdAt := run.GetCreatedAt(); createdAt != nil {
		options.PipelineJobCreateTimeUTC = createdAt.AsTime().UTC().Format(time.RFC3339)
	}
	if scheduledAt := run.GetScheduledAt(); scheduledAt != nil {
		options.PipelineJobScheduleTimeUTC = scheduledAt.AsTime().UTC().Format(time.RFC3339)
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
		TaskID:         resolveStringFlag(parentTaskIDPath, legacyParentTaskIDPath),
		IterationCount: *iterationCountPath,
		CachedDecision: *cachedDecisionPath,
		Condition:      *conditionPath,
		PodSpecPatch:   *podSpecPatchPath,
	}

	return handleExecution(execution, *driverType, executionPaths)
}

func resolveNamespace() (string, error) {
	if namespace := os.Getenv("NAMESPACE"); namespace != "" {
		return namespace, nil
	}
	if namespace := os.Getenv("POD_NAMESPACE"); namespace != "" {
		return namespace, nil
	}
	const serviceAccountNamespacePath = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
	namespaceBytes, err := os.ReadFile(serviceAccountNamespacePath)
	if err != nil {
		return "", fmt.Errorf("NAMESPACE environment variable must be set")
	}
	return string(bytes.TrimSpace(namespaceBytes)), nil
}

func resolveStringFlag(primary, legacy *string) string {
	if primary != nil && *primary != "" {
		return *primary
	}
	if legacy != nil {
		return *legacy
	}
	return ""
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
	rawComponentSpec string,
	rawTaskSpec string,
	rawContainerSpec string,
) (*pipelinespec.ComponentSpec, *pipelinespec.PipelineTaskSpec, *pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec, error) {
	if scopePath == nil || scopePath.GetLast() == nil {
		return nil, nil, nil, fmt.Errorf("scope path is empty")
	}

	componentSpec := scopePath.GetLast().GetComponentSpec()
	if componentSpec == nil && rawComponentSpec != "" {
		componentSpec = &pipelinespec.ComponentSpec{}
		if err := util.UnmarshalString(rawComponentSpec, componentSpec); err != nil {
			return nil, nil, nil, fmt.Errorf("failed to unmarshal legacy component spec: %w", err)
		}
	}
	if componentSpec == nil {
		return nil, nil, nil, fmt.Errorf("component spec not found")
	}

	var taskSpec *pipelinespec.PipelineTaskSpec
	if driverType != ROOT_DAG {
		taskSpec = scopePath.GetLast().GetTaskSpec()
		if taskSpec == nil && rawTaskSpec != "" {
			taskSpec = &pipelinespec.PipelineTaskSpec{}
			if err := util.UnmarshalString(rawTaskSpec, taskSpec); err != nil {
				return nil, nil, nil, fmt.Errorf("failed to unmarshal legacy task spec: %w", err)
			}
		}
		if taskSpec == nil {
			return nil, nil, nil, fmt.Errorf("task spec not found")
		}
	}

	var containerSpec *pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec
	if driverType == CONTAINER {
		var err error
		containerSpec, err = loadContainerSpec(componentSpec, scopePath.GetPipelineSpec())
		if err != nil && rawContainerSpec != "" {
			containerSpec = &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{}
			if unmarshalErr := util.UnmarshalString(rawContainerSpec, containerSpec); unmarshalErr != nil {
				return nil, nil, nil, fmt.Errorf("failed to resolve container spec: %w", err)
			}
			err = nil
		}
		if err != nil {
			return nil, nil, nil, err
		}
	}

	return componentSpec, taskSpec, containerSpec, nil
}

func loadContainerSpec(
	componentSpec *pipelinespec.ComponentSpec,
	pipelineSpec *pipelinespec.PipelineSpec,
) (*pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec, error) {
	if componentSpec == nil {
		return nil, fmt.Errorf("component spec is nil")
	}
	if pipelineSpec == nil {
		return nil, fmt.Errorf("pipeline spec is nil")
	}

	executorLabel := componentSpec.GetExecutorLabel()
	if executorLabel == "" {
		return nil, fmt.Errorf("component executor label is empty")
	}

	deploymentConfig := &pipelinespec.PipelineDeploymentConfig{}
	deploymentSpecJSON, err := protojson.Marshal(pipelineSpec.GetDeploymentSpec())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal deployment spec: %w", err)
	}
	unmarshaler := protojson.UnmarshalOptions{DiscardUnknown: true}
	if err := unmarshaler.Unmarshal(deploymentSpecJSON, deploymentConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal deployment spec: %w", err)
	}

	executor, ok := deploymentConfig.GetExecutors()[executorLabel]
	if !ok || executor == nil {
		return nil, fmt.Errorf("container executor %q not found in deployment spec", executorLabel)
	}
	containerSpec := executor.GetContainer()
	if containerSpec == nil {
		return nil, fmt.Errorf("executor %q does not contain a container spec", executorLabel)
	}
	return containerSpec, nil
}
