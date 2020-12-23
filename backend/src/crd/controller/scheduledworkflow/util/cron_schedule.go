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
	"time"

	swfapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	wraperror "github.com/pkg/errors"
	"github.com/robfig/cron"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// The maximum time that can be used in time.Time
// For more information check: https://stackoverflow.com/questions/25065055/what-is-the-maximum-time-time-in-go
var maxTime = time.Unix(1<<63-62135596801, 999999999)

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

// GetNextScheduledTime returns the next epoch at which a workflow must be
// scheduled.
func (s *CronSchedule) GetNextScheduledTime(lastJobTime *v1.Time,
	defaultStartTime time.Time, location *time.Location) time.Time {
	effectiveLastJobTime := s.getEffectiveLastJobTime(lastJobTime, defaultStartTime)
	return s.getNextScheduledTime(effectiveLastJobTime, location)
}

func (s *CronSchedule) GetNextScheduledTimeNoCatchup(lastJobTime *v1.Time,
	defaultStartTime time.Time, nowTime time.Time, location *time.Location) time.Time {

	effectiveLastJobTime := s.getEffectiveLastJobTime(lastJobTime, defaultStartTime)
	return s.getNextScheduledTimeImp(effectiveLastJobTime, false, nowTime, location)
}

func (s *CronSchedule) getEffectiveLastJobTime(lastJobTime *v1.Time,
	defaultStartTime time.Time) time.Time {

	// Fallback to default start epoch, which will be passed the Job creation
	// time.
	effectiveLastJobTime := defaultStartTime
	if lastJobTime != nil {
		// Last job epoch takes first precedence.
		effectiveLastJobTime = lastJobTime.Time
	} else if s.StartTime != nil {
		// Start time takes second precedence.
		effectiveLastJobTime = s.StartTime.Time
	}
	return effectiveLastJobTime
}

func (s *CronSchedule) getNextScheduledTime(lastJobTime time.Time, location *time.Location) time.Time {
	return s.getNextScheduledTimeImp(lastJobTime,
		true /* nowEpoch doesn't matter when catchup=true */, time.Unix(0, 0), location)
}

func (s *CronSchedule) getNextScheduledTimeImp(lastJobTime time.Time, catchup bool, nowTime time.Time, location *time.Location) time.Time {
	schedule, err := cron.Parse(s.Cron)
	if err != nil {
		// This should never happen, validation should have caught this at resource creation.
		log.Errorf("%+v", wraperror.Errorf(
			"Found invalid schedule (%v): %v", s.Cron, err))
		return maxTime.In(location)
	}

	startTime := lastJobTime
	if s.StartTime != nil && s.StartTime.Time.After(startTime) {
		startTime = s.StartTime.Time
	}

	result := schedule.Next(startTime.In(location))
	var endTime time.Time = maxTime
	if s.EndTime != nil {
		endTime = s.EndTime.Time
	}

	if endTime.Before(result) {
		return maxTime.In(location)
	}

	// When we need to catch up with schedule, just run schedules one by one.
	if catchup == true {
		return result
	}

	// When we don't need to catch up, find the last schedule we need to run
	// now and skip others in between.
	next := result
	var nextNext time.Time
	for {
		nextNext = schedule.Next(next)
		if !nextNext.After(nowTime) && !nextNext.After(endTime) {
			next = nextNext
		} else {
			break
		}
	}
	return next
}
