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
	"testing"

	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

func TestGetInputArtifactsByExecutionID_MultipleArtifactsSameName(t *testing.T) {
	// Two artifacts recorded as INPUT events under the same name, the way
	// GenerateExecutionConfig/PublishExecution write a list-typed artifact
	// input (e.g. dsl.Collected() fan-in): InputArtifactIDs maps one name to
	// multiple artifact IDs, each written as its own event sharing eventPath(name).
	events := []*pb.Event{
		{
			ArtifactId: int64Ptr(101),
			Path:       eventPath("models"),
			Type:       pb.Event_INPUT.Enum(),
		},
		{
			ArtifactId: int64Ptr(102),
			Path:       eventPath("models"),
			Type:       pb.Event_INPUT.Enum(),
		},
	}
	artifacts := []*pb.Artifact{
		{Id: int64Ptr(101), Uri: proto.String("gs://bucket/model-a")},
		{Id: int64Ptr(102), Uri: proto.String("gs://bucket/model-b")},
	}

	mock := &mockMLMDClient{
		getEventsByExecutionIDsFn: func(_ context.Context, _ *pb.GetEventsByExecutionIDsRequest, _ ...grpc.CallOption) (*pb.GetEventsByExecutionIDsResponse, error) {
			return &pb.GetEventsByExecutionIDsResponse{Events: events}, nil
		},
		getArtifactsByIDFn: func(_ context.Context, _ *pb.GetArtifactsByIDRequest, _ ...grpc.CallOption) (*pb.GetArtifactsByIDResponse, error) {
			return &pb.GetArtifactsByIDResponse{Artifacts: artifacts}, nil
		},
	}
	client := newTestClient(mock)

	inputs, err := client.GetInputArtifactsByExecutionID(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	list, ok := inputs["models"]
	if !ok {
		t.Fatalf("expected an artifact list for input %q, got none", "models")
	}
	if len(list.Artifacts) != 2 {
		t.Errorf("expected 2 artifacts for input %q, got %d: %v", "models", len(list.Artifacts), list.Artifacts)
	}
}
