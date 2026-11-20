// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0

package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
)

type productionWriteFixture struct {
	dbs   []*sql.DB
	d     dialect.DBDialect
	runID string
	task  *model.Task
}

func newProductionWriteFixture(t *testing.T, dbs []*sql.DB, d dialect.DBDialect, runID string) productionWriteFixture {
	t.Helper()
	clock := util.NewFakeTimeForEpoch()
	_, err := NewRunStore(dbs[0], clock, d).CreateRun(&model.Run{
		UUID:         runID,
		DisplayName:  runID,
		Namespace:    "retry-ns",
		StorageState: model.StorageStateAvailable,
		RunDetails: model.RunDetails{
			CreatedAtInSec:  1,
			FinishedAtInSec: 100,
			State:           model.RuntimeStateFailed,
			Conditions:      "Failed",
		},
	})
	require.NoError(t, err)
	task, err := NewTaskStore(dbs[0], clock, util.NewUUIDGenerator(), d).CreateTask(&model.Task{
		Namespace:   "retry-ns",
		RunUUID:     runID,
		Name:        "producer",
		Fingerprint: runID + "-producer",
		ScopePath:   "root.producer",
		Type:        model.TaskType(apiv2beta1.PipelineTask_RUNTIME),
		State:       model.TaskStatus(apiv2beta1.PipelineTask_FAILED),
		Pods:        model.JSONSlice{},
		TypeAttrs:   model.JSONData{},
	})
	require.NoError(t, err)
	return productionWriteFixture{dbs: dbs, d: d, runID: runID, task: task}
}

func (f productionWriteFixture) store(i int) *ArtifactStore {
	return NewArtifactStore(f.dbs[i], util.NewFakeTimeForEpoch(), util.NewUUIDGenerator(), f.d)
}

func (f productionWriteFixture) artifact(name string) *model.Artifact {
	return &model.Artifact{
		Namespace: "retry-ns",
		Type:      model.ArtifactType(apiv2beta1.Artifact_Model),
		Name:      name,
		URI:       strPTR("s3://retry-cleanup/" + f.runID),
	}
}

func (f productionWriteFixture) link(key string) *model.ArtifactTask {
	return &model.ArtifactTask{
		TaskID:      f.task.UUID,
		RunUUID:     f.runID,
		Type:        model.IOType(apiv2beta1.IOType_OUTPUT),
		ArtifactKey: key,
		Producer:    model.JSONData{"taskName": "producer"},
	}
}

func (f productionWriteFixture) write(i int, operationID string) (*model.Artifact, *model.ArtifactTask, error) {
	return f.store(i).CreateArtifactWithTask(f.artifact("model"), f.link("model"), operationID)
}

// writeBatch sends a fresh write followed by a replay of operationID in one batch.
func (f productionWriteFixture) writeBatch(i int, freshOperationID, operationID string) ([]*model.Artifact, error) {
	artifacts, _, err := f.store(i).CreateArtifactsWithTasks(
		[]*model.Artifact{f.artifact(freshOperationID), f.artifact("model")},
		[]*model.ArtifactTask{f.link(freshOperationID), f.link("model")},
		[]string{freshOperationID, operationID},
	)
	return artifacts, err
}

// retry runs the storage steps of RetryRun: claim, then reset the task attempt.
func (f productionWriteFixture) retry(t *testing.T) {
	t.Helper()
	_, _, _, generation, err := NewRunStore(f.dbs[0], util.NewFakeTimeForEpoch(), f.d).ClaimRunForRetry(f.runID, false)
	require.NoError(t, err)
	require.Equal(t, int64(1), generation)
	links := NewArtifactTaskStore(f.dbs[0], util.NewUUIDGenerator(), f.d)
	require.NoError(t, links.DeleteOutputArtifactTasksByTaskIDs([]string{f.task.UUID}))
	require.NoError(t, links.DeleteInputArtifactTasksByTaskIDs([]string{f.task.UUID}))
	require.NoError(t, NewTaskStore(f.dbs[1], util.NewFakeTimeForEpoch(), util.NewUUIDGenerator(), f.d).ResetTasksForRetry([]string{f.task.UUID}))
}

func countRetryCleanupRows(t *testing.T, db *sql.DB) (artifacts, links, identities int) {
	t.Helper()
	require.NoError(t, db.QueryRow("SELECT count(*) FROM artifacts").Scan(&artifacts))
	require.NoError(t, db.QueryRow("SELECT count(*) FROM artifact_tasks").Scan(&links))
	require.NoError(t, db.QueryRow("SELECT count(*) FROM artifact_write_identities").Scan(&identities))
	return artifacts, links, identities
}

func statusCode(err error) codes.Code {
	var userErr *util.UserError
	if errors.As(err, &userErr) {
		return userErr.ExternalStatusCode()
	}
	return codes.Unknown
}

func TestCreateArtifactWithTask_ReplayAfterRetryCleanupProductionDatabases(t *testing.T) {
	for _, driver := range []string{"mysql", "pgx"} {
		t.Run(driver, func(t *testing.T) {
			dbs, d := recurringIntegrationDatabases(t, driver)
			f := newProductionWriteFixture(t, dbs, d, "retry-cleanup-run")

			_, _, err := f.write(0, "operation-1")
			require.NoError(t, err)

			replay, _, err := f.write(1, "operation-1")
			require.NoError(t, err, "a replay before retry returns the original result")
			_, _, err = f.store(1).CreateArtifactWithTask(f.artifact("different-model"), f.link("model"), "operation-1")
			require.Equal(t, codes.InvalidArgument, statusCode(err), "unexpected error: %v", err)
			artifacts, links, identities := countRetryCleanupRows(t, dbs[0])
			require.Equal(t, []int{1, 1, 1}, []int{artifacts, links, identities})
			require.NotEmpty(t, replay.UUID)

			f.retry(t)

			_, _, err = f.write(2, "operation-1")
			require.Error(t, err)
			require.Equal(t, codes.FailedPrecondition, statusCode(err), "unexpected error: %v", err)

			artifacts, links, identities = countRetryCleanupRows(t, dbs[0])
			require.Equal(t, 1, artifacts)
			require.Equal(t, 0, links, "a stale replay must not restore the old output link")
			require.Equal(t, 1, identities)

			_, _, err = f.write(3, "operation-2")
			require.NoError(t, err, "a new operation_id in the new attempt must succeed")
		})
	}
}

func TestCreateArtifactWithTask_ReplayRacingRetryCleanupProductionDatabases(t *testing.T) {
	for _, driver := range []string{"mysql", "pgx"} {
		t.Run(driver, func(t *testing.T) {
			dbs, d := recurringIntegrationDatabases(t, driver)
			f := newProductionWriteFixture(t, dbs, d, "retry-race-run")

			original, _, err := f.write(0, "operation-race")
			require.NoError(t, err)

			const replays = 10
			start := make(chan struct{})
			errs := make([]error, replays)
			artifactIDs := make([]string, replays)
			var wg sync.WaitGroup
			for i := range replays {
				wg.Add(1)
				go func() {
					defer wg.Done()
					<-start
					if i%2 == 1 {
						artifacts, err := f.writeBatch(2+i, fmt.Sprintf("operation-fresh-%d", i), "operation-race")
						errs[i] = err
						if err == nil {
							artifactIDs[i] = artifacts[1].UUID
						}
						return
					}
					artifact, _, err := f.write(2+i, "operation-race")
					errs[i] = err
					if artifact != nil {
						artifactIDs[i] = artifact.UUID
					}
				}()
			}

			close(start)
			f.retry(t)
			done := make(chan struct{})
			go func() {
				wg.Wait()
				close(done)
			}()
			select {
			case <-done:
			case <-time.After(60 * time.Second):
				t.Fatal("replays racing retry cleanup did not finish; possible lock wait or deadlock")
			}

			for i, err := range errs {
				if err == nil {
					require.Equal(t, original.UUID, artifactIDs[i], fmt.Sprintf("replay %d returned a different artifact", i))
					continue
				}
				require.Equal(t, codes.FailedPrecondition, statusCode(err), "replay %d: %v", i, err)
			}
			_, links, _ := countRetryCleanupRows(t, dbs[0])
			require.Equal(t, 0, links, "no replay may re-create the deleted link")
		})
	}
}
