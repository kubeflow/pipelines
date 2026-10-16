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
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"

	sq "github.com/Masterminds/squirrel"
	api "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

// FinalizeStoppedDriver is called by the authenticated runtime when the workflow
// controller will not schedule another driver attempt. It cannot claim a task.
func (s *TaskStore) FinalizeStoppedDriver(runID string, generation int64, taskName, parentTaskID string, iterationIndex *int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := s.lockRunForDriverTask(tx, runID, generation, true); err != nil {
		return err
	}
	// The exclusive run lock serializes this stop with every runtime create's
	// shared lock, including creates that have not inserted their task yet.
	key, err := driverTaskStopKey(runID, generation, taskName, parentTaskID, iterationIndex)
	if err != nil {
		return err
	}
	q := s.dbDialect.QuoteIdentifier
	stopSQL, stopArgs, err := s.dbDialect.Upsert("driver_task_stops", []string{"StopKey"}, false, []string{"StopKey"}).SetMap(sq.Eq{q("StopKey"): key, q("RunUUID"): runID}).ToSql()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(stopSQL, stopArgs...); err != nil {
		return err
	}
	parent := sq.Eq{q("ParentTaskUUID"): parentTaskID}
	if parentTaskID == "" {
		parent = sq.Eq{q("ParentTaskUUID"): nil}
	}
	query, args, err := s.dbDialect.QueryBuilder().Select(taskColumnsWithoutRecovery(q)...).From(q(tableName)).
		Where(sq.Eq{q("RunUUID"): runID, q("Name"): taskName}).Where(parent).ToSql()
	if err != nil {
		return err
	}
	rows, err := tx.Query(query, args...)
	if err != nil {
		return err
	}
	candidates, err := s.scanRows(rows)
	if err == nil {
		err = rows.Err()
	}
	rows.Close()
	if err != nil {
		return err
	}
	var target *model.Task
	for _, candidate := range candidates {
		index, err := taskIterationIndex(candidate.TypeAttrs)
		if err != nil {
			return err
		}
		if (index == nil) != (iterationIndex == nil) || (index != nil && *index != *iterationIndex) {
			continue
		}
		if target != nil {
			return util.NewFailedPreconditionError(fmt.Errorf("ambiguous stopped driver identity"), "Task %s has multiple matching runtime identities", taskName)
		}
		target = candidate
	}
	if target == nil {
		if parentTaskID == "" {
			return tx.Commit()
		}
		query, args, err := s.dbDialect.QueryBuilder().Select(taskColumnsWithoutRecovery(q)...).From(q(tableName)).Where(sq.Eq{q("UUID"): parentTaskID, q("RunUUID"): runID}).ToSql()
		if err != nil {
			return err
		}
		target, err = scanTaskRow(tx.QueryRow(query, args...))
		if errors.Is(err, sql.ErrNoRows) {
			return tx.Commit()
		}
		if err != nil {
			return err
		}
	}
	// Discover the immutable parent chain before taking locks, then lock the
	// whole set in UUID order, matching ordinary source/target writes.
	parents := map[string]*model.Task{target.UUID: target}
	for current := target; current.ParentTaskUUID != nil && *current.ParentTaskUUID != ""; {
		id := *current.ParentTaskUUID
		if _, seen := parents[id]; seen {
			return util.NewFailedPreconditionError(fmt.Errorf("cyclic task parents"), "Task %s has a cyclic parent chain", taskName)
		}
		query, args, err := s.dbDialect.QueryBuilder().Select(taskColumnsWithoutRecovery(q)...).From(q(tableName)).Where(sq.Eq{q("UUID"): id, q("RunUUID"): runID}).ToSql()
		if err != nil {
			return err
		}
		current, err = scanTaskRow(tx.QueryRow(query, args...))
		if err != nil {
			return err
		}
		parents[id] = current
	}
	ids := make([]string, 0, len(parents))
	for id := range parents {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		current, err := s.getTaskForUpdateWithRecovery(tx, id, false)
		if err != nil {
			return err
		}
		if current.RunUUID != runID || normalizedParentTaskUUID(current.ParentTaskUUID) != normalizedParentTaskUUID(parents[id].ParentTaskUUID) {
			return util.NewFailedPreconditionError(fmt.Errorf("task ancestry changed"), "Task %s changed parents; retry finalization", taskName)
		}
		parents[id] = current
	}
	target = parents[target.UUID]
	if target.DriverRetryGeneration != nil && *target.DriverRetryGeneration != generation {
		return staleDriverAttempt(target.UUID)
	}
	if target.DriverStoppedGeneration != nil && *target.DriverStoppedGeneration == generation {
		return tx.Commit()
	}
	run := &model.Run{UUID: runID, RunDetails: model.RunDetails{RetryGeneration: generation, FinishedAtInSec: s.time.Now().Unix()}}
	for _, id := range ids {
		task := parents[id]
		if id == target.UUID || !driverTaskCompleted(task) {
			query, args, err := s.dbDialect.QueryBuilder().Update(q(tableName)).SetMap(sq.Eq{q("DriverStoppedGeneration"): generation}).Where(sq.Eq{q("UUID"): id}).ToSql()
			if err != nil {
				return err
			}
			if _, err := tx.Exec(query, args...); err != nil {
				return err
			}
		}
		if driverTaskCompleted(task) || task.State == model.TaskStatus(api.PipelineTask_FAILED) {
			continue
		}
		if err := finalizeDriverTaskFailure(tx, s.dbDialect, run, task); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func driverTaskCompleted(task *model.Task) bool {
	switch api.PipelineTask_TaskState(task.State) {
	case api.PipelineTask_SUCCEEDED, api.PipelineTask_CACHED, api.PipelineTask_SKIPPED:
		return true
	default:
		return false
	}
}

// Length-prefixing preserves boundaries between user-controlled identity parts.
func driverTaskStopKey(runID string, generation int64, taskName, parentTaskID string, iteration *int64) (string, error) {
	var identity bytes.Buffer
	for _, value := range []string{runID, taskName, parentTaskID} {
		if err := writeLengthPrefixedString(&identity, value); err != nil {
			return "", err
		}
	}
	if err := binary.Write(&identity, binary.BigEndian, generation); err != nil {
		return "", err
	}
	if err := writeOptionalInt64(&identity, iteration); err != nil {
		return "", err
	}
	digest := sha256.Sum256(identity.Bytes())
	return hex.EncodeToString(digest[:]), nil
}

// The caller holds a shared run lock until its task create commits.
func (s *TaskStore) checkDriverTaskStop(tx *sql.Tx, task *model.Task) error {
	iteration, err := taskIterationIndex(task.TypeAttrs)
	if err != nil {
		return err
	}
	key, err := driverTaskStopKey(task.RunUUID, task.DriverWriteAuthority.Generation, task.Name, normalizedParentTaskUUID(task.ParentTaskUUID), iteration)
	if err != nil {
		return err
	}
	q := s.dbDialect.QuoteIdentifier
	query, args, err := s.dbDialect.QueryBuilder().Select(q("StopKey")).From(q("driver_task_stops")).Where(sq.Eq{q("StopKey"): key}).ToSql()
	if err != nil {
		return err
	}
	var stopped string
	err = tx.QueryRow(query, args...).Scan(&stopped)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	return staleDriverAttempt(task.Name)
}
