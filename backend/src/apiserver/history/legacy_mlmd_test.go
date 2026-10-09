// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

package history

import (
	"encoding/json"
	"strconv"
	"testing"

	pipelinespec "github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	api "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/transfer"
	mlmd "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func legacyGraphFixture(t *testing.T) (legacyGraph, []legacyRunHistory, []Entry) {
	t.Helper()
	var graph legacyGraph
	require.NoError(t, json.Unmarshal([]byte(`{
 "context_types":[{"id":"1","name":"system.PipelineRun"}],
 "contexts":[{"id":"2","type_id":"1","name":"run"}],
 "execution_types":[{"id":"3","name":"system.DAGExecution"},{"id":"4","name":"system.ContainerExecution"}],
 "executions":[{"id":"10","type_id":"3","last_known_state":"COMPLETE","custom_properties":{"task_name":{"string_value":"root"}}},{"id":"11","type_id":"4","last_known_state":"COMPLETE","custom_properties":{"task_name":{"string_value":"train"},"parent_dag_id":{"int_value":"10"},"outputs":{"struct_value":{"accuracy":0.9}}}}],
 "associations":[{"context_id":"2","execution_id":"10"},{"context_id":"2","execution_id":"11"}],
 "artifact_types":[{"id":"5","name":"system.Dataset"}],
 "artifacts":[{"id":"20","type_id":"5","state":"LIVE","uri":"s3://bucket/data"}],
 "events":[{"artifact_id":"20","execution_id":"11","type":"OUTPUT","path":{"steps":[{"key":"dataset"}]}}]
 }`), &graph))
	return graph, []legacyRunHistory{{Run: legacyRun{UUID: "run", Namespace: "ns", PipelineRunContextId: 2}}}, []Entry{{Run: model.Run{UUID: "run"}}}
}

func TestLegacyMLMDConversion(t *testing.T) {
	graph, runs, entries := legacyGraphFixture(t)
	converted, err := convertLegacyMLMD("source", "ns", runs, graph, entries)
	require.NoError(t, err)
	require.Len(t, converted[0].Tasks, 2)
	require.Len(t, converted[0].Artifacts, 1)
	require.Len(t, converted[0].Links, 1)
	task := converted[0].Tasks[1]
	require.Equal(t, "root.train", task.ScopePath)
	require.Equal(t, converted[0].Tasks[0].UUID, *task.ParentTaskUUID)
	require.Len(t, task.OutputParameters, 1)
	require.Nil(t, task.LogicalKey)
	require.Empty(t, task.Fingerprint)
	require.Equal(t, "s3://bucket/data", *converted[0].Artifacts[0].URI)
	require.Equal(t, task.UUID, converted[0].Links[0].TaskID)
	again, err := convertLegacyMLMD("source", "ns", runs, graph, entries)
	require.NoError(t, err)
	require.Equal(t, converted, again)
	other, err := convertLegacyMLMD("other", "ns", runs, graph, entries)
	require.NoError(t, err)
	require.NotEqual(t, task.UUID, other[0].Tasks[1].UUID)
}

func TestLegacyMLMDRejectsUnrepresentableGraph(t *testing.T) {
	tests := map[string]func(legacyGraph){
		"missing owner":    func(g legacyGraph) { g["associations"] = g["associations"][:1] },
		"foreign run":      func(g legacyGraph) { g["contexts"][0]["name"] = "missing" },
		"unknown type":     func(g legacyGraph) { g["execution_types"][1]["name"] = "custom.Unsupported" },
		"unfinished":       func(g legacyGraph) { g["executions"][1]["last_known_state"] = "RUNNING" },
		"unknown event":    func(g legacyGraph) { g["events"][0]["type"] = "DECLARED_OUTPUT" },
		"missing artifact": func(g legacyGraph) { g["events"][0]["artifact_id"] = "999" },
		"cycle": func(g legacyGraph) {
			g["executions"][1]["custom_properties"].(map[string]any)["parent_dag_id"] = map[string]any{"int_value": "11"}
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			g, r, e := legacyGraphFixture(t)
			mutate(g)
			_, err := convertLegacyMLMD("source", "ns", r, g, e)
			require.Error(t, err)
		})
	}
}

func TestLegacyMLMDMetricFanoutBudget(t *testing.T) {
	artifact := &mlmd.Artifact{Id: proto.Int64(1), State: mlmd.Artifact_LIVE.Enum(), CustomProperties: map[string]*mlmd.Value{"a": {Value: &mlmd.Value_DoubleValue{DoubleValue: 0.7}}, "b": {Value: &mlmd.Value_DoubleValue{DoubleValue: 0.8}}}}
	typ := &mlmd.ArtifactType{Name: proto.String("system.Metrics")}
	converted, err := legacyMLMDArtifacts("s", "ns", artifact, typ, transfer.NewExportBudget(10000))
	require.NoError(t, err)
	require.Len(t, converted, 2)
	require.Equal(t, "a", converted[0].Name)
	require.Equal(t, 0.7, *converted[0].NumberValue)
	require.NotEqual(t, converted[0].UUID, converted[1].UUID)
	_, err = legacyMLMDArtifacts("s", "ns", artifact, typ, transfer.NewExportBudget(10))
	require.Error(t, err)
	state, err := legacyMLMDState(mlmd.Execution_CANCELED)
	require.NoError(t, err)
	require.Equal(t, model.TaskStatus(api.PipelineTask_SKIPPED), state)
}

func TestLegacyMLMDNestedDAGParameterBindings(t *testing.T) {
	for _, oneof := range []bool{false, true} {
		t.Run(map[bool]string{false: "direct", true: "oneof"}[oneof], func(t *testing.T) {
			g, r, e := legacyGraphFixture(t)
			binding := `{"value_from_parameter":{"producer_subtask":"train","output_parameter_key":"accuracy"}}`
			if oneof {
				binding = `{"value_from_oneof":{"parameter_selectors":[{"producer_subtask":"skipped","output_parameter_key":"accuracy"},{"producer_subtask":"train","output_parameter_key":"accuracy"}]}}`
			}
			g["executions"][0]["custom_properties"].(map[string]any)["parameter_producer_task"] = map[string]any{"struct_value": map[string]any{"score": binding}}
			out, err := convertLegacyMLMD("source", "ns", r, g, e)
			require.NoError(t, err)
			require.Len(t, out[0].Tasks[0].OutputParameters, 1)
			data, err := json.Marshal(out[0].Tasks[0].OutputParameters)
			require.NoError(t, err)
			require.Contains(t, string(data), "score")
			require.Contains(t, string(data), "0.9")
			g["executions"][1]["custom_properties"].(map[string]any)["task_name"] = map[string]any{"string_value": "missing"}
			_, err = convertLegacyMLMD("source", "ns", r, g, e)
			require.ErrorContains(t, err, "missing successful")
		})
	}
}

func TestLegacyMLMDScopedNestedParameterBinding(t *testing.T) {
	g, r, e := legacyGraphFixture(t)
	// Match release metadata/client.go: each DAG output spec is protojson
	// serialized into a string field inside the MLMD struct property.
	binding := func(task, port string) string {
		spec := &pipelinespec.DagOutputsSpec_DagOutputParameterSpec{Kind: &pipelinespec.DagOutputsSpec_DagOutputParameterSpec_ValueFromParameter{ValueFromParameter: &pipelinespec.DagOutputsSpec_ParameterSelectorSpec{ProducerSubtask: task, OutputParameterKey: port}}}
		data, err := protojson.Marshal(spec)
		require.NoError(t, err)
		return string(data)
	}
	root := g["executions"][0]["custom_properties"].(map[string]any)
	root["parameter_producer_task"] = map[string]any{"struct_value": map[string]any{"score": binding("nested", "score")}}
	g["executions"] = append(g["executions"], map[string]any{"id": "12", "type_id": "3", "last_known_state": "COMPLETE", "custom_properties": map[string]any{"task_name": map[string]any{"string_value": "nested"}, "parent_dag_id": map[string]any{"int_value": "10"}, "parameter_producer_task": map[string]any{"struct_value": map[string]any{"score": binding("train", "accuracy")}}}}, map[string]any{"id": "13", "type_id": "4", "last_known_state": "COMPLETE", "custom_properties": map[string]any{"task_name": map[string]any{"string_value": "train"}, "parent_dag_id": map[string]any{"int_value": "12"}, "outputs": map[string]any{"struct_value": map[string]any{"accuracy": 0.8}}}})
	for _, id := range []string{"12", "13"} {
		g["associations"] = append(g["associations"], map[string]any{"context_id": "2", "execution_id": id})
	}
	out, err := convertLegacyMLMD("source", "ns", r, g, e)
	require.NoError(t, err)
	data, err := json.Marshal(out[0].Tasks[0].OutputParameters)
	require.NoError(t, err)
	require.Contains(t, string(data), "0.8")
	require.NotContains(t, string(data), "0.9")
}

func TestLegacyMLMDCacheChainReversedIDs(t *testing.T) {
	g, r, e := legacyGraphFixture(t)
	props := g["executions"][1]["custom_properties"].(map[string]any)
	delete(props, "outputs")
	props["cached_execution_id"] = map[string]any{"string_value": "12"}
	for _, id := range []string{"12", "13"} {
		p := map[string]any{"task_name": map[string]any{"string_value": "cached-" + id}, "parent_dag_id": map[string]any{"int_value": "10"}}
		if id == "12" {
			p["cached_execution_id"] = map[string]any{"string_value": "13"}
		} else {
			p["outputs"] = map[string]any{"struct_value": map[string]any{"accuracy": 0.7}}
		}
		g["executions"] = append(g["executions"], map[string]any{"id": id, "type_id": "4", "last_known_state": "COMPLETE", "custom_properties": p})
		g["associations"] = append(g["associations"], map[string]any{"context_id": "2", "execution_id": id})
	}
	g["events"][0]["execution_id"] = "13"
	out, err := convertLegacyMLMD("source", "ns", r, g, e)
	require.NoError(t, err)
	require.Len(t, out[0].Links, 3)
	for _, task := range out[0].Tasks[1:] {
		require.Len(t, task.OutputParameters, 1)
	}
}

func TestLegacyMLMDDAGArtifactBinding(t *testing.T) {
	g, r, e := legacyGraphFixture(t)
	spec := map[string]*pipelinespec.DagOutputsSpec_DagOutputArtifactSpec{"result": {ArtifactSelectors: []*pipelinespec.DagOutputsSpec_ArtifactSelectorSpec{{ProducerSubtask: "train", OutputArtifactKey: "dataset"}}}}
	// Release metadata/client.go uses encoding/json for artifact selectors.
	data, err := json.Marshal(spec)
	require.NoError(t, err)
	g["executions"][0]["custom_properties"].(map[string]any)["artifact_producer_task"] = map[string]any{"string_value": string(data)}
	out, err := convertLegacyMLMD("source", "ns", r, g, e)
	require.NoError(t, err)
	require.Len(t, out[0].Links, 2)
	require.Equal(t, "result", out[0].Links[0].ArtifactKey)
	require.Equal(t, out[0].Tasks[0].UUID, out[0].Links[0].TaskID)
	g["executions"][1]["custom_properties"].(map[string]any)["cached_execution_id"] = map[string]any{"string_value": "999"}
	_, err = convertLegacyMLMD("source", "ns", r, g, e)
	require.ErrorContains(t, err, "cached execution reference")
}

func TestLegacyMLMDLoopParameterAndArtifactCollection(t *testing.T) {
	g, r, e := legacyGraphFixture(t)
	root := g["executions"][0]["custom_properties"].(map[string]any)
	root["iteration_count"] = map[string]any{"int_value": "2"}
	parameterSpec := &pipelinespec.DagOutputsSpec_DagOutputParameterSpec{Kind: &pipelinespec.DagOutputsSpec_DagOutputParameterSpec_ValueFromParameter{ValueFromParameter: &pipelinespec.DagOutputsSpec_ParameterSelectorSpec{ProducerSubtask: "train", OutputParameterKey: "accuracy"}}}
	parameterJSON, err := protojson.Marshal(parameterSpec)
	require.NoError(t, err)
	artifactJSON, err := json.Marshal(map[string]*pipelinespec.DagOutputsSpec_DagOutputArtifactSpec{"dataset": {ArtifactSelectors: []*pipelinespec.DagOutputsSpec_ArtifactSelectorSpec{{ProducerSubtask: "train", OutputArtifactKey: "dataset"}}}})
	require.NoError(t, err)
	root["parameter_producer_task"] = map[string]any{"struct_value": map[string]any{"accuracy": string(parameterJSON)}}
	root["artifact_producer_task"] = map[string]any{"string_value": string(artifactJSON)}
	// Two iteration DAGs hold concrete resolved outputs in source metadata.
	g["executions"] = g["executions"][:1]
	g["associations"] = g["associations"][:1]
	g["events"] = nil
	for index, id := range []string{"11", "12"} {
		g["executions"] = append(g["executions"], map[string]any{"id": id, "type_id": "3", "last_known_state": "COMPLETE", "custom_properties": map[string]any{"task_name": map[string]any{"string_value": "iteration"}, "parent_dag_id": map[string]any{"int_value": "10"}, "iteration_index": map[string]any{"int_value": strconv.Itoa(index)}, "outputs": map[string]any{"struct_value": map[string]any{"accuracy": float64(index)}}}})
		g["associations"] = append(g["associations"], map[string]any{"context_id": "2", "execution_id": id})
		g["events"] = append(g["events"], map[string]any{"artifact_id": "20", "execution_id": id, "type": "OUTPUT", "path": map[string]any{"steps": []any{map[string]any{"key": "dataset"}}}})
	}
	out, err := convertLegacyMLMD("source", "ns", r, g, e)
	require.NoError(t, err)
	require.Len(t, out[0].Links, 3)
	data, err := json.Marshal(out[0].Tasks[0].OutputParameters)
	require.NoError(t, err)
	require.Contains(t, string(data), "[0,1]")
}

func TestLegacyArtifactNamespaceType(t *testing.T) {
	g, r, e := legacyGraphFixture(t)
	g["artifacts"][0]["custom_properties"] = map[string]any{"namespace": map[string]any{"int_value": "1"}}
	_, err := convertLegacyMLMD("source", "ns", r, g, e)
	require.ErrorContains(t, err, "namespace must be a string")
}

func TestLegacyArtifactBindingsRejectTrailingJSON(t *testing.T) {
	g, r, e := legacyGraphFixture(t)
	g["executions"][0]["custom_properties"].(map[string]any)["artifact_producer_task"] = map[string]any{"string_value": "{} {}"}
	_, err := convertLegacyMLMD("source", "ns", r, g, e)
	require.ErrorContains(t, err, "artifact producer bindings")
	err = legacySQLArtifactReferences(legacyTask{MLMDInputs: "{} {}"}, nil)
	require.ErrorContains(t, err, "SQL artifact bindings")
}

func TestLegacyDAGArtifactMultipleSelectors(t *testing.T) {
	g, r, e := legacyGraphFixture(t)
	g["executions"] = append(g["executions"], map[string]any{"id": "12", "type_id": "4", "last_known_state": "COMPLETE", "custom_properties": map[string]any{"task_name": map[string]any{"string_value": "second"}, "parent_dag_id": map[string]any{"int_value": "10"}}})
	g["associations"] = append(g["associations"], map[string]any{"context_id": "2", "execution_id": "12"})
	g["artifacts"] = append(g["artifacts"], map[string]any{"id": "21", "type_id": "5", "state": "LIVE", "uri": "s3://bucket/second"})
	g["events"] = append(g["events"], map[string]any{"artifact_id": "21", "execution_id": "12", "type": "OUTPUT", "path": map[string]any{"steps": []any{map[string]any{"key": "dataset"}}}})
	data, err := json.Marshal(map[string]*pipelinespec.DagOutputsSpec_DagOutputArtifactSpec{"result": {ArtifactSelectors: []*pipelinespec.DagOutputsSpec_ArtifactSelectorSpec{{ProducerSubtask: "train", OutputArtifactKey: "dataset"}, {ProducerSubtask: "second", OutputArtifactKey: "dataset"}}}})
	require.NoError(t, err)
	g["executions"][0]["custom_properties"].(map[string]any)["artifact_producer_task"] = map[string]any{"string_value": string(data)}
	out, err := convertLegacyMLMD("source", "ns", r, g, e)
	require.NoError(t, err)
	require.Len(t, out[0].Links, 4)
	require.Len(t, out[0].Artifacts, 2)
}
