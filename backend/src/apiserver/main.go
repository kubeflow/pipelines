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
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
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

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"google.golang.org/grpc/credentials"

	"github.com/fsnotify/fsnotify"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
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
	"google.golang.org/grpc/credentials/insecure"
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
)

var (
	logLevelFlag                  = flag.String("logLevel", "", "Defines the log level for the application.")
	rpcPortFlag                   = flag.String("rpcPortFlag", ":8887", "RPC Port")
	httpPortFlag                  = flag.String("httpPortFlag", ":8888", "Http Proxy Port")
	webhookPortFlag               = flag.String("webhookPortFlag", ":8443", "Https Proxy Port")
	webhookTLSCertPath            = flag.String("webhookTLSCertPath", "", "Path to the webhook TLS certificate. Defaults to tlsCertPath value")
	webhookTLSKeyPath             = flag.String("webhookTLSKeyPath", "", "Path to the webhook TLS private key. Defaults to tlsCertKeyPath value")
	configPath                    = flag.String("config", "", "Path to JSON file containing config")
	sampleConfigPath              = flag.String("sampleconfig", "", "Path to samples")
	tlsCertPath                   = flag.String("tlsCertPath", "", "Path to the public TLS cert.")
	tlsCertKeyPath                = flag.String("tlsCertKeyPath", "", "Path to the private TLS key cert.")
	collectMetricsFlag            = flag.Bool("collectMetricsFlag", true, "Whether to collect Prometheus metrics in API server.")
	usePipelinesKubernetesStorage = flag.Bool("pipelinesStoreKubernetes", false, "Store and run pipeline versions in Kubernetes")
	disableWebhook                = flag.Bool("disableWebhook", false, "Set this if pipelinesStoreKubernetes is on but using a global webhook in a separate pod")
	globalKubernetesWebhookMode   = flag.Bool("globalKubernetesWebhookMode", false, "Set this to run exclusively in Kubernetes Webhook mode")
)

type RegisterHttpHandlerFromEndpoint func(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error

func initCerts() (*tls.Config, error) {
	switch {
	case *tlsCertPath == "" && *tlsCertKeyPath == "":
		// User can choose not to provide certs
		return nil, nil
	case *tlsCertPath == "":
		return nil, fmt.Errorf("missing tlsCertPath when specifying cert paths, both tlsCertPath and tlsCertKeyPath are required")
	case *tlsCertKeyPath == "":
		return nil, fmt.Errorf("missing tlsCertKeyPath when specifying cert paths, both tlsCertPath and tlsCertKeyPath are required")
	}

	serverCert, err := tls.LoadX509KeyPair(*tlsCertPath, *tlsCertKeyPath)
	if err != nil {
		return nil, err
	}
	tlsCfg := &tls.Config{
		ServerName:   common.GetMLPipelineServiceName() + "." + common.GetPodNamespace() + ".svc." + common.GetClusterDomain(),
		Certificates: []tls.Certificate{serverCert},
	}
	glog.Info("TLS cert key/pair loaded.")
	return tlsCfg, err
}

func main() {
	flag.Parse()

	if err := initConfig(); err != nil {
		glog.Fatalf("Failed to initialize config: %v", err)
	}
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

	tlsCfg, err := initCerts()
	if err != nil {
		glog.Fatalf("Failed to parse Cert paths. Err: %v", err)
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
	pvcSpec, err = getPVCSpec()
	if err != nil {
		glog.Fatalf("Failed to get Workspace PVC Spec: %v", err)
	}

	resourceManager := resource.NewResourceManager(
		clientManager,
		&resource.ResourceManagerOptions{
			CollectMetrics:       *collectMetricsFlag,
			CacheDisabled:        !common.GetBoolConfigWithDefault("CacheEnabled", true),
			DefaultWorkspace:     pvcSpec,
			MLPipelineTLSEnabled: tlsCfg != nil,
			DefaultRunAsUser:     parseOptionalInt64(common.GetDefaultSecurityContextRunAsUser()),
			DefaultRunAsGroup:    parseOptionalInt64(common.GetDefaultSecurityContextRunAsGroup()),
			DefaultRunAsNonRoot:  parseOptionalBool(common.GetDefaultSecurityContextRunAsNonRoot()),
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
	go startRPCServer(resourceManager, tlsCfg)
	// This is blocking
	startHTTPProxy(resourceManager, *usePipelinesKubernetesStorage, tlsCfg)
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
	if strings.EqualFold(key, common.ClearTagsMetadataKey) {
		return common.ClearTagsMetadataKey, true
	}
	return strings.ToLower(key), false
}

// clearTagsMiddleware inspects PUT/PATCH JSON request bodies on pipeline update
// paths and sets the x-clear-tags header when the "tags" field is an empty map
// ("tags":{}). This is needed because protobuf binary encoding cannot
// distinguish an empty map from nil, so the clear-tags intent is lost during
// the HTTP→gRPC proxy roundtrip.
//
// Scoped to pipeline and pipeline version update paths to avoid interfering
// with other endpoints that may have a top-level "tags" field.
func clearTagsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if (r.Method == http.MethodPut || r.Method == http.MethodPatch) && r.Body != nil &&
			isPipelineUpdatePath(r.URL.Path) {
			body, err := io.ReadAll(r.Body)
			r.Body.Close()
			if err == nil {
				var raw map[string]json.RawMessage
				if json.Unmarshal(body, &raw) == nil {
					if tagsVal, hasTags := raw["tags"]; hasTags && string(tagsVal) == "{}" {
						r.Header.Set(common.ClearTagsMetadataKey, "true")
					}
				}
			}
			r.Body = io.NopCloser(bytes.NewReader(body))
		}
		next.ServeHTTP(w, r)
	})
}

// isPipelineUpdatePath returns true if the URL path matches a pipeline or
// pipeline version update endpoint (v2beta1 only, since v1beta1 does not
// support tags).
func isPipelineUpdatePath(path string) bool {
	// v2beta1 UpdatePipeline:        PATCH /apis/v2beta1/pipelines/{pipeline_id}
	// v2beta1 UpdatePipelineVersion: PATCH /apis/v2beta1/pipelines/{pipeline_id}/versions/{version_id}
	return strings.HasPrefix(path, "/apis/v2beta1/pipelines/")
}

func startRPCServer(resourceManager *resource.ResourceManager, tlsCfg *tls.Config) {
	var s *grpc.Server

	grpc_prometheus.EnableHandlingTimeHistogram(
		grpc_prometheus.WithHistogramBuckets([]float64{
			0.005, 0.01, 0.03, 0.1, 0.3, 1, 3, 10, 15, 30, 60, 120, 300, // 5 ms -> 5 min
		}),
	)

	if tlsCfg != nil {
		glog.Info("Starting RPC server (TLS enabled)")
		tlsCredentials := credentials.NewTLS(tlsCfg)
		s = grpc.NewServer(
			grpc.Creds(tlsCredentials),
			grpc.ChainUnaryInterceptor(
				grpc_prometheus.UnaryServerInterceptor,
				apiServerInterceptor,
			),
			grpc.MaxRecvMsgSize(math.MaxInt32),
		)
	} else {
		glog.Info("Starting RPC server")
		s = grpc.NewServer(grpc.ChainUnaryInterceptor(
			grpc_prometheus.UnaryServerInterceptor,
			apiServerInterceptor,
		), grpc.MaxRecvMsgSize(math.MaxInt32))
	}

	listener, err := net.Listen("tcp", *rpcPortFlag)
	if err != nil {
		glog.Fatalf("Failed to start RPC server: %v", err)
	}

	ExperimentServerV1 := server.NewExperimentServerV1(resourceManager, &server.ExperimentServerOptions{CollectMetrics: *collectMetricsFlag})
	ExperimentServer := server.NewExperimentServer(resourceManager, &server.ExperimentServerOptions{CollectMetrics: *collectMetricsFlag})

	PipelineServerV1 := server.NewPipelineServerV1(resourceManager, &server.PipelineServerOptions{CollectMetrics: *collectMetricsFlag})
	PipelineServer := server.NewPipelineServer(resourceManager, &server.PipelineServerOptions{CollectMetrics: *collectMetricsFlag})

	RunServerV1 := server.NewRunServerV1(resourceManager, &server.RunServerOptions{CollectMetrics: *collectMetricsFlag})
	RunServer := server.NewRunServer(resourceManager, &server.RunServerOptions{CollectMetrics: *collectMetricsFlag})

	JobServerV1 := server.NewJobServerV1(resourceManager, &server.JobServerOptions{CollectMetrics: *collectMetricsFlag})
	JobServer := server.NewJobServer(resourceManager, &server.JobServerOptions{CollectMetrics: *collectMetricsFlag})

	ReportServerV1 := server.NewReportServerV1(resourceManager)
	ReportServer := server.NewReportServer(resourceManager)

	apiv1beta1.RegisterExperimentServiceServer(s, ExperimentServerV1)
	apiv1beta1.RegisterPipelineServiceServer(s, PipelineServerV1)
	apiv1beta1.RegisterJobServiceServer(s, JobServerV1)
	apiv1beta1.RegisterRunServiceServer(s, RunServerV1)
	apiv1beta1.RegisterTaskServiceServer(s, server.NewTaskServer(resourceManager))
	apiv1beta1.RegisterReportServiceServer(s, ReportServerV1)

	apiv1beta1.RegisterVisualizationServiceServer(
		s,
		server.NewVisualizationServer(
			resourceManager,
			common.GetStringConfig(cm.VisualizationServiceHost),
			common.GetStringConfig(cm.VisualizationServicePort),
		))
	apiv1beta1.RegisterAuthServiceServer(s, server.NewAuthServer(resourceManager))

	apiv2beta1.RegisterExperimentServiceServer(s, ExperimentServer)
	apiv2beta1.RegisterPipelineServiceServer(s, PipelineServer)
	apiv2beta1.RegisterRecurringRunServiceServer(s, JobServer)
	apiv2beta1.RegisterRunServiceServer(s, RunServer)
	apiv2beta1.RegisterReportServiceServer(s, ReportServer)

	// Register reflection service on gRPC server.
	reflection.Register(s)
	if err := s.Serve(listener); err != nil {
		glog.Fatalf("Failed to serve rpc listener: %v", err)
	}
	glog.Info("RPC server started")
}

func startHTTPProxy(resourceManager *resource.ResourceManager, usePipelinesKubernetesStorage bool, tlsCfg *tls.Config) {
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

	runtimeMux := runtime.NewServeMux(
		runtime.WithIncomingHeaderMatcher(grpcCustomMatcher),
		runtime.WithMarshalerOption(runtime.MIMEWildcard, common.CustomMarshaler()))
	register := func(handler RegisterHttpHandlerFromEndpoint, serviceName string) {
		if err := registerHTTPHandlerFromEndpoint(ctx, handler, serviceName, runtimeMux, tlsCfg); err != nil {
			glog.Fatalf("%v", err)
		}
	}
	register(apiv1beta1.RegisterPipelineServiceHandlerFromEndpoint, "PipelineService")
	register(apiv1beta1.RegisterExperimentServiceHandlerFromEndpoint, "ExperimentService")
	register(apiv1beta1.RegisterJobServiceHandlerFromEndpoint, "JobService")
	register(apiv1beta1.RegisterRunServiceHandlerFromEndpoint, "RunService")
	register(apiv1beta1.RegisterTaskServiceHandlerFromEndpoint, "TaskService")
	register(apiv1beta1.RegisterReportServiceHandlerFromEndpoint, "ReportService")
	register(apiv1beta1.RegisterVisualizationServiceHandlerFromEndpoint, "Visualization")
	register(apiv1beta1.RegisterAuthServiceHandlerFromEndpoint, "AuthService")

	// Create gRPC HTTP MUX and register services for v2beta1 api.
	register(apiv2beta1.RegisterExperimentServiceHandlerFromEndpoint, "ExperimentService")
	register(apiv2beta1.RegisterPipelineServiceHandlerFromEndpoint, "PipelineService")
	register(apiv2beta1.RegisterRecurringRunServiceHandlerFromEndpoint, "RecurringRunService")
	register(apiv2beta1.RegisterRunServiceHandlerFromEndpoint, "RunService")
	register(apiv2beta1.RegisterReportServiceHandlerFromEndpoint, "ReportService")

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

	// Artifact reading endpoints (implemented with streaming for memory efficiency)
	runArtifactServer := server.NewRunArtifactServer(resourceManager)
	topMux.HandleFunc("/apis/v1beta1/runs/{run_id}/nodes/{node_id}/artifacts/{artifact_name}:read", runArtifactServer.ReadArtifactV1).Methods(http.MethodGet)
	topMux.HandleFunc("/apis/v2beta1/runs/{run_id}/nodes/{node_id}/artifacts/{artifact_name}:read", runArtifactServer.ReadArtifact).Methods(http.MethodGet)

	// Driver endpoint moved to the standalone kfp-driver service
	// (manifests/kustomize/base/pipeline/kfp-driver-*.yaml). Argo HTTP templates
	// now POST to http://kfp-driver.kubeflow.svc.cluster.local:8080/api/v1/template.execute.

	topMux.PathPrefix("/apis/").Handler(clearTagsMiddleware(runtimeMux))

	// Register a handler for Prometheus to poll.
	topMux.Handle("/metrics", promhttp.Handler())

	if tlsCfg != nil {
		glog.Info("Starting Https Proxy")
		https := http.Server{
			TLSConfig: tlsCfg,
			Addr:      *httpPortFlag,
			Handler:   topMux,
		}
		err := https.ListenAndServeTLS(*tlsCertPath, *tlsCertKeyPath)
		if err != nil {
			glog.Errorf("Error starting https server: %v", err)
			return
		}
	} else {
		glog.Info("Starting Http Proxy")
		http.ListenAndServe(*httpPortFlag, topMux)
	}

	glog.Info("Http Proxy started")
}

func startWebhook(client ctrlclient.Client, clientNoCahe ctrlclient.Client, wg *sync.WaitGroup) (*http.Server, error) {
	glog.Info("Starting the Kubernetes webhooks...")

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
		certPath, keyPath, useTLS, resolveErr := resolveWebhookTLSPaths(
			*webhookTLSCertPath, *webhookTLSKeyPath,
			*tlsCertPath, *tlsCertKeyPath,
			common.FileExists,
		)
		if resolveErr != nil {
			glog.Fatalf("Failed to resolve webhook TLS paths: %v", resolveErr)
			return
		}

		if !useTLS {
			glog.Warning("TLS certificate/key paths are not set. Starting webhook server without TLS.")
			err = webhookServer.ListenAndServe()
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				glog.Fatalf("Failed to start Kubernetes webhook server: %v", err)
			}
			return
		}

		glog.Info("Starting the Kubernetes webhook with TLS")
		err := webhookServer.ListenAndServeTLS(certPath, keyPath)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			glog.Fatalf("Failed to start the Kubernetes webhook with TLS: %v", err)
		}
	}()

	return webhookServer, nil
}

// resolveWebhookTLSPaths determines which TLS certificate and key paths to use for the
// webhook server. It checks webhook-specific paths first, then falls back to the API server's
// TLS paths. Returns the resolved cert/key paths, whether TLS should be used, and any error.
// The fileExists parameter allows injecting a file-existence checker for testability.
func resolveWebhookTLSPaths(webhookCert, webhookKey, serverCert, serverKey string, fileExists func(string) bool) (certPath, keyPath string, useTLS bool, err error) {
	// Use webhook-specific TLS cert/key if both are specified.
	if webhookCert != "" && webhookKey != "" {
		if !fileExists(webhookCert) || !fileExists(webhookKey) {
			return "", "", false, fmt.Errorf("webhook TLS certificate/key paths are set but files do not exist (cert: %q, key: %q)", webhookCert, webhookKey)
		}
		return webhookCert, webhookKey, true, nil
	}

	// Fall back to the API server's TLS cert/key if both are specified.
	if serverCert != "" && serverKey != "" {
		if !fileExists(serverCert) || !fileExists(serverKey) {
			return "", "", false, fmt.Errorf("API server TLS certificate/key paths are set but files do not exist (cert: %q, key: %q)", serverCert, serverKey)
		}
		return serverCert, serverKey, true, nil
	}

	// No TLS paths configured at all.
	return "", "", false, nil
}

func registerHTTPHandlerFromEndpoint(ctx context.Context, handler RegisterHttpHandlerFromEndpoint, serviceName string, mux *runtime.ServeMux, tlsCfg *tls.Config) error {
	endpoint := "localhost" + *rpcPortFlag
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(math.MaxInt32)),
	}
	if tlsCfg != nil {
		opts = []grpc.DialOption{
			grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg)),
			grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(math.MaxInt32)),
		}
	}
	if err := handler(ctx, mux, endpoint, opts); err != nil {
		return fmt.Errorf("failed to register %v handler: %w", serviceName, err)
	}
	return nil
}

func initConfig() error {
	// Import environment variable, support nested vars e.g. OBJECTSTORECONFIG_ACCESSKEY
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)
	viper.AutomaticEnv()
	// We need empty string env var for e.g. KUBEFLOW_USERID_PREFIX.
	viper.AllowEmptyEnv(true)

	// Set configuration file name. The format is auto detected in this case.
	viper.SetConfigName("config")
	viper.AddConfigPath(*configPath)
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("config file error: %w", err)
	}

	// Watch for configuration change
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		if err := viper.ReadInConfig(); err != nil {
			glog.Errorf("Failed to reload config: %v", err)
		}
	})

	proxy.InitializeConfigWithEnv()
	return nil
}

// parseOptionalInt64 parses a string to *int64. Returns nil if the string is empty
// or consists only of whitespace. Negative values are rejected since they are
// invalid for Kubernetes security context fields like runAsUser/runAsGroup.
func parseOptionalInt64(s string) *int64 {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return nil
	}
	v, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil {
		glog.Errorf("Failed to parse %q as int64: %v", s, err)
		return nil
	}
	if v < 0 {
		glog.Errorf("Invalid value %d: negative values are not allowed", v)
		return nil
	}
	return &v
}

// parseOptionalBool parses a string to *bool. Returns nil if the string is empty
// or consists only of whitespace. Accepts "true" and "false" (case-insensitive).
func parseOptionalBool(s string) *bool {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return nil
	}
	v, err := strconv.ParseBool(trimmed)
	if err != nil {
		glog.Errorf("Failed to parse %q as bool: %v", s, err)
		return nil
	}
	return &v
}

// getPVCSpec retrieves the default workspace PersistentVolumeClaimSpec from the config.
// This default is used for workspace PVCs when users do not specify their own configuration.
func getPVCSpec() (*corev1.PersistentVolumeClaimSpec, error) {
	workspaceConfig := viper.Sub(workspaceConfig)
	if workspaceConfig == nil {
		glog.Info("No workspace config found; proceeding without a default PVC spec")
		return nil, nil
	}
	var pvcSpec corev1.PersistentVolumeClaimSpec
	if err := workspaceConfig.UnmarshalKey("volumeclaimtemplatespec", &pvcSpec); err != nil {
		return nil, fmt.Errorf("failed to unmarshal workspace.volumeclaimtemplatespec: %w", err)
	}
	if len(pvcSpec.AccessModes) == 0 || pvcSpec.StorageClassName == nil || *pvcSpec.StorageClassName == "" {
		return nil, fmt.Errorf("invalid workspace.volumeclaimtemplatespec: must specify accessModes and storageClassName")
	}

	return &pvcSpec, nil
}
