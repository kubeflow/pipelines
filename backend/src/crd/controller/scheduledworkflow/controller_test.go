// Copyright 2018 The Kubeflow Authors
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
	"github.com/stretchr/testify/require"
)

func TestParsePluginsInputJSON(t *testing.T) {
	t.Run("valid single plugin", func(t *testing.T) {
		raw := `{"mlflow":{"experiment_name":"my-exp"}}`
		result, err := parsePluginsInputJSON(raw)
		require.NoError(t, err)
		require.Len(t, result, 1)
		require.Contains(t, result, "mlflow")
		assert.Equal(t, "my-exp", result["mlflow"].Fields["experiment_name"].GetStringValue())
	})

	t.Run("valid multiple plugins", func(t *testing.T) {
		raw := `{"mlflow":{"experiment_name":"exp-1"},"other":{"key":"val"}}`
		result, err := parsePluginsInputJSON(raw)
		require.NoError(t, err)
		require.Len(t, result, 2)
		assert.Contains(t, result, "mlflow")
		assert.Contains(t, result, "other")
	})

	t.Run("empty JSON object", func(t *testing.T) {
		result, err := parsePluginsInputJSON(`{}`)
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		_, err := parsePluginsInputJSON(`{bad`)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid plugins_input JSON")
	})

	t.Run("invalid inner value", func(t *testing.T) {
		_, err := parsePluginsInputJSON(`{"mlflow":"not-an-object"}`)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid plugins_input entry")
	})
}
