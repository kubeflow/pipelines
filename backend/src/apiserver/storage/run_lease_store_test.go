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
	"testing"
	"time"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/require"
)

func TestRunLeaseStaleOwnerCannotUpdateAfterTakeover(t *testing.T) {
	db, tasks, _, task := runtimeFixture(t)
	_, err := db.Exec("UPDATE run_details SET ExecutorType = 'kubernetes' WHERE UUID = ?", task.RunUUID)
	require.NoError(t, err)
	leases := tasks.RunLeaseStore()
	first, err := leases.Acquire(task.RunUUID, "worker-a", time.Minute, false)
	require.NoError(t, err)
	old, err := tasks.LeaseScopedStores(*first)
	require.NoError(t, err)
	require.NoError(t, leases.Release(*first))
	next, err := leases.Acquire(task.RunUUID, "worker-b", time.Minute, false)
	require.NoError(t, err)
	require.Greater(t, next.Generation, first.Generation)
	_, err = old.Tasks.UpdateTask(&model.Task{UUID: task.UUID, StatusMetadata: model.JSONData{"stale-owner": true}})
	require.ErrorIs(t, err, ErrRunLeaseConflict, "an old coordinator must not write after ownership changes")
	current, err := tasks.GetTask(task.UUID)
	require.NoError(t, err)
	require.Empty(t, current.StatusMetadata)
}
