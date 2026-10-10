// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

package transfer

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExportBudgetChargesEscapedJSONCumulatively(t *testing.T) {
	budget := NewExportBudget(10)
	require.NoError(t, budget.Add("\n")) // JSON quotes, escape, separator = 5 bytes.
	require.NoError(t, budget.Add("\n"))
	require.ErrorContains(t, budget.Add(""), "transfer byte limit")
	require.ErrorContains(t, budget.Reserve(-1), "transfer byte limit")
}
