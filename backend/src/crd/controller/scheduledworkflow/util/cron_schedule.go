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

	swfapi "github.com/googleprivate/ml/backend/src/crd/pkg/apis/scheduledworkflow/v1alpha1"
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
	effectiveLastJobEpoch := defaultStartEpoch
	if lastJobEpoch != nil {
		effectiveLastJobEpoch = *lastJobEpoch
	} else if s.StartTime != nil {
		effectiveLastJobEpoch = s.StartTime.Unix()
	}
	return s.getNextScheduledEpoch(effectiveLastJobEpoch)
}

func (s *CronSchedule) getNextScheduledEpoch(lastJobEpoch int64) int64 {
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

	if s.EndTime != nil &&
		s.EndTime.Unix() < result {
		return math.MaxInt64
	}

	return result
}
