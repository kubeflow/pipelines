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
	"testing"
	"time"

	commonutil "github.com/kubeflow/pipelines/backend/src/common/util"
	swfapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	second = 1
	minute = 60 * second
	hour   = 60 * minute
)

func TestCronSchedule_getNextScheduledTime_Cron_StartDate_EndDate(t *testing.T) {
	// First job.
	startTime := time.Unix(10*hour, 0).UTC()
	endTime := time.Unix(11*hour, 0).UTC()
	schedule := NewCronSchedule(&swfapi.CronSchedule{
		StartTime: commonutil.Metav1TimePointer(v1.NewTime(startTime)),
		EndTime:   commonutil.Metav1TimePointer(v1.NewTime(endTime)),
		Cron:      "0 * * * * * ", // trigger every minute
	})

	location, _ := time.LoadLocation("UTC")
	lastJobTime := time.Unix(0, 0).UTC()
	assert.Equal(t, startTime.Add(time.Minute),
		schedule.getNextScheduledTime(lastJobTime, location))

	// Not the first job.
	lastJobTime = time.Unix(10*hour+5*minute, 0).UTC()
	assert.Equal(t, lastJobTime.Add(time.Minute),
		schedule.getNextScheduledTime(lastJobTime, location))

	// Last job
	lastJobTime = time.Unix(13*hour, 0).UTC()
	assert.Equal(t, maxTime.UTC(),
		schedule.getNextScheduledTime(lastJobTime.UTC(), location))

}

func TestCronSchedule_getNextScheduledTime_CronOnly(t *testing.T) {
	schedule := NewCronSchedule(&swfapi.CronSchedule{
		Cron: "0 * * * * * ", // trigger every minute
	})
	location, _ := time.LoadLocation("UTC")
	lastJobTime := time.Unix(10*hour, 0).In(location)
	assert.Equal(t, lastJobTime.Add(time.Minute),
		schedule.getNextScheduledTime(lastJobTime, location))
}

func TestCronSchedule_getNextScheduledTime_NoCron(t *testing.T) {
	schedule := NewCronSchedule(&swfapi.CronSchedule{
		StartTime: commonutil.Metav1TimePointer(v1.NewTime(time.Unix(10*hour, 0).UTC())),
		EndTime:   commonutil.Metav1TimePointer(v1.NewTime(time.Unix(11*hour, 0).UTC())),
		Cron:      "",
	})
	lastJobTime := time.Unix(0, 0).UTC()
	location, _ := time.LoadLocation("UTC")
	assert.Equal(t, maxTime.UTC(),
		schedule.getNextScheduledTime(lastJobTime, location))
}

func TestCronSchedule_getNextScheduledTime_InvalidCron(t *testing.T) {
	schedule := NewCronSchedule(&swfapi.CronSchedule{
		StartTime: commonutil.Metav1TimePointer(v1.NewTime(time.Unix(10*hour, 0).UTC())),
		EndTime:   commonutil.Metav1TimePointer(v1.NewTime(time.Unix(11*hour, 0).UTC())),
		Cron:      "*$&%*(W&",
	})
	location, _ := time.LoadLocation("UTC")
	lastJobTime := time.Unix(0, 0).UTC()
	assert.Equal(t, maxTime.UTC(),
		schedule.getNextScheduledTime(lastJobTime, location))
}

func TestCronSchedule_GetNextScheduledTime(t *testing.T) {
	// There was a previous job.
	startTime := time.Unix(10*hour+10*minute, 0).UTC()
	schedule := NewCronSchedule(&swfapi.CronSchedule{
		StartTime: commonutil.Metav1TimePointer(v1.NewTime(startTime)),
		EndTime:   commonutil.Metav1TimePointer(v1.NewTime(time.Unix(11*hour, 0).UTC())),
		Cron:      "0 * * * * * ", // trigger every minute
	})
	lastJobTime := v1.Time{time.Unix(int64(10*hour+20*minute), 0).UTC()}
	defaultStartTime := time.Unix(int64(10*hour+15*minute), 0).UTC()
	location, _ := time.LoadLocation("UTC")
	assert.Equal(t, lastJobTime.Add(time.Minute),
		schedule.GetNextScheduledTime(&lastJobTime, defaultStartTime, location))

	// There is no previous job, falling back on the start date of the schedule.
	assert.Equal(t, startTime.Add(time.Minute),
		schedule.GetNextScheduledTime(nil, defaultStartTime, location))

	// There is no previous job, no schedule start date, falling back on the
	// creation date of the workflow.
	schedule = NewCronSchedule(&swfapi.CronSchedule{
		EndTime: commonutil.Metav1TimePointer(v1.NewTime(time.Unix(11*hour, 0).UTC())),
		Cron:    "0 * * * * * ", // trigger every minute
	})
	assert.Equal(t, defaultStartTime.Add(time.Minute),
		schedule.GetNextScheduledTime(nil, defaultStartTime, location))
}

func TestCronSchedule_GetNextScheduledTime_LocationsEnvSet(t *testing.T) {
	locationString := "Asia/Shanghai"
	viper.Set(TimeZone, locationString)
	defer viper.Set(TimeZone, "")

	location, err := time.LoadLocation(locationString)
	assert.Nil(t, err)
	startTime, err := time.Parse(time.RFC1123Z, "Mon, 11 Jan 2010 10:00:00 +0800")
	assert.Nil(t, err)
	endTime, err := time.Parse(time.RFC1123Z, "Mon, 11 Jan 2010 11:00:00 +0800")
	assert.Nil(t, err)
	lastJob, err := time.Parse(time.RFC1123Z, "Mon, 11 Jan 2010 10:20:00 +0800")
	assert.Nil(t, err)
	defaultStartTime, err := time.Parse(time.RFC1123Z, "Mon, 11 Jan 2010 10:15:00 +0800")
	schedule := NewCronSchedule(&swfapi.CronSchedule{
		StartTime: commonutil.Metav1TimePointer(v1.NewTime(startTime)),
		EndTime:   commonutil.Metav1TimePointer(v1.NewTime(endTime)),
		Cron:      "0 * * * * * ", // trigger every minute
	})
	lastJobTime := v1.Time{lastJob}

	assert.Equal(t, lastJob.Add(time.Minute*1).In(location),
		schedule.GetNextScheduledTime(&lastJobTime, defaultStartTime, location))

	// There is no previous job, falling back on the start date of the schedule.
	assert.Equal(t, startTime.Add(time.Minute*1).In(location),
		schedule.GetNextScheduledTime(nil, defaultStartTime, location))
}

func TestCronSchedule_GetNextScheduledTimeNoCatchup(t *testing.T) {
	// There was a previous job, hasn't been time for next job
	schedule := NewCronSchedule(&swfapi.CronSchedule{
		StartTime: commonutil.Metav1TimePointer(v1.NewTime(time.Unix(10*hour+10*minute, 0).UTC())),
		EndTime:   commonutil.Metav1TimePointer(v1.NewTime(time.Unix(11*hour, 0).UTC())),
		Cron:      "0 * * * * * ", // trigger every minute
	})

	lastJobTime := v1.Time{time.Unix(int64(10*hour+20*minute), 0).UTC()}
	defaultStartTime := time.Unix(int64(10*hour+15*minute), 0).UTC()
	nowTime := time.Unix(int64(10*hour+20*minute+30*second), 0).UTC()
	location, _ := time.LoadLocation("UTC")

	assert.Equal(t, lastJobTime.Add(time.Minute),
		schedule.GetNextScheduledTimeNoCatchup(&lastJobTime, defaultStartTime, nowTime, location))

	// Exactly now for next job
	nowTime = time.Unix(int64(10*hour+20*minute+minute), 0).UTC()
	assert.Equal(t, nowTime,
		schedule.GetNextScheduledTimeNoCatchup(&lastJobTime, defaultStartTime, nowTime, location))

	// Shortly after next job's original schedule
	nowTime = time.Unix(int64(10*hour+21*minute+30*second), 0).UTC()
	assert.Equal(t, nowTime.Add(-30*time.Second),
		schedule.GetNextScheduledTimeNoCatchup(&lastJobTime, defaultStartTime, nowTime, location))

	// We are behind schedule
	nowTime = time.Unix(int64(10*hour+30*minute), 0).UTC()
	assert.Equal(t, nowTime,
		schedule.GetNextScheduledTimeNoCatchup(&lastJobTime, defaultStartTime, nowTime, location))

	// We are way behind schedule (later than end time)
	nowTime = time.Unix(int64(12*hour), 0).UTC()
	assert.Equal(t, nowTime.Add(-1*time.Hour),
		schedule.GetNextScheduledTimeNoCatchup(&lastJobTime, defaultStartTime, nowTime, location))

	// There is no previous job, falling back on the start date of the schedule
	assert.Equal(t, time.Unix(10*hour+10*minute+minute, 0).UTC(),
		schedule.GetNextScheduledTimeNoCatchup(nil, defaultStartTime, time.Unix(0, 0), location))

	// There is no previous job, no schedule start date, falling back on the
	// creation date of the workflow.
	schedule = NewCronSchedule(&swfapi.CronSchedule{
		EndTime: commonutil.Metav1TimePointer(v1.NewTime(time.Unix(11*hour, 0).UTC())),
		Cron:    "0 * * * * * ", // trigger every minute
	})
	assert.Equal(t, time.Unix(10*hour+15*minute+minute, 0).UTC(),
		schedule.GetNextScheduledTimeNoCatchup(nil, defaultStartTime, time.Unix(0, 0), location))
}
