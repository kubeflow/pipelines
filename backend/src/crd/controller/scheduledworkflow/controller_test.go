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
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

func TestCrdPluginsInputToProto(t *testing.T) {
	t.Run("valid single plugin", func(t *testing.T) {
		input := map[string]apiextensionsv1.JSON{
			"mlflow": {Raw: []byte(`{"experiment_name":"my-exp"}`)},
		}
		result, err := crdPluginsInputToProto(input)
		require.NoError(t, err)
		require.Len(t, result, 1)
		require.Contains(t, result, "mlflow")
		assert.Equal(t, "my-exp", result["mlflow"].Fields["experiment_name"].GetStringValue())
	})

	t.Run("valid multiple plugins", func(t *testing.T) {
		input := map[string]apiextensionsv1.JSON{
			"mlflow": {Raw: []byte(`{"experiment_name":"exp-1"}`)},
			"other":  {Raw: []byte(`{"key":"val"}`)},
		}
		result, err := crdPluginsInputToProto(input)
		require.NoError(t, err)
		require.Len(t, result, 2)
		assert.Contains(t, result, "mlflow")
		assert.Contains(t, result, "other")
	})

	t.Run("empty map", func(t *testing.T) {
		result, err := crdPluginsInputToProto(map[string]apiextensionsv1.JSON{})
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("invalid inner value", func(t *testing.T) {
		input := map[string]apiextensionsv1.JSON{
			"mlflow": {Raw: []byte(`"not-an-object"`)},
		}
		_, err := crdPluginsInputToProto(input)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid plugins_input entry")
	})
}
