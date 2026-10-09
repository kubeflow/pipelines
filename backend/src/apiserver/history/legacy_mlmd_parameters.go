// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

package history

import (
	"fmt"
	"sort"

	pipelinespec "github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	api "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/transfer"
	mlmd "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

// Resolve parameter bindings using the archived ownership tree. Unlike runtime
// resolution this never queries another run or executes a missing component.
func legacyMLMDResolveParameters(g *legacyMLMDGraph, parents, caches map[int64]int64, tasks map[int64]model.Task, budget *transfer.ExportBudget) error {
	children := map[int64][]int64{}
	for child, parent := range parents {
		children[parent] = append(children[parent], child)
	}
	type key struct {
		execution int64
		port      string
	}
	values := map[key]*structpb.Value{}
	visiting := map[key]bool{}
	var resolve func(int64, string, int) (*structpb.Value, error)
	resolve = func(id int64, port string, depth int) (*structpb.Value, error) {
		k := key{id, port}
		if v := values[k]; v != nil {
			return v, nil
		}
		if depth > 128 || visiting[k] {
			return nil, fmt.Errorf("legacy parameter dependency cycle or depth exceeds 128")
		}
		visiting[k] = true
		defer delete(visiting, k)
		execution := g.executions[id]
		if v := execution.CustomProperties["outputs"].GetStructValue().GetFields()[port]; v != nil {
			values[k] = v
			return v, nil
		}
		if cached := caches[id]; cached != 0 {
			v, err := resolve(cached, port, depth+1)
			if err == nil {
				values[k] = v
			}
			return v, err
		}
		if count, err := legacyMLMDInt(execution.CustomProperties, "iteration_count"); err != nil {
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
					if _, exists := iterations[*index]; exists {
						return nil, fmt.Errorf("ambiguous legacy loop iteration")
					}
					iterations[*index] = child
				}
			}
			if int64(len(iterations)) != *count {
				return nil, fmt.Errorf("missing legacy loop iterations")
			}
			list := &structpb.ListValue{}
			for index := int64(0); index < *count; index++ {
				child, exists := iterations[index]
				if !exists {
					return nil, fmt.Errorf("missing legacy loop iteration")
				}
				v, err := resolve(child, port, depth+1)
				if err != nil {
					return nil, err
				}
				if err := budget.Add(v); err != nil {
					return nil, err
				}
				list.Values = append(list.Values, v)
			}
			v := structpb.NewListValue(list)
			values[k] = v
			return v, nil
		}
		binding := execution.CustomProperties["parameter_producer_task"].GetStructValue().GetFields()[port]
		if binding == nil {
			return nil, fmt.Errorf("legacy output parameter %q is absent on execution %d", port, id)
		}
		spec := &pipelinespec.DagOutputsSpec_DagOutputParameterSpec{}
		if err := protojson.Unmarshal([]byte(binding.GetStringValue()), spec); err != nil {
			return nil, fmt.Errorf("invalid legacy DAG parameter binding: %w", err)
		}
		selectors := []*pipelinespec.DagOutputsSpec_ParameterSelectorSpec{}
		if selector := spec.GetValueFromParameter(); selector != nil {
			selectors = append(selectors, selector)
		} else if oneof := spec.GetValueFromOneof(); oneof != nil {
			selectors = oneof.GetParameterSelectors()
		} else {
			return nil, fmt.Errorf("unsupported legacy DAG parameter binding")
		}
		var selected *structpb.Value
		for _, selector := range selectors {
			for _, child := range children[id] {
				if tasks[child].Name != selector.GetProducerSubtask() {
					continue
				}
				state := g.executions[child].GetLastKnownState()
				if state != mlmd.Execution_COMPLETE && state != mlmd.Execution_CACHED {
					continue
				}
				v, err := resolve(child, selector.GetOutputParameterKey(), depth+1)
				if err != nil {
					return nil, err
				}
				if selected != nil {
					return nil, fmt.Errorf("ambiguous legacy DAG parameter producer")
				}
				selected = v
			}
		}
		if selected == nil {
			return nil, fmt.Errorf("missing successful legacy DAG parameter producer for %q", port)
		}
		values[k] = selected
		return selected, nil
	}
	ids := []int64{}
	for id := range tasks {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for _, id := range ids {
		execution := g.executions[id]
		ports := map[string]bool{}
		for next := id; next != 0; next = caches[next] {
			for key := range g.executions[next].CustomProperties["outputs"].GetStructValue().GetFields() {
				ports[key] = true
			}
			if state := g.executions[next].GetLastKnownState(); state == mlmd.Execution_COMPLETE || state == mlmd.Execution_CACHED {
				for key := range g.executions[next].CustomProperties["parameter_producer_task"].GetStructValue().GetFields() {
					ports[key] = true
				}
			}
		}
		if bindings := execution.CustomProperties["parameter_producer_task"]; bindings != nil {
			if _, ok := bindings.Value.(*mlmd.Value_StructValue); !ok {
				return fmt.Errorf("invalid legacy parameter producer bindings")
			}
		}
		fields := map[string]*structpb.Value{}
		for port := range ports {
			v, err := resolve(id, port, 0)
			if err != nil {
				return err
			}
			fields[port] = v
		}
		task := tasks[id]
		var err error
		task.OutputParameters, err = legacyMLMDParameters(&mlmd.Value{Value: &mlmd.Value_StructValue{StructValue: &structpb.Struct{Fields: fields}}}, api.IOType_OUTPUT)
		if err != nil {
			return err
		}
		if err := budget.Add(task.OutputParameters); err != nil {
			return err
		}
		tasks[id] = task
	}
	return nil
}
