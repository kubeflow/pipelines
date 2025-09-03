// Copyright 2018 The Kubeflow Authors
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

package storage

import (
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

type ExperimentStoreInterface interface {
	CreateExperiment(*model.Experiment) (*model.Experiment, error)
	GetExperiment(uuid string) (*model.Experiment, error)
	GetExperimentByNameNamespace(name string, namespace string) (*model.Experiment, error)
	ListExperiments(filterContext *model.FilterContext, opts *list.Options) ([]*model.Experiment, int, string, error)
	ArchiveExperiment(expId string) error
	UnarchiveExperiment(expId string) error
	DeleteExperiment(uuid string) error
	SetLastRunTimestamp(run *model.Run) error
}

type ExperimentStore struct {
	db                     *sql.DB
	time                   util.TimeInterface
	uuid                   util.UUIDGeneratorInterface
	resourceReferenceStore *ResourceReferenceStore
	defaultExperimentStore *DefaultExperimentStore
	dialect                dialect.DBDialect
}

var experimentColumns = []string{
	"UUID",
	"Name",
	"Description",
	"CreatedAtInSec",
	"LastRunCreatedAtInSec",
	"Namespace",
	"StorageState",
}

// Runs two SQL queries in a transaction to return a list of matching experiments, as well as their
// total_size. The total_size does not reflect the page size.
func (s *ExperimentStore) ListExperiments(filterContext *model.FilterContext, opts *list.Options) ([]*model.Experiment, int, string, error) {
	errorF := func(err error) ([]*model.Experiment, int, string, error) {
		return nil, 0, "", util.NewInternalServerError(err, "Failed to list experiments: %v", err)
	}
	q := s.dialect.QuoteIdentifier
	opts.SetQuote(s.dialect.QuoteIdentifier)

	// Fix for wrong sort key prefix.
	opts.SetSortByFieldPrefix(q("experiments") + ".")
	opts.SetKeyFieldPrefix(q("experiments") + ".")

	// SQL for getting the filtered and paginated rows
	qb := s.dialect.QueryBuilder()
	cols := make([]string, len(experimentColumns))
	for i, c := range experimentColumns {
		cols[i] = q(c)
	}
	sqlBuilder := qb.Select(cols...).From(q("experiments"))
	if filterContext.ReferenceKey != nil && filterContext.ReferenceKey.Type == model.NamespaceResourceType {
		sqlBuilder = sqlBuilder.Where(sq.Eq{q("Namespace"): filterContext.ID})
	}
	sqlBuilder = opts.AddFilterToSelect(sqlBuilder)

	rowsSql, rowsArgs, err := opts.AddPaginationToSelect(sqlBuilder).ToSql()
	if err != nil {
		return errorF(err)
	}

	// SQL for getting total size. This matches the query to get all the rows above, in order
	// to do the same filter, but counts instead of scanning the rows.
	sqlBuilder = qb.Select("count(*)").From(q("experiments"))
	if filterContext.ReferenceKey != nil && filterContext.ReferenceKey.Type == model.NamespaceResourceType {
		sqlBuilder = sqlBuilder.Where(sq.Eq{q("Namespace"): filterContext.ID})
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
	glog.Infof("ListExperiments SQL=%s ARGS=%v", rowsSql, rowsArgs)
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
	totalSize, err := list.ScanRowToTotalSize(sizeRow)
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
		return exps, totalSize, "", nil
	}

	npt, err := opts.NextPageToken(exps[opts.PageSize])
	return exps[:opts.PageSize], totalSize, npt, err
}

func (s *ExperimentStore) GetExperiment(uuid string) (*model.Experiment, error) {
	q := s.dialect.QuoteIdentifier
	qb := s.dialect.QueryBuilder()
	cols := make([]string, len(experimentColumns))
	for i, c := range experimentColumns {
		cols[i] = q(c)
	}
	sql, args, err := qb.
		Select(cols...).
		From(q("experiments")).
		Where(sq.Eq{q("UUID"): uuid}).
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

func (s *ExperimentStore) GetExperimentByNameNamespace(name string, namespace string) (*model.Experiment, error) {
	q := s.dialect.QuoteIdentifier
	qb := s.dialect.QueryBuilder()
	cols := make([]string, len(experimentColumns))
	for i, c := range experimentColumns {
		cols[i] = q(c)
	}
	sql, args, err := qb.
		Select(cols...).
		From(q("experiments")).
		Where(sq.Eq{
			q("Name"):      name,
			q("Namespace"): namespace,
		}).
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
		return nil, util.NewResourceNotFoundError("Experiment", fmt.Sprint(name))
	}
	return experiments[0], nil
}

func (s *ExperimentStore) scanRows(rows *sql.Rows) ([]*model.Experiment, error) {
	var experiments []*model.Experiment
	for rows.Next() {
		var uuid, name, description, namespace, storageState string
		var createdAtInSec int64
		var lastRunCreatedAtInSec int64
		err := rows.Scan(&uuid, &name, &description, &createdAtInSec, &lastRunCreatedAtInSec, &namespace, &storageState)
		if err != nil {
			return experiments, err
		}
		experiment := &model.Experiment{
			UUID:                  uuid,
			Name:                  name,
			Description:           description,
			CreatedAtInSec:        createdAtInSec,
			LastRunCreatedAtInSec: lastRunCreatedAtInSec,
			Namespace:             namespace,
			StorageState:          model.StorageState(storageState).ToV2(),
		}
		// Since storage state is a field added after initial KFP release, it is possible that existing experiments don't have this field and we use AVAILABLE in that case.
		if experiment.StorageState == "" || experiment.StorageState == model.StorageStateUnspecified {
			experiment.StorageState = model.StorageStateAvailable
		}
		experiments = append(experiments, experiment)
	}
	return experiments, nil
}

func (s *ExperimentStore) CreateExperiment(experiment *model.Experiment) (*model.Experiment, error) {
	newExperiment := *experiment
	now := s.time.Now().Unix()
	newExperiment.CreatedAtInSec = now
	// When an experiment has no runs
	// we default to "1970-01-01T00:00:00Z"
	newExperiment.LastRunCreatedAtInSec = 0
	id, err := s.uuid.NewRandom()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create an experiment id")
	}
	newExperiment.UUID = id.String()

	if newExperiment.StorageState == "" || newExperiment.StorageState == model.StorageStateUnspecified {
		newExperiment.StorageState = model.StorageStateAvailable
	}
	if !newExperiment.StorageState.IsValid() {
		return nil, util.NewInvalidInputError("Invalid value for StorageState field: %q", newExperiment.StorageState)
	}
	q := s.dialect.QuoteIdentifier
	qb := s.dialect.QueryBuilder()
	sql, args, err := qb.
		Insert(q("experiments")).
		SetMap(map[string]interface{}{
			q("UUID"):                  newExperiment.UUID,
			q("CreatedAtInSec"):        newExperiment.CreatedAtInSec,
			q("LastRunCreatedAtInSec"): newExperiment.LastRunCreatedAtInSec,
			q("Name"):                  newExperiment.Name,
			q("Description"):           newExperiment.Description,
			q("Namespace"):             newExperiment.Namespace,
			q("StorageState"):          newExperiment.StorageState.ToV2().ToString(),
		}).
		ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create query to insert experiment to experiment table: %v",
			err.Error())
	}
	_, err = s.db.Exec(sql, args...)
	if err != nil {
		if isDuplicateError(s.dialect, err) {
			return nil, util.NewAlreadyExistError(
				"Failed to create a new experiment. The name %v already exists. Please specify a new name", experiment.Name)
		}
		return nil, util.NewInternalServerError(err, "Failed to add experiment to experiment table: %v",
			err.Error())
	}
	return &newExperiment, nil
}

func (s *ExperimentStore) DeleteExperiment(id string) error {
	q := s.dialect.QuoteIdentifier
	qb := s.dialect.QueryBuilder()
	experimentSQL, experimentArgs, err := qb.Delete(q("experiments")).Where(sq.Eq{q("UUID"): id}).ToSql()
	if err != nil {
		return util.NewInternalServerError(err,
			"Failed to create query to delete experiment: %s", id)
	}
	// Use a transaction to make sure both experiment and its resource references are deleted.
	tx, err := s.db.Begin()
	if err != nil {
		return util.NewInternalServerError(err, "Failed to create a new transaction to delete experiment")
	}
	_, err = tx.Exec(experimentSQL, experimentArgs...)
	if err != nil {
		tx.Rollback()
		return util.NewInternalServerError(err, "Failed to delete experiment %s from table", id)
	}
	err = s.defaultExperimentStore.UnsetDefaultExperimentIdIfIdMatches(tx, id)
	if err != nil {
		tx.Rollback()
		return util.NewInternalServerError(err, "Failed to clear default experiment ID for experiment %v ", id)
	}
	err = s.resourceReferenceStore.DeleteResourceReferences(tx, id, model.RunResourceType)
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

func (s *ExperimentStore) ArchiveExperiment(expId string) error {
	// ArchiveExperiment results in
	// 1. The experiment getting archived
	// 2. All the runs in the experiment getting archived no matter what previous storage state they are in
	q := s.dialect.QuoteIdentifier
	qb := s.dialect.QueryBuilder()
	sql, args, err := qb.
		Update(q("experiments")).
		SetMap(sq.Eq{
			q("StorageState"): model.StorageStateArchived.ToString(),
		}).
		Where(sq.Eq{q("UUID"): expId}).
		ToSql()
	if err != nil {
		return util.NewInternalServerError(err,
			"Failed to create query to archive experiment %s. error: '%v'", expId, err.Error())
	}

	// Build a subquery to select ResourceUUIDs of runs that belong to the experiment via resource_references.
	// This works across dialects (Postgres/MySQL/SQLite) and removes the need for the legacy UpdateWithJointOrFrom().
	subSel, subArgs, err := qb.
		Select(q("ResourceUUID")).
		From(q("resource_references")).
		Where(sq.Eq{
			q("ResourceType"):  model.RunResourceType,
			q("ReferenceUUID"): expId,
			q("ReferenceType"): model.ExperimentResourceType,
		}).
		ToSql()
	if err != nil {
		return util.NewInternalServerError(err,
			"Failed to create subquery to archive runs with experiment reference being %s. error: '%v'", expId, err.Error())
	}

	// UPDATE run_details SET StorageState = 'ARCHIVED' WHERE UUID IN (subSel)
	b := qb.
		Update(q("run_details")).
		SetMap(sq.Eq{q("StorageState"): model.StorageStateArchived.ToString()}).
		Where(sq.Expr(fmt.Sprintf("%s IN (%s)", q("UUID"), subSel), subArgs...))

	updateRunsSQL, updateRunsArgs, err := b.ToSql()
	if err != nil {
		return util.NewInternalServerError(err,
			"Failed to create query to archive runs with experiment reference being %s. error: '%v'", expId, err.Error())
	}

	updateRunsWithExperimentUUIDSql, updateRunsWithExperimentUUIDArgs, err := qb.
		Update(q("run_details")).
		SetMap(sq.Eq{
			q("StorageState"): model.StorageStateArchived.ToString(),
		}).
		Where(sq.Eq{q("ExperimentUUID"): expId}).
		Where(sq.NotEq{q("StorageState"): model.StorageStateArchived.ToString()}).
		ToSql()
	if err != nil {
		return util.NewInternalServerError(err,
			"Failed to create query to archive the runs in an experiment %s. error: '%v'", expId, err.Error())
	}

	var updateJobsArgs []interface{}
	now := s.time.Now().Unix()
	updateJobsArgs = append(updateJobsArgs, false, now, model.JobResourceType, expId, model.ExperimentResourceType)
	// TODO(gkcalat): deprecate resource_references table once we migrate to v2beta1 and switch to filtering on Job's `experiment_id' instead.
	// Build a subquery to select ResourceUUIDs of jobs that belong to the experiment via resource_references.
	subSelJob, subArgsJob, err := qb.
		Select(q("ResourceUUID")).
		From(q("resource_references")).
		Where(sq.Eq{
			q("ResourceType"):  model.JobResourceType,
			q("ReferenceUUID"): expId,
			q("ReferenceType"): model.ExperimentResourceType,
		}).
		ToSql()
	if err != nil {
		return util.NewInternalServerError(err,
			"Failed to create subquery to disable jobs with experiment reference being %s. error: '%v'", expId, err.Error())
	}

	// UPDATE jobs SET Enabled = false, UpdatedAtInSec = now WHERE UUID IN (subSelJob)
	bJobs := qb.
		Update(q("jobs")).
		SetMap(sq.Eq{
			q("Enabled"):        false,
			q("UpdatedAtInSec"): now,
		}).
		Where(sq.Expr(fmt.Sprintf("%s IN (%s)", q("UUID"), subSelJob), subArgsJob...))

	updateJobsSQL, updateJobsArgs, err := bJobs.ToSql()
	if err != nil {
		return util.NewInternalServerError(err,
			"Failed to create query to disable jobs in experiment %s. error: '%v'", expId, err.Error())
	}

	// In a single transaction, we update experiments, run_details and jobs tables.
	tx, err := s.db.Begin()
	if err != nil {
		return util.NewInternalServerError(err, "Failed to create a new transaction to archive an experiment")
	}

	_, err = tx.Exec(sql, args...)
	if err != nil {
		tx.Rollback()
		return util.NewInternalServerError(err,
			"Failed to archive experiment %s. error: '%v'", expId, err.Error())
	}

	_, err = tx.Exec(updateRunsSQL, updateRunsArgs...)
	if err != nil {
		tx.Rollback()
		return util.NewInternalServerError(err,
			"Failed to archive runs with experiment reference being %s. error: '%v'", expId, err.Error())
	}

	_, err = tx.Exec(updateRunsWithExperimentUUIDSql, updateRunsWithExperimentUUIDArgs...)
	if err != nil {
		tx.Rollback()
		return util.NewInternalServerError(err,
			"Failed to archive runs with ExperimentUUID being %s. error: '%v'", expId, err.Error())
	}

	_, err = tx.Exec(updateJobsSQL, updateJobsArgs...)
	if err != nil {
		tx.Rollback()
		return util.NewInternalServerError(err,
			"Failed to disable all jobs in an experiment %s. error: '%v'", expId, err.Error())
	}

	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		return util.NewInternalServerError(err, "Failed to archive an experiment %s and its runs", expId)
	}

	return nil
}

func (s *ExperimentStore) UnarchiveExperiment(expId string) error {
	// UnarchiveExperiment results in
	// 1. The experiment getting unarchived
	// 2. All the archived runs and disabled jobs will stay archived
	q := s.dialect.QuoteIdentifier
	qb := s.dialect.QueryBuilder()
	sql, args, err := qb.
		Update(q("experiments")).
		SetMap(sq.Eq{
			q("StorageState"): model.StorageStateAvailable.ToString(),
		}).
		Where(sq.Eq{q("UUID"): expId}).
		ToSql()
	if err != nil {
		return util.NewInternalServerError(err,
			"Failed to create query to unarchive experiment %s. error: '%v'", expId, err.Error())
	}

	_, err = s.db.Exec(sql, args...)
	if err != nil {
		return util.NewInternalServerError(err,
			"Failed to unarchive experiment %s. error: '%v'", expId, err.Error())
	}

	return nil
}

func (s *ExperimentStore) SetLastRunTimestamp(run *model.Run) error {
	expId := run.ExperimentId
	// SetLastRunTimestamp results in the experiment getting last_run_created_at updated
	q := s.dialect.QuoteIdentifier
	qb := s.dialect.QueryBuilder()
	query, args, err := qb.
		Update(q("experiments")).
		SetMap(sq.Eq{
			q("LastRunCreatedAtInSec"): run.CreatedAtInSec,
		}).
		Where(sq.Eq{q("UUID"): expId}).
		ToSql()
	if err != nil {
		return util.NewInternalServerError(err,
			"Failed to create query to set experiment LastRunCreatedAtInSec %s. error: '%v'", expId, err.Error())
	}
	_, err = s.db.Exec(query, args...)
	if err != nil {
		return util.NewInternalServerError(err,
			"Failed to set experiment LastRunCreatedAtInSec %s. error: '%v'", expId, err.Error())
	}
	return nil
}

// factory function for experiment store.
func NewExperimentStore(db *sql.DB, time util.TimeInterface, uuid util.UUIDGeneratorInterface, d dialect.DBDialect) *ExperimentStore {
	return &ExperimentStore{
		db:                     db,
		time:                   time,
		uuid:                   uuid,
		resourceReferenceStore: NewResourceReferenceStore(db, nil, d),
		defaultExperimentStore: NewDefaultExperimentStore(db, d),
		dialect:                d,
	}
}
