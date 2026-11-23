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

package storage

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/require"
)

// requireConnectionAvailable fails if the pool's only connection is still held,
// for example by an unclosed rows or an unfinished transaction.
func requireConnectionAvailable(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var one int
	require.NoError(t, db.QueryRowContext(ctx, "SELECT 1").Scan(&one),
		"a failed list call must release its connection")
}

// #13954, #13916: list calls that fail while scanning rows must still close the
// rows and roll back the transaction, otherwise each failure leaks a connection.
func TestListJobsReleasesConnectionOnScanError(t *testing.T) {
	db, _, jobStore := initializeDBAndStore()
	defer db.Close()
	db.SetMaxOpenConns(1)
	_, err := db.Exec(`UPDATE jobs SET MaxConcurrency = 'not-a-number'`)
	require.NoError(t, err)
	opts, err := list.NewOptions(&model.Job{}, 10, "", nil)
	require.NoError(t, err)

	_, _, _, err = jobStore.ListJobs(&model.FilterContext{}, opts)

	require.Error(t, err)
	requireConnectionAvailable(t, db)
}

func TestListRunsReleasesConnectionOnScanError(t *testing.T) {
	db, _, runStore := initializeRunStore()
	defer db.Close()
	db.SetMaxOpenConns(1)
	_, err := db.Exec(`UPDATE run_details SET CreatedAtInSec = 'not-a-number'`)
	require.NoError(t, err)
	opts, err := list.NewOptions(&model.Run{}, 10, "", nil)
	require.NoError(t, err)

	_, _, _, err = runStore.ListRuns(&model.FilterContext{}, opts, false)

	require.Error(t, err)
	requireConnectionAvailable(t, db)
}
