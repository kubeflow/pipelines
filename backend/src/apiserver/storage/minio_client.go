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

	minio "github.com/minio/minio-go/v7"
)

// Create interface for minio client struct, making it more unit testable.
type MinioClientInterface interface {
	PutObject(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (n int64, err error)
	GetObject(ctx context.Context, bucketName, objectName string, opts minio.GetObjectOptions) (io.Reader, error)
	DeleteObject(ctx context.Context, bucketName, objectName string) error
}

type MinioClient struct {
	Client *minio.Client
}

func (c *MinioClient) PutObject(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (n int64, err error) {
	info, err := c.Client.PutObject(ctx, bucketName, objectName, reader, objectSize, opts)
	if err != nil {
		return 0, err
	}
	return info.Size, nil
}

func (c *MinioClient) GetObject(ctx context.Context, bucketName, objectName string, opts minio.GetObjectOptions) (io.Reader, error) {
	return c.Client.GetObject(ctx, bucketName, objectName, opts)
}

func (c *MinioClient) DeleteObject(ctx context.Context, bucketName, objectName string) error {
	return c.Client.RemoveObject(ctx, bucketName, objectName, minio.RemoveObjectOptions{})
}
