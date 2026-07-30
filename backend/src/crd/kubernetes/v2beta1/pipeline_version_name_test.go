/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v2beta1

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPipelineVersionName_CompositeValid(t *testing.T) {
	pvName, err := NewPipelineVersionName("my-pipeline", "v1")
	require.NoError(t, err)
	assert.Equal(t, "my-pipeline-v1", pvName.Name())
}

func TestNewPipelineVersionName_BareValid(t *testing.T) {
	pvName, err := NewPipelineVersionName("", "v1")
	require.NoError(t, err)
	assert.Equal(t, "v1", pvName.Name())
}

func TestNewPipelineVersionName_InvalidBareName(t *testing.T) {
	_, err := NewPipelineVersionName("my-pipeline", "UPPERCASE")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid pipeline version name")

	_, err = NewPipelineVersionName("my-pipeline", "-starts-with-hyphen")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid pipeline version name")
}

func TestNewPipelineVersionName_CompositeTooLong(t *testing.T) {
	versionName := strings.Repeat("v", 253)
	_, err := NewPipelineVersionName("p", versionName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Kubernetes 253-character naming limit")
}
