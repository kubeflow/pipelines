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

func TestAPIRegistration_V2WithLegacyAliases(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	viper.Set(common.PodNamespace, "ns1")
	viper.Set(common.MultiUserMode, false)
	clients := resource.NewFakeClientManagerOrFatalV2()
	t.Cleanup(func() { require.NoError(t, clients.Close()) })
	manager := resource.NewResourceManager(clients, &resource.ResourceManagerOptions{})
	rpc := grpc.NewServer(grpc.UnaryInterceptor(apiServerInterceptor))
	registerRPCServices(rpc, manager)
	require.Len(t, rpc.GetServiceInfo(), 14)
	for _, prefix := range []string{canonicalRPCPackage, legacyRPCPackage} {
		require.NotContains(t, rpc.GetServiceInfo(), prefix+"VisualizationService")
		for _, service := range []string{"AuthService", "ExperimentService", "PipelineService", "RecurringRunService", "RunService", "ReportService", "ArtifactService"} {
			require.Contains(t, rpc.GetServiceInfo(), prefix+service)
		}
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
	// A write through the legacy gateway is immediately visible through v2,
	// and mutations through v2 are visible to legacy readers of the same row.
	created := request(http.MethodPost, legacyAPIPath+"/experiments", `{"display_name":"legacy caller","namespace":"ns1"}`, http.StatusOK)
	var experiment map[string]interface{}
	require.NoError(t, json.Unmarshal(created.Body.Bytes(), &experiment))
	experimentID, ok := experiment["experiment_id"].(string)
	require.True(t, ok)
	require.NotEmpty(t, experimentID)
	fetched := request(http.MethodGet, canonicalAPIPath+"/experiments/"+experimentID, "", http.StatusOK)
	require.JSONEq(t, created.Body.String(), fetched.Body.String())
	request(http.MethodPost, canonicalAPIPath+"/experiments/"+experimentID+":archive", "", http.StatusOK)
	archived := request(http.MethodGet, legacyAPIPath+"/experiments/"+experimentID, "", http.StatusOK)
	require.NoError(t, json.Unmarshal(archived.Body.Bytes(), &experiment))
	require.Equal(t, "ARCHIVED", experiment["storage_state"])

	for _, prefix := range []string{canonicalAPIPath, legacyAPIPath} {
		request(http.MethodGet, prefix+"/auth?namespace=ns1&resources=VIEWERS&verb=GET", "", http.StatusOK)
		request(http.MethodPost, prefix+"/visualizations/ns1", `{"type":"CUSTOM","arguments":"{}"}`, http.StatusNotFound)
		// Report routes reach validation, rather than an unregistered-route 404.
		request(http.MethodPost, prefix+"/workflows", `"invalid"`, http.StatusBadRequest)
		request(http.MethodPost, prefix+"/scheduledworkflows", `"invalid"`, http.StatusBadRequest)
	}
	artifact, err := clients.ArtifactStore().CreateArtifact(&model.Artifact{Namespace: "ns1", Name: "artifact", URI: new("s3://bucket/artifact")})
	require.NoError(t, err)
	result := request(http.MethodGet, "/apis/v2/artifacts/"+artifact.UUID, "", http.StatusOK)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(result.Body.Bytes(), &body))
	require.Equal(t, artifact.UUID, body["artifact_id"])
	legacyResult := request(http.MethodGet, legacyAPIPath+"/artifacts/"+artifact.UUID, "", http.StatusOK)
	require.JSONEq(t, result.Body.String(), legacyResult.Body.String())
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
