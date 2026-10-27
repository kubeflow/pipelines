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
	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/server"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestRecurringRunHTTPDeploymentRestrictions(t *testing.T) {
	viper.Set(common.ScheduledWorkflowControllerEnabled, false)
	t.Cleanup(func() { viper.Set(common.ScheduledWorkflowControllerEnabled, nil) })
	listener := bufconn.Listen(1024 * 1024)
	rpcServer := grpc.NewServer(grpc.UnaryInterceptor(apiServerInterceptor))
	apiv2.RegisterRecurringRunServiceServer(rpcServer, server.NewJobServer(nil, &server.JobServerOptions{}))
	go rpcServer.Serve(listener)
	t.Cleanup(rpcServer.Stop)
	conn, err := grpc.NewClient("passthrough:///capabilities",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }))
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	gateway := runtime.NewServeMux()
	require.NoError(t, apiv2.RegisterRecurringRunServiceHandler(context.Background(), gateway, conn))
	router := buildHTTPRouter(newNoOpHTTPRouterDeps(), gateway, "database")
	for _, prefix := range []string{"/apis/v2", "/apis/v2beta1"} {
		for _, request := range []struct{ method, path string }{
			{http.MethodPost, "/recurringruns"},
			{http.MethodGet, "/recurringruns"},
			{http.MethodGet, "/recurringruns/missing"},
			{http.MethodPost, "/recurringruns/missing:enable"},
			{http.MethodPost, "/recurringruns/missing:disable"},
			{http.MethodDelete, "/recurringruns/missing"},
		} {
			t.Run(prefix+request.method+request.path, func(t *testing.T) {
				recorder := httptest.NewRecorder()
				req := httptest.NewRequest(request.method, prefix+request.path, strings.NewReader("{}"))
				req.Header.Set("Content-Type", "application/json")
				router.ServeHTTP(recorder, req)
				require.Equal(t, http.StatusBadRequest, recorder.Code, recorder.Body.String())
				var response struct {
					Code    int    `json:"code"`
					Message string `json:"message"`
				}
				require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
				require.Equal(t, 9, response.Code)
				require.Contains(t, response.Message, common.RecurringRunsUnavailableReason)
			})
		}
	}
}

func TestHealthzControllerConfigurationCompatibility(t *testing.T) {
	for _, configured := range []any{nil, true, false} {
		viper.Set(common.ScheduledWorkflowControllerEnabled, configured)
		t.Cleanup(func() { viper.Set(common.ScheduledWorkflowControllerEnabled, nil) })
		response := newHealthzResponse("database")
		require.Equal(t, configured != false, response.GetCapabilities().GetRecurringRuns().GetSupported())
		if configured != false {
			require.Empty(t, response.GetCapabilities().GetRecurringRuns().GetReason())
		}
		// The canonical schema must describe the actual HTTP response, including false.
		recorder := httptest.NewRecorder()
		writeJSONResponse(recorder, response)
		var decoded apiv2.GetHealthzResponse
		require.NoError(t, protojson.Unmarshal(recorder.Body.Bytes(), &decoded))
		require.Equal(t, response.GetCapabilities().GetRecurringRuns().GetSupported(),
			decoded.GetCapabilities().GetRecurringRuns().GetSupported())
	}
}
