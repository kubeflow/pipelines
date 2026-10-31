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
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
	UUID            string `json:"UUID"`
	MLMDExecutionID string `json:"MLMDExecutionID"`
	Name            string `json:"Name"`
	PodName         string `json:"PodName"`
	MLMDInputs      string `json:"MLMDInputs"`
	Payload         string `json:"Payload"`
}

type legacyPartialCatalog struct {
	SourceCommit string      `json:"source_commit"`
	Driver       string      `json:"driver"`
	Columns      [][3]string `json:"columns"`
}

func inspectLegacyPartialCatalog(t *testing.T, db *sql.DB, driver string) legacyPartialCatalog {
	t.Helper()
	catalog := legacyPartialCatalog{SourceCommit: "a0733cb51b0484a6050cafee989ffe3312f808ad", Driver: driver}
	query := `SELECT TABLE_NAME, COLUMN_NAME, COLUMN_TYPE FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME IN ('tasks', 'artifacts', 'artifact_tasks')
		ORDER BY TABLE_NAME, ORDINAL_POSITION`
	if driver == "pgx" {
		query = `SELECT c.relname, a.attname, format_type(a.atttypid,a.atttypmod)
			FROM pg_attribute a JOIN pg_class c ON c.oid=a.attrelid
			JOIN pg_namespace n ON n.oid=c.relnamespace
			WHERE n.nspname='public' AND c.relname IN ('tasks', 'artifacts', 'artifact_tasks')
			AND c.relkind='r' AND a.attnum>0 AND NOT a.attisdropped
			ORDER BY c.relname,a.attnum`
	}
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

func readLegacyPartialCatalog(t *testing.T, driver string) legacyPartialCatalog {
	t.Helper()
	data, err := os.ReadFile("../migration/mlmd/testdata/legacy_partial_" + driver + ".json")
	require.NoError(t, err)
	var catalog legacyPartialCatalog
	require.NoError(t, json.Unmarshal(data, &catalog))
	require.Equal(t, "a0733cb51b0484a6050cafee989ffe3312f808ad", catalog.SourceCommit)
	require.Equal(t, driver, catalog.Driver)
	require.NotEmpty(t, catalog.Columns)
	return catalog
}

func readLegacySchema(t *testing.T, driver string) legacySchemaFixture {
	t.Helper()
	directory := os.Getenv("KFP_MLMD_SCHEMA_FIXTURE_DIR")
	if directory == "" {
		directory = "../migration/mlmd/testdata"
	}
	data, err := os.ReadFile(filepath.Join(directory, "legacy_schema_"+driver+".json"))
	require.NoError(t, err)
	var fixture legacySchemaFixture
	require.NoError(t, json.Unmarshal(data, &fixture))
	require.Equal(t, "a0733cb51b0484a6050cafee989ffe3312f808ad", fixture.Commit)
	require.Equal(t, "2.17.3", fixture.SourceVersion)
	require.Equal(t, driver, fixture.Driver)
	return fixture
}

func TestMLMDLegacyCatalog(t *testing.T) {
	for _, driver := range []string{"mysql", "pgx"} {
		t.Run(driver, func(t *testing.T) {
			fixture := readLegacySchema(t, driver)
			if driver == "mysql" {
				require.True(t, strings.HasPrefix(fixture.DatabaseVersion, "8.0."))
			} else {
				require.True(t, strings.HasPrefix(fixture.DatabaseVersion, "PostgreSQL 16."))
			}
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
			require.Equal(t, "13", fixture.SeedTasks[1].MLMDExecutionID)
			require.Len(t, fixture.SeedTasks[1].Name, 240)
			require.Len(t, fixture.SeedTasks[1].PodName, 240)
			require.Len(t, fixture.SeedTasks[1].MLMDInputs, 1024)
			require.Len(t, fixture.SeedTasks[1].Payload, 1024)
		})
	}
}

func TestMLMDLegacyPartialCatalog(t *testing.T) {
	for _, driver := range []string{"mysql", "pgx"} {
		t.Run(driver, func(t *testing.T) {
			baseline := readLegacySchema(t, driver)
			partial := readLegacyPartialCatalog(t, driver)
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
			require.Equal(t, map[string]string{"mysql": "varchar(63)", "pgx": "character varying(63)"}[driver], partialColumns["tasks/Namespace"])
			if driver == "mysql" {
				require.Equal(t, "json", partialColumns["tasks/pods"])
			} else {
				require.NotContains(t, partialColumns, "tasks/pods")
			}
		})
	}
}

// The database name is generated by the test; only that database is mutated or
// dropped. Test DSNs must point to disposable loopback services, including CI.
func legacyStartupDatabase(t *testing.T, driver, dsn string) (*sql.DB, map[string]any) {
	t.Helper()
	name := "kfp_mlmd_startup_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	settings := map[string]any{"DBDriverName": driver, common.DBCredentialProviderEnabled: false}
	var destination, host string
	if driver == "mysql" {
		config, err := mysqlDriver.ParseDSN(dsn)
		require.NoError(t, err)
		require.Equal(t, "tcp", config.Net)
		var port string
		host, port, err = net.SplitHostPort(config.Addr)
		require.NoError(t, err)
		settings[mysqlServiceHost], settings[mysqlServicePort] = host, port
		settings[mysqlUser], settings[mysqlPassword], settings[mysqlDBName] = config.User, config.Passwd, name
		config.DBName = name
		destination = config.FormatDSN()
	} else {
		config, err := pgx.ParseConfig(dsn)
		require.NoError(t, err)
		host = config.Host
		settings[postgresHost], settings[postgresPort] = host, strconv.Itoa(int(config.Port))
		settings[postgresUser], settings[postgresPassword], settings[postgresDBName] = config.User, config.Password, name
		settings[postgresExtraParams] = map[string]string{"sslmode": "disable"}
		childURL := &url.URL{Scheme: "postgres", Host: net.JoinHostPort(host, strconv.Itoa(int(config.Port))), User: url.UserPassword(config.User, config.Password), Path: "/" + name, RawQuery: "sslmode=disable"}
		destination = childURL.String()
	}
	address := net.ParseIP(host)
	require.True(t, host == "localhost" || address != nil && address.IsLoopback(), "use disposable loopback database services")
	root, err := sql.Open(driver, dsn)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, root.Close()) })
	_, err = root.Exec("CREATE DATABASE " + name)
	require.NoError(t, err)
	t.Cleanup(func() { _, err := root.Exec("DROP DATABASE " + name); require.NoError(t, err) })
	db, err := sql.Open(driver, destination)
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
	for _, engine := range []struct{ driver, env string }{
		{"mysql", "KFP_MLMD_MYSQL_TEST_DSN"},
		{"pgx", "KFP_MLMD_POSTGRES_TEST_DSN"},
	} {
		t.Run(engine.driver, func(t *testing.T) {
			dsn := os.Getenv(engine.env)
			if dsn == "" {
				t.Skip("set " + engine.env + " to test the actual database startup entrypoint")
			}
			db, settings := legacyStartupDatabase(t, engine.driver, dsn)
			fixture := readLegacySchema(t, engine.driver)
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
			require.Error(t, err, "baseline has no MLMD migration; replace this characterization with graph-preservation assertions when migration is wired into startup")
			require.Contains(t, string(output), "AutoMigrate failed", string(output))
			query := "SELECT `MLMDExecutionID`, `State` FROM `tasks` WHERE `UUID` = ?"
			if engine.driver == "mysql" {
				require.Contains(t, string(output), "Data truncated for column 'Name'")
			} else {
				require.Contains(t, string(output), `column "pods" of relation "tasks" contains null values`)
				query = `SELECT "MLMDExecutionID", "State" FROM "tasks" WHERE "UUID" = $1`
			}
			var executionID, state string
			require.NoError(t, db.QueryRow(query, "33333333-3333-4333-8333-333333333333").Scan(&executionID, &state))
			require.Equal(t, "7", executionID)
			require.Equal(t, "SUCCEEDED", state)
			var name, podName, inputs, payload string
			longQuery := "SELECT `Name`, `PodName`, `MLMDInputs`, `Payload` FROM `tasks` WHERE `UUID` = ?"
			if engine.driver == "pgx" {
				longQuery = `SELECT "Name", "PodName", "MLMDInputs", "Payload" FROM "tasks" WHERE "UUID" = $1`
			}
			require.NoError(t, db.QueryRow(longQuery, "44444444-4444-4444-8444-444444444444").Scan(&name, &podName, &inputs, &payload))
			require.Equal(t, fixture.SeedTasks[1].Name, name)
			require.Equal(t, fixture.SeedTasks[1].PodName, podName)
			require.Equal(t, fixture.SeedTasks[1].MLMDInputs, inputs)
			require.Equal(t, fixture.SeedTasks[1].Payload, payload)
			partial := inspectLegacyPartialCatalog(t, db, engine.driver)
			if directory := os.Getenv("KFP_MLMD_CAPTURE_PARTIAL_DIR"); directory != "" {
				data, err := json.MarshalIndent(partial, "", "  ")
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(filepath.Join(directory, "legacy_partial_"+engine.driver+".json"), append(data, '\n'), 0o600))
			} else {
				require.Equal(t, readLegacyPartialCatalog(t, engine.driver), partial)
			}
			// Report only the relevant startup diagnostic, not a subprocess stack.
			for _, line := range strings.Split(string(output), "\n") {
				if strings.Contains(line, "AutoMigrate failed") {
					t.Log(line)
				}
			}
		})
	}
}
