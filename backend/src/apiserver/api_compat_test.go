// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0.

package main

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/mux"
	api "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	legacy "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// Sharing gRPC method handlers requires identical wire and JSON contracts. Any
// future incompatible evolution must add an explicit adapter, not weaken this guard.
func TestLegacyAPIContractParity(t *testing.T) {
	files := 0
	protoregistry.GlobalFiles.RangeFiles(func(file protoreflect.FileDescriptor) bool {
		if !strings.HasPrefix(file.Path(), "backend/api/v2beta1/") {
			return true
		}
		files++
		t.Run(file.Path(), func(t *testing.T) {
			canonicalPath := strings.Replace(file.Path(), "/v2beta1/", "/v2/", 1)
			canonical, err := protoregistry.GlobalFiles.FindFileByPath(canonicalPath)
			require.NoError(t, err)
			oldJSON, err := protojson.Marshal(protodesc.ToFileDescriptorProto(file))
			require.NoError(t, err)
			newJSON, err := protojson.Marshal(protodesc.ToFileDescriptorProto(canonical))
			require.NoError(t, err)
			require.JSONEq(t, strings.ReplaceAll(string(oldJSON), "v2beta1", "v2"), string(newJSON))
		})
		return true
	})
	require.Equal(t, 10, files)
}

func TestLegacyHTTPRoutesShareCanonicalHandlers(t *testing.T) {
	for _, endpoint := range []struct{ method, path string }{
		{http.MethodPost, "/pipelines/upload"},
		{http.MethodPost, "/pipelines/upload_version"},
		{http.MethodPost, "/transfer/import"},
		{http.MethodPost, "/transfer/export"},
		{http.MethodGet, "/runs/run/nodes/node/log"},
		{http.MethodGet, "/runs/run/nodes/node/artifacts/artifact:read"},
		{http.MethodPost, "/experiments"},
		{http.MethodGet, "/runs/run/tasks"},
		{http.MethodPost, "/runs/run:retry"},
		{http.MethodDelete, "/pipelines/pipeline"},
	} {
		t.Run(endpoint.path, func(t *testing.T) {
			for _, version := range []string{canonicalAPIPath, legacyAPIPath} {
				request := httptest.NewRequest(endpoint.method, version+endpoint.path+"?namespace=team&page_token=a%2Bb%3D", strings.NewReader("request body"))
				request.Header.Set("Authorization", "Bearer token")
				request.Header.Set("Kubeflow-Userid", "user@example.com")
				request.Header.Set("Content-Type", "application/octet-stream")
				called := false
				handler := func(w http.ResponseWriter, r *http.Request) {
					called = true
					require.Equal(t, canonicalAPIPath+endpoint.path, r.URL.Path)
					require.Equal(t, request.URL.RawQuery, r.URL.RawQuery)
					require.Equal(t, request.Header, r.Header)
					require.Equal(t, request.Method, r.Method)
					require.True(t, request.Body == r.Body, "forwarding must not replace the body stream")
					require.Equal(t, request.Context().Err(), r.Context().Err())
					if strings.Contains(endpoint.path, "/nodes/") {
						require.Equal(t, "run", mux.Vars(r)["run_id"])
						require.Equal(t, "node", mux.Vars(r)["node_id"])
					}
					w.Header().Set("Content-Type", "application/octet-stream")
					w.WriteHeader(http.StatusAccepted)
					w.(http.Flusher).Flush()
					_, err := io.Copy(w, r.Body)
					require.NoError(t, err)
				}
				deps := HTTPRouterDeps{ExportTransfer: handler, ImportTransfer: handler, UploadPipeline: handler, UploadPipelineVersion: handler, ReadRunLog: handler, ReadArtifact: handler}
				router := buildHTTPRouter(deps, http.HandlerFunc(handler), "database")
				response := httptest.NewRecorder()
				router.ServeHTTP(response, request)
				require.True(t, called)
				require.Equal(t, http.StatusAccepted, response.Code)
				require.Empty(t, response.Header().Get("Location"))
				require.True(t, response.Flushed)
				require.Equal(t, "request body", response.Body.String())
				require.Equal(t, version+endpoint.path, request.URL.Path)
			}
		})
	}
}

func TestLegacyHTTPGatewayMiddleware(t *testing.T) {
	for _, version := range []string{canonicalAPIPath, legacyAPIPath} {
		t.Run(version, func(t *testing.T) {
			called := false
			router := buildHTTPRouter(newNoOpHTTPRouterDeps(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				require.Equal(t, "true", r.Header.Get("x-clear-tags"))
			}), "database")
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPatch, version+"/pipelines/pipeline", strings.NewReader(`{"tags":{}}`))
			router.ServeHTTP(response, request)
			require.True(t, called)
			require.Equal(t, http.StatusOK, response.Code)
		})
	}
}

func TestLegacyHTTPBodyLimitAndRouteBoundaries(t *testing.T) {
	t.Setenv(common.MaxPipelineUpdateBodyBytesEnv, "5")
	for _, prefix := range []string{canonicalAPIPath, legacyAPIPath} {
		router := buildHTTPRouter(newNoOpHTTPRouterDeps(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("oversized update reached the gateway")
		}), "database")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodPatch, prefix+"/pipelines/pipeline", strings.NewReader(`{"tags":{}}`)))
		require.Equal(t, http.StatusRequestEntityTooLarge, response.Code)
	}
	deps := newNoOpHTTPRouterDeps()
	deps.ReadRunLog = func(http.ResponseWriter, *http.Request) { t.Error("POST reached the streaming handler") }
	router := buildHTTPRouter(deps, http.NotFoundHandler(), "database")
	for _, path := range []string{legacyAPIPath + "/runs/run/nodes/node/log", "/apis/v2beta10/healthz"} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, path, nil))
		require.Equal(t, http.StatusNotFound, response.Code)
	}
	canonicalHealth, legacyHealth := httptest.NewRecorder(), httptest.NewRecorder()
	router.ServeHTTP(canonicalHealth, httptest.NewRequest(http.MethodGet, canonicalAPIPath+"/healthz", nil))
	router.ServeHTTP(legacyHealth, httptest.NewRequest(http.MethodGet, legacyAPIPath+"/healthz", nil))
	require.Equal(t, http.StatusOK, legacyHealth.Code)
	require.JSONEq(t, canonicalHealth.Body.String(), legacyHealth.Body.String())
}

func TestLegacyHTTPRewritePreservesEscapingAndCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	request := httptest.NewRequest(http.MethodGet, legacyAPIPath+"/pipelines/a%2Fb?filter=v2beta1", nil).WithContext(ctx)
	legacyAPIHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, canonicalAPIPath+"/pipelines/a%2Fb", r.URL.EscapedPath())
		require.Equal(t, "filter=v2beta1", r.URL.RawQuery)
		require.ErrorIs(t, r.Context().Err(), context.Canceled)
	})).ServeHTTP(httptest.NewRecorder(), request)
	require.Equal(t, legacyAPIPath+"/pipelines/a%2Fb", request.URL.EscapedPath())
}

type compatibilityExperimentServer struct {
	api.UnimplementedExperimentServiceServer
}

func (compatibilityExperimentServer) CreateExperiment(ctx context.Context, request *api.CreateExperimentRequest) (*api.Experiment, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	if len(md.Get("authorization")) == 0 {
		return nil, status.Error(codes.Unauthenticated, "authorization required")
	}
	if request.Experiment.GetDisplayName() == "" {
		return nil, status.Error(codes.InvalidArgument, "display name required")
	}
	if err := grpc.SetHeader(ctx, metadata.Pairs("compatibility", "shared-handler")); err != nil {
		return nil, err
	}
	return request.Experiment, nil
}

func TestLegacyGRPCClientUsesCanonicalHandler(t *testing.T) {
	var intercepted atomic.Int32
	rpc := grpc.NewServer(grpc.UnaryInterceptor(func(ctx context.Context, request interface{}, info *grpc.UnaryServerInfo, next grpc.UnaryHandler) (interface{}, error) {
		intercepted.Add(1)
		require.Equal(t, "/"+canonicalRPCPackage+"ExperimentService/CreateExperiment", info.FullMethod)
		_, ok := request.(*api.CreateExperimentRequest)
		require.True(t, ok, "legacy bytes must decode into the canonical request")
		return next(ctx, request)
	}))
	api.RegisterExperimentServiceServer(compatibleServiceRegistrar{rpc}, compatibilityExperimentServer{})
	listener := bufconn.Listen(1024 * 1024)
	go rpc.Serve(listener)
	t.Cleanup(rpc.Stop)
	connection, err := grpc.NewClient("passthrough:///compatibility", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }))
	require.NoError(t, err)
	t.Cleanup(func() { connection.Close() })
	deadline, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	ctx := metadata.NewOutgoingContext(deadline, metadata.Pairs("authorization", "Bearer token"))
	oldClient := legacy.NewExperimentServiceClient(connection)
	newClient := api.NewExperimentServiceClient(connection)
	var headers metadata.MD
	oldResponse, err := oldClient.CreateExperiment(ctx, &legacy.CreateExperimentRequest{Experiment: &legacy.Experiment{DisplayName: "experiment", Namespace: "tenant"}}, grpc.Header(&headers))
	require.NoError(t, err)
	require.Equal(t, []string{"shared-handler"}, headers.Get("compatibility"))
	newResponse, err := newClient.CreateExperiment(ctx, &api.CreateExperimentRequest{Experiment: &api.Experiment{DisplayName: "experiment", Namespace: "tenant"}})
	require.NoError(t, err)
	oldJSON, err := protojson.Marshal(oldResponse)
	require.NoError(t, err)
	newJSON, err := protojson.Marshal(newResponse)
	require.NoError(t, err)
	require.JSONEq(t, string(newJSON), string(oldJSON))
	_, err = oldClient.CreateExperiment(ctx, &legacy.CreateExperimentRequest{})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	_, err = oldClient.CreateExperiment(deadline, &legacy.CreateExperimentRequest{})
	require.Equal(t, codes.Unauthenticated, status.Code(err))
	require.Equal(t, int32(4), intercepted.Load())
}
