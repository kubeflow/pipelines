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
	"errors"
	"fmt"
	"testing"
	"time"

	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunLeaseLifecycleUsesDatabaseClock(t *testing.T) {
	db, tasks, _, task := runtimeFixture(t)
	tasks.time = util.NewFakeTime(time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC))
	_, err := db.Exec("UPDATE run_details SET ExecutorType = 'kubernetes' WHERE UUID = ?", task.RunUUID)
	require.NoError(t, err)
	store := tasks.RunLeaseStore()
	token, err := store.Acquire(task.RunUUID, "owner", time.Minute, false)
	require.NoError(t, err)
	require.EqualValues(t, 1, token.Generation)
	require.NoError(t, store.Check(*token))
	require.NoError(t, store.Renew(*token, time.Minute))
	_, err = store.Acquire(task.RunUUID, "owner", time.Minute, false)
	require.ErrorIs(t, err, ErrRunLeaseConflict, "a live owner must renew, not reacquire")
	var remaining int64
	require.NoError(t, db.QueryRow("SELECT ExpiresAt - "+leaseDatabaseNow(tasks.dbDialect)+" FROM run_leases WHERE RunUUID = ?", task.RunUUID).Scan(&remaining))
	assert.InDelta(t, time.Minute.Milliseconds(), remaining, 1000)
	_, err = db.Exec("UPDATE run_details SET State = 'SUCCEEDED' WHERE UUID = ?", task.RunUUID)
	require.NoError(t, err)
	require.NoError(t, store.Check(*token), "an owned lease must remain usable for terminal cleanup")
	require.NoError(t, store.Renew(*token, time.Minute))
	_, err = store.Acquire(task.RunUUID, "other", time.Minute, false)
	require.ErrorIs(t, err, ErrRunLeaseIneligible)
	require.NoError(t, store.Release(*token))
	var generation int64
	var owner *string
	var expires *int64
	require.NoError(t, db.QueryRow("SELECT Generation, OwnerID, ExpiresAt FROM run_leases WHERE RunUUID = ?", task.RunUUID).Scan(&generation, &owner, &expires))
	assert.Equal(t, token.Generation, generation)
	assert.Nil(t, owner)
	assert.Nil(t, expires)
	cleanup, err := store.Acquire(task.RunUUID, "other", time.Minute, true)
	require.NoError(t, err)
	assert.Greater(t, cleanup.Generation, token.Generation)
	require.ErrorIs(t, store.Renew(*token, time.Minute), ErrRunLeaseConflict)
	require.ErrorIs(t, store.Release(*token), ErrRunLeaseConflict)
	scoped, err := tasks.LeaseScopedStores(*cleanup)
	require.NoError(t, err)
	require.NoError(t, scoped.Runs.DeleteRun(task.RunUUID))
	require.ErrorIs(t, store.Check(*cleanup), ErrRunLeaseConflict)
	var count int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM run_leases").Scan(&count))
	assert.Zero(t, count, "run deletion must cascade ownership metadata")
}

func TestRunLeaseExpiredCannotRenewAndAcquisitionIsExclusive(t *testing.T) {
	db, tasks, _, task := runtimeFixture(t)
	_, err := db.Exec("UPDATE run_details SET ExecutorType = 'kubernetes' WHERE UUID = ?", task.RunUUID)
	require.NoError(t, err)
	store := tasks.RunLeaseStore()
	const contenders = 16
	start := make(chan struct{})
	type result struct {
		token *RunLeaseToken
		err   error
	}
	results := make(chan result, contenders)
	for i := range contenders {
		go func() {
			<-start
			token, err := store.Acquire(task.RunUUID, fmt.Sprintf("worker-%d", i), time.Minute, false)
			results <- result{token, err}
		}()
	}
	close(start)
	var winner *RunLeaseToken
	for range contenders {
		result := <-results
		if result.err == nil {
			require.Nil(t, winner)
			winner = result.token
		} else {
			require.ErrorIs(t, result.err, ErrRunLeaseConflict)
		}
	}
	require.NotNil(t, winner)
	_, err = db.Exec("UPDATE run_leases SET ExpiresAt = "+leaseDatabaseNow(tasks.dbDialect)+" - 1 WHERE RunUUID = ?", task.RunUUID)
	require.NoError(t, err)
	require.ErrorIs(t, store.Check(*winner), ErrRunLeaseConflict)
	require.ErrorIs(t, store.Renew(*winner, time.Hour), ErrRunLeaseConflict, "renewal must never resurrect an expired owner")
	require.ErrorIs(t, store.Release(*winner), ErrRunLeaseConflict)
	next, err := store.Acquire(task.RunUUID, "takeover", time.Minute, false)
	require.NoError(t, err)
	require.Greater(t, next.Generation, winner.Generation)
	_, err = store.Acquire("2", "unmanaged", time.Minute, true)
	require.ErrorIs(t, err, ErrRunLeaseIneligible)
	_, err = store.Acquire("missing", "owner", time.Minute, true)
	require.ErrorIs(t, err, ErrRunLeaseIneligible)
	require.NoError(t, db.Close())
	err = store.Check(*next)
	require.Error(t, err)
	assert.False(t, errors.Is(err, ErrRunLeaseConflict), "database errors must not be converted into expected conflicts")
}

func TestRunLeaseBoundedDiscoveryRecoveryRetriesAndBackoff(t *testing.T) {
	db, tasks, _, _ := runtimeFixture(t)
	runs := NewRunStore(db, util.NewRealTime(), tasks.dbDialect)
	tasks = NewTaskStore(db, util.NewRealTime(), util.NewUUIDGenerator(), tasks.dbDialect)
	executor := "kubernetes"
	var seconds int64
	require.NoError(t, db.QueryRow("SELECT "+leaseDatabaseNow(tasks.dbDialect)+" / 1000").Scan(&seconds))
	for _, candidate := range []struct {
		id      string
		created int64
		phase   string
		retry   int
		next    *int64
	}{
		{"fresh", 1, "", 0, nil},
		{"active", 10, model.RuntimeTaskRunning, 0, nil},
		{"canceling", 20, model.RuntimeTaskBackoff, 0, func() *int64 { value := seconds + 3600; return &value }()},
		{"retry", 30, model.RuntimeTaskInfrastructureBackoff, 1, &seconds},
		{"future", 40, model.RuntimeTaskBackoff, 1, func() *int64 { value := seconds + 3600; return &value }()},
		{"owned", 50, "", 0, nil},
	} {
		_, err := runs.CreateRun(&model.Run{UUID: candidate.id, RunDetails: model.RunDetails{
			State: model.RuntimeStateRunning, ExecutorType: &executor, CreatedAtInSec: candidate.created,
		}})
		require.NoError(t, err)
		if candidate.phase != "" {
			task, err := tasks.CreateTask(&model.Task{RunUUID: candidate.id, Name: "task", Pods: model.JSONSlice{}, TypeAttrs: model.JSONData{}, State: model.TaskStatus(apiv2.PipelineTask_RUNNING)})
			require.NoError(t, err)
			require.NoError(t, tasks.RuntimeStore().Create(&model.RuntimeTask{
				TaskUUID: task.UUID, RunUUID: candidate.id, State: candidate.phase, RetryCount: candidate.retry, NextEligibleAt: candidate.next,
			}))
		}
	}
	_, err := db.Exec("UPDATE run_details SET State = 'CANCELING' WHERE UUID = 'canceling'")
	require.NoError(t, err)
	store := tasks.RunLeaseStore()
	_, err = store.Acquire("owned", "busy", time.Minute, false)
	require.NoError(t, err)
	ids, err := store.ListEligible(executor, 100, true)
	require.NoError(t, err)
	assert.Equal(t, []string{"canceling", "active", "retry", "fresh"}, ids)
	ids, err = store.ListEligible(executor, 100, false)
	require.NoError(t, err)
	assert.Equal(t, []string{"canceling", "active", "fresh", "retry"}, ids)
	ids, err = store.ListEligible(executor, 2, true)
	require.NoError(t, err)
	assert.Equal(t, []string{"canceling", "active"}, ids)
	_, err = store.ListEligible(executor, 1001, true)
	require.Error(t, err)
	_, err = store.Acquire("canceling", "worker", time.Minute, false)
	require.ErrorIs(t, err, ErrRunLeaseIneligible)
	_, err = store.Acquire("canceling", "worker", time.Minute, true)
	require.NoError(t, err)
}
