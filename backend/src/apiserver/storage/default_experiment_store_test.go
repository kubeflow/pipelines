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

func TestInitializeDefaultExperimentTable(t *testing.T) {
	db := NewFakeDbOrFatal()
	defaultExperimentStore := NewDefaultExperimentStore(db)

	// Initialize for the first time
	err := defaultExperimentStore.initializeDefaultExperimentTable()
	assert.Nil(t, err)
	// Initialize again should be no-op and no error
	err = defaultExperimentStore.initializeDefaultExperimentTable()
	assert.Nil(t, err)
	// Default experiment ID is empty after table initialization
	defaultExperimentId, err := defaultExperimentStore.GetDefaultExperimentId()
	assert.Nil(t, err)
	assert.Equal(t, "", defaultExperimentId)

	// Initializing the table with an invalid DB is an error
	db.Close()
	err = defaultExperimentStore.initializeDefaultExperimentTable()
	assert.NotNil(t, err)
}

func TestGetAndSetDefaultExperimentId(t *testing.T) {
	db := NewFakeDbOrFatal()
	defaultExperimentStore := NewDefaultExperimentStore(db)

	// Initialize for the first time
	err := defaultExperimentStore.initializeDefaultExperimentTable()
	assert.Nil(t, err)
	// Set the default experiment ID
	err = defaultExperimentStore.SetDefaultExperimentId("test-ID")
	assert.Nil(t, err)
	// Get the default experiment ID
	defaultExperimentId, err := defaultExperimentStore.GetDefaultExperimentId()
	assert.Nil(t, err)
	assert.Equal(t, "test-ID", defaultExperimentId)
	// Trying to set the default experiment ID again is not an error, but the ID is not changed
	err = defaultExperimentStore.SetDefaultExperimentId("a-different-ID")
	assert.Nil(t, err)
	defaultExperimentId, err = defaultExperimentStore.GetDefaultExperimentId()
	assert.Nil(t, err)
	assert.Equal(t, "test-ID", defaultExperimentId)

	// Setting or getting the default experiment ID with an invalid DB is an error
	db.Close()
	err = defaultExperimentStore.SetDefaultExperimentId("some-ID")
	assert.NotNil(t, err)
	_, err = defaultExperimentStore.GetDefaultExperimentId()
	assert.NotNil(t, err)
}

func TestUnsetDefaultExperimentIdIfIdMatches(t *testing.T) {
	db := NewFakeDbOrFatal()
	defaultExperimentStore := NewDefaultExperimentStore(db)

	// Initialize for the first time
	err := defaultExperimentStore.initializeDefaultExperimentTable()
	assert.Nil(t, err)
	// Set the default experiment ID
	err = defaultExperimentStore.SetDefaultExperimentId("test-ID")
	assert.Nil(t, err)
	// Clear the default experiment ID. This requires a transaction.
	tx, _ := db.Begin()
	err = defaultExperimentStore.UnsetDefaultExperimentIdIfIdMatches(tx, "test-ID")
	assert.Nil(t, err)
	err = tx.Commit()
	assert.Nil(t, err)
	// Get the default experiment ID
	defaultExperimentId, err := defaultExperimentStore.GetDefaultExperimentId()
	assert.Nil(t, err)
	assert.Equal(t, "", defaultExperimentId)

	db.Close()
}
