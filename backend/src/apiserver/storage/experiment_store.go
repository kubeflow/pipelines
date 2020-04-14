package storage

import (
	"database/sql"

	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

type ExperimentStoreInterface interface {
	ListExperiments(filterContext *common.FilterContext, opts *list.Options) ([]*model.Experiment, int, string, error)
	GetExperiment(uuid string) (*model.Experiment, error)
	CreateExperiment(*model.Experiment) (*model.Experiment, error)
	DeleteExperiment(uuid string) error
}

type ExperimentStore struct {
	db                     *DB
	time                   util.TimeInterface
	uuid                   util.UUIDGeneratorInterface
	resourceReferenceStore *ResourceReferenceStore
	defaultExperimentStore *DefaultExperimentStore
}

// Runs two SQL queries in a transaction to return a list of matching experiments, as well as their
// total_size. The total_size does not reflect the page size.
func (s *ExperimentStore) ListExperiments(filterContext *common.FilterContext, opts *list.Options) ([]*model.Experiment, int, string, error) {
	errorF := func(err error) ([]*model.Experiment, int, string, error) {
		return nil, 0, "", util.NewInternalServerError(err, "Failed to list experiments: %v", err)
	}

	// SQL for getting the filtered and paginated rows
	sqlBuilder := sq.Select("*").From("experiments")
	if filterContext.ReferenceKey != nil && filterContext.ReferenceKey.Type == common.Namespace {
		sqlBuilder = sqlBuilder.Where(sq.Eq{"Namespace": filterContext.ReferenceKey.ID})
	}
	sqlBuilder = opts.AddFilterToSelect(sqlBuilder)

	rowsSql, rowsArgs, err := opts.AddPaginationToSelect(sqlBuilder).ToSql()
	if err != nil {
		return errorF(err)
	}

	// SQL for getting total size. This matches the query to get all the rows above, in order
	// to do the same filter, but counts instead of scanning the rows.
	sqlBuilder = sq.Select("count(*)").From("experiments")
	if filterContext.ReferenceKey != nil && filterContext.ReferenceKey.Type == common.Namespace {
		sqlBuilder = sqlBuilder.Where(sq.Eq{"Namespace": filterContext.ReferenceKey.ID})
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

func (s *ExperimentStore) GetExperiment(uuid string) (*model.Experiment, error) {
	sql, args, err := sq.
		Select("*").
		From("experiments").
		Where(sq.Eq{"uuid": uuid}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to get experiment: %v", err.Error())
	}
	r, err := s.db.Query(sql, args...)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to get experiment: %v", err.Error())
	}
	defer r.Close()
	experiments, err := s.scanRows(r)

	if err != nil || len(experiments) > 1 {
		return nil, util.NewInternalServerError(err, "Failed to get experiment: %v", err.Error())
	}
	if len(experiments) == 0 {
		return nil, util.NewResourceNotFoundError("Experiment", fmt.Sprint(uuid))
	}
	return experiments[0], nil
}

func (s *ExperimentStore) scanRows(rows *sql.Rows) ([]*model.Experiment, error) {
	var experiments []*model.Experiment
	for rows.Next() {
		var uuid, name, description, namespace string
		var createdAtInSec int64
		err := rows.Scan(&uuid, &name, &description, &createdAtInSec, &namespace)
		if err != nil {
			return experiments, err
		}
		experiments = append(experiments, &model.Experiment{
			UUID:           uuid,
			Name:           name,
			Description:    description,
			CreatedAtInSec: createdAtInSec,
			Namespace:      namespace,
		})
	}
	return experiments, nil
}

func (s *ExperimentStore) CreateExperiment(experiment *model.Experiment) (*model.Experiment, error) {
	newExperiment := *experiment
	now := s.time.Now().Unix()
	newExperiment.CreatedAtInSec = now
	id, err := s.uuid.NewRandom()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create an experiment id.")
	}
	newExperiment.UUID = id.String()
	sql, args, err := sq.
		Insert("experiments").
		SetMap(sq.Eq{
			"UUID":           newExperiment.UUID,
			"CreatedAtInSec": newExperiment.CreatedAtInSec,
			"Name":           newExperiment.Name,
			"Description":    newExperiment.Description,
			"Namespace":      newExperiment.Namespace,
		}).
		ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create query to insert experiment to experiment table: %v",
			err.Error())
	}
	_, err = s.db.Exec(sql, args...)
	if err != nil {
		if s.db.IsDuplicateError(err) {
			return nil, util.NewInvalidInputError(
				"Failed to create a new experiment. The name %v already exists. Please specify a new name.", experiment.Name)
		}
		return nil, util.NewInternalServerError(err, "Failed to add experiment to experiment table: %v",
			err.Error())
	}
	return &newExperiment, nil
}

func (s *ExperimentStore) DeleteExperiment(id string) error {
	experimentSql, experimentArgs, err := sq.Delete("experiments").Where(sq.Eq{"UUID": id}).ToSql()
	if err != nil {
		return util.NewInternalServerError(err,
			"Failed to create query to delete experiment: %s", id)
	}
	// Use a transaction to make sure both experiment and its resource references are deleted.
	tx, err := s.db.Begin()
	if err != nil {
		return util.NewInternalServerError(err, "Failed to create a new transaction to delete experiment.")
	}
	_, err = tx.Exec(experimentSql, experimentArgs...)
	if err != nil {
		tx.Rollback()
		return util.NewInternalServerError(err, "Failed to delete experiment %s from table", id)
	}
	err = s.defaultExperimentStore.UnsetDefaultExperimentIdIfIdMatches(tx, id)
	if err != nil {
		tx.Rollback()
		return util.NewInternalServerError(err, "Failed to clear default experiment ID for experiment %v ", id)
	}
	err = s.resourceReferenceStore.DeleteResourceReferences(tx, id, common.Run)
	if err != nil {
		tx.Rollback()
		return util.NewInternalServerError(err, "Failed to delete resource references from table for experiment %v ", id)
	}
	err = tx.Commit()
	if err != nil {
		return util.NewInternalServerError(err, "Failed to delete experiment %v and its resource references from table", id)
	}
	return nil
}

// factory function for experiment store
func NewExperimentStore(db *DB, time util.TimeInterface, uuid util.UUIDGeneratorInterface) *ExperimentStore {
	return &ExperimentStore{
		db:                     db,
		time:                   time,
		uuid:                   uuid,
		resourceReferenceStore: NewResourceReferenceStore(db),
		defaultExperimentStore: NewDefaultExperimentStore(db),
	}
}
