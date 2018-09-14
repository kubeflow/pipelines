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
	"bytes"
	"fmt"

	"database/sql"

	"github.com/VividCortex/mysqlerr"
	"github.com/go-sql-driver/mysql"
	"github.com/googleprivate/ml/backend/src/apiserver/model"
	"github.com/googleprivate/ml/backend/src/common/util"
)

const (
	insertJobQuery = `INSERT INTO job_details(
					UUID,DisplayName,Name,Namespace,Description,
          PipelineId,Enabled,Conditions,MaxConcurrency,
          CronScheduleStartTimeInSec,CronScheduleEndTimeInSec,Schedule,
          PeriodicScheduleStartTimeInSec,PeriodicScheduleEndTimeInSec,IntervalSecond,
          Parameters,CreatedAtInSec,UpdatedAtInSec,ScheduledWorkflow) 
          VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`
)

type JobStoreInterface interface {
	ListJobs(pageToken string, pageSize int, sortByFieldName string, isDesc bool) ([]model.Job, string, error)
	GetJob(id string) (*model.Job, error)
	CreateJob(*model.Job) (*model.Job, error)
	DeleteJob(id string) error
	EnableJob(id string, enabled bool) error
	UpdateJob(swf *util.ScheduledWorkflow) error
}

type JobStore struct {
	db   *sql.DB
	time util.TimeInterface
}

func (s *JobStore) ListJobs(pageToken string, pageSize int, sortByFieldName string, isDesc bool) ([]model.Job, string, error) {
	context, err := NewPaginationContext(pageToken, pageSize, sortByFieldName, model.GetJobTablePrimaryKeyColumn(), isDesc)
	if err != nil {
		return nil, "", err
	}
	models, pageToken, err := listModel(context, s.queryJobTable)
	if err != nil {
		return nil, "", util.Wrap(err, "List jobs failed.")
	}
	return s.toJobMetadatas(models), pageToken, err
}

func (s *JobStore) queryJobTable(context *PaginationContext) ([]model.ListableDataModel, error) {
	var query bytes.Buffer
	query.WriteString(fmt.Sprintf("SELECT * FROM job_details "))
	toPaginationQuery("WHERE", &query, context)
	query.WriteString(fmt.Sprintf(" LIMIT %v", context.pageSize))

	rows, err := s.db.Query(query.String())
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
	row, err := s.db.Query(`SELECT * FROM job_details WHERE uuid=? LIMIT 1`, id)
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
	return &jobs[0].Job, nil
}

func (s *JobStore) scanRows(r *sql.Rows) ([]model.JobDetail, error) {
	var jobs []model.JobDetail
	for r.Next() {
		var uuid, displayName, name, namespace, packageId, conditions,
			scheduledWorkflow, description, parameters string
		var cronScheduleStartTimeInSec, cronScheduleEndTimeInSec,
			periodicScheduleStartTimeInSec, periodicScheduleEndTimeInSec, intervalSecond sql.NullInt64
		var cron sql.NullString
		var enabled bool
		var createdAtInSec, updatedAtInSec, maxConcurrency int64
		err := r.Scan(
			&uuid, &displayName, &name, &namespace, &description,
			&packageId, &enabled, &conditions, &maxConcurrency,
			&cronScheduleStartTimeInSec, &cronScheduleEndTimeInSec, &cron,
			&periodicScheduleStartTimeInSec, &periodicScheduleEndTimeInSec, &intervalSecond,
			&parameters, &createdAtInSec, &updatedAtInSec, &scheduledWorkflow)

		if err != nil {
			return nil, err
		}
		jobs = append(jobs, model.JobDetail{Job: model.Job{
			UUID:           uuid,
			DisplayName:    displayName,
			Name:           name,
			Namespace:      namespace,
			Description:    description,
			PipelineId:     packageId,
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
			Parameters:     parameters,
			CreatedAtInSec: createdAtInSec,
			UpdatedAtInSec: updatedAtInSec,
		}, ScheduledWorkflow: scheduledWorkflow})
	}
	return jobs, nil
}

func (s *JobStore) DeleteJob(id string) error {
	_, err := s.db.Exec(`DELETE FROM job_details WHERE UUID=?`, id)
	if err != nil {
		return util.NewInternalServerError(err, "Failed to delete job: %v", err.Error())
	}
	return nil
}

func (s *JobStore) CreateJob(p *model.Job) (*model.Job, error) {
	var newJob model.JobDetail
	newJob.Job = *p
	now := s.time.Now().Unix()
	newJob.CreatedAtInSec = now
	newJob.UpdatedAtInSec = now
	stmt, err := s.db.Prepare(insertJobQuery)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to add job to job table: %v",
			err.Error())
	}
	defer stmt.Close()
	if _, err := stmt.Exec(
		newJob.UUID, newJob.DisplayName, newJob.Name, newJob.Namespace, newJob.Description,
		newJob.PipelineId, newJob.Enabled, newJob.Conditions, newJob.MaxConcurrency,
		PointerToNullInt64(newJob.CronScheduleStartTimeInSec), PointerToNullInt64(newJob.CronScheduleEndTimeInSec), PointerToNullString(newJob.Cron),
		PointerToNullInt64(newJob.PeriodicScheduleStartTimeInSec), PointerToNullInt64(newJob.PeriodicScheduleEndTimeInSec), PointerToNullInt64(newJob.IntervalSecond),
		newJob.Parameters, newJob.CreatedAtInSec, newJob.UpdatedAtInSec, newJob.ScheduledWorkflow); err != nil {
		if sqlError, ok := err.(*mysql.MySQLError); ok && sqlError.Number == mysqlerr.ER_DUP_ENTRY {
			return nil, util.NewInvalidInputError(
				"Failed to create a new job. The name %v already exist. Please specify a new name.", p.DisplayName)
		}
		return nil, util.NewInternalServerError(err, "Failed to add job to job table: %v",
			err.Error())
	}
	return &newJob.Job, nil
}

func (s *JobStore) EnableJob(id string, enabled bool) error {
	now := s.time.Now().Unix()
	stmt, err := s.db.Prepare(`UPDATE job_details SET Enabled=?, UpdatedAtInSec=? WHERE UUID=? and Enabled=?`)
	if err != nil {
		return util.NewInternalServerError(err, "Error when enabling job %v to %v", id, enabled)
	}
	defer stmt.Close()
	if _, err := stmt.Exec(enabled, now, id, !enabled); err != nil {
		return util.NewInternalServerError(err, "Error when enabling job %v to %v", id, enabled)
	}
	return nil
}

func (s *JobStore) UpdateJob(swf *util.ScheduledWorkflow) error {
	now := s.time.Now().Unix()

	if swf.Name == "" {
		return util.NewInvalidInputError("The resource must have a name: %+v", swf.ScheduledWorkflow)
	}
	if swf.Namespace == "" {
		return util.NewInvalidInputError("The resource must have a namespace: %+v", swf.ScheduledWorkflow)
	}

	if swf.UID == "" {
		return util.NewInvalidInputError("The resource must have a UID: %+v", swf.UID)
	}

	parameters, err := swf.ParametersAsString()
	if err != nil {
		return err
	}
	stmt, err := s.db.Prepare(`UPDATE job_details SET 
		Name = ?,
		Namespace = ?,
		Enabled = ?,
		Conditions = ?,
		MaxConcurrency = ?,
		Parameters = ?,
		UpdatedAtInSec = ?,
		CronScheduleStartTimeInSec = ?,
		CronScheduleEndTimeInSec = ?,
		Schedule = ?,
		PeriodicScheduleStartTimeInSec = ?,
		PeriodicScheduleEndTimeInSec = ?,
		IntervalSecond = ? 
		WHERE UUID = ?`)
	if err != nil {
		return util.NewInternalServerError(err,
			"Error while updating job with scheduled workflow: %v: %+v",
			err, swf.ScheduledWorkflow)
	}
	defer stmt.Close()
	r, err := stmt.Exec(
		swf.Name,
		swf.Namespace,
		swf.Spec.Enabled,
		swf.ConditionSummary(),
		swf.MaxConcurrencyOr0(),
		parameters,
		now,
		PointerToNullInt64(swf.CronScheduleStartTimeInSecOrNull()),
		PointerToNullInt64(swf.CronScheduleEndTimeInSecOrNull()),
		swf.CronOrEmpty(),
		PointerToNullInt64(swf.PeriodicScheduleStartTimeInSecOrNull()),
		PointerToNullInt64(swf.PeriodicScheduleEndTimeInSecOrNull()),
		swf.IntervalSecondOr0(),
		string(swf.UID))

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

func (s *JobStore) toListableModels(jobs []model.JobDetail) []model.ListableDataModel {
	models := make([]model.ListableDataModel, len(jobs))
	for i := range models {
		models[i] = jobs[i].Job
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
func NewJobStore(db *sql.DB, time util.TimeInterface) *JobStore {
	return &JobStore{
		db:   db,
		time: time,
	}
}
