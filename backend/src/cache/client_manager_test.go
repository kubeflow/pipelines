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

func TestParseDBExtraParams(t *testing.T) {
	tests := []struct {
		name           string
		dbExtraParams  string
		expectedParams map[string]string
		expectError    bool
	}{
		{
			name:           "empty input yields empty map",
			dbExtraParams:  "",
			expectedParams: map[string]string{},
			expectError:    false,
		},
		{
			name:           "valid empty JSON object",
			dbExtraParams:  "{}",
			expectedParams: map[string]string{},
			expectError:    false,
		},
		{
			name:           "valid PostgreSQL TLS params",
			dbExtraParams:  `{"sslmode":"verify-full","sslrootcert":"/certs/ca.crt"}`,
			expectedParams: map[string]string{"sslmode": "verify-full", "sslrootcert": "/certs/ca.crt"},
			expectError:    false,
		},
		{
			name:           "valid MySQL TLS params",
			dbExtraParams:  `{"tls":"true","charset":"utf8mb4"}`,
			expectedParams: map[string]string{"tls": "true", "charset": "utf8mb4"},
			expectError:    false,
		},
		{
			name:          "malformed JSON with single quotes",
			dbExtraParams: `{'sslmode':'require'}`,
			expectError:   true,
		},
		{
			name:          "malformed JSON with key=value syntax",
			dbExtraParams: "{sslmode=verify-full}",
			expectError:   true,
		},
		{
			name:          "malformed JSON without braces",
			dbExtraParams: "sslmode=disable",
			expectError:   true,
		},
		{
			name:          "truncated JSON",
			dbExtraParams: `{"sslmode":`,
			expectError:   true,
		},
		{
			name:          "valid JSON with non-string value",
			dbExtraParams: `{"timeout": 30}`,
			expectError:   true,
		},
		{
			name:          "valid JSON array instead of object",
			dbExtraParams: `["sslmode","verify-full"]`,
			expectError:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			extraParams, err := parseDBExtraParams(test.dbExtraParams)
			if test.expectError {
				assert.Error(t, err)
				assert.Nil(t, extraParams)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expectedParams, extraParams)
			}
		})
	}
}
