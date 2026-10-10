// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	api1 "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	api2 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

type orderingV1Server struct {
	api1.UnimplementedRunServiceServer
}

func (orderingV1Server) ListRunsV1(ctx context.Context, req *api1.ListRunsRequest) (*api1.ListRunsResponse, error) {
	_, err := validatedListOptions(&model.Run{}, req.GetPageToken(), 2, req.GetSortBy(), req.GetFilter(), "v1beta1")
	return nil, util.ToGRPCError(util.Wrapf(util.Wrap(err, "list request"), "run listing pageToken=%s", req.GetPageToken()))
}

type orderingV2Server struct {
	api2.UnimplementedRunServiceServer
}

func (orderingV2Server) ListRuns(ctx context.Context, req *api2.ListRunsRequest) (*api2.ListRunsResponse, error) {
	_, err := validatedListOptions(&model.Run{}, req.GetPageToken(), 2, req.GetSortBy(), req.GetFilter(), "v2beta1")
	return nil, util.ToGRPCError(util.Wrapf(util.Wrap(err, "list request"), "run listing pageToken=%s", req.GetPageToken()))
}

// The generated gateways use a real gRPC connection here, exercising Any detail
// serialization and the same error wrapping used by server handlers.
func TestPaginationRestartRequiredGRPCAndGateway(t *testing.T) {
	ctx := context.Background()
	listener := bufconn.Listen(1024 * 1024)
	grpcServer := grpc.NewServer()
	api1.RegisterRunServiceServer(grpcServer, orderingV1Server{})
	api2.RegisterRunServiceServer(grpcServer, orderingV2Server{})
	go func() { _ = grpcServer.Serve(listener) }()
	t.Cleanup(func() { grpcServer.Stop(); _ = listener.Close() })
	conn, err := grpc.NewClient("passthrough:///pagination", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	mux := runtime.NewServeMux()
	require.NoError(t, api1.RegisterRunServiceHandler(ctx, mux, conn))
	require.NoError(t, api2.RegisterRunServiceHandler(ctx, mux, conn))

	captured := legacyFilterTokens(t)[0]
	raw, err := base64.StdEncoding.DecodeString(captured.Token)
	require.NoError(t, err)
	var fields map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &fields))
	fields["SortByFieldName"], fields["SortBySQLColumn"] = "State", "State"
	fields["SortByFieldValue"], fields["KeyFieldValue"] = "RUNNING", "run-id"
	fields["ModelName"], fields["SortByFieldPrefix"], fields["KeyFieldPrefix"] = "", "", ""
	fields["IsDesc"], fields["Filter"] = false, nil
	delete(fields, "OrderingVersion")
	raw, err = json.Marshal(fields)
	require.NoError(t, err)
	legacy := base64.StdEncoding.EncodeToString(raw)
	for _, version := range []string{"v1beta1", "v2beta1"} {
		for _, repeat := range []bool{false, true} {
			t.Run(version+map[bool]string{false: "_token_only", true: "_repeated_sort"}[repeat], func(t *testing.T) {
				sortBy := ""
				if repeat {
					sortBy = "state asc"
				}
				if version == "v1beta1" {
					_, err = api1.NewRunServiceClient(conn).ListRunsV1(ctx, &api1.ListRunsRequest{PageToken: legacy, SortBy: sortBy})
				} else {
					_, err = api2.NewRunServiceClient(conn).ListRuns(ctx, &api2.ListRunsRequest{PageToken: legacy, SortBy: sortBy})
				}
				require.Equal(t, codes.FailedPrecondition, status.Code(err))
				found := false
				for _, detail := range status.Convert(err).Details() {
					if info, ok := detail.(*errdetails.ErrorInfo); ok {
						require.Equal(t, "PAGINATION_RESTART_REQUIRED", info.Reason)
						require.Equal(t, "kubeflow.org", info.Domain)
						found = true
					}
				}
				require.True(t, found)
				query := url.Values{"page_token": {legacy}, "sort_by": {sortBy}}
				recorder := httptest.NewRecorder()
				mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/apis/"+version+"/runs?"+query.Encode(), nil))
				require.Equal(t, http.StatusBadRequest, recorder.Code)
				var body struct {
					Code    int                      `json:"code"`
					Message string                   `json:"message"`
					Details []map[string]interface{} `json:"details"`
				}
				require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
				require.Equal(t, 9, body.Code)
				require.Contains(t, body.Message, "Clear page_token")
				require.NotContains(t, body.Message, legacy)
				found = false
				for _, detail := range body.Details {
					if detail["@type"] == "type.googleapis.com/google.rpc.ErrorInfo" {
						require.Equal(t, "PAGINATION_RESTART_REQUIRED", detail["reason"])
						require.Equal(t, "kubeflow.org", detail["domain"])
						found = true
					}
				}
				require.True(t, found, recorder.Body.String())
			})
		}
	}
}
