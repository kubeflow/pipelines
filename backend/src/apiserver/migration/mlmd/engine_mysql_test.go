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
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/kubeflow/pipelines/backend/src/apiserver/migration/mlmd"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type fixtureClient struct {
	pb.MetadataStoreServiceClient
	executions *pb.GetExecutionsResponse
	artifacts  *pb.GetArtifactsResponse
	events     *pb.GetEventsByExecutionIDsResponse
	contexts   map[int64]*pb.GetContextsByExecutionResponse
}

type flakyClient struct {
	fixtureClient
	failures int
}

type pagedClient struct{ fixtureClient }

func (client pagedClient) GetExecutions(_ context.Context, request *pb.GetExecutionsRequest, _ ...grpc.CallOption) (*pb.GetExecutionsResponse, error) {
	if request.GetOptions().GetNextPageToken() == "" {
		return &pb.GetExecutionsResponse{Executions: client.executions.Executions[1:2], NextPageToken: proto.String("next")}, nil
	}
	return &pb.GetExecutionsResponse{Executions: client.executions.Executions[0:1]}, nil
}

func (client *flakyClient) GetExecutions(ctx context.Context, request *pb.GetExecutionsRequest, options ...grpc.CallOption) (*pb.GetExecutionsResponse, error) {
	if client.failures > 0 {
		client.failures--
		return nil, status.Error(codes.Unavailable, "temporary MLMD disconnect")
	}
	return client.fixtureClient.GetExecutions(ctx, request, options...)
}

type fixtureServer struct {
	pb.UnimplementedMetadataStoreServiceServer
	fixtureClient
}

func (server fixtureServer) GetExecutionTypes(context.Context, *pb.GetExecutionTypesRequest) (*pb.GetExecutionTypesResponse, error) {
	return &pb.GetExecutionTypesResponse{}, nil
}

func (server fixtureServer) GetExecutions(ctx context.Context, request *pb.GetExecutionsRequest) (*pb.GetExecutionsResponse, error) {
	return server.fixtureClient.GetExecutions(ctx, request)
}
func (server fixtureServer) GetArtifacts(ctx context.Context, request *pb.GetArtifactsRequest) (*pb.GetArtifactsResponse, error) {
	return server.fixtureClient.GetArtifacts(ctx, request)
}
func (server fixtureServer) GetContextsByExecution(ctx context.Context, request *pb.GetContextsByExecutionRequest) (*pb.GetContextsByExecutionResponse, error) {
	return server.fixtureClient.GetContextsByExecution(ctx, request)
}
func (server fixtureServer) GetEventsByExecutionIDs(ctx context.Context, request *pb.GetEventsByExecutionIDsRequest) (*pb.GetEventsByExecutionIDsResponse, error) {
	return server.fixtureClient.GetEventsByExecutionIDs(ctx, request)
}

func (client fixtureClient) GetExecutions(context.Context, *pb.GetExecutionsRequest, ...grpc.CallOption) (*pb.GetExecutionsResponse, error) {
	return client.executions, nil
}
func (client fixtureClient) GetArtifacts(context.Context, *pb.GetArtifactsRequest, ...grpc.CallOption) (*pb.GetArtifactsResponse, error) {
	return client.artifacts, nil
}
func (client fixtureClient) GetContextsByExecution(_ context.Context, request *pb.GetContextsByExecutionRequest, _ ...grpc.CallOption) (*pb.GetContextsByExecutionResponse, error) {
	return client.contexts[request.GetExecutionId()], nil
}
func (client fixtureClient) GetEventsByExecutionIDs(context.Context, *pb.GetEventsByExecutionIDsRequest, ...grpc.CallOption) (*pb.GetEventsByExecutionIDsResponse, error) {
	return client.events, nil
}

func sourceClient(t *testing.T) fixtureClient {
	t.Helper()
	data, err := os.ReadFile("testdata/engine/legacy_source.json")
	require.NoError(t, err)
	var raw map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &raw))
	client := fixtureClient{executions: &pb.GetExecutionsResponse{}, artifacts: &pb.GetArtifactsResponse{}, events: &pb.GetEventsByExecutionIDsResponse{}, contexts: map[int64]*pb.GetContextsByExecutionResponse{}}
	require.NoError(t, protojson.Unmarshal(raw["executions"], client.executions))
	require.NoError(t, protojson.Unmarshal(raw["artifacts"], client.artifacts))
	require.NoError(t, protojson.Unmarshal(raw["events"], client.events))
	var contexts map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw["execution_contexts"], &contexts))
	for id, value := range contexts {
		parsed, err := strconv.ParseInt(id, 10, 64)
		require.NoError(t, err)
		response := &pb.GetContextsByExecutionResponse{}
		require.NoError(t, protojson.Unmarshal(value, response))
		client.contexts[parsed] = response
	}
	return client
}

func TestReaderFollowsPagesAndSortsSparseIDs(t *testing.T) {
	client := pagedClient{fixtureClient: sourceClient(t)}
	snapshot, err := mlmd.ReadSnapshot(context.Background(), client)
	require.NoError(t, err)
	require.Len(t, snapshot.Executions, 2)
	require.Less(t, snapshot.Executions[0].GetId(), snapshot.Executions[1].GetId())
}

var fixtureDiskBudget = mlmd.DiskBudget{AvailableBytes: 1 << 30, MLMDDatabaseBytes: 1 << 20}

func scratchMySQLDatabase(t *testing.T) (*sql.DB, *sql.DB, mysqlDriver.Config, string) {
	t.Helper()
	dsn := os.Getenv("KFP_MLMD_MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("set KFP_MLMD_MYSQL_TEST_DSN to a disposable loopback MySQL service")
	}
	config, err := mysqlDriver.ParseDSN(dsn)
	require.NoError(t, err)
	if config.Net == "tcp" {
		host, _, err := net.SplitHostPort(config.Addr)
		require.NoError(t, err)
		ip := net.ParseIP(host)
		require.True(t, host == "localhost" || ip != nil && ip.IsLoopback())
	} else {
		require.Equal(t, "unix", config.Net)
		require.True(t, strings.HasPrefix(config.Addr, "/tmp/"))
	}
	admin, err := sql.Open("mysql", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, admin.Close()) })
	name := "kfp_mlmd_migrate_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	_, err = admin.Exec("CREATE DATABASE `" + name + "`")
	require.NoError(t, err)
	t.Cleanup(func() { _, err := admin.Exec("DROP DATABASE `" + name + "`"); require.NoError(t, err) })
	config.DBName = name
	config.MultiStatements = true
	db, err := sql.Open("mysql", config.FormatDSN())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	return db, admin, *config, name
}

func releasedDatabase(t *testing.T) (*sql.DB, *sql.DB, mysqlDriver.Config, string) {
	db, admin, config, name := scratchMySQLDatabase(t)
	var schema struct {
		Statements []string `json:"statements"`
	}
	readFixture(t, "legacy_schema_mysql.json", &schema)
	for _, statement := range schema.Statements {
		_, err := db.Exec(statement)
		require.NoError(t, err, statement)
	}
	_, err := db.Exec("UPDATE run_details SET State='SUCCEEDED', FinishedAtInSec=1700000003")
	require.NoError(t, err, "the direct-writer fixture needs a terminal run before Phase 2/3")
	return db, admin, config, name
}

func TestMySQLReleasedSchemaMigration(t *testing.T) {
	db, admin, config, name := releasedDatabase(t)
	required, err := mlmd.NeedsMigration(context.Background(), db)
	require.NoError(t, err)
	require.True(t, required)
	client := sourceClient(t)
	report, err := mlmd.Migrate(context.Background(), client, db, fixtureDiskBudget)
	require.NoError(t, err)
	required, err = mlmd.NeedsMigration(context.Background(), db)
	require.NoError(t, err)
	require.False(t, required)
	_, err = db.Exec("RENAME TABLE artifact_tasks TO missing_artifact_tasks")
	require.NoError(t, err)
	_, err = mlmd.NeedsMigration(context.Background(), db)
	require.ErrorContains(t, err, "validation marker conflicts")
	_, err = mlmd.Migrate(context.Background(), client, db, fixtureDiskBudget)
	require.ErrorContains(t, err, "validation marker conflicts")
	_, err = db.Exec("RENAME TABLE missing_artifact_tasks TO artifact_tasks")
	require.NoError(t, err)
	_, err = db.Exec("DELETE FROM mlmd_migration_complete WHERE ID=1")
	require.NoError(t, err)
	_, err = mlmd.NeedsMigration(context.Background(), db)
	require.ErrorContains(t, err, "without a validation record or legacy task backup")
	_, err = mlmd.Migrate(context.Background(), client, db, fixtureDiskBudget)
	require.ErrorContains(t, err, "without a validation record or legacy task backup")
	_, err = db.Exec("INSERT INTO mlmd_migration_complete VALUES (1)")
	require.NoError(t, err)
	require.Equal(t, 12, report.Tasks)
	require.Equal(t, 4, report.Artifacts)
	require.Equal(t, 8, report.Links)
	var expected nativeFixture
	readFixture(t, "expected_native.json", &expected)
	actual := readNativeRows(t, db)
	require.ElementsMatch(t, expected.Tasks, actual.Tasks)
	require.ElementsMatch(t, expected.Artifacts, actual.Artifacts)
	require.ElementsMatch(t, expected.Links, actual.Links)
	var backupCount int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='tasks_pre_migration'").Scan(&backupCount))
	require.Zero(t, backupCount)
	// A later native startup must reconcile the migrated schema without MLMD.
	startup, err := gorm.Open(mysql.Open(config.FormatDSN()), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, startup.AutoMigrate(model.AllModels()...))
	repeated, err := mlmd.Migrate(context.Background(), nil, db, fixtureDiskBudget)
	require.NoError(t, err)
	require.Equal(t, report.Tasks, repeated.Tasks)
	require.Equal(t, report.Artifacts, repeated.Artifacts)
	require.Equal(t, report.Links, repeated.Links)
	owner, err := db.Conn(context.Background())
	require.NoError(t, err)
	var acquired int
	require.NoError(t, owner.QueryRowContext(context.Background(), "SELECT GET_LOCK('kfp_mlmd_migration_lock', 0)").Scan(&acquired))
	require.Equal(t, 1, acquired)
	_, err = mlmd.Migrate(context.Background(), client, db, fixtureDiskBudget)
	require.ErrorContains(t, err, "already in progress")
	require.NoError(t, owner.QueryRowContext(context.Background(), "SELECT RELEASE_LOCK('kfp_mlmd_migration_lock')").Scan(&acquired))
	require.NoError(t, owner.Close())
	nativeName := name + "_native"
	_, err = admin.Exec("CREATE DATABASE `" + nativeName + "`")
	require.NoError(t, err)
	t.Cleanup(func() { _, err := admin.Exec("DROP DATABASE `" + nativeName + "`"); require.NoError(t, err) })
	config.DBName = nativeName
	native, err := gorm.Open(mysql.Open(config.FormatDSN()), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, native.AutoMigrate(model.AllModels()...))
	nativeSQL, err := native.DB()
	require.NoError(t, err)
	required, err = mlmd.NeedsMigration(context.Background(), nativeSQL)
	require.NoError(t, err)
	require.False(t, required, "fresh native installs must not require MLMD configuration")
	for _, table := range []string{"artifacts", "artifact_tasks"} {
		var manualName, manualDDL, nativeName, nativeDDL string
		require.NoError(t, db.QueryRow("SHOW CREATE TABLE `"+table+"`").Scan(&manualName, &manualDDL))
		require.NoError(t, native.Raw("SHOW CREATE TABLE `"+table+"`").Row().Scan(&nativeName, &nativeDDL))
		require.Equal(t, nativeDDL, manualDDL, table)
	}
}

func readNativeRows(t *testing.T, db *sql.DB) nativeFixture {
	t.Helper()
	store, err := gorm.Open(mysql.New(mysql.Config{Conn: db}), &gorm.Config{})
	require.NoError(t, err)
	var rows nativeFixture
	require.NoError(t, store.Table("tasks").Order("UUID").Find(&rows.Tasks).Error)
	require.NoError(t, store.Table("artifacts").Order("UUID").Find(&rows.Artifacts).Error)
	require.NoError(t, store.Table("artifact_tasks").Order("UUID").Find(&rows.Links).Error)
	return rows
}

func createStagedTasks(t *testing.T, db *sql.DB) {
	t.Helper()
	data, err := os.ReadFile("schema.sql")
	require.NoError(t, err)
	copyDDL := strings.SplitN(string(data), ";", 2)[0]
	copyDDL = strings.ReplaceAll(copyDDL, "`tasks`", "`tasks_migration_tmp`")
	_, err = db.Exec(copyDDL)
	require.NoError(t, err)
}

func TestMySQLResumeAfterInterruptedTaskCopy(t *testing.T) {
	db, _, _, _ := releasedDatabase(t)
	client := sourceClient(t)
	createStagedTasks(t, db)
	id := mlmd.ExecutionUUID(1)
	_, err := db.Exec("INSERT INTO tasks_migration_tmp (UUID,Namespace,RunUUID,pods,CreatedAtInSec,Fingerprint,State,Type,TypeAttrs,DisplayName) VALUES (?,?,?,?,?,?,?,?,?,?)",
		id, "migration-fixture", "11111111-1111-4111-8111-111111111111", "[]", 1700000000, "", 2, 1, "{}", "conflicting-target")
	require.NoError(t, err)

	_, err = mlmd.Migrate(context.Background(), client, db, fixtureDiskBudget)
	require.ErrorContains(t, err, "conflicting migrated fields")
	var legacyID string
	require.NoError(t, db.QueryRow("SELECT MLMDExecutionID FROM tasks WHERE MLMDExecutionID <> '' LIMIT 1").Scan(&legacyID))
	_, err = db.Exec("DELETE FROM tasks_migration_tmp WHERE UUID=?", id)
	require.NoError(t, err)
	var count int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='tasks_migration_tmp'").Scan(&count))
	require.Equal(t, 1, count)
	required, err := mlmd.NeedsMigration(context.Background(), db)
	require.NoError(t, err)
	require.True(t, required, "a partial task copy must resume before serving")
	report, err := mlmd.Migrate(context.Background(), client, db, fixtureDiskBudget)
	require.NoError(t, err)
	require.Equal(t, 12, report.Tasks)
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count))
	require.Equal(t, 12, count)
}

func TestMySQLResumeTaskCopyRepairsInteriorGap(t *testing.T) {
	clean, _, _, cleanName := releasedDatabase(t)
	client := sourceClient(t)
	report, err := mlmd.Migrate(context.Background(), client, clean, fixtureDiskBudget)
	require.NoError(t, err)

	resumed, _, _, _ := releasedDatabase(t)
	createStagedTasks(t, resumed)
	// Execution 2 lies before the binary search probes in this 12-record stream.
	// Parent links and native fingerprints are applied after the task copy.
	missingID := mlmd.ExecutionUUID(2)
	result, err := resumed.Exec(`INSERT INTO tasks_migration_tmp
		(UUID, Namespace, RunUUID, pods, CreatedAtInSec, StartedInSec, FinishedInSec,
		 Fingerprint, CachedTaskUUID, Name, DisplayName, State, StateHistory, InputParameters,
		 OutputParameters, Type, TypeAttrs, StatusMetadata)
		SELECT UUID, Namespace, RunUUID, pods, CreatedAtInSec, StartedInSec, FinishedInSec,
		 '', CachedTaskUUID, Name, DisplayName, State, StateHistory, InputParameters,
		 OutputParameters, Type, TypeAttrs, StatusMetadata
		FROM `+"`"+cleanName+"`"+`.tasks WHERE UUID<>?`, missingID)
	require.NoError(t, err)
	staged, err := result.RowsAffected()
	require.NoError(t, err)
	require.EqualValues(t, report.Tasks-1, staged)

	_, err = mlmd.Migrate(context.Background(), client, resumed, fixtureDiskBudget)
	require.NoError(t, err)
	require.Equal(t, readNativeRows(t, clean), readNativeRows(t, resumed))
}

func TestBuiltCLIOnReleasedMySQL(t *testing.T) {
	db, _, config, _ := releasedDatabase(t)
	oldPodPresent := true
	unreportedWorkflow := false
	activeWorkflow := false
	kube := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/apis/apps/v1/namespaces/kubeflow/deployments/ml-pipeline-scheduledworkflow":
			_, _ = w.Write([]byte(`{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"name":"ml-pipeline-scheduledworkflow","generation":1},"spec":{"replicas":0,"selector":{"matchLabels":{"app":"ml-pipeline-scheduledworkflow"}}},"status":{"observedGeneration":1}}`))
		case "/apis/apps/v1/namespaces/kubeflow/deployments/ml-pipeline":
			_, _ = w.Write([]byte(`{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"name":"ml-pipeline","generation":1},"spec":{"replicas":0,"selector":{"matchLabels":{"app":"ml-pipeline"}}},"status":{"observedGeneration":1}}`))
		case "/api/v1/namespaces/kubeflow/pods":
			if r.URL.Query().Get("labelSelector") == "app=ml-pipeline-scheduledworkflow" {
				_, _ = w.Write([]byte(`{"apiVersion":"v1","kind":"PodList","items":[]}`))
				return
			}
			if oldPodPresent {
				_, _ = w.Write([]byte(`{"apiVersion":"v1","kind":"PodList","items":[{"metadata":{"name":"old-api"}}]}`))
			} else {
				_, _ = w.Write([]byte(`{"apiVersion":"v1","kind":"PodList","items":[]}`))
			}
		case "/apis/argoproj.io/v1alpha1/workflows":
			switch {
			case activeWorkflow:
				_, _ = w.Write([]byte(`{"apiVersion":"argoproj.io/v1alpha1","kind":"WorkflowList","items":[{"metadata":{"namespace":"migration-fixture","name":"active","labels":{"pipeline/runid":"11111111-1111-4111-8111-111111111111","pipeline/persistedFinalState":"true"}},"status":{"phase":"Running"}}]}`))
			case unreportedWorkflow:
				_, _ = w.Write([]byte(`{"apiVersion":"argoproj.io/v1alpha1","kind":"WorkflowList","items":[{"metadata":{"namespace":"migration-fixture","name":"unfinished","labels":{"pipeline/runid":"11111111-1111-4111-8111-111111111111"}},"status":{"phase":"Succeeded"}}]}`))
			default:
				_, _ = w.Write([]byte(`{"apiVersion":"argoproj.io/v1alpha1","kind":"WorkflowList","items":[{"metadata":{"namespace":"another-installation","name":"unrelated","labels":{"pipeline/runid":"11111111-1111-4111-8111-111111111111"}},"status":{"phase":"Running"}}]}`))
			}
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(kube.Close)
	kubeconfig := filepath.Join(t.TempDir(), "kubeconfig")
	require.NoError(t, os.WriteFile(kubeconfig, []byte("apiVersion: v1\nkind: Config\nclusters:\n- name: test\n  cluster:\n    server: "+kube.URL+"\ncontexts:\n- name: test\n  context:\n    cluster: test\n    user: test\ncurrent-context: test\nusers:\n- name: test\n  user: {}\n"), 0600))
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := grpc.NewServer()
	pb.RegisterMetadataStoreServiceServer(server, fixtureServer{fixtureClient: sourceClient(t)})
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { server.Stop(); _ = listener.Close() })
	binary := filepath.Join(t.TempDir(), "mlmd-migrate")
	build := exec.Command("go", "build", "-tags=mlmd_migration_test", "-o", binary, "./cmd/mlmd-migrate")
	output, err := build.CombinedOutput()
	require.NoError(t, err, string(output))
	oldPodPresent = false
	insufficient := exec.Command(binary, "--mlmd-address="+listener.Addr().String(), "--mysql-dsn="+config.FormatDSN(),
		"--available-disk-bytes=1", "--mlmd-database-bytes=1024")
	insufficient.Env = append(os.Environ(), "KUBECONFIG="+kubeconfig)
	output, err = insufficient.CombinedOutput()
	require.Error(t, err, string(output))
	require.Contains(t, string(output), "insufficient MySQL disk capacity")
	var migrationObjects int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM information_schema.TABLES
		WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME IN
		('mlmd_migration_state','tasks_migration_tmp','tasks_pre_migration','artifacts','artifact_tasks')`).Scan(&migrationObjects))
	require.Zero(t, migrationObjects)
	capacity := exec.Command(binary, "--check-capacity", "--mysql-dsn="+config.FormatDSN(),
		"--available-disk-bytes=1073741824", "--mlmd-database-bytes=1048576")
	output, err = capacity.CombinedOutput()
	require.NoError(t, err, string(output))
	var measured mlmd.CapacityReport
	require.NoError(t, json.Unmarshal(output, &measured))
	require.Equal(t, 2*fixtureDiskBudget.MLMDDatabaseBytes+measured.LegacyTasksBytes, measured.RequiredBytes)
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM information_schema.TABLES
		WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME IN
		('mlmd_migration_state','tasks_migration_tmp','tasks_pre_migration','artifacts','artifact_tasks')`).Scan(&migrationObjects))
	require.Zero(t, migrationObjects)

	runMigration := func(database mysqlDriver.Config) ([]byte, error) {
		command := exec.Command(binary, "--mlmd-address="+listener.Addr().String(), "--mysql-dsn="+database.FormatDSN(),
			"--available-disk-bytes=1073741824", "--mlmd-database-bytes=1048576")
		command.Env = append(os.Environ(), "KUBECONFIG="+kubeconfig)
		return command.CombinedOutput()
	}
	oldPodPresent = true
	validate := func(endpoint string) ([]byte, error) {
		command := exec.Command(binary, "validate", "--mlmd-address="+endpoint, "--mysql-dsn="+config.FormatDSN())
		command.Env = append(os.Environ(), "KUBECONFIG="+kubeconfig)
		return command.CombinedOutput()
	}
	output, err = validate(listener.Addr().String())
	require.NoError(t, err, string(output))
	require.Contains(t, string(output), `"status":"validated"`)
	_, err = db.Exec("UPDATE run_details SET State='SUCCEEDED', FinishedAtInSec=NULL")
	require.NoError(t, err)
	activeWorkflow = true
	output, err = validate(listener.Addr().String())
	require.Error(t, err)
	require.Contains(t, string(output), "no persisted finish time")
	require.Contains(t, string(output), "Workflow migration-fixture/active is not terminal")
	activeWorkflow = false
	_, err = db.Exec("UPDATE run_details SET FinishedAtInSec=1700000003")
	require.NoError(t, err)
	output, err = validate("127.0.0.1:1")
	require.ErrorContains(t, err, "exit status 1")
	require.Contains(t, string(output), "read MLMD execution types")
	_, err = db.Exec("ALTER TABLE tasks RENAME COLUMN PodName TO UnknownPodName")
	require.NoError(t, err)
	output, err = validate(listener.Addr().String())
	require.Error(t, err)
	require.Contains(t, string(output), "unqualified schema")
	_, err = db.Exec("ALTER TABLE tasks RENAME COLUMN UnknownPodName TO PodName")
	require.NoError(t, err)
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE()
		AND TABLE_NAME IN ('mlmd_migration_state','mlmd_migration_complete','tasks_migration_tmp','tasks_pre_migration','artifacts','artifact_tasks')`).Scan(&migrationObjects))
	require.Zero(t, migrationObjects, "preflight must not write")
	check := exec.Command(binary, "--drain-check", "--mysql-dsn="+config.FormatDSN())
	check.Env = append(os.Environ(), "KUBECONFIG="+kubeconfig)
	output, err = check.CombinedOutput()
	require.NoError(t, err, string(output))
	require.Contains(t, string(output), `"status":"drained"`)
	command := exec.Command(binary, "--mlmd-address="+listener.Addr().String(), "--mysql-dsn="+config.FormatDSN())
	command.Env = append(os.Environ(), "KUBECONFIG="+kubeconfig)
	output, err = command.CombinedOutput()
	require.Error(t, err)
	require.Contains(t, string(output), "no remaining pods")
	oldPodPresent = false
	check = exec.Command(binary, "--gate-only", "--mysql-dsn="+config.FormatDSN())
	check.Env = append(os.Environ(), "KUBECONFIG="+kubeconfig)
	output, err = check.CombinedOutput()
	require.NoError(t, err, string(output))
	require.Contains(t, string(output), `"status":"fenced"`)
	_, err = db.Exec("UPDATE run_details SET State='RUNNING', FinishedAtInSec=0")
	require.NoError(t, err)
	output, err = runMigration(config)
	require.Error(t, err)
	require.Contains(t, string(output), "lacks a stored terminal report")
	_, err = db.Exec("UPDATE run_details SET State='SUCCEEDED', FinishedAtInSec=1700000003")
	require.NoError(t, err)
	unreportedWorkflow = true
	output, err = runMigration(config)
	require.Error(t, err)
	require.Contains(t, string(output), "no stored terminal report")
	unreportedWorkflow = false
	activeWorkflow = true
	output, err = runMigration(config)
	require.Error(t, err)
	require.Contains(t, string(output), "is not terminal")
	activeWorkflow = false
	output, err = runMigration(config)
	require.NoError(t, err, string(output))
	var report struct {
		Status string
		mlmd.Report
	}
	require.NoError(t, json.Unmarshal(output, &report))
	require.Equal(t, "complete", report.Status)
	require.Equal(t, 12, report.Tasks)
	require.Len(t, report.Skipped, 2)
	var count int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count))
	require.Equal(t, 12, count)

	resumed, _, resumeConfig, _ := releasedDatabase(t)
	createStagedTasks(t, resumed)
	_, err = resumed.Exec(`CREATE TRIGGER interrupt_copy BEFORE INSERT ON tasks_migration_tmp FOR EACH ROW
		SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'task copy interrupted'`)
	require.NoError(t, err)
	output, err = runMigration(resumeConfig)
	require.Error(t, err)
	require.Contains(t, string(output), "task copy interrupted")
	require.NoError(t, resumed.QueryRow("SELECT COUNT(*) FROM tasks_migration_tmp").Scan(&count))
	require.Zero(t, count, "the first task batch should roll back")
	var cutoff int64
	require.NoError(t, resumed.QueryRow("SELECT CacheCutoffInSec FROM mlmd_migration_state WHERE ID=1").Scan(&cutoff))
	_, err = resumed.Exec("DROP TRIGGER interrupt_copy")
	require.NoError(t, err)
	output, err = runMigration(resumeConfig)
	require.NoError(t, err, string(output))
	require.NoError(t, json.Unmarshal(output, &report))
	require.Equal(t, "complete", report.Status)
	require.Equal(t, 12, report.Tasks)
	var resumedCutoff int64
	require.NoError(t, resumed.QueryRow("SELECT CacheCutoffInSec FROM mlmd_migration_state WHERE ID=1").Scan(&resumedCutoff))
	require.Equal(t, cutoff, resumedCutoff)
	require.Equal(t, readNativeRows(t, db), readNativeRows(t, resumed))

	completed := exec.Command(binary, "--mysql-dsn="+config.FormatDSN())
	output, err = completed.CombinedOutput()
	require.NoError(t, err, string(output))
	require.NoError(t, json.Unmarshal(output, &report))
	require.Equal(t, "complete", report.Status)
	require.Equal(t, 12, report.Tasks)
}

func TestMySQLMigrationRetriesTransientMLMDErrors(t *testing.T) {
	db, _, _, _ := releasedDatabase(t)
	client := &flakyClient{fixtureClient: sourceClient(t), failures: 2}
	report, err := mlmd.Migrate(context.Background(), client, db, fixtureDiskBudget)
	require.NoError(t, err)
	require.Zero(t, client.failures)
	require.Equal(t, 12, report.Tasks)
}

func TestMySQLMigrationFailsBeforeDDLWhenMLMDUnavailable(t *testing.T) {
	db, _, _, _ := releasedDatabase(t)
	client := &flakyClient{fixtureClient: sourceClient(t), failures: 3}
	_, err := mlmd.Migrate(context.Background(), client, db, fixtureDiskBudget)
	require.ErrorContains(t, err, "exhausted connection retries")
	var legacyColumnCount, temporaryTableCount int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='tasks' AND COLUMN_NAME='MLMDExecutionID'").Scan(&legacyColumnCount))
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='tasks_migration_tmp'").Scan(&temporaryTableCount))
	require.Equal(t, 1, legacyColumnCount)
	require.Zero(t, temporaryTableCount)
}

func TestMySQLMigrationRefusesActiveRunBeforeDDL(t *testing.T) {
	db, _, _, _ := releasedDatabase(t)
	_, err := db.Exec("UPDATE run_details SET State='RUNNING', FinishedAtInSec=0")
	require.NoError(t, err)
	_, err = mlmd.Migrate(context.Background(), sourceClient(t), db, fixtureDiskBudget)
	require.ErrorContains(t, err, "is not terminal")
	var temporaryTables int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='tasks_migration_tmp'").Scan(&temporaryTables))
	require.Zero(t, temporaryTables)
	_, err = db.Exec("UPDATE run_details SET State='SUCCEEDED', FinishedAtInSec=0")
	require.NoError(t, err)
	_, err = mlmd.Migrate(context.Background(), sourceClient(t), db, fixtureDiskBudget)
	require.ErrorContains(t, err, "no persisted finish time")
}

func TestMySQLMigrationAllowsMissingSavedSpec(t *testing.T) {
	db, _, _, _ := releasedDatabase(t)
	_, err := db.Exec("UPDATE run_details SET PipelineSpecManifest=NULL")
	require.NoError(t, err)
	report, err := mlmd.Migrate(context.Background(), sourceClient(t), db, fixtureDiskBudget)
	require.NoError(t, err)
	require.Equal(t, 12, report.Tasks)
}

func TestMySQLReportsRejectedLegacyTaskRow(t *testing.T) {
	db, _, _, _ := releasedDatabase(t)
	const legacyID = "33333333-3333-4333-8333-333333333333"
	result, err := db.Exec("UPDATE tasks SET StateHistory='{}' WHERE UUID=?", legacyID)
	require.NoError(t, err)
	updated, err := result.RowsAffected()
	require.NoError(t, err)
	require.EqualValues(t, 1, updated)

	report, err := mlmd.Migrate(context.Background(), sourceClient(t), db, fixtureDiskBudget)
	require.NoError(t, err)
	var retainedID string
	require.NoError(t, db.QueryRow("SELECT UUID FROM tasks WHERE UUID=?", mlmd.ExecutionUUID(7)).Scan(&retainedID))
	require.Equal(t, mlmd.ExecutionUUID(7), retainedID)
	require.Equal(t, 1, report.SkippedByKind["legacy_cache_row"])
	for _, skip := range report.Skipped {
		if skip.Kind == "legacy_cache_row" {
			require.EqualValues(t, 7, skip.ID)
			require.Equal(t, legacyID, skip.Source)
			require.Contains(t, skip.Reason, "decode KFP state history")
			require.Equal(t, 1, report.SkippedByReason[skip.Kind][skip.Reason])
			return
		}
	}
	t.Fatal("missing legacy cache row skip")
}

func TestMySQLRestorePreMigrationDatabaseDump(t *testing.T) {
	db, _, config, _ := releasedDatabase(t)
	before := dumpMySQLDatabase(t, config)
	_, err := mlmd.Migrate(context.Background(), sourceClient(t), db, fixtureDiskBudget)
	require.NoError(t, err)
	require.False(t, bytes.Equal(before, dumpMySQLDatabase(t, config)), "migration must change the schema and data")

	restored, _, restoreConfig, _ := scratchMySQLDatabase(t)
	_, err = restored.Exec(string(before))
	require.NoError(t, err)
	require.True(t, bytes.Equal(before, dumpMySQLDatabase(t, restoreConfig)), "restored schema and data must match the pre-migration dump")
	var state, executionID string
	require.NoError(t, restored.QueryRow("SELECT State, MLMDExecutionID FROM tasks WHERE MLMDExecutionID='7'").Scan(&state, &executionID))
	require.Equal(t, "SUCCEEDED", state)
	require.Equal(t, "7", executionID)
	var nativeColumn int
	require.NoError(t, restored.QueryRow("SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='tasks' AND COLUMN_NAME='TypeAttrs'").Scan(&nativeColumn))
	require.Zero(t, nativeColumn)
}

func dumpMySQLDatabase(t *testing.T, config mysqlDriver.Config) []byte {
	t.Helper()
	args := []string{"--no-defaults", "--single-transaction", "--no-tablespaces", "--column-statistics=0",
		"--set-gtid-purged=OFF", "--skip-comments", "--order-by-primary", "--user=" + config.User}
	if config.Net == "unix" {
		args = append(args, "--protocol=socket", "--socket="+config.Addr)
	} else {
		host, port, err := net.SplitHostPort(config.Addr)
		require.NoError(t, err)
		args = append(args, "--protocol=tcp", "--host="+host, "--port="+port)
	}
	command := exec.Command("mysqldump", append(args, config.DBName)...)
	command.Env = append(os.Environ(), "MYSQL_PWD="+config.Passwd)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	dump, err := command.Output()
	require.NoError(t, err, "mysqldump: %s", stderr.String())
	return dump
}

func TestMySQLUnqualifiedSchemaFailsBeforeDDL(t *testing.T) {
	db, _, _, _ := releasedDatabase(t)
	_, err := db.Exec("ALTER TABLE tasks DROP COLUMN PodName")
	require.NoError(t, err)
	_, err = mlmd.Migrate(context.Background(), sourceClient(t), db, fixtureDiskBudget)
	require.ErrorContains(t, err, "refusing unqualified schema")
	var temporaryTableCount int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='tasks_migration_tmp'").Scan(&temporaryTableCount))
	require.Zero(t, temporaryTableCount)
}

func TestMySQLRejectsChangedMigrationIndex(t *testing.T) {
	db, _, _, _ := releasedDatabase(t)
	_, err := db.Exec("CREATE TABLE released_tasks LIKE tasks")
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO released_tasks SELECT * FROM tasks")
	require.NoError(t, err)
	client := sourceClient(t)
	_, err = mlmd.Migrate(context.Background(), client, db, fixtureDiskBudget)
	require.NoError(t, err)
	_, err = db.Exec("RENAME TABLE released_tasks TO tasks_pre_migration")
	require.NoError(t, err)
	_, err = db.Exec("ALTER TABLE artifact_tasks DROP INDEX UniqueLink")
	require.NoError(t, err)

	_, err = mlmd.Migrate(context.Background(), client, db, fixtureDiskBudget)
	require.ErrorContains(t, err, "index UniqueLink")
	var legacyRows int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM tasks_pre_migration").Scan(&legacyRows))
	require.Positive(t, legacyRows)

	_, err = db.Exec("ALTER TABLE artifact_tasks ADD UNIQUE KEY UniqueLink (ArtifactID, TaskID, Type, Iteration, ArtifactKey)")
	require.NoError(t, err)
	_, err = db.Exec("ALTER TABLE artifact_tasks DROP FOREIGN KEY fk_artifact_tasks_task_run")
	require.NoError(t, err)

	_, err = mlmd.Migrate(context.Background(), client, db, fixtureDiskBudget)
	require.ErrorContains(t, err, "foreign key fk_artifact_tasks_task_run")
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM tasks_pre_migration").Scan(&legacyRows))
	require.Positive(t, legacyRows)
}

func TestMySQLRejectsNarrowedMigrationColumn(t *testing.T) {
	db, _, _, _ := releasedDatabase(t)
	_, err := db.Exec("CREATE TABLE released_tasks LIKE tasks")
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO released_tasks SELECT * FROM tasks")
	require.NoError(t, err)
	client := sourceClient(t)
	_, err = mlmd.Migrate(context.Background(), client, db, fixtureDiskBudget)
	require.NoError(t, err)
	_, err = db.Exec("RENAME TABLE released_tasks TO tasks_pre_migration")
	require.NoError(t, err)
	_, err = db.Exec("ALTER TABLE artifacts MODIFY COLUMN Name varchar(127)")
	require.NoError(t, err)

	_, err = mlmd.Migrate(context.Background(), client, db, fixtureDiskBudget)
	require.ErrorContains(t, err, "artifacts column Name")
	var legacyRows int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM tasks_pre_migration").Scan(&legacyRows))
	require.Positive(t, legacyRows)
}

func TestMySQLCutoverRespectsSessionLockTimeout(t *testing.T) {
	db, _, config, _ := releasedDatabase(t)
	db.SetMaxOpenConns(1)
	_, err := db.Exec("SET SESSION lock_wait_timeout=3")
	require.NoError(t, err)
	blocker, err := sql.Open("mysql", config.FormatDSN())
	require.NoError(t, err)
	defer blocker.Close()
	tx, err := blocker.Begin()
	require.NoError(t, err)
	defer tx.Rollback()
	var taskID string
	require.NoError(t, tx.QueryRow("SELECT UUID FROM tasks LIMIT 1").Scan(&taskID))

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	client := sourceClient(t)
	migrationResult := make(chan error, 1)
	go func() {
		_, err := mlmd.Migrate(ctx, client, db, fixtureDiskBudget)
		migrationResult <- err
	}()
	require.Eventually(t, func() bool {
		var waiting int
		err := blocker.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.PROCESSLIST
			WHERE DB=? AND STATE='Waiting for table metadata lock' AND INFO LIKE 'RENAME TABLE tasks %'`, config.DBName).Scan(&waiting)
		return err == nil && waiting == 1
	}, 5*time.Second, 25*time.Millisecond, "the swap must be waiting on the old tasks table")
	readWriteContext, cancelReadWrite := context.WithTimeout(ctx, time.Second)
	defer cancelReadWrite()
	var runID string
	require.NoError(t, blocker.QueryRowContext(readWriteContext, "SELECT UUID FROM run_details LIMIT 1").Scan(&runID))
	_, err = blocker.ExecContext(readWriteContext, "UPDATE experiments SET Description='concurrent update' WHERE UUID='fixture-experiment'")
	require.NoError(t, err, "unrelated writes must complete while the swap waits")
	select {
	case err := <-migrationResult:
		t.Fatalf("migration ended before the concurrent read and write: %v", err)
	default:
	}

	// MySQL also locks the parent of the task table's foreign key. A write to
	// run_details can wait, but must be allowed through when the DDL gives up.
	relatedWriteContext, cancelRelatedWrite := context.WithTimeout(ctx, 2*time.Second)
	defer cancelRelatedWrite()
	_, err = blocker.ExecContext(relatedWriteContext, "UPDATE run_details SET DisplayName='concurrent update' WHERE UUID=?", runID)
	require.NoError(t, err, "the related write must survive the bounded DDL wait")
	require.ErrorContains(t, <-migrationResult, "MySQL lock wait expired during MLMD migration")
	var timeout int
	require.NoError(t, db.QueryRow("SELECT @@SESSION.lock_wait_timeout").Scan(&timeout))
	require.Equal(t, 3, timeout)
	var legacyID string
	require.NoError(t, db.QueryRow("SELECT MLMDExecutionID FROM tasks WHERE UUID=?", taskID).Scan(&legacyID))

	require.NoError(t, tx.Rollback())
	report, err := mlmd.Migrate(context.Background(), client, db, fixtureDiskBudget)
	require.NoError(t, err)
	require.Equal(t, 12, report.Tasks)
}
