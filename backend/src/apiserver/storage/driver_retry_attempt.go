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
	api "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

func staleDriverAttempt(taskID string) error {
	return util.NewFailedPreconditionError(fmt.Errorf("driver task attempt does not match its owner"),
		"Task %s has a different driver retry attempt; discard this stale driver attempt", taskID)
}

func clearDriverTaskAuthority(task *model.Task) *model.Task {
	result := *task
	result.DriverWriteAuthority = nil
	result.DriverClaim = false
	result.DriverRecoveryUpdate = false
	return &result
}

func validateDriverTaskRequest(task *model.Task) error {
	authority := task.DriverWriteAuthority
	if authority == nil {
		if task.DriverClaim || task.DriverRecoveryUpdate {
			return util.NewInvalidInputError("Driver recovery writes require authenticated runtime authority")
		}
		return nil
	}
	if authority.Generation < 0 || (authority.SourceAttempt != nil && *authority.SourceAttempt < 0) {
		return util.NewInvalidInputError("Driver retry generation and attempt must be nonnegative")
	}
	if task.DriverClaim {
		if authority.SourceTaskID != "" || task.DriverRetryGeneration == nil || task.DriverRetryAttempt == nil ||
			*task.DriverRetryGeneration != authority.Generation || *task.DriverRetryAttempt < 0 {
			return util.NewInvalidInputError("Driver claims require their own generation and nonnegative attempt")
		}
		if authority.SourceAttempt != nil && *authority.SourceAttempt != *task.DriverRetryAttempt {
			return util.NewInvalidInputError("Driver claim attempt must match its runtime authority")
		}
	}
	if task.DriverRecoveryUpdate {
		if authority.SourceTaskID != "" && authority.SourceTaskID != task.UUID {
			return util.NewInvalidInputError("Dependent task writes cannot update driver recovery state")
		}
		for _, value := range []*string{task.DriverCheckpoint, task.DriverCachedOutputs} {
			if value != nil && !json.Valid([]byte(*value)) {
				return util.NewInvalidInputError("Driver recovery data must be valid JSON")
			}
		}
	}
	return nil
}

// Presence is monotonic. Initialize it before any shared run lock is held, so
// independent task claims never upgrade a shared lock and deadlock each other.
func (s *TaskStore) ensureDriverRetryPresence(runID string) error {
	q := s.dbDialect.QuoteIdentifier
	query, args, err := s.dbDialect.QueryBuilder().Select(q("DriverRetryTasksPresent")).From(q("run_details")).Where(sq.Eq{q("UUID"): runID}).ToSql()
	if err != nil {
		return err
	}
	var present bool
	if err := s.db.QueryRow(query, args...).Scan(&present); err != nil {
		return err
	}
	if present {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := tx.QueryRow(s.dbDialect.SelectForUpdate(query), args...).Scan(&present); err != nil {
		return err
	}
	if !present {
		update, values, err := s.dbDialect.QueryBuilder().Update(q("run_details")).SetMap(sq.Eq{q("DriverRetryTasksPresent"): true}).Where(sq.Eq{q("UUID"): runID}).ToSql()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(update, values...); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// CreateTask alone advances ownership, under shared run and exclusive task locks.
func (s *TaskStore) claimDriverTaskAttempt(tx *sql.Tx, stored, request *model.Task) (*model.Task, error) {
	if !request.DriverClaim {
		if stored.DriverRetryGeneration != nil {
			return nil, staleDriverAttempt(stored.UUID)
		}
		return stored, nil
	}
	generation, attempt := *request.DriverRetryGeneration, *request.DriverRetryAttempt
	if stored.DriverStoppedGeneration != nil && *stored.DriverStoppedGeneration == generation {
		return nil, staleDriverAttempt(stored.UUID)
	}
	if stored.DriverRetryGeneration != nil {
		if *stored.DriverRetryGeneration > generation {
			return nil, staleDriverAttempt(stored.UUID)
		}
		if *stored.DriverRetryGeneration < generation {
			switch api.PipelineTask_TaskState(stored.State) {
			case api.PipelineTask_SUCCEEDED, api.PipelineTask_CACHED, api.PipelineTask_SKIPPED:
			default:
				return nil, util.NewFailedPreconditionError(fmt.Errorf("unfinished task belongs to an older retry generation"), "Task %s must be reset before claiming a new retry generation; retry the run through the API", stored.UUID)
			}
		} else if stored.DriverRetryAttempt != nil {
			if attempt < *stored.DriverRetryAttempt {
				return nil, staleDriverAttempt(stored.UUID)
			}
			if attempt == *stored.DriverRetryAttempt {
				return stored, nil
			}
		}
	}
	q := s.dbDialect.QuoteIdentifier
	query, args, err := s.dbDialect.QueryBuilder().Update(q(tableName)).SetMap(sq.Eq{
		q("DriverRetryGeneration"): generation, q("DriverRetryAttempt"): attempt, q("DriverStoppedGeneration"): nil,
	}).Where(sq.Eq{q("UUID"): stored.UUID, q("RunUUID"): stored.RunUUID}).ToSql()
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(query, args...); err != nil {
		return nil, err
	}
	claimed := *stored
	claimed.DriverRetryGeneration, claimed.DriverRetryAttempt = &generation, &attempt
	claimed.DriverStoppedGeneration = nil
	return &claimed, nil
}

// Lock source and target in UUID order, preserving parallel independent writes.
func (s *TaskStore) lockDriverTaskWriteRows(tx *sql.Tx, targetID, sourceID string) (*model.Task, *model.Task, error) {
	ids := []string{targetID}
	if sourceID != "" && sourceID != targetID {
		ids = append(ids, sourceID)
	}
	sort.Strings(ids)
	var target, source *model.Task
	for _, id := range ids {
		task, err := s.getTaskForUpdate(tx, id)
		if err != nil {
			return nil, nil, err
		}
		if id == targetID {
			target = task
		}
		if id == sourceID {
			source = task
		}
	}
	return target, source, nil
}

func (s *TaskStore) validateDriverTaskAttempt(incoming, stored, source *model.Task) (*model.Task, error) {
	authority := incoming.DriverWriteAuthority
	stopped := stored.DriverStoppedGeneration != nil && (authority == nil || authority.Generation <= *stored.DriverStoppedGeneration)
	dependent := authority != nil && authority.SourceTaskID != "" && authority.SourceTaskID != stored.UUID
	if stopped && !dependent {
		return nil, staleDriverAttempt(stored.UUID)
	}
	if authority == nil {
		if stored.DriverRetryGeneration != nil {
			return nil, staleDriverAttempt(stored.UUID)
		}
	} else {
		owner := stored
		if authority.SourceTaskID != "" {
			owner = source
		}
		if owner == nil || owner.RunUUID != stored.RunUUID || owner.RunUUID != incoming.RunUUID {
			return nil, staleDriverAttempt(authority.SourceTaskID)
		}
		if (owner.DriverStoppedGeneration != nil && authority.Generation <= *owner.DriverStoppedGeneration) ||
			(owner.DriverRetryGeneration != nil && *owner.DriverRetryGeneration != authority.Generation) ||
			(owner.DriverRetryAttempt == nil) != (authority.SourceAttempt == nil) ||
			(owner.DriverRetryAttempt != nil && *owner.DriverRetryAttempt != *authority.SourceAttempt) {
			return nil, staleDriverAttempt(owner.UUID)
		}
		if incoming.DriverRecoveryUpdate && owner.DriverRetryAttempt == nil {
			return nil, staleDriverAttempt(owner.UUID)
		}
	}
	updated := *incoming
	if stopped {
		// A completed sibling may still contribute outputs to a failed ancestor.
		// Preserve the failure while accepting that independently fenced write.
		updated.State, updated.FinishedInSec = stored.State, stored.FinishedInSec
		updated.StatusMetadata = stored.StatusMetadata
	}
	// Ownership is never taken from the request or target snapshot. Omitted
	// public metadata stays nil, preserving UpdateTask's established semantics.
	updated.DriverRetryGeneration, updated.DriverRetryAttempt = stored.DriverRetryGeneration, stored.DriverRetryAttempt
	updated.DriverStoppedGeneration = stored.DriverStoppedGeneration
	if !incoming.DriverRecoveryUpdate || incoming.DriverCheckpoint == nil {
		updated.DriverCheckpoint = stored.DriverCheckpoint
	}
	if !incoming.DriverRecoveryUpdate || incoming.DriverCachedOutputs == nil {
		updated.DriverCachedOutputs = stored.DriverCachedOutputs
	}
	return &updated, nil
}
