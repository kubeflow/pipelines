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

package mlmd_test

import (
	"context"
	"net"
	"strconv"
	"strings"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/migration/mlmd"
	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
)

type artifactPages struct {
	pb.UnimplementedMetadataStoreServiceServer
	count, bytes, executions, eventBytes int
}

func (s artifactPages) GetExecutions(context.Context, *pb.GetExecutionsRequest) (*pb.GetExecutionsResponse, error) {
	response := &pb.GetExecutionsResponse{}
	for index := 0; index < s.executions; index++ {
		response.Executions = append(response.Executions, &pb.Execution{Id: proto.Int64(int64(index + 1)), Type: proto.String("system.ContainerExecution")})
	}
	return response, nil
}

func (s artifactPages) GetContextsByExecution(context.Context, *pb.GetContextsByExecutionRequest) (*pb.GetContextsByExecutionResponse, error) {
	return &pb.GetContextsByExecutionResponse{}, nil
}

func (s artifactPages) GetEventsByExecutionIDs(_ context.Context, req *pb.GetEventsByExecutionIDsRequest) (*pb.GetEventsByExecutionIDsResponse, error) {
	response := &pb.GetEventsByExecutionIDsResponse{}
	key := strings.Repeat("a", s.eventBytes)
	for _, id := range req.GetExecutionIds() {
		response.Events = append(response.Events, &pb.Event{
			ExecutionId: proto.Int64(id),
			Path:        &pb.Event_Path{Steps: []*pb.Event_Path_Step{{Value: &pb.Event_Path_Step_Key{Key: key}}}},
		})
	}
	return response, nil
}

func (s artifactPages) GetArtifacts(_ context.Context, req *pb.GetArtifactsRequest) (*pb.GetArtifactsResponse, error) {
	start, _ := strconv.Atoi(req.GetOptions().GetNextPageToken())
	end := min(s.count, start+int(req.GetOptions().GetMaxResultSize()))
	data := strings.Repeat("a", s.bytes)
	response := &pb.GetArtifactsResponse{}
	for index := start; index < end; index++ {
		response.Artifacts = append(response.Artifacts, &pb.Artifact{
			Id: proto.Int64(int64(index + 1)), Type: proto.String("system.ClassificationMetrics"),
			CustomProperties: map[string]*pb.Value{"payload": stringValue(data)},
		})
	}
	if end < s.count {
		response.NextPageToken = proto.String(strconv.Itoa(end))
	}
	return response, nil
}

func TestReadSnapshotReleasedGRPCLimitAndPageFallback(t *testing.T) {
	for _, tc := range []struct {
		name, limit                          string
		count, bytes, executions, eventBytes int
		wantError                            bool
	}{
		{name: "released default accepts large record", count: 1, bytes: 5 << 20},
		{name: "aggregate over configured limit", limit: "4194304", count: 103, bytes: 45000},
		{name: "single record over limit", limit: "4194304", count: 1, bytes: 5 << 20, wantError: true},
		{name: "event batch over configured limit", limit: "524288", executions: 4, eventBytes: 300000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("METADATA_GRPC_MESSAGE_SIZE", tc.limit)
			listener := bufconn.Listen(1 << 20)
			server := grpc.NewServer()
			pb.RegisterMetadataStoreServiceServer(server, artifactPages{count: tc.count, bytes: tc.bytes, executions: tc.executions, eventBytes: tc.eventBytes})
			go func() { _ = server.Serve(listener) }()
			defer server.Stop()
			connection, err := mlmd.DialSource("passthrough:///fixture", false,
				grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return listener.DialContext(ctx) }))
			require.NoError(t, err)
			defer connection.Close()
			snapshot, err := mlmd.ReadSnapshot(context.Background(), pb.NewMetadataStoreServiceClient(connection))
			if tc.wantError {
				require.ErrorContains(t, err, "received message larger than max")
				return
			}
			require.NoError(t, err)
			require.Len(t, snapshot.Events, tc.executions)
			for i, event := range snapshot.Events {
				require.Equal(t, int64(i+1), event.GetExecutionId())
			}
			require.Len(t, snapshot.Artifacts, tc.count)
			for i, artifact := range snapshot.Artifacts {
				require.Equal(t, int64(i+1), artifact.GetId())
			}
		})
	}
}
