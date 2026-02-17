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
	"context"
	"io"
)

// ObjectStore is the interface for object store operations.
type ObjectStore interface {
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
