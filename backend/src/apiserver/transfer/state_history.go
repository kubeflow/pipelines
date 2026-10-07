// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

package transfer

import (
	"encoding/json"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

// Imported history must remain readable through the ordinary run/task stores.
func validateStateHistory(raw model.LargeText) error {
	if raw == "" {
		return nil
	}
	var statuses []*model.RuntimeStatus
	if err := json.Unmarshal([]byte(raw), &statuses); err != nil {
		return util.NewInvalidInputError("Invalid persisted state history")
	}
	for _, status := range statuses {
		if status == nil {
			return util.NewInvalidInputError("Invalid null status in persisted state history")
		}
	}
	return nil
}
