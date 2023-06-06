// Copyright 2018 The Kubeflow Authors
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
	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

type MetricsChan chan prometheus.Metric

// GetMetricValue get metric value from registered Collector
func GetMetricValue(collector prometheus.Collector) float64 {
	var total float64
	collectValue(collector, func(m dto.Metric) {
		// retrieves data if current collector is a gauge
		// if not then retrieves from a counter
		if gauge := m.GetGauge(); gauge != nil {
			total += m.GetGauge().GetValue()
		} else {
			if counter := m.GetCounter(); counter != nil {
				total += m.GetCounter().GetValue()
			} else {
				glog.Errorln("invalid type, only valid collectors are: gauge, counter")
				total = 0
			}
		}
	})
	return total
}

func collectValue(collector prometheus.Collector, do func(dto.Metric)) {
	c := make(MetricsChan)

	// collect calls the function for each metric associated with the Collector
	go func(c MetricsChan) {
		collector.Collect(c)
		close(c)
	}(c)

	// range across distinct label vector values
	for x := range c {
		m := dto.Metric{}
		_ = x.Write(&m)
		do(m)
	}
}
