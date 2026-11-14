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
	"io"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/require"
)

type errStreamConnector struct {
	cols        []string
	firstRow    []driver.Value
	streamErr   error
	execQueries *[]string
}

func (c *errStreamConnector) Connect(context.Context) (driver.Conn, error) {
	return &errStreamConn{
		cols:        c.cols,
		firstRow:    c.firstRow,
		streamErr:   c.streamErr,
		execQueries: c.execQueries,
	}, nil
}

func (c *errStreamConnector) Driver() driver.Driver { return nil }

type errStreamConn struct {
	cols        []string
	firstRow    []driver.Value
	streamErr   error
	execQueries *[]string
}

func (c *errStreamConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("unused") }
func (c *errStreamConn) Close() error                        { return nil }
func (c *errStreamConn) Begin() (driver.Tx, error)           { return errStreamTx{}, nil }

func (c *errStreamConn) ExecContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Result, error) {
	if c.execQueries != nil {
		*c.execQueries = append(*c.execQueries, query)
	}
	return driver.RowsAffected(0), nil
}

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
	if r.streamErr != nil {
		return r.streamErr
	}
	return io.EOF
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

func newErrStreamDBWithExecLog(t *testing.T, cols []string, firstRow []driver.Value, streamErr error) (*sql.DB, *[]string) {
	t.Helper()
	var execQueries []string
	db := sql.OpenDB(&errStreamConnector{
		cols:        cols,
		firstRow:    firstRow,
		streamErr:   streamErr,
		execQueries: &execQueries,
	})
	t.Cleanup(func() { _ = db.Close() })
	return db, &execQueries
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
		loaded, err := store.HaveSamplesLoaded()
		require.ErrorContains(t, err, "Error when getting load sample status")
		require.ErrorContains(t, err, streamErr.Error())
		require.False(t, loaded)

		emptyDB := newErrStreamDB(t, []string{"HaveSamplesLoaded"}, nil, nil)
		emptyStore := &DBStatusStore{db: emptyDB, dbDialect: sqliteDialect}
		loaded, err = emptyStore.HaveSamplesLoaded()
		require.NoError(t, err)
		require.False(t, loaded)
	})

	t.Run("DBStatusStore.InitializeDBStatusTable", func(t *testing.T) {
		db, execQueries := newErrStreamDBWithExecLog(t, []string{"HaveSamplesLoaded"}, nil, streamErr)
		_, err := NewDBStatusStore(db, sqliteDialect)
		require.ErrorContains(t, err, "Failed to load database status")
		require.ErrorContains(t, err, streamErr.Error())
		require.Empty(t, *execQueries, "initialization must not insert when status query fails")

		emptyDB, emptyExecQueries := newErrStreamDBWithExecLog(t, []string{"HaveSamplesLoaded"}, nil, nil)
		_, err = NewDBStatusStore(emptyDB, sqliteDialect)
		require.NoError(t, err)
		require.Len(t, *emptyExecQueries, 1)
		require.Contains(t, (*emptyExecQueries)[0], `INSERT INTO "db_statuses"`)

		popDB, popExecQueries := newErrStreamDBWithExecLog(t, []string{"HaveSamplesLoaded"}, []driver.Value{false}, nil)
		_, err = NewDBStatusStore(popDB, sqliteDialect)
		require.NoError(t, err)
		require.Empty(t, *popExecQueries)
	})

	t.Run("DefaultExperimentStore.GetDefaultExperimentId", func(t *testing.T) {
		db := newErrStreamDB(t, []string{"DefaultExperimentId"}, nil, streamErr)
		store := &DefaultExperimentStore{db: db, dbDialect: sqliteDialect}
		id, err := store.GetDefaultExperimentId()
		require.ErrorContains(t, err, "Error when getting default experiment ID")
		require.ErrorContains(t, err, streamErr.Error())
		require.Empty(t, id)

		emptyDB := newErrStreamDB(t, []string{"DefaultExperimentId"}, nil, nil)
		emptyStore := &DefaultExperimentStore{db: emptyDB, dbDialect: sqliteDialect}
		id, err = emptyStore.GetDefaultExperimentId()
		require.NoError(t, err)
		require.Empty(t, id)
	})

	t.Run("DefaultExperimentStore.initializeDefaultExperimentTable", func(t *testing.T) {
		db, execQueries := newErrStreamDBWithExecLog(t, []string{"DefaultExperimentId"}, nil, streamErr)
		_, err := NewDefaultExperimentStore(db, sqliteDialect)
		require.ErrorContains(t, err, "Failed to get default experiment")
		require.ErrorContains(t, err, streamErr.Error())
		require.Empty(t, *execQueries, "initialization must not insert when default experiment query fails")

		emptyDB, emptyExecQueries := newErrStreamDBWithExecLog(t, []string{"DefaultExperimentId"}, nil, nil)
		_, err = NewDefaultExperimentStore(emptyDB, sqliteDialect)
		require.NoError(t, err)
		require.Len(t, *emptyExecQueries, 1)
		require.Contains(t, (*emptyExecQueries)[0], `INSERT INTO "default_experiments"`)

		popDB, popExecQueries := newErrStreamDBWithExecLog(t, []string{"DefaultExperimentId"}, []driver.Value{"default"}, nil)
		_, err = NewDefaultExperimentStore(popDB, sqliteDialect)
		require.NoError(t, err)
		require.Empty(t, *popExecQueries)
	})

	t.Run("RunStore.GetRunByRecurringRunIDAndDisplayName", func(t *testing.T) {
		db := newErrStreamDB(t, []string{"UUID"}, nil, streamErr)
		store := NewRunStore(db, fakeTime, sqliteDialect)
		uuid, err := store.GetRunByRecurringRunIDAndDisplayName("job-1", "run-1")
		require.ErrorContains(t, err, "Failed to check for existing run")
		require.ErrorContains(t, err, streamErr.Error())
		require.Empty(t, uuid)

		emptyDB := newErrStreamDB(t, []string{"UUID"}, nil, nil)
		emptyStore := NewRunStore(emptyDB, fakeTime, sqliteDialect)
		uuid, err = emptyStore.GetRunByRecurringRunIDAndDisplayName("job-1", "run-1")
		require.NoError(t, err)
		require.Empty(t, uuid)
	})

	t.Run("ArtifactStore.getArtifactTaskByUniqueLink", func(t *testing.T) {
		cols := []string{"UUID", "ArtifactID", "TaskID", "Type", "Iteration", "RunUUID", "Producer", "ArtifactKey"}
		link := &model.ArtifactTask{
			ArtifactID:  "art-1",
			TaskID:      "task-1",
			Type:        model.IOType(1),
			Iteration:   0,
			ArtifactKey: "key-1",
		}
		db := newErrStreamDB(t, cols, nil, streamErr)
		store := NewArtifactStore(db, fakeTime, fakeUUID, sqliteDialect)
		at, err := store.getArtifactTaskByUniqueLink(link)
		require.ErrorContains(t, err, "Failed to iterate artifact-task by unique link")
		require.ErrorContains(t, err, streamErr.Error())
		require.Nil(t, at)

		emptyDB := newErrStreamDB(t, cols, nil, nil)
		emptyStore := NewArtifactStore(emptyDB, fakeTime, fakeUUID, sqliteDialect)
		at, err = emptyStore.getArtifactTaskByUniqueLink(link)
		require.NoError(t, err)
		require.Nil(t, at)
	})

	t.Run("PipelineStore.GetPipelineTags", func(t *testing.T) {
		cols := []string{"TagKey", "TagValue"}
		db := newErrStreamDB(t, cols, nil, streamErr)
		store := NewPipelineStore(db, fakeTime, fakeUUID, sqliteDialect)
		_, err := store.GetPipelineTags("pipe-1")
		require.ErrorContains(t, err, "Failed to iterate tag rows")
		require.ErrorContains(t, err, streamErr.Error())

		midDB := newErrStreamDB(t, cols, []driver.Value{"k1", "v1"}, streamErr)
		midStore := NewPipelineStore(midDB, fakeTime, fakeUUID, sqliteDialect)
		_, err = midStore.GetPipelineTags("pipe-1")
		require.ErrorContains(t, err, streamErr.Error())

		emptyDB := newErrStreamDB(t, cols, nil, nil)
		emptyStore := NewPipelineStore(emptyDB, fakeTime, fakeUUID, sqliteDialect)
		tags, err := emptyStore.GetPipelineTags("pipe-1")
		require.NoError(t, err)
		require.Empty(t, tags)
	})

	t.Run("PipelineStore.GetPipelineTagsForPipelines", func(t *testing.T) {
		cols := []string{"PipelineId", "TagKey", "TagValue"}
		db := newErrStreamDB(t, cols, nil, streamErr)
		store := NewPipelineStore(db, fakeTime, fakeUUID, sqliteDialect)
		_, err := store.GetPipelineTagsForPipelines([]string{"pipe-1"})
		require.ErrorContains(t, err, "Failed to iterate tag rows")
		require.ErrorContains(t, err, streamErr.Error())

		midDB := newErrStreamDB(t, cols, []driver.Value{"pipe-1", "k1", "v1"}, streamErr)
		midStore := NewPipelineStore(midDB, fakeTime, fakeUUID, sqliteDialect)
		_, err = midStore.GetPipelineTagsForPipelines([]string{"pipe-1"})
		require.ErrorContains(t, err, streamErr.Error())

		emptyDB := newErrStreamDB(t, cols, nil, nil)
		emptyStore := NewPipelineStore(emptyDB, fakeTime, fakeUUID, sqliteDialect)
		tagsMap, err := emptyStore.GetPipelineTagsForPipelines([]string{"pipe-1"})
		require.NoError(t, err)
		require.Empty(t, tagsMap)
	})

	runScanRowsModes := func(t *testing.T, name string, cols []string, firstRow []driver.Value, call func(*sql.Rows) (int, error)) {
		t.Helper()
		t.Run(name+"/immediate", func(t *testing.T) {
			rows := queryErrStreamRows(t, cols, nil, streamErr)
			_, err := call(rows)
			require.ErrorContains(t, err, streamErr.Error())
		})
		t.Run(name+"/mid-stream", func(t *testing.T) {
			rows := queryErrStreamRows(t, cols, firstRow, streamErr)
			_, err := call(rows)
			require.ErrorContains(t, err, streamErr.Error())
		})
		t.Run(name+"/clean-empty", func(t *testing.T) {
			rows := queryErrStreamRows(t, cols, nil, nil)
			count, err := call(rows)
			require.NoError(t, err)
			require.Zero(t, count)
		})
	}

	runScanRowsModes(t, "ExperimentStore.scanRows",
		[]string{"UUID", "Name", "Description", "CreatedAtInSec", "LastRunCreatedAtInSec", "Namespace", "StorageState"},
		[]driver.Value{"exp-1", "name", "desc", int64(1), int64(1), "ns", "AVAILABLE"},
		func(rows *sql.Rows) (int, error) {
			store := &ExperimentStore{dbDialect: sqliteDialect}
			items, err := store.scanRows(rows)
			return len(items), err
		},
	)

	runScanRowsModes(t, "ArtifactStore.scanRows",
		[]string{"UUID", "Namespace", "Type", "URI", "URIHash", "Name", "Description", "CreatedAtInSec", "LastUpdateInSec", "Metadata", "NumberValue", "IdentityKey"},
		[]driver.Value{"art-1", "ns", int64(1), "gs://b/k", "h", "n", "d", int64(1), int64(1), nil, nil, nil},
		func(rows *sql.Rows) (int, error) {
			store := NewArtifactStore(nil, fakeTime, fakeUUID, sqliteDialect)
			items, err := store.scanRows(rows)
			return len(items), err
		},
	)

	runScanRowsModes(t, "ArtifactTaskStore.scanRows",
		[]string{"UUID", "ArtifactID", "TaskID", "Type", "Iteration", "RunUUID", "Producer", "ArtifactKey"},
		[]driver.Value{"at-1", "art-1", "task-1", int64(1), int64(0), "run-1", nil, "key"},
		func(rows *sql.Rows) (int, error) {
			store := NewArtifactTaskStore(nil, fakeUUID, sqliteDialect)
			items, err := store.scanRows(rows)
			return len(items), err
		},
	)

	runScanRowsModes(t, "JobStore.scanRows",
		[]string{
			"UUID", "DisplayName", "Name", "Namespace", "ServiceAccount", "Description",
			"MaxConcurrency", "NoCatchup", "CreatedAtInSec", "UpdatedAtInSec", "Enabled",
			"CronScheduleStartTimeInSec", "CronScheduleEndTimeInSec", "Cron",
			"PeriodicScheduleStartTimeInSec", "PeriodicScheduleEndTimeInSec", "IntervalSecond",
			"PipelineId", "PipelineName", "PipelineSpecManifest", "WorkflowSpecManifest", "Parameters",
			"Conditions", "RuntimeParameters", "PipelineRoot", "ExperimentUUID",
			"PipelineVersionId", "PluginsInput", "refs",
		},
		[]driver.Value{
			"job-1", "display", "name", "ns", "sa", "desc",
			int64(1), false, int64(1), int64(1), true,
			nil, nil, nil,
			nil, nil, nil,
			"p-1", "pname", "", "", "[]",
			"", nil, nil, nil,
			nil, nil, nil,
		},
		func(rows *sql.Rows) (int, error) {
			store := NewJobStore(nil, fakeTime, nil, sqliteDialect)
			items, err := store.scanRows(rows)
			return len(items), err
		},
	)

	runScanRowsModes(t, "PipelineStore.scanPipelinesRows",
		[]string{"UUID", "CreatedAtInSec", "Name", "DisplayName", "Description", "Status", "Namespace"},
		[]driver.Value{"p-1", int64(1), "name", "display", "desc", "READY", "ns"},
		func(rows *sql.Rows) (int, error) {
			store := NewPipelineStore(nil, fakeTime, fakeUUID, sqliteDialect)
			items, err := store.scanPipelinesRows(rows)
			return len(items), err
		},
	)

	runScanRowsModes(t, "PipelineStore.scanPipelineVersionsRows",
		[]string{"UUID", "CreatedAtInSec", "Name", "DisplayName", "Parameters", "PipelineId", "Status", "CodeSourceUrl", "Description", "PipelineSpec", "PipelineSpecURI"},
		[]driver.Value{"pv-1", int64(1), "name", "display", "[]", "p-1", "READY", "", "desc", "", ""},
		func(rows *sql.Rows) (int, error) {
			store := NewPipelineStore(nil, fakeTime, fakeUUID, sqliteDialect)
			items, err := store.scanPipelineVersionsRows(rows)
			return len(items), err
		},
	)

	runScanRowsModes(t, "ResourceReferenceStore.scanRows",
		[]string{"ResourceUUID", "ResourceType", "ReferenceUUID", "ReferenceName", "ReferenceType", "Relationship", "Payload"},
		[]driver.Value{"r-1", "Run", "e-1", "exp", "Experiment", "Owner", "{}"},
		func(rows *sql.Rows) (int, error) {
			store := NewResourceReferenceStore(nil, nil, sqliteDialect)
			items, err := store.scanRows(rows)
			return len(items), err
		},
	)

	runScanRowsModes(t, "TaskStore.scanRows",
		[]string{
			"UUID", "Namespace", "RunUUID", "Pods", "CreatedAtInSec", "StartedInSec", "FinishedInSec",
			"Fingerprint", "Name", "DisplayName", "ParentTaskUUID", "State", "StatusMetadata",
			"StateHistory", "InputParameters", "OutputParameters", "Type", "TypeAttrs",
			"ScopePath", "LogicalKey", "LifecycleMessage", "LifecycleCategory",
		},
		[]driver.Value{
			"task-1", "ns", "run-1", nil, int64(1), nil, nil,
			"fp", "tname", "tdisplay", nil, int64(1), nil,
			nil, nil, nil, int64(1), nil,
			nil, nil, nil, nil,
		},
		func(rows *sql.Rows) (int, error) {
			store := NewTaskStore(nil, fakeTime, fakeUUID, sqliteDialect)
			items, err := store.scanRows(rows)
			return len(items), err
		},
	)
}
