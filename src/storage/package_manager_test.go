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
	"errors"
	"io"
	"ml/src/util"
	"testing"

	minio "github.com/minio/minio-go"
	"github.com/stretchr/testify/assert"
)

type FakeBadMinioClient struct {
}

func (c *FakeBadMinioClient) PutObject(bucketName, objectName string, reader io.Reader,
	objectSize int64, opts minio.PutObjectOptions) (n int64, err error) {
	return 0, errors.New("some error")
}
func (c *FakeBadMinioClient) GetObject(bucketName, objectName string,
	opts minio.GetObjectOptions) (io.Reader, error) {
	return nil, errors.New("some error")
}

func TestCreatePackageFile(t *testing.T) {
	minioClient := NewFakeMinioClient()
	manager := &MinioPackageManager{minioClient: minioClient}
	error := manager.CreatePackageFile([]byte("abc"), "file1")
	assert.Nil(t, error, "Expected create package successfully.")
	assert.Equal(t, 1, minioClient.GetObjectCount())
}

func TestCreatePackageFileError(t *testing.T) {
	manager := &MinioPackageManager{minioClient: &FakeBadMinioClient{}}
	error := manager.CreatePackageFile([]byte("abc"), "file1")
	assert.IsType(t, new(util.InternalError), error, "Expected new internal error.")
}

func TestGetTemplate(t *testing.T) {
	manager := &MinioPackageManager{minioClient: NewFakeMinioClient()}
	error := manager.CreatePackageFile([]byte("abc"), "file1")
	file, error := manager.GetTemplate("file1")
	assert.Nil(t, error, "Expected get package successfully.")
	assert.Equal(t, file, []byte("abc"))
}

func TestGetTemplateError(t *testing.T) {
	manager := &MinioPackageManager{minioClient: &FakeBadMinioClient{}}
	_, error := manager.GetTemplate("file name")
	assert.IsType(t, new(util.InternalError), error, "Expected new internal error.")
}
