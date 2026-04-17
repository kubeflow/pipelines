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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParameterFormatter_Format(t *testing.T) {
	formatter := NewSWFParameterFormatter(
		"some-run-uuid",
		25, /* scheduled time */
		26, /* current time */
		27 /* index */)

	// Test [[RunUUID]] substitution
	assert.Equal(t, "FOO some-run-uuid FOO", formatter.Format("FOO [[RunUUID]] FOO"))

	// Test [[ScheduledTime]] substitution
	assert.Equal(t, "FOO 19700101000025 FOO", formatter.Format("FOO [[ScheduledTime]] FOO"))

	// Test [[CurrentTime]] substitution
	assert.Equal(t, "FOO 19700101000026 FOO", formatter.Format("FOO [[CurrentTime]] FOO"))

	// Test [[Index]]
	assert.Equal(t, "FOO 27 FOO", formatter.Format("FOO [[Index]] FOO"))

	// Test [[ScheduledTime.15-04-05]] substitution
	assert.Equal(t, "FOO 00-00-25 FOO", formatter.Format("FOO [[ScheduledTime.15-04-05]] FOO"))

	// Test [[CurrentTime.15-04-05]] substitution
	assert.Equal(t, "FOO 00-00-26 FOO", formatter.Format("FOO [[CurrentTime.15-04-05]] FOO"))

	// Test multiple substitution
	assert.Equal(t, "19700101000025 some-run-uuid 19700101000025 27", formatter.Format("[[ScheduledTime]] [[RunUUID]] [[ScheduledTime]] [[Index]]"))

	// Test no substitution
	assert.Equal(t, "FOO FOO FOO", formatter.Format("FOO FOO FOO"))

	// Test empty string
	assert.Equal(t, "", formatter.Format(""))

	// Test modern time formatters
	assert.Equal(t, "FOO 1970-01-01 00:00:25 FOO", formatter.Format("FOO {{$.scheduledTime.strftime('%Y-%m-%d %H:%M:%S')}} FOO"))
	assert.Equal(t, "FOO 1970-01-01 00:00:26 FOO", formatter.Format("FOO {{$.currentTime.strftime('%Y-%m-%d %H:%M:%S')}} FOO"))
}

func TestNewRunParameterFormatter(t *testing.T) {
	runUUID := "run-abc-123"
	runAt := int64(1609459200) // 2021-01-01 00:00:00 UTC
	formatter := NewRunParameterFormatter(runUUID, runAt)

	// RunUUID substitution works
	result := formatter.Format("prefix-[[RunUUID]]-suffix")
	assert.Equal(t, "prefix-run-abc-123-suffix", result)

	// CurrentTime substitution works
	result = formatter.Format("time-[[CurrentTime]]")
	assert.Equal(t, "time-20210101000000", result)

	// ScheduledTime is disabled, so it should not be substituted
	result = formatter.Format("sched-[[ScheduledTime]]")
	assert.Equal(t, "sched-[[ScheduledTime]]", result)

	// Index is disabled, so it should not be substituted
	result = formatter.Format("idx-[[Index]]")
	assert.Equal(t, "idx-[[Index]]", result)
}

func TestFormatWorkflowParameters(t *testing.T) {
	formatter := NewSWFParameterFormatter("my-run-uuid", 1609459200, 1609545600, 3)

	parameters := map[string]string{
		"run_id":    "[[RunUUID]]",
		"scheduled": "[[ScheduledTime]]",
		"current":   "[[CurrentTime]]",
		"index":     "[[Index]]",
		"plain":     "no-substitution",
	}

	result := formatter.FormatWorkflowParameters(parameters)

	assert.Equal(t, "my-run-uuid", result["run_id"])
	assert.Equal(t, "20210101000000", result["scheduled"])
	assert.Equal(t, "20210102000000", result["current"])
	assert.Equal(t, "3", result["index"])
	assert.Equal(t, "no-substitution", result["plain"])
}

func TestFormatWorkflowParameters_EmptyMap(t *testing.T) {
	formatter := NewRunParameterFormatter("uuid", 1000)
	result := formatter.FormatWorkflowParameters(map[string]string{})
	assert.Empty(t, result)
}

func TestFormat_CustomTimeFormat_ScheduledTime(t *testing.T) {
	formatter := NewSWFParameterFormatter("uuid", 1609459200, 1609545600, 1)

	// Custom scheduled time format
	result := formatter.Format("[[ScheduledTime.2006-01-02]]")
	assert.Equal(t, "2021-01-01", result)

	// Custom current time format
	result = formatter.Format("[[CurrentTime.2006-01-02]]")
	assert.Equal(t, "2021-01-02", result)
}

func TestFormat_Strftime_UnixSeconds(t *testing.T) {
	formatter := NewSWFParameterFormatter("uuid", 1609459200, 1609545600, 1)

	// Strftime with %s for unix seconds (scheduled time)
	result := formatter.Format("{{$.scheduledTime.strftime('%s')}}")
	assert.Equal(t, "1609459200", result)

	// Strftime with %s for unix seconds (current time)
	result = formatter.Format("{{$.currentTime.strftime('%s')}}")
	assert.Equal(t, "1609545600", result)
}

func TestFormat_UnknownExpression(t *testing.T) {
	formatter := NewSWFParameterFormatter("uuid", 1609459200, 1609545600, 1)

	// Unknown [[expression]] falls through to the else branch
	result := formatter.Format("[[UnknownExpression]]")
	assert.Equal(t, "[[UnknownExpression]]", result)
}

func TestFormat_DisabledScheduledTime(t *testing.T) {
	// NewRunParameterFormatter disables ScheduledTime and Index
	formatter := NewRunParameterFormatter("uuid", 1609459200)

	// ScheduledTime prefix with custom format should not be substituted when disabled
	result := formatter.Format("[[ScheduledTime.2006-01-02]]")
	assert.Equal(t, "[[ScheduledTime.2006-01-02]]", result)

	// Strftime scheduledTime should not be substituted when disabled
	result = formatter.Format("{{$.scheduledTime.strftime('%Y')}}")
	assert.Equal(t, "{{$.scheduledTime.strftime('%Y')}}", result)
}
