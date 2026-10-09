// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

package history

import (
	"fmt"
	"sort"

	pipelinespec "github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/transfer"
	mlmd "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"google.golang.org/protobuf/proto"
)

// DAG output artifacts are sometimes recorded only as selectors. Materialize
// their graph links from archived descendants before building native links.
func legacyMLMDResolveArtifacts(g *legacyMLMDGraph, parents, caches map[int64]int64, tasks map[int64]model.Task, events map[int64][]*mlmd.Event, budget *transfer.ExportBudget) error {
	bindings := map[int64]map[string]*pipelinespec.DagOutputsSpec_DagOutputArtifactSpec{}
	children := map[int64][]int64{}
	for child, parent := range parents {
		children[parent] = append(children[parent], child)
	}
	for id, e := range g.executions {
		raw, err := legacyMLMDString(e.CustomProperties, "artifact_producer_task")
		if err != nil {
			return err
		}
		if raw != "" {
			var ports map[string]*pipelinespec.DagOutputsSpec_DagOutputArtifactSpec
			if err := decodeArchiveJSON([]byte(raw), &ports); err != nil {
				return fmt.Errorf("invalid legacy artifact producer bindings: %w", err)
			}
			bindings[id] = ports
		}
	}
	type key struct {
		id   int64
		port string
	}
	memo := map[key][]int64{}
	visiting := map[key]bool{}
	var resolve func(int64, string, int) ([]int64, error)
	resolve = func(id int64, port string, depth int) ([]int64, error) {
		k := key{id, port}
		if ids, ok := memo[k]; ok {
			return ids, nil
		}
		if depth > 128 || visiting[k] {
			return nil, fmt.Errorf("legacy artifact dependency cycle or depth exceeds 128")
		}
		visiting[k] = true
		defer delete(visiting, k)
		ids := []int64{}
		for _, event := range events[id] {
			steps := event.GetPath().GetSteps()
			if event.GetType() == mlmd.Event_OUTPUT && len(steps) == 1 && steps[0].GetKey() == port {
				ids = append(ids, event.GetArtifactId())
			}
		}
		if len(ids) > 0 {
			memo[k] = ids
			return ids, nil
		}
		if source := caches[id]; source != 0 {
			return resolve(source, port, depth+1)
		}
		if count, err := legacyMLMDInt(g.executions[id].CustomProperties, "iteration_count"); err != nil {
			return nil, err
		} else if count != nil {
			if *count < 0 || *count > 20000 {
				return nil, fmt.Errorf("legacy loop iteration count exceeds transfer limit")
			}
			iterations := map[int64]int64{}
			for _, child := range children[id] {
				index, err := legacyMLMDInt(g.executions[child].CustomProperties, "iteration_index")
				if err != nil {
					return nil, err
				}
				if index != nil {
					if _, ok := iterations[*index]; ok {
						return nil, fmt.Errorf("ambiguous artifact loop iteration")
					}
					iterations[*index] = child
				}
			}
			if int64(len(iterations)) != *count {
				return nil, fmt.Errorf("missing artifact loop iterations")
			}
			for index := int64(0); index < *count; index++ {
				child, ok := iterations[index]
				if !ok {
					return nil, fmt.Errorf("missing artifact loop iteration")
				}
				part, err := resolve(child, port, depth+1)
				if err != nil {
					return nil, err
				}
				if err := budget.Add(part); err != nil {
					return nil, err
				}
				ids = append(ids, part...)
			}
			memo[k] = ids
			return ids, nil
		}
		spec := bindings[id][port]
		if spec == nil {
			return nil, fmt.Errorf("missing legacy artifact output %q", port)
		}
		selected := false
		for _, selector := range spec.GetArtifactSelectors() {
			selectorMatched := false
			for _, child := range children[id] {
				if tasks[child].Name != selector.GetProducerSubtask() {
					continue
				}
				state := g.executions[child].GetLastKnownState()
				if state != mlmd.Execution_COMPLETE && state != mlmd.Execution_CACHED {
					continue
				}
				if selectorMatched {
					return nil, fmt.Errorf("ambiguous legacy DAG artifact producer")
				}
				part, err := resolve(child, selector.GetOutputArtifactKey(), depth+1)
				if err != nil {
					return nil, err
				}
				selected = true
				selectorMatched = true
				ids = append(ids, part...)
			}
		}
		if !selected {
			return nil, fmt.Errorf("missing successful legacy DAG artifact producer")
		}
		memo[k] = ids
		return ids, nil
	}
	idsToResolve := []int64{}
	for id := range bindings {
		if state := g.executions[id].GetLastKnownState(); state == mlmd.Execution_COMPLETE || state == mlmd.Execution_CACHED {
			idsToResolve = append(idsToResolve, id)
		}
	}
	sort.Slice(idsToResolve, func(i, j int) bool { return idsToResolve[i] < idsToResolve[j] })
	for _, id := range idsToResolve {
		portsToResolve := []string{}
		for port := range bindings[id] {
			portsToResolve = append(portsToResolve, port)
		}
		sort.Strings(portsToResolve)
		for _, port := range portsToResolve {
			ids, err := resolve(id, port, 0)
			if err != nil {
				return err
			}
			has := false
			for _, event := range events[id] {
				steps := event.GetPath().GetSteps()
				if event.GetType() == mlmd.Event_OUTPUT && len(steps) == 1 && steps[0].GetKey() == port {
					has = true
				}
			}
			if has {
				continue
			}
			seen := map[int64]bool{}
			for _, artifact := range ids {
				if seen[artifact] {
					continue
				}
				seen[artifact] = true
				event := &mlmd.Event{ExecutionId: proto.Int64(id), ArtifactId: proto.Int64(artifact), Type: mlmd.Event_OUTPUT.Enum(), Path: &mlmd.Event_Path{Steps: []*mlmd.Event_Path_Step{{Value: &mlmd.Event_Path_Step_Key{Key: port}}}}}
				if err := budget.Add(event); err != nil {
					return err
				}
				events[id] = append(events[id], event)
			}
		}
	}
	return nil
}
