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
)

// TimeInterface is an interface for objects generating the current time.
type TimeInterface interface {
	Now() time.Time
}

// RealTime is an implementation of TimeInterface that generates the current time.
type RealTime struct {
}

// NewRealTime creates an instance of RealTime.
func NewRealTime() TimeInterface {
	return &RealTime{}
}

// Now returns the current time.
func (r *RealTime) Now() time.Time {
	return time.Now().UTC()
}

// FakeTime is a fake implementation of TimeInterface for testing.
type FakeTime struct {
	now time.Time
}

// NewFakeTime creates an instance of FakeTime that will return a fixed time.
func NewFakeTime(now time.Time) TimeInterface {
	return &FakeTime{
		now: now.UTC(),
	}
}

// NewFakeTimeForEpoch creates an instance of FakeTime that will return a fixed epoch.
func NewFakeTimeForEpoch() TimeInterface {
	return &FakeTime{
		now: time.Unix(0, 0).UTC(),
	}
}

// Now returns the current (fake) time.
func (f *FakeTime) Now() time.Time {
	f.now = time.Unix(f.now.Unix()+1, 0).UTC()
	return f.now
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
