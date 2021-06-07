// Copyright 2021 The Kubeflow Authors
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

package metadata_test

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	"github.com/kubeflow/pipelines/v2/metadata"
	pb "github.com/kubeflow/pipelines/v2/third_party/ml_metadata"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
)

// This test depends on a MLMD grpc server running at localhost:8080.
const (
	testMlmdServerAddress = "localhost"
	testMlmdServerPort    = "8080"
)

func Test_schemaToArtifactType(t *testing.T) {
	tests := []struct {
		name    string
		schema  string
		want    *pb.ArtifactType
		wantErr bool
	}{
		{
			name:   "Parses Schema Title Correctly",
			schema: "properties:\ntitle: kfp.Dataset\ntype: object\n",
			want: &pb.ArtifactType{
				Name: proto.String("kfp.Dataset"),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := metadata.SchemaToArtifactType(tt.schema)
			if (err != nil) != tt.wantErr {
				t.Errorf("schemaToArtifactType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(got, tt.want, cmpopts.EquateEmpty(), protocmp.Transform()); diff != "" {
				t.Errorf("schemaToArtifactType() = %+v, want %+v\nDiff (-want, +got)\n%s", got, tt.want, diff)
			}
		})
	}
}

func Test_GetPipeline(t *testing.T) {
	fatalIf := func(err error) {
		if err != nil {
			debug.PrintStack()
			t.Fatal(err)
		}
	}

	ctx := context.Background()
	runUuid, err := uuid.NewRandom()
	fatalIf(err)
	runId := runUuid.String()
	client, err := metadata.NewClient(testMlmdServerAddress, testMlmdServerPort)
	fatalIf(err)
	mlmdClient, err := NewTestMlmdClient()
	fatalIf(err)

	_, err = client.GetPipeline(ctx, "get-pipeline-test", runId)
	fatalIf(err)
	runCtxType := "kfp.PipelineRun"
	pipeline := "get-pipeline-test"

	res, err := mlmdClient.GetContextByTypeAndName(ctx, &pb.GetContextByTypeAndNameRequest{
		TypeName:    &runCtxType,
		ContextName: &runId,
	})
	fatalIf(err)
	if res.GetContext() == nil {
		t.Fatalf("GetContextByTypeAndName(name=%q, type=%q)=nil", runId, runCtxType)
	}
	resParents, err := mlmdClient.GetParentContextsByContext(ctx, &pb.GetParentContextsByContextRequest{
		ContextId: res.GetContext().Id,
	})
	fatalIf(err)
	parents := resParents.GetContexts()
	if len(parents) != 1 {
		t.Errorf("Got %v parent contexts, want 1", len(parents))
	}
	pipelineCtx := parents[0]
	if pipelineCtx.GetName() != pipeline {
		t.Errorf("GetParentContextsByContext(name=%q, type=%q)=Context(name=%q), want Context(name=%q)",
			runId, runCtxType, pipelineCtx.GetName(), pipeline)
	}
}

func Test_GetPipelineConcurrently(t *testing.T) {
	// This test depends on a MLMD grpc server running at localhost:8080.
	client, err := metadata.NewClient("localhost", "8080")
	if err != nil {
		t.Fatal(err)
	}
	runId, err := uuid.NewRandom()
	if err != nil {
		t.Fatal(err)
	}
	runIdText := runId.String()
	var wg sync.WaitGroup
	ctx := context.Background()
	// Simulates 5 concurrent tasks trying to create the same pipeline contexts.
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := client.GetPipeline(ctx, fmt.Sprintf("get-pipeline-concurrently-test-%s", runIdText), runIdText)
			if err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	// Then another 5 concurrent tasks.
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := client.GetPipeline(ctx, fmt.Sprintf("get-pipeline-concurrently-test-%s", runIdText), runIdText)
			if err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
}

func NewTestMlmdClient() (pb.MetadataStoreServiceClient, error) {
	conn, err := grpc.Dial(fmt.Sprintf("%s:%s", testMlmdServerAddress, testMlmdServerPort),
		grpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("NewMlmdClient() failed: %w", err)
	}
	return pb.NewMetadataStoreServiceClient(conn), nil
}
