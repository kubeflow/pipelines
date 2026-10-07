// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

package history

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	api "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/transfer"
	mlmd "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	statuspb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"google.golang.org/protobuf/types/known/timestamppb"
)

type legacyMLMDGraph struct {
	contexts       map[int64]*mlmd.Context
	contextTypes   map[int64]*mlmd.ContextType
	executions     map[int64]*mlmd.Execution
	executionTypes map[int64]*mlmd.ExecutionType
	artifacts      map[int64]*mlmd.Artifact
	artifactTypes  map[int64]*mlmd.ArtifactType
	events         []*mlmd.Event
	associations   []*mlmd.Association
	attributions   []*mlmd.Attribution
	parents        []*mlmd.ParentContext
	contextJSON    map[int64][]any
}

func legacyMLMDID(source, kind, id string) string {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte("kfp-2.18-transfer\x00"+source+"\x00"+kind+"\x00"+id)).String()
}

func legacyMLMDJSON(message proto.Message) string {
	data, _ := (protojson.MarshalOptions{UseProtoNames: true}).Marshal(message)
	var value any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if decoder.Decode(&value) != nil {
		return string(data)
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return string(data)
	}
	return string(canonical)
}

func decodeLegacyMLMD(g legacyGraph) (*legacyMLMDGraph, error) {
	out := &legacyMLMDGraph{contexts: map[int64]*mlmd.Context{}, contextTypes: map[int64]*mlmd.ContextType{}, executions: map[int64]*mlmd.Execution{}, executionTypes: map[int64]*mlmd.ExecutionType{}, artifacts: map[int64]*mlmd.Artifact{}, artifactTypes: map[int64]*mlmd.ArtifactType{}}
	count := 0
	for kind, rows := range g {
		for _, row := range rows {
			count++
			if count > 100000 {
				return nil, fmt.Errorf("legacy metadata exceeds 100000 records")
			}
			var message proto.Message
			switch kind {
			case "contexts":
				message = &mlmd.Context{}
			case "context_types":
				message = &mlmd.ContextType{}
			case "executions":
				message = &mlmd.Execution{}
			case "execution_types":
				message = &mlmd.ExecutionType{}
			case "artifacts":
				message = &mlmd.Artifact{}
			case "artifact_types":
				message = &mlmd.ArtifactType{}
			case "events":
				message = &mlmd.Event{}
			case "associations":
				message = &mlmd.Association{}
			case "attributions":
				message = &mlmd.Attribution{}
			case "parents":
				message = &mlmd.ParentContext{}
			default:
				return nil, fmt.Errorf("unsupported legacy metadata collection %q", kind)
			}
			data, err := json.Marshal(row)
			if err != nil {
				return nil, err
			}
			if err = protojson.Unmarshal(data, message); err != nil {
				return nil, fmt.Errorf("invalid legacy %s: %w", kind, err)
			}
			duplicate := false
			var id int64
			switch m := message.(type) {
			case *mlmd.Context:
				id = m.GetId()
				duplicate = out.contexts[id] != nil
				out.contexts[id] = m
			case *mlmd.ContextType:
				id = m.GetId()
				duplicate = out.contextTypes[id] != nil
				out.contextTypes[id] = m
			case *mlmd.Execution:
				id = m.GetId()
				duplicate = out.executions[id] != nil
				out.executions[id] = m
			case *mlmd.ExecutionType:
				id = m.GetId()
				duplicate = out.executionTypes[id] != nil
				out.executionTypes[id] = m
			case *mlmd.Artifact:
				id = m.GetId()
				duplicate = out.artifacts[id] != nil
				out.artifacts[id] = m
			case *mlmd.ArtifactType:
				id = m.GetId()
				duplicate = out.artifactTypes[id] != nil
				out.artifactTypes[id] = m
			case *mlmd.Event:
				out.events = append(out.events, m)
				continue
			case *mlmd.Association:
				out.associations = append(out.associations, m)
				continue
			case *mlmd.Attribution:
				out.attributions = append(out.attributions, m)
				continue
			case *mlmd.ParentContext:
				out.parents = append(out.parents, m)
				continue
			}
			if id <= 0 || duplicate {
				return nil, fmt.Errorf("invalid or duplicate legacy %s ID", kind)
			}
		}
		// Reject unsupported empty collections as well.
		switch kind {
		case "contexts", "context_types", "executions", "execution_types", "artifacts", "artifact_types", "events", "associations", "attributions", "parents":
		default:
			return nil, fmt.Errorf("unsupported legacy metadata collection %q", kind)
		}
	}
	for _, c := range out.contexts {
		if out.contextTypes[c.GetTypeId()] == nil {
			return nil, fmt.Errorf("legacy context has missing type")
		}
	}
	for _, e := range out.executions {
		if out.executionTypes[e.GetTypeId()] == nil {
			return nil, fmt.Errorf("legacy execution has missing type")
		}
	}
	for _, a := range out.artifacts {
		if out.artifactTypes[a.GetTypeId()] == nil {
			return nil, fmt.Errorf("legacy artifact has missing type")
		}
	}
	for _, a := range out.associations {
		if out.contexts[a.GetContextId()] == nil || out.executions[a.GetExecutionId()] == nil {
			return nil, fmt.Errorf("legacy association has missing endpoint")
		}
	}
	for _, a := range out.attributions {
		if out.contexts[a.GetContextId()] == nil || out.artifacts[a.GetArtifactId()] == nil {
			return nil, fmt.Errorf("legacy attribution has missing endpoint")
		}
	}
	for _, e := range out.events {
		if out.executions[e.GetExecutionId()] == nil || out.artifacts[e.GetArtifactId()] == nil {
			return nil, fmt.Errorf("legacy event has missing endpoint")
		}
	}
	edges := map[int64][]int64{}
	for _, p := range out.parents {
		if out.contexts[p.GetParentId()] == nil || out.contexts[p.GetChildId()] == nil {
			return nil, fmt.Errorf("legacy context parent has missing endpoint")
		}
		edges[p.GetChildId()] = append(edges[p.GetChildId()], p.GetParentId())
	}
	if err := legacyMLMDAcyclic(edges); err != nil {
		return nil, err
	}
	out.contextJSON = map[int64][]any{}
	for id, c := range out.contexts {
		// Contexts are shared mutable namespace objects. Preserve identity, not
		// properties/timestamps that can change after this execution finishes.
		identity := &mlmd.Context{Id: proto.Int64(id), TypeId: proto.Int64(c.GetTypeId()), Name: proto.String(c.GetName())}
		typ := &mlmd.ContextType{Id: proto.Int64(c.GetTypeId()), Name: proto.String(out.contextTypes[c.GetTypeId()].GetName())}
		out.contextJSON[id] = []any{legacyMLMDJSON(identity), legacyMLMDJSON(typ)}
	}
	return out, nil
}

func legacyMLMDAcyclic(edges map[int64][]int64) error {
	// Iterative topological traversal avoids recursion on untrusted deep graphs.
	indegree := map[int64]int{}
	children := map[int64][]int64{}
	for child, parents := range edges {
		if _, ok := indegree[child]; !ok {
			indegree[child] = 0
		}
		for _, parent := range parents {
			indegree[child]++
			if _, ok := indegree[parent]; !ok {
				indegree[parent] = 0
			}
			children[parent] = append(children[parent], child)
		}
	}
	queue := []int64{}
	for id, n := range indegree {
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
			indegree[child]--
			if indegree[child] == 0 {
				queue = append(queue, child)
			}
		}
	}
	if visited != len(indegree) {
		return fmt.Errorf("legacy metadata relationship contains a cycle")
	}
	return nil
}

func legacyMLMDString(props map[string]*mlmd.Value, key string) (string, error) {
	v := props[key]
	if v == nil {
		return "", nil
	}
	if _, ok := v.Value.(*mlmd.Value_StringValue); !ok {
		return "", fmt.Errorf("legacy property %s must be a string", key)
	}
	return v.GetStringValue(), nil
}
func legacyMLMDInt(props map[string]*mlmd.Value, key string) (*int64, error) {
	v := props[key]
	if v == nil {
		return nil, nil
	}
	if _, ok := v.Value.(*mlmd.Value_IntValue); !ok {
		return nil, fmt.Errorf("legacy property %s must be an integer", key)
	}
	n := v.GetIntValue()
	return &n, nil
}
func legacyMLMDValue(v *mlmd.Value) (any, error) {
	if v == nil {
		return nil, fmt.Errorf("empty legacy metadata value")
	}
	switch p := v.Value.(type) {
	case *mlmd.Value_StringValue:
		return p.StringValue, nil
	case *mlmd.Value_DoubleValue:
		if math.IsNaN(p.DoubleValue) || math.IsInf(p.DoubleValue, 0) {
			return nil, fmt.Errorf("non-finite legacy metadata number")
		}
		return p.DoubleValue, nil
	case *mlmd.Value_IntValue:
		if p.IntValue > 1<<53 || p.IntValue < -(1<<53) {
			return nil, fmt.Errorf("legacy metadata integer cannot be represented exactly in native JSON")
		}
		return float64(p.IntValue), nil
	case *mlmd.Value_StructValue:
		fields := p.StructValue.GetFields()
		if len(fields) == 1 {
			if list := fields["list"]; list != nil && list.GetListValue() != nil {
				return list.AsInterface(), nil
			}
			if object := fields["struct"]; object != nil && object.GetStructValue() != nil {
				return object.AsInterface(), nil
			}
		}
		return p.StructValue.AsMap(), nil
	default:
		return nil, fmt.Errorf("unsupported legacy metadata value %T", v.Value)
	}
}

func legacyMLMDParameters(v *mlmd.Value, kind api.IOType) (model.JSONSlice, error) {
	if v == nil {
		return nil, nil
	}
	if _, ok := v.Value.(*mlmd.Value_StructValue); !ok {
		return nil, fmt.Errorf("legacy task parameters must be a struct")
	}
	keys := []string{}
	for k := range v.GetStructValue().GetFields() {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parameters := []*api.PipelineTask_InputOutputs_IOParameter{}
	for _, k := range keys {
		parameters = append(parameters, &api.PipelineTask_InputOutputs_IOParameter{ParameterKey: k, Type: kind, Value: v.GetStructValue().Fields[k]})
	}
	return model.ProtoSliceToJSONSlice(parameters)
}

func legacyMLMDState(state mlmd.Execution_State) (model.TaskStatus, error) {
	switch state {
	case mlmd.Execution_COMPLETE:
		return model.TaskStatus(api.PipelineTask_SUCCEEDED), nil
	case mlmd.Execution_CACHED:
		return model.TaskStatus(api.PipelineTask_CACHED), nil
	case mlmd.Execution_FAILED:
		return model.TaskStatus(api.PipelineTask_FAILED), nil
	case mlmd.Execution_CANCELED:
		return model.TaskStatus(api.PipelineTask_SKIPPED), nil
	default:
		return 0, fmt.Errorf("legacy graph contains unfinished or unsupported execution state")
	}
}

func legacyMLMDTaskType(e *mlmd.Execution, typeName, name string, parent *int64) (model.TaskType, error) {
	switch typeName {
	case "system.ContainerExecution":
		return model.TaskType(api.PipelineTask_RUNTIME), nil
	case "system.ImporterExecution":
		return model.TaskType(api.PipelineTask_IMPORTER), nil
	case "system.DAGExecution":
		switch {
		case parent == nil || *parent == 0:
			return model.TaskType(api.PipelineTask_ROOT), nil
		case strings.HasPrefix(name, "condition-branches-"):
			return model.TaskType(api.PipelineTask_CONDITION_BRANCH), nil
		case strings.HasPrefix(name, "condition-"):
			return model.TaskType(api.PipelineTask_CONDITION), nil
		case e.CustomProperties["iteration_count"] != nil:
			return model.TaskType(api.PipelineTask_LOOP), nil
		case strings.HasPrefix(name, "exit-handler-") || strings.HasPrefix(name, "on-exit-") || strings.HasPrefix(name, "onexit-"):
			return model.TaskType(api.PipelineTask_EXIT_HANDLER), nil
		default:
			return model.TaskType(api.PipelineTask_DAG), nil
		}
	default:
		return 0, fmt.Errorf("unsupported legacy execution type %q", typeName)
	}
}

// legacyMLMDContextProvenance keeps the execution's context identities stable
// across completion-time batches. Shared context properties and parent graph
// membership can evolve independently of completed executions.
func legacyMLMDContextProvenance(g *legacyMLMDGraph, ids []int64) []any {
	seen := map[int64]bool{}
	keys := []int64{}
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			keys = append(keys, id)
		}
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	result := []any{}
	for _, id := range keys {
		result = append(result, g.contextJSON[id]...)
	}
	return result
}

func convertLegacyMLMD(source, namespace string, runs []legacyRunHistory, graph legacyGraph, entries []Entry) ([]Entry, error) {
	budget := transfer.NewExportBudget(transfer.MaxArchiveBytes)
	g, err := decodeLegacyMLMD(graph)
	if err != nil {
		return nil, err
	}
	if len(runs) != len(entries) {
		return nil, fmt.Errorf("legacy run mapping is inconsistent")
	}
	runIndex := map[string]int{}
	sqlTasks := map[int64]legacyTask{}
	sqlTaskIDs := map[string]int64{}
	for i, r := range runs {
		if r.Run.UUID != entries[i].Run.UUID || runIndex[r.Run.UUID] != 0 {
			return nil, fmt.Errorf("legacy run mapping is inconsistent")
		}
		runIndex[r.Run.UUID] = i + 1
		entries[i].Tasks = nil
		entries[i].Artifacts = nil
		entries[i].Links = nil
		for _, t := range r.Tasks {
			if t.MLMDExecutionID == "" || t.MLMDExecutionID == "0" {
				continue
			}
			id, err := strconv.ParseInt(t.MLMDExecutionID, 10, 64)
			if err != nil || id <= 0 {
				return nil, fmt.Errorf("legacy task %s has no supported MLMD execution", t.UUID)
			}
			if t.RunID != r.Run.UUID || t.Namespace != namespace || g.executions[id] == nil {
				return nil, fmt.Errorf("legacy SQL task has inconsistent ownership or missing execution")
			}
			if _, exists := sqlTasks[id]; exists {
				return nil, fmt.Errorf("multiple legacy SQL tasks reference one execution")
			}
			if _, exists := sqlTaskIDs[t.UUID]; exists {
				return nil, fmt.Errorf("duplicate legacy SQL task identity")
			}
			sqlTasks[id] = t
			sqlTaskIDs[t.UUID] = id
		}
		for _, cid := range []int64{r.Run.PipelineContextId, r.Run.PipelineRunContextId} {
			if cid != 0 && g.contexts[cid] == nil {
				return nil, fmt.Errorf("legacy run references missing context")
			}
		}
	}
	owners := map[int64]int{}
	contextsByExecution := map[int64][]int64{}
	runContexts := map[int64]int{}
	for id, c := range g.contexts {
		if g.contextTypes[c.GetTypeId()].GetName() == "system.PipelineRun" {
			index := runIndex[c.GetName()]
			if index == 0 {
				return nil, fmt.Errorf("legacy graph includes ancestor run %q outside archive; export a wider or all-completed-history window", c.GetName())
			}
			if runs[index-1].Run.PipelineRunContextId != 0 && runs[index-1].Run.PipelineRunContextId != id {
				return nil, fmt.Errorf("legacy run context identity is inconsistent")
			}
			runContexts[id] = index
		}
		for _, props := range []map[string]*mlmd.Value{c.Properties, c.CustomProperties} {
			ns, e := legacyMLMDString(props, "namespace")
			if e != nil {
				return nil, e
			}
			if ns != "" && ns != namespace {
				return nil, fmt.Errorf("legacy context belongs to another namespace")
			}
		}
	}
	for _, a := range g.associations {
		eid, cid := a.GetExecutionId(), a.GetContextId()
		contextsByExecution[eid] = append(contextsByExecution[eid], cid)
		if owner := runContexts[cid]; owner != 0 {
			if owners[eid] != 0 && owners[eid] != owner {
				return nil, fmt.Errorf("legacy execution belongs to multiple runs")
			}
			owners[eid] = owner
		}
	}
	parents := map[int64]int64{}
	cacheSources := map[int64]int64{}
	edges := map[int64][]int64{}
	ids := []int64{}
	for id := range g.executions {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	tasks := map[int64]model.Task{}
	roots := map[int]int{}
	for _, id := range ids {
		e := g.executions[id]
		owner := owners[id]
		if owner == 0 {
			return nil, fmt.Errorf("legacy execution %d has no archived run owner", id)
		}
		p, err := legacyMLMDInt(e.CustomProperties, "parent_dag_id")
		if err != nil {
			return nil, err
		}
		if p != nil && *p != 0 {
			if *p < 0 || g.executions[*p] == nil || owners[*p] != owner {
				return nil, fmt.Errorf("legacy task parent is missing or belongs to another run")
			}
			parents[id] = *p
			edges[id] = append(edges[id], *p)
		}
		cached, err := legacyMLMDString(e.CustomProperties, "cached_execution_id")
		if err != nil {
			return nil, err
		}
		if cached != "" && cached != "0" {
			cid, err := strconv.ParseInt(cached, 10, 64)
			if err != nil || cid <= 0 || g.executions[cid] == nil {
				return nil, fmt.Errorf("legacy cached execution reference is invalid")
			}
			cacheSources[id] = cid
			edges[id] = append(edges[id], cid)
		}
		propertyNamespace, err := legacyMLMDString(e.Properties, "namespace")
		if err != nil {
			return nil, err
		}
		if propertyNamespace != "" && propertyNamespace != namespace {
			return nil, fmt.Errorf("legacy execution belongs to another namespace")
		}
		ns, err := legacyMLMDString(e.CustomProperties, "namespace")
		if err != nil {
			return nil, err
		}
		if ns != "" && ns != namespace {
			return nil, fmt.Errorf("legacy execution belongs to another namespace")
		}
		name, err := legacyMLMDString(e.CustomProperties, "task_name")
		if err != nil {
			return nil, err
		}
		if name == "" {
			return nil, fmt.Errorf("legacy execution %d has no task name", id)
		}
		display, err := legacyMLMDString(e.CustomProperties, "display_name")
		if err != nil {
			return nil, err
		}
		state, err := legacyMLMDState(e.GetLastKnownState())
		if err != nil {
			return nil, err
		}
		taskType, err := legacyMLMDTaskType(e, g.executionTypes[e.GetTypeId()].GetName(), name, p)
		if err != nil {
			return nil, err
		}
		if parents[id] == 0 {
			if taskType != model.TaskType(api.PipelineTask_ROOT) {
				return nil, fmt.Errorf("legacy non-root task has no parent")
			}
			roots[owner]++
		}
		task := model.Task{UUID: legacyMLMDID(source, "execution", strconv.FormatInt(id, 10)), Namespace: namespace, RunUUID: entries[owner-1].Run.UUID, Name: name, DisplayName: display, State: state, Type: taskType, Pods: model.JSONSlice{}, TypeAttrs: model.JSONData{}, CreatedAtInSec: e.GetCreateTimeSinceEpoch() / 1000, FinishedInSec: e.GetLastUpdateTimeSinceEpoch() / 1000}
		if parent := parents[id]; parent != 0 {
			pid := legacyMLMDID(source, "execution", strconv.FormatInt(parent, 10))
			task.ParentTaskUUID = &pid
		}
		for _, key := range []string{"iteration_index", "iteration_count"} {
			v, err := legacyMLMDInt(e.CustomProperties, key)
			if err != nil {
				return nil, err
			}
			if v != nil {
				if *v < 0 {
					return nil, fmt.Errorf("negative legacy %s", key)
				}
				task.TypeAttrs[key] = strconv.FormatInt(*v, 10)
			}
		}
		task.InputParameters, err = legacyMLMDParameters(e.CustomProperties["inputs"], api.IOType_RUNTIME_VALUE_INPUT)
		if err != nil {
			return nil, err
		}
		task.OutputParameters, err = legacyMLMDParameters(e.CustomProperties["outputs"], api.IOType_OUTPUT)
		if err != nil {
			return nil, err
		}
		custom := model.JSONData{"legacy_mlmd_execution": legacyMLMDJSON(e), "legacy_mlmd_execution_type": legacyMLMDJSON(g.executionTypes[e.GetTypeId()]), "legacy_mlmd_contexts": legacyMLMDContextProvenance(g, contextsByExecution[id])}
		task.StatusMetadata = model.JSONData{"custom_properties": custom}
		pod, err := legacyMLMDString(e.CustomProperties, "pod_name")
		if err != nil {
			return nil, err
		}
		podUID, err := legacyMLMDString(e.CustomProperties, "pod_uid")
		if err != nil {
			return nil, err
		}
		if sql, ok := sqlTasks[id]; ok {
			if sql.RunID != task.RunUUID {
				return nil, fmt.Errorf("SQL and MLMD task ownership disagree")
			}
			if sql.ParentTaskId != "" && sqlTaskIDs[sql.ParentTaskId] != parents[id] {
				return nil, fmt.Errorf("SQL and MLMD task parent disagree")
			}
			if sql.CreatedTimestamp != 0 {
				task.CreatedAtInSec = sql.CreatedTimestamp
			}
			task.StartedInSec = sql.StartedTimestamp
			if sql.FinishedTimestamp != 0 {
				task.FinishedInSec = sql.FinishedTimestamp
			}
			if pod == "" {
				pod = sql.PodName
			}
			data, _ := json.Marshal(sql)
			custom["legacy_sql_task"] = string(data)
			task.StateHistory, err = legacySQLStateHistory(sql)
			if err != nil {
				return nil, err
			}

		}
		if pod != "" {
			task.Pods, err = model.ProtoSliceToJSONSlice([]*api.PipelineTask_TaskPod{{Name: pod, Uid: podUID, Type: api.PipelineTask_EXECUTOR}})
			if err != nil {
				return nil, err
			}
		}
		// Imported history never becomes a source of executable cache hits.
		task.Fingerprint = ""
		task.LogicalKey = nil
		if err := budget.Add(task); err != nil {
			return nil, err
		}
		tasks[id] = task
	}
	if err := legacyMLMDAcyclic(edges); err != nil {
		return nil, err
	}
	for owner, count := range roots {
		if count != 1 {
			return nil, fmt.Errorf("legacy run %s has multiple root tasks", entries[owner-1].Run.UUID)
		}
	}
	for _, id := range ids {
		task := tasks[id]
		parts := []string{task.Name}
		p := parents[id]
		for p != 0 {
			if len(parts) >= 128 {
				return nil, fmt.Errorf("legacy task nesting exceeds 128 levels")
			}
			parts = append(parts, tasks[p].Name)
			p = parents[p]
		}
		for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
			parts[i], parts[j] = parts[j], parts[i]
		}
		task.ScopePath = strings.Join(parts, ".")
		if cached := cacheSources[id]; cached != 0 {
			if g.executions[cached].GetLastKnownState() != mlmd.Execution_COMPLETE && g.executions[cached].GetLastKnownState() != mlmd.Execution_CACHED {
				return nil, fmt.Errorf("cached source execution was not successful")
			}
			if len(task.OutputParameters) == 0 {
				for next := cached; next != 0; next = cacheSources[next] {
					if len(tasks[next].OutputParameters) != 0 {
						task.OutputParameters = tasks[next].OutputParameters
						break
					}
				}
			}
		}

		tasks[id] = task
	}
	if err := legacyMLMDResolveParameters(g, parents, cacheSources, tasks, budget); err != nil {
		return nil, err
	}
	artifacts := map[int64][]model.Artifact{}
	for id, a := range g.artifacts {
		converted, err := legacyMLMDArtifacts(source, namespace, a, g.artifactTypes[a.GetTypeId()], budget)
		if err != nil {
			return nil, err
		}
		artifacts[id] = converted
	}
	eventsByExecution := map[int64][]*mlmd.Event{}
	for _, event := range g.events {
		eventsByExecution[event.GetExecutionId()] = append(eventsByExecution[event.GetExecutionId()], event)
	}
	// CACHED executions may retain their output records only on the source execution.
	for _, id := range ids {
		if cached := cacheSources[id]; cached != 0 {
			hasOutput := false
			for _, e := range eventsByExecution[id] {
				if e.GetType() == mlmd.Event_OUTPUT {
					hasOutput = true
				}
			}
			if !hasOutput {
				for next := cached; next != 0; next = cacheSources[next] {
					found := false
					for _, e := range eventsByExecution[next] {
						if e.GetType() == mlmd.Event_OUTPUT {
							found = true
							copy := proto.Clone(e).(*mlmd.Event)
							copy.ExecutionId = proto.Int64(id)
							if err := budget.Add(copy); err != nil {
								return nil, err
							}
							eventsByExecution[id] = append(eventsByExecution[id], copy)
						}
					}
					if found {
						break
					}
				}
			}
		}
	}
	if err := legacyMLMDResolveArtifacts(g, parents, cacheSources, tasks, eventsByExecution, budget); err != nil {
		return nil, err
	}
	producersByArtifact := map[int64][]int64{}
	for producerID, events := range eventsByExecution {
		for _, event := range events {
			if event.GetType() == mlmd.Event_OUTPUT {
				producersByArtifact[event.GetArtifactId()] = append(producersByArtifact[event.GetArtifactId()], producerID)
			}
		}
	}
	seenArtifacts := make([]map[string]bool, len(entries))
	for i := range seenArtifacts {
		seenArtifacts[i] = map[string]bool{}
	}
	usedArtifacts := map[int64]bool{}
	for _, id := range ids {
		task := tasks[id]
		owner := owners[id] - 1
		if sql, ok := sqlTasks[id]; ok {
			if err := legacySQLArtifactReferences(sql, eventsByExecution[id]); err != nil {
				return nil, err
			}
		}
		seenLinks := map[string]bool{}
		rawEvents := []any{}
		for _, event := range eventsByExecution[id] {
			if event.GetType() != mlmd.Event_INPUT && event.GetType() != mlmd.Event_OUTPUT {
				return nil, fmt.Errorf("unsupported legacy event type %s", event.GetType())
			}
			steps := event.GetPath().GetSteps()
			if len(steps) != 1 {
				return nil, fmt.Errorf("unsupported legacy artifact event path; expected one named port")
			}
			step, ok := steps[0].Value.(*mlmd.Event_Path_Step_Key)
			if !ok || step.Key == "" {
				return nil, fmt.Errorf("legacy event has no named artifact port")
			}
			rawEvents = append(rawEvents, legacyMLMDJSON(event))
			usedArtifacts[event.GetArtifactId()] = true
			ioType := api.IOType_OUTPUT
			var inputProducer *api.IOProducer
			if event.GetType() == mlmd.Event_INPUT {
				ioType = api.IOType_RUNTIME_VALUE_INPUT
				if parent := parents[id]; parent != 0 {
					for _, candidate := range eventsByExecution[parent] {
						if candidate.GetType() == mlmd.Event_INPUT && candidate.GetArtifactId() == event.GetArtifactId() {
							ioType = api.IOType_COMPONENT_INPUT
						}
					}
				}
				for _, producerID := range producersByArtifact[event.GetArtifactId()] {
					if producerID == id || owners[producerID] != owners[id] || parents[producerID] != parents[id] {
						continue
					}
					for _, candidate := range eventsByExecution[producerID] {
						if candidate.GetType() == mlmd.Event_OUTPUT && candidate.GetArtifactId() == event.GetArtifactId() {
							if inputProducer != nil && inputProducer.TaskName != tasks[producerID].Name {
								return nil, fmt.Errorf("ambiguous legacy artifact input producer")
							}
							inputProducer = &api.IOProducer{TaskName: tasks[producerID].Name}
							ioType = api.IOType_TASK_OUTPUT_INPUT
						}
					}
				}
			}
			iteration := model.ArtifactTaskNoIteration
			if v, err := legacyMLMDInt(g.executions[id].CustomProperties, "iteration_index"); err != nil {
				return nil, err
			} else if v != nil {
				iteration = *v
				if ioType == api.IOType_OUTPUT {
					ioType = api.IOType_ITERATOR_OUTPUT
				}
			}
			for _, a := range artifacts[event.GetArtifactId()] {
				key := task.UUID + "\x00" + a.UUID + "\x00" + strconv.Itoa(int(ioType)) + "\x00" + step.Key
				if seenLinks[key] {
					return nil, fmt.Errorf("duplicate legacy artifact event relationship")
				}
				seenLinks[key] = true
				link := model.ArtifactTask{UUID: legacyMLMDID(source, "event", key), ArtifactID: a.UUID, TaskID: task.UUID, RunUUID: task.RunUUID, Type: model.IOType(ioType), Iteration: iteration, ArtifactKey: step.Key}
				if inputProducer != nil {
					link.Producer, err = model.ProtoMessageToJSONData(inputProducer)
					if err != nil {
						return nil, err
					}
				}
				if event.GetType() == mlmd.Event_OUTPUT {
					producer := &api.IOProducer{TaskName: task.Name}
					if iteration >= 0 {
						producer.Iteration = &iteration
					}
					link.Producer, err = model.ProtoMessageToJSONData(producer)
					if err != nil {
						return nil, err
					}
				}
				if err := budget.Add(link); err != nil {
					return nil, err
				}
				entries[owner].Links = append(entries[owner].Links, link)
				if !seenArtifacts[owner][a.UUID] {
					if err := budget.Add(a); err != nil {
						return nil, err
					}
					entries[owner].Artifacts = append(entries[owner].Artifacts, a)
					seenArtifacts[owner][a.UUID] = true
				}
			}
		}
		task.StatusMetadata["custom_properties"].(model.JSONData)["legacy_mlmd_events"] = rawEvents
		if err := budget.Add(task); err != nil {
			return nil, err
		}
		entries[owner].Tasks = append(entries[owner].Tasks, task)
	}
	for id := range artifacts {
		if !usedArtifacts[id] {
			return nil, fmt.Errorf("legacy artifact %d has no representable task relationship", id)
		}
	}
	return legacySQLOnlyTasks(source, namespace, runs, entries, budget)
}

func legacyMLMDArtifacts(source, namespace string, a *mlmd.Artifact, typ *mlmd.ArtifactType, budget *transfer.ExportBudget) ([]model.Artifact, error) {
	if a.GetState() != mlmd.Artifact_LIVE {
		return nil, fmt.Errorf("legacy artifact %d is not live", a.GetId())
	}
	metadata := model.JSONData{}
	for _, props := range []map[string]*mlmd.Value{a.Properties, a.CustomProperties} {
		for key, v := range props {
			if _, exists := metadata[key]; exists {
				return nil, fmt.Errorf("legacy artifact property %q is ambiguous", key)
			}
			value, err := legacyMLMDValue(v)
			if err != nil {
				return nil, err
			}
			metadata[key] = value
		}
	}
	if value, present := metadata["namespace"]; present {
		ns, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("legacy artifact namespace must be a string")
		}
		if ns != "" && ns != namespace {
			return nil, fmt.Errorf("legacy artifact belongs to another namespace")
		}
	}
	name := a.GetName()
	if v, ok := metadata["display_name"].(string); ok && v != "" {
		name = v
	}
	if name == "" {
		name = strconv.FormatInt(a.GetId(), 10)
	}
	artifactType := api.Artifact_Artifact
	switch typ.GetName() {
	case "system.Artifact":
	case "system.Model":
		artifactType = api.Artifact_Model
	case "system.Dataset":
		artifactType = api.Artifact_Dataset
	case "system.HTML":
		artifactType = api.Artifact_HTML
	case "system.Markdown":
		artifactType = api.Artifact_Markdown
	case "system.Metrics":
		artifactType = api.Artifact_Metric
	case "system.ClassificationMetrics":
		artifactType = api.Artifact_ClassificationMetric
	case "system.SlicedClassificationMetrics":
		artifactType = api.Artifact_SlicedClassificationMetric
	default:
		metadata["_kfp_schema_title"] = typ.GetName()
	}
	makeArtifact := func(key string, values model.JSONData) model.Artifact {
		uri := a.GetUri()
		return model.Artifact{UUID: legacyMLMDID(source, "artifact", strconv.FormatInt(a.GetId(), 10)+key), Namespace: namespace, Name: name, URI: &uri, Type: model.ArtifactType(artifactType), CreatedAtInSec: a.GetCreateTimeSinceEpoch() / 1000, LastUpdateInSec: a.GetLastUpdateTimeSinceEpoch() / 1000, Metadata: values}
	}
	artifactJSON, typeJSON := legacyMLMDJSON(a), legacyMLMDJSON(typ)
	provenance := func(values model.JSONData) {
		values["_kfp_legacy_mlmd_artifact"] = artifactJSON
		values["_kfp_legacy_mlmd_artifact_type"] = typeJSON
		// Attributions grow as later runs consume a shared artifact; they are
		// relationships, not part of its intrinsic imported identity.
	}
	if artifactType != api.Artifact_Metric {
		provenance(metadata)
		return []model.Artifact{makeArtifact("", metadata)}, nil
	}
	keys := []string{}
	for k := range metadata {
		if k != "display_name" && k != "store_session_info" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		return nil, fmt.Errorf("legacy metrics artifact has no numeric metrics")
	}
	result := []model.Artifact{}
	for _, key := range keys {
		value, ok := metadata[key].(float64)
		if !ok {
			return nil, fmt.Errorf("legacy metric %q is not numeric", key)
		}
		values := model.JSONData{key: value}
		provenance(values)
		artifact := makeArtifact("\x00metric\x00"+key, values)
		artifact.Name = key
		artifact.NumberValue = &value
		if err := budget.Add(artifact); err != nil {
			return nil, err
		}
		result = append(result, artifact)
	}
	return result, nil
}

func legacySQLTaskState(value string) (model.TaskStatus, error) {
	switch strings.ToUpper(value) {
	case "", "RUNTIME_STATE_UNSPECIFIED":
		return model.TaskStatus(api.PipelineTask_RUNTIME_STATE_UNSPECIFIED), nil
	case "RUNNING":
		return model.TaskStatus(api.PipelineTask_RUNNING), nil
	case "SUCCEEDED":
		return model.TaskStatus(api.PipelineTask_SUCCEEDED), nil
	case "FAILED", "ERROR":
		return model.TaskStatus(api.PipelineTask_FAILED), nil
	case "SKIPPED", "CANCELED":
		return model.TaskStatus(api.PipelineTask_SKIPPED), nil
	case "CACHED":
		return model.TaskStatus(api.PipelineTask_CACHED), nil
	default:
		return 0, fmt.Errorf("unsupported legacy task state %q", value)
	}
}

func legacySQLStateHistory(task legacyTask) (model.JSONSlice, error) {
	history := task.StateHistory
	if task.StateHistoryString != "" {
		decoder := json.NewDecoder(strings.NewReader(task.StateHistoryString))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&history); err != nil {
			return nil, fmt.Errorf("invalid legacy task state history: %w", err)
		}
		if decoder.Decode(new(any)) != io.EOF {
			return nil, fmt.Errorf("trailing legacy state history data")
		}
		if len(task.StateHistory) != 0 && !reflect.DeepEqual(task.StateHistory, history) {
			return nil, fmt.Errorf("legacy hydrated and persisted task history disagree")
		}
	}
	statuses := []*api.PipelineTask_TaskStatus{}
	for _, entry := range history {
		if entry == nil {
			return nil, fmt.Errorf("nil legacy task state history")
		}
		state, err := legacySQLTaskState(entry.State)
		if err != nil {
			return nil, err
		}
		timestamp := timestamppb.New(time.Unix(entry.UpdateTimeInSec, 0))
		if err := timestamp.CheckValid(); err != nil {
			return nil, err
		}
		status := &api.PipelineTask_TaskStatus{State: api.PipelineTask_TaskState(state), UpdateTime: timestamp}
		if len(entry.Error) > 0 && !bytes.Equal(entry.Error, []byte("null")) && !bytes.Equal(entry.Error, []byte("{}")) {
			status.Error = &statuspb.Status{}
			if err := protojson.Unmarshal(entry.Error, status.Error); err != nil {
				return nil, fmt.Errorf("unsupported legacy task error: %w", err)
			}
		}
		statuses = append(statuses, status)
	}
	return model.ProtoSliceToJSONSlice(statuses)
}

func legacySQLOnlyTasks(source, namespace string, runs []legacyRunHistory, entries []Entry, budget *transfer.ExportBudget) ([]Entry, error) {
	for i, run := range runs {
		mapping := map[string]string{}
		all := map[string]legacyTask{}
		native := map[string]model.Task{}
		for _, task := range entries[i].Tasks {
			native[task.UUID] = task
		}
		for _, task := range run.Tasks {
			if task.UUID == "" || all[task.UUID].UUID != "" || task.RunID != run.Run.UUID || task.Namespace != namespace {
				return nil, fmt.Errorf("invalid legacy SQL task identity or ownership")
			}
			all[task.UUID] = task
			if task.MLMDExecutionID != "" && task.MLMDExecutionID != "0" {
				mapping[task.UUID] = legacyMLMDID(source, "execution", task.MLMDExecutionID)
			} else {
				mapping[task.UUID] = legacyMLMDID(source, "sql-task", task.UUID)
			}
		}
		rootID := ""
		for _, task := range entries[i].Tasks {
			if task.Type == model.TaskType(api.PipelineTask_ROOT) {
				rootID = task.UUID
			}
		}
		needsRoot := false
		for _, task := range run.Tasks {
			if (task.MLMDExecutionID == "" || task.MLMDExecutionID == "0") && task.ParentTaskId == "" {
				needsRoot = true
			}
		}
		if needsRoot && rootID == "" {
			rootID = legacyMLMDID(source, "sql-root", run.Run.UUID)
			rootState, err := legacySQLTaskState(string(entries[i].Run.State))
			if err != nil {
				return nil, err
			}
			task := model.Task{UUID: rootID, Namespace: namespace, RunUUID: run.Run.UUID, Name: "root", DisplayName: "root", Type: model.TaskType(api.PipelineTask_ROOT), State: rootState, CreatedAtInSec: entries[i].Run.CreatedAtInSec, FinishedInSec: entries[i].Run.FinishedAtInSec, Pods: model.JSONSlice{}, TypeAttrs: model.JSONData{}, ScopePath: "root", StatusMetadata: model.JSONData{"custom_properties": model.JSONData{"legacy_sql_synthetic_root": true}}}
			native[rootID] = task
			if err := budget.Add(task); err != nil {
				return nil, err
			}
			entries[i].Tasks = append(entries[i].Tasks, task)
		}
		for _, sql := range run.Tasks {
			if sql.MLMDExecutionID != "" && sql.MLMDExecutionID != "0" {
				continue
			}
			for _, text := range []string{sql.MLMDInputs, sql.MLMDOutputs} {
				if text != "" && text != "null" && text != "{}" {
					return nil, fmt.Errorf("SQL-only legacy task has unsupported MLMD artifact references")
				}
			}
			state, err := legacySQLTaskState(sql.State)
			if err != nil {
				return nil, err
			}
			history, err := legacySQLStateHistory(sql)
			if err != nil {
				return nil, err
			}
			task := model.Task{UUID: mapping[sql.UUID], Namespace: namespace, RunUUID: run.Run.UUID, Name: sql.Name, DisplayName: sql.Name, Type: model.TaskType(api.PipelineTask_RUNTIME), State: state, CreatedAtInSec: sql.CreatedTimestamp, StartedInSec: sql.StartedTimestamp, FinishedInSec: sql.FinishedTimestamp, StateHistory: history, Pods: model.JSONSlice{}, TypeAttrs: model.JSONData{}}
			parent := rootID
			if sql.ParentTaskId != "" {
				parent = mapping[sql.ParentTaskId]
				if parent == "" {
					return nil, fmt.Errorf("legacy SQL task parent is missing")
				}
			}
			task.ParentTaskUUID = &parent
			pods := []*api.PipelineTask_TaskPod{}
			seen := map[string]bool{}
			names := append([]string{sql.PodName}, sql.ChildrenPods...)
			if sql.ChildrenPodsString != "" {
				var extra []string
				if err := json.Unmarshal([]byte(sql.ChildrenPodsString), &extra); err != nil {
					return nil, fmt.Errorf("invalid legacy child pods: %w", err)
				}
				names = append(names, extra...)
			}
			for _, name := range names {
				if name != "" && !seen[name] {
					seen[name] = true
					pods = append(pods, &api.PipelineTask_TaskPod{Name: name, Type: api.PipelineTask_EXECUTOR})
				}
			}
			task.Pods, err = model.ProtoSliceToJSONSlice(pods)
			if err != nil {
				return nil, err
			}
			raw, _ := json.Marshal(sql)
			task.StatusMetadata = model.JSONData{"custom_properties": model.JSONData{"legacy_sql_task": string(raw)}}
			native[task.UUID] = task
			if err := budget.Add(task); err != nil {
				return nil, err
			}
			entries[i].Tasks = append(entries[i].Tasks, task)
		}
		for j, task := range entries[i].Tasks {
			parts := []string{task.Name}
			seen := map[string]bool{task.UUID: true}
			parent := task.ParentTaskUUID
			for parent != nil {
				if seen[*parent] {
					return nil, fmt.Errorf("legacy SQL task parent cycle")
				}
				seen[*parent] = true
				p, ok := native[*parent]
				if !ok {
					return nil, fmt.Errorf("legacy task parent missing")
				}
				parts = append(parts, p.Name)
				if len(parts) > 128 {
					return nil, fmt.Errorf("legacy task nesting exceeds 128 levels")
				}
				parent = p.ParentTaskUUID
			}
			for a, b := 0, len(parts)-1; a < b; a, b = a+1, b-1 {
				parts[a], parts[b] = parts[b], parts[a]
			}
			entries[i].Tasks[j].ScopePath = strings.Join(parts, ".")
		}
	}
	return entries, nil
}

// SQL records may omit artifact bindings, but every binding they retain must
// have its corresponding graph event. Otherwise accepting would lose history.
func legacySQLArtifactReferences(sql legacyTask, events []*mlmd.Event) error {
	for _, input := range []struct {
		raw  string
		kind mlmd.Event_Type
	}{{sql.MLMDInputs, mlmd.Event_INPUT}, {sql.MLMDOutputs, mlmd.Event_OUTPUT}} {
		if input.raw == "" || input.raw == "null" {
			continue
		}
		var ports map[string]struct {
			IDs []int64 `json:"artifact_ids"`
		}
		if err := decodeArchiveJSON([]byte(input.raw), &ports); err != nil {
			return fmt.Errorf("invalid legacy SQL artifact bindings: %w", err)
		}
		for port, binding := range ports {
			for _, id := range binding.IDs {
				found := false
				for _, e := range events {
					steps := e.GetPath().GetSteps()
					if e.GetType() == input.kind && e.GetArtifactId() == id && len(steps) == 1 && steps[0].GetKey() == port {
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("legacy SQL artifact binding has no matching graph event")
				}
			}
		}
	}
	return nil
}
