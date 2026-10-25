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
	"math"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/common/util"
	swfapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func reportedJob(generation int64, enabled bool) *util.ScheduledWorkflow {
	return util.NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		TypeMeta:   metav1.TypeMeta{APIVersion: "kubeflow.org/v1beta1", Kind: "ScheduledWorkflow"},
		ObjectMeta: metav1.ObjectMeta{UID: "1", Name: "pp1", Namespace: "n1", Generation: generation},
		Spec:       swfapi.ScheduledWorkflowSpec{Enabled: enabled},
	})
}

func TestRecurringRunReportGenerationOrdering(t *testing.T) {
	for _, enabled := range []bool{true, false} {
		t.Run(fmt.Sprintf("enabled-%v", enabled), func(t *testing.T) {
			db, _, store := initializeDBAndStore()
			defer db.Close()
			require.NoError(t, store.ChangeJobModeAtGeneration("1", enabled, 2))
			current, err := store.GetJob("1")
			require.NoError(t, err)
			// Reversed API completions and reports are both fenced, including
			// a mode request that did not change the existing SQL boolean.
			require.NoError(t, store.ChangeJobModeAtGeneration("1", !enabled, 1))
			stale := reportedJob(1, !enabled)
			stale.Name = "stale-name"
			stale.Spec.MaxConcurrency = util.Int64Pointer(99)
			require.NoError(t, store.UpdateJob(stale))
			stored, err := store.GetJob("1")
			require.NoError(t, err)
			require.Equal(t, current, stored)

			// A same-generation report can fill in spec/status that the API's
			// mode-only update did not write; newer standalone edits remain valid.
			same := reportedJob(2, enabled)
			same.Spec.MaxConcurrency = util.Int64Pointer(5)
			same.Spec.Workflow = &swfapi.WorkflowResource{Parameters: []swfapi.Parameter{{Name: "input", Value: "new"}}}
			same.Spec.Trigger.PeriodicSchedule = &swfapi.PeriodicSchedule{IntervalSecond: 20}
			require.NoError(t, store.UpdateJob(same))
			stored, err = store.GetJob("1")
			require.NoError(t, err)
			require.Equal(t, enabled, stored.Enabled)
			require.Equal(t, int64(5), stored.MaxConcurrency)
			require.Equal(t, int64(20), *stored.IntervalSecond)
			require.Contains(t, stored.Parameters, "new")
			require.NoError(t, store.UpdateJob(reportedJob(3, !enabled)))
			// Missing generation on a legacy report or caller cannot reset a fence.
			require.NoError(t, store.UpdateJob(reportedJob(0, enabled)))
			require.NoError(t, store.ChangeJobMode("1", enabled))
			stored, err = store.GetJob("1")
			require.NoError(t, err)
			require.Equal(t, !enabled, stored.Enabled)
		})
	}
}

func TestRecurringRunReportDeletedResource(t *testing.T) {
	db, _, store := initializeDBAndStore()
	defer db.Close()
	require.Error(t, store.ChangeJobModeAtGeneration("1", true, DeletedScheduledWorkflowGeneration))
	require.Error(t, store.UpdateJob(reportedJob(-1, false)))
	require.NoError(t, store.ChangeJobModeAtGeneration("1", false, DeletedScheduledWorkflowGeneration))
	require.NoError(t, store.UpdateJob(reportedJob(math.MaxInt64, true)))
	require.NoError(t, store.ChangeJobModeAtGeneration("1", true, math.MaxInt64))
	stored, err := store.GetJob("1")
	require.NoError(t, err)
	require.False(t, stored.Enabled)
}

func TestRecurringRunReportTransactionRollback(t *testing.T) {
	db, _, store := initializeDBAndStore()
	defer db.Close()
	_, err := db.Exec(`CREATE TRIGGER fail_job_update BEFORE UPDATE ON jobs BEGIN SELECT RAISE(ABORT, 'simulated job failure'); END`)
	require.NoError(t, err)
	require.ErrorContains(t, store.ChangeJobModeAtGeneration("1", false, 7), "simulated job failure")
	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM recurring_run_report_states`).Scan(&count))
	require.Zero(t, count, "a failed job write must roll back its generation fence")
	_, err = db.Exec(`DROP TRIGGER fail_job_update`)
	require.NoError(t, err)
	require.NoError(t, store.UpdateJob(reportedJob(6, false)))
	stored, err := store.GetJob("1")
	require.NoError(t, err)
	require.False(t, stored.Enabled)
}

func TestRecurringRunReportStateDeletion(t *testing.T) {
	db, _, store := initializeDBAndStore()
	defer db.Close()
	require.NoError(t, store.ChangeJobModeAtGeneration("1", false, 2))
	require.NoError(t, store.DeleteJob("1"))
	require.Error(t, store.UpdateJob(reportedJob(3, true)))
	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM recurring_run_report_states WHERE JobUUID = '1'`).Scan(&count))
	require.Zero(t, count, "deletion and late reports must not leave orphan fences")
}
