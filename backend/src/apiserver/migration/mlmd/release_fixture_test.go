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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const releaseFixtureDir = "testdata/release_218"

func releaseFixtureFile(t *testing.T, name string) []byte {
	t.Helper()
	contents, err := os.ReadFile(filepath.Join(releaseFixtureDir, name))
	require.NoError(t, err)
	return contents
}

func TestReleasedSourceFixtureInventory(t *testing.T) {
	entries, err := os.ReadDir(releaseFixtureDir)
	require.NoError(t, err)
	actualFiles := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			actualFiles = append(actualFiles, entry.Name())
		}
	}
	require.ElementsMatch(t, []string{
		"README.md", "api_runs.json", "cases.json", "coverage.tsv",
		"deployment-images.json", "install-config.json", "migration_fixture.json",
		"produce_native_fingerprint.txt", "provenance.json", "runs.json", "source.sql", "table-counts.tsv",
		"workflows.json",
	}, actualFiles)

	source := string(releaseFixtureFile(t, "source.sql"))
	for table, want := range map[string]int{
		"run_details":     7,
		"tasks":           191,
		"Context":         10,
		"ContextProperty": 28,
		"Execution":       34,
		"Artifact":        22,
		"Event":           26,
	} {
		require.Equalf(t, want, strings.Count(source, "INSERT INTO `"+table+"` VALUES"), "table %s", table)
	}
	require.Contains(t, source, strings.Repeat("L", 512))
	require.Contains(t, source, "accuracy")
	require.Contains(t, source, "loss")

	coverage := string(releaseFixtureFile(t, "coverage.tsv"))
	require.Contains(t, coverage, "task-with-mlmd-id\t12\n")
	require.Contains(t, coverage, "task-with-fingerprint\t12\n")
	require.Contains(t, coverage, "skipped-task\t6\n")
	require.Contains(t, coverage, "execution-id-range\t1\t1003\t34\n")
	require.Contains(t, coverage, "artifact-id-range\t1\t1003\t22\n")
	require.Contains(t, coverage, "event-id-range\t1\t1003\t26\n")
	require.Contains(t, coverage, "metric-artifacts\t4\n")
}

func TestReleasedWriterHandoffEvidence(t *testing.T) {
	kfp := readMigrationFixture(t).Source["kfp"]
	reports := map[string]int{}
	reportNames := map[string]map[string]int{}
	for _, task := range kfp["tasks"] {
		if task["PodName"] != "" {
			runID := task["RunUUID"].(string)
			reports[runID]++
			if reportNames[runID] == nil {
				reportNames[runID] = map[string]int{}
			}
			reportNames[runID][task["Name"].(string)]++
		}
	}
	type workflowNode struct {
		DisplayName, Phase, StartedAt, FinishedAt string
		Children                                  []string
	}
	type workflowSnapshot struct {
		Metadata struct {
			Labels map[string]string `json:"labels"`
		} `json:"metadata"`
		Status struct {
			Phase, FinishedAt string
			Nodes             map[string]workflowNode `json:"nodes"`
		} `json:"status"`
	}
	var workflows struct {
		Items []workflowSnapshot `json:"items"`
	}
	require.NoError(t, json.Unmarshal(releaseFixtureFile(t, "workflows.json"), &workflows))
	require.Len(t, workflows.Items, len(kfp["run_details"]))
	manifests := map[string]string{}
	for _, run := range kfp["run_details"] {
		manifests[run["UUID"].(string)] = run["WorkflowRuntimeManifest"].(string)
	}
	seen := map[string]bool{}
	phases := map[string]string{}
	for _, workflow := range workflows.Items {
		runID := workflow.Metadata.Labels["pipeline/runid"]
		require.NotEmpty(t, runID)
		require.False(t, seen[runID])
		seen[runID] = true
		phases[runID] = workflow.Status.Phase
		require.NotEmpty(t, workflow.Status.FinishedAt)
		require.NotEmpty(t, manifests[runID], "missing stored Workflow manifest for run %s", runID)
		var saved workflowSnapshot
		require.NoError(t, json.Unmarshal([]byte(manifests[runID]), &saved), runID)
		require.Equal(t, runID, saved.Metadata.Labels["pipeline/runid"])
		require.Equal(t, workflow.Status.Phase, saved.Status.Phase, runID)
		require.Equal(t, workflow.Status.Nodes, saved.Status.Nodes, runID)
		require.Equal(t, len(workflow.Status.Nodes), reports[runID], "every Workflow node has a persisted report row")
		workflowNames := map[string]int{}
		for _, node := range workflow.Status.Nodes {
			workflowNames[node.DisplayName]++
		}
		require.Equal(t, workflowNames, reportNames[runID], "report names must match Workflow nodes, including repeated loop names")
	}
	for _, run := range kfp["run_details"] {
		runID := run["UUID"].(string)
		require.True(t, seen[runID])
		require.Contains(t, []string{"SUCCEEDED", "FAILED"}, run["State"])
		if run["State"] == "SUCCEEDED" {
			require.Equal(t, "Succeeded", phases[runID])
		} else {
			require.Equal(t, "Failed", phases[runID])
		}
	}
}

func TestReleasedSourceFixtureTerminalRuns(t *testing.T) {
	var runs []struct {
		Case   string `json:"case"`
		RunID  string `json:"run_id"`
		Status string `json:"status"`
	}
	require.NoError(t, json.Unmarshal(releaseFixtureFile(t, "runs.json"), &runs))
	require.Len(t, runs, 7)

	var apiRuns map[string]struct {
		RunID      string `json:"run_id"`
		State      string `json:"state"`
		RunDetails struct {
			TaskDetails []struct {
				TaskID string `json:"task_id"`
			} `json:"task_details"`
		} `json:"run_details"`
	}
	require.NoError(t, json.Unmarshal(releaseFixtureFile(t, "api_runs.json"), &apiRuns))
	require.Len(t, apiRuns, len(runs))

	sourceTaskIDs := map[string][]string{}
	sourceTasks := readMigrationFixture(t).Source["kfp"]["tasks"]
	require.Len(t, sourceTasks, 191)
	for _, task := range sourceTasks {
		runID := task["RunUUID"].(string)
		sourceTaskIDs[runID] = append(sourceTaskIDs[runID], task["UUID"].(string))
	}
	seenIDs := map[string]bool{}
	for _, run := range runs {
		require.NotEmpty(t, run.RunID)
		require.False(t, seenIDs[run.RunID], "duplicate run ID %s", run.RunID)
		seenIDs[run.RunID] = true
		require.Equal(t, run.RunID, apiRuns[run.Case].RunID)
		require.Equal(t, run.Status, apiRuns[run.Case].State)
		var apiTaskIDs []string
		for _, task := range apiRuns[run.Case].RunDetails.TaskDetails {
			apiTaskIDs = append(apiTaskIDs, task.TaskID)
		}
		require.ElementsMatch(t, sourceTaskIDs[run.RunID], apiTaskIDs, run.Case)
		delete(sourceTaskIDs, run.RunID)
		if run.Case == "intentional-failure" {
			require.Equal(t, "FAILED", run.Status)
		} else {
			require.Equal(t, "SUCCEEDED", run.Status)
		}
	}
	require.Empty(t, sourceTaskIDs)

	var workflows struct {
		Items []struct {
			Status struct {
				Phase string `json:"phase"`
			} `json:"status"`
		} `json:"items"`
	}
	require.NoError(t, json.Unmarshal(releaseFixtureFile(t, "workflows.json"), &workflows))
	require.Len(t, workflows.Items, len(runs))
	counts := map[string]int{}
	for _, workflow := range workflows.Items {
		counts[workflow.Status.Phase]++
	}
	require.Equal(t, map[string]int{"Succeeded": 6, "Failed": 1}, counts)

	require.Contains(t, string(releaseFixtureFile(t, "deployment-images.json")), "kfp-api-server:2.18.0")
	require.Contains(t, string(releaseFixtureFile(t, "workflows.json")), "kfp-driver:2.18.0")
	require.Contains(t, string(releaseFixtureFile(t, "workflows.json")), "kfp-launcher:2.18.0")
}
