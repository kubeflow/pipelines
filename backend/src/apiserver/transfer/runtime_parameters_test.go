// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

package transfer

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/require"
)

func signRuntimeArchive(t *testing.T, b Bundle) []byte {
	t.Helper()
	b.Digest = ""
	b.Digest = hash(b)
	data, err := json.Marshal(b)
	require.NoError(t, err)
	return data
}

func TestRuntimeParametersExportImportAndReceipt(t *testing.T) {
	source := fixture(t, testDB(t))
	runParameters := `{"text":"run override","integer":9007199254740993}`
	scheduleParameters := `{"text":"schedule override"}`
	require.NoError(t, source.DB.Model(&model.Run{}).Where(equal("UUID", "run")).Update("RuntimeParameters", runParameters).Error)
	require.NoError(t, source.DB.Model(&model.Job{}).Where(equal("UUID", "schedule")).Update("RuntimeParameters", scheduleParameters).Error)
	data, err := source.Export(context.Background(), "team", ExportOptions{})
	require.NoError(t, err)
	var archive Bundle
	require.NoError(t, json.Unmarshal(data, &archive))
	require.Equal(t, "kfp-namespace-transfer-mlmd-2.18/v2", archive.Format)
	require.Equal(t, map[string]string{"run": runParameters}, archive.RuntimeParameters.Runs)
	require.Equal(t, map[string]string{"schedule": scheduleParameters}, archive.RuntimeParameters.Schedules)
	destination := &Engine{DB: testDB(t), RuntimeNamespace: "team", Metadata: &Metadata{RPC: &emptyRPC{}}, Schedules: &fakeSchedules{}}
	_, err = destination.Import(context.Background(), "team", data, ImportOptions{})
	require.NoError(t, err)
	var run model.Run
	require.NoError(t, destination.DB.Where(equal("UUID", "run")).First(&run).Error)
	require.Equal(t, runParameters, string(run.RuntimeConfig.Parameters))
	var schedule model.Job
	require.NoError(t, destination.DB.Where(equal("UUID", "local-schedule")).First(&schedule).Error)
	require.Equal(t, scheduleParameters, string(schedule.RuntimeConfig.Parameters))
	repeat, err := destination.Import(context.Background(), "team", data, ImportOptions{})
	require.NoError(t, err)
	require.Zero(t, repeat.Imported)
	for _, kind := range []string{"run", "schedule"} {
		var changed Bundle
		require.NoError(t, json.Unmarshal(data, &changed))
		if kind == "run" {
			changed.RuntimeParameters.Runs["run"] = `{"text":"changed"}`
		} else {
			changed.RuntimeParameters.Schedules["schedule"] = `{"text":"changed"}`
		}
		_, err = destination.Import(context.Background(), "team", signRuntimeArchive(t, changed), ImportOptions{})
		require.Error(t, err, "changed override must conflict with the original resource receipt")
	}
	require.Equal(t, 1, destination.Schedules.(*fakeSchedules).writes)
}

func TestRuntimeParametersRejectIncompleteArchiveBeforeWrites(t *testing.T) {
	source := fixture(t, testDB(t))
	data, err := source.Export(context.Background(), "team", ExportOptions{})
	require.NoError(t, err)
	for _, tc := range []struct {
		name   string
		change func(*Bundle)
	}{
		{"missing maps", func(b *Bundle) { b.RuntimeParameters = nil }},
		{"missing empty run entry", func(b *Bundle) { delete(b.RuntimeParameters.Runs, "run") }},
		{"wrong run owner", func(b *Bundle) { delete(b.RuntimeParameters.Runs, "run"); b.RuntimeParameters.Runs["foreign"] = "" }},
		{"extra schedule entry", func(b *Bundle) { b.RuntimeParameters.Schedules["foreign"] = "" }},
		{"malformed overrides", func(b *Bundle) { b.RuntimeParameters.Schedules["schedule"] = "not json" }},
		{"non-object overrides", func(b *Bundle) { b.RuntimeParameters.Runs["run"] = "null" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var archive Bundle
			require.NoError(t, json.Unmarshal(data, &archive))
			tc.change(&archive)
			destination := &Engine{DB: testDB(t), RuntimeNamespace: "team", Metadata: &Metadata{RPC: &emptyRPC{}}, Schedules: &fakeSchedules{}}
			for _, dry := range []bool{true, false} {
				_, err := destination.Import(context.Background(), "team", signRuntimeArchive(t, archive), ImportOptions{DryRun: dry})
				require.Error(t, err)
			}
			require.Zero(t, destination.Schedules.(*fakeSchedules).writes)
			require.Zero(t, destination.Metadata.RPC.(*emptyRPC).writes)
			var count int64
			require.NoError(t, destination.DB.Model(&model.Experiment{}).Count(&count).Error)
			require.Zero(t, count)
		})
	}
}

func TestRuntimeParametersV1HistoryCompatibility(t *testing.T) {
	source := fixture(t, testDB(t))
	data, err := source.Export(context.Background(), "team", ExportOptions{})
	require.NoError(t, err)
	var archive Bundle
	require.NoError(t, json.Unmarshal(data, &archive))
	archive.Format = archiveFormatV1
	archive.RuntimeParameters = nil
	destination := &Engine{DB: testDB(t), RuntimeNamespace: "team", Metadata: &Metadata{RPC: &emptyRPC{}}, Schedules: &fakeSchedules{}}
	_, err = destination.Import(context.Background(), "team", signRuntimeArchive(t, archive), ImportOptions{})
	require.ErrorContains(t, err, "re-export")
	require.Zero(t, destination.Schedules.(*fakeSchedules).writes)
	archive.Schedules = nil
	archive.Runs[0].Run.RecurringRunId = ""
	archive.References = nil
	result, err := destination.Import(context.Background(), "team", signRuntimeArchive(t, archive), ImportOptions{})
	require.NoError(t, err)
	require.Contains(t, strings.Join(result.Warnings, " "), "did not preserve V2 runtime parameter overrides")
}

func TestRuntimeParametersExportRejectsMalformedOverrides(t *testing.T) {
	for _, schedule := range []bool{false, true} {
		source := fixture(t, testDB(t))
		if schedule {
			require.NoError(t, source.DB.Model(&model.Job{}).Where(equal("UUID", "schedule")).Update("RuntimeParameters", "[]").Error)
		} else {
			require.NoError(t, source.DB.Model(&model.Run{}).Where(equal("UUID", "run")).Update("RuntimeParameters", "invalid-json").Error)
		}
		_, err := source.Export(context.Background(), "team", ExportOptions{})
		require.ErrorContains(t, err, "JSON object")
	}
}
