// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0

package resource

import (
	"context"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/require"
)

func TestAdoptionInventoryValidatesSpecBeforeRuntimeReporting(t *testing.T) {
	_, clients := onlineAdoptionManager(t)
	db, err := clients.TransferDB()
	require.NoError(t, err)
	job, swf, run, wf := legacyAdoptionFixture()
	// Reporter recovery uses the execution name, requiring manifest evidence.
	run.DisplayName = run.K8SName
	run.WorkflowSpecManifest = model.LargeText(wf.ToStringForStore())
	require.NoError(t, db.Create(run).Error)
	inventory := clients.RunStore().(interface {
		ListRunsForRecurringRunAdoption(context.Context, string) ([]*model.Run, error)
	})
	runs, err := inventory.ListRunsForRecurringRunAdoption(context.Background(), job.UUID)
	require.NoError(t, err)
	require.Len(t, runs, 1)
	require.Empty(t, runs[0].WorkflowRuntimeManifest)
	state, err := adoptLegacyRecurringRunProgress(job, swf, runs, util.ExecutionSpecList{wf}, 200)
	require.NoError(t, err)
	require.Equal(t, run.UUID, state.LastRunUUID)
	// Fetching these records is not permission to accept invalid persisted evidence.
	runs[0].WorkflowSpecManifest = "{}"
	_, err = adoptLegacyRecurringRunProgress(job, swf, runs, util.ExecutionSpecList{wf}, 200)
	require.Error(t, err)
}
