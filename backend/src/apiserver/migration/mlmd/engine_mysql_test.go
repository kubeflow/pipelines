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
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

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
	data, err := os.ReadFile("testdata/legacy_source.json")
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
	report, err := mlmd.Migrate(context.Background(), client, db)
	require.NoError(t, err)
	required, err = mlmd.NeedsMigration(context.Background(), db)
	require.NoError(t, err)
	require.False(t, required)
	_, err = db.Exec("DROP TABLE artifact_tasks")
	require.NoError(t, err)
	required, err = mlmd.NeedsMigration(context.Background(), db)
	require.NoError(t, err)
	require.True(t, required, "a missing native table invalidates the completion marker")
	_, err = mlmd.Migrate(context.Background(), client, db)
	require.NoError(t, err)
	_, err = db.Exec("DELETE FROM mlmd_migration_complete WHERE ID=1")
	require.NoError(t, err)
	required, err = mlmd.NeedsMigration(context.Background(), db)
	require.NoError(t, err)
	require.True(t, required, "a missing completion row must resume migration")
	_, err = mlmd.Migrate(context.Background(), client, db)
	require.NoError(t, err)
	require.Equal(t, 12, report.Tasks)
	require.Equal(t, 4, report.Artifacts)
	require.Equal(t, 8, report.Links)
	for _, table := range []string{"tasks", "artifacts", "artifact_tasks"} {
		var count int
		require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM `"+table+"`").Scan(&count))
		require.Positive(t, count)
	}
	var backupCount int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='tasks_pre_migration'").Scan(&backupCount))
	require.Zero(t, backupCount)
	// A later native startup must reconcile the migrated schema without MLMD.
	startup, err := gorm.Open(mysql.Open(config.FormatDSN()), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, startup.AutoMigrate(model.AllModels()...))
	var fixture sourceFixture
	readFixture(t, "legacy_source.json", &fixture)
	_, err = db.Exec("DELETE FROM artifacts WHERE UUID=?", mlmd.MetricUUID(fixture.ArtifactIDs["metrics"], "loss"))
	require.NoError(t, err)
	var removedLink string
	require.NoError(t, db.QueryRow("SELECT UUID FROM artifact_tasks ORDER BY UUID DESC LIMIT 1").Scan(&removedLink))
	_, err = db.Exec("DELETE FROM artifact_tasks WHERE UUID=?", removedLink)
	require.NoError(t, err)
	repeated, err := mlmd.Migrate(context.Background(), client, db)
	require.NoError(t, err)
	require.Equal(t, report.Tasks, repeated.Tasks)
	var artifacts, links int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM artifacts").Scan(&artifacts))
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM artifact_tasks").Scan(&links))
	require.Equal(t, 4, artifacts)
	require.Equal(t, 8, links)
	owner, err := db.Conn(context.Background())
	require.NoError(t, err)
	var acquired int
	require.NoError(t, owner.QueryRowContext(context.Background(), "SELECT GET_LOCK('kfp_mlmd_migration_lock', 0)").Scan(&acquired))
	require.Equal(t, 1, acquired)
	_, err = mlmd.Migrate(context.Background(), client, db)
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

func TestMySQLResumeAfterInterruptedTaskCopy(t *testing.T) {
	db, _, _, _ := releasedDatabase(t)
	client := sourceClient(t)
	data, err := os.ReadFile("schema.sql")
	require.NoError(t, err)
	copyDDL := strings.SplitN(string(data), ";", 2)[0]
	copyDDL = strings.Replace(copyDDL, "CREATE TABLE `tasks`", "CREATE TABLE `tasks_migration_tmp`", 1)
	start := strings.Index(copyDDL, ",\n  CONSTRAINT `fk_tasks_parent_task`")
	require.Positive(t, start)
	end := strings.Index(copyDDL[start:], "\n)")
	require.Positive(t, end)
	copyDDL = copyDDL[:start] + copyDDL[start+end:]
	_, err = db.Exec(copyDDL)
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO tasks_migration_tmp (UUID,Namespace,RunUUID,pods,CreatedAtInSec,Fingerprint,State,Type,TypeAttrs) VALUES (?,?,?,?,?,?,?,?,?)",
		mlmd.ExecutionUUID(1), "migration-fixture", "11111111-1111-4111-8111-111111111111", "[]", 1700000000, "", 2, 1, "{}")
	require.NoError(t, err)
	var count int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='tasks_migration_tmp'").Scan(&count))
	require.Equal(t, 1, count)
	required, err := mlmd.NeedsMigration(context.Background(), db)
	require.NoError(t, err)
	require.True(t, required, "a partial task copy must resume before serving")
	report, err := mlmd.Migrate(context.Background(), client, db)
	require.NoError(t, err)
	require.Equal(t, 12, report.Tasks)
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count))
	require.Equal(t, 12, count)
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
			if activeWorkflow {
				_, _ = w.Write([]byte(`{"apiVersion":"argoproj.io/v1alpha1","kind":"WorkflowList","items":[{"metadata":{"namespace":"kubeflow","name":"active","labels":{"pipeline/runid":"run-1","pipeline/persistedFinalState":"true"}},"status":{"phase":"Running"}}]}`))
			} else if unreportedWorkflow {
				_, _ = w.Write([]byte(`{"apiVersion":"argoproj.io/v1alpha1","kind":"WorkflowList","items":[{"metadata":{"namespace":"kubeflow","name":"unfinished","labels":{"pipeline/runid":"run-1"}},"status":{"phase":"Succeeded"}}]}`))
			} else {
				_, _ = w.Write([]byte(`{"apiVersion":"argoproj.io/v1alpha1","kind":"WorkflowList","items":[]}`))
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
	build := exec.Command("go", "build", "-o", binary, "./cmd/mlmd-migrate")
	output, err := build.CombinedOutput()
	require.NoError(t, err, string(output))
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
	command = exec.Command(binary, "--mlmd-address="+listener.Addr().String(), "--mysql-dsn="+config.FormatDSN())
	command.Env = append(os.Environ(), "KUBECONFIG="+kubeconfig)
	output, err = command.CombinedOutput()
	require.Error(t, err)
	require.Contains(t, string(output), "lack a stored terminal report")
	_, err = db.Exec("UPDATE run_details SET State='SUCCEEDED', FinishedAtInSec=1700000003")
	require.NoError(t, err)
	unreportedWorkflow = true
	command = exec.Command(binary, "--mlmd-address="+listener.Addr().String(), "--mysql-dsn="+config.FormatDSN())
	command.Env = append(os.Environ(), "KUBECONFIG="+kubeconfig)
	output, err = command.CombinedOutput()
	require.Error(t, err)
	require.Contains(t, string(output), "no stored terminal report")
	unreportedWorkflow = false
	activeWorkflow = true
	command = exec.Command(binary, "--mlmd-address="+listener.Addr().String(), "--mysql-dsn="+config.FormatDSN())
	command.Env = append(os.Environ(), "KUBECONFIG="+kubeconfig)
	output, err = command.CombinedOutput()
	require.Error(t, err)
	require.Contains(t, string(output), "is not terminal")
	activeWorkflow = false
	command = exec.Command(binary, "--mlmd-address="+listener.Addr().String(), "--mysql-dsn="+config.FormatDSN())
	command.Env = append(os.Environ(), "KUBECONFIG="+kubeconfig)
	output, err = command.CombinedOutput()
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
}

func TestMySQLMigrationRetriesTransientMLMDErrors(t *testing.T) {
	db, _, _, _ := releasedDatabase(t)
	client := &flakyClient{fixtureClient: sourceClient(t), failures: 2}
	report, err := mlmd.Migrate(context.Background(), client, db)
	require.NoError(t, err)
	require.Zero(t, client.failures)
	require.Equal(t, 12, report.Tasks)
}

func TestMySQLMigrationFailsBeforeDDLWhenMLMDUnavailable(t *testing.T) {
	db, _, _, _ := releasedDatabase(t)
	client := &flakyClient{fixtureClient: sourceClient(t), failures: 3}
	_, err := mlmd.Migrate(context.Background(), client, db)
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
	_, err = mlmd.Migrate(context.Background(), sourceClient(t), db)
	require.ErrorContains(t, err, "drain runs before migration")
	var temporaryTables int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='tasks_migration_tmp'").Scan(&temporaryTables))
	require.Zero(t, temporaryTables)
	_, err = db.Exec("UPDATE run_details SET State='SUCCEEDED', FinishedAtInSec=0")
	require.NoError(t, err)
	_, err = mlmd.Migrate(context.Background(), sourceClient(t), db)
	require.ErrorContains(t, err, "no persisted finish time")
}

func TestMySQLMigrationAllowsMissingSavedSpec(t *testing.T) {
	db, _, _, _ := releasedDatabase(t)
	_, err := db.Exec("UPDATE run_details SET PipelineSpecManifest=NULL")
	require.NoError(t, err)
	report, err := mlmd.Migrate(context.Background(), sourceClient(t), db)
	require.NoError(t, err)
	require.Equal(t, 12, report.Tasks)
}

func TestMySQLRestoreReleasedBackupAfterMigration(t *testing.T) {
	db, admin, config, name := releasedDatabase(t)
	_, err := mlmd.Migrate(context.Background(), sourceClient(t), db)
	require.NoError(t, err)
	require.NoError(t, db.Close())
	_, err = admin.Exec("DROP DATABASE `" + name + "`")
	require.NoError(t, err)
	_, err = admin.Exec("CREATE DATABASE `" + name + "`")
	require.NoError(t, err)
	restored, err := sql.Open("mysql", config.FormatDSN())
	require.NoError(t, err)
	defer restored.Close()
	var backup struct {
		Statements []string `json:"statements"`
	}
	readFixture(t, "legacy_schema_mysql.json", &backup)
	for _, statement := range backup.Statements {
		_, err := restored.Exec(statement)
		require.NoError(t, err)
	}
	var state, executionID string
	require.NoError(t, restored.QueryRow("SELECT State, MLMDExecutionID FROM tasks WHERE MLMDExecutionID='7'").Scan(&state, &executionID))
	require.Equal(t, "SUCCEEDED", state)
	require.Equal(t, "7", executionID)
	var nativeColumn int
	require.NoError(t, restored.QueryRow("SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='tasks' AND COLUMN_NAME='TypeAttrs'").Scan(&nativeColumn))
	require.Zero(t, nativeColumn)
}

func TestMySQLUnqualifiedSchemaFailsBeforeDDL(t *testing.T) {
	db, _, _, _ := releasedDatabase(t)
	_, err := db.Exec("ALTER TABLE tasks DROP COLUMN PodName")
	require.NoError(t, err)
	_, err = mlmd.Migrate(context.Background(), sourceClient(t), db)
	require.ErrorContains(t, err, "refusing unqualified schema")
	var temporaryTableCount int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='tasks_migration_tmp'").Scan(&temporaryTableCount))
	require.Zero(t, temporaryTableCount)
}
