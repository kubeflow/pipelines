// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

package history

import (
	"encoding/json"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/require"
)

func TestCreateRecordPreservesColumnWithIgnoredFieldName(t *testing.T) {
	db := database(t)
	create(t, db, &model.Experiment{UUID: "experiment", Name: "history", Namespace: "team"})
	run := model.Run{UUID: "run", ExperimentId: "experiment", Namespace: "team", RunDetails: model.RunDetails{
		StateHistoryString: `[{"UpdateTimeInSec":1700000000,"State":"FAILED","Error":{"code":13,"message":"source error"}}]`,
	}}
	values, _, err := persistedValues(db, &run)
	require.NoError(t, err)
	require.Equal(t, run.StateHistoryString, values["StateHistory"])
	require.NoError(t, createRecord(db, &run))
	var stored model.Run
	require.NoError(t, db.Take(&stored).Error)
	require.Equal(t, run.StateHistoryString, stored.StateHistoryString,
		"the StateHistory column must not resolve to the ignored hydrated StateHistory field")
	task := model.Task{UUID: "task", RunUUID: run.UUID, Namespace: run.Namespace, Pods: model.JSONSlice{}, TypeAttrs: model.JSONData{},
		StateHistory: model.JSONSlice{map[string]any{"state": "FAILED", "large_integer": json.Number("9007199254740993")}},
	}
	require.NoError(t, createRecord(db, &task))
	var tasks []model.Task
	require.NoError(t, readRows(db, &tasks))
	require.Len(t, tasks, 1)
	require.Equal(t, task.StateHistory, tasks[0].StateHistory)
}
