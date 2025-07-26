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

package validation

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateFieldLength_Valid(t *testing.T) {
	// Exact max length should pass
	value := strings.Repeat("a", 191)
	err := ValidateFieldLength("Task", "RunId", value)
	assert.Nil(t, err)
}

func TestValidateFieldLength_TooLong(t *testing.T) {
	// Length exceeding max should return error
	value := strings.Repeat("a", 192)
	err := ValidateFieldLength("Task", "RunId", value)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Task.RunId length cannot exceed 191")
}

func TestValidateFieldLength_NoSpec(t *testing.T) {
	err := ValidateFieldLength("NonExistentModel", "NonExistentField", "value")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Length spec missing for NonExistentModel.NonExistentField")
}
