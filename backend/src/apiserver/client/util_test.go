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

package client

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPathExists(t *testing.T) {
	t.Run("returns true for existing file", func(t *testing.T) {
		filePath := filepath.Join(t.TempDir(), "testfile")
		if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}

		exists, err := PathExists(filePath)
		if err != nil {
			t.Fatalf("PathExists() unexpected error: %v", err)
		}
		if !exists {
			t.Error("PathExists() = false for existing file, want true")
		}
	})

	t.Run("returns true for existing directory", func(t *testing.T) {
		tempDir := t.TempDir()

		exists, err := PathExists(tempDir)
		if err != nil {
			t.Fatalf("PathExists() unexpected error: %v", err)
		}
		if !exists {
			t.Error("PathExists() = false for existing directory, want true")
		}
	})

	t.Run("returns false for non-existent path", func(t *testing.T) {
		nonExistentPath := filepath.Join(t.TempDir(), "does-not-exist")

		exists, err := PathExists(nonExistentPath)
		if err != nil {
			t.Fatalf("PathExists() unexpected error: %v", err)
		}
		if exists {
			t.Error("PathExists() = true for non-existent path, want false")
		}
	})
}
