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
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
)

func TestRecurringRunStateLifecycle(t *testing.T) {
	db, d, store := initializeDBAndStore()
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	initial, err := store.GetRecurringRunState("1")
	require.NoError(t, err)
	require.Equal(t, &model.RecurringRunState{JobUUID: "1"}, initial)

	claim, err := store.ClaimRecurringRun("1", "first-tick", 0, 100, 110, "version-a")
	require.NoError(t, err)
	require.Equal(t, &model.RecurringRunState{
		JobUUID: "1", RequestKey: "first-tick", PipelineVersionID: "version-a", LastRunIndex: 1,
		LastScheduledAtInSec: 100, LastCreatedAtInSec: 110, Pending: true,
	}, claim)
	// A retry preserves the first attempt's time even after the clock advances.
	resumed, err := store.ClaimRecurringRun("1", "first-tick", 0, 200, 210, "version-b")
	require.NoError(t, err)
	require.Equal(t, claim, resumed)
	_, err = store.ClaimRecurringRun("1", "different-tick", 1, 200, 210, "")
	require.ErrorContains(t, err, "pending")

	runStore := NewRunStore(db, util.NewFakeTimeForEpoch(), d)
	firstRun := &model.Run{
		UUID: util.NewDeterministicUUID("1/tick/1"), DisplayName: "first-tick", RecurringRunId: "1",
		ExperimentId: defaultFakeExpId, Namespace: "n1", RunDetails: model.RunDetails{State: model.RuntimeStateSucceeded},
	}
	_, err = runStore.CreateRun(firstRun)
	require.NoError(t, err)
	_, err = runStore.CreateRun(firstRun)
	require.NoError(t, err)
	_, err = store.ClaimRecurringRun("1", "first-tick", 1, 200, 210, "")
	require.ErrorContains(t, err, "already completed")
	_, err = store.ClaimRecurringRun("1", "second-tick", 0, 200, 210, "")
	require.ErrorContains(t, err, "scheduling progress changed")
	_, err = store.ClaimRecurringRun("1", "second-tick", 1, 100, 210, "")
	require.ErrorContains(t, err, "scheduled time does not advance")

	second, err := store.ClaimRecurringRun("1", "second-tick", 1, 200, 210, "")
	require.NoError(t, err)
	require.Equal(t, int64(2), second.LastRunIndex)
	// A delayed completion cannot release the newer claim.
	_, err = runStore.CreateRun(firstRun)
	require.NoError(t, err)
	current, err := store.GetRecurringRunState("1")
	require.NoError(t, err)
	require.Equal(t, second, current)
	require.NoError(t, store.DeleteJob("1"))
	_, err = store.GetRecurringRunState("1")
	require.ErrorContains(t, err, "recreate it through the KFP API")
}

func TestRecurringRunStateRequiresEnabledRegisteredJob(t *testing.T) {
	db, _, store := initializeDBAndStore()
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	_, err := store.ClaimRecurringRun("missing", "tick", 0, 100, 110, "")
	require.Error(t, err)
	_, err = store.ClaimRecurringRun("1", "", 0, 100, 110, "")
	require.ErrorContains(t, err, "idempotency key")
	require.NoError(t, store.ChangeJobMode("1", false))
	_, err = store.ClaimRecurringRun("1", "tick", 0, 100, 110, "")
	require.ErrorContains(t, err, "disabled")
	require.NoError(t, store.ChangeJobMode("1", true))
	_, err = db.Exec(`DELETE FROM recurring_run_states WHERE JobUUID = ?`, "1")
	require.NoError(t, err)
	_, err = store.ClaimRecurringRun("1", "tick", 0, 100, 110, "")
	require.ErrorContains(t, err, "recreate it through the KFP API")
	// Missing state is never reconstructed from an editable Kubernetes report.
	_, err = store.GetJob("1")
	require.NoError(t, err)
	_, err = store.GetRecurringRunState("1")
	require.Error(t, err)
}

func TestRecurringRunClaimConcurrency(t *testing.T) {
	for _, sameKey := range []bool{false, true} {
		t.Run(fmt.Sprintf("same-key-%v", sameKey), func(t *testing.T) {
			db, _, store := initializeDBAndStore()
			t.Cleanup(func() { require.NoError(t, db.Close()) })
			// SQLite is test-only and serializes transactions on one connection;
			// production dialects serialize claims with SELECT FOR UPDATE.
			db.SetMaxOpenConns(1)
			const attempts = 12
			var wg sync.WaitGroup
			errors := make(chan error, attempts)
			claims := make(chan *model.RecurringRunState, attempts)
			for i := range attempts {
				wg.Add(1)
				go func() {
					defer wg.Done()
					key := "same-tick"
					if !sameKey {
						key = fmt.Sprintf("tick-%d", i)
					}
					claim, err := store.ClaimRecurringRun("1", key, 0, 100, int64(110+i), "")
					if err != nil {
						errors <- err
					} else {
						claims <- claim
					}
				}()
			}
			wg.Wait()
			close(errors)
			close(claims)
			if sameKey {
				require.Len(t, claims, attempts)
				require.Empty(t, errors)
			} else {
				require.Len(t, claims, 1)
				require.Len(t, errors, attempts-1)
			}
			stored, err := store.GetRecurringRunState("1")
			require.NoError(t, err)
			require.Equal(t, int64(1), stored.LastRunIndex)
			for claim := range claims {
				require.Equal(t, stored, claim)
			}
			for err := range errors {
				require.ErrorContains(t, err, "pending")
			}
		})
	}
}

func TestRecurringRunClaimCountsActiveRuns(t *testing.T) {
	for _, tc := range []struct {
		name       string
		state      any
		conditions string
		active     bool
	}{
		{"pending", "PENDING", "Pending", true},
		{"running", "RUNNING", "Running", true},
		{"canceling", "CANCELING", "Terminating", true},
		{"paused", "PAUSED", "Pending", true},
		{"unknown", nil, "", true},
		{"v1-running", "", "Running", true},
		{"v2-succeeded", "SUCCEEDED", "Running", false},
		{"v2-canceled", "CANCELED", "Failed", false},
		{"v1-succeeded", nil, "Succeeded", false},
		{"v1-failed", "", "Failed", false},
		{"v1-error", nil, "Error", false},
		{"v1-state-succeeded", "Succeeded", "Running", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, d, store := initializeDBAndStore()
			t.Cleanup(func() { require.NoError(t, db.Close()) })
			runStore := NewRunStore(db, util.NewFakeTimeForEpoch(), d)
			_, err := runStore.CreateRun(&model.Run{
				UUID: "run", DisplayName: "run", RecurringRunId: "1", ExperimentId: defaultFakeExpId,
				Namespace: "n1", RunDetails: model.RunDetails{State: model.RuntimeStateRunning},
			})
			require.NoError(t, err)
			_, err = db.Exec(`UPDATE run_details SET State = ?, Conditions = ? WHERE UUID = ?`, tc.state, tc.conditions, "run")
			require.NoError(t, err)
			_, err = store.ClaimRecurringRun("1", "tick", 0, 100, 110, "")
			if tc.active {
				require.ErrorContains(t, err, "maximum concurrency")
				state, err := store.GetRecurringRunState("1")
				require.NoError(t, err)
				require.Zero(t, state.LastRunIndex)
				// The stored policy can permit another execution.
				_, err = db.Exec(`UPDATE jobs SET MaxConcurrency = ? WHERE UUID = ?`, 2, "1")
				require.NoError(t, err)
				_, err = store.ClaimRecurringRun("1", "tick", 0, 100, 110, "")
				require.NoError(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCreateJobRollsBackWhenSchedulingStateCannotBeCreated(t *testing.T) {
	db, _, store := initializeDBAndStore()
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	_, err := db.Exec(`INSERT INTO recurring_run_states (JobUUID, RequestKey, PipelineVersionID) VALUES (?, ?, ?)`, "conflict", "", "")
	require.NoError(t, err)
	_, err = store.CreateJob(&model.Job{UUID: "conflict", DisplayName: "new-job", ExperimentId: defaultFakeExpId})
	require.ErrorContains(t, err, "initialize scheduling state")
	var id string
	err = db.QueryRow(`SELECT UUID FROM jobs WHERE UUID = ?`, "conflict").Scan(&id)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestCreateRunAtomicallyCompletesRecurringClaim(t *testing.T) {
	for _, reporter := range []bool{false, true} {
		t.Run(fmt.Sprintf("reporter-%v", reporter), func(t *testing.T) {
			db, d, store := initializeDBAndStore()
			t.Cleanup(func() { require.NoError(t, db.Close()) })
			_, err := store.ClaimRecurringRun("1", "original-request", 0, 100, 110, "version-a")
			require.NoError(t, err)
			run := &model.Run{
				UUID: util.NewDeterministicUUID("1/tick/1"), DisplayName: "original-request", RecurringRunId: "1",
				ExperimentId: defaultFakeExpId, Namespace: "n1", RunDetails: model.RunDetails{State: model.RuntimeStateSucceeded},
			}
			if reporter {
				run.K8SName = "run-" + util.NewDeterministicUUID(run.UUID)
				run.DisplayName = run.K8SName
				run.CreatedAtInSec = 900
				run.ScheduledAtInSec = 800
			}
			runStore := NewRunStore(db, util.NewFakeTimeForEpoch(), d)
			created, err := runStore.CreateRun(run)
			require.NoError(t, err)
			require.Equal(t, "original-request", created.DisplayName)
			if reporter {
				require.Equal(t, "version-a", created.PipelineVersionId)
				require.Equal(t, int64(110), created.CreatedAtInSec)
				require.Equal(t, int64(100), created.ScheduledAtInSec)
				stored, err := runStore.GetRun(created.UUID)
				require.NoError(t, err)
				require.Equal(t, created.PipelineVersionId, stored.PipelineVersionId)
				require.Equal(t, created.CreatedAtInSec, stored.CreatedAtInSec)
				require.Equal(t, created.ScheduledAtInSec, stored.ScheduledAtInSec)
			}
			id, err := runStore.GetRunByRecurringRunIDAndDisplayName("1", "original-request")
			require.NoError(t, err)
			require.Equal(t, run.UUID, id)
			state, err := store.GetRecurringRunState("1")
			require.NoError(t, err)
			require.False(t, state.Pending)
			require.Equal(t, int64(1), state.LastRunIndex)
			require.Equal(t, "version-a", state.PipelineVersionID)
			require.NoError(t, runStore.DeleteRun(run.UUID))
			_, err = store.ClaimRecurringRun("1", "original-request", 1, 200, 210, "version-b")
			require.ErrorContains(t, err, "already completed")
		})
	}
}

func TestCreateRunRollsBackWhenRecurringClaimCompletionFails(t *testing.T) {
	db, d, store := initializeDBAndStore()
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	claim, err := store.ClaimRecurringRun("1", "tick", 0, 100, 110, "")
	require.NoError(t, err)
	_, err = db.Exec(`CREATE TRIGGER fail_completion BEFORE UPDATE OF Pending ON recurring_run_states
		WHEN NEW.Pending = 0 BEGIN SELECT RAISE(ABORT, 'simulated completion failure'); END`)
	require.NoError(t, err)
	runStore := NewRunStore(db, util.NewFakeTimeForEpoch(), d)
	run := &model.Run{
		UUID: util.NewDeterministicUUID("1/tick/1"), DisplayName: "tick", RecurringRunId: "1",
		ExperimentId: defaultFakeExpId, Namespace: "n1", RunDetails: model.RunDetails{State: model.RuntimeStateSucceeded},
	}
	_, err = runStore.CreateRun(run)
	require.ErrorContains(t, err, "simulated completion failure")
	state, err := store.GetRecurringRunState("1")
	require.NoError(t, err)
	require.Equal(t, claim, state)
	_, err = runStore.GetRun(run.UUID)
	require.True(t, util.IsUserErrorCodeMatch(err, codes.NotFound), "run insertion must roll back with claim completion: %v", err)
	_, err = db.Exec(`DROP TRIGGER fail_completion`)
	require.NoError(t, err)
	_, err = runStore.CreateRun(run)
	require.NoError(t, err)
	state, err = store.GetRecurringRunState("1")
	require.NoError(t, err)
	require.False(t, state.Pending)
}

func TestCreateRunDoesNotReleaseAnotherRecurringClaim(t *testing.T) {
	for _, kind := range []string{"run-id", "request-key", "reporter-workflow", "job-id"} {
		t.Run(kind, func(t *testing.T) {
			db, d, store := initializeDBAndStore()
			t.Cleanup(func() { require.NoError(t, db.Close()) })
			claim, err := store.ClaimRecurringRun("1", "tick", 0, 100, 110, "")
			require.NoError(t, err)
			run := &model.Run{
				UUID: util.NewDeterministicUUID("1/tick/1"), DisplayName: "tick", RecurringRunId: "1",
				ExperimentId: defaultFakeExpId, Namespace: "n1", RunDetails: model.RunDetails{State: model.RuntimeStateSucceeded},
			}
			switch kind {
			case "run-id":
				run.UUID = "different-run"
			case "request-key":
				run.DisplayName = "different-request"
			case "reporter-workflow":
				run.DisplayName = "run-" + util.NewDeterministicUUID(run.UUID)
				run.K8SName = "different-workflow"
			case "job-id":
				run.RecurringRunId = "2"
			}
			_, err = NewRunStore(db, util.NewFakeTimeForEpoch(), d).CreateRun(run)
			require.NoError(t, err)
			state, err := store.GetRecurringRunState("1")
			require.NoError(t, err)
			require.Equal(t, claim, state)
		})
	}
}

func TestRecurringRunClaimValidatesRequestKeyBeforeReservation(t *testing.T) {
	for _, tc := range []struct {
		name  string
		key   string
		valid bool
	}{
		{"ascii-at-limit", strings.Repeat("a", 255), true},
		{"ascii-over-limit", strings.Repeat("a", 256), false},
		{"multibyte-at-limit", strings.Repeat("é", 255), true},
		{"multibyte-over-limit", strings.Repeat("é", 256), false},
		{"mixed-at-limit", strings.Repeat("a", 254) + "界", true},
		{"mixed-over-limit", strings.Repeat("a", 255) + "界", false},
		{"invalid-utf8", "invalid\xff", false},
		{"empty", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, d, store := initializeDBAndStore()
			t.Cleanup(func() { require.NoError(t, db.Close()) })
			before, err := store.GetRecurringRunState("1")
			require.NoError(t, err)
			claim, err := store.ClaimRecurringRun("1", tc.key, 0, 100, 110, "version-a")
			if !tc.valid {
				require.True(t, util.IsUserErrorCodeMatch(err, codes.InvalidArgument), "%v", err)
				require.Nil(t, claim)
				after, getErr := store.GetRecurringRunState("1")
				require.NoError(t, getErr)
				require.Equal(t, before, after)
				claim, err = store.ClaimRecurringRun("1", "valid-next-attempt", 0, 100, 110, "version-a")
			}
			require.NoError(t, err)
			require.Equal(t, int64(1), claim.LastRunIndex)
			require.True(t, claim.Pending)
			run, err := NewRunStore(db, util.NewFakeTimeForEpoch(), d).CreateRun(&model.Run{
				UUID: util.NewDeterministicUUID("1/tick/1"), DisplayName: claim.RequestKey,
				RecurringRunId: "1", ExperimentId: defaultFakeExpId, Namespace: "n1",
				RunDetails: model.RunDetails{State: model.RuntimeStateSucceeded},
			})
			require.NoError(t, err)
			require.Equal(t, claim.RequestKey, run.DisplayName)
			state, err := store.GetRecurringRunState("1")
			require.NoError(t, err)
			require.False(t, state.Pending)
		})
	}
}
