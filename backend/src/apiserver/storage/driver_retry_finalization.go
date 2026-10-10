// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"

	sq "github.com/Masterminds/squirrel"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// finalizeDriverRetryTasks shares the guarded terminal run update transaction:
// ClaimRunForRetry cannot reset these tasks between the run and task writes.
func finalizeDriverRetryTasks(tx *sql.Tx, dbDialect dialect.DBDialect, run *model.Run) error {
	switch run.State.ToV2() {
	case model.RuntimeStateFailed, model.RuntimeStateCanceled:
	default:
		return nil
	}

	q := dbDialect.QuoteIdentifier
	query, args, err := dbDialect.QueryBuilder().Select(q("DriverRetryTasksPresent"), q("DriverRetryFinalizedGeneration")).From(q("run_details")).Where(sq.Eq{q("UUID"): run.UUID}).ToSql()
	if err != nil {
		return err
	}
	var present bool
	var finalized *int64
	if err := tx.QueryRow(query, args...).Scan(&present, &finalized); err != nil {
		return err
	}
	if !present || (finalized != nil && *finalized == run.RetryGeneration) {
		return nil
	}

	tasks, orderedTasks, err := loadDriverRetryFinalizationTasks(tx, dbDialect, run)
	if err != nil {
		return err
	}

	// A completion-only walk must not block a later failed descendant from
	// correcting the same ancestor's premature successful state.
	visited := make(map[string]bool)
	var failedTasks []*model.Task
	queued := make(map[string]bool)
	for _, task := range orderedTasks {
		if task.DriverRetryGeneration == nil || *task.DriverRetryGeneration != run.RetryGeneration {
			continue
		}
		// A terminal native/cache task may still have unfinished ancestors when
		// its driver died before acknowledging completion to the controller.
		failedDescendant := false
		for ancestor := task; ancestor != nil; {
			failedDescendant = failedDescendant || ancestor.State == model.TaskStatus(apiv2beta1.PipelineTask_RUNNING) || ancestor.State == model.TaskStatus(apiv2beta1.PipelineTask_FAILED)
			if propagatedFailure, seen := visited[ancestor.UUID]; seen && (propagatedFailure || !failedDescendant) {
				break
			}
			visited[ancestor.UUID] = failedDescendant
			prematureSuccess := ancestor.State == model.TaskStatus(apiv2beta1.PipelineTask_SUCCEEDED) || ancestor.State == model.TaskStatus(apiv2beta1.PipelineTask_CACHED) || ancestor.State == model.TaskStatus(apiv2beta1.PipelineTask_SKIPPED)
			if ancestor.State == model.TaskStatus(apiv2beta1.PipelineTask_RUNNING) || (failedDescendant && prematureSuccess) {
				if !queued[ancestor.UUID] {
					queued[ancestor.UUID] = true
					failedTasks = append(failedTasks, ancestor)
				}
			}
			if ancestor.ParentTaskUUID == nil {
				break
			}
			ancestor = tasks[*ancestor.ParentTaskUUID]
		}
	}
	for _, task := range failedTasks {
		if err := finalizeDriverTaskFailure(tx, dbDialect, run, task); err != nil {
			return err
		}
	}
	query, args, err = dbDialect.QueryBuilder().Update(q("run_details")).SetMap(sq.Eq{q("DriverRetryFinalizedGeneration"): run.RetryGeneration}).Where(sq.Eq{q("UUID"): run.UUID}).ToSql()
	if err != nil {
		return err
	}
	_, err = tx.Exec(query, args...)
	return err
}

// The caller holds the exclusive run lock, which excludes every tagged write.
// Discover only lightweight candidate identities first, so runs without driver
// retries never lock or deserialize all of their task inputs and outputs.
func loadDriverRetryFinalizationTasks(tx *sql.Tx, dbDialect dialect.DBDialect, run *model.Run) (map[string]*model.Task, []*model.Task, error) {
	q := dbDialect.QuoteIdentifier
	query, args, err := dbDialect.QueryBuilder().Select(q("UUID")).
		From(q(tableName)).
		Where(sq.Eq{q("RunUUID"): run.UUID}).
		Where(sq.Eq{q("DriverRetryGeneration"): run.RetryGeneration}).
		OrderBy(q("UUID")).ToSql()
	if err != nil {
		return nil, nil, err
	}
	rows, err := tx.Query(query, args...)
	if err != nil {
		return nil, nil, err
	}
	var pending []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, nil, err
		}
		pending = append(pending, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, nil, err
	}

	tasks := make(map[string]*model.Task)
	requested := make(map[string]bool, len(pending))
	for _, id := range pending {
		requested[id] = true
	}
	var orderedTasks []*model.Task
	for len(pending) > 0 {
		// Bound placeholders for large fan-outs and retain deterministic locks
		// within each frontier. Only tagged tasks and their ancestors are read.
		batchSize := min(len(pending), 500)
		batch := pending[:batchSize]
		pending = pending[batchSize:]
		query, args, err := dbDialect.QueryBuilder().Select(taskColumnsWithoutRecovery(q)...).
			From(q(tableName)).
			Where(sq.Eq{q("RunUUID"): run.UUID, q("UUID"): batch}).
			OrderBy(q("UUID")).ToSql()
		if err != nil {
			return nil, nil, err
		}
		rows, err := tx.Query(dbDialect.SelectForUpdate(query), args...)
		if err != nil {
			return nil, nil, err
		}
		for rows.Next() {
			task, err := scanTaskRow(rows)
			if err != nil {
				rows.Close()
				return nil, nil, err
			}
			if _, seen := tasks[task.UUID]; !seen {
				tasks[task.UUID] = task
				orderedTasks = append(orderedTasks, task)
			}
			if task.ParentTaskUUID != nil && !requested[*task.ParentTaskUUID] {
				requested[*task.ParentTaskUUID] = true
				pending = append(pending, *task.ParentTaskUUID)
			}
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, nil, err
		}
		if err := rows.Close(); err != nil {
			return nil, nil, err
		}

	}
	sort.Slice(orderedTasks, func(i, j int) bool { return orderedTasks[i].UUID < orderedTasks[j].UUID })
	return tasks, orderedTasks, nil
}

func finalizeDriverTaskFailure(tx *sql.Tx, dbDialect dialect.DBDialect, run *model.Run, task *model.Task) error {
	failedState := model.TaskStatus(apiv2beta1.PipelineTask_FAILED)
	history := task.StateHistory
	if len(history) == 0 || getLastTaskState(history) != failedState {
		entry, err := model.ProtoSliceToJSONSlice([]*apiv2beta1.PipelineTask_TaskStatus{{
			State:      apiv2beta1.PipelineTask_FAILED,
			UpdateTime: &timestamppb.Timestamp{Seconds: run.FinishedAtInSec},
		}})
		if err != nil {
			return err
		}
		history = append(history, entry...)
	}
	historyJSON, err := json.Marshal(history)
	if err != nil {
		return err
	}
	metadata := task.StatusMetadata
	if metadata == nil {
		metadata = model.JSONData{}
	}
	if message, _ := metadata["message"].(string); message == "" {
		metadata["message"] = "Run ended before the task completed."
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	q := dbDialect.QuoteIdentifier
	query, args, err := dbDialect.QueryBuilder().Update(q(tableName)).
		SetMap(sq.Eq{
			q("State"):          failedState,
			q("FinishedInSec"):  run.FinishedAtInSec,
			q("StatusMetadata"): string(metadataJSON),
			q("StateHistory"):   string(historyJSON),
		}).
		Where(sq.Eq{q("UUID"): task.UUID, q("RunUUID"): run.UUID}).
		ToSql()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(query, args...); err != nil {
		return fmt.Errorf("failed to finalize task %s: %w", task.UUID, err)
	}
	return nil
}
