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
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
)

func TestRecurringRunReferencesAreAtomicWithClaim(t *testing.T) {
	for _, failure := range []string{"claim-completion", "reference-insert", "duplicate-retry"} {
		t.Run(failure, func(t *testing.T) {
			db, d, jobs := initializeDBAndStore()
			t.Cleanup(func() { require.NoError(t, db.Close()) })
			claim, err := jobs.ClaimRecurringRun("1", "tick", 0, 100, 110, "")
			require.NoError(t, err)
			runs := NewRunStore(db, util.NewFakeTimeForEpoch(), d)
			run := &model.Run{
				UUID: util.NewDeterministicUUID("1/tick/1"), DisplayName: "tick", RecurringRunId: "1",
				ExperimentId: defaultFakeExpId, Namespace: "n1", RunDetails: model.RunDetails{State: model.RuntimeStateSucceeded},
			}
			run.ResourceReferences = []*model.ResourceReference{{ResourceUUID: run.UUID, ResourceType: model.RunResourceType,
				ReferenceUUID: defaultFakeExpId, ReferenceType: model.ExperimentResourceType, Relationship: model.OwnerRelationship}}
			switch failure {
			case "claim-completion":
				_, err = db.Exec(`CREATE TRIGGER fail_claim BEFORE UPDATE OF Pending ON recurring_run_states
     WHEN NEW.Pending = 0 BEGIN SELECT RAISE(ABORT, 'claim failure'); END`)
				require.NoError(t, err)
			case "reference-insert":
				_, err = db.Exec(`CREATE TRIGGER fail_reference BEFORE INSERT ON resource_references
     BEGIN SELECT RAISE(ABORT, 'reference failure'); END`)
				require.NoError(t, err)
			}
			_, err = runs.CreateRun(run)
			if failure == "duplicate-retry" {
				require.NoError(t, err)
				completed, err := jobs.GetRecurringRunState("1")
				require.NoError(t, err)
				require.False(t, completed.Pending)
				var beforeCount int
				require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM resource_references WHERE ResourceUUID = ?`, run.UUID).Scan(&beforeCount))
				require.Positive(t, beforeCount)
				// A retry with a different reference must return the stored run without
				// inserting another reference or advancing the completed claim.
				run.ResourceReferences[0].ReferenceType = model.JobResourceType
				run.ResourceReferences[0].ReferenceUUID = "1"
				_, err = runs.CreateRun(run)
				require.NoError(t, err)
				after, err := jobs.GetRecurringRunState("1")
				require.NoError(t, err)
				require.Equal(t, completed, after)
				var count int
				require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM resource_references WHERE ResourceUUID = ?`, run.UUID).Scan(&count))
				require.Equal(t, beforeCount, count)
			} else {
				if failure == "claim-completion" {
					require.ErrorContains(t, err, "claim failure")
				} else {
					require.ErrorContains(t, err, "reference failure")
				}
				_, err = runs.GetRun(run.UUID)
				require.True(t, util.IsUserErrorCodeMatch(err, codes.NotFound))
				after, err := jobs.GetRecurringRunState("1")
				require.NoError(t, err)
				require.Equal(t, claim, after)
				var count int
				require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM resource_references WHERE ResourceUUID = ?`, run.UUID).Scan(&count))
				require.Zero(t, count)
			}
		})
	}
}
