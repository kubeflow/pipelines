// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

package history

import (
	"encoding/json"
	"unicode/utf8"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

func restoreNativeRuntimeParameters(b *NamespaceBundle) error {
	if b.Format != TransferFormat {
		return util.NewInvalidInputError("unsupported namespace archive format")
	}
	p := b.RuntimeParameters
	if p == nil || len(p.Runs) != len(b.Entries) || len(p.Schedules) != len(b.Schedules) {
		return util.NewInvalidInputError("runtime parameter maps must cover every archived run and schedule")
	}
	for i := range b.Entries {
		value, ok := p.Runs[b.Entries[i].Run.UUID]
		if !ok || !validRuntimeParameters(value) {
			return util.NewInvalidInputError("invalid or missing run runtime parameters")
		}
		b.Entries[i].Run.RuntimeConfig.Parameters = model.LargeText(value)
	}
	for i := range b.Schedules {
		value, ok := p.Schedules[b.Schedules[i].UUID]
		if !ok || !validRuntimeParameters(value) {
			return util.NewInvalidInputError("invalid or missing schedule runtime parameters")
		}
		b.Schedules[i].RuntimeConfig.Parameters = model.LargeText(value)
	}
	return nil
}

func validRuntimeParameters(value string) bool {
	if value == "" {
		return true
	}
	var object map[string]json.RawMessage
	return utf8.ValidString(value) && json.Unmarshal([]byte(value), &object) == nil && object != nil
}
