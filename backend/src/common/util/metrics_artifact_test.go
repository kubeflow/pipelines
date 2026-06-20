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
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeRunMetrics_StreamsCompatibleMetrics(t *testing.T) {
	metrics, err := decodeRunMetrics(strings.NewReader(`{
		"runId": "ignored",
		"metrics": [
			{"name": "accuracy", "numberValue": 0.95},
			{"name": "loss", "number_value": 1.25, "format": "RAW"}
		]
	}`))

	require.NoError(t, err)
	require.Len(t, metrics, 2)
	assert.Equal(t, "accuracy", metrics[0].GetName())
	assert.Equal(t, 0.95, metrics[0].GetNumberValue())
	assert.Equal(t, "loss", metrics[1].GetName())
	assert.Equal(t, 1.25, metrics[1].GetNumberValue())
}

func TestDecodeRunMetrics_EnforcesMetricCount(t *testing.T) {
	var input strings.Builder
	input.WriteString(`{"metrics":[`)
	for index := 0; index <= maxMetricsCountLimit; index++ {
		if index > 0 {
			input.WriteByte(',')
		}
		fmt.Fprintf(&input, `{"name":"metric-%d","numberValue":%d}`, index, index)
	}
	input.WriteString(`]}`)

	metrics, err := decodeRunMetrics(strings.NewReader(input.String()))

	assert.Nil(t, metrics)
	assert.ErrorContains(t, err, "metrics file contains more than 50 metrics")
}

func TestDecodeRunMetrics_EnforcesMetricNameLength(t *testing.T) {
	testCases := []struct {
		name          string
		metricName    string
		errorContains string
	}{
		{name: "exact limit", metricName: strings.Repeat("a", maxMetricNameLength)},
		{
			name:          "over limit",
			metricName:    strings.Repeat("a", maxMetricNameLength+1),
			errorContains: "metric name cannot exceed 64 characters",
		},
		{
			name:       "pattern validation remains downstream",
			metricName: "log loss",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			input := fmt.Sprintf(`{"metrics":[{"name":%q,"numberValue":1}]}`, testCase.metricName)
			metrics, err := decodeRunMetrics(strings.NewReader(input))

			if testCase.errorContains == "" {
				require.NoError(t, err)
				require.Len(t, metrics, 1)
				assert.Equal(t, testCase.metricName, metrics[0].GetName())
			} else {
				assert.Nil(t, metrics)
				assert.ErrorContains(t, err, testCase.errorContains)
			}
		})
	}
}

func TestDecodeRunMetrics_RejectsUnknownFieldsAndTrailingJSON(t *testing.T) {
	testCases := []struct {
		name          string
		input         string
		errorContains string
	}{
		{
			name:          "null metrics",
			input:         `{"metrics":null}`,
			errorContains: "metrics field must be an array",
		},
		{
			name:          "unknown top-level field",
			input:         `{"metrics":[],"unknown":true}`,
			errorContains: `unknown metrics field "unknown"`,
		},
		{
			name:          "unknown metric field",
			input:         `{"metrics":[{"name":"accuracy","unknown":true}]}`,
			errorContains: "unknown field",
		},
		{
			name:          "trailing JSON value",
			input:         `{"metrics":[]} {}`,
			errorContains: "unexpected content after metrics object",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			metrics, err := decodeRunMetrics(strings.NewReader(testCase.input))

			assert.Nil(t, metrics)
			assert.ErrorContains(t, err, testCase.errorContains)
		})
	}
}
