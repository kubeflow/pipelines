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
	"io"
	"testing"

	"github.com/googleprivate/ml/backend/src/common/util"
	minio "github.com/minio/minio-go"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
)

type Foo struct{ ID int }

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

func (c *FakeBadMinioClient) DeleteObject(bucketName, objectName string) error {
	return errors.New("some error")
}

func TestAddFile(t *testing.T) {
	minioClient := NewFakeMinioClient()
	manager := &MinioObjectStore{minioClient: minioClient}
	error := manager.AddFile([]byte("abc"), CreatePipelinePath("1"))
	assert.Nil(t, error)
	assert.Equal(t, 1, minioClient.GetObjectCount())
}

func TestAddFileError(t *testing.T) {
	manager := &MinioObjectStore{minioClient: &FakeBadMinioClient{}}
	error := manager.AddFile([]byte("abc"), CreatePipelinePath("1"))
	assert.Equal(t, codes.Internal, error.(*util.UserError).ExternalStatusCode())
}

func TestGetFile(t *testing.T) {
	manager := &MinioObjectStore{minioClient: NewFakeMinioClient()}
	manager.AddFile([]byte("abc"), CreatePipelinePath("1"))
	file, error := manager.GetFile(CreatePipelinePath("1"))
	assert.Nil(t, error)
	assert.Equal(t, file, []byte("abc"))
}

func TestGetFileError(t *testing.T) {
	manager := &MinioObjectStore{minioClient: &FakeBadMinioClient{}}
	_, error := manager.GetFile(CreatePipelinePath("1"))
	assert.Equal(t, codes.Internal, error.(*util.UserError).ExternalStatusCode())
}

func TestDeleteFile(t *testing.T) {
	minioClient := NewFakeMinioClient()
	manager := &MinioObjectStore{minioClient: minioClient}
	manager.AddFile([]byte("abc"), CreatePipelinePath("1"))
	error := manager.DeleteFile(CreatePipelinePath("1"))
	assert.Nil(t, error)
	assert.Equal(t, 0, minioClient.GetObjectCount())
}

func TestDeleteFileError(t *testing.T) {
	manager := &MinioObjectStore{minioClient: &FakeBadMinioClient{}}
	error := manager.DeleteFile(CreatePipelinePath("1"))
	assert.Equal(t, codes.Internal, error.(*util.UserError).ExternalStatusCode())
}

func TestAddAsYamlFile(t *testing.T) {
	minioClient := NewFakeMinioClient()
	manager := &MinioObjectStore{minioClient: minioClient}
	error := manager.AddAsYamlFile(Foo{ID: 1}, CreatePipelinePath("1"))
	assert.Nil(t, error)
	assert.Equal(t, 1, minioClient.GetObjectCount())
}

func TestGetFromYamlFile(t *testing.T) {
	minioClient := NewFakeMinioClient()
	manager := &MinioObjectStore{minioClient: minioClient}
	manager.minioClient.PutObject(
		"", CreatePipelinePath("1"),
		bytes.NewReader([]byte("id: 1")), -1,
		minio.PutObjectOptions{ContentType: "application/octet-stream"})
	expectedFoo := Foo{ID: 1}
	var foo Foo
	error := manager.GetFromYamlFile(&foo, CreatePipelinePath("1"))
	assert.Nil(t, error)
	assert.Equal(t, expectedFoo, foo)
}

func TestGetFromYamlFile_UnmarshalError(t *testing.T) {
	minioClient := NewFakeMinioClient()
	manager := &MinioObjectStore{minioClient: minioClient}
	manager.minioClient.PutObject(
		"", CreatePipelinePath("1"),
		bytes.NewReader([]byte("invalid")), -1,
		minio.PutObjectOptions{ContentType: "application/octet-stream"})
	var foo Foo
	error := manager.GetFromYamlFile(&foo, CreatePipelinePath("1"))
	assert.Equal(t, codes.Internal, error.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, error.Error(), "Failed to unmarshal")
}
