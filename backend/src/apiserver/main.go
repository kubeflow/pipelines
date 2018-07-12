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
	"flag"
	"log"
	"net"
	"net/http"

	"github.com/golang/glog"
	"github.com/googleprivate/ml/backend/api"
	"github.com/googleprivate/ml/backend/src/apiserver/resource"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var (
	rpcPortFlag  = flag.String("rpcPortFlag", ":8887", "RPC Port")
	httpPortFlag = flag.String("httpPortFlag", ":8888", "Http Proxy Port")
	configPath   = flag.String("config", "", "Path to JSON file containing config")
)

type RegisterHttpHandlerFromEndpoint func(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error

func main() {
	flag.Parse()
	glog.Infof("starting API server")

	initConfig()
	clientManager := newClientManager()
	resourceManager := resource.NewResourceManager(&clientManager)
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
	server := grpc.NewServer(grpc.UnaryInterceptor(apiServerInterceptor))
	api.RegisterPackageServiceServer(server, &PackageServer{resourceManager})
	api.RegisterPipelineServiceServer(server, &PipelineServer{resourceManager})
	api.RegisterJobServiceServer(server, &JobServer{resourceManager})
	api.RegisterJobServiceV2Server(server, &JobServerV2{resourceManager})
	api.RegisterPipelineServiceV2Server(server, &PipelineServerV2{resourceManager})
	api.RegisterReportServiceServer(server, &ReportServer{resourceManager})

	// Register reflection service on gRPC server.
	reflection.Register(server)
	if err := server.Serve(listener); err != nil {
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
	registerHttpHandlerFromEndpoint(api.RegisterPackageServiceHandlerFromEndpoint, "PackageService", ctx, mux)
	registerHttpHandlerFromEndpoint(api.RegisterPipelineServiceHandlerFromEndpoint, "PipelineService", ctx, mux)
	registerHttpHandlerFromEndpoint(api.RegisterJobServiceHandlerFromEndpoint, "JobService", ctx, mux)
	registerHttpHandlerFromEndpoint(api.RegisterPipelineServiceV2HandlerFromEndpoint, "PipelineServiceV2", ctx, mux)
	registerHttpHandlerFromEndpoint(api.RegisterJobServiceV2HandlerFromEndpoint, "JobServiceV2", ctx, mux)
	registerHttpHandlerFromEndpoint(api.RegisterReportServiceHandlerFromEndpoint, "ReportService", ctx, mux)

	// Create a top level mux to include both package upload server and gRPC servers.
	topMux := http.NewServeMux()

	// multipart upload is only supported in HTTP. In long term, we should have gRPC endpoints that
	// accept package url for importing.
	// https://github.com/grpc-ecosystem/grpc-gateway/issues/410
	packageUploadServer := &PackageUploadServer{resourceManager: resourceManager}
	topMux.HandleFunc("/apis/v1alpha1/packages/upload", packageUploadServer.UploadPackage)
	topMux.HandleFunc("/apis/v1alpha1/healthz", func(w http.ResponseWriter, r *http.Request) {})

	topMux.Handle("/apis/", mux)

	http.ListenAndServe(*httpPortFlag, topMux)
	glog.Info("Http Proxy started")
}

func registerHttpHandlerFromEndpoint(handler RegisterHttpHandlerFromEndpoint, serviceName string, ctx context.Context, mux *runtime.ServeMux) {
	endpoint := "localhost" + *rpcPortFlag
	opts := []grpc.DialOption{grpc.WithInsecure()}

	if err := handler(ctx, mux, endpoint, opts); err != nil {
		log.Fatalf("Failed to register %v handler: %v", serviceName, err)
	}
}
