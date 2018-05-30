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
	log "github.com/sirupsen/logrus"
	swfapi "github.com/kubeflow/pipelines/pkg/apis/scheduledworkflow/v1alpha1"
	wraperror "github.com/pkg/errors"
	"github.com/robfig/cron"
	"math"
	"time"
)

// CronScheduleWrap is a wrapper to help manipulate CronSchedule objects.
type CronScheduleWrap struct {
	cronSchedule *swfapi.CronSchedule
}

func NewCronScheduleWrap(cronSchedule *swfapi.CronSchedule) *CronScheduleWrap {
	if cronSchedule == nil {
		log.Fatalf("The cronSchedule should never be nil")
	}

	return &CronScheduleWrap{
		cronSchedule: cronSchedule,
	}
}

func (s *CronScheduleWrap) GetNextScheduledEpoch(lastJobEpoch *int64,
	defaultStartEpoch int64) int64 {
	effectiveLastJobEpoch := defaultStartEpoch
	if lastJobEpoch != nil {
		effectiveLastJobEpoch = *lastJobEpoch
	} else if s.cronSchedule.StartTime != nil {
		effectiveLastJobEpoch = s.cronSchedule.StartTime.Unix()
	}
	return s.getNextScheduledEpoch(effectiveLastJobEpoch)
}

func (s *CronScheduleWrap) getNextScheduledEpoch(lastJobEpoch int64) int64 {
	schedule, err := cron.Parse(s.cronSchedule.Cron)
	if err != nil {
		// This should never happen, validation should have caught this at resource creation.
		log.Errorf("%+v", wraperror.Errorf(
			"Found invalid schedule (%v): %v", s.cronSchedule.Cron, err))
		return math.MaxInt64
	}

	startEpoch := lastJobEpoch
	if s.cronSchedule.StartTime != nil && s.cronSchedule.StartTime.Unix() > startEpoch {
		startEpoch = s.cronSchedule.StartTime.Unix()
	}
	result := schedule.Next(time.Unix(startEpoch, 0).UTC()).Unix()

	if s.cronSchedule.EndTime != nil &&
		s.cronSchedule.EndTime.Unix() < result {
		return math.MaxInt64
	}

	return result
}
