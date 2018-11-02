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
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

type JobStoreInterface interface {
	ListJobs(filterContext *common.FilterContext, paginationContext *common.PaginationContext) ([]model.Job, string, error)
	GetJob(id string) (*model.Job, error)
	CreateJob(*model.Job) (*model.Job, error)
	DeleteJob(id string) error
	EnableJob(id string, enabled bool) error
	UpdateJob(swf *util.ScheduledWorkflow) error
}

type JobStore struct {
	db                     *DB
	resourceReferenceStore *ResourceReferenceStore
	time                   util.TimeInterface
}

func (s *JobStore) ListJobs(
	filterContext *common.FilterContext, paginationContext *common.PaginationContext) ([]model.Job, string, error) {
	queryJobTable := func(request *common.PaginationContext) ([]model.ListableDataModel, error) {
		return s.queryJobTable(filterContext, request)
	}
	models, pageToken, err := listModel(paginationContext, queryJobTable)
	if err != nil {
		return nil, "", util.Wrap(err, "List jobs failed.")
	}
	return s.toJobMetadatas(models), pageToken, err
}

func (s *JobStore) queryJobTable(
	filterContext *common.FilterContext, paginationContext *common.PaginationContext) ([]model.ListableDataModel, error) {
	sqlBuilder := s.selectJob()

	// Add filter condition
	sqlBuilder, err := s.toFilteredQuery(sqlBuilder, filterContext)
	if err != nil {
		return nil, util.Wrap(err, "Failed to create query to list job.")
	}

	// Add pagination condition
	sql, args, err := toPaginationQuery(sqlBuilder, paginationContext).Limit(uint64(paginationContext.PageSize)).ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create query to list jobs: %v",
			err.Error())
	}

	rows, err := s.db.Query(sql, args...)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to list jobs: %v",
			err.Error())
	}
	defer rows.Close()
	jobs, err := s.scanRows(rows)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to list jobs: %v",
			err.Error())
	}
	return s.toListableModels(jobs), nil
}

func (s *JobStore) toFilteredQuery(selectBuilder sq.SelectBuilder, filterContext *common.FilterContext) (sq.SelectBuilder, error) {
	sql, args, err := selectBuilder.ToSql()
	if err != nil {
		return selectBuilder, util.NewInternalServerError(err, "Failed to append filter condition to list job: %v",
			err.Error())
	}
	if filterContext.ReferenceKey != nil {
		selectBuilder = sq.Select("list_job.*").
			From("resource_references AS rf").
			LeftJoin(fmt.Sprintf("(%s) as list_job on list_job.UUID=rf.ResourceUUID", sql), args...).
			Where(sq.And{
				sq.Eq{"rf.ReferenceUUID": filterContext.ID},
				sq.Eq{"rf.ReferenceType": filterContext.Type}})
	}
	return selectBuilder, nil
}

func (s *JobStore) GetJob(id string) (*model.Job, error) {
	sql, args, err := s.selectJob().
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
	return &jobs[0], nil
}

func (s *JobStore) selectJob() sq.SelectBuilder {
	resourceRefConcatQuery := s.db.Concat([]string{`"["`, s.db.GroupConcat("r.Payload", ","), `"]"`}, "")
	return sq.
		Select("jobs.*", resourceRefConcatQuery+" AS refs").
		From("jobs").
		// Append all the resource references for the run as a json column
		LeftJoin("resource_references AS r ON jobs.UUID=r.ResourceUUID").
		Where(sq.Eq{"r.ResourceType": common.Job}).GroupBy("jobs.UUID")
}

func (s *JobStore) scanRows(r *sql.Rows) ([]model.Job, error) {
	var jobs []model.Job
	for r.Next() {
		var uuid, displayName, name, namespace, pipelineId, conditions,
			description, parameters, pipelineSpecManifest, workflowSpecManifest string
		var cronScheduleStartTimeInSec, cronScheduleEndTimeInSec,
			periodicScheduleStartTimeInSec, periodicScheduleEndTimeInSec, intervalSecond sql.NullInt64
		var cron, resourceReferencesInString sql.NullString
		var enabled bool
		var createdAtInSec, updatedAtInSec, maxConcurrency int64
		err := r.Scan(
			&uuid, &displayName, &name, &namespace, &description,
			&maxConcurrency, &createdAtInSec, &updatedAtInSec, &enabled,
			&cronScheduleStartTimeInSec, &cronScheduleEndTimeInSec, &cron,
			&periodicScheduleStartTimeInSec, &periodicScheduleEndTimeInSec, &intervalSecond,
			&pipelineId, &pipelineSpecManifest, &workflowSpecManifest, &parameters, &conditions, &resourceReferencesInString)
		if err != nil {
			return nil, err
		}
		resourceReferences, err := parseResourceReferences(resourceReferencesInString)
		jobs = append(jobs, model.Job{
			UUID:               uuid,
			DisplayName:        displayName,
			Name:               name,
			Namespace:          namespace,
			Description:        description,
			Enabled:            enabled,
			Conditions:         conditions,
			MaxConcurrency:     maxConcurrency,
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
			"Description":                    j.Description,
			"MaxConcurrency":                 j.MaxConcurrency,
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

func (s *JobStore) UpdateJob(swf *util.ScheduledWorkflow) error {
	now := s.time.Now().Unix()
	parameters, err := swf.ParametersAsString()
	if err != nil {
		return err
	}

	sql, args, err := sq.
		Update("jobs").
		SetMap(sq.Eq{
			"Name":                           swf.Name,
			"Namespace":                      swf.Namespace,
			"Enabled":                        swf.Spec.Enabled,
			"Conditions":                     swf.ConditionSummary(),
			"MaxConcurrency":                 swf.MaxConcurrencyOr0(),
			"Parameters":                     parameters,
			"UpdatedAtInSec":                 now,
			"CronScheduleStartTimeInSec":     PointerToNullInt64(swf.CronScheduleStartTimeInSecOrNull()),
			"CronScheduleEndTimeInSec":       PointerToNullInt64(swf.CronScheduleEndTimeInSecOrNull()),
			"Schedule":                       swf.CronOrEmpty(),
			"PeriodicScheduleStartTimeInSec": PointerToNullInt64(swf.PeriodicScheduleStartTimeInSecOrNull()),
			"PeriodicScheduleEndTimeInSec":   PointerToNullInt64(swf.PeriodicScheduleEndTimeInSecOrNull()),
			"IntervalSecond":                 swf.IntervalSecondOr0()}).
		Where(sq.Eq{"UUID": string(swf.UID)}).
		ToSql()
	if err != nil {
		return util.NewInternalServerError(err,
			"Error while creating query to update job with scheduled workflow: %v: %+v",
			err, swf.ScheduledWorkflow)
	}
	r, err := s.db.Exec(sql, args...)

	if err != nil {
		return util.NewInternalServerError(err,
			"Error while updating job with scheduled workflow: %v: %+v",
			err, swf.ScheduledWorkflow)
	}

	rowsAffected, err := r.RowsAffected()
	if err != nil {
		return util.NewInternalServerError(err,
			"Error getting affected rows while updating job with scheduled workflow: %v: %+v",
			err, swf.ScheduledWorkflow)
	}
	if rowsAffected <= 0 {
		return util.NewInvalidInputError(
			"There is no job corresponding to this scheduled workflow: %v/%v/%v",
			swf.UID, swf.Namespace, swf.Name)
	}

	return nil
}

func (s *JobStore) toListableModels(jobs []model.Job) []model.ListableDataModel {
	models := make([]model.ListableDataModel, len(jobs))
	for i := range models {
		models[i] = jobs[i]
	}
	return models
}

func (s *JobStore) toJobMetadatas(models []model.ListableDataModel) []model.Job {
	jobs := make([]model.Job, len(models))
	for i := range models {
		jobs[i] = models[i].(model.Job)
	}
	return jobs
}

// factory function for job store
func NewJobStore(db *DB, time util.TimeInterface) *JobStore {
	return &JobStore{
		db:                     db,
		resourceReferenceStore: NewResourceReferenceStore(db),
		time:                   time,
	}
}
