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
	api "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

type driverTaskFence struct {
	generation    int64
	tagged        bool
	attempt       int64
	claimed       bool
	sourceTaskID  string
	sourceAttempt int64
}

func driverTaskProperties(task *model.Task) map[string]interface{} {
	properties, _ := task.StatusMetadata["customProperties"].(map[string]interface{})
	return properties
}

func driverAttemptProperty(properties map[string]interface{}, key string) (int64, bool, error) {
	value, present := properties[key]
	if !present {
		return 0, false, nil
	}
	text, ok := value.(string)
	attempt, err := strconv.ParseInt(text, 10, 64)
	if !ok || err != nil || attempt < 0 || strconv.FormatInt(attempt, 10) != text {
		return 0, true, util.NewInvalidInputError("Driver retry attempt must be a nonnegative decimal string; resubmit the task with its original attempt")
	}
	return attempt, true, nil
}

func parseDriverTaskFence(task *model.Task) (driverTaskFence, error) {
	var fence driverTaskFence
	var err error
	fence.generation, fence.tagged, err = taskRetryGeneration(task)
	if err != nil {
		return fence, err
	}
	properties := driverTaskProperties(task)
	fence.attempt, fence.claimed, err = driverAttemptProperty(properties, util.DriverRetryAttemptKey)
	if err != nil {
		return fence, err
	}
	sourceValue, hasSource := properties[util.DriverRetrySourceTaskKey]
	var hasSourceAttempt bool
	fence.sourceAttempt, hasSourceAttempt, err = driverAttemptProperty(properties, util.DriverRetrySourceAttemptKey)
	if err != nil {
		return fence, err
	}
	if hasSource || hasSourceAttempt {
		var validSource bool
		fence.sourceTaskID, validSource = sourceValue.(string)
		if !hasSource || !hasSourceAttempt || !validSource || fence.sourceTaskID == "" {
			return fence, util.NewInvalidInputError("Driver retry source requires both a task ID and attempt; resubmit with the originating driver's fence")
		}
	}
	if (fence.claimed || hasSource) && !fence.tagged {
		return fence, util.NewInvalidInputError("Driver retry attempt requires a run generation; resubmit with the originating driver's fence")
	}
	return fence, nil
}

func staleDriverAttempt(taskID string) error {
	return util.NewFailedPreconditionError(fmt.Errorf("driver task attempt does not match its owner"),
		"Task %s has a different driver retry attempt; discard this stale driver attempt", taskID)
}

func copyDriverTaskMetadata(task *model.Task) (model.JSONData, map[string]interface{}) {
	metadata := make(model.JSONData, len(task.StatusMetadata))
	for key, value := range task.StatusMetadata {
		metadata[key] = value
	}
	properties := make(map[string]interface{}, len(driverTaskProperties(task)))
	for key, value := range driverTaskProperties(task) {
		properties[key] = value
	}
	metadata["customProperties"] = properties
	return metadata, properties
}

// CreateTask is the only operation that can advance ownership. The caller holds
// the run lock, so a delayed claim or write cannot overtake a newer attempt.
func (s *TaskStore) claimDriverTaskAttempt(tx *sql.Tx, task *model.Task, incoming driverTaskFence) (*model.Task, error) {
	stored, err := parseDriverTaskFence(task)
	if err != nil {
		return nil, err
	}
	if !incoming.claimed {
		if stored.claimed {
			return nil, staleDriverAttempt(task.UUID)
		}
		return task, nil
	}
	if stored.tagged {
		if stored.generation > incoming.generation {
			return nil, staleDriverAttempt(task.UUID)
		}
		if stored.generation < incoming.generation {
			switch api.PipelineTask_TaskState(task.State) {
			case api.PipelineTask_SUCCEEDED, api.PipelineTask_CACHED, api.PipelineTask_SKIPPED:
			default:
				return nil, util.NewFailedPreconditionError(fmt.Errorf("unfinished task belongs to an older retry generation"),
					"Task %s must be reset before claiming a new retry generation; retry the run through the API", task.UUID)
			}
		} else if stored.claimed {
			if incoming.attempt < stored.attempt {
				return nil, staleDriverAttempt(task.UUID)
			}
			if incoming.attempt == stored.attempt {
				return task, nil
			}
		}
	}
	metadata, properties := copyDriverTaskMetadata(task)
	properties[util.DriverRetryGenerationKey] = strconv.FormatInt(incoming.generation, 10)
	properties[util.DriverRetryAttemptKey] = strconv.FormatInt(incoming.attempt, 10)
	encoded, err := json.Marshal(metadata)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to marshal driver task claim")
	}
	q := s.dbDialect.QuoteIdentifier
	query, args, err := s.dbDialect.QueryBuilder().Update(q(tableName)).
		SetMap(sq.Eq{q("StatusMetadata"): string(encoded)}).
		Where(sq.Eq{q("UUID"): task.UUID, q("RunUUID"): task.RunUUID}).ToSql()
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to build driver task claim")
	}
	if _, err := tx.Exec(query, args...); err != nil {
		return nil, util.NewInternalServerError(err, "Failed to claim driver task attempt")
	}
	claimed := *task
	claimed.StatusMetadata = metadata
	return &claimed, nil
}

func (s *TaskStore) validateDriverTaskAttempt(tx *sql.Tx, incoming, stored *model.Task, fence driverTaskFence) (*model.Task, error) {
	owner, err := parseDriverTaskFence(stored)
	if err != nil {
		return nil, err
	}
	if fence.sourceTaskID == "" {
		if owner.claimed != fence.claimed || (owner.claimed &&
			(owner.generation != fence.generation || owner.attempt != fence.attempt)) {
			return nil, staleDriverAttempt(stored.UUID)
		}
		return incoming, nil
	}
	// All claimed writes lock their run first, serializing source and target
	// locks with CreateTask claims, manual retries, and terminal finalization.
	source, err := s.getTaskForUpdate(tx, fence.sourceTaskID)
	if err != nil {
		return nil, err
	}
	sourceOwner, err := parseDriverTaskFence(source)
	if err != nil {
		return nil, err
	}
	if source.RunUUID != incoming.RunUUID || !sourceOwner.claimed ||
		sourceOwner.generation != fence.generation || sourceOwner.attempt != fence.sourceAttempt {
		return nil, staleDriverAttempt(fence.sourceTaskID)
	}
	updated := *incoming
	metadata, properties := copyDriverTaskMetadata(incoming)
	delete(properties, util.DriverRetrySourceTaskKey)
	delete(properties, util.DriverRetrySourceAttemptKey)
	// A child authorizes this write, but cannot replace its parent's ownership
	// or the parent's saved handoff with recovery data from a stale snapshot.
	keys := []string{util.DriverRetryAttemptKey, "_kfp_driver_checkpoint", "_kfp_driver_cached_outputs"}
	if owner.claimed {
		keys = append(keys, util.DriverRetryGenerationKey)
	}
	storedProperties := driverTaskProperties(stored)
	for _, key := range keys {
		delete(properties, key)
		if value, present := storedProperties[key]; present {
			properties[key] = value
		}
	}
	updated.StatusMetadata = metadata
	return &updated, nil
}
