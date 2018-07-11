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
	swfapi "github.com/kubeflow/pipelines/pkg/apis/scheduledworkflow/v1alpha1"
	log "github.com/sirupsen/logrus"
	"math"
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
	effectiveLastJobEpoch := defaultStartEpoch
	if lastJobEpoch != nil {
		effectiveLastJobEpoch = *lastJobEpoch
	} else if s.StartTime != nil {
		effectiveLastJobEpoch = s.StartTime.Unix()
	}
	return s.getNextScheduledEpoch(effectiveLastJobEpoch)
}

func (s *PeriodicSchedule) getNextScheduledEpoch(lastJobEpoch int64) int64 {
	startEpoch := lastJobEpoch
	if s.StartTime != nil && s.StartTime.Unix() > startEpoch {
		startEpoch = s.StartTime.Unix()
	}

	interval := s.IntervalSecond
	if interval == 0 {
		interval = 1
	}

	result := startEpoch + interval

	if s.EndTime != nil &&
		s.EndTime.Unix() < result {
		return math.MaxInt64
	}

	return result
}
