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
	"fmt"
	"strings"
	"testing"

	api "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/filter"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/require"
)

// Large task aggregates must not become GROUP BY keys: MySQL can exhaust its
// sort buffer, and a run list must preserve every task, metric and reference.
func TestRunStoreLargeTaskAggregates(t *testing.T) {
	for _, driver := range []string{"mysql", "pgx"} {
		t.Run(driver, func(t *testing.T) { testRunStoreLargeTaskAggregates(t, driver) })
	}
}

func testRunStoreLargeTaskAggregates(t *testing.T, driver string) {
	dbs, d := recurringIntegrationDatabases(t, driver)
	db := dbs[0]
	if driver == "mysql" {
		// Match the production aggregation limit while retaining the default sort buffer.
		_, err := db.Exec("SET SESSION group_concat_max_len = 4194304")
		require.NoError(t, err)
	}
	store := NewRunStore(db, util.NewFakeTimeForEpoch(), d)
	for _, size := range []int{4 * 1024, 16 * 1024, 64 * 1024, 256 * 1024} {
		t.Run(fmt.Sprintf("task_payload_bytes_%d", size), func(t *testing.T) {
			ns := "fixture"
			jobID := fmt.Sprintf("schedule-%d", size)
			_, err := NewJobStore(db, util.NewFakeTimeForEpoch(), nil, d).CreateJob(&model.Job{UUID: jobID, DisplayName: jobID, Namespace: ns, Enabled: true, MaxConcurrency: 1})
			require.NoError(t, err)
			payload := model.LargeText(`{"padding":"` + strings.Repeat("x", 100*1024) + `"}`)
			for i := 0; i < 50; i++ {
				_, err := store.CreateRun(&model.Run{UUID: fmt.Sprintf("%d-%03d", size, i), Namespace: ns, RecurringRunId: jobID, StorageState: model.StorageStateAvailable, RunDetails: model.RunDetails{CreatedAtInSec: int64(i + 1), State: model.RuntimeStateSucceeded, WorkflowRuntimeManifest: payload, PipelineRuntimeManifest: payload}})
				require.NoError(t, err)
				runID := fmt.Sprintf("%d-%03d", size, i)
				for taskIndex := 0; taskIndex < 3; taskIndex++ {
					taskID := util.NewDeterministicUUID(fmt.Sprintf("%s/task/%d", runID, taskIndex))
					taskStore := NewTaskStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(taskID, nil), d)
					_, err := taskStore.CreateTask(&model.Task{RunID: runID, Namespace: ns, Name: fmt.Sprintf("task-%d", taskIndex), State: model.RuntimeStateSucceeded, MLMDInputs: model.LargeText(strings.Repeat("x", size))})
					require.NoError(t, err)
				}
				require.NoError(t, store.CreateMetric(&model.RunMetric{RunUUID: runID, NodeID: "task-0", Name: "accuracy", NumberValue: 1}))
				require.NoError(t, store.CreateMetric(&model.RunMetric{RunUUID: runID, NodeID: "task-1", Name: "loss", NumberValue: 0.25}))
			}
			f, err := filter.New(&api.Filter{Predicates: []*api.Predicate{{Key: "recurring_run_id", Operation: api.Predicate_EQUALS, Value: &api.Predicate_StringValue{StringValue: jobID}}}})
			require.NoError(t, err)
			opts, err := list.NewOptions(&model.Run{}, 100, "", f)
			require.NoError(t, err)
			ctx := &model.FilterContext{ReferenceKey: &model.ReferenceKey{Type: model.NamespaceResourceType, ID: ns}}
			query, args, err := store.buildSelectRunsQuery(false, opts, ctx)
			require.NoError(t, err)
			rows, queryErr := db.Query(query, args...)
			count := 0
			if queryErr == nil {
				for rows.Next() {
					count++
				}
				queryErr = rows.Err()
				rows.Close()
			}
			runs, total, token, listErr := store.ListRuns(ctx, opts)
			got, err := store.GetRun(fmt.Sprintf("%d-000", size))
			require.NoError(t, err)
			require.Equal(t, len(payload), len(got.WorkflowRuntimeManifest))
			require.Equal(t, len(payload), len(got.PipelineRuntimeManifest))
			require.NoError(t, listErr)
			require.Equal(t, 50, total)
			require.Equal(t, 50, len(runs))
			require.Empty(t, token)
			require.NoError(t, queryErr)
			require.Equal(t, 50, count)
			for i, run := range runs {
				require.Equal(t, fmt.Sprintf("%d-%03d", size, i), run.UUID)
				require.Equal(t, 3, len(run.TaskDetails))
				require.Equal(t, 2, len(run.Metrics))
				metrics := map[string]float64{}
				for _, metric := range run.Metrics {
					metrics[metric.Name] = metric.NumberValue
				}
				require.Equal(t, map[string]float64{"accuracy": 1, "loss": 0.25}, metrics)
				require.Equal(t, 2, len(run.ResourceReferences))
				require.Equal(t, jobID, run.RecurringRunId)
				require.Equal(t, ns, run.Namespace)
				for _, task := range run.TaskDetails {
					require.Equal(t, size, len(task.MLMDInputs))
					require.Equal(t, run.UUID, task.RunID)
				}
			}
			require.Equal(t, 3, len(got.TaskDetails))
			require.Equal(t, 2, len(got.Metrics))
			// Follow every page to catch omissions or repeats after aggregation.
			opts.PageSize = 17
			var ids []string
			for page := 0; page < 4; page++ {
				batch, total, next, err := store.ListRuns(ctx, opts)
				require.NoError(t, err)
				require.Equal(t, 50, total)
				for _, run := range batch {
					ids = append(ids, run.UUID)
				}
				if next == "" {
					break
				}
				opts, err = list.NewOptionsFromToken(next, 17)
				require.NoError(t, err)
			}
			require.Equal(t, 50, len(ids))
			for i, id := range ids {
				require.Equal(t, fmt.Sprintf("%d-%03d", size, i), id)
			}

		})
	}
	t.Run("missing aggregates", func(t *testing.T) {
		_, err := store.CreateRun(&model.Run{UUID: "empty-run", Namespace: "empty-ns", StorageState: model.StorageStateAvailable, RunDetails: model.RunDetails{State: model.RuntimeStateSucceeded}})
		require.NoError(t, err)
		placeholder := "?"
		if driver == "pgx" {
			placeholder = "$1"
		}
		_, err = db.Exec("DELETE FROM resource_references WHERE "+d.QuoteIdentifier("ResourceUUID")+" = "+placeholder, "empty-run")
		require.NoError(t, err)
		opts, err := list.NewOptions(&model.Run{}, 100, "", nil)
		require.NoError(t, err)
		runs, total, token, err := store.ListRuns(&model.FilterContext{ReferenceKey: &model.ReferenceKey{Type: model.NamespaceResourceType, ID: "empty-ns"}}, opts)
		require.NoError(t, err)
		require.Equal(t, 1, total)
		require.Len(t, runs, 1)
		require.Empty(t, token)
		require.Empty(t, runs[0].ResourceReferences)
		require.Empty(t, runs[0].TaskDetails)
		require.Empty(t, runs[0].Metrics)
	})

}
