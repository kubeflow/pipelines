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
	"google.golang.org/protobuf/types/known/structpb"
)

func TestMarshalJsonWithError(t *testing.T) {
	t.Run("success with struct", func(t *testing.T) {
		input := map[string]string{"key": "value"}
		result, err := MarshalJsonWithError(input)
		assert.NoError(t, err)
		assert.JSONEq(t, `{"key":"value"}`, string(result))
	})

	t.Run("error with unmarshalable type", func(t *testing.T) {
		channel := make(chan int)
		result, err := MarshalJsonWithError(channel)
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestUnmarshalJsonWithError(t *testing.T) {
	t.Run("string input", func(t *testing.T) {
		var output interface{}
		err := UnmarshalJsonWithError(`{"name":"test"}`, &output)
		assert.NoError(t, err)
		resultMap, ok := output.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "test", resultMap["name"])
	})

	t.Run("byte slice input", func(t *testing.T) {
		var output interface{}
		err := UnmarshalJsonWithError([]byte(`{"count":5}`), &output)
		assert.NoError(t, err)
		resultMap, ok := output.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, float64(5), resultMap["count"])
	})

	t.Run("byte slice pointer input", func(t *testing.T) {
		var output interface{}
		data := []byte(`{"active":true}`)
		err := UnmarshalJsonWithError(&data, &output)
		assert.NoError(t, err)
		resultMap, ok := output.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, true, resultMap["active"])
	})

	t.Run("unsupported type returns error", func(t *testing.T) {
		var output interface{}
		err := UnmarshalJsonWithError(42, &output)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not implemented")
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		var output interface{}
		err := UnmarshalJsonWithError(`{invalid}`, &output)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Failed to unmarshal")
	})
}

func TestUnmarshalString(t *testing.T) {
	t.Run("valid proto JSON", func(t *testing.T) {
		message := &structpb.Struct{}
		err := UnmarshalString(`{"key":"value"}`, message)
		assert.NoError(t, err)
		assert.Contains(t, message.GetFields(), "key")
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		message := &structpb.Struct{}
		err := UnmarshalString(`{not valid json}`, message)
		assert.Error(t, err)
	})
}
