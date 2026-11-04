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
	"net"
	"net/url"
	"os"
	"strings"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/kubeflow/pipelines/backend/src/apiserver/migration/mlmd"
	"github.com/stretchr/testify/require"
)

func TestLegacyPostgreSQLIsRejectedWithoutChanges(t *testing.T) {
	dsn := os.Getenv("KFP_MLMD_POSTGRES_TEST_DSN")
	if dsn == "" {
		t.Skip("set KFP_MLMD_POSTGRES_TEST_DSN to a disposable loopback PostgreSQL service")
	}
	address, err := url.Parse(dsn)
	require.NoError(t, err)
	require.Equal(t, "postgres", address.Scheme)
	host := address.Hostname()
	ip := net.ParseIP(host)
	require.True(t, host == "localhost" || ip != nil && ip.IsLoopback())
	admin, err := sql.Open("pgx", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, admin.Close()) })
	name := "kfp_mlmd_pg_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	_, err = admin.Exec("CREATE DATABASE " + name)
	require.NoError(t, err)
	t.Cleanup(func() { _, err := admin.Exec("DROP DATABASE " + name); require.NoError(t, err) })
	address.Path = "/" + name
	childURL := address.String()
	postgres, err := sql.Open("pgx", childURL)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, postgres.Close()) })
	_, err = postgres.Exec(`CREATE TABLE tasks ("UUID" text PRIMARY KEY, "MLMDExecutionID" text NOT NULL)`)
	require.NoError(t, err)
	_, err = postgres.Exec(`INSERT INTO tasks ("UUID", "MLMDExecutionID") VALUES ('old-task', '7')`)
	require.NoError(t, err)
	mysqlConnection, err := sql.Open("mysql", childURL)
	if err == nil {
		defer mysqlConnection.Close()
		_, err = mlmd.Migrate(context.Background(), sourceClient(t), mysqlConnection)
	}
	require.Error(t, err)
	var count int
	require.NoError(t, postgres.QueryRow(`SELECT COUNT(*) FROM information_schema.columns WHERE table_schema='public' AND table_name='tasks' AND column_name='MLMDExecutionID'`).Scan(&count))
	require.Equal(t, 1, count)
	require.NoError(t, postgres.QueryRow(`SELECT COUNT(*) FROM tasks`).Scan(&count))
	require.Equal(t, 1, count)
}
