// Copyright 2026 The Kubeflow Authors
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

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLeaderElectionEnabled(t *testing.T) {
	tests := []struct {
		value       string
		wantEnabled bool
		wantError   bool
	}{
		{value: "", wantEnabled: false},
		{value: "false", wantEnabled: false},
		{value: "true", wantEnabled: true},
		{value: "TRUE", wantEnabled: true},
		{value: "yes", wantError: true},
	}
	for _, test := range tests {
		t.Run("value="+test.value, func(t *testing.T) {
			enabled, parseError := leaderElectionEnabled(test.value)
			if test.wantError {
				assert.ErrorContains(t, parseError, "set it to \"true\" or \"false\"")
				return
			}
			assert.NoError(t, parseError)
			assert.Equal(t, test.wantEnabled, enabled)
		})
	}
}
