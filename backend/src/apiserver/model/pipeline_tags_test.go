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

package model

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTagsJSON(t *testing.T) {
	t.Run("nil tags", func(t *testing.T) {
		tags, err := ParseTagsJSON(nil)
		require.NoError(t, err)
		assert.Nil(t, tags)
	})

	t.Run("valid tags", func(t *testing.T) {
		tagsJSON := `{"team":"ml-ops","env":"prod"}`
		tags, err := ParseTagsJSON(&tagsJSON)
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"team": "ml-ops", "env": "prod"}, tags)
	})

	t.Run("invalid json", func(t *testing.T) {
		tagsJSON := "not-valid-json"
		_, err := ParseTagsJSON(&tagsJSON)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse tags JSON")
	})

	t.Run("invalid tag value", func(t *testing.T) {
		tagsJSON := `{"team":"` + strings.Repeat("v", MaxTagValueLength+1) + `"}`
		_, err := ParseTagsJSON(&tagsJSON)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds maximum length")
	})
}
