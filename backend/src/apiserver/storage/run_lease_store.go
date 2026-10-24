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
	"fmt"
	"math"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
)

var (
	// ErrRunLeaseConflict means ownership is live elsewhere, expired, or stale.
	ErrRunLeaseConflict = errors.New("run lease is held, expired, released, or replaced; reload ownership")
	// ErrRunLeaseIneligible rejects execution claims for unmanaged/closed runs.
	ErrRunLeaseIneligible = errors.New("run is not eligible for this managed execution lease")
	// ErrRunLeaseScope rejects cross-run or global mutations through a bound view.
	ErrRunLeaseScope = errors.New("lease-scoped store cannot mutate outside its owning run")
)

const runLeaseOperationTimeout = 5 * time.Second

// RunLeaseToken is immutable write authority, not a worker-clock deadline.
type RunLeaseToken struct {
	RunUUID    string
	OwnerID    string
	Generation int64
}

// RunLeaseStoreInterface uses database time for every ownership decision.
type RunLeaseStoreInterface interface {
	Acquire(runID, ownerID string, ttl time.Duration, cleanup bool) (*RunLeaseToken, error)
	Renew(token RunLeaseToken, ttl time.Duration) error
	Check(token RunLeaseToken) error
	Release(token RunLeaseToken) error
	ListEligible(executor string, limit int, preferRetry bool) ([]string, error)
}

type runLeaseStore struct {
	db *sql.DB
	d  dialect.DBDialect
}

var _ RunLeaseStoreInterface = (*runLeaseStore)(nil)

// RunLeaseStore exposes durable ownership using this task store's DB/dialect.
func (s *TaskStore) RunLeaseStore() RunLeaseStoreInterface {
	return &runLeaseStore{db: s.db, d: s.dbDialect}
}

func leaseDatabaseNow(d dialect.DBDialect) string {
	switch d.Name() {
	case "mysql":
		return "CAST(UNIX_TIMESTAMP(CURRENT_TIMESTAMP(3)) * 1000 AS SIGNED)"
	case "pgx":
		return "CAST(FLOOR(EXTRACT(EPOCH FROM clock_timestamp()) * 1000) AS BIGINT)"
	case "sqlite":
		return "CAST((julianday('now') - 2440587.5) * 86400000 AS INTEGER)"
	default:
		panic("unsupported lease database dialect: " + d.Name())
	}
}

func validateLeaseToken(token RunLeaseToken) error {
	if token.RunUUID == "" || strings.TrimSpace(token.OwnerID) == "" || token.Generation <= 0 ||
		len(token.RunUUID) > 191 || len(token.OwnerID) > 191 {
		return errors.New("run lease requires nonempty run/owner IDs up to 191 bytes and a positive generation")
	}
	return nil
}

func leaseTTL(ttl time.Duration) (int64, error) {
	millis := ttl.Milliseconds()
	if millis <= 0 {
		return 0, errors.New("run lease TTL must be at least one millisecond")
	}
	return millis, nil
}

func (s *runLeaseStore) Acquire(runID, ownerID string, ttl time.Duration, cleanup bool) (*RunLeaseToken, error) {
	if err := validateLeaseToken(RunLeaseToken{RunUUID: runID, OwnerID: ownerID, Generation: 1}); err != nil {
		return nil, err
	}
	millis, err := leaseTTL(ttl)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), runLeaseOperationTimeout)
	defer cancel()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin run lease acquisition: %w", err)
	}
	defer tx.Rollback()
	q, qb := s.d.QuoteIdentifier, s.d.QueryBuilder()
	query, args, err := qb.Select(q("ExecutorType"), q("State"), q("Conditions")).
		From(q("run_details")).Where(sq.Eq{q("UUID"): runID}).ToSql()
	if err != nil {
		return nil, err
	}
	var executor, state, conditions sql.NullString
	if err := tx.QueryRowContext(ctx, s.d.SelectForUpdate(query), args...).Scan(&executor, &state, &conditions); errors.Is(err, sql.ErrNoRows) {
		return nil, ErrRunLeaseIneligible
	} else if err != nil {
		return nil, fmt.Errorf("lock run for lease acquisition: %w", err)
	}
	effective := model.RuntimeState(state.String).ToV2()
	if effective == model.RuntimeStateUnspecified {
		effective = model.RuntimeState(conditions.String).ToV2()
	}
	// Archival affects visibility, not execution or recovery eligibility.
	if !executor.Valid || executor.String == "" || (!cleanup &&
		(effective != model.RuntimeStatePending && effective != model.RuntimeStateRunning)) {
		return nil, ErrRunLeaseIneligible
	}
	// Touch only the identical primary key on conflict; never reset ownership
	// or generation. PostgreSQL requires a qualified excluded-column RHS.
	query, args, err = s.d.Upsert("run_leases", []string{"RunUUID"}, true, []string{"RunUUID"}).
		SetMap(sq.Eq{q("RunUUID"): runID, q("OwnerID"): nil, q("Generation"): 0, q("ExpiresAt"): nil}).ToSql()
	if err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return nil, fmt.Errorf("initialize run lease generation: %w", err)
	}
	now := leaseDatabaseNow(s.d)
	query, args, err = qb.Update(q("run_leases")).SetMap(sq.Eq{
		q("OwnerID"): ownerID, q("Generation"): sq.Expr(q("Generation") + " + 1"),
		q("ExpiresAt"): sq.Expr(now+" + ?", millis),
	}).Where(sq.Eq{q("RunUUID"): runID}).
		Where(sq.Lt{q("Generation"): int64(math.MaxInt64)}).
		Where(sq.Or{sq.Eq{q("OwnerID"): nil}, sq.Eq{q("ExpiresAt"): nil}, sq.Expr(q("ExpiresAt") + " <= " + now)}).ToSql()
	if err != nil {
		return nil, err
	}
	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("acquire expired or released run lease: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rows != 1 {
		return nil, ErrRunLeaseConflict
	}
	query, args, err = qb.Select(q("Generation")).From(q("run_leases")).Where(sq.Eq{q("RunUUID"): runID}).ToSql()
	if err != nil {
		return nil, err
	}
	token := &RunLeaseToken{RunUUID: runID, OwnerID: ownerID}
	if err := tx.QueryRowContext(ctx, query, args...).Scan(&token.Generation); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit run lease acquisition: %w", err)
	}
	return token, nil
}

func (s *runLeaseStore) tokenCondition(token RunLeaseToken) sq.And {
	q := s.d.QuoteIdentifier
	return sq.And{
		sq.Eq{q("RunUUID"): token.RunUUID, q("OwnerID"): token.OwnerID, q("Generation"): token.Generation},
		sq.Expr(q("ExpiresAt") + " > " + leaseDatabaseNow(s.d)),
	}
}

func (s *runLeaseStore) Renew(token RunLeaseToken, ttl time.Duration) error {
	if err := validateLeaseToken(token); err != nil {
		return err
	}
	millis, err := leaseTTL(ttl)
	if err != nil {
		return err
	}
	q := s.d.QuoteIdentifier
	query, args, err := s.d.QueryBuilder().Update(q("run_leases")).
		SetMap(sq.Eq{q("ExpiresAt"): sq.Expr(leaseDatabaseNow(s.d)+" + ?", millis)}).
		Where(s.tokenCondition(token)).ToSql()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), runLeaseOperationTimeout)
	defer cancel()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := s.lockOwned(ctx, tx, token); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("renew current unexpired run lease: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		// MySQL may report no changed rows for a same-millisecond renewal.
		if err := s.check(ctx, tx, token, false); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *runLeaseStore) Check(token RunLeaseToken) error {
	if err := validateLeaseToken(token); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), runLeaseOperationTimeout)
	defer cancel()
	return s.check(ctx, s.db, token, false)
}

type leaseRowQuerier interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func (s *runLeaseStore) check(ctx context.Context, db leaseRowQuerier, token RunLeaseToken, lock bool) error {
	q := s.d.QuoteIdentifier
	query, args, err := s.d.QueryBuilder().Select(q("Generation")).From(q("run_leases")).
		Where(s.tokenCondition(token)).ToSql()
	if err != nil {
		return err
	}
	if lock {
		query = s.d.SelectForUpdate(query)
	}
	var generation int64
	if err := db.QueryRowContext(ctx, query, args...).Scan(&generation); errors.Is(err, sql.ErrNoRows) {
		return ErrRunLeaseConflict
	} else if err != nil {
		return fmt.Errorf("check current database-time run lease: %w", err)
	}
	return nil
}

func (s *runLeaseStore) Release(token RunLeaseToken) error {
	if err := validateLeaseToken(token); err != nil {
		return err
	}
	q := s.d.QuoteIdentifier
	query, args, err := s.d.QueryBuilder().Update(q("run_leases")).
		SetMap(sq.Eq{q("OwnerID"): nil, q("ExpiresAt"): nil}).
		Where(s.tokenCondition(token)).ToSql()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), runLeaseOperationTimeout)
	defer cancel()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := s.lockOwned(ctx, tx, token); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("release current run lease while retaining generation: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return ErrRunLeaseConflict
	}
	return tx.Commit()
}

// lockOwned checks time only after the row lock is acquired. Checking expiry in
// a SELECT ... FOR UPDATE predicate alone can use a clock sampled before a wait.
func (s *runLeaseStore) lockOwned(ctx context.Context, tx *sql.Tx, token RunLeaseToken) error {
	q := s.d.QuoteIdentifier
	query, args, err := s.d.QueryBuilder().Select(q("OwnerID"), q("Generation"), q("ExpiresAt")).
		From(q("run_leases")).Where(sq.Eq{q("RunUUID"): token.RunUUID}).ToSql()
	if err != nil {
		return err
	}
	var owner sql.NullString
	var generation int64
	var expiry sql.NullInt64
	if err := tx.QueryRowContext(ctx, s.d.SelectForUpdate(query), args...).Scan(&owner, &generation, &expiry); errors.Is(err, sql.ErrNoRows) {
		return ErrRunLeaseConflict
	} else if err != nil {
		return fmt.Errorf("lock current run lease: %w", err)
	}
	var now int64
	if err := tx.QueryRowContext(ctx, "SELECT "+leaseDatabaseNow(s.d)).Scan(&now); err != nil {
		return err
	}
	if !owner.Valid || owner.String != token.OwnerID || generation != token.Generation || !expiry.Valid || expiry.Int64 <= now {
		return ErrRunLeaseConflict
	}
	return nil
}

// ListEligible is bounded discovery, not a claim. Acquire performs the final
// DB-time CAS. Cancellation precedes active recovery; callers periodically turn
// retry preference off so retry-ready work cannot indefinitely starve fresh runs.
func (s *runLeaseStore) ListEligible(executor string, limit int, preferRetry bool) ([]string, error) {
	if executor == "" || limit < 1 || limit > 1000 {
		return nil, errors.New("lease discovery requires an executor and a limit between 1 and 1000")
	}
	q := s.d.QuoteIdentifier
	rd, rl, rt := q("rd"), q("rl"), q("rt")
	runID := rd + "." + q("UUID")
	taskRun := rt + "." + q("RunUUID")
	state := rt + "." + q("State")
	next := rt + "." + q("NextEligibleAt")
	now := leaseDatabaseNow(s.d)
	seconds := "(" + now + " / 1000)"
	ready := "(" + state + " <> '" + model.RuntimeTaskBackoff + "' OR " + next + " IS NULL OR " + next + " <= " + seconds + ")"
	taskFrom := " FROM " + q("runtime_tasks") + " AS " + rt + " WHERE " + taskRun + " = " + runID
	existsAny := "EXISTS (SELECT 1" + taskFrom + ")"
	existsReady := "EXISTS (SELECT 1" + taskFrom + " AND " + ready + ")"
	cancellation := rd + "." + q("State") + " = '" + string(model.RuntimeStateCancelling) + "'"
	recovery := "EXISTS (SELECT 1" + taskFrom + " AND " + state + " IN ('" +
		strings.Join([]string{model.RuntimeTaskSubmitting, model.RuntimeTaskRunning, model.RuntimeTaskPublishing,
			model.RuntimeTaskFailed, model.RuntimeTaskFinalizing, model.RuntimeTaskFailureFinalizing,
			model.RuntimeTaskCanceling, model.RuntimeTaskCancelFinalizing}, "','") + "'))"
	retry := "EXISTS (SELECT 1" + taskFrom + " AND (" + rt + "." + q("RetryCount") +
		" > 0 OR " + state + " = '" + model.RuntimeTaskBackoff + "') AND " + ready + ")"
	priority := "CASE WHEN " + cancellation + " THEN 0 WHEN " + recovery + " THEN 1 WHEN " + retry + " THEN 3 ELSE 2 END"
	if preferRetry {
		priority = "CASE WHEN " + cancellation + " THEN 0 WHEN " + recovery + " THEN 1 WHEN " + retry + " THEN 2 ELSE 3 END"
	}
	query, args, err := s.d.QueryBuilder().Select(runID).
		From(q("run_details")+" AS "+rd).
		LeftJoin(q("run_leases")+" AS "+rl+" ON "+rl+"."+q("RunUUID")+" = "+runID).
		Where(sq.Eq{rd + "." + q("ExecutorType"): executor,
			rd + "." + q("State"): []string{string(model.RuntimeStatePending), string(model.RuntimeStateRunning), string(model.RuntimeStateCancelling)}}).
		Where(sq.Or{sq.Eq{rl + "." + q("OwnerID"): nil}, sq.Eq{rl + "." + q("ExpiresAt"): nil},
			sq.Expr(rl + "." + q("ExpiresAt") + " <= " + now)}).
		Where(sq.Or{sq.Eq{rd + "." + q("State"): string(model.RuntimeStateCancelling)},
			sq.Expr("NOT " + existsAny), sq.Expr(existsReady)}).
		OrderBy(priority, rd+"."+q("CreatedAtInSec")+" ASC", runID+" ASC").Limit(uint64(limit)).ToSql()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), runLeaseOperationTimeout)
	defer cancel()
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("discover database-time eligible managed runs: %w", err)
	}
	defer rows.Close()
	ids := make([]string, 0, limit)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ids, nil
}
