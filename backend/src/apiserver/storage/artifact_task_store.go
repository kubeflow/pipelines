// Copyright 2025 The Kubeflow Authors
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

// Package storage contains the implementation of the storage interface.
package storage

import (
	"database/sql"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/golang/glog"
	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

const artifactTaskTableName = "artifact_tasks"

var artifactTaskColumns = []string{
	"UUID",
	"ArtifactID",
	"TaskID",
	"Type",
	"Iteration",
	"RunUUID",
	"Producer",
	"ArtifactKey",
}

type ArtifactTaskStoreInterface interface {
	// CreateArtifactTask Create an artifact-task relationship entry in the database.
	CreateArtifactTask(artifactTask *model.ArtifactTask) (*model.ArtifactTask, error)

	// CreateArtifactTasks Create multiple artifact-task relationships in a single transaction.
	CreateArtifactTasks(artifactTasks []*model.ArtifactTask) ([]*model.ArtifactTask, error)

	// GetArtifactTask Fetches an artifact-task relationship with a given id.
	GetArtifactTask(id string) (*model.ArtifactTask, error)

	// ListArtifactTasks fetches artifact-task relationships for given filtering and listing options.
	// filterContexts supports multiple filters: ArtifactID, TaskID, RunUUID.
	// ioType optionally filters by IOType (pass nil to skip filtering by type).
	// It returns the current page of artifact-task rows, the total count across all pages,
	// the next page token, and an error.
	ListArtifactTasks(filterContexts []*model.FilterContext, ioType *model.IOType, opts *list.Options) ([]*model.ArtifactTask, int, string, error)

	// DeleteOutputArtifactTasksByTaskIDs deletes attempt-local output links for the given tasks.
	DeleteOutputArtifactTasksByTaskIDs(taskIDs []string) error

	// DeleteInputArtifactTasksByTaskIDs deletes attempt-local input links for the given tasks.
	// Required on retry so CreateArtifactTasks can recreate UniqueLink rows for the new attempt.
	DeleteInputArtifactTasksByTaskIDs(taskIDs []string) error
}

type ArtifactTaskStore struct {
	dbDialect dialect.DBDialect
	db        *sql.DB
	uuid      util.UUIDGeneratorInterface
}

// NewArtifactTaskStore creates a new ArtifactTaskStore.
func NewArtifactTaskStore(db *sql.DB, uuid util.UUIDGeneratorInterface, d dialect.DBDialect) *ArtifactTaskStore {
	return &ArtifactTaskStore{
		dbDialect: d,
		db:        db,
		uuid:      uuid,
	}
}

func (s *ArtifactTaskStore) CreateArtifactTask(artifactTask *model.ArtifactTask) (*model.ArtifactTask, error) {
	return createArtifactTaskWithExecutor(s.db.Exec, s.uuid, artifactTask, s.dbDialect)
}

func createArtifactTaskWithExecutor(exec func(string, ...any) (sql.Result, error), uuid util.UUIDGeneratorInterface, artifactTask *model.ArtifactTask, d dialect.DBDialect) (*model.ArtifactTask, error) {
	q := d.QuoteIdentifier
	qb := d.QueryBuilder()
	// Set up UUID for artifact-task relationship.
	newArtifactTask := *artifactTask
	if err := newArtifactTask.SyncIterationFromProducer(); err != nil {
		return nil, util.NewInternalServerError(err, "Failed to derive artifact-task iteration: %v", err.Error())
	}
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create an artifact-task id")
	}
	newArtifactTask.UUID = id.String()

	// Serialize Producer to JSON if present
	producerValue, err := newArtifactTask.Producer.Value()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to marshal producer JSON: %v", err.Error())
	}

	sql, args, err := qb.
		Insert(q(artifactTaskTableName)).
		SetMap(
			sq.Eq{
				q("UUID"):        newArtifactTask.UUID,
				q("ArtifactID"):  newArtifactTask.ArtifactID,
				q("TaskID"):      newArtifactTask.TaskID,
				q("Type"):        newArtifactTask.Type,
				q("Iteration"):   newArtifactTask.Iteration,
				q("RunUUID"):     newArtifactTask.RunUUID,
				q("Producer"):    producerValue,
				q("ArtifactKey"): newArtifactTask.ArtifactKey,
			},
		).
		ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create query to insert artifact-task to artifact_tasks table: %v",
			err.Error())
	}

	_, err = exec(sql, args...)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to add artifact-task to artifact_tasks table: %v",
			err.Error())
	}

	return &newArtifactTask, nil
}

func (s *ArtifactTaskStore) CreateArtifactTasks(artifactTasks []*model.ArtifactTask) ([]*model.ArtifactTask, error) {
	q := s.dbDialect.QuoteIdentifier
	qb := s.dbDialect.QueryBuilder()
	if len(artifactTasks) == 0 {
		return []*model.ArtifactTask{}, nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to start transaction for creating artifact-tasks")
	}
	defer func() {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			glog.Warningf("Failed to rollback artifact-task create transaction: %v", rbErr)
		}
	}()

	var newArtifactTasks []*model.ArtifactTask
	for _, artifactTask := range artifactTasks {
		newArtifactTask := *artifactTask
		if err := newArtifactTask.SyncIterationFromProducer(); err != nil {
			return nil, util.NewInternalServerError(err, "Failed to derive artifact-task iteration: %v", err.Error())
		}
		id, err := s.uuid.NewRandom()
		if err != nil {
			return nil, util.NewInternalServerError(err, "Failed to create an artifact-task id")
		}
		newArtifactTask.UUID = id.String()

		// Serialize Producer to JSON if present
		producerValue, err := newArtifactTask.Producer.Value()
		if err != nil {
			return nil, util.NewInternalServerError(err, "Failed to marshal producer JSON: %v", err.Error())
		}

		toSQL, args, err := qb.
			Insert(q(artifactTaskTableName)).
			SetMap(
				sq.Eq{
					q("UUID"):        newArtifactTask.UUID,
					q("ArtifactID"):  newArtifactTask.ArtifactID,
					q("TaskID"):      newArtifactTask.TaskID,
					q("Type"):        newArtifactTask.Type,
					q("Iteration"):   newArtifactTask.Iteration,
					q("RunUUID"):     newArtifactTask.RunUUID,
					q("Producer"):    producerValue,
					q("ArtifactKey"): newArtifactTask.ArtifactKey,
				},
			).
			ToSql()
		if err != nil {
			return nil, util.NewInternalServerError(err, "Failed to create query to insert artifact-task: %v", err.Error())
		}

		_, err = tx.Exec(toSQL, args...)
		if err != nil {
			return nil, util.NewInternalServerError(err, "Failed to add artifact-task: %v", err.Error())
		}

		newArtifactTasks = append(newArtifactTasks, &newArtifactTask)
	}

	err = tx.Commit()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to commit transaction for creating artifact-tasks")
	}

	return newArtifactTasks, nil
}

func (s *ArtifactTaskStore) scanRows(rows *sql.Rows) ([]*model.ArtifactTask, error) {
	var artifactTasks []*model.ArtifactTask
	for rows.Next() {
		var uuid, artifactID, taskID string
		var runUUID, key string
		var ioType int32
		var iteration int64
		var producer model.JSONData

		err := rows.Scan(
			&uuid,
			&artifactID,
			&taskID,
			&ioType,
			&iteration,
			&runUUID,
			&producer,
			&key,
		)
		if err != nil {
			return nil, err
		}

		artifactTask := &model.ArtifactTask{
			UUID:        uuid,
			ArtifactID:  artifactID,
			TaskID:      taskID,
			Type:        model.IOType(ioType),
			Iteration:   iteration,
			RunUUID:     runUUID,
			Producer:    producer,
			ArtifactKey: key,
		}
		artifactTasks = append(artifactTasks, artifactTask)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return artifactTasks, nil
}

// applyFilterContextsToQuery applies multiple filter contexts to the query builder
// Supports filtering by multiple artifact_ids, task_ids, and run_ids simultaneously
func (s *ArtifactTaskStore) applyFilterContextsToQuery(sqlBuilder sq.SelectBuilder, filterContexts []*model.FilterContext) (sq.SelectBuilder, error) {
	q := s.dbDialect.QuoteIdentifier
	var artifactIDs []string
	var taskIDs []string
	var runIDs []string

	// Collect all filter values by type
	for _, filterContext := range filterContexts {
		if filterContext == nil || filterContext.ReferenceKey == nil {
			continue
		}

		switch filterContext.Type {
		case model.ArtifactResourceType:
			artifactIDs = append(artifactIDs, filterContext.ID)
		case model.TaskResourceType:
			taskIDs = append(taskIDs, filterContext.ID)
		case model.RunResourceType:
			runIDs = append(runIDs, filterContext.ID)
		default:
			return sqlBuilder, util.NewInvalidInputError("Unsupported artifact-task filter type %q", filterContext.Type)
		}
	}

	// Apply artifact ID filters (OR within artifact IDs)
	if len(artifactIDs) > 0 {
		if len(artifactIDs) == 1 {
			sqlBuilder = sqlBuilder.Where(sq.Eq{q("artifact_tasks") + "." + q("ArtifactID"): artifactIDs[0]})
		} else {
			sqlBuilder = sqlBuilder.Where(sq.Eq{q("artifact_tasks") + "." + q("ArtifactID"): artifactIDs})
		}
	}

	// Apply task ID filters (OR within task IDs)
	if len(taskIDs) > 0 {
		if len(taskIDs) == 1 {
			sqlBuilder = sqlBuilder.Where(sq.Eq{q("artifact_tasks") + "." + q("TaskID"): taskIDs[0]})
		} else {
			sqlBuilder = sqlBuilder.Where(sq.Eq{q("artifact_tasks") + "." + q("TaskID"): taskIDs})
		}
	}

	// Apply run ID filters (OR within run IDs) now directly on artifact_tasks
	if len(runIDs) > 0 {
		if len(runIDs) == 1 {
			sqlBuilder = sqlBuilder.Where(sq.Eq{q("artifact_tasks") + "." + q("RunUUID"): runIDs[0]})
		} else {
			sqlBuilder = sqlBuilder.Where(sq.Eq{q("artifact_tasks") + "." + q("RunUUID"): runIDs})
		}
	}

	return sqlBuilder, nil
}

func (s *ArtifactTaskStore) ListArtifactTasks(filterContexts []*model.FilterContext, ioType *model.IOType, opts *list.Options) ([]*model.ArtifactTask, int, string, error) {
	q := s.dbDialect.QuoteIdentifier
	qb := s.dbDialect.QueryBuilder()
	errorF := func(err error) ([]*model.ArtifactTask, int, string, error) {
		return nil, 0, "", util.NewInternalServerError(err, "Failed to list artifact-tasks: %v", err)
	}

	// SQL for getting the filtered and paginated rows
	sqlBuilder := qb.Select(dialect.QuoteAll(q, artifactTaskColumns)...).From(q(artifactTaskTableName))
	sqlBuilder, err := s.applyFilterContextsToQuery(sqlBuilder, filterContexts)
	if err != nil {
		return errorF(err)
	}

	// Apply IOType filter if provided
	if ioType != nil {
		sqlBuilder = sqlBuilder.Where(sq.Eq{q("artifact_tasks") + "." + q("Type"): *ioType})
	}

	sqlBuilder = opts.AddFilterToSelect(sqlBuilder, q)

	rowsSQL, rowsArgs, err := opts.AddPaginationToSelect(sqlBuilder, q, s.dbDialect.StringCollation()).ToSql()
	if err != nil {
		return errorF(err)
	}

	// SQL for getting total size
	countBuilder := qb.Select("count(*)").From(q(artifactTaskTableName))
	countBuilder, err = s.applyFilterContextsToQuery(countBuilder, filterContexts)
	if err != nil {
		return errorF(err)
	}

	// Apply IOType filter if provided
	if ioType != nil {
		countBuilder = countBuilder.Where(sq.Eq{q("artifact_tasks") + "." + q("Type"): *ioType})
	}

	sizeSQL, sizeArgs, err := opts.AddFilterToSelect(countBuilder, q).ToSql()
	if err != nil {
		return errorF(err)
	}

	// Use a transaction to make sure we're returning the totalSize of the same rows queried
	tx, err := s.db.Begin()
	if err != nil {
		return errorF(err)
	}
	rollback := func() {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			glog.Warningf("Failed to rollback artifact-task list transaction: %v", rbErr)
		}
	}

	rows, err := tx.Query(rowsSQL, rowsArgs...)
	if err != nil {
		rollback()
		return errorF(err)
	}
	if err := rows.Err(); err != nil {
		rollback()
		return errorF(err)
	}
	artifactTasks, err := s.scanRows(rows)
	if err != nil {
		rollback()
		return errorF(err)
	}
	defer rows.Close()

	sizeRow, err := tx.Query(sizeSQL, sizeArgs...)
	if err != nil {
		rollback()
		return errorF(err)
	}
	if err := sizeRow.Err(); err != nil {
		rollback()
		return errorF(err)
	}
	totalSize, err := list.ScanRowToTotalSize(sizeRow)
	if err != nil {
		rollback()
		return errorF(err)
	}
	defer sizeRow.Close()

	err = tx.Commit()
	if err != nil {
		return errorF(err)
	}

	if len(artifactTasks) <= opts.PageSize {
		return artifactTasks, totalSize, "", nil
	}

	npt, err := opts.NextPageToken(artifactTasks[opts.PageSize])
	if err != nil {
		return errorF(err)
	}
	return artifactTasks[:opts.PageSize], totalSize, npt, nil
}

func (s *ArtifactTaskStore) GetArtifactTask(id string) (*model.ArtifactTask, error) {
	q := s.dbDialect.QuoteIdentifier
	qb := s.dbDialect.QueryBuilder()
	sql, args, err := qb.
		Select(dialect.QuoteAll(q, artifactTaskColumns)...).
		From(q(artifactTaskTableName)).
		Where(sq.Eq{q("UUID"): id}).
		Limit(1).ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create query to get artifact-task: %v", err.Error())
	}

	r, err := s.db.Query(sql, args...)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to get artifact-task: %v", err.Error())
	}
	defer r.Close()

	artifactTasks, err := s.scanRows(r)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to get artifact-task: %v", err.Error())
	}
	if len(artifactTasks) > 1 {
		return nil, util.NewInternalServerError(errors.New("multiple artifact-task rows found"), "Failed to get artifact-task %s: multiple rows returned", id)
	}
	if len(artifactTasks) == 0 {
		return nil, util.NewResourceNotFoundError("artifact-task", fmt.Sprint(id))
	}

	return artifactTasks[0], nil
}

func (s *ArtifactTaskStore) DeleteOutputArtifactTasksByTaskIDs(taskIDs []string) error {
	if len(taskIDs) == 0 {
		return nil
	}
	outputLinkTypes := []model.IOType{
		model.IOType(apiv2.IOType_OUTPUT),
		model.IOType(apiv2.IOType_ITERATOR_OUTPUT),
		model.IOType(apiv2.IOType_ONE_OF_OUTPUT),
		model.IOType(apiv2.IOType_TASK_FINAL_STATUS_OUTPUT),
	}
	return s.deleteArtifactTasksByTaskIDs(taskIDs, outputLinkTypes, "output")
}

// deleteArtifactTasksByTaskIDs deletes links of the given types in one
// transaction that first locks the owning run rows. Replayed artifact writes
// lock the run before the artifact and link rows, so this keeps a consistent
// lock order (run, then links) and avoids index-order deadlocks on MySQL.
func (s *ArtifactTaskStore) deleteArtifactTasksByTaskIDs(taskIDs []string, linkTypes []model.IOType, kind string) error {
	q := s.dbDialect.QuoteIdentifier
	qb := s.dbDialect.QueryBuilder()

	tx, err := s.db.Begin()
	if err != nil {
		return util.NewInternalServerError(err, "Failed to start transaction to delete %s artifact-tasks", kind)
	}
	defer tx.Rollback()

	runSQL, runArgs, err := qb.
		Select(q("RunUUID")).
		Distinct().
		From(q(artifactTaskTableName)).
		Where(sq.Eq{q("TaskID"): taskIDs}).
		Where(sq.Eq{q("Type"): linkTypes}).
		ToSql()
	if err != nil {
		return util.NewInternalServerError(err, "Failed to create query to read runs of %s artifact-tasks: %v", kind, err.Error())
	}
	rows, err := tx.Query(runSQL, runArgs...)
	if err != nil {
		return util.NewInternalServerError(err, "Failed to read runs of %s artifact-tasks: %v", kind, err.Error())
	}
	var runUUIDs []string
	for rows.Next() {
		var runUUID string
		if err := rows.Scan(&runUUID); err != nil {
			rows.Close()
			return util.NewInternalServerError(err, "Failed to scan run of %s artifact-tasks: %v", kind, err.Error())
		}
		runUUIDs = append(runUUIDs, runUUID)
	}
	rowsErr := rows.Err()
	rows.Close()
	if rowsErr != nil {
		return util.NewInternalServerError(rowsErr, "Failed to read runs of %s artifact-tasks: %v", kind, rowsErr.Error())
	}
	if len(runUUIDs) == 0 {
		return nil
	}

	lockSQL, lockArgs, err := qb.
		Select(q("UUID")).
		From(q("run_details")).
		Where(sq.Eq{q("UUID"): runUUIDs}).
		OrderBy(q("UUID")).
		ToSql()
	if err != nil {
		return util.NewInternalServerError(err, "Failed to create query to lock runs of %s artifact-tasks: %v", kind, err.Error())
	}
	lockRows, err := tx.Query(s.dbDialect.SelectForUpdate(lockSQL), lockArgs...)
	if err != nil {
		return util.NewInternalServerError(err, "Failed to lock runs of %s artifact-tasks: %v", kind, err.Error())
	}
	for lockRows.Next() {
		var lockedRunUUID string
		if err := lockRows.Scan(&lockedRunUUID); err != nil {
			lockRows.Close()
			return util.NewInternalServerError(err, "Failed to scan locked run of %s artifact-tasks: %v", kind, err.Error())
		}
	}
	lockErr := lockRows.Err()
	lockRows.Close()
	if lockErr != nil {
		return util.NewInternalServerError(lockErr, "Failed to lock runs of %s artifact-tasks: %v", kind, lockErr.Error())
	}

	deleteSQL, deleteArgs, err := qb.
		Delete(q(artifactTaskTableName)).
		Where(sq.Eq{q("TaskID"): taskIDs}).
		Where(sq.Eq{q("Type"): linkTypes}).
		ToSql()
	if err != nil {
		return util.NewInternalServerError(err, "Failed to create query to delete %s artifact-tasks: %v", kind, err.Error())
	}
	if _, err := tx.Exec(deleteSQL, deleteArgs...); err != nil {
		return util.NewInternalServerError(err, "Failed to delete %s artifact-tasks: %v", kind, err.Error())
	}
	if err := tx.Commit(); err != nil {
		return util.NewInternalServerError(err, "Failed to commit deletion of %s artifact-tasks: %v", kind, err.Error())
	}
	return nil
}

func (s *ArtifactTaskStore) DeleteInputArtifactTasksByTaskIDs(taskIDs []string) error {
	if len(taskIDs) == 0 {
		return nil
	}
	inputLinkTypes := []model.IOType{
		model.IOType(apiv2.IOType_COMPONENT_INPUT),
		model.IOType(apiv2.IOType_COLLECTED_INPUTS),
		model.IOType(apiv2.IOType_TASK_OUTPUT_INPUT),
		model.IOType(apiv2.IOType_RUNTIME_VALUE_INPUT),
		model.IOType(apiv2.IOType_ITERATOR_INPUT),
		model.IOType(apiv2.IOType_ITERATOR_INPUT_RAW),
		model.IOType(apiv2.IOType_COMPONENT_DEFAULT_INPUT),
	}
	return s.deleteArtifactTasksByTaskIDs(taskIDs, inputLinkTypes, "input")
}
