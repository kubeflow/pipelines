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
	"fmt"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/require"
)

func TestListRuns_NullCursorDoesNotRepeatRows(t *testing.T) {
	for _, field := range []struct{ api, column string }{{"state", "State"}, {"recurring_run_id", "JobUUID"}, {"scheduled_at", "ScheduledAtInSec"}, {"finished_at", "FinishedAtInSec"}} {
		for _, nullCount := range []int{0, 1, 3, 5} {
			for _, direction := range []string{"asc", "desc"} {
				t.Run(fmt.Sprintf("%s_%s_nulls_%d", field.api, direction, nullCount), func(t *testing.T) {
					db, dialect := NewFakeDBOrFatal()
					defer db.Close()
					exps, err := NewExperimentStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(defaultFakeExpId, nil), dialect)
					require.NoError(t, err)
					_, err = exps.CreateExperiment(&model.Experiment{Name: "pagination"})
					require.NoError(t, err)
					store := NewRunStore(db, util.NewFakeTimeForEpoch(), dialect)
					for i := 0; i < 5; i++ {
						_, err = store.CreateRun(&model.Run{
							UUID: fmt.Sprint(i), ExperimentId: defaultFakeExpId, K8SName: fmt.Sprint(i),
							RunDetails: model.RunDetails{
								State: model.RuntimeStateRunning, ScheduledAtInSec: 1, FinishedAtInSec: 1,
							},
						})
						require.NoError(t, err)
					}
					_, err = db.Exec("UPDATE run_details SET JobUUID = 'job'")
					require.NoError(t, err)
					// Legacy state is normalized in API responses but cursors must use SQL values.
					if field.api == "state" {
						_, err = db.Exec("UPDATE run_details SET State = 'Enabled'")
						require.NoError(t, err)
					}
					_, err = db.Exec("UPDATE run_details SET "+field.column+" = NULL WHERE UUID < ?", fmt.Sprint(nullCount))
					require.NoError(t, err)
					opts, err := list.NewOptions(&model.Run{}, 2, field.api+" "+direction, nil)
					require.NoError(t, err)
					seen := map[string]bool{}
					for page := 0; page < 10; page++ {
						rows, _, token, err := store.ListRuns(&model.FilterContext{}, opts)
						require.NoError(t, err)
						for _, row := range rows {
							require.False(t, seen[row.UUID], "duplicate run on page %d", page)
							seen[row.UUID] = true
						}
						if token == "" {
							require.Len(t, seen, 5)
							return
						}
						opts, err = list.NewOptionsFromToken(token, 2)
						require.NoError(t, err)
					}
					t.Fatal("pagination did not finish")
				})
			}
		}
	}
}

func TestListJobs_NullUpdatedAtCursorDoesNotRepeatRows(t *testing.T) {
	for _, nullCount := range []int{0, 1, 3, 5} {
		for _, direction := range []string{"asc", "desc"} {
			t.Run(fmt.Sprintf("nulls_%d_%s", nullCount, direction), func(t *testing.T) { testNullableJobUpdatedAt(t, nullCount, direction) })
		}
	}
}
func testNullableJobUpdatedAt(t *testing.T, nullCount int, direction string) {
	t.Helper()
	db, dialect := NewFakeDBOrFatal()
	defer db.Close()
	store := NewJobStore(db, util.NewFakeTimeForEpoch(), nil, dialect)
	for i := 0; i < 5; i++ {
		_, err := store.CreateJob(&model.Job{UUID: fmt.Sprint(i), DisplayName: fmt.Sprint(i), K8SName: fmt.Sprint(i)})
		require.NoError(t, err)
	}
	_, err := db.Exec("UPDATE jobs SET UpdatedAtInSec = 1")
	require.NoError(t, err)
	_, err = db.Exec("UPDATE jobs SET UpdatedAtInSec = NULL WHERE UUID < ?", fmt.Sprint(nullCount))
	require.NoError(t, err)
	opts, err := list.NewOptions(&model.Job{}, 2, "updated_at "+direction, nil)
	require.NoError(t, err)
	seen := map[string]bool{}
	for page := 0; page < 10; page++ {
		rows, _, token, err := store.ListJobs(&model.FilterContext{}, opts)
		require.NoError(t, err)
		for _, row := range rows {
			require.False(t, seen[row.UUID], "duplicate job on page %d", page)
			seen[row.UUID] = true
		}
		if token == "" {
			require.Len(t, seen, 5)
			return
		}
		opts, err = list.NewOptionsFromToken(token, 2)
		require.NoError(t, err)
	}
	t.Fatal("pagination did not finish")
}

func TestListPipelines_NullNamespaceCursorDoesNotRepeatRows(t *testing.T) {
	for _, nullCount := range []int{0, 1, 3, 5} {
		for _, direction := range []string{"asc", "desc"} {
			for _, v1 := range []bool{false, true} {
				t.Run(fmt.Sprintf("v1_%t_nulls_%d_%s", v1, nullCount, direction), func(t *testing.T) {
					db, dialect := NewFakeDBOrFatal()
					defer db.Close()
					store := NewPipelineStore(db, util.NewFakeTimeForEpoch(), nil, dialect)
					for i := 0; i < 5; i++ {
						store.uuid = util.NewFakeUUIDGeneratorOrFatal(fmt.Sprintf("123e4567-e89b-12d3-a456-42665544010%d", i), nil)
						_, err := store.CreatePipeline(&model.Pipeline{Name: fmt.Sprint(i), Namespace: "namespace", Status: model.PipelineReady})
						require.NoError(t, err)
					}
					_, err := db.Exec("UPDATE pipelines SET Namespace = NULL WHERE Name < ?", fmt.Sprint(nullCount))
					require.NoError(t, err)
					opts, err := list.NewOptions(&model.Pipeline{}, 2, "namespace "+direction, nil)
					require.NoError(t, err)
					seen := map[string]bool{}
					for page := 0; page < 10; page++ {
						var rows []*model.Pipeline
						var token string
						if v1 {
							rows, _, _, token, err = store.ListPipelinesV1(&model.FilterContext{}, opts)
						} else {
							rows, _, token, err = store.ListPipelines(&model.FilterContext{}, opts, nil)
						}
						require.NoError(t, err)
						for _, row := range rows {
							require.False(t, seen[row.UUID], "duplicate pipeline on page %d", page)
							seen[row.UUID] = true
						}
						if token == "" {
							require.Len(t, seen, 5)
							return
						}
						opts, err = list.NewOptionsFromToken(token, 2)
						require.NoError(t, err)
					}
					t.Fatal("pagination did not finish")
				})
			}
		}

	}
}
