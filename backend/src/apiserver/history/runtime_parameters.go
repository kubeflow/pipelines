// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

package history

import (
	"encoding/json"
	"unicode/utf8"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

func restoreNativeRuntimeParameters(b *NamespaceBundle) ([]string, error) {
	if b.Format == "kfp-namespace-transfer/v1" {
		if b.RuntimeParameters != nil {
			return nil, util.NewInvalidInputError("v1 archive cannot contain v2 runtime parameters")
		}
		if len(b.Schedules) != 0 {
			return nil, util.NewInvalidInputError("v1 archives omit schedule runtime parameters; update the source transfer exporter and export again")
		}
		b.Format = TransferFormat
		return []string{"This older archive did not retain V2 run runtime parameter overrides. Re-export with an updated source to preserve them."}, nil
	}
	if b.Format != TransferFormat {
		return nil, util.NewInvalidInputError("unsupported namespace archive format")
	}
	p := b.RuntimeParameters
	if p == nil || len(p.Runs) != len(b.Entries) || len(p.Schedules) != len(b.Schedules) {
		return nil, util.NewInvalidInputError("runtime parameter maps must cover every archived run and schedule")
	}
	for i := range b.Entries {
		value, ok := p.Runs[b.Entries[i].Run.UUID]
		if !ok || !validRuntimeParameters(value) {
			return nil, util.NewInvalidInputError("invalid or missing run runtime parameters")
		}
		b.Entries[i].Run.RuntimeConfig.Parameters = model.LargeText(value)
	}
	for i := range b.Schedules {
		value, ok := p.Schedules[b.Schedules[i].UUID]
		if !ok || !validRuntimeParameters(value) {
			return nil, util.NewInvalidInputError("invalid or missing schedule runtime parameters")
		}
		b.Schedules[i].RuntimeConfig.Parameters = model.LargeText(value)
	}
	return nil, nil
}

func validRuntimeParameters(value string) bool {
	if value == "" {
		return true
	}
	var object map[string]json.RawMessage
	return utf8.ValidString(value) && json.Unmarshal([]byte(value), &object) == nil && object != nil
}
