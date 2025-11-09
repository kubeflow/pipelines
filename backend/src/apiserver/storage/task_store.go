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
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/golang/glog"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const tableName = "tasks"

var taskColumns = []string{
	"UUID",
	"Namespace",
	"RunUUID",
	"Pods",
	"CreatedAtInSec",
	"StartedInSec",
	"FinishedInSec",
	"Fingerprint",
	"Name",
	"DisplayName",
	"ParentTaskUUID",
	"State",
	"StatusMetadata",
	"StateHistory",
	"InputParameters",
	"OutputParameters",
	"Type",
	"TypeAttrs",
	"ScopePath",
}

// Ensure TaskStore implements TaskStoreInterface
var _ TaskStoreInterface = (*TaskStore)(nil)

type TaskStoreInterface interface {
	// CreateTask Create a task entry in the database.
	CreateTask(task *model.Task) (*model.Task, error)

	// GetTask Fetches a task with a given id.
	GetTask(id string) (*model.Task, error)

	// ListTasks Fetches tasks for given filtering and listing options.
	ListTasks(filterContext *model.FilterContext, opts *list.Options) ([]*model.Task, int, string, error)

	// UpdateTask Updates an existing task entry in the database.
	UpdateTask(new *model.Task) (*model.Task, error)

	// GetChildTasks Fetches all child tasks for a given task UUID.
	GetChildTasks(taskID string) ([]*model.Task, error)
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

// scanTaskRow scans a single row into a model.Task. It expects the column order to match taskColumns.
func scanTaskRow(rowscanner interface{ Scan(dest ...any) error }) (*model.Task, error) {
	var uuid, namespace, runUUID, fingerprint string
	var name, displayName, parentTaskID, pods, statusMetadata, stateHistory, inputParams, outputParams, typeAttrs, scopePath sql.NullString
	var createdAtInSec, startedInSec, finishedInSec sql.NullInt64
	var taskState, taskType int32
	if err := rowscanner.Scan(
		&uuid,
		&namespace,
		&runUUID,
		&pods,
		&createdAtInSec,
		&startedInSec,
		&finishedInSec,
		&fingerprint,
		&name,
		&displayName,
		&parentTaskID,
		&taskState,
		&statusMetadata,
		&stateHistory,
		&inputParams,
		&outputParams,
		&taskType,
		&typeAttrs,
		&scopePath,
	); err != nil {
		return nil, err
	}
	var statusMetadataNew model.JSONData
	if statusMetadata.Valid {
		if err := json.Unmarshal([]byte(statusMetadata.String), &statusMetadataNew); err != nil {
			return nil, err
		}
	}
	var stateHistoryNew model.JSONSlice
	if stateHistory.Valid {
		if err := json.Unmarshal([]byte(stateHistory.String), &stateHistoryNew); err != nil {
			return nil, err
		}
	}
	var podsNew model.JSONSlice
	if pods.Valid {
		if err := json.Unmarshal([]byte(pods.String), &podsNew); err != nil {
			return nil, err
		}
	}
	var inputParameters model.JSONSlice
	if inputParams.Valid {
		if err := json.Unmarshal([]byte(inputParams.String), &inputParameters); err != nil {
			return nil, err
		}
	}
	var outputParameters model.JSONSlice
	if outputParams.Valid {
		if err := json.Unmarshal([]byte(outputParams.String), &outputParameters); err != nil {
			return nil, err
		}
	}
	var typeAttrsData model.JSONData
	if typeAttrs.Valid {
		if err := json.Unmarshal([]byte(typeAttrs.String), &typeAttrsData); err != nil {
			return nil, err
		}
	}
	var scopePathData model.JSONSlice
	if scopePath.Valid {
		if err := json.Unmarshal([]byte(scopePath.String), &scopePathData); err != nil {
			return nil, err
		}
	}
	var parentTaskIDNew *string
	if parentTaskID.Valid {
		parentTaskIDNew = &parentTaskID.String
	}
	return &model.Task{
		UUID:             uuid,
		Namespace:        namespace,
		RunUUID:          runUUID,
		Pods:             podsNew,
		CreatedAtInSec:   createdAtInSec.Int64,
		StartedInSec:     startedInSec.Int64,
		FinishedInSec:    finishedInSec.Int64,
		Fingerprint:      fingerprint,
		Name:             name.String,
		DisplayName:      displayName.String,
		ParentTaskUUID:   parentTaskIDNew,
		State:            model.TaskStatus(taskState),
		StatusMetadata:   statusMetadataNew,
		StateHistory:     stateHistoryNew,
		InputParameters:  inputParameters,
		OutputParameters: outputParameters,
		Type:             model.TaskType(taskType),
		TypeAttrs:        typeAttrsData,
		ScopePath:        scopePathData,
	}, nil
}

// hydrateArtifactsForTasks fills InputArtifactsHydrated and OutputArtifactsHydrated for provided tasks by
// querying artifact_tasks joined with artifacts. It uses TaskID IN (...) to limit scope.
func hydrateArtifactsForTasks(db *DB, tasks []*model.Task) error {
	if len(tasks) == 0 {
		return nil
	}
	// Build map and list of task IDs
	taskByID := make(map[string]*model.Task, len(tasks))
	taskIDs := make([]string, 0, len(tasks))
	for _, t := range tasks {
		if t == nil || t.UUID == "" {
			continue
		}
		if _, ok := taskByID[t.UUID]; !ok {
			taskByID[t.UUID] = t
			taskIDs = append(taskIDs, t.UUID)
		}
	}
	if len(taskIDs) == 0 {
		return nil
	}

	// Query artifact links for these tasks
	sqlStr, args, err := sq.
		Select(
			"artifact_tasks.TaskID",
			"artifact_tasks.Type",
			"artifact_tasks.Producer",
			"artifact_tasks.ArtifactKey",
			"artifacts.UUID",
			"artifacts.Namespace",
			"artifacts.Type",
			"artifacts.URI",
			"artifacts.Name",
			"artifacts.CreatedAtInSec",
			"artifacts.LastUpdateInSec",
			"artifacts.Metadata",
			"artifacts.NumberValue",
		).
		From("artifact_tasks").
		Join("artifacts ON artifact_tasks.ArtifactID = artifacts.UUID").
		Where(sq.Eq{"artifact_tasks.TaskID": taskIDs}).
		ToSql()
	if err != nil {
		return err
	}

	rows, err := db.Query(sqlStr, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var taskID string
		var linkType sql.NullInt32
		var producer sql.NullString
		var key string
		var artUUID, artNamespace, artName string
		var artType sql.NullInt32
		var createdAt, updatedAt sql.NullInt64
		var metadata, artURI sql.NullString
		var numberValue sql.NullFloat64

		if err := rows.Scan(&taskID, &linkType, &producer, &key,
			&artUUID, &artNamespace, &artType, &artURI, &artName, &createdAt, &updatedAt, &metadata, &numberValue); err != nil {
			return err
		}

		task := taskByID[taskID]
		if task == nil {
			continue
		}

		var metaMap model.JSONData
		if metadata.Valid {
			if err := json.Unmarshal([]byte(metadata.String), &metaMap); err != nil {
				return err
			}
		}
		mArtifact := &model.Artifact{
			UUID:            artUUID,
			Namespace:       artNamespace,
			Type:            model.ArtifactType(artType.Int32),
			Name:            artName,
			CreatedAtInSec:  createdAt.Int64,
			LastUpdateInSec: updatedAt.Int64,
			Metadata:        metaMap,
		}
		if artURI.Valid {
			mArtifact.URI = &artURI.String
		}
		if numberValue.Valid {
			mArtifact.NumberValue = &numberValue.Float64
		}

		// Parse producer JSON to IOProducer
		var producerProto *model.IOProducer
		if producer.Valid && producer.String != "" {
			var producerData model.JSONData
			if err := json.Unmarshal([]byte(producer.String), &producerData); err == nil {
				producerProto = &model.IOProducer{}
				if taskName, ok := producerData["taskName"].(string); ok {
					producerProto.TaskName = taskName
				}
				if iteration, ok := producerData["iteration"].(float64); ok {
					iterInt := int64(iteration)
					producerProto.Iteration = &iterInt
				}
			}
		}

		h := model.TaskArtifactHydrated{
			Value:    mArtifact,
			Producer: producerProto,
			Key:      key,
			Type:     apiv2beta1.IOType(linkType.Int32),
		}

		isOutput, err := iOTypeIsOutput(apiv2beta1.IOType(linkType.Int32))
		if err != nil {
			return err
		}
		if isOutput {
			task.OutputArtifactsHydrated = append(task.OutputArtifactsHydrated, h)
		} else {
			task.InputArtifactsHydrated = append(task.InputArtifactsHydrated, h)
		}
	}
	return rows.Err()
}

func iOTypeIsOutput(ioType apiv2beta1.IOType) (bool, error) {
	switch ioType {
	case apiv2beta1.IOType_OUTPUT,
		apiv2beta1.IOType_ITERATOR_OUTPUT,
		apiv2beta1.IOType_ONE_OF_OUTPUT,
		apiv2beta1.IOType_TASK_FINAL_STATUS_OUTPUT:
		return true, nil
	case apiv2beta1.IOType_COMPONENT_INPUT,
		apiv2beta1.IOType_COLLECTED_INPUTS,
		apiv2beta1.IOType_TASK_OUTPUT_INPUT,
		apiv2beta1.IOType_RUNTIME_VALUE_INPUT,
		apiv2beta1.IOType_ITERATOR_INPUT,
		apiv2beta1.IOType_ITERATOR_INPUT_RAW,
		apiv2beta1.IOType_COMPONENT_DEFAULT_INPUT:
		return false, nil
	default:
		return false, fmt.Errorf("unknown IOType %v", ioType)
	}
}

func (s *TaskStore) scanRows(rows *sql.Rows) ([]*model.Task, error) {
	var tasks []*model.Task
	for rows.Next() {
		t, err := scanTaskRow(rows)
		if err != nil {
			fmt.Printf("scan error is %v", err)
			return tasks, err
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func (s *TaskStore) CreateTask(task *model.Task) (*model.Task, error) {
	// Set up UUID for task.
	newTask := *task
	id, err := s.uuid.NewRandom()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create an task id")
	}
	newTask.UUID = id.String()

	if newTask.CreatedAtInSec == 0 {
		if newTask.StartedInSec == 0 {
			now := s.time.Now().Unix()
			newTask.StartedInSec = now
			newTask.CreatedAtInSec = now
		} else {
			newTask.CreatedAtInSec = newTask.StartedInSec
		}
	}

	// Auto-populate state history if state is set (mirrors Run behavior)
	// Only append if state_history is empty OR if last state differs from current state
	if newTask.State != 0 {
		if len(newTask.StateHistory) == 0 || getLastTaskState(newTask.StateHistory) != newTask.State {
			taskStatus := &apiv2beta1.PipelineTaskDetail_TaskStatus{
				UpdateTime: &timestamppb.Timestamp{Seconds: s.time.Now().Unix()},
				State:      apiv2beta1.PipelineTaskDetail_TaskState(newTask.State),
			}
			newEntry, err := model.ProtoSliceToJSONSlice([]*apiv2beta1.PipelineTaskDetail_TaskStatus{taskStatus})
			if err != nil {
				return nil, util.NewInternalServerError(err, "Failed to create state history entry")
			}
			if len(newEntry) > 0 {
				newTask.StateHistory = append(newTask.StateHistory, newEntry[0])
			}
		}
	}

	stateHistoryString := ""
	if history, err := json.Marshal(newTask.StateHistory); err == nil {
		stateHistoryString = string(history)
	} else {
		return nil, util.NewInternalServerError(err, "Failed to marshal state history in a new run")
	}

	podsString := ""
	if podNames, err := json.Marshal(newTask.Pods); err == nil {
		podsString = string(podNames)
	} else {
		return nil, util.NewInternalServerError(err, "Failed to marshal pod names in a new task")
	}

	inputParamsString := ""
	if inputParams, err := json.Marshal(newTask.InputParameters); err == nil {
		inputParamsString = string(inputParams)
	} else {
		return nil, util.NewInternalServerError(err, "Failed to marshal input parameters in a new task")
	}

	outputParamsString := ""
	if outputParams, err := json.Marshal(newTask.OutputParameters); err == nil {
		outputParamsString = string(outputParams)
	} else {
		return nil, util.NewInternalServerError(err, "Failed to marshal output parameters in a new task")
	}

	typeAttrsString := ""
	if typeAttrs, err := json.Marshal(newTask.TypeAttrs); err == nil {
		typeAttrsString = string(typeAttrs)
	} else {
		return nil, util.NewInternalServerError(err, "Failed to marshal type attributes in a new task")
	}

	scopePathStr := ""
	if b, err := json.Marshal(newTask.ScopePath); err == nil {
		scopePathStr = string(b)
	} else {
		return nil, util.NewInternalServerError(err, "Failed to marshal scope path in a new task")
	}

	sql, args, err := sq.
		Insert(tableName).
		SetMap(
			sq.Eq{
				"UUID":             newTask.UUID,
				"Namespace":        newTask.Namespace,
				"RunUUID":          newTask.RunUUID,
				"Pods":             podsString,
				"CreatedAtInSec":   newTask.CreatedAtInSec,
				"StartedInSec":     newTask.StartedInSec,
				"FinishedInSec":    newTask.FinishedInSec,
				"Fingerprint":      newTask.Fingerprint,
				"Name":             newTask.Name,
				"ParentTaskUUID":   newTask.ParentTaskUUID,
				"ScopePath":        scopePathStr,
				"State":            newTask.State,
				"StateHistory":     stateHistoryString,
				"InputParameters":  inputParamsString,
				"OutputParameters": outputParamsString,
				"Type":             newTask.Type,
				"TypeAttrs":        typeAttrsString,
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

// ListTasks Runs two SQL queries in a transaction to return a list of matching experiments, as well as their
// total_size. The total_size does not reflect the page size.
func (s *TaskStore) ListTasks(filterContext *model.FilterContext, opts *list.Options) ([]*model.Task, int, string, error) {
	errorF := func(err error) ([]*model.Task, int, string, error) {
		return nil, 0, "", util.NewInternalServerError(err, "Failed to list tasks: %v", err)
	}

	// SQL for getting the filtered and paginated rows
	sqlBuilder := sq.Select(taskColumns...).From("tasks")
	if filterContext.ReferenceKey != nil && filterContext.ReferenceKey.Type == model.RunResourceType {
		sqlBuilder = sqlBuilder.Where(sq.Eq{"RunUUID": filterContext.ReferenceKey.ID})
	}
	if filterContext.ReferenceKey != nil && filterContext.Type == model.TaskResourceType {
		sqlBuilder = sqlBuilder.Where(sq.Eq{"ParentTaskUUID": filterContext.ID})
	}
	if filterContext.ReferenceKey != nil && filterContext.Type == model.NamespaceResourceType {
		// Only add namespace filter if namespace is not empty
		// Empty namespace in single-user mode means list all tasks
		if filterContext.ID != "" {
			sqlBuilder = sqlBuilder.Where(sq.Eq{"Namespace": filterContext.ID})
		}
	}
	sqlBuilder = opts.AddFilterToSelect(sqlBuilder)

	rowsSql, rowsArgs, err := opts.AddPaginationToSelect(sqlBuilder).ToSql()
	if err != nil {
		return errorF(err)
	}

	// SQL for getting total size. This matches the query to get all the rows above, in order
	// to do the same filter, but counts instead of scanning the rows.
	sqlBuilder = sq.Select("count(*)").From("tasks")
	if filterContext.ReferenceKey != nil && filterContext.ReferenceKey.Type == model.RunResourceType {
		sqlBuilder = sqlBuilder.Where(sq.Eq{"RunUUID": filterContext.ReferenceKey.ID})
	}
	if filterContext.ReferenceKey != nil && filterContext.Type == model.TaskResourceType {
		sqlBuilder = sqlBuilder.Where(sq.Eq{"ParentTaskUUID": filterContext.ID})
	}
	if filterContext.ReferenceKey != nil && filterContext.Type == model.NamespaceResourceType {
		// Only add namespace filter if namespace is not empty
		// Empty namespace in single-user mode means list all tasks
		if filterContext.ID != "" {
			sqlBuilder = sqlBuilder.Where(sq.Eq{"Namespace": filterContext.ID})
		}
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
		if err := hydrateArtifactsForTasks(s.db, exps); err != nil {
			return errorF(err)
		}
		return exps, total_size, "", nil
	}

	npt, err := opts.NextPageToken(exps[opts.PageSize])
	page := exps[:opts.PageSize]
	if err := hydrateArtifactsForTasks(s.db, page); err != nil {
		return errorF(err)
	}
	return page, total_size, npt, err
}

func (s *TaskStore) GetTask(id string) (*model.Task, error) {
	toSQL, args, err := sq.
		Select(taskColumns...).
		From("tasks").
		Where(sq.Eq{"tasks.uuid": id}).
		Limit(1).ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create query to get task: %v", err.Error())
	}
	r, err := s.db.Query(toSQL, args...)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to get task: %v", err.Error())
	}
	defer r.Close()
	tasks, err := s.scanRows(r)

	if err != nil || len(tasks) > 1 {
		return nil, util.NewInternalServerError(err, "Failed to get pipeline: %v", err)
	}
	if len(tasks) == 0 {
		return nil, util.NewResourceNotFoundError("task", fmt.Sprint(id))
	}
	// Hydrate artifacts for this task
	if err := hydrateArtifactsForTasks(s.db, []*model.Task{tasks[0]}); err != nil {
		return nil, util.NewInternalServerError(err, "Failed to hydrate task artifacts")
	}
	return tasks[0], nil
}

// getTaskForUpdate retrieves a task with a row-level lock (SELECT ... FOR UPDATE).
// This must be called within a transaction.
// The lock ensures that no other transaction can modify this row until the current transaction completes.
// For MySQL/PostgreSQL, this adds FOR UPDATE. For SQLite (tests), it's a no-op since SQLite doesn't support row locks.
func (s *TaskStore) getTaskForUpdate(tx *sql.Tx, id string) (*model.Task, error) {
	// Build SELECT query
	sqlStr, args, err := sq.
		Select(taskColumns...).
		From("tasks").
		Where(sq.Eq{"tasks.uuid": id}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create query to get task for update: %v", err.Error())
	}

	// Add FOR UPDATE clause using the dialect (MySQL adds it, SQLite doesn't)
	sqlStr = s.db.SelectForUpdate(sqlStr)

	// Execute query within the transaction
	row := tx.QueryRow(sqlStr, args...)

	task, err := scanTaskRow(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, util.NewResourceNotFoundError("task", fmt.Sprint(id))
		}
		return nil, util.NewInternalServerError(err, "Failed to get task for update: %v", err)
	}

	return task, nil
}

// UpdateTask updates an existing task in the tasks table and returns the updated task.
// Uses row-level locking to prevent race conditions when multiple concurrent updates
// try to modify the same task (e.g., loop iterations propagating parameters to parent task).
func (s *TaskStore) UpdateTask(new *model.Task) (*model.Task, error) {
	if new == nil {
		return nil, util.NewInvalidInputError("Failed to update task: task cannot be nil")
	}
	if new.UUID == "" {
		return nil, util.NewInvalidInputError("Failed to update task: task ID cannot be empty")
	}

	// Start a transaction to ensure atomic read-merge-write with row locking
	tx, err := s.db.Begin()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to start transaction for task update")
	}
	defer tx.Rollback() // Will be no-op if Commit() succeeds

	// Get the current task state with a row-level lock (SELECT ... FOR UPDATE)
	// This prevents other concurrent updates from reading the same old state
	lockedOld, err := s.getTaskForUpdate(tx, new.UUID)
	if err != nil {
		return nil, err
	}

	// Use the locked version for merging instead of the 'old' parameter
	// This ensures we merge against the most recent state

	// Build SET map dynamically so we only update provided fields.
	setMap := sq.Eq{}

	// Simple scalar/string fields: update if non-empty OR explicitly zero is meaningful.
	// For strings: only update when not empty to avoid erasing existing values unintentionally.
	if new.Namespace != "" {
		setMap["Namespace"] = new.Namespace
	}
	if new.RunUUID != "" {
		setMap["RunUUID"] = new.RunUUID
	}
	if new.Fingerprint != "" {
		setMap["Fingerprint"] = new.Fingerprint
	}
	if new.Name != "" {
		setMap["Name"] = new.Name
	}
	if new.DisplayName != "" {
		setMap["DisplayName"] = new.DisplayName
	}
	if new.ParentTaskUUID != nil {
		if *new.ParentTaskUUID == "" {
			setMap["ParentTaskUUID"] = nil
		} else {
			setMap["ParentTaskUUID"] = *new.ParentTaskUUID
		}
	}

	// State and Type default to 0 which are valid enums; update only when non-zero to avoid accidental resets.
	if new.State != 0 {
		setMap["State"] = new.State

		// Auto-populate state history when state changes (mirrors Run behavior)
		// Use lockedOld.StateHistory as the base to prevent race conditions
		mergedHistory := lockedOld.StateHistory

		// Check if we need to append new state to history
		if len(mergedHistory) == 0 || getLastTaskState(mergedHistory) != new.State {
			taskStatus := &apiv2beta1.PipelineTaskDetail_TaskStatus{
				UpdateTime: &timestamppb.Timestamp{Seconds: s.time.Now().Unix()},
				State:      apiv2beta1.PipelineTaskDetail_TaskState(new.State),
			}
			newEntry, err := model.ProtoSliceToJSONSlice([]*apiv2beta1.PipelineTaskDetail_TaskStatus{taskStatus})
			if err != nil {
				return nil, util.NewInternalServerError(err, "Failed to create state history entry")
			}
			if len(newEntry) > 0 {
				mergedHistory = append(mergedHistory, newEntry[0])
			}
		}

		// Marshal merged history
		if b, err := json.Marshal(mergedHistory); err == nil {
			setMap["StateHistory"] = string(b)
		} else {
			return nil, util.NewInternalServerError(err, "Failed to marshal state history in an updated task")
		}
	}

	if new.Type != 0 {
		setMap["Type"] = new.Type
	}
	// Timestamps: allow update when non-zero.
	if new.StartedInSec != 0 {
		setMap["StartedInSec"] = new.StartedInSec
	}
	if new.FinishedInSec != 0 {
		setMap["FinishedInSec"] = new.FinishedInSec
	}

	// JSON/slice/map fields: update only if not nil (presence indicates intent).
	// Note: StateHistory is now auto-populated above when State changes
	if new.StatusMetadata != nil {
		if b, err := json.Marshal(new.StatusMetadata); err == nil {
			setMap["StatusMetadata"] = string(b)
		} else {
			return nil, util.NewInternalServerError(err, "Failed to marshal status metadata in an updated task")
		}
	}
	if new.Pods != nil {
		if b, err := json.Marshal(new.Pods); err == nil {
			setMap["Pods"] = string(b)
		} else {
			return nil, util.NewInternalServerError(err, "Failed to marshal pod names in an updated task")
		}
	}

	// Merge input parameters using the locked old state
	// This prevents race conditions where concurrent updates might overwrite each other's parameters
	if new.InputParameters != nil {
		// Use lockedOld (from SELECT FOR UPDATE) instead of 'old' parameter
		oldInputParams := lockedOld.InputParameters
		merged, err := mergeParameters(oldInputParams, new.InputParameters)
		if err != nil {
			return nil, util.NewInternalServerError(err, "Failed to merge input parameters in an updated task")
		}
		if b, err := json.Marshal(merged); err == nil {
			setMap["InputParameters"] = string(b)
		} else {
			return nil, util.NewInternalServerError(err, "Failed to marshal input parameters in an updated task")
		}
	}

	// Merge output parameters using the locked old state
	// This prevents race conditions where concurrent updates might overwrite each other's parameters
	if new.OutputParameters != nil {
		// Use lockedOld (from SELECT FOR UPDATE) instead of 'old' parameter
		oldOutputParams := lockedOld.OutputParameters

		merged, err := mergeParameters(oldOutputParams, new.OutputParameters)
		if err != nil {
			return nil, util.NewInternalServerError(err, "Failed to merge output parameters in an updated task")
		}
		if b, err := json.Marshal(merged); err == nil {
			setMap["OutputParameters"] = string(b)
		} else {
			return nil, util.NewInternalServerError(err, "Failed to marshal output parameters in an updated task")
		}
	}

	if new.TypeAttrs != nil {
		if b, err := json.Marshal(new.TypeAttrs); err == nil {
			setMap["TypeAttrs"] = string(b)
		} else {
			return nil, util.NewInternalServerError(err, "Failed to marshal type attributes in an updated task")
		}
	}

	if len(setMap) == 0 {
		// Nothing to update; commit transaction and return current record
		if err := tx.Commit(); err != nil {
			return nil, util.NewInternalServerError(err, "Failed to commit transaction (no changes)")
		}
		return s.GetTask(new.UUID)
	}

	// Build UPDATE query
	sqlStr, args, err := sq.
		Update(tableName).
		SetMap(setMap).
		Where(sq.Eq{"UUID": new.UUID}).
		ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create query to update task: %v", err.Error())
	}

	// Execute UPDATE within the transaction
	// The row is already locked by our SELECT FOR UPDATE, so this is safe
	res, err := tx.Exec(sqlStr, args...)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to update task: %v", err.Error())
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return nil, util.NewResourceNotFoundError("task", new.UUID)
	}

	// Commit the transaction to release the row lock and make changes visible
	if err := tx.Commit(); err != nil {
		return nil, util.NewInternalServerError(err, "Failed to commit transaction for task update")
	}

	glog.Infof("Successfully updated task %s with row-level locking", new.UUID)
	return s.GetTask(new.UUID)
}

// mergeParameters merges the new parameters with the old parameters.
func mergeParameters(old, new model.JSONSlice) (model.JSONSlice, error) {
	typeFunc := func() *apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter {
		return &apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{}
	}
	oldParams, err := model.JSONSliceToProtoSlice(old, typeFunc)
	if err != nil {
		return nil, err
	}
	newParams, err := model.JSONSliceToProtoSlice(new, typeFunc)
	if err != nil {
		return nil, err
	}
	makeKey := func(p *apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter) string {
		key := fmt.Sprintf("%v-%s", p.Type, p.ParameterKey)
		if p.Producer != nil {
			key = fmt.Sprintf("%s-%s", key, p.Producer.TaskName)
			if p.Producer.Iteration != nil {
				key = fmt.Sprintf("%s-%d", key, *p.Producer.Iteration)
			}
		}
		// Include the value hash, in cases like the iterator case where
		// iterations propagate values to upstream tasks, the iteration
		// index is not propagated (like in a for-loop-task), so we need
		// to include the value hash to avoid collisions.
		valueHash, err := hashProtoValue(p.GetValue())
		if err != nil {
			glog.Errorf("Failed to hash parameter value: %v", err)
		}
		key = fmt.Sprintf("%s-%s", key, valueHash)
		return key
	}
	mergedParams := map[string]*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{}
	for _, p := range oldParams {
		key := makeKey(p)
		mergedParams[key] = p
	}
	for _, p := range newParams {
		key := makeKey(p)
		mergedParams[key] = p
	}
	paramsSlice := make([]*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter, 0, len(mergedParams))
	for _, p := range mergedParams {
		paramsSlice = append(paramsSlice, p)
	}
	parameters, err := model.ProtoSliceToJSONSlice(paramsSlice)
	if err != nil {
		return nil, err
	}
	return parameters, nil
}

func (s *TaskStore) GetChildTasks(taskID string) ([]*model.Task, error) {
	toSQL, args, err := sq.
		Select(taskColumns...).
		From("tasks").
		Where(sq.Eq{"ParentTaskUUID": taskID}).
		ToSql()

	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create query to get child tasks: %v", err.Error())
	}

	rows, err := s.db.Query(toSQL, args...)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to get child tasks: %v", err.Error())
	}
	defer rows.Close()

	return s.scanRows(rows)
}

// GetTaskCountForRun returns the total count of tasks for a given run ID.
// This is a lightweight operation that doesn't perform task hydration.
func (s *TaskStore) GetTaskCountForRun(runID string) (int, error) {
	sizeSQL, sizeArgs, err := sq.
		Select("count(*)").
		From("tasks").
		Where(sq.Eq{"RunUUID": runID}).
		ToSql()

	if err != nil {
		return 0, util.NewInternalServerError(err, "Failed to create task count query: %v", err.Error())
	}

	sizeRow, err := s.db.Query(sizeSQL, sizeArgs...)
	if err != nil {
		return 0, util.NewInternalServerError(err, "Failed to get task count: %v", err.Error())
	}
	defer sizeRow.Close()

	var total int
	sizeRow.Next()
	if err := sizeRow.Scan(&total); err != nil {
		return 0, util.NewInternalServerError(err, "Failed to scan task count: %v", err.Error())
	}

	return total, nil
}

func hashProtoValue(v *structpb.Value) (string, error) {
	// Deterministic binary marshal
	b, err := proto.MarshalOptions{Deterministic: true}.Marshal(v)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

// getLastTaskState retrieves the state from the last entry in task state history.
// Returns 0 (unspecified) if history is empty or cannot be parsed.
func getLastTaskState(history model.JSONSlice) model.TaskStatus {
	if len(history) == 0 {
		return 0
	}

	// Convert JSONSlice to TaskStatus protobuf slice
	typeFunc := func() *apiv2beta1.PipelineTaskDetail_TaskStatus {
		return &apiv2beta1.PipelineTaskDetail_TaskStatus{}
	}

	histProtos, err := model.JSONSliceToProtoSlice(history, typeFunc)
	if err != nil || len(histProtos) == 0 {
		glog.Warningf("Failed to parse state history: %v", err)
		return 0
	}

	lastEntry := histProtos[len(histProtos)-1]
	return model.TaskStatus(lastEntry.GetState())
}

// DeleteTasksForRun deletes all tasks associated with a specific run.
// This should be called before deleting a run to avoid foreign key constraint violations.
func (s *TaskStore) DeleteTasksForRun(tx *sql.Tx, runUUID string) error {
	deleteSQL, deleteArgs, err := sq.Delete(tableName).Where(sq.Eq{"RunUUID": runUUID}).ToSql()
	if err != nil {
		return util.NewInternalServerError(err, "Failed to create query to delete tasks for run: %s", runUUID)
	}

	var result sql.Result
	if tx != nil {
		result, err = tx.Exec(deleteSQL, deleteArgs...)
	} else {
		result, err = s.db.Exec(deleteSQL, deleteArgs...)
	}

	if err != nil {
		return util.NewInternalServerError(err, "Failed to delete tasks for run %s from table", runUUID)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		glog.Warningf("Failed to get rows affected when deleting tasks for run %s: %v", runUUID, err)
	} else {
		glog.V(4).Infof("Deleted %d tasks for run %s", rowsAffected, runUUID)
	}

	return nil
}
