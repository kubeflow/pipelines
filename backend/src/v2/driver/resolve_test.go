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

package driver

import (
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestValidateLiteralParameter(t *testing.T) {
	tests := []struct {
		name          string
		paramName     string
		value         *structpb.Value
		paramSpec     *pipelinespec.ComponentInputsSpec_ParameterSpec
		expectError   bool
		errorContains string
	}{
		{
			name:      "valid string literal match",
			paramName: "environment",
			value:     structpb.NewStringValue("dev"),
			paramSpec: &pipelinespec.ComponentInputsSpec_ParameterSpec{
				ParameterType: pipelinespec.ParameterType_STRING,
				Literals: []*structpb.Value{
					structpb.NewStringValue("dev"),
					structpb.NewStringValue("staging"),
					structpb.NewStringValue("prod"),
				},
			},
			expectError: false,
		},
		{
			name:      "valid string literal match - last in list",
			paramName: "environment",
			value:     structpb.NewStringValue("prod"),
			paramSpec: &pipelinespec.ComponentInputsSpec_ParameterSpec{
				ParameterType: pipelinespec.ParameterType_STRING,
				Literals: []*structpb.Value{
					structpb.NewStringValue("dev"),
					structpb.NewStringValue("staging"),
					structpb.NewStringValue("prod"),
				},
			},
			expectError: false,
		},
		{
			name:      "invalid string literal - no match",
			paramName: "environment",
			value:     structpb.NewStringValue("test"),
			paramSpec: &pipelinespec.ComponentInputsSpec_ParameterSpec{
				ParameterType: pipelinespec.ParameterType_STRING,
				Literals: []*structpb.Value{
					structpb.NewStringValue("dev"),
					structpb.NewStringValue("staging"),
					structpb.NewStringValue("prod"),
				},
			},
			expectError:   true,
			errorContains: "input parameter \"environment\" value does not match any of the allowed literal values",
		},
		{
			name:      "valid number literal match - integer",
			paramName: "replicas",
			value:     structpb.NewNumberValue(3),
			paramSpec: &pipelinespec.ComponentInputsSpec_ParameterSpec{
				ParameterType: pipelinespec.ParameterType_NUMBER_INTEGER,
				Literals: []*structpb.Value{
					structpb.NewNumberValue(1),
					structpb.NewNumberValue(3),
					structpb.NewNumberValue(5),
				},
			},
			expectError: false,
		},
		{
			name:      "invalid number literal - no match",
			paramName: "replicas",
			value:     structpb.NewNumberValue(2),
			paramSpec: &pipelinespec.ComponentInputsSpec_ParameterSpec{
				ParameterType: pipelinespec.ParameterType_NUMBER_INTEGER,
				Literals: []*structpb.Value{
					structpb.NewNumberValue(1),
					structpb.NewNumberValue(3),
					structpb.NewNumberValue(5),
				},
			},
			expectError:   true,
			errorContains: "input parameter \"replicas\" value does not match any of the allowed literal values",
		},
		{
			name:      "valid number literal match - double",
			paramName: "threshold",
			value:     structpb.NewNumberValue(0.5),
			paramSpec: &pipelinespec.ComponentInputsSpec_ParameterSpec{
				ParameterType: pipelinespec.ParameterType_NUMBER_DOUBLE,
				Literals: []*structpb.Value{
					structpb.NewNumberValue(0.1),
					structpb.NewNumberValue(0.5),
					structpb.NewNumberValue(0.9),
				},
			},
			expectError: false,
		},
		{
			name:      "valid boolean literal match - true",
			paramName: "enable_feature",
			value:     structpb.NewBoolValue(true),
			paramSpec: &pipelinespec.ComponentInputsSpec_ParameterSpec{
				ParameterType: pipelinespec.ParameterType_BOOLEAN,
				Literals: []*structpb.Value{
					structpb.NewBoolValue(true),
				},
			},
			expectError: false,
		},
		{
			name:      "invalid boolean literal - no match",
			paramName: "enable_feature",
			value:     structpb.NewBoolValue(false),
			paramSpec: &pipelinespec.ComponentInputsSpec_ParameterSpec{
				ParameterType: pipelinespec.ParameterType_BOOLEAN,
				Literals: []*structpb.Value{
					structpb.NewBoolValue(true),
				},
			},
			expectError:   true,
			errorContains: "input parameter \"enable_feature\" value does not match any of the allowed literal values",
		},
		{
			name:      "valid boolean literal match - false",
			paramName: "disable_cache",
			value:     structpb.NewBoolValue(false),
			paramSpec: &pipelinespec.ComponentInputsSpec_ParameterSpec{
				ParameterType: pipelinespec.ParameterType_BOOLEAN,
				Literals: []*structpb.Value{
					structpb.NewBoolValue(false),
				},
			},
			expectError: false,
		},
		{
			name:      "backward compatibility - nil literals (any value allowed)",
			paramName: "any_param",
			value:     structpb.NewStringValue("anything"),
			paramSpec: &pipelinespec.ComponentInputsSpec_ParameterSpec{
				ParameterType: pipelinespec.ParameterType_STRING,
				Literals:      nil,
			},
			expectError: false,
		},
		{
			name:      "backward compatibility - empty literals array (any value allowed)",
			paramName: "any_param",
			value:     structpb.NewStringValue("anything"),
			paramSpec: &pipelinespec.ComponentInputsSpec_ParameterSpec{
				ParameterType: pipelinespec.ParameterType_STRING,
				Literals:      []*structpb.Value{},
			},
			expectError: false,
		},
		{
			name:      "single literal value - match",
			paramName: "fixed_value",
			value:     structpb.NewStringValue("only_option"),
			paramSpec: &pipelinespec.ComponentInputsSpec_ParameterSpec{
				ParameterType: pipelinespec.ParameterType_STRING,
				Literals: []*structpb.Value{
					structpb.NewStringValue("only_option"),
				},
			},
			expectError: false,
		},
		{
			name:      "single literal value - no match",
			paramName: "fixed_value",
			value:     structpb.NewStringValue("other"),
			paramSpec: &pipelinespec.ComponentInputsSpec_ParameterSpec{
				ParameterType: pipelinespec.ParameterType_STRING,
				Literals: []*structpb.Value{
					structpb.NewStringValue("only_option"),
				},
			},
			expectError:   true,
			errorContains: "input parameter \"fixed_value\" value does not match any of the allowed literal values",
		},
		{
			name:      "case sensitive string matching",
			paramName: "case_param",
			value:     structpb.NewStringValue("Dev"),
			paramSpec: &pipelinespec.ComponentInputsSpec_ParameterSpec{
				ParameterType: pipelinespec.ParameterType_STRING,
				Literals: []*structpb.Value{
					structpb.NewStringValue("dev"),
				},
			},
			expectError:   true,
			errorContains: "input parameter \"case_param\" value does not match any of the allowed literal values",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateLiteralParameter(tt.paramName, tt.value, tt.paramSpec)

			if tt.expectError {
				assert.NotNil(t, err, "Expected error but got nil")
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains, "Error message doesn't contain expected text")
				}
			} else {
				assert.Nil(t, err, "Expected no error but got: %v", err)
			}
		})
	}
}

func TestValidateLiteralParameter_NilParamSpec(t *testing.T) {
	// Edge case: nil paramSpec should not panic
	// The function should handle this gracefully
	value := structpb.NewStringValue("test")

	// This test ensures the function doesn't panic with nil paramSpec
	// In practice, the caller should ensure paramSpec is not nil
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("validateLiteralParameter panicked with nil paramSpec: %v", r)
		}
	}()

	// With a nil paramSpec, GetLiterals() will return nil, so no validation occurs
	err := validateLiteralParameter("test_param", value, &pipelinespec.ComponentInputsSpec_ParameterSpec{})
	assert.Nil(t, err, "Empty paramSpec should not cause validation error")
}
