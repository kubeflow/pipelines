// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

package transfer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func ancestorRootFixture(t *testing.T) (*Engine, *memoryMetadata) {
	t.Helper()
	db := testDB(t)
	engine := fixture(t, db)
	ancestor := model.Run{UUID: "ancestor", Namespace: "team", ExperimentId: "experiment", RunDetails: model.RunDetails{State: model.RuntimeStateSucceeded, FinishedAtInSec: 10}}
	require.NoError(t, create(db, &ancestor))
	graph := lineageGraph("run")
	graph["contexts"] = append(graph["contexts"], Node{"id": "8", "type_id": "1", "name": "ancestor"})
	graph["execution_types"] = append(graph["execution_types"], Node{"id": "7", "name": "system.DAGExecution"})
	graph["executions"][1] = Node{"id": "11", "type_id": "7", "name": "run/ancestor", "last_known_state": "RUNNING"}
	graph["associations"] = append(graph["associations"], Node{"context_id": "8", "execution_id": "11"})
	store := &memoryMetadata{graph: graph}
	engine.Metadata = &Metadata{RPC: ProtoRPC{Conn: descriptorConn{store}}}
	return engine, store
}

func TestExportCompletedAncestorRootOutsideWindow(t *testing.T) {
	for _, contextID := range []int64{0, 8} {
		t.Run(fmt.Sprint(contextID), func(t *testing.T) {
			engine, store := ancestorRootFixture(t)
			require.NoError(t, engine.DB.Model(&model.Run{}).Where("UUID = ?", "ancestor").Update("PipelineRunContextId", contextID).Error)
			data, err := engine.Export(context.Background(), "team", ExportOptions{CompletedAfter: 20})
			require.NoError(t, err)
			var bundle Bundle
			require.NoError(t, json.Unmarshal(data, &bundle))
			require.Len(t, bundle.Runs, 1, "ancestor history must not widen the selected window")
			require.Equal(t, "run", bundle.Runs[0].Run.UUID)
			found := false
			for _, execution := range bundle.Metadata["executions"] {
				if sid(execution["id"]) == "11" {
					found = true
					require.Equal(t, "COMPLETE", execution["last_known_state"])
				}
			}
			require.True(t, found)
			require.Equal(t, "RUNNING", store.graph["executions"][1]["last_known_state"])
			require.Zero(t, store.puts)
			var sourceRun model.Run
			require.NoError(t, engine.DB.Where("UUID = ?", "ancestor").First(&sourceRun).Error)
			require.Equal(t, contextID, sourceRun.PipelineRunContextId)
		})
	}
}

func TestExportAncestorRootRequiresCompletionEvidence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*testing.T, *Engine, *memoryMetadata)
		want   string
	}{
		{"active", func(t *testing.T, e *Engine, _ *memoryMetadata) {
			require.NoError(t, e.DB.Model(&model.Run{}).Where("UUID = ?", "ancestor").Update("State", model.RuntimeStateRunning).Error)
		}, "unfinished execution"},
		{"mismatched context", func(t *testing.T, e *Engine, _ *memoryMetadata) {
			require.NoError(t, e.DB.Model(&model.Run{}).Where("UUID = ?", "ancestor").Update("PipelineRunContextId", 99).Error)
		}, "unfinished execution"},
		{"ambiguous run contexts", func(t *testing.T, _ *Engine, store *memoryMetadata) {
			store.graph["contexts"] = append(store.graph["contexts"], Node{"id": "9", "type_id": "1", "name": "ancestor"})
			store.graph["associations"] = append(store.graph["associations"], Node{"context_id": "9", "execution_id": "11"})
		}, "unfinished execution"},
		{"actual task", func(t *testing.T, e *Engine, _ *memoryMetadata) {
			require.NoError(t, create(e.DB, &model.Task{UUID: "ancestor-task", RunID: "ancestor", Namespace: "team", MLMDExecutionID: "11", State: model.RuntimeStateRunning}))
		}, "unfinished execution"},
		{"cross namespace", func(t *testing.T, e *Engine, _ *memoryMetadata) {
			require.NoError(t, e.DB.Model(&model.Run{}).Where("UUID = ?", "ancestor").Update("Namespace", "private").Error)
		}, "crosses namespaces"},
		{"cross experiment namespace", func(t *testing.T, e *Engine, _ *memoryMetadata) {
			require.NoError(t, e.DB.Model(&model.Run{}).Where("UUID = ?", "ancestor").Update("ExperimentId", "other").Error)
		}, "crosses experiment namespaces"},
		{"missing SQL", func(t *testing.T, e *Engine, _ *memoryMetadata) {
			require.NoError(t, e.DB.Where("UUID = ?", "ancestor").Delete(&model.Run{}).Error)
		}, "restore its producer history"},
		{"provenance alone", func(t *testing.T, e *Engine, store *memoryMetadata) {
			require.NoError(t, e.DB.Where("UUID = ?", "ancestor").Delete(&model.Run{}).Error)
			store.graph["contexts"][len(store.graph["contexts"])-1]["custom_properties"] = Node{
				provenance + "namespace": Node{"string_value": "team"},
				provenance + "source":    Node{"string_value": "source"},
			}
		}, "unfinished execution"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			engine, store := ancestorRootFixture(t)
			tc.change(t, engine, store)
			_, err := engine.Export(context.Background(), "team", ExportOptions{CompletedAfter: 20})
			require.ErrorContains(t, err, tc.want)
			require.Equal(t, "RUNNING", store.graph["executions"][1]["last_known_state"])
			require.Zero(t, store.puts)
		})
	}
}

func TestExportAncestorTaskReadFailureCannotUseProvenanceFallback(t *testing.T) {
	engine, store := ancestorRootFixture(t)
	store.graph["contexts"][len(store.graph["contexts"])-1]["custom_properties"] = Node{
		provenance + "namespace": Node{"string_value": "team"},
		provenance + "source":    Node{"string_value": "source"},
	}
	readErr := errors.New("ancestor task read failed")
	require.NoError(t, engine.DB.Callback().Row().Before("gorm:row").Register("fail_ancestor_tasks", func(db *gorm.DB) {
		if db.Statement.Table != "tasks" {
			return
		}
		where, ok := db.Statement.Clauses["WHERE"].Expression.(clause.Where)
		if !ok {
			return
		}
		for _, expression := range where.Exprs {
			if eq, ok := expression.(clause.Eq); ok && eq.Value == "ancestor" {
				db.AddError(readErr)
			}
		}
	}))
	_, err := engine.Export(context.Background(), "team", ExportOptions{CompletedAfter: 20})
	require.ErrorIs(t, err, readErr)
	require.Zero(t, store.puts)
}
