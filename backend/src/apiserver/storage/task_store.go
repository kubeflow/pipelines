package storage

import (
	"database/sql"
	"fmt"
	sq "github.com/Masterminds/squirrel"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

const table_name = "tasks"

var (
	taskColumns = []string{
		"UUID",
		"Namespace",
		"PipelineName",
		"MLMDExecutionID",
		"CreatedTimestamp",
		"StorageState",
		"EndedTimestamp",
		"Fingerprint",
	}
)

type TaskStoreInterface interface {
	// Create a task entry in the database
	CreateTask(task *model.Task) (*model.Task, error)

	// Get the latest started task entry with fingerprint
	GetLatestStartedTaskByFingerprint(fingerprint string) (*model.Task, error)
}

type TaskStore struct {
	db   *DB
	time util.TimeInterface
	uuid util.UUIDGeneratorInterface
}

func (s *TaskStore) CreateTask(task *model.Task) (*model.Task, error) {
	// Set up UUID for task.
	newTask := *task
	id, err := s.uuid.NewRandom()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create an task id.")
	}
	newTask.UUID = id.String()

	sql, args, err := sq.
		Insert(table_name).
		SetMap(sq.Eq{
			"UUID":             newTask.UUID,
			"Namespace":        newTask.Namespace,
			"PipelineName":     newTask.PipelineName,
			"RunUUID":          newTask.RunUUID,
			"MLMDExecutionID":  newTask.MLMDExecutionID,
			"CreatedTimestamp": newTask.CreatedTimestamp,
			"EndedTimestamp":   newTask.EndedTimestamp,
			"Fingerprint":      newTask.Fingerprint,
		}).
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

func (s *TaskStore) GetLatestStartedTaskByFingerprint(fingerprint, pipelineName string) (*model.Task, error) {
	sql, args, err := sq.
		Select(taskColumns...).
		From(table_name).
		Where(sq.Eq{"Fingerprint": fingerprint, "PipelineName":pipelineName}).
		ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to get task: %v", err.Error())
	}
	r, err := s.db.Query(sql, args...)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to get task: %v", err.Error())
	}
	defer r.Close()
	tasks, err := s.scanRows(r)

	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to get experiment: %v", err.Error())
	}
	if len(tasks) == 0 {
		return nil, util.NewResourceNotFoundError("Task", fmt.Sprint(fingerprint))
	}
	return tasks[0], nil
}

func (s *TaskStore) scanRows(rows *sql.Rows) ([]*model.Task, error) {
	var tasks []*model.Task
	for rows.Next() {
		var uuid, namespace, pipelineName, runUUID, mlmdExecutionID, fingerprint string
		var createdTimestamp, endedTimestamp int64
		err := rows.Scan(&uuid, &namespace, &pipelineName, &runUUID, &namespace, &mlmdExecutionID, createdTimestamp, endedTimestamp, fingerprint)
		if err != nil {
			return tasks, err
		}
		task := &model.Task{
			UUID:             uuid,
			Namespace:        namespace,
			PipelineName:     pipelineName,
			RunUUID:          runUUID,
			MLMDExecutionID:  mlmdExecutionID,
			CreatedTimestamp: createdTimestamp,
			EndedTimestamp:   endedTimestamp,
			Fingerprint:      fingerprint,
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}
