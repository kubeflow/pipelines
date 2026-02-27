// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseOptionalInt64(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *int64
	}{
		{"empty string returns nil", "", nil},
		{"whitespace returns nil", "  ", nil},
		{"valid positive", "1000", int64Ptr(1000)},
		{"zero", "0", int64Ptr(0)},
		{"negative returns nil", "-1", nil},
		{"invalid string returns nil", "abc", nil},
		{"leading/trailing whitespace trimmed", " 42 ", int64Ptr(42)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseOptionalInt64(tt.input)
			if tt.expected == nil {
				assert.Nil(t, got)
			} else {
				assert.NotNil(t, got)
				assert.Equal(t, *tt.expected, *got)
			}
		})
	}
}

func TestParseOptionalBool(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *bool
	}{
		{"empty string returns nil", "", nil},
		{"whitespace returns nil", "  ", nil},
		{"true", "true", boolPtr(true)},
		{"false", "false", boolPtr(false)},
		{"True (case insensitive)", "True", boolPtr(true)},
		{"FALSE (case insensitive)", "FALSE", boolPtr(false)},
		{"1 is true", "1", boolPtr(true)},
		{"0 is false", "0", boolPtr(false)},
		{"invalid string returns nil", "abc", nil},
		{"leading/trailing whitespace trimmed", " true ", boolPtr(true)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseOptionalBool(tt.input)
			if tt.expected == nil {
				assert.Nil(t, got)
			} else {
				assert.NotNil(t, got)
				assert.Equal(t, *tt.expected, *got)
			}
		})
	}
}

func int64Ptr(v int64) *int64 { return &v }
func boolPtr(v bool) *bool    { return &v }
