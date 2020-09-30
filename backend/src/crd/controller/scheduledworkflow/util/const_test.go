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
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestgetLocationDefault(t *testing.T) {
	location, err := getLocation()
	assert.Nil(t, err)
	assert.Equal(t, time.Now().Location(), location)
}

func TestgetLocationEnvSet(t *testing.T) {
	locationString := "Asia/Shanghai"
	os.Setenv("CRON_SCHEDULE_TIMEZONE", locationString)
	location, err := getLocation()
	assert.Nil(t, err)
	expectedLocation, err := time.LoadLocation(locationString)
	assert.Nil(t, err)
	assert.Equal(t, expectedLocation, location)
}
