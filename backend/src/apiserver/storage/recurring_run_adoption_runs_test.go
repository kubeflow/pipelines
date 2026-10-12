// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0

package storage

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
)

func TestRecurringRunAdoptionInventoryPreservesLegacyEvidence(t *testing.T) {
	db, d, _ := initializeDBAndStore()
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	orm, err := OpenTransferDB(db, "sqlite")
	require.NoError(t, err)
	store := NewRunStore(db, util.NewFakeTimeForEpoch(), d)
	for _, id := range []string{"direct", "legacy", "foreign"} {
		run := &model.Run{UUID: id, DisplayName: "scheduled-request", K8SName: "workflow", PipelineSpec: model.PipelineSpec{WorkflowSpecManifest: "persisted-spec"}}
		if id == "direct" {
			run.RecurringRunId = "job"
			run.Namespace = "explicit-ns"
			run.PipelineVersionId = "explicit-version"
		}
		if id == "foreign" {
			run.RecurringRunId = "other-job"
		}
		require.NoError(t, orm.Create(run).Error)
		if id == "direct" {
			_, err = store.GetRun(id)
			require.True(t, util.IsUserErrorCodeMatch(err, codes.NotFound), "expected the public API runtime-reporting guard, got %v", err)
		}
		for typ, value := range map[model.ResourceType]string{model.JobResourceType: "job", model.NamespaceResourceType: "legacy-ns", model.PipelineVersionResourceType: "legacy-version"} {
			require.NoError(t, orm.Create(&model.ResourceReference{ResourceUUID: id, ResourceType: model.RunResourceType, ReferenceType: typ, ReferenceUUID: value}).Error)
		}
	}
	// The schema permits only one fallback of each type per run, across dialects.
	require.Error(t, orm.Create(&model.ResourceReference{ResourceUUID: "legacy", ResourceType: model.RunResourceType, ReferenceType: model.NamespaceResourceType, ReferenceUUID: "conflicting-ns"}).Error)

	runs, err := store.ListRunsForRecurringRunAdoption(context.Background(), "job")
	require.NoError(t, err)
	require.Len(t, runs, 2)
	require.Equal(t, "direct", runs[0].UUID)
	require.Equal(t, "explicit-ns", runs[0].Namespace)
	require.Equal(t, "explicit-version", runs[0].PipelineVersionId)
	require.Equal(t, "legacy", runs[1].UUID)
	require.Equal(t, "legacy-ns", runs[1].Namespace)
	require.Equal(t, "legacy-version", runs[1].PipelineVersionId)
	for _, run := range runs {
		require.Equal(t, "job", run.RecurringRunId)
		require.EqualValues(t, "persisted-spec", run.WorkflowSpecManifest)
		require.Empty(t, run.WorkflowRuntimeManifest)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = store.ListRunsForRecurringRunAdoption(ctx, "job")
	require.ErrorContains(t, err, "context canceled")
}

func TestRecurringRunAdoptionQueryDialects(t *testing.T) {
	for _, name := range []string{"mysql", "pgx", "sqlite"} {
		t.Run(name, func(t *testing.T) {
			d := dialect.NewDBDialect(name)
			query, _, err := d.FinalizeSelect(recurringRunAdoptionQuery(d.QuoteIdentifier, "job"))
			require.NoError(t, err)
			for _, unwanted := range []string{"run_metrics", "tasks", "PipelineRuntimeManifest", "PipelineSpecManifest", "LIMIT"} {
				require.NotContains(t, query, unwanted)
			}
			require.Contains(t, query, d.QuoteIdentifier("WorkflowSpecManifest"))
			if name == "pgx" {
				require.NotContains(t, query, "?")
			}
		})
	}
}

func TestRecurringRunInventoryStreamsLargeHistory(t *testing.T) {
	db, d, _ := initializeDBAndStore()
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	orm, err := OpenTransferDB(db, "sqlite")
	require.NoError(t, err)
	const count = 1205
	for i := 0; i < count; i++ {
		require.NoError(t, orm.Create(&model.Run{UUID: fmt.Sprintf("history-%04d", i), RecurringRunId: "large-job"}).Error)
	}
	store := NewRunStore(db, util.NewFakeTimeForEpoch(), d)
	seen := 0
	err = store.VisitRunsForRecurringRunAdoption(context.Background(), "large-job", func(run *model.Run) error {
		require.Equal(t, fmt.Sprintf("history-%04d", seen), run.UUID)
		seen++
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, count, seen)

	// An error beyond the old limit must stop the scan and remain actionable.
	rejected := errors.New("conflicting scheduling evidence")
	seen = 0
	err = store.VisitRunsForRecurringRunAdoption(context.Background(), "large-job", func(run *model.Run) error {
		seen++
		if seen == 1100 {
			return rejected
		}
		return nil
	})
	require.ErrorIs(t, err, rejected)
	require.Equal(t, 1100, seen)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	seen = 0
	err = store.VisitRunsForRecurringRunAdoption(ctx, "large-job", func(run *model.Run) error {
		seen++
		cancel()
		return nil
	})
	require.ErrorContains(t, err, "context canceled")
	require.Equal(t, 1, seen)

	// Early exits close their cursor and do not prevent a complete retry.
	runs, err := store.ListRunsForRecurringRunAdoption(context.Background(), "large-job")
	require.NoError(t, err)
	require.Len(t, runs, count)
}

func TestRecurringRunInventoryCancelsStalledVisitor(t *testing.T) {
	db, d, _ := initializeDBAndStore()
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	orm, err := OpenTransferDB(db, "sqlite")
	require.NoError(t, err)
	require.NoError(t, orm.Create(&model.Run{UUID: "history", RecurringRunId: "job"}).Error)
	store := NewRunStore(db, util.NewFakeTimeForEpoch(), d)
	err = store.visitRunsForRecurringRunAdoption(context.Background(), "job", func(run *model.Run) error {
		time.Sleep(50 * time.Millisecond)
		return nil
	}, 10*time.Millisecond)
	require.ErrorContains(t, err, "context canceled")
	// Cancellation also closes the cursor, permitting another complete inventory.
	runs, err := store.ListRunsForRecurringRunAdoption(context.Background(), "job")
	require.NoError(t, err)
	require.Len(t, runs, 1)
}
