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
	"github.com/googleprivate/ml/backend/src/apiserver/common"
	"github.com/googleprivate/ml/backend/src/apiserver/model"
	"github.com/googleprivate/ml/backend/src/common/util"
)

type JobStoreInterface interface {
	ListJobs(context *common.PaginationContext) ([]model.Job, string, error)
	GetJob(id string) (*model.Job, error)
	CreateJob(*model.Job) (*model.Job, error)
	DeleteJob(id string) error
	EnableJob(id string, enabled bool) error
	UpdateJob(swf *util.ScheduledWorkflow) error
}

type JobStore struct {
	db   *DB
	time util.TimeInterface
}

func (s *JobStore) ListJobs(context *common.PaginationContext) ([]model.Job, string, error) {
	models, pageToken, err := listModel(context, s.queryJobTable)
	if err != nil {
		return nil, "", util.Wrap(err, "List jobs failed.")
	}
	return s.toJobMetadatas(models), pageToken, err
}

func (s *JobStore) queryJobTable(context *common.PaginationContext) ([]model.ListableDataModel, error) {
	sqlBuilder := sq.Select("*").From("jobs")
	sql, args, err := toPaginationQuery(sqlBuilder, context).Limit(uint64(context.PageSize)).ToSql()
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

func (s *JobStore) GetJob(id string) (*model.Job, error) {
	sql, args, err := sq.
		Select("*").
		From("jobs").
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

func (s *JobStore) scanRows(r *sql.Rows) ([]model.Job, error) {
	var jobs []model.Job
	for r.Next() {
		var uuid, displayName, name, namespace, pipelineId, conditions,
			description, parameters, pipelineSpecManifest, workflowSpecManifest string
		var cronScheduleStartTimeInSec, cronScheduleEndTimeInSec,
			periodicScheduleStartTimeInSec, periodicScheduleEndTimeInSec, intervalSecond sql.NullInt64
		var cron sql.NullString
		var enabled bool
		var createdAtInSec, updatedAtInSec, maxConcurrency int64
		err := r.Scan(
			&uuid, &displayName, &name, &namespace, &description,
			&maxConcurrency, &createdAtInSec, &updatedAtInSec, &enabled,
			&cronScheduleStartTimeInSec, &cronScheduleEndTimeInSec, &cron,
			&periodicScheduleStartTimeInSec, &periodicScheduleEndTimeInSec, &intervalSecond,
			&pipelineId, &pipelineSpecManifest, &workflowSpecManifest, &parameters, &conditions)

		if err != nil {
			return nil, err
		}
		jobs = append(jobs, model.Job{
			UUID:           uuid,
			DisplayName:    displayName,
			Name:           name,
			Namespace:      namespace,
			Description:    description,
			Enabled:        enabled,
			Conditions:     conditions,
			MaxConcurrency: maxConcurrency,
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
				PipelineId: pipelineId,
				Parameters: parameters,
			},
			CreatedAtInSec: createdAtInSec,
			UpdatedAtInSec: updatedAtInSec,
		})
	}
	return jobs, nil
}

func (s *JobStore) DeleteJob(id string) error {
	_, err := s.db.Exec(`DELETE FROM jobs WHERE UUID=?`, id)
	if err != nil {
		return util.NewInternalServerError(err, "Failed to delete job: %v", err.Error())
	}
	return nil
}

func (s *JobStore) CreateJob(j *model.Job) (*model.Job, error) {
	sql, args, err := sq.
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
			"Parameters":                     j.Parameters,
			"CreatedAtInSec":                 j.CreatedAtInSec,
			"UpdatedAtInSec":                 j.UpdatedAtInSec,
			"PipelineId":                     j.PipelineId,
			// TODO(yangpa) store actual value instead before v1beta1
			"PipelineSpecManifest": "",
			"WorkflowSpecManifest": "",
		}).ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create query to add job to job table: %v",
			err.Error())
	}
	_, err = s.db.Exec(sql, args...)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to add job to job table: %v",
			err.Error())
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
		db:   db,
		time: time,
	}
}
