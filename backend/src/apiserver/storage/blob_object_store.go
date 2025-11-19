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

// Package storage provides blob storage implementation using gocloud.dev/blob for provider-agnostic object storage.
package storage

import (
	"bytes"
	"context"
	"io"
	"path"

	"github.com/kubeflow/pipelines/backend/src/common/util"
	"gocloud.dev/blob"
	"sigs.k8s.io/yaml"
)

// BlobObjectStore implements ObjectStore using gocloud.dev/blob
type BlobObjectStore struct {
	bucket     *blob.Bucket
	baseFolder string
}

var _ ObjectStore = &BlobObjectStore{}

// GetPipelineKey adds the configured base folder to pipeline id.
func (b *BlobObjectStore) GetPipelineKey(pipelineID string) string {
	return path.Join(b.baseFolder, pipelineID)
}

func (b *BlobObjectStore) AddFile(ctx context.Context, file []byte, filePath string) error {
	writer, err := b.bucket.NewWriter(ctx, filePath, &blob.WriterOptions{
		ContentType: "application/octet-stream",
	})
	if err != nil {
		return util.NewInternalServerError(err, "Failed to create writer for file %v", filePath)
	}
	defer writer.Close()

	_, err = writer.Write(file)
	if err != nil {
		return util.NewInternalServerError(err, "Failed to write file %v", filePath)
	}

	return nil
}

func (b *BlobObjectStore) DeleteFile(ctx context.Context, filePath string) error {
	err := b.bucket.Delete(ctx, filePath)
	if err != nil {
		return util.NewInternalServerError(err, "Failed to delete file %v", filePath)
	}
	return nil
}

func (b *BlobObjectStore) GetFile(ctx context.Context, filePath string) ([]byte, error) {
	reader, err := b.bucket.NewReader(ctx, filePath, nil)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to get file %v", filePath)
	}
	defer reader.Close()

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(reader)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to read file %v", filePath)
	}

	return buf.Bytes(), nil
}

// GetFileReader returns a streaming reader for safe access to large files.
// This method streams directly from blob storage without buffering.
func (b *BlobObjectStore) GetFileReader(ctx context.Context, filePath string) (io.ReadCloser, error) {
	if b.bucket == nil {
		return nil, util.NewInternalServerError(nil, "Bucket is not configured")
	}

	reader, err := b.bucket.NewReader(ctx, filePath, nil)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to get file reader for %v", filePath)
	}

	return reader, nil
}

func (b *BlobObjectStore) AddAsYamlFile(ctx context.Context, o interface{}, filePath string) error {
	yamlBytes, err := yaml.Marshal(o)
	if err != nil {
		return util.NewInternalServerError(err, "Failed to marshal file %v: %v", filePath, err.Error())
	}
	err = b.AddFile(ctx, yamlBytes, filePath)
	if err != nil {
		return util.Wrap(err, "Failed to add a yaml file")
	}
	return nil
}

func (b *BlobObjectStore) GetFromYamlFile(ctx context.Context, o interface{}, filePath string) error {
	yamlBytes, err := b.GetFile(ctx, filePath)
	if err != nil {
		return util.Wrap(err, "Failed to read from a yaml file")
	}
	err = yaml.Unmarshal(yamlBytes, o)
	if err != nil {
		return util.NewInternalServerError(err, "Failed to unmarshal file %v: %v", filePath, err.Error())
	}
	return nil
}

func NewBlobObjectStore(bucket *blob.Bucket, baseFolder string) *BlobObjectStore {
	return &BlobObjectStore{bucket: bucket, baseFolder: baseFolder}
}
