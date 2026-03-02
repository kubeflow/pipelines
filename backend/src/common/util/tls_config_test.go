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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetTLSConfigEmptyPath(t *testing.T) {
	config, err := GetTLSConfig("")
	assert.Nil(t, config)
	assert.NoError(t, err)
}

func TestGetTLSConfigNonexistentPath(t *testing.T) {
	nonexistentPath := filepath.Join(t.TempDir(), "nonexistent-cert.pem")
	config, err := GetTLSConfig(nonexistentPath)
	assert.Nil(t, config)
	assert.Error(t, err)
}

func TestGetTLSConfigInvalidPEM(t *testing.T) {
	invalidPEMFile := filepath.Join(t.TempDir(), "invalid-cert.pem")
	err := os.WriteFile(invalidPEMFile, []byte("this is not valid PEM content"), 0644)
	assert.NoError(t, err)

	config, err := GetTLSConfig(invalidPEMFile)
	assert.Nil(t, config)
	// Note: the source returns nil error here even though PEM parsing failed,
	// because AppendCertsFromPEM returns false but the prior ReadFile err is nil.
	assert.Nil(t, err)
}
