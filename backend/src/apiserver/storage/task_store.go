// Copyright 2021 The Kubeflow Authors
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
// limitations under the License.package storage

package storage

import (
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

const table_name = "tasks"

var taskColumns = []string{
	"UUID",
	"Namespace",
	"PipelineName",
	"RunUUID",
	"MLMDExecutionID",
	"CreatedTimestamp",
	"StartedTimestamp",
	"FinishedTimestamp",
	"Fingerprint",
	"Name",
	"ParentTaskUUID",
	"State",
	"StateHistory",
	"MLMDInputs",
	"MLMDOutputs",
}

type TaskStoreInterface interface {
	// Create a task entry in the database.
	CreateTask(task *model.Task) (*model.Task, error)

	// Fetches a task with a given id.
	GetTask(id string) (*model.Task, error)

	// Fetches tasks for given filtering and listing options.
	ListTasks(filterContext *model.FilterContext, opts *list.Options) ([]*model.Task, int, string, error)
}

type TaskStore struct {
	db   *DB
	time util.TimeInterface
	uuid util.UUIDGeneratorInterface
}

// NewTaskStore creates a new TaskStore.
func NewTaskStore(db *DB, time util.TimeInterface, uuid util.UUIDGeneratorInterface) *TaskStore {
	return &TaskStore{
		db:   db,
		time: time,
		uuid: uuid,
	}
}

func (s *TaskStore) CreateTask(task *model.Task) (*model.Task, error) {
	// Set up UUID for task.
	newTask := *task
	id, err := s.uuid.NewRandom()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create an task id")
	}
	newTask.UUID = id.String()

	sql, args, err := sq.
		Insert(table_name).
		SetMap(
			sq.Eq{
				"UUID":              newTask.UUID,
				"Namespace":         newTask.Namespace,
				"PipelineName":      newTask.PipelineName,
				"RunUUID":           newTask.RunId,
				"MLMDExecutionID":   newTask.MLMDExecutionID,
				"CreatedTimestamp":  newTask.CreatedTimestamp,
				"StartedTimestamp":  newTask.StartedTimestamp,
				"FinishedTimestamp": newTask.FinishedTimestamp,
				"Fingerprint":       newTask.Fingerprint,
				"Name":              newTask.Name,
				"ParentTaskUUID":    newTask.ParentTaskId,
				"State":             newTask.State,
				"StateHistory":      newTask.StateHistory,
				"MLMDInputs":        newTask.MLMDInputs,
				"MLMDOutputs":       newTask.MLMDOutputs,
			},
		).
		ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create query to insert task to task table: %v",
			err.Error())
	}
	_, err = s.db.Exec(sql, args...)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to add task to task table: %v",
			err.Error())
	}
	return &newTask, nil
}

func (s *TaskStore) scanRows(rows *sql.Rows) ([]*model.Task, error) {
	var tasks []*model.Task
	for rows.Next() {
		var uuid, namespace, pipelineName, runUUID, mlmdExecutionID, fingerprint string
		var name, parentTaskId, state, stateHistory, inputs, outputs sql.NullString
		var createdTimestamp, startedTimestamp, finishedTimestamp sql.NullInt64
		err := rows.Scan(
			&uuid,
			&namespace,
			&pipelineName,
			&runUUID,
			&mlmdExecutionID,
			&createdTimestamp,
			&startedTimestamp,
			&finishedTimestamp,
			&fingerprint,
			&name,
			&parentTaskId,
			&state,
			&stateHistory,
			&inputs,
			&outputs,
		)
		if err != nil {
			fmt.Printf("scan error is %v", err)
			return tasks, err
		}
		task := &model.Task{
			UUID:              uuid,
			Namespace:         namespace,
			PipelineName:      pipelineName,
			RunId:             runUUID,
			MLMDExecutionID:   mlmdExecutionID,
			CreatedTimestamp:  createdTimestamp.Int64,
			StartedTimestamp:  startedTimestamp.Int64,
			FinishedTimestamp: finishedTimestamp.Int64,
			Fingerprint:       fingerprint,
			Name:              name.String,
			ParentTaskId:      parentTaskId.String,
			StateHistory:      stateHistory.String,
			MLMDInputs:        inputs.String,
			MLMDOutputs:       outputs.String,
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

// Runs two SQL queries in a transaction to return a list of matching experiments, as well as their
// total_size. The total_size does not reflect the page size.
func (s *TaskStore) ListTasks(filterContext *model.FilterContext, opts *list.Options) ([]*model.Task, int, string, error) {
	errorF := func(err error) ([]*model.Task, int, string, error) {
		return nil, 0, "", util.NewInternalServerError(err, "Failed to list tasks: %v", err)
	}

	// SQL for getting the filtered and paginated rows
	sqlBuilder := sq.Select(taskColumns...).From("tasks")
	if filterContext.ReferenceKey != nil && filterContext.ReferenceKey.Type == model.PipelineResourceType {
		sqlBuilder = sqlBuilder.Where(sq.Eq{"PipelineName": filterContext.ReferenceKey.ID})
	}
	if filterContext.ReferenceKey != nil && filterContext.ReferenceKey.Type == model.RunResourceType {
		sqlBuilder = sqlBuilder.Where(sq.Eq{"RunUUID": filterContext.ReferenceKey.ID})
	}
	sqlBuilder = opts.AddFilterToSelect(sqlBuilder)

	rowsSql, rowsArgs, err := opts.AddPaginationToSelect(sqlBuilder).ToSql()
	if err != nil {
		return errorF(err)
	}

	// SQL for getting total size. This matches the query to get all the rows above, in order
	// to do the same filter, but counts instead of scanning the rows.
	sqlBuilder = sq.Select("count(*)").From("tasks")
	if filterContext.ReferenceKey != nil && filterContext.ReferenceKey.Type == model.PipelineResourceType {
		sqlBuilder = sqlBuilder.Where(sq.Eq{"PipelineName": filterContext.ReferenceKey.ID})
	}
	if filterContext.ReferenceKey != nil && filterContext.ReferenceKey.Type == model.RunResourceType {
		sqlBuilder = sqlBuilder.Where(sq.Eq{"RunUUID": filterContext.ReferenceKey.ID})
	}
	sizeSql, sizeArgs, err := opts.AddFilterToSelect(sqlBuilder).ToSql()
	if err != nil {
		return errorF(err)
	}

	// Use a transaction to make sure we're returning the total_size of the same rows queried
	tx, err := s.db.Begin()
	if err != nil {
		glog.Errorf("Failed to start transaction to list tasks")
		return errorF(err)
	}

	rows, err := tx.Query(rowsSql, rowsArgs...)
	if err != nil {
		tx.Rollback()
		return errorF(err)
	}
	if err := rows.Err(); err != nil {
		tx.Rollback()
		return errorF(err)
	}
	exps, err := s.scanRows(rows)
	if err != nil {
		tx.Rollback()
		return errorF(err)
	}
	defer rows.Close()

	sizeRow, err := tx.Query(sizeSql, sizeArgs...)
	if err != nil {
		tx.Rollback()
		return errorF(err)
	}
	if err := sizeRow.Err(); err != nil {
		tx.Rollback()
		return errorF(err)
	}
	total_size, err := list.ScanRowToTotalSize(sizeRow)
	if err != nil {
		tx.Rollback()
		return errorF(err)
	}
	defer sizeRow.Close()

	err = tx.Commit()
	if err != nil {
		glog.Errorf("Failed to commit transaction to list experiments")
		return errorF(err)
	}

	if len(exps) <= opts.PageSize {
		return exps, total_size, "", nil
	}

	npt, err := opts.NextPageToken(exps[opts.PageSize])
	return exps[:opts.PageSize], total_size, npt, err
}

func (s *TaskStore) GetTask(id string) (*model.Task, error) {
	sql, args, err := sq.
		Select(taskColumns...).
		From("tasks").
		Where(sq.Eq{"tasks.uuid": id}).
		Limit(1).ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create query to get task: %v", err.Error())
	}
	r, err := s.db.Query(sql, args...)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to get task: %v", err.Error())
	}
	defer r.Close()
	tasks, err := s.scanRows(r)

	if err != nil || len(tasks) > 1 {
		return nil, util.NewInternalServerError(err, "Failed to get pipeline: %v", err.Error())
	}
	if len(tasks) == 0 {
		return nil, util.NewResourceNotFoundError("task", fmt.Sprint(id))
	}
	return tasks[0], nil
}
