// Copyright 2025 The Kubeflow Authors
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

	"github.com/stretchr/testify/assert"
)

func TestRealTimeNow(t *testing.T) {
	realTime := NewRealTime()
	assert.NotNil(t, realTime)

	before := time.Now().UTC()
	now := realTime.Now()
	after := time.Now().UTC()

	assert.Equal(t, time.UTC, now.Location())
	assert.False(t, now.Before(before))
	assert.False(t, now.After(after))
}

func TestFakeTimeNow(t *testing.T) {
	startTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	fakeTime := NewFakeTime(startTime)
	assert.NotNil(t, fakeTime)

	// Each call to Now() increments by 1 second
	firstCall := fakeTime.Now()
	assert.Equal(t, startTime.Add(1*time.Second), firstCall)

	secondCall := fakeTime.Now()
	assert.Equal(t, startTime.Add(2*time.Second), secondCall)

	thirdCall := fakeTime.Now()
	assert.Equal(t, startTime.Add(3*time.Second), thirdCall)
}

func TestNewFakeTimeForEpoch(t *testing.T) {
	fakeTime := NewFakeTimeForEpoch()
	assert.NotNil(t, fakeTime)

	firstCall := fakeTime.Now()
	assert.Equal(t, time.Unix(1, 0).UTC(), firstCall)

	secondCall := fakeTime.Now()
	assert.Equal(t, time.Unix(2, 0).UTC(), secondCall)
}

func TestFormatTimeForLogging(t *testing.T) {
	testCases := []struct {
		name     string
		epoch    int64
		expected string
	}{
		{
			name:     "negative epoch returns INVALID TIME",
			epoch:    -1,
			expected: "INVALID TIME",
		},
		{
			name:     "zero epoch returns INVALID TIME",
			epoch:    0,
			expected: "INVALID TIME",
		},
		{
			name:     "MaxInt64 returns NEVER",
			epoch:    math.MaxInt64,
			expected: "NEVER",
		},
		{
			name:     "normal epoch returns formatted time",
			epoch:    1609459200,
			expected: time.Unix(1609459200, 0).UTC().String(),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := FormatTimeForLogging(testCase.epoch)
			assert.Equal(t, testCase.expected, result)
		})
	}
}

func TestParseTimeOrFatal(t *testing.T) {
	validTime := "2025-06-15T12:30:00Z"
	result := ParseTimeOrFatal(validTime)
	expected := time.Date(2025, 6, 15, 12, 30, 0, 0, time.UTC)
	assert.Equal(t, expected, result)
	assert.Equal(t, time.UTC, result.Location())
}
