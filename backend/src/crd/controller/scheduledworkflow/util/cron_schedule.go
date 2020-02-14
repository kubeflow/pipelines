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
	"math"
	"time"

	swfapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	wraperror "github.com/pkg/errors"
	"github.com/robfig/cron"
	log "github.com/sirupsen/logrus"
)

// CronSchedule is a type to help manipulate CronSchedule objects.
type CronSchedule struct {
	*swfapi.CronSchedule
}

// NewCronSchedule creates a CronSchedule.
func NewCronSchedule(cronSchedule *swfapi.CronSchedule) *CronSchedule {
	if cronSchedule == nil {
		log.Fatalf("The cronSchedule should never be nil")
	}

	return &CronSchedule{
		cronSchedule,
	}
}

// GetNextScheduledEpoch returns the next epoch at which a workflow must be
// scheduled.
func (s *CronSchedule) GetNextScheduledEpoch(lastJobEpoch *int64,
	defaultStartEpoch int64) int64 {
	effectiveLastJobEpoch := s.getEffectiveLastJobEpoch(lastJobEpoch, defaultStartEpoch)
	return s.getNextScheduledEpoch(effectiveLastJobEpoch)
}

func (s *CronSchedule) GetNextScheduledEpochNoCatchup(lastJobEpoch *int64,
	defaultStartEpoch int64, nowEpoch int64) int64 {

	effectiveLastJobEpoch := s.getEffectiveLastJobEpoch(lastJobEpoch, defaultStartEpoch)
	return s.getNextScheduledEpochImp(effectiveLastJobEpoch, false, nowEpoch)
}

func (s *CronSchedule) getEffectiveLastJobEpoch(lastJobEpoch *int64,
	defaultStartEpoch int64) int64 {

	// Fallback to default start epoch, which will be passed the Job creation
	// time.
	effectiveLastJobEpoch := defaultStartEpoch
	if lastJobEpoch != nil {
		// Last job epoch takes first precedence.
		effectiveLastJobEpoch = *lastJobEpoch
	} else if s.StartTime != nil {
		// Start time takes second precedence.
		effectiveLastJobEpoch = s.StartTime.Unix()
	}
	return effectiveLastJobEpoch
}

func (s *CronSchedule) getNextScheduledEpoch(lastJobEpoch int64) int64 {
	return s.getNextScheduledEpochImp(lastJobEpoch,
		true, 0 /* nowEpoch doesn't matter when catchup=true */)
}

func (s *CronSchedule) getNextScheduledEpochImp(lastJobEpoch int64, catchup bool, nowEpoch int64) int64 {
	schedule, err := cron.Parse(s.Cron)
	if err != nil {
		// This should never happen, validation should have caught this at resource creation.
		log.Errorf("%+v", wraperror.Errorf(
			"Found invalid schedule (%v): %v", s.Cron, err))
		return math.MaxInt64
	}

	startEpoch := lastJobEpoch
	if s.StartTime != nil && s.StartTime.Unix() > startEpoch {
		startEpoch = s.StartTime.Unix()
	}
	result := schedule.Next(time.Unix(startEpoch, 0).UTC()).Unix()

	var endTime int64 = math.MaxInt64
	if s.EndTime != nil {
		endTime = s.EndTime.Unix()
	}
	if endTime < result {
		return math.MaxInt64
	}

	// When we need to catch up with schedule, just run schedules one by one.
	if catchup == true {
		return result
	}

	// When we don't need to catch up, find the last schedule we need to run
	// now and skip others in between.
	next := result
	var nextNext int64
	for {
		nextNext = schedule.Next(time.Unix(next, 0).UTC()).Unix()
		if nextNext <= nowEpoch && nextNext <= endTime {
			next = nextNext
		} else {
			break
		}
	}
	return next
}
