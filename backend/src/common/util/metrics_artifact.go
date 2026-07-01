// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	api "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	// More than 50 metrics is not scalable with the current UI design.
	maxMetricsCountLimit = 50
	// This matches the metric-name validation in the v1 API server.
	maxMetricNameLength = 64
)

func decodeRunMetrics(reader io.Reader) ([]*api.RunMetric, error) {
	decoder := json.NewDecoder(reader)

	token, err := decoder.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to read metrics object: %w", err)
	}
	if delimiter, ok := token.(json.Delim); !ok || delimiter != '{' {
		return nil, fmt.Errorf("metrics file must contain a JSON object")
	}

	var metrics []*api.RunMetric
	metricsSeen := false
	for decoder.More() {
		fieldToken, err := decoder.Token()
		if err != nil {
			return nil, fmt.Errorf("failed to read metrics field: %w", err)
		}
		fieldName, ok := fieldToken.(string)
		if !ok {
			return nil, fmt.Errorf("metrics field name must be a string")
		}

		switch fieldName {
		case "metrics":
			if metricsSeen {
				return nil, fmt.Errorf("metrics field must not be repeated")
			}
			metricsSeen = true
			metrics, err = decodeMetricsArray(decoder)
			if err != nil {
				return nil, err
			}
		case "runId", "run_id":
			var ignoredRunID string
			if err := decoder.Decode(&ignoredRunID); err != nil {
				return nil, fmt.Errorf("failed to decode %s: %w", fieldName, err)
			}
		default:
			return nil, fmt.Errorf("unknown metrics field %q", fieldName)
		}
	}

	if _, err := decoder.Token(); err != nil {
		return nil, fmt.Errorf("failed to finish metrics object: %w", err)
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err == nil {
		return nil, fmt.Errorf("unexpected content after metrics object")
	} else if err != io.EOF {
		return nil, fmt.Errorf("failed to finish metrics file: %w", err)
	}
	return metrics, nil
}

func decodeMetricsArray(decoder *json.Decoder) ([]*api.RunMetric, error) {
	token, err := decoder.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to read metrics array: %w", err)
	}
	if delimiter, ok := token.(json.Delim); !ok || delimiter != '[' {
		return nil, fmt.Errorf("metrics field must be an array")
	}

	metrics := make([]*api.RunMetric, 0, maxMetricsCountLimit)
	for decoder.More() {
		if len(metrics) >= maxMetricsCountLimit {
			return nil, fmt.Errorf("metrics file contains more than %d metrics", maxMetricsCountLimit)
		}

		var rawMetric json.RawMessage
		if err := decoder.Decode(&rawMetric); err != nil {
			return nil, fmt.Errorf("failed to decode metric: %w", err)
		}
		transformedMetric, err := transformJSONForBackwardCompatibility(string(rawMetric))
		if err != nil {
			return nil, err
		}
		metric := new(api.RunMetric)
		if err := protojson.Unmarshal([]byte(transformedMetric), metric); err != nil {
			return nil, fmt.Errorf("failed to decode metric: %w", err)
		}
		if len(metric.GetName()) > maxMetricNameLength {
			return nil, fmt.Errorf("metric name cannot exceed %d characters", maxMetricNameLength)
		}
		metrics = append(metrics, metric)
	}
	if _, err := decoder.Token(); err != nil {
		return nil, fmt.Errorf("failed to finish metrics array: %w", err)
	}
	return metrics, nil
}

// Previously number_value for RunMetrics in backend/api/v1beta1/run.proto
// allowed camelCase field values in JSON. Keep accepting both forms.
func transformJSONForBackwardCompatibility(jsonString string) (string, error) {
	replacer := strings.NewReplacer(
		`"numberValue":`, `"number_value":`,
	)
	return replacer.Replace(jsonString), nil
}
