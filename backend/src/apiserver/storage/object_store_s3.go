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
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

// S3ObjectStore uses s3 to manage objects
type S3ObjectStore struct {
	s3Client   S3Client
	bucketName string
}

// AddFile ...
func (m *S3ObjectStore) AddFile(file []byte, filePath string) error {

	_, err := m.s3Client.PutObject(
		m.bucketName,
		filePath,
		bytes.NewReader(file),
		"application/octet-stream",
	)
	if err != nil {
		return util.NewInternalServerError(err, "Failed to store %v", filePath)
	}
	return nil
}

// DeleteFile ...
func (m *S3ObjectStore) DeleteFile(filePath string) error {

	err := m.s3Client.DeleteObject(m.bucketName, filePath)
	if err != nil {
		return util.NewInternalServerError(err, "Failed to delete %v", filePath)
	}
	return nil
}

// GetFile ...
func (m *S3ObjectStore) GetFile(filePath string) ([]byte, error) {

	reader, err := m.s3Client.GetObject(
		m.bucketName,
		filePath,
	)
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to get %v", filePath)
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(reader)
	return buf.Bytes(), nil
}

// AddAsYamlFile ...
func (m *S3ObjectStore) AddAsYamlFile(o interface{}, filePath string) error {

	bytes, err := yaml.Marshal(o)
	if err != nil {
		return util.NewInternalServerError(err, "Failed to marshal %v: %v", filePath, err.Error())
	}
	err = m.AddFile(bytes, filePath)
	if err != nil {
		return util.Wrap(err, "Failed to add a yaml file.")
	}
	return nil
}

// GetFromYamlFile ...
func (m *S3ObjectStore) GetFromYamlFile(o interface{}, filePath string) error {

	bytes, err := m.GetFile(filePath)
	if err != nil {
		return util.Wrap(err, "Failed to read from a yaml file.")
	}
	err = yaml.Unmarshal(bytes, o)
	if err != nil {
		return util.NewInternalServerError(err, "Failed to unmarshal %v: %v", filePath, err.Error())
	}
	return nil
}
