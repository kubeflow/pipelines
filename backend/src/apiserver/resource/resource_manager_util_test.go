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

package resource

import (
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/timestamp"
	api "github.com/googleprivate/ml/backend/api/go_client"
	scheduledworkflow "github.com/googleprivate/ml/backend/src/crd/pkg/apis/scheduledworkflow/v1alpha1"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestToSwfCRDResourceGeneratedName_SpecialCharsAndSpace(t *testing.T) {
	name, err := toSWFCRDResourceGeneratedName("! HaVe ä £unky name")
	assert.Nil(t, err)
	assert.Equal(t, name, "haveunkyname")
}

func TestToSwfCRDResourceGeneratedName_TruncateLongName(t *testing.T) {
	name, err := toSWFCRDResourceGeneratedName("AloooooooooooooooooongName")
	assert.Nil(t, err)
	assert.Equal(t, name, "aloooooooooooooooooongnam")
}

func TestToSwfCRDResourceGeneratedName_EmptyName(t *testing.T) {
	name, err := toSWFCRDResourceGeneratedName("")
	assert.Nil(t, err)
	assert.Equal(t, name, "job-")
}

func TestToCrdParameter(t *testing.T) {
	assert.Equal(t,
		toCRDParameter([]*api.Parameter{{Name: "param2", Value: "world"}, {Name: "param1", Value: "hello"}}),
		[]scheduledworkflow.Parameter{{Name: "param2", Value: "world"}, {Name: "param1", Value: "hello"}})
}

func TestToCrdCronSchedule(t *testing.T) {
	actualCronSchedule := toCRDCronSchedule(&api.CronSchedule{
		Cron:      "123",
		StartTime: &timestamp.Timestamp{Seconds: 123},
		EndTime:   &timestamp.Timestamp{Seconds: 456},
	})
	startTime := v1.NewTime(time.Unix(123, 0))
	endTime := v1.NewTime(time.Unix(456, 0))
	assert.Equal(t, actualCronSchedule, &scheduledworkflow.CronSchedule{
		Cron:      "123",
		StartTime: &startTime,
		EndTime:   &endTime,
	})
}

func TestToCrdCronSchedule_NilCron(t *testing.T) {
	actualCronSchedule := toCRDCronSchedule(&api.CronSchedule{
		StartTime: &timestamp.Timestamp{Seconds: 123},
		EndTime:   &timestamp.Timestamp{Seconds: 456},
	})
	assert.Nil(t, actualCronSchedule)
}

func TestToCrdCronSchedule_NilStartTime(t *testing.T) {
	actualCronSchedule := toCRDCronSchedule(&api.CronSchedule{
		Cron:    "123",
		EndTime: &timestamp.Timestamp{Seconds: 456},
	})
	endTime := v1.NewTime(time.Unix(456, 0))
	assert.Equal(t, actualCronSchedule, &scheduledworkflow.CronSchedule{
		Cron:    "123",
		EndTime: &endTime,
	})
}

func TestToCrdCronSchedule_NilEndTime(t *testing.T) {
	actualCronSchedule := toCRDCronSchedule(&api.CronSchedule{
		Cron:      "123",
		StartTime: &timestamp.Timestamp{Seconds: 123},
	})
	startTime := v1.NewTime(time.Unix(123, 0))
	assert.Equal(t, actualCronSchedule, &scheduledworkflow.CronSchedule{
		Cron:      "123",
		StartTime: &startTime,
	})
}

func TestToCrdPeriodicSchedule(t *testing.T) {
	actualPeriodicSchedule := toCRDPeriodicSchedule(&api.PeriodicSchedule{
		IntervalSecond: 123,
		StartTime:      &timestamp.Timestamp{Seconds: 1},
		EndTime:        &timestamp.Timestamp{Seconds: 2},
	})
	startTime := v1.NewTime(time.Unix(1, 0))
	endTime := v1.NewTime(time.Unix(2, 0))
	assert.Equal(t, actualPeriodicSchedule, &scheduledworkflow.PeriodicSchedule{
		IntervalSecond: 123,
		StartTime:      &startTime,
		EndTime:        &endTime,
	})
}

func TestToCrdPeriodicSchedule_NilInterval(t *testing.T) {
	actualPeriodicSchedule := toCRDPeriodicSchedule(&api.PeriodicSchedule{
		StartTime: &timestamp.Timestamp{Seconds: 1},
		EndTime:   &timestamp.Timestamp{Seconds: 2},
	})
	assert.Nil(t, actualPeriodicSchedule)
}

func TestToCrdPeriodicSchedule_NilStartTime(t *testing.T) {
	actualPeriodicSchedule := toCRDPeriodicSchedule(&api.PeriodicSchedule{
		IntervalSecond: 123,
		EndTime:        &timestamp.Timestamp{Seconds: 2},
	})
	endTime := v1.NewTime(time.Unix(2, 0))
	assert.Equal(t, actualPeriodicSchedule, &scheduledworkflow.PeriodicSchedule{
		IntervalSecond: 123,
		EndTime:        &endTime,
	})
}

func TestToCrdPeriodicSchedule_NilEndTime(t *testing.T) {
	actualPeriodicSchedule := toCRDPeriodicSchedule(&api.PeriodicSchedule{
		IntervalSecond: 123,
		StartTime:      &timestamp.Timestamp{Seconds: 1},
	})
	startTime := v1.NewTime(time.Unix(1, 0))
	assert.Equal(t, actualPeriodicSchedule, &scheduledworkflow.PeriodicSchedule{
		IntervalSecond: 123,
		StartTime:      &startTime,
	})
}
