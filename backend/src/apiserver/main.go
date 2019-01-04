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
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"fmt"
	"os"

	"github.com/golang/glog"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/apiserver/server"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
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
	glog.Infof("starting API server")

	initConfig()
	clientManager := newClientManager()
	resourceManager := resource.NewResourceManager(&clientManager)
	err := loadSamples(resourceManager)
	if err != nil {
		glog.Fatalf("Failed to load samples. Err: %v", err.Error())
	}

	go startRpcServer(resourceManager)
	startHttpProxy(resourceManager)

	clientManager.Close()
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
	mux := runtime.NewServeMux()
	registerHttpHandlerFromEndpoint(api.RegisterPipelineServiceHandlerFromEndpoint, "PipelineService", ctx, mux)
	registerHttpHandlerFromEndpoint(api.RegisterExperimentServiceHandlerFromEndpoint, "ExperimentService", ctx, mux)
	registerHttpHandlerFromEndpoint(api.RegisterJobServiceHandlerFromEndpoint, "JobService", ctx, mux)
	registerHttpHandlerFromEndpoint(api.RegisterRunServiceHandlerFromEndpoint, "RunService", ctx, mux)
	registerHttpHandlerFromEndpoint(api.RegisterReportServiceHandlerFromEndpoint, "ReportService", ctx, mux)

	// Create a top level mux to include both pipeline upload server and gRPC servers.
	topMux := http.NewServeMux()

	// multipart upload is only supported in HTTP. In long term, we should have gRPC endpoints that
	// accept pipeline url for importing.
	// https://github.com/grpc-ecosystem/grpc-gateway/issues/410
	pipelineUploadServer := server.NewPipelineUploadServer(resourceManager)
	topMux.HandleFunc("/apis/v1beta1/pipelines/upload", pipelineUploadServer.UploadPipeline)
	topMux.HandleFunc("/apis/v1beta1/healthz", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"commit_sha":"`+getStringConfig("COMMIT_SHA")+`"}`)
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
		return errors.New(fmt.Sprintf("Failed to read sample configurations file. Err: %v", err.Error()))
	}
	type config struct {
		Name        string
		Description string
		File        string
	}
	var configs []config
	if err := json.Unmarshal(configBytes, &configs); err != nil {
		return errors.New(fmt.Sprintf("Failed to read sample configurations. Err: %v", err.Error()))
	}
	for _, config := range configs {
		reader, err := os.Open(config.File)
		if err != nil {
			return errors.New(fmt.Sprintf("Failed to load sample %s. Error: %v", config.Name, err.Error()))
		}
		pipelineFile, err := server.ReadPipelineFile(config.File, reader, server.MaxFileLength)
		if err != nil {
			return errors.New(fmt.Sprintf("Failed to decompress the file %s. Error: %v", config.Name, err.Error()))
		}
		_, err = resourceManager.CreatePipeline(config.Name, config.Description, pipelineFile)
		if err != nil {
			// Log the error but not fail. The API Server pod can restart and it could potentially cause name collision.
			// In the future, we might consider loading samples during deployment, instead of when API server starts.
			glog.Warningf(fmt.Sprintf("Failed to create pipeline for %s. Error: %v", config.Name, err.Error()))
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
