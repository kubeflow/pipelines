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

package util

import (
	"testing"
	"time"

	workflowapi "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestStringPointer(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"non-empty string", "hello"},
		{"empty string", ""},
		{"string with special chars", "hello\nworld"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StringPointer(tt.input)
			assert.NotNil(t, result)
			assert.Equal(t, tt.input, *result)
		})
	}
}

func TestBoolPointer(t *testing.T) {
	tests := []struct {
		name  string
		input bool
	}{
		{"true", true},
		{"false", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BoolPointer(tt.input)
			assert.NotNil(t, result)
			assert.Equal(t, tt.input, *result)
		})
	}
}

func TestTimePointer(t *testing.T) {
	now := time.Now().UTC()
	result := TimePointer(now)
	assert.NotNil(t, result)
	assert.Equal(t, now, *result)
}

func TestInt64Pointer(t *testing.T) {
	tests := []struct {
		name  string
		input int64
	}{
		{"zero", 0},
		{"positive", 42},
		{"negative", -1},
		{"max", int64(1<<63 - 1)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Int64Pointer(tt.input)
			assert.NotNil(t, result)
			assert.Equal(t, tt.input, *result)
		})
	}
}

func TestUInt32Pointer(t *testing.T) {
	tests := []struct {
		name  string
		input uint32
	}{
		{"zero", 0},
		{"positive", 42},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := UInt32Pointer(tt.input)
			assert.NotNil(t, result)
			assert.Equal(t, tt.input, *result)
		})
	}
}

func TestInt32Pointer(t *testing.T) {
	tests := []struct {
		name  string
		input int32
	}{
		{"zero", 0},
		{"positive", 42},
		{"negative", -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Int32Pointer(tt.input)
			assert.NotNil(t, result)
			assert.Equal(t, tt.input, *result)
		})
	}
}

func TestStringNilOrValue(t *testing.T) {
	tests := []struct {
		name     string
		input    *string
		expected string
	}{
		{"nil pointer", nil, "<nil>"},
		{"non-empty string", StringPointer("hello"), "hello"},
		{"empty string", StringPointer(""), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, StringNilOrValue(tt.input))
		})
	}
}

func TestInt64NilOrValue(t *testing.T) {
	tests := []struct {
		name     string
		input    *int64
		expected string
	}{
		{"nil pointer", nil, "<nil>"},
		{"zero value", Int64Pointer(0), "0"},
		{"positive value", Int64Pointer(42), "42"},
		{"negative value", Int64Pointer(-1), "-1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, Int64NilOrValue(tt.input))
		})
	}
}

func TestBoolNilOrValue(t *testing.T) {
	tests := []struct {
		name     string
		input    *bool
		expected string
	}{
		{"nil pointer", nil, "<nil>"},
		{"true", BoolPointer(true), "true"},
		{"false", BoolPointer(false), "false"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, BoolNilOrValue(tt.input))
		})
	}
}

func TestBooleanPointer(t *testing.T) {
	resultTrue := BooleanPointer(true)
	assert.NotNil(t, resultTrue)
	assert.True(t, *resultTrue)

	resultFalse := BooleanPointer(false)
	assert.NotNil(t, resultFalse)
	assert.False(t, *resultFalse)
}

func TestMetav1TimePointer(t *testing.T) {
	now := metav1.Now()
	result := Metav1TimePointer(now)
	assert.NotNil(t, result)
	assert.Equal(t, now, *result)
}

func TestMetaV1TimePointer(t *testing.T) {
	now := metav1.Now()
	result := MetaV1TimePointer(now)
	assert.NotNil(t, result)
	assert.Equal(t, now, *result)
}

func TestToInt64Pointer(t *testing.T) {
	tests := []struct {
		name     string
		input    *metav1.Time
		expected *int64
	}{
		{"nil input", nil, nil},
		{
			"valid time",
			&metav1.Time{Time: time.Unix(1234567890, 0)},
			Int64Pointer(1234567890),
		},
		{
			"epoch time",
			&metav1.Time{Time: time.Unix(0, 0)},
			Int64Pointer(0),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToInt64Pointer(tt.input)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, *tt.expected, *result)
			}
		})
	}
}

func TestToAnyStringPointer(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		result := ToAnyStringPointer(nil)
		assert.Nil(t, result)
	})

	t.Run("non-nil input", func(t *testing.T) {
		input := StringPointer("hello")
		result := ToAnyStringPointer(input)
		assert.NotNil(t, result)
		assert.Equal(t, "hello", result.String())
	})
}

func TestToStringPointer(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		result := ToStringPointer(nil)
		assert.Nil(t, result)
	})

	t.Run("non-nil input", func(t *testing.T) {
		input := workflowapi.AnyStringPtr("world")
		result := ToStringPointer(input)
		assert.NotNil(t, result)
		assert.Equal(t, "world", *result)
	})
}

func TestAnyStringPtr(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{"string value", "hello", "hello"},
		{"integer value", 42, "42"},
		{"boolean value", true, "true"},
		{"float value", 3.14, "3.14"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AnyStringPtr(tt.input)
			assert.NotNil(t, result)
			assert.Equal(t, tt.expected, *result)
		})
	}
}
