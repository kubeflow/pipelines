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
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewUUIDGenerator(t *testing.T) {
	generator := NewUUIDGenerator()
	assert.NotNil(t, generator)
}

func TestUUIDGeneratorNewRandom(t *testing.T) {
	generator := NewUUIDGenerator()
	generatedUUID, err := generator.NewRandom()
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, generatedUUID)
}

func TestNewFakeUUIDGeneratorOrFatal(t *testing.T) {
	expectedUUID := "123e4567-e89b-12d3-a456-426614174000"
	generator := NewFakeUUIDGeneratorOrFatal(expectedUUID, nil)
	assert.NotNil(t, generator)

	generatedUUID, err := generator.NewRandom()
	assert.NoError(t, err)
	assert.Equal(t, expectedUUID, generatedUUID.String())
}

func TestFakeUUIDGeneratorWithError(t *testing.T) {
	expectedUUID := "123e4567-e89b-12d3-a456-426614174000"
	expectedError := fmt.Errorf("fake error")
	generator := NewFakeUUIDGeneratorOrFatal(expectedUUID, expectedError)

	generatedUUID, err := generator.NewRandom()
	assert.Equal(t, expectedError, err)
	assert.Equal(t, expectedUUID, generatedUUID.String())
}
