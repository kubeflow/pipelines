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

	"github.com/stretchr/testify/assert"
)

func TestParameterFormatter_Format(t *testing.T) {
	formatter := NewParameterFormatter(
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

	// Test [[ScheduledTime.15-04-05]] substition
	assert.Equal(t, "FOO 00-00-25 FOO", formatter.Format("FOO [[ScheduledTime.15-04-05]] FOO"))

	// Test [[CurrentTime.15-04-05]] substitution
	assert.Equal(t, "FOO 00-00-26 FOO", formatter.Format("FOO [[CurrentTime.15-04-05]] FOO"))

	// Test multiple substitution
	assert.Equal(t, "19700101000025 some-run-uuid 19700101000025 27", formatter.Format("[[ScheduledTime]] [[RunUUID]] [[ScheduledTime]] [[Index]]"))

	// Test no substitution
	assert.Equal(t, "FOO FOO FOO", formatter.Format("FOO FOO FOO"))

	// Test empty string
	assert.Equal(t, "", formatter.Format(""))
}
