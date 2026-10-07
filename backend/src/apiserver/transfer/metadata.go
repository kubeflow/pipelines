// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0

package transfer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	_ "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

type Node map[string]any
type Graph map[string][]Node
type IDMapping map[string]map[string]string

var metadataKinds = map[string]string{"contexts": "Context", "executions": "Execution", "artifacts": "Artifact"}
var executionReferences = map[string]string{"parent_dag_id": "int_value", "cached_execution_id": "string_value"}

const provenance = "kfp_transfer_"

// MetadataRPC isolates the generated protobuf wire boundary for deterministic tests.
type MetadataRPC interface {
	Call(context.Context, string, Node) (Node, error)
}
type Metadata struct {
	RPC       MetadataRPC
	Namespace string
}

// ProtoRPC uses the pinned generated descriptors, including strict protobuf JSON validation.
type ProtoRPC struct{ Conn grpc.ClientConnInterface }

func (p ProtoRPC) Call(ctx context.Context, method string, fields Node) (Node, error) {
	reqType, err := protoregistry.GlobalTypes.FindMessageByName(protoreflect.FullName("ml_metadata." + method + "Request"))
	if err != nil {
		return nil, err
	}
	respType, err := protoregistry.GlobalTypes.FindMessageByName(protoreflect.FullName("ml_metadata." + method + "Response"))
	if err != nil {
		return nil, err
	}
	req, resp := reqType.New().Interface(), respType.New().Interface()
	data, err := json.Marshal(fields)
	if err != nil {
		return nil, err
	}
	if err := protojson.Unmarshal(data, req); err != nil {
		return nil, err
	}
	err = p.Conn.Invoke(ctx, "/ml_metadata.MetadataStoreService/"+method, req, resp)
	if status.Code(err) == codes.NotFound && strings.HasPrefix(method, "Get") {
		return Node{}, nil
	}
	if err != nil {
		return nil, err
	}
	data, err = (protojson.MarshalOptions{UseProtoNames: true}).Marshal(resp)
	if err != nil {
		return nil, err
	}
	var result Node
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	err = decoder.Decode(&result)
	return result, err
}

func sid(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}
func object(value any) Node {
	if n, ok := value.(Node); ok {
		return n
	}
	if n, ok := value.(map[string]any); ok {
		return Node(n)
	}
	return Node{}
}
func nodes(value any) []Node {
	if n, ok := value.([]Node); ok {
		return n
	}
	var result []Node
	for _, v := range array(value) {
		result = append(result, object(v))
	}
	return result
}
func array(value any) []any { a, _ := value.([]any); return a }
func clone(n Node) Node {
	data, _ := json.Marshal(n)
	var out Node
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	_ = decoder.Decode(&out)
	return out
}
func nodeDigest(n Node) string {
	n = clone(n)
	delete(n, "create_time_since_epoch")
	delete(n, "last_update_time_since_epoch")
	return hash(n)
}
func typeCollection(collection string) string { return strings.TrimSuffix(collection, "s") + "_types" }
func positiveID(id string) bool               { n, err := strconv.ParseInt(id, 10, 64); return err == nil && n > 0 }
func acyclic(edges [][2]string) error {
	in := map[string]int{}
	children := map[string][]string{}
	seen := map[[2]string]bool{}
	for _, e := range edges {
		if seen[e] {
			continue
		}
		seen[e] = true
		if _, ok := in[e[0]]; !ok {
			in[e[0]] = 0
		}
		in[e[1]]++
		children[e[0]] = append(children[e[0]], e[1])
	}
	var queue []string
	for id, n := range in {
		if n == 0 {
			queue = append(queue, id)
		}
	}
	visited := 0
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		visited++
		for _, child := range children[id] {
			in[child]--
			if in[child] == 0 {
				queue = append(queue, child)
			}
		}
	}
	if visited != len(in) {
		return util.NewInvalidInputError("history relationship graph contains a cycle")
	}
	return nil
}

func validateGraph(g Graph, runs []RunHistory) error {
	allowed := map[string]string{"contexts": "Context", "executions": "Execution", "artifacts": "Artifact", "context_types": "ContextType", "execution_types": "ExecutionType", "artifact_types": "ArtifactType", "events": "Event", "associations": "Association", "attributions": "Attribution", "parents": "ParentContext"}
	ids := IDMapping{}
	count := 0
	for collection, rows := range g {
		name, ok := allowed[collection]
		if !ok {
			return util.NewInvalidInputError("unknown metadata graph collection")
		}
		ids[collection] = map[string]string{}
		for _, row := range rows {
			mt, err := protoregistry.GlobalTypes.FindMessageByName(protoreflect.FullName("ml_metadata." + name))
			if err != nil {
				return err
			}
			data, err := json.Marshal(row)
			if err != nil {
				return err
			}
			if err := protojson.Unmarshal(data, mt.New().Interface()); err != nil {
				return util.NewInvalidInputError("Invalid metadata %s: %v", collection, err)
			}
			if metadataKinds[collection] != "" || strings.HasSuffix(collection, "_types") {
				id := sid(row["id"])
				if !positiveID(id) || ids[collection][id] != "" {
					return util.NewInvalidInputError("invalid or duplicate metadata ID")
				}
				ids[collection][id] = id
				count++
			}
			for key := range object(row["custom_properties"]) {
				if strings.HasPrefix(key, provenance) || strings.HasPrefix(key, "kfp_history_") {
					return util.NewInvalidInputError("archive includes previously imported metadata")
				}
			}
		}
	}
	if count > maxRecords {
		return util.NewInvalidInputError("metadata exceeds 100000 nodes")
	}
	check := func(collection string, id any) error {
		if ids[collection][sid(id)] == "" {
			return fmt.Errorf("metadata graph contains missing %s reference", collection)
		}
		return nil
	}
	var executionEdges, contextEdges [][2]string
	for collection := range metadataKinds {
		for _, node := range g[collection] {
			if err := check(typeCollection(collection), node["type_id"]); err != nil {
				return err
			}
		}
	}
	for _, node := range g["executions"] {
		switch sid(node["last_known_state"]) {
		case "COMPLETE", "FAILED", "CACHED", "CANCELED":
		default:
			return util.NewInvalidInputError("metadata includes unfinished execution")
		}
		for key, field := range executionReferences {
			value := object(object(node["custom_properties"])[key])[field]
			if sid(value) != "" && sid(value) != "0" {
				if err := check("executions", value); err != nil {
					return err
				}
				executionEdges = append(executionEdges, [2]string{sid(value), sid(node["id"])})
			}
		}
	}
	for collection, fields := range map[string]map[string]string{"events": {"execution_id": "executions", "artifact_id": "artifacts"}, "associations": {"execution_id": "executions", "context_id": "contexts"}, "attributions": {"artifact_id": "artifacts", "context_id": "contexts"}, "parents": {"parent_id": "contexts", "child_id": "contexts"}} {
		for _, edge := range g[collection] {
			for field, target := range fields {
				if err := check(target, edge[field]); err != nil {
					return err
				}
			}
			if collection == "parents" {
				contextEdges = append(contextEdges, [2]string{sid(edge["parent_id"]), sid(edge["child_id"])})
			}
		}
	}
	if err := acyclic(executionEdges); err != nil {
		return err
	}
	if err := acyclic(contextEdges); err != nil {
		return err
	}
	for _, h := range runs {
		for _, id := range []int64{h.Run.PipelineContextId, h.Run.PipelineRunContextId} {
			if id != 0 {
				if err := check("contexts", id); err != nil {
					return err
				}
			}
		}
		for _, task := range h.Tasks {
			if task.MLMDExecutionID != "" && task.MLMDExecutionID != "0" {
				if err := check("executions", task.MLMDExecutionID); err != nil {
					return err
				}
			}
			for _, text := range []model.LargeText{task.MLMDInputs, task.MLMDOutputs} {
				if _, err := mapArtifacts(text, ids["artifacts"]); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (m *Metadata) call(ctx context.Context, method string, fields Node) (Node, error) {
	if m == nil || m.RPC == nil {
		return nil, util.NewInvalidInputError("metadata service unavailable")
	}
	return m.RPC.Call(ctx, method, fields)
}
func (m *Metadata) fetch(ctx context.Context, collection string, ids []string) ([]Node, error) {
	var result []Node
	seen := map[string]bool{}
	var wanted []string
	for _, id := range ids {
		if id != "" && id != "0" && !seen[id] {
			seen[id] = true
			wanted = append(wanted, id)
		}
	}
	for start := 0; start < len(wanted); start += 100 {
		end := start + 100
		if end > len(wanted) {
			end = len(wanted)
		}
		part := wanted[start:end]
		resp, err := m.call(ctx, "Get"+metadataKinds[collection]+"sByID", Node{strings.TrimSuffix(collection, "s") + "_ids": part})
		if err != nil {
			return nil, err
		}
		rows := nodes(resp[collection])
		if len(rows) != len(part) {
			return nil, util.NewInvalidInputError("source metadata is missing referenced nodes")
		}
		result = append(result, rows...)
	}
	return result, nil
}
func (m *Metadata) contextNodes(ctx context.Context, collection, id string) ([]Node, error) {
	var rows []Node
	token := ""
	for {
		options := Node{"max_result_size": 100}
		if token != "" {
			options["next_page_token"] = token
		}
		r, err := m.call(ctx, "Get"+metadataKinds[collection]+"sByContext", Node{"context_id": id, "options": options})
		if err != nil {
			return nil, err
		}
		rows = append(rows, nodes(r[collection])...)
		if len(rows) > maxRecords {
			return nil, util.NewInvalidInputError("metadata context exceeds transfer node limit")
		}
		next := sid(r["next_page_token"])
		if next == "" {
			return rows, nil
		}
		if next == token {
			return nil, util.NewInvalidInputError("metadata returned a repeated page token")
		}
		token = next
	}
}

// Export follows artifact producers and execution parents, never unrelated consumers.
func (m *Metadata) Export(ctx context.Context, runs []RunHistory, authorizeRun func(string) error) (Graph, error) {
	g := Graph{}
	all := map[string]map[string]Node{"contexts": {}, "executions": {}, "artifacts": {}}
	graphBytes := 0
	var graphError error
	add := func(collection string, rows []Node) {
		for _, row := range rows {
			if all[collection][sid(row["id"])] == nil {
				data, _ := json.Marshal(row)
				graphBytes += len(data)
			}
			if graphBytes > MaxArchiveBytes {
				graphError = util.NewInvalidInputError("Metadata exceeds transfer size limit; use a smaller completion-time interval")
				return
			}
			all[collection][sid(row["id"])] = row
		}
	}
	for i := range runs {
		if graphError != nil {
			return nil, graphError
		}
		h := &runs[i]
		resp, err := m.call(ctx, "GetContextByTypeAndName", Node{"type_name": "system.PipelineRun", "context_name": h.Run.UUID})
		if err != nil {
			return nil, err
		}
		runContext := object(resp["context"])
		if len(runContext) > 0 {
			id := sid(runContext["id"])
			h.Run.PipelineRunContextId, _ = strconv.ParseInt(id, 10, 64)
			add("contexts", []Node{runContext})
			for _, collection := range []string{"executions", "artifacts"} {
				rows, err := m.contextNodes(ctx, collection, id)
				if err != nil {
					return nil, err
				}
				add(collection, rows)
			}
		} else if h.Run.PipelineRuntimeManifest != "" {
			return nil, util.NewInvalidInputError("completed v2 run has no metadata run context")
		}
		rows, err := m.fetch(ctx, "contexts", []string{strconv.FormatInt(h.Run.PipelineContextId, 10), strconv.FormatInt(h.Run.PipelineRunContextId, 10)})
		if err != nil {
			return nil, err
		}
		add("contexts", rows)
		var execIDs []string
		for _, task := range h.Tasks {
			execIDs = append(execIDs, task.MLMDExecutionID)
		}
		rows, err = m.fetch(ctx, "executions", execIDs)
		if err != nil {
			return nil, err
		}
		add("executions", rows)
	}
	processed := map[string]map[string]bool{"contexts": {}, "executions": {}, "artifacts": {}}
	for {
		if graphError != nil {
			return nil, graphError
		}
		progress := false
		if len(all["contexts"])+len(all["executions"])+len(all["artifacts"]) > maxRecords {
			return nil, util.NewInvalidInputError("metadata graph exceeds transfer limit")
		}
		for id, node := range all["executions"] {
			if processed["executions"][id] {
				continue
			}
			progress = true
			processed["executions"][id] = true
			var refs []string
			for key, field := range executionReferences {
				value := sid(object(object(node["custom_properties"])[key])[field])
				if all["executions"][value] == nil {
					refs = append(refs, value)
				}
			}
			rows, err := m.fetch(ctx, "executions", refs)
			if err != nil {
				return nil, err
			}
			add("executions", rows)
			resp, err := m.call(ctx, "GetEventsByExecutionIDs", Node{"execution_ids": []string{id}})
			if err != nil {
				return nil, err
			}
			events := nodes(resp["events"])
			g["events"] = append(g["events"], events...)
			refs = nil
			for _, event := range events {
				aid := sid(event["artifact_id"])
				if all["artifacts"][aid] == nil {
					refs = append(refs, aid)
				}
			}
			rows, err = m.fetch(ctx, "artifacts", refs)
			if err != nil {
				return nil, err
			}
			add("artifacts", rows)
			resp, err = m.call(ctx, "GetContextsByExecution", Node{"execution_id": id})
			if err != nil {
				return nil, err
			}
			rows = nodes(resp["contexts"])
			add("contexts", rows)
			for _, row := range rows {
				g["associations"] = append(g["associations"], Node{"context_id": sid(row["id"]), "execution_id": id})
			}
		}
		for id := range all["artifacts"] {
			if processed["artifacts"][id] {
				continue
			}
			progress = true
			processed["artifacts"][id] = true
			resp, err := m.call(ctx, "GetEventsByArtifactIDs", Node{"artifact_ids": []string{id}})
			if err != nil {
				return nil, err
			}
			var refs []string
			for _, event := range nodes(resp["events"]) {
				switch sid(event["type"]) {
				case "OUTPUT", "DECLARED_OUTPUT", "INTERNAL_OUTPUT":
					eid := sid(event["execution_id"])
					if all["executions"][eid] == nil {
						refs = append(refs, eid)
					}
				}
			}
			rows, err := m.fetch(ctx, "executions", refs)
			if err != nil {
				return nil, err
			}
			add("executions", rows)
			resp, err = m.call(ctx, "GetContextsByArtifact", Node{"artifact_id": id})
			if err != nil {
				return nil, err
			}
			rows = nodes(resp["contexts"])
			add("contexts", rows)
			for _, row := range rows {
				g["attributions"] = append(g["attributions"], Node{"context_id": sid(row["id"]), "artifact_id": id})
			}
		}
		if !progress {
			break
		}
	}
	for {
		if graphError != nil {
			return nil, graphError
		}
		progress := false
		for id := range all["contexts"] {
			if processed["contexts"][id] {
				continue
			}
			progress = true
			processed["contexts"][id] = true
			resp, err := m.call(ctx, "GetParentContextsByContext", Node{"context_id": id})
			if err != nil {
				return nil, err
			}
			rows := nodes(resp["contexts"])
			add("contexts", rows)
			for _, row := range rows {
				g["parents"] = append(g["parents"], Node{"parent_id": sid(row["id"]), "child_id": id})
			}
		}
		if len(all["contexts"]) > maxRecords {
			return nil, util.NewInvalidInputError("metadata context graph exceeds transfer limit")
		}
		if !progress {
			break
		}
	}
	for collection, items := range all {
		types := map[string]bool{}
		for _, node := range items {
			g[collection] = append(g[collection], node)
			types[sid(node["type_id"])] = true
		}
		var ids []string
		for id := range types {
			ids = append(ids, id)
		}
		if len(ids) > 0 {
			resp, err := m.call(ctx, "Get"+metadataKinds[collection]+"TypesByID", Node{"type_ids": ids})
			if err != nil {
				return nil, err
			}
			g[typeCollection(collection)] = nodes(resp[typeCollection(collection)])
		}
	}
	for key, rows := range g {
		unique := map[string]Node{}
		for _, row := range rows {
			data, _ := json.Marshal(row)
			unique[string(data)] = row
		}
		var keys []string
		for k := range unique {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		g[key] = nil
		for _, k := range keys {
			g[key] = append(g[key], unique[k])
		}
	}
	contextTypes := map[string]string{}
	for _, t := range g["context_types"] {
		contextTypes[sid(t["id"])] = sid(t["name"])
	}
	for _, c := range g["contexts"] {
		if contextTypes[sid(c["type_id"])] == "system.PipelineRun" {
			if err := authorizeRun(sid(c["name"])); err != nil {
				props := object(c["custom_properties"])
				ns, exists := props[provenance+"namespace"]
				if !exists || sid(object(ns)["string_value"]) != m.Namespace || sid(object(props[provenance+"source"])["string_value"]) == "" {
					return nil, err
				}
			}
		}
	}
	for collection := range metadataKinds {
		for _, node := range g[collection] {
			props := object(node["custom_properties"])
			for key := range props {
				if strings.HasPrefix(key, provenance) || strings.HasPrefix(key, "kfp_history_") {
					delete(props, key)
				}
			}
		}
	}
	return g, validateGraph(g, runs)
}

func typeCompatible(a, b Node) bool {
	normalize := func(n Node) Node {
		result := Node{"name": "", "version": "", "properties": Node{}, "base_type": "UNSET", "input_type": Node{}, "output_type": Node{}}
		for key := range result {
			if value, ok := n[key]; ok {
				result[key] = value
			}
		}
		return result
	}
	return hash(normalize(a)) == hash(normalize(b))
}
func importedName(collection string, node, definition Node, source string) string {
	if collection == "contexts" && sid(definition["name"]) == "system.PipelineRun" {
		return sid(node["name"])
	}
	return "kfp-transfer-" + hash(source)[:24] + "-" + collection + "-" + sid(node["id"])
}
func (m *Metadata) typeByName(ctx context.Context, kind string, definition Node) (Node, error) {
	fields := Node{"type_name": definition["name"]}
	if v := definition["version"]; v != nil {
		fields["type_version"] = v
	}
	resp, err := m.call(ctx, "Get"+kind+"Type", fields)
	return object(resp[strings.ToLower(kind)+"_type"]), err
}
func (m *Metadata) nodeByName(ctx context.Context, collection, name string, definition Node) (Node, error) {
	kind := metadataKinds[collection]
	fields := Node{"type_name": definition["name"], strings.ToLower(kind) + "_name": name}
	if v := definition["version"]; v != nil {
		fields["type_version"] = v
	}
	resp, err := m.call(ctx, "Get"+kind+"ByTypeAndName", fields)
	return object(resp[strings.ToLower(kind)]), err
}
func definitions(g Graph, collection string) map[string]Node {
	result := map[string]Node{}
	for _, n := range g[typeCollection(collection)] {
		result[sid(n["id"])] = n
	}
	return result
}

func (m *Metadata) Preflight(ctx context.Context, g Graph, source string) error {
	for collection, kind := range metadataKinds {
		defs := definitions(g, collection)
		for _, def := range defs {
			current, err := m.typeByName(ctx, kind, def)
			if err != nil {
				return err
			}
			if len(current) > 0 && !typeCompatible(def, current) {
				return util.NewInvalidInputError("destination metadata type schema conflicts with archive")
			}
		}
		for _, node := range g[collection] {
			def := defs[sid(node["type_id"])]
			current, err := m.nodeByName(ctx, collection, importedName(collection, node, def, source), def)
			if err != nil {
				return err
			}
			if len(current) > 0 {
				props := object(current["custom_properties"])
				if sid(object(props[provenance+"source"])["string_value"]) != source || sid(object(props[provenance+"digest"])["string_value"]) != nodeDigest(node) {
					return util.NewInvalidInputError("destination metadata node has conflicting provenance")
				}
			}
		}
	}
	return nil
}

func (m *Metadata) Stage(ctx context.Context, g Graph, source string) (IDMapping, error) {
	if err := m.Preflight(ctx, g, source); err != nil {
		return nil, err
	}
	mapping := IDMapping{}
	payloads := map[string]map[string]Node{}
	for _, collection := range []string{"contexts", "executions", "artifacts"} {
		kind := metadataKinds[collection]
		mapping[collection] = map[string]string{}
		payloads[collection] = map[string]Node{}
		defs := definitions(g, collection)
		types := map[string]string{}
		for old, def := range defs {
			current, err := m.typeByName(ctx, kind, def)
			if err != nil {
				return nil, err
			}
			if len(current) > 0 {
				types[old] = sid(current["id"])
			} else {
				payload := clone(def)
				delete(payload, "id")
				resp, err := m.call(ctx, "Put"+kind+"Type", Node{strings.ToLower(kind) + "_type": payload})
				if err != nil {
					return nil, err
				}
				types[old] = sid(resp["type_id"])
			}
		}
		for _, node := range g[collection] {
			old := sid(node["id"])
			def := defs[sid(node["type_id"])]
			name := importedName(collection, node, def, source)
			current, err := m.nodeByName(ctx, collection, name, def)
			if err != nil {
				return nil, err
			}
			payload := clone(node)
			for _, field := range []string{"id", "type", "external_id", "system_metadata", "create_time_since_epoch", "last_update_time_since_epoch"} {
				delete(payload, field)
			}
			payload["type_id"] = types[sid(node["type_id"])]
			payload["name"] = name
			props := object(payload["custom_properties"])
			payload["custom_properties"] = props
			props[provenance+"source"] = Node{"string_value": source}
			props[provenance+"namespace"] = Node{"string_value": m.Namespace}
			props[provenance+"digest"] = Node{"string_value": nodeDigest(node)}
			original, _ := json.Marshal(node)
			props[provenance+"original"] = Node{"string_value": string(original)}
			for key := range executionReferences {
				delete(props, key)
			}
			if _, ok := props["cache_fingerprint"]; ok {
				props["cache_fingerprint"] = Node{"string_value": "transfer:" + source + ":" + old}
			}
			id := sid(current["id"])
			if id == "" {
				resp, err := m.call(ctx, "Put"+kind+"s", Node{collection: []Node{payload}})
				if err != nil {
					return nil, err
				}
				ids := array(resp[strings.TrimSuffix(collection, "s")+"_ids"])
				if len(ids) != 1 {
					return nil, util.NewInvalidInputError("metadata store returned invalid node ID")
				}
				id = sid(ids[0])
			}
			mapping[collection][old] = id
			payload["id"] = id
			payloads[collection][old] = payload
		}
	}
	for _, node := range g["executions"] {
		payload := payloads["executions"][sid(node["id"])]
		props := object(payload["custom_properties"])
		for key, field := range executionReferences {
			value := sid(object(object(node["custom_properties"])[key])[field])
			if value != "" && value != "0" {
				props[key] = Node{field: mapping["executions"][value]}
			}
		}
		if _, err := m.call(ctx, "PutExecutions", Node{"executions": []Node{payload}}); err != nil {
			return nil, err
		}
	}
	for old, id := range mapping["executions"] {
		resp, err := m.call(ctx, "GetEventsByExecutionIDs", Node{"execution_ids": []string{id}})
		if err != nil {
			return nil, err
		}
		eventKey := func(n Node) string { n = clone(n); delete(n, "milliseconds_since_epoch"); return hash(n) }
		seen := map[string]bool{}
		for _, event := range nodes(resp["events"]) {
			seen[eventKey(event)] = true
		}
		for _, event := range g["events"] {
			if sid(event["execution_id"]) != old {
				continue
			}
			row := clone(event)
			row["execution_id"] = id
			row["artifact_id"] = mapping["artifacts"][sid(event["artifact_id"])]
			key := eventKey(row)
			if !seen[key] {
				if _, err := m.call(ctx, "PutEvents", Node{"events": []Node{row}}); err != nil {
					return nil, err
				}
				seen[key] = true
			}
		}
	}
	fields := Node{}
	for collection, target := range map[string]string{"associations": "executions", "attributions": "artifacts"} {
		var edges []Node
		field := strings.TrimSuffix(target, "s") + "_id"
		for _, edge := range g[collection] {
			edges = append(edges, Node{"context_id": mapping["contexts"][sid(edge["context_id"])], field: mapping[target][sid(edge[field])]})
		}
		fields[collection] = edges
	}
	if len(g["associations"])+len(g["attributions"]) > 0 {
		if _, err := m.call(ctx, "PutAttributionsAndAssociations", fields); err != nil {
			return nil, err
		}
	}
	for _, edge := range g["parents"] {
		parent := mapping["contexts"][sid(edge["parent_id"])]
		child := mapping["contexts"][sid(edge["child_id"])]
		resp, err := m.call(ctx, "GetParentContextsByContext", Node{"context_id": child})
		if err != nil {
			return nil, err
		}
		found := false
		for _, node := range nodes(resp["contexts"]) {
			if sid(node["id"]) == parent {
				found = true
			}
		}
		if !found {
			if _, err := m.call(ctx, "PutParentContexts", Node{"parent_contexts": []Node{{"parent_id": parent, "child_id": child}}}); err != nil {
				return nil, err
			}
		}
	}
	return mapping, nil
}

func mapArtifacts(text model.LargeText, mapping map[string]string) (model.LargeText, error) {
	if text == "" {
		return text, nil
	}
	var root any
	dec := json.NewDecoder(strings.NewReader(string(text)))
	dec.UseNumber()
	if err := dec.Decode(&root); err != nil {
		return "", util.NewInvalidInputError("Invalid task artifact JSON: %v", err)
	}
	var visit func(any) error
	visit = func(v any) error {
		switch node := v.(type) {
		case map[string]any:
			for key, child := range node {
				if key == "artifact_ids" || key == "artifactIds" {
					items, ok := child.([]any)
					if !ok {
						return util.NewInvalidInputError("invalid artifact ID list")
					}
					for i, item := range items {
						id := mapping[sid(item)]
						if id == "" {
							return util.NewInvalidInputError("task refers to absent metadata artifact")
						}
						items[i] = json.Number(id)
					}
				} else if err := visit(child); err != nil {
					return err
				}
			}
		case []any:
			for _, child := range node {
				if err := visit(child); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := visit(root); err != nil {
		return "", err
	}
	data, err := json.Marshal(root)
	return model.LargeText(data), err
}
func remapMetadata(h *RunHistory, m IDMapping) error {
	for _, p := range []*int64{&h.Run.PipelineContextId, &h.Run.PipelineRunContextId} {
		if *p != 0 {
			id := m["contexts"][strconv.FormatInt(*p, 10)]
			value, err := strconv.ParseInt(id, 10, 64)
			if err != nil {
				return util.NewInvalidInputError("run refers to absent metadata context")
			}
			*p = value
		}
	}
	for i := range h.Tasks {
		t := &h.Tasks[i]
		t.Fingerprint = ""
		if t.MLMDExecutionID != "" && t.MLMDExecutionID != "0" {
			id := m["executions"][t.MLMDExecutionID]
			if id == "" {
				return util.NewInvalidInputError("task refers to absent metadata execution")
			}
			t.MLMDExecutionID = id
		}
		var err error
		t.MLMDInputs, err = mapArtifacts(t.MLMDInputs, m["artifacts"])
		if err != nil {
			return err
		}
		t.MLMDOutputs, err = mapArtifacts(t.MLMDOutputs, m["artifacts"])
		if err != nil {
			return err
		}
		if t.Payload != "" {
			var payload model.Task
			if err := json.Unmarshal([]byte(t.Payload), &payload); err != nil {
				return err
			}
			payload.Fingerprint = ""
			payload.MLMDExecutionID = t.MLMDExecutionID
			payload.MLMDInputs = t.MLMDInputs
			payload.MLMDOutputs = t.MLMDOutputs
			data, err := json.Marshal(payload)
			if err != nil {
				return err
			}
			t.Payload = model.LargeText(data)
		}
	}
	return nil
}
