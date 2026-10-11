// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

package history

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/transfer"
	"github.com/stretchr/testify/require"
)

func readLegacyGolden(t *testing.T, name string) legacyArchive {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	require.NoError(t, err)
	var b legacyArchive
	require.NoError(t, decodeArchiveJSON(data, &b))
	checksum := b.Digest
	b.Digest = ""
	actual, err := digest(b)
	require.NoError(t, err)
	require.Equal(t, checksum, actual, "frozen DTO must reproduce the genuine source exporter digest")
	return b
}
func encodeLegacy(t *testing.T, b legacyArchive) []byte {
	t.Helper()
	b.Digest = ""
	var err error
	b.Digest, err = digest(b)
	require.NoError(t, err)
	data, err := json.Marshal(b)
	require.NoError(t, err)
	return data
}
func legacyGolden(t *testing.T) legacyArchive {
	t.Helper()
	return readLegacyGolden(t, "legacy-218-v2-export.json")
}
func TestLegacyArchiveExporterWireDigest(t *testing.T) {
	b := legacyGolden(t)
	require.Len(t, b.Experiments, 2)
	require.Len(t, b.Versions, 2)
	require.Len(t, b.Schedules, 1)
	_, _, err := DecodeNamespaceArchive(database(t), encodeLegacy(t, b), "team", "team", false)
	require.NoError(t, err)
	// Nonempty runtime-status errors retain their original JSON representation,
	// including precision above the floating point exact-integer range.
	b.Runs[0].Run.StateHistory = []*legacyRuntimeStatus{{State: "FAILED", UpdateTimeInSec: 9007199254740993, Error: json.RawMessage(`{"code":3,"message":"failed"}`)}}
	wire := encodeLegacy(t, b)
	var copy legacyArchive
	require.NoError(t, decodeArchiveJSON(wire, &copy))
	require.Equal(t, b.Runs[0].Run.StateHistory, copy.Runs[0].Run.StateHistory)
	checksum := copy.Digest
	copy.Digest = ""
	actual, err := digest(copy)
	require.NoError(t, err)
	require.Equal(t, checksum, actual)
}
func TestLegacyArchiveConversionMergePreviewRepeat(t *testing.T) {
	db := database(t)
	create(t, db, &model.Experiment{UUID: "existing", Name: "Default", Namespace: "team"})
	source := legacyGolden(t)
	data := encodeLegacy(t, source)
	decode := func() *NamespaceBundle {
		b, w, err := DecodeNamespaceArchive(db, data, "team", "team", false)
		require.NoError(t, err)
		require.Empty(t, w)
		return b
	}
	b := decode()
	require.Equal(t, TransferFormat, b.Format)
	require.NotEqual(t, source.Schema, b.Schema)
	require.Equal(t, "version", b.Pipelines[0].DefaultVersionId)
	require.Equal(t, source.RuntimeParameters.Runs["run"], string(b.Entries[0].Run.RuntimeConfig.Parameters))
	require.Equal(t, source.RuntimeParameters.Schedules["schedule"], string(b.Schedules[0].RuntimeConfig.Parameters))
	require.Len(t, b.PipelineTags, 1)
	require.Len(t, b.VersionTags, 1)
	require.Len(t, b.Entries[0].Metrics, 1)
	ctx := context.Background()
	opts := transfer.ImportOptions{NamePrefix: "old-", DryRun: true}
	plan, err := PrepareTransfer(ctx, db, b, opts)
	require.NoError(t, err)
	require.NoError(t, CommitTransfer(ctx, db, plan, true))
	require.Zero(t, count(t, db, &model.TransferIdentity{}))
	require.Zero(t, count(t, db, &model.Run{}))
	opts.DryRun = false
	plan, err = PrepareTransfer(ctx, db, decode(), opts)
	require.NoError(t, err)
	require.NoError(t, CommitTransfer(ctx, db, plan, false))
	var run model.Run
	require.NoError(t, db.Take(&run).Error)
	require.Equal(t, source.Source, run.ImportedFrom)
	require.Contains(t, string(run.StateHistoryString), "source error")
	require.Zero(t, run.PipelineRunContextId)
	var job model.Job
	require.NoError(t, db.Take(&job).Error)
	require.False(t, job.Enabled)
	require.True(t, job.NoCatchup)
	var state model.RecurringRunState
	require.NoError(t, db.Take(&state).Error)
	require.False(t, state.Pending)
	repeat, err := PrepareTransfer(ctx, db, decode(), opts)
	require.NoError(t, err)
	require.Zero(t, repeat.Summary.Imported)
	require.NoError(t, CommitTransfer(ctx, db, repeat, false))
	require.EqualValues(t, 1, count(t, db, &model.Run{}))
	source.Runs[0].Run.DisplayName = "changed"
	data = encodeLegacy(t, source)
	_, err = PrepareTransfer(ctx, db, decode(), opts)
	require.ErrorContains(t, err, "changed")
}
func TestLegacyArchiveRejectsBeforeDestinationWrites(t *testing.T) {
	cases := map[string]func(*legacyArchive){
		"source":             func(b *legacyArchive) { b.Source = "not-uuid" },
		"schema":             func(b *legacyArchive) { b.Schema = "not-a-hash" },
		"namespace":          func(b *legacyArchive) { b.RuntimeNamespace = "private" },
		"run ownership":      func(b *legacyArchive) { b.Runs[0].Run.Namespace = "private" },
		"task ownership":     func(b *legacyArchive) { b.Runs[0].Tasks[0].Namespace = "private" },
		"schedule ownership": func(b *legacyArchive) { b.Schedules[0].Namespace = "private" },
		"nested pipeline":    func(b *legacyArchive) { b.Versions[0].Pipeline.Name = "nested" },
		"source URI":         func(b *legacyArchive) { b.Versions[0].PipelineSpecURI = "s3://source/spec" },
		"task parent cycle":  func(b *legacyArchive) { b.Runs[0].Tasks[0].ParentTaskId = b.Runs[0].Tasks[0].UUID },
		"active run":         func(b *legacyArchive) { b.Runs[0].Run.State = "RUNNING" },
		"import marker":      func(b *legacyArchive) { s := "source"; b.Runs[0].Run.ImportedFrom = &s },
		"malformed persisted state history": func(b *legacyArchive) {
			b.Runs[0].Run.StateHistoryString = `[{"State":"FAILED","Error":{"unknown":true}}]`
		},
		"trailing persisted state history": func(b *legacyArchive) { b.Runs[0].Run.StateHistoryString = `[] {}` },
		"unsupported persisted state":      func(b *legacyArchive) { b.Runs[0].Run.StateHistoryString = `[{"State":"BOGUS"}]` },
		"unsupported hydrated state": func(b *legacyArchive) {
			b.Runs[0].Run.StateHistoryString = ""
			b.Runs[0].Run.StateHistory = []*legacyRuntimeStatus{{State: "BOGUS"}}
		},
		"null persisted status":      func(b *legacyArchive) { b.Runs[0].Run.StateHistoryString = `[null]` },
		"metric owner":               func(b *legacyArchive) { b.Runs[0].Metrics[0].RunUUID = "other" },
		"missing runtime parameters": func(b *legacyArchive) { delete(b.RuntimeParameters.Runs, "run") },
		"invalid runtime parameters": func(b *legacyArchive) { b.RuntimeParameters.Runs["run"] = "[]" },
		"foreign runtime parameters": func(b *legacyArchive) { b.RuntimeParameters.Runs["private"] = "{}" },
		"reference mismatch": func(b *legacyArchive) {
			b.References = append(b.References, legacyResourceReference{ResourceUUID: "run", ResourceType: "Run", ReferenceUUID: "empty", ReferenceType: "Experiment", Relationship: "Owner"})
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			db := database(t)
			b := legacyGolden(t)
			mutate(&b)
			_, _, err := DecodeNamespaceArchive(db, encodeLegacy(t, b), "team", "team", false)
			require.Error(t, err)
			require.Zero(t, count(t, db, &model.TransferIdentity{}))
			require.Zero(t, count(t, db, &model.TransferReceipt{}))
			require.Zero(t, count(t, db, &model.Experiment{}))
		})
	}
}
func TestLegacyArchiveStrictJSONAndDefaults(t *testing.T) {
	db := database(t)
	b := legacyGolden(t)
	data := encodeLegacy(t, b)
	trailing := append(append([]byte{}, data...), []byte(" {}")...)
	_, _, trailingErr := DecodeNamespaceArchive(db, trailing, "team", "team", false)
	require.Error(t, trailingErr)
	unknown := append([]byte(`{"unrecognized":true,`), data[1:]...)
	_, _, err := DecodeNamespaceArchive(db, unknown, "team", "team", false)
	require.ErrorContains(t, err, "unknown field")
	tampered := []byte(strings.Replace(string(data), "Finished", "tampered", 1))
	_, _, err = DecodeNamespaceArchive(db, tampered, "team", "team", false)
	require.ErrorContains(t, err, "digest")
	b.CatalogDefaults = map[string]string{"pipeline": "v2"}
	_, _, err = DecodeNamespaceArchive(db, encodeLegacy(t, b), "team", "team", false)
	require.ErrorContains(t, err, "Kubernetes")
	converted, _, err := DecodeNamespaceArchive(db, encodeLegacy(t, b), "team", "team", true)
	require.NoError(t, err)
	require.Equal(t, "unused", converted.Pipelines[0].DefaultVersionId)
	b.CatalogDefaults["pipeline"] = "missing"
	_, _, err = DecodeNamespaceArchive(db, encodeLegacy(t, b), "team", "team", true)
	require.ErrorContains(t, err, "resolve")
}
func TestLegacyArchiveStateHistory(t *testing.T) {
	b := legacyGolden(t)
	b.Schedules = nil
	b.RuntimeParameters.Schedules = map[string]string{}
	b.Runs[0].Run.StateHistoryString = ""
	b.Runs[0].Run.RecurringRunId = ""
	b.References = nil
	b.Runs[0].Run.StateHistory = []*legacyRuntimeStatus{{State: "FAILED", UpdateTimeInSec: 10, Error: json.RawMessage(`{"message":"failure"}`)}}
	converted, warnings, err := DecodeNamespaceArchive(database(t), encodeLegacy(t, b), "team", "team", false)
	require.NoError(t, err)
	require.Empty(t, warnings)
	require.Contains(t, string(converted.Entries[0].Run.StateHistoryString), "failure")
	b.Runs[0].Run.StateHistoryString = `[{"State":"SUCCEEDED"}]`
	_, _, err = DecodeNamespaceArchive(database(t), encodeLegacy(t, b), "team", "team", false)
	require.ErrorContains(t, err, "state history disagrees")
}

func TestLegacyArchiveActualMLMDExport(t *testing.T) {
	old := readLegacyGolden(t, "legacy-218-mlmd-v2-export.json")
	db := database(t)
	b, _, err := DecodeNamespaceArchive(db, encodeLegacy(t, old), "team", "team", false)
	require.NoError(t, err)
	require.Len(t, b.Entries, 1)
	e := b.Entries[0]
	require.Len(t, e.Tasks, 3)
	require.Len(t, e.Artifacts, 1)
	require.NotEmpty(t, e.Links)
	require.Equal(t, "s3://shared-bucket/unchanged", *e.Artifacts[0].URI)
	require.Contains(t, string(e.Run.RuntimeConfig.Parameters), "run override")
	require.Contains(t, string(e.Run.RuntimeConfig.Parameters), "9007199254740993")
	require.Contains(t, string(e.Run.StateHistoryString), "source error")
	require.Contains(t, string(b.Schedules[0].RuntimeConfig.Parameters), "schedule override")
	for _, task := range e.Tasks {
		require.Empty(t, task.Fingerprint)
		require.Nil(t, task.LogicalKey)
	}
	require.Nil(t, e.Artifacts[0].IdentityKey)
	plan, err := PrepareTransfer(context.Background(), db, b, transfer.ImportOptions{})
	require.NoError(t, err)
	require.NoError(t, CommitTransfer(context.Background(), db, plan, false))
	b, _, err = DecodeNamespaceArchive(db, encodeLegacy(t, old), "team", "team", false)
	require.NoError(t, err)
	plan, err = PrepareTransfer(context.Background(), db, b, transfer.ImportOptions{})
	require.NoError(t, err)
	require.Zero(t, plan.Summary.Imported)
	require.NoError(t, CommitTransfer(context.Background(), db, plan, false))
}

func TestLegacyArchiveSingleUserNamespace(t *testing.T) {
	old := legacyGolden(t)
	old.Namespace = ""
	old.RuntimeNamespace = "runtime"
	for i := range old.Experiments {
		old.Experiments[i].Namespace = ""
	}
	for i := range old.Pipelines {
		old.Pipelines[i].Namespace = ""
	}
	for i := range old.Schedules {
		old.Schedules[i].Namespace = "runtime"
	}
	for i := range old.Runs {
		old.Runs[i].Run.Namespace = "runtime"
		for j := range old.Runs[i].Tasks {
			old.Runs[i].Tasks[j].Namespace = "runtime"
		}
	}
	db := database(t)
	_, _, err := DecodeNamespaceArchive(db, encodeLegacy(t, old), "", "runtime", false)
	require.NoError(t, err)
	for i := range old.Runs {
		old.Runs[i].Run.Namespace = ""
		for j := range old.Runs[i].Tasks {
			old.Runs[i].Tasks[j].Namespace = ""
		}
	}
	_, _, err = DecodeNamespaceArchive(db, encodeLegacy(t, old), "", "runtime", false)
	require.NoError(t, err)
	_, _, err = DecodeNamespaceArchive(db, encodeLegacy(t, old), "", "different-runtime", false)
	require.ErrorContains(t, err, "namespace")
	old.References = append(old.References, legacyResourceReference{ResourceUUID: "run", ResourceType: "Run", ReferenceUUID: "private", ReferenceType: "Namespace", Relationship: "Owner"})
	_, _, err = DecodeNamespaceArchive(db, encodeLegacy(t, old), "", "runtime", false)
	require.ErrorContains(t, err, "namespace")
}
func TestLegacyArchiveWiderBatchRetainsCompletedHistory(t *testing.T) {
	old := readLegacyGolden(t, "legacy-218-mlmd-v2-export.json")
	old.Metadata["contexts"][1]["last_update_time_since_epoch"] = "1000"
	db := database(t)
	ctx := context.Background()
	apply := func(source legacyArchive) *TransferPlan {
		b, _, err := DecodeNamespaceArchive(db, encodeLegacy(t, source), "team", "team", false)
		require.NoError(t, err)
		plan, err := PrepareTransfer(ctx, db, b, transfer.ImportOptions{})
		require.NoError(t, err)
		require.NoError(t, CommitTransfer(ctx, db, plan, false))
		return plan
	}
	apply(old)
	old.Metadata["contexts"][1]["last_update_time_since_epoch"] = "2000"
	second := old.Runs[0].Run
	second.UUID = "run-2"
	second.DisplayName = "Later completed run"
	second.PipelineRunContextId = 7
	second.FinishedAtInSec = 30
	old.Runs = append(old.Runs, legacyRunHistory{Run: second})
	old.RuntimeParameters.Runs[second.UUID] = `{"text":"later"}`
	old.Metadata["contexts"] = append(old.Metadata["contexts"], map[string]any{"id": "7", "type_id": "1", "name": "run-2"})
	old.Metadata["parents"] = append(old.Metadata["parents"], map[string]any{"parent_id": "6", "child_id": "7"})
	old.Metadata["executions"] = append(old.Metadata["executions"],
		map[string]any{"id": "30", "type_id": "5", "last_known_state": "COMPLETE", "custom_properties": map[string]any{"task_name": map[string]any{"string_value": "root"}}},
		map[string]any{"id": "31", "type_id": "3", "last_known_state": "COMPLETE", "custom_properties": map[string]any{"task_name": map[string]any{"string_value": "consume"}, "parent_dag_id": map[string]any{"int_value": "30"}}})
	for _, id := range []string{"30", "31"} {
		old.Metadata["associations"] = append(old.Metadata["associations"], map[string]any{"context_id": "7", "execution_id": id})
	}
	old.Metadata["attributions"] = append(old.Metadata["attributions"], map[string]any{"context_id": "7", "artifact_id": "20"})
	old.Metadata["events"] = append(old.Metadata["events"], map[string]any{"execution_id": "31", "artifact_id": "20", "type": "INPUT", "path": map[string]any{"steps": []any{map[string]any{"key": "model"}}}})
	plan := apply(old)
	require.Equal(t, 1, plan.Summary.Imported)
	require.Equal(t, 7, plan.Summary.Skipped)
	require.EqualValues(t, 2, count(t, db, &model.Run{}))
	require.EqualValues(t, 1, count(t, db, &model.Artifact{}))
	repeat := apply(old)
	require.Zero(t, repeat.Summary.Imported)
}

func TestLegacyRunHistorySupportedStates(t *testing.T) {
	for _, state := range []string{"", "RUNTIME_STATE_UNSPECIFIED", "PENDING", "RUNNING", "SUCCEEDED", "SKIPPED", "FAILED", "CANCELING", "CANCELED", "PAUSED", "Pending", "Running", "Succeeded", "Skipped", "Terminating", "Failed", "Error", "Unknown", "NO_STATUS", "ENABLED", "DISABLED", "Ready", "Done"} {
		t.Run(state, func(t *testing.T) {
			data, err := json.Marshal([]*legacyRuntimeStatus{{State: state}})
			require.NoError(t, err)
			require.NoError(t, validateLegacyRunStateHistory(data))
		})
	}
}

func TestLegacyArchivePreservesPendingTaskHistory(t *testing.T) {
	for _, fixture := range []string{"legacy-218-v2-export.json", "legacy-218-mlmd-v2-export.json"} {
		t.Run(fixture, func(t *testing.T) {
			old := readLegacyGolden(t, fixture)
			pending := &legacyRuntimeStatus{State: "PENDING", UpdateTimeInSec: 123, Error: json.RawMessage(`{"code":14,"message":"waiting for executor"}`)}
			old.Runs[0].Tasks[0].StateHistory = []*legacyRuntimeStatus{pending, {State: "SUCCEEDED", UpdateTimeInSec: 125}}
			old.Runs[0].Tasks[0].StateHistoryString = ""
			out, _, err := DecodeNamespaceArchive(database(t), encodeLegacy(t, old), "team", "team", false)
			require.NoError(t, err)
			found := false
			for _, task := range out.Entries[0].Tasks {
				custom, _ := task.StatusMetadata["custom_properties"].(model.JSONData)
				raw, _ := custom["legacy_sql_task"].(string)
				if raw == "" {
					continue
				}
				var original legacyTask
				require.NoError(t, json.Unmarshal([]byte(raw), &original))
				if original.UUID != old.Runs[0].Tasks[0].UUID {
					continue
				}
				found = true
				require.Equal(t, "PENDING", original.StateHistory[0].State)
				require.EqualValues(t, 123, original.StateHistory[0].UpdateTimeInSec)
				require.JSONEq(t, string(pending.Error), string(original.StateHistory[0].Error))
				require.Len(t, task.StateHistory, 2)
				status := task.StateHistory[0].(map[string]any)
				require.EqualValues(t, float64(14), status["error"].(map[string]any)["code"])
				require.Equal(t, "waiting for executor", status["error"].(map[string]any)["message"])
			}
			require.True(t, found)
			if fixture == "legacy-218-v2-export.json" {
				old.Runs[0].Tasks[0].State = "PENDING"
				_, _, err = DecodeNamespaceArchive(database(t), encodeLegacy(t, old), "team", "team", false)
				require.ErrorContains(t, err, "unsupported legacy task state")
			}
		})
	}
}
