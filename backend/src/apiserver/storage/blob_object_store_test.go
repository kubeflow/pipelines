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

package storage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"gocloud.dev/blob/memblob"
)

func TestBlobObjectStore_AddFile(t *testing.T) {
	// Create a memory-based bucket for testing
	bucket := memblob.OpenBucket(nil)
	defer bucket.Close()

	store := NewBlobObjectStore(bucket, "pipelines")
	ctx := context.Background()

	// Test adding a file
	content := []byte("test content")
	err := store.AddFile(ctx, content, "test/file.txt")
	require.Nil(t, err)

	// Verify the file was added by reading it back
	readContent, err := store.GetFile(ctx, "test/file.txt")
	require.Nil(t, err)
	require.Equal(t, content, readContent)
}

func TestBlobObjectStore_GetFileReader(t *testing.T) {
	// Create a memory-based bucket for testing
	bucket := memblob.OpenBucket(nil)
	defer bucket.Close()

	store := NewBlobObjectStore(bucket, "pipelines")
	ctx := context.Background()

	// Add a test file
	content := []byte("streaming test content")
	err := store.AddFile(ctx, content, "test/stream.txt")
	require.Nil(t, err)

	// Test streaming read
	reader, err := store.GetFileReader(ctx, "test/stream.txt")
	require.Nil(t, err)
	require.NotNil(t, reader)
	defer reader.Close()

	// Read the content via streaming
	buffer := make([]byte, len(content))
	n, err := reader.Read(buffer)
	require.Nil(t, err)
	require.Equal(t, len(content), n)
	require.Equal(t, content, buffer)
}

func TestBlobObjectStore_DeleteFile(t *testing.T) {
	// Create a memory-based bucket for testing
	bucket := memblob.OpenBucket(nil)
	defer bucket.Close()

	store := NewBlobObjectStore(bucket, "pipelines")
	ctx := context.Background()

	// Add a test file
	content := []byte("delete test content")
	err := store.AddFile(ctx, content, "test/delete.txt")
	require.Nil(t, err)

	// Delete the file
	err = store.DeleteFile(ctx, "test/delete.txt")
	require.Nil(t, err)

	// Verify the file is gone
	_, err = store.GetFile(ctx, "test/delete.txt")
	require.NotNil(t, err)
}

func TestBlobObjectStore_AddAsYamlFile(t *testing.T) {
	// Create a memory-based bucket for testing
	bucket := memblob.OpenBucket(nil)
	defer bucket.Close()

	store := NewBlobObjectStore(bucket, "pipelines")
	ctx := context.Background()

	// Test data structure
	data := struct {
		Name  string `yaml:"name"`
		Value int    `yaml:"value"`
	}{
		Name:  "test",
		Value: 42,
	}

	// Add as YAML file
	err := store.AddAsYamlFile(ctx, data, "test/config.yaml")
	require.Nil(t, err)

	// Read back as YAML
	var readData struct {
		Name  string `yaml:"name"`
		Value int    `yaml:"value"`
	}
	err = store.GetFromYamlFile(ctx, &readData, "test/config.yaml")
	require.Nil(t, err)
	require.Equal(t, data.Name, readData.Name)
	require.Equal(t, data.Value, readData.Value)
}

func TestBlobObjectStore_GetPipelineKey(t *testing.T) {
	// Create a memory-based bucket for testing
	bucket := memblob.OpenBucket(nil)
	defer bucket.Close()

	store := NewBlobObjectStore(bucket, "pipelines")

	// Test pipeline key generation
	pipelineID := "test-pipeline-123"
	expectedKey := "pipelines/test-pipeline-123"
	actualKey := store.GetPipelineKey(pipelineID)
	require.Equal(t, expectedKey, actualKey)
}
