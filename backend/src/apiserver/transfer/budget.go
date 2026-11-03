// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

package transfer

import (
	"encoding/json"

	"github.com/kubeflow/pipelines/backend/src/common/util"
)

// ExportBudget bounds the cumulative encoded data retained during collection.
// A single source record is read before charging it; no subsequent records are
// read once the budget is exhausted. The final archive still has an exact check.
type ExportBudget struct{ remaining int }

// NewExportBudget creates a cumulative byte budget for source collection.
func NewExportBudget(limit int) *ExportBudget { return &ExportBudget{remaining: limit} }

// Add reserves one encoded record, including its collection separator.
func (b *ExportBudget) Add(value any) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return b.Reserve(len(encoded) + 1)
}

// Reserve accounts for archive framing that is not part of an individual row.
func (b *ExportBudget) Reserve(size int) error {
	if size < 0 || size > b.remaining {
		return util.NewInvalidInputError("Archive exceeds the transfer byte limit; narrow the completion time window")
	}
	b.remaining -= size
	return nil
}
