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
	"context"
	"database/sql"
	"errors"

	sq "github.com/Masterminds/squirrel"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
)

// LeaseScopedStores binds immutable ownership to a per-run coordinator copy.
// Reads retain their ordinary behavior. Every mutation validates this token in
// its own transaction; launcher reports remain independently attempt-fenced.
type LeaseScopedStores struct {
	Runs          *RunStore
	Tasks         *TaskStore
	Artifacts     *ArtifactStore
	ArtifactTasks *ArtifactTaskStore
	Runtime       RuntimeStoreInterface
}

type leaseFence struct {
	db    *sql.DB
	d     dialect.DBDialect
	token RunLeaseToken
}

// sqlTransaction lets leased transactions propagate a bounded context to every
// query without changing unscoped *sql.Tx behavior or holding locks across I/O.
type sqlTransaction interface {
	taskSQLExecutor
	artifactSQLQuerier
	Commit() error
	Rollback() error
}

type leasedTransaction struct {
	tx            *sql.Tx
	ctx           context.Context
	cancel        context.CancelFunc
	fence         *leaseFence
	expiresAt     int64
	allowDeletion bool
}

func (tx *leasedTransaction) Exec(query string, args ...any) (sql.Result, error) {
	return tx.tx.ExecContext(tx.ctx, query, args...)
}

func (tx *leasedTransaction) Query(query string, args ...any) (*sql.Rows, error) {
	return tx.tx.QueryContext(tx.ctx, query, args...)
}

func (tx *leasedTransaction) QueryRow(query string, args ...any) *sql.Row {
	return tx.tx.QueryRowContext(tx.ctx, query, args...)
}

func (tx *leasedTransaction) Rollback() error {
	defer tx.cancel()
	return tx.tx.Rollback()
}

func (tx *leasedTransaction) Commit() error {
	defer tx.cancel()
	// The lease row remains locked until commit. Recheck DB time after all
	// writes, including deletions that cascade the lease row away.
	if tx.allowDeletion {
		var now int64
		if err := tx.QueryRow("SELECT " + leaseDatabaseNow(tx.fence.d)).Scan(&now); err != nil {
			_ = tx.tx.Rollback()
			return err
		}
		if now >= tx.expiresAt {
			_ = tx.tx.Rollback()
			return ErrRunLeaseConflict
		}
	} else if err := tx.fence.store().check(tx.ctx, tx.tx, tx.fence.token, false); err != nil {
		_ = tx.tx.Rollback()
		return err
	}
	return tx.tx.Commit()
}

func (f *leaseFence) store() *runLeaseStore { return &runLeaseStore{db: f.db, d: f.d} }

func (f *leaseFence) begin() (sqlTransaction, error) {
	ctx, cancel := context.WithTimeout(context.Background(), runLeaseOperationTimeout)
	tx, err := f.db.BeginTx(ctx, nil)
	if err != nil {
		cancel()
		return nil, err
	}
	fail := func(err error) (sqlTransaction, error) {
		_ = tx.Rollback()
		cancel()
		return nil, err
	}
	q, qb := f.d.QuoteIdentifier, f.d.QueryBuilder()
	query, args, err := qb.Select(q("ExecutorType")).From(q("run_details")).
		Where(sq.Eq{q("UUID"): f.token.RunUUID}).ToSql()
	if err != nil {
		return fail(err)
	}
	var executor sql.NullString
	if err := tx.QueryRowContext(ctx, f.d.SelectForUpdate(query), args...).Scan(&executor); errors.Is(err, sql.ErrNoRows) {
		return fail(ErrRunLeaseConflict)
	} else if err != nil {
		return fail(err)
	}
	if !executor.Valid || executor.String == "" {
		return fail(ErrRunLeaseConflict)
	}
	query, args, err = qb.Select(q("OwnerID"), q("Generation"), q("ExpiresAt")).
		From(q("run_leases")).Where(sq.Eq{q("RunUUID"): f.token.RunUUID}).ToSql()
	if err != nil {
		return fail(err)
	}
	var owner sql.NullString
	var generation int64
	var expiry sql.NullInt64
	if err := tx.QueryRowContext(ctx, f.d.SelectForUpdate(query), args...).Scan(&owner, &generation, &expiry); errors.Is(err, sql.ErrNoRows) {
		return fail(ErrRunLeaseConflict)
	} else if err != nil {
		return fail(err)
	}
	var now int64
	if err := tx.QueryRowContext(ctx, "SELECT "+leaseDatabaseNow(f.d)).Scan(&now); err != nil {
		return fail(err)
	}
	if !owner.Valid || owner.String != f.token.OwnerID || generation != f.token.Generation || !expiry.Valid || expiry.Int64 <= now {
		return fail(ErrRunLeaseConflict)
	}
	return &leasedTransaction{tx: tx, ctx: ctx, cancel: cancel, fence: f, expiresAt: expiry.Int64}, nil
}

func beginStoreMutation(db *sql.DB, fence *leaseFence) (sqlTransaction, error) {
	if fence == nil {
		return db.Begin()
	}
	return fence.begin()
}

func checkLeaseRun(fence *leaseFence, runID string) error {
	if fence != nil && runID != fence.token.RunUUID {
		return ErrRunLeaseScope
	}
	return nil
}

func requireLeaseTransaction(fence *leaseFence, tx sqlTransaction) error {
	if fence == nil {
		return nil
	}
	guarded, ok := tx.(*leasedTransaction)
	if !ok || guarded.fence.token != fence.token {
		return ErrRunLeaseConflict
	}
	return nil
}

func checkLeaseTasks(tx sqlTransaction, fence *leaseFence, ids []string) error {
	if fence == nil {
		return nil
	}
	if err := requireLeaseTransaction(fence, tx); err != nil {
		return err
	}
	q := fence.d.QuoteIdentifier
	for _, id := range ids {
		query, args, err := fence.d.QueryBuilder().Select(q("RunUUID")).From(q("tasks")).
			Where(sq.Eq{q("UUID"): id}).ToSql()
		if err != nil {
			return err
		}
		var runID string
		if err := tx.QueryRow(fence.d.SelectForUpdate(query), args...).Scan(&runID); err != nil {
			return err
		}
		if err := checkLeaseRun(fence, runID); err != nil {
			return err
		}
	}
	return nil
}

func checkLeaseLinks(tx sqlTransaction, fence *leaseFence, links []*model.ArtifactTask) error {
	if fence == nil {
		return nil
	}
	for _, link := range links {
		if link == nil || link.RunUUID != fence.token.RunUUID {
			return ErrRunLeaseScope
		}
		if err := checkLeaseTasks(tx, fence, []string{link.TaskID}); err != nil {
			return err
		}
	}
	return nil
}

func execTaskLinkedMutation(db *sql.DB, fence *leaseFence, ids []string, query string, args ...any) (sql.Result, error) {
	if fence == nil {
		return db.Exec(query, args...)
	}
	tx, err := fence.begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if err := checkLeaseTasks(tx, fence, ids); err != nil {
		return nil, err
	}
	result, err := tx.Exec(query, args...)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}

func execStoreMutation(db *sql.DB, fence *leaseFence, query string, args ...any) (sql.Result, error) {
	if fence == nil {
		return db.Exec(query, args...)
	}
	tx, err := fence.begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	result, err := tx.Exec(query, args...)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}

// LeaseScopedStores creates views without checking host time or mutable global
// owner maps. Validation occurs again, transactionally, at every mutation.
func (s *TaskStore) LeaseScopedStores(token RunLeaseToken) (*LeaseScopedStores, error) {
	if err := validateLeaseToken(token); err != nil {
		return nil, err
	}
	fence := &leaseFence{db: s.db, d: s.dbDialect, token: token}
	tasks := *s
	tasks.lease = fence
	runs := NewRunStore(s.db, s.time, s.dbDialect)
	runs.lease = fence
	runs.taskStore = &tasks
	artifacts := NewArtifactStore(s.db, s.time, s.uuid, s.dbDialect)
	artifacts.lease = fence
	links := NewArtifactTaskStore(s.db, s.uuid, s.dbDialect)
	links.lease = fence
	return &LeaseScopedStores{
		Runs: runs, Tasks: &tasks, Artifacts: artifacts, ArtifactTasks: links, Runtime: tasks.RuntimeStore(),
	}, nil
}

func (s *ArtifactStore) findOrCreateLeasedArtifact(artifact *model.Artifact, link *model.ArtifactTask) (*model.Artifact, *model.ArtifactTask, error) {
	tx, err := beginStoreMutation(s.db, s.lease)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback()
	if err := checkLeaseLinks(tx, s.lease, []*model.ArtifactTask{link}); err != nil {
		return nil, nil, err
	}
	tasks := NewTaskStore(s.db, s.time, s.uuid, s.dbDialect)
	created, err := tasks.reuseReportedArtifact(tx, s, artifact)
	if err != nil {
		return nil, nil, err
	}
	candidate := *link
	candidate.ArtifactID = created.UUID
	if err := candidate.SyncIterationFromProducer(); err != nil {
		return nil, nil, err
	}
	storedLink, err := s.getArtifactTaskByUniqueLinkWithQuerier(tx, &candidate)
	if err != nil {
		return nil, nil, err
	}
	if storedLink == nil {
		storedLink, err = s.createArtifactTaskInTransaction(tx, &candidate)
		if err != nil {
			return nil, nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, nil, err
	}
	return created, storedLink, nil
}
