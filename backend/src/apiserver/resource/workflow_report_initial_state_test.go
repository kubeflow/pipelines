// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

package resource

import (
	"context"
	"testing"

	workflowapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestWorkflowReportPreservesActiveStateBeforePhase(t *testing.T) {
	for _, state := range []model.RuntimeState{model.RuntimeStatePending, model.RuntimeStateRunning, model.RuntimeStatePaused} {
		t.Run(string(state), func(t *testing.T) {
			store, manager, created := initWithOneTimeRunV2(t)
			defer store.Close()
			require.Equal(t, model.RuntimeStatePending, created.State)
			ctx := context.Background()
			if state != model.RuntimeStatePending {
				created.State = state
				created.Conditions = string(state.ToExecutionPhase())
				require.NoError(t, store.RunStore().UpdateRun(created))
			}
			live, err := store.ExecClient().Execution(created.Namespace).Get(ctx, created.K8SName, metav1.GetOptions{})
			require.NoError(t, err)
			report := util.NewWorkflow(live.(*util.Workflow).DeepCopy())
			report.Status = workflowapi.WorkflowStatus{}
			syncWorkflowReportWithFakeCluster(t, store, report)
			_, err = manager.ReportWorkflowResource(ctx, report)
			require.NoError(t, err)
			persisted, err := manager.GetRun(created.UUID)
			require.NoError(t, err)
			require.Equal(t, state, persisted.State, "an initial empty phase must not erase a known active state")
			require.Equal(t, string(state.ToExecutionPhase()), persisted.Conditions)
			require.NotEmpty(t, persisted.WorkflowRuntimeManifest, "retain runtime evidence even before Argo assigns a phase")
			require.Zero(t, persisted.FinishedAtInSec)

			report.Status.Phase = workflowapi.WorkflowRunning
			syncWorkflowReportWithFakeCluster(t, store, report)
			_, err = manager.ReportWorkflowResource(ctx, report)
			require.NoError(t, err)
			persisted, err = manager.GetRun(created.UUID)
			require.NoError(t, err)
			require.Equal(t, model.RuntimeStateRunning, persisted.State)
		})
	}
}
