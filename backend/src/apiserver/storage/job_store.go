// Copyright 2018 Google LLC
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
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

var jobColumns = []string{"UUID", "DisplayName", "Name", "Namespace", "ServiceAccount", "Description", "MaxConcurrency",
	"NoCatchup", "CreatedAtInSec", "UpdatedAtInSec", "Enabled", "CronScheduleStartTimeInSec", "CronScheduleEndTimeInSec",
	"Schedule", "PeriodicScheduleStartTimeInSec", "PeriodicScheduleEndTimeInSec", "IntervalSecond",
	"PipelineId", "PipelineName", "PipelineSpecManifest", "WorkflowSpecManifest", "Parameters", "Conditions",
}

type JobStoreInterface interface {
	ListJobs(filterContext *common.FilterContext, opts *list.Options) ([]*model.Job, int, string, error)
	GetJob(id string) (*model.Job, error)
	CreateJob(*model.Job) (*model.Job, error)
	DeleteJob(id string) error
	EnableJob(id string, enabled bool) error
	UpdateJob(*model.Job) error
}

type JobStore struct {
	db                     *DB
	resourceReferenceStore *ResourceReferenceStore
	time                   util.TimeInterface
}

// Runs two SQL queries in a transaction to return a list of matching jobs, as well as their
// total_size. The total_size does not reflect the page size, but it does reflect the number of jobs
// matching the supplied filters and resource references.
func (s *JobStore) ListJobs(
	filterContext *common.FilterContext, opts *list.Options) ([]*model.Job, int, string, error) {
	errorF := func(err error) ([]*model.Job, int, string, error) {
		return nil, 0, "", util.NewInternalServerError(err, "Failed to list jobs: %v", err)
	}

	rowsSql, rowsArgs, err := s.buildSelectJobsQuery(false, opts, filterContext)
	if err != nil {
		return errorF(err)
	}

	sizeSql, sizeArgs, err := s.buildSelectJobsQuery(true, opts, filterContext)
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
		return errorF(err)
	}
	jobs, err := s.scanRows(rows)
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
		glog.Errorf("Failed to commit transaction to list jobs")
		return errorF(err)
	}

	if len(jobs) <= opts.PageSize {
		return jobs, total_size, "", nil
	}

	npt, err := opts.NextPageToken(jobs[opts.PageSize])
	return jobs[:opts.PageSize], total_size, npt, err
}

func (s *JobStore) buildSelectJobsQuery(selectCount bool, opts *list.Options,
	filterContext *common.FilterContext) (string, []interface{}, error) {

	var filteredSelectBuilder sq.SelectBuilder
	var err error

	refKey := filterContext.ReferenceKey
	if refKey != nil && refKey.Type == common.Namespace {
		filteredSelectBuilder, err = list.FilterOnNamespace("jobs", jobColumns,
			selectCount, refKey.ID)
	} else {
		filteredSelectBuilder, err = list.FilterOnResourceReference("jobs", jobColumns,
			common.Job, selectCount, filterContext)
	}
	if err != nil {
		return "", nil, util.NewInternalServerError(err, "Failed to list jobs: %v", err)
	}

	sqlBuilder := opts.AddFilterToSelect(filteredSelectBuilder)

	// If we're not just counting, then also add select columns and perform a left join
	// to get resource reference information. Also add pagination.
	if !selectCount {
		sqlBuilder = opts.AddPaginationToSelect(sqlBuilder)
		sqlBuilder = s.addResourceReferences(sqlBuilder)
		sqlBuilder = opts.AddSortingToSelect(sqlBuilder)
	}
	sql, args, err := sqlBuilder.ToSql()
	if err != nil {
		return "", nil, util.NewInternalServerError(err, "Failed to list jobs: %v", err)
	}

	return sql, args, err
}

func (s *JobStore) GetJob(id string) (*model.Job, error) {
	sql, args, err := s.addResourceReferences(sq.Select(jobColumns...).From("jobs")).
		Where(sq.Eq{"uuid": id}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create query to get job: %v",
			err.Error())
	}
	row, err := s.db.Query(sql, args...)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to get job: %v",
			err.Error())
	}
	defer row.Close()
	jobs, err := s.scanRows(row)
	if err != nil || len(jobs) > 1 {
		return nil, util.NewInternalServerError(err, "Failed to get job: %v", err.Error())
	}
	if len(jobs) == 0 {
		return nil, util.NewResourceNotFoundError("Job", fmt.Sprint(id))
	}
	return jobs[0], nil
}

func (s *JobStore) addResourceReferences(filteredSelectBuilder sq.SelectBuilder) sq.SelectBuilder {
	resourceRefConcatQuery := s.db.Concat([]string{`"["`, s.db.GroupConcat("r.Payload", ","), `"]"`}, "")
	return sq.
		Select("jobs.*", resourceRefConcatQuery+" AS refs").
		FromSelect(filteredSelectBuilder, "jobs").
		// Append all the resource references for the run as a json column
		LeftJoin("(select * from resource_references where ResourceType='Job') AS r ON jobs.UUID=r.ResourceUUID").
		GroupBy("jobs.UUID")
}

func (s *JobStore) scanRows(r *sql.Rows) ([]*model.Job, error) {
	var jobs []*model.Job
	for r.Next() {
		var uuid, displayName, name, namespace, pipelineId, pipelineName, conditions, serviceAccount,
			description, parameters, pipelineSpecManifest, workflowSpecManifest string
		var cronScheduleStartTimeInSec, cronScheduleEndTimeInSec,
			periodicScheduleStartTimeInSec, periodicScheduleEndTimeInSec, intervalSecond sql.NullInt64
		var cron, resourceReferencesInString sql.NullString
		var enabled, noCatchup bool
		var createdAtInSec, updatedAtInSec, maxConcurrency int64
		err := r.Scan(
			&uuid, &displayName, &name, &namespace, &serviceAccount, &description,
			&maxConcurrency, &noCatchup, &createdAtInSec, &updatedAtInSec, &enabled,
			&cronScheduleStartTimeInSec, &cronScheduleEndTimeInSec, &cron,
			&periodicScheduleStartTimeInSec, &periodicScheduleEndTimeInSec, &intervalSecond,
			&pipelineId, &pipelineName, &pipelineSpecManifest, &workflowSpecManifest, &parameters, &conditions, &resourceReferencesInString)
		if err != nil {
			return nil, err
		}
		resourceReferences, err := parseResourceReferences(resourceReferencesInString)
		jobs = append(jobs, &model.Job{
			UUID:               uuid,
			DisplayName:        displayName,
			Name:               name,
			Namespace:          namespace,
			ServiceAccount:     serviceAccount,
			Description:        description,
			Enabled:            enabled,
			Conditions:         conditions,
			MaxConcurrency:     maxConcurrency,
			NoCatchup:          noCatchup,
			ResourceReferences: resourceReferences,
			Trigger: model.Trigger{
				CronSchedule: model.CronSchedule{
					CronScheduleStartTimeInSec: NullInt64ToPointer(cronScheduleStartTimeInSec),
					CronScheduleEndTimeInSec:   NullInt64ToPointer(cronScheduleEndTimeInSec),
					Cron:                       NullStringToPointer(cron),
				},
				PeriodicSchedule: model.PeriodicSchedule{
					PeriodicScheduleStartTimeInSec: NullInt64ToPointer(periodicScheduleStartTimeInSec),
					PeriodicScheduleEndTimeInSec:   NullInt64ToPointer(periodicScheduleEndTimeInSec),
					IntervalSecond:                 NullInt64ToPointer(intervalSecond),
				},
			},
			PipelineSpec: model.PipelineSpec{
				PipelineId:           pipelineId,
				PipelineName:         pipelineName,
				PipelineSpecManifest: pipelineSpecManifest,
				WorkflowSpecManifest: workflowSpecManifest,
				Parameters:           parameters,
			},
			CreatedAtInSec: createdAtInSec,
			UpdatedAtInSec: updatedAtInSec,
		})
	}
	return jobs, nil
}

func (s *JobStore) DeleteJob(id string) error {
	jobSql, jobArgs, err := sq.Delete("jobs").Where(sq.Eq{"UUID": id}).ToSql()
	if err != nil {
		return util.NewInternalServerError(err,
			"Failed to create query to delete job: %s", id)
	}
	// Use a transaction to make sure both run and its resource references are stored.
	tx, err := s.db.Begin()
	if err != nil {
		return util.NewInternalServerError(err, "Failed to create a new transaction to delete job.")
	}
	_, err = tx.Exec(jobSql, jobArgs...)
	if err != nil {
		tx.Rollback()
		return util.NewInternalServerError(err, "Failed to delete job %s from table", id)
	}
	err = s.resourceReferenceStore.DeleteResourceReferences(tx, id, common.Job)
	if err != nil {
		tx.Rollback()
		return util.NewInternalServerError(err, "Failed to delete resource references from table for job %v ", id)
	}
	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		return util.NewInternalServerError(err, "Failed to delete job %v and its resource references from table", id)
	}
	return nil
}

func (s *JobStore) CreateJob(j *model.Job) (*model.Job, error) {
	jobSql, jobArgs, err := sq.
		Insert("jobs").
		SetMap(sq.Eq{
			"UUID":                           j.UUID,
			"DisplayName":                    j.DisplayName,
			"Name":                           j.Name,
			"Namespace":                      j.Namespace,
			"ServiceAccount":                 j.ServiceAccount,
			"Description":                    j.Description,
			"MaxConcurrency":                 j.MaxConcurrency,
			"NoCatchup":                      j.NoCatchup,
			"Enabled":                        j.Enabled,
			"Conditions":                     j.Conditions,
			"CronScheduleStartTimeInSec":     PointerToNullInt64(j.CronScheduleStartTimeInSec),
			"CronScheduleEndTimeInSec":       PointerToNullInt64(j.CronScheduleEndTimeInSec),
			"Schedule":                       PointerToNullString(j.Cron),
			"PeriodicScheduleStartTimeInSec": PointerToNullInt64(j.PeriodicScheduleStartTimeInSec),
			"PeriodicScheduleEndTimeInSec":   PointerToNullInt64(j.PeriodicScheduleEndTimeInSec),
			"IntervalSecond":                 PointerToNullInt64(j.IntervalSecond),
			"CreatedAtInSec":                 j.CreatedAtInSec,
			"UpdatedAtInSec":                 j.UpdatedAtInSec,
			"PipelineId":                     j.PipelineId,
			"PipelineName":                   j.PipelineName,
			"PipelineSpecManifest":           j.PipelineSpecManifest,
			"WorkflowSpecManifest":           j.WorkflowSpecManifest,
			"Parameters":                     j.Parameters,
		}).ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create query to add job to job table: %v",
			err.Error())
	}

	// Use a transaction to make sure both job and its resource references are stored.
	tx, err := s.db.Begin()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create a new transaction to create job.")
	}
	_, err = tx.Exec(jobSql, jobArgs...)
	if err != nil {
		tx.Rollback()
		return nil, util.NewInternalServerError(err, "Failed to store job %v to table", j.Name)
	}
	err = s.resourceReferenceStore.CreateResourceReferences(tx, j.ResourceReferences)
	if err != nil {
		tx.Rollback()
		return nil, util.NewInternalServerError(err, "Failed to store resource references to table for job %v ", j.Name)
	}

	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		return nil, util.NewInternalServerError(err, "Failed to store job %v and its resource references to table", j.Name)
	}
	return j, nil
}

func (s *JobStore) EnableJob(id string, enabled bool) error {
	now := s.time.Now().Unix()
	sql, args, err := sq.
		Update("jobs").
		SetMap(sq.Eq{
			"Enabled":        enabled,
			"UpdatedAtInSec": now}).
		Where(sq.Eq{"UUID": string(id)}).
		Where(sq.Eq{"Enabled": !enabled}).
		ToSql()
	if err != nil {
		return util.NewInternalServerError(err, "Error when creating query to enable job %v to %v", id, enabled)
	}
	_, err = s.db.Exec(sql, args...)
	if err != nil {
		return util.NewInternalServerError(err, "Error when enabling job %v to %v", id, enabled)
	}
	return nil
}

func (s *JobStore) UpdateJob(j *model.Job) error {
	now := s.time.Now().Unix()

	sqlStatement, args, err := sq.
		Update("jobs").
		SetMap(sq.Eq{
			"Name":                           j.Name,
			"DisplayName":                    j.DisplayName,
			"Namespace":                      j.Namespace,
			"ServiceAccount":                 j.ServiceAccount,
			"Description":                    j.Description,
			"MaxConcurrency":                 j.MaxConcurrency,
			"NoCatchup":                      j.NoCatchup,
			"Enabled":                        j.Enabled,
			"Conditions":                     j.Conditions,
			"CronScheduleStartTimeInSec":     PointerToNullInt64(j.CronScheduleStartTimeInSec),
			"CronScheduleEndTimeInSec":       PointerToNullInt64(j.CronScheduleEndTimeInSec),
			"Schedule":                       PointerToNullString(j.Cron),
			"PeriodicScheduleStartTimeInSec": PointerToNullInt64(j.PeriodicScheduleStartTimeInSec),
			"PeriodicScheduleEndTimeInSec":   PointerToNullInt64(j.PeriodicScheduleEndTimeInSec),
			"IntervalSecond":                 PointerToNullInt64(j.IntervalSecond),
			"UpdatedAtInSec":                 now,
			"PipelineId":                     j.PipelineId,
			"PipelineName":                   j.PipelineName,
			"PipelineSpecManifest":           j.PipelineSpecManifest,
			"WorkflowSpecManifest":           j.WorkflowSpecManifest,
			"Parameters":                     j.Parameters}).
		Where(sq.Eq{"UUID": j.UUID, "UpdatedAtInSec": j.UpdatedAtInSec}).
		ToSql()
	if err != nil {
		return util.NewInternalServerError(err,
			"Error while creating query to update job: %q", j.Name)
	}

	r, err := s.db.Exec(sqlStatement, args...)
	if err != nil {
		return util.NewInternalServerError(err, "Error while updating job: %q", j.Name)
	}

	rowsAffected, err := r.RowsAffected()
	if err != nil {
		return util.NewInternalServerError(err,
			"error getting affected rows while updating job: %q", j.Name)
	}
	if rowsAffected <= 0 {
		return util.NewBadRequestError(err, "job not found or conflict: %q", j.Name)
	}

	return nil
}

// factory function for job store
func NewJobStore(db *DB, time util.TimeInterface) *JobStore {
	return &JobStore{
		db:                     db,
		resourceReferenceStore: NewResourceReferenceStore(db),
		time:                   time,
	}
}
