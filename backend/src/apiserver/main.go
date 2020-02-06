// Copyright 2018 Google LLC
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
	"encoding/json"
	"flag"
	"github.com/fsnotify/fsnotify"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/spf13/viper"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	"fmt"
	"os"

	"github.com/golang/glog"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/apiserver/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	HasDefaultBucketEnvVar  = "HAS_DEFAULT_BUCKET"
	ProjectIDEnvVar         = "PROJECT_ID"
	DefaultBucketNameEnvVar = "BUCKET_NAME"
)

var (
	rpcPortFlag      = flag.String("rpcPortFlag", ":8887", "RPC Port")
	httpPortFlag     = flag.String("httpPortFlag", ":8888", "Http Proxy Port")
	configPath       = flag.String("config", "", "Path to JSON file containing config")
	sampleConfigPath = flag.String("sampleconfig", "", "Path to samples")
)

type RegisterHttpHandlerFromEndpoint func(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error

func main() {
	flag.Parse()

	initConfig()
	clientManager := newClientManager()
	resourceManager := resource.NewResourceManager(&clientManager)
	err := loadSamples(resourceManager)
	if err != nil {
		glog.Fatalf("Failed to load samples. Err: %v", err)
	}

	_, err = resourceManager.CreateDefaultExperiment()
	if err != nil {
		glog.Fatalf("Failed to create default experiment. Err: %v", err)
	}

	go startRpcServer(resourceManager)
	startHttpProxy(resourceManager)

	clientManager.Close()
}

// A custom http request header matcher to pass on the user identity
// Reference: https://github.com/grpc-ecosystem/grpc-gateway/blob/master/docs/_docs/customizingyourgateway.md#mapping-from-http-request-headers-to-grpc-client-metadata
func grpcCustomMatcher(key string) (string, bool) {
	switch strings.ToLower(key) {
	case common.GoogleIAPUserIdentityHeader:
		return strings.ToLower(key), true
	default:
		return strings.ToLower(key), false
	}
}

func startRpcServer(resourceManager *resource.ResourceManager) {
	glog.Info("Starting RPC server")
	listener, err := net.Listen("tcp", *rpcPortFlag)
	if err != nil {
		glog.Fatalf("Failed to start RPC server: %v", err)
	}
	s := grpc.NewServer(grpc.UnaryInterceptor(apiServerInterceptor))
	api.RegisterPipelineServiceServer(s, server.NewPipelineServer(resourceManager))
	api.RegisterExperimentServiceServer(s, server.NewExperimentServer(resourceManager))
	api.RegisterRunServiceServer(s, server.NewRunServer(resourceManager))
	api.RegisterJobServiceServer(s, server.NewJobServer(resourceManager))
	api.RegisterReportServiceServer(s, server.NewReportServer(resourceManager))
	api.RegisterVisualizationServiceServer(
		s,
		server.NewVisualizationServer(
			resourceManager,
			common.GetStringConfig(visualizationServiceHost),
			common.GetStringConfig(visualizationServicePort),
		))

	// Register reflection service on gRPC server.
	reflection.Register(s)
	if err := s.Serve(listener); err != nil {
		glog.Fatalf("Failed to serve rpc listener: %v", err)
	}
	glog.Info("RPC server started")
}

func startHttpProxy(resourceManager *resource.ResourceManager) {
	glog.Info("Starting Http Proxy")

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Create gRPC HTTP MUX and register services.
	mux := runtime.NewServeMux(runtime.WithIncomingHeaderMatcher(grpcCustomMatcher))
	registerHttpHandlerFromEndpoint(api.RegisterPipelineServiceHandlerFromEndpoint, "PipelineService", ctx, mux)
	registerHttpHandlerFromEndpoint(api.RegisterExperimentServiceHandlerFromEndpoint, "ExperimentService", ctx, mux)
	registerHttpHandlerFromEndpoint(api.RegisterJobServiceHandlerFromEndpoint, "JobService", ctx, mux)
	registerHttpHandlerFromEndpoint(api.RegisterRunServiceHandlerFromEndpoint, "RunService", ctx, mux)
	registerHttpHandlerFromEndpoint(api.RegisterReportServiceHandlerFromEndpoint, "ReportService", ctx, mux)
	registerHttpHandlerFromEndpoint(api.RegisterVisualizationServiceHandlerFromEndpoint, "Visualization", ctx, mux)

	// Create a top level mux to include both pipeline upload server and gRPC servers.
	topMux := http.NewServeMux()

	// multipart upload is only supported in HTTP. In long term, we should have gRPC endpoints that
	// accept pipeline url for importing.
	// https://github.com/grpc-ecosystem/grpc-gateway/issues/410
	pipelineUploadServer := server.NewPipelineUploadServer(resourceManager)
	topMux.HandleFunc("/apis/v1beta1/pipelines/upload", pipelineUploadServer.UploadPipeline)
	topMux.HandleFunc("/apis/v1beta1/healthz", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"commit_sha":"`+common.GetStringConfig("COMMIT_SHA")+`"}`)
	})

	topMux.Handle("/apis/", mux)

	http.ListenAndServe(*httpPortFlag, topMux)
	glog.Info("Http Proxy started")
}

func registerHttpHandlerFromEndpoint(handler RegisterHttpHandlerFromEndpoint, serviceName string, ctx context.Context, mux *runtime.ServeMux) {
	endpoint := "localhost" + *rpcPortFlag
	opts := []grpc.DialOption{grpc.WithInsecure()}

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
	configBytes, err := ioutil.ReadFile(*sampleConfigPath)
	if err != nil {
		return fmt.Errorf("Failed to read sample configurations file. Err: %v", err)
	}
	type config struct {
		Name        string
		Description string
		File        string
	}
	var configs []config
	if err = json.Unmarshal(configBytes, &configs); err != nil {
		return fmt.Errorf("Failed to read sample configurations. Err: %v", err)
	}
	for _, config := range configs {
		reader, configErr := os.Open(config.File)
		if configErr != nil {
			return fmt.Errorf("Failed to load sample %s. Error: %v", config.Name, configErr)
		}
		pipelineFile, configErr := server.ReadPipelineFile(config.File, reader, server.MaxFileLength)
		if configErr != nil {
			return fmt.Errorf("Failed to decompress the file %s. Error: %v", config.Name, configErr)
		}
		// Patch the default bucket name read from ConfigMap
		if common.GetBoolConfigWithDefault(HasDefaultBucketEnvVar, false) {
			defaultBucket := common.GetStringConfig(DefaultBucketNameEnvVar)
			projectId := common.GetStringConfig(ProjectIDEnvVar)
			patchMap := map[string]string{
				"<your-gcs-bucket>": defaultBucket,
				"<your-project-id>": projectId,
			}
			pipelineFile, err = server.PatchPipelineDefaultParameter(pipelineFile, patchMap)
			if err != nil {
				return fmt.Errorf("Failed to patch default value to %s. Error: %v", config.Name, err)
			}
		}
		_, configErr = resourceManager.CreatePipeline(config.Name, config.Description, pipelineFile)
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
