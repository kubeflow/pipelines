// Copyright 2025 The Kubeflow Authors
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
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestGetMetricValue(t *testing.T) {
	t.Run("gauge", func(t *testing.T) {
		gauge := prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "test_gauge",
			Help: "A test gauge",
		})
		gauge.Set(42.5)

		value := GetMetricValue(gauge)
		assert.Equal(t, 42.5, value)
	})

	t.Run("counter", func(t *testing.T) {
		counter := prometheus.NewCounter(prometheus.CounterOpts{
			Name: "test_counter",
			Help: "A test counter",
		})
		counter.Add(10)

		value := GetMetricValue(counter)
		assert.Equal(t, float64(10), value)
	})

	t.Run("histogram is invalid type", func(t *testing.T) {
		histogram := prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "test_histogram",
			Help:    "A test histogram",
			Buckets: prometheus.DefBuckets,
		})
		histogram.Observe(1.5)

		value := GetMetricValue(histogram)
		assert.Equal(t, float64(0), value)
	})
}
