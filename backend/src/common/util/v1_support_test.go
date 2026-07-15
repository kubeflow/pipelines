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

package util

import (
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestIsV1PipelinesBlocked(t *testing.T) {
	tests := []struct {
		name              string
		blockV1           string
		allowedNamespaces string
		namespace         string
		expected          bool
	}{
		{
			name:      "Blocking disabled - not blocked",
			blockV1:   "false",
			namespace: "ns1",
			expected:  false,
		},
		{
			name:      "Blocking not set - not blocked",
			namespace: "ns1",
			expected:  false,
		},
		{
			name:      "Blocking enabled, no allowed namespaces - blocked",
			blockV1:   "true",
			namespace: "ns1",
			expected:  true,
		},
		{
			name:              "Blocking enabled, namespace allowed - not blocked",
			blockV1:           "true",
			allowedNamespaces: "ns1",
			namespace:         "ns1",
			expected:          false,
		},
		{
			name:              "Blocking enabled, namespace not in allowed list - blocked",
			blockV1:           "true",
			allowedNamespaces: "ns2,ns3",
			namespace:         "ns1",
			expected:          true,
		},
		{
			name:              "Blocking enabled, namespace in allowed list - not blocked",
			blockV1:           "true",
			allowedNamespaces: "ns1,ns2,ns3",
			namespace:         "ns2",
			expected:          false,
		},
		{
			name:              "Blocking enabled, case insensitive namespace match - not blocked",
			blockV1:           "true",
			allowedNamespaces: "NS1",
			namespace:         "ns1",
			expected:          false,
		},
		{
			name:              "Blocking enabled, namespace with spaces in allowed list - not blocked",
			blockV1:           "true",
			allowedNamespaces: " ns1 , ns2 ",
			namespace:         "ns1",
			expected:          false,
		},
		{
			name:              "Blocking enabled, whitespace around namespace input - not blocked",
			blockV1:           "true",
			allowedNamespaces: "ns1,ns2",
			namespace:         "  ns1  ",
			expected:          false,
		},
		{
			name:              "Blocking enabled, case insensitive allowed list - not blocked",
			blockV1:           "true",
			allowedNamespaces: "NS1,NS2",
			namespace:         "ns1",
			expected:          false,
		},
		{
			name:              "Blocking enabled, whitespace and case insensitive - not blocked",
			blockV1:           "true",
			allowedNamespaces: "  ns1  ,  ns2  ",
			namespace:         "  NS1  ",
			expected:          false,
		},
		{
			name:              "Blocking enabled, empty namespace with non-empty allowed - blocked",
			blockV1:           "true",
			allowedNamespaces: "ns1,ns2",
			namespace:         "",
			expected:          true,
		},
		{
			name:      "Blocking enabled, empty namespace with empty allowed - blocked",
			blockV1:   "true",
			namespace: "",
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.blockV1 != "" {
				viper.Set(BlockV1Pipelines, tt.blockV1)
			}
			viper.Set(v1AllowedNamespaces, tt.allowedNamespaces)
			defer func() {
				viper.Set(BlockV1Pipelines, nil)
				viper.Set(v1AllowedNamespaces, nil)
			}()

			result := IsV1PipelinesBlocked(tt.namespace)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsV1PipelinesBlocked_MalformedValueFatals(t *testing.T) {
	var fatalCalled bool
	log.StandardLogger().ExitFunc = func(int) { fatalCalled = true }
	defer func() { log.StandardLogger().ExitFunc = nil }()

	viper.Set(BlockV1Pipelines, "notabool")
	defer viper.Set(BlockV1Pipelines, nil)

	IsV1PipelinesBlocked("ns1")
	assert.True(t, fatalCalled, "expected Fatalf to be called for malformed BLOCK_V1_PIPELINES")
}
