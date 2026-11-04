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

package mlmd

import (
	"context"
	"fmt"
	"sort"
	"time"

	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"google.golang.org/protobuf/proto"
)

// Snapshot is an ordered MLMD source graph and its execution-to-context links.
type Snapshot struct {
	Executions []*pb.Execution
	Artifacts  []*pb.Artifact
	Events     []*pb.Event
	Contexts   map[int64][]*pb.Context
}

// ReadSnapshot fetches all pages and relationships from the source store. The
// caller must quiesce source writers before using it for the final migration.
func ReadSnapshot(ctx context.Context, client pb.MetadataStoreServiceClient) (*Snapshot, error) {
	const pageSize int32 = 100
	snapshot := &Snapshot{Contexts: map[int64][]*pb.Context{}}
	seenTokens := map[string]bool{}
	for token := ""; ; {
		callContext, cancel := context.WithTimeout(ctx, 30*time.Second)
		response, err := client.GetExecutions(callContext, &pb.GetExecutionsRequest{Options: &pb.ListOperationOptions{MaxResultSize: proto.Int32(pageSize), NextPageToken: proto.String(token)}})
		cancel()
		if err != nil {
			return nil, fmt.Errorf("list MLMD executions: %w", err)
		}
		snapshot.Executions = append(snapshot.Executions, response.GetExecutions()...)
		token = response.GetNextPageToken()
		if token == "" {
			break
		}
		if seenTokens[token] {
			return nil, fmt.Errorf("MLMD execution pagination repeated token")
		}
		seenTokens[token] = true
	}
	seenTokens = map[string]bool{}
	for token := ""; ; {
		callContext, cancel := context.WithTimeout(ctx, 30*time.Second)
		response, err := client.GetArtifacts(callContext, &pb.GetArtifactsRequest{Options: &pb.ListOperationOptions{MaxResultSize: proto.Int32(pageSize), NextPageToken: proto.String(token)}})
		cancel()
		if err != nil {
			return nil, fmt.Errorf("list MLMD artifacts: %w", err)
		}
		snapshot.Artifacts = append(snapshot.Artifacts, response.GetArtifacts()...)
		token = response.GetNextPageToken()
		if token == "" {
			break
		}
		if seenTokens[token] {
			return nil, fmt.Errorf("MLMD artifact pagination repeated token")
		}
		seenTokens[token] = true
	}
	sort.Slice(snapshot.Executions, func(i, j int) bool { return snapshot.Executions[i].GetId() < snapshot.Executions[j].GetId() })
	sort.Slice(snapshot.Artifacts, func(i, j int) bool { return snapshot.Artifacts[i].GetId() < snapshot.Artifacts[j].GetId() })
	for index := 1; index < len(snapshot.Executions); index++ {
		if snapshot.Executions[index-1].GetId() == snapshot.Executions[index].GetId() {
			return nil, fmt.Errorf("MLMD execution pages contain duplicate ID %d", snapshot.Executions[index].GetId())
		}
	}
	for index := 1; index < len(snapshot.Artifacts); index++ {
		if snapshot.Artifacts[index-1].GetId() == snapshot.Artifacts[index].GetId() {
			return nil, fmt.Errorf("MLMD artifact pages contain duplicate ID %d", snapshot.Artifacts[index].GetId())
		}
	}
	missingExecutionType := false
	for _, execution := range snapshot.Executions {
		if execution.GetType() == "" {
			missingExecutionType = true
			break
		}
	}
	if missingExecutionType {
		callContext, cancel := context.WithTimeout(ctx, 30*time.Second)
		response, err := client.GetExecutionTypes(callContext, &pb.GetExecutionTypesRequest{})
		cancel()
		if err != nil {
			return nil, fmt.Errorf("list MLMD execution types: %w", err)
		}
		types := map[int64]string{}
		for _, entry := range response.GetExecutionTypes() {
			types[entry.GetId()] = entry.GetName()
		}
		for _, execution := range snapshot.Executions {
			if execution.GetType() == "" {
				execution.Type = proto.String(types[execution.GetTypeId()])
			}
		}
	}
	missingArtifactType := false
	for _, artifact := range snapshot.Artifacts {
		if artifact.GetType() == "" {
			missingArtifactType = true
			break
		}
	}
	if missingArtifactType {
		callContext, cancel := context.WithTimeout(ctx, 30*time.Second)
		response, err := client.GetArtifactTypes(callContext, &pb.GetArtifactTypesRequest{})
		cancel()
		if err != nil {
			return nil, fmt.Errorf("list MLMD artifact types: %w", err)
		}
		types := map[int64]string{}
		for _, entry := range response.GetArtifactTypes() {
			types[entry.GetId()] = entry.GetName()
		}
		for _, artifact := range snapshot.Artifacts {
			if artifact.GetType() == "" {
				artifact.Type = proto.String(types[artifact.GetTypeId()])
			}
		}
	}
	for start := 0; start < len(snapshot.Executions); start += 50 {
		end := start + 50
		if end > len(snapshot.Executions) {
			end = len(snapshot.Executions)
		}
		ids := make([]int64, 0, end-start)
		for _, execution := range snapshot.Executions[start:end] {
			ids = append(ids, execution.GetId())
			callContext, cancel := context.WithTimeout(ctx, 30*time.Second)
			response, err := client.GetContextsByExecution(callContext, &pb.GetContextsByExecutionRequest{ExecutionId: proto.Int64(execution.GetId())})
			cancel()
			if err != nil {
				return nil, fmt.Errorf("contexts for MLMD execution %d: %w", execution.GetId(), err)
			}
			snapshot.Contexts[execution.GetId()] = response.GetContexts()
		}
		callContext, cancel := context.WithTimeout(ctx, 30*time.Second)
		response, err := client.GetEventsByExecutionIDs(callContext, &pb.GetEventsByExecutionIDsRequest{ExecutionIds: ids})
		cancel()
		if err != nil {
			return nil, fmt.Errorf("events for MLMD executions: %w", err)
		}
		snapshot.Events = append(snapshot.Events, response.GetEvents()...)
	}
	missingContextType := false
	for _, contexts := range snapshot.Contexts {
		for _, entry := range contexts {
			if entry.GetType() == "" {
				missingContextType = true
				break
			}
		}
	}
	if missingContextType {
		callContext, cancel := context.WithTimeout(ctx, 30*time.Second)
		response, err := client.GetContextTypes(callContext, &pb.GetContextTypesRequest{})
		cancel()
		if err != nil {
			return nil, fmt.Errorf("list MLMD context types: %w", err)
		}
		types := map[int64]string{}
		for _, entry := range response.GetContextTypes() {
			types[entry.GetId()] = entry.GetName()
		}
		for _, contexts := range snapshot.Contexts {
			for _, entry := range contexts {
				if entry.GetType() == "" {
					entry.Type = proto.String(types[entry.GetTypeId()])
				}
			}
		}
	}
	return snapshot, nil
}
