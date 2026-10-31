//go:build ignore

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

// Run from the pinned legacy checkout, against an empty disposable MLMD server.
// See README.md. This generator deliberately does not generate expected.json.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/v2/cacheutils"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	"github.com/kubeflow/pipelines/backend/src/v2/objectstore"
	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	runID        = "11111111-1111-4111-8111-111111111111"
	orphanRunID  = "22222222-2222-4222-8222-222222222222"
	namespace    = "migration-fixture"
	pipelineName = "migration-contract"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func raw(message proto.Message) json.RawMessage {
	b, err := protojson.Marshal(message)
	must(err)
	return b
}

func main() {
	// An archive has no .git directory. Verify the source files/dependency pins
	// so running in another checkout cannot falsely label its output as 2.17.3.
	for path, expected := range map[string]string{
		"go.mod":                              "2017f283e017f3eb68f499bee1367a3a3d1cdb3926c9bc24716a4aba977aafe6",
		"go.sum":                              "4083b567a2f9aed27dacfdac5d6e7df7eda7dd488e96327e121da1c1c6689e13",
		"backend/src/v2/metadata/client.go":   "e40e17b5681f53ee55841a5c4e7f7a007977bf9164c902dd7bd92dd0eadca482",
		"backend/src/v2/cacheutils/cache.go":  "024d365bdc68617418620ac1c69455c55a7bef01c45598f93fa209dd7dcdc6b1",
		"backend/src/apiserver/model/task.go": "ccd5ccbffe0eadc61038672e0206bef56347d850a97ba4f052049fc31b2e182f",
	} {
		data, err := os.ReadFile(path)
		must(err)
		if fmt.Sprintf("%x", sha256.Sum256(data)) != expected {
			panic("source differs from pinned 2.17.3: " + path)
		}
	}
	// A fixed loopback endpoint prevents this tool from being aimed at a cluster.
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	conn, err := grpc.NewClient("127.0.0.1:18080", grpc.WithTransportCredentials(insecure.NewCredentials()))
	must(err)
	defer conn.Close()
	store := pb.NewMetadataStoreServiceClient(conn)
	existing, err := store.GetExecutions(ctx, &pb.GetExecutionsRequest{})
	must(err)
	artifactsBefore, err := store.GetArtifacts(ctx, &pb.GetArtifactsRequest{})
	must(err)
	contextsBefore, err := store.GetContexts(ctx, &pb.GetContextsRequest{})
	must(err)
	if len(existing.Executions)+len(artifactsBefore.Artifacts)+len(contextsBefore.Contexts) != 0 {
		panic("fixture generation requires an empty disposable MLMD store")
	}
	client, err := metadata.NewClient("127.0.0.1", "18080", nil)
	must(err)
	pipeline, err := client.GetPipeline(ctx, pipelineName, runID, namespace, "workflows.argoproj.io/fixture", "s3://fixture/root", "{}")
	must(err)
	cases := map[string]int64{}
	create := func(name string, config metadata.ExecutionConfig) *metadata.Execution {
		config.TaskName = name
		config.DisplayName = name
		config.Name = name
		execution, err := client.CreateExecution(ctx, pipeline, &config)
		must(err)
		cases[name] = execution.Execution.GetId()
		return execution
	}
	finish := func(execution *metadata.Execution, state pb.Execution_State, artifacts ...*metadata.OutputArtifact) {
		must(client.PublishExecution(ctx, execution, map[string]*structpb.Value{"result": structpb.NewStringValue("done")}, artifacts, state))
	}
	root := create("root", metadata.ExecutionConfig{ExecutionType: metadata.DagExecutionTypeName})
	rootID := root.Execution.GetId()
	nested := create("nested", metadata.ExecutionConfig{ExecutionType: metadata.DagExecutionTypeName, ParentDagID: rootID})
	create("skipped", metadata.ExecutionConfig{ExecutionType: metadata.ContainerExecutionTypeName, ParentDagID: rootID, NotTriggered: true})

	record := func(name, kind string, fields map[string]interface{}) *metadata.OutputArtifact {
		md, err := structpb.NewStruct(fields)
		must(err)
		schema := "title: " + kind + "\ntype: object\n"
		artifact, err := client.RecordArtifact(ctx, name, schema, &pipelinespec.RuntimeArtifact{
			Uri:      "s3://fixture/" + name,
			Type:     &pipelinespec.ArtifactTypeSchema{Kind: &pipelinespec.ArtifactTypeSchema_InstanceSchema{InstanceSchema: schema}},
			Metadata: md,
		}, pb.Artifact_LIVE, &objectstore.Config{})
		must(err)
		cases[name] = artifact.Artifact.GetId()
		return artifact
	}
	dataset := record("dataset", "system.Dataset", map[string]interface{}{"rows": 12.0, "description": "shared input"})
	metrics := record("metrics", "system.Metrics", map[string]interface{}{"accuracy": 0.875, "loss": 0.125, "note": "keep nonnumeric metadata"})
	emptyMetrics := record("empty-metrics", "system.Metrics", map[string]interface{}{"note": "no measurements"})
	importer := create("importer", metadata.ExecutionConfig{ExecutionType: metadata.ImporterExecutionTypeName, ParentDagID: rootID})
	finish(importer, pb.Execution_COMPLETE, dataset)
	producer := create("producer", metadata.ExecutionConfig{ExecutionType: metadata.ContainerExecutionTypeName, ParentDagID: nested.Execution.GetId(), Image: "fixture/train:v1", PodName: "producer-pod", PodUID: "producer-uid", Namespace: namespace, InputParameters: map[string]*structpb.Value{"epochs": structpb.NewNumberValue(3)}})
	finish(producer, pb.Execution_COMPLETE, metrics, emptyMetrics)
	cached := create("cached", metadata.ExecutionConfig{ExecutionType: metadata.ContainerExecutionTypeName, ParentDagID: rootID, CachedMLMDExecutionID: strconv.FormatInt(producer.Execution.GetId(), 10)})
	finish(cached, pb.Execution_CACHED, metrics)

	runtimeDataset, err := dataset.ToRuntimeArtifact()
	must(err)
	runtimeMetrics, err := metrics.ToRuntimeArtifact()
	must(err)
	inputs := &pipelinespec.ExecutorInput_Inputs{Artifacts: map[string]*pipelinespec.ArtifactList{
		"dataset": {Artifacts: []*pipelinespec.RuntimeArtifact{runtimeDataset}},
		"metrics": {Artifacts: []*pipelinespec.RuntimeArtifact{runtimeMetrics}},
	}, ParameterValues: map[string]*structpb.Value{"epochs": structpb.NewNumberValue(3)}}
	outputs := &pipelinespec.ExecutorInput_Outputs{}
	cache, err := cacheutils.NewClient("127.0.0.1", "1", false, nil)
	must(err)
	key, err := cache.GenerateCacheKey(inputs, outputs, nil, []string{"python", "train.py"}, "fixture/train:v1", nil)
	must(err)
	fingerprint, err := cache.GenerateFingerPrint(key)
	must(err)
	consumer := create("consumer", metadata.ExecutionConfig{ExecutionType: metadata.ContainerExecutionTypeName, ParentDagID: rootID, FingerPrint: fingerprint, Image: "fixture/train:v1", Namespace: namespace, InputParameters: inputs.ParameterValues, InputArtifactIDs: map[string][]int64{"dataset": {dataset.Artifact.GetId()}, "metrics": {metrics.Artifact.GetId()}}})
	finish(consumer, pb.Execution_COMPLETE)
	count := 2
	loop := create("loop", metadata.ExecutionConfig{ExecutionType: metadata.DagExecutionTypeName, ParentDagID: rootID, IterationCount: &count})
	for i := 0; i < count; i++ {
		iteration := create(fmt.Sprintf("iteration-%d", i), metadata.ExecutionConfig{ExecutionType: metadata.ContainerExecutionTypeName, ParentDagID: loop.Execution.GetId(), IterationIndex: &i, InputArtifactIDs: map[string][]int64{"dataset": {dataset.Artifact.GetId()}}})
		finish(iteration, pb.Execution_COMPLETE)
	}
	failed := create("failed", metadata.ExecutionConfig{ExecutionType: metadata.ContainerExecutionTypeName, ParentDagID: rootID})
	finish(failed, pb.Execution_FAILED)
	// An execution in an unowned run creates a real gap in the retained ID
	// sequence. A later execution in the retained run must still be scanned.
	orphan, err := client.GetPipeline(ctx, pipelineName, orphanRunID, namespace, "workflows.argoproj.io/orphan", "s3://fixture/root", "{}")
	must(err)
	gap, err := client.CreateExecution(ctx, orphan, &metadata.ExecutionConfig{TaskName: "gap", Name: "gap", ExecutionType: metadata.DagExecutionTypeName})
	must(err)
	cases["gap"] = gap.Execution.GetId()
	late := create("late", metadata.ExecutionConfig{ExecutionType: metadata.ContainerExecutionTypeName, ParentDagID: rootID})
	finish(late, pb.Execution_COMPLETE)
	finish(loop, pb.Execution_COMPLETE)
	finish(nested, pb.Execution_COMPLETE)
	finish(root, pb.Execution_COMPLETE)
	// Deliberately omit the matching KFP run from the destination fixture.
	orphanExecution, err := client.CreateExecution(ctx, orphan, &metadata.ExecutionConfig{TaskName: "orphan", Name: "orphan", ExecutionType: metadata.DagExecutionTypeName})
	must(err)
	cases["orphan"] = orphanExecution.Execution.GetId()

	executions, err := store.GetExecutions(ctx, &pb.GetExecutionsRequest{})
	must(err)
	artifacts, err := store.GetArtifacts(ctx, &pb.GetArtifactsRequest{})
	must(err)
	contexts, err := store.GetContexts(ctx, &pb.GetContextsRequest{})
	must(err)
	executionTypes, err := store.GetExecutionTypes(ctx, &pb.GetExecutionTypesRequest{})
	must(err)
	artifactTypes, err := store.GetArtifactTypes(ctx, &pb.GetArtifactTypesRequest{})
	must(err)
	contextTypes, err := store.GetContextTypes(ctx, &pb.GetContextTypesRequest{})
	must(err)
	if executions.GetNextPageToken() != "" || artifacts.GetNextPageToken() != "" || contexts.GetNextPageToken() != "" {
		panic("fixture unexpectedly exceeded one MLMD result page")
	}
	sort.Slice(executions.Executions, func(i, j int) bool { return executions.Executions[i].GetId() < executions.Executions[j].GetId() })
	sort.Slice(artifacts.Artifacts, func(i, j int) bool { return artifacts.Artifacts[i].GetId() < artifacts.Artifacts[j].GetId() })
	sort.Slice(contexts.Contexts, func(i, j int) bool { return contexts.Contexts[i].GetId() < contexts.Contexts[j].GetId() })
	executionContexts := map[string]json.RawMessage{}
	ids := []int64{}
	// Normalize only wall-clock fields. Preserve IDs, types, values and relationships.
	for _, execution := range executions.Executions {
		execution.CreateTimeSinceEpoch = proto.Int64(1700000000123)
		execution.LastUpdateTimeSinceEpoch = proto.Int64(1700000001456)
		ids = append(ids, execution.GetId())
		associations, err := store.GetContextsByExecution(ctx, &pb.GetContextsByExecutionRequest{ExecutionId: execution.Id})
		must(err)
		for _, c := range associations.Contexts {
			c.CreateTimeSinceEpoch = proto.Int64(1700000000123)
			c.LastUpdateTimeSinceEpoch = proto.Int64(1700000001456)
		}
		sort.Slice(associations.Contexts, func(i, j int) bool { return associations.Contexts[i].GetId() < associations.Contexts[j].GetId() })
		executionContexts[strconv.FormatInt(execution.GetId(), 10)] = raw(associations)
	}
	for _, a := range artifacts.Artifacts {
		a.CreateTimeSinceEpoch = proto.Int64(1700000000123)
		a.LastUpdateTimeSinceEpoch = proto.Int64(1700000001456)
	}
	for _, c := range contexts.Contexts {
		c.CreateTimeSinceEpoch = proto.Int64(1700000000123)
		c.LastUpdateTimeSinceEpoch = proto.Int64(1700000001456)
	}
	events, err := store.GetEventsByExecutionIDs(ctx, &pb.GetEventsByExecutionIDsRequest{ExecutionIds: ids})
	must(err)
	for _, event := range events.Events {
		event.MillisecondsSinceEpoch = proto.Int64(1700000001000)
	}
	sort.Slice(events.Events, func(i, j int) bool { return string(raw(events.Events[i])) < string(raw(events.Events[j])) })

	fixture := map[string]interface{}{
		"source_commit":  "a0733cb51b0484a6050cafee989ffe3312f808ad",
		"source_version": "2.17.3", "mlmd_version": "1.14.0", "cases": cases,
		"run":        map[string]string{"uuid": runID, "namespace": namespace, "pipeline_name": pipelineName},
		"executions": raw(executions), "artifacts": raw(artifacts), "events": raw(events), "contexts": raw(contexts),
		"execution_types": raw(executionTypes), "artifact_types": raw(artifactTypes), "context_types": raw(contextTypes),
		"execution_contexts": executionContexts,
		"cache":              map[string]interface{}{"inputs": raw(inputs), "outputs": raw(outputs), "command": []string{"python", "train.py"}, "image": "fixture/train:v1", "key": raw(key), "fingerprint": fingerprint},
		// This is a source-model sample, not a database catalog or an executed run.
		"legacy_task": model.Task{UUID: "33333333-3333-4333-8333-333333333333", RunID: runID, Namespace: namespace, MLMDExecutionID: strconv.FormatInt(consumer.Execution.GetId(), 10), Fingerprint: fingerprint, CreatedTimestamp: 1700000000},
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	must(encoder.Encode(fixture))
}
