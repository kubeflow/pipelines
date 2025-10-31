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
