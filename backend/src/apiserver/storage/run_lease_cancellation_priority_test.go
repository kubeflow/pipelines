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
	"fmt"
	"testing"

	sq "github.com/Masterminds/squirrel"
	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertCancellationPrecedesFullRecoveryBatch(t *testing.T, tasks *TaskStore) {
	t.Helper()
	runs := NewRunStore(tasks.db, tasks.time, tasks.dbDialect)
	executor := "kubernetes"
	const limit = 8
	for i := range limit {
		id := fmt.Sprintf("older-active-recovery-%d", i)
		_, err := runs.CreateRun(&model.Run{UUID: id, RunDetails: model.RunDetails{
			ExecutorType: &executor, State: model.RuntimeStateRunning, CreatedAtInSec: int64(i + 1),
		}})
		require.NoError(t, err)
		task, err := tasks.CreateTask(&model.Task{RunUUID: id, Name: "active", State: model.TaskStatus(apiv2.PipelineTask_RUNNING), Pods: model.JSONSlice{}, TypeAttrs: model.JSONData{}})
		require.NoError(t, err)
		require.NoError(t, tasks.RuntimeStore().Create(runtimeAttemptFixture(task, fmt.Sprintf("recovery-attempt-%d", i))))
	}
	const cancellation = "newer-cancellation"
	_, err := runs.CreateRun(&model.Run{UUID: cancellation, RunDetails: model.RunDetails{
		ExecutorType: &executor, State: model.RuntimeStateRunning, CreatedAtInSec: 100,
	}})
	require.NoError(t, err)
	task, err := tasks.CreateTask(&model.Task{RunUUID: cancellation, Name: "cancel-me", State: model.TaskStatus(apiv2.PipelineTask_RUNNING), Pods: model.JSONSlice{}, TypeAttrs: model.JSONData{}})
	require.NoError(t, err)
	require.NoError(t, tasks.RuntimeStore().Create(runtimeAttemptFixture(task, "cancel-attempt")))
	q := tasks.dbDialect.QuoteIdentifier
	query, args, err := tasks.dbDialect.QueryBuilder().Update(q("run_details")).
		SetMap(sq.Eq{q("State"): string(model.RuntimeStateCancelling)}).Where(sq.Eq{q("UUID"): cancellation}).ToSql()
	require.NoError(t, err)
	_, err = tasks.db.Exec(query, args...)
	require.NoError(t, err)
	for _, preferRetry := range []bool{false, true} {
		t.Run(fmt.Sprintf("prefer-retry-%t", preferRetry), func(t *testing.T) {
			ids, err := tasks.RunLeaseStore().ListEligible(executor, limit, preferRetry)
			require.NoError(t, err)
			require.Len(t, ids, limit)
			assert.Equal(t, cancellation, ids[0], "cancellation must rank before recovery before the bounded LIMIT")
			for i := 1; i < limit; i++ {
				assert.Equal(t, fmt.Sprintf("older-active-recovery-%d", i-1), ids[i], "remaining recovery ordering must stay stable")
			}
			one, err := tasks.RunLeaseStore().ListEligible(executor, 1, preferRetry)
			require.NoError(t, err)
			assert.Equal(t, []string{cancellation}, one)
		})
	}
}

func TestRunLeaseCancellationPrecedesFullRecoveryBatch(t *testing.T) {
	_, tasks, _, _ := runtimeFixture(t)
	assertCancellationPrecedesFullRecoveryBatch(t, tasks)
}

func TestRunLeaseLiveCancellationPrecedesFullRecoveryBatch(t *testing.T) {
	for _, driver := range []string{"mysql", "pgx"} {
		t.Run(driver, func(t *testing.T) {
			_, _, tasks, _ := runLeaseLiveFixture(t, driver)
			assertCancellationPrecedesFullRecoveryBatch(t, tasks)
		})
	}
}
