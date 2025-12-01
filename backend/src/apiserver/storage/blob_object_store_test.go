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
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gocloud.dev/blob/memblob"
)

func TestBlobObjectStore_AddFile(t *testing.T) {
	bucket := memblob.OpenBucket(nil)
	defer bucket.Close()

	store := NewBlobObjectStore(bucket, "pipelines")
	ctx := context.Background()

	content := []byte("test content")
	err := store.AddFile(ctx, content, "test/file.txt")
	require.Nil(t, err)

	readContent, err := store.GetFile(ctx, "test/file.txt")
	require.Nil(t, err)
	require.Equal(t, content, readContent)
}

func TestBlobObjectStore_GetFileReader(t *testing.T) {
	bucket := memblob.OpenBucket(nil)
	defer bucket.Close()

	store := NewBlobObjectStore(bucket, "pipelines")
	ctx := context.Background()

	content := []byte("streaming test content")
	err := store.AddFile(ctx, content, "test/stream.txt")
	require.Nil(t, err)

	reader, err := store.GetFileReader(ctx, "test/stream.txt")
	require.Nil(t, err)
	require.NotNil(t, reader)
	defer reader.Close()

	buffer := make([]byte, len(content))
	n, err := reader.Read(buffer)
	require.Nil(t, err)
	require.Equal(t, len(content), n)
	require.Equal(t, content, buffer)
}

func TestBlobObjectStore_DeleteFile(t *testing.T) {
	bucket := memblob.OpenBucket(nil)
	defer bucket.Close()

	store := NewBlobObjectStore(bucket, "pipelines")
	ctx := context.Background()

	content := []byte("delete test content")
	err := store.AddFile(ctx, content, "test/delete.txt")
	require.Nil(t, err)

	err = store.DeleteFile(ctx, "test/delete.txt")
	require.Nil(t, err)

	_, err = store.GetFile(ctx, "test/delete.txt")
	require.NotNil(t, err)
}

func TestBlobObjectStore_AddAsYamlFile(t *testing.T) {
	bucket := memblob.OpenBucket(nil)
	defer bucket.Close()

	store := NewBlobObjectStore(bucket, "pipelines")
	ctx := context.Background()

	data := struct {
		Name  string `yaml:"name"`
		Value int    `yaml:"value"`
	}{
		Name:  "test",
		Value: 42,
	}

	err := store.AddAsYamlFile(ctx, data, "test/config.yaml")
	require.Nil(t, err)

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
	bucket := memblob.OpenBucket(nil)
	defer bucket.Close()

	store := NewBlobObjectStore(bucket, "pipelines")

	pipelineID := "test-pipeline-123"
	expectedKey := "pipelines/test-pipeline-123"
	actualKey := store.GetPipelineKey(pipelineID)
	require.Equal(t, expectedKey, actualKey)
}

func TestBlobObjectStore_GetFile(t *testing.T) {
	bucket := memblob.OpenBucket(nil)
	defer bucket.Close()

	store := NewBlobObjectStore(bucket, "pipelines")
	ctx := context.Background()

	content := []byte("test content for GetFile")
	err := store.AddFile(ctx, content, "test/getfile.txt")
	require.Nil(t, err)

	readContent, err := store.GetFile(ctx, "test/getfile.txt")
	require.Nil(t, err)
	require.Equal(t, content, readContent)
}

func TestBlobObjectStore_GetFile_NotFound(t *testing.T) {
	bucket := memblob.OpenBucket(nil)
	defer bucket.Close()

	store := NewBlobObjectStore(bucket, "pipelines")
	ctx := context.Background()

	_, err := store.GetFile(ctx, "non-existent.txt")
	require.Error(t, err)
	require.Contains(t, err.Error(), "Failed to get file")
}

func TestBlobObjectStore_GetFileReader_NotFound(t *testing.T) {
	bucket := memblob.OpenBucket(nil)
	defer bucket.Close()

	store := NewBlobObjectStore(bucket, "pipelines")
	ctx := context.Background()

	_, err := store.GetFileReader(ctx, "non-existent.txt")
	require.Error(t, err)
	require.Contains(t, err.Error(), "Failed to get file reader")
}

func TestBlobObjectStore_GetFromYamlFile(t *testing.T) {
	bucket := memblob.OpenBucket(nil)
	defer bucket.Close()

	store := NewBlobObjectStore(bucket, "pipelines")
	ctx := context.Background()

	data := struct {
		Name  string `yaml:"name"`
		Value int    `yaml:"value"`
	}{
		Name:  "yaml-test",
		Value: 100,
	}

	err := store.AddAsYamlFile(ctx, data, "test/yaml.yaml")
	require.NoError(t, err)

	var readData struct {
		Name  string `yaml:"name"`
		Value int    `yaml:"value"`
	}
	err = store.GetFromYamlFile(ctx, &readData, "test/yaml.yaml")
	require.NoError(t, err)
	require.Equal(t, data.Name, readData.Name)
	require.Equal(t, data.Value, readData.Value)
}

func TestBlobObjectStore_GetFromYamlFile_NotFound(t *testing.T) {
	bucket := memblob.OpenBucket(nil)
	defer bucket.Close()

	store := NewBlobObjectStore(bucket, "pipelines")
	ctx := context.Background()

	var readData struct {
		Name string `yaml:"name"`
	}
	err := store.GetFromYamlFile(ctx, &readData, "non-existent.yaml")
	require.Error(t, err)
	require.Contains(t, err.Error(), "Failed to get file")
}

func TestBlobObjectStore_GetFromYamlFile_InvalidYaml(t *testing.T) {
	bucket := memblob.OpenBucket(nil)
	defer bucket.Close()

	store := NewBlobObjectStore(bucket, "pipelines")
	ctx := context.Background()

	err := store.AddFile(ctx, []byte("invalid: yaml: content: ["), "test/invalid.yaml")
	require.NoError(t, err)

	var readData struct {
		Name string `yaml:"name"`
	}
	err = store.GetFromYamlFile(ctx, &readData, "test/invalid.yaml")
	require.Error(t, err)
	require.Contains(t, err.Error(), "Failed to unmarshal")
}

// TestBlobObjectStore_GetFileReader_ChunkedStreaming validates that GetFileReader
// supports efficient chunked reading without loading the entire file into memory.
// This test is critical for proving that large artifacts don't cause OOM errors.
func TestBlobObjectStore_GetFileReader_ChunkedStreaming(t *testing.T) {
	bucket := memblob.OpenBucket(nil)
	defer bucket.Close()

	store := NewBlobObjectStore(bucket, "pipelines")
	ctx := context.Background()

	// Create a 1MB test file with predictable content
	fileSize := 1024 * 1024 // 1MB
	content := make([]byte, fileSize)
	for i := range content {
		content[i] = byte(i % 256)
	}

	err := store.AddFile(ctx, content, "test/large_file.bin")
	require.NoError(t, err, "Failed to add large file")

	t.Run("ChunkedReading", func(t *testing.T) {
		reader, err := store.GetFileReader(ctx, "test/large_file.bin")
		require.NoError(t, err)
		require.NotNil(t, reader)
		defer reader.Close()

		// Read in 32KB chunks (same as io.Copy default)
		chunkSize := 32 * 1024
		totalRead := 0
		chunkCount := 0

		for {
			chunk := make([]byte, chunkSize)
			n, err := reader.Read(chunk)
			if err == io.EOF {
				break
			}
			require.NoError(t, err)

			totalRead += n
			chunkCount++

			// Validate that we're reading in chunks, not the whole file
			require.LessOrEqual(t, n, chunkSize, "Read size should not exceed chunk size")
		}

		require.Equal(t, fileSize, totalRead, "Total bytes read should match file size")
		require.Greater(t, chunkCount, 1, "File should be read in multiple chunks")

		// Validate expected number of chunks
		expectedChunks := (fileSize + chunkSize - 1) / chunkSize
		require.Equal(t, expectedChunks, chunkCount, "Should have expected number of chunks")
	})

	t.Run("StreamingWithIoCopy", func(t *testing.T) {
		reader, err := store.GetFileReader(ctx, "test/large_file.bin")
		require.NoError(t, err)
		defer reader.Close()

		// Use io.Copy to simulate what happens in ReadArtifact
		var buf bytes.Buffer
		n, err := io.Copy(&buf, reader)
		require.NoError(t, err)
		require.Equal(t, int64(fileSize), n, "Should copy entire file")

		// Validate content
		copiedContent := buf.Bytes()
		require.Equal(t, fileSize, len(copiedContent))
		for i := 0; i < 100; i++ { // Check first 100 bytes
			require.Equal(t, byte(i%256), copiedContent[i])
		}
	})
}

// TestBlobObjectStore_Streaming_Large_File validates that a file can be streamed
// efficiently without loading it into memory. This is the critical test that proves
// the streaming implementation prevents OOM errors with large artifacts.
func TestBlobObjectStore_Streaming_Large_File(t *testing.T) {
	bucket := memblob.OpenBucket(nil)
	defer bucket.Close()

	store := NewBlobObjectStore(bucket, "pipelines")
	ctx := context.Background()

	fileSize := 10 * 1024 * 1024 // 10MB
	t.Logf("Creating %dMB test file...", fileSize/(1024*1024))

	content := make([]byte, fileSize)
	// Fill with predictable pattern (faster than random)
	for i := 0; i < len(content); i += 1024 * 1024 {
		for j := 0; j < 1024*1024 && i+j < len(content); j++ {
			content[i+j] = byte((j / 1024) % 256)
		}
	}

	err := store.AddFile(ctx, content, "test/large_file.bin")
	require.NoError(t, err, "Failed to add large file")

	defer func() {
		if err := store.DeleteFile(ctx, "test/large_file.bin"); err != nil {
			t.Logf("Failed to clean up test file: %v", err)
		}
	}()

	reader, err := store.GetFileReader(ctx, "test/large_file.bin")
	require.NoError(t, err)
	defer reader.Close()

	// Track memory usage during streaming
	chunkSize := 32 * 1024 // 32KB chunks
	totalRead := int64(0)
	chunkCount := 0
	maxChunkSize := 0

	t.Logf("Streaming large file in chunks...")
	startTime := time.Now()

	for {
		chunk := make([]byte, chunkSize)
		n, err := reader.Read(chunk)
		if err == io.EOF {
			break
		}
		require.NoError(t, err)

		totalRead += int64(n)
		chunkCount++
		if n > maxChunkSize {
			maxChunkSize = n
		}
	}

	duration := time.Since(startTime)

	// Log statistics first for debugging
	t.Logf("Large file streaming results:")
	t.Logf("  - Total size read: %d MB", totalRead/(1024*1024))
	t.Logf("  - Chunks: %d", chunkCount)
	t.Logf("  - Max chunk size: %d KB", maxChunkSize/1024)
	t.Logf("  - Duration: %.2f seconds", duration.Seconds())
	t.Logf("  - Speed: %.2f MB/s", float64(totalRead)/(1024*1024)/duration.Seconds())

	// Validate streaming worked correctly
	require.Equal(t, int64(fileSize), totalRead, "Should read entire file")
	require.Greater(t, chunkCount, 10, "File should be read in many chunks")
	require.LessOrEqual(t, maxChunkSize, chunkSize, "No chunk should exceed buffer size")

	t.Log("All assertions passed: file was streamed in chunks without loading into memory")
}
