// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	workflowapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	_ "github.com/lib/pq"
)

// Opt-in real offload DB validation. Default CI uses MemoryOffloadNodeStatusRepo
// (TestOffloadHydration_EndToEnd_MemoryRepo and resource GC survival tests).
//
//	export KFP_ARGO_OFFLOAD_IT=1
//	GOMODCACHE=$HOME/go/pkg/mod go test ./backend/src/common/util/ \
//	  -count=1 -run TestOffloadHydration_RealDB_OptIn -v
//
// Requires Docker. Without KFP_ARGO_OFFLOAD_IT=1 this test Skip's so CI stays green.
func TestOffloadHydration_RealDB_OptIn(t *testing.T) {
	if os.Getenv("KFP_ARGO_OFFLOAD_IT") != "1" {
		t.Skip("opt-in: set KFP_ARGO_OFFLOAD_IT=1 with Docker to run testcontainers Postgres offload round-trip")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	const (
		dbName   = "argo"
		dbUser   = "argo"
		dbPass   = "argo"
		secretNS = "argo-offload-it"
		userSec  = "pg-user"
		passSec  = "pg-pass"
		table    = "argo_workflows"
	)

	pgContainer, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithDatabase(dbName),
		postgres.WithUsername(dbUser),
		postgres.WithPassword(dbPass),
		testcontainers.WithWaitStrategy(wait.ForListeningPort("5432/tcp")),
	)
	if err != nil {
		t.Skipf("testcontainers Postgres unavailable (is Docker running?): %v", err)
	}
	t.Cleanup(func() {
		_ = pgContainer.Terminate(context.Background())
	})

	host, err := pgContainer.Host(ctx)
	require.NoError(t, err)
	port, err := pgContainer.MappedPort(ctx, "5432/tcp")
	require.NoError(t, err)

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port.Port(), dbUser, dbPass, dbName)
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, db.PingContext(ctx))

	// Final Argo offload node-status schema (after MigrateChanges).
	_, err = db.ExecContext(ctx, fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
  clustername varchar(64) NOT NULL,
  uid varchar(128) NOT NULL,
  namespace varchar(256) NOT NULL,
  version varchar(64) NOT NULL,
  nodes json NOT NULL,
  updatedat timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (clustername, uid, version)
)`, table))
	require.NoError(t, err)

	kube := k8sfake.NewClientset(
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: secretNS}},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: userSec, Namespace: secretNS},
			Data:       map[string][]byte{"username": []byte(dbUser)},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: passSec, Namespace: secretNS},
			Data:       map[string][]byte{"password": []byte(dbPass)},
		},
	)

	persistYAML := fmt.Sprintf(`nodeStatusOffLoad: true
clusterName: kfp-offload-it
postgresql:
  host: %s
  port: %s
  database: %s
  tableName: %s
  userNameSecret:
    name: %s
    key: username
  passwordSecret:
    name: %s
    key: password
`, host, port.Port(), dbName, table, userSec, passSec)
	persist, err := ParseArgoPersistConfig([]byte(persistYAML))
	require.NoError(t, err)

	h, err := CreateWorkflowHydrator(ctx, kube, persist, secretNS)
	require.NoError(t, err)

	wf := &workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "offload-it-wf", UID: "offload-it-uid", Namespace: secretNS},
		Status: workflowapi.WorkflowStatus{
			Nodes: workflowapi.Nodes{
				"n1": {ID: "n1", Name: "offload-it-wf", Phase: workflowapi.NodeSucceeded, Type: workflowapi.NodeTypePod},
			},
		},
	}
	require.NoError(t, h.Dehydrate(ctx, wf))
	require.NotEmpty(t, wf.Status.OffloadNodeStatusVersion)
	require.Empty(t, wf.Status.Nodes)

	require.NoError(t, h.Hydrate(ctx, wf))
	require.Empty(t, wf.Status.OffloadNodeStatusVersion)
	require.Equal(t, workflowapi.NodeSucceeded, wf.Status.Nodes["n1"].Phase)
}
