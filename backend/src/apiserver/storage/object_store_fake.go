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
	"github.com/kubeflow/pipelines/backend/src/v2/objectstore"
	"k8s.io/api/core/v1"
	"time"
)

type fakeMinioObjectStore struct {
	minioObjectStore *MinioObjectStore
}

func (m *fakeMinioObjectStore) GetPipelineKey(pipelineID string) string {
	return m.minioObjectStore.GetPipelineKey(pipelineID)
}

func (m *fakeMinioObjectStore) AddFile(file []byte, filePath string) error {
	return m.minioObjectStore.AddFile(file, filePath)
}

func (m *fakeMinioObjectStore) DeleteFile(filePath string) error {
	return m.minioObjectStore.DeleteFile(filePath)
}

func (m *fakeMinioObjectStore) GetFile(filePath string) ([]byte, error) {
	return m.minioObjectStore.GetFile(filePath)
}

func (m *fakeMinioObjectStore) AddAsYamlFile(o interface{}, filePath string) error {
	return m.minioObjectStore.AddAsYamlFile(o, filePath)
}

func (m *fakeMinioObjectStore) GetFromYamlFile(o interface{}, filePath string) error {
	return m.minioObjectStore.GetFromYamlFile(o, filePath)
}

func (m *fakeMinioObjectStore) GetSignedUrl(*objectstore.Config, *v1.Secret, time.Duration, string) (string, error) {
	return "dummy-signed-url", nil
}

func (m *fakeMinioObjectStore) GetObjectSize(*objectstore.Config, *v1.Secret, string) (int64, error) {
	return 123, nil
}

// Return the object store with faked minio client.
func NewFakeObjectStore() ObjectStoreInterface {
	newMinioObjectStore := NewMinioObjectStore(NewFakeMinioClient(), "", "pipelines", false)
	return &fakeMinioObjectStore{newMinioObjectStore}
}
