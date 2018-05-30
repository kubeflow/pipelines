// Copyright 2018 Google LLC
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

	"github.com/ghodss/yaml"
	"github.com/googleprivate/ml/backend/src/util"
	minio "github.com/minio/minio-go"
)

const (
	PackageFolder  = "packages"
	PipelineFolder = "pipelines"
)

// Interface for object store.
type ObjectStoreInterface interface {
	AddFile(template []byte, folder string, fileName string) error
	DeleteFile(folder string, fileName string) error
	GetFile(folder string, fileName string) ([]byte, error)
	AddAsYamlFile(o interface{}, folder string, fileName string) error
	GetFromYamlFile(o interface{}, folder string, fileName string) error
}

// Managing package using Minio
type MinioObjectStore struct {
	minioClient MinioClientInterface
	bucketName  string
}

func (m *MinioObjectStore) AddFile(file []byte, folder string, fileName string) error {
	_, err := m.minioClient.PutObject(m.bucketName, buildPath(folder, fileName), bytes.NewReader(file), -1, minio.PutObjectOptions{ContentType: "application/octet-stream"})
	if err != nil {
		return util.NewInternalServerError(err, "Failed to store %v to %v", fileName, folder)
	}
	return nil
}

func (m *MinioObjectStore) DeleteFile(folder string, fileName string) error {
	err := m.minioClient.DeleteObject(m.bucketName, buildPath(folder, fileName))
	if err != nil {
		return util.NewInternalServerError(err, "Failed to delete %v from %", fileName, folder, err.Error())
	}
	return nil
}

func (m *MinioObjectStore) GetFile(folder string, fileName string) ([]byte, error) {
	reader, err := m.minioClient.GetObject(m.bucketName, buildPath(folder, fileName), minio.GetObjectOptions{})
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to get %v from %v", fileName, folder)
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(reader)
	return buf.Bytes(), nil
}

func (m *MinioObjectStore) AddAsYamlFile(o interface{}, folder string, fileName string) error {
	bytes, err := yaml.Marshal(o)
	if err != nil {
		return util.NewInternalServerError(err, "Failed to marshal %v and store to %v: %v", fileName, folder, err.Error())
	}
	err = m.AddFile(bytes, folder, fileName)
	if err != nil {
		return util.Wrap(err, "Failed to add a yaml file.")
	}
	return nil
}

func (m *MinioObjectStore) GetFromYamlFile(o interface{}, folder string, fileName string) error {
	bytes, err := m.GetFile(folder, fileName)
	if err != nil {
		return util.Wrap(err, "Failed to read from a yaml file.")
	}
	err = yaml.Unmarshal(bytes, o)
	if err != nil {
		return util.NewInternalServerError(err, "Failed to unmarshal %v from %v: %v", fileName, folder, err.Error())
	}
	return nil
}

func buildPath(folder, file string) string {
	return folder + "/" + file
}

func NewMinioObjectStore(minioClient MinioClientInterface, bucketName string) *MinioObjectStore {
	return &MinioObjectStore{minioClient: minioClient, bucketName: bucketName}
}
