// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package storage

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	swfapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func guardedReport() *util.ScheduledWorkflow {
	return util.NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		TypeMeta:   metav1.TypeMeta{APIVersion: "kubeflow.org/v2beta1", Kind: "ScheduledWorkflow"},
		ObjectMeta: metav1.ObjectMeta{UID: "1", Name: "pp1", Namespace: "n1", Generation: 2},
		Spec:       swfapi.ScheduledWorkflowSpec{Enabled: true},
	})
}

func TestUpdateJobWithGuard_StaleReportPreservesDisabledMode(t *testing.T) {
	db, _, store := initializeDBAndStore()
	defer db.Close()
	require.NoError(t, store.ChangeJobMode("1", false))
	guardCalled := false
	guard := func(context.Context) (bool, error) { guardCalled = true; return false, nil }
	err := store.UpdateJobWithGuard(context.Background(), guardedReport(), guard)
	require.NoError(t, err)
	job, err := store.GetJob("1")
	require.NoError(t, err)
	require.False(t, job.Enabled, "an old ENABLE report must not undo a completed Disable")
	require.True(t, guardCalled)
}

func TestUpdateJobWithGuard_CurrentReportPreservesStandaloneEdits(t *testing.T) {
	db, _, store := initializeDBAndStore()
	defer db.Close()
	swf := guardedReport()
	swf.Spec.Enabled = false
	swf.Spec.MaxConcurrency = util.Int64Pointer(17)
	swf.Spec.NoCatchup = util.BoolPointer(true)
	require.NoError(t, store.UpdateJobWithGuard(context.Background(), swf, func(context.Context) (bool, error) { return true, nil }))
	job, err := store.GetJob("1")
	require.NoError(t, err)
	require.False(t, job.Enabled)
	require.Equal(t, int64(17), job.MaxConcurrency)
	require.True(t, job.NoCatchup)
}

func TestUpdateJobWithGuard_GuardFailureAndMissingJob(t *testing.T) {
	db, _, store := initializeDBAndStore()
	defer db.Close()
	swf := guardedReport()
	swf.Spec.Enabled = false
	err := store.UpdateJobWithGuard(context.Background(), swf, func(context.Context) (bool, error) { return false, errors.New("Kubernetes unavailable") })
	require.EqualError(t, err, "Kubernetes unavailable")
	job, err := store.GetJob("1")
	require.NoError(t, err)
	require.True(t, job.Enabled)
	swf.UID = "missing"
	called := false
	require.Error(t, store.UpdateJobWithGuard(context.Background(), swf, func(context.Context) (bool, error) { called = true; return true, nil }))
	require.False(t, called)
	require.Error(t, store.UpdateJobWithGuard(context.Background(), guardedReport(), nil))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.Error(t, store.UpdateJobWithGuard(ctx, guardedReport(), func(context.Context) (bool, error) { called = true; return true, nil }))
	require.False(t, called)
}

// This driver verifies transaction ordering and production SQL dialects. It does
// not simulate the database engine's row locks; those require engine integration.
type jobGuardConnector struct {
	events    []string
	lockQuery string
	missing   bool
	queryErr  error
	execErr   error
	commitErr error
	ctx       context.Context
}

func (c *jobGuardConnector) Connect(context.Context) (driver.Conn, error) {
	return &jobGuardConn{c}, nil
}
func (c *jobGuardConnector) Driver() driver.Driver { return nil }

type jobGuardConn struct{ state *jobGuardConnector }

func (c *jobGuardConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("unexpected Prepare")
}
func (c *jobGuardConn) Close() error              { return nil }
func (c *jobGuardConn) Begin() (driver.Tx, error) { return nil, errors.New("use BeginTx") }
func (c *jobGuardConn) BeginTx(ctx context.Context, _ driver.TxOptions) (driver.Tx, error) {
	c.state.events = append(c.state.events, "begin")
	c.state.ctx = ctx
	return c, nil
}
func (c *jobGuardConn) Commit() error {
	c.state.events = append(c.state.events, "commit")
	return c.state.commitErr
}
func (c *jobGuardConn) Rollback() error {
	c.state.events = append(c.state.events, "rollback")
	return nil
}
func (c *jobGuardConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if ctx != c.state.ctx {
		return nil, errors.New("lock lost caller context")
	}
	if len(args) != 1 || args[0].Value != "1" {
		return nil, errors.New("wrong job lock")
	}
	c.state.events = append(c.state.events, "lock")
	c.state.lockQuery = query
	if c.state.queryErr != nil {
		return nil, c.state.queryErr
	}
	rows := &errStreamRows{cols: []string{"UUID"}}
	if !c.state.missing {
		rows.firstRow = []driver.Value{"1"}
	}
	return rows, nil
}
func (c *jobGuardConn) ExecContext(ctx context.Context, query string, _ []driver.NamedValue) (driver.Result, error) {
	if ctx != c.state.ctx {
		return nil, errors.New("write lost caller context")
	}
	if !strings.HasPrefix(query, "UPDATE ") {
		return nil, errors.New("unexpected write")
	}
	c.state.events = append(c.state.events, "update")
	if c.state.execErr != nil {
		return nil, c.state.execErr
	}
	return driver.RowsAffected(1), nil
}

func TestUpdateJobWithGuard_ProductionDialectLockOrdering(t *testing.T) {
	for _, engine := range []string{"mysql", "pgx"} {
		t.Run(engine, func(t *testing.T) {
			for _, scenario := range []string{"current", "stale", "guard-error", "missing", "lock-error", "write-error", "commit-error"} {
				t.Run(scenario, func(t *testing.T) {
					state := &jobGuardConnector{}
					switch scenario {
					case "missing":
						state.missing = true
					case "lock-error":
						state.queryErr = errors.New("lock unavailable")
					case "write-error":
						state.execErr = errors.New("write unavailable")
					case "commit-error":
						state.commitErr = errors.New("commit unavailable")
					}
					db := sql.OpenDB(state)
					defer db.Close()
					store := NewJobStore(db, util.NewFakeTimeForEpoch(), nil, dialect.NewDBDialect(engine))
					ctx, cancel := context.WithTimeout(context.Background(), time.Second)
					defer cancel()
					called := false
					err := store.UpdateJobWithGuard(ctx, guardedReport(), func(got context.Context) (bool, error) {
						called = true
						require.Equal(t, []string{"begin", "lock"}, state.events)
						require.Equal(t, ctx, got)
						state.events = append(state.events, "guard")
						if scenario == "guard-error" {
							return false, errors.New("guard unavailable")
						}
						return scenario != "stale", nil
					})
					lock := "SELECT `UUID` FROM `jobs` WHERE `UUID` = ? FOR UPDATE"
					if engine == "pgx" {
						lock = `SELECT "UUID" FROM "jobs" WHERE "UUID" = $1 FOR UPDATE`
					}
					require.Equal(t, lock, state.lockQuery)
					switch scenario {
					case "current":
						require.NoError(t, err)
						require.Equal(t, []string{"begin", "lock", "guard", "update", "commit"}, state.events)
					case "stale":
						require.NoError(t, err)
						require.Equal(t, []string{"begin", "lock", "guard", "rollback"}, state.events)
					case "guard-error":
						require.Error(t, err)
						require.Equal(t, []string{"begin", "lock", "guard", "rollback"}, state.events)
					case "missing", "lock-error":
						require.Error(t, err)
						require.False(t, called)
						require.Equal(t, []string{"begin", "lock", "rollback"}, state.events)
					case "write-error":
						require.Error(t, err)
						require.Equal(t, []string{"begin", "lock", "guard", "update", "rollback"}, state.events)
					case "commit-error":
						require.Error(t, err)
						require.Equal(t, []string{"begin", "lock", "guard", "update", "commit"}, state.events)
					}
				})
			}
		})
	}
}
