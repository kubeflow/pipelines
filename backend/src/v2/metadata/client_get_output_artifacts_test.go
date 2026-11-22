// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metadata

import (
	"context"
	"slices"
	"testing"

	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

func TestPublishedOutputArtifactAttempt(t *testing.T) {
	ctx := context.Background()
	execution := &Execution{Execution: makeExecution(10, pb.Execution_RUNNING, map[string]*pb.Value{keyPodUID: StringValue("first-pod")})}
	var stored *pb.Execution
	var events []*pb.Event
	client := newTestClient(&MockMLMDClient{
		PutExecutionFn: func(_ context.Context, req *pb.PutExecutionRequest, _ ...grpc.CallOption) (*pb.PutExecutionResponse, error) {
			stored = proto.Clone(req.Execution).(*pb.Execution)
			for _, pair := range req.GetArtifactEventPairs() {
				event := proto.Clone(pair.GetEvent()).(*pb.Event)
				// Transfer imports artifacts under different IDs and remaps each event.
				event.ArtifactId = proto.Int64(event.GetArtifactId() + 100)
				events = append(events, event)
			}
			return &pb.PutExecutionResponse{ExecutionId: stored.Id}, nil
		},
		getExecutionsByIDFn: func(context.Context, *pb.GetExecutionsByIDRequest, ...grpc.CallOption) (*pb.GetExecutionsByIDResponse, error) {
			return &pb.GetExecutionsByIDResponse{Executions: []*pb.Execution{stored}}, nil
		},
		getEventsByExecutionIDsFn: func(context.Context, *pb.GetEventsByExecutionIDsRequest, ...grpc.CallOption) (*pb.GetEventsByExecutionIDsResponse, error) {
			return &pb.GetEventsByExecutionIDsResponse{Events: events}, nil
		},
		getArtifactsByIDFn: func(_ context.Context, req *pb.GetArtifactsByIDRequest, _ ...grpc.CallOption) (*pb.GetArtifactsByIDResponse, error) {
			response := &pb.GetArtifactsByIDResponse{}
			for _, id := range req.GetArtifactIds() {
				response.Artifacts = append(response.Artifacts, &pb.Artifact{Id: proto.Int64(id)})
			}
			slices.Reverse(response.Artifacts)
			return response, nil
		},
	})
	require.NoError(t, client.PublishExecution(ctx, execution, nil, []*OutputArtifact{
		{Name: "model", Artifact: &pb.Artifact{Id: proto.Int64(1)}},
		{Name: "old", Artifact: &pb.Artifact{Id: proto.Int64(2)}},
	}, pb.Execution_COMPLETE))
	require.NoError(t, client.PublishExecution(ctx, execution, nil, nil, pb.Execution_FAILED))
	require.Len(t, events, 2)
	require.Equal(t, "first-pod", stored.GetCustomProperties()[keyOutputAttempt].GetStringValue())

	execution.Execution.CustomProperties[keyPodUID] = StringValue("second-pod")
	require.NoError(t, client.PublishExecution(ctx, execution, nil, []*OutputArtifact{
		{Name: "model", Artifact: &pb.Artifact{Id: proto.Int64(3)}},
		{Name: "metrics", Artifact: &pb.Artifact{Id: proto.Int64(4)}},
	}, pb.Execution_COMPLETE))
	for range 2 {
		got, err := client.GetOutputArtifactsByExecutionId(ctx, execution.GetID())
		require.NoError(t, err)
		require.Len(t, got, 2)
		require.EqualValues(t, 103, got["model"].Artifact.GetId())
		require.EqualValues(t, 104, got["metrics"].Artifact.GetId())
		slices.Reverse(events)
	}

	execution.Execution.CustomProperties[keyPodUID] = StringValue("empty-pod")
	require.NoError(t, client.PublishExecution(ctx, execution, nil, []*OutputArtifact{}, pb.Execution_COMPLETE))
	got, err := client.GetOutputArtifactsByExecutionId(ctx, execution.GetID())
	require.NoError(t, err)
	require.Empty(t, got)
}

func TestOutputArtifactLookupWithoutAttemptMarker(t *testing.T) {
	ctx := context.Background()
	execution := makeExecution(10, pb.Execution_COMPLETE, map[string]*pb.Value{keyPodUID: StringValue("old-launcher-pod")})
	artifact := &pb.Artifact{Id: proto.Int64(123)}
	client := newTestClient(&MockMLMDClient{
		getExecutionsByIDFn: func(context.Context, *pb.GetExecutionsByIDRequest, ...grpc.CallOption) (*pb.GetExecutionsByIDResponse, error) {
			return &pb.GetExecutionsByIDResponse{Executions: []*pb.Execution{execution}}, nil
		},
		getEventsByExecutionIDsFn: func(context.Context, *pb.GetEventsByExecutionIDsRequest, ...grpc.CallOption) (*pb.GetEventsByExecutionIDsResponse, error) {
			events := []*pb.Event{{Type: pb.Event_OUTPUT.Enum(), Path: eventPath("model"), ArtifactId: artifact.Id}}
			if previous := execution.CustomProperties[keyOutputAttempt].GetStringValue(); previous != "" {
				path := eventPath("model")
				path.Steps = append(path.Steps, &pb.Event_Path_Step{Value: &pb.Event_Path_Step_Key{Key: previous}})
				events = append(events, &pb.Event{Type: pb.Event_OUTPUT.Enum(), Path: path, ArtifactId: proto.Int64(456)})
			}
			return &pb.GetEventsByExecutionIDsResponse{Events: events}, nil
		},
		getArtifactsByIDFn: func(_ context.Context, req *pb.GetArtifactsByIDRequest, _ ...grpc.CallOption) (*pb.GetArtifactsByIDResponse, error) {
			require.Equal(t, []int64{artifact.GetId()}, req.GetArtifactIds())
			return &pb.GetArtifactsByIDResponse{Artifacts: []*pb.Artifact{artifact}}, nil
		},
	})
	for _, previousAttempt := range []string{"", "previous-pod"} {
		if previousAttempt != "" {
			execution.CustomProperties[keyOutputAttempt] = StringValue(previousAttempt)
		}
		got, err := client.GetOutputArtifactsByExecutionId(ctx, execution.GetId())
		require.NoError(t, err)
		require.Contains(t, got, "model")
		require.Equal(t, artifact.GetId(), got["model"].Artifact.GetId())
	}
}
