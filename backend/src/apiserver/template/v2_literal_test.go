package template

import (
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/structpb"
)

// TestValidatePipelineJobInputs_LiteralValidation tests the literal parameter validation
// functionality added for KEP-11385.
func TestValidatePipelineJobInputs_LiteralValidation(t *testing.T) {
	tests := []struct {
		name          string
		paramSpec     *pipelinespec.ComponentInputsSpec_ParameterSpec
		runtimeValue  *structpb.Value
		expectError   bool
		errorContains string
	}{
		{
			name: "valid literal string match",
			paramSpec: &pipelinespec.ComponentInputsSpec_ParameterSpec{
				ParameterType: pipelinespec.ParameterType_STRING,
				Literals: []*structpb.Value{
					structpb.NewStringValue("dev"),
					structpb.NewStringValue("staging"),
					structpb.NewStringValue("prod"),
				},
			},
			runtimeValue: structpb.NewStringValue("dev"),
			expectError:  false,
		},
		{
			name: "invalid literal string - doesn't match",
			paramSpec: &pipelinespec.ComponentInputsSpec_ParameterSpec{
				ParameterType: pipelinespec.ParameterType_STRING,
				Literals: []*structpb.Value{
					structpb.NewStringValue("dev"),
					structpb.NewStringValue("staging"),
					structpb.NewStringValue("prod"),
				},
			},
			runtimeValue:  structpb.NewStringValue("test"),
			expectError:   true,
			errorContains: "does not match any of the allowed literal values",
		},
		{
			name: "valid literal number match",
			paramSpec: &pipelinespec.ComponentInputsSpec_ParameterSpec{
				ParameterType: pipelinespec.ParameterType_NUMBER_INTEGER,
				Literals: []*structpb.Value{
					structpb.NewNumberValue(1),
					structpb.NewNumberValue(3),
					structpb.NewNumberValue(5),
				},
			},
			runtimeValue: structpb.NewNumberValue(3),
			expectError:  false,
		},
		{
			name: "invalid literal number - doesn't match",
			paramSpec: &pipelinespec.ComponentInputsSpec_ParameterSpec{
				ParameterType: pipelinespec.ParameterType_NUMBER_INTEGER,
				Literals: []*structpb.Value{
					structpb.NewNumberValue(1),
					structpb.NewNumberValue(3),
					structpb.NewNumberValue(5),
				},
			},
			runtimeValue:  structpb.NewNumberValue(2),
			expectError:   true,
			errorContains: "does not match any of the allowed literal values",
		},
		{
			name: "valid literal boolean match",
			paramSpec: &pipelinespec.ComponentInputsSpec_ParameterSpec{
				ParameterType: pipelinespec.ParameterType_BOOLEAN,
				Literals: []*structpb.Value{
					structpb.NewBoolValue(true),
				},
			},
			runtimeValue: structpb.NewBoolValue(true),
			expectError:  false,
		},
		{
			name: "invalid literal boolean - doesn't match",
			paramSpec: &pipelinespec.ComponentInputsSpec_ParameterSpec{
				ParameterType: pipelinespec.ParameterType_BOOLEAN,
				Literals: []*structpb.Value{
					structpb.NewBoolValue(true),
				},
			},
			runtimeValue:  structpb.NewBoolValue(false),
			expectError:   true,
			errorContains: "does not match any of the allowed literal values",
		},
		{
			name: "no literals - backward compatibility (any value allowed)",
			paramSpec: &pipelinespec.ComponentInputsSpec_ParameterSpec{
				ParameterType: pipelinespec.ParameterType_STRING,
				Literals:      nil,
			},
			runtimeValue: structpb.NewStringValue("any-value-is-fine"),
			expectError:  false,
		},
		{
			name: "empty literals array - backward compatibility",
			paramSpec: &pipelinespec.ComponentInputsSpec_ParameterSpec{
				ParameterType: pipelinespec.ParameterType_STRING,
				Literals:      []*structpb.Value{},
			},
			runtimeValue: structpb.NewStringValue("any-value-is-fine"),
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a minimal V2Spec with the parameter spec
			v2Spec := &V2Spec{
				spec: &pipelinespec.PipelineSpec{
					PipelineInfo: &pipelinespec.PipelineInfo{
						Name: "test-pipeline",
					},
					SchemaVersion: "2.1.0",
					Root: &pipelinespec.ComponentSpec{
						InputDefinitions: &pipelinespec.ComponentInputsSpec{
							Parameters: map[string]*pipelinespec.ComponentInputsSpec_ParameterSpec{
								"test_param": tt.paramSpec,
							},
						},
					},
				},
			}

			// Create a pipeline job with runtime parameters
			job := &pipelinespec.PipelineJob{
				RuntimeConfig: &pipelinespec.PipelineJob_RuntimeConfig{
					ParameterValues: map[string]*structpb.Value{
						"test_param": tt.runtimeValue,
					},
				},
			}

			// Test validation
			err := v2Spec.validatePipelineJobInputs(job)

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

// TestValidatePipelineJobInputs_LiteralValidation_MultipleParams tests validation with multiple parameters
func TestValidatePipelineJobInputs_LiteralValidation_MultipleParams(t *testing.T) {
	v2Spec := &V2Spec{
		spec: &pipelinespec.PipelineSpec{
			PipelineInfo: &pipelinespec.PipelineInfo{
				Name: "test-pipeline",
			},
			SchemaVersion: "2.1.0",
			Root: &pipelinespec.ComponentSpec{
				InputDefinitions: &pipelinespec.ComponentInputsSpec{
					Parameters: map[string]*pipelinespec.ComponentInputsSpec_ParameterSpec{
						"env": {
							ParameterType: pipelinespec.ParameterType_STRING,
							Literals: []*structpb.Value{
								structpb.NewStringValue("dev"),
								structpb.NewStringValue("prod"),
							},
						},
						"replicas": {
							ParameterType: pipelinespec.ParameterType_NUMBER_INTEGER,
							Literals: []*structpb.Value{
								structpb.NewNumberValue(1),
								structpb.NewNumberValue(3),
							},
						},
					},
				},
			},
		},
	}

	// Test all valid
	job := &pipelinespec.PipelineJob{
		RuntimeConfig: &pipelinespec.PipelineJob_RuntimeConfig{
			ParameterValues: map[string]*structpb.Value{
				"env":      structpb.NewStringValue("dev"),
				"replicas": structpb.NewNumberValue(3),
			},
		},
	}
	err := v2Spec.validatePipelineJobInputs(job)
	assert.Nil(t, err, "Expected no error for all valid values")

	// Test one invalid
	job = &pipelinespec.PipelineJob{
		RuntimeConfig: &pipelinespec.PipelineJob_RuntimeConfig{
			ParameterValues: map[string]*structpb.Value{
				"env":      structpb.NewStringValue("dev"),
				"replicas": structpb.NewNumberValue(2), // Invalid!
			},
		},
	}
	err = v2Spec.validatePipelineJobInputs(job)
	assert.NotNil(t, err, "Expected error for invalid replicas value")
	assert.Contains(t, err.Error(), "does not match any of the allowed literal values")
}
