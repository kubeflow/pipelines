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

package storage

import (
	"bytes"
	"context"
	"io"
	"path"
	"regexp"

	"github.com/kubeflow/pipelines/backend/src/common/util"
	minio "github.com/minio/minio-go/v7"
	"sigs.k8s.io/yaml"
)

const (
	multipartDefaultSize = -1
)

// Interface for object store.
type ObjectStoreInterface interface {
	AddFile(ctx context.Context, template []byte, filePath string) error
	DeleteFile(ctx context.Context, filePath string) error
	GetFile(ctx context.Context, filePath string) ([]byte, error)
	// GetFileReader returns a streaming reader for the file content.
	// Use this method instead of GetFile for streaming access to large files.
	GetFileReader(ctx context.Context, filePath string) (io.ReadCloser, error)
	AddAsYamlFile(ctx context.Context, o interface{}, filePath string) error
	GetFromYamlFile(ctx context.Context, o interface{}, filePath string) error
	GetPipelineKey(pipelineId string) string
}

// Managing pipeline using Minio.
type MinioObjectStore struct {
	minioClient      MinioClientInterface
	bucketName       string
	baseFolder       string
	disableMultipart bool
}

// GetPipelineKey adds the configured base folder to pipeline id.
func (m *MinioObjectStore) GetPipelineKey(pipelineID string) string {
	return path.Join(m.baseFolder, pipelineID)
}

func (m *MinioObjectStore) AddFile(ctx context.Context, file []byte, filePath string) error {
	var parts int64

	if m.disableMultipart {
		parts = int64(len(file))
	} else {
		parts = multipartDefaultSize
	}

	_, err := m.minioClient.PutObject(
		ctx,
		m.bucketName, filePath, bytes.NewReader(file),
		parts, minio.PutObjectOptions{ContentType: "application/octet-stream"})
	if err != nil {
		return util.NewInternalServerError(err, "Failed to store file %v", filePath)
	}
	return nil
}

func (m *MinioObjectStore) DeleteFile(ctx context.Context, filePath string) error {
	err := m.minioClient.DeleteObject(ctx, m.bucketName, filePath)
	if err != nil {
		return util.NewInternalServerError(err, "Failed to delete file %v", filePath)
	}
	return nil
}

func (m *MinioObjectStore) GetFile(ctx context.Context, filePath string) ([]byte, error) {
	reader, err := m.minioClient.GetObject(ctx, m.bucketName, filePath, minio.GetObjectOptions{})
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to get file %v", filePath)
	}
	defer reader.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(reader)

	bytes := buf.Bytes()

	// Remove single part signature if exists
	if m.disableMultipart {
		re := regexp.MustCompile(`\w+;chunk-signature=\w+`)
		bytes = []byte(re.ReplaceAllString(string(bytes), ""))
	}

	return bytes, nil
}

// GetFileReader returns a streaming reader for safe access to large files.
func (m *MinioObjectStore) GetFileReader(ctx context.Context, filePath string) (io.ReadCloser, error) {
	if m.bucketName == "" {
		return nil, util.NewInternalServerError(nil, "Bucket name cannot be empty")
	}

	if m.minioClient == nil {
		return nil, util.NewInternalServerError(nil, "MinioClient is not configured")
	}

	reader, err := m.minioClient.GetObject(ctx, m.bucketName, filePath, minio.GetObjectOptions{})
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to get file reader for %v", filePath)
	}

	// For minio objects, we need to wrap the reader to handle multipart signatures
	if m.disableMultipart {
		// If multipart is disabled, we need to filter out signatures while streaming
		// This is more complex for streaming, so for now we'll use a wrapper
		return &filteredReader{reader: reader, re: regexp.MustCompile(`\w+;chunk-signature=\w+`)}, nil
	}

	return reader, nil
}

// filteredReader wraps a reader to filter out chunk signatures on the fly
type filteredReader struct {
	reader io.ReadCloser
	re     *regexp.Regexp
	buffer []byte
}

func (f *filteredReader) Read(p []byte) (n int, err error) {
	for len(f.buffer) == 0 {
		// Read a chunk from underlying reader
		chunk := make([]byte, 8192)
		n, err := f.reader.Read(chunk)
		if err != nil && err != io.EOF {
			return 0, err
		}
		if n > 0 {
			// Filter the chunk
			filtered := f.re.ReplaceAllString(string(chunk[:n]), "")
			f.buffer = []byte(filtered)
		}
		// If we got EOF and no more data to process, return EOF
		if err == io.EOF {
			if len(f.buffer) == 0 {
				return 0, io.EOF
			}
			// We have data in buffer, will return it and EOF on next call
			break
		}
	}

	// Copy from buffer to output
	copyLen := len(p)
	if len(f.buffer) < copyLen {
		copyLen = len(f.buffer)
	}
	copy(p, f.buffer[:copyLen])
	f.buffer = f.buffer[copyLen:]

	return copyLen, nil
}

func (f *filteredReader) Close() error {
	if f.reader != nil {
		return f.reader.Close()
	}
	return nil
}

func (m *MinioObjectStore) AddAsYamlFile(ctx context.Context, o interface{}, filePath string) error {
	bytes, err := yaml.Marshal(o)
	if err != nil {
		return util.NewInternalServerError(err, "Failed to marshal file %v: %v", filePath, err.Error())
	}
	err = m.AddFile(ctx, bytes, filePath)
	if err != nil {
		return util.Wrap(err, "Failed to add a yaml file")
	}
	return nil
}

func (m *MinioObjectStore) GetFromYamlFile(ctx context.Context, o interface{}, filePath string) error {
	bytes, err := m.GetFile(ctx, filePath)
	if err != nil {
		return util.Wrap(err, "Failed to read from a yaml file")
	}
	err = yaml.Unmarshal(bytes, o)
	if err != nil {
		return util.NewInternalServerError(err, "Failed to unmarshal file %v: %v", filePath, err.Error())
	}
	return nil
}

func NewMinioObjectStore(minioClient MinioClientInterface, bucketName string, baseFolder string, disableMultipart bool) *MinioObjectStore {
	return &MinioObjectStore{minioClient: minioClient, bucketName: bucketName, baseFolder: baseFolder, disableMultipart: disableMultipart}
}
