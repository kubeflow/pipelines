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
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"github.com/kubeflow/pipelines/backend/src/apiserver/migration/mlmd"
	"github.com/kubeflow/pipelines/backend/src/apiserver/migration/mlmdfixture"
	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

type emptyMigrationSource struct {
	pb.UnimplementedMetadataStoreServiceServer
}

func writeMigrationKubeconfig(t *testing.T, serverURL string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "kubeconfig")
	require.NoError(t, os.WriteFile(path, []byte("apiVersion: v1\nkind: Config\nclusters:\n- name: test\n  cluster:\n    server: "+serverURL+"\ncontexts:\n- name: test\n  context:\n    cluster: test\n    user: test\ncurrent-context: test\nusers:\n- name: test\n  user: {}\n"), 0600))
	return path
}

func (emptyMigrationSource) GetExecutions(context.Context, *pb.GetExecutionsRequest) (*pb.GetExecutionsResponse, error) {
	return &pb.GetExecutionsResponse{}, nil
}
func (emptyMigrationSource) GetArtifacts(context.Context, *pb.GetArtifactsRequest) (*pb.GetArtifactsResponse, error) {
	return &pb.GetArtifactsResponse{}, nil
}

func TestMLMDStartupMigrationAndNativeRestart(t *testing.T) {
	dsn := os.Getenv("KFP_MLMD_MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("set KFP_MLMD_MYSQL_TEST_DSN to a disposable loopback MySQL service")
	}
	db, settings := legacyStartupDatabase(t, dsn)
	for _, query := range readLegacySchema(t).Statements {
		_, err := db.Exec(query)
		require.NoError(t, err)
	}
	// This test covers startup routing and completion. The migration package's
	// released-source tests cover nonempty graph conversion.
	_, err := db.Exec("DELETE FROM tasks")
	require.NoError(t, err)
	_, err = db.Exec("UPDATE run_details SET State='SUCCEEDED', FinishedAtInSec=1700000003")
	require.NoError(t, err)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	grpcServer := grpc.NewServer()
	pb.RegisterMetadataStoreServiceServer(grpcServer, emptyMigrationSource{})
	go func() { _ = grpcServer.Serve(listener) }()
	t.Cleanup(func() { grpcServer.Stop(); _ = listener.Close() })
	oldPodPresent := true
	recurringRunning := false
	kube := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/apis/apps/v1/namespaces/kubeflow/deployments/ml-pipeline-scheduledworkflow":
			if recurringRunning {
				_, _ = w.Write([]byte(`{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"name":"ml-pipeline-scheduledworkflow","generation":1},"spec":{"replicas":1,"selector":{"matchLabels":{"app":"ml-pipeline-scheduledworkflow"}}},"status":{"observedGeneration":1}}`))
			} else {
				_, _ = w.Write([]byte(`{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"name":"ml-pipeline-scheduledworkflow","generation":1},"spec":{"replicas":0,"selector":{"matchLabels":{"app":"ml-pipeline-scheduledworkflow"}}},"status":{"observedGeneration":1}}`))
			}
		case "/apis/apps/v1/namespaces/kubeflow/deployments/ml-pipeline":
			if oldPodPresent {
				_, _ = w.Write([]byte(`{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"name":"ml-pipeline","generation":1},"spec":{"replicas":1,"selector":{"matchLabels":{"app":"ml-pipeline"}},"template":{"spec":{"containers":[{"name":"ml-pipeline-api-server","image":"test/new"}]}}},"status":{"observedGeneration":1,"replicas":2,"updatedReplicas":1}}`))
			} else {
				_, _ = w.Write([]byte(`{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"name":"ml-pipeline","generation":1},"spec":{"replicas":1,"selector":{"matchLabels":{"app":"ml-pipeline"}},"template":{"spec":{"containers":[{"name":"ml-pipeline-api-server","image":"test/new"}]}}},"status":{"observedGeneration":1,"replicas":1,"updatedReplicas":1}}`))
			}
		case "/api/v1/namespaces/kubeflow/pods":
			if r.URL.Query().Get("labelSelector") == "app=ml-pipeline-scheduledworkflow" {
				_, _ = w.Write([]byte(`{"apiVersion":"v1","kind":"PodList","items":[]}`))
				return
			}
			if oldPodPresent {
				_, _ = w.Write([]byte(`{"apiVersion":"v1","kind":"PodList","items":[{"metadata":{"name":"new-api"},"spec":{"containers":[{"name":"ml-pipeline-api-server","image":"test/new"}]},"status":{"containerStatuses":[{"name":"ml-pipeline-api-server","imageID":"sha256:new"}]}},{"metadata":{"name":"old-api"},"spec":{"containers":[{"name":"ml-pipeline-api-server","image":"test/old"}]},"status":{"containerStatuses":[{"name":"ml-pipeline-api-server","imageID":"sha256:old"}]}}]}`))
			} else {
				_, _ = w.Write([]byte(`{"apiVersion":"v1","kind":"PodList","items":[{"metadata":{"name":"new-api"},"spec":{"containers":[{"name":"ml-pipeline-api-server","image":"test/new"}]},"status":{"containerStatuses":[{"name":"ml-pipeline-api-server","imageID":"sha256:new"}]}}]}`))
			}
		case "/apis/argoproj.io/v1alpha1/workflows":
			_, _ = w.Write([]byte(`{"apiVersion":"argoproj.io/v1alpha1","kind":"WorkflowList","items":[]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(kube.Close)
	kubeconfig := writeMigrationKubeconfig(t, kube.URL)
	config, err := json.Marshal(settings)
	require.NoError(t, err)
	host, port, err := net.SplitHostPort(listener.Addr().String())
	require.NoError(t, err)
	start := func(migrationSettings, wantSuccess bool) string {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestMLMDLegacyStartup$")
		command.Env = append(os.Environ(), "KFP_MLMD_STARTUP_CHILD_CONFIG="+string(config), "KUBECONFIG="+kubeconfig, "POD_NAME=new-api")
		if migrationSettings {
			command.Env = append(command.Env, "METADATA_GRPC_SERVICE_HOST="+host, "METADATA_GRPC_SERVICE_PORT="+port)
		} else {
			command.Env = append(command.Env, "METADATA_GRPC_SERVICE_HOST=", "METADATA_GRPC_SERVICE_PORT=")
		}
		output, err := command.CombinedOutput()
		require.NoError(t, ctx.Err(), string(output))
		if wantSuccess {
			require.NoError(t, err, string(output))
		} else {
			require.Error(t, err, string(output))
		}
		return string(output)
	}
	require.Contains(t, start(true, false), "old replicas may restart")
	required, err := mlmd.NeedsMigration(context.Background(), db)
	require.NoError(t, err)
	require.True(t, required, "mixed-version rollout must fail before migration")
	oldPodPresent = false
	recurringRunning = true
	require.Contains(t, start(true, false), "recurring-run controller must be scaled to zero")
	recurringRunning = false
	start(true, true)
	required, err = mlmd.NeedsMigration(context.Background(), db)
	require.NoError(t, err)
	require.False(t, required)
	start(false, true)
}

func TestMLMD218ReleasedStartupMigration(t *testing.T) {
	for _, mode := range []string{"API", "CLI"} {
		t.Run(mode, func(t *testing.T) {
			dsn := os.Getenv("KFP_MLMD_MYSQL_TEST_DSN")
			if dsn == "" {
				t.Skip("set KFP_MLMD_MYSQL_TEST_DSN to a disposable loopback MySQL service")
			}
			db, settings := legacyStartupDatabase(t, dsn)
			directory := "../migration/mlmd/testdata/release_218"
			dump, err := os.ReadFile(filepath.Join(directory, "source.sql"))
			require.NoError(t, err)
			prelude, rest, found := strings.Cut(string(dump), "CREATE DATABASE /*!32312 IF NOT EXISTS*/ `mlpipeline`")
			require.True(t, found)
			_, rest, found = strings.Cut(rest, "USE `mlpipeline`;")
			require.True(t, found)
			sourceSQL, _, found := strings.Cut(rest, "CREATE DATABASE /*!32312 IF NOT EXISTS*/ `metadb`")
			require.True(t, found)
			_, err = db.Exec(prelude + sourceSQL)
			require.NoError(t, err)
			const installationUUID = "11111111-2222-4333-8444-555555555555"
			_, err = db.Exec("INSERT INTO transfer_identities (ID, UUID) VALUES (1, ?)", installationUUID)
			require.NoError(t, err)
			source, err := mlmdfixture.Load(filepath.Join(directory, "source_graph.json"))
			require.NoError(t, err)
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err)
			grpcServer := grpc.NewServer()
			pb.RegisterMetadataStoreServiceServer(grpcServer, mlmdfixture.Server{Source: source})
			go func() { _ = grpcServer.Serve(listener) }()
			t.Cleanup(func() { grpcServer.Stop(); _ = listener.Close() })
			var workflows []json.RawMessage
			for _, name := range []string{"parameters_workflow.json", "parameters_cached_workflow.json", "metrics_workflow.json"} {
				data, err := os.ReadFile(filepath.Join(directory, name))
				require.NoError(t, err)
				workflows = append(workflows, data)
			}
			apiPodPresent := mode == "API"
			kube := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/apis/apps/v1/namespaces/kubeflow/deployments/ml-pipeline-scheduledworkflow":
					_, _ = w.Write([]byte(`{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"name":"ml-pipeline-scheduledworkflow","generation":1},"spec":{"replicas":0,"selector":{"matchLabels":{"app":"ml-pipeline-scheduledworkflow"}}},"status":{"observedGeneration":1}}`))
				case "/apis/apps/v1/namespaces/kubeflow/deployments/ml-pipeline":
					if apiPodPresent {
						_, _ = w.Write([]byte(`{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"name":"ml-pipeline","generation":1},"spec":{"replicas":1,"selector":{"matchLabels":{"app":"ml-pipeline"}},"template":{"spec":{"containers":[{"name":"ml-pipeline-api-server","image":"test/new"}]}}},"status":{"observedGeneration":1,"replicas":1,"updatedReplicas":1}}`))
					} else {
						_, _ = w.Write([]byte(`{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"name":"ml-pipeline","generation":1},"spec":{"replicas":0,"selector":{"matchLabels":{"app":"ml-pipeline"}},"template":{"spec":{"containers":[{"name":"ml-pipeline-api-server","image":"test/old"}]}}},"status":{"observedGeneration":1,"replicas":0,"updatedReplicas":0}}`))
					}
				case "/api/v1/namespaces/kubeflow/pods":
					if r.URL.Query().Get("labelSelector") == "app=ml-pipeline-scheduledworkflow" {
						_, _ = w.Write([]byte(`{"apiVersion":"v1","kind":"PodList","items":[]}`))
						return
					}
					if apiPodPresent {
						_, _ = w.Write([]byte(`{"apiVersion":"v1","kind":"PodList","items":[{"metadata":{"name":"new-api"},"spec":{"containers":[{"name":"ml-pipeline-api-server","image":"test/new"}]},"status":{"containerStatuses":[{"name":"ml-pipeline-api-server","imageID":"sha256:new"}]}}]}`))
					} else {
						_, _ = w.Write([]byte(`{"apiVersion":"v1","kind":"PodList","items":[]}`))
					}
				case "/apis/argoproj.io/v1alpha1/workflows":
					_ = json.NewEncoder(w).Encode(map[string]any{"apiVersion": "argoproj.io/v1alpha1", "kind": "WorkflowList", "items": workflows})
				default:
					http.NotFound(w, r)
				}
			}))
			t.Cleanup(kube.Close)
			kubeconfig := writeMigrationKubeconfig(t, kube.URL)
			config, err := json.Marshal(settings)
			require.NoError(t, err)
			host, port, err := net.SplitHostPort(listener.Addr().String())
			require.NoError(t, err)
			start := func(withMLMD bool) {
				ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
				defer cancel()
				command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestMLMDLegacyStartup$")
				command.Env = append(os.Environ(), "KFP_MLMD_STARTUP_CHILD_CONFIG="+string(config), "KUBECONFIG="+kubeconfig, "POD_NAME=new-api")
				if withMLMD {
					command.Env = append(command.Env, "METADATA_GRPC_SERVICE_HOST="+host, "METADATA_GRPC_SERVICE_PORT="+port)
				} else {
					command.Env = append(command.Env, "METADATA_GRPC_SERVICE_HOST=", "METADATA_GRPC_SERVICE_PORT=")
				}
				output, err := command.CombinedOutput()
				require.NoError(t, ctx.Err(), string(output))
				require.NoError(t, err, string(output))
			}
			if mode == "CLI" {
				binary := filepath.Join(t.TempDir(), "mlmd-migrate")
				build := exec.Command("go", "build", "-o", binary, "github.com/kubeflow/pipelines/backend/src/apiserver/migration/mlmd/cmd/mlmd-migrate")
				output, err := build.CombinedOutput()
				require.NoError(t, err, string(output))
				command := exec.Command(binary, "--mlmd-address="+listener.Addr().String())
				mysqlConfig, err := mysqlDriver.ParseDSN(dsn)
				require.NoError(t, err)
				mysqlConfig.DBName = settings[mysqlDBName].(string)
				command.Env = append(os.Environ(), "KFP_MLMD_MYSQL_DSN="+mysqlConfig.FormatDSN(), "KUBECONFIG="+kubeconfig)
				output, err = command.CombinedOutput()
				require.NoError(t, err, string(output))
				apiPodPresent = true
				start(false)
			} else {
				start(true)
			}
			required, err := mlmd.NeedsMigration(context.Background(), db)
			require.NoError(t, err)
			require.False(t, required)
			var tasks, artifacts, links int
			require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&tasks))
			require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM artifacts").Scan(&artifacts))
			require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM artifact_tasks").Scan(&links))
			require.GreaterOrEqual(t, tasks, 10)
			require.Positive(t, artifacts)
			require.Positive(t, links)
			var transferredUUID string
			require.NoError(t, db.QueryRow("SELECT UUID FROM transfer_identities WHERE `Key`='installation'").Scan(&transferredUUID))
			require.Equal(t, installationUUID, transferredUUID)
			start(false)
		})
	}
}

func TestLegacyTransferIdentityResume(t *testing.T) {
	dsn := os.Getenv("KFP_MLMD_MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("set KFP_MLMD_MYSQL_TEST_DSN to a disposable loopback MySQL service")
	}
	db, _ := legacyStartupDatabase(t, dsn)
	_, err := db.Exec("CREATE TABLE transfer_identities (ID tinyint unsigned NOT NULL PRIMARY KEY, UUID varchar(36) NOT NULL) ENGINE=InnoDB")
	require.NoError(t, err)
	const identity = "11111111-2222-4333-8444-555555555555"
	_, err = db.Exec("INSERT INTO transfer_identities (ID,UUID) VALUES (1,?)", identity)
	require.NoError(t, err)
	_, err = db.Exec("CREATE TABLE transfer_identities_migration_tmp (`Key` varchar(32) NOT NULL PRIMARY KEY, UUID varchar(64) NOT NULL) ENGINE=InnoDB")
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO transfer_identities_migration_tmp (`Key`,UUID) VALUES ('installation',?)", identity)
	require.NoError(t, err)
	require.NoError(t, reconcileLegacyTransferIdentity(context.Background(), db))
	_, err = db.Exec("CREATE TABLE transfer_identities_pre_migration (ID tinyint unsigned NOT NULL PRIMARY KEY, UUID varchar(36) NOT NULL) ENGINE=InnoDB")
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO transfer_identities_pre_migration (ID,UUID) VALUES (1,?)", identity)
	require.NoError(t, err)
	require.NoError(t, reconcileLegacyTransferIdentity(context.Background(), db))
	require.NoError(t, reconcileLegacyTransferIdentity(context.Background(), db))
	var actual string
	require.NoError(t, db.QueryRow("SELECT UUID FROM transfer_identities WHERE `Key`='installation'").Scan(&actual))
	require.Equal(t, identity, actual)
	for _, table := range []string{"transfer_identities_migration_tmp", "transfer_identities_pre_migration"} {
		present, err := transferTableExists(context.Background(), db, table)
		require.NoError(t, err)
		require.False(t, present)
	}
}

func TestLegacyTransferIdentityConcurrentReplicas(t *testing.T) {
	dsn := os.Getenv("KFP_MLMD_MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("set KFP_MLMD_MYSQL_TEST_DSN to a disposable loopback MySQL service")
	}
	db, _ := legacyStartupDatabase(t, dsn)
	_, err := db.Exec("CREATE TABLE transfer_identities (ID tinyint unsigned NOT NULL PRIMARY KEY, UUID varchar(36) NOT NULL) ENGINE=InnoDB")
	require.NoError(t, err)
	const identity = "11111111-2222-4333-8444-555555555555"
	_, err = db.Exec("INSERT INTO transfer_identities (ID,UUID) VALUES (1,?)", identity)
	require.NoError(t, err)
	results := make(chan error, 2)
	for range 2 {
		go func() { results <- reconcileLegacyTransferIdentity(context.Background(), db) }()
	}
	for range 2 {
		require.NoError(t, <-results)
	}
	var actual string
	require.NoError(t, db.QueryRow("SELECT UUID FROM transfer_identities WHERE `Key`='installation'").Scan(&actual))
	require.Equal(t, identity, actual)
	backup, err := transferTableExists(context.Background(), db, "transfer_identities_pre_migration")
	require.NoError(t, err)
	require.False(t, backup)
}
