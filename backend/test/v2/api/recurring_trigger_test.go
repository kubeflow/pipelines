// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import (
	"testing"
	"time"

	"github.com/go-openapi/strfmt"
	recurringmodel "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/recurring_run_model"
)

func TestRecurringTriggerComparison(t *testing.T) {
	start := time.Unix(36000, 0).UTC()
	end := start.Add(2 * time.Minute)
	for _, periodic := range []bool{true, false} {
		makeTrigger := func(start, end time.Time) *recurringmodel.V2beta1Trigger {
			if periodic {
				return &recurringmodel.V2beta1Trigger{PeriodicSchedule: &recurringmodel.V2beta1PeriodicSchedule{
					StartTime: strfmt.DateTime(start), EndTime: strfmt.DateTime(end), IntervalSecond: 60,
				}}
			}
			return &recurringmodel.V2beta1Trigger{CronSchedule: &recurringmodel.V2beta1CronSchedule{
				StartTime: strfmt.DateTime(start), EndTime: strfmt.DateTime(end), Cron: "0 * * * * ?",
			}}
		}
		expected := makeTrigger(start, end)
		changedSchedule := makeTrigger(start, end)
		if periodic {
			changedSchedule.PeriodicSchedule.IntervalSecond++
		} else {
			changedSchedule.CronSchedule.Cron = "0 */2 * * * ?"
		}
		for _, tc := range []struct {
			name   string
			actual *recurringmodel.V2beta1Trigger
			want   bool
		}{
			{"UTC round trip", makeTrigger(start, end), true},
			{"local UTC round trip", makeTrigger(start.In(time.FixedZone("Local", 0)), end.In(time.FixedZone("Local", 0))), true},
			{"offset round trip", makeTrigger(start.In(time.FixedZone("offset", 3600)), end.In(time.FixedZone("offset", 3600))), true},
			{"changed start", makeTrigger(start.Add(time.Second), end), false},
			{"changed end", makeTrigger(start, end.Add(time.Second)), false},
			{"changed schedule", changedSchedule, false},
			{"missing schedule", nil, false},
		} {
			t.Run(tc.name, func(t *testing.T) {
				got, err := matchRecurringTrigger(expected).Match(tc.actual)
				if err != nil || got != tc.want {
					t.Fatalf("periodic=%v: match=%v, error=%v; want %v", periodic, got, err, tc.want)
				}
			})
		}
	}
	for _, expected := range []*recurringmodel.V2beta1Trigger{nil, {}} {
		for _, actual := range []*recurringmodel.V2beta1Trigger{nil, {}} {
			if got, err := matchRecurringTrigger(expected).Match(actual); err != nil || !got {
				t.Fatalf("absent trigger: match=%v, error=%v", got, err)
			}
		}
	}
}
