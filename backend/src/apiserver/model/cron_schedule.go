// Copyright 2024 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package model

type CronSchedule struct {
	// Time at which scheduling starts.
	// If no start time is specified, the StartTime is the creation time of the schedule.
	CronScheduleStartTimeInSec *int64 `gorm:"column:CronScheduleStartTimeInSec;"`

	// Time at which scheduling ends.
	// If no end time is specified, the EndTime is the end of time.
	CronScheduleEndTimeInSec *int64 `gorm:"column:CronScheduleEndTimeInSec;"`

	// Cron string describing when a workflow should be created within the
	// time interval defined by StartTime and EndTime.
	Cron *string `gorm:"column:Schedule;"`
}

func (c CronSchedule) IsEmpty() bool {
	return (c.CronScheduleStartTimeInSec == nil || *c.CronScheduleStartTimeInSec == 0) &&
		(c.CronScheduleEndTimeInSec == nil || *c.CronScheduleEndTimeInSec == 0) &&
		(c.Cron == nil || *c.Cron == "")
}
