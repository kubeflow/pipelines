// Copyright 2018 Google LLC
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
	"math"
	"time"

	swfapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	log "github.com/sirupsen/logrus"
)

// PeriodicSchedule is a type to help manipulate PeriodicSchedule objects.
type PeriodicSchedule struct {
	*swfapi.PeriodicSchedule
}

// NewPeriodicSchedule creates a new PeriodicSchedule.
func NewPeriodicSchedule(periodicSchedule *swfapi.PeriodicSchedule) *PeriodicSchedule {
	if periodicSchedule == nil {
		log.Fatalf("The periodicSchedule should never be nil")
	}

	return &PeriodicSchedule{
		periodicSchedule,
	}
}

// GetNextScheduledEpoch returns the next epoch at which a workflow should be
// scheduled.
func (s *PeriodicSchedule) GetNextScheduledEpoch(lastJobEpoch *int64,
	defaultStartEpoch int64) int64 {
	fmt.Println("Default start")
	fmt.Println(time.Unix(defaultStartEpoch, 0).UTC())
	effectiveLastJobEpoch := defaultStartEpoch
	if lastJobEpoch != nil {
		effectiveLastJobEpoch = *lastJobEpoch
		fmt.Println("lastJobEpoch")
		fmt.Println(time.Unix(effectiveLastJobEpoch, 0))
	} else if s.StartTime != nil {
		fmt.Println("Start time?")
		effectiveLastJobEpoch = s.StartTime.Unix()
	}
	fmt.Println("Effective last epoch")
	fmt.Println(effectiveLastJobEpoch)
	fmt.Println(time.Unix(effectiveLastJobEpoch, 0).UTC())
	return s.getNextScheduledEpoch(effectiveLastJobEpoch)
}

func (s *PeriodicSchedule) getNextScheduledEpoch(lastJobEpoch int64) int64 {
	startEpoch := lastJobEpoch
	if s.StartTime != nil && s.StartTime.Unix() > startEpoch {
		startEpoch = s.StartTime.Unix()
	}

	result := startEpoch + s.getInterval()

	if s.EndTime != nil &&
		s.EndTime.Unix() < result {
		return math.MaxInt64
	}

	return result
}

func (s *PeriodicSchedule) getInterval() int64 {
	interval := s.IntervalSecond
	if interval == 0 {
		interval = 1
	}
	return interval
}

func (s *PeriodicSchedule) GetNextScheduledEpochNoCatchup(
	lastJobEpoch *int64, defaultStartEpoch int64, nowEpoch int64) int64 {

	nextScheduledEpoch := s.GetNextScheduledEpoch(lastJobEpoch, defaultStartEpoch)
	if nextScheduledEpoch == math.MaxInt64 {
		// No next schedule.
		fmt.Println("Inside here ")
		fmt.Println("Max we have ...")
		return math.MaxInt64
	}

	nextNextScheduledEpoch := nextScheduledEpoch + s.getInterval()

	if nowEpoch >= nextNextScheduledEpoch {
		// If we cannot catch up with schedule, just reschedule to min(now, endTime).
		if s.EndTime != nil && s.EndTime.Unix() < nowEpoch {
			return s.EndTime.Unix()
		}
		return nowEpoch
	}
	fmt.Println("HERE HERE HERE")
	fmt.Println("HERE WE GO!!!")
	fmt.Println(nextScheduledEpoch)
	return nextScheduledEpoch
}
