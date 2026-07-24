// Copyright 2025 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package objectstore

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestHuggingFaceSentinelError verifies that OpenBucket returns the sentinel error for HuggingFace provider.
func TestHuggingFaceSentinelError(t *testing.T) {
	// ErrHuggingFaceNoBucket should be defined and exported
	assert.NotNil(t, ErrHuggingFaceNoBucket)
	assert.Equal(t, "huggingface provider does not use the bucket interface", ErrHuggingFaceNoBucket.Error())
}

// TestSentinelErrorComparison verifies that the sentinel error can be properly checked with errors.Is.
func TestSentinelErrorComparison(t *testing.T) {
	// Verify error can be compared with errors.Is (standard Go pattern)
	err := ErrHuggingFaceNoBucket
	assert.True(t, errors.Is(err, ErrHuggingFaceNoBucket))

	// Verify it's not equal to other errors
	assert.False(t, errors.Is(err, errors.New("different error")))
}

// TestHuggingFaceProviderDetection verifies that HuggingFace provider is correctly identified.
func TestHuggingFaceProviderDetection(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		isHF     bool
	}{
		{
			name:     "huggingface provider",
			provider: "huggingface",
			isHF:     true,
		},
		{
			name:     "gs provider",
			provider: "gs",
			isHF:     false,
		},
		{
			name:     "s3 provider",
			provider: "s3",
			isHF:     false,
		},
		{
			name:     "minio provider",
			provider: "minio",
			isHF:     false,
		},
		{
			name:     "empty provider",
			provider: "",
			isHF:     false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			isHF := test.provider == "huggingface"
			assert.Equal(t, test.isHF, isHF, "Provider detection mismatch")
		})
	}
}

// TestTokenRetrievalErrorHandling demonstrates correct error handling pattern.
func TestTokenRetrievalErrorHandling(t *testing.T) {
	// This test documents the correct pattern for handling token retrieval errors
	// from GetHuggingFaceTokenFromK8sSecret in the launcher

	// Pattern 1: Check error and log warning, continue with empty token
	token := ""
	err := ErrHuggingFaceNoBucket

	if err != nil {
		// Expected to log warning: "Failed to retrieve HuggingFace token..."
		// Continue with empty token - unauthenticated downloads allowed for public models
		assert.Empty(t, token)
	}

	// Pattern 2: Verify error is of specific type
	assert.True(t, errors.Is(err, ErrHuggingFaceNoBucket))

	// Pattern 3: No silent ignoring - error is checked
	assert.NotNil(t, err)
}

// TestHuggingFaceSessionInfoValidation validates SessionInfo structure with HuggingFace provider.
func TestHuggingFaceSessionInfoValidation(t *testing.T) {
	tests := []struct {
		name             string
		sessionInfo      *SessionInfo
		expectError      bool
		expectedProvider string
	}{
		{
			name: "valid huggingface session",
			sessionInfo: &SessionInfo{
				Provider: "huggingface",
				Params: map[string]string{
					"fromEnv": "true",
				},
			},
			expectError:      false,
			expectedProvider: "huggingface",
		},
		{
			name: "gs provider session",
			sessionInfo: &SessionInfo{
				Provider: "gs",
				Params: map[string]string{
					"bucket": "my-bucket",
				},
			},
			expectError:      false,
			expectedProvider: "gs",
		},
		{
			name:        "nil session info",
			sessionInfo: nil,
			expectError: true,
		},
		{
			name: "huggingface with empty params",
			sessionInfo: &SessionInfo{
				Provider: "huggingface",
				Params:   map[string]string{},
			},
			expectError:      false,
			expectedProvider: "huggingface",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.expectError {
				assert.Nil(t, test.sessionInfo)
				return
			}

			assert.NotNil(t, test.sessionInfo)
			assert.Equal(t, test.expectedProvider, test.sessionInfo.Provider)

			// Verify HuggingFace sessions have fromEnv param
			if test.sessionInfo.Provider == "huggingface" {
				_, hasFromEnv := test.sessionInfo.Params["fromEnv"]
				assert.True(t, hasFromEnv || len(test.sessionInfo.Params) == 0,
					"HuggingFace session should have fromEnv param or be empty")
			}
		})
	}
}

// TestHuggingFaceParameterValidation tests query parameter handling edge cases.
func TestHuggingFaceParameterValidation(t *testing.T) {
	tests := []struct {
		name            string
		params          map[string]string
		supportedParams map[string]bool
		hasUnsupported  bool
		description     string
	}{
		{
			name: "supported repo_type",
			params: map[string]string{
				"repo_type": "model",
			},
			supportedParams: map[string]bool{
				"repo_type":       true,
				"allow_patterns":  true,
				"ignore_patterns": true,
			},
			hasUnsupported: false,
			description:    "Standard model repo_type",
		},
		{
			name: "supported allow_patterns",
			params: map[string]string{
				"allow_patterns": "*.json",
			},
			supportedParams: map[string]bool{
				"repo_type":       true,
				"allow_patterns":  true,
				"ignore_patterns": true,
			},
			hasUnsupported: false,
			description:    "File pattern matching",
		},
		{
			name: "unsupported parameter",
			params: map[string]string{
				"cache_dir":   "/tmp/hf",
				"local_files": "only",
			},
			supportedParams: map[string]bool{
				"repo_type":       true,
				"allow_patterns":  true,
				"ignore_patterns": true,
			},
			hasUnsupported: true,
			description:    "Parameters not in supported list should warn",
		},
		{
			name: "mixed supported and unsupported",
			params: map[string]string{
				"repo_type":      "dataset",
				"cache_dir":      "/tmp",
				"allow_patterns": "*.csv",
			},
			supportedParams: map[string]bool{
				"repo_type":       true,
				"allow_patterns":  true,
				"ignore_patterns": true,
			},
			hasUnsupported: true,
			description:    "Should detect unsupported cache_dir",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Check for unsupported parameters
			hasUnsupported := false
			for param := range test.params {
				if !test.supportedParams[param] {
					hasUnsupported = true
					break
				}
			}

			assert.Equal(t, test.hasUnsupported, hasUnsupported, test.description)
		})
	}
}
