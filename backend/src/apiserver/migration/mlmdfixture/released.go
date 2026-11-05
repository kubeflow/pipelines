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

// Package mlmdfixture serves the released 2.18 source graph to migration tests.
package mlmdfixture

import (
	"context"
	"encoding/json"
	"os"

	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

type graph struct {
	Runs []struct {
		ID string `json:"id"`
	} `json:"runs"`
	Types []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"types"`
	Contexts []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"contexts"`
	Executions []struct {
		ID        int    `json:"id"`
		Name      string `json:"name"`
		TypeID    int    `json:"type_id"`
		State     int    `json:"state"`
		CreatedMS int64  `json:"created_ms"`
		UpdatedMS int64  `json:"updated_ms"`
	} `json:"executions"`
	Associations []struct {
		ContextID   int `json:"context_id"`
		ExecutionID int `json:"execution_id"`
	} `json:"associations"`
	Properties []struct {
		ExecutionID int    `json:"execution_id"`
		Name        string `json:"name"`
		IntValue    *int64 `json:"int_value"`
		StringValue string `json:"string_value"`
	} `json:"execution_properties"`
	Artifacts []struct {
		ID        int    `json:"id"`
		TypeID    int    `json:"type_id"`
		URI       string `json:"uri"`
		CreatedMS int64  `json:"created_ms"`
		UpdatedMS int64  `json:"updated_ms"`
	} `json:"artifacts"`
	ArtifactProperties []struct {
		ArtifactID  int      `json:"artifact_id"`
		Name        string   `json:"name"`
		DoubleValue *float64 `json:"double_value"`
		StringValue string   `json:"string_value"`
	} `json:"artifact_properties"`
	Events []struct {
		ID          int `json:"id"`
		ExecutionID int `json:"execution_id"`
		ArtifactID  int `json:"artifact_id"`
		Type        int `json:"type"`
	} `json:"events"`
	EventPaths []struct {
		EventID int    `json:"event_id"`
		StepKey string `json:"step_key"`
	} `json:"event_paths"`
}

// Source holds the pinned released graph for client and gRPC server tests.
type Source struct {
	pb.MetadataStoreServiceClient
	executions *pb.GetExecutionsResponse
	artifacts  *pb.GetArtifactsResponse
	events     *pb.GetEventsByExecutionIDsResponse
	contexts   map[int64]*pb.GetContextsByExecutionResponse
}

// Load reads the released graph captured alongside its KFP SQL and Workflow.
func Load(path string) (*Source, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var graph graph
	if err := json.Unmarshal(data, &graph); err != nil {
		return nil, err
	}
	source := &Source{executions: &pb.GetExecutionsResponse{}, artifacts: &pb.GetArtifactsResponse{}, events: &pb.GetEventsByExecutionIDsResponse{}, contexts: map[int64]*pb.GetContextsByExecutionResponse{}}
	types := map[int]string{}
	for _, entry := range graph.Types {
		types[entry.ID] = entry.Name
	}
	runIDs := map[string]bool{}
	for _, run := range graph.Runs {
		runIDs[run.ID] = true
	}
	contexts := map[int]*pb.Context{}
	for _, entry := range graph.Contexts {
		kind := "system.Pipeline"
		if runIDs[entry.Name] {
			kind = "system.PipelineRun"
		}
		contexts[entry.ID] = &pb.Context{Id: proto.Int64(int64(entry.ID)), Name: proto.String(entry.Name), Type: proto.String(kind)}
	}
	for _, association := range graph.Associations {
		id := int64(association.ExecutionID)
		if source.contexts[id] == nil {
			source.contexts[id] = &pb.GetContextsByExecutionResponse{}
		}
		source.contexts[id].Contexts = append(source.contexts[id].Contexts, contexts[association.ContextID])
	}
	executions := map[int]*pb.Execution{}
	for _, entry := range graph.Executions {
		state := pb.Execution_State(entry.State)
		execution := &pb.Execution{Id: proto.Int64(int64(entry.ID)), Name: proto.String(entry.Name), Type: proto.String(types[entry.TypeID]), LastKnownState: &state, CreateTimeSinceEpoch: proto.Int64(entry.CreatedMS), LastUpdateTimeSinceEpoch: proto.Int64(entry.UpdatedMS), CustomProperties: map[string]*pb.Value{}}
		source.executions.Executions = append(source.executions.Executions, execution)
		executions[entry.ID] = execution
	}
	for _, property := range graph.Properties {
		value := &pb.Value{Value: &pb.Value_StringValue{StringValue: property.StringValue}}
		if property.IntValue != nil {
			value.Value = &pb.Value_IntValue{IntValue: *property.IntValue}
		}
		executions[property.ExecutionID].CustomProperties[property.Name] = value
	}
	artifacts := map[int]*pb.Artifact{}
	for _, entry := range graph.Artifacts {
		artifact := &pb.Artifact{Id: proto.Int64(int64(entry.ID)), Type: proto.String(types[entry.TypeID]), Uri: proto.String(entry.URI), CreateTimeSinceEpoch: proto.Int64(entry.CreatedMS), LastUpdateTimeSinceEpoch: proto.Int64(entry.UpdatedMS), CustomProperties: map[string]*pb.Value{}}
		source.artifacts.Artifacts = append(source.artifacts.Artifacts, artifact)
		artifacts[entry.ID] = artifact
	}
	for _, property := range graph.ArtifactProperties {
		value := &pb.Value{Value: &pb.Value_StringValue{StringValue: property.StringValue}}
		if property.DoubleValue != nil {
			value.Value = &pb.Value_DoubleValue{DoubleValue: *property.DoubleValue}
		}
		artifacts[property.ArtifactID].CustomProperties[property.Name] = value
	}
	paths := map[int]*pb.Event_Path{}
	for _, path := range graph.EventPaths {
		if paths[path.EventID] == nil {
			paths[path.EventID] = &pb.Event_Path{}
		}
		paths[path.EventID].Steps = append(paths[path.EventID].Steps, &pb.Event_Path_Step{Value: &pb.Event_Path_Step_Key{Key: path.StepKey}})
	}
	for _, entry := range graph.Events {
		kind := pb.Event_Type(entry.Type)
		source.events.Events = append(source.events.Events, &pb.Event{ArtifactId: proto.Int64(int64(entry.ArtifactID)), ExecutionId: proto.Int64(int64(entry.ExecutionID)), Type: &kind, Path: paths[entry.ID]})
	}
	return source, nil
}

func (source *Source) GetExecutions(context.Context, *pb.GetExecutionsRequest, ...grpc.CallOption) (*pb.GetExecutionsResponse, error) {
	return source.executions, nil
}
func (source *Source) GetArtifacts(context.Context, *pb.GetArtifactsRequest, ...grpc.CallOption) (*pb.GetArtifactsResponse, error) {
	return source.artifacts, nil
}
func (source *Source) GetContextsByExecution(_ context.Context, request *pb.GetContextsByExecutionRequest, _ ...grpc.CallOption) (*pb.GetContextsByExecutionResponse, error) {
	return source.contexts[request.GetExecutionId()], nil
}
func (source *Source) GetEventsByExecutionIDs(context.Context, *pb.GetEventsByExecutionIDsRequest, ...grpc.CallOption) (*pb.GetEventsByExecutionIDsResponse, error) {
	return source.events, nil
}

// Server serves the same fixture over gRPC for entrypoint tests.
type Server struct {
	pb.UnimplementedMetadataStoreServiceServer
	Source *Source
}

func (server Server) GetExecutions(ctx context.Context, request *pb.GetExecutionsRequest) (*pb.GetExecutionsResponse, error) {
	return server.Source.GetExecutions(ctx, request)
}
func (server Server) GetArtifacts(ctx context.Context, request *pb.GetArtifactsRequest) (*pb.GetArtifactsResponse, error) {
	return server.Source.GetArtifacts(ctx, request)
}
func (server Server) GetContextsByExecution(ctx context.Context, request *pb.GetContextsByExecutionRequest) (*pb.GetContextsByExecutionResponse, error) {
	return server.Source.GetContextsByExecution(ctx, request)
}
func (server Server) GetEventsByExecutionIDs(ctx context.Context, request *pb.GetEventsByExecutionIDsRequest) (*pb.GetEventsByExecutionIDsResponse, error) {
	return server.Source.GetEventsByExecutionIDs(ctx, request)
}
