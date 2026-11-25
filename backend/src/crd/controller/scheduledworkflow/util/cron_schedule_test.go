// Copyright 2018 The Kubeflow Authors
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
	"math/rand"
	"testing"
	"time"

	commonutil "github.com/kubeflow/pipelines/backend/src/common/util"
	swfapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	"github.com/robfig/cron"
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
	lastJobTime := v1.NewTime(time.Unix(int64(10*hour+20*minute), 0).UTC())
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
	lastJobTime := v1.NewTime(lastJob)

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

	lastJobTime := v1.NewTime(time.Unix(int64(10*hour+20*minute), 0).UTC())
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

func TestCronSchedule_GetNextScheduledTimeNoCatchup_LongAgoStartTime(t *testing.T) {
	// The number of schedules missed since the start time is too large to
	// visit them one by one.
	startTime := time.Date(1, time.January, 2, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(2024, time.May, 17, 0, 0, 0, 0, time.UTC)
	nowTime := time.Date(2026, time.March, 14, 15, 9, 26, 0, time.UTC)
	defaultStartTime := nowTime
	location, _ := time.LoadLocation("UTC")

	tests := []struct {
		cron     string
		endTime  *v1.Time
		expected time.Time
	}{
		{"* * * * * *", nil, nowTime},
		{"0 * * * * *", nil, time.Date(2026, time.March, 14, 15, 9, 0, 0, time.UTC)},
		{"0 30 6 1 * *", nil, time.Date(2026, time.March, 1, 6, 30, 0, 0, time.UTC)},
		{"0 0 0 1 1 *", nil, time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)},
		{"@every 1s", nil, nowTime},
		// Now is 63909011366 seconds after the start time, 4 more than a multiple of 7.
		{"@every 7s", nil, time.Date(2026, time.March, 14, 15, 9, 22, 0, time.UTC)},
		{"@every 1h", nil, time.Date(2026, time.March, 14, 15, 0, 0, 0, time.UTC)},
		// Later than the end time.
		{"* * * * * *", commonutil.Metav1TimePointer(v1.NewTime(endTime)), endTime},
		{"0 30 6 * * *", commonutil.Metav1TimePointer(v1.NewTime(endTime)),
			time.Date(2024, time.May, 16, 6, 30, 0, 0, time.UTC)},
		{"@every 1s", commonutil.Metav1TimePointer(v1.NewTime(endTime)), endTime},
	}
	for _, test := range tests {
		schedule := NewCronSchedule(&swfapi.CronSchedule{
			StartTime: commonutil.Metav1TimePointer(v1.NewTime(startTime)),
			EndTime:   test.endTime,
			Cron:      test.cron,
		})
		assert.Equal(t, test.expected,
			schedule.GetNextScheduledTimeNoCatchup(nil, defaultStartTime, nowTime, location), test.cron)
	}
}

func TestCronSchedule_GetNextScheduledTimeNoCatchup_ConstantDelay(t *testing.T) {
	// "@every" schedules are counted from the last job, not from the clock.
	schedule := NewCronSchedule(&swfapi.CronSchedule{
		Cron: "@every 7s",
	})
	lastJobTime := v1.NewTime(time.Unix(10*hour+3*second, 0).UTC())
	defaultStartTime := time.Unix(9*hour, 0).UTC()
	location, _ := time.LoadLocation("UTC")

	// Hasn't been time for next job
	nowTime := time.Unix(10*hour+9*second, 0).UTC()
	assert.Equal(t, time.Unix(10*hour+10*second, 0).UTC(),
		schedule.GetNextScheduledTimeNoCatchup(&lastJobTime, defaultStartTime, nowTime, location))

	// Exactly now for next job
	nowTime = time.Unix(10*hour+10*second, 0).UTC()
	assert.Equal(t, nowTime,
		schedule.GetNextScheduledTimeNoCatchup(&lastJobTime, defaultStartTime, nowTime, location))

	// We are behind schedule
	nowTime = time.Unix(10*hour+minute, 0).UTC()
	assert.Equal(t, time.Unix(10*hour+59*second, 0).UTC(),
		schedule.GetNextScheduledTimeNoCatchup(&lastJobTime, defaultStartTime, nowTime, location))
}

func TestCronSchedule_GetNextScheduledTimeNoCatchup_ClocksSetBack(t *testing.T) {
	// On 2028-10-29 the clocks in Antarctica/Troll are set back from 03:00
	// (+02:00) to 01:00 (+00:00), so the schedule runs twice on that day.
	location, err := time.LoadLocation("Antarctica/Troll")
	assert.Nil(t, err)
	schedule := NewCronSchedule(&swfapi.CronSchedule{
		Cron: "59 0-3 1 * * 0", // trigger 4 times from 01:00:59 on Sundays
	})
	lastJobTime := v1.NewTime(time.Date(2028, time.October, 22, 5, 22, 44, 0, location))
	defaultStartTime := lastJobTime.Time

	// During the first time
	nowTime := time.Date(2028, time.October, 28, 23, 2, 30, 0, time.UTC)
	assert.Equal(t, time.Date(2028, time.October, 28, 23, 1, 59, 0, time.UTC).In(location),
		schedule.GetNextScheduledTimeNoCatchup(&lastJobTime, defaultStartTime, nowTime, location))

	// After the second time
	nowTime = time.Date(2028, time.October, 29, 7, 21, 41, 0, time.UTC)
	assert.Equal(t, time.Date(2028, time.October, 29, 1, 3, 59, 0, time.UTC).In(location),
		schedule.GetNextScheduledTimeNoCatchup(&lastJobTime, defaultStartTime, nowTime, location))
}

func TestCronSchedule_GetNextScheduledTimeNoCatchup_NeverMatchingCron(t *testing.T) {
	// A valid cron expression that never triggers: there is no February 30th.
	schedule := NewCronSchedule(&swfapi.CronSchedule{
		Cron: "0 0 0 30 2 *",
	})
	lastJobTime := v1.NewTime(time.Unix(10*hour, 0).UTC())
	defaultStartTime := time.Unix(9*hour, 0).UTC()
	nowTime := time.Unix(11*hour, 0).UTC()
	location, _ := time.LoadLocation("UTC")

	assert.Equal(t, maxTime.UTC(),
		schedule.GetNextScheduledTimeNoCatchup(&lastJobTime, defaultStartTime, nowTime, location))
	assert.Equal(t, maxTime.UTC(),
		schedule.GetNextScheduledTimeNoCatchup(nil, defaultStartTime, nowTime, location))
}

// stepToLastActivationNotAfter is the activation-by-activation walk that
// lastActivationNotAfter replaces. It is kept as the reference for the
// comparison below, and is only usable over short spans.
func stepToLastActivationNotAfter(schedule cron.Schedule, activation time.Time, limit time.Time) (time.Time, bool) {
	next := activation
	for steps := 0; steps < 200000; steps++ {
		nextNext := schedule.Next(next)
		if nextNext.IsZero() || !nextNext.After(next) || nextNext.After(limit) {
			return next, true
		}
		next = nextNext
	}
	return time.Time{}, false
}

func TestLastActivationNotAfter_MatchesStepByStepWalk(t *testing.T) {
	specs := []struct {
		cron    string
		maxSpan time.Duration
	}{
		{"* * * * * *", 6 * time.Hour},
		{"*/7 * * * * *", 24 * time.Hour},
		{"0 * * * * *", 10 * 24 * time.Hour},
		{"30 */5 * * * *", 30 * 24 * time.Hour},
		{"0 0 * * * *", 400 * 24 * time.Hour},
		{"0 15 2 * * *", 5 * 365 * 24 * time.Hour},
		{"0 30 1 * * 0", 5 * 365 * 24 * time.Hour},
		{"0 0 0 1 * *", 20 * 365 * 24 * time.Hour},
		{"0 0 12 29 2 *", 40 * 365 * 24 * time.Hour},
		{"0 0 6 * * 1-5", 5 * 365 * 24 * time.Hour},
		{"@every 1s", 6 * time.Hour},
		{"@every 7s", 24 * time.Hour},
		{"@every 90m", 400 * 24 * time.Hour},
		{"@hourly", 400 * 24 * time.Hour},
		{"@daily", 5 * 365 * 24 * time.Hour},
	}
	// UTC, zones whose clocks change, and one with a 30 minute change.
	zones := []string{"UTC", "America/New_York", "Europe/Berlin", "Australia/Lord_Howe", "Asia/Kolkata"}
	random := rand.New(rand.NewSource(20261123))
	origin := time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC).Unix()
	differences, cases := 0, 0

	for _, spec := range specs {
		schedule, err := cron.Parse(spec.cron)
		assert.Nil(t, err, spec.cron)
		for _, zone := range zones {
			location, err := time.LoadLocation(zone)
			assert.Nil(t, err, zone)
			for i := 0; i < 40; i++ {
				start := time.Unix(origin+random.Int63n(10*365*24*3600), 0).In(location)
				activation := schedule.Next(start)
				// Before, at and after the first activation.
				offset := time.Duration(random.Int63n(int64(spec.maxSpan/time.Second)+120)-60) * time.Second
				limit := activation.Add(offset)

				expected, ok := stepToLastActivationNotAfter(schedule, activation, limit)
				if !ok {
					t.Fatalf("reference walk of %q in %s from %v to %v did not finish", spec.cron, zone, activation, limit)
				}
				actual := lastActivationNotAfter(schedule, activation, limit)
				cases++
				describe := fmt.Sprintf("%q in %s: activation %v, limit %v: step by step %v, lastActivationNotAfter %v",
					spec.cron, zone, activation, limit, expected, actual)
				if expected.Equal(actual) {
					continue
				}
				// The walk can stop short: after some clock changes, cron's Next
				// passes over an activation when it is asked from the one before
				// (midnight schedules in Australia/Lord_Howe do this). The result
				// may then be later than the walk's, but never earlier, never
				// after the limit, and always an activation of the schedule.
				if actual.Before(expected) {
					t.Fatalf("earlier than the step by step walk: %s", describe)
				}
				if actual.After(limit) {
					t.Fatalf("after the limit: %s", describe)
				}
				if !schedule.Next(actual.Add(-time.Second)).Equal(actual) {
					t.Fatalf("not an activation: %s", describe)
				}
				differences++
			}
		}
	}
	// The walk and the result agree except in those rare clock-change cases.
	assert.Less(t, differences*100, cases, "%d of %d cases differ from the step by step walk", differences, cases)
}

func TestLastActivationNotAfter_ConstantDelayWithoutWholeSeconds(t *testing.T) {
	// cron.Every never returns a delay below one second, but the type allows
	// it, and dividing by such a delay must not be attempted.
	activation := time.Unix(10*hour, 0).UTC()
	limit := time.Unix(11*hour, 0).UTC()
	for _, delay := range []time.Duration{0, 500 * time.Millisecond} {
		assert.Equal(t, activation,
			lastActivationNotAfter(cron.ConstantDelaySchedule{Delay: delay}, activation, limit), delay)
	}
}

func TestCronSchedule_ActivationGapLongerThanNextSearches(t *testing.T) {
	// February 29th is eight years away across 1900 and 2100, which are not
	// leap years. cron's Next gives up after five years and returns the zero
	// time, which does not mean that the schedule has no later activation.
	leapDay := func(year int) time.Time {
		return time.Date(year, time.February, 29, 12, 0, 0, 0, time.UTC)
	}
	schedule, err := cron.Parse("0 0 12 29 2 *")
	assert.Nil(t, err)
	assert.True(t, schedule.Next(leapDay(2096)).IsZero(), "the library limit this test is about")

	// A search that starts inside the gap still finds what follows it.
	assert.Equal(t, leapDay(2104), nextActivation(schedule, leapDay(2096), maxTime))
	assert.Equal(t, time.Time{}, nextActivation(schedule, leapDay(2096), leapDay(2104).Add(-time.Second)))

	// Skipping to the last activation crosses the gap, whichever side of it
	// the search points fall on.
	assert.Equal(t, leapDay(2024),
		lastActivationNotAfter(schedule, leapDay(1768), time.Date(2026, time.March, 14, 12, 0, 0, 0, time.UTC)))
	assert.Equal(t, leapDay(2108),
		lastActivationNotAfter(schedule, leapDay(2096), time.Date(2110, time.January, 1, 0, 0, 0, 0, time.UTC)))
	assert.Equal(t, leapDay(2096),
		lastActivationNotAfter(schedule, leapDay(2092), time.Date(2103, time.January, 1, 0, 0, 0, 0, time.UTC)))

	// A schedule that starts inside the gap is not suppressed, with or
	// without catch-up.
	cronSchedule := NewCronSchedule(&swfapi.CronSchedule{
		StartTime: commonutil.Metav1TimePointer(v1.NewTime(time.Date(2097, time.January, 1, 0, 0, 0, 0, time.UTC))),
		Cron:      "0 0 12 29 2 *",
	})
	defaultStartTime := time.Date(2097, time.January, 1, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, leapDay(2104),
		cronSchedule.GetNextScheduledTime(nil, defaultStartTime, time.UTC))
	assert.Equal(t, leapDay(2108),
		cronSchedule.GetNextScheduledTimeNoCatchup(nil, defaultStartTime, time.Date(2110, time.January, 1, 0, 0, 0, 0, time.UTC), time.UTC))
}

func TestCronSchedule_GetNextScheduledTime_NeverMatchingCron(t *testing.T) {
	// With catch-up as well, a schedule that can never match has no next time.
	schedule := NewCronSchedule(&swfapi.CronSchedule{
		Cron: "0 0 0 30 2 *",
	})
	lastJobTime := v1.NewTime(time.Unix(10*hour, 0).UTC())
	assert.Equal(t, maxTime.UTC(),
		schedule.GetNextScheduledTime(&lastJobTime, time.Unix(9*hour, 0).UTC(), time.UTC))
}
