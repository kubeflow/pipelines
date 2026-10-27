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
	"time"

	sq "github.com/Masterminds/squirrel"
	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func leaseSchedulingRun(t *testing.T, tasks *TaskStore, id, phase string, created int64, retry int, next *int64) {
	t.Helper()
	executor := "kubernetes"
	_, err := NewRunStore(tasks.db, tasks.time, tasks.dbDialect).CreateRun(&model.Run{
		UUID: id, RunDetails: model.RunDetails{ExecutorType: &executor, State: model.RuntimeStateRunning, CreatedAtInSec: created},
	})
	require.NoError(t, err)
	if phase != "" {
		task, err := tasks.CreateTask(&model.Task{
			RunUUID: id, Name: "workload", State: model.TaskStatus(apiv2.PipelineTask_RUNNING),
			Type: model.TaskType(apiv2.PipelineTask_RUNTIME), Pods: model.JSONSlice{}, TypeAttrs: model.JSONData{},
		})
		require.NoError(t, err)
		row := &model.RuntimeTask{TaskUUID: task.UUID, RunUUID: id, State: phase, RetryCount: retry, NextEligibleAt: next}
		if phase != model.RuntimeTaskWaiting && phase != model.RuntimeTaskReady {
			attempt := "attempt-" + id
			row.AttemptUUID = &attempt
		}
		require.NoError(t, tasks.RuntimeStore().Create(row))
	}
	for _, taskType := range []apiv2.PipelineTask_TaskType{
		apiv2.PipelineTask_ROOT, apiv2.PipelineTask_DAG, apiv2.PipelineTask_LOOP, apiv2.PipelineTask_RUNTIME,
	} {
		_, err := tasks.RuntimeStore().CreateTask(&model.Task{
			RunUUID: id, Name: "waiting-" + taskType.String(), ScopePath: "root." + taskType.String(),
			Type: model.TaskType(taskType), State: model.TaskStatus(apiv2.PipelineTask_RUNNING),
			Pods: model.JSONSlice{}, TypeAttrs: model.JSONData{},
		})
		require.NoError(t, err)
	}
}

func assertWaitingRowsRespectWorkloadBackoff(t *testing.T, tasks *TaskStore) {
	t.Helper()
	var seconds int64
	require.NoError(t, tasks.db.QueryRow("SELECT "+leaseDatabaseNow(tasks.dbDialect)).Scan(&seconds))
	seconds /= 1000
	future := seconds + 3600
	for i := range 8 {
		phase := model.RuntimeTaskBackoff
		if i%2 == 0 {
			phase = model.RuntimeTaskInfrastructureBackoff
		}
		leaseSchedulingRun(t, tasks, fmt.Sprintf("older-backoff-%d", i), phase, int64(i+1), 4, &future)
	}
	leaseSchedulingRun(t, tasks, "newer-ready", model.RuntimeTaskReady, 100, 0, nil)
	store := tasks.RunLeaseStore()
	for _, preferRetry := range []bool{false, true} {
		ids, err := store.ListEligible("kubernetes", 1, preferRetry)
		require.NoError(t, err)
		assert.Equal(t, []string{"newer-ready"}, ids, "WAITING ancestors and blocked tasks must not fill a bounded discovery batch")
	}
	leaseSchedulingRun(t, tasks, "cancel-backoff", model.RuntimeTaskInfrastructureBackoff, 200, 1, &future)
	q := tasks.dbDialect.QuoteIdentifier
	query, args, err := tasks.dbDialect.QueryBuilder().Update(q("run_details")).
		SetMap(sq.Eq{q("State"): model.RuntimeStateCancelling}).Where(sq.Eq{q("UUID"): "cancel-backoff"}).ToSql()
	require.NoError(t, err)
	_, err = tasks.db.Exec(query, args...)
	require.NoError(t, err)
	leaseSchedulingRun(t, tasks, "recover-inflight", model.RuntimeTaskSubmitting, 300, 0, &future)
	ids, err := store.ListEligible("kubernetes", 3, true)
	require.NoError(t, err)
	assert.Equal(t, []string{"cancel-backoff", "recover-inflight", "newer-ready"}, ids)
	_, err = store.Acquire("cancel-backoff", "cleanup", time.Minute, true)
	require.NoError(t, err)
	_, err = store.Acquire("recover-inflight", "recovery", time.Minute, false)
	require.NoError(t, err)
	ids, err = store.ListEligible("kubernetes", 1, false)
	require.NoError(t, err)
	assert.Equal(t, []string{"newer-ready"}, ids, "owned cancellation/recovery runs must not consume discovery slots")
	leaseSchedulingRun(t, tasks, "waiting-only-initialization", "", 400, 0, nil)
	ids, err = store.ListEligible("kubernetes", 3, false)
	require.NoError(t, err)
	assert.Equal(t, []string{"newer-ready", "waiting-only-initialization"}, ids, "WAITING-only runs still need dependency/framework evaluation")
}

func assertOnlyEligibleInfrastructureBackoffGetsRetryPriority(t *testing.T, tasks *TaskStore) {
	t.Helper()
	var seconds int64
	require.NoError(t, tasks.db.QueryRow("SELECT "+leaseDatabaseNow(tasks.dbDialect)).Scan(&seconds))
	seconds /= 1000
	past, future := seconds-1, seconds+3600
	leaseSchedulingRun(t, tasks, "user-retry", model.RuntimeTaskBackoff, 1, 99, &past)
	leaseSchedulingRun(t, tasks, "ready-after-retry", model.RuntimeTaskReady, 2, 50, nil)
	leaseSchedulingRun(t, tasks, "infrastructure-retry", model.RuntimeTaskInfrastructureBackoff, 3, 0, &past)
	leaseSchedulingRun(t, tasks, "future-infrastructure", model.RuntimeTaskInfrastructureBackoff, 0, 9, &future)
	ids, err := tasks.RunLeaseStore().ListEligible("kubernetes", 10, true)
	require.NoError(t, err)
	assert.Equal(t, []string{"infrastructure-retry", "user-retry", "ready-after-retry"}, ids)
	ids, err = tasks.RunLeaseStore().ListEligible("kubernetes", 10, false)
	require.NoError(t, err)
	assert.Equal(t, []string{"user-retry", "ready-after-retry", "infrastructure-retry"}, ids)
}

func TestRunLeaseWaitingRowsRespectWorkloadBackoff(t *testing.T) {
	_, tasks, _, _ := runtimeFixture(t)
	assertWaitingRowsRespectWorkloadBackoff(t, tasks)
}

func TestRunLeaseOnlyInfrastructureBackoffGetsRetryPriority(t *testing.T) {
	_, tasks, _, _ := runtimeFixture(t)
	assertOnlyEligibleInfrastructureBackoffGetsRetryPriority(t, tasks)
}

func TestRunLeaseLiveWaitingRowsAndInfrastructurePriority(t *testing.T) {
	for _, driver := range []string{"mysql", "pgx"} {
		for _, scenario := range []string{"waiting", "priority"} {
			t.Run(driver+"/"+scenario, func(t *testing.T) {
				_, _, tasks, _ := runLeaseLiveFixture(t, driver)
				_, err := tasks.RunLeaseStore().Acquire("lease-run", "fixture-owner", time.Minute, false)
				require.NoError(t, err)
				if scenario == "waiting" {
					assertWaitingRowsRespectWorkloadBackoff(t, tasks)
				} else {
					assertOnlyEligibleInfrastructureBackoffGetsRetryPriority(t, tasks)
				}
			})
		}
	}
}
