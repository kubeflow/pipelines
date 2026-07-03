// Copyright 2025 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
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
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestValidateLiteralParameter(t *testing.T) {
	tests := []struct {
		name          string
		paramName     string
		value         *structpb.Value
		literals      []*structpb.Value
		expectError   bool
		errorContains string
	}{
		{
			name:      "valid input - string literal",
			paramName: "test-parameter",
			value:     structpb.NewStringValue("dev"),
			literals: []*structpb.Value{
				structpb.NewStringValue("dev"),
				structpb.NewStringValue("staging"),
				structpb.NewStringValue("prod"),
			},
			expectError: false,
		},
		{
			name:      "invalid input - string literal",
			paramName: "test-parameter",
			value:     structpb.NewStringValue("test"),
			literals: []*structpb.Value{
				structpb.NewStringValue("dev"),
				structpb.NewStringValue("staging"),
				structpb.NewStringValue("prod"),
			},
			expectError:   true,
			errorContains: "input parameter \"test-parameter\" value",
		},
		{
			name:      "invalid input - string literal, incorrect case",
			paramName: "test-parameter",
			value:     structpb.NewStringValue("Dev"),
			literals: []*structpb.Value{
				structpb.NewStringValue("dev"),
			},
			expectError:   true,
			errorContains: "input parameter \"test-parameter\" value",
		},
		{
			name:      "valid input - int literal",
			paramName: "test-parameter",
			value:     structpb.NewNumberValue(3),
			literals: []*structpb.Value{
				structpb.NewNumberValue(1),
				structpb.NewNumberValue(3),
				structpb.NewNumberValue(5),
			},
			expectError: false,
		},
		{
			name:      "invalid input - int literal",
			paramName: "test-parameter",
			value:     structpb.NewNumberValue(2),
			literals: []*structpb.Value{
				structpb.NewNumberValue(1),
				structpb.NewNumberValue(3),
				structpb.NewNumberValue(5),
			},
			expectError:   true,
			errorContains: "input parameter \"test-parameter\" value",
		},
		{
			name:      "valid input - float literal",
			paramName: "test-parameter",
			value:     structpb.NewNumberValue(0.5),
			literals: []*structpb.Value{
				structpb.NewNumberValue(0.1),
				structpb.NewNumberValue(0.5),
				structpb.NewNumberValue(0.9),
			},
			expectError: false,
		},
		{
			name:      "invalid input - float literal",
			paramName: "test-parameter",
			value:     structpb.NewNumberValue(0.3),
			literals: []*structpb.Value{
				structpb.NewNumberValue(0.1),
				structpb.NewNumberValue(0.5),
				structpb.NewNumberValue(0.9),
			},
			expectError:   true,
			errorContains: "input parameter \"test-parameter\" value",
		},
		{
			name:      "valid input - boolean literal",
			paramName: "test-parameter",
			value:     structpb.NewBoolValue(true),
			literals: []*structpb.Value{
				structpb.NewBoolValue(true),
			},
			expectError: false,
		},
		{
			name:      "invalid input - boolean literal",
			paramName: "test-parameter",
			value:     structpb.NewBoolValue(false),
			literals: []*structpb.Value{
				structpb.NewBoolValue(true),
			},
			expectError:   true,
			errorContains: "input parameter \"test-parameter\" value",
		},
		{
			name:        "valid input - nil literals field",
			paramName:   "test-parameter",
			value:       structpb.NewStringValue("anything"),
			literals:    nil,
			expectError: false,
		},
		{
			name:        "valid input - empty literals field",
			paramName:   "test-parameter",
			value:       structpb.NewStringValue("anything"),
			literals:    []*structpb.Value{},
			expectError: false,
		},
		{
			name:      "valid input - single-value literal",
			paramName: "test-parameter",
			value:     structpb.NewStringValue("only_option"),
			literals: []*structpb.Value{
				structpb.NewStringValue("only_option"),
			},
			expectError: false,
		},
		{
			name:      "invalid input - single-value literal",
			paramName: "test-parameter",
			value:     structpb.NewStringValue("other"),
			literals: []*structpb.Value{
				structpb.NewStringValue("only_option"),
			},
			expectError:   true,
			errorContains: "input parameter \"test-parameter\" value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLiteralParameter(tt.paramName, tt.value, tt.literals)

			if tt.expectError {
				require.Error(t, err, "Expected error but got nil")
				if tt.errorContains != "" {
					assert.ErrorContains(t, err, tt.errorContains, "Error message doesn't contain expected text")
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateLiteralParameter_EmptyLiterals(t *testing.T) {
	// Test that empty literals (no constraints) allows any value
	value := structpb.NewStringValue("test")

	err := ValidateLiteralParameter("test-parameter", value, nil)
	assert.NoError(t, err, "Empty literals should not cause validation error")
}
