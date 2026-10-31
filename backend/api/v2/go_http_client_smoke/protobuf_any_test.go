// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0.

package go_http_client_smoke_test

import (
	"encoding/json"
	"testing"

	artifact "github.com/kubeflow/pipelines/backend/api/v2/go_http_client/artifact_model"
	experiment "github.com/kubeflow/pipelines/backend/api/v2/go_http_client/experiment_model"
	healthz "github.com/kubeflow/pipelines/backend/api/v2/go_http_client/healthz_model"
	pipeline "github.com/kubeflow/pipelines/backend/api/v2/go_http_client/pipeline_model"
	recurringrun "github.com/kubeflow/pipelines/backend/api/v2/go_http_client/recurring_run_model"
	run "github.com/kubeflow/pipelines/backend/api/v2/go_http_client/run_model"
	legacyartifact "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/artifact_model"
	legacyexperiment "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/experiment_model"
	legacyhealthz "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/healthz_model"
	legacypipeline "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_model"
	legacyrecurringrun "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/recurring_run_model"
	legacyrun "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_model"
	"github.com/stretchr/testify/require"
)

type anyJSON interface {
	json.Marshaler
	json.Unmarshaler
}

func TestProtobufAnySourceAndJSONCompatibility(t *testing.T) {
	const typeURL = "type.googleapis.com/example.Message"
	const wireJSON = `{"@type":"type.googleapis.com/example.Message","text":"value","number":42,"enabled":true,"nested":{"field":"value"},"items":["item",7],"nullable":null}`
	properties := map[string]any{
		"text":     "value",
		"number":   42,
		"enabled":  true,
		"nested":   map[string]any{"field": "value"},
		"items":    []any{"item", 7},
		"nullable": nil,
	}
	// Typed literals guard the exported field used by downstream Go clients,
	// including users of the legacy aliases, independently of JSON behavior.
	tests := []struct {
		name string
		new  func(map[string]any) anyJSON
	}{
		{"artifact", func(p map[string]any) anyJSON { return &artifact.ProtobufAny{AtType: typeURL, ProtobufAny: p} }},
		{"experiment", func(p map[string]any) anyJSON { return &experiment.ProtobufAny{AtType: typeURL, ProtobufAny: p} }},
		{"healthz", func(p map[string]any) anyJSON { return &healthz.ProtobufAny{AtType: typeURL, ProtobufAny: p} }},
		{"pipeline", func(p map[string]any) anyJSON { return &pipeline.ProtobufAny{AtType: typeURL, ProtobufAny: p} }},
		{"recurring_run", func(p map[string]any) anyJSON { return &recurringrun.ProtobufAny{AtType: typeURL, ProtobufAny: p} }},
		{"run", func(p map[string]any) anyJSON { return &run.ProtobufAny{AtType: typeURL, ProtobufAny: p} }},
		{"legacy_artifact", func(p map[string]any) anyJSON { return &legacyartifact.ProtobufAny{AtType: typeURL, ProtobufAny: p} }},
		{"legacy_experiment", func(p map[string]any) anyJSON { return &legacyexperiment.ProtobufAny{AtType: typeURL, ProtobufAny: p} }},
		{"legacy_healthz", func(p map[string]any) anyJSON { return &legacyhealthz.ProtobufAny{AtType: typeURL, ProtobufAny: p} }},
		{"legacy_pipeline", func(p map[string]any) anyJSON { return &legacypipeline.ProtobufAny{AtType: typeURL, ProtobufAny: p} }},
		{"legacy_recurring_run", func(p map[string]any) anyJSON {
			return &legacyrecurringrun.ProtobufAny{AtType: typeURL, ProtobufAny: p}
		}},
		{"legacy_run", func(p map[string]any) anyJSON { return &legacyrun.ProtobufAny{AtType: typeURL, ProtobufAny: p} }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := json.Marshal(tt.new(properties))
			require.NoError(t, err)
			require.JSONEq(t, wireJSON, string(encoded))

			decoded := tt.new(nil)
			require.NoError(t, json.Unmarshal([]byte(wireJSON), decoded))
			roundTrip, err := json.Marshal(decoded)
			require.NoError(t, err)
			require.JSONEq(t, wireJSON, string(roundTrip))
		})
	}
}
