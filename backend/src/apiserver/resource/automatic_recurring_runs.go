// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0

package resource

import (
	"context"
	"time"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

type recurringRunAdopter interface {
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
}

func NewAutomaticRecurringRunAdoption(adopter recurringRunAdopter, writersReady func(context.Context) error) *AutomaticRecurringRunAdoption {
	return &AutomaticRecurringRunAdoption{adopter: adopter, writersReady: writersReady}
}

// Ensure leaves healthy schedules alone, and retries legacy records only after
// old API/controller writers have left the managed rollout.
func (a *AutomaticRecurringRunAdoption) Ensure(ctx context.Context, id string) error {
	if err := a.adopter.RequireRecurringRunAdoptionReady(ctx, id); err == nil {
		return nil
	}
	if err := a.writersReady(ctx); err != nil {
		return util.NewUnavailableServerError(err, "Scheduling is waiting for the automatic upgrade handoff; retry shortly")
	}
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
	candidates, err := a.adopter.ListLegacyRecurringRunAdoptionCandidates(ctx, a.cursor, pageSize)
	if err != nil {
		return err
	}
	for _, candidate := range candidates {
		if err := ctx.Err(); err != nil {
			return err
		}
		recordCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		err := a.Ensure(recordCtx, candidate.ID)
		cancel()
		if err != nil {
			glog.Warningf("Automatic recurring-run adoption will retry schedule %s: %v", candidate.ID, err)
		}
		a.cursor = candidate.ID
	}
	if len(candidates) < pageSize {
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
