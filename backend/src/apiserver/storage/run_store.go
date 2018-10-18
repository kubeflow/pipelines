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
	"k8s.io/apimachinery/pkg/util/json"
)

type RunStoreInterface interface {
	GetRun(runId string) (*model.RunDetail, error)
	// TODO(yangpa) support filtering and remove (jobId *string) parameter.
	ListRuns(jobId *string, context *common.PaginationContext) ([]model.Run, string, error)
	CreateOrUpdateRun(workflow *util.Workflow) (err error)
}

type RunStore struct {
	db   *sql.DB
	time util.TimeInterface
}

// ListRuns list the run metadata for a job from DB
func (s *RunStore) ListRuns(jobId *string, context *common.PaginationContext) ([]model.Run, string, error) {
	queryRunTable := func(request *common.PaginationContext) ([]model.ListableDataModel, error) {
		return s.queryRunTable(jobId, request)
	}
	models, pageToken, err := listModel(context, queryRunTable)
	if err != nil {
		return nil, "", util.Wrap(err, "List runs failed.")
	}
	return s.toRunMetadatas(models), pageToken, err
}

func (s *RunStore) queryRunTable(jobId *string, context *common.PaginationContext) ([]model.ListableDataModel, error) {
	sqlBuilder := sq.Select("*").From("run_details")
	if jobId != nil {
		sqlBuilder = sqlBuilder.Where(sq.Eq{"JobID": *jobId})
	}
	sql, args, err := toPaginationQuery(sqlBuilder, context).Limit(uint64(context.PageSize)).ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create query to list jobs: %v",
			err.Error())
	}

	r, err := s.db.Query(sql, args...)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to list runs: %v", err.Error())
	}
	defer r.Close()
	runs, err := s.scanRows(r)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to list runs: %v", err.Error())
	}

	return s.toListableModels(runs), nil
}

// GetRun Get the run manifest from Workflow CRD
func (s *RunStore) GetRun(runId string) (*model.RunDetail, error) {
	sql, args, err := sq.
		Select("*").
		From("run_details").
		Where(sq.Eq{"uuid": runId}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to get run: %v", err.Error())
	}
	r, err := s.db.Query(sql, args...)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to get run: %v", err.Error())
	}
	defer r.Close()
	runs, err := s.scanRows(r)

	if err != nil || len(runs) > 1 {
		return nil, util.NewInternalServerError(err, "Failed to get run: %v", err.Error())
	}
	if len(runs) == 0 {
		return nil, util.NewResourceNotFoundError("Run", fmt.Sprint(runId))
	}
	return &runs[0], nil
}

func (s *RunStore) scanRows(rows *sql.Rows) ([]model.RunDetail, error) {
	var runs []model.RunDetail
	for rows.Next() {
		var uuid, name, namespace, jobID, conditions, workflow string
		var CreatedAtInSec, ScheduledAtInSec int64
		err := rows.Scan(&uuid, &name, &namespace, &jobID, &CreatedAtInSec, &ScheduledAtInSec, &conditions, &workflow)
		if err != nil {
			return runs, nil
		}
		runs = append(runs, model.RunDetail{Run: model.Run{
			UUID:             uuid,
			Name:             name,
			Namespace:        namespace,
			JobID:            jobID,
			CreatedAtInSec:   CreatedAtInSec,
			ScheduledAtInSec: ScheduledAtInSec,
			Conditions:       conditions},
			Workflow: workflow})
	}
	return runs, nil
}

func (s *RunStore) createRun(
	jobID string,
	name string,
	namespace string,
	workflowUID string,
	createdAtInSec int64,
	scheduledAtInSec int64,
	condition string,
	workflow string) (err error) {
	sql, args, err := sq.
		Insert("run_details").
		SetMap(sq.Eq{
			"UUID":             workflowUID,
			"Name":             name,
			"Namespace":        namespace,
			"JobID":            jobID,
			"CreatedAtInSec":   createdAtInSec,
			"ScheduledAtInSec": scheduledAtInSec,
			"Conditions":       condition,
			"Workflow":         workflow}).ToSql()
	if err != nil {
		return util.NewInternalServerError(err, "Failed to create query while creating run using workflow: %v, %s",
			err, workflow)
	}
	_, err = s.db.Exec(sql, args...)
	if err != nil {
		return util.NewInternalServerError(err, "Error while creating run for workflow: '%v/%v",
			namespace, name)
	}

	return nil
}

func (s *RunStore) CreateOrUpdateRun(workflow *util.Workflow) (err error) {
	ownerUID := workflow.ScheduledWorkflowUUIDAsStringOrEmpty()
	marshalledWorkflow, err := json.Marshal(workflow.Workflow)
	if err != nil {
		return util.NewInternalServerError(err, "Unable to marshal a workflow: %+v", workflow.Workflow)
	}

	scheduledAtInSec := workflow.ScheduledAtInSecOr0()

	condition := workflow.Condition()

	// Attempting to create the run in the DB.

	createError := s.createRun(
		ownerUID,
		workflow.Name,
		workflow.Namespace,
		string(workflow.UID),
		workflow.CreationTimestamp.Unix(),
		scheduledAtInSec,
		condition,
		string(marshalledWorkflow))

	if createError == nil {
		return nil
	}

	// If creating the run did not work, attempting to update the run in the DB.
	sql, args, err := sq.
		Update("run_details").
		SetMap(sq.Eq{
			"Name":             workflow.Name,
			"Namespace":        workflow.Namespace,
			"JobID":            ownerUID,
			"CreatedAtInSec":   workflow.CreationTimestamp.Unix(),
			"ScheduledAtInSec": scheduledAtInSec,
			"Conditions":       condition,
			"Workflow":         string(marshalledWorkflow)}).
		Where(sq.Eq{"UUID": string(workflow.UID)}).
		ToSql()
	if err != nil {
		return util.NewInternalServerError(err,
			"Failed to create query while creating or updating run for workflow: '%v/%v'. Create error: '%v'. Update error: '%v'",
			workflow.Namespace, workflow.Name, createError.Error(), err.Error())
	}
	_, err = s.db.Exec(sql, args...)
	if err != nil {
		return util.NewInternalServerError(err,
			"Error while creating or updating run for workflow: '%v/%v'. Create error: '%v'. Update error: '%v'",
			workflow.Namespace, workflow.Name, createError.Error(), err.Error())
	}

	return nil
}

func (s *RunStore) toListableModels(runs []model.RunDetail) []model.ListableDataModel {
	models := make([]model.ListableDataModel, len(runs))
	for i := range models {
		models[i] = runs[i].Run
	}
	return models
}

func (s *RunStore) toRunMetadatas(models []model.ListableDataModel) []model.Run {
	runMetadatas := make([]model.Run, len(models))
	for i := range models {
		runMetadatas[i] = models[i].(model.Run)
	}
	return runMetadatas
}

// factory function for run store
func NewRunStore(db *sql.DB, time util.TimeInterface) *RunStore {
	return &RunStore{db: db, time: time}
}
