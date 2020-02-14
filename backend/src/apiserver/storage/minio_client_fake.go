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

	"github.com/minio/minio-go"
	"github.com/pkg/errors"
)

type FakeMinioClient struct {
	minioClient map[string][]byte
}

func NewFakeMinioClient() *FakeMinioClient {
	return &FakeMinioClient{
		minioClient: make(map[string][]byte),
	}
}

func (c *FakeMinioClient) PutObject(bucketName, objectName string, reader io.Reader,
	objectSize int64, opts minio.PutObjectOptions) (n int64, err error) {
	buf := new(bytes.Buffer)
	buf.ReadFrom(reader)
	c.minioClient[objectName] = buf.Bytes()
	return 1, nil
}

func (c *FakeMinioClient) GetObject(bucketName, objectName string,
	opts minio.GetObjectOptions) (io.Reader, error) {
	if _, ok := c.minioClient[objectName]; !ok {
		return nil, errors.New("object not found")
	}
	return bytes.NewReader(c.minioClient[objectName]), nil
}

func (c *FakeMinioClient) DeleteObject(bucketName, objectName string) error {
	if _, ok := c.minioClient[objectName]; !ok {
		return errors.New("object not found")
	}
	delete(c.minioClient, objectName)
	return nil
}

func (c *FakeMinioClient) GetObjectCount() int {
	return len(c.minioClient)
}

func (c *FakeMinioClient) ExistObject(objectName string) bool {
	_, ok := c.minioClient[objectName]
	return ok
}
