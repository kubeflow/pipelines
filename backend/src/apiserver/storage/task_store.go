package storage

import (
	"database/sql"
	sq "github.com/Masterminds/squirrel"
	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
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

	ListTasks(filterContext *common.FilterContext, opts *list.Options) ([]*model.Task, int, string, error)
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

// Runs two SQL queries in a transaction to return a list of matching experiments, as well as their
// total_size. The total_size does not reflect the page size.
func (s *TaskStore) ListTasks(filterContext *common.FilterContext, opts *list.Options) ([]*model.Task, int, string, error) {
	errorF := func(err error) ([]*model.Task, int, string, error) {
		return nil, 0, "", util.NewInternalServerError(err, "Failed to list tasks: %v", err)
	}

	// SQL for getting the filtered and paginated rows
	sqlBuilder := sq.Select(taskColumns...).From("tasks")
	if filterContext.ReferenceKey != nil && filterContext.ReferenceKey.Type == common.Pipeline {
		sqlBuilder = sqlBuilder.Where(sq.Eq{"PipelineName": filterContext.ReferenceKey.ID})
	}
	sqlBuilder = opts.AddFilterToSelect(sqlBuilder)

	rowsSql, rowsArgs, err := opts.AddPaginationToSelect(sqlBuilder).ToSql()
	if err != nil {
		return errorF(err)
	}

	// SQL for getting total size. This matches the query to get all the rows above, in order
	// to do the same filter, but counts instead of scanning the rows.
	sqlBuilder = sq.Select("count(*)").From("tasks")
	if filterContext.ReferenceKey != nil && filterContext.ReferenceKey.Type == common.Pipeline {
		sqlBuilder = sqlBuilder.Where(sq.Eq{"PipelineName": filterContext.ReferenceKey.ID})
	}
	sizeSql, sizeArgs, err := opts.AddFilterToSelect(sqlBuilder).ToSql()
	if err != nil {
		return errorF(err)
	}

	// Use a transaction to make sure we're returning the total_size of the same rows queried
	tx, err := s.db.Begin()
	if err != nil {
		glog.Errorf("Failed to start transaction to list jobs")
		return errorF(err)
	}

	rows, err := tx.Query(rowsSql, rowsArgs...)
	if err != nil {
		tx.Rollback()
		return errorF(err)
	}
	exps, err := s.scanRows(rows)
	if err != nil {
		tx.Rollback()
		return errorF(err)
	}
	rows.Close()

	sizeRow, err := tx.Query(sizeSql, sizeArgs...)
	if err != nil {
		tx.Rollback()
		return errorF(err)
	}
	total_size, err := list.ScanRowToTotalSize(sizeRow)
	if err != nil {
		tx.Rollback()
		return errorF(err)
	}
	sizeRow.Close()

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

