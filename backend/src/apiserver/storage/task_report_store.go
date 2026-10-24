// Copyright 2026 The Kubeflow Authors
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
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	sq "github.com/Masterminds/squirrel"
	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

var (
	// ErrTaskReportIdentityRequired rejects managed reports even after finalization.
	ErrTaskReportIdentityRequired = errors.New("managed task reports require the reporting task and current attempt IDs")
	// ErrTaskReportScope rejects writes outside the reporting task's lineage.
	ErrTaskReportScope = errors.New("task report may mutate only the reporting task and its ancestors in the same run")
)

type taskReportScope struct {
	managed  bool
	reporter *model.Task
	attempt  string
	targets  map[string]*model.Task
}

func uniqueReportTaskIDs(ids []string) ([]string, error) {
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		if id == "" {
			return nil, errors.New("task report requires nonempty target task IDs")
		}
		seen[id] = true
	}
	if len(seen) == 0 {
		return nil, errors.New("task report requires at least one target task")
	}
	unique := make([]string, 0, len(seen))
	for id := range seen {
		unique = append(unique, id)
	}
	sort.Strings(unique)
	return unique, nil
}

// withTaskReport binds validation and every report mutation to the same
// transaction. ExecutorType, not runtime-row existence, distinguishes Argo
// reports from managed reports after an attempt has been finalized.
func (s *TaskStore) withTaskReport(identity *model.TaskReportIdentity, ids []string, change func(sqlTransaction, *taskReportScope) error) error {
	if s.lease != nil {
		unbound := *s
		unbound.lease = nil
		return unbound.withTaskReport(identity, ids, change)
	}
	ids, err := uniqueReportTaskIDs(ids)
	if err != nil {
		return err
	}
	if identity != nil {
		if identity.TaskUUID == "" || identity.AttemptUUID == "" {
			return ErrTaskReportIdentityRequired
		}
		return (&runtimeStore{tasks: s}).withAttempt(identity.TaskUUID, identity.AttemptUUID, false,
			func(tx sqlTransaction, runtime *model.RuntimeTask, reporter *model.Task) error {
				if runtimeCancellationPhase(runtime.State) {
					return ErrRuntimeRunCanceled
				}
				if (runtime.State != model.RuntimeTaskSubmitting && runtime.State != model.RuntimeTaskRunning) || terminalRuntimeTask(reporter.State) {
					return fmt.Errorf("%w: launcher reporting is closed in runtime phase %s", ErrRuntimeTaskConflict, runtime.State)
				}
				q := s.dbDialect.QuoteIdentifier
				query, args, err := s.dbDialect.QueryBuilder().Select(q("ExecutorType"), q("State"), q("Conditions")).
					From(q("run_details")).Where(sq.Eq{q("UUID"): reporter.RunUUID}).ToSql()
				if err != nil {
					return err
				}
				var executor, state, conditions sql.NullString
				if err := tx.QueryRow(query, args...).Scan(&executor, &state, &conditions); err != nil {
					return err
				}
				if !executor.Valid {
					return ErrTaskReportScope
				}
				effectiveState := model.RuntimeState(state.String).ToV2()
				if effectiveState == model.RuntimeStateUnspecified {
					effectiveState = model.RuntimeState(conditions.String).ToV2()
				}
				if terminalRunStateValues[string(effectiveState)] {
					return fmt.Errorf("%w: owning run is terminal", ErrRuntimeTaskConflict)
				}
				scope := &taskReportScope{managed: true, reporter: reporter, attempt: identity.AttemptUUID, targets: make(map[string]*model.Task)}
				current := reporter
				for current != nil {
					if current.RunUUID != reporter.RunUUID || scope.targets[current.UUID] != nil {
						return ErrTaskReportScope
					}
					scope.targets[current.UUID] = current
					if current.ParentTaskUUID == nil || *current.ParentTaskUUID == "" {
						break
					}
					current, err = s.getTaskForUpdate(tx, *current.ParentTaskUUID)
					if err != nil {
						return err
					}
				}
				for _, id := range ids {
					if scope.targets[id] == nil {
						return ErrTaskReportScope
					}
				}
				return change(tx, scope)
			})
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	scope, err := s.lockUnmanagedReportTargets(tx, ids)
	if err != nil {
		return err
	}
	if err := change(tx, scope); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *TaskStore) lockUnmanagedReportTargets(tx sqlTransaction, ids []string) (*taskReportScope, error) {
	q, qb := s.dbDialect.QuoteIdentifier, s.dbDialect.QueryBuilder()
	ownership := make(map[string]string, len(ids))
	runs := make(map[string]bool)
	for _, id := range ids {
		query, args, err := qb.Select(q("RunUUID")).From(q("tasks")).Where(sq.Eq{q("UUID"): id}).ToSql()
		if err != nil {
			return nil, err
		}
		var runID string
		if err := tx.QueryRow(query, args...).Scan(&runID); err != nil {
			return nil, err
		}
		ownership[id], runs[runID] = runID, true
	}
	runIDs := make([]string, 0, len(runs))
	for id := range runs {
		runIDs = append(runIDs, id)
	}
	sort.Strings(runIDs)
	for _, runID := range runIDs {
		if err := s.lockUnmanagedReportRun(tx, runID); err != nil {
			return nil, err
		}
	}
	scope := &taskReportScope{targets: make(map[string]*model.Task, len(ids))}
	for _, id := range ids {
		task, err := s.getTaskForUpdate(tx, id)
		if err != nil {
			return nil, err
		}
		if task.RunUUID != ownership[id] {
			return nil, ErrTaskReportScope
		}
		scope.targets[id] = task
	}
	return scope, nil
}

func (s *TaskStore) lockUnmanagedReportRun(tx sqlTransaction, runID string) error {
	q := s.dbDialect.QuoteIdentifier
	query, args, err := s.dbDialect.QueryBuilder().Select(q("ExecutorType")).From(q("run_details")).Where(sq.Eq{q("UUID"): runID}).ToSql()
	if err != nil {
		return err
	}
	var executor sql.NullString
	if err := tx.QueryRow(s.dbDialect.SelectForUpdate(query), args...).Scan(&executor); err != nil {
		return err
	}
	if executor.Valid {
		return ErrTaskReportIdentityRequired
	}
	return nil
}

// ReportCreateTask permits the legacy Argo create protocol only. Managed tasks
// are materialized by the coordinator, never recreated by a delayed launcher.
func (s *TaskStore) ReportCreateTask(task *model.Task) (*model.Task, error) {
	if task == nil || task.RunUUID == "" {
		return nil, errors.New("reported task creation requires an owning run")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if err := s.lockUnmanagedReportRun(tx, task.RunUUID); err != nil {
		return nil, err
	}
	created, err := s.createTaskWithExecutor(tx, task)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return created, nil
}

// ReportTasks applies an entire launcher task batch under its attempt fence.
// UpdateTask remains an internal/coordinator API; report handlers use this API.
func (s *TaskStore) ReportTasks(identity *model.TaskReportIdentity, updates []*model.Task) ([]*model.Task, error) {
	if s.lease != nil {
		unbound := *s
		unbound.lease = nil
		return unbound.ReportTasks(identity, updates)
	}
	ids := make([]string, len(updates))
	for i, update := range updates {
		if update == nil {
			return nil, errors.New("reported task cannot be nil")
		}
		ids[i] = update.UUID
	}
	err := s.withTaskReport(identity, ids, func(tx sqlTransaction, scope *taskReportScope) error {
		for _, update := range updates {
			current := scope.targets[update.UUID]
			if update.RunUUID != "" && update.RunUUID != current.RunUUID {
				return ErrTaskReportScope
			}
			candidate := *update
			if scope.managed {
				if terminalRuntimeTask(current.State) {
					return ErrRuntimeTaskConflict
				}
				if (candidate.ParentTaskUUID != nil && normalizedParentTaskUUID(candidate.ParentTaskUUID) != normalizedParentTaskUUID(current.ParentTaskUUID)) ||
					(candidate.Namespace != "" && candidate.Namespace != current.Namespace) ||
					(candidate.ScopePath != "" && candidate.ScopePath != current.ScopePath) ||
					(candidate.Type != 0 && candidate.Type != current.Type) {
					return ErrTaskReportScope
				}
				// Lineage and materialization attributes belong to the coordinator.
				candidate.ParentTaskUUID, candidate.TypeAttrs = nil, nil
				candidate.Name, candidate.DisplayName, candidate.Fingerprint = "", "", ""
				candidate.Namespace, candidate.RunUUID, candidate.Type = "", "", 0
				candidate.FinishedInSec = 0
				if update.UUID != scope.reporter.UUID {
					// Parent propagation cannot replace parent diagnostics or status.
					candidate = model.Task{UUID: update.UUID, OutputParameters: update.OutputParameters}
				} else if candidate.State != 0 {
					candidate.State = model.TaskStatus(apiv2.PipelineTask_RUNNING)
				}
			}
			if err := s.updateTaskInTransaction(tx, &candidate, false); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	result := make([]*model.Task, len(updates))
	for i, update := range updates {
		result[i], err = s.GetTask(update.UUID)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

// ReportArtifacts creates metadata and links, or links existing metadata, under
// the same fence as task reports. Managed creations use attempt-scoped stable
// identities so a lost RPC response can be replayed without duplicate outputs.
func (s *TaskStore) ReportArtifacts(identity *model.TaskReportIdentity, artifacts []*model.Artifact, links []*model.ArtifactTask, reuse bool) ([]*model.Artifact, []*model.ArtifactTask, error) {
	if s.lease != nil {
		unbound := *s
		unbound.lease = nil
		return unbound.ReportArtifacts(identity, artifacts, links, reuse)
	}
	if len(artifacts) != 0 && len(artifacts) != len(links) {
		return nil, nil, errors.New("reported artifacts and links must be index-aligned")
	}
	ids := make([]string, len(links))
	for i, link := range links {
		if link == nil {
			return nil, nil, errors.New("reported artifact link cannot be nil")
		}
		ids[i] = link.TaskID
	}
	created := make([]*model.Artifact, 0, len(artifacts))
	createdLinks := make([]*model.ArtifactTask, 0, len(links))
	err := s.withTaskReport(identity, ids, func(tx sqlTransaction, scope *taskReportScope) error {
		store := NewArtifactStore(s.db, s.time, s.uuid, s.dbDialect)
		for i, source := range links {
			task := scope.targets[source.TaskID]
			if scope.managed && terminalRuntimeTask(task.State) {
				return ErrRuntimeTaskConflict
			}
			if source.RunUUID != task.RunUUID {
				return ErrTaskReportScope
			}
			link := *source
			if err := link.SyncIterationFromProducer(); err != nil {
				return err
			}
			if len(artifacts) != 0 {
				if artifacts[i] == nil {
					return errors.New("reported artifact cannot be nil")
				}
				if scope.managed && artifacts[i].Namespace != "" && task.Namespace != "" && artifacts[i].Namespace != task.Namespace {
					return ErrTaskReportScope
				}
				artifact, err := s.createReportedArtifact(tx, store, scope, artifacts[i], &link, reuse)
				if err != nil {
					return err
				}
				created = append(created, artifact)
				link.ArtifactID = artifact.UUID
			} else {
				artifact, err := s.getReportArtifact(tx, link.ArtifactID)
				if err != nil {
					return err
				}
				if scope.managed && artifact.Namespace != "" && task.Namespace != "" && artifact.Namespace != task.Namespace {
					return ErrTaskReportScope
				}
			}
			if scope.managed || reuse {
				existing, err := store.getArtifactTaskByUniqueLinkWithQuerier(tx, &link)
				if err != nil {
					return err
				}
				if existing != nil {
					createdLinks = append(createdLinks, existing)
					continue
				}
				identityJSON, err := json.Marshal([]any{"runtime-report-link", link.ArtifactID, link.TaskID, link.Type, link.Iteration, link.ArtifactKey})
				if err != nil {
					return err
				}
				link.UUID = util.NewDeterministicUUID(string(identityJSON))
				if err := (&runtimeStore{tasks: s}).checkpointArtifactLink(tx, &link); err != nil {
					return err
				}
			} else {
				stored, err := createArtifactTaskWithExecutor(tx.Exec, s.uuid, &link, s.dbDialect)
				if err != nil {
					return err
				}
				link = *stored
			}
			createdLinks = append(createdLinks, &link)
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return created, createdLinks, nil
}

func (s *TaskStore) getReportArtifact(tx sqlTransaction, id string) (*model.Artifact, error) {
	q := s.dbDialect.QuoteIdentifier
	query, args, err := s.dbDialect.QueryBuilder().Select(dialect.QuoteAll(q, artifactColumns)...).
		From(q("artifacts")).Where(sq.Eq{q("UUID"): id}).ToSql()
	if err != nil {
		return nil, err
	}
	rows, err := tx.Query(s.dbDialect.SelectForUpdate(query), args...)
	if err != nil {
		return nil, err
	}
	artifacts, err := (&ArtifactStore{}).scanRows(rows)
	closeErr := rows.Close()
	if err != nil {
		return nil, err
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(artifacts) == 0 {
		return nil, sql.ErrNoRows
	}
	return artifacts[0], nil
}

func (s *TaskStore) createReportedArtifact(tx sqlTransaction, store *ArtifactStore, scope *taskReportScope, source *model.Artifact, link *model.ArtifactTask, reuse bool) (*model.Artifact, error) {
	if reuse {
		return s.reuseReportedArtifact(tx, store, source)
	}
	if !scope.managed {
		return store.createArtifactWithExecutor(tx.Exec, source)
	}
	key, err := computeArtifactIdentityKey(source)
	if err != nil {
		return nil, err
	}
	identity, err := json.Marshal([]any{"runtime-report-artifact", scope.reporter.UUID, scope.attempt, link.TaskID, link.Type, link.Iteration, link.ArtifactKey, key, source.NumberValue})
	if err != nil {
		return nil, err
	}
	artifact := *source
	artifact.UUID = util.NewDeterministicUUID(string(identity))
	artifact.CreatedAtInSec = s.time.Now().Unix()
	artifact.LastUpdateInSec = artifact.CreatedAtInSec
	if err := (&runtimeStore{tasks: s}).checkpointArtifact(tx, artifact.Namespace, artifact.UUID, &artifact); err != nil {
		return nil, err
	}
	return s.getReportArtifact(tx, artifact.UUID)
}

func (s *TaskStore) reuseReportedArtifact(tx sqlTransaction, store *ArtifactStore, source *model.Artifact) (*model.Artifact, error) {
	key, err := computeArtifactIdentityKey(source)
	if err != nil {
		return nil, err
	}
	q, qb := s.dbDialect.QuoteIdentifier, s.dbDialect.QueryBuilder()
	find := func(lock bool) (*model.Artifact, error) {
		identity := sq.Or{sq.Eq{q("IdentityKey"): key}}
		if source.URI != nil && *source.URI != "" {
			identity = append(identity, sq.Eq{q("URIHash"): artifactURIHash(*source.URI), q("URI"): *source.URI})
		}
		query, args, err := qb.Select(dialect.QuoteAll(q, artifactColumns)...).From(q("artifacts")).
			Where(sq.Eq{q("Namespace"): source.Namespace}).Where(identity).ToSql()
		if err != nil {
			return nil, err
		}
		if lock {
			query = s.dbDialect.SelectForUpdate(query)
		}
		rows, err := tx.Query(query, args...)
		if err != nil {
			return nil, err
		}
		candidates, err := store.scanRows(rows)
		closeErr := rows.Close()
		if err != nil {
			return nil, err
		}
		if closeErr != nil {
			return nil, closeErr
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
		for _, candidate := range candidates {
			if modelArtifactsEqualForReuse(candidate, source) {
				return candidate, nil
			}
		}
		return nil, nil
	}
	existing, err := find(false)
	if err != nil || existing != nil {
		return existing, err
	}
	if _, err := tx.Exec("SAVEPOINT runtime_report_artifact"); err != nil {
		return nil, err
	}
	artifact := *source
	artifact.IdentityKey = &key
	created, err := store.createArtifactWithExecutor(tx.Exec, &artifact)
	if err != nil {
		if _, rollbackErr := tx.Exec("ROLLBACK TO SAVEPOINT runtime_report_artifact"); rollbackErr != nil {
			return nil, errors.Join(err, rollbackErr)
		}
		if s.dbDialect.IsDuplicateKeyError(err) {
			created, err = find(true)
			if err == nil && created == nil {
				err = errors.New("artifact reuse identity collision did not resolve to matching metadata")
			}
		}
	}
	if _, releaseErr := tx.Exec("RELEASE SAVEPOINT runtime_report_artifact"); releaseErr != nil {
		return nil, errors.Join(err, releaseErr)
	}
	return created, err
}
