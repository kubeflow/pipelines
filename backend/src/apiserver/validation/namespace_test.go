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

package validation

import (
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestValidateNamespaceRequired(t *testing.T) {
	tests := []struct {
		name                         string
		multiUser                    string
		requireNamespaceForPipelines string
		namespace                    string
		expectError                  bool
		errorContains                string
	}{
		{
			name:                         "single-user mode with empty namespace passes",
			multiUser:                    "false",
			requireNamespaceForPipelines: "false",
			namespace:                    "",
			expectError:                  false,
		},
		{
			name:                         "multi-user mode with namespace not required and empty namespace passes",
			multiUser:                    "true",
			requireNamespaceForPipelines: "false",
			namespace:                    "",
			expectError:                  false,
		},
		{
			name:                         "multi-user mode with namespace required and non-empty namespace passes",
			multiUser:                    "true",
			requireNamespaceForPipelines: "true",
			namespace:                    "team-a",
			expectError:                  false,
		},
		{
			name:                         "multi-user mode with namespace required and empty namespace returns error",
			multiUser:                    "true",
			requireNamespaceForPipelines: "true",
			namespace:                    "",
			expectError:                  true,
			errorContains:                "Namespace is required",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Cleanup(viper.Reset)
			viper.Set(common.MultiUserMode, testCase.multiUser)
			viper.Set(common.RequireNamespaceForPipelines, testCase.requireNamespaceForPipelines)

			err := ValidateNamespaceRequired(testCase.namespace)
			if testCase.expectError {
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), testCase.errorContains)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}
