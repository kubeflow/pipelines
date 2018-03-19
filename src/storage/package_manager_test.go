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
	"strings"
	"testing"

	"github.com/minio/minio-go"
	"github.com/stretchr/testify/assert"
)

type FakeMinioClient struct {
}

func (c *FakeMinioClient) PutObject(bucketName, objectName string, reader io.Reader,
	objectSize int64, opts minio.PutObjectOptions) (n int64, err error) {
	return 1, nil
}
func (c *FakeMinioClient) GetObject(bucketName, objectName string,
	opts minio.GetObjectOptions) (io.Reader, error) {
	return strings.NewReader("I'm a file"), nil
}

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
	manager := &MinioPackageManager{minioClient: &FakeMinioClient{}}
	error := manager.CreatePackageFile([]byte{}, "file  name")
	assert.Nil(t, error, "Expect create package successfully.")
}

func TestCreatePackageFileError(t *testing.T) {
	manager := &MinioPackageManager{minioClient: &FakeBadMinioClient{}}
	error := manager.CreatePackageFile([]byte{}, "field name")
	assert.IsType(t, new(util.InternalError), error, "Expect new internal error.")
}

func TestGetTemplate(t *testing.T) {
	manager := &MinioPackageManager{minioClient: &FakeMinioClient{}}
	file, error := manager.GetTemplate("file name")
	assert.Nil(t, error, "Expect get package successfully.")
	assert.Equal(t, file, []byte("I'm a file"))
}

func TestGetTemplateError(t *testing.T) {
	manager := &MinioPackageManager{minioClient: &FakeBadMinioClient{}}
	_, error := manager.GetTemplate("file name")
	assert.IsType(t, new(util.InternalError), error, "Expect new internal error.")
}
