// Copyright 2025 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestScopePath(t *testing.T) {
	// Load pipeline spec
	pipelineSpec, _, err := LoadPipelineAndPlatformSpec("../../v2/driver/test_data/loop_collected_raw_Iterator.yaml")
	require.NoError(t, err)
	require.NotNil(t, pipelineSpec)

	// Convert PipelineSpec to Struct
	b, err := protojson.Marshal(pipelineSpec)
	require.NoError(t, err)
	var pipelineSpecStruct structpb.Struct
	err = protojson.Unmarshal(b, &pipelineSpecStruct)
	require.NoError(t, err)

	scopePath, err := NewScopePathFromStruct(&pipelineSpecStruct)
	require.NoError(t, err)
	require.NotNil(t, scopePath)

	require.Empty(t, scopePath.StringPath())

	err = scopePath.Push("not-root")
	require.Error(t, err)

	require.Equal(t, 0, scopePath.GetSize())

	err = scopePath.Push("root")
	require.NoError(t, err)
	require.NotNil(t, scopePath)
	require.Equal(t, 1, scopePath.GetSize())

	head := scopePath.GetRoot()
	last := scopePath.GetLast()
	require.Equal(t, head, last)
	require.NotNil(t, head)
	require.NotNil(t, head.GetComponentSpec())
	require.Nil(t, head.GetTaskSpec())
	require.Equal(t, 2, len(head.componentSpec.GetDag().GetTasks()))
	require.NotNil(t, head.componentSpec.GetDag().GetTasks()["analyze-artifact-list"])

	err = scopePath.Push("secondary-pipeline")
	last = scopePath.GetLast()
	require.NotEqual(t, head, last)
	require.NoError(t, err)
	require.NotNil(t, last.GetComponentSpec())
	require.NotNil(t, last.GetTaskSpec())

	err = scopePath.Push("does-not-exist")
	require.Error(t, err)

	err = scopePath.Push("for-loop-2")
	require.NoError(t, err)
	last = scopePath.GetLast()
	require.Equal(t, last.GetTaskSpec().GetTaskInfo().GetName(), "for-loop-2")
	require.Len(t, last.GetComponentSpec().GetDag().GetTasks(), 2)

	require.Equal(t, []string{"root", "secondary-pipeline", "for-loop-2"}, scopePath.StringPath())
	require.Equal(t, 3, scopePath.GetSize())

	spe, ok := scopePath.Pop()
	require.True(t, ok)
	require.Equal(t, spe.GetTaskSpec().GetTaskInfo().GetName(), "for-loop-2")

	spe, ok = scopePath.Pop()
	require.True(t, ok)
	require.Equal(t, "secondary-pipeline", spe.GetTaskSpec().GetTaskInfo().GetName())

	require.Equal(t, []string{"root"}, scopePath.StringPath())

	// Back to the head
	spe, ok = scopePath.Pop()
	require.True(t, ok)
	require.NotNil(t, head)
	require.NotNil(t, head.GetComponentSpec())
	require.Nil(t, head.GetTaskSpec())

	spe, ok = scopePath.Pop()
	require.False(t, ok)
	require.Empty(t, spe)

	require.Equal(t, 0, scopePath.GetSize())

	require.Empty(t, scopePath.StringPath())
}

func TestBuildFromStringPath(t *testing.T) {
	// Load pipeline spec
	pipelineSpec, _, err := LoadPipelineAndPlatformSpec("../../v2/driver/test_data/loop_collected_raw_Iterator.yaml")
	require.NoError(t, err)

	// Convert PipelineSpec to Struct
	b, err := protojson.Marshal(pipelineSpec)
	require.NoError(t, err)
	var st structpb.Struct
	err = protojson.Unmarshal(b, &st)
	require.NoError(t, err)

	// Test successful path construction using dot notation
	dotNotationPath := "root.secondary-pipeline"
	scopePath, err := ScopePathFromStringPathWithNewTask(&st, dotNotationPath, "for-loop-2")
	require.NoError(t, err)
	require.Equal(t, []string{"root", "secondary-pipeline", "for-loop-2"}, scopePath.StringPath())
	require.Equal(t, "root.secondary-pipeline.for-loop-2", scopePath.DotNotation())

	// Test invalid path using dot notation
	invalidDotPath := "root.non-existent-task"
	_, err = ScopePathFromStringPathWithNewTask(&st, invalidDotPath, "")
	require.Error(t, err)

	// Test building from dot notation directly
	scopePath2, err := ScopePathFromDotNotation(&st, "root.secondary-pipeline.for-loop-2")
	require.NoError(t, err)
	require.Equal(t, "root.secondary-pipeline.for-loop-2", scopePath2.DotNotation())

	// Test empty path
	scopePath3, err := ScopePathFromDotNotation(&st, "")
	require.NoError(t, err)
	require.Equal(t, "", scopePath3.DotNotation())
	require.Equal(t, 0, scopePath3.GetSize())
}

func TestDotNotationConversion(t *testing.T) {
	// Test conversion from array to dot notation
	tests := []struct {
		name     string
		input    []string
		expected string
	}{
		{
			name:     "simple path",
			input:    []string{"root", "pipeline", "task"},
			expected: "root.pipeline.task",
		},
		{
			name:     "single element",
			input:    []string{"root"},
			expected: "root",
		},
		{
			name:     "empty array",
			input:    []string{},
			expected: "",
		},
		{
			name:     "path with hyphens",
			input:    []string{"root", "primary-pipeline", "secondary-pipeline", "task"},
			expected: "root.primary-pipeline.secondary-pipeline.task",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StringPathToDotNotation(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}

	// Test conversion from dot notation to array
	reverseTests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "simple path",
			input:    "root.pipeline.task",
			expected: []string{"root", "pipeline", "task"},
		},
		{
			name:     "single element",
			input:    "root",
			expected: []string{"root"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "path with hyphens",
			input:    "root.primary-pipeline.secondary-pipeline.task",
			expected: []string{"root", "primary-pipeline", "secondary-pipeline", "task"},
		},
	}

	for _, tt := range reverseTests {
		t.Run(tt.name, func(t *testing.T) {
			result := DotNotationToStringPath(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}

	// Test round-trip conversion
	t.Run("round-trip conversion", func(t *testing.T) {
		original := []string{"root", "primary-pipeline", "secondary-pipeline", "task"}
		dotNotation := StringPathToDotNotation(original)
		backToArray := DotNotationToStringPath(dotNotation)
		require.Equal(t, original, backToArray)
	})
}
