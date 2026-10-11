// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0

package storage

import (
	"context"
	"fmt"
	"testing"

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
			for _, unwanted := range []string{"run_metrics", "tasks", "PipelineRuntimeManifest", "PipelineSpecManifest"} {
				require.NotContains(t, query, unwanted)
			}
			require.Contains(t, query, d.QuoteIdentifier("WorkflowSpecManifest"))
			if name == "pgx" {
				require.NotContains(t, query, "?")
			}
		})
	}
}

func TestRecurringRunInventoryRejectsTruncatedHistory(t *testing.T) {
	db, d, _ := initializeDBAndStore()
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	orm, err := OpenTransferDB(db, "sqlite")
	require.NoError(t, err)
	for i := 0; i <= legacyRunInventoryLimit; i++ {
		require.NoError(t, orm.Create(&model.Run{UUID: fmt.Sprint("history-", i), RecurringRunId: "bounded-job"}).Error)
	}
	store := NewRunStore(db, util.NewFakeTimeForEpoch(), d)
	runs, err := store.ListRunsForRecurringRunAdoption(context.Background(), "bounded-job")
	require.ErrorContains(t, err, "retained run inventory exceeds")
	require.Nil(t, runs)
}
