// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0

package resource

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

type automaticSynchronizerFake struct {
	ready    map[string]bool
	failures map[string]bool
	adopted  []string
	pages    map[string][]storage.RecurringRunMigrationCandidate
	cursors  []string
}

func (f *automaticSynchronizerFake) RequireRecurringRunAdoptionReady(_ context.Context, id string) error {
	if f.ready[id] {
		return nil
	}
	return errors.New("not ready")
}
func (f *automaticSynchronizerFake) SynchronizeRecurringRun(_ context.Context, id string) error {
	f.adopted = append(f.adopted, id)
	if f.failures[id] {
		return errors.New("bad record")
	}
	f.ready[id] = true
	return nil
}
func (f *automaticSynchronizerFake) ListPendingRecurringRunSynchronizations(_ context.Context, cursor string, _ uint64) ([]storage.RecurringRunMigrationCandidate, error) {
	f.cursors = append(f.cursors, cursor)
	return f.pages[cursor], nil
}
func TestAutomaticSynchronizationHealthyScheduleDoesNotWaitForRollout(t *testing.T) {
	f := &automaticSynchronizerFake{ready: map[string]bool{"healthy": true}}
	a := NewAutomaticRecurringRunSynchronization(f, func(context.Context) error { t.Fatal("healthy record checked rollout"); return nil })
	require.NoError(t, a.Ensure(context.Background(), "healthy"))
	require.Empty(t, f.adopted)
}
func TestAutomaticSynchronizationWaitsAndRetries(t *testing.T) {
	f := &automaticSynchronizerFake{ready: map[string]bool{}}
	waiting := true
	a := NewAutomaticRecurringRunSynchronization(f, func(context.Context) error {
		if waiting {
			return errors.New("old writer")
		}
		return nil
	})
	require.Error(t, a.Ensure(context.Background(), "legacy"))
	require.Empty(t, f.adopted)
	waiting = false
	require.NoError(t, a.Ensure(context.Background(), "legacy"))
	require.Equal(t, []string{"legacy"}, f.adopted)
}
func TestAutomaticSynchronizationScanIsolatesFailuresAndResumesPages(t *testing.T) {
	page := make([]storage.RecurringRunMigrationCandidate, 100)
	for i := range page {
		page[i].ID = fmt.Sprintf("%03d", i)
	}
	f := &automaticSynchronizerFake{ready: map[string]bool{}, failures: map[string]bool{"000": true}, pages: map[string][]storage.RecurringRunMigrationCandidate{
		"": page, "099": {{ID: "disabled"}},
	}}
	a := NewAutomaticRecurringRunSynchronization(f, func(context.Context) error { return nil })
	require.NoError(t, a.scan(context.Background()))
	require.False(t, f.ready["000"])
	require.True(t, f.ready["099"])
	require.Equal(t, "099", a.cursor)
	require.NoError(t, a.scan(context.Background()))
	require.True(t, f.ready["disabled"])
	require.Empty(t, a.cursor)
	f.failures["000"] = false
	require.NoError(t, a.scan(context.Background()))
	require.True(t, f.ready["000"])
	require.Equal(t, []string{"", "099", ""}, f.cursors)
}
func TestAutomaticSynchronizationScanStopsOnCancellation(t *testing.T) {
	f := &automaticSynchronizerFake{ready: map[string]bool{}, pages: map[string][]storage.RecurringRunMigrationCandidate{"": {{ID: "legacy"}}}}
	a := NewAutomaticRecurringRunSynchronization(f, func(context.Context) error { return nil })
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.ErrorIs(t, a.scan(ctx), context.Canceled)
	require.Empty(t, f.adopted)
}

func TestPrepareRecurringRunSynchronizesOnlyAfterAuthorization(t *testing.T) {
	initEnvVars()
	previous := viper.Get(common.MultiUserMode)
	viper.Set(common.MultiUserMode, "true")
	t.Cleanup(func() { viper.Set(common.MultiUserMode, previous) })
	store := NewFakeClientManagerOrFatalV2()
	defer store.Close()
	calls := 0
	sentinel := errors.New("automatic handoff waiting")
	manager := NewResourceManager(store, &ResourceManagerOptions{EnsureRecurringRunSynchronized: func(context.Context, string) error { calls++; return sentinel }})
	ctx := multiUserContext()
	experiment, err := manager.CreateExperiment(&model.Experiment{Name: "automatic", Namespace: "ns1"})
	require.NoError(t, err)
	job, err := manager.CreateJob(ctx, &model.Job{DisplayName: "automatic", Namespace: "ns1", ExperimentId: experiment.UUID, Enabled: true, MaxConcurrency: 1, PipelineSpec: model.PipelineSpec{PipelineSpecManifest: model.LargeText(v2SpecHelloWorld), RuntimeConfig: model.RuntimeConfig{Parameters: `{"text":"test"}`}}})
	require.NoError(t, err)
	require.Error(t, manager.PrepareRecurringRun(context.Background(), &model.Run{RecurringRunId: job.UUID}))
	require.Zero(t, calls)
	require.Error(t, manager.PrepareRecurringRun(ctx, &model.Run{RecurringRunId: job.UUID, Namespace: "other"}))
	require.Zero(t, calls)
	require.ErrorIs(t, manager.PrepareRecurringRun(ctx, &model.Run{RecurringRunId: job.UUID}), sentinel)
	require.Equal(t, 1, calls)
}
