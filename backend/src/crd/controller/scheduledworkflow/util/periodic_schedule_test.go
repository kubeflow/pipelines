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
	"testing"
	"time"

	commonutil "github.com/kubeflow/pipelines/backend/src/common/util"
	swfapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPeriodicSchedule_getNextScheduledEpoch_StartDate_EndDate(t *testing.T) {
	// First job.
	schedule := NewPeriodicSchedule(&swfapi.PeriodicSchedule{
		StartTime:      commonutil.Metav1TimePointer(v1.NewTime(time.Unix(10*hour, 0).UTC())),
		EndTime:        commonutil.Metav1TimePointer(v1.NewTime(time.Unix(11*hour, 0).UTC())),
		IntervalSecond: minute,
	})
	lastJobEpoch := int64(0)
	assert.Equal(t, int64(10*hour),
		schedule.getNextScheduledEpoch(lastJobEpoch))

	// Not the first job.
	lastJobEpoch = int64(10*hour + 5*minute)
	assert.Equal(t, int64(10*hour+6*minute),
		schedule.getNextScheduledEpoch(lastJobEpoch))

	// Last job
	lastJobEpoch = int64(13 * hour)
	assert.Equal(t, int64(math.MaxInt64),
		schedule.getNextScheduledEpoch(lastJobEpoch))

}

func TestPeriodicSchedule_getNextScheduledEpoch_PeriodOnly(t *testing.T) {
	schedule := NewPeriodicSchedule(&swfapi.PeriodicSchedule{
		IntervalSecond: minute,
	})
	lastJobEpoch := int64(10 * hour)
	assert.Equal(t, int64(10*hour+minute),
		schedule.getNextScheduledEpoch(lastJobEpoch))
}

func TestPeriodicSchedule_getNextScheduledEpoch_NoPeriod(t *testing.T) {
	schedule := NewPeriodicSchedule(&swfapi.PeriodicSchedule{
		StartTime:      commonutil.Metav1TimePointer(v1.NewTime(time.Unix(10*hour, 0).UTC())),
		EndTime:        commonutil.Metav1TimePointer(v1.NewTime(time.Unix(11*hour, 0).UTC())),
		IntervalSecond: 0,
	})
	lastJobEpoch := int64(10 * hour)
	assert.Equal(t, int64(10*hour+second),
		schedule.getNextScheduledEpoch(lastJobEpoch))
}

func TestPeriodicSchedule_GetNextScheduledEpoch(t *testing.T) {
	// There was a previous job.
	schedule := NewPeriodicSchedule(&swfapi.PeriodicSchedule{
		StartTime:      commonutil.Metav1TimePointer(v1.NewTime(time.Unix(10*hour+10*minute, 0).UTC())),
		EndTime:        commonutil.Metav1TimePointer(v1.NewTime(time.Unix(11*hour, 0).UTC())),
		IntervalSecond: 60,
	})
	lastJobEpoch := int64(10*hour + 20*minute)
	defaultStartEpoch := int64(10*hour + 15*minute)
	assert.Equal(t, int64(10*hour+20*minute+minute),
		schedule.GetNextScheduledEpoch(&lastJobEpoch, defaultStartEpoch))

	// There is no previous job, falling back on the start date of the schedule.
	assert.Equal(t, int64(10*hour+10*minute+minute),
		schedule.GetNextScheduledEpoch(nil, defaultStartEpoch))

	// There is no previous job, no schedule start date, falling back on the
	// creation date of the workflow.
	schedule = NewPeriodicSchedule(&swfapi.PeriodicSchedule{
		EndTime:        commonutil.Metav1TimePointer(v1.NewTime(time.Unix(11*hour, 0).UTC())),
		IntervalSecond: 60,
	})
	assert.Equal(t, int64(10*hour+15*minute+minute),
		schedule.GetNextScheduledEpoch(nil, defaultStartEpoch))
}

func TestPeriodicSchedule_GetNextScheduledEpochNoCatchup(t *testing.T) {
	// There was a previous job, hasn't been time for next job
	schedule := NewPeriodicSchedule(&swfapi.PeriodicSchedule{
		StartTime:      commonutil.Metav1TimePointer(v1.NewTime(time.Unix(10*hour+10*minute, 0).UTC())),
		EndTime:        commonutil.Metav1TimePointer(v1.NewTime(time.Unix(11*hour, 0).UTC())),
		IntervalSecond: 60,
	})
	lastJobEpoch := int64(10*hour + 20*minute)
	defaultStartEpoch := int64(10*hour + 15*minute)
	nowEpoch := int64(10*hour + 20*minute + 30*second)
	assert.Equal(t, int64(10*hour+20*minute+minute),
		schedule.GetNextScheduledEpochNoCatchup(&lastJobEpoch, defaultStartEpoch, nowEpoch))

	// Exactly now for next job
	lastJobEpoch = int64(10*hour + 20*minute)
	nowEpoch = int64(10*hour + 20*minute + minute)
	assert.Equal(t, int64(10*hour+20*minute+minute),
		schedule.GetNextScheduledEpochNoCatchup(&lastJobEpoch, defaultStartEpoch, nowEpoch))

	// Shortly after next job's original schedule
	lastJobEpoch = int64(10*hour + 20*minute)
	nowEpoch = int64(10*hour + 21*minute + 30*second)
	assert.Equal(t, int64(10*hour+21*minute),
		schedule.GetNextScheduledEpochNoCatchup(&lastJobEpoch, defaultStartEpoch, nowEpoch))

	// We are behind schedule
	lastJobEpoch = int64(10*hour + 20*minute)
	nowEpoch = int64(10*hour + 30*minute)
	assert.Equal(t, nowEpoch,
		schedule.GetNextScheduledEpochNoCatchup(&lastJobEpoch, defaultStartEpoch, nowEpoch))

	// We are way behind schedule (later than end time)
	lastJobEpoch = int64(10*hour + 20*minute)
	nowEpoch = int64(12 * hour)
	assert.Equal(t, int64(11*hour), // TODO: should this be end time?
		schedule.GetNextScheduledEpochNoCatchup(&lastJobEpoch, defaultStartEpoch, nowEpoch))

	// There is no previous job, falling back on the start date of the schedule
	assert.Equal(t, int64(10*hour+10*minute+minute),
		schedule.GetNextScheduledEpochNoCatchup(nil, defaultStartEpoch, 0))

	// There is no previous job, no schedule start date, falling back on the
	// creation date of the workflow.
	schedule = NewPeriodicSchedule(&swfapi.PeriodicSchedule{
		EndTime:        commonutil.Metav1TimePointer(v1.NewTime(time.Unix(11*hour, 0).UTC())),
		IntervalSecond: 60,
	})
	assert.Equal(t, int64(10*hour+15*minute+minute),
		schedule.GetNextScheduledEpochNoCatchup(nil, defaultStartEpoch, 0))
}

func TestPeriodicSchedule_TestStarttime(t *testing.T) {
	// First job.

	t1, err := time.Parse(
		time.RFC3339,
		"2012-11-01T22:00:00+00:00")
	assert.Nil(t, err)

	schedule := NewPeriodicSchedule(&swfapi.PeriodicSchedule{
		StartTime:      commonutil.Metav1TimePointer(v1.NewTime(t1)),
		IntervalSecond: minute * 60 * 24,
	})

	lastJobEpoch := int64(0)
	val := schedule.getNextScheduledEpoch(lastJobEpoch)
	assert.Equal(t, t1.Unix(), val)
}
