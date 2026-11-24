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
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/require"
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
	if engine == "mysql" {
		return jobUUIDMySQL(t), dialect.NewDBDialect("mysql")
	}
	db := jobUUIDPostgres(t)
	require.NoError(t, preserveRunJobUUID(db).AutoMigrate(model.AllModels()...))
	return db, dialect.NewDBDialect("pgx")
}

// Hold the first UPDATE open so concurrent writes reliably overlap its locks,
// even on a fast server with a warm cache.
func TestBackfillPipelineRefs_ConcurrentWrites(t *testing.T) {
	for _, engine := range []string{"mysql", "postgres"} {
		t.Run(engine, func(t *testing.T) {
			db, d := setupBackfillTestDB(t, engine)
			require.NoError(t, db.Create(&model.Experiment{UUID: "exp", Name: "experiment", Namespace: "ns"}).Error)
			insertBackfillCandidates(t, db, 2000)
			pool, err := db.DB()
			require.NoError(t, err)
			runs := storage.NewRunStore(pool, util.NewRealTime(), d)
			// This run also needs repair, but its key sorts after the first batch.
			// A whole-table UPDATE locks it immediately and stalls its reporter.
			reportedRun, err := runs.CreateRun(&model.Run{UUID: "run-999999", DisplayName: "report", Namespace: "ns", ExperimentId: "exp", RunDetails: model.RunDetails{State: model.RuntimeStatePending}})
			require.NoError(t, err)
			require.NoError(t, db.Create(&model.ResourceReference{
				ResourceUUID: reportedRun.UUID, ResourceType: model.RunResourceType,
				ReferenceUUID: "pipeline", ReferenceType: model.PipelineResourceType, Relationship: model.OwnerRelationship,
			}).Error)
			reportedRun.State = model.RuntimeStateRunning
			locked, release := make(chan struct{}), make(chan struct{})
			var releaseOnce, holdOnce sync.Once
			resume := func() { releaseOnce.Do(func() { close(release) }) }
			require.NoError(t, db.Callback().Raw().After("gorm:raw").Register("test:hold-first-backfill-write", func(tx *gorm.DB) {
				if isRunBackfillWrite(tx) && tx.Error == nil {
					holdOnce.Do(func() { close(locked); <-release })
				}
			}))
			finished := make(chan error, 1)
			var background sync.WaitGroup
			defer func() { resume(); background.Wait() }()
			background.Add(1)
			go func() { defer background.Done(); finished <- backfillPipelineRefsToRunTable(db, d) }()
			select {
			case <-locked:
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
			resume()
			require.NoError(t, <-finished)
		})
	}
}
