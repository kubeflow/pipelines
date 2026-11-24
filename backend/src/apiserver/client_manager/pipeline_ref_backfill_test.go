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

package clientmanager

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	api "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
	"github.com/kubeflow/pipelines/backend/src/apiserver/filter"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func insertBackfillCandidates(t *testing.T, db *gorm.DB, count int) {
	t.Helper()
	runs := make([]model.Run, count)
	refs := make([]model.ResourceReference, count)
	for i := range runs {
		id := fmt.Sprintf("run-%06d", i)
		runs[i] = model.Run{UUID: id, DisplayName: id, K8SName: id, ExperimentId: "exp", Namespace: "ns", StorageState: model.StorageStateAvailable}
		refs[i] = model.ResourceReference{ResourceUUID: id, ResourceType: model.RunResourceType,
			ReferenceUUID: "pipeline", ReferenceType: model.PipelineResourceType, Relationship: model.OwnerRelationship}
	}
	require.NoError(t, db.CreateInBatches(&runs, 100).Error)
	require.NoError(t, db.CreateInBatches(&refs, 100).Error)
}

func isRunBackfillWrite(tx *gorm.DB) bool {
	return strings.HasPrefix(strings.TrimSpace(tx.Statement.SQL.String()), "UPDATE run_details")
}

func setupBackfillTestDB(t *testing.T, engine string) (*gorm.DB, dialect.DBDialect) {
	t.Helper()
	envName := "KFP_RECURRING_" + strings.ToUpper(engine) + "_TEST_DSN"
	dsn := os.Getenv(envName)
	if dsn == "" {
		t.Skipf("set %s to run against %s", envName, engine)
	}
	if engine == "mysql" {
		db := newTestMySQL(t, dsn)
		require.NoError(t, autoMigrate(db))
		return db, dialect.NewDBDialect("mysql")
	}

	cfg, err := pgx.ParseConfig(dsn)
	require.NoError(t, err)
	admin := stdlib.OpenDB(*cfg)
	t.Cleanup(func() { require.NoError(t, admin.Close()) })
	schema := fmt.Sprintf("kfp_backfill_%d", time.Now().UnixNano())
	_, err = admin.Exec("CREATE SCHEMA " + schema)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := admin.Exec("DROP SCHEMA " + schema + " CASCADE")
		require.NoError(t, err)
	})

	cfg.RuntimeParams["search_path"] = schema
	pool := stdlib.OpenDB(*cfg)
	t.Cleanup(func() { require.NoError(t, pool.Close()) })
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: pool}), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, autoMigrate(db))
	return db, dialect.NewDBDialect("pgx")
}

// Hold the first UPDATE open so concurrent writes reliably overlap its locks,
// even on a fast server with a warm cache.
func TestBackfillPipelineRefs_ConcurrentWrites(t *testing.T) {
	for _, tc := range []struct {
		name, engine  string
		disableBinlog bool
	}{
		{name: "mysql", engine: "mysql"},
		{name: "mysql-no-binlog", engine: "mysql", disableBinlog: true},
		{name: "postgres", engine: "postgres"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, d := setupBackfillTestDB(t, tc.engine)
			require.NoError(t, db.Create(&model.Experiment{UUID: "exp", Name: "experiment", Namespace: "ns"}).Error)
			insertBackfillCandidates(t, db, 2000)
			pool, err := db.DB()
			require.NoError(t, err)
			runs := storage.NewRunStore(pool, util.NewRealTime(), d)
			checkStandaloneReads := prepareStandaloneBackfillReads(t, db, runs)
			// This run also needs repair, but its key sorts after the first batch.
			// A whole-table UPDATE locks it immediately and stalls its reporter.
			reportedRun, err := runs.CreateRun(&model.Run{UUID: "run-000499-a", DisplayName: "report", Namespace: "ns", ExperimentId: "exp", RunDetails: model.RunDetails{State: model.RuntimeStatePending}})
			require.NoError(t, err)
			require.NoError(t, db.Create(&model.ResourceReference{
				ResourceUUID: reportedRun.UUID, ResourceType: model.RunResourceType,
				ReferenceUUID: "pipeline", ReferenceType: model.PipelineResourceType, Relationship: model.OwnerRelationship,
			}).Error)
			reportedRun.State = model.RuntimeStateRunning
			locked, release := make(chan int64, 1), make(chan struct{})
			var releaseOnce, holdOnce sync.Once
			resume := func() { releaseOnce.Do(func() { close(release) }) }
			require.NoError(t, db.Callback().Raw().After("gorm:raw").Register("test:hold-first-backfill-write", func(tx *gorm.DB) {
				if isRunBackfillWrite(tx) && tx.Error == nil {
					holdOnce.Do(func() { locked <- tx.RowsAffected; <-release })
				}
			}))
			finished := make(chan error, 1)
			var background sync.WaitGroup
			defer func() { resume(); background.Wait() }()
			background.Add(1)
			go func() {
				defer background.Done()
				finished <- db.Connection(func(pinned *gorm.DB) error {
					pinned = pinned.Session(&gorm.Session{})
					if tc.disableBinlog {
						// Discard the session so its binlog setting cannot reach other callers.
						conn := pinned.Statement.ConnPool.(*sql.Conn)
						defer conn.Raw(func(any) error { return driver.ErrBadConn })
						if err := pinned.Exec("SET SESSION sql_log_bin = OFF").Error; err != nil {
							return err
						}
					}
					return backfillPipelineRefsToRunTable(pinned, d)
				})
			}()
			var heldRows int64
			select {
			case heldRows = <-locked:
			case err := <-finished:
				t.Fatalf("backfill ended before its first UPDATE: %v", err)
			case <-time.After(15 * time.Second):
				t.Fatal("backfill did not reach its first UPDATE")
			}
			created, reported := make(chan error, 1), make(chan error, 1)
			background.Add(2)
			go func() {
				defer background.Done()
				_, err := runs.CreateRun(&model.Run{UUID: "run-000000-b", DisplayName: "created-during-backfill", Namespace: "ns", ExperimentId: "exp"})
				created <- err
			}()
			go func() {
				defer background.Done()
				applied, err := runs.UpdateRunFromWorkflow(reportedRun, model.RuntimeStatePending, "", "")
				if err == nil && !applied {
					err = errors.New("report was not applied")
				}
				reported <- err
			}()
			createResult, reportResult := created, reported
			deadline := time.After(3 * time.Second)
			for createResult != nil || reportResult != nil {
				select {
				case err := <-createResult:
					require.NoError(t, err)
					createResult = nil
				case err := <-reportResult:
					require.NoError(t, err)
					reportResult = nil
				case <-deadline:
					t.Fatalf("writes blocked for 3s during backfill: CreateRun pending=%t, UpdateRunFromWorkflow pending=%t", createResult != nil, reportResult != nil)
				}
			}
			if tc.disableBinlog {
				require.EqualValues(t, 500, heldRows, "exercise a full production batch")
			}
			t.Logf("CreateRun and workflow report committed while the first %d-row UPDATE was held", heldRows)
			checkStandaloneReads()
			resume()
			require.NoError(t, <-finished)
			checkStandaloneReads()
		})
	}
}

// Read the same standalone runs while the first repair transaction is paused
// and after it commits. Both the stored values and the API's resource-reference
// fallback must produce the same filtered count and keyset traversal.
func prepareStandaloneBackfillReads(t *testing.T, db *gorm.DB, runs *storage.RunStore) func() {
	t.Helper()
	const exp = "standalone"
	require.NoError(t, db.Create(&model.Experiment{UUID: exp, Name: exp, Namespace: "ns"}).Error)
	require.NoError(t, db.Create(&model.Pipeline{UUID: "a-parent", Name: "p", DisplayName: "p", Status: model.PipelineReady, Namespace: "ns"}).Error)
	require.NoError(t, db.Create(&model.PipelineVersion{UUID: "version", Name: "v", DisplayName: "v", PipelineId: "a-parent", Status: model.PipelineVersionReady, PipelineSpec: "spec"}).Error)

	fixtures := []struct {
		id, storedID, storedVersion, directRef, versionRef, expected string
	}{
		{id: "legacy-version", versionRef: "version", expected: "a-parent"},
		{id: "legacy-direct", directRef: "b-direct", versionRef: "version", expected: "b-direct"},
		{id: "legacy-stored", storedID: "c-stored", directRef: "b-direct", expected: "c-stored"},
		{id: "legacy-stored-version", storedVersion: "version", versionRef: "stale-version", expected: "a-parent"},
	}
	for _, row := range fixtures {
		require.NoError(t, db.Create(&model.Run{UUID: row.id, DisplayName: row.id, K8SName: row.id, ExperimentId: exp,
			Namespace: "ns", StorageState: model.StorageStateAvailable,
			PipelineSpec: model.PipelineSpec{PipelineId: row.storedID, PipelineVersionId: row.storedVersion}}).Error)
		for kind, id := range map[model.ResourceType]string{model.PipelineResourceType: row.directRef, model.PipelineVersionResourceType: row.versionRef} {
			if id == "" {
				continue
			}
			ref := model.ResourceReference{ResourceUUID: row.id, ResourceType: model.RunResourceType,
				ReferenceUUID: id, ReferenceType: kind, Relationship: model.OwnerRelationship}
			payload, err := json.Marshal(&ref)
			require.NoError(t, err)
			ref.Payload = model.LargeText(payload)
			require.NoError(t, db.Create(&ref).Error)
		}
	}

	return func() {
		for _, row := range fixtures {
			got, err := runs.GetRun(row.id, false)
			require.NoError(t, err)
			require.Equal(t, row.expected, got.PipelineId, row.id)
		}
		ctx := &model.FilterContext{ReferenceKey: &model.ReferenceKey{Type: model.ExperimentResourceType, ID: exp}}
		for _, tt := range []struct {
			op  api.Predicate_Operation
			ids []string
		}{
			{api.Predicate_EQUALS, []string{"legacy-stored-version", "legacy-version"}},
			{api.Predicate_NOT_EQUALS, []string{"legacy-direct", "legacy-stored"}},
		} {
			pred, err := filter.New(&api.Filter{Predicates: []*api.Predicate{{
				Key: "pipeline_id", Operation: tt.op, Value: &api.Predicate_StringValue{StringValue: "a-parent"},
			}}})
			require.NoError(t, err)
			opts, err := list.NewOptions(&model.Run{}, 1, "pipeline_id", pred)
			require.NoError(t, err)
			for i, id := range tt.ids {
				page, count, token, err := runs.ListRuns(ctx, opts, false)
				require.NoError(t, err)
				require.Equal(t, len(tt.ids), count)
				require.Len(t, page, 1)
				require.Equal(t, id, page[0].UUID)
				if i == len(tt.ids)-1 {
					require.Empty(t, token)
				} else {
					opts, err = list.NewOptionsFromToken(token, 1)
					require.NoError(t, err)
				}
			}
		}
	}
}

// A retained replica can snapshot a legacy job, pause during report processing,
// and insert its recurring execution after the migration has marked completion.
func TestBackfillPipelineRefs_InFlightRecurringReport(t *testing.T) {
	for _, engine := range []string{"mysql", "postgres"} {
		t.Run(engine, func(t *testing.T) {
			db, d := setupBackfillTestDB(t, engine)
			require.NoError(t, db.Create(&model.Experiment{UUID: "exp", Name: "experiment", Namespace: "ns"}).Error)
			require.NoError(t, db.Create(&model.Pipeline{UUID: "parent", Name: "parent", DisplayName: "parent", Status: model.PipelineReady, Namespace: "ns"}).Error)
			require.NoError(t, db.Create(&model.PipelineVersion{UUID: "version", Name: "version", DisplayName: "version", PipelineId: "parent", Status: model.PipelineVersionReady, PipelineSpec: "spec"}).Error)
			require.NoError(t, db.Create(&model.Job{
				UUID: "legacy-job", DisplayName: "legacy", K8SName: "legacy", Namespace: "ns", ExperimentId: "exp",
				PipelineSpec: model.PipelineSpec{PipelineVersionId: "version"},
			}).Error)

			pool, err := db.DB()
			require.NoError(t, err)
			clock := util.NewRealTime()
			jobs := storage.NewJobStore(pool, clock, nil, d)
			snapshot, err := jobs.GetJob("legacy-job")
			require.NoError(t, err)
			require.Empty(t, snapshot.PipelineId)
			require.Equal(t, "version", snapshot.PipelineVersionId)

			require.NoError(t, backfillPipelineRefsToRunTable(db, d))
			repaired, err := jobs.GetJob("legacy-job")
			require.NoError(t, err)
			require.Equal(t, "parent", repaired.PipelineId)

			// Insert the payload retained by the older reporter, independently of
			// any normalization performed by a newer RunStore.CreateRun.
			insert := func(id, pipelineID string) {
				spec := snapshot.PipelineSpec
				spec.PipelineId = pipelineID
				require.NoError(t, db.Create(&model.Run{
					UUID: id, DisplayName: id, K8SName: id, StorageState: model.StorageStateAvailable,
					ExperimentId: snapshot.ExperimentId, Namespace: snapshot.Namespace,
					RecurringRunId: snapshot.UUID, PipelineSpec: spec,
				}).Error)
			}
			insert("stale", "")
			insert("explicit", "z-explicit")
			insert("referenced", "")
			ref := &model.ResourceReference{ResourceUUID: "referenced", ResourceType: model.RunResourceType,
				ReferenceUUID: "a-direct", ReferenceType: model.PipelineResourceType, Relationship: model.OwnerRelationship}
			payload, err := json.Marshal(ref)
			require.NoError(t, err)
			ref.Payload = model.LargeText(payload)
			require.NoError(t, db.Create(ref).Error)
			require.NoError(t, backfillPipelineRefsToRunTable(db, d))
			runs := storage.NewRunStore(pool, clock, d)
			for id, want := range map[string]string{"stale": "parent", "explicit": "z-explicit", "referenced": "a-direct"} {
				got, err := runs.GetRun(id, false)
				require.NoError(t, err)
				assert.Equal(t, want, got.PipelineId, id)
			}
			for _, tt := range []struct {
				op   api.Predicate_Operation
				want []string
			}{
				{api.Predicate_EQUALS, []string{"stale"}},
				{api.Predicate_NOT_EQUALS, []string{"referenced", "explicit"}},
			} {
				pred, err := filter.New(&api.Filter{Predicates: []*api.Predicate{{
					Key: "pipeline_id", Operation: tt.op, Value: &api.Predicate_StringValue{StringValue: "parent"},
				}}})
				require.NoError(t, err)
				opts, err := list.NewOptions(&model.Run{}, 10, "pipeline_id", pred)
				require.NoError(t, err)
				matches, count, _, err := runs.ListRuns(&model.FilterContext{}, opts, false)
				require.NoError(t, err)
				assert.Equal(t, len(tt.want), count, tt.op)
				var ids []string
				for _, run := range matches {
					ids = append(ids, run.UUID)
				}
				assert.Equal(t, tt.want, ids, tt.op)
			}

			opts, err := list.NewOptions(&model.Run{}, 1, "pipeline_id", nil)
			require.NoError(t, err)
			for _, want := range []string{"a-direct", "parent", "z-explicit"} {
				page, _, token, err := runs.ListRuns(&model.FilterContext{}, opts, false)
				require.NoError(t, err)
				require.Len(t, page, 1)
				assert.Equal(t, want, page[0].PipelineId)
				if want == "z-explicit" {
					require.Empty(t, token)
				} else {
					opts, err = list.NewOptionsFromToken(token, 1)
					require.NoError(t, err)
				}
			}

			// The version record is authoritative while present. If it is
			// deleted, only a job still pointing at that exact version can
			// supply the missing ID for an old execution.
			job := db.Model(&model.Job{}).Where(map[string]any{"UUID": snapshot.UUID})
			require.NoError(t, job.Update("PipelineId", "job-parent").Error)
			getPipelineID := func() string {
				got, err := runs.GetRun("stale", false)
				require.NoError(t, err)
				return got.PipelineId
			}
			assert.Equal(t, "parent", getPipelineID())
			require.NoError(t, db.Where(map[string]any{"UUID": "version"}).Delete(&model.PipelineVersion{}).Error)
			assert.Equal(t, "job-parent", getPipelineID())
			require.NoError(t, job.Update("PipelineVersionId", "different-version").Error)
			assert.Empty(t, getPipelineID())
		})
	}
}
