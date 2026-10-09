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

type recurringRunAdopter interface {
	PrepareRecurringRunStartupRepairs(context.Context, string, uint64) ([]storage.RecurringRunMigrationCandidate, error)
	RequireRecurringRunAdoptionReady(context.Context, string) error
	AdoptLegacyRecurringRun(context.Context, string) error
	ListLegacyRecurringRunAdoptionCandidates(context.Context, string, uint64) ([]storage.RecurringRunMigrationCandidate, error)
}

// AutomaticRecurringRunAdoption recovers schedules during ordinary API startup.
// Neither API availability nor recovery of other records depends on a bad record.
type AutomaticRecurringRunAdoption struct {
	adopter      recurringRunAdopter
	writersReady func(context.Context) error
	cursor       string
	startupDone  bool
	retryMu      sync.Mutex
	retries      map[string]*recurringRunRetry
	now          func() time.Time
}

func NewAutomaticRecurringRunAdoption(adopter recurringRunAdopter, writersReady func(context.Context) error) *AutomaticRecurringRunAdoption {
	return &AutomaticRecurringRunAdoption{adopter: adopter, writersReady: writersReady, retries: make(map[string]*recurringRunRetry), now: time.Now}
}

const recurringRunRetryLimit = 4096
const recurringRunRetryInitial = 15 * time.Second
const recurringRunRetryMaximum = 5 * time.Minute

type recurringRunRetry struct {
	next    time.Time
	last    time.Time
	delay   time.Duration
	running bool
	err     error
}

// beginRetry serializes expensive per-record work across lazy and background
// callers. Durable receipts retain repair work if the bounded cache evicts an
// idle entry. Cache pressure never permits duplicate in-flight work.
func (a *AutomaticRecurringRunAdoption) beginRetry(id string) (bool, error) {
	a.retryMu.Lock()
	defer a.retryMu.Unlock()
	now := a.now()
	if retry := a.retries[id]; retry != nil {
		if retry.running {
			return false, util.NewUnavailableServerError(fmt.Errorf("schedule reconciliation is already running"), "Retry this recurring run shortly")
		}
		if now.Before(retry.next) {
			return false, retry.err
		}
		retry.running = true
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
	a.retries[id] = &recurringRunRetry{running: true, last: now}
	return true, nil
}
func (a *AutomaticRecurringRunAdoption) finishRetry(id string, err error) {
	a.retryMu.Lock()
	defer a.retryMu.Unlock()
	retry := a.retries[id]
	if retry == nil {
		return
	}
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

// Ensure leaves healthy schedules alone. Failed expensive attempts share a
// capped backoff across both callers; another replica's recovery clears it.
func (a *AutomaticRecurringRunAdoption) Ensure(ctx context.Context, id string) (result error) {
	if err := a.adopter.RequireRecurringRunAdoptionReady(ctx, id); err == nil {
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
	if ok, err := a.beginRetry(id); !ok {
		return err
	}
	defer func() {
		// Cancellation belongs to this caller, never to a subsequent request.
		if status.Code(result) == codes.Canceled || status.Code(result) == codes.DeadlineExceeded || ctx.Err() != nil || errors.Is(result, context.Canceled) || errors.Is(result, context.DeadlineExceeded) || util.IsUserErrorCodeMatch(result, codes.Canceled) || util.IsUserErrorCodeMatch(result, codes.DeadlineExceeded) {
			a.finishRetry(id, nil)
		} else {
			a.finishRetry(id, result)
		}
	}()
	if err := a.adopter.AdoptLegacyRecurringRun(ctx, id); err != nil {
		return err
	}
	return a.adopter.RequireRecurringRunAdoptionReady(ctx, id)
}

// scan processes a bounded page, including disabled schedules. Advancing past
// failures prevents a malformed early record from starving later records.
func (a *AutomaticRecurringRunAdoption) scan(ctx context.Context) error {
	if err := a.writersReady(ctx); err != nil {
		return err
	}
	const pageSize = 100
	var candidates []storage.RecurringRunMigrationCandidate
	var err error
	if !a.startupDone {
		candidates, err = a.adopter.PrepareRecurringRunStartupRepairs(ctx, a.cursor, pageSize)
	} else {
		candidates, err = a.adopter.ListLegacyRecurringRunAdoptionCandidates(ctx, a.cursor, pageSize)
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

// Run is called in a background goroutine; listeners never wait for adoption.
func (a *AutomaticRecurringRunAdoption) Run(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		scanCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		err := a.scan(scanCtx)
		cancel()
		if err != nil && ctx.Err() == nil {
			glog.Warningf("Automatic recurring-run adoption is waiting and will retry: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
