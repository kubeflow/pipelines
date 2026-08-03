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

package matcher

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPipelineSpecNonEmpty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		spec interface{}
		want bool
	}{
		{name: "nil", spec: nil, want: false},
		{name: "empty map", spec: map[string]interface{}{}, want: false},
		{name: "empty interface map", spec: map[interface{}]interface{}{}, want: false},
		{name: "string scalar", spec: "pipelineInfo: {}", want: false},
		{name: "slice", spec: []interface{}{"x"}, want: false},
		{name: "number", spec: 42, want: false},
		{
			name: "valid map",
			spec: map[string]interface{}{
				"pipelineInfo": map[string]interface{}{"name": "hello"},
			},
			want: true,
		},
		{
			name: "valid interface map",
			spec: map[interface{}]interface{}{"pipelineInfo": "x"},
			want: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, pipelineSpecNonEmpty(tc.spec))
		})
	}
}

func TestEmbeddedPipelineSpecContentDiff_RejectsUnrelatedMaps(t *testing.T) {
	t.Parallel()

	actual := map[string]interface{}{
		"pipelineInfo": map[string]interface{}{"name": "expected-pipeline"},
	}
	unrelated := map[string]interface{}{
		"pipelineInfo": map[string]interface{}{"name": "other-pipeline"},
	}

	require.True(t, pipelineSpecNonEmpty(actual))
	require.True(t, pipelineSpecNonEmpty(unrelated))
	assert.NotEmpty(t, cmp.Diff(actual, unrelated),
		"mixed-form matcher must not treat unrelated embedded specs as equal")
	assert.Empty(t, cmp.Diff(actual, map[string]interface{}{
		"pipelineInfo": map[string]interface{}{"name": "expected-pipeline"},
	}))
}
