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
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
)

// This test depends on a MLMD grpc server running at localhost:8080.
const (
	testMlmdServerAddress = "localhost"
	testMlmdServerPort    = "8080"
	namespace             = "kubeflow"
	runResource           = "workflows.argoproj.io/hello-world-abcd"
	pipelineRoot          = "gs://my-bucket/path/to/root"
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
	t.Skip("Temporarily disable the test that requires cluster connection.")

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
	client, err := metadata.NewClient(testMlmdServerAddress, testMlmdServerPort, false, "unused-ca-cert-path")
	fatalIf(err)
	mlmdClient, err := NewTestMlmdClient()
	fatalIf(err)

	pipeline, err := client.GetPipeline(ctx, "get-pipeline-test", runId, namespace, runResource, pipelineRoot, "")
	fatalIf(err)
	expectPipelineRoot := fmt.Sprintf("%s/get-pipeline-test/%s", pipelineRoot, runId)
	if pipeline.GetPipelineRoot() != expectPipelineRoot {
		t.Errorf("client.GetPipeline(pipelineRoot=%q)=%q, expect %q", pipelineRoot, pipeline.GetPipelineRoot(), expectPipelineRoot)
	}
	runCtxType := "system.PipelineRun"
	pipelineName := "get-pipeline-test"

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
	if pipelineCtx.GetName() != pipelineName {
		t.Errorf("GetParentContextsByContext(name=%q, type=%q)=Context(name=%q), want Context(name=%q)",
			runId, runCtxType, pipelineCtx.GetName(), pipelineName)
	}
}

func Test_GetPipeline_Twice(t *testing.T) {
	t.Skip("Temporarily disable the test that requires cluster connection.")

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
	client, err := metadata.NewClient(testMlmdServerAddress, testMlmdServerPort, false, "unused-ca-cert-path")
	fatalIf(err)

	pipeline, err := client.GetPipeline(ctx, "get-pipeline-test", runId, namespace, runResource, pipelineRoot, "")
	fatalIf(err)
	// The second call to GetPipeline won't fail because it avoid inserting to MLMD again.
	samePipeline, err := client.GetPipeline(ctx, "get-pipeline-test", runId, namespace, runResource, pipelineRoot, "")
	fatalIf(err)
	if pipeline.GetCtxID() != samePipeline.GetCtxID() {
		t.Errorf("Expect pipeline context ID %d, actual is %d", pipeline.GetCtxID(), samePipeline.GetCtxID())
	}
}

func Test_GetPipelineFromExecution(t *testing.T) {
	t.Skip("Temporarily disable the test that requires cluster connection.")

	fatalIf := func(err error) {
		if err != nil {
			debug.PrintStack()
			t.Fatal(err)
		}
	}
	client := newLocalClientOrFatal(t)
	ctx := context.Background()
	pipeline, err := client.GetPipeline(ctx, "get-pipeline-from-execution", newUUIDOrFatal(t), "kubeflow", "workflow/abc", "gs://my-bucket/root", "")
	fatalIf(err)
	execution, err := client.CreateExecution(ctx, pipeline, &metadata.ExecutionConfig{
		TaskName:      "task1",
		ExecutionType: metadata.ContainerExecutionTypeName,
	})
	fatalIf(err)
	gotPipeline, err := client.GetPipelineFromExecution(ctx, execution.GetID())
	fatalIf(err)
	if gotPipeline.GetRunCtxID() != pipeline.GetRunCtxID() {
		t.Errorf("client.GetPipelineFromExecution(id=%v)=Pipeline(runCtxID=%v), expect Pipeline(runCtxID=%v)", execution.GetID(), gotPipeline.GetRunCtxID(), pipeline.GetRunCtxID())
	}
}

func Test_GetPipelineConcurrently(t *testing.T) {
	t.Skip("Temporarily disable the test that requires cluster connection.")

	// This test depends on a MLMD grpc server running at localhost:8080.
	client, err := metadata.NewClient("localhost", "8080", false, "unused-ca-cert-path")
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
			_, err := client.GetPipeline(ctx, fmt.Sprintf("get-pipeline-concurrently-test-%s", runIdText), runIdText, namespace, "workflows.argoproj.io/hello-world-"+runIdText, pipelineRoot, "")
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
			_, err := client.GetPipeline(ctx, fmt.Sprintf("get-pipeline-concurrently-test-%s", runIdText), runIdText, namespace, "workflows.argoproj.io/hello-world-"+runIdText, pipelineRoot, "")
			if err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
}

func Test_GenerateOutputURI(t *testing.T) {
	// Const define the artifact name
	const (
		pipelineName      = "my-pipeline-name"
		runID             = "my-run-id"
		pipelineRoot      = "minio://mlpipeline/v2/artifacts"
		pipelineRootQuery = "?query=string&another=query"
	)
	tests := []struct {
		name                string
		queryString         string
		paths               []string
		preserveQueryString bool
		want                string
	}{
		{
			name:                "plain pipeline root without preserveQueryString",
			queryString:         "",
			paths:               []string{pipelineName, runID},
			preserveQueryString: false,
			want:                fmt.Sprintf("%s/%s/%s", pipelineRoot, pipelineName, runID),
		},
		{
			name:                "plain pipeline root with preserveQueryString",
			queryString:         "",
			paths:               []string{pipelineName, runID},
			preserveQueryString: true,
			want:                fmt.Sprintf("%s/%s/%s", pipelineRoot, pipelineName, runID),
		},
		{
			name:                "pipeline root with query string without preserveQueryString",
			queryString:         pipelineRootQuery,
			paths:               []string{pipelineName, runID},
			preserveQueryString: false,
			want:                fmt.Sprintf("%s/%s/%s", pipelineRoot, pipelineName, runID),
		},
		{
			name:                "pipeline root with query string with preserveQueryString",
			queryString:         pipelineRootQuery,
			paths:               []string{pipelineName, runID},
			preserveQueryString: true,
			want:                fmt.Sprintf("%s/%s/%s%s", pipelineRoot, pipelineName, runID, pipelineRootQuery),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := metadata.GenerateOutputURI(fmt.Sprintf("%s%s", pipelineRoot, tt.queryString), tt.paths, tt.preserveQueryString)
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("GenerateOutputURI() = %v, want %v\nDiff (-want, +got)\n%s", got, tt.want, diff)
			}
		})
	}
}

func Test_DAG(t *testing.T) {
	t.Skip("Temporarily disable the test that requires cluster connection.")

	client := newLocalClientOrFatal(t)
	ctx := context.Background()
	// These parameters do not matter.
	pipeline, err := client.GetPipeline(ctx, "pipeline-name", newUUIDOrFatal(t), "ns1", "workflow/pipeline-1234", pipelineRoot, "")
	if err != nil {
		t.Fatal(err)
	}
	root, err := client.CreateExecution(ctx, pipeline, &metadata.ExecutionConfig{
		TaskName:      "root",
		ExecutionType: metadata.DagExecutionTypeName,
		ParentDagID:   0, // this is root DAG
	})
	if err != nil {
		t.Fatal(err)
	}
	task1DAG, err := client.CreateExecution(ctx, pipeline, &metadata.ExecutionConfig{
		TaskName:      "task1",
		ExecutionType: metadata.DagExecutionTypeName,
		ParentDagID:   root.GetID(),
	})
	if err != nil {
		t.Fatal(err)
	}
	task1ChildA, err := client.CreateExecution(ctx, pipeline, &metadata.ExecutionConfig{
		TaskName:      "task1ChildA",
		ExecutionType: metadata.ContainerExecutionTypeName,
		ParentDagID:   task1DAG.GetID(),
	})
	if err != nil {
		t.Fatal(err)
	}
	task2, err := client.CreateExecution(ctx, pipeline, &metadata.ExecutionConfig{
		TaskName:      "task2",
		ExecutionType: metadata.ContainerExecutionTypeName,
		ParentDagID:   root.GetID(),
	})
	if err != nil {
		t.Fatal(err)
	}
	rootDAG := &metadata.DAG{Execution: root}
	rootChildren, err := client.GetExecutionsInDAG(ctx, rootDAG, pipeline)
	if err != nil {
		t.Fatal(err)
	}
	if len(rootChildren) != 2 {
		t.Errorf("len(rootChildren)=%v, expect 2", len(rootChildren))
	}
	if rootChildren["task1"].GetID() != task1DAG.GetID() {
		t.Errorf("executions[\"task1\"].GetID()=%v, task1.GetID()=%v. Not equal", rootChildren["task1"].GetID(), task1DAG.GetID())
	}
	if rootChildren["task2"].GetID() != task2.GetID() {
		t.Errorf("executions[\"task2\"].GetID()=%v, task2.GetID()=%v. Not equal", rootChildren["task2"].GetID(), task2.GetID())
	}
	task1Children, err := client.GetExecutionsInDAG(ctx, &metadata.DAG{Execution: task1DAG}, pipeline)
	if len(task1Children) != 1 {
		t.Errorf("len(task1Children)=%v, expect 1", len(task1Children))
	}
	if task1Children["task1ChildA"].GetID() != task1ChildA.GetID() {
		t.Errorf("executions[\"task1ChildA\"].GetID()=%v, task1ChildA.GetID()=%v. Not equal", task1Children["task1ChildA"].GetID(), task1ChildA.GetID())
	}
}

func newLocalClientOrFatal(t *testing.T) *metadata.Client {
	t.Helper()
	client, err := metadata.NewClient("localhost", "8080", false, "unused-ca-cert-path")
	if err != nil {
		t.Fatalf("metadata.NewClient failed: %v", err)
	}
	return client
}

func newUUIDOrFatal(t *testing.T) string {
	t.Helper()
	uuid, err := uuid.NewRandom()
	if err != nil {
		t.Fatalf("uuid.NewRandom failed: %v", err)
	}
	return uuid.String()
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
