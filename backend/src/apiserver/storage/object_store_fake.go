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
	"net/url"
	"time"

	"github.com/kubeflow/pipelines/backend/src/v2/objectstore"
	v1 "k8s.io/api/core/v1"
)

type fakeMinioObjectStore struct {
	minioObjectStore *MinioObjectStore
}

func (m *fakeMinioObjectStore) GetPipelineKey(pipelineID string) string {
	return m.minioObjectStore.GetPipelineKey(pipelineID)
}

func (m *fakeMinioObjectStore) AddFile(ctx context.Context, file []byte, filePath string) error {
	return m.minioObjectStore.AddFile(ctx, file, filePath)
}

func (m *fakeMinioObjectStore) DeleteFile(ctx context.Context, filePath string) error {
	return m.minioObjectStore.DeleteFile(ctx, filePath)
}

func (m *fakeMinioObjectStore) GetFile(ctx context.Context, filePath string) ([]byte, error) {
	return m.minioObjectStore.GetFile(ctx, filePath)
}

func (m *fakeMinioObjectStore) AddAsYamlFile(ctx context.Context, o interface{}, filePath string) error {
	return m.minioObjectStore.AddAsYamlFile(ctx, o, filePath)
}

func (m *fakeMinioObjectStore) GetFromYamlFile(ctx context.Context, o interface{}, filePath string) error {
	return m.minioObjectStore.GetFromYamlFile(ctx, o, filePath)
}

func (m *fakeMinioObjectStore) GetSignedUrl(
	ctx context.Context,
	bucketConfig *objectstore.Config, secret *v1.Secret, expiry time.Duration,
	uri string, queryParams url.Values) (string, error) {

	if queryParams.Get("response-content-disposition") == "inline" {
		return "dummy-render-url", nil
	}
	return "dummy-signed-url", nil
}

func (m *fakeMinioObjectStore) GetObjectSize(context.Context, *objectstore.Config, *v1.Secret, string) (int64, error) {
	return 123, nil
}

// Return the object store with faked minio client.
func NewFakeObjectStore() ObjectStoreInterface {
	newMinioObjectStore := NewMinioObjectStore(NewFakeMinioClient(), "", "pipelines", false)
	return &fakeMinioObjectStore{newMinioObjectStore}
}
