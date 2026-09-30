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
	"strconv"

	sq "github.com/Masterminds/squirrel"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
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
	qb := dbDialect.QueryBuilder()
	query, args, err := qb.Select(dialect.QuoteAll(q, taskColumns)...).
		From(q(tableName)).
		Where(sq.Eq{q("RunUUID"): run.UUID}).
		OrderBy(q("UUID")).
		ToSql()
	if err != nil {
		return err
	}
	rows, err := tx.Query(dbDialect.SelectForUpdate(query), args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	tasks := make(map[string]*model.Task)
	var orderedTasks []*model.Task
	for rows.Next() {
		task, err := scanTaskRow(rows)
		if err != nil {
			return err
		}
		tasks[task.UUID] = task
		orderedTasks = append(orderedTasks, task)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}

	generation := strconv.FormatInt(run.RetryGeneration, 10)
	visited := make(map[string]bool)
	for _, task := range orderedTasks {
		properties, ok := task.StatusMetadata["customProperties"].(map[string]interface{})
		if !ok || properties[util.DriverRetryGenerationKey] != generation {
			continue
		}
		// A terminal native/cache task may still have unfinished ancestors when
		// its driver died before acknowledging completion to the controller.
		for ancestor := task; ancestor != nil && !visited[ancestor.UUID]; {
			visited[ancestor.UUID] = true
			if ancestor.State == model.TaskStatus(apiv2beta1.PipelineTask_RUNNING) {
				if err := finalizeUnfinishedDriverTask(tx, dbDialect, run, ancestor); err != nil {
					return err
				}
			}
			if ancestor.ParentTaskUUID == nil {
				break
			}
			ancestor = tasks[*ancestor.ParentTaskUUID]
		}
	}
	return nil
}

func finalizeUnfinishedDriverTask(tx *sql.Tx, dbDialect dialect.DBDialect, run *model.Run, task *model.Task) error {
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
