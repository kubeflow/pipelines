// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

package transfer

import (
	"encoding/json"
	"unicode/utf8"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

func captureRuntimeParameters(b *Bundle) *RuntimeParameters {
	values := &RuntimeParameters{Runs: map[string]string{}, Schedules: map[string]string{}}
	for _, h := range b.Runs {
		values.Runs[h.Run.UUID] = string(h.Run.RuntimeConfig.Parameters)
	}
	for _, j := range b.Schedules {
		values.Schedules[j.UUID] = string(j.RuntimeConfig.Parameters)
	}
	return values
}

func restoreRuntimeParameters(b *Bundle) error {
	if b.Format == archiveFormatV1 {
		if b.RuntimeParameters != nil {
			return util.NewInvalidInputError("Version 1 archives cannot contain runtime_parameters")
		}
		if len(b.Schedules) > 0 {
			return util.NewInvalidInputError("Version 1 archives omitted schedule runtime parameters; re-export with the current API server before importing schedules")
		}
		return nil
	}
	if b.Format != archiveFormat {
		return util.NewInvalidInputError("Unsupported archive format")
	}
	values := b.RuntimeParameters
	if values == nil || len(values.Runs) != len(b.Runs) || len(values.Schedules) != len(b.Schedules) {
		return util.NewInvalidInputError("Version 2 archives require runtime_parameters for every run and schedule")
	}
	valid := func(value string, present bool) error {
		if !present {
			return util.NewInvalidInputError("Archive is missing a run or schedule runtime_parameters entry")
		}
		if value == "" {
			return nil
		}
		var parameters map[string]json.RawMessage
		if !utf8.ValidString(value) || json.Unmarshal([]byte(value), &parameters) != nil || parameters == nil {
			return util.NewInvalidInputError("Runtime parameters must be empty or a JSON object")
		}
		return nil
	}
	for i := range b.Runs {
		r := &b.Runs[i].Run
		value, present := values.Runs[r.UUID]
		if err := valid(value, present); err != nil {
			return err
		}
		r.RuntimeConfig.Parameters = model.LargeText(value)
	}
	for i := range b.Schedules {
		j := &b.Schedules[i]
		value, present := values.Schedules[j.UUID]
		if err := valid(value, present); err != nil {
			return err
		}
		j.RuntimeConfig.Parameters = model.LargeText(value)
	}
	return nil
}

// Preserve old receipts when overrides are empty, but bind every nonempty V2
// override explicitly because model JSON otherwise omits the embedded field.
func runtimeParametersDigest(value any, parameters string) any {
	if parameters == "" {
		return value
	}
	return struct {
		Value             any
		RuntimeParameters string
	}{value, parameters}
}
