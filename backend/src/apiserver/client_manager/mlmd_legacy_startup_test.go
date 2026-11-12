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

package clientmanager

import (
	"context"
	"database/sql"
	"encoding/json"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/stretchr/testify/require"
)

type legacySchemaFixture struct {
	Commit          string       `json:"source_commit"`
	SourceVersion   string       `json:"source_version"`
	DatabaseVersion string       `json:"database_version"`
	Driver          string       `json:"driver"`
	Statements      []string     `json:"statements"`
	Columns         [][5]string  `json:"columns"`
	SeedTasks       []legacyTask `json:"seed_tasks"`
}

type legacyTask struct {
	UUID              string `json:"UUID"`
	RunID             string `json:"RunID"`
	MLMDExecutionID   string `json:"MLMDExecutionID"`
	Name              string `json:"Name"`
	PodName           string `json:"PodName"`
	State             string `json:"State"`
	CreatedTimestamp  int64  `json:"CreatedTimestamp"`
	StartedTimestamp  int64  `json:"StartedTimestamp"`
	FinishedTimestamp int64  `json:"FinishedTimestamp"`
	MLMDInputs        string `json:"MLMDInputs"`
	Payload           string `json:"Payload"`
}

type legacyPartialCatalog struct {
	SourceCommit string      `json:"source_commit"`
	Driver       string      `json:"driver"`
	Columns      [][3]string `json:"columns"`
}

func inspectLegacyPartialCatalog(t *testing.T, db *sql.DB) legacyPartialCatalog {
	t.Helper()
	catalog := legacyPartialCatalog{SourceCommit: "a0733cb51b0484a6050cafee989ffe3312f808ad", Driver: "mysql"}
	query := `SELECT TABLE_NAME, COLUMN_NAME, COLUMN_TYPE FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME IN ('tasks', 'artifacts', 'artifact_tasks')
		ORDER BY TABLE_NAME, ORDINAL_POSITION`
	rows, err := db.Query(query)
	require.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		var column [3]string
		require.NoError(t, rows.Scan(&column[0], &column[1], &column[2]))
		catalog.Columns = append(catalog.Columns, column)
	}
	require.NoError(t, rows.Err())
	return catalog
}

func readLegacyPartialCatalog(t *testing.T) legacyPartialCatalog {
	t.Helper()
	data, err := os.ReadFile("../migration/mlmd/testdata/legacy_partial_mysql.json")
	require.NoError(t, err)
	var catalog legacyPartialCatalog
	require.NoError(t, json.Unmarshal(data, &catalog))
	require.Equal(t, "a0733cb51b0484a6050cafee989ffe3312f808ad", catalog.SourceCommit)
	require.Equal(t, "mysql", catalog.Driver)
	require.NotEmpty(t, catalog.Columns)
	return catalog
}

func readLegacySchema(t *testing.T) legacySchemaFixture {
	t.Helper()
	directory := os.Getenv("KFP_MLMD_SCHEMA_FIXTURE_DIR")
	if directory == "" {
		directory = "../migration/mlmd/testdata"
	}
	data, err := os.ReadFile(filepath.Join(directory, "legacy_schema_mysql.json"))
	require.NoError(t, err)
	var fixture legacySchemaFixture
	require.NoError(t, json.Unmarshal(data, &fixture))
	require.Equal(t, "a0733cb51b0484a6050cafee989ffe3312f808ad", fixture.Commit)
	require.Equal(t, "2.17.3", fixture.SourceVersion)
	require.Equal(t, "mysql", fixture.Driver)
	return fixture
}

func TestMLMDLegacyCatalog(t *testing.T) {
	data, err := os.ReadFile("../migration/mlmd/testdata/legacy_source.json")
	require.NoError(t, err)
	var source struct {
		LegacyTask   legacyTask       `json:"legacy_task"`
		ExecutionIDs map[string]int64 `json:"execution_ids"`
	}
	require.NoError(t, json.Unmarshal(data, &source))
	require.Equal(t, strconv.FormatInt(source.ExecutionIDs["consumer"], 10), source.LegacyTask.MLMDExecutionID)
	fixture := readLegacySchema(t)
	require.True(t, strings.HasPrefix(fixture.DatabaseVersion, "8.0."))
	require.NotEmpty(t, fixture.Statements)
	columns := map[string][5]string{}
	for _, column := range fixture.Columns {
		if column[0] == "tasks" {
			columns[column[1]] = column
		}
	}
	for _, name := range []string{"MLMDExecutionID", "PodName", "CreatedTimestamp", "StartedTimestamp", "FinishedTimestamp", "MLMDInputs", "MLMDOutputs", "Payload"} {
		require.Contains(t, columns, name)
	}
	require.NotContains(t, columns, "pods")
	require.NotContains(t, columns, "TypeAttrs")
	require.Contains(t, columns, "State")
	require.NotContains(t, columns["State"][2], "int", "legacy state is a string, native state is an integer enum")
	require.Contains(t, strings.Join(fixture.Statements, "\n"), "tasks_RunUUID_run_details_UUID_foreign")
	require.Len(t, fixture.SeedTasks, 2)
	require.Equal(t, "7", fixture.SeedTasks[0].MLMDExecutionID)
	require.Equal(t, source.LegacyTask.UUID, fixture.SeedTasks[0].UUID)
	require.Equal(t, source.LegacyTask.RunID, fixture.SeedTasks[0].RunID)
	require.Equal(t, source.LegacyTask.Name, fixture.SeedTasks[0].Name)
	require.Equal(t, source.LegacyTask.PodName, fixture.SeedTasks[0].PodName)
	require.Equal(t, source.LegacyTask.State, fixture.SeedTasks[0].State)
	require.Equal(t, source.LegacyTask.CreatedTimestamp, fixture.SeedTasks[0].CreatedTimestamp)
	require.Equal(t, source.LegacyTask.StartedTimestamp, fixture.SeedTasks[0].StartedTimestamp)
	require.Equal(t, source.LegacyTask.FinishedTimestamp, fixture.SeedTasks[0].FinishedTimestamp)
	require.Equal(t, "13", fixture.SeedTasks[1].MLMDExecutionID)
	require.Len(t, fixture.SeedTasks[1].Name, 240)
	require.Len(t, fixture.SeedTasks[1].PodName, 240)
	require.Len(t, fixture.SeedTasks[1].MLMDInputs, 1024)
	require.Len(t, fixture.SeedTasks[1].Payload, 1024)
}

func TestMLMDLegacyPartialCatalog(t *testing.T) {
	baseline := readLegacySchema(t)
	partial := readLegacyPartialCatalog(t)
	baselineColumns := map[string]string{}
	for _, column := range baseline.Columns {
		if column[0] == "tasks" || column[0] == "artifacts" || column[0] == "artifact_tasks" {
			baselineColumns[column[0]+"/"+column[1]] = column[2]
		}
	}
	partialColumns := map[string]string{}
	for _, column := range partial.Columns {
		partialColumns[column[0]+"/"+column[1]] = column[2]
	}
	require.Contains(t, partialColumns, "tasks/UUID")
	require.NotEqual(t, baselineColumns, partialColumns, "capture a distinct interrupted startup catalog")
	require.Equal(t, "varchar(63)", partialColumns["tasks/Namespace"])
	require.Equal(t, "json", partialColumns["tasks/pods"])
}

// The database name is generated by the test; only that database is mutated or
// dropped. Test DSNs must point to disposable loopback services, including CI.
func legacyStartupDatabase(t *testing.T, dsn string) (*sql.DB, map[string]any) {
	t.Helper()
	name := "kfp_mlmd_startup_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	settings := map[string]any{"DBDriverName": "mysql", common.DBCredentialProviderEnabled: false}
	config, err := mysqlDriver.ParseDSN(dsn)
	require.NoError(t, err)
	require.Equal(t, "tcp", config.Net)
	host, port, err := net.SplitHostPort(config.Addr)
	require.NoError(t, err)
	settings[mysqlServiceHost], settings[mysqlServicePort] = host, port
	settings[mysqlUser], settings[mysqlPassword], settings[mysqlDBName] = config.User, config.Passwd, name
	config.DBName = name
	config.MultiStatements = true // The released mysqldump is replayed as one script.
	destination := config.FormatDSN()
	address := net.ParseIP(host)
	require.True(t, host == "localhost" || address != nil && address.IsLoopback(), "use disposable loopback database services")
	root, err := sql.Open("mysql", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, root.Close()) })
	_, err = root.Exec("CREATE DATABASE " + name)
	require.NoError(t, err)
	t.Cleanup(func() { _, err := root.Exec("DROP DATABASE " + name); require.NoError(t, err) })
	db, err := sql.Open("mysql", destination)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	return db, settings
}

func TestMLMDLegacyStartup(t *testing.T) {
	// InitDBClient terminates the process on errors. Use this test binary as an
	// isolated child so the test exercises that real entrypoint without changing it.
	if config := os.Getenv("KFP_MLMD_STARTUP_CHILD_CONFIG"); config != "" {
		var values map[string]any
		require.NoError(t, json.Unmarshal([]byte(config), &values))
		setConfig(t, values)
		db, _, _ := InitDBClient(10 * time.Second)
		require.NoError(t, db.Close())
		return
	}
	dsn := os.Getenv("KFP_MLMD_MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("set KFP_MLMD_MYSQL_TEST_DSN to test the actual database startup entrypoint")
	}
	db, settings := legacyStartupDatabase(t, dsn)
	fixture := readLegacySchema(t)
	for _, query := range fixture.Statements {
		_, err := db.Exec(query)
		require.NoError(t, err, "replaying pinned legacy schema/seed")
	}
	config, err := json.Marshal(settings)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestMLMDLegacyStartup$", "-test.v")
	cmd.Env = append(os.Environ(), "KFP_MLMD_STARTUP_CHILD_CONFIG="+string(config), "GOTRACEBACK=none")
	output, err := cmd.CombinedOutput()
	require.NoError(t, ctx.Err(), "startup must fail promptly, not time out")
	require.Error(t, err, "legacy startup must refuse migration without MLMD connection settings")
	require.Contains(t, string(output), "requires METADATA_GRPC_SERVICE_HOST and METADATA_GRPC_SERVICE_PORT", string(output))
	var executionID, state string
	require.NoError(t, db.QueryRow("SELECT `MLMDExecutionID`, `State` FROM `tasks` WHERE `UUID` = ?", "33333333-3333-4333-8333-333333333333").Scan(&executionID, &state))
	require.Equal(t, "7", executionID)
	require.Equal(t, "SUCCEEDED", state)
	var name, podName, inputs, payload string
	require.NoError(t, db.QueryRow("SELECT `Name`, `PodName`, `MLMDInputs`, `Payload` FROM `tasks` WHERE `UUID` = ?", "44444444-4444-4444-8444-444444444444").Scan(&name, &podName, &inputs, &payload))
	require.Equal(t, fixture.SeedTasks[1].Name, name)
	require.Equal(t, fixture.SeedTasks[1].PodName, podName)
	require.Equal(t, fixture.SeedTasks[1].MLMDInputs, inputs)
	require.Equal(t, fixture.SeedTasks[1].Payload, payload)
	actual := inspectLegacyPartialCatalog(t, db)
	var expected [][3]string
	for _, column := range fixture.Columns {
		if column[0] == "tasks" || column[0] == "artifacts" || column[0] == "artifact_tasks" {
			expected = append(expected, [3]string{column[0], column[1], column[2]})
		}
	}
	require.Equal(t, expected, actual.Columns, "missing MLMD settings must fail before any schema change")
}

func TestMLMDReleasedStartup(t *testing.T) {
	dsn := os.Getenv("KFP_MLMD_MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("set KFP_MLMD_MYSQL_TEST_DSN to replay the released source in a disposable MySQL database")
	}
	db, settings := legacyStartupDatabase(t, dsn)
	dump, err := os.ReadFile("../migration/mlmd/testdata/release_218/source.sql")
	require.NoError(t, err)
	prelude, remainder, found := strings.Cut(string(dump), "CREATE DATABASE /*!32312 IF NOT EXISTS*/ `mlpipeline`")
	require.True(t, found, "released dump must contain the KFP schema")
	_, source, found := strings.Cut(remainder, "USE `mlpipeline`;")
	require.True(t, found)
	source, _, found = strings.Cut(source, "CREATE DATABASE /*!32312 IF NOT EXISTS*/ `metadb`")
	require.True(t, found, "released dump must contain the MLMD schema after the KFP schema")
	_, err = db.Exec(prelude + source)
	require.NoError(t, err, "replaying the released KFP schema and rows")
	var runtimeManifest string
	const conditionalRun = "1dd61cdf-fd5f-471c-81a7-a14532a31fdd"
	require.NoError(t, db.QueryRow("SELECT `WorkflowRuntimeManifest` FROM `run_details` WHERE `UUID` = ?", conditionalRun).Scan(&runtimeManifest))
	var runtimeWorkflow struct {
		Metadata struct {
			Labels map[string]string `json:"labels"`
		} `json:"metadata"`
		Status struct {
			Phase string `json:"phase"`
			Nodes map[string]struct {
				DisplayName string `json:"displayName"`
				Phase       string `json:"phase"`
				Message     string `json:"message"`
				Type        string `json:"type"`
			} `json:"nodes"`
		} `json:"status"`
	}
	require.NoError(t, json.Unmarshal([]byte(runtimeManifest), &runtimeWorkflow))
	require.Equal(t, conditionalRun, runtimeWorkflow.Metadata.Labels["pipeline/runid"])
	require.Equal(t, "Failed", runtimeWorkflow.Status.Phase)
	var skippedNodes int
	for _, node := range runtimeWorkflow.Status.Nodes {
		if node.DisplayName == "condition-3" && node.Type == "Skipped" {
			require.Equal(t, "Skipped", node.Phase)
			require.Contains(t, node.Message, "evaluated false")
			skippedNodes++
		}
	}
	require.Equal(t, 1, skippedNodes, "saved Argo status must show the conditional node was not triggered")
	metadb, metaSettings := legacyStartupDatabase(t, dsn)
	_, metadata, found := strings.Cut(string(dump), "USE `metadb`;")
	require.True(t, found)
	metadata, _, found = strings.Cut(metadata, "/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;")
	require.True(t, found)
	_, err = metadb.Exec(prelude + metadata)
	require.NoError(t, err, "replaying the matching released MLMD records")
	var canceledState, kfpSkippedRows, runAssociations int
	var canceledTaskName string
	require.NoError(t, metadb.QueryRow("SELECT e.last_known_state, p.string_value FROM `Execution` e JOIN `ExecutionProperty` p ON p.execution_id = e.id AND p.name = 'task_name' WHERE e.id = 7").Scan(&canceledState, &canceledTaskName))
	require.Equal(t, 6, canceledState) // MLMD CANCELED.
	require.Equal(t, "condition-3", canceledTaskName)
	require.NoError(t, metadb.QueryRow("SELECT COUNT(*) FROM `Association` a JOIN `Context` c ON c.id = a.context_id WHERE a.execution_id = 7 AND c.name = ?", conditionalRun).Scan(&runAssociations))
	require.Equal(t, 1, runAssociations)
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM `tasks` WHERE `RunUUID` = ? AND `Name` = ? AND `State` = 'SKIPPED' AND `MLMDExecutionID` = ''", conditionalRun, canceledTaskName).Scan(&kfpSkippedRows))
	require.Equal(t, 1, kfpSkippedRows, "the same run has a skipped KFP row without a direct MLMD execution join")

	var graph struct {
		Tasks []struct {
			ID                string `json:"id"`
			RunID             string `json:"run_id"`
			ExecutionID       string `json:"execution_id"`
			Name              string `json:"name"`
			PodName           string `json:"pod_name"`
			State             string `json:"state"`
			CreatedTimestamp  int64  `json:"created_timestamp"`
			StartedTimestamp  int64  `json:"started_timestamp"`
			FinishedTimestamp int64  `json:"finished_timestamp"`
		} `json:"tasks"`
	}
	data, err := os.ReadFile("../migration/mlmd/testdata/release_218/source_graph.json")
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, &graph))
	var selected legacyTask
	for _, task := range graph.Tasks {
		if task.ExecutionID == "19" {
			selected = legacyTask{UUID: task.ID, RunID: task.RunID, MLMDExecutionID: task.ExecutionID, Name: task.Name, PodName: task.PodName, State: task.State, CreatedTimestamp: task.CreatedTimestamp, StartedTimestamp: task.StartedTimestamp, FinishedTimestamp: task.FinishedTimestamp}
			break
		}
	}
	require.NotEmpty(t, selected.UUID)
	type releasedNativeTask struct {
		SourceExecutionID int   `json:"source_execution_id"`
		CreatedAtInSec    int64 `json:"CreatedAtInSec"`
		StartedInSec      int64 `json:"StartedInSec"`
		FinishedInSec     int64 `json:"FinishedInSec"`
		State             int   `json:"State"`
		Pods              []struct {
			Name string `json:"name"`
		} `json:"Pods"`
	}
	var expected struct {
		Tasks []releasedNativeTask `json:"tasks"`
	}
	data, err = os.ReadFile("../migration/mlmd/testdata/release_218/expected_native.json")
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, &expected))
	var projected *releasedNativeTask
	for i := range expected.Tasks {
		if expected.Tasks[i].SourceExecutionID == 19 {
			projected = &expected.Tasks[i]
			break
		}
	}
	require.NotNil(t, projected)
	require.Len(t, projected.Pods, 1)
	var sourceCreated, sourceStarted, sourceFinished, mlmdCreated int64
	var sourcePod, sourceState, mlmdPod string
	var mlmdState int
	join := "SELECT t.CreatedTimestamp, t.StartedTimestamp, t.FinishedTimestamp, t.PodName, t.State, e.create_time_since_epoch, e.last_known_state, p.string_value FROM tasks t " +
		"JOIN `" + metaSettings[mysqlDBName].(string) + "`.`Execution` e ON e.id = CAST(t.MLMDExecutionID AS UNSIGNED) " +
		"JOIN `" + metaSettings[mysqlDBName].(string) + "`.`ExecutionProperty` p ON p.execution_id = e.id AND p.name = 'pod_name' WHERE t.UUID = ?"
	require.NoError(t, db.QueryRow(join, selected.UUID).Scan(&sourceCreated, &sourceStarted, &sourceFinished, &sourcePod, &sourceState, &mlmdCreated, &mlmdState, &mlmdPod))
	require.NotEqual(t, sourceCreated, mlmdCreated/1000, "the KFP and MLMD creation times must disagree")
	require.Empty(t, sourcePod)
	require.Equal(t, "RUNTIME_STATE_UNSPECIFIED", sourceState)
	require.Equal(t, 3, mlmdState) // MLMD COMPLETE becomes native SUCCEEDED.
	require.NotEmpty(t, mlmdPod)
	require.Equal(t, sourceCreated, projected.CreatedAtInSec)
	require.Equal(t, sourceStarted, projected.StartedInSec)
	require.Equal(t, sourceFinished, projected.FinishedInSec)
	require.Equal(t, mlmdPod, projected.Pods[0].Name)
	require.Equal(t, 2, projected.State)
	readTask := func() legacyTask {
		var task legacyTask
		err := db.QueryRow("SELECT `UUID`, `RunUUID`, `MLMDExecutionID`, COALESCE(`Name`, ''), COALESCE(`PodName`, ''), COALESCE(`State`, ''), `CreatedTimestamp`, `StartedTimestamp`, `FinishedTimestamp` FROM `tasks` WHERE `UUID` = ?", selected.UUID).
			Scan(&task.UUID, &task.RunID, &task.MLMDExecutionID, &task.Name, &task.PodName, &task.State, &task.CreatedTimestamp, &task.StartedTimestamp, &task.FinishedTimestamp)
		require.NoError(t, err)
		return task
	}
	require.Equal(t, selected, readTask(), "startup must use the same released row as the merge and API expectations")
	config, err := json.Marshal(settings)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestMLMDLegacyStartup$", "-test.v")
	cmd.Env = append(os.Environ(), "KFP_MLMD_STARTUP_CHILD_CONFIG="+string(config), "GOTRACEBACK=none")
	output, err := cmd.CombinedOutput()
	require.NoError(t, ctx.Err(), "startup must fail promptly, not time out")
	require.Error(t, err, "legacy startup must refuse migration without MLMD connection settings")
	require.Contains(t, string(output), "requires METADATA_GRPC_SERVICE_HOST and METADATA_GRPC_SERVICE_PORT")
	require.Equal(t, selected, readTask(), "failed startup must preserve the released source row")
}
