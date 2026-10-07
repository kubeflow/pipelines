// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

package history

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/transfer"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/clause"
)

func TestNativeTransferRuntimeParametersRoundTrip(t *testing.T) {
	source, dest := database(t), database(t)
	b := namespaceFixture(t, source)
	const runParams = `{"count":9007199254740993,"label":"historical"}`
	const jobParams = `{"count":7,"label":"scheduled"}`
	require.NoError(t, source.Model(&model.Run{}).Where(clause.Eq{Column: "UUID", Value: "run"}).Update("RuntimeParameters", runParams).Error)
	require.NoError(t, source.Model(&model.Job{}).Where(clause.Eq{Column: "UUID", Value: "schedule"}).Update("RuntimeParameters", jobParams).Error)
	b, err := ExportNamespace(context.Background(), source, "team", "team", transfer.ExportOptions{}, b.Pipelines, b.Versions)
	require.NoError(t, err)
	archive, err := json.Marshal(b)
	require.NoError(t, err)
	importArchive := func(raw []byte, dry bool) error {
		decoded, _, err := DecodeNamespaceArchive(dest, raw, "team", "team", false)
		if err != nil {
			return err
		}
		require.Equal(t, runParams, string(decoded.Entries[0].Run.RuntimeConfig.Parameters))
		plan, err := PrepareTransfer(context.Background(), dest, decoded, transfer.ImportOptions{DryRun: dry})
		if err != nil {
			return err
		}
		return CommitTransfer(context.Background(), dest, plan, dry)
	}
	require.NoError(t, importArchive(archive, true))
	require.Zero(t, count(t, dest, &model.Run{}))
	require.NoError(t, importArchive(archive, false))
	require.NoError(t, importArchive(archive, false))
	var run model.Run
	var job model.Job
	require.NoError(t, dest.Take(&run).Error)
	require.NoError(t, dest.Take(&job).Error)
	require.Equal(t, runParams, string(run.RuntimeConfig.Parameters))
	require.Equal(t, jobParams, string(job.RuntimeConfig.Parameters))
	require.False(t, job.Enabled)
	b.RuntimeParameters.Schedules["schedule"] = `{"count":8}`
	changed, err := json.Marshal(b)
	require.NoError(t, err)
	require.ErrorContains(t, importArchive(changed, false), "definition changed")
}

func TestNativeRuntimeParameterValidation(t *testing.T) {
	source, dest := database(t), database(t)
	b := namespaceFixture(t, source)
	for _, tc := range []struct {
		name   string
		change func(*NamespaceBundle)
	}{
		{"missing maps", func(b *NamespaceBundle) { b.RuntimeParameters = nil }},
		{"schema mismatch", func(b *NamespaceBundle) { b.Schema = "wrong" }},
		{"missing run", func(b *NamespaceBundle) { delete(b.RuntimeParameters.Runs, "run") }},
		{"foreign run", func(b *NamespaceBundle) {
			delete(b.RuntimeParameters.Runs, "run")
			b.RuntimeParameters.Runs["other"] = ""
		}},
		{"not object", func(b *NamespaceBundle) { b.RuntimeParameters.Schedules["schedule"] = "[]" }},
		{"v1 schedules", func(b *NamespaceBundle) { b.Format = "kfp-namespace-transfer/v1"; b.RuntimeParameters = nil }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			copy := cloneTransfer(t, b)
			tc.change(copy)
			raw, err := json.Marshal(copy)
			require.NoError(t, err)
			_, _, err = DecodeNamespaceArchive(dest, raw, "team", "team", false)
			require.Error(t, err)
			require.Zero(t, count(t, dest, &model.TransferIdentity{}))
		})
	}
	old := cloneTransfer(t, b)
	old.Format = "kfp-namespace-transfer/v1"
	old.RuntimeParameters = nil
	old.Schedules = nil
	raw, err := json.Marshal(old)
	require.NoError(t, err)
	_, warnings, err := DecodeNamespaceArchive(dest, raw, "team", "team", false)
	require.NoError(t, err)
	require.NotEmpty(t, warnings)
}

func TestNativeReaderBudgetsRuntimeOverrides(t *testing.T) {
	db := database(t)
	create(t, db, &model.Experiment{UUID: "exp", Name: "exp", Namespace: "team"})
	create(t, db, &model.Run{UUID: "large", ExperimentId: "exp", Namespace: "team", PipelineSpec: model.PipelineSpec{RuntimeConfig: model.RuntimeConfig{Parameters: model.LargeText(`{"value":"` + strings.Repeat("x", 8192) + `"}`)}}})
	var rows []model.Run
	err := readTransferRows(db, &rows, transfer.NewExportBudget(4096))
	require.Error(t, err)
	require.Empty(t, rows, "oversized runtime parameters must be charged before retaining the row")
}
