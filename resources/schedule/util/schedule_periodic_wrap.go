// Copyright 2018 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"github.com/golang/glog"
	scheduleapi "github.com/kubeflow/pipelines/pkg/apis/schedule/v1alpha1"
	"math"
)

type PeriodicScheduleWrap struct {
	periodicSchedule *scheduleapi.PeriodicSchedule
}

func NewPeriodicScheduleWrap(periodicSchedule *scheduleapi.PeriodicSchedule) *PeriodicScheduleWrap {
	if periodicSchedule == nil {
		glog.Fatalf("The periodicSchedule should never be nil")
	}

	return &PeriodicScheduleWrap{
		periodicSchedule: periodicSchedule,
	}
}

func (s *PeriodicScheduleWrap) GetNextScheduledEpoch(lastJobEpoch *int64,
	defaultStartEpoch int64) int64 {
	effectiveLastJobEpoch := defaultStartEpoch
	if lastJobEpoch != nil {
		effectiveLastJobEpoch = *lastJobEpoch
	} else if s.periodicSchedule.StartTime != nil {
		effectiveLastJobEpoch = s.periodicSchedule.StartTime.Unix()
	}
	return s.getNextScheduledEpoch(effectiveLastJobEpoch)
}

func (s *PeriodicScheduleWrap) getNextScheduledEpoch(lastJobEpoch int64) int64 {
	startEpoch := lastJobEpoch
	if s.periodicSchedule.StartTime != nil && s.periodicSchedule.StartTime.Unix() > startEpoch {
		startEpoch = s.periodicSchedule.StartTime.Unix()
	}

	interval := s.periodicSchedule.IntervalSecond
	if interval == 0 {
		interval = 1
	}

	result := startEpoch + interval

	if s.periodicSchedule.EndTime != nil &&
		s.periodicSchedule.EndTime.Unix() < result {
		return math.MaxInt64
	}

	return result
}
