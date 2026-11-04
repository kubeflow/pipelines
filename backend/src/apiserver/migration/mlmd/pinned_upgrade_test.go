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
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"

	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
	"github.com/kubeflow/pipelines/backend/src/apiserver/migration/mlmd"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/apiserver/server"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func pinnedReleaseClient(t *testing.T, fixture migrationFixture) fixtureClient {
	t.Helper()
	source := fixture.Source["mlmd"]
	id := func(value any) int64 {
		number, _ := value.(float64)
		return int64(number)
	}
	str := func(value any) string {
		text, _ := value.(string)
		return text
	}
	properties := func(table, fk string, owner int64) map[string]*pb.Value {
		result := map[string]*pb.Value{}
		for _, row := range source[table] {
			if id(row[fk]) != owner || id(row["is_custom_property"]) == 0 {
				continue
			}
			value := &pb.Value{}
			switch {
			case row["string_value"] != nil:
				value.Value = &pb.Value_StringValue{StringValue: str(row["string_value"])}
			case row["int_value"] != nil:
				value.Value = &pb.Value_IntValue{IntValue: id(row["int_value"])}
			case row["double_value"] != nil:
				value.Value = &pb.Value_DoubleValue{DoubleValue: row["double_value"].(float64)}
			case row["bool_value"] != nil:
				value.Value = &pb.Value_BoolValue{BoolValue: id(row["bool_value"]) != 0}
			default:
				t.Fatalf("unsupported captured property %s.%s", table, row["name"])
			}
			result[str(row["name"])] = value
		}
		return result
	}
	typeNames := map[int64]string{}
	for _, row := range source["types"] {
		typeNames[id(row["id"])] = str(row["name"])
	}
	client := fixtureClient{
		executions: &pb.GetExecutionsResponse{}, artifacts: &pb.GetArtifactsResponse{},
		events: &pb.GetEventsByExecutionIDsResponse{}, contexts: map[int64]*pb.GetContextsByExecutionResponse{},
	}
	contexts := map[int64]*pb.Context{}
	for _, row := range source["contexts"] {
		contexts[id(row["id"])] = &pb.Context{Name: proto.String(str(row["name"])), Type: proto.String(typeNames[id(row["type_id"])])}
	}
	for _, row := range source["executions"] {
		executionID := id(row["id"])
		client.executions.Executions = append(client.executions.Executions, &pb.Execution{
			Id: proto.Int64(executionID), Name: proto.String(str(row["name"])), Type: proto.String(typeNames[id(row["type_id"])]),
			LastKnownState:       pb.Execution_State(id(row["last_known_state"])).Enum(),
			CreateTimeSinceEpoch: proto.Int64(id(row["create_time_since_epoch"])),
			CustomProperties:     properties("execution_properties", "execution_id", executionID),
		})
		client.contexts[executionID] = &pb.GetContextsByExecutionResponse{}
	}
	for _, row := range source["associations"] {
		response := client.contexts[id(row["execution_id"])]
		response.Contexts = append(response.Contexts, contexts[id(row["context_id"])])
	}
	for _, row := range source["artifacts"] {
		artifactID := id(row["id"])
		client.artifacts.Artifacts = append(client.artifacts.Artifacts, &pb.Artifact{
			Id: proto.Int64(artifactID), Name: proto.String(str(row["name"])), Type: proto.String(typeNames[id(row["type_id"])]),
			Uri: proto.String(str(row["uri"])), CreateTimeSinceEpoch: proto.Int64(id(row["create_time_since_epoch"])),
			LastUpdateTimeSinceEpoch: proto.Int64(id(row["last_update_time_since_epoch"])),
			CustomProperties:         properties("artifact_properties", "artifact_id", artifactID),
		})
	}
	paths := map[int64][]*pb.Event_Path_Step{}
	for _, row := range source["event_paths"] {
		eventID := id(row["event_id"])
		step := &pb.Event_Path_Step{Value: &pb.Event_Path_Step_Key{Key: str(row["step_key"])}}
		if id(row["is_index_step"]) != 0 {
			step.Value = &pb.Event_Path_Step_Index{Index: id(row["step_index"])}
		}
		paths[eventID] = append(paths[eventID], step)
	}
	for _, row := range source["events"] {
		client.events.Events = append(client.events.Events, &pb.Event{
			ExecutionId: proto.Int64(id(row["execution_id"])), ArtifactId: proto.Int64(id(row["artifact_id"])),
			Type: pb.Event_Type(id(row["type"])).Enum(), Path: &pb.Event_Path{Steps: paths[id(row["id"])]},
			MillisecondsSinceEpoch: proto.Int64(id(row["milliseconds_since_epoch"])),
		})
	}
	return client
}

func TestMySQLPinned218UpgradeContract(t *testing.T) {
	db, _, _, _ := scratchMySQLDatabase(t)
	fixture := readMigrationFixture(t)
	dump := string(releaseFixtureFile(t, "source.sql"))
	prelude, remainder, found := strings.Cut(dump, "CREATE DATABASE /*!32312 IF NOT EXISTS*/ `mlpipeline`")
	require.True(t, found)
	_, source, found := strings.Cut(remainder, "USE `mlpipeline`;")
	require.True(t, found)
	source, _, found = strings.Cut(source, "CREATE DATABASE /*!32312 IF NOT EXISTS*/ `metadb`")
	require.True(t, found)
	_, err := db.Exec(prelude + source)
	require.NoError(t, err)
	_, err = db.Exec("CREATE TABLE mlmd_migration_state (ID tinyint NOT NULL PRIMARY KEY, CacheCutoffInSec bigint NOT NULL) ENGINE=InnoDB")
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO mlmd_migration_state VALUES (1, 1795060000)")
	require.NoError(t, err)

	// Recreate an interrupted cutover later with the original task rows.
	_, err = db.Exec("CREATE TABLE released_tasks LIKE tasks")
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO released_tasks SELECT * FROM tasks")
	require.NoError(t, err)

	report, err := mlmd.Migrate(context.Background(), pinnedReleaseClient(t, fixture), db, fixtureDiskBudget)
	require.NoError(t, err)
	require.Equal(t, fixture.Assertions.Counts.Target["tasks"], report.Tasks)
	require.Equal(t, fixture.Assertions.Counts.Target["artifacts"], report.Artifacts)
	require.Equal(t, fixture.Assertions.Counts.Target["artifact_tasks"], report.Links)
	require.Equal(t, 22, report.MigratedByKind["artifact"], "count source artifacts before metric splitting")
	require.Equal(t, 3, report.SkippedByReason["cache_key"][`task name "consume" is ambiguous in the saved pipeline spec`])
	reportJSON, err := json.Marshal(report)
	require.NoError(t, err)
	require.Contains(t, string(reportJSON), fmt.Sprintf(`"cache_keys_written":%d`, report.CacheKeysVerified))
	require.Contains(t, string(reportJSON), `"cache_keys_preserved":0`)
	for _, row := range fixture.Expected.Tasks {
		var runID string
		var state, taskType int
		var inputs, outputs []byte
		require.NoError(t, db.QueryRow("SELECT RunUUID, State, Type, InputParameters, OutputParameters FROM tasks WHERE UUID=?", row.TargetUUID).
			Scan(&runID, &state, &taskType, &inputs, &outputs), row.Source)
		require.Equal(t, row.Fields["RunUUID"], runID, row.Source)
		require.EqualValues(t, row.Fields["State"], state, row.Source)
		require.EqualValues(t, row.Fields["Type"], taskType, row.Source)
		for column, data := range map[string][]byte{"InputParameters": inputs, "OutputParameters": outputs} {
			expected, err := json.Marshal(row.Fields[column])
			require.NoError(t, err)
			require.JSONEq(t, string(expected), string(data), "%s %s", row.Source, column)
		}
	}
	for _, row := range fixture.Expected.ArtifactTasks {
		var taskID, artifactID, key string
		require.NoError(t, db.QueryRow("SELECT TaskID, ArtifactID, ArtifactKey FROM artifact_tasks WHERE UUID=?", row.TargetUUID).
			Scan(&taskID, &artifactID, &key), row.Source)
		require.Equal(t, row.Fields["TaskID"], taskID, row.Source)
		require.Equal(t, row.Fields["ArtifactID"], artifactID, row.Source)
		require.Equal(t, row.Fields["ArtifactKey"], key, row.Source)
	}
	for _, row := range fixture.Source["kfp"]["run_details"] {
		var manifest string
		require.NoError(t, db.QueryRow("SELECT WorkflowRuntimeManifest FROM run_details WHERE UUID=?", row["UUID"]).Scan(&manifest))
		require.Equal(t, row["WorkflowRuntimeManifest"], manifest)
	}
	var sourceID, fingerprint string
	require.NoError(t, db.QueryRow("SELECT CachedTaskUUID FROM tasks WHERE UUID=?", mlmd.ExecutionUUID(7)).Scan(&sourceID))
	require.Equal(t, mlmd.ExecutionUUID(2), sourceID)
	require.NoError(t, db.QueryRow("SELECT Fingerprint FROM tasks WHERE UUID=?", sourceID).Scan(&fingerprint))
	require.Equal(t, strings.TrimSpace(string(releaseFixtureFile(t, "produce_native_fingerprint.txt"))), fingerprint)
	_, err = db.Exec("RENAME TABLE released_tasks TO tasks_pre_migration")
	require.NoError(t, err)
	primary, secondary := mlmd.ArtifactUUID(3), mlmd.MetricUUID(3, "loss")
	_, err = db.Exec("DELETE FROM artifacts WHERE UUID IN (?, ?)", primary, secondary)
	require.NoError(t, err)
	_, err = db.Exec("UPDATE tasks SET Fingerprint='', ParentTaskUUID=NULL WHERE UUID=?", sourceID)
	require.NoError(t, err)
	_, err = db.Exec(fmt.Sprintf(`CREATE TRIGGER interrupt_metric BEFORE INSERT ON artifacts FOR EACH ROW
		BEGIN IF NEW.UUID = '%s' THEN SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'metric interruption'; END IF; END`, secondary))
	require.NoError(t, err)
	_, err = mlmd.Migrate(context.Background(), pinnedReleaseClient(t, fixture), db, fixtureDiskBudget)
	require.ErrorContains(t, err, "metric interruption")
	var partialRows int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM artifacts WHERE UUID IN (?, ?)", primary, secondary).Scan(&partialRows))
	require.Zero(t, partialRows, "split artifact must be written atomically")
	_, err = db.Exec("DROP TRIGGER interrupt_metric")
	require.NoError(t, err)
	resumedSource := pinnedReleaseClient(t, fixture)
	slices.Reverse(resumedSource.events.Events)
	report, err = mlmd.Migrate(context.Background(), resumedSource, db, fixtureDiskBudget)
	require.NoError(t, err)
	reportJSON, err = json.Marshal(report)
	require.NoError(t, err)
	require.Contains(t, string(reportJSON), `"cache_keys_written":1`)
	require.Contains(t, string(reportJSON), fmt.Sprintf(`"cache_keys_preserved":%d`, report.CacheKeysVerified-1))
	require.Positive(t, report.CacheKeysPreserved)
	_, err = mlmd.Migrate(context.Background(), nil, db, fixtureDiskBudget)
	require.NoError(t, err, "completed upgrade should not need MLMD")
	var resumedKey string
	require.NoError(t, db.QueryRow("SELECT Fingerprint FROM tasks WHERE UUID=?", sourceID).Scan(&resumedKey))
	require.Equal(t, fingerprint, resumedKey)
	assertMigratedAPI(t, db, fixture, fingerprint)
}

// Use the native handlers and stores against the migrated database. Only
// services unrelated to historical reads use the fake client manager.
type migratedReadClients struct {
	*resource.FakeClientManager
	run      *storage.RunStore
	task     *storage.TaskStore
	artifact *storage.ArtifactStore
	link     *storage.ArtifactTaskStore
}

func (c migratedReadClients) RunStore() storage.RunStoreInterface                   { return c.run }
func (c migratedReadClients) TaskStore() storage.TaskStoreInterface                 { return c.task }
func (c migratedReadClients) ArtifactStore() storage.ArtifactStoreInterface         { return c.artifact }
func (c migratedReadClients) ArtifactTaskStore() storage.ArtifactTaskStoreInterface { return c.link }

func assertMigratedAPI(t *testing.T, db *sql.DB, fixture migrationFixture, fingerprint string) {
	t.Helper()
	// The surrounding KFP schema is reconciled by startup in PR 3.
	// This nullable column is required to read runs with the current API.
	_, err := db.Exec("ALTER TABLE run_details ADD COLUMN EnableCaching boolean NULL, ALGORITHM=INSTANT")
	require.NoError(t, err)
	clock, uuid := util.NewFakeTimeForEpoch(), util.NewUUIDGenerator()
	d := dialect.NewDBDialect("mysql")
	clients := migratedReadClients{
		FakeClientManager: resource.NewFakeClientManagerOrFatalV2(),
		run:               storage.NewRunStore(db, clock, d), task: storage.NewTaskStore(db, clock, uuid, d),
		artifact: storage.NewArtifactStore(db, clock, uuid, d), link: storage.NewArtifactTaskStore(db, uuid, d),
	}
	experiments, err := storage.NewExperimentStore(db, clock, uuid, d)
	require.NoError(t, err)
	clients.SetExperimentStore(experiments)
	manager := resource.NewResourceManager(clients, &resource.ResourceManagerOptions{})
	runs := server.NewRunServer(manager, &server.RunServerOptions{})
	artifacts := server.NewArtifactServer(manager)
	ctx := context.Background()

	listedRuns, err := runs.ListRuns(ctx, &apiv2.ListRunsRequest{PageSize: 100})
	require.NoError(t, err)
	require.Len(t, listedRuns.GetRuns(), len(fixture.Expected.API))
	for _, expected := range fixture.Expected.API {
		run, err := runs.GetRun(ctx, &apiv2.GetRunRequest{RunId: expected.RunID, View: apiv2.GetRunRequest_FULL.Enum()})
		require.NoError(t, err)
		require.Len(t, run.GetTasks(), len(expected.TaskIDs))
		listed, err := runs.ListTasks(ctx, &apiv2.ListTasksRequest{RunId: expected.RunID, PageSize: 100})
		require.NoError(t, err)
		var ids []string
		for _, task := range listed.GetTasks() {
			ids = append(ids, task.GetTaskId())
		}
		require.ElementsMatch(t, expected.TaskIDs, ids)
	}
	for _, expected := range fixture.Expected.Tasks {
		task, err := runs.GetTask(ctx, &apiv2.GetTaskRequest{RunId: expected.Fields["RunUUID"].(string), TaskId: expected.TargetUUID})
		require.NoError(t, err)
		require.Equal(t, expected.TargetUUID, task.GetTaskId())
		require.EqualValues(t, expected.Fields["State"], task.GetState())
		parent, _ := expected.Fields["ParentTaskUUID"].(string)
		require.Equal(t, parent, task.GetParentTaskId())
	}
	listedArtifacts, err := artifacts.ListArtifacts(ctx, &apiv2.ListArtifactRequest{Namespace: "kubeflow", PageSize: 100})
	require.NoError(t, err)
	require.Len(t, listedArtifacts.GetArtifacts(), len(fixture.Expected.Artifacts))
	for _, expected := range fixture.Expected.Artifacts {
		artifact, err := artifacts.GetArtifact(ctx, &apiv2.GetArtifactRequest{ArtifactId: expected.TargetUUID})
		require.NoError(t, err)
		require.Equal(t, expected.Fields["Name"], artifact.GetName())
		require.Equal(t, expected.Fields["Namespace"], artifact.GetNamespace())
	}
	linkByID := map[string]map[string]any{}
	for _, link := range fixture.Expected.ArtifactTasks {
		linkByID[link.TargetUUID] = link.Fields
	}
	for _, expected := range fixture.Expected.API {
		links, err := artifacts.ListArtifactTasks(ctx, &apiv2.ListArtifactTasksRequest{RunIds: []string{expected.RunID}, PageSize: 100})
		require.NoError(t, err)
		var want, actual []string
		for id, link := range linkByID {
			if link["RunUUID"] == expected.RunID {
				want = append(want, id)
			}
		}
		for _, link := range links.GetArtifactTasks() {
			expected, exists := linkByID[link.GetId()]
			require.True(t, exists, "unexpected link %s", link.GetId())
			require.Equal(t, expected["ArtifactID"], link.GetArtifactId())
			require.Equal(t, expected["TaskID"], link.GetTaskId())
			require.Equal(t, expected["ArtifactKey"], link.GetKey())
			require.EqualValues(t, expected["Type"], link.GetType())
			actual = append(actual, link.GetId())
		}
		require.ElementsMatch(t, want, actual)
	}
	expectedLink := fixture.Expected.ArtifactTasks[0].Fields
	taskID, artifactID := expectedLink["TaskID"].(string), expectedLink["ArtifactID"].(string)
	for _, request := range []*apiv2.ListArtifactTasksRequest{
		{TaskIds: []string{taskID}, PageSize: 100},
		{ArtifactIds: []string{artifactID}, PageSize: 100},
	} {
		links, err := artifacts.ListArtifactTasks(ctx, request)
		require.NoError(t, err)
		var want, actual []string
		for id, link := range linkByID {
			if link["TaskID"] == taskID && len(request.TaskIds) != 0 ||
				link["ArtifactID"] == artifactID && len(request.ArtifactIds) != 0 {
				want = append(want, id)
			}
		}
		for _, link := range links.GetArtifactTasks() {
			actual = append(actual, link.GetId())
		}
		require.ElementsMatch(t, want, actual)
	}
	hit, err := runs.FindCachedTask(ctx, &apiv2.FindCachedTaskRequest{Namespace: "kubeflow", CacheFingerprint: fingerprint})
	require.NoError(t, err)
	require.Equal(t, mlmd.ExecutionUUID(2), hit.GetTask().GetTaskId())
	require.Len(t, hit.GetTask().GetOutputs().GetParameters(), 1)
	require.Equal(t, "fixture-produced", hit.GetTask().GetOutputs().GetParameters()[0].GetValue().GetStringValue())
	for _, row := range fixture.Expected.Tasks {
		if row.TargetUUID == mlmd.ExecutionUUID(7) {
			cached, err := runs.GetTask(ctx, &apiv2.GetTaskRequest{RunId: row.Fields["RunUUID"].(string), TaskId: row.TargetUUID})
			require.NoError(t, err)
			require.Equal(t, mlmd.ExecutionUUID(2), cached.GetCachedTaskId())
		}
	}
}
