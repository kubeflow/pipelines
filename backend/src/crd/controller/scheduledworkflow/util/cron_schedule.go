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
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
func (s *CronSchedule) GetNextScheduledEpoch(lastJobTime *v1.Time,
	defaultStartTime time.Time, location *time.Location) int64 {
	effectiveLastJobTime := s.getEffectiveLastJobEpoch(lastJobTime, defaultStartTime, location)
	return s.getNextScheduledEpoch(effectiveLastJobTime, location)
}

func (s *CronSchedule) GetNextScheduledEpochNoCatchup(lastJobTime *v1.Time,
	defaultStartTime time.Time, nowTime time.Time, location *time.Location) int64 {

	effectiveLastJobTime := s.getEffectiveLastJobEpoch(lastJobTime, defaultStartTime, location)
	return s.getNextScheduledEpochImp(effectiveLastJobTime, false, nowTime, location)
}

func (s *CronSchedule) getEffectiveLastJobEpoch(lastJobTime *v1.Time,
	defaultStartTime time.Time, location *time.Location) time.Time {

	// Fallback to default start epoch, which will be passed the Job creation
	// time.
	effectiveLastJobTime := defaultStartTime
	if lastJobTime != nil {
		// Last job epoch takes first precedence.
		effectiveLastJobTime = lastJobTime.Time
	} else if s.StartTime != nil {
		// Start time takes second precedence.
		effectiveLastJobTime = s.StartTime.In(location)
	}
	return effectiveLastJobTime
}

func (s *CronSchedule) getNextScheduledEpoch(lastJobTime time.Time, location *time.Location) int64 {
	return s.getNextScheduledEpochImp(lastJobTime,
		true /* nowEpoch doesn't matter when catchup=true */, time.Unix(0, 0).In(location), location)
}

func (s *CronSchedule) getNextScheduledEpochImp(lastJobTime time.Time, catchup bool, nowTime time.Time, location *time.Location) int64 {
	schedule, err := cron.Parse(s.Cron)
	if err != nil {
		// This should never happen, validation should have caught this at resource creation.
		log.Errorf("%+v", wraperror.Errorf(
			"Found invalid schedule (%v): %v", s.Cron, err))
		return math.MaxInt64
	}

	startTime := lastJobTime
	if s.StartTime != nil && s.StartTime.Time.After(startTime) {
		startTime = s.StartTime.Time
	}
	result := schedule.Next(startTime)
	var endTime time.Time = time.Unix(1<<63-62135596801, 999999999).In(location)
	// math.int64 max will break the comparison.
	// Examle playground https://play.golang.org/p/LERg0aq2mU6
	// Max date https://stackoverflow.com/questions/25065055/what-is-the-maximum-time-time-in-go
	if s.EndTime != nil {
		endTime = s.EndTime.Time.In(location)
	}

	if endTime.Before(result) {
		return math.MaxInt64
	}

	// When we need to catch up with schedule, just run schedules one by one.
	if catchup == true {
		return result.UTC().Unix()
	}

	// When we don't need to catch up, find the last schedule we need to run
	// now and skip others in between.
	next := result
	var nextNext time.Time
	for {
		nextNext = schedule.Next(next)
		if (nextNext.Before(nowTime) || nextNext.Equal(nowTime)) && (nextNext.Before(endTime) || nextNext.Equal(endTime)) {
			next = nextNext
		} else {
			break
		}
	}
	return next.UTC().Unix()
}
