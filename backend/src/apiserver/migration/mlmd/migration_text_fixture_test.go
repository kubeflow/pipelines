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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type fixtureRow struct {
	Source      string         `json:"source"`
	TargetUUID  string         `json:"target_uuid"`
	Disposition string         `json:"disposition"`
	Reason      string         `json:"reason"`
	RunUUID     string         `json:"run_uuid"`
	Fields      map[string]any `json:"fields"`
}

type migrationFixture struct {
	ContractStatus  string `json:"contract_status"`
	DesignConflicts []struct {
		Topic  string `json:"topic"`
		Status string `json:"status"`
	} `json:"design_conflicts"`
	Source   map[string]map[string][]map[string]any `json:"source"`
	Expected struct {
		API []struct {
			RunID   string   `json:"run_id"`
			TaskIDs []string `json:"task_ids"`
		} `json:"api"`
		RunDetails     []fixtureRow `json:"run_details"`
		Tasks          []fixtureRow `json:"tasks"`
		OldTaskAliases []fixtureRow `json:"old_task_aliases"`
		Skips          []fixtureRow `json:"skips"`
		Artifacts      []fixtureRow `json:"artifacts"`
		ArtifactTasks  []fixtureRow `json:"artifact_tasks"`
	} `json:"expected"`
	Dispositions []struct {
		Source  string   `json:"source"`
		Status  string   `json:"status"`
		Targets []string `json:"targets"`
		Reason  string   `json:"reason"`
	} `json:"dispositions"`
	Assertions struct {
		TargetCountsStatus string `json:"target_counts_status"`
		Counts             struct {
			Source map[string]int `json:"source"`
			Target map[string]int `json:"target"`
		} `json:"counts"`
	} `json:"assertions"`
}

func readMigrationFixture(t *testing.T) migrationFixture {
	t.Helper()
	var fixture migrationFixture
	require.NoError(t, json.Unmarshal(releaseFixtureFile(t, "migration_fixture.json"), &fixture))
	return fixture
}

func TestMigrationTextFixture(t *testing.T) {
	// The independent mysqldump parser compares every captured source value,
	// including properties that the target design has not yet mapped.
	command := exec.Command("python3", filepath.Join("fixturetools", "build_migration_fixture.py"), "--check")
	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))
	command = exec.Command("python3", filepath.Join("fixturetools", "test_build_migration_fixture.py"))
	output, err = command.CombinedOutput()
	require.NoError(t, err, string(output))

	fixture := readMigrationFixture(t)
	require.Equal(t, "provisional_due_to_unresolved_fields", fixture.ContractStatus)
	require.Equal(t, "provisional", fixture.Assertions.TargetCountsStatus)
	require.Empty(t, fixture.DesignConflicts)
	require.Equal(t, map[string]int{"tasks": 34, "artifacts": 26, "artifact_tasks": 26}, fixture.Assertions.Counts.Target)
	require.Len(t, fixture.Expected.Tasks, 34)
	require.Len(t, fixture.Expected.API, 7)
	require.Len(t, fixture.Expected.OldTaskAliases, 12)
	require.Len(t, fixture.Expected.Artifacts, 26)
	require.Len(t, fixture.Expected.ArtifactTasks, 26)

	sourceFields := map[string]bool{}
	for section, tables := range fixture.Source {
		for table, rows := range tables {
			require.Equal(t, len(rows), fixture.Assertions.Counts.Source[section+"."+table])
			for index, row := range rows {
				key := row["id"]
				if key == nil {
					key = row["UUID"]
				}
				if key == nil {
					key = index
				}
				for field := range row {
					path := fmt.Sprintf("%s.%s[%v].%s", section, table, key, field)
					require.False(t, sourceFields[path], "duplicate source field %s", path)
					sourceFields[path] = true
				}
			}
		}
	}
	require.Len(t, fixture.Dispositions, len(sourceFields))
	for _, disposition := range fixture.Dispositions {
		require.True(t, sourceFields[disposition.Source], "unknown or duplicate disposition %s", disposition.Source)
		delete(sourceFields, disposition.Source)
		switch disposition.Status {
		case "mapped":
			require.NotEmpty(t, disposition.Targets, disposition.Source)
		case "mapped_pending_design":
			require.NotEmpty(t, disposition.Targets, disposition.Source)
			require.NotEmpty(t, disposition.Reason, disposition.Source)
		case "lookup_only", "pending_design", "redundant", "skip":
			require.Empty(t, disposition.Targets, disposition.Source)
			require.NotEmpty(t, disposition.Reason, disposition.Source)
		default:
			t.Fatalf("unknown disposition %s for %s", disposition.Status, disposition.Source)
		}
	}
	require.Empty(t, sourceFields)

	var provenance struct {
		Checksums map[string]string `json:"checksums_sha256"`
	}
	require.NoError(t, json.Unmarshal(releaseFixtureFile(t, "provenance.json"), &provenance))
	for name, want := range provenance.Checksums {
		actual := sha256.Sum256(releaseFixtureFile(t, name))
		require.Equal(t, want, hex.EncodeToString(actual[:]), name)
	}
}

func TestReleasedSourceContract(t *testing.T) {
	fixture := readMigrationFixture(t)
	t.Run("tasks", func(t *testing.T) {
		assertTaskMappings(t, fixture)
	})
	t.Run("artifacts and lineage", func(t *testing.T) {
		assertArtifactMappings(t, fixture)
	})
}

func migrationUUID(name string) string {
	return uuid.NewSHA1(uuid.NameSpaceDNS, []byte(name)).String()
}

func assertTaskMappings(t *testing.T, fixture migrationFixture) {
	t.Helper()
	kfp, mlmd := fixture.Source["kfp"], fixture.Source["mlmd"]
	runs := map[string]map[string]any{}
	for _, row := range kfp["run_details"] {
		runID := row["UUID"].(string)
		require.NotContains(t, runs, runID)
		runs[runID] = row
	}
	require.Len(t, fixture.Expected.RunDetails, len(runs))
	seenRuns := map[string]bool{}
	for _, row := range fixture.Expected.RunDetails {
		runID := row.Fields["UUID"].(string)
		require.NotContains(t, seenRuns, runID)
		seenRuns[runID] = true
		require.Equal(t, "kfp.run_details[UUID="+runID+"]", row.Source)
		require.Equal(t, runs[runID], row.Fields)
	}

	types := map[any]string{}
	for _, row := range mlmd["types"] {
		types[row["id"]] = row["name"].(string)
	}
	runContexts := map[any]string{}
	for _, row := range mlmd["contexts"] {
		if types[row["type_id"]] == "system.PipelineRun" {
			runID := row["name"].(string)
			require.Contains(t, runs, runID)
			runContexts[row["id"]] = runID
		}
	}
	executionRuns := map[any]string{}
	for _, row := range mlmd["associations"] {
		if runID, ok := runContexts[row["context_id"]]; ok {
			require.NotContains(t, executionRuns, row["execution_id"])
			executionRuns[row["execution_id"]] = runID
		}
	}
	properties := map[any]map[string]any{}
	for _, row := range mlmd["execution_properties"] {
		if properties[row["execution_id"]] == nil {
			properties[row["execution_id"]] = map[string]any{}
		}
		value := row["string_value"]
		if value == nil {
			value = row["int_value"]
		}
		properties[row["execution_id"]][row["name"].(string)] = value
	}

	tasks := map[string]fixtureRow{}
	taskIDs := map[string]fixtureRow{}
	for _, row := range fixture.Expected.Tasks {
		require.NotContains(t, tasks, row.Source)
		require.NotContains(t, taskIDs, row.TargetUUID)
		tasks[row.Source] = row
		taskIDs[row.TargetUUID] = row
	}
	require.Len(t, tasks, len(mlmd["executions"]))
	runTaskIDs := map[string][]string{}
	for _, source := range mlmd["executions"] {
		executionID := source["id"]
		ref := fmt.Sprintf("mlmd.executions[id=%v]", executionID)
		row, ok := tasks[ref]
		require.True(t, ok, ref)
		require.Equal(t, migrationUUID(fmt.Sprintf("execution:%v", executionID)), row.TargetUUID)
		require.Equal(t, row.TargetUUID, row.Fields["UUID"])
		runID := executionRuns[executionID]
		require.NotEmpty(t, runID, ref)
		require.Equal(t, runID, row.Fields["RunUUID"], ref)
		runTaskIDs[runID] = append(runTaskIDs[runID], row.TargetUUID)

		props := properties[executionID]
		parent := props["parent_dag_id"]
		if parent == nil {
			require.Nil(t, row.Fields["ParentTaskUUID"], ref)
		} else {
			require.Equal(t, migrationUUID(fmt.Sprintf("execution:%v", parent)), row.Fields["ParentTaskUUID"], ref)
			require.Equal(t, runID, executionRuns[parent], "parent must exist in the same run")
		}
		name, _ := props["task_name"].(string)
		displayName, _ := props["display_name"].(string)
		require.Equal(t, name, row.Fields["Name"], ref)
		require.Equal(t, displayName, row.Fields["DisplayName"], ref)
		state := int(source["last_known_state"].(float64))
		wantStates := map[int]float64{1: 0, 2: 1, 3: 2, 4: 4, 5: 5, 6: 4}
		require.Contains(t, wantStates, state)
		require.Equal(t, wantStates[state], row.Fields["State"], ref)
		fingerprint := ""
		if state == 3 || state == 5 {
			fingerprint, _ = props["cache_fingerprint"].(string)
		}
		require.Equal(t, fingerprint, row.Fields["Fingerprint"], ref)
		status, _ := row.Fields["StatusMetadata"].(map[string]any)
		custom, _ := status["customProperties"].(map[string]any)
		if cached, ok := props["cached_execution_id"].(string); ok && cached != "" {
			producerID := migrationUUID("execution:" + cached)
			require.Contains(t, taskIDs, producerID)
			require.Equal(t, producerID, custom["cached_task_id"], ref)
		} else {
			require.NotContains(t, custom, "cached_task_id", ref)
		}
		assertTaskType(t, types[source["type_id"]], name, props, row)
	}
	for _, api := range fixture.Expected.API {
		require.ElementsMatch(t, runTaskIDs[api.RunID], api.TaskIDs, api.RunID)
		delete(runTaskIDs, api.RunID)
	}
	require.Empty(t, runTaskIDs)

	aliases := map[string]fixtureRow{}
	for _, row := range fixture.Expected.OldTaskAliases {
		require.NotContains(t, aliases, row.Source)
		aliases[row.Source] = row
	}
	skips := map[string]fixtureRow{}
	for _, row := range fixture.Expected.Skips {
		require.NotContains(t, skips, row.Source)
		skips[row.Source] = row
	}
	require.Len(t, skips, 179)
	for _, row := range kfp["tasks"] {
		ref := fmt.Sprintf("kfp.tasks[UUID=%v]", row["UUID"])
		require.Contains(t, runs, row["RunUUID"])
		require.NotContains(t, taskIDs, row["UUID"], ref)
		executionID := row["MLMDExecutionID"].(string)
		if executionID == "" {
			require.NotEmpty(t, row["PodName"], ref)
			skip, ok := skips[ref]
			require.True(t, ok, ref)
			require.Equal(t, row["RunUUID"], skip.RunUUID, ref)
			require.Equal(t, "Workflow report stays in the saved run manifest", skip.Reason)
			require.Empty(t, skip.TargetUUID)
			delete(skips, ref)
			continue
		}
		require.Empty(t, row["PodName"], ref)
		alias, ok := aliases[ref]
		require.True(t, ok, ref)
		require.Equal(t, "covered_by_mlmd_execution", alias.Disposition)
		require.Equal(t, migrationUUID("execution:"+executionID), alias.TargetUUID)
		target := taskIDs[alias.TargetUUID]
		require.Equal(t, row["RunUUID"], target.Fields["RunUUID"], ref)
		require.Equal(t, float64(2), target.Fields["Type"], ref)
		require.Contains(t, []float64{2, 5}, target.Fields["State"], ref)
		require.NotEmpty(t, row["Fingerprint"], ref)
		require.Equal(t, row["Fingerprint"], target.Fields["Fingerprint"], ref)
		delete(aliases, ref)
	}
	require.Empty(t, aliases)
	require.Empty(t, skips)
}

func assertTaskType(t *testing.T, kind, name string, props map[string]any, row fixtureRow) {
	t.Helper()
	var wantType float64
	switch {
	case kind == "system.ContainerExecution":
		wantType = 2
	case kind == "system.ImporterExecution":
		wantType = 7
	case kind != "system.DAGExecution":
		t.Fatalf("unexpected execution type %q for %s", kind, row.Source)
	case props["parent_dag_id"] == nil:
		wantType = 1
	case strings.HasPrefix(name, "condition-branches-"):
		wantType = 3
	case strings.HasPrefix(name, "condition-"):
		wantType = 4
	case strings.HasPrefix(name, "exit-handler-") || strings.HasPrefix(name, "on-exit-") || strings.HasPrefix(name, "onexit-"):
		wantType = 6
	case props["iteration_count"] != nil:
		wantType = 5
	default:
		wantType = 8
	}
	require.Equal(t, wantType, row.Fields["Type"], row.Source)
}

func assertArtifactMappings(t *testing.T, fixture migrationFixture) {
	t.Helper()
	mlmd := fixture.Source["mlmd"]
	types := map[any]string{}
	for _, row := range mlmd["types"] {
		types[row["id"]] = row["name"].(string)
	}
	numericProperties := map[any]map[string]any{}
	for _, row := range mlmd["artifact_properties"] {
		value := row["double_value"]
		if value == nil {
			value = row["int_value"]
		}
		if value != nil {
			id := row["artifact_id"]
			if numericProperties[id] == nil {
				numericProperties[id] = map[string]any{}
			}
			numericProperties[id][row["name"].(string)] = value
		}
	}
	artifacts := map[string][]fixtureRow{}
	artifactIDs := map[string]bool{}
	for _, row := range fixture.Expected.Artifacts {
		require.NotContains(t, artifactIDs, row.TargetUUID)
		artifactIDs[row.TargetUUID] = true
		artifacts[row.Source] = append(artifacts[row.Source], row)
	}
	require.Len(t, artifacts, len(mlmd["artifacts"]))
	for _, source := range mlmd["artifacts"] {
		id := source["id"]
		ref := fmt.Sprintf("mlmd.artifacts[id=%v]", id)
		rows := artifacts[ref]
		require.NotEmpty(t, rows, ref)
		kind := types[source["type_id"]]
		base := fmt.Sprintf("artifact:%v", id)
		if kind == "system.Metrics" {
			metrics := numericProperties[id]
			names := make([]string, 0, len(metrics))
			for name := range metrics {
				names = append(names, name)
			}
			sort.Strings(names)
			require.Len(t, rows, len(names), ref)
			for _, row := range rows {
				name := row.Fields["Name"].(string)
				require.Contains(t, metrics, name, ref)
				uuidName := base
				if name != names[0] {
					uuidName += ":metric:" + name
				}
				require.Equal(t, migrationUUID(uuidName), row.TargetUUID, ref)
				require.Equal(t, float64(6), row.Fields["Type"], ref)
				require.Equal(t, metrics[name], row.Fields["NumberValue"], ref)
			}
		} else {
			require.Len(t, rows, 1, ref)
			wantTypes := map[string]float64{"system.Dataset": 3, "system.Artifact": 1}
			require.Contains(t, wantTypes, kind)
			require.Equal(t, wantTypes[kind], rows[0].Fields["Type"], ref)
			require.Equal(t, migrationUUID(base), rows[0].TargetUUID, ref)
			require.Nil(t, rows[0].Fields["NumberValue"], ref)
		}
		for _, row := range rows {
			require.Equal(t, source["uri"], row.Fields["URI"], ref)
			require.Equal(t, row.TargetUUID, row.Fields["UUID"], ref)
		}
	}

	paths := map[any]any{}
	for _, row := range mlmd["event_paths"] {
		require.NotContains(t, paths, row["event_id"])
		paths[row["event_id"]] = row["step_key"]
	}
	tasks := map[string]fixtureRow{}
	for _, row := range fixture.Expected.Tasks {
		tasks[row.TargetUUID] = row
	}
	links := map[string]fixtureRow{}
	for _, row := range fixture.Expected.ArtifactTasks {
		require.NotContains(t, links, row.Source)
		links[row.Source] = row
	}
	require.Len(t, links, len(mlmd["events"]))
	for _, source := range mlmd["events"] {
		id := source["id"]
		ref := fmt.Sprintf("mlmd.events[id=%v]", id)
		row, ok := links[ref]
		require.True(t, ok, ref)
		require.Equal(t, migrationUUID(fmt.Sprintf("event:%v", id)), row.TargetUUID)
		require.Equal(t, row.TargetUUID, row.Fields["UUID"])
		require.Equal(t, float64(4), source["type"], ref)
		require.Equal(t, float64(9), row.Fields["Type"], ref)
		taskID := migrationUUID(fmt.Sprintf("execution:%v", source["execution_id"]))
		require.Contains(t, tasks, taskID, ref)
		require.Equal(t, taskID, row.Fields["TaskID"], ref)
		require.Equal(t, tasks[taskID].Fields["RunUUID"], row.Fields["RunUUID"], ref)
		artifactID := migrationUUID(fmt.Sprintf("artifact:%v", source["artifact_id"]))
		require.Contains(t, artifactIDs, artifactID, ref)
		require.Equal(t, artifactID, row.Fields["ArtifactID"], ref)
		require.Contains(t, paths, id)
		require.Equal(t, paths[id], row.Fields["ArtifactKey"], ref)
	}
}

func TestMigrationTextCases(t *testing.T) {
	var table struct {
		FormatVersion int                       `json:"format_version"`
		Cases         map[string]map[string]any `json:"cases"`
	}
	require.NoError(t, json.Unmarshal(releaseFixtureFile(t, "cases.json"), &table))
	require.Equal(t, 1, table.FormatVersion)
	require.Len(t, table.Cases, 5)

	readCase := func(t *testing.T, name string) map[string]any {
		t.Helper()
		fixture, ok := table.Cases[name]
		require.True(t, ok, "missing case %s", name)
		return fixture
	}
	objects := func(t *testing.T, value any) []map[string]any {
		t.Helper()
		array := value.([]any)
		result := make([]map[string]any, len(array))
		for index, item := range array {
			result[index] = item.(map[string]any)
		}
		return result
	}
	uuidFor := func(kind string, sourceID float64) string {
		return uuid.NewSHA1(uuid.NameSpaceDNS,
			[]byte(kind+":"+strconv.Itoa(int(sourceID)))).String()
	}

	t.Run("states_and_cache", func(t *testing.T) {
		states := readCase(t, "states_and_cache")
		stateSource := objects(t, states["source"].(map[string]any)["executions"])
		stateTargets := objects(t, states["expected"].(map[string]any)["tasks"])
		require.Len(t, stateTargets, 6)
		nativeStates := map[string]float64{"NEW": 0, "RUNNING": 1, "COMPLETE": 2, "FAILED": 4, "CACHED": 5, "CANCELED": 4}
		for index, target := range stateTargets {
			source := stateSource[index]
			require.Equal(t, uuidFor("execution", source["id"].(float64)), target["uuid"])
			legacyState := source["last_known_state"].(string)
			require.Contains(t, nativeStates, legacyState)
			require.Equal(t, nativeStates[legacyState], target["state"], legacyState)
			eligible := legacyState == "COMPLETE" || legacyState == "CACHED"
			if eligible {
				require.Equal(t, source["cache_fingerprint"], target["fingerprint"])
			} else {
				require.Empty(t, target["fingerprint"])
			}
		}
	})

	t.Run("task_types", func(t *testing.T) {
		types := readCase(t, "task_types")
		typeSource := objects(t, types["source"].(map[string]any)["executions"])
		typeTargets := objects(t, types["expected"].(map[string]any)["tasks"])
		require.Len(t, typeTargets, 8)
		for index, target := range typeTargets {
			source := typeSource[index]
			require.Equal(t, float64(index+1), target["type"])
			require.Equal(t, uuidFor("execution", source["id"].(float64)), target["uuid"])
			if parent := source["parent_dag_id"]; parent != nil {
				require.Equal(t, uuidFor("execution", parent.(float64)), target["parent_uuid"])
			} else {
				require.Nil(t, target["parent_uuid"])
			}
		}
	})

	t.Run("orphan_and_missing_parent", func(t *testing.T) {
		orphans := readCase(t, "orphan_and_missing_parent")
		orphanExpected := orphans["expected"].(map[string]any)
		require.Empty(t, orphanExpected["tasks"])
		require.Empty(t, orphanExpected["artifact_tasks"])
		skips := objects(t, orphanExpected["skips"])
		require.Len(t, skips, 3)
		require.Contains(t, skips[0]["reason"], "missing parent")
		require.Contains(t, skips[1]["reason"], "no matching run_details")
		require.Contains(t, skips[2]["reason"], "dependent task")
	})

	t.Run("metric_splits", func(t *testing.T) {
		metrics := readCase(t, "metric_splits")
		metricExpected := metrics["expected"].(map[string]any)
		artifacts := objects(t, metricExpected["artifacts"])
		require.Len(t, artifacts, 2)
		require.Equal(t, uuidFor("artifact", 30), artifacts[0]["uuid"])
		require.Equal(t, uuid.NewSHA1(uuid.NameSpaceDNS,
			[]byte("artifact:30:metric:loss")).String(), artifacts[1]["uuid"])
		links := objects(t, metricExpected["artifact_tasks"])
		require.Len(t, links, 1)
		require.Equal(t, artifacts[0]["uuid"], links[0]["artifact_id"])
		require.Equal(t, uuidFor("event", 40), links[0]["uuid"])
	})

	t.Run("missing_optional_and_source_error", func(t *testing.T) {
		missing := readCase(t, "missing_optional_and_source_error")
		missingExpected := missing["expected"].(map[string]any)
		retained := objects(t, missingExpected["tasks"])
		require.Len(t, retained, 1)
		require.Equal(t, uuidFor("execution", 300), retained[0]["uuid"])
		require.Empty(t, retained[0]["fingerprint"])
		unknownSkips := objects(t, missingExpected["skips"])
		require.Len(t, unknownSkips, 1)
		require.Equal(t, "execution:301", unknownSkips[0]["source"])
		diagnostics := objects(t, missingExpected["diagnostics"])
		require.Len(t, diagnostics, 1)
		require.Equal(t, "warning", diagnostics[0]["severity"])
	})
}
