// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0.

package main

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/mux"
	api "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	legacy "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	promtestutil "github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	reflectionv1 "google.golang.org/grpc/reflection/grpc_reflection_v1"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

type recordingCompatibilityRegistrar struct {
	descriptors []*grpc.ServiceDesc
}

func (r *recordingCompatibilityRegistrar) RegisterService(desc *grpc.ServiceDesc, _ any) {
	r.descriptors = append(r.descriptors, desc)
}

func TestCompatibleRegistrarPassesOtherServicesOnce(t *testing.T) {
	for _, name := range []string{"grpc.health.v1.Health", "grpc.reflection.v1.ServerReflection", "other." + canonicalRPCPackage + "Service"} {
		t.Run(name, func(t *testing.T) {
			recorder := new(recordingCompatibilityRegistrar)
			compatibleServiceRegistrar{recorder}.RegisterService(&grpc.ServiceDesc{ServiceName: name}, nil)
			require.Len(t, recorder.descriptors, 1)
			require.Equal(t, name, recorder.descriptors[0].ServiceName)
		})
	}
}

func TestCompatibleRegistrarDoesNotMutateCanonicalHandlers(t *testing.T) {
	recorder := new(recordingCompatibilityRegistrar)
	descriptor := &api.ExperimentService_ServiceDesc
	original := reflect.ValueOf(descriptor.Methods[0].Handler).Pointer()
	compatibleServiceRegistrar{recorder}.RegisterService(descriptor, nil)
	require.Len(t, recorder.descriptors, 2)
	require.Equal(t, original, reflect.ValueOf(descriptor.Methods[0].Handler).Pointer())
	require.Equal(t, canonicalRPCPackage+"ExperimentService", recorder.descriptors[0].ServiceName)
	require.Equal(t, legacyRPCPackage+"ExperimentService", recorder.descriptors[1].ServiceName)
}

func TestLegacyUsageMetricsDistinguishCanonicalRequests(t *testing.T) {
	httpBefore := promtestutil.ToFloat64(legacyAPIRequests.WithLabelValues("http"))
	router := buildHTTPRouter(newNoOpHTTPRouterDeps(), http.NotFoundHandler(), "database")
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, canonicalAPIPath+"/healthz", nil))
	require.Equal(t, httpBefore, promtestutil.ToFloat64(legacyAPIRequests.WithLabelValues("http")))
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, legacyAPIPath+"/healthz", nil))
	require.Equal(t, httpBefore+1, promtestutil.ToFloat64(legacyAPIRequests.WithLabelValues("http")))

	connection, _ := compatibilityRPCConnection(t)
	grpcBefore := promtestutil.ToFloat64(legacyAPIRequests.WithLabelValues("grpc"))
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("authorization", "Bearer token"))
	_, err := api.NewExperimentServiceClient(connection).CreateExperiment(ctx, &api.CreateExperimentRequest{Experiment: &api.Experiment{DisplayName: "metrics"}})
	require.NoError(t, err)
	require.Equal(t, grpcBefore, promtestutil.ToFloat64(legacyAPIRequests.WithLabelValues("grpc")))
	err = connection.Invoke(ctx, "/"+legacyRPCPackage+"ExperimentService/CreateExperiment", legacyMessage(t, "CreateExperimentRequest"), legacyMessage(t, "Experiment"))
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	require.Equal(t, grpcBefore+1, promtestutil.ToFloat64(legacyAPIRequests.WithLabelValues("grpc")))
}

func TestLegacyStreamUsageIsCountedWithoutChangingCanonicalHandler(t *testing.T) {
	recorder := new(recordingCompatibilityRegistrar)
	descriptor := &grpc.ServiceDesc{ServiceName: canonicalRPCPackage + "StreamService", Streams: []grpc.StreamDesc{{
		StreamName: "Observe", Handler: func(any, grpc.ServerStream) error { return nil },
	}}}
	compatibleServiceRegistrar{recorder}.RegisterService(descriptor, nil)
	before := promtestutil.ToFloat64(legacyAPIRequests.WithLabelValues("grpc"))
	require.NoError(t, descriptor.Streams[0].Handler(nil, nil))
	require.Equal(t, before, promtestutil.ToFloat64(legacyAPIRequests.WithLabelValues("grpc")))
	require.NoError(t, recorder.descriptors[1].Streams[0].Handler(nil, nil))
	require.Equal(t, before+1, promtestutil.ToFloat64(legacyAPIRequests.WithLabelValues("grpc")))
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

func TestLegacyHTTPBodyLimit(t *testing.T) {
	t.Setenv(common.MaxPipelineUpdateBodyBytesEnv, "5")
	for _, prefix := range []string{canonicalAPIPath, legacyAPIPath} {
		router := buildHTTPRouter(newNoOpHTTPRouterDeps(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("oversized update reached the gateway")
		}), "database")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodPatch, prefix+"/pipelines/pipeline", strings.NewReader(`{"tags":{}}`)))
		require.Equal(t, http.StatusRequestEntityTooLarge, response.Code)
	}
}

func TestLegacyHTTPRouteBoundaries(t *testing.T) {
	deps := newNoOpHTTPRouterDeps()
	deps.ReadRunLog = func(http.ResponseWriter, *http.Request) { t.Error("POST reached the streaming handler") }
	router := buildHTTPRouter(deps, http.NotFoundHandler(), "database")
	for _, path := range []string{legacyAPIPath + "/runs/run/nodes/node/log", "/apis/v2beta10/healthz"} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, path, nil))
		require.Equal(t, http.StatusNotFound, response.Code)
	}
}

func TestLegacyHTTPHealthMatchesCanonical(t *testing.T) {
	router := buildHTTPRouter(newNoOpHTTPRouterDeps(), http.NotFoundHandler(), "database")
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

func legacyMessage(t *testing.T, name string) *dynamicpb.Message {
	t.Helper()
	descriptor, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(legacyRPCPackage + name))
	require.NoError(t, err)
	message, ok := descriptor.(protoreflect.MessageDescriptor)
	require.True(t, ok)
	return dynamicpb.NewMessage(message)
}

func compatibilityRPCConnection(t *testing.T) (*grpc.ClientConn, *atomic.Int32) {
	t.Helper()
	intercepted := new(atomic.Int32)
	rpc := grpc.NewServer(grpc.UnaryInterceptor(func(ctx context.Context, request interface{}, info *grpc.UnaryServerInfo, next grpc.UnaryHandler) (interface{}, error) {
		intercepted.Add(1)
		assert.Equal(t, "/"+canonicalRPCPackage+"ExperimentService/CreateExperiment", info.FullMethod)
		_, ok := request.(*api.CreateExperimentRequest)
		assert.True(t, ok, "legacy bytes must decode into the canonical request")
		return next(ctx, request)
	}))
	api.RegisterExperimentServiceServer(compatibleServiceRegistrar{rpc}, compatibilityExperimentServer{})
	reflection.Register(rpc)
	listener := bufconn.Listen(1024 * 1024)
	go rpc.Serve(listener)
	t.Cleanup(rpc.Stop)
	connection, err := grpc.NewClient("passthrough:///compatibility", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }))
	require.NoError(t, err)
	t.Cleanup(func() { connection.Close() })
	return connection, intercepted
}

func TestLegacyGRPCWireAndImportShimsUseCanonicalHandler(t *testing.T) {
	connection, intercepted := compatibilityRPCConnection(t)
	deadline, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	ctx := metadata.NewOutgoingContext(deadline, metadata.Pairs("authorization", "Bearer token"))
	newClient := api.NewExperimentServiceClient(connection)
	// These messages use the immutable pre-promotion schema, not Go aliases.
	// Invoke the legacy wire name explicitly so this tests already-built clients.
	oldRequest := legacyMessage(t, "CreateExperimentRequest")
	require.NoError(t, protojson.Unmarshal([]byte(`{"experiment":{"display_name":"experiment","namespace":"tenant"}}`), oldRequest))
	oldResponse := legacyMessage(t, "Experiment")
	oldMethod := "/" + legacyRPCPackage + "ExperimentService/CreateExperiment"
	var headers metadata.MD
	err := connection.Invoke(ctx, oldMethod, oldRequest, oldResponse, grpc.Header(&headers))
	require.NoError(t, err)
	require.Equal(t, []string{"shared-handler"}, headers.Get("compatibility"))
	newResponse, err := newClient.CreateExperiment(ctx, &api.CreateExperimentRequest{Experiment: &api.Experiment{DisplayName: "experiment", Namespace: "tenant"}})
	require.NoError(t, err)
	oldJSON, err := protojson.Marshal(oldResponse)
	require.NoError(t, err)
	newJSON, err := protojson.Marshal(newResponse)
	require.NoError(t, err)
	require.JSONEq(t, string(newJSON), string(oldJSON))
	// Recompiled legacy Go imports delegate directly to the canonical client.
	aliasedResponse, err := legacy.NewExperimentServiceClient(connection).CreateExperiment(ctx, &legacy.CreateExperimentRequest{Experiment: &legacy.Experiment{DisplayName: "experiment", Namespace: "tenant"}})
	require.NoError(t, err)
	require.True(t, proto.Equal(newResponse, aliasedResponse))
	require.Equal(t, int32(3), intercepted.Load())
}

func TestLegacyGRPCValidationAndAuthentication(t *testing.T) {
	connection, intercepted := compatibilityRPCConnection(t)
	deadline, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	ctx := metadata.NewOutgoingContext(deadline, metadata.Pairs("authorization", "Bearer token"))
	oldMethod := "/" + legacyRPCPackage + "ExperimentService/CreateExperiment"
	err := connection.Invoke(ctx, oldMethod, legacyMessage(t, "CreateExperimentRequest"), legacyMessage(t, "Experiment"))
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	err = connection.Invoke(deadline, oldMethod, legacyMessage(t, "CreateExperimentRequest"), legacyMessage(t, "Experiment"))
	require.Equal(t, codes.Unauthenticated, status.Code(err))
	require.Equal(t, int32(2), intercepted.Load())
}

func TestLegacyGRPCReflection(t *testing.T) {
	connection, _ := compatibilityRPCConnection(t)
	deadline, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	stream, err := reflectionv1.NewServerReflectionClient(connection).ServerReflectionInfo(deadline)
	require.NoError(t, err)
	require.NoError(t, stream.Send(&reflectionv1.ServerReflectionRequest{
		MessageRequest: &reflectionv1.ServerReflectionRequest_FileContainingSymbol{FileContainingSymbol: legacyRPCPackage + "ExperimentService"},
	}))
	response, err := stream.Recv()
	require.NoError(t, err)
	require.NotNil(t, response.GetFileDescriptorResponse())
	found := false
	for _, encoded := range response.GetFileDescriptorResponse().FileDescriptorProto {
		file := new(descriptorpb.FileDescriptorProto)
		require.NoError(t, proto.Unmarshal(encoded, file))
		if file.GetName() == "backend/api/v2beta1/experiment.proto" {
			found = true
			require.Equal(t, strings.TrimSuffix(legacyRPCPackage, "."), file.GetPackage())
		}
	}
	require.True(t, found, "reflection must expose the frozen legacy contract")
	require.NoError(t, stream.CloseSend())
}
