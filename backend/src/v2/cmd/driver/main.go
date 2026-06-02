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
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"time"

	argoclient "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	workflowcommon "github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/spf13/viper"
	"google.golang.org/protobuf/encoding/protojson"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/common/plugins"

	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/cacheutils"
	"github.com/kubeflow/pipelines/backend/src/v2/config"
	"github.com/kubeflow/pipelines/backend/src/v2/driver"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	"github.com/kubeflow/pipelines/kubernetes_platform/go/kubernetesplatform"

	_ "github.com/kubeflow/pipelines/backend/src/v2/common/plugins/all"
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
	driverType                      = flag.String(driverTypeArg, "", "task driver type, one of ROOT_DAG, DAG, CONTAINER")
	pipelineName                    = flag.String("pipeline_name", "", "pipeline context name")
	runID                           = flag.String("run_id", "", "pipeline run uid")
	runName                         = flag.String("run_name", "", "pipeline run name (Kubernetes object name)")
	runDisplayName                  = flag.String("run_display_name", "", "pipeline run display name")
	pipelineJobCreateTimeUTCArg     = flag.String("pipeline_job_create_time_utc", "", "pipeline job creation time in UTC")
	pipelineJobScheduleTimeEpochArg = flag.String("pipeline_job_schedule_time_epoch_seconds", "", "pipeline job scheduled time as Unix epoch seconds")
	componentSpecJSON               = flag.String("component", "{}", "component spec")
	taskSpecJSON                    = flag.String("task", "", "task spec")
	runtimeConfigJSON               = flag.String("runtime_config", "", "jobruntime config")
	iterationIndex                  = flag.Int("iteration_index", -1, "iteration index, -1 means not an interation")
	taskName                        = flag.String("task_name", "", "original task name, used for proper input resolution in the container/dag driver")

	// container inputs
	dagExecutionID    = flag.Int64("dag_execution_id", 0, "DAG execution ID")
	containerSpecJson = flag.String("container", "{}", "container spec")
	k8sExecConfigJson = flag.String("kubernetes_config", "{}", "kubernetes executor config")

	// config
	mlPipelineServerAddress = flag.String("ml_pipeline_server_address", "ml-pipeline", "The name of the ML pipeline API server address.")
	mlPipelineServerPort    = flag.String("ml_pipeline_server_port", "8887", "The port of the ML pipeline API server.")
	mlmdServerAddress       = flag.String("mlmd_server_address", "", "MLMD server address")
	mlmdServerPort          = flag.String("mlmd_server_port", "", "MLMD server port")

	// output paths
	executionIDPath    = flag.String("execution_id_path", "", "Exeucution ID output path")
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
	metadataTLSEnabled   = flag.Bool("metadata_tls_enabled", false, "Set to true if MLMD serves over TLS.")
	caCertPath           = flag.String("ca_cert_path", "", "The path to the CA certificate to trust on connections to the ML pipeline API server and metadata server.")
	defaultRunAsUser     = flag.Int64("default_run_as_user", -1, "Admin-configured default runAsUser for user containers. -1 means not set.")
	defaultRunAsGroup    = flag.Int64("default_run_as_group", -1, "Admin-configured default runAsGroup for user containers. -1 means not set.")
	defaultRunAsNonRoot  = flag.String("default_run_as_non_root", "", "Admin-configured default runAsNonRoot for user containers. Empty means not set.")
	defaultHostUsers     = flag.String("default_host_users", "", "Administrator-configured default hostUsers for user workload pods. Empty means not set. Set to false to run pods in a dedicated Linux user namespace.")
)

// func RootDAG(pipelineName string, runID string, component *pipelinespec.ComponentSpec, task *pipelinespec.PipelineTaskSpec, mlmd *metadata.Client) (*Execution, error) {

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

// getCurrentWorkflowMetadata returns the owning Argo Workflow metadata for the
// current driver pod.
//
// The compiler can safely pass workflow creation time directly via
// {{workflow.creationTimestamp}}, but recurring-run schedule time is stored in
// the workflowEpoch label and that label is absent for ad hoc runs. Referencing
// the label directly from the compiled template causes Argo to reject manual
// runs before the driver starts, so the driver resolves the label at runtime
// from the Workflow object instead.
func getCurrentWorkflowMetadata(ctx context.Context, namespace string) (*metav1.ObjectMeta, error) {
	restConfig, err := util.GetKubernetesConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize kubernetes config for workflow metadata: %w", err)
	}
	k8sClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize kubernetes client for workflow metadata: %w", err)
	}
	podName, err := config.InPodName()
	if err != nil {
		return nil, fmt.Errorf("failed to determine driver pod name: %w", err)
	}
	pod, err := k8sClient.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve driver pod %q: %w", podName, err)
	}
	workflowName := pod.Labels[workflowcommon.LabelKeyWorkflow]
	if workflowName == "" {
		return nil, nil
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
	defer func() {
		if err != nil {
			err = fmt.Errorf("KFP driver: %w", err)
		}
	}()
	ctx := context.Background()
	if err = validate(); err != nil {
		return err
	}

	// Support reading component spec from a file if value starts with @
	// This bypasses exec() argument size limits for large workflows
	if strings.HasPrefix(*componentSpecJSON, "@") {
		filePath := (*componentSpecJSON)[1:] // Remove the "@" prefix
		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read component spec from file %s: %w", filePath, err)
		}
		*componentSpecJSON = string(data)
		glog.Infof("Read component spec from file: %s (%d bytes)", filePath, len(data))
	}

	proxy.InitializeConfig(*httpProxy, *httpsProxy, *noProxy)
	glog.Infof("input ComponentSpec:%s\n", prettyPrint(*componentSpecJSON))
	componentSpec := &pipelinespec.ComponentSpec{}
	if err := util.UnmarshalString(*componentSpecJSON, componentSpec); err != nil {
		return fmt.Errorf("failed to unmarshal component spec, error: %w\ncomponentSpec: %v", err, prettyPrint(*componentSpecJSON))
	}
	var taskSpec *pipelinespec.PipelineTaskSpec
	if *taskSpecJSON != "" {
		glog.Infof("input TaskSpec:%s\n", prettyPrint(*taskSpecJSON))
		taskSpec = &pipelinespec.PipelineTaskSpec{}
		if err := util.UnmarshalString(*taskSpecJSON, taskSpec); err != nil {
			return fmt.Errorf("failed to unmarshal task spec, error: %w\ntask: %v", err, taskSpecJSON)
		}
	}
	glog.Infof("input ContainerSpec:%s\n", prettyPrint(*containerSpecJson))
	containerSpec := &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{}
	if err := util.UnmarshalString(*containerSpecJson, containerSpec); err != nil {
		return fmt.Errorf("failed to unmarshal container spec, error: %w\ncontainerSpec: %v", err, containerSpecJson)
	}
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
	namespace, err := config.InPodNamespace()
	if err != nil {
		return err
	}
	var tlsCfg *tls.Config
	if *metadataTLSEnabled {
		tlsCfg, err = util.GetTLSConfig(*caCertPath)
		if err != nil {
			return err
		}
	}
	client, err := newMlmdClient(*mlmdServerAddress, *mlmdServerPort, tlsCfg)
	if err != nil {
		return err
	}
	cacheClient, err := cacheutils.NewClient(*mlPipelineServerAddress, *mlPipelineServerPort, *cacheDisabledFlag, tlsCfg)
	if err != nil {
		return err
	}
	// pluginDispatcher executes task-level plugin lifecycle hooks
	pluginDispatcher, err := plugins.GetPluginDispatcher()
	if err != nil {
		glog.Errorf("Failed to initialize plugin dispatcher: %v", err)
	}
	var workflowMeta *metav1.ObjectMeta
	if *pipelineJobCreateTimeUTCArg == "" || *pipelineJobScheduleTimeEpochArg == "" {
		workflowMeta, err = getCurrentWorkflowMetadata(ctx, namespace)
		if err != nil {
			return err
		}
	}
	resolvedPipelineJobCreateTimeUTC, resolvedPipelineJobScheduleTimeUTC, err := resolvePipelineJobTimes(
		*pipelineJobCreateTimeUTCArg,
		*pipelineJobScheduleTimeEpochArg,
		workflowMeta,
	)
	if err != nil {
		return err
	}
	options := driver.Options{
		PipelineName:               *pipelineName,
		RunID:                      *runID,
		RunName:                    *runName,
		RunDisplayName:             *runDisplayName,
		PipelineJobCreateTimeUTC:   resolvedPipelineJobCreateTimeUTC,
		PipelineJobScheduleTimeUTC: resolvedPipelineJobScheduleTimeUTC,
		Namespace:                  namespace,
		Component:                  componentSpec,
		Task:                       taskSpec,
		DAGExecutionID:             *dagExecutionID,
		IterationIndex:             *iterationIndex,
		PipelineLogLevel:           *logLevel,
		PublishLogs:                *publishLogs,
		CacheDisabled:              *cacheDisabledFlag,
		DriverType:                 *driverType,
		TaskName:                   *taskName,
		MLPipelineServerAddress:    *mlPipelineServerAddress,
		MLPipelineServerPort:       *mlPipelineServerPort,
		MLMDServerAddress:          *mlmdServerAddress,
		MLMDServerPort:             *mlmdServerPort,
		MLPipelineTLSEnabled:       *mlPipelineTLSEnabled,
		MLMDTLSEnabled:             *metadataTLSEnabled,
		CaCertPath:                 *caCertPath,
		PluginDispatcher:           pluginDispatcher,
	}
	var execution *driver.Execution
	var driverErr error
	switch *driverType {
	case ROOT_DAG:
		options.RuntimeConfig = runtimeConfig
		execution, driverErr = driver.RootDAG(ctx, options, client)
	case DAG:
		execution, driverErr = driver.DAG(ctx, options, client)
	case CONTAINER:
		options.Container = containerSpec
		options.KubernetesExecutorConfig = k8sExecCfg
		// Set admin defaults only when explicitly configured (non-negative).
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
			v, err := strconv.ParseBool(*defaultHostUsers)
			if err != nil {
				return fmt.Errorf("invalid --default_host_users value %q: %w", *defaultHostUsers, err)
			}
			options.DefaultHostUsers = &v
		}
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

	executionPaths := &ExecutionPaths{
		ExecutionID:    *executionIDPath,
		IterationCount: *iterationCountPath,
		CachedDecision: *cachedDecisionPath,
		Condition:      *conditionPath,
		PodSpecPatch:   *podSpecPatchPath,
	}

	return handleExecution(execution, *driverType, executionPaths)
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

func handleExecution(execution *driver.Execution, driverType string, executionPaths *ExecutionPaths) error {
	if execution.ID != 0 {
		glog.Infof("output execution.ID=%v", execution.ID)
		if executionPaths.ExecutionID != "" {
			if err := writeFile(executionPaths.ExecutionID, []byte(fmt.Sprint(execution.ID))); err != nil {
				return fmt.Errorf("failed to write execution ID to file: %w", err)
			}
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

func newMlmdClient(mlmdServerAddress string, mlmdServerPort string, tlsCfg *tls.Config) (*metadata.Client, error) {
	return metadata.NewClient(mlmdServerAddress, mlmdServerPort, tlsCfg)
}

func initConfig() {
	viper.AutomaticEnv()
}
