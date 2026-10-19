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

package storage

import (
	"testing"

	api "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImportedHistoryRunProvenanceRoundTrip(t *testing.T) {
	db, d, store := initializeRunStore()
	defer db.Close()
	run := &model.Run{
		UUID: "imported-run", ExperimentId: defaultFakeExpId, ImportedFrom: "old-installation", ImportDigest: "bundle-sha256",
		RunDetails: model.RunDetails{State: model.RuntimeStateSucceeded},
	}
	_, err := store.CreateRun(run)
	require.NoError(t, err)
	for _, hydrate := range []bool{false, true} {
		got, err := store.GetRun(run.UUID, hydrate)
		require.NoError(t, err)
		assert.Equal(t, run.ImportedFrom, got.ImportedFrom)
		assert.Equal(t, run.ImportDigest, got.ImportDigest)
		runs, _, _, err := store.ListRuns(&model.FilterContext{}, list.EmptyOptions(), hydrate)
		require.NoError(t, err)
		found := false
		for _, got := range runs {
			if got.UUID == run.UUID {
				found = true
				assert.Equal(t, run.ImportedFrom, got.ImportedFrom)
				assert.Equal(t, run.ImportDigest, got.ImportDigest)
			}
		}
		assert.True(t, found)
	}
	q := d.QuoteIdentifier
	_, err = db.Exec("UPDATE " + q("run_details") + " SET " + q("ImportedFrom") + " = NULL, " + q("ImportDigest") + " = NULL WHERE " + q("UUID") + " = '1'")
	require.NoError(t, err)
	legacy, err := store.GetRun("1", false)
	require.NoError(t, err)
	assert.Empty(t, legacy.ImportedFrom)
	assert.Empty(t, legacy.ImportDigest)
}

func TestImportedHistoryTasksAreExcludedFromCache(t *testing.T) {
	db, taskStore, runStore := initializeTaskStore()
	defer db.Close()
	_, err := runStore.CreateRun(&model.Run{
		UUID: "imported-run", ImportedFrom: "old-installation", Namespace: "ns1",
		RunDetails: model.RunDetails{State: model.RuntimeStateSucceeded},
	})
	require.NoError(t, err)
	for i, runID := range []string{"run-1", "imported-run"} {
		taskStore.uuid = util.NewFakeUUIDGeneratorOrFatal([]string{testUUID1, testUUID2}[i], nil)
		_, err := taskStore.CreateTask(&model.Task{
			RunUUID: runID, Namespace: "ns1", Name: "task", Fingerprint: "same-inputs",
			CreatedAtInSec: int64(i + 1), State: model.TaskStatus(api.PipelineTask_SUCCEEDED),
		})
		require.NoError(t, err)
	}
	for _, namespace := range []string{"ns1", ""} {
		cached, err := taskStore.FindLatestCachedTask(namespace, "same-inputs")
		require.NoError(t, err)
		require.NotNil(t, cached)
		assert.Equal(t, "run-1", cached.RunUUID)
	}
	q := taskStore.dbDialect.QuoteIdentifier
	_, err = db.Exec("UPDATE " + q("run_details") + " SET " + q("ImportedFrom") + " = 'old-installation' WHERE " + q("UUID") + " = 'run-1'")
	require.NoError(t, err)
	cached, err := taskStore.FindLatestCachedTask("ns1", "same-inputs")
	require.NoError(t, err)
	assert.Nil(t, cached)
}
