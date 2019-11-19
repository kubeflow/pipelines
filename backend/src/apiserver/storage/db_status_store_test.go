// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitializeDBStatusTable(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	dBStatusStore := NewDBStatusStore(db)

	// Initialize for the first time
	err := dBStatusStore.InitializeDBStatusTable()
	assert.Nil(t, err)
	// Initialize again should be no-op and no error
	err = dBStatusStore.InitializeDBStatusTable()
	assert.Nil(t, err)
	haveSamplesLoaded, err := dBStatusStore.HaveSamplesLoaded()
	assert.Nil(t, err)
	assert.False(t, haveSamplesLoaded)

	db.Close()
	err = dBStatusStore.InitializeDBStatusTable()
	assert.NotNil(t, err)
}

func TestMarkSampleLoaded(t *testing.T) {
	db := NewFakeDbOrFatal()
	defer db.Close()
	dBStatusStore := NewDBStatusStore(db)

	// Initialize for the first time
	err := dBStatusStore.InitializeDBStatusTable()
	assert.Nil(t, err)
	// Mark the samples as loaded
	err = dBStatusStore.MarkSampleLoaded()
	assert.Nil(t, err)
	haveSamplesLoaded, err := dBStatusStore.HaveSamplesLoaded()
	assert.Nil(t, err)
	assert.True(t, haveSamplesLoaded)

	db.Close()
	err = dBStatusStore.MarkSampleLoaded()
	assert.NotNil(t, err)
}
