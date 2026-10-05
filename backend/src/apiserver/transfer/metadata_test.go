// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0

package transfer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// descriptorConn exercises the production ProtoRPC against the pinned generated
// request/response types. A misspelled or unsupported field fails protojson here.
type descriptorConn struct{ store *memoryMetadata }

func (c descriptorConn) Invoke(ctx context.Context, method string, args, reply any, _ ...grpc.CallOption) error {
	data, err := (protojson.MarshalOptions{UseProtoNames: true}).Marshal(args.(proto.Message))
	if err != nil {
		return err
	}
	var fields Node
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&fields); err != nil {
		return err
	}
	parts := strings.Split(method, "/")
	result, err := c.store.Call(ctx, parts[len(parts)-1], fields)
	if err != nil {
		return err
	}
	data, err = json.Marshal(result)
	if err != nil {
		return err
	}
	return protojson.Unmarshal(data, reply.(proto.Message))
}
func (c descriptorConn) NewStream(context.Context, *grpc.StreamDesc, string, ...grpc.CallOption) (grpc.ClientStream, error) {
	return nil, errors.New("streaming is unused")
}

type memoryMetadata struct {
	graph Graph
	next  int
	puts  int
}

func (f *memoryMetadata) Call(_ context.Context, method string, fields Node) (Node, error) {
	if f.graph == nil {
		f.graph = Graph{}
	}
	if f.next == 0 {
		f.next = 1000
	}
	if strings.HasPrefix(method, "Put") {
		f.puts++
	}
	for collection, kind := range metadataKinds {
		tc := typeCollection(collection)
		switch method {
		case "Get" + kind + "Type":
			for _, row := range f.graph[tc] {
				if sid(row["name"]) == sid(fields["type_name"]) {
					return Node{strings.ToLower(kind) + "_type": row}, nil
				}
			}
			return Node{}, nil
		case "Get" + kind + "TypesByID":
			var rows []Node
			for _, id := range array(fields["type_ids"]) {
				for _, row := range f.graph[tc] {
					if sid(row["id"]) == sid(id) {
						rows = append(rows, row)
					}
				}
			}
			return Node{tc: rows}, nil
		case "Put" + kind + "Type":
			row := clone(object(fields[strings.ToLower(kind)+"_type"]))
			f.next++
			row["id"] = strconv.Itoa(f.next)
			f.graph[tc] = append(f.graph[tc], row)
			return Node{"type_id": row["id"]}, nil
		case "Get" + kind + "ByTypeAndName":
			typeID := ""
			for _, typ := range f.graph[tc] {
				if sid(typ["name"]) == sid(fields["type_name"]) {
					typeID = sid(typ["id"])
				}
			}
			for _, row := range f.graph[collection] {
				if sid(row["type_id"]) == typeID && sid(row["name"]) == sid(fields[strings.ToLower(kind)+"_name"]) {
					return Node{strings.ToLower(kind): row}, nil
				}
			}
			return Node{}, nil
		case "Get" + kind + "sByID":
			var rows []Node
			for _, id := range array(fields[strings.TrimSuffix(collection, "s")+"_ids"]) {
				for _, row := range f.graph[collection] {
					if sid(row["id"]) == sid(id) {
						rows = append(rows, row)
					}
				}
			}
			return Node{collection: rows}, nil
		case "Put" + kind + "s":
			var ids []any
			for _, input := range nodes(fields[collection]) {
				row := clone(input)
				id := sid(row["id"])
				if id == "" {
					f.next++
					id = strconv.Itoa(f.next)
					row["id"] = id
				}
				found := false
				for i, old := range f.graph[collection] {
					if sid(old["id"]) == id {
						f.graph[collection][i] = row
						found = true
					}
				}
				if !found {
					f.graph[collection] = append(f.graph[collection], row)
				}
				ids = append(ids, id)
			}
			return Node{strings.TrimSuffix(collection, "s") + "_ids": ids}, nil
		}
	}
	switch method {
	case "GetExecutionsByContext", "GetArtifactsByContext":
		collection, edges, field := "executions", "associations", "execution_id"
		if method == "GetArtifactsByContext" {
			collection, edges, field = "artifacts", "attributions", "artifact_id"
		}
		var result []Node
		for _, edge := range f.graph[edges] {
			if sid(edge["context_id"]) == sid(fields["context_id"]) {
				for _, row := range f.graph[collection] {
					if sid(row["id"]) == sid(edge[field]) {
						result = append(result, row)
					}
				}
			}
		}
		return Node{collection: result}, nil
	case "GetContextsByExecution", "GetContextsByArtifact":
		edges, field := "associations", "execution_id"
		if method == "GetContextsByArtifact" {
			edges, field = "attributions", "artifact_id"
		}
		var result []Node
		for _, edge := range f.graph[edges] {
			if sid(edge[field]) == sid(fields[field]) {
				for _, row := range f.graph["contexts"] {
					if sid(row["id"]) == sid(edge["context_id"]) {
						result = append(result, row)
					}
				}
			}
		}
		return Node{"contexts": result}, nil
	case "GetEventsByExecutionIDs", "GetEventsByArtifactIDs":
		field := "execution_id"
		if method == "GetEventsByArtifactIDs" {
			field = "artifact_id"
		}
		var rows []Node
		for _, id := range array(fields[field+"s"]) {
			for _, row := range f.graph["events"] {
				if sid(row[field]) == sid(id) {
					rows = append(rows, row)
				}
			}
		}
		return Node{"events": rows}, nil
	case "PutEvents":
		f.graph["events"] = append(f.graph["events"], nodes(fields["events"])...)
		return Node{}, nil
	case "PutAttributionsAndAssociations":
		for _, collection := range []string{"associations", "attributions"} {
			for _, row := range nodes(fields[collection]) {
				found := false
				for _, old := range f.graph[collection] {
					if hash(row) == hash(old) {
						found = true
					}
				}
				if !found {
					f.graph[collection] = append(f.graph[collection], row)
				}
			}
		}
		return Node{}, nil
	case "GetParentContextsByContext":
		var rows []Node
		for _, edge := range f.graph["parents"] {
			if sid(edge["child_id"]) == sid(fields["context_id"]) {
				for _, row := range f.graph["contexts"] {
					if sid(row["id"]) == sid(edge["parent_id"]) {
						rows = append(rows, row)
					}
				}
			}
		}
		return Node{"contexts": rows}, nil
	case "PutParentContexts":
		f.graph["parents"] = append(f.graph["parents"], nodes(fields["parent_contexts"])...)
		return Node{}, nil
	default:
		return nil, errors.New("unexpected RPC " + method)
	}
}

func lineageGraph(runID string) Graph {
	return Graph{
		"context_types":   {{"id": "1", "name": "system.PipelineRun"}, {"id": "2", "name": "system.Pipeline"}},
		"execution_types": {{"id": "3", "name": "system.ContainerExecution"}},
		"artifact_types":  {{"id": "4", "name": "system.Artifact"}},
		"contexts":        {{"id": "5", "type_id": "1", "name": runID}, {"id": "6", "type_id": "2", "name": "pipeline"}},
		"executions":      {{"id": "10", "type_id": "3", "name": "child", "last_known_state": "COMPLETE", "custom_properties": Node{"parent_dag_id": Node{"int_value": "11"}, "cached_execution_id": Node{"string_value": "12"}, "cache_fingerprint": Node{"string_value": "source-fingerprint"}}}, {"id": "11", "type_id": "3", "name": "parent", "last_known_state": "COMPLETE"}, {"id": "12", "type_id": "3", "name": "cached", "last_known_state": "COMPLETE"}},
		"artifacts":       {{"id": "20", "type_id": "4", "uri": "s3://shared-bucket/unchanged"}},
		"events":          {{"execution_id": "10", "artifact_id": "20", "type": "OUTPUT", "milliseconds_since_epoch": "9007199254740993"}},
		"associations":    {{"context_id": "5", "execution_id": "10"}, {"context_id": "6", "execution_id": "11"}, {"context_id": "5", "execution_id": "12"}},
		"attributions":    {{"context_id": "5", "artifact_id": "20"}},
		"parents":         {{"parent_id": "6", "child_id": "5"}},
	}
}

func TestGeneratedMLMDContractExportStageAndRetry(t *testing.T) {
	ctx := context.Background()
	runID := uuid.NewString()
	sourceStore := &memoryMetadata{graph: lineageGraph(runID)}
	source := Metadata{RPC: ProtoRPC{Conn: descriptorConn{sourceStore}}}
	runs := []RunHistory{{Run: model.Run{UUID: runID, Namespace: "team", RunDetails: model.RunDetails{PipelineRuntimeManifest: "v2"}}, Tasks: []model.Task{{UUID: "task", RunID: runID, Namespace: "team", MLMDExecutionID: "10", Fingerprint: "source-cache", MLMDOutputs: `{"artifacts":{"artifactIds":[20]}}`}}}}
	var authorized []string
	graph, err := source.Export(ctx, runs, func(id string) error { authorized = append(authorized, id); return nil })
	require.NoError(t, err)
	require.Equal(t, []string{runID}, authorized)
	require.Len(t, graph["executions"], 3)
	require.EqualValues(t, 5, runs[0].Run.PipelineRunContextId)
	targetStore := &memoryMetadata{}
	target := Metadata{RPC: ProtoRPC{Conn: descriptorConn{targetStore}}}
	require.NoError(t, target.Preflight(ctx, graph, "source"))
	require.Zero(t, targetStore.puts)
	mapping, err := target.Stage(ctx, graph, "source")
	require.NoError(t, err)
	require.NotEqual(t, "10", mapping["executions"]["10"])
	require.NoError(t, remapMetadata(&runs[0], mapping))
	require.Empty(t, runs[0].Tasks[0].Fingerprint)
	require.Equal(t, mapping["executions"]["10"], runs[0].Tasks[0].MLMDExecutionID)
	require.Contains(t, string(runs[0].Tasks[0].MLMDOutputs), mapping["artifacts"]["20"])
	require.Equal(t, "s3://shared-bucket/unchanged", targetStore.graph["artifacts"][0]["uri"])
	_, err = target.Stage(ctx, graph, "source")
	require.NoError(t, err)
	require.Len(t, targetStore.graph["contexts"], 2)
	require.Len(t, targetStore.graph["executions"], 3)
	require.Len(t, targetStore.graph["events"], 1)
	require.Len(t, targetStore.graph["parents"], 1)
	var imported Node
	for _, node := range targetStore.graph["executions"] {
		if sid(node["id"]) == mapping["executions"]["10"] {
			imported = node
		}
	}
	props := object(imported["custom_properties"])
	require.Equal(t, mapping["executions"]["11"], sid(object(props["parent_dag_id"])["int_value"]))
	require.Equal(t, mapping["executions"]["12"], sid(object(props["cached_execution_id"])["string_value"]))
	require.NotEqual(t, "source-fingerprint", sid(object(props["cache_fingerprint"])["string_value"]))
	require.Equal(t, "9007199254740993", sid(targetStore.graph["events"][0]["milliseconds_since_epoch"]))
}

func TestMLMDRejectsNamespaceProvenanceAndTypeConflicts(t *testing.T) {
	ctx := context.Background()
	g := lineageGraph("run")
	g["contexts"] = append(g["contexts"], Node{"id": "7", "type_id": "1", "name": "other-namespace-run"})
	g["associations"] = append(g["associations"], Node{"context_id": "7", "execution_id": "12"})
	m := Metadata{RPC: ProtoRPC{Conn: descriptorConn{&memoryMetadata{graph: g}}}}
	_, err := m.Export(ctx, []RunHistory{{Run: model.Run{UUID: "run"}}}, func(id string) error {
		if id != "run" {
			return errors.New("cross namespace")
		}
		return nil
	})
	require.ErrorContains(t, err, "cross namespace")
	target := &memoryMetadata{graph: Graph{"execution_types": {{"id": "30", "name": "system.ContainerExecution", "properties": Node{"different": "STRING"}}}}}
	m.RPC = ProtoRPC{Conn: descriptorConn{target}}
	require.ErrorContains(t, m.Preflight(ctx, lineageGraph("run"), "source"), "schema conflicts")
	require.Zero(t, target.puts)
	target = &memoryMetadata{graph: Graph{"context_types": {{"id": "1", "name": "system.PipelineRun"}}, "contexts": {{"id": "5", "type_id": "1", "name": "run"}}}}
	m.RPC = ProtoRPC{Conn: descriptorConn{target}}
	require.ErrorContains(t, m.Preflight(ctx, lineageGraph("run"), "source"), "provenance")
	require.Zero(t, target.puts)
}

func TestReexportImportedMetadataUsesRecordedNamespace(t *testing.T) {
	ctx := context.Background()
	runID := uuid.NewString()
	store := &memoryMetadata{}
	metadata := Metadata{RPC: ProtoRPC{Conn: descriptorConn{store}}, Namespace: "team"}
	mapping, err := metadata.Stage(ctx, lineageGraph(runID), "source")
	require.NoError(t, err)
	runs := []RunHistory{{Run: model.Run{UUID: runID, RunDetails: model.RunDetails{PipelineRuntimeManifest: "v2"}}}}
	graph, err := metadata.Export(ctx, runs, func(string) error { return errors.New("ancestor producer SQL row was not included in prior batch") })
	require.NoError(t, err)
	require.NotEmpty(t, graph["executions"])
	require.Equal(t, mapping["contexts"]["5"], strconv.FormatInt(runs[0].Run.PipelineRunContextId, 10))
	for _, node := range graph["contexts"] {
		for key := range object(node["custom_properties"]) {
			require.False(t, strings.HasPrefix(key, provenance))
		}
	}
	metadata.Namespace = "other"
	_, err = metadata.Export(ctx, runs, func(string) error { return errors.New("cross namespace") })
	require.ErrorContains(t, err, "cross namespace")
}

// The CI service supplies a real MLMD server; the API server needs no Python runtime.
func TestTransferLiveMLMDIntegration(t *testing.T) {
	address := os.Getenv("KFP_TRANSFER_MLMD_ADDRESS")
	if address == "" {
		if os.Getenv("KFP_TRANSFER_REQUIRE_INTEGRATION") == "1" {
			t.Fatal("KFP_TRANSFER_MLMD_ADDRESS is required")
		}
		t.Skip("live MLMD fixture not configured")
	}
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()
	m := Metadata{RPC: ProtoRPC{Conn: conn}}
	graph := lineageGraph(uuid.NewString())
	source := uuid.NewString()
	require.NoError(t, m.Preflight(context.Background(), graph, source))
	mapping, err := m.Stage(context.Background(), graph, source)
	require.NoError(t, err)
	again, err := m.Stage(context.Background(), graph, source)
	require.NoError(t, err)
	require.Equal(t, mapping, again)
	resp, err := m.call(context.Background(), "GetContextByTypeAndName", Node{"type_name": "system.PipelineRun", "context_name": graph["contexts"][0]["name"]})
	require.NoError(t, err)
	require.Equal(t, mapping["contexts"]["5"], sid(object(resp["context"])["id"]))
}
