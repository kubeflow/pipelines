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

package executorplugin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/driver/driverapi"
	"github.com/kubeflow/pipelines/backend/src/v2/client_manager"
	"github.com/kubeflow/pipelines/backend/src/v2/common/plugins"
	// Register runtime plugin factories.
	_ "github.com/kubeflow/pipelines/backend/src/v2/common/plugins/all"
	"github.com/kubeflow/pipelines/backend/src/v2/config"
	"github.com/kubeflow/pipelines/backend/src/v2/driver"
	drivercommon "github.com/kubeflow/pipelines/backend/src/v2/driver/common"
	"github.com/kubeflow/pipelines/backend/src/v2/objectstore"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/http/httpproxy"
	"k8s.io/client-go/kubernetes"
)

const (
	caCertPathEnvVar            = "CA_CERT_PATH"
	driverLogDirectory          = "/kfp/log"
	driverLogPublicationTimeout = 5 * time.Second
)

type driverRunner struct {
	newClients   func(context.Context, driverapi.DriverPluginArgs, string) (*invocationClients, error)
	execute      func(context.Context, drivercommon.Options, client_manager.ClientManagerInterface) (*driver.Execution, error)
	uploadLogs   func(context.Context, *driverLogArtifactContext, kubernetes.Interface) error
	logDirectory string
}

func newDriverRunner() *driverRunner {
	return &driverRunner{
		newClients:   inClusterClientFactory().newDriverClientManager,
		execute:      executeDriver,
		uploadLogs:   logUploader{openBucket: objectstore.OpenBucket, uploadBlob: objectstore.UploadBlob}.uploadDriverLogArtifact,
		logDirectory: driverLogDirectory,
	}
}

type driverInvocation struct {
	clients    *invocationClients
	logContext driverLogArtifactContext
	logCloser  io.Closer
}

func (r *driverRunner) drive(ctx context.Context, args driverapi.DriverPluginArgs) (*driver.Execution, error) {
	verbosity, err := parseLogLevel(args.LogLevel)
	if err != nil {
		return nil, err
	}
	proxyConfig := proxy.NewConfig(args.HTTPProxy, args.HTTPSProxy, args.NoProxy)
	resolveProxy := (&httpproxy.Config{
		HTTPProxy: proxyConfig.GetHttpProxy(), HTTPSProxy: proxyConfig.GetHttpsProxy(), NoProxy: proxyConfig.GetNoProxy(),
	}).ProxyFunc()
	ctx = util.WithHTTPProxy(ctx, func(req *http.Request) (*url.URL, error) { return resolveProxy(req.URL) })
	logID := uuid.NewString()
	logFile := filepath.Join(r.logDirectory, logID+".log")
	ctx, logCloser, err := util.WithLogger(ctx, logFile)
	if err != nil {
		return nil, fmt.Errorf("KFP driver: failed to create driver logger in %q: check directory permissions: %w", r.logDirectory, err)
	}
	if verbosity >= 4 {
		util.GetLoggerFrom(ctx).SetLevel(logrus.TraceLevel)
	}
	invocation := &driverInvocation{
		logCloser:  logCloser,
		logContext: driverLogArtifactContext{Task: args.TaskName, LocalPath: logFile, LogID: logID, RunID: args.RunID},
	}
	defer invocation.publishAndClose(ctx, r.uploadLogs)
	options, err := r.setup(ctx, args, invocation)
	if err == nil {
		invocation.logContext.Execution, err = r.execute(ctx, *options, invocation.clients.manager)
		if err != nil {
			util.GetLoggerFrom(ctx).Errorf("driver execution failed: %v", err)
		}
	}
	if err != nil {
		err = fmt.Errorf("KFP driver: %w", err)
	}
	return invocation.logContext.Execution, err
}

func parseLogLevel(value string) (int, error) {
	level, err := strconv.Atoi(value)
	if err != nil || level < 0 {
		return 0, fmt.Errorf("invalid log_level %q: provide a nonnegative integer verbosity", value)
	}
	return level, nil
}

// publishAndClose keeps log publication best effort, after flushing the file and
// before removing it or releasing the clients used by publication.
func (i *driverInvocation) publishAndClose(ctx context.Context, upload func(context.Context, *driverLogArtifactContext, kubernetes.Interface) error) {
	if i.logCloser != nil {
		if err := i.logCloser.Close(); err != nil {
			glog.Errorf("Failed to close driver log file: %v", err)
		}
	}
	if i.logContext.PipelineRoot != "" && i.clients != nil {
		// Best-effort publication must not delay task completion indefinitely.
		uploadContext, cancel := context.WithTimeout(ctx, driverLogPublicationTimeout)
		i.logContext.KFPAPI = i.clients.manager.KFPAPIClient()
		if err := upload(uploadContext, &i.logContext, i.clients.manager.K8sClient()); err != nil {
			glog.Errorf("Failed to upload driver-logs artifact: %v", err)
		}
		cancel()
	}
	if err := os.Remove(i.logContext.LocalPath); err != nil {
		glog.Errorf("Failed to remove processed driver log file: %v", err)
	}
	if i.clients != nil {
		if err := i.clients.manager.Close(); err != nil {
			glog.Errorf("Failed to close driver clients: %v", err)
		}
	}
}

func executeDriver(ctx context.Context, options drivercommon.Options, clients client_manager.ClientManagerInterface) (*driver.Execution, error) {
	switch options.DriverType {
	case rootDAG:
		return driver.RootDAG(ctx, options, clients)
	case dagDriver:
		return driver.DAG(ctx, options, clients)
	case containerDriver:
		return driver.Container(ctx, options, clients)
	default:
		return nil, fmt.Errorf("unknown driver type %q: use ROOT_DAG, DAG, or CONTAINER", options.DriverType)
	}
}

func (r *driverRunner) setup(ctx context.Context, args driverapi.DriverPluginArgs, invocation *driverInvocation) (*drivercommon.Options, error) {
	log := util.GetLoggerFrom(ctx)

	log.Infof("driver invocation: type=%s run_id=%s task_name=%s", args.Type, args.RunID, args.TaskName)
	namespace, err := resolveNamespace(args.Namespace)
	if err != nil {
		return nil, err
	}
	iterationIndex, err := strconv.Atoi(args.IterationIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to parse iteration index; provide an integer iteration_index: %w", err)
	}

	caCertPath := os.Getenv(caCertPathEnvVar)
	clients, err := r.newClients(ctx, args, caCertPath)
	if err != nil {
		return nil, err
	}
	invocation.clients = clients
	clientManager, pod := clients.manager, clients.pod
	podName, podUID := pod.Name, string(pod.UID)

	var runtimeConfig *pipelinespec.PipelineJob_RuntimeConfig
	if args.RuntimeConfig != "" {
		runtimeConfig = &pipelinespec.PipelineJob_RuntimeConfig{}
		if err := util.UnmarshalString(args.RuntimeConfig, runtimeConfig); err != nil {
			return nil, fmt.Errorf("failed to unmarshal runtime config; provide valid PipelineJob runtime_config JSON: %w", err)
		}
	}
	k8sExecCfg, err := parseExecConfigJSON(&args.KubernetesConfig)
	if err != nil {
		return nil, err
	}

	fullView := go_client.GetRunRequest_FULL
	run, err := clientManager.KFPAPIClient().GetRun(ctx, &go_client.GetRunRequest{RunId: args.RunID, View: &fullView})
	if err != nil {
		return nil, fmt.Errorf("failed to get run %q: %w", args.RunID, err)
	}
	var parentTask *go_client.PipelineTask
	if args.ParentTaskID != "" {
		parentTask, err = clientManager.KFPAPIClient().GetTask(ctx, &go_client.GetTaskRequest{TaskId: args.ParentTaskID, RunId: args.RunID})
		if err != nil {
			return nil, fmt.Errorf("failed to get parent task %q for run %q: %w", args.ParentTaskID, args.RunID, err)
		}
	}
	scopePath, err := buildScopePath(ctx, run, parentTask, args.TaskName, args.Type, clientManager.KFPAPIClient())
	if err != nil {
		return nil, fmt.Errorf("failed to build scope path: %w", err)
	}
	componentSpec, taskSpec, containerSpec, err := resolveDriverSpecsFromScopePath(scopePath, args.Type)
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
		placeholderUsage, createTimeUTC, scheduleTimeEpochSeconds, clients.workflowMetadata)
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
	pipelineRoot, err := config.GetPipelineRootWithPipelineRunContext(ctx, args.PipelineName, namespace, clientManager.K8sClient(), run)
	var storeSessionInfo string
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

	invocation.logContext.Namespace = namespace
	invocation.logContext.PipelineRoot = pipelineRoot
	invocation.logContext.StoreSessionInfo = storeSessionInfo
	switch args.Type {
	case rootDAG:
		options.RuntimeConfig = runtimeConfig
	case containerDriver:
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
		options.OutputPathPrefix = uuid.NewString()
		invocation.logContext.OutputPathPrefix = options.OutputPathPrefix
	}
	return &options, nil
}
