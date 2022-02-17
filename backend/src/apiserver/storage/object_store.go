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
	"path"
	"regexp"

	"github.com/ghodss/yaml"
	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/minio/minio-go"
)

const (
	multipartDefaultSize = -1
)

// Interface for object store.
type ObjectStoreInterface interface {
	AddFile(template []byte, filePath string, bucketName string) error
	DeleteFile(filePath string, bucketName string) error
	GetFile(filePath string, bucketName string) ([]byte, error)
	AddAsYamlFile(o interface{}, filePath string, bucketName string) error
	GetFromYamlFile(o interface{}, filePath string) error
	GetPipelineKey(pipelineId string) string
}

// Managing pipeline using Minio
type MinioObjectStore struct {
	minioClient       MinioClientInterface
	bucketName        string
	baseFolder        string
	disableMultipart  bool
	defaultBucketName string
	region            string
}

// GetPipelineKey adds the configured base folder to pipeline id.
func (m *MinioObjectStore) GetPipelineKey(pipelineID string) string {
	return path.Join(m.baseFolder, pipelineID)
}

func (m *MinioObjectStore) AddFile(file []byte, filePath string, bucketName string) error {

	var parts int64

	if m.disableMultipart {
		parts = int64(len(file))
	} else {
		parts = multipartDefaultSize
	}

	_, err := m.minioClient.PutObject(
		m.getBucketNameOrDefault(bucketName), filePath, bytes.NewReader(file),
		parts, minio.PutObjectOptions{ContentType: "application/octet-stream"})
	if err != nil {
		return util.NewInternalServerError(err, "Failed to store %v", filePath)
	}
	return nil
}

func (m *MinioObjectStore) DeleteFile(filePath string, bucketName string) error {
	err := m.minioClient.DeleteObject(m.getBucketNameOrDefault(bucketName), filePath)
	if err != nil {
		return util.NewInternalServerError(err, "Failed to delete %v", filePath)
	}
	return nil
}

func (m *MinioObjectStore) GetFile(filePath string, bucketName string) ([]byte, error) {
	glog.Infof("GetFile path %s bucketName %s", filePath, bucketName)
	reader, err := m.minioClient.GetObject(m.getBucketNameOrDefault(bucketName), filePath, minio.GetObjectOptions{})
	if err != nil {
		return nil, util.NewInternalServerError(err, "Failed to get %v", filePath)
	}

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

func (m *MinioObjectStore) AddAsYamlFile(o interface{}, filePath string, bucketName string) error {
	bytes, err := yaml.Marshal(o)
	if err != nil {
		return util.NewInternalServerError(err, "Failed to marshal %v: %v", filePath, err.Error())
	}
	err = m.AddFile(bytes, filePath, bucketName)
	if err != nil {
		return util.Wrap(err, "Failed to add a yaml file.")
	}
	return nil
}

func (m *MinioObjectStore) GetFromYamlFile(o interface{}, filePath string) error {
	bytes, err := m.GetFile(filePath, "")
	if err != nil {
		return util.Wrap(err, "Failed to read from a yaml file.")
	}
	err = yaml.Unmarshal(bytes, o)
	if err != nil {
		return util.NewInternalServerError(err, "Failed to unmarshal %v: %v", filePath, err.Error())
	}
	return nil
}

func buildPath(folder, file string) string {
	return folder + "/" + file
}

// getBucketNameOrDefault returns namespaced bucketname if valid or the default.
func (m *MinioObjectStore) getBucketNameOrDefault(bucketName string) string {
	glog.Infof("getBucketNameOrDefault: %s , length: %d", bucketName, len(bucketName))

	if len(bucketName) > 0 {
		return bucketName
	}
	glog.Infof("returning default bucket %s", m.defaultBucketName)
	return m.defaultBucketName
}

//createMinioBucket checks if a bucket already exists and creates it
func (m *MinioObjectStore) createMinioBucket(bucketName string) {
	// Check to see if we already own this bucket.
	exists, err := m.minioClient.BucketExists(bucketName)
	if err != nil {
		glog.Fatalf("Failed to check if Minio bucket exists. Error: %v", err)
	}
	if exists {
		glog.Infof("We already own %s\n", bucketName)
		return
	}
	// Create bucket if it does not exist
	err = m.minioClient.MakeBucket(bucketName, m.region)
	if err != nil {
		glog.Fatalf("Failed to create Minio bucket. Error: %v", err)
	}
	glog.Infof("Successfully created bucket %s\n", bucketName)
}

func NewMinioObjectStore(minioClient MinioClientInterface, region string, defaultBucketName string, baseFolder string, disableMultipart bool) *MinioObjectStore {
	return &MinioObjectStore{minioClient: minioClient, region: region,
		defaultBucketName: defaultBucketName, baseFolder: baseFolder, disableMultipart: disableMultipart}
}
