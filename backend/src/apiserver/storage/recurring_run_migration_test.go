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

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/require"
)

func TestListJobsWithoutRecurringRunState(t *testing.T) {
	db, _, store := initializeDBAndStore()
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	candidates, err := store.ListJobsWithoutRecurringRunState("", 1)
	require.NoError(t, err)
	require.Empty(t, candidates)

	// Simulate pre-upgrade rows. Include a disabled schedule that still needs
	// review before an operator enables it, and leave a newly created job alone.
	require.NoError(t, store.ChangeJobMode("2", false))
	_, err = store.CreateJob(&model.Job{UUID: "3", Namespace: "n1", ExperimentId: defaultFakeExpId, Enabled: true})
	require.NoError(t, err)
	_, err = db.Exec(`DELETE FROM recurring_run_states WHERE JobUUID IN (?, ?)`, "1", "2")
	require.NoError(t, err)
	candidates, err = store.ListJobsWithoutRecurringRunState("", 1)
	require.NoError(t, err)
	require.Equal(t, []RecurringRunMigrationCandidate{{ID: "1", Namespace: "n1", Name: "pp1", Enabled: true}}, candidates)
	candidates, err = store.ListJobsWithoutRecurringRunState(candidates[0].ID, 1)
	require.NoError(t, err)
	require.Equal(t, []RecurringRunMigrationCandidate{{ID: "2", Namespace: "n1", Name: "pp2", Enabled: false}}, candidates)
	candidates, err = store.ListJobsWithoutRecurringRunState(candidates[0].ID, 1)
	require.NoError(t, err)
	require.Empty(t, candidates)

	// Inventory must never seed trusted state from legacy records.
	_, err = store.GetRecurringRunState("1")
	require.ErrorContains(t, err, "complete recurring-run adoption in KFP 2.18 before upgrading")
	_, err = store.ListJobsWithoutRecurringRunState("", 0)
	require.Error(t, err)
	_, err = store.ListJobsWithoutRecurringRunState("", 1001)
	require.Error(t, err)
	_, err = db.Exec(`DROP TABLE recurring_run_states`)
	require.NoError(t, err)
	_, err = store.ListJobsWithoutRecurringRunState("", 100)
	require.ErrorContains(t, err, "migration inventory")
}
