// Copyright 2018 The Kubeflow Authors
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
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/json"
)

var runColumns = []string{
	"UUID",
	"ExperimentUUID",
	"DisplayName",
	"Name",
	"StorageState",
	"Namespace",
	"ServiceAccount",
	"Description",
	"CreatedAtInSec",
	"ScheduledAtInSec",
	"FinishedAtInSec",
	"Conditions",
	"PipelineId",
	"PipelineVersionId",
	"PipelineName",
	"PipelineSpecManifest",
	"WorkflowSpecManifest",
	"Parameters",
	"RuntimeParameters",
	"PipelineRoot",
	"PipelineRuntimeManifest",
	"WorkflowRuntimeManifest",
	"JobUUID",
	"State",
	"StateHistory",
	"PipelineContextId",
	"PipelineRunContextId",
}

var runMetricsColumns = []string{
	"RunUUID",
	"NodeID",
	"Name",
	"NumberValue",
	"Format",
	"Payload",
}

type RunStoreInterface interface {
	// Creates a run entry. Does not create children tasks.
	CreateRun(run *model.Run) (*model.Run, error)

	// Fetches a run.
	GetRun(runId string) (*model.Run, error)

	// Fetches runs with specified options. Joins with children tasks.
	ListRuns(filterContext *model.FilterContext, opts *list.Options) ([]*model.Run, int, string, error)

	// Updates a run.
	// Note: only state, runtime manifest can be updated. Does not update dependent tasks.
	UpdateRun(run *model.Run) (err error)

	// Archives a run.
	ArchiveRun(runId string) error

	// Un-archives a run.
	UnarchiveRun(runId string) error

	// Deletes a run.
	DeleteRun(runId string) error

	// Creates a new metric entry.
	CreateMetric(metric *model.RunMetric) (err error)

	// Fetches metrics for a given run.
	GetMetrics(runId string) ([]*model.RunMetric, error)

	// Terminates a run.
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
	filterContext *model.FilterContext, opts *list.Options,
) ([]*model.Run, int, string, error) {
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
	if err := rows.Err(); err != nil {
		return errorF(err)
	}
	runs, err := s.scanRowsToRuns(rows)
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
	total_size, err := list.ScanRowToTotalSize(sizeRow)
	if err != nil {
		tx.Rollback()
		return errorF(err)
	}
	defer sizeRow.Close()

	err = tx.Commit()
	if err != nil {
		glog.Error("Failed to commit transaction to list runs")
		return errorF(err)
	}

	if len(runs) <= opts.PageSize {
		return runs, total_size, "", nil
	}

	npt, err := opts.NextPageToken(runs[opts.PageSize])
	return runs[:opts.PageSize], total_size, npt, err
}

func (s *RunStore) buildSelectRunsQuery(selectCount bool, opts *list.Options,
	filterContext *model.FilterContext,
) (string, []interface{}, error) {
	var filteredSelectBuilder sq.SelectBuilder
	var err error

	refKey := filterContext.ReferenceKey
	if refKey != nil && refKey.Type == model.ExperimentResourceType && (refKey.ID != "" || common.IsMultiUserMode()) {
		// for performance reasons need to special treat experiment ID filter on runs
		// currently only the run table have experiment UUID column
		filteredSelectBuilder, err = list.FilterOnExperiment("run_details", runColumns,
			selectCount, refKey.ID)
	} else if refKey != nil && refKey.Type == model.NamespaceResourceType && (refKey.ID != "" || common.IsMultiUserMode()) {
		filteredSelectBuilder, err = list.FilterOnNamespace("run_details", runColumns,
			selectCount, refKey.ID)
	} else {
		filteredSelectBuilder, err = list.FilterOnResourceReference("run_details", runColumns,
			model.RunResourceType, selectCount, filterContext)
	}
	if err != nil {
		return "", nil, util.NewInternalServerError(err, "Failed to list runs: %v", err)
	}

	sqlBuilder := opts.AddFilterToSelect(filteredSelectBuilder)

	// If we're not just counting, then also add select columns and perform a left join
	// to get resource reference information. Also add pagination.
	if !selectCount {
		sqlBuilder = s.addSortByRunMetricToSelect(sqlBuilder, opts)
		sqlBuilder = opts.AddPaginationToSelect(sqlBuilder)
		sqlBuilder = s.addMetricsResourceReferencesAndTasks(sqlBuilder, opts)
		sqlBuilder = opts.AddSortingToSelect(sqlBuilder)
	}
	sql, args, err := sqlBuilder.ToSql()
	if err != nil {
		return "", nil, util.NewInternalServerError(err, "Failed to list runs: %v", err)
	}
	return sql, args, err
}

// GetRun Get the run manifest from Workflow CRD.
func (s *RunStore) GetRun(runId string) (*model.Run, error) {
	sql, args, err := s.addMetricsResourceReferencesAndTasks(
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
	runs, err := s.scanRowsToRuns(r)

	if err != nil || len(runs) > 1 {
		return nil, util.NewInternalServerError(err, "Failed to get run: %v", err.Error())
	}
	if len(runs) == 0 {
		return nil, util.NewResourceNotFoundError("Run", fmt.Sprint(runId))
	}
	if runs[0].RunDetails.WorkflowRuntimeManifest == "" && runs[0].WorkflowSpecManifest != "" {
		// This can only happen when workflow reporting is failed.
		return nil, util.NewResourceNotFoundError("Failed to get run: %s", runId)
	}
	return runs[0], nil
}

// Applies a func f to every string in a given string slice.
func apply(f func(string) string, vs []string) []string {
	vsm := make([]string, len(vs))
	for i, v := range vs {
		vsm[i] = f(v)
	}
	return vsm
}

func (s *RunStore) addMetricsResourceReferencesAndTasks(filteredSelectBuilder sq.SelectBuilder, opts *list.Options) sq.SelectBuilder {
	var r model.Run
	resourceRefConcatQuery := s.db.Concat([]string{`"["`, s.db.GroupConcat("rr.Payload", ","), `"]"`}, "")
	columnsAfterJoiningResourceReferences := append(
		apply(func(column string) string { return "rd." + column }, runColumns), // Add prefix "rd." to runColumns
		resourceRefConcatQuery+" AS refs")
	if opts != nil && !r.IsRegularField(opts.SortByFieldName) {
		columnsAfterJoiningResourceReferences = append(columnsAfterJoiningResourceReferences, "rd."+opts.SortByFieldName)
	}
	subQ := sq.
		Select(columnsAfterJoiningResourceReferences...).
		FromSelect(filteredSelectBuilder, "rd").
		LeftJoin("resource_references AS rr ON rr.ResourceType='Run' AND rd.UUID=rr.ResourceUUID").
		GroupBy("rd.UUID")

	tasksConcatQuery := s.db.Concat([]string{`"["`, s.db.GroupConcat("tasks.Payload", ","), `"]"`}, "")
	columnsAfterJoiningTasks := append(
		apply(func(column string) string { return "rdref." + column }, runColumns),
		"rdref.refs",
		tasksConcatQuery+" AS taskDetails")
	if opts != nil && !r.IsRegularField(opts.SortByFieldName) {
		columnsAfterJoiningTasks = append(columnsAfterJoiningTasks, "rdref."+opts.SortByFieldName)
	}
	subQ = sq.
		Select(columnsAfterJoiningTasks...).
		FromSelect(subQ, "rdref").
		LeftJoin("tasks AS tasks ON rdref.UUID=tasks.RunUUID").
		GroupBy("rdref.UUID")

	// TODO(jingzhang36): address the case where some runs don't have the metric used in order by.
	metricConcatQuery := s.db.Concat([]string{`"["`, s.db.GroupConcat("rm.Payload", ","), `"]"`}, "")
	columnsAfterJoiningRunMetrics := append(
		apply(func(column string) string { return "subq." + column }, runColumns), // Add prefix "subq." to runColumns
		"subq.refs",
		"subq.taskDetails",
		metricConcatQuery+" AS metrics")
	return sq.
		Select(columnsAfterJoiningRunMetrics...).
		FromSelect(subQ, "subq").
		LeftJoin("run_metrics AS rm ON subq.UUID=rm.RunUUID").
		GroupBy("subq.UUID")
}

func (s *RunStore) scanRowsToRuns(rows *sql.Rows) ([]*model.Run, error) {
	var runs []*model.Run
	for rows.Next() {
		var uuid, experimentUUID, displayName, name, storageState, namespace, serviceAccount, conditions, description, pipelineId,
			pipelineName, pipelineSpecManifest, workflowSpecManifest, parameters, pipelineRuntimeManifest,
			workflowRuntimeManifest string
		var createdAtInSec, scheduledAtInSec, finishedAtInSec, pipelineContextId, pipelineRunContextId sql.NullInt64
		var metricsInString, resourceReferencesInString, tasksInString, runtimeParameters, pipelineRoot, jobId, state, stateHistory, pipelineVersionId sql.NullString
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
			&pipelineVersionId,
			&pipelineName,
			&pipelineSpecManifest,
			&workflowSpecManifest,
			&parameters,
			&runtimeParameters,
			&pipelineRoot,
			&pipelineRuntimeManifest,
			&workflowRuntimeManifest,
			&jobId,
			&state,
			&stateHistory,
			&pipelineContextId,
			&pipelineRunContextId,
			&resourceReferencesInString,
			&tasksInString,
			&metricsInString,
		)
		if err != nil {
			glog.Errorf("Failed to scan row into a run: %v", err)
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
			return nil, util.NewInternalServerError(err, "Failed to parse resource reference")
		}
		tasks, err := parseTaskDetails(tasksInString)
		if err != nil {
			return nil, util.NewInternalServerError(err, "Failed to parse task details")
		}
		jId := jobId.String
		pvId := pipelineVersionId.String
		if len(resourceReferences) > 0 {
			if experimentUUID == "" {
				experimentUUID = model.GetRefIdFromResourceReferences(resourceReferences, model.ExperimentResourceType)
			}
			if namespace == "" {
				namespace = model.GetRefIdFromResourceReferences(resourceReferences, model.NamespaceResourceType)
			}
			if pipelineId == "" {
				pipelineId = model.GetRefIdFromResourceReferences(resourceReferences, model.PipelineResourceType)
			}
			if pvId == "" {
				pvId = model.GetRefIdFromResourceReferences(resourceReferences, model.PipelineVersionResourceType)
			}
			if jId == "" {
				jId = model.GetRefIdFromResourceReferences(resourceReferences, model.JobResourceType)
			}
		}
		runtimeConfig := parseRuntimeConfig(runtimeParameters, pipelineRoot)
		var stateHistoryNew []*model.RuntimeStatus
		if stateHistory.Valid {
			json.Unmarshal([]byte(stateHistory.String), &stateHistoryNew)
		}
		run := &model.Run{
			UUID:           uuid,
			ExperimentId:   experimentUUID,
			DisplayName:    displayName,
			K8SName:        name,
			StorageState:   model.StorageState(storageState),
			Namespace:      namespace,
			ServiceAccount: serviceAccount,
			Description:    description,
			RecurringRunId: jId,
			RunDetails: model.RunDetails{
				CreatedAtInSec:          createdAtInSec.Int64,
				ScheduledAtInSec:        scheduledAtInSec.Int64,
				FinishedAtInSec:         finishedAtInSec.Int64,
				Conditions:              conditions,
				State:                   model.RuntimeState(state.String),
				PipelineRuntimeManifest: pipelineRuntimeManifest,
				WorkflowRuntimeManifest: workflowRuntimeManifest,
				PipelineContextId:       pipelineContextId.Int64,
				PipelineRunContextId:    pipelineRunContextId.Int64,
				TaskDetails:             tasks,
				StateHistory:            stateHistoryNew,
			},
			Metrics:            metrics,
			ResourceReferences: resourceReferences,
			PipelineSpec: model.PipelineSpec{
				PipelineId:           pipelineId,
				PipelineVersionId:    pvId,
				PipelineName:         pipelineName,
				PipelineSpecManifest: pipelineSpecManifest,
				WorkflowSpecManifest: workflowSpecManifest,
				Parameters:           parameters,
				RuntimeConfig:        runtimeConfig,
			},
		}
		run = run.ToV2()
		runs = append(runs, run)
	}
	return runs, nil
}

func parseMetrics(metricsInString sql.NullString) ([]*model.RunMetric, error) {
	if !metricsInString.Valid {
		return nil, nil
	}
	var metrics []*model.RunMetric
	if err := json.Unmarshal([]byte(metricsInString.String), &metrics); err != nil {
		return nil, util.Wrapf(err, "Failed to parse a run metric '%s'", metricsInString.String)
	}
	return metrics, nil
}

func parseRuntimeConfig(runtimeParameters sql.NullString, pipelineRoot sql.NullString) model.RuntimeConfig {
	var runtimeParametersString, pipelineRootString string
	if runtimeParameters.Valid {
		runtimeParametersString = runtimeParameters.String
	}
	if pipelineRoot.Valid {
		pipelineRootString = pipelineRoot.String
	}
	return model.RuntimeConfig{Parameters: runtimeParametersString, PipelineRoot: pipelineRootString}
}

func parseResourceReferences(resourceRefString sql.NullString) ([]*model.ResourceReference, error) {
	if !resourceRefString.Valid {
		return nil, nil
	}
	var refs []*model.ResourceReference
	if err := json.Unmarshal([]byte(resourceRefString.String), &refs); err != nil {
		return nil, util.Wrapf(err, "Failed to parse resource references '%s'", resourceRefString.String)
	}
	return refs, nil
}

func parseTaskDetails(tasksInString sql.NullString) ([]*model.Task, error) {
	if !tasksInString.Valid {
		return nil, nil
	}
	var taskDetails []*model.Task
	if err := json.Unmarshal([]byte(tasksInString.String), &taskDetails); err != nil {
		return nil, util.Wrapf(err, "Failed to parse task details '%s'", tasksInString.String)
	}
	return taskDetails, nil
}

func (s *RunStore) CreateRun(r *model.Run) (*model.Run, error) {
	r = r.ToV1().ToV2()
	if r.StorageState == "" || r.StorageState == model.StorageStateUnspecified || r.StorageState == model.StorageStateUnspecifiedV1 {
		r.StorageState = model.StorageStateAvailable
	}

	if !r.StorageState.IsValid() {
		return nil, util.NewInvalidInputError("Invalid value for StorageState field: %q", r.StorageState)
	}

	r.RunDetails.StateHistory = append(r.RunDetails.StateHistory, &model.RuntimeStatus{
		UpdateTimeInSec: s.time.Now().Unix(),
		State:           r.RunDetails.State,
	})

	stateHistoryString := ""
	if history, err := json.Marshal(r.RunDetails.StateHistory); err == nil {
		stateHistoryString = string(history)
	} else {
		return nil, util.NewInternalServerError(err, "Failed to marshal state history in a new run")
	}
	runSql, runArgs, err := sq.
		Insert("run_details").
		SetMap(sq.Eq{
			"UUID":                    r.UUID,
			"ExperimentUUID":          r.ExperimentId,
			"DisplayName":             r.DisplayName,
			"Name":                    r.K8SName,
			"StorageState":            r.StorageState.ToString(),
			"Namespace":               r.Namespace,
			"ServiceAccount":          r.ServiceAccount,
			"Description":             r.Description,
			"CreatedAtInSec":          r.RunDetails.CreatedAtInSec,
			"ScheduledAtInSec":        r.RunDetails.ScheduledAtInSec,
			"FinishedAtInSec":         r.RunDetails.FinishedAtInSec,
			"Conditions":              r.RunDetails.Conditions,
			"WorkflowRuntimeManifest": r.RunDetails.WorkflowRuntimeManifest,
			"PipelineRuntimeManifest": r.RunDetails.PipelineRuntimeManifest,
			"PipelineId":              r.PipelineSpec.PipelineId,
			"PipelineName":            r.PipelineSpec.PipelineName,
			"PipelineSpecManifest":    r.PipelineSpec.PipelineSpecManifest,
			"WorkflowSpecManifest":    r.PipelineSpec.WorkflowSpecManifest,
			"Parameters":              r.PipelineSpec.Parameters,
			"RuntimeParameters":       r.PipelineSpec.RuntimeConfig.Parameters,
			"PipelineRoot":            r.PipelineSpec.RuntimeConfig.PipelineRoot,
			"PipelineVersionId":       r.PipelineSpec.PipelineVersionId,
			"JobUUID":                 r.RecurringRunId,
			"State":                   r.RunDetails.State.ToString(),
			"StateHistory":            stateHistoryString,
		}).ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create query to store run to run table: '%v/%v",
			r.Namespace, r.DisplayName)
	}

	// Use a transaction to make sure both run and its resource references are stored.
	tx, err := s.db.Begin()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to create a new transaction to create run")
	}

	_, err = tx.Exec(runSql, runArgs...)
	if err != nil {
		tx.Rollback()
		return nil, util.NewInternalServerError(err, "Failed to store run %v to table", r.DisplayName)
	}

	// TODO(gkcalat): consider moving resource reference management to ResourceManager
	// and provide logic for data migration for v1beta1 data.
	err = s.resourceReferenceStore.CreateResourceReferences(tx, r.ResourceReferences)
	if err != nil {
		tx.Rollback()
		return nil, util.NewInternalServerError(err, "Failed to store resource references to table for run %v ", r.DisplayName)
	}
	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		return nil, util.NewInternalServerError(err, "Failed to store run %v and its resource references to table", r.DisplayName)
	}
	return r, nil
}

func (s *RunStore) UpdateRun(run *model.Run) error {
	tx, err := s.db.DB.Begin()
	if err != nil {
		return util.NewInternalServerError(err, "transaction creation failed")
	}
	run.RunDetails.StateHistory = append(run.RunDetails.StateHistory, &model.RuntimeStatus{
		UpdateTimeInSec: s.time.Now().Unix(),
		State:           run.RunDetails.State,
	})
	stateHistoryString := ""
	if historyString, err := json.Marshal(run.RunDetails.StateHistory); err == nil {
		stateHistoryString = string(historyString)
	}
	sql, args, err := sq.
		Update("run_details").
		SetMap(sq.Eq{
			"Conditions":              run.Conditions,
			"State":                   run.State.ToString(),
			"StateHistory":            stateHistoryString,
			"FinishedAtInSec":         run.FinishedAtInSec,
			"WorkflowRuntimeManifest": run.WorkflowRuntimeManifest,
		}).
		Where(sq.Eq{"UUID": run.UUID}).
		ToSql()
	if err != nil {
		tx.Rollback()
		return util.NewInternalServerError(err,
			"Failed to create query to update run %s", run.UUID)
	}
	result, err := tx.Exec(sql, args...)
	if err != nil {
		tx.Rollback()
		return util.NewInternalServerError(err,
			"Failed to update run %s", run.UUID)
	}
	r, err := result.RowsAffected()
	if err != nil {
		tx.Rollback()
		return util.NewInternalServerError(err,
			"Failed to update run %s", run.UUID)
	}
	if r > 1 {
		tx.Rollback()
		return util.NewInternalServerError(errors.New("Failed to update run"), "Failed to update run %s. More than 1 rows affected", run.UUID)
	} else if r == 0 {
		tx.Rollback()
		return util.Wrap(util.NewResourceNotFoundError("Run", run.UUID), "Failed to update run")
	}

	if err := tx.Commit(); err != nil {
		return util.NewInternalServerError(err, "failed to commit transaction for run %s", run.UUID)
	}
	return nil
}

func (s *RunStore) ArchiveRun(runId string) error {
	sql, args, err := sq.
		Update("run_details").
		SetMap(sq.Eq{
			"StorageState": model.StorageStateArchived.ToString(),
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
			"StorageState": model.StorageStateAvailable.ToString(),
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
		return util.NewInternalServerError(err, "Failed to create a new transaction to delete run")
	}
	_, err = tx.Exec(runSql, runArgs...)
	if err != nil {
		tx.Rollback()
		return util.NewInternalServerError(err, "Failed to delete run %s from table", id)
	}
	err = s.resourceReferenceStore.DeleteResourceReferences(tx, id, model.RunResourceType)
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

// Creates a new metric in run_metrics table if does not exist.
func (s *RunStore) CreateMetric(metric *model.RunMetric) error {
	payloadBytes, err := json.Marshal(metric)
	if err != nil {
		return util.NewInternalServerError(err,
			"Failed to marshal a run metric to json: %+v", metric)
	}
	sql, args, err := sq.
		Insert("run_metrics").
		SetMap(sq.Eq{
			"RunUUID":     metric.RunUUID,
			"NodeID":      metric.NodeID,
			"Name":        metric.Name,
			"NumberValue": metric.NumberValue,
			"Format":      metric.Format,
			"Payload":     string(payloadBytes),
		}).ToSql()
	if err != nil {
		return util.NewInternalServerError(err,
			"Failed to create query for inserting a run metric: %+v", metric)
	}
	_, err = s.db.Exec(sql, args...)
	if err != nil {
		if s.db.IsDuplicateError(err) {
			return util.NewAlreadyExistError(
				"Failed to create a run metric. Same metric has been reported before: %s/%s", metric.NodeID, metric.Name)
		}
		return util.NewInternalServerError(err, "Failed to insert a run metric: %v", metric)
	}
	return nil
}

// Returns a new RunStore.
func NewRunStore(db *DB, time util.TimeInterface) *RunStore {
	return &RunStore{
		db:                     db,
		resourceReferenceStore: NewResourceReferenceStore(db),
		time:                   time,
	}
}

func (s *RunStore) TerminateRun(runId string) error {
	// TODO(gkcalat): append CANCELLING to StateHistory
	result, err := s.db.Exec(`
		UPDATE run_details
		SET Conditions = ?, State = ?
		WHERE UUID = ? AND (State = ? OR State = ? OR State = ? OR State = ?)`,
		string(model.RuntimeStateCancelling.ToV1()),
		model.RuntimeStateCancelling.ToString(),
		runId,
		model.RuntimeStatePaused.ToString(),
		model.RuntimeStatePending.ToString(),
		model.RuntimeStateRunning.ToString(),
		model.RuntimeStateUnspecified.ToString(),
	)
	if err != nil {
		return util.NewInternalServerError(err,
			"Failed to terminate a run %s. Error: '%v'", runId, err.Error())
	}

	if r, _ := result.RowsAffected(); r != 1 {
		return util.NewInvalidInputError("Failed to terminate a run %s. Row not found", runId)
	}
	return nil
}

// Add a metric as a new field to the select clause by join the passed-in SQL query with run_metrics table.
// With the metric as a field in the select clause enable sorting on this metric afterwards.
// TODO(jingzhang36): example of resulting SQL query and explanation for it.
func (s *RunStore) addSortByRunMetricToSelect(sqlBuilder sq.SelectBuilder, opts *list.Options) sq.SelectBuilder {
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

func (s *RunStore) scanRowsToRunMetrics(rows *sql.Rows) ([]*model.RunMetric, error) {
	var metrics []*model.RunMetric
	for rows.Next() {
		var runId, nodeId, name, form, payload string
		var val float64
		err := rows.Scan(
			&runId,
			&nodeId,
			&name,
			&val,
			&form,
			&payload,
		)
		if err != nil {
			glog.Errorf("Failed to scan row into a run metric: %v", err)
			return metrics, nil
		}

		metrics = append(
			metrics,
			&model.RunMetric{
				RunUUID:     runId,
				NodeID:      nodeId,
				Name:        name,
				NumberValue: val,
				Format:      form,
				Payload:     payload,
			},
		)
	}
	return metrics, nil
}

// TODO(gkcalat): consider removing this if we no longer support separate metrics API in v2beta1 API.
// Fetches run metrics for a given run id.
func (s *RunStore) GetMetrics(runId string) ([]*model.RunMetric, error) {
	sql, args, err := sq.Select(runMetricsColumns...).
		From("run_metrics").
		Where(sq.Eq{"RunUUID": runId}).
		ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to get run metrics: %v", err.Error())
	}
	r, err := s.db.Query(sql, args...)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to get run metrics: %v", err.Error())
	}
	defer r.Close()
	metrics, err := s.scanRowsToRunMetrics(r)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to get run metrics: %v", err.Error())
	}
	return metrics, nil
}
