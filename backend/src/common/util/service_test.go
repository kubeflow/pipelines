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
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/rest"
)

func TestExtractMasterIPAndPort(t *testing.T) {
	testCases := []struct {
		name     string
		host     string
		expected string
	}{
		{
			name:     "strips http prefix",
			host:     "http://host:8080",
			expected: "host:8080",
		},
		{
			name:     "strips https prefix",
			host:     "https://host:443",
			expected: "host:443",
		},
		{
			name:     "no prefix unchanged",
			host:     "host:8080",
			expected: "host:8080",
		},
		{
			name:     "empty host",
			host:     "",
			expected: "",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			config := &rest.Config{Host: testCase.host}
			result := ExtractMasterIPAndPort(config)
			assert.Equal(t, testCase.expected, result)
		})
	}
}
