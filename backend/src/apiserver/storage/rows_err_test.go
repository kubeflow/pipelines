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
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/require"
)

type errStreamConnector struct {
	cols      []string
	firstRow  []driver.Value
	streamErr error
}

func (c *errStreamConnector) Connect(context.Context) (driver.Conn, error) {
	return &errStreamConn{cols: c.cols, firstRow: c.firstRow, streamErr: c.streamErr}, nil
}

func (c *errStreamConnector) Driver() driver.Driver { return nil }

type errStreamConn struct {
	cols      []string
	firstRow  []driver.Value
	streamErr error
}

func (c *errStreamConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("unused") }
func (c *errStreamConn) Close() error                        { return nil }
func (c *errStreamConn) Begin() (driver.Tx, error)           { return errStreamTx{}, nil }

func (c *errStreamConn) QueryContext(_ context.Context, _ string, _ []driver.NamedValue) (driver.Rows, error) {
	return &errStreamRows{
		cols:      c.cols,
		firstRow:  c.firstRow,
		streamErr: c.streamErr,
	}, nil
}

type errStreamTx struct{}

func (errStreamTx) Commit() error   { return nil }
func (errStreamTx) Rollback() error { return nil }

type errStreamRows struct {
	cols         []string
	firstRow     []driver.Value
	emittedFirst bool
	streamErr    error
}

func (r *errStreamRows) Columns() []string { return r.cols }
func (r *errStreamRows) Close() error      { return nil }
func (r *errStreamRows) Next(dest []driver.Value) error {
	if r.firstRow != nil && !r.emittedFirst {
		copy(dest, r.firstRow)
		r.emittedFirst = true
		return nil
	}
	return r.streamErr
}

func newErrStreamDB(t *testing.T, cols []string, firstRow []driver.Value, streamErr error) *sql.DB {
	t.Helper()
	db := sql.OpenDB(&errStreamConnector{
		cols:      cols,
		firstRow:  firstRow,
		streamErr: streamErr,
	})
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func queryErrStreamRows(t *testing.T, cols []string, firstRow []driver.Value, streamErr error) *sql.Rows {
	t.Helper()
	db := newErrStreamDB(t, cols, firstRow, streamErr)
	rows, err := db.Query("SELECT 1")
	require.NoError(t, err)
	t.Cleanup(func() { _ = rows.Close() })
	return rows
}

func TestStoragePropagatesRowsErr(t *testing.T) {
	streamErr := errors.New("simulated row stream failure")
	fakeDB, sqliteDialect := NewFakeDBOrFatal()
	defer fakeDB.Close()

	fakeTime := util.NewFakeTimeForEpoch()
	fakeUUID := util.NewFakeUUIDGeneratorOrFatal(DefaultFakeUUID, nil)

	t.Run("DBStatusStore.HaveSamplesLoaded", func(t *testing.T) {
		db := newErrStreamDB(t, []string{"HaveSamplesLoaded"}, nil, streamErr)
		store := &DBStatusStore{db: db, dbDialect: sqliteDialect}
		_, err := store.HaveSamplesLoaded()
		require.ErrorContains(t, err, streamErr.Error())
	})

	t.Run("DBStatusStore.InitializeDBStatusTable", func(t *testing.T) {
		db := newErrStreamDB(t, []string{"HaveSamplesLoaded"}, nil, streamErr)
		_, err := NewDBStatusStore(db, sqliteDialect)
		require.ErrorContains(t, err, streamErr.Error())
	})

	t.Run("DefaultExperimentStore.GetDefaultExperimentId", func(t *testing.T) {
		db := newErrStreamDB(t, []string{"DefaultExperimentId"}, nil, streamErr)
		store := &DefaultExperimentStore{db: db, dbDialect: sqliteDialect}
		_, err := store.GetDefaultExperimentId()
		require.ErrorContains(t, err, streamErr.Error())
	})

	t.Run("DefaultExperimentStore.initializeDefaultExperimentTable", func(t *testing.T) {
		db := newErrStreamDB(t, []string{"DefaultExperimentId"}, nil, streamErr)
		_, err := NewDefaultExperimentStore(db, sqliteDialect)
		require.ErrorContains(t, err, streamErr.Error())
	})

	t.Run("RunStore.GetRunByRecurringRunIDAndDisplayName", func(t *testing.T) {
		db := newErrStreamDB(t, []string{"UUID"}, nil, streamErr)
		store := NewRunStore(db, fakeTime, sqliteDialect)
		_, err := store.GetRunByRecurringRunIDAndDisplayName("job-1", "run-1")
		require.ErrorContains(t, err, streamErr.Error())
	})

	t.Run("ArtifactStore.getArtifactTaskByUniqueLink", func(t *testing.T) {
		db := newErrStreamDB(t, []string{"UUID", "ArtifactID", "TaskID", "Type", "Iteration", "RunUUID", "Producer", "ArtifactKey"}, nil, streamErr)
		store := NewArtifactStore(db, fakeTime, fakeUUID, sqliteDialect)
		_, err := store.getArtifactTaskByUniqueLink(&model.ArtifactTask{
			ArtifactID:  "art-1",
			TaskID:      "task-1",
			Type:        model.IOType(1),
			Iteration:   0,
			ArtifactKey: "key-1",
		})
		require.ErrorContains(t, err, streamErr.Error())
	})

	t.Run("PipelineStore.GetPipelineTags", func(t *testing.T) {
		db := newErrStreamDB(t, []string{"TagKey", "TagValue"}, nil, streamErr)
		store := NewPipelineStore(db, fakeTime, fakeUUID, sqliteDialect)
		_, err := store.GetPipelineTags("pipe-1")
		require.ErrorContains(t, err, streamErr.Error())
	})

	t.Run("PipelineStore.GetPipelineTagsForPipelines", func(t *testing.T) {
		db := newErrStreamDB(t, []string{"PipelineId", "TagKey", "TagValue"}, nil, streamErr)
		store := NewPipelineStore(db, fakeTime, fakeUUID, sqliteDialect)
		_, err := store.GetPipelineTagsForPipelines([]string{"pipe-1"})
		require.ErrorContains(t, err, streamErr.Error())
	})

	t.Run("ExperimentStore.scanRows mid-stream", func(t *testing.T) {
		rows := queryErrStreamRows(t,
			[]string{"UUID", "Name", "Description", "CreatedAtInSec", "LastRunCreatedAtInSec", "Namespace", "StorageState"},
			[]driver.Value{"exp-1", "name", "desc", int64(1), int64(1), "ns", "AVAILABLE"},
			streamErr,
		)
		store := &ExperimentStore{dbDialect: sqliteDialect}
		_, err := store.scanRows(rows)
		require.ErrorContains(t, err, streamErr.Error())
	})

	t.Run("ArtifactStore.scanRows mid-stream", func(t *testing.T) {
		rows := queryErrStreamRows(t,
			[]string{"UUID", "Namespace", "Type", "URI", "URIHash", "Name", "Description", "CreatedAtInSec", "LastUpdateInSec", "Metadata", "NumberValue", "IdentityKey"},
			[]driver.Value{"art-1", "ns", int64(1), "gs://b/k", "h", "n", "d", int64(1), int64(1), nil, nil, nil},
			streamErr,
		)
		store := NewArtifactStore(nil, fakeTime, fakeUUID, sqliteDialect)
		_, err := store.scanRows(rows)
		require.ErrorContains(t, err, streamErr.Error())
	})

	t.Run("ArtifactTaskStore.scanRows mid-stream", func(t *testing.T) {
		rows := queryErrStreamRows(t,
			[]string{"UUID", "ArtifactID", "TaskID", "Type", "Iteration", "RunUUID", "Producer", "ArtifactKey"},
			[]driver.Value{"at-1", "art-1", "task-1", int64(1), int64(0), "run-1", nil, "key"},
			streamErr,
		)
		store := NewArtifactTaskStore(nil, fakeUUID, sqliteDialect)
		_, err := store.scanRows(rows)
		require.ErrorContains(t, err, streamErr.Error())
	})

	t.Run("JobStore.scanRows", func(t *testing.T) {
		rows := queryErrStreamRows(t, []string{"UUID"}, nil, streamErr)
		store := NewJobStore(nil, fakeTime, nil, sqliteDialect)
		_, err := store.scanRows(rows)
		require.ErrorContains(t, err, streamErr.Error())
	})

	t.Run("PipelineStore.scanPipelinesRows mid-stream", func(t *testing.T) {
		rows := queryErrStreamRows(t,
			[]string{"UUID", "CreatedAtInSec", "Name", "DisplayName", "Description", "Status", "Namespace"},
			[]driver.Value{"p-1", int64(1), "name", "display", "desc", "READY", "ns"},
			streamErr,
		)
		store := NewPipelineStore(nil, fakeTime, fakeUUID, sqliteDialect)
		_, err := store.scanPipelinesRows(rows)
		require.ErrorContains(t, err, streamErr.Error())
	})

	t.Run("PipelineStore.scanPipelineVersionsRows mid-stream", func(t *testing.T) {
		rows := queryErrStreamRows(t,
			[]string{"UUID", "CreatedAtInSec", "Name", "DisplayName", "Parameters", "PipelineId", "Status", "CodeSourceUrl", "Description", "PipelineSpec", "PipelineSpecURI"},
			[]driver.Value{"pv-1", int64(1), "name", "display", "[]", "p-1", "READY", "", "desc", "", ""},
			streamErr,
		)
		store := NewPipelineStore(nil, fakeTime, fakeUUID, sqliteDialect)
		_, err := store.scanPipelineVersionsRows(rows)
		require.ErrorContains(t, err, streamErr.Error())
	})

	t.Run("ResourceReferenceStore.scanRows mid-stream", func(t *testing.T) {
		rows := queryErrStreamRows(t,
			[]string{"ResourceUUID", "ResourceType", "ReferenceUUID", "ReferenceName", "ReferenceType", "Relationship", "Payload"},
			[]driver.Value{"r-1", "Run", "e-1", "exp", "Experiment", "Owner", "{}"},
			streamErr,
		)
		store := NewResourceReferenceStore(nil, nil, sqliteDialect)
		_, err := store.scanRows(rows)
		require.ErrorContains(t, err, streamErr.Error())
	})

	t.Run("TaskStore.scanRows", func(t *testing.T) {
		rows := queryErrStreamRows(t, []string{"UUID"}, nil, streamErr)
		store := NewTaskStore(nil, fakeTime, fakeUUID, sqliteDialect)
		_, err := store.scanRows(rows)
		require.ErrorContains(t, err, streamErr.Error())
	})
}
