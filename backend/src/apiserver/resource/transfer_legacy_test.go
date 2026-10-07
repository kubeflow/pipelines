// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

package resource

import (
	"context"
	"os"
	"testing"

	api "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/transfer"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm/clause"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestTransferLegacyArchiveThroughResourceManager(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()
	oldNS, oldMulti := viper.Get("POD_NAMESPACE"), viper.Get("MULTIUSER")
	viper.Set("POD_NAMESPACE", "team")
	viper.Set("MULTIUSER", false)
	t.Cleanup(func() { viper.Set("POD_NAMESPACE", oldNS); viper.Set("MULTIUSER", oldMulti) })
	archive, err := os.ReadFile("../history/testdata/legacy-218-mlmd-v2-export.json")
	require.NoError(t, err)
	r, cs := transferTestManager(t)
	db, err := r.transferDB()
	require.NoError(t, err)
	native := model.Experiment{UUID: "native", Name: "Default", Namespace: "team"}
	require.NoError(t, db.Create(&native).Error)
	ctx := context.Background()
	opts := transfer.ImportOptions{NamePrefix: "old-", DryRun: true}
	preview, err := r.ImportTransfer(ctx, "team", archive, opts)
	require.NoError(t, err)
	require.Equal(t, 1, preview.Counts.Runs)
	require.Equal(t, 1, preview.Counts.Schedules)
	require.NotEmpty(t, preview.Warnings)
	for _, warning := range preview.Warnings {
		require.NotContains(t, warning, "release-2.18")
		require.NotContains(t, warning, "MLMD")
	}
	var count int64
	require.NoError(t, db.Model(&model.Run{}).Count(&count).Error)
	require.Zero(t, count)
	for _, a := range cs.Actions() {
		require.NotEqual(t, "create", a.GetVerb())
	}
	opts.DryRun = false
	applied, err := r.ImportTransfer(ctx, "team", archive, opts)
	require.NoError(t, err)
	require.Positive(t, applied.Imported)
	var job model.Job
	var run model.Run
	require.NoError(t, db.Take(&job).Error)
	require.NoError(t, db.Take(&run).Error)
	require.NotEmpty(t, run.StateHistoryString, "persisted imported state history")
	hydrated, err := r.GetRun(run.UUID)
	require.NoError(t, err)
	require.Len(t, hydrated.StateHistory, 1)
	require.EqualValues(t, 1700000000, hydrated.StateHistory[0].UpdateTimeInSec)
	require.ErrorContains(t, hydrated.StateHistory[0].Error, "source error")
	require.Equal(t, codes.Internal, status.Code(hydrated.StateHistory[0].Error))

	require.JSONEq(t, `{"text":"run override","large_integer":9007199254740993}`, string(run.RuntimeConfig.Parameters))
	require.JSONEq(t, `{"text":"schedule override"}`, string(job.RuntimeConfig.Parameters))
	require.NotEmpty(t, run.ImportedFrom)
	require.Zero(t, run.PipelineRunContextId)
	swf, err := cs.ScheduledworkflowV1beta1().ScheduledWorkflows("team").Get(ctx, job.K8SName, metav1.GetOptions{})
	require.NoError(t, err)
	require.False(t, swf.Spec.Enabled)
	require.True(t, *swf.Spec.NoCatchup)
	var tasks []model.Task
	require.NoError(t, db.Find(&tasks).Error)
	require.Len(t, tasks, 3)
	for _, task := range tasks {
		require.Nil(t, task.LogicalKey)
		require.Empty(t, task.Fingerprint)
		_, err := model.JSONDataToProtoMessage(task.StatusMetadata, func() *api.PipelineTask_StatusMetadata { return &api.PipelineTask_StatusMetadata{} })
		require.NoError(t, err)
	}
	var artifacts []model.Artifact
	require.NoError(t, db.Find(&artifacts).Error)
	require.Len(t, artifacts, 1)
	require.NotNil(t, artifacts[0].URI)
	require.Contains(t, *artifacts[0].URI, "s3://")
	var links []model.ArtifactTask
	require.NoError(t, db.Find(&links).Error)
	require.Len(t, links, 2, "cached task retains its source output artifact")
	require.Equal(t, links[0].ArtifactID, links[1].ArtifactID)
	repeat, err := r.ImportTransfer(ctx, "team", archive, opts)
	require.NoError(t, err)
	require.Zero(t, repeat.Imported)
	require.NoError(t, db.Where(clause.Eq{Column: "UUID", Value: "native"}).Take(&native).Error)
	require.Equal(t, "Default", native.Name)
	require.NoError(t, db.Model(&model.Run{}).Count(&count).Error)
	require.EqualValues(t, 1, count)
}
