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

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/assert"
)

func TestValidateFieldLength_Valid(t *testing.T) {
	// Exact max length should pass
	value := strings.Repeat("a", 191)
	err := ValidateFieldLength("Task", "RunID", value)
	assert.Nil(t, err)
}

func TestValidateFieldLength_TooLong(t *testing.T) {
	// Length exceeding max should return error
	value := strings.Repeat("a", 192)
	err := ValidateFieldLength("Task", "RunID", value)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Task.RunID length cannot exceed 191")
}

func TestValidateFieldLength_NoSpec(t *testing.T) {
	err := ValidateFieldLength("NonExistentModel", "NonExistentField", "value")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Length spec missing for NonExistentModel.NonExistentField")
}

func TestValidateModel(t *testing.T) {
	tests := []struct {
		name          string
		input         interface{}
		expectError   bool
		errorContains string
	}{
		{
			name:          "nil pointer returns InternalServerError",
			input:         nil,
			expectError:   true,
			errorContains: "non-nil pointer",
		},
		{
			name:          "non-pointer value type returns InternalServerError",
			input:         model.Pipeline{},
			expectError:   true,
			errorContains: "non-nil pointer",
		},
		{
			name: "valid model passes validation",
			input: &model.Pipeline{
				UUID:      "abc",
				Name:      "test",
				Namespace: "ns",
			},
			expectError: false,
		},
		{
			name: "field exceeding max length returns InvalidInputError",
			input: &model.Pipeline{
				Name: strings.Repeat("a", 129),
			},
			expectError:   true,
			errorContains: "Pipeline.Name length cannot exceed 128",
		},
		{
			name: "fields without length specs are skipped",
			input: &model.Experiment{
				UUID:                  "x",
				Name:                  "y",
				Namespace:             "z",
				CreatedAtInSec:        9999999,
				LastRunCreatedAtInSec: 9999999,
			},
			expectError: false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			err := ValidateModel(testCase.input)
			if testCase.expectError {
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), testCase.errorContains)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}
