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

	"github.com/golang/glog"
)

type TimeInterface interface {
	Now() time.Time
}

type RealTime struct {
}

func NewRealTime() TimeInterface {
	return &RealTime{}
}

func (r *RealTime) Now() time.Time {
	return time.Now().UTC()
}

type FakeTime struct {
	now time.Time
}

func NewFakeTime(now time.Time) TimeInterface {
	return &FakeTime{
		now: now.UTC(),
	}
}

func NewFakeTimeForEpoch() TimeInterface {
	return &FakeTime{
		now: time.Unix(0, 0).UTC(),
	}
}

func (f *FakeTime) Now() time.Time {
	f.now = time.Unix(f.now.Unix()+1, 0).UTC()
	return f.now
}

func ParseTimeOrFatal(value string) time.Time {
	result, err := time.Parse(time.RFC3339, value)
	if err != nil {
		glog.Fatalf("Could not parse time: %+v", err)
	}
	return result.UTC()
}

// FormatTimeForLogging formats an epoch for logging purposes.
func FormatTimeForLogging(epoch int64) string {
	if epoch <= 0 {
		return "INVALID TIME"
	} else if epoch == math.MaxInt64 {
		return "NEVER"
	} else {
		return time.Unix(epoch, 0).UTC().String()
	}
}
