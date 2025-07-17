// Copyright 2018 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	apiv1beta1 "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	cm "github.com/kubeflow/pipelines/backend/src/apiserver/client_manager"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/config"
	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/apiserver/server"
	"github.com/kubeflow/pipelines/backend/src/apiserver/template"
	"github.com/kubeflow/pipelines/backend/src/apiserver/webhook"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	corev1 "k8s.io/api/core/v1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

const (
	executionTypeEnv = "ExecutionType"
	launcherEnv      = "Launcher"
	workspaceConfig  = "workspace"
	workspaceSize    = "size"
)

var (
	logLevelFlag                  = flag.String("logLevel", "", "Defines the log level for the application.")
	rpcPortFlag                   = flag.String("rpcPortFlag", ":8887", "RPC Port")
	httpPortFlag                  = flag.String("httpPortFlag", ":8888", "Http Proxy Port")
	webhookPortFlag               = flag.String("webhookPortFlag", ":8443", "Https Proxy Port")
	webhookTLSCertPath            = flag.String("webhookTLSCertPath", "", "Path to the webhook TLS certificate.")
	webhookTLSKeyPath             = flag.String("webhookTLSKeyPath", "", "Path to the webhook TLS private key.")
	configPath                    = flag.String("config", "", "Path to JSON file containing config")
	sampleConfigPath              = flag.String("sampleconfig", "", "Path to samples")
	collectMetricsFlag            = flag.Bool("collectMetricsFlag", true, "Whether to collect Prometheus metrics in API server.")
	usePipelinesKubernetesStorage = flag.Bool("pipelinesStoreKubernetes", false, "Store and run pipeline versions in Kubernetes")
	disableWebhook                = flag.Bool("disableWebhook", false, "Set this if pipelinesStoreKubernetes is on but using a global webhook in a separate pod")
	globalKubernetesWebhookMode   = flag.Bool("globalKubernetesWebhookMode", false, "Set this to run exclusively in Kubernetes Webhook mode")
)

type RegisterHttpHandlerFromEndpoint func(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error

func main() {
	flag.Parse()

	initConfig()
	// check ExecutionType Settings if presents
	if viper.IsSet(executionTypeEnv) {
		util.SetExecutionType(util.ExecutionType(common.GetStringConfig(executionTypeEnv)))
	}
	if viper.IsSet(launcherEnv) {
		template.Launcher = common.GetStringConfig(launcherEnv)
	}

	backgroundCtx, backgroundCancel := context.WithCancel(signals.SetupSignalHandler())
	defer backgroundCancel()

	wg := sync.WaitGroup{}

	options := &cm.Options{
		UsePipelineKubernetesStorage: *usePipelinesKubernetesStorage,
		GlobalKubernetesWebhookMode:  *globalKubernetesWebhookMode,
		Context:                      backgroundCtx,
		WaitGroup:                    &wg,
	}

	logLevel := *logLevelFlag
	if logLevel == "" {
		logLevel = "info"
	}

	level, err := log.ParseLevel(logLevel)
	if err != nil {
		log.Fatal("Invalid log level:", err)
	}
	log.SetLevel(level)

	zapLevel, err := common.ParseLogLevel(logLevel)
	if err != nil {
		glog.Infof("%v. Defaulting to info level.", err)
		zapLevel = zapcore.InfoLevel
	}

	ctrllog.SetLogger(zap.New(zap.UseFlagOptions(&zap.Options{Level: zapLevel})))

	clientManager, err := cm.NewClientManager(options)
	if err != nil {
		glog.Fatalf("Failed to initialize ClientManager: %v", err)
	}

	defer clientManager.Close()
	webhookOnlyMode := *globalKubernetesWebhookMode

	if (*usePipelinesKubernetesStorage && !*disableWebhook) || webhookOnlyMode {
		if *disableWebhook && webhookOnlyMode {
			glog.Fatalf("Invalid configuration: globalKubernetesWebhookMode is enabled but the webhook is disabled")
		}

		wg.Add(1)
		webhookServer, err := startWebhook(
			clientManager.ControllerClient(true), clientManager.ControllerClient(false), &wg,
		)
		if err != nil {
			glog.Fatalf("Failed to start Kubernetes webhook server: %v", err)
		}
		go func() {
			<-backgroundCtx.Done()
			glog.Info("Shutting down Kubernetes webhook server...")
			if err := webhookServer.Shutdown(context.Background()); err != nil {
				glog.Errorf("Error shutting down webhook server: %v", err)
			}
		}()

		// This mode is used when there are multiple Kubeflow Pipelines installations on the same cluster but the
		// administrator only wants to have a single webhook endpoint for all of them. This causes only the webhook
		// endpoints to be available.
		if webhookOnlyMode {
			wg.Wait()
			return
		}

	}

	var pvcSpec *corev1.PersistentVolumeClaimSpec
	var defaultWorkspaceSize string
	pvcSpec, defaultWorkspaceSize, err = getPVCSpec()
	if err != nil {
		glog.Fatalf("Failed to get Workspace PVC Spec: %v", err)
	}

	resourceManager := resource.NewResourceManager(
		clientManager,
		&resource.ResourceManagerOptions{
			CollectMetrics:       *collectMetricsFlag,
			CacheDisabled:        !common.GetBoolConfigWithDefault("CacheEnabled", true),
			DefaultWorkspace:     pvcSpec,
			DefaultWorkspaceSize: defaultWorkspaceSize,
		},
	)
	err = config.LoadSamples(resourceManager, *sampleConfigPath)
	if err != nil {
		glog.Fatalf("Failed to load samples. Err: %v", err)
	}

	if !common.IsMultiUserMode() {
		_, err = resourceManager.CreateDefaultExperiment("")
		if err != nil {
			glog.Fatalf("Failed to create default experiment. Err: %v", err)
		}
	}

	wg.Add(1)
	go reconcileSwfCrs(resourceManager, backgroundCtx, &wg)
	go startRpcServer(resourceManager)
	// This is blocking
	startHttpProxy(resourceManager, *usePipelinesKubernetesStorage)
	backgroundCancel()
	wg.Wait()
}

func reconcileSwfCrs(resourceManager *resource.ResourceManager, ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	err := resourceManager.ReconcileSwfCrs(ctx)
	if err != nil {
		log.Errorf("Could not reconcile the ScheduledWorkflow Kubernetes resources: %v", err)
	}
}

// A custom http request header matcher to pass on the user identity
// Reference: https://github.com/grpc-ecosystem/grpc-gateway/blob/v1.16.0/docs/_docs/customizingyourgateway.md#mapping-from-http-request-headers-to-grpc-client-metadata
func grpcCustomMatcher(key string) (string, bool) {
	if strings.EqualFold(key, common.GetKubeflowUserIDHeader()) {
		return strings.ToLower(key), true
	}
	return strings.ToLower(key), false
}

func startRpcServer(resourceManager *resource.ResourceManager) {
	glog.Info("Starting RPC server")
	listener, err := net.Listen("tcp", *rpcPortFlag)
	if err != nil {
		glog.Fatalf("Failed to start RPC server: %v", err)
	}
	s := grpc.NewServer(grpc.UnaryInterceptor(apiServerInterceptor), grpc.MaxRecvMsgSize(math.MaxInt32))

	sharedExperimentServer := server.NewExperimentServer(resourceManager, &server.ExperimentServerOptions{CollectMetrics: *collectMetricsFlag})
	sharedPipelineServer := server.NewPipelineServer(
		resourceManager,
		&server.PipelineServerOptions{
			CollectMetrics: *collectMetricsFlag,
		},
	)
	sharedJobServer := server.NewJobServer(resourceManager, &server.JobServerOptions{CollectMetrics: *collectMetricsFlag})
	sharedRunServer := server.NewRunServer(resourceManager, &server.RunServerOptions{CollectMetrics: *collectMetricsFlag})
	sharedReportServer := server.NewReportServer(resourceManager)

	apiv1beta1.RegisterExperimentServiceServer(s, sharedExperimentServer)
	apiv1beta1.RegisterPipelineServiceServer(s, sharedPipelineServer)
	apiv1beta1.RegisterJobServiceServer(s, sharedJobServer)
	apiv1beta1.RegisterRunServiceServer(s, sharedRunServer)
	apiv1beta1.RegisterTaskServiceServer(s, server.NewTaskServer(resourceManager))
	apiv1beta1.RegisterReportServiceServer(s, sharedReportServer)

	apiv1beta1.RegisterVisualizationServiceServer(
		s,
		server.NewVisualizationServer(
			resourceManager,
			common.GetStringConfig(cm.VisualizationServiceHost),
			common.GetStringConfig(cm.VisualizationServicePort),
		))
	apiv1beta1.RegisterAuthServiceServer(s, server.NewAuthServer(resourceManager))

	apiv2beta1.RegisterExperimentServiceServer(s, sharedExperimentServer)
	apiv2beta1.RegisterPipelineServiceServer(s, sharedPipelineServer)
	apiv2beta1.RegisterRecurringRunServiceServer(s, sharedJobServer)
	apiv2beta1.RegisterRunServiceServer(s, sharedRunServer)
	apiv2beta1.RegisterReportServiceServer(s, sharedReportServer)

	// Register reflection service on gRPC server.
	reflection.Register(s)
	if err := s.Serve(listener); err != nil {
		glog.Fatalf("Failed to serve rpc listener: %v", err)
	}
	glog.Info("RPC server started")
}

func startHttpProxy(resourceManager *resource.ResourceManager, usePipelinesKubernetesStorage bool) {
	glog.Info("Starting Http Proxy")

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var pipelineStore string

	if usePipelinesKubernetesStorage {
		pipelineStore = "kubernetes"
	} else {
		pipelineStore = "database"
	}

	// Create gRPC HTTP MUX and register services for v1beta1 api.
	runtimeMux := runtime.NewServeMux(runtime.WithIncomingHeaderMatcher(grpcCustomMatcher))
	registerHttpHandlerFromEndpoint(apiv1beta1.RegisterPipelineServiceHandlerFromEndpoint, "PipelineService", ctx, runtimeMux)
	registerHttpHandlerFromEndpoint(apiv1beta1.RegisterExperimentServiceHandlerFromEndpoint, "ExperimentService", ctx, runtimeMux)
	registerHttpHandlerFromEndpoint(apiv1beta1.RegisterJobServiceHandlerFromEndpoint, "JobService", ctx, runtimeMux)
	registerHttpHandlerFromEndpoint(apiv1beta1.RegisterRunServiceHandlerFromEndpoint, "RunService", ctx, runtimeMux)
	registerHttpHandlerFromEndpoint(apiv1beta1.RegisterTaskServiceHandlerFromEndpoint, "TaskService", ctx, runtimeMux)
	registerHttpHandlerFromEndpoint(apiv1beta1.RegisterReportServiceHandlerFromEndpoint, "ReportService", ctx, runtimeMux)
	registerHttpHandlerFromEndpoint(apiv1beta1.RegisterVisualizationServiceHandlerFromEndpoint, "Visualization", ctx, runtimeMux)
	registerHttpHandlerFromEndpoint(apiv1beta1.RegisterAuthServiceHandlerFromEndpoint, "AuthService", ctx, runtimeMux)

	// Create gRPC HTTP MUX and register services for v2beta1 api.
	registerHttpHandlerFromEndpoint(apiv2beta1.RegisterExperimentServiceHandlerFromEndpoint, "ExperimentService", ctx, runtimeMux)
	registerHttpHandlerFromEndpoint(apiv2beta1.RegisterPipelineServiceHandlerFromEndpoint, "PipelineService", ctx, runtimeMux)
	registerHttpHandlerFromEndpoint(apiv2beta1.RegisterRecurringRunServiceHandlerFromEndpoint, "RecurringRunService", ctx, runtimeMux)
	registerHttpHandlerFromEndpoint(apiv2beta1.RegisterRunServiceHandlerFromEndpoint, "RunService", ctx, runtimeMux)
	registerHttpHandlerFromEndpoint(apiv2beta1.RegisterReportServiceHandlerFromEndpoint, "ReportService", ctx, runtimeMux)

	// Create a top level mux to include both pipeline upload server and gRPC servers.
	topMux := mux.NewRouter()

	// multipart upload is only supported in HTTP. In long term, we should have gRPC endpoints that
	// accept pipeline url for importing.
	// https://github.com/grpc-ecosystem/grpc-gateway/issues/410
	sharedPipelineUploadServer := server.NewPipelineUploadServer(resourceManager, &server.PipelineUploadServerOptions{CollectMetrics: *collectMetricsFlag})
	// API v1beta1
	topMux.HandleFunc("/apis/v1beta1/pipelines/upload", sharedPipelineUploadServer.UploadPipelineV1)
	topMux.HandleFunc("/apis/v1beta1/pipelines/upload_version", sharedPipelineUploadServer.UploadPipelineVersionV1)
	topMux.HandleFunc("/apis/v1beta1/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"commit_sha":"`+common.GetStringConfigWithDefault("COMMIT_SHA", "unknown")+`", "tag_name":"`+common.GetStringConfigWithDefault("TAG_NAME", "unknown")+`", "multi_user":`+strconv.FormatBool(common.IsMultiUserMode())+`}`)
	})
	// API v2beta1
	topMux.HandleFunc("/apis/v2beta1/pipelines/upload", sharedPipelineUploadServer.UploadPipeline)
	topMux.HandleFunc("/apis/v2beta1/pipelines/upload_version", sharedPipelineUploadServer.UploadPipelineVersion)
	topMux.HandleFunc("/apis/v2beta1/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"commit_sha":"`+common.GetStringConfigWithDefault("COMMIT_SHA", "unknown")+`", "tag_name":"`+common.GetStringConfigWithDefault("TAG_NAME", "unknown")+`", "multi_user":`+strconv.FormatBool(common.IsMultiUserMode())+`, "pipeline_store": "`+pipelineStore+`"}`)
	})

	// log streaming is provided via HTTP.
	runLogServer := server.NewRunLogServer(resourceManager)
	topMux.HandleFunc("/apis/v1alpha1/runs/{run_id}/nodes/{node_id}/log", runLogServer.ReadRunLogV1)

	topMux.PathPrefix("/apis/").Handler(runtimeMux)

	// Register a handler for Prometheus to poll.
	topMux.Handle("/metrics", promhttp.Handler())

	http.ListenAndServe(*httpPortFlag, topMux)
	glog.Info("Http Proxy started")
}

func startWebhook(client ctrlclient.Client, clientNoCahe ctrlclient.Client, wg *sync.WaitGroup) (*http.Server, error) {
	glog.Info("Starting the Kubernetes webhooks...")

	tlsCertPath := *webhookTLSCertPath
	tlsKeyPath := *webhookTLSKeyPath

	topMux := mux.NewRouter()

	pvValidateWebhook, pvMutateWebhook, err := webhook.NewPipelineVersionWebhook(client, clientNoCahe)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate the Kubernetes webhook: %v", err)
	}

	topMux.Handle("/webhooks/validate-pipelineversion", pvValidateWebhook)
	topMux.Handle("/webhooks/mutate-pipelineversion", pvMutateWebhook)

	webhookServer := &http.Server{
		Addr:    *webhookPortFlag,
		Handler: topMux,
	}

	go func() {
		defer wg.Done()

		if tlsCertPath != "" && tlsKeyPath != "" {
			if !common.FileExists(tlsCertPath) || !common.FileExists(tlsKeyPath) {
				glog.Fatalf("TLS certificate/key paths are set but files do not exist")
			}
			glog.Info("Starting the Kubernetes webhook with TLS")
			err := webhookServer.ListenAndServeTLS(tlsCertPath, tlsKeyPath)
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				glog.Fatalf("Failed to start the Kubernetes webhook with TLS: %v", err)
			}
			return
		}
		glog.Warning("TLS certificate/key paths are not set. Starting webhook server without TLS.")
		err := webhookServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			glog.Fatalf("Failed to start Kubernetes webhook server: %v", err)
		}
	}()

	return webhookServer, nil
}

func registerHttpHandlerFromEndpoint(handler RegisterHttpHandlerFromEndpoint, serviceName string, ctx context.Context, mux *runtime.ServeMux) {
	endpoint := "localhost" + *rpcPortFlag
	opts := []grpc.DialOption{grpc.WithInsecure(), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(math.MaxInt32))}

	if err := handler(ctx, mux, endpoint, opts); err != nil {
		glog.Fatalf("Failed to register %v handler: %v", serviceName, err)
	}
}

func initConfig() {
	// Import environment variable, support nested vars e.g. OBJECTSTORECONFIG_ACCESSKEY
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)
	viper.AutomaticEnv()
	// We need empty string env var for e.g. KUBEFLOW_USERID_PREFIX.
	viper.AllowEmptyEnv(true)

	// Set configuration file name. The format is auto detected in this case.
	viper.SetConfigName("config")
	viper.AddConfigPath(*configPath)
	err := viper.ReadInConfig()
	if err != nil {
		glog.Fatalf("Fatal error config file: %s", err)
	}

	// Watch for configuration change
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		// Read in config again
		viper.ReadInConfig()
	})

	proxy.InitializeConfigWithEnv()
}

// getPVCSpec retrieves the default workspace PersistentVolumeClaimSpec and size from the config.
// These defaults are used for workspace PVCs when users do not specify their own configuration.
func getPVCSpec() (*corev1.PersistentVolumeClaimSpec, string, error) {
	workspaceConfig := viper.Sub(workspaceConfig)
	if workspaceConfig == nil {
		glog.Info("No workspace config found; proceeding without a default PVC spec")
		return nil, "", nil
	}
	var pvcSpec corev1.PersistentVolumeClaimSpec
	if err := workspaceConfig.UnmarshalKey("volumeclaimtemplatespec", &pvcSpec); err != nil {
		return nil, "", fmt.Errorf("failed to unmarshal workspace.volumeclaimtemplatespec: %w", err)
	}
	if len(pvcSpec.AccessModes) == 0 || pvcSpec.StorageClassName == nil || *pvcSpec.StorageClassName == "" {
		return nil, "", fmt.Errorf("invalid workspace.volumeclaimtemplatespec: must specify accessModes and storageClassName")
	}
	defaultSize := workspaceConfig.GetString(workspaceSize)
	return &pvcSpec, defaultSize, nil
}
