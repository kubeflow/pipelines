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
	"github.com/googleprivate/ml/backend/src/apiserver/common"
	"github.com/googleprivate/ml/backend/src/apiserver/model"
	"github.com/googleprivate/ml/backend/src/common/util"
	"k8s.io/apimachinery/pkg/util/json"
)

type RunStoreInterface interface {
	GetRun(runId string) (*model.RunDetail, error)
	// TODO(yangpa) support filtering and remove (jobId *string) parameter.
	ListRuns(jobId *string, context *common.PaginationContext) ([]model.Run, string, error)

	// Create a run entry in the database
	CreateRun(run *model.RunDetail) (*model.RunDetail, error)

	// Method to store a reported argo workflow to DB.
	// If it's a one-time run, we assume the entry is already exist in DB, and only do update.
	// If it's a run from a recurrent job, the run might not exist in DB before. Either update or create one if not exist.
	ReportRun(workflow *util.Workflow) (err error)

	// Store a new metric entry to run_metrics table.
	ReportMetric(metric *model.RunMetric) (err error)
}

type RunStore struct {
	db   *DB
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
	sqlBuilder := s.selectRunDetailsWithMetrics()
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
	sql, args, err := s.selectRunDetailsWithMetrics().
		Where(sq.Eq{"UUID": runId}).
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

func (s *RunStore) selectRunDetailsWithMetrics() sq.SelectBuilder {
	sd := s.db.GetDialect()
	metricConcatQuery := sd.Concat([]string{`"["`, sd.GroupConcat("m.Payload", ","), `"]"`}, "")
	return sq.
		Select("rd.*", metricConcatQuery+" AS metrics").
		From("run_details AS rd").
		LeftJoin("run_metrics AS m ON rd.UUID=m.RunUUID").
		GroupBy("rd.UUID")
}

func (s *RunStore) scanRows(rows *sql.Rows) ([]model.RunDetail, error) {
	var runs []model.RunDetail
	for rows.Next() {
		var uuid, displayName, name, namespace, description, pipelineId, pipelineSpecManifest, workflowSpecManifest,
			parameters, jobID, conditions, pipelineRuntimeManifest, workflowRuntimeManifest string
		var createdAtInSec, scheduledAtInSec int64
		var metricsInString sql.NullString
		err := rows.Scan(
			&uuid, &displayName, &name, &namespace, &description, &createdAtInSec, &scheduledAtInSec,
			&conditions, &pipelineId, &pipelineSpecManifest, &workflowSpecManifest, &parameters,
			&jobID, &pipelineRuntimeManifest, &workflowRuntimeManifest,
			&metricsInString)
		if err != nil {
			glog.Errorf("Failed to scan row: %v", err)
			return runs, nil
		}
		metrics, err := parseMetrics(metricsInString, uuid)
		if err != nil {
			glog.Errorf("Failed to parse metrics (%v) from DB: %v", metricsInString, err)
			// Skip the error to allow user to get runs even when metrics data
			// are invalid.
			metrics = []*model.RunMetric{}
		}
		runs = append(runs, model.RunDetail{Run: model.Run{
			UUID:             uuid,
			DisplayName:      displayName,
			Name:             name,
			Namespace:        namespace,
			Description:      description,
			CreatedAtInSec:   createdAtInSec,
			ScheduledAtInSec: scheduledAtInSec,
			Conditions:       conditions,
			Metrics:          metrics,
			PipelineSpec: model.PipelineSpec{
				PipelineId:           pipelineId,
				PipelineSpecManifest: pipelineRuntimeManifest,
				WorkflowSpecManifest: workflowSpecManifest,
				Parameters:           parameters,
			},
			JobID: jobID,
		},
			PipelineRuntime: model.PipelineRuntime{
				PipelineRuntimeManifest: pipelineRuntimeManifest,
				WorkflowRuntimeManifest: workflowRuntimeManifest}})
	}
	return runs, nil
}

func parseMetrics(metricsInString sql.NullString, runUUID string) ([]*model.RunMetric, error) {
	if !metricsInString.Valid {
		return nil, nil
	}
	var metrics []*model.RunMetric
	if err := json.Unmarshal([]byte(metricsInString.String), &metrics); err != nil {
		return nil, fmt.Errorf("failed unmarshal metrics '%s'. error: %v", metricsInString.String, err)
	}
	return metrics, nil
}

func (s *RunStore) CreateRun(r *model.RunDetail) (*model.RunDetail, error) {
	sql, args, err := sq.
		Insert("run_details").
		SetMap(sq.Eq{
			"UUID":                    r.UUID,
			"DisplayName":             r.DisplayName,
			"Name":                    r.Name,
			"Namespace":               r.Namespace,
			"Description":             r.Description,
			"JobID":                   r.JobID,
			"CreatedAtInSec":          r.CreatedAtInSec,
			"ScheduledAtInSec":        r.ScheduledAtInSec,
			"Conditions":              r.Conditions,
			"WorkflowRuntimeManifest": r.WorkflowRuntimeManifest,
			"PipelineRuntimeManifest": r.PipelineRuntimeManifest,
			"PipelineId":              r.PipelineId,
			"PipelineSpecManifest":    r.PipelineSpec.PipelineSpecManifest,
			"WorkflowSpecManifest":    r.PipelineSpec.WorkflowSpecManifest,
			"Parameters":              r.PipelineSpec.Parameters,
		}).ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Error while creating run for workflow: '%v/%v",
			r.Namespace, r.Name)
	}
	_, err = s.db.Exec(sql, args...)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Error while creating run for workflow: '%v/%v",
			r.Namespace, r.Name)
	}
	return r, nil
}

// Update run table. Only condition and runtime manifest is allowed to be updated.
func (s *RunStore) updateRun(
	workflowUID string,
	condition string,
	workflowRuntimeManifest string) (err error) {
	sql, args, err := sq.
		Update("run_details").
		SetMap(sq.Eq{
			"Conditions":              condition,
			"WorkflowRuntimeManifest": workflowRuntimeManifest}).
		Where(sq.Eq{"UUID": workflowUID}).
		ToSql()
	if err != nil {
		return util.NewInternalServerError(err,
			"Failed to create query to update workflow %s. error: '%v'", workflowUID, err.Error())
	}
	_, err = s.db.Exec(sql, args...)
	if err != nil {
		return util.NewInternalServerError(err,
			"Failed to update workflow %s. error: '%v'", workflowUID, err.Error())
	}
	return nil
}

func (s *RunStore) CreateOrUpdateRun(workflow *util.Workflow) (err error) {
	// TODO(yangpa): store scheduledworkflow id in resource reference table.
	runDetail := &model.RunDetail{
		Run: model.Run{
			UUID:             string(workflow.UID),
			Name:             workflow.Name,
			Namespace:        workflow.Namespace,
			CreatedAtInSec:   workflow.CreationTimestamp.Unix(),
			ScheduledAtInSec: workflow.ScheduledAtInSecOr0(),
			Conditions:       workflow.Condition(),
			JobID:            workflow.ScheduledWorkflowUUIDAsStringOrEmpty(),
			PipelineSpec: model.PipelineSpec{
				WorkflowSpecManifest: workflow.GetSpec().ToStringForStore(),
			},
		},
		PipelineRuntime: model.PipelineRuntime{
			WorkflowRuntimeManifest: workflow.ToStringForStore(),
		},
	}
	_, createError := s.CreateRun(runDetail)
	if createError == nil {
		return nil
	}

	updateError := s.updateRun(string(workflow.UID), workflow.Condition(), workflow.ToStringForStore())

	if updateError != nil {
		return util.Wrap(updateError, fmt.Sprintf(
			"Error while creating or updating run for workflow: '%v/%v'. Create error: '%v'. Update error: '%v'",
			workflow.Namespace, workflow.Name, createError.Error(), updateError.Error()))
	}
	return nil
}

func (s *RunStore) ReportRun(workflow *util.Workflow) (err error) {
	if workflow.ScheduledWorkflowUUIDAsStringOrEmpty() == "" {
		// If a run doesn't have owner UID, it's a one-time run created by Pipeline API server.
		// In this case the DB entry should already be created when argo workflow is created.
		return s.updateRun(string(workflow.UID), workflow.Condition(), workflow.ToStringForStore())
	}
	return s.CreateOrUpdateRun(workflow)
}

// ReportMetric inserts a new metric to run_metrics table. Conflicting metrics
// are ignored.
func (s *RunStore) ReportMetric(metric *model.RunMetric) (err error) {
	payloadBytes, err := json.Marshal(metric)
	if err != nil {
		return util.NewInternalServerError(err,
			"failed to marshal metric to json: %+v", metric)
	}
	sql, args, err := sq.
		Insert("run_metrics").
		SetMap(sq.Eq{
			"RunUUID":     metric.RunUUID,
			"NodeID":      metric.NodeID,
			"Name":        metric.Name,
			"NumberValue": metric.NumberValue,
			"Format":      metric.Format,
			"Payload":     string(payloadBytes)}).ToSql()
	if err != nil {
		return util.NewInternalServerError(err,
			"failed to create query for inserting metric: %+v", metric)
	}
	_, err = s.db.Exec(sql, args...)
	if err != nil {
		if s.db.IsDuplicateError(err) {
			return util.NewAlreadyExistError(
				"same metric has been reported before: %s/%s", metric.NodeID, metric.Name)
		}
		return util.NewInternalServerError(err, "failed to insert metric: %v", metric)
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
func NewRunStore(db *DB, time util.TimeInterface) *RunStore {
	return &RunStore{db: db, time: time}
}
