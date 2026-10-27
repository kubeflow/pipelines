// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

package history

import (
	"encoding/json"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/transfer"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/clause"
)

func TestTransferReaderStopsAtBudgetAndPreservesNumbers(t *testing.T) {
	db := database(t)
	for _, id := range []string{"a", "b", "c"} {
		create(t, db, &model.Artifact{UUID: id, Namespace: "team", Metadata: model.JSONData{"integer": json.Number("9007199254740993")}})
	}
	var first []model.Artifact
	require.NoError(t, readTransferRows(db.Order(clause.OrderByColumn{Column: clause.Column{Name: "UUID"}}).Limit(1), &first, transfer.NewExportBudget(transfer.MaxArchiveBytes)))
	require.Equal(t, json.Number("9007199254740993"), first[0].Metadata["integer"])
	encoded, err := json.Marshal(first[0])
	require.NoError(t, err)
	// A malformed third row makes eager decoding fail before a budget check.
	require.NoError(t, db.Model(&model.Artifact{}).Where(clause.Eq{Column: "UUID", Value: "c"}).Update("Metadata", "invalid-json").Error)
	var collected []model.Artifact
	err = readTransferRows(db.Order(clause.OrderByColumn{Column: clause.Column{Name: "UUID"}}), &collected, transfer.NewExportBudget(len(encoded)+2))
	require.ErrorContains(t, err, "transfer byte limit")
	require.Len(t, collected, 1)
	require.Equal(t, json.Number("9007199254740993"), collected[0].Metadata["integer"])
	// Exhaustion closes the cursor even with the single-connection test pool.
	require.Equal(t, int64(3), count(t, db, &model.Artifact{}))
}
