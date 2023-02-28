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
	"encoding/json"
	"fmt"
	"strings"

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
	"PodName",
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
	"ChildrenPods",
}

var (
	taskColumnsWithPayload = append(taskColumns, "Payload")
	taskColumnsUpdates     = prepareUpdateSuffix(taskColumnsWithPayload)
)

func prepareUpdateSuffix(columns []string) string {
	columnsExtended := make([]string, 0)
	for _, c := range taskColumnsWithPayload {
		columnsExtended = append(columnsExtended, fmt.Sprintf("%[1]v=VALUES(%[1]v)", c))
	}
	return strings.Join(columnsExtended, ",")
}

type TaskStoreInterface interface {
	// Create a task entry in the database.
	CreateTask(task *model.Task) (*model.Task, error)

	// Fetches a task with a given id.
	GetTask(id string) (*model.Task, error)

	// Fetches tasks for given filtering and listing options.
	ListTasks(filterContext *model.FilterContext, opts *list.Options) ([]*model.Task, int, string, error)

	// Creates new tasks or updates the existing ones.
	CreateOrUpdateTasks(tasks []*model.Task) ([]*model.Task, error)
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

	if newTask.CreatedTimestamp == 0 {
		if newTask.StartedTimestamp == 0 {
			now := s.time.Now().Unix()
			newTask.StartedTimestamp = now
			newTask.CreatedTimestamp = now
		} else {
			newTask.CreatedTimestamp = newTask.StartedTimestamp
		}
	}

	stateHistoryString := ""
	if history, err := json.Marshal(newTask.StateHistory); err == nil {
		stateHistoryString = string(history)
	} else {
		return nil, util.NewInternalServerError(err, "Failed to marshal state history in a new run")
	}

	childrenPodsString := ""
	if children, err := json.Marshal(newTask.ChildrenPods); err == nil {
		childrenPodsString = string(children)
	} else {
		return nil, util.NewInternalServerError(err, "Failed to marshal children pods in a new run")
	}

	sql, args, err := sq.
		Insert(table_name).
		SetMap(
			sq.Eq{
				"UUID":              newTask.UUID,
				"Namespace":         newTask.Namespace,
				"PipelineName":      newTask.PipelineName,
				"RunUUID":           newTask.RunId,
				"PodName":           newTask.PodName,
				"MLMDExecutionID":   newTask.MLMDExecutionID,
				"CreatedTimestamp":  newTask.CreatedTimestamp,
				"StartedTimestamp":  newTask.StartedTimestamp,
				"FinishedTimestamp": newTask.FinishedTimestamp,
				"Fingerprint":       newTask.Fingerprint,
				"Name":              newTask.Name,
				"ParentTaskUUID":    newTask.ParentTaskId,
				"State":             newTask.State.ToString(),
				"StateHistory":      stateHistoryString,
				"MLMDInputs":        newTask.MLMDInputs,
				"MLMDOutputs":       newTask.MLMDOutputs,
				"ChildrenPods":      childrenPodsString,
				"Payload":           newTask.ToString(),
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
		var uuid, namespace, pipelineName, runUUID, podName, mlmdExecutionID, fingerprint string
		var name, parentTaskId, state, stateHistory, inputs, outputs, children sql.NullString
		var createdTimestamp, startedTimestamp, finishedTimestamp sql.NullInt64
		err := rows.Scan(
			&uuid,
			&namespace,
			&pipelineName,
			&runUUID,
			&podName,
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
			&children,
		)
		if err != nil {
			fmt.Printf("scan error is %v", err)
			return tasks, err
		}
		var stateHistoryNew []*model.RuntimeStatus
		if stateHistory.Valid {
			json.Unmarshal([]byte(stateHistory.String), &stateHistoryNew)
		}
		var childrenPods []string
		if children.Valid {
			json.Unmarshal([]byte(children.String), &childrenPods)
		}
		task := &model.Task{
			UUID:              uuid,
			Namespace:         namespace,
			PipelineName:      pipelineName,
			RunId:             runUUID,
			PodName:           podName,
			MLMDExecutionID:   mlmdExecutionID,
			CreatedTimestamp:  createdTimestamp.Int64,
			StartedTimestamp:  startedTimestamp.Int64,
			FinishedTimestamp: finishedTimestamp.Int64,
			Fingerprint:       fingerprint,
			Name:              name.String,
			ParentTaskId:      parentTaskId.String,
			StateHistory:      stateHistoryNew,
			MLMDInputs:        inputs.String,
			MLMDOutputs:       outputs.String,
			ChildrenPods:      childrenPods,
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

// Updates missing fields with existing data entries.
func (s *TaskStore) patchWithExistingTasks(tasks []*model.Task) error {
	var podNames []string
	for _, task := range tasks {
		podNames = append(podNames, task.PodName)
	}
	sql, args, err := sq.
		Select(taskColumns...).
		From("tasks").
		Where(sq.Eq{"PodName": podNames}).
		ToSql()
	if err != nil {
		return util.NewInternalServerError(err, "Failed to create query to check existing tasks")
	}
	r, err := s.db.Query(sql, args...)
	if err != nil {
		return util.NewInternalServerError(err, "Failed to check existing tasks")
	}
	defer r.Close()
	existingTasks, err := s.scanRows(r)
	if err != nil {
		return util.NewInternalServerError(err, "Failed to parse existing tasks")
	}
	mapTasks := make(map[string]*model.Task, 0)
	for _, task := range existingTasks {
		mapTasks[task.PodName] = task
	}
	for _, task := range tasks {
		if existingTask, ok := mapTasks[task.PodName]; ok {
			patchTask(task, existingTask)
		}
	}
	return nil
}

// Creates new entries or updates existing ones.
func (s *TaskStore) CreateOrUpdateTasks(tasks []*model.Task) ([]*model.Task, error) {
	buildQuery := func(ts []*model.Task) (string, []interface{}, error) {
		sqlInsert := sq.Insert("tasks").Columns(taskColumnsWithPayload...)
		for _, t := range ts {
			childrenPodsString := ""
			if len(t.ChildrenPods) > 0 {
				children, err := json.Marshal(t.ChildrenPods)
				if err != nil {
					return "", nil, util.NewInternalServerError(err, "Failed to marshal child task ids in a task")
				}
				childrenPodsString = string(children)
			}
			stateHistoryString := ""
			if len(t.StateHistory) > 0 {
				history, err := json.Marshal(t.StateHistory)
				if err != nil {
					return "", nil, util.NewInternalServerError(err, "Failed to marshal state history in a task")
				}
				stateHistoryString = string(history)
			}
			sqlInsert = sqlInsert.Values(
				t.UUID,
				t.Namespace,
				t.PipelineName,
				t.RunId,
				t.PodName,
				t.MLMDExecutionID,
				t.CreatedTimestamp,
				t.StartedTimestamp,
				t.FinishedTimestamp,
				t.Fingerprint,
				t.Name,
				t.ParentTaskId,
				t.State.ToString(),
				stateHistoryString,
				t.MLMDInputs,
				t.MLMDOutputs,
				childrenPodsString,
				t.ToString(),
			)
		}
		sqlInsert = sqlInsert.Suffix(fmt.Sprintf("ON DUPLICATE KEY UPDATE %v", taskColumnsUpdates))
		sql, args, err := sqlInsert.ToSql()
		if err != nil {
			return "", nil, util.NewInternalServerError(err, "Failed to create query to check existing tasks")
		}
		return sql, args, nil
	}

	// Check for existing tasks and fill empty field with existing data.
	// Assumes that PodName column is a unique key.
	if err := s.patchWithExistingTasks(tasks); err != nil {
		return nil, util.NewInternalServerError(err, "Failed to check for existing tasks")
	}
	for _, task := range tasks {
		if task.UUID == "" {
			id, err := s.uuid.NewRandom()
			if err != nil {
				return nil, util.NewInternalServerError(err, "Failed to create an task id")
			}
			task.UUID = id.String()
		}
		if task.CreatedTimestamp == 0 {
			now := s.time.Now().Unix()
			if task.StartedTimestamp < now {
				task.CreatedTimestamp = task.StartedTimestamp
			} else {
				task.CreatedTimestamp = now
			}
		}
	}

	// Execute the query
	sql, arg, err := buildQuery(tasks)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to build query to update or insert tasks")
	}
	_, err = s.db.Exec(sql, arg...)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to update or insert tasks. Query: %v. Args: %v", sql, arg)
	}
	return tasks, nil
}

// Fills empty fields in a new task with the data from an existing task.
func patchTask(original *model.Task, patch *model.Task) {
	if original.UUID == "" {
		original.UUID = patch.UUID
	}
	if original.Namespace == "" {
		original.Namespace = patch.Namespace
	}
	if original.RunId == "" {
		original.RunId = patch.RunId
	}
	if original.PodName == "" {
		original.PodName = patch.PodName
	}
	if original.MLMDExecutionID == "" {
		original.MLMDExecutionID = patch.MLMDExecutionID
	}
	if original.CreatedTimestamp == 0 {
		original.CreatedTimestamp = patch.CreatedTimestamp
	}
	if original.StartedTimestamp == 0 {
		original.StartedTimestamp = patch.StartedTimestamp
	}
	if original.FinishedTimestamp == 0 {
		original.FinishedTimestamp = patch.FinishedTimestamp
	}
	if original.Fingerprint == "" {
		original.Fingerprint = patch.Fingerprint
	}
	if original.Name == "" {
		original.Name = patch.Name
	}
	if original.ParentTaskId == "" {
		original.ParentTaskId = patch.ParentTaskId
	}
	if original.State.ToV2() == model.RuntimeStateUnspecified {
		original.State = patch.State.ToV2()
	}
	if original.MLMDInputs == "" {
		original.MLMDInputs = patch.MLMDInputs
	}
	if original.MLMDOutputs == "" {
		original.MLMDOutputs = patch.MLMDOutputs
	}
	if original.StateHistory == nil {
		original.StateHistory = patch.StateHistory
	}
	if len(original.ChildrenPods) == 0 {
		original.ChildrenPods = patch.ChildrenPods
	}
}
