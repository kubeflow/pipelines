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
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"google.golang.org/grpc/credentials"

	"github.com/fsnotify/fsnotify"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	apiv1beta1 "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	cm "github.com/kubeflow/pipelines/backend/src/apiserver/client_manager"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/apiserver/server"
	"github.com/kubeflow/pipelines/backend/src/apiserver/template"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	executionTypeEnv = "ExecutionType"
	launcherEnv      = "Launcher"
)

var (
	logLevelFlag       = flag.String("logLevel", "", "Defines the log level for the application.")
	rpcPortFlag        = flag.String("rpcPortFlag", ":8887", "RPC Port")
	httpPortFlag       = flag.String("httpPortFlag", ":8888", "Http Proxy Port")
	configPath         = flag.String("config", "", "Path to JSON file containing config")
	sampleConfigPath   = flag.String("sampleconfig", "", "Path to samples")
	tlsCertPath        = flag.String("tlsCertPath", "", "Path to the public tls cert.")
	tlsCertKeyPath     = flag.String("tlsCertKeyPath", "", "Path to the private tls key cert.")
	collectMetricsFlag = flag.Bool("collectMetricsFlag", true, "Whether to collect Prometheus metrics in API server.")
)

type RegisterHttpHandlerFromEndpoint func(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error

func initCerts() (*tls.Config, error) {
	if *tlsCertPath == "" && *tlsCertKeyPath == "" {
		// User can choose not to provide certs
		return nil, nil
	} else if *tlsCertPath == "" {
		return nil, fmt.Errorf("Missing tlsCertPath when specifying cert paths, both tlsCertPath and tlsCertKeyPath are required.")
	} else if *tlsCertKeyPath == "" {
		return nil, fmt.Errorf("Missing tlsCertKeyPath when specifying cert paths, both tlsCertPath and tlsCertKeyPath are required.")
	}
	serverCert, err := tls.LoadX509KeyPair(*tlsCertPath, *tlsCertKeyPath)
	if err != nil {
		return nil, err
	}
	config := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
	}
	glog.Info("TLS cert key/pair loaded.")
	return config, err
}

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

	clientManager := cm.NewClientManager()

	tlsConfig, err := initCerts()
	if err != nil {
		glog.Fatalf("Failed to parse Cert paths. Err: %v", err)
	}

	resourceManager := resource.NewResourceManager(
		&clientManager,
		&resource.ResourceManagerOptions{CollectMetrics: *collectMetricsFlag},
	)
	err = loadSamples(resourceManager)
	if err != nil {
		glog.Infof("Couldn't load optional pipeline samples: %v", err)
	}

	if !common.IsMultiUserMode() {
		_, err = resourceManager.CreateDefaultExperiment("")
		if err != nil {
			glog.Infof("Couldn't create optional default experiment: %v", err)
		}
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

	go startRpcServer(resourceManager, tlsConfig)
	startHttpProxy(resourceManager, tlsConfig)

	clientManager.Close()
}

// A custom http request header matcher to pass on the user identity
// Reference: https://github.com/grpc-ecosystem/grpc-gateway/blob/master/docs/_docs/customizingyourgateway.md#mapping-from-http-request-headers-to-grpc-client-metadata
func grpcCustomMatcher(key string) (string, bool) {
	if strings.EqualFold(key, common.GetKubeflowUserIDHeader()) {
		return strings.ToLower(key), true
	}
	return strings.ToLower(key), false
}

func startRpcServer(resourceManager *resource.ResourceManager, tlsConfig *tls.Config) {
	var s *grpc.Server
	if tlsConfig != nil {
		glog.Info("Starting RPC server (TLS enabled)")
		tlsCredentials := credentials.NewTLS(tlsConfig)
		s = grpc.NewServer(
			grpc.Creds(tlsCredentials),
			grpc.UnaryInterceptor(apiServerInterceptor),
			grpc.MaxRecvMsgSize(math.MaxInt32),
		)
	} else {
		glog.Info("Starting RPC server")
		s = grpc.NewServer(grpc.UnaryInterceptor(apiServerInterceptor), grpc.MaxRecvMsgSize(math.MaxInt32))
	}

	listener, err := net.Listen("tcp", *rpcPortFlag)
	if err != nil {
		glog.Fatalf("Failed to start RPC server: %v", err)
	}

	sharedExperimentServer := server.NewExperimentServer(resourceManager, &server.ExperimentServerOptions{CollectMetrics: *collectMetricsFlag})
	sharedPipelineServer := server.NewPipelineServer(
		resourceManager,
		&server.PipelineServerOptions{
			CollectMetrics: *collectMetricsFlag,
		},
	)
	sharedJobServer := server.NewJobServer(resourceManager, &server.JobServerOptions{CollectMetrics: *collectMetricsFlag})
	sharedRunServer := server.NewRunServer(resourceManager, &server.RunServerOptions{CollectMetrics: *collectMetricsFlag})
	sharedArtifactServer := server.NewArtifactServer(resourceManager, &server.ArtifactServerOptions{CollectMetrics: *collectMetricsFlag})
	apiv1beta1.RegisterExperimentServiceServer(s, sharedExperimentServer)
	apiv2beta1.RegisterArtifactServiceServer(s, sharedArtifactServer)
	apiv1beta1.RegisterPipelineServiceServer(s, sharedPipelineServer)
	apiv1beta1.RegisterJobServiceServer(s, sharedJobServer)
	apiv1beta1.RegisterRunServiceServer(s, sharedRunServer)
	apiv1beta1.RegisterTaskServiceServer(s, server.NewTaskServer(resourceManager))
	apiv1beta1.RegisterReportServiceServer(s, server.NewReportServer(resourceManager))

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
	apiv2beta1.RegisterReportServiceServer(s, server.NewReportServer(resourceManager))

	// Register reflection service on gRPC server.
	reflection.Register(s)
	if err := s.Serve(listener); err != nil {
		glog.Fatalf("Failed to serve rpc listener: %v", err)
	}
	glog.Info("RPC server started")
}

func startHttpProxy(resourceManager *resource.ResourceManager, tlsConfig *tls.Config) {

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Create gRPC HTTP MUX and register services for v1beta1 api.
	runtimeMux := runtime.NewServeMux(runtime.WithIncomingHeaderMatcher(grpcCustomMatcher))
	registerHttpHandlerFromEndpoint(apiv1beta1.RegisterPipelineServiceHandlerFromEndpoint, "PipelineService", ctx, runtimeMux, tlsConfig)
	registerHttpHandlerFromEndpoint(apiv1beta1.RegisterExperimentServiceHandlerFromEndpoint, "ExperimentService", ctx, runtimeMux, tlsConfig)
	registerHttpHandlerFromEndpoint(apiv1beta1.RegisterJobServiceHandlerFromEndpoint, "JobService", ctx, runtimeMux, tlsConfig)
	registerHttpHandlerFromEndpoint(apiv1beta1.RegisterRunServiceHandlerFromEndpoint, "RunService", ctx, runtimeMux, tlsConfig)
	registerHttpHandlerFromEndpoint(apiv1beta1.RegisterTaskServiceHandlerFromEndpoint, "TaskService", ctx, runtimeMux, tlsConfig)
	registerHttpHandlerFromEndpoint(apiv1beta1.RegisterReportServiceHandlerFromEndpoint, "ReportService", ctx, runtimeMux, tlsConfig)
	registerHttpHandlerFromEndpoint(apiv1beta1.RegisterVisualizationServiceHandlerFromEndpoint, "Visualization", ctx, runtimeMux, tlsConfig)
	registerHttpHandlerFromEndpoint(apiv1beta1.RegisterAuthServiceHandlerFromEndpoint, "AuthService", ctx, runtimeMux, tlsConfig)

	// Create gRPC HTTP MUX and register services for v2beta1 api.
	registerHttpHandlerFromEndpoint(apiv2beta1.RegisterExperimentServiceHandlerFromEndpoint, "ExperimentService", ctx, runtimeMux, tlsConfig)
	registerHttpHandlerFromEndpoint(apiv2beta1.RegisterPipelineServiceHandlerFromEndpoint, "PipelineService", ctx, runtimeMux, tlsConfig)
	registerHttpHandlerFromEndpoint(apiv2beta1.RegisterRecurringRunServiceHandlerFromEndpoint, "RecurringRunService", ctx, runtimeMux, tlsConfig)
	registerHttpHandlerFromEndpoint(apiv2beta1.RegisterRunServiceHandlerFromEndpoint, "RunService", ctx, runtimeMux, tlsConfig)
	registerHttpHandlerFromEndpoint(apiv2beta1.RegisterArtifactServiceHandlerFromEndpoint, "ArtifactService", ctx, runtimeMux, tlsConfig)

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
		io.WriteString(w, `{"commit_sha":"`+common.GetStringConfigWithDefault("COMMIT_SHA", "unknown")+`", "tag_name":"`+common.GetStringConfigWithDefault("TAG_NAME", "unknown")+`", "multi_user":`+strconv.FormatBool(common.IsMultiUserMode())+`}`)
	})

	// log streaming is provided via HTTP.
	runLogServer := server.NewRunLogServer(resourceManager)
	topMux.HandleFunc("/apis/v1alpha1/runs/{run_id}/nodes/{node_id}/log", runLogServer.ReadRunLogV1)

	topMux.PathPrefix("/apis/").Handler(runtimeMux)

	// Register a handler for Prometheus to poll.
	topMux.Handle("/metrics", promhttp.Handler())

	if tlsConfig != nil {
		glog.Info("Starting Https Proxy")
		https := http.Server{
			TLSConfig: tlsConfig,
			Addr:      *httpPortFlag,
			Handler:   topMux,
		}
		https.ListenAndServeTLS("", "")
	} else {
		glog.Info("Starting Http Proxy")
		http.ListenAndServe(*httpPortFlag, topMux)
	}

	glog.Info("Http Proxy started")
}

func registerHttpHandlerFromEndpoint(handler RegisterHttpHandlerFromEndpoint, serviceName string, ctx context.Context, mux *runtime.ServeMux, tlsConfig *tls.Config) {
	endpoint := "localhost" + *rpcPortFlag
	var opts []grpc.DialOption
	if tlsConfig != nil {
		// local client connections via http proxy to grpc should not require tls
		tlsConfig.InsecureSkipVerify = true
		opts = []grpc.DialOption{
			grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
			grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(math.MaxInt32)),
		}
	} else {
		opts = []grpc.DialOption{grpc.WithInsecure(), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(math.MaxInt32))}
	}

	if err := handler(ctx, mux, endpoint, opts); err != nil {
		glog.Fatalf("Failed to register %v handler: %v", serviceName, err)
	}
}

// Preload a bunch of pipeline samples
// Samples are only loaded once when the pipeline system is initially installed.
// They won't be loaded when upgrade or pod restart, to prevent them reappear if user explicitly
// delete the samples.
func loadSamples(resourceManager *resource.ResourceManager) error {
	// Check if sample has being loaded already and skip loading if true.
	haveSamplesLoaded, err := resourceManager.HaveSamplesLoaded()
	if err != nil {
		return err
	}
	if haveSamplesLoaded {
		glog.Infof("Samples already loaded in the past. Skip loading.")
		return nil
	}

	pathExists, err := client.PathExists(*sampleConfigPath)
	if err != nil {
		return err
	}

	if !pathExists {
		glog.Infof("No samples path provided, skipping loading samples..")
		return nil
	}

	configBytes, err := ioutil.ReadFile(*sampleConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read sample configurations file. Err: %v", err)
	}
	type config struct {
		Name        string
		Description string
		File        string
	}
	var configs []config
	if err = json.Unmarshal(configBytes, &configs); err != nil {
		return fmt.Errorf("failed to read sample configurations. Err: %v", err)
	}
	for _, config := range configs {
		reader, configErr := os.Open(config.File)
		if configErr != nil {
			return fmt.Errorf("failed to load sample %s. Error: %v", config.Name, configErr)
		}
		pipelineFile, configErr := server.ReadPipelineFile(config.File, reader, common.MaxFileLength)
		if configErr != nil {
			return fmt.Errorf("failed to decompress the file %s. Error: %v", config.Name, configErr)
		}
		p, configErr := resourceManager.CreatePipeline(
			&model.Pipeline{
				Name:        config.Name,
				Description: config.Description,
			},
		)
		if configErr != nil {
			// Log the error but not fail. The API Server pod can restart and it could potentially cause name collision.
			// In the future, we might consider loading samples during deployment, instead of when API server starts.
			glog.Warningf(fmt.Sprintf("Failed to create pipeline for %s. Error: %v", config.Name, configErr))

			continue
		}

		_, configErr = resourceManager.CreatePipelineVersion(
			&model.PipelineVersion{
				Name:         config.Name,
				Description:  config.Description,
				PipelineId:   p.UUID,
				PipelineSpec: string(pipelineFile),
			},
		)
		if configErr != nil {
			// Log the error but not fail. The API Server pod can restart and it could potentially cause name collision.
			// In the future, we might consider loading samples during deployment, instead of when API server starts.
			glog.Warningf(fmt.Sprintf("Failed to create pipeline for %s. Error: %v", config.Name, configErr))

			continue
		}
		// Since the default sorting is by create time,
		// Sleep one second makes sure the samples are showing up in the same order as they are added.
		time.Sleep(1 * time.Second)
	}
	// Mark sample as loaded
	err = resourceManager.MarkSampleLoaded()
	if err != nil {
		return err
	}
	glog.Info("All samples are loaded.")
	return nil
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
}
