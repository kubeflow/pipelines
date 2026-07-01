package server

import (
	"strings"
	"testing"

	apiv1beta1 "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"google.golang.org/protobuf/types/known/structpb"
)

// TestValidateRuntimeConfigV1 increases coverage for api_util.go:380
func TestValidateRuntimeConfigV1(t *testing.T) {
	// 1. Create a "Huge" string that exceeds the limit (util.MaxParameterBytes)
	// MaxParameterBytes is usually around 64KB or similar. We add 100 bytes to ensure overflow.
	hugeString := make([]byte, util.MaxParameterBytes+100)
	for i := range hugeString {
		hugeString[i] = 'a'
	}

	tests := []struct {
		name      string
		config    *apiv1beta1.PipelineSpec_RuntimeConfig
		wantErr   bool
		errSubstr string // Substring we expect in the error message
	}{
		{
			name: "Valid small parameters",
			config: &apiv1beta1.PipelineSpec_RuntimeConfig{
				Parameters: map[string]*structpb.Value{
					"param1": {Kind: &structpb.Value_StringValue{StringValue: "small-value"}},
				},
			},
			wantErr: false,
		},
		{
			name:   "Nil parameters",
			config: &apiv1beta1.PipelineSpec_RuntimeConfig{},
			wantErr: false,
		},
		{
			name: "Parameters exceed max size",
			config: &apiv1beta1.PipelineSpec_RuntimeConfig{
				Parameters: map[string]*structpb.Value{
					// This giant value should trigger the size limit check
					"huge_param": {Kind: &structpb.Value_StringValue{StringValue: string(hugeString)}},
				},
			},
			wantErr:   true,
			errSubstr: "The input parameter length exceed maximum size",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRuntimeConfigV1(tt.config)
			
			// Check if error presence matches expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("validateRuntimeConfigV1() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			// Check if error message matches expectation
			if tt.wantErr && err != nil {
				if !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("expected error containing %q, got %q", tt.errSubstr, err.Error())
				}
			}
		})
	}
}

// TestValidatePipelineManifest increases coverage for api_util.go:416
func TestValidatePipelineManifest(t *testing.T) {
	tests := []struct {
		name             string
		pipelineManifest string
		wantErr          bool
	}{
		{
			name:             "Empty manifest",
			pipelineManifest: "",
			wantErr:          false,
		},
		{
			name:             "Valid YAML",
			pipelineManifest: "apiVersion: argoproj.io/v1alpha1\nkind: Workflow",
			wantErr:          false,
		},
		{
			name:             "Invalid YAML",
			// A colon without a value or key structure is invalid YAML
			pipelineManifest: ": : : invalid yaml : : :",
			wantErr:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePipelineManifest(tt.pipelineManifest)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePipelineManifest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}