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

// Run from the pinned legacy checkout after the local driver and launcher
// fixture tests. This exports their records without writing to MLMD.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"time"

	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	connection, err := grpc.NewClient("127.0.0.1:18081", grpc.WithTransportCredentials(insecure.NewCredentials()))
	checkRuntime(err)
	defer connection.Close()
	store := pb.NewMetadataStoreServiceClient(connection)
	executions, err := store.GetExecutions(ctx, &pb.GetExecutionsRequest{})
	checkRuntime(err)
	artifacts, err := store.GetArtifacts(ctx, &pb.GetArtifactsRequest{})
	checkRuntime(err)
	contexts, err := store.GetContexts(ctx, &pb.GetContextsRequest{})
	checkRuntime(err)
	if len(executions.Executions) != 3 || len(artifacts.Artifacts) != 1 || executions.GetNextPageToken() != "" || artifacts.GetNextPageToken() != "" || contexts.GetNextPageToken() != "" {
		panic("expected only the root, skipped DAG and importer launcher records in the disposable MLMD store")
	}
	sort.Slice(executions.Executions, func(i, j int) bool { return executions.Executions[i].GetId() < executions.Executions[j].GetId() })
	sort.Slice(artifacts.Artifacts, func(i, j int) bool { return artifacts.Artifacts[i].GetId() < artifacts.Artifacts[j].GetId() })
	sort.Slice(contexts.Contexts, func(i, j int) bool { return contexts.Contexts[i].GetId() < contexts.Contexts[j].GetId() })
	ids := make([]int64, 0, len(executions.Executions))
	associations := map[string]json.RawMessage{}
	for _, execution := range executions.Executions {
		ids = append(ids, execution.GetId())
		execution.CreateTimeSinceEpoch = proto.Int64(1700000000123)
		execution.LastUpdateTimeSinceEpoch = proto.Int64(1700000001456)
		response, err := store.GetContextsByExecution(ctx, &pb.GetContextsByExecutionRequest{ExecutionId: execution.Id})
		checkRuntime(err)
		for _, associated := range response.Contexts {
			associated.CreateTimeSinceEpoch = proto.Int64(1700000000123)
			associated.LastUpdateTimeSinceEpoch = proto.Int64(1700000001456)
		}
		sort.Slice(response.Contexts, func(i, j int) bool { return response.Contexts[i].GetId() < response.Contexts[j].GetId() })
		associations[strconv.FormatInt(execution.GetId(), 10)] = runtimeJSON(response)
	}
	for _, artifact := range artifacts.Artifacts {
		artifact.CreateTimeSinceEpoch = proto.Int64(1700000000123)
		artifact.LastUpdateTimeSinceEpoch = proto.Int64(1700000001456)
	}
	for _, context := range contexts.Contexts {
		context.CreateTimeSinceEpoch = proto.Int64(1700000000123)
		context.LastUpdateTimeSinceEpoch = proto.Int64(1700000001456)
	}
	events, err := store.GetEventsByExecutionIDs(ctx, &pb.GetEventsByExecutionIDsRequest{ExecutionIds: ids})
	checkRuntime(err)
	for _, event := range events.Events {
		event.MillisecondsSinceEpoch = proto.Int64(1700000001000)
	}
	sort.Slice(events.Events, func(i, j int) bool {
		return string(runtimeJSON(events.Events[i])) < string(runtimeJSON(events.Events[j]))
	})
	fixture := map[string]any{
		"source_commit": "a0733cb51b0484a6050cafee989ffe3312f808ad",
		"source_kind":   "pinned driver.RootDAG, driver.DAG and component.ImportLauncher run locally against MLMD; no Argo workflow was submitted",
		"run_uuid":      "22222222-2222-4222-8222-222222222222",
		"executions":    runtimeJSON(executions), "artifacts": runtimeJSON(artifacts),
		"contexts": runtimeJSON(contexts), "events": runtimeJSON(events),
		"execution_contexts": associations,
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	checkRuntime(encoder.Encode(fixture))
}

func runtimeJSON(message proto.Message) json.RawMessage {
	encoded, err := protojson.Marshal(message)
	checkRuntime(err)
	return encoded
}

func checkRuntime(err error) {
	if err != nil {
		panic(fmt.Errorf("runtime fixture export: %w", err))
	}
}
