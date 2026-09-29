// Copyright 2026 The Kubeflow Authors
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
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestAPIRegistration_V2Only(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	viper.Set(common.PodNamespace, "ns1")
	viper.Set(common.MultiUserMode, false)
	clients := resource.NewFakeClientManagerOrFatalV2()
	t.Cleanup(func() { require.NoError(t, clients.Close()) })
	manager := resource.NewResourceManager(clients, &resource.ResourceManagerOptions{})
	rpc := grpc.NewServer(grpc.UnaryInterceptor(apiServerInterceptor))
	registerRPCServices(rpc, manager)
	require.Len(t, rpc.GetServiceInfo(), 7)
	require.NotContains(t, rpc.GetServiceInfo(), "kubeflow.pipelines.backend.api.v2beta1.VisualizationService")
	for name := range rpc.GetServiceInfo() {
		require.True(t, strings.HasPrefix(name, "kubeflow.pipelines.backend.api.v2beta1."), name)
	}
	for _, service := range []string{"AuthService", "ReportService", "ArtifactService"} {
		require.Contains(t, rpc.GetServiceInfo(), "kubeflow.pipelines.backend.api.v2beta1."+service)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	go rpc.Serve(listener)
	t.Cleanup(rpc.Stop)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	gateway := runtime.NewServeMux(runtime.WithIncomingHeaderMatcher(grpcCustomMatcher), runtime.WithMarshalerOption(runtime.MIMEWildcard, common.CustomMarshaler()))
	registerGatewayServices(func(register RegisterHttpHandlerFromEndpoint, _ string) {
		require.NoError(t, register(ctx, gateway, listener.Addr().String(), []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}))
	})
	router := buildHTTPRouter(newNoOpHTTPRouterDeps(), gateway, "database")
	request := func(method, path, body string, want int) *httptest.ResponseRecorder {
		t.Helper()
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(recorder, req)
		require.Equal(t, want, recorder.Code, "%s: %s", path, recorder.Body.String())
		return recorder
	}
	request(http.MethodGet, "/apis/v2beta1/auth?namespace=ns1&resources=VIEWERS&verb=GET", "", http.StatusOK)
	request(http.MethodPost, "/apis/v2beta1/visualizations/ns1", `{"type":"CUSTOM","arguments":"{}"}`, http.StatusNotFound)
	// Report routes reach validation, rather than an unregistered-route 404.
	request(http.MethodPost, "/apis/v2beta1/workflows", `"invalid"`, http.StatusBadRequest)
	request(http.MethodPost, "/apis/v2beta1/scheduledworkflows", `"invalid"`, http.StatusBadRequest)
	artifact, err := clients.ArtifactStore().CreateArtifact(&model.Artifact{Namespace: "ns1", Name: "artifact", URI: new("s3://bucket/artifact")})
	require.NoError(t, err)
	result := request(http.MethodGet, "/apis/v2beta1/artifacts/"+artifact.UUID, "", http.StatusOK)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(result.Body.Bytes(), &body))
	require.Equal(t, artifact.UUID, body["artifact_id"])
	for _, path := range []string{
		"/apis/v1beta1/healthz", "/apis/v1beta1/auth", "/apis/v1beta1/pipelines/upload",
		"/apis/v1beta1/runs", "/apis/v1beta1/workflows", "/apis/v1beta1/scheduledworkflows",
		"/apis/v1beta1/visualizations/ns1", "/apis/v1alpha1/runs/run/nodes/node/log",
		"/apis/v1beta1/runs/run/nodes/node/artifacts/artifact:read",
	} {
		for _, method := range []string{http.MethodGet, http.MethodPost} {
			request(method, path, "", http.StatusNotFound)
		}
	}
}
