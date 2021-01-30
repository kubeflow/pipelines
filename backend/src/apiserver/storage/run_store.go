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

	"github.com/pkg/errors"

	sq "github.com/Masterminds/squirrel"
	workflowapi "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/golang/glog"

	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"k8s.io/apimachinery/pkg/util/json"
)

var runColumns = []string{"UUID", "ExperimentUUID", "DisplayName", "Name", "StorageState", "Namespace", "ServiceAccount", "Description",
	"CreatedAtInSec", "ScheduledAtInSec", "FinishedAtInSec", "Conditions", "PipelineId", "PipelineName", "PipelineSpecManifest",
	"WorkflowSpecManifest", "Parameters", "pipelineRuntimeManifest", "WorkflowRuntimeManifest",
}

type RunStoreInterface interface {
	GetRun(runId string) (*model.RunDetail, error)

	ListRuns(filterContext *common.FilterContext, opts *list.Options) ([]*model.Run, int, string, error)

	// Create a run entry in the database
	CreateRun(run *model.RunDetail) (*model.RunDetail, error)

	// Update run table. Only condition and runtime manifest is allowed to be updated.
	UpdateRun(id string, condition string, finishedAtInSec int64, workflowRuntimeManifest string) (err error)

	// Archive a run
	ArchiveRun(id string) error

	// Unarchive a run
	UnarchiveRun(id string) error

	// Delete a run entry from the database
	DeleteRun(id string) error

	// Update the run table or create one if the run doesn't exist
	CreateOrUpdateRun(run *model.RunDetail) error

	// Store a new metric entry to run_metrics table.
	ReportMetric(metric *model.RunMetric) (err error)

	// Terminate a run
	TerminateRun(runId string) error
}

type RunStore struct {
	db                     *DB
	resourceReferenceStore *ResourceReferenceStore
	time                   util.TimeInterface
}

// Runs two SQL queries in a transaction to return a list of matching runs, as well as their
// total_size. The total_size does not reflect the page size, but it does reflect the number of runs
// matching the supplied filters and resource references.
func (s *RunStore) ListRuns(
	filterContext *common.FilterContext, opts *list.Options) ([]*model.Run, int, string, error) {
	errorF := func(err error) ([]*model.Run, int, string, error) {
		return nil, 0, "", util.NewInternalServerError(err, "Failed to list runs: %v", err)
	}

	rowsSql, rowsArgs, err := s.buildSelectRunsQuery(false, opts, filterContext)
	if err != nil {
		return errorF(err)
	}

	sizeSql, sizeArgs, err := s.buildSelectRunsQuery(true, opts, filterContext)
	if err != nil {
		return errorF(err)
	}

	// Use a transaction to make sure we're returning the total_size of the same rows queried
	tx, err := s.db.Begin()
	if err != nil {
		glog.Error("Failed to start transaction to list runs")
		return errorF(err)
	}

	rows, err := tx.Query(rowsSql, rowsArgs...)
	if err != nil {
		return errorF(err)
	}
	runDetails, err := s.scanRowsToRunDetails(rows)
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
		glog.Error("Failed to commit transaction to list runs")
		return errorF(err)
	}

	var runs []*model.Run
	for _, rd := range runDetails {
		r := rd.Run
		runs = append(runs, &r)
	}

	if len(runs) <= opts.PageSize {
		return runs, total_size, "", nil
	}

	npt, err := opts.NextPageToken(runs[opts.PageSize])
	return runs[:opts.PageSize], total_size, npt, err
}

func (s *RunStore) buildSelectRunsQuery(selectCount bool, opts *list.Options,
	filterContext *common.FilterContext) (string, []interface{}, error) {

	var filteredSelectBuilder sq.SelectBuilder
	var err error

	refKey := filterContext.ReferenceKey
	if refKey != nil && refKey.Type == common.Experiment {
		// for performance reasons need to special treat experiment ID filter on runs
		// currently only the run table have experiment UUID column
		filteredSelectBuilder, err = list.FilterOnExperiment("run_details", runColumns,
			selectCount, refKey.ID)
	} else if refKey != nil && refKey.Type == common.Namespace {
		filteredSelectBuilder, err = list.FilterOnNamespace("run_details", runColumns,
			selectCount, refKey.ID)
	} else {
		filteredSelectBuilder, err = list.FilterOnResourceReference("run_details", runColumns,
			common.Run, selectCount, filterContext)
	}
	if err != nil {
		return "", nil, util.NewInternalServerError(err, "Failed to list runs: %v", err)
	}

	sqlBuilder := opts.AddFilterToSelect(filteredSelectBuilder)

	// If we're not just counting, then also add select columns and perform a left join
	// to get resource reference information. Also add pagination.
	if !selectCount {
		sqlBuilder = s.AddSortByRunMetricToSelect(sqlBuilder, opts)
		sqlBuilder = opts.AddPaginationToSelect(sqlBuilder)
		sqlBuilder = s.addMetricsAndResourceReferences(sqlBuilder, opts)
		sqlBuilder = opts.AddSortingToSelect(sqlBuilder)
	}
	sql, args, err := sqlBuilder.ToSql()
	if err != nil {
		return "", nil, util.NewInternalServerError(err, "Failed to list runs: %v", err)
	}
	return sql, args, err
}

// GetRun Get the run manifest from Workflow CRD
func (s *RunStore) GetRun(runId string) (*model.RunDetail, error) {
	sql, args, err := s.addMetricsAndResourceReferences(
		sq.Select(runColumns...).
			From("run_details").
			Where(sq.Eq{"UUID": runId}).
			Limit(1), nil).
		ToSql()

	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to get run: %v", err.Error())
	}
	r, err := s.db.Query(sql, args...)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to get run: %v", err.Error())
	}
	defer r.Close()
	runs, err := s.scanRowsToRunDetails(r)

	if err != nil || len(runs) > 1 {
		return nil, util.NewInternalServerError(err, "Failed to get run: %v", err.Error())
	}
	if len(runs) == 0 {
		return nil, util.NewResourceNotFoundError("Run", fmt.Sprint(runId))
	}
	if runs[0].WorkflowRuntimeManifest == "" {
		// This can only happen when workflow reporting is failed.
		return nil, util.NewResourceNotFoundError("Failed to get run: %s", runId)
	}
	return runs[0], nil
}

// Apply func f to every string in a given string slice.
func Map(vs []string, f func(string) string) []string {
	vsm := make([]string, len(vs))
	for i, v := range vs {
		vsm[i] = f(v)
	}
	return vsm
}

func (s *RunStore) addMetricsAndResourceReferences(filteredSelectBuilder sq.SelectBuilder, opts *list.Options) sq.SelectBuilder {
	var r model.Run
	resourceRefConcatQuery := s.db.Concat([]string{`"["`, s.db.GroupConcat("rr.Payload", ","), `"]"`}, "")
	columnsAfterJoiningResourceReferences := append(
		Map(runColumns, func(column string) string { return "rd." + column }), // Add prefix "rd." to runColumns
		resourceRefConcatQuery+" AS refs")
	if opts != nil && !r.IsRegularField(opts.SortByFieldName) {
		columnsAfterJoiningResourceReferences = append(columnsAfterJoiningResourceReferences, "rd."+opts.SortByFieldName)
	}
	subQ := sq.
		Select(columnsAfterJoiningResourceReferences...).
		FromSelect(filteredSelectBuilder, "rd").
		LeftJoin("resource_references AS rr ON rr.ResourceType='Run' AND rd.UUID=rr.ResourceUUID").
		GroupBy("rd.UUID")

	// TODO(jingzhang36): address the case where some runs don't have the metric used in order by.
	metricConcatQuery := s.db.Concat([]string{`"["`, s.db.GroupConcat("rm.Payload", ","), `"]"`}, "")
	columnsAfterJoiningRunMetrics := append(
		Map(runColumns, func(column string) string { return "subq." + column }), // Add prefix "subq." to runColumns
		"subq.refs",
		metricConcatQuery+" AS metrics")
	return sq.
		Select(columnsAfterJoiningRunMetrics...).
		FromSelect(subQ, "subq").
		LeftJoin("run_metrics AS rm ON subq.UUID=rm.RunUUID").
		GroupBy("subq.UUID")
}

func (s *RunStore) scanRowsToRunDetails(rows *sql.Rows) ([]*model.RunDetail, error) {
	var runs []*model.RunDetail
	for rows.Next() {
		var uuid, experimentUUID, displayName, name, storageState, namespace, serviceAccount, description, pipelineId,
			pipelineName, pipelineSpecManifest, workflowSpecManifest, parameters, conditions, pipelineRuntimeManifest,
			workflowRuntimeManifest string
		var createdAtInSec, scheduledAtInSec, finishedAtInSec int64
		var metricsInString, resourceReferencesInString sql.NullString
		err := rows.Scan(
			&uuid,
			&experimentUUID,
			&displayName,
			&name,
			&storageState,
			&namespace,
			&serviceAccount,
			&description,
			&createdAtInSec,
			&scheduledAtInSec,
			&finishedAtInSec,
			&conditions,
			&pipelineId,
			&pipelineName,
			&pipelineSpecManifest,
			&workflowSpecManifest,
			&parameters,
			&pipelineRuntimeManifest,
			&workflowRuntimeManifest,
			&resourceReferencesInString,
			&metricsInString,
		)
		if err != nil {
			glog.Errorf("Failed to scan row: %v", err)
			return runs, nil
		}
		metrics, err := parseMetrics(metricsInString)
		if err != nil {
			glog.Errorf("Failed to parse metrics (%v) from DB: %v", metricsInString, err)
			// Skip the error to allow user to get runs even when metrics data
			// are invalid.
			metrics = []*model.RunMetric{}
		}
		resourceReferences, err := parseResourceReferences(resourceReferencesInString)
		if err != nil {
			// throw internal exception if failed to parse the resource reference.
			return nil, util.NewInternalServerError(err, "Failed to parse resource reference.")
		}
		runs = append(runs, &model.RunDetail{Run: model.Run{
			UUID:               uuid,
			ExperimentUUID:     experimentUUID,
			DisplayName:        displayName,
			Name:               name,
			StorageState:       storageState,
			Namespace:          namespace,
			ServiceAccount:     serviceAccount,
			Description:        description,
			CreatedAtInSec:     createdAtInSec,
			ScheduledAtInSec:   scheduledAtInSec,
			FinishedAtInSec:    finishedAtInSec,
			Conditions:         conditions,
			Metrics:            metrics,
			ResourceReferences: resourceReferences,
			PipelineSpec: model.PipelineSpec{
				PipelineId:           pipelineId,
				PipelineName:         pipelineName,
				PipelineSpecManifest: pipelineRuntimeManifest,
				WorkflowSpecManifest: workflowSpecManifest,
				Parameters:           parameters,
			},
		},
			PipelineRuntime: model.PipelineRuntime{
				PipelineRuntimeManifest: pipelineRuntimeManifest,
				WorkflowRuntimeManifest: workflowRuntimeManifest}})
	}
	return runs, nil
}

func parseMetrics(metricsInString sql.NullString) ([]*model.RunMetric, error) {
	if !metricsInString.Valid {
		return nil, nil
	}
	var metrics []*model.RunMetric
	if err := json.Unmarshal([]byte(metricsInString.String), &metrics); err != nil {
		return nil, fmt.Errorf("failed unmarshal metrics '%s'. error: %v", metricsInString.String, err)
	}
	return metrics, nil
}

func parseResourceReferences(resourceRefString sql.NullString) ([]*model.ResourceReference, error) {
	if !resourceRefString.Valid {
		return nil, nil
	}
	var refs []*model.ResourceReference
	if err := json.Unmarshal([]byte(resourceRefString.String), &refs); err != nil {
		return nil, fmt.Errorf("failed unmarshal resource references '%s'. error: %v", resourceRefString.String, err)
	}
	return refs, nil
}

func (s *RunStore) CreateRun(r *model.RunDetail) (*model.RunDetail, error) {
	if r.StorageState == "" {
		r.StorageState = api.Run_STORAGESTATE_AVAILABLE.String()
	} else if r.StorageState != api.Run_STORAGESTATE_AVAILABLE.String() &&
		r.StorageState != api.Run_STORAGESTATE_ARCHIVED.String() {
		return nil, util.NewInvalidInputError("Invalid value for StorageState field: %q.", r.StorageState)
	}

	runSql, runArgs, err := sq.
		Insert("run_details").
		SetMap(sq.Eq{
			"UUID":                    r.UUID,
			"ExperimentUUID":          r.ExperimentUUID,
			"DisplayName":             r.DisplayName,
			"Name":                    r.Name,
			"StorageState":            r.StorageState,
			"Namespace":               r.Namespace,
			"ServiceAccount":          r.ServiceAccount,
			"Description":             r.Description,
			"CreatedAtInSec":          r.CreatedAtInSec,
			"ScheduledAtInSec":        r.ScheduledAtInSec,
			"FinishedAtInSec":         r.FinishedAtInSec,
			"Conditions":              r.Conditions,
			"WorkflowRuntimeManifest": r.WorkflowRuntimeManifest,
			"PipelineRuntimeManifest": r.PipelineRuntimeManifest,
			"PipelineId":              r.PipelineId,
			"PipelineName":            r.PipelineName,
			"PipelineSpecManifest":    r.PipelineSpecManifest,
			"WorkflowSpecManifest":    r.WorkflowSpecManifest,
			"Parameters":              r.Parameters,
		}).ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create query to store run to run table: '%v/%v",
			r.Namespace, r.Name)
	}

	// Use a transaction to make sure both run and its resource references are stored.
	tx, err := s.db.Begin()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create a new transaction to create run.")
	}
	_, err = tx.Exec(runSql, runArgs...)
	if err != nil {
		tx.Rollback()
		return nil, util.NewInternalServerError(err, "Failed to store run %v to table", r.Name)
	}

	err = s.resourceReferenceStore.CreateResourceReferences(tx, r.ResourceReferences)
	if err != nil {
		tx.Rollback()
		return nil, util.NewInternalServerError(err, "Failed to store resource references to table for run %v ", r.Name)
	}
	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		return nil, util.NewInternalServerError(err, "Failed to store run %v and its resource references to table", r.Name)
	}
	return r, nil
}

func (s *RunStore) UpdateRun(runID string, condition string, finishedAtInSec int64, workflowRuntimeManifest string) (err error) {
	tx, err := s.db.DB.Begin()
	if err != nil {
		return util.NewInternalServerError(err, "transaction creation failed")
	}

	sql, args, err := sq.
		Update("run_details").
		SetMap(sq.Eq{
			"Conditions":              condition,
			"FinishedAtInSec":         finishedAtInSec,
			"WorkflowRuntimeManifest": workflowRuntimeManifest}).
		Where(sq.Eq{"UUID": runID}).
		ToSql()
	if err != nil {
		tx.Rollback()
		return util.NewInternalServerError(err,
			"Failed to create query to update run %s. error: '%v'", runID, err.Error())
	}
	result, err := tx.Exec(sql, args...)
	if err != nil {
		tx.Rollback()
		return util.NewInternalServerError(err,
			"Failed to update run %s. error: '%v'", runID, err.Error())
	}
	r, err := result.RowsAffected()
	if err != nil {
		tx.Rollback()
		return util.NewInternalServerError(err,
			"Failed to update run %s. error: '%v'", runID, err.Error())
	}
	if r > 1 {
		tx.Rollback()
		return util.NewInternalServerError(errors.New("Failed to update run"), "Failed to update run %s. More than 1 rows affected", runID)
	} else if r == 0 {
		tx.Rollback()
		return util.NewInternalServerError(errors.New("Failed to update run"), "Failed to update run %s. Row not found", runID)
	}

	if err := tx.Commit(); err != nil {
		return util.NewInternalServerError(err, "failed to commit transaction")
	}
	return nil
}

func (s *RunStore) CreateOrUpdateRun(runDetail *model.RunDetail) error {
	_, createError := s.CreateRun(runDetail)
	if createError == nil {
		return nil
	}

	updateError := s.UpdateRun(runDetail.UUID, runDetail.Conditions, runDetail.FinishedAtInSec, runDetail.WorkflowRuntimeManifest)
	if updateError != nil {
		return util.Wrap(updateError, fmt.Sprintf(
			"Error while creating or updating run for workflow: '%v/%v'. Create error: '%v'. Update error: '%v'",
			runDetail.Namespace, runDetail.Name, createError.Error(), updateError.Error()))
	}
	return nil
}

func (s *RunStore) ArchiveRun(runId string) error {
	sql, args, err := sq.
		Update("run_details").
		SetMap(sq.Eq{
			"StorageState": api.Run_STORAGESTATE_ARCHIVED.String(),
		}).
		Where(sq.Eq{"UUID": runId}).
		ToSql()

	if err != nil {
		return util.NewInternalServerError(err,
			"Failed to create query to archive run %s. error: '%v'", runId, err.Error())
	}

	_, err = s.db.Exec(sql, args...)
	if err != nil {
		return util.NewInternalServerError(err,
			"Failed to archive run %s. error: '%v'", runId, err.Error())
	}

	return nil
}

func (s *RunStore) UnarchiveRun(runId string) error {
	sql, args, err := sq.
		Update("run_details").
		SetMap(sq.Eq{
			"StorageState": api.Run_STORAGESTATE_AVAILABLE.String(),
		}).
		Where(sq.Eq{"UUID": runId}).
		ToSql()

	if err != nil {
		return util.NewInternalServerError(err,
			"Failed to create query to unarchive run %s. error: '%v'", runId, err.Error())
	}

	_, err = s.db.Exec(sql, args...)
	if err != nil {
		return util.NewInternalServerError(err,
			"Failed to unarchive run %s. error: '%v'", runId, err.Error())
	}

	return nil
}

func (s *RunStore) DeleteRun(id string) error {
	runSql, runArgs, err := sq.Delete("run_details").Where(sq.Eq{"UUID": id}).ToSql()
	if err != nil {
		return util.NewInternalServerError(err,
			"Failed to create query to delete run: %s", id)
	}
	// Use a transaction to make sure both run and its resource references are stored.
	tx, err := s.db.Begin()
	if err != nil {
		return util.NewInternalServerError(err, "Failed to create a new transaction to delete run.")
	}
	_, err = tx.Exec(runSql, runArgs...)
	if err != nil {
		tx.Rollback()
		return util.NewInternalServerError(err, "Failed to delete run %s from table", id)
	}
	err = s.resourceReferenceStore.DeleteResourceReferences(tx, id, common.Run)
	if err != nil {
		tx.Rollback()
		return util.NewInternalServerError(err, "Failed to delete resource references from table for run %v ", id)
	}
	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		return util.NewInternalServerError(err, "Failed to delete run %v and its resource references from table", id)
	}
	return nil
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

// NewRunStore creates a new RunStore.
func NewRunStore(db *DB, time util.TimeInterface) *RunStore {
	return &RunStore{
		db:                     db,
		resourceReferenceStore: NewResourceReferenceStore(db),
		time:                   time,
	}
}

func (s *RunStore) TerminateRun(runId string) error {
	result, err := s.db.Exec(`
		UPDATE run_details
		SET Conditions = ?
		WHERE UUID = ? AND (Conditions = ? OR Conditions = ? OR Conditions = ?)`,
		model.RunTerminatingConditions, runId, string(workflowapi.NodeRunning), string(workflowapi.NodePending), "")

	if err != nil {
		return util.NewInternalServerError(err,
			"Failed to terminate run %s. error: '%v'", runId, err.Error())
	}

	if r, _ := result.RowsAffected(); r != 1 {
		return util.NewInvalidInputError("Failed to terminate run %s. Row not found.", runId)
	}

	return nil
}

// Add a metric as a new field to the select clause by join the passed-in SQL query with run_metrics table.
// With the metric as a field in the select clause enable sorting on this metric afterwards.
// TODO(jingzhang36): example of resulting SQL query and explanation for it.
func (s *RunStore) AddSortByRunMetricToSelect(sqlBuilder sq.SelectBuilder, opts *list.Options) sq.SelectBuilder {
	var r model.Run
	if r.IsRegularField(opts.SortByFieldName) {
		return sqlBuilder
	}
	// TODO(jingzhang36): address the case where runs doesn't have the specified metric.
	return sq.
		Select("selected_runs.*, run_metrics.numbervalue as "+opts.SortByFieldName).
		FromSelect(sqlBuilder, "selected_runs").
		LeftJoin("run_metrics ON selected_runs.uuid=run_metrics.runuuid AND run_metrics.name='" + opts.SortByFieldName + "'")
}
