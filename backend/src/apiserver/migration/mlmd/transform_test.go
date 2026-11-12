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

package mlmd_test

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/migration/mlmd"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func stringValue(value string) *pb.Value {
	return &pb.Value{Value: &pb.Value_StringValue{StringValue: value}}
}

func TestReleasedDirectWriterArtifactTransforms(t *testing.T) {
	var source sourceFixture
	var native nativeFixture
	readFixture(t, "legacy_source.json", &source)
	readFixture(t, "expected_native.json", &native)
	want := map[string]int{}
	for _, artifact := range native.Artifacts {
		want[artifact.UUID] = int(artifact.Type)
	}
	for _, raw := range source.Artifacts.Artifacts {
		encoded, err := json.Marshal(raw)
		require.NoError(t, err)
		var artifact pb.Artifact
		require.NoError(t, protojson.Unmarshal(encoded, &artifact))
		for _, row := range mlmd.TransformArtifact(&artifact, "migration-fixture") {
			require.Equal(t, want[row.UUID], int(row.Type), row.UUID)
			delete(want, row.UUID)
		}
	}
	require.Empty(t, want)
}

func TestExecutionIdentityAndStateGate(t *testing.T) {
	id := int64(13)
	require.Equal(t, uuid.NewSHA1(uuid.NameSpaceDNS, []byte("execution:13")).String(), mlmd.ExecutionUUID(id))
	require.NotEqual(t, mlmd.ExecutionUUID(id), mlmd.ArtifactUUID(id))
	for _, scenario := range []struct {
		state       pb.Execution_State
		want        apiv2.PipelineTask_TaskState
		fingerprint string
	}{
		{pb.Execution_COMPLETE, apiv2.PipelineTask_SUCCEEDED, "source-key"},
		{pb.Execution_CACHED, apiv2.PipelineTask_CACHED, "source-key"},
		{pb.Execution_CANCELED, apiv2.PipelineTask_FAILED, ""},
		{pb.Execution_RUNNING, apiv2.PipelineTask_RUNNING, ""},
	} {
		execution := &pb.Execution{Id: proto.Int64(id), Type: proto.String("system.ContainerExecution"), LastKnownState: &scenario.state,
			CustomProperties: map[string]*pb.Value{"cache_fingerprint": stringValue("source-key")}}
		task, parent, err := mlmd.TransformExecution(execution, "run-id", "namespace")
		require.NoError(t, err)
		require.Zero(t, parent)
		require.Equal(t, scenario.want, apiv2.PipelineTask_TaskState(task.State))
		require.Empty(t, task.Fingerprint)
		if scenario.fingerprint != "" {
			require.Equal(t, scenario.fingerprint, task.StatusMetadata["customProperties"].(map[string]any)["legacy_cache_fingerprint"])
		}
	}
}

func TestExecutionTaskTypes(t *testing.T) {
	parent := &pb.Value{Value: &pb.Value_IntValue{IntValue: 1}}
	iteration := &pb.Value{Value: &pb.Value_IntValue{IntValue: 0}}
	truth := &pb.Value{Value: &pb.Value_BoolValue{BoolValue: true}}
	for _, scenario := range []struct {
		name, kind string
		properties map[string]*pb.Value
		want       apiv2.PipelineTask_TaskType
	}{
		{"root", "system.DAGExecution", nil, apiv2.PipelineTask_ROOT},
		{"nested", "system.DAGExecution", map[string]*pb.Value{"parent_dag_id": parent}, apiv2.PipelineTask_DAG},
		{"condition-branches-main", "system.DAGExecution", map[string]*pb.Value{"parent_dag_id": parent}, apiv2.PipelineTask_CONDITION_BRANCH},
		{"condition-main", "system.DAGExecution", map[string]*pb.Value{"parent_dag_id": parent}, apiv2.PipelineTask_CONDITION},
		{"loop", "system.DAGExecution", map[string]*pb.Value{"parent_dag_id": parent, "iteration_count": iteration}, apiv2.PipelineTask_LOOP},
		{"on-exit-main", "system.DAGExecution", map[string]*pb.Value{"parent_dag_id": parent}, apiv2.PipelineTask_EXIT_HANDLER},
		{"custom", "system.DAGExecution", map[string]*pb.Value{"parent_dag_id": parent, "is_exit_handler": truth}, apiv2.PipelineTask_EXIT_HANDLER},
		{"importer", "system.ImporterExecution", nil, apiv2.PipelineTask_IMPORTER},
		{"runtime", "system.ContainerExecution", nil, apiv2.PipelineTask_RUNTIME},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			execution := &pb.Execution{Id: proto.Int64(4), Name: proto.String(scenario.name), Type: proto.String(scenario.kind), CustomProperties: scenario.properties}
			task, _, err := mlmd.TransformExecution(execution, "run", "team")
			require.NoError(t, err)
			require.Equal(t, scenario.want, apiv2.PipelineTask_TaskType(task.Type))
		})
	}
}

func TestMetricSplitUsesPrimaryLineageIdentity(t *testing.T) {
	artifact := &pb.Artifact{Id: proto.Int64(7), Type: proto.String("system.Metrics"), Uri: proto.String("s3://metrics"),
		CustomProperties: map[string]*pb.Value{
			"accuracy": {Value: &pb.Value_DoubleValue{DoubleValue: 0.9}},
			"loss":     {Value: &pb.Value_DoubleValue{DoubleValue: 0.1}},
		}}
	rows := mlmd.TransformArtifact(artifact, "namespace")
	require.Len(t, rows, 2)
	require.Equal(t, mlmd.ArtifactUUID(7), rows[0].UUID)
	require.Equal(t, "accuracy", rows[0].Name)
	require.Equal(t, mlmd.MetricUUID(7, "loss"), rows[1].UUID)
	require.Equal(t, "loss", rows[1].Name)
	require.Equal(t, 0.1, *rows[1].NumberValue)
}

func TestLargeMetricSplitIsDeterministic(t *testing.T) {
	properties := map[string]*pb.Value{}
	for index := 0; index < 50; index++ {
		properties[fmt.Sprintf("metric-%02d", index)] = &pb.Value{Value: &pb.Value_DoubleValue{DoubleValue: float64(index)}}
	}
	artifact := &pb.Artifact{Id: proto.Int64(99), Type: proto.String("system.Metrics"), CustomProperties: properties}
	first := mlmd.TransformArtifact(artifact, "team")
	second := mlmd.TransformArtifact(artifact, "team")
	require.Len(t, first, 50)
	require.Equal(t, first, second)
	require.Equal(t, mlmd.ArtifactUUID(99), first[0].UUID)
	seen := map[string]bool{}
	for _, row := range first {
		require.False(t, seen[row.UUID])
		seen[row.UUID] = true
	}
}

func TestEventLinksPrimaryMetricRow(t *testing.T) {
	eventType := pb.Event_OUTPUT
	event := &pb.Event{ArtifactId: proto.Int64(7), ExecutionId: proto.Int64(13), Type: &eventType}
	link, err := mlmd.TransformEvent(event, "run-id", "metrics", nil)
	require.NoError(t, err)
	require.Equal(t, mlmd.ArtifactUUID(7), link.ArtifactID)
	require.Equal(t, mlmd.ExecutionUUID(13), link.TaskID)
	require.Equal(t, "run-id", link.RunUUID)
	require.Equal(t, "metrics", link.ArtifactKey)
}

func TestLoopOutputRetainsProducerIteration(t *testing.T) {
	intProperty := func(value int64) *pb.Value {
		return &pb.Value{Value: &pb.Value_IntValue{IntValue: value}}
	}
	context := &pb.Context{Type: proto.String("system.PipelineRun"), Name: proto.String("run")}
	eventType := pb.Event_OUTPUT
	snapshot := &mlmd.Snapshot{
		Executions: []*pb.Execution{
			{Id: proto.Int64(10), Name: proto.String("root"), Type: proto.String("system.DAGExecution")},
			{Id: proto.Int64(15), Name: proto.String("iteration-0"), Type: proto.String("system.DAGExecution"), CustomProperties: map[string]*pb.Value{"parent_dag_id": intProperty(10), "iteration_index": intProperty(0)}},
			{Id: proto.Int64(19), Name: proto.String("double"), Type: proto.String("system.ContainerExecution"), CustomProperties: map[string]*pb.Value{"parent_dag_id": intProperty(15)}},
		},
		Artifacts: []*pb.Artifact{{Id: proto.Int64(6), Type: proto.String("system.Artifact")}},
		Events:    []*pb.Event{{ExecutionId: proto.Int64(19), ArtifactId: proto.Int64(6), Type: &eventType}},
		Contexts:  map[int64][]*pb.Context{10: {context}, 15: {context}, 19: {context}},
	}
	plan := mlmd.BuildPlan(snapshot, map[string]string{"run": "team"}, nil)
	require.Empty(t, plan.Skipped)
	require.Len(t, plan.Links, 1)
	require.Equal(t, int64(0), plan.Links[0].Iteration)
	require.Equal(t, model.JSONData{"taskName": "double", "iteration": "0"}, plan.Links[0].Producer)
}

func TestMissingParentSkipsChildAndItsLineage(t *testing.T) {
	execution := &pb.Execution{Id: proto.Int64(42), Type: proto.String("system.ContainerExecution"), CustomProperties: map[string]*pb.Value{
		"parent_dag_id": {Value: &pb.Value_IntValue{IntValue: 999}},
	}}
	context := &pb.Context{Type: proto.String("system.PipelineRun"), Name: proto.String("run")}
	eventType := pb.Event_OUTPUT
	snapshot := &mlmd.Snapshot{
		Executions: []*pb.Execution{execution},
		Artifacts:  []*pb.Artifact{{Id: proto.Int64(7), Type: proto.String("system.Dataset")}},
		Events:     []*pb.Event{{ExecutionId: proto.Int64(42), ArtifactId: proto.Int64(7), Type: &eventType}},
		Contexts:   map[int64][]*pb.Context{42: {context}},
	}
	plan := mlmd.BuildPlan(snapshot, map[string]string{"run": "team"}, nil)
	require.Empty(t, plan.Tasks)
	require.Empty(t, plan.Artifacts)
	require.Empty(t, plan.Links)
	require.Contains(t, plan.Skipped, mlmd.Skip{Kind: "execution", ID: 42, Reason: "parent execution 999 is absent from the same run"})
}

func TestOversizedExecutionIsSkipped(t *testing.T) {
	execution := &pb.Execution{Id: proto.Int64(42), Type: proto.String("system.ContainerExecution"), CustomProperties: map[string]*pb.Value{
		"display_name": stringValue(strings.Repeat("x", 129)),
	}}
	context := &pb.Context{Type: proto.String("system.PipelineRun"), Name: proto.String("run")}
	plan := mlmd.BuildPlan(&mlmd.Snapshot{Executions: []*pb.Execution{execution}, Contexts: map[int64][]*pb.Context{42: {context}}}, map[string]string{"run": "team"}, nil)
	require.Empty(t, plan.Tasks)
	require.Len(t, plan.Skipped, 1)
	require.Equal(t, int64(42), plan.Skipped[0].ID)
}

func TestJoinedTaskPrecedenceAndFallback(t *testing.T) {
	complete := pb.Execution_COMPLETE
	execution := &pb.Execution{Id: proto.Int64(12), Type: proto.String("system.ContainerExecution"), LastKnownState: &complete,
		CreateTimeSinceEpoch: proto.Int64(100000), CustomProperties: map[string]*pb.Value{
			"pod_name": stringValue("mlmd-pod"), "task_name": stringValue("joined"),
		}}
	context := &pb.Context{Type: proto.String("system.PipelineRun"), Name: proto.String("run")}
	snapshot := &mlmd.Snapshot{Executions: []*pb.Execution{execution}, Contexts: map[int64][]*pb.Context{12: {context}}}
	for _, scenario := range []struct {
		name    string
		legacy  mlmd.LegacyTask
		state   apiv2.PipelineTask_TaskState
		pod     string
		created int64
	}{
		{"KFP precedence", mlmd.LegacyTask{RunUUID: "run", Namespace: "team", PodName: "kfp-pod", State: "SKIPPED", CreatedAtInSec: 123}, apiv2.PipelineTask_SKIPPED, "kfp-pod", 123},
		{"MLMD fallback", mlmd.LegacyTask{RunUUID: "run", Namespace: "team", State: "RUNTIME_STATE_UNSPECIFIED"}, apiv2.PipelineTask_SUCCEEDED, "mlmd-pod", 100},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			plan := mlmd.BuildPlan(snapshot, map[string]string{"run": "team"}, map[int64]mlmd.LegacyTask{12: scenario.legacy})
			require.Len(t, plan.Tasks, 1)
			task := plan.Tasks[0]
			require.Equal(t, scenario.state, apiv2.PipelineTask_TaskState(task.State))
			require.Equal(t, scenario.pod, task.Pods[0].(map[string]any)["name"])
			require.Equal(t, scenario.created, task.CreatedAtInSec)
		})
	}
}

func TestBuildPlanFromPinnedSource(t *testing.T) {
	data, err := os.ReadFile("testdata/legacy_source.json")
	require.NoError(t, err)
	var source map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &source))
	var executions pb.GetExecutionsResponse
	var artifacts pb.GetArtifactsResponse
	var events pb.GetEventsByExecutionIDsResponse
	require.NoError(t, protojson.Unmarshal(source["executions"], &executions))
	require.NoError(t, protojson.Unmarshal(source["artifacts"], &artifacts))
	require.NoError(t, protojson.Unmarshal(source["events"], &events))
	var contexts map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(source["execution_contexts"], &contexts))
	snapshot := &mlmd.Snapshot{Executions: executions.GetExecutions(), Artifacts: artifacts.GetArtifacts(), Events: events.GetEvents(), Contexts: map[int64][]*pb.Context{}}
	for id, raw := range contexts {
		parsedID, err := strconv.ParseInt(id, 10, 64)
		require.NoError(t, err)
		var response pb.GetContextsByExecutionResponse
		require.NoError(t, protojson.Unmarshal(raw, &response))
		snapshot.Contexts[parsedID] = response.GetContexts()
	}
	var fixture sourceFixture
	readFixture(t, "legacy_source.json", &fixture)
	legacy := map[int64]mlmd.LegacyTask{fixture.ExecutionIDs["consumer"]: {
		ExecutionID: fixture.ExecutionIDs["consumer"], RunUUID: fixture.LegacyTask.RunID,
		Namespace: fixture.LegacyTask.Namespace, PodName: fixture.LegacyTask.PodName,
		State: fixture.LegacyTask.State, CreatedAtInSec: fixture.LegacyTask.CreatedTimestamp,
		StartedInSec: fixture.LegacyTask.StartedTimestamp, FinishedInSec: fixture.LegacyTask.FinishedTimestamp,
	}}
	plan := mlmd.BuildPlan(snapshot, map[string]string{fixture.LegacyTask.RunID: fixture.LegacyTask.Namespace}, legacy)
	var expected nativeFixture
	readFixture(t, "expected_native.json", &expected)
	require.Len(t, plan.Tasks, 12)
	require.Len(t, plan.Artifacts, 4)
	require.Len(t, plan.Links, 8)
	require.Len(t, plan.Skipped, 2)
	var consumerFound bool
	for _, task := range plan.Tasks {
		if task.UUID == mlmd.ExecutionUUID(fixture.ExecutionIDs["consumer"]) {
			consumerFound = true
			require.Equal(t, fixture.LegacyTask.CreatedTimestamp, task.CreatedAtInSec)
			require.Equal(t, fixture.LegacyTask.PodName, task.Pods[0].(map[string]any)["name"])
		}
	}
	require.True(t, consumerFound)
	expectedTasks := map[string]model.Task{}
	for _, task := range expected.Tasks {
		expectedTasks[task.UUID] = task
	}
	for _, task := range plan.Tasks {
		want := expectedTasks[task.UUID]
		require.Equal(t, want.Type, task.Type, task.Name)
		require.Equal(t, want.DisplayName, task.DisplayName, task.Name)
		require.Equal(t, want.State, task.State, task.Name)
		require.Equal(t, want.ParentTaskUUID, task.ParentTaskUUID, task.Name)
		require.Equal(t, want.TypeAttrs, task.TypeAttrs, task.Name)
		require.Equal(t, want.Fingerprint, task.Fingerprint, task.Name)
		require.Equal(t, want.CreatedAtInSec, task.CreatedAtInSec, task.Name)
		require.Equal(t, want.StartedInSec, task.StartedInSec, task.Name)
		require.Equal(t, want.FinishedInSec, task.FinishedInSec, task.Name)
		require.Equal(t, want.InputParameters, task.InputParameters, task.Name)
		require.Equal(t, want.OutputParameters, task.OutputParameters, task.Name)
		require.Equal(t, want.StatusMetadata, task.StatusMetadata, task.Name)
		require.Equal(t, want.Pods, task.Pods, task.Name)
	}
	expectedArtifacts := map[string]model.Artifact{}
	for _, artifact := range expected.Artifacts {
		expectedArtifacts[artifact.UUID] = artifact
	}
	for _, artifact := range plan.Artifacts {
		want := expectedArtifacts[artifact.UUID]
		require.Equal(t, want.Type, artifact.Type, artifact.UUID)
		require.Equal(t, want.Name, artifact.Name, artifact.UUID)
		require.Equal(t, want.Description, artifact.Description, artifact.UUID)
		require.Equal(t, want.URI, artifact.URI, artifact.UUID)
		require.Equal(t, want.URIHash, artifact.URIHash, artifact.UUID)
		require.Equal(t, want.CreatedAtInSec, artifact.CreatedAtInSec, artifact.UUID)
		require.Equal(t, want.LastUpdateInSec, artifact.LastUpdateInSec, artifact.UUID)
		require.Equal(t, want.Metadata, artifact.Metadata, artifact.UUID)
		require.Equal(t, want.NumberValue, artifact.NumberValue, artifact.UUID)
	}
	expectedLinks := map[string]model.ArtifactTask{}
	for _, link := range expected.Links {
		expectedLinks[link.UUID] = link
	}
	for _, link := range plan.Links {
		want, found := expectedLinks[link.UUID]
		require.True(t, found, link.UUID)
		require.Equal(t, want.TaskID, link.TaskID)
		require.Equal(t, want.ArtifactID, link.ArtifactID)
		require.Equal(t, want.Type, link.Type)
		require.Equal(t, want.ArtifactKey, link.ArtifactKey)
	}
}
