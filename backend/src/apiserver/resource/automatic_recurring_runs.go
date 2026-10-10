// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0

package resource

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

type recurringRunSynchronizer interface {
	PrepareRecurringRunStartupRepairs(context.Context, string, uint64) ([]storage.RecurringRunMigrationCandidate, error)
	RequireRecurringRunAdoptionReady(context.Context, string) error
	SynchronizeRecurringRun(context.Context, string) error
	ListPendingRecurringRunSynchronizations(context.Context, string, uint64) ([]storage.RecurringRunMigrationCandidate, error)
}

// AutomaticRecurringRunSynchronization recovers schedules during ordinary API startup.
// Neither API availability nor recovery of other records depends on a bad record.
type AutomaticRecurringRunSynchronization struct {
	synchronizer recurringRunSynchronizer
	writersReady func(context.Context) error
	cursor       string
	startupDone  bool
	retryMu      sync.Mutex
	retries      map[string]*recurringRunRetry
	now          func() time.Time
}

func NewAutomaticRecurringRunSynchronization(synchronizer recurringRunSynchronizer, writersReady func(context.Context) error) *AutomaticRecurringRunSynchronization {
	return &AutomaticRecurringRunSynchronization{synchronizer: synchronizer, writersReady: writersReady, retries: make(map[string]*recurringRunRetry), now: time.Now}
}

const recurringRunRetryLimit = 4096
const recurringRunRetryInitial = 15 * time.Second
const recurringRunRetryMaximum = 5 * time.Minute

type recurringRunRetry struct {
	next    time.Time
	last    time.Time
	delay   time.Duration
	running bool
	done    chan struct{}
	err     error
}

// beginRetry serializes expensive per-record work across lazy and background
// callers. Durable receipts retain repair work if the bounded cache evicts an
// idle entry. Cache pressure never permits duplicate in-flight work.
func (a *AutomaticRecurringRunSynchronization) beginRetry(id string) (bool, error) {
	a.retryMu.Lock()
	defer a.retryMu.Unlock()
	return a.beginRetryLocked(id)
}

func (a *AutomaticRecurringRunSynchronization) beginRetryLocked(id string) (bool, error) {
	now := a.now()
	if retry := a.retries[id]; retry != nil {
		if retry.running {
			return false, util.NewUnavailableServerError(fmt.Errorf("schedule reconciliation is already running"), "Retry this recurring run shortly")
		}
		if now.Before(retry.next) {
			return false, retry.err
		}
		retry.running = true
		retry.done = make(chan struct{})
		retry.last = now
		return true, nil
	}
	if len(a.retries) >= recurringRunRetryLimit {
		oldestID := ""
		var oldest time.Time
		for candidate, retry := range a.retries {
			if !retry.running && (oldestID == "" || retry.last.Before(oldest)) {
				oldestID = candidate
				oldest = retry.last
			}
		}
		if oldestID == "" {
			return false, util.NewUnavailableServerError(fmt.Errorf("schedule reconciliation is busy"), "Retry this recurring run shortly")
		}
		delete(a.retries, oldestID)
	}
	a.retries[id] = &recurringRunRetry{running: true, last: now, done: make(chan struct{})}
	return true, nil
}
func (a *AutomaticRecurringRunSynchronization) finishRetry(id string, err error) {
	a.retryMu.Lock()
	defer a.retryMu.Unlock()
	retry := a.retries[id]
	if retry == nil {
		return
	}
	close(retry.done)
	if err == nil {
		delete(a.retries, id)
		return
	}
	glog.Warningf("Recurring-run reconciliation will retry schedule %s: %v", id, err)
	retry.running = false
	retry.err = err
	if retry.delay == 0 {
		retry.delay = recurringRunRetryInitial
	} else {
		retry.delay = min(2*retry.delay, recurringRunRetryMaximum)
	}
	retry.last = a.now()
	retry.next = retry.last.Add(retry.delay)
}

// beginModeChangeRetry discards errors for the previous desired state, but waits
// for its in-flight attempt before starting work for the new state.
func (a *AutomaticRecurringRunSynchronization) beginModeChangeRetry(ctx context.Context, id string) (bool, error) {
	for {
		a.retryMu.Lock()
		if err := ctx.Err(); err != nil {
			a.retryMu.Unlock()
			return false, err
		}
		retry := a.retries[id]
		if retry == nil || !retry.running {
			delete(a.retries, id)
			ok, err := a.beginRetryLocked(id)
			a.retryMu.Unlock()
			return ok, err
		}
		done := retry.done
		a.retryMu.Unlock()
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-done:
		}
	}
}

// EnsureAfterModeChange reconciles newly persisted desired state without replaying
// a previous mode's cached failure or overlapping an existing attempt.
func (a *AutomaticRecurringRunSynchronization) EnsureAfterModeChange(ctx context.Context, id string) error {
	return a.ensure(ctx, id, true)
}

// Ensure leaves healthy schedules alone. Failed expensive attempts share a
// capped backoff across both callers; another replica's recovery clears it.
func (a *AutomaticRecurringRunSynchronization) Ensure(ctx context.Context, id string) error {
	return a.ensure(ctx, id, false)
}

func (a *AutomaticRecurringRunSynchronization) ensure(ctx context.Context, id string, modeChanged bool) (result error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := a.synchronizer.RequireRecurringRunAdoptionReady(ctx, id); err == nil {
		a.retryMu.Lock()
		if retry := a.retries[id]; retry != nil && !retry.running {
			delete(a.retries, id)
		}
		a.retryMu.Unlock()
		return nil
	}
	if err := a.writersReady(ctx); err != nil {
		return util.NewUnavailableServerError(err, "Scheduling is waiting for the automatic upgrade handoff; retry shortly")
	}
	var ok bool
	var err error
	if modeChanged {
		ok, err = a.beginModeChangeRetry(ctx, id)
	} else {
		ok, err = a.beginRetry(id)
	}
	if !ok {
		return err
	}
	defer func() {
		// Retain backoff after interrupted work, but do not replay one caller's
		// cancellation or deadline status to later callers.
		if result != nil && (status.Code(result) == codes.Canceled || status.Code(result) == codes.DeadlineExceeded || ctx.Err() != nil || errors.Is(result, context.Canceled) || errors.Is(result, context.DeadlineExceeded) || util.IsUserErrorCodeMatch(result, codes.Canceled) || util.IsUserErrorCodeMatch(result, codes.DeadlineExceeded)) {
			a.finishRetry(id, util.NewUnavailableServerError(fmt.Errorf("previous schedule reconciliation was interrupted"), "Retry this recurring run after reconciliation backoff"))
		} else {
			a.finishRetry(id, result)
		}
	}()
	if err := a.synchronizer.SynchronizeRecurringRun(ctx, id); err != nil {
		return err
	}
	return a.synchronizer.RequireRecurringRunAdoptionReady(ctx, id)
}

// scan processes a bounded page, including disabled schedules. Advancing past
// failures prevents a malformed early record from starving later records.
func (a *AutomaticRecurringRunSynchronization) scan(ctx context.Context) error {
	if err := a.writersReady(ctx); err != nil {
		return err
	}
	const pageSize = 100
	var candidates []storage.RecurringRunMigrationCandidate
	var err error
	if !a.startupDone {
		candidates, err = a.synchronizer.PrepareRecurringRunStartupRepairs(ctx, a.cursor, pageSize)
	} else {
		candidates, err = a.synchronizer.ListPendingRecurringRunSynchronizations(ctx, a.cursor, pageSize)
	}
	if err != nil {
		return err
	}
	for _, candidate := range candidates {
		if err := ctx.Err(); err != nil {
			return err
		}
		recordCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		_ = a.Ensure(recordCtx, candidate.ID)
		cancel()
		a.cursor = candidate.ID
	}
	if len(candidates) < pageSize {
		a.startupDone = true
		a.cursor = ""
	}
	return nil
}

// Run is called in a background goroutine; listeners never wait for synchronization.
func (a *AutomaticRecurringRunSynchronization) Run(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		scanCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		err := a.scan(scanCtx)
		cancel()
		if err != nil && ctx.Err() == nil {
			glog.Warningf("Automatic recurring-run synchronization is waiting and will retry: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
